package normalize

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/events"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/core"
)

func TestToEventsJSONShape(t *testing.T) {
	t.Parallel()

	ts := time.Date(2026, 7, 8, 12, 0, 0, 0, time.UTC)
	evs := ToEvents(DeviceReading{
		SiteID:        "site-001",
		DeviceID:      "dev-001",
		IPAddress:     "10.255.0.1",
		Timestamp:     ts,
		UptimeSeconds: 86400,
		Interfaces: []core.InterfaceReading{
			{IfIndex: 2, InOctets: 123, OutOctets: 456, InErrors: 0, OutErrors: 0},
		},
	})

	if len(evs) != 2 {
		t.Fatalf("got %d events, want 2", len(evs))
	}

	dev, ok := evs[0].(events.DeviceMetricEvent)
	if !ok {
		t.Fatalf("first event type %T", evs[0])
	}
	if dev.Topic() != "site/site-001/device/dev-001/metric/device" {
		t.Fatalf("device topic: %s", dev.Topic())
	}
	raw, err := json.Marshal(dev)
	if err != nil {
		t.Fatal(err)
	}
	var deviceMap map[string]any
	if err := json.Unmarshal(raw, &deviceMap); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"site_id", "device_id", "ip_address", "timestamp", "metric", "value"} {
		if _, ok := deviceMap[key]; !ok {
			t.Fatalf("missing device field %q in %s", key, raw)
		}
	}
	if deviceMap["metric"] != "uptime_seconds" {
		t.Fatalf("metric=%v", deviceMap["metric"])
	}
	if deviceMap["value"].(float64) != 86400 {
		t.Fatalf("value=%v", deviceMap["value"])
	}

	iface, ok := evs[1].(events.InterfaceMetricEvent)
	if !ok {
		t.Fatalf("second event type %T", evs[1])
	}
	if iface.Topic() != "site/site-001/device/dev-001/metric/interface" {
		t.Fatalf("interface topic: %s", iface.Topic())
	}
	raw, err = json.Marshal(iface)
	if err != nil {
		t.Fatal(err)
	}
	var ifaceMap map[string]any
	if err := json.Unmarshal(raw, &ifaceMap); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"site_id", "device_id", "timestamp", "if_index", "in_octets", "out_octets", "in_errors", "out_errors"} {
		if _, ok := ifaceMap[key]; !ok {
			t.Fatalf("missing interface field %q in %s", key, raw)
		}
	}
	if int(ifaceMap["if_index"].(float64)) != 2 {
		t.Fatalf("if_index=%v", ifaceMap["if_index"])
	}
}
