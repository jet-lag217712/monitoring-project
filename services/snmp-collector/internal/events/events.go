package events

import (
	"fmt"
	"time"
)

// Event is a telemetry payload that can be published.
type Event interface {
	Topic() string
}

// DeviceMetricEvent is a device-level metric sample.
// Future MQTT topic: site/{site_id}/device/{device_id}/metric/device
type DeviceMetricEvent struct {
	SiteID    string    `json:"site_id"`
	DeviceID  string    `json:"device_id"`
	Timestamp time.Time `json:"timestamp"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
}

// Topic returns the contract route for this event.
func (e DeviceMetricEvent) Topic() string {
	return fmt.Sprintf("site/%s/device/%s/metric/device", e.SiteID, e.DeviceID)
}

// InterfaceMetricEvent is an interface-level counter sample.
// Future MQTT topic: site/{site_id}/device/{device_id}/metric/interface
type InterfaceMetricEvent struct {
	SiteID    string    `json:"site_id"`
	DeviceID  string    `json:"device_id"`
	Timestamp time.Time `json:"timestamp"`
	IfIndex   int       `json:"if_index"`
	InOctets  uint64    `json:"in_octets"`
	OutOctets uint64    `json:"out_octets"`
	InErrors  uint64    `json:"in_errors"`
	OutErrors uint64    `json:"out_errors"`
}

// Topic returns the contract route for this event.
func (e InterfaceMetricEvent) Topic() string {
	return fmt.Sprintf("site/%s/device/%s/metric/interface", e.SiteID, e.DeviceID)
}
