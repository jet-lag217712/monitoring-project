package normalize

import (
	"github.com/equate/ogsd/services/snmp-collector/internal/events"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/readings"
)

// ToEvents bridges the Phase 2 internal result to the unchanged v1 events.
// Unsupported v1 vendor fields remain internal until the Phase 4 contract.
func ToEvents(r readings.DevicePollResult) []events.Event {
	ts := r.ObservedAt.UTC()
	out := make([]events.Event, 0, 1+len(r.Interfaces))

	out = append(out, events.DeviceMetricEvent{
		SiteID:    r.SiteID,
		DeviceID:  r.DeviceID,
		IPAddress: r.IPAddress,
		Timestamp: ts,
		Metric:    "uptime_seconds",
		Value:     r.Identity.UptimeSeconds,
	})

	for _, iface := range r.Interfaces {
		if iface.Selection != readings.Selected || !iface.Reading.HasCounters {
			continue
		}
		out = append(out, events.InterfaceMetricEvent{
			SiteID:    r.SiteID,
			DeviceID:  r.DeviceID,
			Timestamp: ts,
			IfIndex:   iface.Reading.IfIndex,
			InOctets:  iface.Reading.InOctets,
			OutOctets: iface.Reading.OutOctets,
			InErrors:  iface.Reading.InErrors,
			OutErrors: iface.Reading.OutErrors,
		})
	}
	return out
}
