package tui

import (
	"strings"
	"testing"

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

func TestModelSwitchesViewsWithoutPanic(t *testing.T) {
	m := newModel(nil, NewTheme(ThemeLight))
	m.body = "initial"
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

func TestFormatDeviceHasNoMapLeak(t *testing.T) {
	th := NewTheme(ThemeLight)
	out := formatDevice(th, map[string]any{
		"id":                    "dev-001",
		"host":                  "10.0.0.1",
		"port":                  161,
		"version":               "2c",
		"community_env":         "SNMP_COMMUNITY_DEV_001",
		"temperature_warning_c": 65.0,
		"upstream_device_ids":   []string{"core-1"},
		"config_revision":       "rev1234567890",
		"health": map[string]any{
			"state":                           "warning",
			"reason":                          "temperature",
			"failure_count":                   0,
			"unavailable_upstream_device_ids": []string{},
			"root_cause_device_ids":           []string{},
		},
	})
	if strings.Contains(out, "map[") {
		t.Fatalf("device output leaked map dump: %q", out)
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
