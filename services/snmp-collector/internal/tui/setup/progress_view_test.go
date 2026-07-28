package setup

import (
	"strings"
	"testing"

	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

func TestRenderLoadProgress(t *testing.T) {
	th := tui.NewTheme(tui.ThemeLight)
	out := renderLoadProgress(th, "Waiting for site-001", 2, 4, 10)
	if !strings.Contains(out, "Waiting for site-001") {
		t.Fatalf("missing label: %s", out)
	}
	if !strings.Contains(out, "█") || !strings.Contains(out, "░") {
		t.Fatalf("missing bar chars: %s", out)
	}
	if !strings.Contains(out, "50% (2/4)") {
		t.Fatalf("missing fraction: %s", out)
	}
}

func TestRenderLoadProgressZeroTotal(t *testing.T) {
	th := tui.NewTheme(tui.ThemeLight)
	out := renderLoadProgress(th, "Starting", 0, 0, 10)
	if !strings.Contains(out, "Starting") {
		t.Fatalf("missing label: %s", out)
	}
	if strings.Contains(out, "█") {
		t.Fatalf("unexpected bar: %s", out)
	}
}
