// Package update implements connected appliance update discovery, download,
// verification, and extraction. It does not apply upgrades; callers hand the
// extracted bundle to configure-vm.sh --upgrade.
package update

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	// DefaultChannelConfigPath is the appliance update channel config.
	DefaultChannelConfigPath = "/etc/equate/update-channel.conf"
	// DefaultDownloadDir stores downloaded .eqa artifacts.
	DefaultDownloadDir = "/var/lib/equate/downloads"
	// DefaultStagingDir is where verified .eqa contents are extracted.
	DefaultStagingDir = "/tmp/equate-staging/bundle"
	// EditionStandard is the Google-authenticated appliance line.
	EditionStandard = "standard"
	// EditionNoAuth is the isolated NoAuth appliance line (appliance-3).
	EditionNoAuth = "noauth"
)

// ChannelConfig is the on-appliance update channel settings.
type ChannelConfig struct {
	ChannelURL string
	Edition    string
}

// LoadChannelConfig reads an INI-like key=value file. Missing file returns
// (nil, nil) so callers can fall back to offline/local bundle mode.
func LoadChannelConfig(path string) (*ChannelConfig, error) {
	if path == "" {
		path = DefaultChannelConfigPath
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open update channel config: %w", err)
	}
	defer f.Close()

	cfg := &ChannelConfig{Edition: EditionStandard}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "channel_url":
			cfg.ChannelURL = val
		case "edition":
			cfg.Edition = strings.ToLower(val)
		}
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("read update channel config: %w", err)
	}
	if cfg.ChannelURL == "" {
		return nil, fmt.Errorf("update channel config missing channel_url")
	}
	if cfg.Edition == "" {
		cfg.Edition = EditionStandard
	}
	switch cfg.Edition {
	case EditionStandard, EditionNoAuth:
	default:
		return nil, fmt.Errorf("unsupported edition %q (want %s or %s)", cfg.Edition, EditionStandard, EditionNoAuth)
	}
	return cfg, nil
}
