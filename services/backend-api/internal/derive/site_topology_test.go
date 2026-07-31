package derive

import (
	"testing"
)

func TestEvaluateSiteTopology_DownstreamImpactedByCore(t *testing.T) {
	topology := BuildSiteTopologyIndex([]SiteTopologyNode{
		{Name: "do-core", HubDeviceIDs: []string{"do-core"}},
		{Name: "site-a-mdf", UpstreamSiteIDs: []string{"do-core"}},
	})
	raw := map[string]SiteRawHealth{
		"do-core": {
			Devices: []SiteDeviceHealth{{
				InventoryDeviceID: "do-core",
				Projection: DeviceProjection{
					Status:        StatusCritical,
					StatusReason:  "direct_unreachable",
					AlertsEnabled: true,
				},
			}},
			Counts: SiteHealthCounts{CriticalCount: 1},
		},
		"site-a-mdf": {
			Devices: []SiteDeviceHealth{
				{Projection: DeviceProjection{Status: StatusCritical, StatusReason: "direct_unreachable", AlertsEnabled: true}},
				{Projection: DeviceProjection{Status: StatusCritical, StatusReason: "direct_unreachable", AlertsEnabled: true}},
			},
			Counts: SiteHealthCounts{CriticalCount: 2},
		},
	}

	eval := EvaluateSiteTopology(topology, raw)
	state := eval.States["site-a-mdf"]
	if !state.SiteDependencyImpacted {
		t.Fatalf("expected site dependency impact, got %+v", state)
	}
	if len(state.RootCauseSiteIDs) != 1 || state.RootCauseSiteIDs[0] != "do-core" {
		t.Fatalf("root causes=%v", state.RootCauseSiteIDs)
	}

	overlay := ApplySiteDependencyOverlay(state, raw["site-a-mdf"].Devices[0].Projection)
	if overlay.Status != StatusUnknown || overlay.StatusReason != ReasonUpstreamSiteUnreachable {
		t.Fatalf("overlay=%+v", overlay)
	}

	counts := ProjectedSiteHealth([]DeviceProjection{overlay, overlay})
	if counts.CriticalCount != 0 || counts.UnknownCount != 2 || counts.DependencyImpactedCount != 2 {
		t.Fatalf("counts=%+v", counts)
	}
}

func TestEvaluateSiteTopology_PartialFailureNotImpacted(t *testing.T) {
	topology := BuildSiteTopologyIndex([]SiteTopologyNode{
		{Name: "do-core", HubDeviceIDs: []string{"do-core"}},
		{Name: "site-a-mdf", UpstreamSiteIDs: []string{"do-core"}},
	})
	raw := map[string]SiteRawHealth{
		"do-core": {
			Devices: []SiteDeviceHealth{{
				InventoryDeviceID: "do-core",
				Projection:        DeviceProjection{Status: StatusCritical, StatusReason: "direct_unreachable", AlertsEnabled: true},
			}},
			Counts: SiteHealthCounts{CriticalCount: 1},
		},
		"site-a-mdf": {
			Devices: []SiteDeviceHealth{
				{Projection: DeviceProjection{Status: StatusHealthy, StatusReason: "reachable", AlertsEnabled: true}},
				{Projection: DeviceProjection{Status: StatusCritical, StatusReason: "direct_unreachable", AlertsEnabled: true}},
			},
			Counts: SiteHealthCounts{HealthyCount: 1, CriticalCount: 1},
		},
	}

	eval := EvaluateSiteTopology(topology, raw)
	if eval.States["site-a-mdf"].SiteDependencyImpacted {
		t.Fatal("expected no site dependency impact for partial failure")
	}
}
