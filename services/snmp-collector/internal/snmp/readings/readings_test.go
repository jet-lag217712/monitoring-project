package readings

import (
	"testing"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/core"
)

func TestNewDevicePollResultOwnsCoreData(t *testing.T) {
	t.Parallel()

	observed := time.Date(2026, 7, 17, 10, 0, 0, 0, time.FixedZone("test", -7*60*60))
	result := NewDevicePollResult(
		"site-001",
		"device-001",
		"192.0.2.1",
		observed,
		core.DeviceIdentity{SysObjectID: "1.3.6.1.4.1.55555", UptimeSeconds: 42},
		[]core.InterfaceReading{{IfIndex: 1}},
	)
	if !result.ObservedAt.Equal(observed.UTC()) {
		t.Fatalf("observed_at=%v", result.ObservedAt)
	}
	if len(result.Interfaces) != 1 || result.Interfaces[0].Reading.IfIndex != 1 {
		t.Fatalf("interfaces=%#v", result.Interfaces)
	}
	if result.Vendor.Capabilities != 0 || result.Interfaces[0].Selection != "" {
		t.Fatalf("later-stage fields were populated: %#v", result)
	}
}

func TestCapabilityFlags(t *testing.T) {
	t.Parallel()

	capabilities := CapabilityCPU | CapabilityTemperature
	if !capabilities.Has(CapabilityCPU) || !capabilities.Has(CapabilityTemperature) {
		t.Fatalf("missing declared capabilities: %b", capabilities)
	}
	if capabilities.Has(CapabilityMemory) || capabilities.Has(CapabilityPower) {
		t.Fatalf("unexpected capability: %b", capabilities)
	}
}
