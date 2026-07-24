package setup

import (
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

func renderProgressBar(th tui.Theme, current, total, width int) string {
	if total <= 0 {
		total = 1
	}
	if current < 0 {
		current = 0
	}
	if current > total {
		current = total
	}
	if width < 12 {
		width = 12
	}
	filled := (current * width) / total
	if current > 0 && filled == 0 {
		filled = 1
	}
	if filled > width {
		filled = width
	}
	salmon := lipgloss.NewStyle().Foreground(th.Salmon)
	bar := salmon.Render(strings.Repeat("█", filled)) +
		th.Muted.Render(strings.Repeat("░", width-filled))
	pct := (current * 100) / total
	return lipgloss.NewStyle().Foreground(th.InkSoft).Render(
		bar + "  " + strings.TrimSpace(th.Soft.Render(strconv.Itoa(pct)+"%")) +
			"  " + th.Muted.Render(strconv.Itoa(current)+"/"+strconv.Itoa(total)),
	)
}
