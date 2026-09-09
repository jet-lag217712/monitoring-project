package handlers

import (
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
