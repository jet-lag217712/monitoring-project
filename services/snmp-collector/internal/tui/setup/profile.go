package setup

import (
	"fmt"
	"strings"
)

// Profile selects setup behavior. The supported product profile is appliance.
type Profile string

const (
	ProfileAppliance Profile = "appliance"
)

// ProfileConfig holds per-profile naming and wizard behavior.
type ProfileConfig struct {
	ComposeProjectName  string
	CollectorIDPrefix   string
	MQTTClientIDPrefix  string
	AutoAcceptDiscovery bool
	RequireAdminUser    bool
	UserManagement      bool
	BaseAdminPort       int
}

// ParseProfile normalizes a profile flag value.
func ParseProfile(raw string) (Profile, error) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", "appliance", "prod-appliance", "production-appliance":
		return ProfileAppliance, nil
	default:
		return "", fmt.Errorf("unknown setup profile %q (use appliance)", raw)
	}
}

// ProfileConfigFor returns deployment constants for a profile.
func ProfileConfigFor(_ Profile) ProfileConfig {
	return ProfileConfig{
		ComposeProjectName:  "equate-appliance",
		CollectorIDPrefix:   "collector-appliance-",
		MQTTClientIDPrefix:  "appliance-collector-",
		AutoAcceptDiscovery: false,
		RequireAdminUser:    true,
		UserManagement:      true,
		BaseAdminPort:       baseAdminPort,
	}
}

func collectorIDForProfile(cfg ProfileConfig, siteID string) string {
	return cfg.CollectorIDPrefix + siteID
}

func mqttClientIDForProfile(cfg ProfileConfig, siteID string) string {
	return cfg.MQTTClientIDPrefix + siteID
}
