package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

func TestApplianceEnvInputsOnlySNMP(t *testing.T) {
	m := newModel(t.TempDir(), tui.NewTheme(tui.ThemeLight), "test", ProfileAppliance)
	if len(m.envInputs) != 2 {
		t.Fatalf("inputs=%d", len(m.envInputs))
	}
	if m.envInputs[0].Placeholder != "SNMP community" {
		t.Fatalf("placeholder=%q", m.envInputs[0].Placeholder)
	}
}

func TestSharedEnvValuesApplianceMergesMQTT(t *testing.T) {
	dir := t.TempDir()
	composeEnv := filepath.Join(dir, "compose.env")
	if err := os.WriteFile(composeEnv, []byte(
		"MQTT_BROKER=tls://mosquitto:8883\nMQTT_COLLECTOR_PASSWORD=collector-secret\nSNMP_COMMUNITY=CHANGE_ME\n",
	), 0o600); err != nil {
		t.Fatal(err)
	}
	orig := applianceComposeEnv
	applianceComposeEnv = composeEnv
	t.Cleanup(func() { applianceComposeEnv = orig })

	m := newModel(t.TempDir(), tui.NewTheme(tui.ThemeLight), "test", ProfileAppliance)
	m.envInputs[0].SetValue("lab-read")
	m.envInputs[1].SetValue("lab-discover")

	values, err := m.sharedEnvValues()
	if err != nil {
		t.Fatal(err)
	}
	if values["MQTT_BROKER"] != "tls://mosquitto:8883" {
		t.Fatalf("broker=%q", values["MQTT_BROKER"])
	}
	if values["MQTT_PASSWORD"] != "collector-secret" {
		t.Fatalf("password=%q", values["MQTT_PASSWORD"])
	}
	if values["SNMP_COMMUNITY"] != "lab-read" {
		t.Fatalf("community=%q", values["SNMP_COMMUNITY"])
	}
	if values["SNMP_DISCOVERY_COMMUNITY"] != "lab-discover" {
		t.Fatalf("discovery=%q", values["SNMP_DISCOVERY_COMMUNITY"])
	}
}
