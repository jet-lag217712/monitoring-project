package transform_test

import (
	"testing"
	"time"

	"github.com/equate/ogsd/services/ingestion-service/internal/transform"
	"github.com/equate/ogsd/services/ingestion-service/internal/validate"
)

func TestUUIDV5_Stable(t *testing.T) {
	a := transform.SiteUUID("site-001")
	b := transform.SiteUUID("site-001")
	if a != b {
		t.Fatalf("%s != %s", a, b)
	}
	devA := transform.DeviceUUID("site-001", "dev-001")
	devB := transform.DeviceUUID("site-001", "dev-001")
	if devA != devB {
		t.Fatalf("%s != %s", devA, devB)
	}
	ifaceA := transform.InterfaceUUID(devA, 2)
	ifaceB := transform.InterfaceUUID(devA, 2)
	if ifaceA != ifaceB {
		t.Fatalf("%s != %s", ifaceA, ifaceB)
	}
}

func TestUUIDV5_DifferentIDsDiffer(t *testing.T) {
	if transform.SiteUUID("a") == transform.SiteUUID("b") {
		t.Fatal("site UUIDs should differ")
	}
	if transform.DeviceUUID("s", "d1") == transform.DeviceUUID("s", "d2") {
		t.Fatal("device UUIDs should differ")
	}
	dev := transform.DeviceUUID("s", "d")
	if transform.InterfaceUUID(dev, 1) == transform.InterfaceUUID(dev, 2) {
		t.Fatal("interface UUIDs should differ")
	}
}

func TestDeviceSample_FromValidated(t *testing.T) {
	ts := time.Date(2026, 6, 1, 18, 0, 0, 0, time.UTC)
	sample := transform.DeviceSampleFromValidated(validate.DeviceMessage{
		SiteID:    "site-001",
		DeviceID:  "dev-001",
		Timestamp: ts,
		Metric:    "uptime_seconds",
		Value:     42,
	})
	if sample.SiteName != "site-001" || sample.DeviceHostname != "dev-001" {
		t.Fatalf("%+v", sample)
	}
	if sample.SiteUUID != transform.SiteUUID("site-001") {
		t.Fatal("site uuid mismatch")
	}
	if sample.DeviceUUID != transform.DeviceUUID("site-001", "dev-001") {
		t.Fatal("device uuid mismatch")
	}
	if sample.MetricName != "uptime_seconds" || sample.Value != 42 || !sample.CollectedAt.Equal(ts) {
		t.Fatalf("%+v", sample)
	}
}

func TestInterfaceSample_FromValidated(t *testing.T) {
	ts := time.Date(2026, 6, 1, 18, 0, 0, 0, time.UTC)
	sample := transform.InterfaceSampleFromValidated(validate.InterfaceMessage{
		SiteID:    "site-001",
		DeviceID:  "dev-001",
		Timestamp: ts,
		IfIndex:   3,
		InOctets:  1,
		OutOctets: 2,
		InErrors:  3,
		OutErrors: 4,
	})
	dev := transform.DeviceUUID("site-001", "dev-001")
	if sample.InterfaceUUID != transform.InterfaceUUID(dev, 3) {
		t.Fatal("interface uuid mismatch")
	}
	if sample.IfIndex != 3 || sample.OutErrors != 4 {
		t.Fatalf("%+v", sample)
	}
}
