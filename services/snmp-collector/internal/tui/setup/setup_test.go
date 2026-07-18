package setup

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/control"
	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

func TestLoadEnvFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := writeEnvFile(path, map[string]string{
		"SNMP_COMMUNITY": "lab-community",
		"MQTT_BROKER":    "tls://127.0.0.1:8883",
	}); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SNMP_COMMUNITY", "")
	if err := loadEnvFile(path); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv("SNMP_COMMUNITY"); got != "lab-community" {
		t.Fatalf("SNMP_COMMUNITY=%q", got)
	}
}

func TestWriteEnvFile(t *testing.T) {
	path := t.TempDir() + "/.env"
	if err := writeEnvFile(path, map[string]string{
		"MQTT_BROKER":    "tls://127.0.0.1:8883",
		"MQTT_PASSWORD":  "secret",
		"SNMP_COMMUNITY": "public",
	}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "MQTT_BROKER=") {
		t.Fatalf("missing broker: %s", data)
	}
	if strings.Contains(string(data), "COLLECTOR_ADMIN_PORT") {
		t.Fatalf("should not write per-site admin port to shared env: %s", data)
	}
}

func TestNewModelInitialStep(t *testing.T) {
	m := newModel(t.TempDir(), tui.NewTheme(tui.ThemeLight))
	if m.step != stepEnv {
		t.Fatalf("step=%v", m.step)
	}
	if m.siteCountInput.Value() != "4" {
		t.Fatalf("site count=%q", m.siteCountInput.Value())
	}
	if len(m.cidrInputs) != 4 {
		t.Fatalf("cidr inputs=%d", len(m.cidrInputs))
	}
	if len(m.siteIDInputs) != 4 {
		t.Fatalf("site id inputs=%d", len(m.siteIDInputs))
	}
}

func TestBuildSiteSpecsFourSites(t *testing.T) {
	siteIDs := []string{"site-001", "site-002", "site-003", "site-004"}
	cidrs := []string{"10.255.0.0/24", "10.255.1.0/24", "10.255.2.0/24", "10.255.3.0/24"}
	specs, err := BuildSiteSpecs(4, siteIDs, cidrs)
	if err != nil {
		t.Fatal(err)
	}
	if len(specs) != 4 {
		t.Fatalf("len=%d", len(specs))
	}
	seen := map[string]struct{}{}
	for i, spec := range specs {
		if spec.SiteID != siteIDs[i] {
			t.Fatalf("site_id=%q", spec.SiteID)
		}
		if spec.CollectorID != collectorIDForSiteID(siteIDs[i]) {
			t.Fatalf("collector_id=%q", spec.CollectorID)
		}
		if spec.MQTTClientID != mqttClientIDForSiteID(siteIDs[i]) {
			t.Fatalf("client_id=%q", spec.MQTTClientID)
		}
		if spec.AdminPort != baseAdminPort+i {
			t.Fatalf("admin_port=%d", spec.AdminPort)
		}
		if spec.ServiceName != serviceNameForSiteID(spec.SiteID) {
			t.Fatalf("service=%q", spec.ServiceName)
		}
		if _, ok := seen[spec.CIDR]; ok {
			t.Fatalf("duplicate cidr %q", spec.CIDR)
		}
		seen[spec.CIDR] = struct{}{}
	}
}

func TestBuildSiteSpecsRejectsInvalidInput(t *testing.T) {
	if _, err := BuildSiteSpecs(0, nil, nil); err == nil {
		t.Fatal("expected error for zero count")
	}
	if _, err := BuildSiteSpecs(2, []string{"site-a"}, []string{"10.0.0.0/24"}); err == nil {
		t.Fatal("expected site id count mismatch error")
	}
	if _, err := BuildSiteSpecs(2, []string{"site-a", "site-b"}, []string{"10.0.0.0/24"}); err == nil {
		t.Fatal("expected cidr count mismatch error")
	}
	if _, err := BuildSiteSpecs(2, []string{"site-a", "site-b"}, []string{"bad", "10.0.1.0/24"}); err == nil {
		t.Fatal("expected invalid cidr error")
	}
	if _, err := BuildSiteSpecs(2, []string{"site-a", "site-b"}, []string{"10.0.0.0/24", "10.0.0.0/24"}); err == nil {
		t.Fatal("expected duplicate cidr error")
	}
	if _, err := BuildSiteSpecs(2, []string{"Site A", "site-b"}, []string{"10.0.0.0/24", "10.0.1.0/24"}); err == nil {
		t.Fatal("expected invalid site id error")
	}
	if _, err := BuildSiteSpecs(2, []string{"-bad", "site-b"}, []string{"10.0.0.0/24", "10.0.1.0/24"}); err == nil {
		t.Fatal("expected invalid site id error")
	}
	if _, err := BuildSiteSpecs(2, []string{"site-a", "site-a"}, []string{"10.0.0.0/24", "10.0.1.0/24"}); err == nil {
		t.Fatal("expected duplicate site id error")
	}
}

func TestBuildSiteSpecsCustomIDs(t *testing.T) {
	siteIDs := []string{"do-core", "site-a-mdf"}
	cidrs := []string{"10.255.0.0/24", "10.255.1.0/24"}
	specs, err := BuildSiteSpecs(2, siteIDs, cidrs)
	if err != nil {
		t.Fatal(err)
	}
	if specs[0].SiteID != "do-core" {
		t.Fatalf("site_id=%q", specs[0].SiteID)
	}
	if specs[0].ServiceName != "snmp-collector-do-core" {
		t.Fatalf("service=%q", specs[0].ServiceName)
	}
	if specs[1].CollectorID != "collector-development-vxrail-site-a-mdf" {
		t.Fatalf("collector_id=%q", specs[1].CollectorID)
	}
}

func TestGenerateComposeFourSites(t *testing.T) {
	deployDir := t.TempDir()
	template := filepath.Join(deployDir, collectorTemplate)
	if err := os.MkdirAll(filepath.Dir(template), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(template, []byte("site_id: site-001\ncollector:\n  id: collector-development-vxrail\nmqtt:\n  client_id: development-vxrail-collector\n  broker: tls://127.0.0.1:8883\n  username: collector\n  password_env: MQTT_PASSWORD\n  tls:\n    ca_file: /certs/ca.crt\ninventory:\n  managed_path: /var/lib/snmp-collector/managed/managed-inventory.yaml\ndevices: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	siteIDs := []string{"site-001", "site-002", "site-003", "site-004"}
	cidrs := []string{"10.255.0.0/24", "10.255.1.0/24", "10.255.2.0/24", "10.255.3.0/24"}
	specs, err := BuildSiteSpecs(4, siteIDs, cidrs)
	if err != nil {
		t.Fatal(err)
	}
	if err := persistMultiSiteArtifacts(deployDir, specs, 5, 2); err != nil {
		t.Fatal(err)
	}
	composeData, err := os.ReadFile(generatedComposePath(deployDir))
	if err != nil {
		t.Fatal(err)
	}
	text := string(composeData)
	serviceCount := strings.Count(text, "container_name: snmp-collector-site-")
	if serviceCount != 4 {
		t.Fatalf("service count=%d compose:\n%s", serviceCount, text)
	}
	parts := strings.Split(text, "\nvolumes:\n")
	if len(parts) != 2 {
		t.Fatalf("missing volumes section:\n%s", text)
	}
	volumeCount := strings.Count(parts[1], "collector-state-site-")
	if volumeCount != 4 {
		t.Fatalf("volume count=%d", volumeCount)
	}
	for _, spec := range specs {
		if !strings.Contains(text, spec.ServiceName) {
			t.Fatalf("missing service %s", spec.ServiceName)
		}
		cfgPath := spec.ConfigPath(deployDir)
		cfgData, err := os.ReadFile(cfgPath)
		if err != nil {
			t.Fatalf("config for %s: %v", spec.SiteID, err)
		}
		if !strings.Contains(string(cfgData), spec.SiteID) {
			t.Fatalf("config missing site_id for %s", spec.SiteID)
		}
		if !strings.Contains(string(cfgData), spec.MQTTClientID) {
			t.Fatalf("config missing client_id for %s", spec.SiteID)
		}
		managedData, err := os.ReadFile(spec.ManagedInventoryPath(deployDir))
		if err != nil {
			t.Fatalf("managed for %s: %v", spec.SiteID, err)
		}
		if !strings.Contains(string(managedData), spec.CIDR) {
			t.Fatalf("managed missing cidr for %s", spec.SiteID)
		}
	}
	manifest, err := LoadManifest(deployDir)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SiteCount != 4 || len(manifest.Sites) != 4 {
		t.Fatalf("manifest site_count=%d sites=%d", manifest.SiteCount, len(manifest.Sites))
	}
	if _, err := exec.LookPath("docker"); err == nil {
		cmd := exec.Command("docker", "compose", "-f", filepath.Join(deployDir, "docker-compose.yml"), "-f", generatedComposePath(deployDir), "config")
		cmd.Env = append(os.Environ(),
			"MQTT_BROKER=tls://example.com:8883",
			"MQTT_PASSWORD=placeholder",
			"SNMP_COMMUNITY=placeholder",
			"SNMP_DISCOVERY_COMMUNITY=placeholder",
		)
		baseCompose := filepath.Join(deployDir, "docker-compose.yml")
		if err := os.WriteFile(baseCompose, []byte("name: test-vxrail\nservices: {}\nvolumes: {}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("docker compose config: %v\n%s", err, out)
		}
	}
}

func TestApplyThresholdToAllSitesRequiresEverySite(t *testing.T) {
	deployDir := t.TempDir()
	specs := []SiteSpec{
		{Index: 1, SiteID: "site-001", CollectorID: "collector-development-vxrail-site-001", MQTTClientID: "development-vxrail-collector-site-001", ServiceName: "snmp-collector-site-001", CIDR: "10.255.0.0/24", AdminPort: 9090},
	}
	manifest := Manifest{SiteCount: 1, BaseAdminPort: baseAdminPort, ProbeRate: 5, ProbeBurst: 2, Sites: specs}
	if err := WriteManifest(deployDir, manifest); err != nil {
		t.Fatal(err)
	}
	for _, spec := range specs {
		if err := applyThresholdToSite(spec, deployDir, 65); err == nil {
			t.Fatal("expected error without running collector")
		}
	}
}

func TestRequestTimeoutDiscoveryScan(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	got := requestTimeout(ctx, "discovery.scan.start")
	if got < 9*time.Minute {
		t.Fatalf("timeout=%s", got)
	}
}

func TestRequestTimeoutDefault(t *testing.T) {
	got := requestTimeout(context.Background(), "status.summary")
	if got != control.DefaultRequestTimeout {
		t.Fatalf("timeout=%s", got)
	}
}

func TestIsDiscoveryRetryable(t *testing.T) {
	if !isDiscoveryRetryable(errors.New("rpc: read unix @->/run/snmp-collector/control.sock: i/o timeout")) {
		t.Fatal("expected retryable")
	}
	if isDiscoveryRetryable(errors.New("validation failed")) {
		t.Fatal("expected not retryable")
	}
}

type fakeControl struct {
	methods []string
}

func (f *fakeControl) Call(ctx context.Context, id, method string, params map[string]any) (control.Response, error) {
	f.methods = append(f.methods, method)
	switch method {
	case "discovery.scan.start":
		return control.Response{OK: true}, nil
	case "discovery.candidates.list":
		return control.Response{OK: true, Result: map[string]any{"candidates": []any{}}}, nil
	default:
		return control.Response{OK: true}, nil
	}
}

func TestReviewSiteNoCandidates(t *testing.T) {
	deployDir := t.TempDir()
	spec := SiteSpec{SiteID: "site-001", ServiceName: "snmp-collector-site-001", AdminPort: 9090}
	orig := newDeployControl
	t.Cleanup(func() {
		_ = orig
	})
	// reviewSite uses newDeployControl directly; smoke-test candidate-free path via helper logic.
	fake := &fakeControl{}
	candidates, err := runDiscoveryScan(fake)
	if err != nil {
		t.Fatal(err)
	}
	if len(candidates) != 0 {
		t.Fatalf("candidates=%d", len(candidates))
	}
	_ = deployDir
	_ = spec
}
