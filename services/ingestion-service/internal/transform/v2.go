package transform

import (
	"fmt"
	"time"

	"github.com/equate/ogsd/services/ingestion-service/internal/validate"
	"github.com/google/uuid"
)

// DeviceTelemetrySample is a transformed v2 device telemetry event.
type DeviceTelemetrySample struct {
	EventID               uuid.UUID
	SiteUUID              uuid.UUID
	SiteName              string
	DeviceUUID            uuid.UUID
	DeviceID              string
	CollectorID           string
	ConfigRevision        string
	ObservedAt            time.Time
	Hostname              string
	ManagementAddress     string
	Role                  string
	SysObjectID           string
	SysName               string
	SysDescr              string
	Vendor                string
	Model                 string
	Serial                string
	SNMPVersion           string
	ProfileName           string
	Capabilities          []string
	UptimeSeconds         float64
	CPUUtilizationPct     *float64
	MemoryUtilizationPct  *float64
	PrimaryTemperatureC   *float64
	TemperatureComponents []validate.ComponentReading
	PowerComponents       []validate.ComponentReading
}

// InterfaceTelemetrySample is a transformed v2 interface telemetry event.
type InterfaceTelemetrySample struct {
	EventID       uuid.UUID
	SiteUUID      uuid.UUID
	SiteName      string
	DeviceUUID    uuid.UUID
	DeviceID      string
	InterfaceUUID uuid.UUID
	CollectorID   string
	ConfigRevision string
	ObservedAt    time.Time
	IfIndex       int
	Name          string
	Alias         string
	Type          string
	AdminStatus   string
	OperStatus    string
	SpeedBps      *int64
	InOctets      uint64
	OutOctets     uint64
	InErrors      uint64
	OutErrors     uint64
	InDiscards    *uint64
	OutDiscards   *uint64
}

// HealthSample is a transformed v2 health state event.
type HealthSample struct {
	EventID                      uuid.UUID
	SiteUUID                     uuid.UUID
	SiteName                     string
	DeviceUUID                   uuid.UUID
	DeviceID                     string
	CollectorID                  string
	ConfigRevision               string
	ObservedAt                   time.Time
	State                        string
	Reason                       string
	PreviousState                *string
	Transition                   string
	FailureCount                 int
	FailureThreshold             int
	TemperatureC                 *float64
	TemperatureWarningC          *float64
	TemperaturePolicyRevision    *string
	UpstreamDeviceIDs            []string
	UnavailableUpstreamDeviceIDs []string
	RootCauseDeviceIDs           []string
}

// HeartbeatSample is a transformed v2 collector heartbeat event.
type HeartbeatSample struct {
	EventID          uuid.UUID
	SiteUUID         uuid.UUID
	SiteName         string
	CollectorUUID    uuid.UUID
	CollectorID      string
	ConfigRevision   string
	ObservedAt       time.Time
	Hostname         string
	Version          string
	GitCommit        string
	BuildTime        string
	UptimeSeconds    int64
	SQLiteQueueDepth int64
	MemoryUsageBytes int64
	GoroutineCount   int
}

// CollectorUUID returns a deterministic UUID for a site+collector pair.
func CollectorUUID(siteID, collectorID string) uuid.UUID {
	return uuid.NewSHA1(ogsdNamespace, []byte("collector:"+siteID+"/"+collectorID))
}

// ComponentUUID returns a deterministic UUID for a device component.
func ComponentUUID(deviceUUID uuid.UUID, kind, componentID string) uuid.UUID {
	return uuid.NewSHA1(ogsdNamespace, []byte(fmt.Sprintf("%s:%s/%s", kind, deviceUUID.String(), componentID)))
}

// DeviceTelemetryFromValidated maps a validated v2 device event.
func DeviceTelemetryFromValidated(msg validate.DeviceTelemetryV2) DeviceTelemetrySample {
	return DeviceTelemetrySample{
		EventID:               msg.Envelope.EventID,
		SiteUUID:              SiteUUID(msg.Envelope.SiteID),
		SiteName:              msg.Envelope.SiteID,
		DeviceUUID:            DeviceUUID(msg.Envelope.SiteID, msg.Envelope.DeviceID),
		DeviceID:              msg.Envelope.DeviceID,
		CollectorID:           msg.Envelope.CollectorID,
		ConfigRevision:        msg.Envelope.ConfigRevision,
		ObservedAt:            msg.Envelope.ObservedAt,
		Hostname:              msg.Hostname,
		ManagementAddress:     msg.ManagementAddress,
		Role:                  msg.Role,
		SysObjectID:           msg.SysObjectID,
		SysName:               msg.SysName,
		SysDescr:              msg.SysDescr,
		Vendor:                msg.Vendor,
		Model:                 msg.Model,
		Serial:                msg.Serial,
		SNMPVersion:           msg.SNMPVersion,
		ProfileName:           msg.ProfileName,
		Capabilities:          cloneStrings(msg.Capabilities),
		UptimeSeconds:         msg.UptimeSeconds,
		CPUUtilizationPct:     msg.CPUUtilizationPct,
		MemoryUtilizationPct:  msg.MemoryUtilizationPct,
		PrimaryTemperatureC:   msg.PrimaryTemperatureC,
		TemperatureComponents: append([]validate.ComponentReading(nil), msg.TemperatureComponents...),
		PowerComponents:       append([]validate.ComponentReading(nil), msg.PowerComponents...),
	}
}

// InterfaceTelemetryFromValidated maps a validated v2 interface event.
func InterfaceTelemetryFromValidated(msg validate.InterfaceTelemetryV2) InterfaceTelemetrySample {
	devUUID := DeviceUUID(msg.Envelope.SiteID, msg.Envelope.DeviceID)
	return InterfaceTelemetrySample{
		EventID:        msg.Envelope.EventID,
		SiteUUID:       SiteUUID(msg.Envelope.SiteID),
		SiteName:       msg.Envelope.SiteID,
		DeviceUUID:     devUUID,
		DeviceID:       msg.Envelope.DeviceID,
		InterfaceUUID:  InterfaceUUID(devUUID, msg.IfIndex),
		CollectorID:    msg.Envelope.CollectorID,
		ConfigRevision: msg.Envelope.ConfigRevision,
		ObservedAt:     msg.Envelope.ObservedAt,
		IfIndex:        msg.IfIndex,
		Name:           msg.Name,
		Alias:          msg.Alias,
		Type:           msg.Type,
		AdminStatus:    msg.AdminStatus,
		OperStatus:     msg.OperStatus,
		SpeedBps:       msg.SpeedBps,
		InOctets:       msg.InOctets,
		OutOctets:      msg.OutOctets,
		InErrors:       msg.InErrors,
		OutErrors:      msg.OutErrors,
		InDiscards:     msg.InDiscards,
		OutDiscards:    msg.OutDiscards,
	}
}

// HealthFromValidated maps a validated v2 health event.
func HealthFromValidated(msg validate.HealthStateV2) HealthSample {
	return HealthSample{
		EventID:                      msg.Envelope.EventID,
		SiteUUID:                     SiteUUID(msg.Envelope.SiteID),
		SiteName:                     msg.Envelope.SiteID,
		DeviceUUID:                   DeviceUUID(msg.Envelope.SiteID, msg.Envelope.DeviceID),
		DeviceID:                     msg.Envelope.DeviceID,
		CollectorID:                  msg.Envelope.CollectorID,
		ConfigRevision:               msg.Envelope.ConfigRevision,
		ObservedAt:                   msg.Envelope.ObservedAt,
		State:                        msg.State,
		Reason:                       msg.Reason,
		PreviousState:                msg.PreviousState,
		Transition:                   msg.Transition,
		FailureCount:                 msg.FailureCount,
		FailureThreshold:             msg.FailureThreshold,
		TemperatureC:                 msg.TemperatureC,
		TemperatureWarningC:          msg.TemperatureWarningC,
		TemperaturePolicyRevision:    msg.TemperaturePolicyRevision,
		UpstreamDeviceIDs:            cloneStrings(msg.UpstreamDeviceIDs),
		UnavailableUpstreamDeviceIDs: cloneStrings(msg.UnavailableUpstreamDeviceIDs),
		RootCauseDeviceIDs:           cloneStrings(msg.RootCauseDeviceIDs),
	}
}

// HeartbeatFromValidated maps a validated v2 heartbeat event.
func HeartbeatFromValidated(msg validate.HeartbeatV2) HeartbeatSample {
	return HeartbeatSample{
		EventID:          msg.Envelope.EventID,
		SiteUUID:         SiteUUID(msg.Envelope.SiteID),
		SiteName:         msg.Envelope.SiteID,
		CollectorUUID:    CollectorUUID(msg.Envelope.SiteID, msg.Envelope.CollectorID),
		CollectorID:      msg.Envelope.CollectorID,
		ConfigRevision:   msg.Envelope.ConfigRevision,
		ObservedAt:       msg.Envelope.ObservedAt,
		Hostname:         msg.Hostname,
		Version:          msg.Version,
		GitCommit:        msg.GitCommit,
		BuildTime:        msg.BuildTime,
		UptimeSeconds:    msg.UptimeSeconds,
		SQLiteQueueDepth: msg.SQLiteQueueDepth,
		MemoryUsageBytes: msg.MemoryUsageBytes,
		GoroutineCount:   msg.GoroutineCount,
	}
}

// cloneStrings copies in and always returns a non-nil slice.
// append([]string(nil), empty...) yields nil, which pgx encodes as SQL NULL.
func cloneStrings(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}
