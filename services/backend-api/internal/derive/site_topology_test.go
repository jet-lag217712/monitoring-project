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

func crit(id string) SiteDeviceHealth {
	return SiteDeviceHealth{
		InventoryDeviceID: id,
		Projection: DeviceProjection{
			Status:        StatusCritical,
			StatusReason:  "direct_unreachable",
			AlertsEnabled: true,
		},
	}
}

func unknownDevice(id, reason string, unavailable []string) SiteDeviceHealth {
	return SiteDeviceHealth{
		InventoryDeviceID: id,
		Projection: DeviceProjection{
			Status:                 StatusUnknown,
			StatusReason:           reason,
			UnavailableUpstreamIDs: unavailable,
			AlertsEnabled:          true,
		},
	}
}

func TestEvaluateSiteTopology_NestedIDFImpactedWhenCoreDown(t *testing.T) {
	topology := BuildSiteTopologyIndex([]SiteTopologyNode{
		{Name: "do-core", HubDeviceIDs: []string{"do-core"}},
		{Name: "site-a-mdf"},
		{Name: "site-a-idf1"},
		{Name: "site-b-mdf"},
	})
	raw := map[string]SiteRawHealth{
		"do-core": {
			Devices: []SiteDeviceHealth{crit("do-core")},
			Counts:  SiteHealthCounts{CriticalCount: 1},
		},
		"site-a-mdf": {
			Devices: []SiteDeviceHealth{crit("site-a-mdf")},
			Counts:  SiteHealthCounts{CriticalCount: 1},
		},
		"site-a-idf1": {
			Devices: []SiteDeviceHealth{crit("site-a-idf1")},
			Counts:  SiteHealthCounts{CriticalCount: 1},
		},
		"site-b-mdf": {
			Devices: []SiteDeviceHealth{crit("site-b-mdf")},
			Counts:  SiteHealthCounts{CriticalCount: 1},
		},
	}

	eval := EvaluateSiteTopology(topology, raw)
	for _, name := range []string{"site-a-mdf", "site-a-idf1", "site-b-mdf"} {
		state := eval.States[name]
		if !state.SiteDependencyImpacted {
			t.Fatalf("%s expected impacted, got %+v", name, state)
		}
		if len(state.RootCauseSiteIDs) != 1 || state.RootCauseSiteIDs[0] != "do-core" {
			t.Fatalf("%s root causes=%v", name, state.RootCauseSiteIDs)
		}
		overlay := ApplySiteDependencyOverlay(state, crit(name).Projection)
		if overlay.Status != StatusUnknown || overlay.StatusReason != ReasonUpstreamSiteUnreachable {
			t.Fatalf("%s overlay=%+v", name, overlay)
		}
	}
	if eval.States["do-core"].SiteDependencyImpacted {
		t.Fatal("do-core must remain the critical root cause")
	}
	if got := eval.States["site-a-idf1"].UpstreamSiteIDs; len(got) != 1 || got[0] != "site-a-mdf" {
		t.Fatalf("inferred idf upstream=%v", got)
	}
}

func TestEvaluateSiteTopology_CombinedSiteUnknownsCountTowardCascade(t *testing.T) {
	topology := BuildSiteTopologyIndex([]SiteTopologyNode{
		{Name: "do-core", HubDeviceIDs: []string{"do-core"}},
		{Name: "site-a-mdf", UpstreamSiteIDs: []string{"do-core"}},
	})
	raw := map[string]SiteRawHealth{
		"do-core": {
			Devices: []SiteDeviceHealth{crit("do-core")},
			Counts:  SiteHealthCounts{CriticalCount: 1},
		},
		"site-a-mdf": {
			Devices: []SiteDeviceHealth{
				crit("site-a-mdf"),
				unknownDevice("site-a-idf1", "upstream_unreachable", []string{"site-a-mdf"}),
				unknownDevice("site-a-idf2", "upstream_unreachable", []string{"site-a-mdf"}),
			},
			Counts: SiteHealthCounts{CriticalCount: 1, UnknownCount: 2},
		},
	}

	eval := EvaluateSiteTopology(topology, raw)
	state := eval.States["site-a-mdf"]
	if !state.SiteDependencyImpacted {
		t.Fatalf("expected combined site impact, got %+v", state)
	}
	overlay := ApplySiteDependencyOverlay(state, raw["site-a-mdf"].Devices[0].Projection)
	if overlay.Status != StatusUnknown || overlay.StatusReason != ReasonUpstreamSiteUnreachable {
		t.Fatalf("mdf overlay=%+v", overlay)
	}
}
