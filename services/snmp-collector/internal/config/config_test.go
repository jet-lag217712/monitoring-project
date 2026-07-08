package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeTempConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "collector.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadValid(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `
site_id: "site-001"
poll_interval: 30s
max_workers: 5
admin:
  listen: ":9091"
snmp:
  timeout: 3s
  retries: 1
devices:
  - id: "dev-001"
    host: "127.0.0.1"
    port: 161
    community: "public"
    version: "2c"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.SiteID != "site-001" {
		t.Fatalf("site_id=%q", cfg.SiteID)
	}
	if cfg.PollInterval != 30*time.Second {
		t.Fatalf("poll_interval=%v", cfg.PollInterval)
	}
	if len(cfg.Devices) != 1 || cfg.Devices[0].ID != "dev-001" {
		t.Fatalf("devices=%#v", cfg.Devices)
	}
}

func TestValidateMissingSiteID(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `
devices:
  - id: "dev-001"
    host: "127.0.0.1"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for missing site_id")
	}
}

func TestValidateDuplicateDeviceID(t *testing.T) {
	t.Parallel()

	path := writeTempConfig(t, `
site_id: "site-001"
devices:
  - id: "dev-001"
    host: "127.0.0.1"
  - id: "dev-001"
    host: "127.0.0.2"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected duplicate device id error")
	}
}

func TestCommunityEnvOverride(t *testing.T) {
	path := writeTempConfig(t, `
site_id: "site-001"
devices:
  - id: "dev-001"
    host: "127.0.0.1"
    community: "public"
    version: "2c"
`)
	t.Setenv("SNMP_COMMUNITY_DEV_001", "secret")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Devices[0].Community != "secret" {
		t.Fatalf("community=%q, want secret", cfg.Devices[0].Community)
	}
}

func TestCommunityEnvKey(t *testing.T) {
	t.Parallel()
	if got := communityEnvKey("dev-001"); got != "SNMP_COMMUNITY_DEV_001" {
		t.Fatalf("got %q", got)
	}
}
