package health

import (
	"math"
	"testing"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/readings"
)

func TestPrimaryTemperature(t *testing.T) {
	t.Parallel()

	neg := -5.0
	nan := math.NaN()
	a := 40.0
	b := 55.0
	got, ok := PrimaryTemperature([]readings.ComponentReading{
		{Value: nil},
		{Value: &neg},
		{Value: &nan},
		{Value: &a},
		{Value: &b},
	})
	if !ok || got != 55 {
		t.Fatalf("PrimaryTemperature=%v ok=%v want 55 true", got, ok)
	}

	if _, ok := PrimaryTemperature(nil); ok {
		t.Fatal("expected no valid temperature")
	}
}

func baseCfg(devices ...config.DeviceConfig) *config.Config {
	return &config.Config{
		SiteID:    "site-001",
		Collector: config.CollectorConfig{ID: "collector-001"},
		Health: config.HealthConfig{
			TemperatureWarningC: 65,
			FailureThreshold:    2,
		},
		Devices: devices,
	}
}

func success(id string, at time.Time, temp *float64) PollOutcome {
	return PollOutcome{DeviceID: id, Success: true, ObservedAt: at, TemperatureC: temp}
}

func failure(id string, at time.Time) PollOutcome {
	return PollOutcome{DeviceID: id, Success: false, ObservedAt: at}
}

func temp(v float64) *float64 { return &v }

func eventMap(events []Event) map[string]Event {
	out := make(map[string]Event, len(events))
	for _, ev := range events {
		out[ev.DeviceID] = ev
	}
	return out
}

func TestRootFailureBecomesCriticalAfterThreshold(t *testing.T) {
	t.Parallel()

	tracker := NewTracker()
	cfg := baseCfg(config.DeviceConfig{ID: "core-01", Host: "10.0.0.1", CommunityEnv: "C", Version: "2c"})
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)

	events := tracker.ApplyBatch(cfg, []PollOutcome{success("core-01", now, nil)})
	if len(events) != 1 || events[0].State != StateHealthy || events[0].Transition != TransitionInitial {
		t.Fatalf("initial healthy: %#v", events)
	}

	events = tracker.ApplyBatch(cfg, []PollOutcome{failure("core-01", now.Add(time.Minute))})
	if len(events) != 0 {
		t.Fatalf("pending failure must not emit: %#v", events)
	}
	dev, _ := tracker.Device("core-01")
	if dev.State != StateHealthy || dev.FailureCount != 1 {
		t.Fatalf("pending retain: %#v", dev)
	}

	events = tracker.ApplyBatch(cfg, []PollOutcome{failure("core-01", now.Add(2*time.Minute))})
	if len(events) != 1 {
		t.Fatalf("expected critical event, got %#v", events)
	}
	if events[0].State != StateCritical || events[0].Reason != ReasonDirectUnreachable || events[0].Transition != TransitionEntered {
		t.Fatalf("critical event: %#v", events[0])
	}
	if events[0].FailureCount != 2 {
		t.Fatalf("failure_count=%d", events[0].FailureCount)
	}
}

func TestMultiLevelCascadeUnknownWithRootCause(t *testing.T) {
	t.Parallel()

	tracker := NewTracker()
	cfg := baseCfg(
		config.DeviceConfig{ID: "core-01", Host: "10.0.0.1", CommunityEnv: "C", Version: "2c"},
		config.DeviceConfig{ID: "dist-01", Host: "10.0.0.2", CommunityEnv: "C", Version: "2c", UpstreamDeviceIDs: []string{"core-01"}},
		config.DeviceConfig{ID: "access-01", Host: "10.0.0.3", CommunityEnv: "C", Version: "2c", UpstreamDeviceIDs: []string{"dist-01"}},
	)
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)

	_ = tracker.ApplyBatch(cfg, []PollOutcome{
		success("core-01", now, nil),
		success("dist-01", now, nil),
		success("access-01", now, nil),
	})

	// Drive all three to failure threshold together.
	_ = tracker.ApplyBatch(cfg, []PollOutcome{
		failure("core-01", now.Add(time.Minute)),
		failure("dist-01", now.Add(time.Minute)),
		failure("access-01", now.Add(time.Minute)),
	})
	events := tracker.ApplyBatch(cfg, []PollOutcome{
		failure("core-01", now.Add(2*time.Minute)),
		failure("dist-01", now.Add(2*time.Minute)),
		failure("access-01", now.Add(2*time.Minute)),
	})
	byID := eventMap(events)

	if byID["core-01"].State != StateCritical {
		t.Fatalf("core=%#v", byID["core-01"])
	}
	if byID["dist-01"].State != StateUnknown || byID["dist-01"].Reason != ReasonUpstreamUnreachable {
		t.Fatalf("dist=%#v", byID["dist-01"])
	}
	if len(byID["dist-01"].RootCauseDeviceIDs) != 1 || byID["dist-01"].RootCauseDeviceIDs[0] != "core-01" {
		t.Fatalf("dist root causes=%v", byID["dist-01"].RootCauseDeviceIDs)
	}
	if byID["access-01"].State != StateUnknown {
		t.Fatalf("access=%#v", byID["access-01"])
	}
	if len(byID["access-01"].RootCauseDeviceIDs) != 1 || byID["access-01"].RootCauseDeviceIDs[0] != "core-01" {
		t.Fatalf("access root causes=%v", byID["access-01"].RootCauseDeviceIDs)
	}
}

func TestAlternatePathSuccessStaysHealthy(t *testing.T) {
	t.Parallel()

	tracker := NewTracker()
	cfg := baseCfg(
		config.DeviceConfig{ID: "core-01", Host: "10.0.0.1", CommunityEnv: "C", Version: "2c"},
		config.DeviceConfig{ID: "access-01", Host: "10.0.0.3", CommunityEnv: "C", Version: "2c", UpstreamDeviceIDs: []string{"core-01"}},
	)
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	_ = tracker.ApplyBatch(cfg, []PollOutcome{
		success("core-01", now, nil),
		success("access-01", now, nil),
	})
	_ = tracker.ApplyBatch(cfg, []PollOutcome{failure("core-01", now.Add(time.Minute))})
	events := tracker.ApplyBatch(cfg, []PollOutcome{
		failure("core-01", now.Add(2*time.Minute)),
		success("access-01", now.Add(2*time.Minute), nil),
	})
	byID := eventMap(events)
	if byID["core-01"].State != StateCritical {
		t.Fatalf("core=%#v", byID["core-01"])
	}
	if ev, ok := byID["access-01"]; ok && ev.State != StateHealthy {
		t.Fatalf("access should stay healthy, got %#v", ev)
	}
	dev, _ := tracker.Device("access-01")
	if dev.State != StateHealthy {
		t.Fatalf("access ledger=%#v", dev)
	}
}

func TestIndependentChildFailureWithRespondingUpstream(t *testing.T) {
	t.Parallel()

	tracker := NewTracker()
	cfg := baseCfg(
		config.DeviceConfig{ID: "core-01", Host: "10.0.0.1", CommunityEnv: "C", Version: "2c"},
		config.DeviceConfig{ID: "access-01", Host: "10.0.0.3", CommunityEnv: "C", Version: "2c", UpstreamDeviceIDs: []string{"core-01"}},
	)
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	_ = tracker.ApplyBatch(cfg, []PollOutcome{
		success("core-01", now, nil),
		success("access-01", now, nil),
	})
	_ = tracker.ApplyBatch(cfg, []PollOutcome{
		success("core-01", now.Add(time.Minute), nil),
		failure("access-01", now.Add(time.Minute)),
	})
	events := tracker.ApplyBatch(cfg, []PollOutcome{
		success("core-01", now.Add(2*time.Minute), nil),
		failure("access-01", now.Add(2*time.Minute)),
	})
	byID := eventMap(events)
	if byID["access-01"].State != StateCritical || byID["access-01"].Reason != ReasonDirectUnreachable {
		t.Fatalf("access=%#v", byID["access-01"])
	}
}

func TestRecoveryTransitions(t *testing.T) {
	t.Parallel()

	tracker := NewTracker()
	cfg := baseCfg(config.DeviceConfig{ID: "core-01", Host: "10.0.0.1", CommunityEnv: "C", Version: "2c"})
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)

	_ = tracker.ApplyBatch(cfg, []PollOutcome{success("core-01", now, temp(70))})
	dev, _ := tracker.Device("core-01")
	if dev.State != StateWarning {
		t.Fatalf("want warning got %#v", dev)
	}

	events := tracker.ApplyBatch(cfg, []PollOutcome{success("core-01", now.Add(time.Minute), temp(40))})
	if len(events) != 1 || events[0].Transition != TransitionRecovered || events[0].Reason != ReasonRecovered {
		t.Fatalf("temp recovery: %#v", events)
	}

	_ = tracker.ApplyBatch(cfg, []PollOutcome{failure("core-01", now.Add(2*time.Minute))})
	_ = tracker.ApplyBatch(cfg, []PollOutcome{failure("core-01", now.Add(3*time.Minute))})
	events = tracker.ApplyBatch(cfg, []PollOutcome{success("core-01", now.Add(4*time.Minute), nil)})
	if len(events) != 1 || events[0].Transition != TransitionRecovered || events[0].State != StateHealthy {
		t.Fatalf("critical recovery: %#v", events)
	}
	dev, _ = tracker.Device("core-01")
	if dev.FailureCount != 0 {
		t.Fatalf("failure count not cleared: %#v", dev)
	}
}

func TestReloadRetention(t *testing.T) {
	t.Parallel()

	tracker := NewTracker()
	cfg := baseCfg(
		config.DeviceConfig{ID: "keep-01", Host: "10.0.0.1", CommunityEnv: "C", Version: "2c"},
		config.DeviceConfig{ID: "drop-01", Host: "10.0.0.2", CommunityEnv: "C", Version: "2c"},
	)
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	_ = tracker.ApplyBatch(cfg, []PollOutcome{
		success("keep-01", now, nil),
		success("drop-01", now, nil),
	})
	_ = tracker.ApplyBatch(cfg, []PollOutcome{failure("keep-01", now.Add(time.Minute))})

	tracker.Prune([]string{"keep-01", "new-01"})
	if _, ok := tracker.Device("drop-01"); ok {
		t.Fatal("drop-01 should be pruned")
	}
	keep, ok := tracker.Device("keep-01")
	if !ok || keep.FailureCount != 1 || keep.State != StateHealthy || keep.LastSuccessAt != now {
		t.Fatalf("retained keep-01: ok=%v %#v", ok, keep)
	}
	if _, ok := tracker.Device("new-01"); ok {
		t.Fatal("new device must stay unobserved until ApplyBatch")
	}

	cfg2 := baseCfg(
		config.DeviceConfig{ID: "keep-01", Host: "10.0.0.1", CommunityEnv: "C", Version: "2c"},
		config.DeviceConfig{ID: "new-01", Host: "10.0.0.3", CommunityEnv: "C", Version: "2c"},
	)
	events := tracker.ApplyBatch(cfg2, []PollOutcome{success("new-01", now.Add(2*time.Minute), nil)})
	byID := eventMap(events)
	if byID["new-01"].Transition != TransitionInitial || byID["new-01"].State != StateHealthy {
		t.Fatalf("new device: %#v", byID["new-01"])
	}
	keep, _ = tracker.Device("keep-01")
	if keep.FailureCount != 1 {
		t.Fatalf("keep failure count lost: %#v", keep)
	}
}

func TestDiamondTopologyRedundantUpstream(t *testing.T) {
	t.Parallel()

	tracker := NewTracker()
	cfg := baseCfg(
		config.DeviceConfig{ID: "a", Host: "10.0.0.1", CommunityEnv: "C", Version: "2c"},
		config.DeviceConfig{ID: "b", Host: "10.0.0.2", CommunityEnv: "C", Version: "2c", UpstreamDeviceIDs: []string{"a"}},
		config.DeviceConfig{ID: "c", Host: "10.0.0.3", CommunityEnv: "C", Version: "2c", UpstreamDeviceIDs: []string{"a"}},
		config.DeviceConfig{ID: "d", Host: "10.0.0.4", CommunityEnv: "C", Version: "2c", UpstreamDeviceIDs: []string{"b", "c"}},
	)
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	_ = tracker.ApplyBatch(cfg, []PollOutcome{
		success("a", now, nil),
		success("b", now, nil),
		success("c", now, nil),
		success("d", now, nil),
	})

	_ = tracker.ApplyBatch(cfg, []PollOutcome{
		success("a", now.Add(time.Minute), nil),
		failure("b", now.Add(time.Minute)),
		success("c", now.Add(time.Minute), nil),
		success("d", now.Add(time.Minute), nil),
	})
	events := tracker.ApplyBatch(cfg, []PollOutcome{
		success("a", now.Add(2*time.Minute), nil),
		failure("b", now.Add(2*time.Minute)),
		success("c", now.Add(2*time.Minute), nil),
		success("d", now.Add(2*time.Minute), nil),
	})
	byID := eventMap(events)
	if byID["b"].State != StateCritical {
		t.Fatalf("b should be critical: %#v", byID["b"])
	}
	if ev, ok := byID["d"]; ok && ev.State == StateUnknown {
		t.Fatalf("d must not become unknown while c is up: %#v", ev)
	}
	dev, _ := tracker.Device("d")
	if dev.State != StateHealthy {
		t.Fatalf("d ledger=%#v", dev)
	}

	// When both B and C are down and D fails, D becomes Unknown.
	_ = tracker.ApplyBatch(cfg, []PollOutcome{
		success("a", now.Add(3*time.Minute), nil),
		failure("b", now.Add(3*time.Minute)),
		failure("c", now.Add(3*time.Minute)),
		failure("d", now.Add(3*time.Minute)),
	})
	events = tracker.ApplyBatch(cfg, []PollOutcome{
		success("a", now.Add(4*time.Minute), nil),
		failure("b", now.Add(4*time.Minute)),
		failure("c", now.Add(4*time.Minute)),
		failure("d", now.Add(4*time.Minute)),
	})
	byID = eventMap(events)
	if byID["d"].State != StateUnknown {
		t.Fatalf("d should be unknown when all upstreams down: %#v", byID["d"])
	}
	if len(byID["d"].UnavailableUpstreamDeviceIDs) != 2 {
		t.Fatalf("unavailable=%v", byID["d"].UnavailableUpstreamDeviceIDs)
	}
}

func TestNoEventWhenUnchangedAndNoReasserted(t *testing.T) {
	t.Parallel()

	tracker := NewTracker()
	cfg := baseCfg(config.DeviceConfig{ID: "core-01", Host: "10.0.0.1", CommunityEnv: "C", Version: "2c"})
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	_ = tracker.ApplyBatch(cfg, []PollOutcome{success("core-01", now, nil)})
	events := tracker.ApplyBatch(cfg, []PollOutcome{success("core-01", now.Add(time.Minute), nil)})
	if len(events) != 0 {
		t.Fatalf("unchanged must not emit: %#v", events)
	}
}

func TestAlertsEnabledToggleReasserts(t *testing.T) {
	t.Parallel()

	tracker := NewTracker()
	cfg := baseCfg(config.DeviceConfig{ID: "core-01", Host: "10.0.0.1", CommunityEnv: "C", Version: "2c"})
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	first := tracker.ApplyBatch(cfg, []PollOutcome{success("core-01", now, nil)})
	if len(first) != 1 || !first[0].AlertsEnabled {
		t.Fatalf("first emit=%#v", first)
	}

	disabled := false
	cfg.Devices[0].AlertsEnabled = &disabled
	events := tracker.ApplyBatch(cfg, []PollOutcome{success("core-01", now.Add(time.Minute), nil)})
	if len(events) != 1 {
		t.Fatalf("expected reassert on alerts_enabled toggle, got %#v", events)
	}
	if events[0].Transition != TransitionReasserted {
		t.Fatalf("transition=%s want reasserted", events[0].Transition)
	}
	if events[0].AlertsEnabled {
		t.Fatal("expected alerts_enabled=false")
	}
}

func TestDeterministicEventOrdering(t *testing.T) {
	t.Parallel()

	tracker := NewTracker()
	cfg := baseCfg(
		config.DeviceConfig{ID: "z-device", Host: "10.0.0.3", CommunityEnv: "C", Version: "2c"},
		config.DeviceConfig{ID: "a-device", Host: "10.0.0.1", CommunityEnv: "C", Version: "2c"},
		config.DeviceConfig{ID: "m-device", Host: "10.0.0.2", CommunityEnv: "C", Version: "2c"},
	)
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	events := tracker.ApplyBatch(cfg, []PollOutcome{
		success("z-device", now, nil),
		success("a-device", now, nil),
		success("m-device", now, nil),
	})
	if len(events) != 3 {
		t.Fatalf("events=%d", len(events))
	}
	if events[0].DeviceID != "a-device" || events[1].DeviceID != "m-device" || events[2].DeviceID != "z-device" {
		t.Fatalf("order=%v %v %v", events[0].DeviceID, events[1].DeviceID, events[2].DeviceID)
	}
}

func TestCPUMemoryPowerDoNotAffectHealth(t *testing.T) {
	t.Parallel()

	tracker := NewTracker()
	cfg := baseCfg(config.DeviceConfig{ID: "core-01", Host: "10.0.0.1", CommunityEnv: "C", Version: "2c"})
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	// High CPU/memory are not represented on PollOutcome; only temperature is.
	events := tracker.ApplyBatch(cfg, []PollOutcome{success("core-01", now, nil)})
	if events[0].State != StateHealthy {
		t.Fatalf("expected healthy without temp: %#v", events[0])
	}
}

func TestDuplicateUpstreamIDsNormalized(t *testing.T) {
	t.Parallel()

	tracker := NewTracker()
	cfg := baseCfg(
		config.DeviceConfig{ID: "core-01", Host: "10.0.0.1", CommunityEnv: "C", Version: "2c"},
		config.DeviceConfig{
			ID: "access-01", Host: "10.0.0.3", CommunityEnv: "C", Version: "2c",
			UpstreamDeviceIDs: []string{"core-01", "core-01"},
		},
	)
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	events := tracker.ApplyBatch(cfg, []PollOutcome{
		success("core-01", now, nil),
		success("access-01", now, nil),
	})
	byID := eventMap(events)
	if len(byID["access-01"].UpstreamDeviceIDs) != 1 || byID["access-01"].UpstreamDeviceIDs[0] != "core-01" {
		t.Fatalf("upstreams=%v", byID["access-01"].UpstreamDeviceIDs)
	}
}

func TestSnapshotGauges(t *testing.T) {
	t.Parallel()

	tracker := NewTracker()
	cfg := baseCfg(
		config.DeviceConfig{ID: "core-01", Host: "10.0.0.1", CommunityEnv: "C", Version: "2c"},
		config.DeviceConfig{ID: "access-01", Host: "10.0.0.3", CommunityEnv: "C", Version: "2c", UpstreamDeviceIDs: []string{"core-01"}},
	)
	now := time.Date(2026, 7, 17, 12, 0, 0, 0, time.UTC)
	_ = tracker.ApplyBatch(cfg, []PollOutcome{
		success("core-01", now, nil),
		success("access-01", now, nil),
	})
	_ = tracker.ApplyBatch(cfg, []PollOutcome{
		failure("core-01", now.Add(time.Minute)),
		failure("access-01", now.Add(time.Minute)),
	})
	_ = tracker.ApplyBatch(cfg, []PollOutcome{
		failure("core-01", now.Add(2*time.Minute)),
		failure("access-01", now.Add(2*time.Minute)),
	})
	snap := tracker.Snapshot()
	if snap.DevicesByState[StateCritical] != 1 || snap.DevicesByState[StateUnknown] != 1 {
		t.Fatalf("snapshot=%#v", snap)
	}
	if snap.DependencyImpacted != 1 {
		t.Fatalf("impacted=%d", snap.DependencyImpacted)
	}
}
