package health_test

import (
	"testing"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/health"
)

func TestTrackerConcurrentSnapshotAndApplyBatch(t *testing.T) {
	tracker := health.NewTracker()
	cfg := &config.Config{
		Health: config.HealthConfig{FailureThreshold: 2, TemperatureWarningC: 65},
		Devices: []config.DeviceConfig{
			{ID: "dev-001", Host: "127.0.0.1"},
			{ID: "dev-002", Host: "127.0.0.2"},
		},
	}
	outcomes := []health.PollOutcome{
		{DeviceID: "dev-001", Success: true},
		{DeviceID: "dev-002", Success: false},
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 0; i < 200; i++ {
			_ = tracker.Snapshot()
			_, _ = tracker.Device("dev-001")
		}
	}()

	for i := 0; i < 200; i++ {
		_ = tracker.ApplyBatch(cfg, outcomes)
		tracker.Prune([]string{"dev-001", "dev-002"})
	}
	<-done
}
