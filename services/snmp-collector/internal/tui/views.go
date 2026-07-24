package tui

import (
	"fmt"
	"sort"
	"strings"
)

func formatInventory(th Theme, result map[string]any) string {
	devices, _ := result["devices"].([]any)
	rev, _ := result["config_revision"].(string)
	if len(devices) == 0 {
		return renderEmpty(th, "Inventory", "No devices configured.")
	}
	headers := []string{"ID", "Host", "Port", "Health", "Last Poll", "Upstreams"}
	rows := make([][]string, 0, len(devices))
	for _, raw := range devices {
		d, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		healthState := nestedString(d, "health", "state")
		lastPoll := formatLastPoll(d["last_poll"])
		upstreams := formatStringSlice(d["upstream_device_ids"])
		rows = append(rows, []string{
			fmt.Sprint(d["id"]),
			fmt.Sprint(d["host"]),
			fmt.Sprint(d["port"]),
			renderStatusBadge(th, healthState),
			lastPoll,
			upstreams,
		})
	}
	var b strings.Builder
	b.WriteString(th.Title.Render("Inventory"))
	if rev != "" {
		b.WriteString("  ")
		b.WriteString(th.Muted.Render("rev " + shortRev(rev)))
	}
	b.WriteString("\n\n")
	b.WriteString(renderTable(th, headers, rows))
	return b.String()
}

func formatDiscovery(th Theme, result map[string]any) string {
	rows := [][2]string{
		{"allowed_cidrs", formatStringSlice(result["allowed_cidrs"])},
		{"max_probes_per_second", fmt.Sprint(result["max_probes_per_second"])},
		{"probe_burst", fmt.Sprint(result["probe_burst"])},
		{"max_targets", fmt.Sprint(result["max_targets"])},
		{"max_workers", fmt.Sprint(result["max_workers"])},
		{"community_env", fmt.Sprint(result["community_env"])},
	}
	var b strings.Builder
	b.WriteString(renderKVPanel(th, "Discovery", rows))
	if note, ok := result["note"].(string); ok && note != "" {
		b.WriteString("\n\n")
		b.WriteString(th.Muted.Render(note))
	}
	return b.String()
}

func formatDiscoveryView(th Theme, status, candidates map[string]any) string {
	var b strings.Builder
	b.WriteString(th.Title.Render("Discovery Results"))
	b.WriteString("\n")
	raw, _ := candidates["candidates"].([]any)
	if len(raw) == 0 {
		b.WriteString(th.Muted.Render("No scan results yet."))
	} else {
		headers := []string{"IP", "Profile", "Hostname", "Result"}
		rows := make([][]string, 0, len(raw))
		for _, item := range raw {
			c, ok := item.(map[string]any)
			if !ok {
				continue
			}
			state := fmt.Sprint(c["result"])
			rows = append(rows, []string{
				fmt.Sprint(c["ip"]),
				fmt.Sprint(c["detected_profile"]),
				fmt.Sprint(c["hostname"]),
				renderStatusBadge(th, state),
			})
		}
		b.WriteString(renderTable(th, headers, rows))
	}
	_ = status
	return b.String()
}

func formatTransit(th Theme, result map[string]any) string {
	mqtt := "n/a"
	if v, ok := result["mqtt_connected"]; ok {
		if b, ok := v.(bool); ok {
			if b {
				mqtt = renderStatusBadge(th, "ok") + " (ACTIVE)"
			} else {
				mqtt = renderStatusBadge(th, "alert") + " (DISCONNECTED)"
			}
		} else {
			mqtt = fmt.Sprint(v)
		}
	}
	rows := [][2]string{
		{"publisher_mode", fmt.Sprint(result["publisher_mode"])},
		{"telemetry_version", fmt.Sprint(result["telemetry_version"])},
		{"buffer_depth", fmt.Sprint(result["buffer_depth"])},
		{"buffer_available", fmt.Sprint(result["buffer_available"])},
		{"mqtt", mqtt},
		{"revision", shortRev(fmt.Sprint(result["config_revision"]))},
	}
	return renderKVPanel(th, "Transit Configuration", rows)
}

func formatConfig(th Theme, result map[string]any) string {
	rows := [][2]string{
		{"site_id", fmt.Sprint(result["site_id"])},
		{"collector_id", fmt.Sprint(result["collector_id"])},
		{"revision", shortRev(fmt.Sprint(result["config_revision"]))},
		{"managed_path", fmt.Sprint(result["managed_path"])},
	}
	if health, ok := result["health"].(map[string]any); ok {
		rows = append(rows,
			[2]string{"temp_warning_c", fmt.Sprint(health["temperature_warning_c"])},
			[2]string{"failure_threshold", fmt.Sprint(health["failure_threshold"])},
		)
	}
	if disc, ok := result["discovery"].(map[string]any); ok {
		rows = append(rows,
			[2]string{"discovery_cidrs", formatStringSlice(disc["allowed_cidrs"])},
			[2]string{"discovery_rate", fmt.Sprintf("%v/s burst %v", disc["max_probes_per_second"], disc["probe_burst"])},
		)
	}
	if admin, ok := result["admin"].(map[string]any); ok {
		rows = append(rows,
			[2]string{"admin_listen", fmt.Sprint(admin["listen"])},
			[2]string{"control_socket", fmt.Sprint(admin["control_socket"])},
		)
	}
	var b strings.Builder
	b.WriteString(renderKVPanel(th, "Configuration", rows))
	if reload, ok := result["last_reload"].(map[string]any); ok {
		b.WriteString("\n\n")
		b.WriteString(th.Title.Render("Last Reload"))
		b.WriteString("\n")
		reloadRows := make([][2]string, 0, len(reload))
		for _, k := range sortedKeys(reload) {
			reloadRows = append(reloadRows, [2]string{k, fmt.Sprint(reload[k])})
		}
		b.WriteString(renderKVPanel(th, "", reloadRows))
	}
	return b.String()
}

func formatReloadResult(th Theme, result map[string]any) string {
	rows := make([][2]string, 0, len(result))
	for _, k := range sortedKeys(result) {
		rows = append(rows, [2]string{k, fmt.Sprint(result[k])})
	}
	return renderKVPanel(th, "Reload", rows)
}

func formatStringSlice(v any) string {
	switch list := v.(type) {
	case []string:
		if len(list) == 0 {
			return "—"
		}
		return strings.Join(list, ", ")
	case []any:
		if len(list) == 0 {
			return "—"
		}
		parts := make([]string, 0, len(list))
		for _, item := range list {
			parts = append(parts, fmt.Sprint(item))
		}
		return strings.Join(parts, ", ")
	case nil:
		return "—"
	default:
		s := fmt.Sprint(v)
		if s == "" || s == "[]" || s == "<nil>" {
			return "—"
		}
		return s
	}
}

func formatLastPoll(v any) string {
	if v == nil {
		return "—"
	}
	if m, ok := v.(map[string]any); ok {
		for _, key := range []string{"at", "timestamp", "time", "last_success_at", "finished_at"} {
			if s, ok := m[key].(string); ok && s != "" {
				return s
			}
		}
		if ok, exists := m["ok"]; exists {
			return fmt.Sprintf("ok=%v", ok)
		}
	}
	s := fmt.Sprint(v)
	if strings.Contains(s, "map[") {
		return "recorded"
	}
	return s
}

func nestedString(m map[string]any, keys ...string) string {
	cur := any(m)
	for _, k := range keys {
		obj, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur = obj[k]
	}
	if cur == nil {
		return ""
	}
	return fmt.Sprint(cur)
}

func shortRev(rev string) string {
	if rev == "" || rev == "<nil>" {
		return "—"
	}
	if len(rev) > 12 {
		return rev[:12]
	}
	return rev
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// formatResult is retained for mutation reload messages that need a simple dump fallback.
func formatResult(th Theme, title string, result map[string]any) string {
	rows := make([][2]string, 0, len(result))
	for _, key := range sortedKeys(result) {
		val := result[key]
		if _, isMap := val.(map[string]any); isMap {
			continue
		}
		if _, isSlice := val.([]any); isSlice {
			rows = append(rows, [2]string{key, formatStringSlice(val)})
			continue
		}
		s := fmt.Sprint(val)
		if strings.Contains(s, "map[") {
			continue
		}
		rows = append(rows, [2]string{key, s})
	}
	return renderKVPanel(th, title, rows)
}
