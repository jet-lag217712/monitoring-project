package events

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// V2 event type constants.
const (
	EventTypeDeviceTelemetry    = "device_telemetry"
	EventTypeInterfaceTelemetry = "interface_telemetry"
	EventTypeHealthState        = "health_state"
	EventTypeCollectorHeartbeat = "collector_heartbeat"
	SchemaVersionV2             = "2.0"
)

// EnvelopeV2 is the shared MQTT envelope for all v2 events.
type EnvelopeV2 struct {
	SchemaVersion  string    `json:"schema_version"`
	EventID        uuid.UUID `json:"event_id"`
	EventType      string    `json:"event_type"`
	SiteID         string    `json:"site_id"`
	CollectorID    string    `json:"collector_id"`
	DeviceID       string    `json:"device_id,omitempty"`
	ObservedAt     time.Time `json:"observed_at"`
	EmittedAt      time.Time `json:"emitted_at"`
	ConfigRevision string    `json:"config_revision"`
}

// DeviceTelemetryEvent is a v2 device telemetry message.
type DeviceTelemetryEvent struct {
	EnvelopeV2
	Payload DeviceTelemetryPayload `json:"payload"`
}

// DeviceTelemetryPayload is the device event body.
type DeviceTelemetryPayload struct {
	Identity              DeviceIdentityPayload `json:"identity"`
	Profile               ProfilePayload        `json:"profile"`
	Readings              DeviceReadingsPayload `json:"readings"`
	TemperatureComponents []ComponentPayload    `json:"temperature_components"`
	PowerComponents       []ComponentPayload    `json:"power_components"`
}

// DeviceIdentityPayload holds SNMP identity fields.
type DeviceIdentityPayload struct {
	Hostname    string `json:"hostname"`
	SysObjectID string `json:"sys_object_id"`
	SysName     string `json:"sys_name"`
	SysDescr    string `json:"sys_descr"`
	Vendor      string `json:"vendor,omitempty"`
	Model       string `json:"model,omitempty"`
	Serial      string `json:"serial,omitempty"`
	SNMPVersion string `json:"snmp_version"`
}

// ProfilePayload holds detected profile metadata.
type ProfilePayload struct {
	Name         string   `json:"name"`
	Capabilities []string `json:"capabilities"`
}

// DeviceReadingsPayload holds scalar readings.
type DeviceReadingsPayload struct {
	UptimeSeconds        float64  `json:"uptime_seconds"`
	CPUUtilizationPct    *float64 `json:"cpu_utilization_pct,omitempty"`
	MemoryUtilizationPct *float64 `json:"memory_utilization_pct,omitempty"`
	PrimaryTemperatureC  *float64 `json:"primary_temperature_c,omitempty"`
}

// ComponentPayload is a temperature or power component reading.
type ComponentPayload struct {
	ComponentID string    `json:"component_id"`
	Name        string    `json:"name"`
	Index       string    `json:"index"`
	Value       *float64  `json:"value"`
	Unit        string    `json:"unit"`
	Status      string    `json:"status"`
	ObservedAt  time.Time `json:"observed_at"`
}

// Topic returns the v2 device telemetry route.
func (e DeviceTelemetryEvent) Topic() string {
	return fmt.Sprintf("site/%s/device/%s/telemetry/v2/device", e.SiteID, e.DeviceID)
}

// InterfaceTelemetryEvent is a v2 interface telemetry message.
type InterfaceTelemetryEvent struct {
	EnvelopeV2
	Payload InterfaceTelemetryPayload `json:"payload"`
}

// InterfaceTelemetryPayload is the interface event body.
type InterfaceTelemetryPayload struct {
	Interface InterfaceIdentityPayload `json:"interface"`
	Counters  InterfaceCountersPayload `json:"counters"`
}

// InterfaceIdentityPayload holds IF-MIB identity/status.
type InterfaceIdentityPayload struct {
	IfIndex     int    `json:"if_index"`
	Name        string `json:"name"`
	Alias       string `json:"alias,omitempty"`
	Type        string `json:"type,omitempty"`
	AdminStatus string `json:"admin_status"`
	OperStatus  string `json:"oper_status"`
	SpeedBps    *int64 `json:"speed_bps"`
}

// InterfaceCountersPayload holds interface counters.
type InterfaceCountersPayload struct {
	InOctets    uint64  `json:"in_octets"`
	OutOctets   uint64  `json:"out_octets"`
	InErrors    uint64  `json:"in_errors"`
	OutErrors   uint64  `json:"out_errors"`
	InDiscards  *uint64 `json:"in_discards,omitempty"`
	OutDiscards *uint64 `json:"out_discards,omitempty"`
}

// Topic returns the v2 interface telemetry route.
func (e InterfaceTelemetryEvent) Topic() string {
	return fmt.Sprintf("site/%s/device/%s/telemetry/v2/interface", e.SiteID, e.DeviceID)
}

// HealthStateEvent is a v2 health state message.
type HealthStateEvent struct {
	EnvelopeV2
	Payload HealthStatePayload `json:"payload"`
}

// HealthStatePayload is the health event body.
type HealthStatePayload struct {
	State                        string   `json:"state"`
	Reason                       string   `json:"reason"`
	PreviousState                *string  `json:"previous_state,omitempty"`
	Transition                   string   `json:"transition"`
	FailureCount                 int      `json:"failure_count"`
	FailureThreshold             int      `json:"failure_threshold"`
	TemperatureC                 *float64 `json:"temperature_c,omitempty"`
	TemperatureWarningC          *float64 `json:"temperature_warning_c,omitempty"`
	TemperaturePolicyRevision    *string  `json:"temperature_policy_revision,omitempty"`
	UpstreamDeviceIDs            []string `json:"upstream_device_ids"`
	UnavailableUpstreamDeviceIDs []string `json:"unavailable_upstream_device_ids"`
	RootCauseDeviceIDs           []string `json:"root_cause_device_ids"`
}

// Topic returns the v2 health route.
func (e HealthStateEvent) Topic() string {
	return fmt.Sprintf("site/%s/device/%s/telemetry/v2/health", e.SiteID, e.DeviceID)
}

// HeartbeatEvent is a v2 collector heartbeat message.
type HeartbeatEvent struct {
	EnvelopeV2
	Payload HeartbeatPayload `json:"payload"`
}

// HeartbeatPayload is the heartbeat event body.
type HeartbeatPayload struct {
	Hostname         string `json:"hostname"`
	Version          string `json:"version"`
	GitCommit        string `json:"git_commit"`
	BuildTime        string `json:"build_time"`
	UptimeSeconds    int64  `json:"uptime_seconds"`
	SQLiteQueueDepth int64  `json:"sqlite_queue_depth"`
	MemoryUsageBytes int64  `json:"memory_usage_bytes"`
	GoroutineCount   int    `json:"goroutine_count"`
}

// Topic returns the v2 heartbeat route.
func (e HeartbeatEvent) Topic() string {
	return fmt.Sprintf("site/%s/collector/%s/telemetry/v2/heartbeat", e.SiteID, e.CollectorID)
}
