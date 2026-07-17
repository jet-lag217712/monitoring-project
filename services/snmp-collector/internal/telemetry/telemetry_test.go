package telemetry_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/events"
	"github.com/equate/ogsd/services/snmp-collector/internal/health"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/core"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/readings"
	"github.com/equate/ogsd/services/snmp-collector/internal/telemetry"
)

func TestDeviceEvents_ModeMatrix(t *testing.T) {
	result := readings.NewDevicePollResult("site-001", "dev-001", "10.0.0.1", time.Now().UTC(), core.DeviceIdentity{
		SysObjectID:   "1.2.3",
		SysName:       "dev-001",
		SysDescr:      "lab",
		UptimeSeconds: 10,
	}, []core.InterfaceReading{{
		IfIndex:     1,
		IfName:      "Gi0/1",
		AdminStatus: "up",
		OperStatus:  "up",
		SpeedBPS:    1000,
		InOctets:    1,
		OutOctets:   2,
		HasCounters: true,
	}})
	result.Interfaces[0].Selection = readings.Selected
	ctx := telemetry.Context{
		SiteID:         "site-001",
		CollectorID:    "collector-1",
		ConfigRevision: "revision-test",
		EmittedAt:      time.Now().UTC(),
	}

	v1 := telemetry.DeviceEvents(telemetry.ModeV1, ctx, result)
	if len(v1) != 2 {
		t.Fatalf("v1 events=%d want 2", len(v1))
	}
	for _, ev := range v1 {
		if strings.Contains(ev.Topic(), "telemetry/v2") {
			t.Fatalf("v1 mode emitted v2 topic %s", ev.Topic())
		}
	}

	v2 := telemetry.DeviceEvents(telemetry.ModeV2, ctx, result)
	if len(v2) != 2 {
		t.Fatalf("v2 events=%d want 2", len(v2))
	}
	for _, ev := range v2 {
		if !strings.Contains(ev.Topic(), "telemetry/v2") {
			t.Fatalf("v2 mode emitted non-v2 topic %s", ev.Topic())
		}
	}

	both := telemetry.DeviceEvents(telemetry.ModeBoth, ctx, result)
	if len(both) != 4 {
		t.Fatalf("both events=%d want 4", len(both))
	}
}

func TestDeviceTelemetry_JSONShape(t *testing.T) {
	temp := 42.0
	result := readings.NewDevicePollResult("site-001", "dist-01", "10.0.0.1", time.Date(2026, 7, 16, 18, 0, 0, 0, time.UTC), core.DeviceIdentity{
		SysObjectID:   "1.3.6.1.4.1.9.1.9999",
		SysName:       "dist-01",
		SysDescr:      "Sanitized Cisco lab fixture",
		UptimeSeconds: 12345,
	}, nil)
	result.Vendor = readings.VendorReadings{
		Profile:      "cisco",
		Capabilities: readings.CapabilityCPU | readings.CapabilityTemperature,
		CPU:          &readings.ScalarReading{Value: 34.2},
		Temperatures: []readings.ComponentReading{{
			Index: 1,
			Name:  "inlet",
			Value: &temp,
			Unit:  "celsius",
			Status: "ok",
		}},
	}
	ev := telemetry.DeviceTelemetry(telemetry.Context{
		SiteID:         "site-001",
		CollectorID:    "collector-west-01",
		ConfigRevision: "revision-test",
		EmittedAt:      time.Date(2026, 7, 16, 18, 0, 2, 0, time.UTC),
	}, result)

	if ev.Topic() != "site/site-001/device/dist-01/telemetry/v2/device" {
		t.Fatalf("topic=%s", ev.Topic())
	}
	raw, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"schema_version", "event_id", "event_type", "site_id", "collector_id", "device_id", "observed_at", "emitted_at", "config_revision", "payload"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing envelope field %s", key)
		}
	}
	if decoded["schema_version"] != "2.0" {
		t.Fatalf("schema_version=%v", decoded["schema_version"])
	}
	if strings.Contains(string(raw), "community") || strings.Contains(string(raw), "password") {
		t.Fatal("payload leaked secret-looking fields")
	}
}

func TestHealthEvent_TopicAndPayload(t *testing.T) {
	prev := health.StateHealthy
	ev := telemetry.HealthEvent(telemetry.Context{
		SiteID:         "site-001",
		CollectorID:    "collector-1",
		ConfigRevision: "revision-test",
		EmittedAt:      time.Now().UTC(),
	}, "site-001", health.Event{
		DeviceID:                     "access-01",
		State:                        health.StateUnknown,
		Reason:                       health.ReasonUpstreamUnreachable,
		PreviousState:                &prev,
		Transition:                   health.TransitionEntered,
		FailureCount:                 2,
		FailureThreshold:             2,
		UpstreamDeviceIDs:            []string{"dist-01"},
		UnavailableUpstreamDeviceIDs: []string{"dist-01"},
		RootCauseDeviceIDs:           []string{"core-01"},
		ObservedAt:                   time.Now().UTC(),
	})
	if ev.Topic() != "site/site-001/device/access-01/telemetry/v2/health" {
		t.Fatalf("topic=%s", ev.Topic())
	}
	if ev.EventType != events.EventTypeHealthState {
		t.Fatalf("event_type=%s", ev.EventType)
	}
}

func TestHeartbeat_DepthBeforeSelf(t *testing.T) {
	ev := telemetry.Heartbeat(telemetry.Context{
		SiteID:         "site-001",
		CollectorID:    "collector-1",
		ConfigRevision: "revision-test",
		EmittedAt:      time.Now().UTC(),
	}, telemetry.HeartbeatInput{
		Hostname:         "host",
		Version:          "",
		GitCommit:        "",
		BuildTime:        "",
		UptimeSeconds:    5,
		SQLiteQueueDepth: 14,
		MemoryUsageBytes: 100,
		GoroutineCount:   3,
		ObservedAt:       time.Now().UTC(),
	})
	if ev.Payload.Version != "unknown" || ev.Payload.GitCommit != "unknown" || ev.Payload.BuildTime != "unknown" {
		t.Fatalf("build fallbacks=%+v", ev.Payload)
	}
	if ev.Payload.SQLiteQueueDepth != 14 {
		t.Fatalf("depth=%d", ev.Payload.SQLiteQueueDepth)
	}
	if ev.Topic() != "site/site-001/collector/collector-1/telemetry/v2/heartbeat" {
		t.Fatalf("topic=%s", ev.Topic())
	}
}

func TestHealthEvents_SkippedForV1(t *testing.T) {
	evs := telemetry.HealthEvents(telemetry.ModeV1, telemetry.Context{}, "site", []health.Event{{DeviceID: "d1"}})
	if len(evs) != 0 {
		t.Fatalf("v1 should skip health publish, got %d", len(evs))
	}
}

func TestPowerComponents_NormalizeAristaUnits(t *testing.T) {
	result := readings.NewDevicePollResult("site-001", "arista-01", "10.0.0.1", time.Now().UTC(), core.DeviceIdentity{
		SysObjectID:   "1.2.3",
		SysName:       "arista-01",
		SysDescr:      "lab",
		UptimeSeconds: 10,
	}, nil)
	result.Vendor = readings.VendorReadings{
		Profile: "arista",
		Power: []readings.ComponentReading{
			{Index: 1, Name: "psu1", Unit: "volts_ac", Status: "ok"},
			{Index: 2, Name: "psu2", Unit: "volts_dc", Status: "ok"},
			{Index: 3, Name: "psu3", Unit: "amperes", Status: "ok"},
		},
	}
	ev := telemetry.DeviceTelemetry(telemetry.Context{
		SiteID:      "site-001",
		CollectorID: "collector-1",
		EmittedAt:   time.Now().UTC(),
	}, result)

	units := make([]string, 0, len(ev.Payload.PowerComponents))
	for _, c := range ev.Payload.PowerComponents {
		units = append(units, c.Unit)
	}
	want := []string{"volts", "volts", "amps"}
	for i, u := range want {
		if units[i] != u {
			t.Fatalf("unit[%d]=%q want %q (all=%v)", i, units[i], u, units)
		}
	}
}
