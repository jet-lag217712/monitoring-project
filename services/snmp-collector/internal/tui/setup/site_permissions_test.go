package setup

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestFixCollectorManagedPathRequiresRootOrDocker(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "managed-inventory.yaml")
	if err := os.WriteFile(path, []byte("devices: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if os.Geteuid() == 0 {
		if err := fixCollectorManagedPath(path); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		stat, ok := info.Sys().(*syscall.Stat_t)
		if !ok {
			t.Fatal("expected syscall.Stat_t")
		}
		if stat.Uid != collectorUID {
			t.Fatalf("uid=%d", stat.Uid)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("mode=%o", info.Mode().Perm())
		}
		return
	}
	t.Skip("non-root: fixCollectorManagedPath requires docker in integration environments")
}

func TestFinalizeSiteArtifactsPermissionsAppliance(t *testing.T) {
	if os.Geteuid() != 0 {
		t.Skip("requires root to chown site artifacts")
	}
	deployDir := t.TempDir()
	spec := SiteSpec{SiteID: "site-001", CollectorID: "c1", MQTTClientID: "m1", ServiceName: "snmp-collector-site-001", CIDR: "10.0.0.0/24", AdminPort: 19090}
	if err := os.MkdirAll(filepath.Join(deployDir, "sites", spec.SiteID, "configs"), 0o750); err != nil {
		t.Fatal(err)
	}
	cfgPath := spec.ConfigPath(deployDir)
	if err := os.WriteFile(cfgPath, []byte("site_id: site-001\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	managedPath := spec.ManagedInventoryPath(deployDir)
	if err := os.MkdirAll(filepath.Dir(managedPath), 0o750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(managedPath, []byte("devices: []\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := finalizeSiteArtifactsPermissions(ProfileAppliance, deployDir, []SiteSpec{spec}); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("cfg mode=%o", info.Mode().Perm())
	}
	minfo, err := os.Stat(managedPath)
	if err != nil {
		t.Fatal(err)
	}
	if minfo.Mode().Perm() != 0o600 {
		t.Fatalf("managed mode=%o", minfo.Mode().Perm())
	}
	stat, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		t.Fatal("expected syscall.Stat_t")
	}
	if stat.Uid != collectorUID {
		t.Fatalf("cfg uid=%d", stat.Uid)
	}
}
