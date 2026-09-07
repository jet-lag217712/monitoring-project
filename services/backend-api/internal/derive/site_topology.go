package derive

import (
	"sort"
	"strings"
)

// ReasonUpstreamSiteUnreachable is the API reason when a device is unreachable
// because every configured upstream site path is unavailable.
const ReasonUpstreamSiteUnreachable = "upstream_site_unreachable"

// SiteTopologyNode is one site in the cross-collector dependency graph.
type SiteTopologyNode struct {
	Name            string
	UpstreamSiteIDs []string
	HubDeviceIDs    []string
}

// SiteDeviceHealth is raw per-device health before site-level overlay.
type SiteDeviceHealth struct {
	InventoryDeviceID string
	Projection        DeviceProjection
}

// SiteRawHealth is pre-overlay health evidence for one site.
type SiteRawHealth struct {
	Devices []SiteDeviceHealth
	Counts  SiteHealthCounts
}

// SiteDependencyState is evaluated cross-site dependency evidence.
type SiteDependencyState struct {
	UpstreamSiteIDs            []string
	UnavailableUpstreamSiteIDs []string
	RootCauseSiteIDs           []string
	SiteDependencyImpacted     bool
}

// SiteTopologyEval holds dependency state keyed by site name.
type SiteTopologyEval struct {
	States map[string]SiteDependencyState
}

// BuildSiteTopologyIndex indexes site topology nodes by name.
func BuildSiteTopologyIndex(nodes []SiteTopologyNode) map[string]SiteTopologyNode {
	out := make(map[string]SiteTopologyNode, len(nodes))
	for _, node := range nodes {
		out[node.Name] = SiteTopologyNode{
			Name:            node.Name,
			UpstreamSiteIDs: append([]string(nil), node.UpstreamSiteIDs...),
			HubDeviceIDs:    append([]string(nil), node.HubDeviceIDs...),
		}
	}
	return out
}

// TopologicalSiteOrder returns site names in upstream-first order.
func TopologicalSiteOrder(index map[string]SiteTopologyNode) []string {
	ids := make([]string, 0, len(index))
	for id := range index {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	visited := make(map[string]struct{}, len(ids))
	var out []string
	var visit func(id string)
	visit = func(id string) {
		if _, ok := visited[id]; ok {
			return
		}
		visited[id] = struct{}{}
		for _, upstream := range index[id].UpstreamSiteIDs {
			visit(upstream)
		}
		out = append(out, id)
	}
	for _, id := range ids {
		visit(id)
	}
	return out
}

// CompleteMissingSiteUpstreams fills empty upstream_site_ids using the same
// core → MDF → IDF naming heuristic as setup. Operator-defined edges are kept.
func CompleteMissingSiteUpstreams(index map[string]SiteTopologyNode) map[string]SiteTopologyNode {
	out := make(map[string]SiteTopologyNode, len(index))
	idByLower := make(map[string]string, len(index))
	coreID := ""
	for name, node := range index {
		out[name] = SiteTopologyNode{
			Name:            node.Name,
			UpstreamSiteIDs: append([]string(nil), node.UpstreamSiteIDs...),
			HubDeviceIDs:    append([]string(nil), node.HubDeviceIDs...),
		}
		lower := strings.ToLower(name)
		idByLower[lower] = name
		if coreID == "" && (lower == "do-core" || strings.Contains(lower, "core")) {
			coreID = name
		}
	}
	for name, node := range out {
		if len(node.UpstreamSiteIDs) > 0 {
			continue
		}
		lower := strings.ToLower(name)
		switch {
		case lower == "do-core" || strings.Contains(lower, "core"):
			continue
		case strings.Contains(lower, "-idf"):
			if mdf := matchingMDFSiteID(lower, idByLower); mdf != "" {
				node.UpstreamSiteIDs = []string{mdf}
			} else if coreID != "" {
				node.UpstreamSiteIDs = []string{coreID}
			}
		case strings.Contains(lower, "-mdf"):
			if coreID != "" {
				node.UpstreamSiteIDs = []string{coreID}
			}
		}
		out[name] = node
	}
	return out
}

func matchingMDFSiteID(lowerIDF string, idByLower map[string]string) string {
	i := strings.Index(lowerIDF, "-idf")
	if i <= 0 {
		return ""
	}
	return idByLower[lowerIDF[:i]+"-mdf"]
}

// EvaluateSiteTopology computes unavailable upstream sites and downstream impact.
func EvaluateSiteTopology(index map[string]SiteTopologyNode, raw map[string]SiteRawHealth) SiteTopologyEval {
	index = CompleteMissingSiteUpstreams(index)
	order := TopologicalSiteOrder(index)
	unavailable := make(map[string]bool, len(index))
	states := make(map[string]SiteDependencyState, len(index))

	for _, name := range order {
		node := index[name]
		state := SiteDependencyState{
			UpstreamSiteIDs: append([]string(nil), node.UpstreamSiteIDs...),
		}
		if siteUnavailable(raw[name], node) {
			unavailable[name] = true
		}
		states[name] = state
	}

	for _, name := range order {
		node := index[name]
		state := states[name]
		for _, upstream := range node.UpstreamSiteIDs {
			if unavailable[upstream] {
				state.UnavailableUpstreamSiteIDs = append(state.UnavailableUpstreamSiteIDs, upstream)
			}
		}
		sort.Strings(state.UnavailableUpstreamSiteIDs)

		if len(node.UpstreamSiteIDs) > 0 &&
			len(state.UnavailableUpstreamSiteIDs) == len(node.UpstreamSiteIDs) &&
			siteShowsCascadeSignature(raw[name]) {
			state.SiteDependencyImpacted = true
			state.RootCauseSiteIDs = collectRootCauseSites(state.UnavailableUpstreamSiteIDs, raw, index)
		}
		states[name] = state
	}

	return SiteTopologyEval{States: states}
}

func siteUnavailable(raw SiteRawHealth, node SiteTopologyNode) bool {
	if len(node.HubDeviceIDs) > 0 {
		hubs := make(map[string]struct{}, len(node.HubDeviceIDs))
		for _, hub := range node.HubDeviceIDs {
			hubs[hub] = struct{}{}
		}
		for _, device := range raw.Devices {
			if _, ok := hubs[device.InventoryDeviceID]; !ok {
				continue
			}
			if isDirectCritical(device.Projection) {
				return true
			}
		}
		return false
	}
	return raw.Counts.CriticalCount > 0
}

func siteShowsCascadeSignature(raw SiteRawHealth) bool {
	if len(raw.Devices) == 0 {
		return false
	}
	failed := 0
	for _, device := range raw.Devices {
		if isDirectCritical(device.Projection) || device.Projection.Status == StatusUnknown {
			failed++
		}
	}
	return failed*2 > len(raw.Devices)
}

func isDirectCritical(proj DeviceProjection) bool {
	return proj.Status == StatusCritical && isDirectFailureReason(proj.StatusReason)
}

func isDirectFailureReason(reason string) bool {
	switch strings.ToLower(strings.TrimSpace(reason)) {
	case "direct_unreachable", "poll_failed", "":
		return true
	default:
		return false
	}
}

func collectRootCauseSites(unavailable []string, raw map[string]SiteRawHealth, index map[string]SiteTopologyNode) []string {
	seen := make(map[string]struct{})
	var out []string
	var walk func(name string)
	walk = func(name string) {
		if _, ok := seen[name]; ok {
			return
		}
		node, ok := index[name]
		if !ok || !siteUnavailable(raw[name], node) {
			return
		}
		if len(node.UpstreamSiteIDs) == 0 {
			seen[name] = struct{}{}
			out = append(out, name)
			return
		}
		found := false
		for _, upstream := range node.UpstreamSiteIDs {
			if siteUnavailable(raw[upstream], index[upstream]) {
				walk(upstream)
				found = true
			}
		}
		if !found {
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	for _, name := range unavailable {
		walk(name)
	}
	sort.Strings(out)
	return out
}

// ApplySiteDependencyOverlay reclassifies direct critical devices when the site is dependency-impacted.
func ApplySiteDependencyOverlay(state SiteDependencyState, proj DeviceProjection) DeviceProjection {
	if !state.SiteDependencyImpacted || !isDirectCritical(proj) {
		return proj
	}
	out := proj
	out.Status = StatusUnknown
	out.StatusReason = ReasonUpstreamSiteUnreachable
	return out
}

// ProjectedSiteHealth recomputes site counts after site-level device overlay.
func ProjectedSiteHealth(devices []DeviceProjection) SiteHealthCounts {
	var counts SiteHealthCounts
	for _, proj := range devices {
		if !proj.AlertsEnabled {
			continue
		}
		if strings.EqualFold(proj.StatusReason, ReasonUpstreamSiteUnreachable) {
			counts.UnknownCount++
			counts.DependencyImpactedCount++
			continue
		}
		counts.Accumulate(proj.Status, proj.UnavailableUpstreamIDs, proj.AlertsEnabled)
	}
	return counts
}
