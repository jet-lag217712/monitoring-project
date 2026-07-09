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
	IDFCount     int     `json:"idf_count"`
	DeviceCount  int     `json:"device_count"`
	OnlineCount  int     `json:"online_count"`
	AvgCPU       float64 `json:"avg_cpu"`
	AvgMemory    float64 `json:"avg_memory"`
	ActiveAlerts int     `json:"active_alerts"`
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
	TotalDevices int `json:"total_devices"`
	OnlineCount  int `json:"online_count"`
	IDFCount     int `json:"idf_count"`
	ActiveAlerts int `json:"active_alerts"`
}

// SiteDetailLatest wraps the devices map.
type SiteDetailLatest struct {
	Devices map[string]DeviceSummary `json:"devices"`
}

// DeviceSummary is a device row in site detail / device list.
type DeviceSummary struct {
	Hostname   string   `json:"hostname"`
	Role       string   `json:"role"`
	Status     int      `json:"status"`
	CPUPct     float64  `json:"cpu_pct"`
	MemoryPct  float64  `json:"memory_pct"`
	UptimeDays *float64 `json:"uptime_days"`
	LatencyMs  *float64 `json:"latency_ms"`
}

// DeviceDetail is GET /api/devices/{deviceId}.
type DeviceDetail struct {
	ID         uuid.UUID  `json:"id"`
	SiteID     string     `json:"site_id"`
	Hostname   string     `json:"hostname"`
	IPAddress  string     `json:"ip_address"`
	Vendor     string     `json:"vendor"`
	Model      string     `json:"model"`
	Status     int        `json:"status"`
	Role       string     `json:"role"`
	CPUPct     float64    `json:"cpu_pct"`
	MemoryPct  float64    `json:"memory_pct"`
	UptimeDays *float64   `json:"uptime_days"`
	LatencyMs  *float64   `json:"latency_ms"`
	LastSeen   *time.Time `json:"last_seen,omitempty"`
}

// InterfaceInfo is one interface on a device.
type InterfaceInfo struct {
	ID          uuid.UUID `json:"id"`
	IfIndex     int       `json:"if_index"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	AdminStatus string    `json:"admin_status,omitempty"`
	OperStatus  string    `json:"oper_status,omitempty"`
	SpeedBps    *int64    `json:"speed_bps,omitempty"`
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
