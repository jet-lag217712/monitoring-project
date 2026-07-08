package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the collector runtime configuration.
type Config struct {
	SiteID       string         `yaml:"site_id"`
	PollInterval time.Duration  `yaml:"poll_interval"`
	MaxWorkers   int            `yaml:"max_workers"`
	Admin        AdminConfig    `yaml:"admin"`
	SNMP         SNMPConfig     `yaml:"snmp"`
	Devices      []DeviceConfig `yaml:"devices"`
}

// AdminConfig controls the admin HTTP server (metrics/health).
type AdminConfig struct {
	Listen string `yaml:"listen"`
}

// SNMPConfig holds shared SNMP client defaults.
type SNMPConfig struct {
	Timeout time.Duration `yaml:"timeout"`
	Retries int           `yaml:"retries"`
}

// DeviceConfig describes a single SNMP poll target.
type DeviceConfig struct {
	ID        string `yaml:"id"`
	Host      string `yaml:"host"`
	Port      uint16 `yaml:"port"`
	Community string `yaml:"community"`
	Version   string `yaml:"version"`
	Vendor    string `yaml:"vendor"`
}

// Load reads and validates a collector config file.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.applyDefaults()
	cfg.applyEnvOverrides()

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.PollInterval == 0 {
		c.PollInterval = 60 * time.Second
	}
	if c.MaxWorkers == 0 {
		c.MaxWorkers = 10
	}
	if c.Admin.Listen == "" {
		c.Admin.Listen = ":9090"
	}
	if c.SNMP.Timeout == 0 {
		c.SNMP.Timeout = 5 * time.Second
	}
	if c.SNMP.Retries == 0 {
		c.SNMP.Retries = 2
	}
	for i := range c.Devices {
		if c.Devices[i].Port == 0 {
			c.Devices[i].Port = 161
		}
		if c.Devices[i].Version == "" {
			c.Devices[i].Version = "2c"
		}
		if c.Devices[i].Community == "" {
			c.Devices[i].Community = "public"
		}
	}
}

// applyEnvOverrides replaces community strings from SNMP_COMMUNITY_<device_id>.
// Device IDs with non-alphanumeric characters are uppercased with those chars
// replaced by underscores (e.g. "dev-001" → SNMP_COMMUNITY_DEV_001).
func (c *Config) applyEnvOverrides() {
	for i := range c.Devices {
		key := communityEnvKey(c.Devices[i].ID)
		if v := os.Getenv(key); v != "" {
			c.Devices[i].Community = v
		}
	}
}

func communityEnvKey(deviceID string) string {
	var b strings.Builder
	b.WriteString("SNMP_COMMUNITY_")
	for _, r := range strings.ToUpper(deviceID) {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	return b.String()
}

// Validate checks required fields and uniqueness constraints.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.SiteID) == "" {
		return fmt.Errorf("site_id is required")
	}
	if c.PollInterval <= 0 {
		return fmt.Errorf("poll_interval must be positive")
	}
	if c.MaxWorkers <= 0 {
		return fmt.Errorf("max_workers must be positive")
	}
	if len(c.Devices) == 0 {
		return fmt.Errorf("at least one device is required")
	}

	seen := make(map[string]struct{}, len(c.Devices))
	for i, d := range c.Devices {
		if strings.TrimSpace(d.ID) == "" {
			return fmt.Errorf("devices[%d].id is required", i)
		}
		if strings.TrimSpace(d.Host) == "" {
			return fmt.Errorf("devices[%d].host is required", i)
		}
		if d.Version != "2c" {
			return fmt.Errorf("devices[%d].version: only \"2c\" is supported in Phase 1", i)
		}
		if _, ok := seen[d.ID]; ok {
			return fmt.Errorf("duplicate device id %q", d.ID)
		}
		seen[d.ID] = struct{}{}
	}
	return nil
}
