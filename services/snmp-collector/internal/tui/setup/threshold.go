package setup

import (
	"fmt"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
)

// ApplyGlobalTemperature sets the temperature warning threshold on every site
// collector via the control-plane prepare → commit → reload path.
func ApplyGlobalTemperature(deployDir string, temp float64) error {
	if err := config.ValidateTemperatureWarningC(temp); err != nil {
		return err
	}
	manifest, err := LoadManifest(deployDir)
	if err != nil {
		return err
	}
	for _, spec := range manifest.Sites {
		if err := applyThresholdToSite(spec, deployDir, temp); err != nil {
			return err
		}
	}
	return nil
}

// FormatTemperatureApplied returns a human-readable success message.
func FormatTemperatureApplied(temp float64, siteCount int) string {
	return fmt.Sprintf("Threshold %.0f°C applied to %d site(s).", temp, siteCount)
}
