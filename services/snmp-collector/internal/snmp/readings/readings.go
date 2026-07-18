// Package readings defines the stage-owned internal Phase 2 polling model.
package readings

import (
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/core"
)

// Capability is a compile-time profile capability bit set.
type Capability uint64

const (
	CapabilityCPU Capability = 1 << iota
	CapabilityMemory
	CapabilityTemperature
	CapabilityPower
)

// Has reports whether a capability is present.
func (c Capability) Has(capability Capability) bool {
	return c&capability != 0
}

// Selection is the interface filter outcome.
type Selection string

const (
	Selected        Selection = "selected"
	ExcludedDefault Selection = "excluded_default"
	ExcludedRule    Selection = "excluded_rule"
)

// InterfaceResult combines core IF-MIB data with filter-owned annotations.
type InterfaceResult struct {
	Reading      core.InterfaceReading
	Selection    Selection
	FilterReason string
	RuleID       string
}

// ScalarReading is an optional vendor-neutral percentage reading.
type ScalarReading struct {
	Value     float64
	SourceOID string
}

// ComponentReading is a vendor-neutral sensor or power component reading.
type ComponentReading struct {
	Index     int
	Name      string
	Value     *float64
	Unit      string
	Status    string
	SourceOID string
}

// VendorReadings contains fields owned exclusively by profile enrichment.
type VendorReadings struct {
	Profile      string
	Capabilities Capability
	CPU          *ScalarReading
	Memory       *ScalarReading
	Temperatures []ComponentReading
	Power        []ComponentReading
}

// FilterSummary contains bounded aggregate interface selection counts.
type FilterSummary struct {
	Selected        int
	ExcludedDefault int
	ExcludedRule    int
}

// DevicePollResult is produced whenever core polling succeeds. Core,
// profile, and filter stages only mutate their corresponding sections.
type DevicePollResult struct {
	SiteID     string
	DeviceID   string
	IPAddress  string
	ObservedAt time.Time
	// InventoryRole is operator metadata from managed inventory (not from SNMP).
	InventoryRole string

	Identity   core.DeviceIdentity
	Interfaces []InterfaceResult
	Vendor     VendorReadings
	Filter     FilterSummary
}

// NewDevicePollResult creates the valid result required by the core-success invariant.
func NewDevicePollResult(siteID, deviceID, ipAddress string, observedAt time.Time, identity core.DeviceIdentity, interfaces []core.InterfaceReading) DevicePollResult {
	result := DevicePollResult{
		SiteID:     siteID,
		DeviceID:   deviceID,
		IPAddress:  ipAddress,
		ObservedAt: observedAt.UTC(),
		Identity:   identity,
		Interfaces: make([]InterfaceResult, len(interfaces)),
	}
	for i, iface := range interfaces {
		result.Interfaces[i].Reading = iface
	}
	return result
}
