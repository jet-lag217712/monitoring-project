package poller

import (
	"testing"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
)

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
