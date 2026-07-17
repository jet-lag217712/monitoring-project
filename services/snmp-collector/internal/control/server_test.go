package control_test

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/control"
	"github.com/equate/ogsd/services/snmp-collector/internal/health"
	"github.com/equate/ogsd/services/snmp-collector/internal/status"
)

func TestUnsupportedProtocolVersion(t *testing.T) {
	env := startControlEnv(t)
	defer env.close()

	conn, err := net.Dial("unix", env.socket)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	_, _ = conn.Write([]byte(`{"version":99,"id":"x","method":"status.summary","params":{}}` + "\n"))
	line, err := bufio.NewReader(conn).ReadBytes('\n')
	if err != nil {
		t.Fatal(err)
	}
	var resp control.Response
	if err := json.Unmarshal(line, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.OK || resp.Error == nil || resp.Error.Code != control.CodeUnsupportedVersion {
		t.Fatalf("response=%#v", resp)
	}
}

func TestOperatorWorkflowPrepareCommitReloadAudit(t *testing.T) {
	env := startControlEnv(t)
	defer env.close()

	client := control.NewClient(env.socket)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	statusResp, err := client.Call(ctx, "1", "status.summary", nil)
	if err != nil || !statusResp.OK {
		t.Fatalf("status.summary: err=%v resp=%#v", err, statusResp)
	}
	if statusResp.Result["device_count"].(float64) < 1 {
		t.Fatalf("expected devices: %#v", statusResp.Result)
	}

	deviceResp, err := client.Call(ctx, "2", "device.get", map[string]any{"device_id": "dev-001"})
	if err != nil || !deviceResp.OK {
		t.Fatalf("device.get: err=%v resp=%#v", err, deviceResp)
	}

	prepare, err := client.Call(ctx, "3", "thresholds.prepare", map[string]any{
		"temperature_warning_c": 71.0,
	})
	if err != nil || !prepare.OK {
		t.Fatalf("thresholds.prepare: err=%v resp=%#v", err, prepare)
	}
	token := prepare.Result["confirm_token"].(string)
	revision := prepare.Result["revision"].(string)
	if token == "" || revision == "" {
		t.Fatalf("prepare result=%#v", prepare.Result)
	}

	bad, err := client.Call(ctx, "4", "thresholds.commit", map[string]any{
		"confirm_token": token,
		"revision":      "revision-stale",
	})
	if err != nil {
		t.Fatal(err)
	}
	if bad.OK || bad.Error == nil || bad.Error.Code != control.CodeRevisionMismatch {
		t.Fatalf("expected revision mismatch, got %#v", bad)
	}

	commit, err := client.Call(ctx, "5", "thresholds.commit", map[string]any{
		"confirm_token": token,
		"revision":      revision,
	})
	if err != nil || !commit.OK {
		t.Fatalf("thresholds.commit: err=%v resp=%#v", err, commit)
	}

	reload, err := client.Call(ctx, "6", "config.reload", nil)
	if err != nil || !reload.OK {
		t.Fatalf("config.reload: err=%v resp=%#v", err, reload)
	}

	cfg := env.manager.Current()
	if cfg.Health.TemperatureWarningC != 71 {
		t.Fatalf("runtime temperature=%v", cfg.Health.TemperatureWarningC)
	}

	auditData, err := os.ReadFile(env.audit)
	if err != nil {
		t.Fatal(err)
	}
	auditText := string(auditData)
	if !strings.Contains(auditText, `"action":"thresholds.commit"`) || !strings.Contains(auditText, `"success":true`) {
		t.Fatalf("audit missing successful commit: %s", auditText)
	}
	if !strings.Contains(auditText, `"action":"config.reload"`) {
		t.Fatalf("audit missing reload: %s", auditText)
	}
	if strings.Contains(auditText, "community") || strings.Contains(strings.ToLower(auditText), "password") {
		t.Fatalf("audit appears to contain secrets: %s", auditText)
	}
}

func TestConfirmTokenUnknown(t *testing.T) {
	env := startControlEnv(t)
	defer env.close()
	client := control.NewClient(env.socket)
	ctx := context.Background()

	prepare, err := client.Call(ctx, "1", "thresholds.prepare", map[string]any{"temperature_warning_c": 66.0})
	if err != nil || !prepare.OK {
		t.Fatalf("prepare: %#v err=%v", prepare, err)
	}

	resp, err := client.Call(ctx, "2", "thresholds.commit", map[string]any{
		"confirm_token": "deadbeef",
		"revision":      prepare.Result["revision"],
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.OK || resp.Error == nil || resp.Error.Code != control.CodeConfirmExpired {
		t.Fatalf("expected confirm expired, got %#v", resp)
	}
}

type controlEnv struct {
	socket  string
	audit   string
	manager *config.Manager
	server  *control.Server
	cancel  context.CancelFunc
}

func (e *controlEnv) close() {
	e.cancel()
	_ = e.server.Close()
}

func startControlEnv(t *testing.T) *controlEnv {
	t.Helper()
	root := t.TempDir()
	configPath := filepath.Join(root, "collector.yaml")
	managedPath := filepath.Join(root, "managed.yaml")
	// macOS sun_path is short; keep the socket under /tmp with a compact name.
	socketPath := filepath.Join("/tmp", fmt.Sprintf("sc-%d.sock", time.Now().UnixNano()%1_000_000_000))
	auditPath := filepath.Join(root, "a.log")
	t.Cleanup(func() { _ = os.Remove(socketPath) })

	writeFile(t, configPath, "site_id: site-001\ncollector:\n  id: collector-001\ninventory:\n  managed_path: managed.yaml\nadmin:\n  listen: \"127.0.0.1:0\"\n  control_socket: "+strconvQuote(socketPath)+"\nhealth:\n  temperature_warning_c: 65\ndevices:\n  - id: dev-001\n    host: 127.0.0.1\n    community_env: SNMP_COMMUNITY_DEV_001\n")
	writeFile(t, managedPath, "devices: []\n")

	cfg, err := config.LoadForValidation(configPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	manager, err := config.NewManager(configPath, cfg)
	if err != nil {
		t.Fatal(err)
	}
	store := status.New()
	store.SetRevision(config.ConfigRevision(cfg))
	store.RecordPoll(status.DevicePoll{DeviceID: "dev-001", Result: status.PollSuccess})

	server, err := control.NewServer(control.Options{
		SocketPath: socketPath,
		Manager:    manager,
		Status:     store,
		Health:     health.NewTracker(),
		Transport:  staticTransport{},
		AuditPath:  auditPath,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := server.Listen(); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() { _ = server.Serve(ctx) }()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(socketPath); err == nil {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	return &controlEnv{
		socket:  socketPath,
		audit:   auditPath,
		manager: manager,
		server:  server,
		cancel:  cancel,
	}
}

type staticTransport struct{}

func (staticTransport) Snapshot() status.TransportSnapshot {
	return status.TransportSnapshot{PublisherMode: "stdout", BufferAvailable: true}
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}
