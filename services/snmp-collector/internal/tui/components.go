package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var viewTabs = []struct {
	name string
	v    view
}{
	{"Inventory", viewInventory},
	{"Discovery", viewDiscovery},
	{"Transit", viewTransit},
	{"Settings", viewSettings},
}

func renderHeader(th Theme, siteID, collectorID, revision string, lastUpdated time.Time, loading bool) string {
	logo := Logo(th)

	var meta strings.Builder
	meta.WriteString(th.Eyebrow.Render("SNMP COLLECTOR"))
	if loading {
		meta.WriteString("  ")
		meta.WriteString(th.Spinner.Render("…"))
	}
	meta.WriteString("\n")
	if siteID != "" {
		meta.WriteString(th.Muted.Render(siteID))
		meta.WriteString("\n")
	} else if collectorID != "" {
		meta.WriteString(th.Muted.Render(collectorID))
		meta.WriteString("\n")
	}
	if !lastUpdated.IsZero() {
		meta.WriteString(th.Muted.Render("updated " + lastUpdated.Format("15:04:05")))
	}

	header := JoinBlocks(logo, "  ", meta.String())
	return WithTopPadding(header) + "\n"
}

func renderTabs(th Theme, active view) string {
	parts := make([]string, 0, len(viewTabs)*2-1)
	for i, tab := range viewTabs {
		if i > 0 {
			parts = append(parts, th.Muted.Render(" | "))
		}
		if tab.v == active {
			parts = append(parts, lipgloss.NewStyle().Foreground(th.Salmon).Bold(true).Render(tab.name))
		} else {
			parts = append(parts, th.TabIdle.Render(tab.name))
		}
	}
	return strings.Join(parts, "")
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

func renderActions(th Theme, v view) string {
	switch v {
	case viewDiscovery:
		return th.Muted.Render("S scan  ·  A accept successful  ·  E edit CIDRs")
	default:
		return ""
	}
}

func renderHelp(th Theme) string {
	keys := []string{
		"1-4 views",
		"tab/←→ switch",
		"r refresh",
		"R reload",
		"t threshold",
		"d deps",
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
