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
