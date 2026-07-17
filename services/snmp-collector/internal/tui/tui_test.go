package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestModelSwitchesViewsWithoutPanic(t *testing.T) {
	m := model{view: viewInventory, body: "initial"}
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	updated := next.(model)
	if updated.view != viewDevice {
		t.Fatalf("view=%v", updated.view)
	}
	next, _ = updated.Update(refreshMsg{body: "device body"})
	updated = next.(model)
	if updated.body != "device body" {
		t.Fatalf("body=%q", updated.body)
	}
	next, _ = updated.Update(pendingPreparedMsg{token: "tok", revision: "rev", action: "thresholds"})
	updated = next.(model)
	if updated.confirmPrompt != "commit" || updated.pendingToken != "tok" {
		t.Fatalf("confirm state=%#v", updated)
	}
	next, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	updated = next.(model)
	if updated.confirmPrompt != "" || updated.pendingToken != "" {
		t.Fatalf("expected cancelled mutation, got %#v", updated)
	}
}
