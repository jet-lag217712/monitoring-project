package control

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/health"
	"github.com/equate/ogsd/services/snmp-collector/internal/status"
)

// ConfigManager is the reloadable configuration surface used by control handlers.
type ConfigManager interface {
	Current() *config.Config
	Reload() error
}

// TransportView supplies publisher/buffer readiness for operator views.
type TransportView interface {
	Snapshot() status.TransportSnapshot
}

// Server is the Unix NDJSON control plane.
type Server struct {
	socketPath string
	manager    ConfigManager
	status     *status.Store
	health     *health.Tracker
	transport  TransportView
	audit      *Auditor
	log        *slog.Logger
	timeout    time.Duration

	pending *pendingStore

	discovery *discoveryStore

	scanMu    sync.RWMutex
	scanState activeScanState

	mu         sync.Mutex
	mutationMu sync.Mutex
	listener   net.Listener
}

// Options configures a control server.
type Options struct {
	StateDir   string
	SocketPath string
	Manager    ConfigManager
	Status     *status.Store
	Health     *health.Tracker
	Transport  TransportView
	AuditPath  string
	Log        *slog.Logger
	Timeout    time.Duration
}

// NewServer constructs a control server. Call Listen to bind the socket.
func NewServer(opts Options) (*Server, error) {
	if opts.SocketPath == "" {
		return nil, errors.New("control socket path is required")
	}
	if opts.Manager == nil {
		return nil, errors.New("config manager is required")
	}
	if opts.Status == nil {
		opts.Status = status.New()
	}
	if opts.Health == nil {
		opts.Health = health.NewTracker()
	}
	if opts.Log == nil {
		opts.Log = slog.Default()
	}
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultRequestTimeout
	}
	auditPath := opts.AuditPath
	if auditPath == "" {
		auditPath = filepath.Join(filepath.Dir(opts.SocketPath), "control-audit.log")
	}
	auditor, err := NewAuditor(auditPath)
	if err != nil {
		return nil, err
	}
	stateDir := opts.StateDir
	if stateDir == "" && opts.SocketPath != "" {
		stateDir = filepath.Dir(opts.SocketPath)
	}
	return &Server{
		socketPath: opts.SocketPath,
		manager:    opts.Manager,
		status:     opts.Status,
		health:     opts.Health,
		transport:  opts.Transport,
		audit:      auditor,
		log:        opts.Log,
		timeout:    opts.Timeout,
		pending:    newPendingStore(),
		discovery:  newDiscoveryStore(stateDir),
	}, nil
}

// SocketPath returns the configured Unix socket path.
func (s *Server) SocketPath() string { return s.socketPath }

// AuditPath returns the audit log path.
func (s *Server) AuditPath() string { return s.audit.Path() }

// Listen binds the Unix socket and best-effort sets mode 0600.
//
// Docker Desktop / macOS host bind mounts often reject chmod on Unix sockets
// with EINVAL. In that case we keep the listener up and log a warning: OS
// path access still gates the socket, and Linux volumes/systemd RuntimeDirectory
// continue to get a real 0600 mode.
func (s *Server) Listen() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return errors.New("control server already listening")
	}
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0o700); err != nil {
		return fmt.Errorf("create control socket directory: %w", err)
	}
	_ = os.Remove(s.socketPath)

	// Prefer creating the socket as 0600 when the filesystem honors umask.
	oldMask := syscall.Umask(0o077)
	ln, err := net.Listen("unix", s.socketPath)
	syscall.Umask(oldMask)
	if err != nil {
		return fmt.Errorf("listen control socket: %w", err)
	}
	if err := os.Chmod(s.socketPath, 0o600); err != nil {
		if isSocketChmodUnsupported(err) {
			s.log.Warn("control socket chmod unsupported on this filesystem; continuing without 0600",
				"socket", s.socketPath,
				"err", err,
			)
		} else {
			_ = ln.Close()
			_ = os.Remove(s.socketPath)
			return fmt.Errorf("chmod control socket: %w", err)
		}
	}
	s.listener = ln
	return nil
}

// isSocketChmodUnsupported reports filesystems that cannot chmod Unix sockets
// (notably Docker Desktop bind mounts on macOS).
func isSocketChmodUnsupported(err error) bool {
	if err == nil {
		return false
	}
	var pathErr *os.PathError
	if errors.As(err, &pathErr) {
		err = pathErr.Err
	}
	return errors.Is(err, syscall.EINVAL) || errors.Is(err, syscall.EOPNOTSUPP)
}

// Serve accepts connections until ctx is cancelled or Close is called.
func (s *Server) Serve(ctx context.Context) error {
	s.mu.Lock()
	ln := s.listener
	s.mu.Unlock()
	if ln == nil {
		return errors.New("control server is not listening")
	}

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
			}
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			s.log.Error("control accept failed", "err", err)
			continue
		}
		go s.handleConn(ctx, conn)
	}
}

// Close stops the listener and removes the socket file.
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var err error
	if s.listener != nil {
		err = s.listener.Close()
		s.listener = nil
	}
	_ = os.Remove(s.socketPath)
	return err
}

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	reader := bufio.NewReaderSize(conn, MaxRequestBytes+1024)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		_ = conn.SetReadDeadline(time.Now().Add(s.timeout))
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.log.Debug("control read ended", "err", err)
			}
			return
		}
		if len(line) > MaxRequestBytes {
			_ = writeResponse(conn, errorResponse("", CodeInvalidRequest, "request exceeds maximum size"))
			return
		}
		reqCtx, cancel := context.WithTimeout(ctx, s.timeout)
		resp := s.dispatch(reqCtx, line)
		cancel()
		data, err := json.Marshal(resp)
		if err != nil {
			_ = writeResponse(conn, errorResponse(resp.ID, CodeInternal, "encode response failed"))
			return
		}
		if len(data) > MaxResponseBytes {
			_ = writeResponse(conn, errorResponse(resp.ID, CodeInternal, "response exceeds maximum size"))
			return
		}
		if err := writeRaw(conn, data); err != nil {
			return
		}
	}
}

func (s *Server) dispatch(ctx context.Context, line []byte) Response {
	var req Request
	if err := json.Unmarshal(line, &req); err != nil {
		return errorResponse("", CodeInvalidRequest, "malformed JSON request")
	}
	if req.Version != ProtocolVersion {
		return errorResponse(req.ID, CodeUnsupportedVersion, fmt.Sprintf("unsupported protocol version %d", req.Version))
	}
	if req.ID == "" || req.Method == "" {
		return errorResponse(req.ID, CodeInvalidRequest, "id and method are required")
	}
	if req.Params == nil {
		req.Params = map[string]any{}
	}

	result, err := s.handle(ctx, req)
	if err != nil {
		var pe *ProtocolError
		if errors.As(err, &pe) {
			return errorResponse(req.ID, pe.Code, pe.Message)
		}
		return errorResponse(req.ID, CodeInternal, err.Error())
	}
	return Response{
		Version: ProtocolVersion,
		ID:      req.ID,
		OK:      true,
		Result:  result,
	}
}

func errorResponse(id, code, message string) Response {
	return Response{
		Version: ProtocolVersion,
		ID:      id,
		OK:      false,
		Error:   &ErrorBody{Code: code, Message: message},
	}
}

func writeResponse(w io.Writer, resp Response) error {
	data, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return writeRaw(w, data)
}

func writeRaw(w io.Writer, data []byte) error {
	if _, err := w.Write(data); err != nil {
		return err
	}
	_, err := w.Write([]byte("\n"))
	return err
}
