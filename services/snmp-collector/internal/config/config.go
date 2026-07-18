package config

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	defaultPollInterval       = 60 * time.Second
	defaultMaxWorkers         = 10
	defaultSNMPTimeout        = 5 * time.Second
	defaultSNMPRetries        = 2
	defaultPublisherTimeout   = 10 * time.Second
	defaultHeartbeatInterval  = 60 * time.Second
	defaultTemperatureWarning = 65.0
	defaultFailureThreshold   = 2

	maxPollInterval       = 24 * time.Hour
	maxSNMPTimeout        = 5 * time.Minute
	maxRetries            = 10
	maxWorkers            = 1024
	maxDiscoveryTargets   = 1_000_000
	maxDiscoveryRate      = 100_000.0
	maxDiscoveryBurst     = 100_000
	maxTemperatureCelsius = 250.0
	minTemperatureCelsius = -100.0
)

var (
	identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	envNamePattern    = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
)

// Config is the collector runtime configuration.
type Config struct {
	SiteID       string        `yaml:"site_id"`
	PollInterval time.Duration `yaml:"poll_interval"`
	MaxWorkers   int           `yaml:"max_workers"`

	Collector CollectorConfig `yaml:"collector"`
	Inventory InventoryConfig `yaml:"inventory"`
	Health    HealthConfig    `yaml:"health"`
	Discovery DiscoveryConfig `yaml:"discovery"`
	Admin     AdminConfig     `yaml:"admin"`
	SNMP      SNMPConfig      `yaml:"snmp"`
	Publisher PublisherConfig `yaml:"publisher"`
	Buffer    BufferConfig    `yaml:"buffer"`
	MQTT      MQTTConfig      `yaml:"mqtt"`

	Devices []DeviceConfig `yaml:"devices"`

	configPath string
}

// CollectorConfig contains stable collector identity and heartbeat settings.
type CollectorConfig struct {
	ID                string        `yaml:"id"`
	HeartbeatInterval time.Duration `yaml:"heartbeat_interval"`
}

// InventoryConfig controls the optional TUI-managed inventory source.
type InventoryConfig struct {
	ManagedPath string `yaml:"managed_path"`
}

// HealthConfig contains device health evaluation policies.
type HealthConfig struct {
	TemperatureWarningC float64 `yaml:"temperature_warning_c"`
	FailureThreshold    int     `yaml:"failure_threshold"`
}

// DiscoveryConfig contains the isolated operator-invoked discovery policy.
type DiscoveryConfig struct {
	AllowedCIDRs       []string      `yaml:"allowed_cidrs"`
	CommunityEnv       string        `yaml:"community_env"`
	MaxTargets         int           `yaml:"max_targets"`
	Timeout            time.Duration `yaml:"timeout"`
	Retries            int           `yaml:"retries"`
	MaxWorkers         int           `yaml:"max_workers"`
	MaxProbesPerSecond float64       `yaml:"max_probes_per_second"`
	ProbeBurst         int           `yaml:"probe_burst"`
}

// PublisherConfig selects the publish backend and poller publish timeout.
type PublisherConfig struct {
	Mode             string        `yaml:"mode"` // stdout | mqtt
	Timeout          time.Duration `yaml:"timeout"`
	TelemetryVersion string        `yaml:"telemetry_version"` // v1 | v2 | both
}

// BufferConfig controls the durable local SQLite buffer (mqtt mode).
type BufferConfig struct {
	Path          string        `yaml:"path"`
	MaxEntries    int           `yaml:"max_entries"`
	BusyTimeoutMS int           `yaml:"busy_timeout_ms"`
	BatchSize     int           `yaml:"batch_size"`
	IdleBackoff   time.Duration `yaml:"idle_backoff"`
}

// MQTTConfig controls the outbound MQTT/TLS client.
type MQTTConfig struct {
	Broker      string          `yaml:"broker"`
	ClientID    string          `yaml:"client_id"`
	Username    string          `yaml:"username"`
	PasswordEnv string          `yaml:"password_env"`
	QoS         byte            `yaml:"qos"`
	TLS         MQTTTLSConfig   `yaml:"tls"`
	Reconnect   ReconnectConfig `yaml:"reconnect"`
}

// MQTTTLSConfig holds TLS material paths for the MQTT client.
type MQTTTLSConfig struct {
	CAFile   string `yaml:"ca_file"`
	CertFile string `yaml:"cert_file"`
	KeyFile  string `yaml:"key_file"`
}

// ReconnectConfig controls MQTT reconnect backoff.
type ReconnectConfig struct {
	Initial time.Duration `yaml:"initial"`
	Max     time.Duration `yaml:"max"`
}

// AdminConfig controls the admin HTTP server (metrics/health) and local control socket.
type AdminConfig struct {
	Listen        string `yaml:"listen"`
	ControlSocket string `yaml:"control_socket"`
}

// SNMPConfig holds shared SNMP client defaults.
type SNMPConfig struct {
	Timeout time.Duration `yaml:"timeout"`
	Retries int           `yaml:"retries"`
}

// DeviceConfig describes a single SNMP poll target.
type DeviceConfig struct {
	ID           string `yaml:"id"`
	Host         string `yaml:"host"`
	Port         uint16 `yaml:"port"`
	CommunityEnv string `yaml:"community_env"`
	Version      string `yaml:"version"`
	Vendor       string `yaml:"vendor"`

	PollInterval        time.Duration         `yaml:"poll_interval"`
	Timeout             time.Duration         `yaml:"timeout"`
	Retries             int                   `yaml:"retries"`
	TemperatureWarningC *float64              `yaml:"temperature_warning_c"`
	Role                string                `yaml:"role"`
	UpstreamDeviceIDs   []string              `yaml:"upstream_device_ids"`
	InterfaceFilters    InterfaceFilterConfig `yaml:"interface_filters"`
}

// InterfaceFilterConfig contains ordered rules and roadmap shorthand filters.
type InterfaceFilterConfig struct {
	Rules []InterfaceFilterRule `yaml:"rules"`

	IncludeIfIndexes     []int    `yaml:"include_if_indexes"`
	ExcludeIfIndexes     []int    `yaml:"exclude_if_indexes"`
	IncludeNameRegex     []string `yaml:"include_name_regex"`
	ExcludeNameRegex     []string `yaml:"exclude_name_regex"`
	IncludeAliasRegex    []string `yaml:"include_alias_regex"`
	ExcludeAliasRegex    []string `yaml:"exclude_alias_regex"`
	IncludeTypes         []string `yaml:"include_types"`
	ExcludeTypes         []string `yaml:"exclude_types"`
	IncludeAdminStatuses []string `yaml:"include_admin_statuses"`
	ExcludeAdminStatuses []string `yaml:"exclude_admin_statuses"`
	IncludeOperStatuses  []string `yaml:"include_oper_statuses"`
	ExcludeOperStatuses  []string `yaml:"exclude_oper_statuses"`
}

// InterfaceFilterRule is an ordered include/exclude matcher.
type InterfaceFilterRule struct {
	Action      string `yaml:"action"`
	IfIndex     *int   `yaml:"if_index"`
	NameRegex   string `yaml:"name_regex"`
	AliasRegex  string `yaml:"alias_regex"`
	IfType      string `yaml:"if_type"`
	AdminStatus string `yaml:"admin_status"`
	OperStatus  string `yaml:"oper_status"`
}

// ManagedHealthPolicy is the optional global temperature override in the managed file.
type ManagedHealthPolicy struct {
	TemperatureWarningC *float64 `yaml:"temperature_warning_c"`
}

// ManagedDiscoveryPolicy is the optional discovery overlay in the managed file.
// When set, fields override the static discovery policy for operator workflows.
type ManagedDiscoveryPolicy struct {
	AllowedCIDRs       []string `yaml:"allowed_cidrs,omitempty"`
	CommunityEnv       string   `yaml:"community_env,omitempty"`
	MaxTargets         *int     `yaml:"max_targets,omitempty"`
	MaxWorkers         *int     `yaml:"max_workers,omitempty"`
	Retries            *int     `yaml:"retries,omitempty"`
	Timeout            string   `yaml:"timeout,omitempty"`
	MaxProbesPerSecond *float64 `yaml:"max_probes_per_second,omitempty"`
	ProbeBurst         *int     `yaml:"probe_burst,omitempty"`
}

// ManagedInventory is the on-disk shape written by local operator tooling.
// It may contain policy overlays and device overlays/appends. Runtime never
// writes back into this document except through explicit operator mutations.
type ManagedInventory struct {
	Health    ManagedHealthPolicy    `yaml:"health"`
	Discovery ManagedDiscoveryPolicy `yaml:"discovery"`
	Devices   []DeviceConfig         `yaml:"devices"`
}

// Load reads and validates a collector config file for daemon startup.
func Load(path string) (*Config, error) {
	return load(path, true)
}

// LoadForValidation reads and validates a config without requiring secret values.
func LoadForValidation(path string) (*Config, error) {
	return load(path, false)
}

func load(path string, requireRuntimeSecrets bool) (*Config, error) {
	if strings.TrimSpace(path) == "" {
		return nil, errors.New("config path is required")
	}
	if err := validateSourceFile(path, false); err != nil {
		return nil, fmt.Errorf("config file: %w", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := decodeYAML(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.configPath, err = filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve config path: %w", err)
	}
	cfg.applyDefaults()
	cfg.applyEnvOverrides()

	staticDevices := append([]DeviceConfig(nil), cfg.Devices...)
	if err := validateDeviceSource(staticDevices, "devices"); err != nil {
		return nil, err
	}

	managed, err := loadManagedDocument(cfg.Inventory.ManagedPath, cfg.configPath)
	if err != nil {
		return nil, err
	}
	if err := validateManagedDocument(managed, staticDevices); err != nil {
		return nil, err
	}
	staticIDs := make(map[string]struct{}, len(staticDevices))
	for _, device := range staticDevices {
		staticIDs[device.ID] = struct{}{}
	}
	for i := range managed.Devices {
		if _, exists := staticIDs[managed.Devices[i].ID]; exists {
			continue
		}
		applyDeviceDefaults(managed.Devices[i : i+1])
	}

	cfg.Devices = mergeInventories(staticDevices, managed.Devices)
	applyManagedPolicy(&cfg, managed)
	if err := cfg.validate(false); err != nil {
		return nil, err
	}
	if requireRuntimeSecrets {
		if err := cfg.validateRuntimeSecrets(); err != nil {
			return nil, err
		}
	}
	return &cfg, nil
}

func decodeYAML(data []byte, dst any) error {
	decoder := yaml.NewDecoder(strings.NewReader(string(data)))
	decoder.KnownFields(true)
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err == nil {
		return errors.New("multiple YAML documents are not supported")
	} else if !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

func loadManagedDocument(path, configPath string) (ManagedInventory, error) {
	var empty ManagedInventory
	if strings.TrimSpace(path) == "" {
		return empty, nil
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(filepath.Dir(configPath), path)
	}
	path = filepath.Clean(path)

	if err := validateSourceFile(path, true); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return empty, nil
		}
		return empty, fmt.Errorf("managed inventory: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return empty, fmt.Errorf("read managed inventory: %w", err)
	}
	var inventory ManagedInventory
	if err := decodeYAML(data, &inventory); err != nil {
		return empty, fmt.Errorf("parse managed inventory: %w", err)
	}
	return inventory, nil
}

func validateManagedDocument(managed ManagedInventory, staticDevices []DeviceConfig) error {
	if managed.Health.TemperatureWarningC != nil {
		v := *managed.Health.TemperatureWarningC
		if v < minTemperatureCelsius || v > maxTemperatureCelsius {
			return fmt.Errorf("managed health.temperature_warning_c must be between %.0f and %.0f", minTemperatureCelsius, maxTemperatureCelsius)
		}
	}
	if managed.Discovery.MaxProbesPerSecond != nil && *managed.Discovery.MaxProbesPerSecond <= 0 {
		return fmt.Errorf("managed discovery.max_probes_per_second must be positive")
	}
	if managed.Discovery.ProbeBurst != nil && *managed.Discovery.ProbeBurst <= 0 {
		return fmt.Errorf("managed discovery.probe_burst must be positive")
	}
	if err := validateManagedDiscoveryPolicy(managed.Discovery); err != nil {
		return err
	}

	staticIDs := make(map[string]struct{}, len(staticDevices))
	for _, device := range staticDevices {
		staticIDs[device.ID] = struct{}{}
	}

	overlays := make([]DeviceConfig, 0)
	appends := make([]DeviceConfig, 0)
	for i, device := range managed.Devices {
		if _, exists := staticIDs[device.ID]; exists {
			if err := validateManagedOverlay(device, fmt.Sprintf("managed devices[%d]", i)); err != nil {
				return err
			}
			overlays = append(overlays, device)
			continue
		}
		appends = append(appends, device)
	}
	applyDeviceDefaults(appends)
	if err := validateDeviceSource(appends, "managed devices"); err != nil {
		return err
	}
	// Overlays may omit host; uniqueness among overlay IDs is still required.
	seenOverlayIDs := make(map[string]struct{}, len(overlays))
	for _, device := range overlays {
		if _, ok := seenOverlayIDs[device.ID]; ok {
			return fmt.Errorf("duplicate device id %q in managed devices", device.ID)
		}
		seenOverlayIDs[device.ID] = struct{}{}
	}
	return nil
}

func validateManagedOverlay(device DeviceConfig, prefix string) error {
	if strings.TrimSpace(device.ID) == "" {
		return fmt.Errorf("%s.id is required", prefix)
	}
	if !identifierPattern.MatchString(device.ID) {
		return fmt.Errorf("%s.id has invalid format", prefix)
	}
	if device.TemperatureWarningC != nil {
		v := *device.TemperatureWarningC
		if v < minTemperatureCelsius || v > maxTemperatureCelsius {
			return fmt.Errorf("%s.temperature_warning_c must be between %.0f and %.0f", prefix, minTemperatureCelsius, maxTemperatureCelsius)
		}
	}
	if err := validateInterfaceFilters(device.InterfaceFilters, prefix+".interface_filters"); err != nil {
		return err
	}
	return nil
}

func applyManagedPolicy(cfg *Config, managed ManagedInventory) {
	if managed.Health.TemperatureWarningC != nil {
		cfg.Health.TemperatureWarningC = *managed.Health.TemperatureWarningC
	}
	if len(managed.Discovery.AllowedCIDRs) > 0 {
		cfg.Discovery.AllowedCIDRs = append([]string(nil), managed.Discovery.AllowedCIDRs...)
	}
	if strings.TrimSpace(managed.Discovery.CommunityEnv) != "" {
		cfg.Discovery.CommunityEnv = strings.TrimSpace(managed.Discovery.CommunityEnv)
	}
	if managed.Discovery.MaxTargets != nil {
		cfg.Discovery.MaxTargets = *managed.Discovery.MaxTargets
	}
	if managed.Discovery.MaxWorkers != nil {
		cfg.Discovery.MaxWorkers = *managed.Discovery.MaxWorkers
	}
	if managed.Discovery.Retries != nil {
		cfg.Discovery.Retries = *managed.Discovery.Retries
	}
	if strings.TrimSpace(managed.Discovery.Timeout) != "" {
		if d, err := time.ParseDuration(strings.TrimSpace(managed.Discovery.Timeout)); err == nil {
			cfg.Discovery.Timeout = d
		}
	}
	if managed.Discovery.MaxProbesPerSecond != nil {
		cfg.Discovery.MaxProbesPerSecond = *managed.Discovery.MaxProbesPerSecond
	}
	if managed.Discovery.ProbeBurst != nil {
		cfg.Discovery.ProbeBurst = *managed.Discovery.ProbeBurst
	}
	applyDiscoveryDefaults(&cfg.Discovery)
}

func validateManagedDiscoveryPolicy(policy ManagedDiscoveryPolicy) error {
	for i, cidr := range policy.AllowedCIDRs {
		if _, _, err := net.ParseCIDR(strings.TrimSpace(cidr)); err != nil {
			return fmt.Errorf("managed discovery.allowed_cidrs[%d] is invalid: %w", i, err)
		}
	}
	if policy.CommunityEnv != "" {
		if err := validateEnvName(policy.CommunityEnv, "managed discovery.community_env"); err != nil {
			return err
		}
	}
	if policy.MaxTargets != nil && *policy.MaxTargets <= 0 {
		return fmt.Errorf("managed discovery.max_targets must be positive")
	}
	if policy.MaxWorkers != nil && *policy.MaxWorkers <= 0 {
		return fmt.Errorf("managed discovery.max_workers must be positive")
	}
	if policy.Retries != nil && *policy.Retries < 0 {
		return fmt.Errorf("managed discovery.retries must not be negative")
	}
	if strings.TrimSpace(policy.Timeout) != "" {
		if _, err := time.ParseDuration(strings.TrimSpace(policy.Timeout)); err != nil {
			return fmt.Errorf("managed discovery.timeout is invalid: %w", err)
		}
	}
	return nil
}

func applyDiscoveryDefaults(d *DiscoveryConfig) {
	if len(d.AllowedCIDRs) == 0 {
		return
	}
	if d.MaxTargets == 0 {
		d.MaxTargets = 256
	}
	if d.Timeout == 0 {
		d.Timeout = 2 * time.Second
	}
	if d.MaxWorkers == 0 {
		d.MaxWorkers = 4
	}
	if d.MaxProbesPerSecond == 0 {
		d.MaxProbesPerSecond = 5
	}
	if d.ProbeBurst == 0 {
		d.ProbeBurst = 2
	}
	if d.CommunityEnv == "" {
		d.CommunityEnv = "SNMP_DISCOVERY_COMMUNITY"
	}
}

func validateSourceFile(path string, managed bool) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("must be a regular file")
	}
	if info.Mode().Perm()&0o022 != 0 {
		return fmt.Errorf("must not be group/world writable")
	}
	if managed && info.Mode().Perm()&0o077 != 0 {
		return fmt.Errorf("managed inventory must be owner-only (0600)")
	}
	return nil
}

// mergeInventories builds the runtime device list: static devices first, then
// overlays for matching IDs, then unique managed appends. Static-authoritative
// identity fields (host, port, version, community_env) are never replaced.
func mergeInventories(staticDevices, managedDevices []DeviceConfig) []DeviceConfig {
	merged := make([]DeviceConfig, 0, len(staticDevices)+len(managedDevices))
	indexByID := make(map[string]int, len(staticDevices))
	for _, device := range staticDevices {
		indexByID[device.ID] = len(merged)
		merged = append(merged, device)
	}
	for _, device := range managedDevices {
		if idx, exists := indexByID[device.ID]; exists {
			merged[idx] = applyDeviceOverlay(merged[idx], device)
			continue
		}
		merged = append(merged, device)
	}
	return merged
}

// applyDeviceOverlay copies allowed managed overlay fields onto a static device.
func applyDeviceOverlay(base, overlay DeviceConfig) DeviceConfig {
	if overlay.TemperatureWarningC != nil {
		value := *overlay.TemperatureWarningC
		base.TemperatureWarningC = &value
	}
	if overlay.UpstreamDeviceIDs != nil {
		base.UpstreamDeviceIDs = append([]string(nil), overlay.UpstreamDeviceIDs...)
	}
	if strings.TrimSpace(overlay.Role) != "" {
		base.Role = strings.TrimSpace(overlay.Role)
	}
	if hasInterfaceFilterConfig(overlay.InterfaceFilters) {
		base.InterfaceFilters = cloneInterfaceFilters(overlay.InterfaceFilters)
	}
	return base
}

func hasInterfaceFilterConfig(filters InterfaceFilterConfig) bool {
	return len(filters.Rules) > 0 ||
		len(filters.IncludeIfIndexes) > 0 ||
		len(filters.ExcludeIfIndexes) > 0 ||
		len(filters.IncludeNameRegex) > 0 ||
		len(filters.ExcludeNameRegex) > 0 ||
		len(filters.IncludeAliasRegex) > 0 ||
		len(filters.ExcludeAliasRegex) > 0 ||
		len(filters.IncludeTypes) > 0 ||
		len(filters.ExcludeTypes) > 0 ||
		len(filters.IncludeAdminStatuses) > 0 ||
		len(filters.ExcludeAdminStatuses) > 0 ||
		len(filters.IncludeOperStatuses) > 0 ||
		len(filters.ExcludeOperStatuses) > 0
}

func cloneInterfaceFilters(filters InterfaceFilterConfig) InterfaceFilterConfig {
	out := InterfaceFilterConfig{
		Rules:                append([]InterfaceFilterRule(nil), filters.Rules...),
		IncludeIfIndexes:     append([]int(nil), filters.IncludeIfIndexes...),
		ExcludeIfIndexes:     append([]int(nil), filters.ExcludeIfIndexes...),
		IncludeNameRegex:     append([]string(nil), filters.IncludeNameRegex...),
		ExcludeNameRegex:     append([]string(nil), filters.ExcludeNameRegex...),
		IncludeAliasRegex:    append([]string(nil), filters.IncludeAliasRegex...),
		ExcludeAliasRegex:    append([]string(nil), filters.ExcludeAliasRegex...),
		IncludeTypes:         append([]string(nil), filters.IncludeTypes...),
		ExcludeTypes:         append([]string(nil), filters.ExcludeTypes...),
		IncludeAdminStatuses: append([]string(nil), filters.IncludeAdminStatuses...),
		ExcludeAdminStatuses: append([]string(nil), filters.ExcludeAdminStatuses...),
		IncludeOperStatuses:  append([]string(nil), filters.IncludeOperStatuses...),
		ExcludeOperStatuses:  append([]string(nil), filters.ExcludeOperStatuses...),
	}
	return out
}

func (c *Config) applyDefaults() {
	if c.PollInterval == 0 {
		c.PollInterval = defaultPollInterval
	}
	if c.MaxWorkers == 0 {
		c.MaxWorkers = defaultMaxWorkers
	}
	if c.Collector.HeartbeatInterval == 0 {
		c.Collector.HeartbeatInterval = defaultHeartbeatInterval
	}
	if c.Health.TemperatureWarningC == 0 {
		c.Health.TemperatureWarningC = defaultTemperatureWarning
	}
	if c.Health.FailureThreshold == 0 {
		c.Health.FailureThreshold = defaultFailureThreshold
	}
	if c.Admin.Listen == "" {
		c.Admin.Listen = ":9090"
	}
	if c.SNMP.Timeout == 0 {
		c.SNMP.Timeout = defaultSNMPTimeout
	}
	if c.SNMP.Retries == 0 {
		c.SNMP.Retries = defaultSNMPRetries
	}
	if c.Publisher.Mode == "" {
		c.Publisher.Mode = "stdout"
	}
	if c.Publisher.Timeout == 0 {
		c.Publisher.Timeout = defaultPublisherTimeout
	}
	if c.Publisher.TelemetryVersion == "" {
		c.Publisher.TelemetryVersion = "v2"
	}
	if c.Buffer.Path == "" {
		c.Buffer.Path = "buffer.db"
	}
	if c.Buffer.MaxEntries == 0 {
		c.Buffer.MaxEntries = 50000
	}
	if c.Buffer.BusyTimeoutMS == 0 {
		c.Buffer.BusyTimeoutMS = 5000
	}
	if c.Buffer.BatchSize == 0 {
		c.Buffer.BatchSize = 50
	}
	if c.Buffer.IdleBackoff == 0 {
		c.Buffer.IdleBackoff = 500 * time.Millisecond
	}
	if c.MQTT.PasswordEnv == "" {
		c.MQTT.PasswordEnv = "MQTT_PASSWORD"
	}
	if c.MQTT.QoS == 0 {
		c.MQTT.QoS = 1
	}
	if c.MQTT.ClientID == "" && c.SiteID != "" {
		c.MQTT.ClientID = "collector-" + c.SiteID
	}
	if c.MQTT.Reconnect.Initial == 0 {
		c.MQTT.Reconnect.Initial = time.Second
	}
	if c.MQTT.Reconnect.Max == 0 {
		c.MQTT.Reconnect.Max = 60 * time.Second
	}
	if c.Discovery.ProbeBurst == 0 && c.Discovery.MaxProbesPerSecond > 0 {
		c.Discovery.ProbeBurst = 1
	}
	applyDeviceDefaults(c.Devices)
}

func applyDeviceDefaults(devices []DeviceConfig) {
	for i := range devices {
		if devices[i].Port == 0 {
			devices[i].Port = 161
		}
		if devices[i].Version == "" {
			devices[i].Version = "2c"
		}
	}
}

// applyEnvOverrides replaces non-secret MQTT settings from the environment.
func (c *Config) applyEnvOverrides() {
	if v := os.Getenv("MQTT_BROKER"); v != "" {
		c.MQTT.Broker = v
	}
}

// MQTTPassword returns the MQTT password from the configured environment variable.
func (c *Config) MQTTPassword() string {
	if c.MQTT.PasswordEnv == "" {
		return ""
	}
	return os.Getenv(c.MQTT.PasswordEnv)
}

// DiscoveryCommunity resolves the discovery secret only when a probe is requested.
func (c *Config) DiscoveryCommunity() string {
	if c.Discovery.CommunityEnv == "" {
		return ""
	}
	return os.Getenv(c.Discovery.CommunityEnv)
}

// ManagedInventoryPath returns the configured managed path relative to the config file.
func (c *Config) ManagedInventoryPath() string {
	if strings.TrimSpace(c.Inventory.ManagedPath) == "" {
		return ""
	}
	if filepath.IsAbs(c.Inventory.ManagedPath) {
		return filepath.Clean(c.Inventory.ManagedPath)
	}
	return filepath.Join(filepath.Dir(c.configPath), c.Inventory.ManagedPath)
}

// MQTTInsecureSkipVerify reports whether TLS verification should be skipped (dev only).
func MQTTInsecureSkipVerify() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("MQTT_TLS_INSECURE")))
	return v == "1" || v == "true" || v == "yes"
}

// Validate checks the configuration without requiring secret values.
func (c *Config) Validate() error {
	return c.validate(false)
}

func (c *Config) validate(requireRuntimeSecrets bool) error {
	if strings.TrimSpace(c.SiteID) == "" {
		return fmt.Errorf("site_id is required")
	}
	if !identifierPattern.MatchString(c.SiteID) {
		return fmt.Errorf("site_id has invalid format")
	}
	if strings.TrimSpace(c.Collector.ID) == "" {
		return fmt.Errorf("collector.id is required")
	}
	if !identifierPattern.MatchString(c.Collector.ID) {
		return fmt.Errorf("collector.id has invalid format")
	}
	if c.PollInterval <= 0 || c.PollInterval > maxPollInterval {
		return fmt.Errorf("poll_interval must be between 1ns and %s", maxPollInterval)
	}
	if c.MaxWorkers <= 0 || c.MaxWorkers > maxWorkers {
		return fmt.Errorf("max_workers must be between 1 and %d", maxWorkers)
	}
	if c.Collector.HeartbeatInterval <= 0 || c.Collector.HeartbeatInterval > maxPollInterval {
		return fmt.Errorf("collector.heartbeat_interval must be between 1ns and %s", maxPollInterval)
	}
	if c.Health.TemperatureWarningC < minTemperatureCelsius || c.Health.TemperatureWarningC > maxTemperatureCelsius {
		return fmt.Errorf("health.temperature_warning_c must be between %.0f and %.0f", minTemperatureCelsius, maxTemperatureCelsius)
	}
	if c.Health.FailureThreshold <= 0 || c.Health.FailureThreshold > maxRetries+1 {
		return fmt.Errorf("health.failure_threshold must be between 1 and %d", maxRetries+1)
	}
	if c.SNMP.Timeout <= 0 || c.SNMP.Timeout > maxSNMPTimeout {
		return fmt.Errorf("snmp.timeout must be between 1ns and %s", maxSNMPTimeout)
	}
	if c.SNMP.Retries < 0 || c.SNMP.Retries > maxRetries {
		return fmt.Errorf("snmp.retries must be between 0 and %d", maxRetries)
	}
	if c.Publisher.Timeout <= 0 || c.Publisher.Timeout > maxSNMPTimeout {
		return fmt.Errorf("publisher.timeout must be between 1ns and %s", maxSNMPTimeout)
	}
	switch c.Publisher.Mode {
	case "stdout", "mqtt":
	default:
		return fmt.Errorf("publisher.mode must be \"stdout\" or \"mqtt\"")
	}
	switch c.Publisher.TelemetryVersion {
	case "v1", "v2", "both":
	default:
		return fmt.Errorf("publisher.telemetry_version must be \"v1\", \"v2\", or \"both\"")
	}
	if c.Publisher.Mode == "mqtt" {
		if err := c.validateMQTT(); err != nil {
			return err
		}
	}
	if err := validateDiscovery(c.Discovery); err != nil {
		return err
	}
	if len(c.Devices) == 0 && len(c.Discovery.AllowedCIDRs) == 0 {
		return fmt.Errorf("at least one device is required")
	}
	if err := validateDeviceSource(c.Devices, "devices"); err != nil {
		return err
	}
	if err := validateInventoryUniqueness(c.Devices); err != nil {
		return err
	}
	if err := validateDependencies(c.Devices); err != nil {
		return err
	}
	return nil
}

func (c *Config) validateRuntimeSecrets() error {
	if c.Publisher.Mode == "mqtt" && c.MQTTPassword() == "" {
		return fmt.Errorf("environment variable %q is required when publisher.mode is mqtt", c.MQTT.PasswordEnv)
	}
	return nil
}

func (c *Config) validateMQTT() error {
	if strings.TrimSpace(c.MQTT.Broker) == "" {
		return fmt.Errorf("mqtt.broker is required when publisher.mode is mqtt")
	}
	u, err := url.Parse(c.MQTT.Broker)
	if err != nil || u.Scheme != "tls" || u.Hostname() == "" {
		return fmt.Errorf("mqtt.broker must be a tls URL with a host")
	}
	if strings.TrimSpace(c.MQTT.Username) == "" {
		return fmt.Errorf("mqtt.username is required when publisher.mode is mqtt")
	}
	if err := validateEnvName(c.MQTT.PasswordEnv, "mqtt.password_env"); err != nil {
		return err
	}
	if c.MQTT.QoS != 1 {
		return fmt.Errorf("mqtt.qos must be 1")
	}
	if c.Buffer.MaxEntries < 0 {
		return fmt.Errorf("buffer.max_entries must be >= 0")
	}
	if c.Buffer.BusyTimeoutMS <= 0 {
		return fmt.Errorf("buffer.busy_timeout_ms must be positive")
	}
	if c.Buffer.BatchSize <= 0 {
		return fmt.Errorf("buffer.batch_size must be positive")
	}
	if c.Buffer.IdleBackoff <= 0 {
		return fmt.Errorf("buffer.idle_backoff must be positive")
	}
	if c.MQTT.Reconnect.Initial <= 0 || c.MQTT.Reconnect.Max <= 0 {
		return fmt.Errorf("mqtt.reconnect.initial and max must be positive")
	}
	if c.MQTT.Reconnect.Max < c.MQTT.Reconnect.Initial {
		return fmt.Errorf("mqtt.reconnect.max must be >= initial")
	}
	if strings.TrimSpace(c.MQTT.TLS.CAFile) == "" && !MQTTInsecureSkipVerify() {
		return fmt.Errorf("mqtt.tls.ca_file is required unless MQTT_TLS_INSECURE is set")
	}
	if (c.MQTT.TLS.CertFile == "") != (c.MQTT.TLS.KeyFile == "") {
		return fmt.Errorf("mqtt.tls.cert_file and key_file must be configured together")
	}
	return nil
}

func validateDeviceSource(devices []DeviceConfig, source string) error {
	seenIDs := make(map[string]struct{}, len(devices))
	seenHosts := make(map[string]struct{}, len(devices))
	for i, d := range devices {
		prefix := fmt.Sprintf("%s[%d]", source, i)
		if strings.TrimSpace(d.ID) == "" {
			return fmt.Errorf("%s.id is required", prefix)
		}
		if !identifierPattern.MatchString(d.ID) {
			return fmt.Errorf("%s.id has invalid format", prefix)
		}
		if strings.TrimSpace(d.Host) == "" {
			return fmt.Errorf("%s.host is required", prefix)
		}
		if d.Port == 0 {
			return fmt.Errorf("%s.port must be between 1 and 65535", prefix)
		}
		if d.Version != "2c" {
			return fmt.Errorf("%s.version: only \"2c\" is supported", prefix)
		}
		if err := validateEnvName(d.CommunityEnv, prefix+".community_env"); err != nil {
			return err
		}
		switch strings.ToLower(strings.TrimSpace(d.Vendor)) {
		case "", "core", "cisco", "arista":
		default:
			return fmt.Errorf("%s.vendor must be empty, core, cisco, or arista", prefix)
		}
		if _, ok := seenIDs[d.ID]; ok {
			return fmt.Errorf("duplicate device id %q in %s", d.ID, source)
		}
		seenIDs[d.ID] = struct{}{}
		host := canonicalHost(d.Host)
		if _, ok := seenHosts[host]; ok {
			return fmt.Errorf("duplicate device host %q in %s", d.Host, source)
		}
		seenHosts[host] = struct{}{}
		if d.PollInterval < 0 || d.PollInterval > maxPollInterval {
			return fmt.Errorf("%s.poll_interval must be between 0 and %s", prefix, maxPollInterval)
		}
		if d.Timeout < 0 || d.Timeout > maxSNMPTimeout {
			return fmt.Errorf("%s.timeout must be between 0 and %s", prefix, maxSNMPTimeout)
		}
		if d.Retries < 0 || d.Retries > maxRetries {
			return fmt.Errorf("%s.retries must be between 0 and %d", prefix, maxRetries)
		}
		if d.TemperatureWarningC != nil && (*d.TemperatureWarningC < minTemperatureCelsius || *d.TemperatureWarningC > maxTemperatureCelsius) {
			return fmt.Errorf("%s.temperature_warning_c must be between %.0f and %.0f", prefix, minTemperatureCelsius, maxTemperatureCelsius)
		}
		if err := validateInterfaceFilters(d.InterfaceFilters, prefix+".interface_filters"); err != nil {
			return err
		}
	}
	return nil
}

func validateEnvName(value, field string) error {
	if strings.TrimSpace(value) == "" {
		return fmt.Errorf("%s is required", field)
	}
	if !envNamePattern.MatchString(value) {
		return fmt.Errorf("%s must be a valid environment variable name", field)
	}
	return nil
}

func validateInventoryUniqueness(devices []DeviceConfig) error {
	seenIDs := make(map[string]struct{}, len(devices))
	seenHosts := make(map[string]struct{}, len(devices))
	for _, d := range devices {
		if _, ok := seenIDs[d.ID]; ok {
			return fmt.Errorf("duplicate device id %q", d.ID)
		}
		seenIDs[d.ID] = struct{}{}
		host := canonicalHost(d.Host)
		if _, ok := seenHosts[host]; ok {
			return fmt.Errorf("duplicate device host %q", d.Host)
		}
		seenHosts[host] = struct{}{}
	}
	return nil
}

func canonicalHost(host string) string {
	host = strings.TrimSpace(strings.TrimSuffix(host, "."))
	if ip := net.ParseIP(host); ip != nil {
		return ip.String()
	}
	return strings.ToLower(host)
}

// ValidateDependencies checks upstream references and cycles across a device inventory.
func ValidateDependencies(devices []DeviceConfig) error {
	return validateDependencies(devices)
}

// ValidatePendingDependencyMutation checks that applying upstream_device_ids to the
// on-disk managed inventory would remain valid when merged with static inventory.
// Un-reloaded managed commits are included so sequential dependency writes cannot
// persist a cycle that bricks reload or restart.
func (c *Config) ValidatePendingDependencyMutation(deviceID string, upstreams []string) error {
	if c == nil {
		return errors.New("configuration is required")
	}
	if strings.TrimSpace(c.configPath) == "" {
		return errors.New("configuration path is required")
	}
	staticDevices, err := readStaticDevicesFromConfig(c.configPath)
	if err != nil {
		return fmt.Errorf("read static inventory: %w", err)
	}
	managed, err := ReadManagedDocument(c.ManagedInventoryPath())
	if err != nil {
		return err
	}
	found := false
	for i := range managed.Devices {
		if managed.Devices[i].ID == deviceID {
			managed.Devices[i].UpstreamDeviceIDs = append([]string(nil), upstreams...)
			found = true
			break
		}
	}
	if !found {
		managed.Devices = append(managed.Devices, DeviceConfig{
			ID:                deviceID,
			UpstreamDeviceIDs: upstreams,
		})
	}
	merged := mergeInventories(staticDevices, managed.Devices)
	deviceExists := false
	for _, device := range merged {
		if device.ID == deviceID {
			deviceExists = true
			break
		}
	}
	if !deviceExists {
		return fmt.Errorf("device_id not found in active inventory")
	}
	return validateDependencies(merged)
}

func readStaticDevicesFromConfig(configPath string) ([]DeviceConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	var partial struct {
		Devices []DeviceConfig `yaml:"devices"`
	}
	if err := yaml.Unmarshal(data, &partial); err != nil {
		return nil, fmt.Errorf("parse config devices: %w", err)
	}
	devices := append([]DeviceConfig(nil), partial.Devices...)
	applyDeviceDefaults(devices)
	return devices, nil
}

func validateDependencies(devices []DeviceConfig) error {
	byID := make(map[string]DeviceConfig, len(devices))
	for _, device := range devices {
		byID[device.ID] = device
	}
	for _, device := range devices {
		seen := make(map[string]struct{}, len(device.UpstreamDeviceIDs))
		for _, upstream := range device.UpstreamDeviceIDs {
			if _, duplicate := seen[upstream]; duplicate {
				return fmt.Errorf("device %q has duplicate upstream_device_id %q", device.ID, upstream)
			}
			seen[upstream] = struct{}{}
			if upstream == device.ID {
				return fmt.Errorf("device %q cannot reference itself as an upstream", device.ID)
			}
			if _, exists := byID[upstream]; !exists {
				return fmt.Errorf("device %q references missing upstream %q", device.ID, upstream)
			}
		}
	}

	const (
		unvisited = 0
		visiting  = 1
		visited   = 2
	)
	state := make(map[string]int, len(devices))
	var visit func(string, []string) error
	visit = func(id string, path []string) error {
		switch state[id] {
		case visiting:
			return fmt.Errorf("dependency cycle detected at device %q", id)
		case visited:
			return nil
		}
		state[id] = visiting
		path = append(path, id)
		for _, upstream := range byID[id].UpstreamDeviceIDs {
			if err := visit(upstream, path); err != nil {
				return err
			}
		}
		state[id] = visited
		return nil
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := visit(id, nil); err != nil {
			return err
		}
	}
	return nil
}

func validateDiscovery(discovery DiscoveryConfig) error {
	if len(discovery.AllowedCIDRs) == 0 && discovery.CommunityEnv == "" && discovery.MaxTargets == 0 && discovery.Timeout == 0 && discovery.Retries == 0 && discovery.MaxWorkers == 0 && discovery.MaxProbesPerSecond == 0 && discovery.ProbeBurst == 0 {
		return nil
	}
	if len(discovery.AllowedCIDRs) == 0 {
		return fmt.Errorf("discovery.allowed_cidrs is required when discovery is configured")
	}
	if err := validateEnvName(discovery.CommunityEnv, "discovery.community_env"); err != nil {
		return err
	}
	if discovery.MaxTargets <= 0 || discovery.MaxTargets > maxDiscoveryTargets {
		return fmt.Errorf("discovery.max_targets must be between 1 and %d", maxDiscoveryTargets)
	}
	if discovery.Timeout <= 0 || discovery.Timeout > maxSNMPTimeout {
		return fmt.Errorf("discovery.timeout must be between 1ns and %s", maxSNMPTimeout)
	}
	if discovery.Retries < 0 || discovery.Retries > maxRetries {
		return fmt.Errorf("discovery.retries must be between 0 and %d", maxRetries)
	}
	if discovery.MaxWorkers <= 0 || discovery.MaxWorkers > maxWorkers {
		return fmt.Errorf("discovery.max_workers must be between 1 and %d", maxWorkers)
	}
	if discovery.MaxProbesPerSecond <= 0 || math.IsNaN(discovery.MaxProbesPerSecond) || math.IsInf(discovery.MaxProbesPerSecond, 0) || discovery.MaxProbesPerSecond > maxDiscoveryRate {
		return fmt.Errorf("discovery.max_probes_per_second must be positive and <= %.0f", maxDiscoveryRate)
	}
	if discovery.ProbeBurst <= 0 || discovery.ProbeBurst > maxDiscoveryBurst {
		return fmt.Errorf("discovery.probe_burst must be between 1 and %d", maxDiscoveryBurst)
	}
	for i, cidr := range discovery.AllowedCIDRs {
		if _, _, err := net.ParseCIDR(cidr); err != nil {
			return fmt.Errorf("discovery.allowed_cidrs[%d] is invalid: %w", i, err)
		}
	}
	return nil
}

var validInterfaceStates = map[string]struct{}{
	"up": {}, "down": {}, "testing": {}, "unknown": {},
}

var validInterfaceTypes = map[string]struct{}{
	"other": {}, "ethernetcsmacd": {}, "ieee8023adlag": {}, "l2vlan": {},
	"softwareloopback": {}, "propvirtual": {}, "tunnel": {}, "bridge": {},
	"ieee80211": {}, "fddi": {}, "ppp": {}, "framerelay": {}, "atm": {},
}

func validateInterfaceFilters(filters InterfaceFilterConfig, field string) error {
	for name, indexes := range map[string][]int{
		"include_if_indexes": filters.IncludeIfIndexes,
		"exclude_if_indexes": filters.ExcludeIfIndexes,
	} {
		for i, index := range indexes {
			if index <= 0 {
				return fmt.Errorf("%s.%s[%d] must be positive", field, name, i)
			}
		}
	}
	for name, patterns := range map[string][]string{
		"include_name_regex":  filters.IncludeNameRegex,
		"exclude_name_regex":  filters.ExcludeNameRegex,
		"include_alias_regex": filters.IncludeAliasRegex,
		"exclude_alias_regex": filters.ExcludeAliasRegex,
	} {
		for i, pattern := range patterns {
			if _, err := regexp.Compile(pattern); err != nil {
				return fmt.Errorf("%s.%s[%d] invalid regex: %w", field, name, i, err)
			}
		}
	}
	for name, types := range map[string][]string{
		"include_types": filters.IncludeTypes,
		"exclude_types": filters.ExcludeTypes,
	} {
		for i, value := range types {
			if err := validateInterfaceType(value); err != nil {
				return fmt.Errorf("%s.%s[%d]: %w", field, name, i, err)
			}
		}
	}
	for name, states := range map[string][]string{
		"include_admin_statuses": filters.IncludeAdminStatuses,
		"exclude_admin_statuses": filters.ExcludeAdminStatuses,
		"include_oper_statuses":  filters.IncludeOperStatuses,
		"exclude_oper_statuses":  filters.ExcludeOperStatuses,
	} {
		for i, state := range states {
			if _, ok := validInterfaceStates[strings.ToLower(state)]; !ok {
				return fmt.Errorf("%s.%s[%d] has unsupported state %q", field, name, i, state)
			}
		}
	}
	for i, rule := range filters.Rules {
		prefix := fmt.Sprintf("%s.rules[%d]", field, i)
		if rule.Action != "include" && rule.Action != "exclude" {
			return fmt.Errorf("%s.action must be include or exclude", prefix)
		}
		if rule.IfIndex != nil && *rule.IfIndex <= 0 {
			return fmt.Errorf("%s.if_index must be positive", prefix)
		}
		matched := rule.IfIndex != nil
		for name, pattern := range map[string]string{"name_regex": rule.NameRegex, "alias_regex": rule.AliasRegex} {
			if pattern == "" {
				continue
			}
			matched = true
			if _, err := regexp.Compile(pattern); err != nil {
				return fmt.Errorf("%s.%s invalid regex: %w", prefix, name, err)
			}
		}
		if rule.IfType != "" {
			matched = true
			if err := validateInterfaceType(rule.IfType); err != nil {
				return fmt.Errorf("%s.if_type: %w", prefix, err)
			}
		}
		for name, state := range map[string]string{"admin_status": rule.AdminStatus, "oper_status": rule.OperStatus} {
			if state == "" {
				continue
			}
			matched = true
			if _, ok := validInterfaceStates[strings.ToLower(state)]; !ok {
				return fmt.Errorf("%s.%s has unsupported state %q", prefix, name, state)
			}
		}
		if !matched {
			return fmt.Errorf("%s must contain at least one matcher", prefix)
		}
	}
	return nil
}

func validateInterfaceType(value string) error {
	normalized := strings.ToLower(strings.TrimSpace(value))
	if normalized == "" {
		return errors.New("interface type is required")
	}
	if _, ok := validInterfaceTypes[normalized]; ok {
		return nil
	}
	if n := parsePositiveInt(normalized); n > 0 && n <= 255 {
		return nil
	}
	return fmt.Errorf("unsupported interface type %q", value)
}

func parsePositiveInt(value string) int {
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0
	}
	return n
}

// EffectivePollInterval returns the device interval or the shared default.
func (d DeviceConfig) EffectivePollInterval(global time.Duration) time.Duration {
	if d.PollInterval > 0 {
		return d.PollInterval
	}
	return global
}

// EffectiveTemperatureWarningC returns the device temperature threshold or the shared default.
func (d DeviceConfig) EffectiveTemperatureWarningC(global float64) float64 {
	if d.TemperatureWarningC != nil {
		return *d.TemperatureWarningC
	}
	return global
}

// EffectiveSNMP returns shared SNMP settings with device overrides applied.
func (d DeviceConfig) EffectiveSNMP(shared SNMPConfig) SNMPConfig {
	if d.Timeout > 0 {
		shared.Timeout = d.Timeout
	}
	if d.Retries > 0 {
		shared.Retries = d.Retries
	}
	return shared
}

// TemperaturePolicyRevision returns a non-secret fingerprint of the active temperature policy.
func TemperaturePolicyRevision(cfg *Config) string {
	if cfg == nil {
		return "policy-none"
	}
	type override struct {
		ID    string
		Value float64
	}
	overrides := make([]override, 0)
	for _, device := range cfg.Devices {
		if device.TemperatureWarningC == nil {
			continue
		}
		overrides = append(overrides, override{ID: device.ID, Value: *device.TemperatureWarningC})
	}
	sort.Slice(overrides, func(i, j int) bool {
		return overrides[i].ID < overrides[j].ID
	})

	h := sha256.New()
	_, _ = fmt.Fprintf(h, "global=%.4f;", cfg.Health.TemperatureWarningC)
	for _, item := range overrides {
		_, _ = fmt.Fprintf(h, "%s=%.4f;", item.ID, item.Value)
	}
	return fmt.Sprintf("policy-%x", h.Sum(nil)[:8])
}

// ConfigRevision returns a non-secret fingerprint of the active configuration snapshot
// used as the MQTT envelope config_revision field.
func ConfigRevision(cfg *Config) string {
	if cfg == nil {
		return "revision-none"
	}
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "site=%s;collector=%s;telemetry=%s;temp=%.4f;fail=%d;",
		cfg.SiteID, cfg.Collector.ID, cfg.Publisher.TelemetryVersion,
		cfg.Health.TemperatureWarningC, cfg.Health.FailureThreshold)
	devices := append([]DeviceConfig(nil), cfg.Devices...)
	sort.Slice(devices, func(i, j int) bool { return devices[i].ID < devices[j].ID })
	for _, d := range devices {
		_, _ = fmt.Fprintf(h, "dev=%s;host=%s;", d.ID, d.Host)
		ups := append([]string(nil), d.UpstreamDeviceIDs...)
		sort.Strings(ups)
		for _, u := range ups {
			_, _ = fmt.Fprintf(h, "up=%s;", u)
		}
		if d.TemperatureWarningC != nil {
			_, _ = fmt.Fprintf(h, "tw=%.4f;", *d.TemperatureWarningC)
		}
		if d.PollInterval > 0 {
			_, _ = fmt.Fprintf(h, "pi=%s;", d.PollInterval)
		}
	}
	return fmt.Sprintf("revision-%x", h.Sum(nil)[:10])
}

// ReadManagedDocument loads the full managed inventory document. A missing file is empty.
func ReadManagedDocument(path string) (ManagedInventory, error) {
	var empty ManagedInventory
	if strings.TrimSpace(path) == "" {
		return empty, nil
	}
	path = filepath.Clean(path)
	if err := validateSourceFile(path, true); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return empty, nil
		}
		return empty, fmt.Errorf("managed inventory: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return empty, fmt.Errorf("read managed inventory: %w", err)
	}
	var inventory ManagedInventory
	if err := decodeYAML(data, &inventory); err != nil {
		return empty, fmt.Errorf("parse managed inventory: %w", err)
	}
	return inventory, nil
}

// ReadManagedInventory loads managed device entries. A missing file is empty.
func ReadManagedInventory(path string) ([]DeviceConfig, error) {
	inventory, err := ReadManagedDocument(path)
	if err != nil {
		return nil, err
	}
	return append([]DeviceConfig(nil), inventory.Devices...), nil
}

// WriteManagedDocument atomically persists a secret-free managed inventory document.
func WriteManagedDocument(path string, inventory ManagedInventory) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("managed inventory path is required")
	}
	normalized := inventory
	normalized.Devices = append([]DeviceConfig(nil), inventory.Devices...)
	applyDeviceDefaults(normalized.Devices)
	// Full validation of appends requires static context; here we only validate
	// entries that look like complete devices. Overlay-only entries (no host)
	// are validated lightly so control-plane writers can persist overlays.
	appends := make([]DeviceConfig, 0)
	for i, device := range normalized.Devices {
		if strings.TrimSpace(device.Host) == "" {
			if err := validateManagedOverlay(device, fmt.Sprintf("managed devices[%d]", i)); err != nil {
				return err
			}
			continue
		}
		appends = append(appends, device)
	}
	if err := validateDeviceSource(appends, "managed devices"); err != nil {
		return err
	}
	if normalized.Health.TemperatureWarningC != nil {
		v := *normalized.Health.TemperatureWarningC
		if v < minTemperatureCelsius || v > maxTemperatureCelsius {
			return fmt.Errorf("managed health.temperature_warning_c must be between %.0f and %.0f", minTemperatureCelsius, maxTemperatureCelsius)
		}
	}
	if normalized.Discovery.MaxProbesPerSecond != nil && *normalized.Discovery.MaxProbesPerSecond <= 0 {
		return fmt.Errorf("managed discovery.max_probes_per_second must be positive")
	}
	if normalized.Discovery.ProbeBurst != nil && *normalized.Discovery.ProbeBurst <= 0 {
		return fmt.Errorf("managed discovery.probe_burst must be positive")
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create managed inventory directory: %w", err)
	}
	tmp, err := os.CreateTemp(dir, ".managed-inventory-*.tmp")
	if err != nil {
		return fmt.Errorf("create managed inventory temp file: %w", err)
	}
	tmpName := tmp.Name()
	cleanup := true
	defer func() {
		_ = tmp.Close()
		if cleanup {
			_ = os.Remove(tmpName)
		}
	}()

	if err := tmp.Chmod(0o600); err != nil {
		return fmt.Errorf("set managed inventory permissions: %w", err)
	}
	encoder := yaml.NewEncoder(tmp)
	if err := encoder.Encode(normalized); err != nil {
		_ = encoder.Close()
		return fmt.Errorf("encode managed inventory: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return fmt.Errorf("close managed inventory encoder: %w", err)
	}
	if err := tmp.Sync(); err != nil {
		return fmt.Errorf("sync managed inventory: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close managed inventory: %w", err)
	}
	if err := replaceManagedInventoryFile(tmpName, path); err != nil {
		return fmt.Errorf("replace managed inventory: %w", err)
	}
	cleanup = false
	directory, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("open managed inventory directory: %w", err)
	}
	if err := directory.Sync(); err != nil {
		_ = directory.Close()
		return fmt.Errorf("sync managed inventory directory: %w", err)
	}
	if err := directory.Close(); err != nil {
		return fmt.Errorf("close managed inventory directory: %w", err)
	}
	return nil
}

func replaceManagedInventoryFile(tmpName, path string) error {
	if err := os.Rename(tmpName, path); err == nil {
		return nil
	} else if !isCrossDeviceRenameError(err) {
		return err
	}
	data, err := os.ReadFile(tmpName)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return err
	}
	return os.Remove(tmpName)
}

func isCrossDeviceRenameError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, syscall.EXDEV) || strings.Contains(err.Error(), "cross-device")
}

// WriteManagedInventory atomically persists managed devices while preserving
// any existing managed health/discovery policy sections.
func WriteManagedInventory(path string, devices []DeviceConfig) error {
	existing, err := ReadManagedDocument(path)
	if err != nil {
		return err
	}
	existing.Devices = append([]DeviceConfig(nil), devices...)
	return WriteManagedDocument(path, existing)
}
