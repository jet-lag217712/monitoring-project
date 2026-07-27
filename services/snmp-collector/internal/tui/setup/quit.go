package setup

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

func (m model) handleQuitKey(msg tea.KeyMsg) (model, tea.Cmd, bool) {
	key := msg.String()
	if m.confirmQuit {
		switch key {
		case "y", "Y", "enter", "ctrl+c":
			return m, tea.Quit, true
		case "n", "N", "esc":
			m.confirmQuit = false
			return m, nil, true
		}
		return m, nil, true
	}
	if m.step == stepDone && (key == "q" || key == "enter" || key == "ctrl+c") {
		return m, tea.Quit, true
	}
	if key == "ctrl+c" {
		if m.splash {
			return m, tea.Quit, true
		}
		m.confirmQuit = true
		return m, nil, true
	}
	if key == "q" && m.splash {
		return m, tea.Quit, true
	}
	return m, nil, false
}

func (m model) quitFooter(th tui.Theme) string {
	if m.step == stepDone {
		return th.Muted.Render("press q or enter to quit")
	}
	if m.confirmQuit {
		return th.Error.Render("Quit setup? ") + th.Muted.Render("y yes · n or esc cancel")
	}
	return th.Muted.Render("ctrl+c to quit")
}

func (m model) appendQuitFooter(th tui.Theme, body *strings.Builder) {
	if m.step == stepDone {
		return
	}
	body.WriteString("\n\n")
	body.WriteString(m.quitFooter(th))
}
