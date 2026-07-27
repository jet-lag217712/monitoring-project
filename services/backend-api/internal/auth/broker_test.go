package auth_test

import (
	"bufio"
	"context"
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/equate/ogsd/services/backend-api/internal/auth"
)

func TestBrokerClientAuthenticate(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "auth.sock")
	server := startBrokerServer(t, socketPath, func(req authTestRequest) authTestResponse {
		if req.Operation == "authenticate" && req.Username == "alice" && req.Password == "secret" {
			return authTestResponse{OK: true, Username: "alice"}
		}
		return authTestResponse{OK: false}
	})
	defer server.Close()

	client, err := auth.NewBrokerClient(socketPath, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	username, ok, err := client.Authenticate(context.Background(), "alice", "secret")
	if err != nil || !ok || username != "alice" {
		t.Fatalf("authenticate ok=%v username=%q err=%v", ok, username, err)
	}

	_, ok, err = client.Authenticate(context.Background(), "alice", "wrong")
	if err != nil || ok {
		t.Fatalf("expected failed auth, ok=%v err=%v", ok, err)
	}
}

func TestBrokerClientAccountStatus(t *testing.T) {
	socketPath := filepath.Join(t.TempDir(), "auth.sock")
	server := startBrokerServer(t, socketPath, func(req authTestRequest) authTestResponse {
		if req.Operation == "account_status" && req.Username == "alice" {
			return authTestResponse{OK: true, Username: "alice"}
		}
		return authTestResponse{OK: false}
	})
	defer server.Close()

	client, err := auth.NewBrokerClient(socketPath, time.Second)
	if err != nil {
		t.Fatal(err)
	}

	active, err := client.AccountStatus(context.Background(), "alice")
	if err != nil || !active {
		t.Fatalf("active=%v err=%v", active, err)
	}
}

func TestBrokerClientRejectsRelativeSocketPath(t *testing.T) {
	_, err := auth.NewBrokerClient("relative.sock", time.Second)
	if err == nil {
		t.Fatal("expected error for relative socket path")
	}
}

type authTestRequest struct {
	Operation string `json:"operation"`
	Username  string `json:"username"`
	Password  string `json:"password,omitempty"`
}

type authTestResponse struct {
	OK       bool   `json:"ok"`
	Username string `json:"username"`
	Error    string `json:"error"`
}

func startBrokerServer(t *testing.T, socketPath string, handler func(authTestRequest) authTestResponse) net.Listener {
	t.Helper()
	_ = os.Remove(socketPath)
	ln, err := net.Listen("unix", socketPath)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				reader := bufio.NewReader(c)
				line, err := reader.ReadBytes('\n')
				if err != nil {
					return
				}
				var req authTestRequest
				if err := json.Unmarshal(line, &req); err != nil {
					return
				}
				resp := handler(req)
				encoded, _ := json.Marshal(resp)
				encoded = append(encoded, '\n')
				_, _ = c.Write(encoded)
			}(conn)
		}
	}()
	return ln
}
