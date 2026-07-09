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
	SiteID       string        `yaml:"site_id"`
	PollInterval time.Duration `yaml:"poll_interval"`
	MaxWorkers   int           `yaml:"max_workers"`

	Admin     AdminConfig     `yaml:"admin"`
	SNMP      SNMPConfig      `yaml:"snmp"`
	Publisher PublisherConfig `yaml:"publisher"`
	Buffer    BufferConfig    `yaml:"buffer"`
	MQTT      MQTTConfig      `yaml:"mqtt"`

	Devices []DeviceConfig `yaml:"devices"`
}

// PublisherConfig selects the publish backend and poller publish timeout.
type PublisherConfig struct {
	Mode    string        `yaml:"mode"` // stdout | mqtt
	Timeout time.Duration `yaml:"timeout"`
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
	if c.Publisher.Mode == "" {
		c.Publisher.Mode = "stdout"
	}
	if c.Publisher.Timeout == 0 {
		c.Publisher.Timeout = 10 * time.Second
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

// applyEnvOverrides replaces secrets and optional MQTT settings from the environment.
func (c *Config) applyEnvOverrides() {
	for i := range c.Devices {
		key := communityEnvKey(c.Devices[i].ID)
		if v := os.Getenv(key); v != "" {
			c.Devices[i].Community = v
		}
	}
	if v := os.Getenv("MQTT_BROKER"); v != "" {
		c.MQTT.Broker = v
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

// MQTTPassword returns the MQTT password from the configured environment variable.
func (c *Config) MQTTPassword() string {
	if c.MQTT.PasswordEnv == "" {
		return ""
	}
	return os.Getenv(c.MQTT.PasswordEnv)
}

// MQTTInsecureSkipVerify reports whether TLS verification should be skipped (dev only).
func MQTTInsecureSkipVerify() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("MQTT_TLS_INSECURE")))
	return v == "1" || v == "true" || v == "yes"
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
	if c.Publisher.Timeout <= 0 {
		return fmt.Errorf("publisher.timeout must be positive")
	}
	switch c.Publisher.Mode {
	case "stdout", "mqtt":
	default:
		return fmt.Errorf("publisher.mode must be \"stdout\" or \"mqtt\"")
	}
	if c.Publisher.Mode == "mqtt" {
		if err := c.validateMQTT(); err != nil {
			return err
		}
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
			return fmt.Errorf("devices[%d].version: only \"2c\" is supported", i)
		}
		if _, ok := seen[d.ID]; ok {
			return fmt.Errorf("duplicate device id %q", d.ID)
		}
		seen[d.ID] = struct{}{}
	}
	return nil
}

func (c *Config) validateMQTT() error {
	if strings.TrimSpace(c.MQTT.Broker) == "" {
		return fmt.Errorf("mqtt.broker is required when publisher.mode is mqtt")
	}
	if strings.TrimSpace(c.MQTT.Username) == "" {
		return fmt.Errorf("mqtt.username is required when publisher.mode is mqtt")
	}
	if strings.TrimSpace(c.MQTT.PasswordEnv) == "" {
		return fmt.Errorf("mqtt.password_env is required when publisher.mode is mqtt")
	}
	if c.MQTTPassword() == "" {
		return fmt.Errorf("environment variable %q is required when publisher.mode is mqtt", c.MQTT.PasswordEnv)
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
	if c.MQTT.Reconnect.Initial <= 0 || c.MQTT.Reconnect.Max <= 0 {
		return fmt.Errorf("mqtt.reconnect.initial and max must be positive")
	}
	if c.MQTT.Reconnect.Max < c.MQTT.Reconnect.Initial {
		return fmt.Errorf("mqtt.reconnect.max must be >= initial")
	}
	insecure := MQTTInsecureSkipVerify()
	if strings.TrimSpace(c.MQTT.TLS.CAFile) == "" && !insecure {
		return fmt.Errorf("mqtt.tls.ca_file is required unless MQTT_TLS_INSECURE is set")
	}
	return nil
}
