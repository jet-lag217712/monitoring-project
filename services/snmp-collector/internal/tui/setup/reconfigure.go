package setup

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ReconfigureMode selects which setup wizard sections run during reconfiguration.
type ReconfigureMode string

const (
	ReconfigureNone  ReconfigureMode = ""
	ReconfigureFull  ReconfigureMode = "full"
	ReconfigureSites ReconfigureMode = "sites"
	ReconfigureUsers ReconfigureMode = "users"
)

// ParseReconfigureMode normalizes a reconfigure flag value.
func ParseReconfigureMode(raw string) (ReconfigureMode, error) {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "", "full":
		return ReconfigureFull, nil
	case "sites", "site":
		return ReconfigureSites, nil
	case "users", "user":
		return ReconfigureUsers, nil
	case "none", "off":
		return ReconfigureNone, nil
	default:
		return "", fmt.Errorf("unknown reconfigure mode %q", raw)
	}
}

// ReconfigureModeFromEnv reads EQUATE_SETUP_RECONFIGURE when set.
func ReconfigureModeFromEnv() ReconfigureMode {
	raw := strings.TrimSpace(os.Getenv("EQUATE_SETUP_RECONFIGURE"))
	if raw == "" {
		return ReconfigureNone
	}
	mode, err := ParseReconfigureMode(raw)
	if err != nil {
		return ReconfigureFull
	}
	return mode
}

// preloadExistingState hydrates wizard fields from an already-configured deploy dir.
func preloadExistingState(m *model) {
	if m.reconfigureMode == ReconfigureNone {
		return
	}
	manifest, err := LoadManifest(m.deployDir)
	if err != nil {
		return
	}
	m.sites = manifest.Sites
	m.siteCountInput.SetValue(strconv.Itoa(len(manifest.Sites)))
	m.resizeSiteInputs(len(manifest.Sites))
	for i, spec := range manifest.Sites {
		if i < len(m.siteIDInputs) {
			m.siteIDInputs[i].SetValue(spec.SiteID)
		}
		if i < len(m.cidrInputs) {
			m.cidrInputs[i].SetValue(spec.CIDR)
		}
	}
	if manifest.ProbeRate > 0 {
		m.probeRate = strconv.FormatFloat(manifest.ProbeRate, 'f', -1, 64)
	}
	if manifest.ProbeBurst > 0 {
		m.probeBurst = strconv.Itoa(manifest.ProbeBurst)
	}
	if err := loadEnvFile(envPath(m.deployDir)); err == nil {
		if v := strings.TrimSpace(os.Getenv("SNMP_COMMUNITY")); v != "" && len(m.envInputs) > 0 {
			m.envInputs[0].SetValue(v)
		}
		if v := strings.TrimSpace(os.Getenv("SNMP_DISCOVERY_COMMUNITY")); v != "" && len(m.envInputs) > 1 {
			m.envInputs[1].SetValue(v)
		}
	}
}
