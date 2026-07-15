package normalize

import (
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/events"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/core"
)

// DeviceReading is the device-level input to normalization.
type DeviceReading struct {
	SiteID        string
	DeviceID      string
	IPAddress     string
	Timestamp     time.Time
	UptimeSeconds float64
	Interfaces    []core.InterfaceReading
}

// ToEvents maps SNMP readings into contract-aligned telemetry events.
func ToEvents(r DeviceReading) []events.Event {
	ts := r.Timestamp.UTC()
	out := make([]events.Event, 0, 1+len(r.Interfaces))

	out = append(out, events.DeviceMetricEvent{
		SiteID:    r.SiteID,
		DeviceID:  r.DeviceID,
		IPAddress: r.IPAddress,
		Timestamp: ts,
		Metric:    "uptime_seconds",
		Value:     r.UptimeSeconds,
	})

	for _, iface := range r.Interfaces {
		out = append(out, events.InterfaceMetricEvent{
			SiteID:    r.SiteID,
			DeviceID:  r.DeviceID,
			Timestamp: ts,
			IfIndex:   iface.IfIndex,
			InOctets:  iface.InOctets,
			OutOctets: iface.OutOctets,
			InErrors:  iface.InErrors,
			OutErrors: iface.OutErrors,
		})
	}
	return out
}
