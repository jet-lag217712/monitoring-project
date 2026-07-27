package auth

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	maxUsernameBytes       = 64
	maxPasswordBytes       = 1024
	maxBrokerResponseBytes = 4096
)

// Broker verifies credentials and account status outside the API container.
type Broker interface {
	Authenticate(ctx context.Context, username, password string) (string, bool, error)
	AccountStatus(ctx context.Context, username string) (bool, error)
}

// BrokerClient speaks newline-delimited JSON to the host authentication broker.
type BrokerClient struct {
	socketPath string
	timeout    time.Duration
}

type brokerRequest struct {
	Operation string `json:"operation"`
	Username  string `json:"username"`
	Password  string `json:"password,omitempty"`
}

type brokerResponse struct {
	OK       bool   `json:"ok"`
	Username string `json:"username"`
	Error    string `json:"error"`
}

// NewBrokerClient creates a bounded Unix-socket broker client.
func NewBrokerClient(socketPath string, timeout time.Duration) (*BrokerClient, error) {
	if !strings.HasPrefix(socketPath, "/") {
		return nil, fmt.Errorf("broker socket must be an absolute path")
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("broker timeout must be positive")
	}
	return &BrokerClient{socketPath: socketPath, timeout: timeout}, nil
}

// Authenticate asks the host broker to validate an OS username and password.
func (c *BrokerClient) Authenticate(ctx context.Context, username, password string) (string, bool, error) {
	if err := validateUsername(username); err != nil {
		return "", false, err
	}
	if err := validatePassword(password); err != nil {
		return "", false, err
	}
	resp, err := c.request(ctx, brokerRequest{
		Operation: "authenticate",
		Username:  username,
		Password:  password,
	})
	if err != nil {
		return "", false, err
	}
	if !resp.OK {
		return "", false, nil
	}
	if err := validateUsername(resp.Username); err != nil {
		return "", false, fmt.Errorf("broker returned invalid username: %w", err)
	}
	if resp.Username != username {
		return "", false, errors.New("broker returned a different username")
	}
	return resp.Username, true, nil
}

// AccountStatus asks the host broker whether an authenticated OS account remains usable.
func (c *BrokerClient) AccountStatus(ctx context.Context, username string) (bool, error) {
	if err := validateUsername(username); err != nil {
		return false, err
	}
	resp, err := c.request(ctx, brokerRequest{
		Operation: "account_status",
		Username:  username,
	})
	if err != nil {
		return false, err
	}
	if !resp.OK {
		return false, nil
	}
	if resp.Username != username {
		return false, errors.New("broker returned a different username")
	}
	return true, nil
}

func (c *BrokerClient) request(ctx context.Context, req brokerRequest) (brokerResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", c.socketPath)
	if err != nil {
		return brokerResponse{}, fmt.Errorf("connect authentication broker: %w", err)
	}
	defer conn.Close()

	deadline := time.Now().Add(c.timeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return brokerResponse{}, fmt.Errorf("set authentication broker deadline: %w", err)
	}

	encoded, err := json.Marshal(req)
	if err != nil {
		return brokerResponse{}, fmt.Errorf("encode authentication broker request: %w", err)
	}
	encoded = append(encoded, '\n')
	if _, err := conn.Write(encoded); err != nil {
		return brokerResponse{}, fmt.Errorf("write authentication broker request: %w", err)
	}

	reader := bufio.NewReader(io.LimitReader(conn, maxBrokerResponseBytes+1))
	line, err := reader.ReadBytes('\n')
	if err != nil {
		return brokerResponse{}, fmt.Errorf("read authentication broker response: %w", err)
	}
	if len(line) > maxBrokerResponseBytes {
		return brokerResponse{}, errors.New("authentication broker response is too large")
	}

	var resp brokerResponse
	decoder := json.NewDecoder(strings.NewReader(string(line)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&resp); err != nil {
		return brokerResponse{}, fmt.Errorf("decode authentication broker response: %w", err)
	}
	if resp.OK && resp.Error != "" {
		return brokerResponse{}, errors.New("authentication broker returned inconsistent response")
	}
	return resp, nil
}

func validateUsername(username string) error {
	if username == "" {
		return errors.New("username is required")
	}
	if len(username) > maxUsernameBytes || !utf8.ValidString(username) {
		return errors.New("username is too long or invalid")
	}
	if strings.TrimSpace(username) != username {
		return errors.New("username has surrounding whitespace")
	}
	for _, r := range username {
		if unicode.IsControl(r) {
			return errors.New("username contains control characters")
		}
	}
	return nil
}

func validatePassword(password string) error {
	if password == "" {
		return errors.New("password is required")
	}
	if len(password) > maxPasswordBytes || !utf8.ValidString(password) {
		return errors.New("password is too long or invalid")
	}
	return nil
}
