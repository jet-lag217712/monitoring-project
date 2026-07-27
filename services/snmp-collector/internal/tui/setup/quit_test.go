package setup

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

func TestHandleQuitKeyFromStartStep(t *testing.T) {
	m := newModel("/tmp/deploy", tui.Theme{}, "test", ProfileAppliance)
	m.splash = false
	m.step = stepStart

	next, cmd, handled := m.handleQuitKey(tea.KeyMsg{Type: tea.KeyCtrlC})
	if !handled {
		t.Fatal("expected ctrl+c to be handled")
	}
	if cmd != nil {
		t.Fatal("expected confirmation before quitting")
	}
	if !next.confirmQuit {
		t.Fatal("expected confirmQuit to be set")
	}

	next, cmd, handled = next.handleQuitKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	if !handled || cmd == nil {
		t.Fatal("expected y to quit")
	}
}

func TestHandleQuitKeyImmediateOnDone(t *testing.T) {
	m := newModel("/tmp/deploy", tui.Theme{}, "test", ProfileAppliance)
	m.splash = false
	m.step = stepDone

	_, cmd, handled := m.handleQuitKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if !handled || cmd == nil {
		t.Fatal("expected q to quit immediately on done step")
	}
}

func TestHandleQuitKeyIgnoresQDuringInputSteps(t *testing.T) {
	m := newModel("/tmp/deploy", tui.Theme{}, "test", ProfileAppliance)
	m.splash = false
	m.step = stepEnv

	_, cmd, handled := m.handleQuitKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if handled || cmd != nil {
		t.Fatal("expected q to pass through during input steps")
	}
}

func TestHandleQuitKeyCancel(t *testing.T) {
	m := newModel("/tmp/deploy", tui.Theme{}, "test", ProfileAppliance)
	m.splash = false
	m.step = stepEnv
	m.confirmQuit = true

	next, cmd, handled := m.handleQuitKey(tea.KeyMsg{Type: tea.KeyEsc})
	if !handled || cmd != nil {
		t.Fatal("expected esc to cancel quit")
	}
	if next.confirmQuit {
		t.Fatal("expected confirmQuit to be cleared")
	}
}
