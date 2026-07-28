package setup

import (
	"fmt"
	"strings"

	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

const defaultProgressBarWidth = 24

func renderLoadProgress(th tui.Theme, label string, current, total, width int) string {
	if width <= 0 {
		width = defaultProgressBarWidth
	}
	var b strings.Builder
	if label != "" {
		b.WriteString(th.Muted.Render(label))
		b.WriteString("\n")
	}
	if total <= 0 {
		return strings.TrimRight(b.String(), "\n")
	}
	if current > total {
		current = total
	}
	filled := current * width / total
	if current > 0 && filled == 0 {
		filled = 1
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	pct := current * 100 / total
	b.WriteString(th.Value.Render(bar))
	b.WriteString("  ")
	b.WriteString(th.Muted.Render(fmt.Sprintf("%d%% (%d/%d)", pct, current, total)))
	return b.String()
}
