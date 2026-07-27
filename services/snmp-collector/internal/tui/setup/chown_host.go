package setup

import (
	"fmt"
	"os"
	"path/filepath"
)

const collectorUID = 65532
const collectorGID = 65532

func ensureSiteOwnershipAppliance(deployDir string, profile Profile, specs []SiteSpec) error {
	project := composeProjectName(deployDir, profile)
	for _, spec := range specs {
		runDir := spec.RunDir(deployDir)
		runMount := runDir + ":/run/snmp-collector"
		socket := filepath.Join(runDir, "control.sock")
		_ = os.Remove(socket)
		if err := runBusybox(runMount, "rm", "-f", "/run/snmp-collector/control.sock"); err != nil {
			return fmt.Errorf("%s run dir: %w", spec.SiteID, err)
		}
		if err := chownContainerPath(runMount, "/run/snmp-collector", false); err != nil {
			return fmt.Errorf("%s run dir: %w", spec.SiteID, err)
		}
		volume := project + "_" + spec.VolumeName()
		if err := chownDockerVolume(volume); err != nil {
			return fmt.Errorf("%s state volume: %w", spec.SiteID, err)
		}
		if err := chownApplianceSiteArtifacts(deployDir, spec); err != nil {
			return fmt.Errorf("%s managed artifacts: %w", spec.SiteID, err)
		}
	}
	return nil
}

func chownDockerVolume(volumeName string) error {
	mount := volumeName + ":/data"
	if err := chownContainerPath(mount, "/data", true); err != nil {
		return err
	}
	return runBusybox(mount, "chmod", "-R", "u+rwX", "/data")
}
