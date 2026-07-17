package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
)

func TestPhase2MetricsRegisteredAndObserved(t *testing.T) {
	t.Parallel()

	registry := prometheus.NewRegistry()
	collector := NewWithRegisterer(registry)
	collector.ProfileDuration.Observe(0.25)
	collector.InterfaceSelectionTotal.WithLabelValues("selected").Add(2)

	families, err := registry.Gather()
	if err != nil {
		t.Fatalf("Gather: %v", err)
	}
	seen := make(map[string]bool, len(families))
	for _, family := range families {
		seen[family.GetName()] = true
	}
	for _, name := range []string{
		"collector_profile_duration_seconds",
		"collector_interface_selection_total",
		"collector_discovery_attempts_total",
	} {
		if !seen[name] {
			t.Fatalf("metric %q was not registered", name)
		}
	}
}
