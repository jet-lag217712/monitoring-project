package poller

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/gosnmp/gosnmp"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/metrics"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/core"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/profile"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/readings"
)

type failingProfile struct{}

func (failingProfile) Name() string                      { return "cisco" }
func (failingProfile) Capabilities() readings.Capability { return readings.CapabilityCPU }
func (failingProfile) ExactObjectIDs() []string          { return []string{"1.3.6.1.4.1.9.1.1"} }
func (failingProfile) ObjectIDPrefixes() []string        { return nil }
func (failingProfile) GenericVendorPrefix() string       { return "1.3.6.1.4.1.9" }
func (failingProfile) Collect(context.Context, profile.Client) (readings.VendorReadings, error) {
	return readings.VendorReadings{
		Profile:      "cisco",
		Capabilities: readings.CapabilityCPU,
		CPU:          &readings.ScalarReading{Value: 12, SourceOID: "1.2.3"},
	}, errors.New("context deadline exceeded")
}

type noopClient struct{}

func (noopClient) Get(context.Context, []string) (*gosnmp.SnmpPacket, error) {
	return nil, errors.New("unexpected Get")
}
func (noopClient) Walk(context.Context, string, gosnmp.WalkFunc) error {
	return errors.New("unexpected Walk")
}

func TestEnrichProfilePreservesCoreOnFailure(t *testing.T) {
	t.Parallel()

	p := &Poller{
		metrics:  metrics.NewWithRegisterer(prometheus.NewRegistry()),
		log:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		registry: profile.NewRegistry(failingProfile{}),
	}
	result := readings.NewDevicePollResult(
		"site-001",
		"dev-001",
		"127.0.0.1",
		time.Unix(1700000000, 0).UTC(),
		core.DeviceIdentity{
			SysObjectID:   "1.3.6.1.4.1.9.1.1",
			SysName:       "core-switch",
			SysDescr:      "should not select profile",
			UptimeSeconds: 42,
		},
		[]core.InterfaceReading{{IfIndex: 1, HasCounters: true}},
	)

	p.enrichProfile(context.Background(), noopClient{}, &result)

	if result.Identity.UptimeSeconds != 42 || result.Identity.SysName != "core-switch" {
		t.Fatalf("core identity mutated: %#v", result.Identity)
	}
	if len(result.Interfaces) != 1 || result.Interfaces[0].Reading.IfIndex != 1 {
		t.Fatalf("core interfaces mutated: %#v", result.Interfaces)
	}
	if result.Vendor.Profile != "cisco" || result.Vendor.CPU == nil || result.Vendor.CPU.Value != 12 {
		t.Fatalf("partial vendor readings missing: %#v", result.Vendor)
	}
}

func TestNextDueWaitEmptyScheduleRechecksSoon(t *testing.T) {
	wait := nextDueWait(map[string]time.Time{}, time.Now())
	if wait != time.Second {
		t.Fatalf("wait=%v, want %v", wait, time.Second)
	}
}

func TestPruneNextDueDropsRemovedDevices(t *testing.T) {
	now := time.Now()
	nextDue := map[string]time.Time{
		"removed": now.Add(-time.Second),
		"active":  now.Add(time.Minute),
	}
	pruneNextDue(nextDue, []config.DeviceConfig{{ID: "active"}})

	if _, ok := nextDue["removed"]; ok {
		t.Fatal("stale removed device schedule was not pruned")
	}
	wait := nextDueWait(nextDue, now)
	if wait == 0 {
		t.Fatal("nextDueWait returned 0 after pruning stale schedules")
	}
	if wait != time.Minute {
		t.Fatalf("wait=%v, want %v", wait, time.Minute)
	}
}

func TestDueDevicesIgnoresRemovedSchedules(t *testing.T) {
	now := time.Now()
	nextDue := map[string]time.Time{
		"removed": now.Add(-time.Second),
		"active":  now.Add(time.Minute),
	}
	cfg := &config.Config{Devices: []config.DeviceConfig{{ID: "active"}}}
	pruneNextDue(nextDue, cfg.Devices)

	due := dueDevices(cfg, nextDue, false, now)
	if len(due) != 0 {
		t.Fatalf("due=%v, want empty", due)
	}
}
