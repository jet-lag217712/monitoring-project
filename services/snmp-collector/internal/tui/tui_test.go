package tui

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

func TestParseThemeName(t *testing.T) {
	cases := map[string]ThemeName{
		"auto":  ThemeAuto,
		"LIGHT": ThemeLight,
		"dark":  ThemeDark,
		"":      ThemeAuto,
		"nope":  ThemeAuto,
	}
	for in, want := range cases {
		if got := ParseThemeName(in); got != want {
			t.Fatalf("ParseThemeName(%q)=%q want %q", in, got, want)
		}
	}
}

func TestNewThemeLightAndDark(t *testing.T) {
	light := NewTheme(ThemeLight)
	if light.Name != ThemeLight {
		t.Fatalf("light name=%q", light.Name)
	}
	dark := NewTheme(ThemeDark)
	if dark.Name != ThemeDark {
		t.Fatalf("dark name=%q", dark.Name)
	}
	if light.Salmon != dark.Salmon {
		t.Fatalf("salmon should be brand-stable across themes")
	}
}

func TestLayoutChromeDiagnostics(t *testing.T) {
	th := NewTheme(ThemeDark)
	now := time.Now()
	header := renderHeader(th, "site-a", "", "", now, false)
	tabs := renderTabs(th, viewInventory)

	if !strings.HasSuffix(header, "\n") {
		t.Fatalf("header must end with newline")
	}
	joined := header + tabs
	if idx := strings.Index(joined, "Inventory |"); idx >= 0 {
		lineStart := strings.LastIndex(joined[:idx], "\n") + 1
		lineEnd := strings.Index(joined[idx:], "\n")
		if lineEnd < 0 {
			lineEnd = len(joined) - idx
		}
		line := joined[lineStart : idx+lineEnd]
		if strings.Contains(line, "╚═╝") {
			t.Fatalf("tabs share line with logo art: %q", line)
		}
	}

	m := newModel(nil, th)
	m.siteID = "site-a"
	m.lastUpdated = now
	m.body = formatInventory(th, map[string]any{
		"config_revision": "revision-b72",
		"devices": []any{
			map[string]any{
				"id": "site-a-mdf", "host": "10.255.1.1", "port": 161,
				"health": map[string]any{"state": "healthy"},
			},
		},
	})
	m.width = 120
	m.height = 40
	m.ready = true
	m.viewport = viewport.New(120, 20)

	view := m.View()
	lines := strings.Split(view, "\n")

	tabsLine := lineAt(lines, findLineContaining(lines, "Inventory |"))
	if strings.Contains(tabsLine, "╚═╝") {
		t.Fatalf("tabs still on logo line: %q", tabsLine)
	}
	if !strings.HasPrefix(strings.TrimSpace(tabsLine), "Inventory") {
		t.Fatalf("tabs not left-aligned: %q", tabsLine)
	}
}

func TestDiscoveryAcceptanceRequiresConfirmation(t *testing.T) {
	m := newModel(nil, NewTheme(ThemeLight))
	next, _ := m.Update(pendingPreparedMsg{
		token:    "accept-token",
		revision: "revision-1",
		action:   "discovery.accept",
	})
	updated := next.(model)
	if updated.confirmPrompt != "commit" {
		t.Fatalf("confirm prompt = %q, want commit", updated.confirmPrompt)
	}
	if updated.pendingAction != "discovery.accept" {
		t.Fatalf("pending action = %q", updated.pendingAction)
	}
}

func findLineContaining(lines []string, substr string) int {
	for i, line := range lines {
		if strings.Contains(line, substr) {
			return i
		}
	}
	return -1
}

func lineAt(lines []string, idx int) string {
	if idx < 0 || idx >= len(lines) {
		return ""
	}
	return lines[idx]
}

func TestLogoRendersBlockArt(t *testing.T) {
	out := Logo(NewTheme(ThemeDark))
	if !strings.Contains(out, "█") {
		t.Fatalf("expected block art, got %q", out)
	}
}

func TestModelSwitchesViewsWithoutPanic(t *testing.T) {
	m := newModel(nil, NewTheme(ThemeLight))
	m.body = "initial"
	next, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	updated := next.(model)
	if updated.view != viewDiscovery {
		t.Fatalf("view=%v", updated.view)
	}
	next, _ = updated.Update(refreshMsg{body: "discovery body"})
	updated = next.(model)
	if updated.body != "discovery body" {
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

func TestFormatInventoryHasNoMapLeak(t *testing.T) {
	th := NewTheme(ThemeLight)
	out := formatInventory(th, map[string]any{
		"config_revision": "abcdef1234567890",
		"devices": []any{
			map[string]any{
				"id":   "dev-001",
				"host": "10.0.0.1",
				"port": 161,
				"health": map[string]any{
					"state":  "ok",
					"reason": "healthy",
				},
				"last_poll": map[string]any{
					"at": "2026-07-17T12:00:00Z",
					"ok": true,
				},
				"upstream_device_ids": []any{"core-1"},
			},
		},
	})
	if strings.Contains(out, "map[") {
		t.Fatalf("inventory output leaked map dump: %q", out)
	}
	if !strings.Contains(out, "dev-001") || !strings.Contains(out, "10.0.0.1") {
		t.Fatalf("inventory missing device fields: %q", out)
	}
}

func TestFormatDiscoveryViewHasNoMapLeak(t *testing.T) {
	th := NewTheme(ThemeLight)
	out := formatDiscoveryView(th, map[string]any{}, map[string]any{
		"candidates": []any{
			map[string]any{
				"ip":               "10.0.0.1",
				"detected_profile": "cisco",
				"hostname":         "switch-1",
				"result":           "success",
			},
		},
	})
	if strings.Contains(out, "map[") {
		t.Fatalf("discovery output leaked map dump: %q", out)
	}
	if !strings.Contains(out, "10.0.0.1") {
		t.Fatalf("discovery missing candidate: %q", out)
	}
}

func TestFormatResultSkipsNestedMaps(t *testing.T) {
	th := NewTheme(ThemeLight)
	out := formatResult(th, "Reload", map[string]any{
		"config_revision": "abc",
		"device_count":    3,
		"nested":          map[string]any{"x": 1},
	})
	if strings.Contains(out, "map[") {
		t.Fatalf("formatResult leaked map: %q", out)
	}
	if !strings.Contains(out, "abc") {
		t.Fatalf("missing revision: %q", out)
	}
}

func TestNOColorTheme(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	th := NewTheme(ThemeDark)
	rendered := th.Title.Render("Equate")
	if rendered == "" {
		t.Fatal("expected rendered title")
	}
	if !strings.Contains(th.Wordmark.Render("Equate"), "Equate") {
		t.Fatal("expected wordmark text")
	}
}
