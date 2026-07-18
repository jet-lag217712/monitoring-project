package setup

import (
	"fmt"
	"os"
	"path/filepath"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

// Run starts the first-boot setup wizard for a deployment directory.
func Run(deployDir string, theme tui.ThemeName) error {
	deployDir, err := filepath.Abs(deployDir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(deployDir); err != nil {
		return fmt.Errorf("deploy dir: %w", err)
	}
	m := newModel(deployDir, tui.NewTheme(theme))
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
