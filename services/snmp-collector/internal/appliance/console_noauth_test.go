package appliance

import (
	"os"
	"strings"
	"testing"
)

const noAuthApplicationConfig = `
auth:
  enabled: false
  mode: disabled
`

func TestNoAuthSetupOmitsWorkspaceDomains(t *testing.T) {
	path := t.TempDir() + "/application.yaml"
	if err := os.WriteFile(path, []byte(noAuthApplicationConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	fields, googleAuth, err := setupFieldsForApplication(path)
	if err != nil {
		t.Fatalf("setup fields: %v", err)
	}
	if googleAuth {
		t.Fatal("NoAuth configuration selected Google authentication")
	}
	for _, field := range fields {
		if field.key == "workspace_domains" || strings.Contains(field.label, "Google") {
			t.Fatalf("NoAuth setup exposes a Google field: %+v", field)
		}
	}
}

func TestNoAuthSetupPreservesDisabledAuthentication(t *testing.T) {
	layout := NewLayout(t.TempDir())
	if err := layout.Ensure(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.ApplicationYML, []byte(noAuthApplicationConfig), 0o600); err != nil {
		t.Fatal(err)
	}
	identity, err := NewSecretIdentity()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(layout.Identity, []byte(identity.String()+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	model, err := newConsoleModel(layout, screenSetup)
	if err != nil {
		t.Fatal(err)
	}
	model.tlsHostname = "airgapped.equatecloud.tech"
	model.setupValues = []string{"community", "", "10.10.0.0/24", "5", "5", "65"}
	if err := model.completeSNMPSetup(); err != nil {
		t.Fatalf("complete NoAuth setup: %v", err)
	}
	data, err := os.ReadFile(layout.ApplicationYML)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "enabled: false") || !strings.Contains(string(data), "mode: disabled") || strings.Contains(strings.ToLower(string(data)), "google") {
		t.Fatalf("NoAuth configuration changed unexpectedly: %s", data)
	}
}
