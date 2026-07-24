package setup

import (
	"strings"
	"testing"

	"github.com/equate/ogsd/services/snmp-collector/internal/tui"
)

func TestRenderProgressBar(t *testing.T) {
	th := tui.NewTheme(tui.ThemeDark)
	out := renderProgressBar(th, 2, 4, 20)
	if !strings.Contains(out, "50%") {
		t.Fatalf("expected percent in bar: %q", out)
	}
	if !strings.Contains(out, "2/4") {
		t.Fatalf("expected step count: %q", out)
	}
}

func TestParsedProbeDefaults(t *testing.T) {
	m := newModel(t.TempDir(), tui.NewTheme(tui.ThemeDark), "test")
	m.probeRateInput.SetValue("")
	m.probeBurstInput.SetValue("")
	if got := m.parsedProbeRate(); got != 20 {
		t.Fatalf("rate=%v", got)
	}
	if got := m.parsedProbeBurst(); got != 10 {
		t.Fatalf("burst=%v", got)
	}
}

func TestSiteFieldCountIncludesDiscovery(t *testing.T) {
	m := newModel(t.TempDir(), tui.NewTheme(tui.ThemeDark), "test")
	if m.siteFieldCount() != 1+2*defaultSiteCount+2 {
		t.Fatalf("field count=%d", m.siteFieldCount())
	}
}
