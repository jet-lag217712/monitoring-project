package setup

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvFileValue(t *testing.T) {
	path := filepath.Join(t.TempDir(), "compose.env")
	if err := os.WriteFile(path, []byte("COMPOSE_PROJECT_NAME=equate-appliance\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := envFileValue(path, "COMPOSE_PROJECT_NAME")
	if err != nil {
		t.Fatal(err)
	}
	if got != "equate-appliance" {
		t.Fatalf("value=%q", got)
	}
}

func TestApplianceProfileProjectName(t *testing.T) {
	cfg := ProfileConfigFor(ProfileAppliance)
	if cfg.ComposeProjectName != "equate-appliance" {
		t.Fatalf("project=%q", cfg.ComposeProjectName)
	}
}
