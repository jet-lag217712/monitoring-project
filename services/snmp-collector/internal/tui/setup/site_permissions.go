package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func finalizeSiteArtifactsPermissions(profile Profile, deployDir string, specs []SiteSpec) error {
	if profile != ProfileAppliance {
		return nil
	}
	sitesRoot := filepath.Join(deployDir, "sites")
	if err := os.Chmod(sitesRoot, 0o755); err != nil && !os.IsNotExist(err) {
		return err
	}
	for _, spec := range specs {
		for _, dir := range []string{
			spec.siteRoot(deployDir),
			filepath.Dir(spec.ConfigPath(deployDir)),
			spec.RunDir(deployDir),
		} {
			if err := os.Chmod(dir, 0o755); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		if err := chownApplianceSiteArtifacts(deployDir, spec); err != nil {
			return fmt.Errorf("%s: %w", spec.SiteID, err)
		}
	}
	return nil
}

// chownApplianceSiteArtifacts fixes bind-mounted site files for the collector
// container user (65532). Uses direct chown when setup runs as root.
func chownApplianceSiteArtifacts(deployDir string, spec SiteSpec) error {
	if err := fixCollectorManagedPath(spec.ManagedInventoryPath(deployDir)); err != nil {
		return fmt.Errorf("managed inventory: %w", err)
	}
	cfgPath := spec.ConfigPath(deployDir)
	if os.Geteuid() == 0 {
		if err := os.Chown(cfgPath, collectorUID, collectorGID); err != nil {
			return fmt.Errorf("collector config: %w", err)
		}
		if err := os.Chmod(cfgPath, 0o640); err != nil {
			return fmt.Errorf("collector config mode: %w", err)
		}
		return nil
	}
	configDir := filepath.Dir(cfgPath)
	configMount := configDir + ":/mnt"
	cfgName := filepath.Base(cfgPath)
	if err := chownContainerPath(configMount, "/mnt/"+cfgName, false); err != nil {
		return fmt.Errorf("collector config: %w", err)
	}
	if err := runBusybox(configMount, "chmod", "640", "/mnt/"+cfgName); err != nil {
		return fmt.Errorf("collector config mode: %w", err)
	}
	return nil
}

// fixCollectorManagedPath ensures the managed inventory bind mount is owned by the
// collector container user. Host-side setup writes (seed, topology enrich) run as
// root and would otherwise leave root-owned 0600 files the collector cannot read.
func fixCollectorManagedPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	if os.Geteuid() == 0 {
		if err := os.Chown(dir, collectorUID, collectorGID); err != nil {
			return err
		}
		if err := os.Chmod(dir, 0o750); err != nil {
			return err
		}
		if _, err := os.Stat(path); err == nil {
			if err := os.Chown(path, collectorUID, collectorGID); err != nil {
				return err
			}
			if err := os.Chmod(path, 0o600); err != nil {
				return err
			}
		}
		return nil
	}
	mount := dir + ":/mnt"
	if err := chownContainerPath(mount, "/mnt", true); err != nil {
		return err
	}
	if err := runBusybox(mount, "chmod", "750", "/mnt"); err != nil {
		return err
	}
	base := filepath.Base(path)
	if _, err := os.Stat(path); err == nil {
		if err := runBusybox(mount, "chmod", "600", "/mnt/"+base); err != nil {
			return err
		}
	}
	return nil
}
