package appliance

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
)

// ManagerRequest is the stable, local-only Appliance Manager protocol.
// No network listener or public management HTTP API is provided.
type ManagerRequest struct {
	Version int            `json:"version"`
	ID      string         `json:"id"`
	Method  string         `json:"method"`
	Params  map[string]any `json:"params,omitempty"`
}

// ManagerResponse is the NDJSON response for a ManagerRequest.
type ManagerResponse struct {
	Version int            `json:"version"`
	ID      string         `json:"id"`
	OK      bool           `json:"ok"`
	Result  map[string]any `json:"result,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// ManagerServer exposes a deliberately small privileged appliance API over a
// Unix socket. The Equate Core socket remains separate and owns application
// configuration and polling actions.
type ManagerServer struct {
	layout     Layout
	socketPath string
	mu         sync.Mutex
	listener   net.Listener
}

func NewManagerServer(layout Layout, socketPath string) *ManagerServer {
	if socketPath == "" {
		socketPath = layout.ManagerSocket
	}
	return &ManagerServer{layout: layout, socketPath: socketPath}
}

func (s *ManagerServer) Listen() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return errors.New("appliance manager socket already listening")
	}
	if err := s.layout.Ensure(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0o750); err != nil {
		return fmt.Errorf("create manager socket directory: %w", err)
	}
	_ = os.Remove(s.socketPath)
	oldMask := syscall.Umask(0o117) // socket starts at 0660.
	ln, err := net.Listen("unix", s.socketPath)
	syscall.Umask(oldMask)
	if err != nil {
		return fmt.Errorf("listen appliance manager socket: %w", err)
	}
	if err := os.Chmod(s.socketPath, 0o660); err != nil {
		_ = ln.Close()
		return fmt.Errorf("set manager socket permissions: %w", err)
	}
	s.listener = ln
	return nil
}

func (s *ManagerServer) Serve(ctx context.Context) error {
	s.mu.Lock()
	ln := s.listener
	s.mu.Unlock()
	if ln == nil {
		return errors.New("appliance manager socket is not listening")
	}
	go func() { <-ctx.Done(); _ = ln.Close() }()
	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil || errors.Is(err, net.ErrClosed) {
				return nil
			}
			continue
		}
		go s.handleConnection(ctx, conn)
	}
}

func (s *ManagerServer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		_ = s.listener.Close()
		s.listener = nil
	}
	return os.Remove(s.socketPath)
}

func (s *ManagerServer) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReader(conn)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return
			}
			return
		}
		var request ManagerRequest
		if err := json.Unmarshal(line, &request); err != nil {
			_ = writeManagerResponse(conn, ManagerResponse{Version: 1, OK: false, Error: "invalid request"})
			continue
		}
		response := s.dispatch(ctx, request)
		if err := writeManagerResponse(conn, response); err != nil {
			return
		}
	}
}

func (s *ManagerServer) dispatch(ctx context.Context, request ManagerRequest) ManagerResponse {
	response := ManagerResponse{Version: 1, ID: request.ID}
	if request.Version != 1 || request.ID == "" || request.Method == "" {
		response.Error = "invalid request"
		return response
	}
	switch request.Method {
	case "status.get":
		current, err := s.layout.CurrentRelease()
		if err != nil {
			current = "unconfigured"
		}
		response.OK = true
		response.Result = map[string]any{"release": current}
	case "release.activate":
		version, _ := request.Params["version"].(string)
		if err := s.layout.Activate(version); err != nil {
			response.Error = err.Error()
			return response
		}
		response.OK = true
		response.Result = map[string]any{"activated": version}
	case "factory_reset":
		confirmation, _ := request.Params["confirmation"].(string)
		if confirmation != "FACTORY-RESET" {
			response.Error = "factory reset requires explicit confirmation"
			return response
		}
		if err := s.layout.FactoryResetAndReboot(); err != nil {
			response.Error = err.Error()
			return response
		}
		response.OK = true
		response.Result = map[string]any{"factory_reset": true}
	case "support_bundle.create":
		output, _ := request.Params["output"].(string)
		path, err := s.layout.CreateSupportBundle(output)
		if err != nil {
			response.Error = err.Error()
			return response
		}
		response.OK = true
		response.Result = map[string]any{"path": path}
	default:
		response.Error = "unknown method"
	}
	return response
}

func writeManagerResponse(writer io.Writer, response ManagerResponse) error {
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}
	_, err = writer.Write(append(data, '\n'))
	return err
}
