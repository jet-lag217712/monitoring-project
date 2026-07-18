// Package tui is the Bubble Tea local operator client for the control socket.
package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/equate/ogsd/services/snmp-collector/internal/control"
)

// Options configures the operator TUI.
type Options struct {
	Theme ThemeName
}

// Run starts the interactive TUI against the given control socket.
func Run(socketPath string, opts Options) error {
	theme := NewTheme(ParseThemeName(string(opts.Theme)))
	client := control.NewClient(socketPath)
	m := newModel(client, theme)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
