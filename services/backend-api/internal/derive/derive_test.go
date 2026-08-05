package derive_test

import (
	"testing"
	"time"

	"github.com/equate/ogsd/services/backend-api/internal/derive"
)

func TestDeviceOnline(t *testing.T) {
	now := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	threshold := 5 * time.Minute
	recent := now.Add(-2 * time.Minute)
	stale := now.Add(-10 * time.Minute)

	tests := []struct {
		name     string
		status   string
		lastSeen *time.Time
		want     bool
	}{
		{"online recent", "online", &recent, true},
		{"online stale", "online", &stale, false},
		{"online nil last_seen", "online", nil, false},
		{"unknown recent", "unknown", &recent, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := derive.DeviceOnline(tt.status, tt.lastSeen, now, threshold)
			if got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}

func TestDeviceStatusCode(t *testing.T) {
	if derive.DeviceStatusCode(true) != 1 {
		t.Fatal("online should be 1")
	}
	if derive.DeviceStatusCode(false) != 3 {
		t.Fatal("offline should be 3")
	}
}

func TestHealthStatusCode(t *testing.T) {
	tests := []struct {
		state  string
		online bool
		want   int
	}{
		{"healthy", true, derive.StatusHealthy},
		{"warning", true, derive.StatusWarning},
		{"critical", false, derive.StatusCritical},
		{"unknown", false, derive.StatusUnknown},
		{"", true, derive.StatusHealthy},
		{"", false, derive.StatusCritical},
		{"healthy", false, derive.StatusCritical},
		{"warning", false, derive.StatusCritical},
		{"critical", true, derive.StatusCritical},
		{"unknown", true, derive.StatusUnknown},
	}
	for _, tt := range tests {
		got := derive.HealthStatusCode(tt.state, tt.online)
		if got != tt.want {
			t.Fatalf("state=%q online=%v got %d want %d", tt.state, tt.online, got, tt.want)
		}
	}
}

func TestProjectDeviceStatus_StaleHealthyFallsBackToCritical(t *testing.T) {
	proj := derive.ProjectDeviceStatus("healthy", true, "", 0, nil, nil, nil, false, true)
	if proj.Status != derive.StatusCritical {
		t.Fatalf("stale healthy device should be critical, got %d", proj.Status)
	}
}

func TestProjectDeviceStatus_CriticalVsUnknown(t *testing.T) {
	critical := derive.ProjectDeviceStatus("critical", true, "poll_failed", 2, nil, nil, nil, false, true)
	if critical.Status != derive.StatusCritical {
		t.Fatalf("critical status=%d", critical.Status)
	}
	if critical.StatusReason != "poll_failed" || critical.FailureCount == nil || *critical.FailureCount != 2 {
		t.Fatalf("critical projection incomplete: %+v", critical)
	}

	unknown := derive.ProjectDeviceStatus(
		"unknown", true, "upstream_unreachable", 2,
		[]string{"dist-01", "dist-02"},
		[]string{"dist-01", "dist-02"},
		[]string{"core-01"},
		false,
		true,
	)
	if unknown.Status != derive.StatusUnknown {
		t.Fatalf("unknown status=%d", unknown.Status)
	}
	if unknown.StatusReason != "upstream_unreachable" {
		t.Fatalf("reason=%q", unknown.StatusReason)
	}
	if len(unknown.RootCauseDeviceIDs) != 1 || unknown.RootCauseDeviceIDs[0] != "core-01" {
		t.Fatalf("root cause=%v", unknown.RootCauseDeviceIDs)
	}

	fallback := derive.ProjectDeviceStatus("", false, "", 0, nil, nil, nil, true, true)
	if fallback.Status != derive.StatusHealthy {
		t.Fatalf("v1 fallback online should be healthy, got %d", fallback.Status)
	}
}

func TestSiteHealthCounts_DistinguishCriticalAndUnknown(t *testing.T) {
	var counts derive.SiteHealthCounts
	counts.Accumulate(derive.StatusCritical, nil, true)
	counts.Accumulate(derive.StatusUnknown, []string{"dist-01"}, true)
	counts.Accumulate(derive.StatusWarning, nil, true)
	counts.Accumulate(derive.StatusHealthy, nil, true)

	if counts.CriticalCount != 1 {
		t.Fatalf("critical_count=%d", counts.CriticalCount)
	}
	if counts.UnknownCount != 1 || counts.DependencyImpactedCount != 1 {
		t.Fatalf("unknown=%d impacted=%d", counts.UnknownCount, counts.DependencyImpactedCount)
	}
	if counts.WarningCount != 1 || counts.HealthyCount != 1 {
		t.Fatalf("warning=%d healthy=%d", counts.WarningCount, counts.HealthyCount)
	}
	if derive.SiteStatusFromHealth(counts) != "alert" {
		t.Fatal("site with critical should be alert")
	}

	ignored := derive.SiteHealthCounts{}
	ignored.Accumulate(derive.StatusCritical, nil, false)
	if ignored.CriticalCount != 0 {
		t.Fatal("administratively ignored critical must not count")
	}

	warningOnly := derive.SiteHealthCounts{WarningCount: 1}
	if derive.SiteStatusFromHealth(warningOnly) != "caution" {
		t.Fatal("warning-only site should be caution")
	}
	unknownOnly := derive.SiteHealthCounts{UnknownCount: 1}
	if derive.SiteStatusFromHealth(unknownOnly) != "caution" {
		t.Fatal("unknown-only site should be caution, not alert")
	}
	if derive.SiteStatusFromHealth(derive.SiteHealthCounts{}) != "ok" {
		t.Fatal("empty site should be ok")
	}
}

func TestSiteStatus(t *testing.T) {
	if derive.SiteStatus(false) != "ok" {
		t.Fatal("expected ok")
	}
	if derive.SiteStatus(true) != "alert" {
		t.Fatal("expected alert")
	}
}

func TestDeviceMapKey(t *testing.T) {
	tests := []struct {
		ip, hostname, want string
	}{
		{"10.10.0.1", "core-sw", "10.10.0.1"},
		{"0.0.0.0", "core-sw", "core-sw"},
		{"0.0.0.0/32", "core-sw", "core-sw"},
		{"", "core-sw", "core-sw"},
		{"0.0.0.0", "", "0.0.0.0"},
	}
	for _, tt := range tests {
		got := derive.DeviceMapKey(tt.ip, tt.hostname)
		if got != tt.want {
			t.Fatalf("ip=%q host=%q got %q want %q", tt.ip, tt.hostname, got, tt.want)
		}
	}
}

func TestUptimeDays(t *testing.T) {
	if derive.UptimeDays(nil) != nil {
		t.Fatal("nil input should return nil")
	}
	sec := 172800.0 // 2 days
	got := derive.UptimeDays(&sec)
	if got == nil || *got != 2.0 {
		t.Fatalf("got %v want 2.0", got)
	}
}

func TestLocationOrName(t *testing.T) {
	if derive.LocationOrName("District Office", "district-office") != "District Office" {
		t.Fatal("prefer location")
	}
	if derive.LocationOrName("", "district-office") != "district-office" {
		t.Fatal("fallback to name")
	}
}

func TestAvgNullable(t *testing.T) {
	a, b := 10.0, 20.0
	if got := derive.AvgNullable([]*float64{&a, nil, &b}); got != 15.0 {
		t.Fatalf("got %v", got)
	}
	if got := derive.AvgNullable(nil); got != 0 {
		t.Fatalf("empty avg=%v", got)
	}
}

func TestDeviceUUID_Stable(t *testing.T) {
	a := derive.DeviceUUID("site-001", "dev-001")
	b := derive.DeviceUUID("site-001", "dev-001")
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
	if derive.DeviceUUID("site-a", "dev-001") == derive.DeviceUUID("site-b", "dev-001") {
		t.Fatal("device UUIDs must differ across sites")
	}
}
