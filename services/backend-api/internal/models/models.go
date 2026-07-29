package models

import (
	"time"

	"github.com/google/uuid"
)

// SiteOverview is one entry in GET /api/sites (keyed by site string ID).
type SiteOverview struct {
	Location string             `json:"location"`
	Type     string             `json:"type"`
	Status   string             `json:"status"`
	Latest   SiteOverviewLatest `json:"latest"`
}

// SiteOverviewLatest wraps the summary block expected by normalizeSites().
type SiteOverviewLatest struct {
	Summary SiteSummary `json:"summary"`
}

// SiteSummary holds aggregate counts for a site card / detail header.
type SiteSummary struct {
	IDFCount                int     `json:"idf_count"`
	DeviceCount             int     `json:"device_count"`
	OnlineCount             int     `json:"online_count"`
	AvgCPU                  float64 `json:"avg_cpu"`
	AvgMemory               float64 `json:"avg_memory"`
	ActiveAlerts            int     `json:"active_alerts"`
	HealthyCount            int     `json:"healthy_count"`
	WarningCount            int     `json:"warning_count"`
	CriticalCount           int     `json:"critical_count"`
	UnknownCount            int     `json:"unknown_count"`
	DependencyImpactedCount int     `json:"dependency_impacted_count"`
}

// SiteDetail is the GET /api/sites/{siteId} response.
type SiteDetail struct {
	SiteID   string            `json:"site_id"`
	Location string            `json:"location"`
	Summary  SiteDetailSummary `json:"summary"`
	Latest   SiteDetailLatest  `json:"latest"`
}

// SiteDetailSummary is the detail-view summary (field names match mockData).
type SiteDetailSummary struct {
	TotalDevices            int `json:"total_devices"`
	OnlineCount             int `json:"online_count"`
	IDFCount                int `json:"idf_count"`
	ActiveAlerts            int `json:"active_alerts"`
	HealthyCount            int `json:"healthy_count"`
	WarningCount            int `json:"warning_count"`
	CriticalCount           int `json:"critical_count"`
	UnknownCount            int `json:"unknown_count"`
	DependencyImpactedCount int `json:"dependency_impacted_count"`
}

// SiteDetailLatest wraps the devices map.
type SiteDetailLatest struct {
	Devices map[string]DeviceSummary `json:"devices"`
}

// DeviceSummary is a device row in site detail / device list.
type DeviceSummary struct {
	DeviceID               string   `json:"device_id,omitempty"`
	Hostname               string   `json:"hostname"`
	IPAddress              string   `json:"ip_address,omitempty"`
	Role                   string   `json:"role"`
	Status                 int      `json:"status"`
	StatusReason           string   `json:"status_reason,omitempty"`
	FailureCount           *int     `json:"failure_count,omitempty"`
	UpstreamDeviceIDs      []string `json:"upstream_device_ids,omitempty"`
	UnavailableUpstreamIDs []string `json:"unavailable_upstream_device_ids,omitempty"`
	RootCauseDeviceIDs     []string `json:"root_cause_device_ids,omitempty"`
	CPUPct                 *float64 `json:"cpu_pct"`
	MemoryPct              *float64 `json:"memory_pct"`
	UptimeDays             *float64 `json:"uptime_days"`
	LatencyMs              *float64 `json:"latency_ms"`
}

// ComponentReading is a temperature or power component projection.
type ComponentReading struct {
	ComponentID string   `json:"component_id"`
	Name        string   `json:"name"`
	Index       string   `json:"index,omitempty"`
	Status      string   `json:"status"`
	Value       *float64 `json:"value"`
	Unit        string   `json:"unit"`
}

// SNMPIdentity holds persisted SNMP sys* fields.
type SNMPIdentity struct {
	SysName     string `json:"sysName,omitempty"`
	SysObjectID string `json:"sysObjectID,omitempty"`
	SysDescr    string `json:"sysDescr,omitempty"`
	SysUpTime   *int64 `json:"sysUpTime,omitempty"`
}

// DeviceHistory holds embedded metric series for device detail.
type DeviceHistory struct {
	CPU         []MetricPoint `json:"cpu"`
	Memory      []MetricPoint `json:"memory"`
	Temperature []MetricPoint `json:"temperature"`
	Uptime      []MetricPoint `json:"uptime"`
}

// DeviceDetail is GET /api/devices/{deviceId}.
type DeviceDetail struct {
	ID                     uuid.UUID          `json:"id"`
	SiteID                 string             `json:"site_id"`
	Hostname               string             `json:"hostname"`
	IPAddress              string             `json:"ip_address"`
	Vendor                 string             `json:"vendor"`
	Model                  string             `json:"model"`
	SerialNumber           string             `json:"serial_number,omitempty"`
	Profile                string             `json:"profile,omitempty"`
	Capabilities           []string           `json:"capabilities,omitempty"`
	Status                 int                `json:"status"`
	StatusReason           string             `json:"status_reason,omitempty"`
	FailureCount           *int               `json:"failure_count,omitempty"`
	UpstreamDeviceIDs      []string           `json:"upstream_device_ids,omitempty"`
	UnavailableUpstreamIDs []string           `json:"unavailable_upstream_device_ids,omitempty"`
	RootCauseDeviceIDs     []string           `json:"root_cause_device_ids,omitempty"`
	Role                   string             `json:"role"`
	CPUPct                 *float64           `json:"cpu_pct"`
	MemoryPct              *float64           `json:"memory_pct"`
	TemperatureC           *float64           `json:"temperature_c"`
	UptimeDays             *float64           `json:"uptime_days"`
	LatencyMs              *float64           `json:"latency_ms"`
	LastSeen               *time.Time         `json:"last_seen,omitempty"`
	SNMP                   *SNMPIdentity      `json:"snmp,omitempty"`
	TemperatureComponents  []ComponentReading `json:"temperature_components,omitempty"`
	PowerComponents        []ComponentReading `json:"power_components,omitempty"`
	History                *DeviceHistory     `json:"history,omitempty"`
}

// InterfaceInfo is one interface on a device.
type InterfaceInfo struct {
	ID             uuid.UUID     `json:"id"`
	IfIndex        int           `json:"if_index"`
	Name           string        `json:"name"`
	Description    string        `json:"description,omitempty"`
	IfAlias        string        `json:"if_alias,omitempty"`
	IfType         string        `json:"if_type,omitempty"`
	AdminStatus    string        `json:"admin_status,omitempty"`
	OperStatus     string        `json:"oper_status,omitempty"`
	SpeedBps       *int64        `json:"speed_bps,omitempty"`
	InOctets       *int64        `json:"in_octets,omitempty"`
	OutOctets      *int64        `json:"out_octets,omitempty"`
	InErrors       *int64        `json:"in_errors,omitempty"`
	OutErrors      *int64        `json:"out_errors,omitempty"`
	InDiscards     *int64        `json:"in_discards,omitempty"`
	OutDiscards    *int64        `json:"out_discards,omitempty"`
	TrafficHistory []TrafficHistoryPoint `json:"traffic_history,omitempty"`
}

// TrafficHistoryPoint is one interface traffic counter sample.
type TrafficHistoryPoint struct {
	TS        time.Time `json:"ts"`
	InOctets  float64   `json:"in_octets"`
	OutOctets float64   `json:"out_octets"`
}

// MetricPoint is one time-series sample.
type MetricPoint struct {
	TS    time.Time `json:"ts"`
	Value float64   `json:"value"`
}

// MetricsResponse is GET /api/devices/{deviceId}/metrics.
type MetricsResponse struct {
	DeviceID string        `json:"device_id"`
	Metric   string        `json:"metric"`
	Start    *time.Time    `json:"start,omitempty"`
	End      *time.Time    `json:"end,omitempty"`
	Points   []MetricPoint `json:"points"`
}

// AlertInfo is one active alert.
type AlertInfo struct {
	ID           uuid.UUID  `json:"id"`
	DeviceID     *uuid.UUID `json:"device_id,omitempty"`
	InterfaceID  *uuid.UUID `json:"interface_id,omitempty"`
	Severity     string     `json:"severity"`
	AlertType    string     `json:"alert_type"`
	Message      string     `json:"message"`
	Acknowledged bool       `json:"acknowledged"`
	CreatedAt    time.Time  `json:"created_at"`
}

// TestConfig is GET /api/test-config.
type TestConfig struct {
	Mode           string `json:"mode"`
	PollingEnabled bool   `json:"polling_enabled"`
}

// APIError is the error envelope body.
type APIError struct {
	Error ErrorBody `json:"error"`
}

// ErrorBody holds code + message.
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}
