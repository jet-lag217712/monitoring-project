package telemetry

import (
	"fmt"
	"strconv"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/events"
	"github.com/equate/ogsd/services/snmp-collector/internal/health"
	"github.com/equate/ogsd/services/snmp-collector/internal/normalize"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/readings"
	"github.com/google/uuid"
)

// PublishMode selects which telemetry families to emit.
type PublishMode string

const (
	ModeV1   PublishMode = "v1"
	ModeV2   PublishMode = "v2"
	ModeBoth PublishMode = "both"
)

// ParsePublishMode validates publisher.telemetry_version.
func ParsePublishMode(raw string) (PublishMode, error) {
	switch PublishMode(raw) {
	case ModeV1, ModeV2, ModeBoth:
		return PublishMode(raw), nil
	default:
		return "", fmt.Errorf("publisher.telemetry_version must be \"v1\", \"v2\", or \"both\"")
	}
}

// Context carries envelope identity shared by all v2 events for a poll/publish.
type Context struct {
	SiteID         string
	CollectorID    string
	ConfigRevision string
	EmittedAt      time.Time
}

// DeviceEvents builds device/interface events according to the publish mode.
func DeviceEvents(mode PublishMode, ctx Context, result readings.DevicePollResult) []events.Event {
	var out []events.Event
	if mode == ModeV1 || mode == ModeBoth {
		out = append(out, normalize.ToEvents(result)...)
	}
	if mode == ModeV2 || mode == ModeBoth {
		out = append(out, DeviceTelemetry(ctx, result))
		out = append(out, InterfaceTelemetry(ctx, result)...)
	}
	return out
}

// DeviceTelemetry maps a poll result to a v2 device event.
func DeviceTelemetry(ctx Context, result readings.DevicePollResult) events.DeviceTelemetryEvent {
	profileName := result.Vendor.Profile
	if profileName == "" {
		profileName = "core"
	}
	caps := capabilities(result.Vendor.Capabilities)
	hostname := result.Identity.SysName
	if hostname == "" {
		hostname = result.DeviceID
	}
	observedAt := result.ObservedAt.UTC()
	emittedAt := ctx.EmittedAt.UTC()
	if emittedAt.IsZero() {
		emittedAt = time.Now().UTC()
	}

	payload := events.DeviceTelemetryPayload{
		Identity: events.DeviceIdentityPayload{
			Hostname:    hostname,
			SysObjectID: result.Identity.SysObjectID,
			SysName:     result.Identity.SysName,
			SysDescr:    result.Identity.SysDescr,
			SNMPVersion: "2c",
		},
		Profile: events.ProfilePayload{
			Name:         profileName,
			Capabilities: caps,
		},
		Readings: events.DeviceReadingsPayload{
			UptimeSeconds: result.Identity.UptimeSeconds,
		},
		TemperatureComponents: temperatureComponents(result.Vendor.Temperatures, observedAt),
		PowerComponents:       powerComponents(result.Vendor.Power, observedAt),
	}
	if result.Vendor.CPU != nil {
		v := result.Vendor.CPU.Value
		payload.Readings.CPUUtilizationPct = &v
	}
	if result.Vendor.Memory != nil {
		v := result.Vendor.Memory.Value
		payload.Readings.MemoryUtilizationPct = &v
	}
	if primary := health.PrimaryTemperaturePtr(result.Vendor.Temperatures); primary != nil {
		payload.Readings.PrimaryTemperatureC = primary
	}

	return events.DeviceTelemetryEvent{
		EnvelopeV2: events.EnvelopeV2{
			SchemaVersion:  events.SchemaVersionV2,
			EventID:        uuid.New(),
			EventType:      events.EventTypeDeviceTelemetry,
			SiteID:         ctx.SiteID,
			CollectorID:    ctx.CollectorID,
			DeviceID:       result.DeviceID,
			ObservedAt:     observedAt,
			EmittedAt:      emittedAt,
			ConfigRevision: ctx.ConfigRevision,
		},
		Payload: payload,
	}
}

// InterfaceTelemetry maps selected interfaces to v2 interface events.
func InterfaceTelemetry(ctx Context, result readings.DevicePollResult) []events.Event {
	observedAt := result.ObservedAt.UTC()
	emittedAt := ctx.EmittedAt.UTC()
	if emittedAt.IsZero() {
		emittedAt = time.Now().UTC()
	}
	out := make([]events.Event, 0)
	for _, iface := range result.Interfaces {
		if iface.Selection != readings.Selected || !iface.Reading.HasCounters {
			continue
		}
		name := iface.Reading.IfName
		if name == "" {
			name = iface.Reading.IfDescr
		}
		if name == "" {
			name = fmt.Sprintf("ifIndex-%d", iface.Reading.IfIndex)
		}
		var speed *int64
		if iface.Reading.SpeedBPS > 0 {
			v := int64(iface.Reading.SpeedBPS)
			speed = &v
		}
		out = append(out, events.InterfaceTelemetryEvent{
			EnvelopeV2: events.EnvelopeV2{
				SchemaVersion:  events.SchemaVersionV2,
				EventID:        uuid.New(),
				EventType:      events.EventTypeInterfaceTelemetry,
				SiteID:         ctx.SiteID,
				CollectorID:    ctx.CollectorID,
				DeviceID:       result.DeviceID,
				ObservedAt:     observedAt,
				EmittedAt:      emittedAt,
				ConfigRevision: ctx.ConfigRevision,
			},
			Payload: events.InterfaceTelemetryPayload{
				Interface: events.InterfaceIdentityPayload{
					IfIndex:     iface.Reading.IfIndex,
					Name:        name,
					Alias:       iface.Reading.IfAlias,
					Type:        iface.Reading.IfTypeName,
					AdminStatus: iface.Reading.AdminStatus,
					OperStatus:  iface.Reading.OperStatus,
					SpeedBps:    speed,
				},
				Counters: events.InterfaceCountersPayload{
					InOctets:  iface.Reading.InOctets,
					OutOctets: iface.Reading.OutOctets,
					InErrors:  iface.Reading.InErrors,
					OutErrors: iface.Reading.OutErrors,
				},
			},
		})
	}
	return out
}

// HealthEvent maps a local health.Event to a v2 health MQTT event.
func HealthEvent(ctx Context, siteID string, ev health.Event) events.HealthStateEvent {
	emittedAt := ctx.EmittedAt.UTC()
	if emittedAt.IsZero() {
		emittedAt = time.Now().UTC()
	}
	var previous *string
	if ev.PreviousState != nil {
		s := string(*ev.PreviousState)
		previous = &s
	}
	var policy *string
	if ev.TemperaturePolicyRevision != "" {
		policy = &ev.TemperaturePolicyRevision
	}
	upstream := append([]string(nil), ev.UpstreamDeviceIDs...)
	if upstream == nil {
		upstream = []string{}
	}
	unavailable := append([]string(nil), ev.UnavailableUpstreamDeviceIDs...)
	if unavailable == nil {
		unavailable = []string{}
	}
	rootCause := append([]string(nil), ev.RootCauseDeviceIDs...)
	if rootCause == nil {
		rootCause = []string{}
	}
	return events.HealthStateEvent{
		EnvelopeV2: events.EnvelopeV2{
			SchemaVersion:  events.SchemaVersionV2,
			EventID:        uuid.New(),
			EventType:      events.EventTypeHealthState,
			SiteID:         siteID,
			CollectorID:    ctx.CollectorID,
			DeviceID:       ev.DeviceID,
			ObservedAt:     ev.ObservedAt.UTC(),
			EmittedAt:      emittedAt,
			ConfigRevision: ctx.ConfigRevision,
		},
		Payload: events.HealthStatePayload{
			State:                        string(ev.State),
			Reason:                       string(ev.Reason),
			PreviousState:                previous,
			Transition:                   string(ev.Transition),
			FailureCount:                 ev.FailureCount,
			FailureThreshold:             ev.FailureThreshold,
			TemperatureC:                 ev.TemperatureC,
			TemperatureWarningC:          ev.TemperatureWarningC,
			TemperaturePolicyRevision:    policy,
			UpstreamDeviceIDs:            upstream,
			UnavailableUpstreamDeviceIDs: unavailable,
			RootCauseDeviceIDs:           rootCause,
		},
	}
}

// HealthEvents maps a batch of local health events.
func HealthEvents(mode PublishMode, ctx Context, siteID string, evs []health.Event) []events.Event {
	if mode != ModeV2 && mode != ModeBoth {
		return nil
	}
	out := make([]events.Event, 0, len(evs))
	for _, ev := range evs {
		out = append(out, HealthEvent(ctx, siteID, ev))
	}
	return out
}

// HeartbeatInput carries runtime fields for a heartbeat event.
type HeartbeatInput struct {
	Hostname         string
	Version          string
	GitCommit        string
	BuildTime        string
	UptimeSeconds    int64
	SQLiteQueueDepth int64
	MemoryUsageBytes int64
	GoroutineCount   int
	ObservedAt       time.Time
}

// Heartbeat builds a v2 collector heartbeat event.
func Heartbeat(ctx Context, in HeartbeatInput) events.HeartbeatEvent {
	emittedAt := ctx.EmittedAt.UTC()
	if emittedAt.IsZero() {
		emittedAt = time.Now().UTC()
	}
	observedAt := in.ObservedAt.UTC()
	if observedAt.IsZero() {
		observedAt = emittedAt
	}
	version := in.Version
	if version == "" {
		version = "unknown"
	}
	gitCommit := in.GitCommit
	if gitCommit == "" {
		gitCommit = "unknown"
	}
	buildTime := in.BuildTime
	if buildTime == "" {
		buildTime = "unknown"
	}
	return events.HeartbeatEvent{
		EnvelopeV2: events.EnvelopeV2{
			SchemaVersion:  events.SchemaVersionV2,
			EventID:        uuid.New(),
			EventType:      events.EventTypeCollectorHeartbeat,
			SiteID:         ctx.SiteID,
			CollectorID:    ctx.CollectorID,
			ObservedAt:     observedAt,
			EmittedAt:      emittedAt,
			ConfigRevision: ctx.ConfigRevision,
		},
		Payload: events.HeartbeatPayload{
			Hostname:         in.Hostname,
			Version:          version,
			GitCommit:        gitCommit,
			BuildTime:        buildTime,
			UptimeSeconds:    in.UptimeSeconds,
			SQLiteQueueDepth: in.SQLiteQueueDepth,
			MemoryUsageBytes: in.MemoryUsageBytes,
			GoroutineCount:   in.GoroutineCount,
		},
	}
}

// ShouldPublishHeartbeat reports whether heartbeats are enabled for the mode.
func ShouldPublishHeartbeat(mode PublishMode) bool {
	return mode == ModeV2 || mode == ModeBoth
}

// ModeFromConfig reads publisher.telemetry_version from config.
func ModeFromConfig(cfg *config.Config) PublishMode {
	if cfg == nil {
		return ModeBoth
	}
	mode, err := ParsePublishMode(cfg.Publisher.TelemetryVersion)
	if err != nil {
		return ModeBoth
	}
	return mode
}

func capabilities(c readings.Capability) []string {
	out := make([]string, 0, 4)
	if c.Has(readings.CapabilityCPU) {
		out = append(out, "cpu")
	}
	if c.Has(readings.CapabilityMemory) {
		out = append(out, "memory")
	}
	if c.Has(readings.CapabilityTemperature) {
		out = append(out, "temperature")
	}
	if c.Has(readings.CapabilityPower) {
		out = append(out, "power")
	}
	return out
}

func temperatureComponents(comps []readings.ComponentReading, observedAt time.Time) []events.ComponentPayload {
	out := make([]events.ComponentPayload, 0, len(comps))
	for _, c := range comps {
		if c.Value == nil {
			continue // schema requires a numeric temperature value
		}
		status := c.Status
		if status == "" {
			status = "unknown"
		}
		unit := c.Unit
		if unit == "" {
			unit = "celsius"
		}
		out = append(out, events.ComponentPayload{
			ComponentID: "temperature-" + strconv.Itoa(c.Index),
			Name:        c.Name,
			Index:       strconv.Itoa(c.Index),
			Value:       c.Value,
			Unit:        unit,
			Status:      status,
			ObservedAt:  observedAt,
		})
	}
	return out
}

func powerComponents(comps []readings.ComponentReading, observedAt time.Time) []events.ComponentPayload {
	out := make([]events.ComponentPayload, 0, len(comps))
	for _, c := range comps {
		status := c.Status
		if status == "" {
			status = "unknown"
		}
		unit := normalizePowerUnit(c.Unit)
		out = append(out, events.ComponentPayload{
			ComponentID: "power-" + strconv.Itoa(c.Index),
			Name:        c.Name,
			Index:       strconv.Itoa(c.Index),
			Value:       c.Value,
			Unit:        unit,
			Status:      status,
			ObservedAt:  observedAt,
		})
	}
	return out
}

// normalizePowerUnit maps vendor-internal sensor units to v2 contract values.
func normalizePowerUnit(unit string) string {
	switch unit {
	case "":
		return "state"
	case "volts_ac", "volts_dc":
		return "volts"
	case "amperes":
		return "amps"
	default:
		return unit
	}
}
