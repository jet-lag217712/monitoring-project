package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

func TestParseReconfigureMode(t *testing.T) {
	mode, err := ParseReconfigureMode("sites")
	if err != nil || mode != ReconfigureSites {
		t.Fatalf("mode=%q err=%v", mode, err)
	}
}

func TestPreloadExistingStateFromManifest(t *testing.T) {
	deployDir := t.TempDir()
	manifest := Manifest{
		SiteCount:  2,
		ProbeRate:  5,
		ProbeBurst: 2,
		Sites: []SiteSpec{
			{Index: 1, SiteID: "site-a", CIDR: "10.1.0.0/24"},
			{Index: 2, SiteID: "site-b", CIDR: "10.2.0.0/24"},
		},
	}
	if err := WriteManifest(deployDir, manifest); err != nil {
		t.Fatal(err)
	}
	m := newModel(deployDir, tui.NewTheme(tui.ThemeLight), "test", ProfileAppliance, RunOptions{Reconfigure: ReconfigureSites})
	if len(m.sites) != 2 {
		t.Fatalf("sites=%d", len(m.sites))
	}
	if m.siteIDInputs[0].Value() != "site-a" {
		t.Fatalf("site-a=%q", m.siteIDInputs[0].Value())
	}
}

func TestWriteSiteArtifactsPreservesManagedInventory(t *testing.T) {
	deployDir := t.TempDir()
	template := filepath.Join(deployDir, collectorTemplate)
	if err := os.MkdirAll(filepath.Dir(template), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(template, []byte("site_id: placeholder\ncollector:\n  id: c1\nmqtt:\n  client_id: m1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	spec := SiteSpec{Index: 1, SiteID: "site-a", CollectorID: "c1", MQTTClientID: "m1", CIDR: "10.0.0.0/24"}
	managed := spec.ManagedInventoryPath(deployDir)
	if err := os.MkdirAll(filepath.Dir(managed), 0o750); err != nil {
		t.Fatal(err)
	}
	seed := "devices:\n  - device_id: router-1\n"
	if err := os.WriteFile(managed, []byte(seed), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := WriteSiteArtifacts(deployDir, []SiteSpec{spec}, 5, 2, "SNMP_DISCOVERY_COMMUNITY"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(managed)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != seed {
		t.Fatalf("managed inventory overwritten: %q", string(data))
	}
}
