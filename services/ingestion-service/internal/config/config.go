package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the ingestion service runtime configuration.
type Config struct {
	Admin    AdminConfig    `yaml:"admin"`
	MQTT     MQTTConfig     `yaml:"mqtt"`
	Database DatabaseConfig `yaml:"database"`
}

// AdminConfig controls the admin HTTP server (metrics/health).
type AdminConfig struct {
	Listen string `yaml:"listen"`
}

// MQTTConfig controls the inbound MQTT/TLS subscriber.
type MQTTConfig struct {
	Broker      string          `yaml:"broker"`
	ClientID    string          `yaml:"client_id"`
	Username    string          `yaml:"username"`
	PasswordEnv string          `yaml:"password_env"`
	QoS         byte            `yaml:"qos"`
	Topic       string          `yaml:"topic"`
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

// DatabaseConfig controls the PostgreSQL connection pool.
type DatabaseConfig struct {
	URLEnv      string        `yaml:"url_env"`
	MaxConns    int32         `yaml:"max_conns"`
	MinConns    int32         `yaml:"min_conns"`
	MaxLifetime time.Duration `yaml:"max_lifetime"`
}

// Load reads and validates an ingestion config file.
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
	if c.Admin.Listen == "" {
		c.Admin.Listen = ":9091"
	}
	if c.MQTT.PasswordEnv == "" {
		c.MQTT.PasswordEnv = "MQTT_PASSWORD"
	}
	if c.MQTT.QoS == 0 {
		c.MQTT.QoS = 1
	}
	if c.MQTT.ClientID == "" {
		c.MQTT.ClientID = "ingestion"
	}
	if c.MQTT.Topic == "" {
		c.MQTT.Topic = "site/+/device/+/metric/#"
	}
	if c.MQTT.Username == "" {
		c.MQTT.Username = "ingestion"
	}
	if c.MQTT.Reconnect.Initial == 0 {
		c.MQTT.Reconnect.Initial = time.Second
	}
	if c.MQTT.Reconnect.Max == 0 {
		c.MQTT.Reconnect.Max = 60 * time.Second
	}
	if c.Database.URLEnv == "" {
		c.Database.URLEnv = "DATABASE_URL"
	}
	if c.Database.MaxConns == 0 {
		c.Database.MaxConns = 10
	}
	if c.Database.MinConns == 0 {
		c.Database.MinConns = 1
	}
	if c.Database.MaxLifetime == 0 {
		c.Database.MaxLifetime = time.Hour
	}
}

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

// DatabaseURL returns the PostgreSQL URL from the configured environment variable.
func (c *Config) DatabaseURL() string {
	if c.Database.URLEnv == "" {
		return ""
	}
	return os.Getenv(c.Database.URLEnv)
}

// MQTTInsecureSkipVerify reports whether TLS verification should be skipped (dev only).
func MQTTInsecureSkipVerify() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("MQTT_TLS_INSECURE")))
	return v == "1" || v == "true" || v == "yes"
}

// Validate checks required fields.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.MQTT.Broker) == "" {
		return fmt.Errorf("mqtt.broker is required")
	}
	if strings.TrimSpace(c.MQTT.Username) == "" {
		return fmt.Errorf("mqtt.username is required")
	}
	if strings.TrimSpace(c.MQTT.PasswordEnv) == "" {
		return fmt.Errorf("mqtt.password_env is required")
	}
	if c.MQTTPassword() == "" {
		return fmt.Errorf("environment variable %q is required", c.MQTT.PasswordEnv)
	}
	if c.MQTT.QoS != 1 {
		return fmt.Errorf("mqtt.qos must be 1")
	}
	if strings.TrimSpace(c.MQTT.Topic) == "" {
		return fmt.Errorf("mqtt.topic is required")
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
	if strings.TrimSpace(c.Database.URLEnv) == "" {
		return fmt.Errorf("database.url_env is required")
	}
	if c.DatabaseURL() == "" {
		return fmt.Errorf("environment variable %q is required", c.Database.URLEnv)
	}
	if c.Database.MaxConns <= 0 {
		return fmt.Errorf("database.max_conns must be positive")
	}
	if c.Database.MinConns < 0 {
		return fmt.Errorf("database.min_conns must be >= 0")
	}
	return nil
}
