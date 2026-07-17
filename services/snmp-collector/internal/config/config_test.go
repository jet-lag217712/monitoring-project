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
collector:
  id: "collector-001"
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
    community_env: "SNMP_COMMUNITY_DEV_001"
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
collector:
  id: "collector-001"
devices:
  - id: "dev-001"
    host: "127.0.0.1"
    community_env: "SNMP_COMMUNITY_DEV_001"
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
collector:
  id: "collector-001"
devices:
  - id: "dev-001"
    host: "127.0.0.1"
    community_env: "SNMP_COMMUNITY_DEV_001"
  - id: "dev-001"
    host: "127.0.0.2"
    community_env: "SNMP_COMMUNITY_DEV_002"
`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected duplicate device id error")
	}
}

func TestCommunityEnvReference(t *testing.T) {
	path := writeTempConfig(t, `
site_id: "site-001"
collector:
  id: "collector-001"
devices:
  - id: "dev-001"
    host: "127.0.0.1"
    community_env: "SNMP_COMMUNITY_DEV_001"
    version: "2c"
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Devices[0].CommunityEnv != "SNMP_COMMUNITY_DEV_001" {
		t.Fatalf("community_env=%q", cfg.Devices[0].CommunityEnv)
	}
}

func TestMQTTModeRequiresPassword(t *testing.T) {
	path := writeTempConfig(t, `
site_id: "site-001"
collector:
  id: "collector-001"
publisher:
  mode: mqtt
  timeout: 5s
mqtt:
  broker: "tls://127.0.0.1:8883"
  username: "collector"
  password_env: "MQTT_PASSWORD"
  qos: 1
  tls:
    ca_file: "/tmp/ca.crt"
  reconnect:
    initial: 1s
    max: 60s
devices:
  - id: "dev-001"
    host: "127.0.0.1"
    community_env: "SNMP_COMMUNITY_DEV_001"
    version: "2c"
`)
	t.Setenv("MQTT_PASSWORD", "")
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected missing password error")
	}
}

func TestMQTTModeValid(t *testing.T) {
	path := writeTempConfig(t, `
site_id: "site-001"
collector:
  id: "collector-001"
publisher:
  mode: mqtt
  timeout: 5s
mqtt:
  broker: "tls://127.0.0.1:8883"
  username: "collector"
  password_env: "MQTT_PASSWORD"
  qos: 1
  tls:
    ca_file: "/tmp/ca.crt"
  reconnect:
    initial: 1s
    max: 60s
devices:
  - id: "dev-001"
    host: "127.0.0.1"
    community_env: "SNMP_COMMUNITY_DEV_001"
    version: "2c"
`)
	t.Setenv("MQTT_PASSWORD", "secret")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Publisher.Mode != "mqtt" {
		t.Fatalf("mode=%q", cfg.Publisher.Mode)
	}
	if cfg.MQTT.ClientID != "collector-site-001" {
		t.Fatalf("client_id=%q", cfg.MQTT.ClientID)
	}
	if cfg.Buffer.BusyTimeoutMS != 5000 {
		t.Fatalf("busy_timeout_ms=%d", cfg.Buffer.BusyTimeoutMS)
	}
}
