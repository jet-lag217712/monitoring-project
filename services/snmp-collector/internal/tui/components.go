package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var viewTabs = []struct {
	key  string
	name string
	v    view
}{
	{"1", "Inventory", viewInventory},
	{"2", "Device", viewDevice},
	{"3", "Discovery", viewDiscovery},
	{"4", "Thresholds", viewThresholds},
	{"5", "Transport", viewTransport},
	{"6", "Config", viewConfig},
}

func renderHeader(th Theme, siteID, collectorID, revision string, lastUpdated time.Time, loading bool) string {
	var b strings.Builder
	b.WriteString(th.LogoMark.Render("//"))
	b.WriteString(" ")
	b.WriteString(th.Wordmark.Render("Equate"))
	b.WriteString("  ")
	b.WriteString(th.Eyebrow.Render("SNMP COLLECTOR"))
	if loading {
		b.WriteString("  ")
		b.WriteString(th.Spinner.Render("…"))
	}
	b.WriteString("\n")

	meta := []string{}
	if siteID != "" {
		meta = append(meta, "site "+siteID)
	}
	if collectorID != "" {
		meta = append(meta, "collector "+collectorID)
	}
	if revision != "" {
		short := revision
		if len(short) > 12 {
			short = short[:12]
		}
		meta = append(meta, "rev "+short)
	}
	if !lastUpdated.IsZero() {
		meta = append(meta, "updated "+lastUpdated.Format("15:04:05"))
	}
	if len(meta) > 0 {
		b.WriteString(th.Muted.Render(strings.Join(meta, "  ·  ")))
		b.WriteString("\n")
	}
	return b.String()
}

func renderTabs(th Theme, active view) string {
	parts := make([]string, 0, len(viewTabs))
	for _, tab := range viewTabs {
		label := fmt.Sprintf("%s %s", tab.key, tab.name)
		if tab.v == active {
			parts = append(parts, th.TabActive.Render(label))
		} else {
			parts = append(parts, th.TabIdle.Render(label))
		}
	}
	return strings.Join(parts, "  ")
}

func renderStatusBadge(th Theme, state string) string {
	label := strings.ToUpper(strings.TrimSpace(state))
	if label == "" {
		label = "UNKNOWN"
	}
	dot := "●"
	style := th.StatusStyle(state)
	return style.Render(dot + " " + label)
}

func renderKVPanel(th Theme, title string, rows [][2]string) string {
	var b strings.Builder
	if title != "" {
		b.WriteString(th.Title.Render(title))
		b.WriteString("\n")
	}
	maxLabel := 0
	for _, row := range rows {
		if len(row[0]) > maxLabel {
			maxLabel = len(row[0])
		}
	}
	for _, row := range rows {
		label := row[0]
		pad := strings.Repeat(" ", maxLabel-len(label))
		b.WriteString(th.Label.Render(strings.ToUpper(label)))
		b.WriteString(pad)
		b.WriteString("  ")
		b.WriteString(th.Value.Render(row[1]))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func renderTable(th Theme, headers []string, rows [][]string) string {
	if len(headers) == 0 {
		return ""
	}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i := 0; i < len(headers) && i < len(row); i++ {
			if len(row[i]) > widths[i] {
				widths[i] = len(row[i])
			}
		}
	}
	// Cap columns so wide terminals stay readable.
	for i := range widths {
		if widths[i] > 40 {
			widths[i] = 40
		}
	}

	var b strings.Builder
	headerCells := make([]string, len(headers))
	for i, h := range headers {
		headerCells[i] = th.TableHeader.Render(padClip(strings.ToUpper(h), widths[i]))
	}
	b.WriteString(strings.Join(headerCells, "  "))
	b.WriteString("\n")
	b.WriteString(th.Muted.Render(strings.Repeat("─", min(lipgloss.Width(strings.Join(headerCells, "  ")), 120))))
	b.WriteString("\n")
	for _, row := range rows {
		cells := make([]string, len(headers))
		for i := range headers {
			cell := ""
			if i < len(row) {
				cell = row[i]
			}
			cells[i] = th.Value.Render(padClip(cell, widths[i]))
		}
		b.WriteString(strings.Join(cells, "  "))
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func padClip(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) > width {
		if width <= 1 {
			return string(runes[:width])
		}
		return string(runes[:width-1]) + "…"
	}
	return s + strings.Repeat(" ", width-len(runes))
}

func renderHelp(th Theme) string {
	keys := []string{
		"1-6 views",
		"tab/←→ switch",
		"r refresh",
		"R reload",
		"t threshold",
		"d deps",
		"i ignore",
		"↑↓ scroll",
		"q quit",
	}
	return th.Help.Render(strings.Join(keys, "  ·  "))
}

func renderConfirm(th Theme, action, revision string) string {
	var b strings.Builder
	b.WriteString(th.Confirm.Render("Confirm mutation commit?"))
	b.WriteString(" ")
	b.WriteString(th.Muted.Render("[y/n]"))
	b.WriteString("\n")
	b.WriteString(th.Soft.Render(fmt.Sprintf("action=%s  revision=%s", action, revision)))
	return b.String()
}

func renderError(th Theme, err string) string {
	if err == "" {
		return ""
	}
	return th.Error.Render("error: " + err)
}

func renderEmpty(th Theme, title, copy string) string {
	var b strings.Builder
	b.WriteString(th.Title.Render(title))
	b.WriteString("\n")
	b.WriteString(th.Muted.Render(copy))
	return b.String()
}
