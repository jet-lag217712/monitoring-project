package setup

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/control"
)

const containerControlSocket = "/run/snmp-collector/control.sock"

type controlCaller interface {
	Call(ctx context.Context, id, method string, params map[string]any) (control.Response, error)
}

type deployControl struct {
	deployDir   string
	hostSocket  string
	innerSocket string
	service     string
	preferExec  bool
	direct      *control.Client
}

func newDeployControl(deployDir string, spec SiteSpec) *deployControl {
	hostSocket := spec.SocketPath(deployDir)
	d := &deployControl{
		deployDir:   deployDir,
		hostSocket:  hostSocket,
		innerSocket: containerControlSocket,
		service:     spec.ServiceName,
		direct:      control.NewClient(hostSocket),
	}
	ctx, cancel := context.WithTimeout(context.Background(), control.DefaultRequestTimeout)
	defer cancel()
	if _, err := d.direct.Call(ctx, "probe", "status.summary", nil); err != nil && isUnixSocketDialError(err) {
		d.preferExec = true
	}
	return d
}

func (d *deployControl) Call(ctx context.Context, id, method string, params map[string]any) (control.Response, error) {
	if !d.preferExec {
		resp, err := d.direct.Call(ctx, id, method, params)
		if err == nil {
			return resp, nil
		}
		if !isUnixSocketDialError(err) {
			return resp, err
		}
		d.preferExec = true
	}
	return d.callViaComposeExec(ctx, id, method, params)
}

func (d *deployControl) callViaComposeExec(ctx context.Context, id, method string, params map[string]any) (control.Response, error) {
	if params == nil {
		params = map[string]any{}
	}
	line, err := json.Marshal(control.Request{
		Version: control.ProtocolVersion,
		ID:      id,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return control.Response{}, err
	}
	line = append(line, '\n')

	timeout := requestTimeout(ctx, method)
	cmd := exec.CommandContext(
		ctx,
		"docker", "compose", "-f", "docker-compose.yml", "-f", generatedComposeFile,
		"exec", "-T", d.service,
		"/collector", "rpc", "-socket", d.innerSocket, "-timeout", timeout.String(),
	)
	cmd.Dir = d.deployDir
	cmd.Stdin = bytes.NewReader(line)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return control.Response{}, fmt.Errorf("control via docker compose exec: %w (%s)", err, msg)
	}
	var resp control.Response
	if err := json.Unmarshal(bytes.TrimSpace(stdout.Bytes()), &resp); err != nil {
		return control.Response{}, fmt.Errorf("decode control response: %w", err)
	}
	return resp, nil
}

func isUnixSocketDialError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "dial control socket") ||
		strings.Contains(msg, "connect: connection refused") ||
		strings.Contains(msg, "no such file or directory")
}

func requestTimeout(ctx context.Context, method string) time.Duration {
	floor := control.DefaultRequestTimeout
	if method == "discovery.scan.start" {
		floor = 10 * time.Minute
	}
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining > floor {
			return remaining
		}
		if remaining > 0 {
			return remaining
		}
	}
	return floor
}

func isDiscoveryRetryable(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "timeout") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "deadline exceeded") ||
		strings.Contains(msg, "context canceled")
}
