package status

import "testing"

func TestStoreRecordAndSnapshot(t *testing.T) {
	store := New()
	store.SetRevision("revision-abc")
	store.RecordPoll(DevicePoll{
		DeviceID:   "dev-001",
		Result:     PollSuccess,
		SysName:    "core-01",
		Interfaces: InterfaceSummary{Selected: 2},
	})
	store.RecordReload(true, "", "revision-def")

	snap := store.Snapshot()
	if snap.ConfigRevision != "revision-def" {
		t.Fatalf("revision=%q", snap.ConfigRevision)
	}
	if snap.Reload == nil || !snap.Reload.Success {
		t.Fatalf("reload=%#v", snap.Reload)
	}
	poll, ok := snap.Devices["dev-001"]
	if !ok || poll.SysName != "core-01" || poll.Interfaces.Selected != 2 {
		t.Fatalf("device poll=%#v ok=%v", poll, ok)
	}

	store.Prune(map[string]struct{}{"other": {}})
	if _, ok := store.Device("dev-001"); ok {
		t.Fatal("expected pruned device")
	}
}
