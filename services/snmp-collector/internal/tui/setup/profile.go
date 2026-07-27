package setup

import (
	"fmt"
	"strings"
)

// Profile selects deployment-specific setup behavior (dev lab vs on-prem appliance).
type Profile string

const (
	ProfileDevVxrail Profile = "dev-vxrail"
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
	case "", "dev-vxrail", "vxrail", "dev":
		return ProfileDevVxrail, nil
	case "appliance", "prod-appliance", "production-appliance":
		return ProfileAppliance, nil
	default:
		return "", fmt.Errorf("unknown setup profile %q (use dev-vxrail or appliance)", raw)
	}
}

// ProfileConfigFor returns deployment constants for a profile.
func ProfileConfigFor(profile Profile) ProfileConfig {
	switch profile {
	case ProfileAppliance:
		return ProfileConfig{
			ComposeProjectName:  "equate-appliance",
			CollectorIDPrefix:   "collector-appliance-",
			MQTTClientIDPrefix:  "appliance-collector-",
			AutoAcceptDiscovery: false,
			RequireAdminUser:    true,
			UserManagement:      true,
			BaseAdminPort:       baseAdminPort,
		}
	default:
		return ProfileConfig{
			ComposeProjectName:  "ogsd-development-vxrail",
			CollectorIDPrefix:   "collector-development-vxrail-",
			MQTTClientIDPrefix:  "development-vxrail-collector-",
			AutoAcceptDiscovery: true,
			RequireAdminUser:    false,
			UserManagement:      false,
			BaseAdminPort:       baseAdminPort,
		}
	}
}

func collectorIDForProfile(cfg ProfileConfig, siteID string) string {
	return cfg.CollectorIDPrefix + siteID
}

func mqttClientIDForProfile(cfg ProfileConfig, siteID string) string {
	return cfg.MQTTClientIDPrefix + siteID
}
