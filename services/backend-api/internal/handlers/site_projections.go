package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/equate/ogsd/services/backend-api/internal/derive"
	"github.com/equate/ogsd/services/backend-api/internal/models"
	"github.com/equate/ogsd/services/backend-api/internal/store"
	"github.com/google/uuid"
)

type siteProjectionBundle struct {
	counts  derive.SiteHealthCounts
	state   derive.SiteDependencyState
	devices map[string]projectedDevice
}

type projectedDevice struct {
	row  store.DeviceRow
	proj derive.DeviceProjection
}

type siteProjectionIndex struct {
	bySiteID map[uuid.UUID]siteProjectionBundle
	byName   map[string]siteProjectionBundle
}

func buildSiteProjections(sites []store.SiteRow, devices []store.DeviceRow, now time.Time, onlineThreshold time.Duration) siteProjectionIndex {
	topologyNodes := make([]derive.SiteTopologyNode, 0, len(sites))
	siteByID := make(map[uuid.UUID]store.SiteRow, len(sites))
	for _, site := range sites {
		siteByID[site.ID] = site
		topologyNodes = append(topologyNodes, derive.SiteTopologyNode{
			Name:            site.Name,
			UpstreamSiteIDs: append([]string(nil), site.UpstreamSiteIDs...),
			HubDeviceIDs:    append([]string(nil), site.HubDeviceIDs...),
		})
	}
	topology := derive.BuildSiteTopologyIndex(topologyNodes)

	bySiteDevices := make(map[uuid.UUID][]store.DeviceRow)
	for _, device := range devices {
		bySiteDevices[device.SiteID] = append(bySiteDevices[device.SiteID], device)
	}

	raw := make(map[string]derive.SiteRawHealth, len(sites))
	for _, site := range sites {
		devs := bySiteDevices[site.ID]
		rawSite := derive.SiteRawHealth{Devices: make([]derive.SiteDeviceHealth, 0, len(devs))}
		for _, device := range devs {
			online := derive.DeviceOnline(device.Status, device.LastSeen, now, onlineThreshold)
			proj := projectDevice(device, online)
			rawSite.Devices = append(rawSite.Devices, derive.SiteDeviceHealth{
				InventoryDeviceID: device.InventoryDeviceID,
				Projection:        proj,
			})
			rawSite.Counts.Accumulate(proj.Status, proj.UnavailableUpstreamIDs, proj.AlertsEnabled)
		}
		raw[site.Name] = rawSite
	}

	eval := derive.EvaluateSiteTopology(topology, raw)
	// #region agent log
	debugLogSiteTopologyEval(sites, raw, eval)
	// #endregion

	out := siteProjectionIndex{
		bySiteID: make(map[uuid.UUID]siteProjectionBundle, len(sites)),
		byName:   make(map[string]siteProjectionBundle, len(sites)),
	}
	for _, site := range sites {
		state := eval.States[site.Name]
		devs := bySiteDevices[site.ID]
		projected := make(map[string]projectedDevice, len(devs))
		overlayed := make([]derive.DeviceProjection, 0, len(devs))
		for _, device := range devs {
			online := derive.DeviceOnline(device.Status, device.LastSeen, now, onlineThreshold)
			proj := derive.ApplySiteDependencyOverlay(state, projectDevice(device, online))
			overlayed = append(overlayed, proj)
			key := derive.DeviceMapKey(device.IPAddress, device.Hostname)
			projected[key] = projectedDevice{row: device, proj: proj}
		}
		bundle := siteProjectionBundle{
			counts:  derive.ProjectedSiteHealth(overlayed),
			state:   state,
			devices: projected,
		}
		out.bySiteID[site.ID] = bundle
		out.byName[site.Name] = bundle
	}
	return out
}

func siteTopologyFields(state derive.SiteDependencyState) ([]string, []string, []string, bool) {
	return emptyToNil(state.UpstreamSiteIDs),
		emptyToNil(state.UnavailableUpstreamSiteIDs),
		emptyToNil(state.RootCauseSiteIDs),
		state.SiteDependencyImpacted
}

func deviceSiteTopologyFields(state derive.SiteDependencyState) ([]string, []string, []string) {
	return emptyToNil(state.UpstreamSiteIDs),
		emptyToNil(state.UnavailableUpstreamSiteIDs),
		emptyToNil(state.RootCauseSiteIDs)
}

func toDeviceSummaryWithSite(d store.DeviceRow, proj derive.DeviceProjection, state derive.SiteDependencyState) models.DeviceSummary {
	upstreamSites, unavailableSites, rootCauseSites := deviceSiteTopologyFields(state)
	summary := toDeviceSummary(d, proj)
	summary.UpstreamSiteIDs = upstreamSites
	summary.UnavailableUpstreamSiteIDs = unavailableSites
	summary.RootCauseSiteIDs = rootCauseSites
	return summary
}

func (idx siteProjectionIndex) bundleForSite(site store.SiteRow) siteProjectionBundle {
	if bundle, ok := idx.bySiteID[site.ID]; ok {
		return bundle
	}
	return siteProjectionBundle{state: derive.SiteDependencyState{UpstreamSiteIDs: append([]string(nil), site.UpstreamSiteIDs...)}}
}

func (idx siteProjectionIndex) bundleForSiteName(name string) (siteProjectionBundle, bool) {
	bundle, ok := idx.byName[name]
	return bundle, ok
}

// #region agent log
var lastSiteTopoDebugSig string

func debugLogSiteTopologyEval(sites []store.SiteRow, raw map[string]derive.SiteRawHealth, eval derive.SiteTopologyEval) {
	type deviceSnap struct {
		ID     string `json:"id"`
		Status int    `json:"status"`
		Reason string `json:"reason"`
	}
	type siteSnap struct {
		Name                 string       `json:"name"`
		Upstreams            []string     `json:"upstreams"`
		Hubs                 []string     `json:"hubs"`
		DeviceCount          int          `json:"device_count"`
		Critical             int          `json:"critical"`
		Unknown              int          `json:"unknown"`
		UnavailableUpstreams []string     `json:"unavailable_upstreams"`
		RootCauses           []string     `json:"root_causes"`
		Impacted             bool         `json:"impacted"`
		Devices              []deviceSnap `json:"devices"`
	}
	out := make([]siteSnap, 0, len(sites))
	for _, site := range sites {
		rawSite := raw[site.Name]
		state := eval.States[site.Name]
		snap := siteSnap{
			Name:                 site.Name,
			Upstreams:            append([]string(nil), state.UpstreamSiteIDs...),
			Hubs:                 append([]string(nil), site.HubDeviceIDs...),
			DeviceCount:          len(rawSite.Devices),
			Critical:             rawSite.Counts.CriticalCount,
			Unknown:              rawSite.Counts.UnknownCount,
			UnavailableUpstreams: append([]string(nil), state.UnavailableUpstreamSiteIDs...),
			RootCauses:           append([]string(nil), state.RootCauseSiteIDs...),
			Impacted:             state.SiteDependencyImpacted,
		}
		for _, d := range rawSite.Devices {
			snap.Devices = append(snap.Devices, deviceSnap{
				ID:     d.InventoryDeviceID,
				Status: d.Projection.Status,
				Reason: d.Projection.StatusReason,
			})
		}
		out = append(out, snap)
	}
	data, err := json.Marshal(out)
	if err != nil {
		return
	}
	sig := string(data)
	if sig == lastSiteTopoDebugSig {
		return
	}
	lastSiteTopoDebugSig = sig
	body, err := json.Marshal(map[string]any{
		"sessionId":    "f7c9cd",
		"runId":        "post-fix",
		"hypothesisId": "A",
		"location":     "site_projections.go:EvaluateSiteTopology",
		"message":      "backend site topology evaluation",
		"data":         map[string]any{"sites": json.RawMessage(data)},
		"timestamp":    time.Now().UnixMilli(),
	})
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, "http://127.0.0.1:7535/ingest/67222a7b-79e8-4cfd-9a12-c85ccde20fea", bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Debug-Session-Id", "f7c9cd")
	go func() {
		client := &http.Client{Timeout: 500 * time.Millisecond}
		resp, err := client.Do(req)
		if err != nil {
			return
		}
		_ = resp.Body.Close()
	}()
}

// #endregion
