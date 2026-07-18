package topology

import (
	"context"
	"strings"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/core"
)

// NeighborProbe returns CDP neighbor IPs for a management address.
type NeighborProbe func(ctx context.Context, host, community string) ([]string, error)

// Enrich assigns role and upstream_device_ids using CDP link hints when available,
// with naming-pattern fallback for devices that do not expose CDP.
func Enrich(devices []config.DeviceConfig, community string, probe NeighborProbe) []config.DeviceConfig {
	if len(devices) == 0 {
		return devices
	}
	out := cloneDevices(devices)
	hostToID := make(map[string]string, len(out))
	idToIndex := make(map[string]int, len(out))
	for i, d := range out {
		hostToID[strings.TrimSpace(d.Host)] = d.ID
		idToIndex[d.ID] = i
	}

	adj := make(map[string]map[string]struct{}, len(out))
	addEdge := func(a, b string) {
		if a == "" || b == "" || a == b {
			return
		}
		if adj[a] == nil {
			adj[a] = make(map[string]struct{})
		}
		if adj[b] == nil {
			adj[b] = make(map[string]struct{})
		}
		adj[a][b] = struct{}{}
		adj[b][a] = struct{}{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if probe == nil {
		probe = defaultNeighborProbe(community)
	}
	for _, d := range out {
		neighbors, err := probe(ctx, d.Host, community)
		if err != nil {
			continue
		}
		for _, nip := range neighbors {
			if nid, ok := hostToID[nip]; ok {
				addEdge(d.ID, nid)
			}
		}
	}

	root := pickRoot(out, adj)
	parent, depth := bfsParents(root, adj, out)

	for i := range out {
		out[i].Role = roleForDevice(out[i].ID, depth[out[i].ID])
		if p := parent[out[i].ID]; p != "" {
			out[i].UpstreamDeviceIDs = []string{p}
		} else {
			out[i].UpstreamDeviceIDs = nil
		}
		_ = idToIndex // reserved for future overlay merges
	}
	return out
}

func defaultNeighborProbe(community string) NeighborProbe {
	return func(ctx context.Context, host, _ string) ([]string, error) {
		return core.ProbeCDPNeighborIPs(ctx, host, community, 2*time.Second, 0)
	}
}

func cloneDevices(in []config.DeviceConfig) []config.DeviceConfig {
	out := make([]config.DeviceConfig, len(in))
	copy(out, in)
	return out
}

func pickRoot(devices []config.DeviceConfig, adj map[string]map[string]struct{}) string {
	for _, d := range devices {
		if strings.EqualFold(d.ID, "do-core") || strings.Contains(strings.ToLower(d.ID), "core") {
			return d.ID
		}
	}
	bestID := devices[0].ID
	bestDegree := -1
	for _, d := range devices {
		degree := len(adj[d.ID])
		if degree > bestDegree {
			bestDegree = degree
			bestID = d.ID
		}
	}
	return bestID
}

func bfsParents(root string, adj map[string]map[string]struct{}, devices []config.DeviceConfig) (parent map[string]string, depth map[string]int) {
	parent = make(map[string]string, len(devices))
	depth = make(map[string]int, len(devices))
	seen := map[string]struct{}{root: {}}
	queue := []string{root}
	depth[root] = 0

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for neighbor := range adj[cur] {
			if _, ok := seen[neighbor]; ok {
				continue
			}
			seen[neighbor] = struct{}{}
			parent[neighbor] = cur
			depth[neighbor] = depth[cur] + 1
			queue = append(queue, neighbor)
		}
	}

	// Devices with no CDP edges still get naming-based upstream hints.
	for _, d := range devices {
		if _, ok := depth[d.ID]; ok {
			continue
		}
		depth[d.ID] = namingDepth(d.ID)
		if up := namingUpstream(d.ID, devices); up != "" {
			parent[d.ID] = up
		}
	}
	return parent, depth
}

func roleForDevice(id string, depth int) string {
	lower := strings.ToLower(id)
	switch {
	case strings.Contains(lower, "core") || depth == 0:
		return "Core Switch"
	case strings.Contains(lower, "-mdf"):
		return "Distribution Switch"
	case strings.Contains(lower, "-idf"):
		return "Access Switch"
	case depth == 1:
		return "Distribution Switch"
	case depth >= 2:
		return "Access Switch"
	default:
		return "Network Device"
	}
}

func namingDepth(id string) int {
	lower := strings.ToLower(id)
	switch {
	case strings.Contains(lower, "core"):
		return 0
	case strings.Contains(lower, "-mdf"):
		return 1
	case strings.Contains(lower, "-idf"):
		return 2
	default:
		return 1
	}
}

func namingUpstream(id string, devices []config.DeviceConfig) string {
	lower := strings.ToLower(id)
	ids := make(map[string]struct{}, len(devices))
	for _, d := range devices {
		ids[d.ID] = struct{}{}
	}
	if strings.Contains(lower, "-idf") {
		if i := strings.Index(lower, "-idf"); i > 0 {
			mdf := lower[:i] + "-mdf"
			if _, ok := ids[mdf]; ok {
				return mdf
			}
		}
	}
	if strings.Contains(lower, "-mdf") || strings.Contains(lower, "-idf") {
		if _, ok := ids["do-core"]; ok {
			return "do-core"
		}
	}
	return ""
}
