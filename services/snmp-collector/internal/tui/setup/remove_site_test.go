package setup

import (
	"strings"
	"testing"
)

func TestRemoveSiteFromManifest(t *testing.T) {
	manifest := Manifest{
		SiteCount: 3,
		Sites: []SiteSpec{
			{Index: 1, SiteID: "and", ServiceName: "snmp-collector-and", UpstreamSiteIDs: []string{"district"}},
			{Index: 2, SiteID: "district", ServiceName: "snmp-collector-district"},
			{Index: 3, SiteID: "bal", ServiceName: "snmp-collector-bal", UpstreamSiteIDs: []string{"district", "and"}},
		},
	}
	updated, removed, err := RemoveSiteFromManifest(manifest, "district")
	if err != nil {
		t.Fatal(err)
	}
	if removed.SiteID != "district" {
		t.Fatalf("removed=%q", removed.SiteID)
	}
	if updated.SiteCount != 2 || len(updated.Sites) != 2 {
		t.Fatalf("site_count=%d sites=%d", updated.SiteCount, len(updated.Sites))
	}
	if updated.Sites[0].SiteID != "and" || updated.Sites[1].SiteID != "bal" {
		t.Fatalf("remaining=%v", updated.Sites)
	}
	if len(updated.Sites[0].UpstreamSiteIDs) != 0 {
		t.Fatalf("and upstreams=%v", updated.Sites[0].UpstreamSiteIDs)
	}
	if len(updated.Sites[1].UpstreamSiteIDs) != 1 || updated.Sites[1].UpstreamSiteIDs[0] != "and" {
		t.Fatalf("bal upstreams=%v", updated.Sites[1].UpstreamSiteIDs)
	}
	if updated.Sites[0].Index != 1 || updated.Sites[1].Index != 2 {
		t.Fatalf("indexes=%d,%d", updated.Sites[0].Index, updated.Sites[1].Index)
	}
}

func TestRemoveSiteFromManifestLastSite(t *testing.T) {
	manifest := Manifest{SiteCount: 1, Sites: []SiteSpec{{SiteID: "only"}}}
	if _, _, err := RemoveSiteFromManifest(manifest, "only"); err == nil {
		t.Fatal("expected error for last site")
	}
}

func TestRemoveSiteFromManifestMissing(t *testing.T) {
	manifest := Manifest{
		SiteCount: 2,
		Sites: []SiteSpec{
			{SiteID: "a"},
			{SiteID: "b"},
		},
	}
	if _, _, err := RemoveSiteFromManifest(manifest, "missing"); err == nil {
		t.Fatal("expected missing site error")
	}
}

func TestManifestHasSite(t *testing.T) {
	manifest := Manifest{Sites: []SiteSpec{{SiteID: "campus-a"}}}
	if !ManifestHasSite(manifest, "campus-a") {
		t.Fatal("expected campus-a present")
	}
	if ManifestHasSite(manifest, "ghost") {
		t.Fatal("expected ghost absent")
	}
}

func TestSiteUUIDStable(t *testing.T) {
	a := SiteUUID("district")
	b := SiteUUID("district")
	if a != b {
		t.Fatalf("unstable uuid: %s vs %s", a, b)
	}
	if SiteUUID("and") == a {
		t.Fatal("expected distinct uuids")
	}
}

func TestSiteDeleteSQLContainsOrderedDeletes(t *testing.T) {
	sql := SiteDeleteSQL("district")
	sid := SiteUUID("district").String()
	if !strings.Contains(sql, sid) {
		t.Fatalf("missing site uuid in sql")
	}
	if !strings.Contains(sql, "OR name = 'district'") {
		t.Fatalf("expected name match for orphan rows: %s", sql)
	}
	devicesIdx := strings.Index(sql, "DELETE FROM devices")
	sitesIdx := strings.Index(sql, "DELETE FROM sites")
	if devicesIdx < 0 || sitesIdx < 0 || devicesIdx > sitesIdx {
		t.Fatalf("devices must be deleted before sites")
	}
}

func TestOrphanSitesPruneSQL(t *testing.T) {
	if OrphanSitesPruneSQL(nil) != "" {
		t.Fatal("empty keep list must refuse prune")
	}
	sql := OrphanSitesPruneSQL([]string{"campus-a", "campus-b", "campus-a"})
	if !strings.Contains(sql, "name NOT IN ('campus-a', 'campus-b')") {
		t.Fatalf("unexpected keep list: %s", sql)
	}
	devicesIdx := strings.Index(sql, "DELETE FROM devices")
	sitesIdx := strings.Index(sql, "DELETE FROM sites WHERE name NOT IN")
	if devicesIdx < 0 || sitesIdx < 0 || devicesIdx > sitesIdx {
		t.Fatalf("devices must be deleted before sites")
	}
	for _, table := range []string{
		"device_health_history",
		"device_health_current",
		"collectors",
		"devices",
		"sites",
	} {
		if !strings.Contains(sql, "DELETE FROM "+table) {
			t.Fatalf("missing delete for %s", table)
		}
	}
}

func TestApplyGlobalTemperatureRejectsOutOfRange(t *testing.T) {
	if err := ApplyGlobalTemperature(t.TempDir(), 999); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestSiteUUIDMatchesTopologyScript(t *testing.T) {
	// python: uuid.uuid5(uuid.uuid5(DNS, "equate-ogsd"), "site:district")
	want := "cee2380c-5095-5c94-a5a3-923c685744f6"
	if got := SiteUUID("district").String(); got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
