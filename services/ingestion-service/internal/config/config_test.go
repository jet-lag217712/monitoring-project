package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/equate/ogsd/services/ingestion-service/internal/config"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "ingestion.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func baseYAML() string {
	return `
admin:
  listen: ":9091"
mqtt:
  broker: "tls://127.0.0.1:8883"
  client_id: "ingestion"
  username: "ingestion"
  password_env: "MQTT_PASSWORD"
  qos: 1
  topic: "site/+/device/+/metric/#"
  tls:
    ca_file: "/tmp/ca.crt"
  reconnect:
    initial: 1s
    max: 60s
database:
  url_env: "DATABASE_URL"
  max_conns: 10
  min_conns: 1
  max_lifetime: 1h
`
}

func TestLoad_RequiresMQTTPassword(t *testing.T) {
	t.Setenv("MQTT_PASSWORD", "")
	t.Setenv("DATABASE_URL", "postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable")
	_, err := config.Load(writeConfig(t, baseYAML()))
	if err == nil {
		t.Fatal("expected error for missing MQTT password")
	}
}

func TestLoad_RequiresDatabaseURL(t *testing.T) {
	t.Setenv("MQTT_PASSWORD", "ingestion")
	t.Setenv("DATABASE_URL", "")
	_, err := config.Load(writeConfig(t, baseYAML()))
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
}

func TestLoad_QoSMustBe1(t *testing.T) {
	t.Setenv("MQTT_PASSWORD", "ingestion")
	t.Setenv("DATABASE_URL", "postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable")
	body := `
admin:
  listen: ":9091"
mqtt:
  broker: "tls://127.0.0.1:8883"
  username: "ingestion"
  password_env: "MQTT_PASSWORD"
  qos: 2
  topic: "site/+/device/+/metric/#"
  tls:
    ca_file: "/tmp/ca.crt"
  reconnect:
    initial: 1s
    max: 60s
database:
  url_env: "DATABASE_URL"
`
	_, err := config.Load(writeConfig(t, body))
	if err == nil {
		t.Fatal("expected error for qos != 1")
	}
}

func TestLoad_OK(t *testing.T) {
	t.Setenv("MQTT_PASSWORD", "ingestion")
	t.Setenv("DATABASE_URL", "postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable")
	cfg, err := config.Load(writeConfig(t, baseYAML()))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MQTT.QoS != 1 {
		t.Fatalf("qos=%d", cfg.MQTT.QoS)
	}
	if cfg.Admin.Listen != ":9091" {
		t.Fatalf("listen=%q", cfg.Admin.Listen)
	}
}
