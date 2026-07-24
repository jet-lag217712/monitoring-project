// Package appliance implements the host-only control plane for an Equate
// ISO-installed appliance.
// It deliberately has no network listener; callers reach it through the local
// console or the restricted SSH ForceCommand.
package appliance

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Layout is the complete persistent/runtime filesystem contract for an
// appliance. Root is injectable solely for tests and image-build staging.
type Layout struct {
	Root           string
	Etc            string
	Secrets        string
	Data           string
	Backups        string
	Updates        string
	Cache          string
	Releases       string
	Runtime        string
	Rendered       string
	Certificates   string
	TLSCertificate string
	TLSImport      string
	Identity       string
	Sockets        string
	ManagerDir     string
	ManagerSocket  string
	ApplicationYML string
	SetupMarker    string
}

// NewLayout returns the standard Equate appliance layout rooted at root.
func NewLayout(root string) Layout {
	if strings.TrimSpace(root) == "" {
		root = "/"
	}
	root = filepath.Clean(root)
	join := func(parts ...string) string {
		return filepath.Join(append([]string{root}, parts...)...)
	}
	etc := join("etc", "equate")
	data := join("var", "lib", "equate")
	runtime := join("run", "equate")
	return Layout{
		Root:           root,
		Etc:            etc,
		Secrets:        filepath.Join(etc, "secrets"),
		Data:           data,
		Backups:        filepath.Join(data, "backups"),
		Updates:        filepath.Join(data, "updates"),
		Cache:          filepath.Join(data, "cache"),
		Releases:       join("opt", "equate", "releases"),
		Runtime:        runtime,
		Rendered:       filepath.Join(runtime, "rendered"),
		Certificates:   filepath.Join(etc, "certificates"),
		TLSCertificate: filepath.Join(etc, "certificates", "tls.crt"),
		TLSImport:      filepath.Join(runtime, "tls-import"),
		Identity:       filepath.Join(runtime, "identity.txt"),
		Sockets:        filepath.Join(runtime, "sockets"),
		ManagerDir:     filepath.Join(runtime, "manager"),
		ManagerSocket:  filepath.Join(runtime, "manager", "appliance.sock"),
		ApplicationYML: filepath.Join(etc, "application.yaml"),
		SetupMarker:    filepath.Join(data, ".setup-complete"),
	}
}

// Ensure creates every mutable appliance directory with conservative modes.
func (l Layout) Ensure() error {
	for _, dir := range []string{l.Etc, l.Secrets, l.Data, l.Backups, l.Updates, l.Cache, l.Runtime, l.Rendered, l.Certificates, l.TLSImport, l.Sockets, l.ManagerDir, l.Releases} {
		mode := os.FileMode(0o750)
		if dir == l.Secrets || dir == l.Rendered || dir == l.Sockets {
			mode = 0o700
		}
		if err := os.MkdirAll(dir, mode); err != nil {
			return fmt.Errorf("create %s: %w", dir, err)
		}
		if err := os.Chmod(dir, mode); err != nil {
			return fmt.Errorf("set mode on %s: %w", dir, err)
		}
	}
	return nil
}

// AtomicWriteFile writes a complete file, fsyncs it, and atomically replaces
// the destination. Callers must select an explicit restrictive file mode.
func AtomicWriteFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("create parent directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".equate-*")
	if err != nil {
		return fmt.Errorf("create temporary file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName) //nolint:errcheck
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("set temporary file mode: %w", err)
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temporary file: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("sync temporary file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temporary file: %w", err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replace %s: %w", path, err)
	}
	return nil
}
