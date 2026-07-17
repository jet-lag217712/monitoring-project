package control

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

// Client is a Unix NDJSON control client.
type Client struct {
	socketPath string
	timeout    time.Duration
}

// NewClient creates a control client for socketPath.
func NewClient(socketPath string) *Client {
	return &Client{socketPath: socketPath, timeout: DefaultRequestTimeout}
}

// Call sends one request and returns the decoded response.
func (c *Client) Call(ctx context.Context, id, method string, params map[string]any) (Response, error) {
	if params == nil {
		params = map[string]any{}
	}
	req := Request{
		Version: ProtocolVersion,
		ID:      id,
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return Response{}, err
	}
	data = append(data, '\n')
	if len(data) > MaxRequestBytes {
		return Response{}, fmt.Errorf("request exceeds maximum size")
	}

	dialer := net.Dialer{Timeout: c.timeout}
	conn, err := dialer.DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return Response{}, fmt.Errorf("dial control socket: %w", err)
	}
	defer conn.Close()

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(c.timeout)
	}
	_ = conn.SetDeadline(deadline)

	if _, err := conn.Write(data); err != nil {
		return Response{}, err
	}
	reader := bufio.NewReader(conn)
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return Response{}, err
	}
	if len(line) > MaxResponseBytes {
		return Response{}, fmt.Errorf("response exceeds maximum size")
	}
	var resp Response
	if err := json.Unmarshal(line, &resp); err != nil {
		return Response{}, err
	}
	return resp, nil
}
