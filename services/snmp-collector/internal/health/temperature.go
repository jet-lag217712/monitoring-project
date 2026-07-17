package health

import (
	"math"

	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/readings"
)

// PrimaryTemperature returns the maximum valid temperature from vendor components.
// Nil, NaN, and negative values are ignored. ok is false when no valid reading exists.
func PrimaryTemperature(components []readings.ComponentReading) (value float64, ok bool) {
	for _, component := range components {
		if component.Value == nil {
			continue
		}
		candidate := *component.Value
		if math.IsNaN(candidate) || candidate < 0 {
			continue
		}
		if !ok || candidate > value {
			value = candidate
			ok = true
		}
	}
	return value, ok
}

// PrimaryTemperaturePtr is PrimaryTemperature with a pointer result for event payloads.
func PrimaryTemperaturePtr(components []readings.ComponentReading) *float64 {
	value, ok := PrimaryTemperature(components)
	if !ok {
		return nil
	}
	return &value
}
