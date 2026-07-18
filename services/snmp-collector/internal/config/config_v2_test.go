package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"
)

func writeFile(t *testing.T, path, body string, mode os.FileMode) string {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), mode); err != nil {
		t.Fatal(err)
	}
	return path
}

func configWithDevices(devices string) string {
	return "site_id: site-001\ncollector:\n  id: collector-001\ndevices:\n" + devices
}

func TestStrictCommunityReferenceAndSecretFreeConfig(t *testing.T) {
	path := writeTempConfig(t, configWithDevices("  - id: dev-001\n    host: 127.0.0.1\n    community_env: SNMP_COMMUNITY_DEV_001\n"))
	cfg, err := LoadForValidation(path)
	if err != nil {
		t.Fatalf("LoadForValidation: %v", err)
	}
	if cfg.Devices[0].CommunityEnv != "SNMP_COMMUNITY_DEV_001" {
		t.Fatalf("community_env=%q", cfg.Devices[0].CommunityEnv)
	}
	if strings.Contains(string(mustRead(t, path)), "SNMP_SECRET") {
		t.Fatal("config unexpectedly contains a secret value")
	}

	legacy := configWithDevices("  - id: dev-001\n    host: 127.0.0.1\n    community: public\n")
	if _, err := LoadForValidation(writeTempConfig(t, legacy)); err == nil || !strings.Contains(err.Error(), "field community not found") {
		t.Fatalf("legacy community error=%v", err)
	}
}

func TestManagedInventoryStaticPrecedence(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "collector.yaml")
	managedPath := filepath.Join(root, "managed.yaml")
	writeFile(t, configPath, "site_id: site-001\ncollector:\n  id: collector-001\ninventory:\n  managed_path: managed.yaml\ndevices:\n  - id: static-device\n    host: 127.0.0.1\n    community_env: SNMP_COMMUNITY_STATIC\n", 0o600)
	writeFile(t, managedPath, "devices:\n  - id: static-device\n    host: 127.0.0.2\n    community_env: SNMP_COMMUNITY_MANAGED\n  - id: managed-device\n    host: 127.0.0.3\n    community_env: SNMP_COMMUNITY_MANAGED_2\n", 0o600)

	cfg, err := LoadForValidation(configPath)
	if err != nil {
		t.Fatalf("LoadForValidation: %v", err)
	}
	if len(cfg.Devices) != 2 || cfg.Devices[0].Host != "127.0.0.1" || cfg.Devices[1].ID != "managed-device" {
		t.Fatalf("merged devices=%#v", cfg.Devices)
	}
	if cfg.Devices[0].CommunityEnv != "SNMP_COMMUNITY_STATIC" {
		t.Fatalf("static community overridden: %q", cfg.Devices[0].CommunityEnv)
	}
}

func TestManagedOverlayAppliesAllowedFieldsOnly(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "collector.yaml")
	writeFile(t, configPath, "site_id: site-001\ncollector:\n  id: collector-001\nhealth:\n  temperature_warning_c: 65\ndiscovery:\n  allowed_cidrs: [\"192.0.2.0/30\"]\n  community_env: SNMP_DISCOVERY_COMMUNITY\n  max_targets: 4\n  timeout: 2s\n  retries: 0\n  max_workers: 4\n  max_probes_per_second: 5\n  probe_burst: 2\ninventory:\n  managed_path: managed.yaml\ndevices:\n  - id: static-device\n    host: 127.0.0.1\n    community_env: SNMP_COMMUNITY_STATIC\n", 0o600)
	writeFile(t, filepath.Join(root, "managed.yaml"), "health:\n  temperature_warning_c: 70\ndiscovery:\n  max_probes_per_second: 3\n  probe_burst: 1\ndevices:\n  - id: static-device\n    temperature_warning_c: 72\n    upstream_device_ids: []\n    interface_filters:\n      exclude_name_regex: [\"^Lo\"]\n", 0o600)

	cfg, err := LoadForValidation(configPath)
	if err != nil {
		t.Fatalf("LoadForValidation: %v", err)
	}
	if cfg.Health.TemperatureWarningC != 70 {
		t.Fatalf("global temperature=%v", cfg.Health.TemperatureWarningC)
	}
	if cfg.Discovery.MaxProbesPerSecond != 3 || cfg.Discovery.ProbeBurst != 1 {
		t.Fatalf("discovery rate=%v burst=%v", cfg.Discovery.MaxProbesPerSecond, cfg.Discovery.ProbeBurst)
	}
	if got := cfg.Discovery.AllowedCIDRs; len(got) != 1 || got[0] != "192.0.2.0/30" {
		t.Fatalf("allowlist changed: %#v", got)
	}
	if cfg.Devices[0].Host != "127.0.0.1" {
		t.Fatalf("host overridden: %q", cfg.Devices[0].Host)
	}
	if cfg.Devices[0].TemperatureWarningC == nil || *cfg.Devices[0].TemperatureWarningC != 72 {
		t.Fatalf("device temperature overlay=%v", cfg.Devices[0].TemperatureWarningC)
	}
}

func TestManagedDiscoveryCIDROverlay(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "collector.yaml")
	writeFile(t, configPath, "site_id: site-001\ncollector:\n  id: collector-001\ndiscovery:\n  community_env: SNMP_DISCOVERY_COMMUNITY\n  max_targets: 4\n  timeout: 2s\n  retries: 0\n  max_workers: 4\n  max_probes_per_second: 5\n  probe_burst: 2\ninventory:\n  managed_path: managed.yaml\ndevices:\n  - id: static-device\n    host: 127.0.0.1\n    community_env: SNMP_COMMUNITY_STATIC\n", 0o600)
	writeFile(t, filepath.Join(root, "managed.yaml"), "discovery:\n  allowed_cidrs: [\"10.255.0.0/24\"]\n  max_probes_per_second: 8\n  probe_burst: 3\n", 0o600)

	cfg, err := LoadForValidation(configPath)
	if err != nil {
		t.Fatalf("LoadForValidation: %v", err)
	}
	if len(cfg.Discovery.AllowedCIDRs) != 1 || cfg.Discovery.AllowedCIDRs[0] != "10.255.0.0/24" {
		t.Fatalf("cidrs=%#v", cfg.Discovery.AllowedCIDRs)
	}
	if cfg.Discovery.MaxProbesPerSecond != 8 || cfg.Discovery.ProbeBurst != 3 {
		t.Fatalf("rate=%v burst=%v", cfg.Discovery.MaxProbesPerSecond, cfg.Discovery.ProbeBurst)
	}
}

func TestZeroDevicesAllowedWhenDiscoveryConfigured(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "collector.yaml")
	writeFile(t, configPath, "site_id: site-001\ncollector:\n  id: collector-001\ndiscovery:\n  community_env: SNMP_DISCOVERY_COMMUNITY\n  max_targets: 4\n  timeout: 2s\n  retries: 0\n  max_workers: 4\n  max_probes_per_second: 5\n  probe_burst: 2\ninventory:\n  managed_path: managed.yaml\ndevices: []\n", 0o600)
	writeFile(t, filepath.Join(root, "managed.yaml"), "discovery:\n  allowed_cidrs: [\"10.255.0.0/24\"]\n", 0o600)

	cfg, err := LoadForValidation(configPath)
	if err != nil {
		t.Fatalf("LoadForValidation: %v", err)
	}
	if len(cfg.Devices) != 0 {
		t.Fatalf("devices=%#v", cfg.Devices)
	}
	if len(cfg.Discovery.AllowedCIDRs) != 1 {
		t.Fatalf("cidrs=%#v", cfg.Discovery.AllowedCIDRs)
	}
}

func TestIsCrossDeviceRenameError(t *testing.T) {
	if !isCrossDeviceRenameError(syscall.Errno(syscall.EXDEV)) {
		t.Fatal("expected EXDEV to be cross-device")
	}
	if isCrossDeviceRenameError(errors.New("permission denied")) {
		t.Fatal("expected permission error to be distinct")
	}
}

func TestWriteManagedInventoryPreservesPolicy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "managed.yaml")
	temp := 68.0
	rate := 4.0
	burst := 2
	if err := WriteManagedDocument(path, ManagedInventory{
		Health:    ManagedHealthPolicy{TemperatureWarningC: &temp},
		Discovery: ManagedDiscoveryPolicy{MaxProbesPerSecond: &rate, ProbeBurst: &burst},
		Devices:   []DeviceConfig{{ID: "dev-001", Host: "127.0.0.1", Port: 161, CommunityEnv: "SNMP_COMMUNITY_DEV_001", Version: "2c"}},
	}); err != nil {
		t.Fatal(err)
	}
	if err := WriteManagedInventory(path, []DeviceConfig{{ID: "dev-002", Host: "127.0.0.2", Port: 161, CommunityEnv: "SNMP_COMMUNITY_DEV_002", Version: "2c"}}); err != nil {
		t.Fatal(err)
	}
	doc, err := ReadManagedDocument(path)
	if err != nil {
		t.Fatal(err)
	}
	if doc.Health.TemperatureWarningC == nil || *doc.Health.TemperatureWarningC != 68 {
		t.Fatalf("health policy lost: %#v", doc.Health)
	}
	if doc.Discovery.MaxProbesPerSecond == nil || *doc.Discovery.MaxProbesPerSecond != 4 {
		t.Fatalf("discovery policy lost: %#v", doc.Discovery)
	}
	if len(doc.Devices) != 1 || doc.Devices[0].ID != "dev-002" {
		t.Fatalf("devices=%#v", doc.Devices)
	}
}

func TestManagedInventoryCrossSourceDuplicateHostRejected(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "collector.yaml")
	writeFile(t, configPath, "site_id: site-001\ncollector:\n  id: collector-001\ninventory:\n  managed_path: managed.yaml\ndevices:\n  - id: static-device\n    host: 127.0.0.1\n    community_env: SNMP_COMMUNITY_STATIC\n", 0o600)
	writeFile(t, filepath.Join(root, "managed.yaml"), "devices:\n  - id: managed-device\n    host: 127.0.0.1\n    community_env: SNMP_COMMUNITY_MANAGED\n", 0o600)

	_, err := LoadForValidation(configPath)
	if err == nil || !strings.Contains(err.Error(), "duplicate device host") {
		t.Fatalf("error=%v", err)
	}
}

func TestValidateDependenciesTable(t *testing.T) {
	tests := []struct {
		name    string
		devices []DeviceConfig
		want    string
	}{
		{name: "missing", devices: []DeviceConfig{{ID: "a", UpstreamDeviceIDs: []string{"missing"}}}, want: "missing upstream"},
		{name: "self", devices: []DeviceConfig{{ID: "a", UpstreamDeviceIDs: []string{"a"}}}, want: "cannot reference itself"},
		{name: "duplicate", devices: []DeviceConfig{{ID: "a", UpstreamDeviceIDs: []string{"b", "b"}}, {ID: "b"}}, want: "duplicate upstream"},
		{name: "cycle", devices: []DeviceConfig{{ID: "a", UpstreamDeviceIDs: []string{"b"}}, {ID: "b", UpstreamDeviceIDs: []string{"a"}}}, want: "dependency cycle"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDependencies(tt.devices)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v, want %q", err, tt.want)
			}
		})
	}
}

func TestValidateInterfaceFiltersTable(t *testing.T) {
	goodIndex := 2
	tests := []struct {
		name    string
		filters InterfaceFilterConfig
		want    string
	}{
		{name: "valid", filters: InterfaceFilterConfig{ExcludeNameRegex: []string{"^(Lo|Vlan|Null)"}, IncludeTypes: []string{"ethernetCsmacd"}, Rules: []InterfaceFilterRule{{Action: "include", IfIndex: &goodIndex}}}},
		{name: "regex", filters: InterfaceFilterConfig{ExcludeNameRegex: []string{"["}}, want: "invalid regex"},
		{name: "state", filters: InterfaceFilterConfig{IncludeOperStatuses: []string{"broken"}}, want: "unsupported state"},
		{name: "type", filters: InterfaceFilterConfig{IncludeTypes: []string{"bogus"}}, want: "unsupported interface type"},
		{name: "rule", filters: InterfaceFilterConfig{Rules: []InterfaceFilterRule{{Action: "include"}}}, want: "at least one matcher"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateInterfaceFilters(tt.filters, "interface_filters")
			if tt.want == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error=%v, want %q", err, tt.want)
			}
		})
	}
}

func TestValidateDiscovery(t *testing.T) {
	valid := DiscoveryConfig{AllowedCIDRs: []string{"10.0.0.0/24"}, CommunityEnv: "SNMP_DISCOVERY_COMMUNITY", MaxTargets: 100, Timeout: 2 * time.Second, Retries: 1, MaxWorkers: 5, MaxProbesPerSecond: 10, ProbeBurst: 2}
	if err := validateDiscovery(valid); err != nil {
		t.Fatalf("valid discovery: %v", err)
	}
	invalid := valid
	invalid.AllowedCIDRs = []string{"10.0.0.0/99"}
	if err := validateDiscovery(invalid); err == nil {
		t.Fatal("expected invalid CIDR error")
	}
	invalid = valid
	invalid.MaxProbesPerSecond = 0
	if err := validateDiscovery(invalid); err == nil {
		t.Fatal("expected invalid probe rate error")
	}
	invalid = valid
	invalid.CommunityEnv = "not-valid!"
	if err := validateDiscovery(invalid); err == nil || !strings.Contains(err.Error(), "community_env") {
		t.Fatalf("expected invalid community env error, got %v", err)
	}
}

func TestDevicePollingOverrides(t *testing.T) {
	device := DeviceConfig{PollInterval: 15 * time.Second, Timeout: 2 * time.Second, Retries: 4}
	shared := SNMPConfig{Timeout: 5 * time.Second, Retries: 2}
	if got := device.EffectivePollInterval(time.Minute); got != 15*time.Second {
		t.Fatalf("poll interval=%v", got)
	}
	effective := device.EffectiveSNMP(shared)
	if effective.Timeout != 2*time.Second || effective.Retries != 4 {
		t.Fatalf("effective SNMP=%#v", effective)
	}
	if got := (DeviceConfig{}).EffectivePollInterval(time.Minute); got != time.Minute {
		t.Fatalf("inherited poll interval=%v", got)
	}
}

func TestWriteManagedInventoryIsAtomicAndSecretFree(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "managed.yaml")
	devices := []DeviceConfig{{ID: "dev-001", Host: "127.0.0.1", Port: 161, CommunityEnv: "SNMP_COMMUNITY_DEV_001", Version: "2c"}}
	if err := WriteManagedInventory(path, devices); err != nil {
		t.Fatalf("WriteManagedInventory: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("permissions=%o, want 600", info.Mode().Perm())
	}
	content := string(mustRead(t, path))
	if !strings.Contains(content, "community_env: SNMP_COMMUNITY_DEV_001") || strings.Contains(content, "community:") {
		t.Fatalf("unexpected managed contents: %s", content)
	}
}

func TestWriteManagedInventoryCleansTempOnFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "managed.yaml")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	devices := []DeviceConfig{{ID: "dev-001", Host: "127.0.0.1", CommunityEnv: "SNMP_COMMUNITY_DEV_001", Version: "2c"}}
	if err := WriteManagedInventory(target, devices); err == nil {
		t.Fatal("expected replacement failure when target is a directory")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "managed.yaml" {
		t.Fatalf("temporary files remain: %#v", entries)
	}
}

func TestManagerReloadRetainsPreviousSnapshotOnFailure(t *testing.T) {
	path := writeTempConfig(t, configWithDevices("  - id: dev-001\n    host: 127.0.0.1\n    community_env: SNMP_COMMUNITY_DEV_001\n"))
	initial, err := LoadForValidation(path)
	if err != nil {
		t.Fatalf("initial load: %v", err)
	}
	manager, err := NewManager(path, initial)
	if err != nil {
		t.Fatal(err)
	}

	updated := configWithDevices("  - id: dev-001\n    host: 127.0.0.2\n    community_env: SNMP_COMMUNITY_DEV_001\n")
	writeFile(t, path, updated, 0o600)
	if err := manager.Reload(); err != nil {
		t.Fatalf("valid reload: %v", err)
	}
	if got := manager.Current().Devices[0].Host; got != "127.0.0.2" {
		t.Fatalf("host=%q", got)
	}

	invalid := configWithDevices("  - id: dev-001\n    host: 127.0.0.3\n    community: public\n")
	writeFile(t, path, invalid, 0o600)
	if err := manager.Reload(); err == nil {
		t.Fatal("expected invalid reload error")
	}
	if got := manager.Current().Devices[0].Host; got != "127.0.0.2" {
		t.Fatalf("snapshot changed after invalid reload: host=%q", got)
	}
}

func TestValidateReloadRejectsStartupOnlyChanges(t *testing.T) {
	base := &Config{
		SiteID:    "site-001",
		Collector: CollectorConfig{ID: "collector-001"},
		Admin:     AdminConfig{Listen: ":9090"},
		Publisher: PublisherConfig{Mode: "stdout", Timeout: time.Second},
		Buffer:    BufferConfig{Path: "buffer.db"},
	}
	next := *base
	next.SiteID = "site-002"
	if err := validateReload(base, &next); err == nil || !strings.Contains(err.Error(), "site_id") {
		t.Fatalf("site change error=%v", err)
	}
	next = *base
	next.Publisher.Mode = "mqtt"
	if err := validateReload(base, &next); err == nil || !strings.Contains(err.Error(), "publisher") {
		t.Fatalf("publisher change error=%v", err)
	}
}

func TestMQTTValidationSeparatesSecretPresence(t *testing.T) {
	path := writeTempConfig(t, "site_id: site-001\ncollector:\n  id: collector-001\npublisher:\n  mode: mqtt\nmqtt:\n  broker: tls://127.0.0.1:8883\n  username: collector\n  password_env: MQTT_PASSWORD\n  tls:\n    ca_file: /tmp/ca.crt\ndevices:\n  - id: dev-001\n    host: 127.0.0.1\n    community_env: SNMP_COMMUNITY_DEV_001\n")
	t.Setenv("MQTT_PASSWORD", "")
	if _, err := LoadForValidation(path); err != nil {
		t.Fatalf("validation should not require MQTT secret: %v", err)
	}
	if _, err := Load(path); err == nil || !strings.Contains(err.Error(), "MQTT_PASSWORD") {
		t.Fatalf("runtime load error=%v", err)
	}
}

func mustRead(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestEffectiveTemperatureWarningCAndPolicyRevision(t *testing.T) {
	t.Parallel()

	override := 70.0
	cfg := &Config{
		Health: HealthConfig{TemperatureWarningC: 65, FailureThreshold: 2},
		Devices: []DeviceConfig{
			{ID: "b-device", TemperatureWarningC: &override},
			{ID: "a-device"},
		},
	}
	if got := cfg.Devices[1].EffectiveTemperatureWarningC(cfg.Health.TemperatureWarningC); got != 65 {
		t.Fatalf("default override=%v", got)
	}
	if got := cfg.Devices[0].EffectiveTemperatureWarningC(cfg.Health.TemperatureWarningC); got != 70 {
		t.Fatalf("device override=%v", got)
	}
	rev1 := TemperaturePolicyRevision(cfg)
	rev2 := TemperaturePolicyRevision(cfg)
	if rev1 == "" || rev1 != rev2 {
		t.Fatalf("revision unstable: %q %q", rev1, rev2)
	}
	cfg.Health.TemperatureWarningC = 66
	if TemperaturePolicyRevision(cfg) == rev1 {
		t.Fatal("expected revision to change with global threshold")
	}
}

func TestValidatePendingDependencyMutationDetectsSequentialCycle(t *testing.T) {
	root := t.TempDir()
	configPath := filepath.Join(root, "collector.yaml")
	managedPath := filepath.Join(root, "managed.yaml")
	writeFile(t, configPath, "site_id: site-001\ncollector:\n  id: collector-001\ninventory:\n  managed_path: managed.yaml\ndevices:\n  - id: dev-001\n    host: 127.0.0.1\n    community_env: SNMP_COMMUNITY_DEV_001\n  - id: dev-002\n    host: 127.0.0.2\n    community_env: SNMP_COMMUNITY_DEV_002\n", 0o600)
	writeFile(t, managedPath, "devices:\n  - id: dev-001\n    upstream_device_ids:\n      - dev-002\n", 0o600)

	cfg, err := LoadForValidation(configPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if err := cfg.ValidatePendingDependencyMutation("dev-002", []string{"dev-001"}); err == nil {
		t.Fatal("expected cycle detection")
	}
	if err := cfg.ValidatePendingDependencyMutation("dev-001", []string{"dev-002"}); err != nil {
		t.Fatalf("valid upstream mutation: %v", err)
	}
}
