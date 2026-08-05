package setup

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// equateSiteNamespace matches deployments/production/appliance/scripts/sync-site-topology.sh.
var equateSiteNamespace = uuid.NewSHA1(uuid.NameSpaceDNS, []byte("equate-ogsd"))

// SiteUUID returns the deterministic Postgres sites.id for a site_id string.
func SiteUUID(siteID string) uuid.UUID {
	return uuid.NewSHA1(equateSiteNamespace, []byte("site:"+siteID))
}

// RemoveSiteFromManifest drops siteID from the manifest, strips it from other
// sites' upstream_site_ids, and updates site_count. The removed SiteSpec is returned.
func RemoveSiteFromManifest(manifest Manifest, siteID string) (Manifest, SiteSpec, error) {
	siteID = strings.TrimSpace(siteID)
	if siteID == "" {
		return Manifest{}, SiteSpec{}, fmt.Errorf("site id is required")
	}
	if len(manifest.Sites) <= minSiteCount {
		return Manifest{}, SiteSpec{}, fmt.Errorf("cannot delete the last remaining site")
	}
	idx := -1
	var removed SiteSpec
	for i, spec := range manifest.Sites {
		if spec.SiteID == siteID {
			idx = i
			removed = spec
			break
		}
	}
	if idx < 0 {
		return Manifest{}, SiteSpec{}, fmt.Errorf("site %q not found in manifest", siteID)
	}
	remaining := make([]SiteSpec, 0, len(manifest.Sites)-1)
	for i, spec := range manifest.Sites {
		if i == idx {
			continue
		}
		spec.UpstreamSiteIDs = filterString(spec.UpstreamSiteIDs, siteID)
		remaining = append(remaining, spec)
	}
	for i := range remaining {
		remaining[i].Index = i + 1
	}
	manifest.Sites = remaining
	manifest.SiteCount = len(remaining)
	return manifest, removed, nil
}

func filterString(values []string, drop string) []string {
	if len(values) == 0 {
		return values
	}
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == drop {
			continue
		}
		out = append(out, v)
	}
	return out
}

// SiteDeleteSQL returns ordered DELETE statements that remove a site and its
// dependent rows. FKs have no ON DELETE CASCADE.
func SiteDeleteSQL(siteID string) string {
	sid := SiteUUID(siteID).String()
	return strings.Join([]string{
		fmt.Sprintf(`DELETE FROM device_health_history WHERE site_id = '%s';`, sid),
		fmt.Sprintf(`DELETE FROM device_health_current WHERE site_id = '%s';`, sid),
		fmt.Sprintf(`DELETE FROM device_temperature_readings WHERE device_id IN (SELECT id FROM devices WHERE site_id = '%s');`, sid),
		fmt.Sprintf(`DELETE FROM device_temperature_components WHERE device_id IN (SELECT id FROM devices WHERE site_id = '%s');`, sid),
		fmt.Sprintf(`DELETE FROM device_power_readings WHERE device_id IN (SELECT id FROM devices WHERE site_id = '%s');`, sid),
		fmt.Sprintf(`DELETE FROM device_power_components WHERE device_id IN (SELECT id FROM devices WHERE site_id = '%s');`, sid),
		fmt.Sprintf(`DELETE FROM interface_samples WHERE interface_id IN (SELECT id FROM interfaces WHERE device_id IN (SELECT id FROM devices WHERE site_id = '%s'));`, sid),
		fmt.Sprintf(`DELETE FROM metric_samples WHERE device_id IN (SELECT id FROM devices WHERE site_id = '%s');`, sid),
		fmt.Sprintf(`DELETE FROM alerts WHERE device_id IN (SELECT id FROM devices WHERE site_id = '%s');`, sid),
		fmt.Sprintf(`DELETE FROM interfaces WHERE device_id IN (SELECT id FROM devices WHERE site_id = '%s');`, sid),
		fmt.Sprintf(`DELETE FROM collector_status_current WHERE site_id = '%s';`, sid),
		fmt.Sprintf(`DELETE FROM collector_heartbeat_history WHERE site_id = '%s';`, sid),
		fmt.Sprintf(`DELETE FROM collectors WHERE site_id = '%s';`, sid),
		fmt.Sprintf(`DELETE FROM devices WHERE site_id = '%s';`, sid),
		fmt.Sprintf(`DELETE FROM sites WHERE id = '%s';`, sid),
	}, "\n")
}
