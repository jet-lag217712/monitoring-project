package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunValidate(t *testing.T) {
	dir := t.TempDir()
	valid := filepath.Join(dir, "valid.yaml")
	if err := os.WriteFile(valid, []byte("site_id: site-001\ncollector:\n  id: collector-001\ndevices:\n  - id: dev-001\n    host: 127.0.0.1\n    community_env: SNMP_COMMUNITY_DEV_001\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := runValidate([]string{"-config", valid}); got != 0 {
		t.Fatalf("valid exit code=%d", got)
	}

	invalid := filepath.Join(dir, "invalid.yaml")
	if err := os.WriteFile(invalid, []byte("site_id: site-001\ncollector:\n  id: collector-001\ndevices:\n  - id: dev-001\n    host: 127.0.0.1\n    community: public\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := runValidate([]string{"-config", invalid}); got != 1 {
		t.Fatalf("invalid exit code=%d", got)
	}
}
