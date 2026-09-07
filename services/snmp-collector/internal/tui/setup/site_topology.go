package setup

import (
	"fmt"
	"strings"
)

// ValidateSiteTopology checks upstream references and cycles across site specs.
func ValidateSiteTopology(specs []SiteSpec) error {
	byID := make(map[string]SiteSpec, len(specs))
	for _, spec := range specs {
		byID[spec.SiteID] = spec
	}
	for _, spec := range specs {
		seen := make(map[string]struct{}, len(spec.UpstreamSiteIDs))
		for _, upstream := range spec.UpstreamSiteIDs {
			if _, duplicate := seen[upstream]; duplicate {
				return fmt.Errorf("site %q has duplicate upstream_site_id %q", spec.SiteID, upstream)
			}
			seen[upstream] = struct{}{}
			if upstream == spec.SiteID {
				return fmt.Errorf("site %q cannot reference itself as an upstream", spec.SiteID)
			}
			if _, exists := byID[upstream]; !exists {
				return fmt.Errorf("site %q references missing upstream site %q", spec.SiteID, upstream)
			}
		}
		for _, hub := range spec.HubDeviceIDs {
			if strings.TrimSpace(hub) == "" {
				return fmt.Errorf("site %q has empty hub_device_id", spec.SiteID)
			}
		}
	}

	visiting := make(map[string]bool)
	visited := make(map[string]bool)
	var visit func(id string) error
	visit = func(id string) error {
		if visited[id] {
			return nil
		}
		if visiting[id] {
			return fmt.Errorf("site dependency cycle detected at site %q", id)
		}
		visiting[id] = true
		for _, upstream := range byID[id].UpstreamSiteIDs {
			if err := visit(upstream); err != nil {
				return err
			}
		}
		visiting[id] = false
		visited[id] = true
		return nil
	}
	for _, spec := range specs {
		if err := visit(spec.SiteID); err != nil {
			return err
		}
	}
	return nil
}

// SuggestSiteTopology fills upstream_site_ids and hub_device_ids using naming heuristics.
// Nested lab/campus chains are Internet → core → MDF → IDF.
func SuggestSiteTopology(specs []SiteSpec) []SiteSpec {
	out := make([]SiteSpec, len(specs))
	copy(out, specs)
	ids := make(map[string]string, len(out))
	coreID := ""
	for _, spec := range out {
		lower := strings.ToLower(spec.SiteID)
		ids[lower] = spec.SiteID
		if coreID == "" && (lower == "do-core" || strings.Contains(lower, "core")) {
			coreID = spec.SiteID
		}
	}
	for i := range out {
		lower := strings.ToLower(out[i].SiteID)
		switch {
		case lower == "do-core" || strings.Contains(lower, "core"):
			out[i].UpstreamSiteIDs = nil
			if out[i].HubDeviceIDs == nil {
				out[i].HubDeviceIDs = []string{out[i].SiteID}
			}
		case strings.Contains(lower, "-idf"):
			if mdf := matchingMDFSiteID(lower, ids); mdf != "" {
				out[i].UpstreamSiteIDs = []string{mdf}
			} else if coreID != "" {
				out[i].UpstreamSiteIDs = []string{coreID}
			}
		case strings.Contains(lower, "-mdf"):
			if coreID != "" {
				out[i].UpstreamSiteIDs = []string{coreID}
			}
		}
	}
	return out
}

func matchingMDFSiteID(lowerIDF string, ids map[string]string) string {
	i := strings.Index(lowerIDF, "-idf")
	if i <= 0 {
		return ""
	}
	return ids[lowerIDF[:i]+"-mdf"]
}

// ParseUpstreamSiteIDs parses comma-separated upstream site IDs.
func ParseUpstreamSiteIDs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "—" || raw == "-" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

// FormatUpstreamSiteIDs renders upstream site IDs for text input.
func FormatUpstreamSiteIDs(ids []string) string {
	if len(ids) == 0 {
		return ""
	}
	return strings.Join(ids, ", ")
}

// ApplyUpstreamSiteIDs applies parsed upstream IDs to specs by index.
func ApplyUpstreamSiteIDs(specs []SiteSpec, upstreams [][]string) ([]SiteSpec, error) {
	if len(specs) != len(upstreams) {
		return nil, fmt.Errorf("expected %d upstream rows, got %d", len(specs), len(upstreams))
	}
	out := make([]SiteSpec, len(specs))
	copy(out, specs)
	for i := range out {
		out[i].UpstreamSiteIDs = append([]string(nil), upstreams[i]...)
	}
	if err := ValidateSiteTopology(out); err != nil {
		return nil, err
	}
	return out, nil
}
