package config

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the backend API runtime configuration.
type Config struct {
	API             APIConfig      `yaml:"api"`
	Admin           AdminConfig    `yaml:"admin"`
	Auth            AuthConfig     `yaml:"auth"`
	Database        DatabaseConfig `yaml:"database"`
	OnlineThreshold time.Duration  `yaml:"online_threshold"`
}

// APIConfig controls the public REST server.
type APIConfig struct {
	Listen      string `yaml:"listen"`
	CORSOrigins string `yaml:"cors_origins"`
}

// AuthConfig controls Google OIDC validation for /api/*.
type AuthConfig struct {
	// Enabled protects /api/* with Google ID token validation when true.
	Enabled bool `yaml:"enabled"`
	// GoogleClientID is the OAuth Web client ID (aud claim).
	GoogleClientID string `yaml:"google_client_id"`
	// GoogleClientIDEnv optionally loads the client ID from an environment variable.
	GoogleClientIDEnv string `yaml:"google_client_id_env"`
}

// AdminConfig controls the admin HTTP server (metrics/health).
type AdminConfig struct {
	Listen string `yaml:"listen"`
}

// DatabaseConfig controls the PostgreSQL connection pool.
type DatabaseConfig struct {
	URLEnv      string        `yaml:"url_env"`
	MaxConns    int32         `yaml:"max_conns"`
	MinConns    int32         `yaml:"min_conns"`
	MaxLifetime time.Duration `yaml:"max_lifetime"`
}

// Load reads and validates an API config file.
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

	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.API.Listen == "" {
		c.API.Listen = ":8000"
	}
	if c.Admin.Listen == "" {
		c.Admin.Listen = ":9092"
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
	if c.OnlineThreshold == 0 {
		c.OnlineThreshold = 5 * time.Minute
	}
	if c.Auth.GoogleClientIDEnv == "" {
		c.Auth.GoogleClientIDEnv = "GOOGLE_CLIENT_ID"
	}
	if strings.TrimSpace(c.Auth.GoogleClientID) == "" {
		c.Auth.GoogleClientID = strings.TrimSpace(os.Getenv(c.Auth.GoogleClientIDEnv))
	}
}

// AuthEnabled reports whether Google OIDC protection is active.
func (c *Config) AuthEnabled() bool {
	return c.Auth.Enabled
}

// GoogleClientID returns the configured OAuth client ID.
func (c *Config) GoogleClientID() string {
	return strings.TrimSpace(c.Auth.GoogleClientID)
}

// DatabaseURL returns the PostgreSQL URL from the configured environment variable.
func (c *Config) DatabaseURL() string {
	if c.Database.URLEnv == "" {
		return ""
	}
	return os.Getenv(c.Database.URLEnv)
}

// CORSOriginList returns allowed CORS origins (trimmed, non-empty).
func (c *Config) CORSOriginList() []string {
	if strings.TrimSpace(c.API.CORSOrigins) == "" {
		return nil
	}
	parts := strings.Split(c.API.CORSOrigins, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Validate checks required fields.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.API.Listen) == "" {
		return fmt.Errorf("api.listen is required")
	}
	if strings.TrimSpace(c.Admin.Listen) == "" {
		return fmt.Errorf("admin.listen is required")
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
	if c.OnlineThreshold <= 0 {
		return fmt.Errorf("online_threshold must be positive")
	}
	if c.Auth.Enabled && c.GoogleClientID() == "" {
		return fmt.Errorf("auth.google_client_id or %s is required when auth.enabled is true", c.Auth.GoogleClientIDEnv)
	}
	return nil
}
