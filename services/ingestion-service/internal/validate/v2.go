package validate

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
)

const schemaVersionV2 = "2.0"

// V2 event kinds.
const (
	KindDeviceV2    Kind = "device_v2"
	KindInterfaceV2 Kind = "interface_v2"
	KindHealthV2    Kind = "health_v2"
	KindHeartbeatV2 Kind = "heartbeat_v2"
)

var identifierPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// EnvelopeV2 is the shared v2 event envelope after validation.
type EnvelopeV2 struct {
	SchemaVersion  string
	EventID        uuid.UUID
	EventType      string
	SiteID         string
	CollectorID    string
	DeviceID       string // empty for heartbeat
	ObservedAt     time.Time
	EmittedAt      time.Time
	ConfigRevision string
}

// ComponentReading is a validated temperature or power component.
type ComponentReading struct {
	ComponentID string
	Name        string
	Index       string
	Value       *float64
	Unit        string
	Status      string
	ObservedAt  time.Time
}

// DeviceTelemetryV2 is a validated v2 device telemetry event.
type DeviceTelemetryV2 struct {
	Envelope              EnvelopeV2
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
	TemperatureComponents []ComponentReading
	PowerComponents       []ComponentReading
}

// InterfaceTelemetryV2 is a validated v2 interface telemetry event.
type InterfaceTelemetryV2 struct {
	Envelope    EnvelopeV2
	IfIndex     int
	Name        string
	Alias       string
	Type        string
	AdminStatus string
	OperStatus  string
	SpeedBps    *int64
	InOctets    uint64
	OutOctets   uint64
	InErrors    uint64
	OutErrors   uint64
	InDiscards  *uint64
	OutDiscards *uint64
}

// HealthStateV2 is a validated v2 health state event.
type HealthStateV2 struct {
	Envelope                     EnvelopeV2
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
	AlertsEnabled                bool
}

// HeartbeatV2 is a validated v2 collector heartbeat event.
type HeartbeatV2 struct {
	Envelope         EnvelopeV2
	Hostname         string
	Version          string
	GitCommit        string
	BuildTime        string
	UptimeSeconds    int64
	SQLiteQueueDepth int64
	MemoryUsageBytes int64
	GoroutineCount   int
}

type envelopeRaw struct {
	SchemaVersion  string          `json:"schema_version"`
	EventID        string          `json:"event_id"`
	EventType      string          `json:"event_type"`
	SiteID         string          `json:"site_id"`
	CollectorID    string          `json:"collector_id"`
	DeviceID       string          `json:"device_id"`
	ObservedAt     string          `json:"observed_at"`
	EmittedAt      string          `json:"emitted_at"`
	ConfigRevision string          `json:"config_revision"`
	Payload        json.RawMessage `json:"payload"`
}

type devicePayloadV2 struct {
	Identity struct {
		Hostname          string `json:"hostname"`
		ManagementAddress string `json:"management_address"`
		Role              string `json:"role"`
		SysObjectID       string `json:"sys_object_id"`
		SysName           string `json:"sys_name"`
		SysDescr          string `json:"sys_descr"`
		Vendor            string `json:"vendor"`
		Model             string `json:"model"`
		Serial            string `json:"serial"`
		SNMPVersion       string `json:"snmp_version"`
	} `json:"identity"`
	Profile struct {
		Name         string   `json:"name"`
		Capabilities []string `json:"capabilities"`
	} `json:"profile"`
	Readings struct {
		UptimeSeconds        *float64 `json:"uptime_seconds"`
		CPUUtilizationPct    *float64 `json:"cpu_utilization_pct"`
		MemoryUtilizationPct *float64 `json:"memory_utilization_pct"`
		PrimaryTemperatureC  *float64 `json:"primary_temperature_c"`
	} `json:"readings"`
	TemperatureComponents []componentRaw `json:"temperature_components"`
	PowerComponents       []componentRaw `json:"power_components"`
}

type componentRaw struct {
	ComponentID string   `json:"component_id"`
	Name        string   `json:"name"`
	Index       string   `json:"index"`
	Value       *float64 `json:"value"`
	Unit        string   `json:"unit"`
	Status      string   `json:"status"`
	ObservedAt  string   `json:"observed_at"`
}

type interfacePayloadV2 struct {
	Interface struct {
		IfIndex     *int    `json:"if_index"`
		Name        string  `json:"name"`
		Alias       string  `json:"alias"`
		Type        string  `json:"type"`
		AdminStatus string  `json:"admin_status"`
		OperStatus  string  `json:"oper_status"`
		SpeedBps    *int64  `json:"speed_bps"`
	} `json:"interface"`
	Counters struct {
		InOctets    *uint64 `json:"in_octets"`
		OutOctets   *uint64 `json:"out_octets"`
		InErrors    *uint64 `json:"in_errors"`
		OutErrors   *uint64 `json:"out_errors"`
		InDiscards  *uint64 `json:"in_discards"`
		OutDiscards *uint64 `json:"out_discards"`
	} `json:"counters"`
}

type healthPayloadV2 struct {
	State                        string   `json:"state"`
	Reason                       string   `json:"reason"`
	PreviousState                *string  `json:"previous_state"`
	Transition                   string   `json:"transition"`
	FailureCount                 *int     `json:"failure_count"`
	FailureThreshold             *int     `json:"failure_threshold"`
	TemperatureC                 *float64 `json:"temperature_c"`
	TemperatureWarningC          *float64 `json:"temperature_warning_c"`
	TemperaturePolicyRevision    *string  `json:"temperature_policy_revision"`
	UpstreamDeviceIDs            []string `json:"upstream_device_ids"`
	UnavailableUpstreamDeviceIDs []string `json:"unavailable_upstream_device_ids"`
	RootCauseDeviceIDs           []string `json:"root_cause_device_ids"`
	AlertsEnabled                *bool    `json:"alerts_enabled"`
}

type heartbeatPayloadV2 struct {
	Hostname         string `json:"hostname"`
	Version          string `json:"version"`
	GitCommit        string `json:"git_commit"`
	BuildTime        string `json:"build_time"`
	UptimeSeconds    *int64 `json:"uptime_seconds"`
	SQLiteQueueDepth *int64 `json:"sqlite_queue_depth"`
	MemoryUsageBytes *int64 `json:"memory_usage_bytes"`
	GoroutineCount   *int   `json:"goroutine_count"`
}

// TopicPartsV2 holds identifiers parsed from a v2 MQTT topic.
type TopicPartsV2 struct {
	SiteID      string
	DeviceID    string
	CollectorID string
	Kind        Kind
}

// ParseTopicV2 extracts identifiers from v2 telemetry routes.
func ParseTopicV2(topic string) (TopicPartsV2, error) {
	parts := strings.Split(topic, "/")
	// site/{site}/device/{device}/telemetry/v2/{device|interface|health}
	if len(parts) == 7 && parts[0] == "site" && parts[2] == "device" && parts[4] == "telemetry" && parts[5] == "v2" {
		siteID := strings.TrimSpace(parts[1])
		deviceID := strings.TrimSpace(parts[3])
		if siteID == "" || deviceID == "" {
			return TopicPartsV2{}, fmt.Errorf("invalid v2 topic: empty site_id or device_id")
		}
		if err := checkIdentifier(siteID, "site_id"); err != nil {
			return TopicPartsV2{}, err
		}
		if err := checkIdentifier(deviceID, "device_id"); err != nil {
			return TopicPartsV2{}, err
		}
		var kind Kind
		switch parts[6] {
		case "device":
			kind = KindDeviceV2
		case "interface":
			kind = KindInterfaceV2
		case "health":
			kind = KindHealthV2
		default:
			return TopicPartsV2{}, fmt.Errorf("invalid v2 device topic kind %q", parts[6])
		}
		return TopicPartsV2{SiteID: siteID, DeviceID: deviceID, Kind: kind}, nil
	}
	// site/{site}/collector/{collector}/telemetry/v2/heartbeat
	if len(parts) == 7 && parts[0] == "site" && parts[2] == "collector" && parts[4] == "telemetry" && parts[5] == "v2" && parts[6] == "heartbeat" {
		siteID := strings.TrimSpace(parts[1])
		collectorID := strings.TrimSpace(parts[3])
		if siteID == "" || collectorID == "" {
			return TopicPartsV2{}, fmt.Errorf("invalid heartbeat topic: empty site_id or collector_id")
		}
		if err := checkIdentifier(siteID, "site_id"); err != nil {
			return TopicPartsV2{}, err
		}
		if err := checkIdentifier(collectorID, "collector_id"); err != nil {
			return TopicPartsV2{}, err
		}
		return TopicPartsV2{SiteID: siteID, CollectorID: collectorID, Kind: KindHeartbeatV2}, nil
	}
	return TopicPartsV2{}, fmt.Errorf("invalid v2 topic layout: %q", topic)
}

// ValidateV2 parses and validates a v2 MQTT topic and JSON envelope payload.
func ValidateV2(topic string, payload []byte) (Message, error) {
	tp, err := ParseTopicV2(topic)
	if err != nil {
		return Message{}, err
	}
	if !json.Valid(payload) {
		return Message{}, fmt.Errorf("invalid JSON")
	}

	dec := json.NewDecoder(strings.NewReader(string(payload)))
	dec.DisallowUnknownFields()
	var raw envelopeRaw
	if err := dec.Decode(&raw); err != nil {
		return Message{}, fmt.Errorf("decode envelope: %w", err)
	}

	env, err := validateEnvelope(raw, tp)
	if err != nil {
		return Message{}, err
	}

	switch tp.Kind {
	case KindDeviceV2:
		msg, err := validateDeviceV2(env, raw.Payload)
		if err != nil {
			return Message{}, err
		}
		return Message{Kind: KindDeviceV2, DeviceV2: &msg}, nil
	case KindInterfaceV2:
		msg, err := validateInterfaceV2(env, raw.Payload)
		if err != nil {
			return Message{}, err
		}
		return Message{Kind: KindInterfaceV2, InterfaceV2: &msg}, nil
	case KindHealthV2:
		msg, err := validateHealthV2(env, raw.Payload)
		if err != nil {
			return Message{}, err
		}
		return Message{Kind: KindHealthV2, HealthV2: &msg}, nil
	case KindHeartbeatV2:
		msg, err := validateHeartbeatV2(env, raw.Payload)
		if err != nil {
			return Message{}, err
		}
		return Message{Kind: KindHeartbeatV2, HeartbeatV2: &msg}, nil
	default:
		return Message{}, fmt.Errorf("unsupported v2 kind %q", tp.Kind)
	}
}

func validateEnvelope(raw envelopeRaw, tp TopicPartsV2) (EnvelopeV2, error) {
	if raw.SchemaVersion != schemaVersionV2 {
		return EnvelopeV2{}, fmt.Errorf("unsupported schema_version %q", raw.SchemaVersion)
	}
	eventID, err := uuid.Parse(strings.TrimSpace(raw.EventID))
	if err != nil {
		return EnvelopeV2{}, fmt.Errorf("invalid event_id: %w", err)
	}
	if err := checkIdentifier(raw.SiteID, "site_id"); err != nil {
		return EnvelopeV2{}, err
	}
	if err := checkIdentifier(raw.CollectorID, "collector_id"); err != nil {
		return EnvelopeV2{}, err
	}
	if raw.SiteID != tp.SiteID {
		return EnvelopeV2{}, fmt.Errorf("site_id body %q does not match topic %q", raw.SiteID, tp.SiteID)
	}
	observedAt, err := parseTimestamp(raw.ObservedAt)
	if err != nil {
		return EnvelopeV2{}, fmt.Errorf("observed_at: %w", err)
	}
	emittedAt, err := parseTimestamp(raw.EmittedAt)
	if err != nil {
		return EnvelopeV2{}, fmt.Errorf("emitted_at: %w", err)
	}
	if strings.TrimSpace(raw.ConfigRevision) == "" || len(raw.ConfigRevision) > 128 {
		return EnvelopeV2{}, fmt.Errorf("config_revision is required and must be <= 128 chars")
	}
	if len(raw.Payload) == 0 || string(raw.Payload) == "null" || string(raw.Payload) == "{}" {
		return EnvelopeV2{}, fmt.Errorf("payload is required")
	}

	expectedType := ""
	switch tp.Kind {
	case KindDeviceV2:
		expectedType = "device_telemetry"
	case KindInterfaceV2:
		expectedType = "interface_telemetry"
	case KindHealthV2:
		expectedType = "health_state"
	case KindHeartbeatV2:
		expectedType = "collector_heartbeat"
	}
	if raw.EventType != expectedType {
		return EnvelopeV2{}, fmt.Errorf("event_type %q does not match topic kind", raw.EventType)
	}

	env := EnvelopeV2{
		SchemaVersion:  raw.SchemaVersion,
		EventID:        eventID,
		EventType:      raw.EventType,
		SiteID:         raw.SiteID,
		CollectorID:    raw.CollectorID,
		ObservedAt:     observedAt,
		EmittedAt:      emittedAt,
		ConfigRevision: raw.ConfigRevision,
	}

	switch tp.Kind {
	case KindHeartbeatV2:
		if raw.DeviceID != "" {
			return EnvelopeV2{}, fmt.Errorf("device_id must be absent for heartbeat")
		}
		if raw.CollectorID != tp.CollectorID {
			return EnvelopeV2{}, fmt.Errorf("collector_id body %q does not match topic %q", raw.CollectorID, tp.CollectorID)
		}
	default:
		if err := checkIdentifier(raw.DeviceID, "device_id"); err != nil {
			return EnvelopeV2{}, err
		}
		if raw.DeviceID != tp.DeviceID {
			return EnvelopeV2{}, fmt.Errorf("device_id body %q does not match topic %q", raw.DeviceID, tp.DeviceID)
		}
		env.DeviceID = raw.DeviceID
	}
	return env, nil
}

func validateDeviceV2(env EnvelopeV2, payload json.RawMessage) (DeviceTelemetryV2, error) {
	dec := json.NewDecoder(strings.NewReader(string(payload)))
	dec.DisallowUnknownFields()
	var p devicePayloadV2
	if err := dec.Decode(&p); err != nil {
		return DeviceTelemetryV2{}, fmt.Errorf("decode device payload: %w", err)
	}
	if strings.TrimSpace(p.Identity.Hostname) == "" || strings.TrimSpace(p.Identity.SysObjectID) == "" ||
		strings.TrimSpace(p.Identity.SysName) == "" || p.Identity.SysDescr == "" {
		return DeviceTelemetryV2{}, fmt.Errorf("device identity fields are required")
	}
	if p.Identity.SNMPVersion != "2c" {
		return DeviceTelemetryV2{}, fmt.Errorf("snmp_version must be 2c")
	}
	switch p.Profile.Name {
	case "core", "cisco", "arista", "unknown":
	default:
		return DeviceTelemetryV2{}, fmt.Errorf("invalid profile name %q", p.Profile.Name)
	}
	caps := make([]string, 0, len(p.Profile.Capabilities))
	seenCap := map[string]struct{}{}
	for _, c := range p.Profile.Capabilities {
		switch c {
		case "cpu", "memory", "temperature", "power":
		default:
			return DeviceTelemetryV2{}, fmt.Errorf("invalid capability %q", c)
		}
		if _, ok := seenCap[c]; ok {
			return DeviceTelemetryV2{}, fmt.Errorf("duplicate capability %q", c)
		}
		seenCap[c] = struct{}{}
		caps = append(caps, c)
	}
	if p.Readings.UptimeSeconds == nil || *p.Readings.UptimeSeconds < 0 {
		return DeviceTelemetryV2{}, fmt.Errorf("uptime_seconds is required and must be >= 0")
	}
	if err := checkOptionalPct(p.Readings.CPUUtilizationPct, "cpu_utilization_pct"); err != nil {
		return DeviceTelemetryV2{}, err
	}
	if err := checkOptionalPct(p.Readings.MemoryUtilizationPct, "memory_utilization_pct"); err != nil {
		return DeviceTelemetryV2{}, err
	}
	temps, err := validateTempComponents(p.TemperatureComponents)
	if err != nil {
		return DeviceTelemetryV2{}, err
	}
	powers, err := validatePowerComponents(p.PowerComponents)
	if err != nil {
		return DeviceTelemetryV2{}, err
	}
	if p.TemperatureComponents == nil || p.PowerComponents == nil {
		return DeviceTelemetryV2{}, fmt.Errorf("temperature_components and power_components are required")
	}
	return DeviceTelemetryV2{
		Envelope:              env,
		Hostname:              p.Identity.Hostname,
		ManagementAddress:     strings.TrimSpace(p.Identity.ManagementAddress),
		Role:                  strings.TrimSpace(p.Identity.Role),
		SysObjectID:           p.Identity.SysObjectID,
		SysName:               p.Identity.SysName,
		SysDescr:              p.Identity.SysDescr,
		Vendor:                p.Identity.Vendor,
		Model:                 p.Identity.Model,
		Serial:                p.Identity.Serial,
		SNMPVersion:           p.Identity.SNMPVersion,
		ProfileName:           p.Profile.Name,
		Capabilities:          caps,
		UptimeSeconds:         *p.Readings.UptimeSeconds,
		CPUUtilizationPct:     p.Readings.CPUUtilizationPct,
		MemoryUtilizationPct:  p.Readings.MemoryUtilizationPct,
		PrimaryTemperatureC:   p.Readings.PrimaryTemperatureC,
		TemperatureComponents: temps,
		PowerComponents:       powers,
	}, nil
}

func validateInterfaceV2(env EnvelopeV2, payload json.RawMessage) (InterfaceTelemetryV2, error) {
	dec := json.NewDecoder(strings.NewReader(string(payload)))
	dec.DisallowUnknownFields()
	var p interfacePayloadV2
	if err := dec.Decode(&p); err != nil {
		return InterfaceTelemetryV2{}, fmt.Errorf("decode interface payload: %w", err)
	}
	if p.Interface.IfIndex == nil || *p.Interface.IfIndex < 1 {
		return InterfaceTelemetryV2{}, fmt.Errorf("if_index must be >= 1")
	}
	if strings.TrimSpace(p.Interface.Name) == "" {
		return InterfaceTelemetryV2{}, fmt.Errorf("interface name is required")
	}
	if err := checkStatusEnum(p.Interface.AdminStatus, "admin_status"); err != nil {
		return InterfaceTelemetryV2{}, err
	}
	if err := checkStatusEnum(p.Interface.OperStatus, "oper_status"); err != nil {
		return InterfaceTelemetryV2{}, err
	}
	if p.Interface.SpeedBps != nil && *p.Interface.SpeedBps < 0 {
		return InterfaceTelemetryV2{}, fmt.Errorf("speed_bps must be >= 0")
	}
	if p.Counters.InOctets == nil || p.Counters.OutOctets == nil || p.Counters.InErrors == nil || p.Counters.OutErrors == nil {
		return InterfaceTelemetryV2{}, fmt.Errorf("counter fields are required")
	}
	return InterfaceTelemetryV2{
		Envelope:    env,
		IfIndex:     *p.Interface.IfIndex,
		Name:        p.Interface.Name,
		Alias:       p.Interface.Alias,
		Type:        p.Interface.Type,
		AdminStatus: p.Interface.AdminStatus,
		OperStatus:  p.Interface.OperStatus,
		SpeedBps:    p.Interface.SpeedBps,
		InOctets:    *p.Counters.InOctets,
		OutOctets:   *p.Counters.OutOctets,
		InErrors:    *p.Counters.InErrors,
		OutErrors:   *p.Counters.OutErrors,
		InDiscards:  p.Counters.InDiscards,
		OutDiscards: p.Counters.OutDiscards,
	}, nil
}

func validateHealthV2(env EnvelopeV2, payload json.RawMessage) (HealthStateV2, error) {
	dec := json.NewDecoder(strings.NewReader(string(payload)))
	dec.DisallowUnknownFields()
	var p healthPayloadV2
	if err := dec.Decode(&p); err != nil {
		return HealthStateV2{}, fmt.Errorf("decode health payload: %w", err)
	}
	switch p.State {
	case "healthy", "warning", "critical", "unknown":
	default:
		return HealthStateV2{}, fmt.Errorf("invalid health state %q", p.State)
	}
	switch p.Reason {
	case "reachable", "temperature_threshold", "direct_unreachable", "upstream_unreachable", "recovered":
	default:
		return HealthStateV2{}, fmt.Errorf("invalid health reason %q", p.Reason)
	}
	switch p.Transition {
	case "initial", "entered", "recovered", "reasserted":
	default:
		return HealthStateV2{}, fmt.Errorf("invalid health transition %q", p.Transition)
	}
	if p.PreviousState != nil {
		switch *p.PreviousState {
		case "healthy", "warning", "critical", "unknown":
		default:
			return HealthStateV2{}, fmt.Errorf("invalid previous_state %q", *p.PreviousState)
		}
	}
	if p.FailureCount == nil || *p.FailureCount < 0 {
		return HealthStateV2{}, fmt.Errorf("failure_count is required and must be >= 0")
	}
	if p.FailureThreshold == nil || *p.FailureThreshold < 1 {
		return HealthStateV2{}, fmt.Errorf("failure_threshold is required and must be >= 1")
	}
	if p.UpstreamDeviceIDs == nil || p.UnavailableUpstreamDeviceIDs == nil || p.RootCauseDeviceIDs == nil {
		return HealthStateV2{}, fmt.Errorf("upstream evidence arrays are required")
	}
	for _, id := range p.UpstreamDeviceIDs {
		if err := checkIdentifier(id, "upstream_device_ids"); err != nil {
			return HealthStateV2{}, err
		}
	}
	for _, id := range p.UnavailableUpstreamDeviceIDs {
		if err := checkIdentifier(id, "unavailable_upstream_device_ids"); err != nil {
			return HealthStateV2{}, err
		}
	}
	for _, id := range p.RootCauseDeviceIDs {
		if err := checkIdentifier(id, "root_cause_device_ids"); err != nil {
			return HealthStateV2{}, err
		}
	}
	alertsEnabled := true
	if p.AlertsEnabled != nil {
		alertsEnabled = *p.AlertsEnabled
	}
	return HealthStateV2{
		Envelope:                     env,
		State:                        p.State,
		Reason:                       p.Reason,
		PreviousState:                p.PreviousState,
		Transition:                   p.Transition,
		FailureCount:                 *p.FailureCount,
		FailureThreshold:             *p.FailureThreshold,
		TemperatureC:                 p.TemperatureC,
		TemperatureWarningC:          p.TemperatureWarningC,
		TemperaturePolicyRevision:    p.TemperaturePolicyRevision,
		UpstreamDeviceIDs:            cloneStrings(p.UpstreamDeviceIDs),
		UnavailableUpstreamDeviceIDs: cloneStrings(p.UnavailableUpstreamDeviceIDs),
		RootCauseDeviceIDs:           cloneStrings(p.RootCauseDeviceIDs),
		AlertsEnabled:                alertsEnabled,
	}, nil
}

func validateHeartbeatV2(env EnvelopeV2, payload json.RawMessage) (HeartbeatV2, error) {
	dec := json.NewDecoder(strings.NewReader(string(payload)))
	dec.DisallowUnknownFields()
	var p heartbeatPayloadV2
	if err := dec.Decode(&p); err != nil {
		return HeartbeatV2{}, fmt.Errorf("decode heartbeat payload: %w", err)
	}
	if strings.TrimSpace(p.Hostname) == "" || strings.TrimSpace(p.Version) == "" ||
		strings.TrimSpace(p.GitCommit) == "" || strings.TrimSpace(p.BuildTime) == "" {
		return HeartbeatV2{}, fmt.Errorf("heartbeat identity fields are required")
	}
	if p.BuildTime != "unknown" {
		if _, err := parseTimestamp(p.BuildTime); err != nil {
			return HeartbeatV2{}, fmt.Errorf("build_time must be RFC3339 or \"unknown\"")
		}
	}
	if p.UptimeSeconds == nil || *p.UptimeSeconds < 0 {
		return HeartbeatV2{}, fmt.Errorf("uptime_seconds is required and must be >= 0")
	}
	if p.SQLiteQueueDepth == nil || *p.SQLiteQueueDepth < 0 {
		return HeartbeatV2{}, fmt.Errorf("sqlite_queue_depth is required and must be >= 0")
	}
	if p.MemoryUsageBytes == nil || *p.MemoryUsageBytes < 0 {
		return HeartbeatV2{}, fmt.Errorf("memory_usage_bytes is required and must be >= 0")
	}
	if p.GoroutineCount == nil || *p.GoroutineCount < 0 {
		return HeartbeatV2{}, fmt.Errorf("goroutine_count is required and must be >= 0")
	}
	return HeartbeatV2{
		Envelope:         env,
		Hostname:         p.Hostname,
		Version:          p.Version,
		GitCommit:        p.GitCommit,
		BuildTime:        p.BuildTime,
		UptimeSeconds:    *p.UptimeSeconds,
		SQLiteQueueDepth: *p.SQLiteQueueDepth,
		MemoryUsageBytes: *p.MemoryUsageBytes,
		GoroutineCount:   *p.GoroutineCount,
	}, nil
}

func validateTempComponents(raw []componentRaw) ([]ComponentReading, error) {
	out := make([]ComponentReading, 0, len(raw))
	seen := map[string]struct{}{}
	for _, c := range raw {
		if err := checkComponentCommon(c); err != nil {
			return nil, err
		}
		if c.Unit != "celsius" {
			return nil, fmt.Errorf("temperature component unit must be celsius")
		}
		if c.Value == nil {
			return nil, fmt.Errorf("temperature component value is required")
		}
		if _, ok := seen[c.ComponentID]; ok {
			return nil, fmt.Errorf("duplicate temperature component_id %q", c.ComponentID)
		}
		seen[c.ComponentID] = struct{}{}
		observedAt, err := parseTimestamp(c.ObservedAt)
		if err != nil {
			return nil, fmt.Errorf("temperature component observed_at: %w", err)
		}
		val := *c.Value
		out = append(out, ComponentReading{
			ComponentID: c.ComponentID,
			Name:        c.Name,
			Index:       c.Index,
			Value:       &val,
			Unit:        c.Unit,
			Status:      c.Status,
			ObservedAt:  observedAt,
		})
	}
	return out, nil
}

func validatePowerComponents(raw []componentRaw) ([]ComponentReading, error) {
	out := make([]ComponentReading, 0, len(raw))
	seen := map[string]struct{}{}
	for _, c := range raw {
		if err := checkComponentCommon(c); err != nil {
			return nil, err
		}
		switch c.Unit {
		case "state", "watts", "volts", "amps", "percent":
		default:
			return nil, fmt.Errorf("invalid power component unit %q", c.Unit)
		}
		if _, ok := seen[c.ComponentID]; ok {
			return nil, fmt.Errorf("duplicate power component_id %q", c.ComponentID)
		}
		seen[c.ComponentID] = struct{}{}
		observedAt, err := parseTimestamp(c.ObservedAt)
		if err != nil {
			return nil, fmt.Errorf("power component observed_at: %w", err)
		}
		out = append(out, ComponentReading{
			ComponentID: c.ComponentID,
			Name:        c.Name,
			Index:       c.Index,
			Value:       c.Value,
			Unit:        c.Unit,
			Status:      c.Status,
			ObservedAt:  observedAt,
		})
	}
	return out, nil
}

func checkComponentCommon(c componentRaw) error {
	if strings.TrimSpace(c.ComponentID) == "" || strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Index) == "" {
		return fmt.Errorf("component_id, name, and index are required")
	}
	switch c.Status {
	case "ok", "warning", "critical", "not_present", "unknown":
	default:
		return fmt.Errorf("invalid component status %q", c.Status)
	}
	return nil
}

func checkIdentifier(v, field string) error {
	v = strings.TrimSpace(v)
	if v == "" || len(v) > 128 || !identifierPattern.MatchString(v) {
		return fmt.Errorf("invalid %s %q", field, v)
	}
	return nil
}

func checkOptionalPct(v *float64, field string) error {
	if v == nil {
		return nil
	}
	if *v < 0 || *v > 100 {
		return fmt.Errorf("%s must be between 0 and 100", field)
	}
	return nil
}

func checkStatusEnum(v, field string) error {
	switch v {
	case "up", "down", "testing", "unknown":
		return nil
	default:
		return fmt.Errorf("invalid %s %q", field, v)
	}
}

// cloneStrings copies in and always returns a non-nil slice.
// append([]string(nil), empty...) yields nil, which pgx encodes as SQL NULL.
func cloneStrings(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}
