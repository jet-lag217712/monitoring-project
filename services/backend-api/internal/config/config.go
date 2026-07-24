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

// AuthConfig controls browser authentication for /api/*.
type AuthConfig struct {
	// Enabled protects /api/* when true.
	Enabled bool `yaml:"enabled"`
	// Mode is local, google_bearer, google_session, oidc, google, or disabled.
	// External provider modes are configured by the appliance control plane and
	// added without public configuration APIs.
	Mode string `yaml:"mode"`
	// CookieName and SessionTTL apply to local browser sessions.
	CookieName   string        `yaml:"cookie_name"`
	SessionTTL   time.Duration `yaml:"session_ttl"`
	CookieSecure bool          `yaml:"cookie_secure"`
	// GoogleClientID is the OAuth Web client ID (aud claim).
	GoogleClientID string `yaml:"google_client_id"`
	// GoogleClientIDEnv optionally loads the client ID from an environment variable.
	GoogleClientIDEnv string `yaml:"google_client_id_env"`
	// OIDC describes a browser redirect flow for generic OIDC and Google.
	// The client secret is deliberately supplied through a rendered runtime
	// environment variable, never stored in application.yaml.
	OIDCIssuer          string   `yaml:"oidc_issuer"`
	OIDCClientID        string   `yaml:"oidc_client_id"`
	OIDCClientSecretEnv string   `yaml:"oidc_client_secret_env"`
	OIDCRedirectURL     string   `yaml:"oidc_redirect_url"`
	AllowedEmails       []string `yaml:"allowed_emails"`
	AllowedGroups       []string `yaml:"allowed_groups"`
	// AllowedDomains is the Google Workspace hd claim allow-list used by the
	// appliance's GIS-to-session authentication mode.
	AllowedDomains []string `yaml:"allowed_domains"`
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
	if c.Auth.OIDCClientSecretEnv == "" {
		c.Auth.OIDCClientSecretEnv = "OIDC_CLIENT_SECRET"
	}
	if c.Auth.Mode == "" {
		if c.Auth.Enabled {
			c.Auth.Mode = "google_bearer"
		} else {
			c.Auth.Mode = "disabled"
		}
	}
	if c.Auth.CookieName == "" {
		c.Auth.CookieName = "__Host-equate_session"
	}
	if c.Auth.SessionTTL <= 0 {
		c.Auth.SessionTTL = 12 * time.Hour
	}
	if strings.TrimSpace(c.Auth.GoogleClientID) == "" {
		c.Auth.GoogleClientID = strings.TrimSpace(os.Getenv(c.Auth.GoogleClientIDEnv))
	}
}

// AuthEnabled reports whether Google OIDC protection is active.
func (c *Config) AuthEnabled() bool {
	return c.Auth.Enabled && c.Auth.Mode != "disabled"
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
	if c.Auth.SessionTTL <= 0 {
		return fmt.Errorf("auth.session_ttl must be positive")
	}
	switch c.Auth.Mode {
	case "disabled", "local", "google_bearer", "google_session", "oidc", "google":
	default:
		return fmt.Errorf("unsupported auth.mode %q", c.Auth.Mode)
	}
	if c.AuthEnabled() && c.Auth.Mode == "google_bearer" && c.GoogleClientID() == "" {
		return fmt.Errorf("auth.google_client_id or %s is required when auth.enabled is true", c.Auth.GoogleClientIDEnv)
	}
	if c.AuthEnabled() && c.Auth.Mode == "google_session" && c.GoogleClientID() == "" {
		return fmt.Errorf("auth.google_client_id or %s is required for google_session auth", c.Auth.GoogleClientIDEnv)
	}
	if c.AuthEnabled() && (c.Auth.Mode == "oidc" || c.Auth.Mode == "google") {
		if strings.TrimSpace(c.Auth.OIDCIssuer) == "" || strings.TrimSpace(c.Auth.OIDCClientID) == "" || strings.TrimSpace(c.Auth.OIDCRedirectURL) == "" {
			return fmt.Errorf("auth.oidc_issuer, auth.oidc_client_id, and auth.oidc_redirect_url are required for %s auth", c.Auth.Mode)
		}
		if strings.TrimSpace(os.Getenv(c.Auth.OIDCClientSecretEnv)) == "" {
			return fmt.Errorf("environment variable %q is required for %s auth", c.Auth.OIDCClientSecretEnv, c.Auth.Mode)
		}
		if len(c.Auth.AllowedEmails) == 0 && len(c.Auth.AllowedGroups) == 0 {
			return fmt.Errorf("auth.allowed_emails or auth.allowed_groups is required for %s auth", c.Auth.Mode)
		}
	}
	return nil
}
