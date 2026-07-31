package setup

import (
	"strings"
	"testing"

	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

func TestSplashLogoRenders(t *testing.T) {
	out := splashLogo(tui.NewTheme(tui.ThemeDark))
	if !strings.Contains(out, "█") {
		t.Fatalf("expected block art, got %q", out)
	}
}

func TestConfiguratorLogoRenders(t *testing.T) {
	out := configuratorLogo(tui.NewTheme(tui.ThemeDark))
	if !strings.Contains(out, "█") {
		t.Fatalf("expected block art, got %q", out)
	}
}

func TestFormatVersion(t *testing.T) {
	if got := formatVersion(""); got != "1.5.0" {
		t.Fatalf("got %q", got)
	}
	if got := formatVersion("unknown"); got != "1.5.0" {
		t.Fatalf("got %q", got)
	}
	if got := formatVersion("2.0.1"); got != "2.0.1" {
		t.Fatalf("got %q", got)
	}
}

func TestWithTopPadding(t *testing.T) {
	out := withTopPadding("content")
	if out == "" {
		t.Fatal("expected padded content")
	}
}

func TestViewSplashContainsFooter(t *testing.T) {
	m := newModel(t.TempDir(), tui.NewTheme(tui.ThemeDark), "test", ProfileDevVxrail, RunOptions{})
	m.width = 100
	m.height = 40
	out := m.viewSplash()
	if !strings.Contains(out, "West") || !strings.Contains(out, "Lafayette") {
		t.Fatalf("missing footer: %q", out)
	}
	if !strings.Contains(out, "enter to continue") {
		t.Fatalf("missing prompt: %q", out)
	}
}
