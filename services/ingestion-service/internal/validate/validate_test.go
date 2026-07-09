package validate_test

import (
	"testing"
	"time"

	"github.com/equate/ogsd/services/ingestion-service/internal/validate"
)

func TestParseTopic_Device(t *testing.T) {
	tp, err := validate.ParseTopic("site/site-001/device/dev-001/metric/device")
	if err != nil {
		t.Fatal(err)
	}
	if tp.SiteID != "site-001" || tp.DeviceID != "dev-001" || tp.Kind != validate.KindDevice {
		t.Fatalf("%+v", tp)
	}
}

func TestParseTopic_Interface(t *testing.T) {
	tp, err := validate.ParseTopic("site/hub/device/rtr-01/metric/interface")
	if err != nil {
		t.Fatal(err)
	}
	if tp.Kind != validate.KindInterface {
		t.Fatalf("kind=%q", tp.Kind)
	}
}

func TestParseTopic_RejectBad(t *testing.T) {
	cases := []string{
		"bad",
		"site/a/device/b/metric/cpu",
		"site//device/b/metric/device",
		"foo/a/device/b/metric/device",
	}
	for _, topic := range cases {
		if _, err := validate.ParseTopic(topic); err == nil {
			t.Fatalf("expected error for %q", topic)
		}
	}
}

func TestValidateDevice_OK(t *testing.T) {
	payload := []byte(`{
		"timestamp":"2026-06-01T18:00:00Z",
		"site_id":"site-001",
		"device_id":"dev-001",
		"metric":"uptime_seconds",
		"value":123
	}`)
	msg, err := validate.Validate("site/site-001/device/dev-001/metric/device", payload)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Device == nil || msg.Device.Metric != "uptime_seconds" || msg.Device.Value != 123 {
		t.Fatalf("%+v", msg.Device)
	}
	want := time.Date(2026, 6, 1, 18, 0, 0, 0, time.UTC)
	if !msg.Device.Timestamp.Equal(want) {
		t.Fatalf("ts=%v", msg.Device.Timestamp)
	}
}

func TestValidateDevice_MissingMetric(t *testing.T) {
	payload := []byte(`{"timestamp":"2026-06-01T18:00:00Z","value":1}`)
	if _, err := validate.Validate("site/site-001/device/dev-001/metric/device", payload); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateDevice_BadTimestamp(t *testing.T) {
	payload := []byte(`{"timestamp":"not-a-time","metric":"uptime_seconds","value":1}`)
	if _, err := validate.Validate("site/site-001/device/dev-001/metric/device", payload); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidateDevice_TopicBodyMismatch(t *testing.T) {
	payload := []byte(`{
		"timestamp":"2026-06-01T18:00:00Z",
		"site_id":"other",
		"device_id":"dev-001",
		"metric":"uptime_seconds",
		"value":1
	}`)
	if _, err := validate.Validate("site/site-001/device/dev-001/metric/device", payload); err == nil {
		t.Fatal("expected mismatch error")
	}
}

func TestValidateInterface_OK(t *testing.T) {
	payload := []byte(`{
		"timestamp":"2026-06-01T18:00:00Z",
		"if_index":2,
		"in_octets":10,
		"out_octets":20,
		"in_errors":0,
		"out_errors":1
	}`)
	msg, err := validate.Validate("site/site-001/device/dev-001/metric/interface", payload)
	if err != nil {
		t.Fatal(err)
	}
	if msg.Interface == nil || msg.Interface.IfIndex != 2 || msg.Interface.OutErrors != 1 {
		t.Fatalf("%+v", msg.Interface)
	}
}

func TestValidateInterface_MissingIfIndex(t *testing.T) {
	payload := []byte(`{
		"timestamp":"2026-06-01T18:00:00Z",
		"in_octets":10,
		"out_octets":20,
		"in_errors":0,
		"out_errors":0
	}`)
	if _, err := validate.Validate("site/site-001/device/dev-001/metric/interface", payload); err == nil {
		t.Fatal("expected error")
	}
}
