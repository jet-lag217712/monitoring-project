package control

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditEntry is a secret-free mutation audit record.
type AuditEntry struct {
	Timestamp time.Time      `json:"timestamp"`
	Action    string         `json:"action"`
	Success   bool           `json:"success"`
	Code      string         `json:"code,omitempty"`
	Message   string         `json:"message,omitempty"`
	Revision  string         `json:"revision,omitempty"`
	Details   map[string]any `json:"details,omitempty"`
}

// Auditor appends secret-free audit lines.
type Auditor struct {
	mu   sync.Mutex
	path string
}

// NewAuditor creates an auditor writing JSON lines to path (mode 0600).
func NewAuditor(path string) (*Auditor, error) {
	if path == "" {
		return nil, fmt.Errorf("audit path is required")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create audit directory: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open audit log: %w", err)
	}
	_ = f.Close()
	if err := os.Chmod(path, 0o600); err != nil {
		return nil, fmt.Errorf("chmod audit log: %w", err)
	}
	return &Auditor{path: path}, nil
}

// Path returns the audit log path.
func (a *Auditor) Path() string {
	if a == nil {
		return ""
	}
	return a.path
}

// Record appends one audit entry. Details must never include secrets.
func (a *Auditor) Record(entry AuditEntry) error {
	if a == nil {
		return nil
	}
	if entry.Timestamp.IsZero() {
		entry.Timestamp = time.Now().UTC()
	} else {
		entry.Timestamp = entry.Timestamp.UTC()
	}
	data, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	f, err := os.OpenFile(a.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return f.Sync()
}
