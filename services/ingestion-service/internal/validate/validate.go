package validate

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Kind identifies the metric message type from the topic path.
type Kind string

const (
	KindDevice    Kind = "device"
	KindInterface Kind = "interface"
)

// TopicParts holds identifiers parsed from an MQTT topic.
type TopicParts struct {
	SiteID   string
	DeviceID string
	Kind     Kind
}

// DeviceMessage is a validated device-level metric payload.
type DeviceMessage struct {
	SiteID    string
	DeviceID  string
	IPAddress string
	Timestamp time.Time
	Metric    string
	Value     float64
}

// InterfaceMessage is a validated interface-level metric payload.
type InterfaceMessage struct {
	SiteID    string
	DeviceID  string
	Timestamp time.Time
	IfIndex   int
	InOctets  uint64
	OutOctets uint64
	InErrors  uint64
	OutErrors uint64
}

// Message is the result of validating a topic + payload.
type Message struct {
	Kind         Kind
	Device       *DeviceMessage
	Interface    *InterfaceMessage
	DeviceV2     *DeviceTelemetryV2
	InterfaceV2  *InterfaceTelemetryV2
	HealthV2     *HealthStateV2
	HeartbeatV2  *HeartbeatV2
}

type devicePayload struct {
	Timestamp string   `json:"timestamp"`
	SiteID    string   `json:"site_id"`
	DeviceID  string   `json:"device_id"`
	IPAddress string   `json:"ip_address"`
	Metric    string   `json:"metric"`
	Value     *float64 `json:"value"`
}

type interfacePayload struct {
	Timestamp string  `json:"timestamp"`
	SiteID    string  `json:"site_id"`
	DeviceID  string  `json:"device_id"`
	IfIndex   *int    `json:"if_index"`
	InOctets  *uint64 `json:"in_octets"`
	OutOctets *uint64 `json:"out_octets"`
	InErrors  *uint64 `json:"in_errors"`
	OutErrors *uint64 `json:"out_errors"`
}

// ParseTopic extracts site, device, and kind from
// site/{site_id}/device/{device_id}/metric/{device|interface}.
func ParseTopic(topic string) (TopicParts, error) {
	parts := strings.Split(topic, "/")
	if len(parts) != 6 {
		return TopicParts{}, fmt.Errorf("invalid topic: expected 6 segments, got %d", len(parts))
	}
	if parts[0] != "site" || parts[2] != "device" || parts[4] != "metric" {
		return TopicParts{}, fmt.Errorf("invalid topic layout: %q", topic)
	}
	siteID := strings.TrimSpace(parts[1])
	deviceID := strings.TrimSpace(parts[3])
	if siteID == "" || deviceID == "" {
		return TopicParts{}, fmt.Errorf("invalid topic: empty site_id or device_id")
	}
	kind := Kind(parts[5])
	switch kind {
	case KindDevice, KindInterface:
	default:
		return TopicParts{}, fmt.Errorf("invalid topic kind %q", parts[5])
	}
	return TopicParts{SiteID: siteID, DeviceID: deviceID, Kind: kind}, nil
}

// Validate parses and validates an MQTT topic and JSON payload (v1 or v2).
func Validate(topic string, payload []byte) (Message, error) {
	if strings.Contains(topic, "/telemetry/v2/") {
		return ValidateV2(topic, payload)
	}

	tp, err := ParseTopic(topic)
	if err != nil {
		return Message{}, err
	}
	if !json.Valid(payload) {
		return Message{}, fmt.Errorf("invalid JSON")
	}

	switch tp.Kind {
	case KindDevice:
		msg, err := validateDevice(tp, payload)
		if err != nil {
			return Message{}, err
		}
		return Message{Kind: KindDevice, Device: &msg}, nil
	case KindInterface:
		msg, err := validateInterface(tp, payload)
		if err != nil {
			return Message{}, err
		}
		return Message{Kind: KindInterface, Interface: &msg}, nil
	default:
		return Message{}, fmt.Errorf("unsupported kind %q", tp.Kind)
	}
}

func validateDevice(tp TopicParts, payload []byte) (DeviceMessage, error) {
	var p devicePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return DeviceMessage{}, fmt.Errorf("decode device payload: %w", err)
	}
	if err := checkOptionalIDs(tp, p.SiteID, p.DeviceID); err != nil {
		return DeviceMessage{}, err
	}
	ts, err := parseTimestamp(p.Timestamp)
	if err != nil {
		return DeviceMessage{}, err
	}
	if strings.TrimSpace(p.Metric) == "" {
		return DeviceMessage{}, fmt.Errorf("metric is required")
	}
	if p.Value == nil {
		return DeviceMessage{}, fmt.Errorf("value is required")
	}
	return DeviceMessage{
		SiteID:    tp.SiteID,
		DeviceID:  tp.DeviceID,
		IPAddress: strings.TrimSpace(p.IPAddress),
		Timestamp: ts,
		Metric:    p.Metric,
		Value:     *p.Value,
	}, nil
}

func validateInterface(tp TopicParts, payload []byte) (InterfaceMessage, error) {
	var p interfacePayload
	if err := json.Unmarshal(payload, &p); err != nil {
		return InterfaceMessage{}, fmt.Errorf("decode interface payload: %w", err)
	}
	if err := checkOptionalIDs(tp, p.SiteID, p.DeviceID); err != nil {
		return InterfaceMessage{}, err
	}
	ts, err := parseTimestamp(p.Timestamp)
	if err != nil {
		return InterfaceMessage{}, err
	}
	if p.IfIndex == nil {
		return InterfaceMessage{}, fmt.Errorf("if_index is required")
	}
	if p.InOctets == nil || p.OutOctets == nil || p.InErrors == nil || p.OutErrors == nil {
		return InterfaceMessage{}, fmt.Errorf("in_octets, out_octets, in_errors, and out_errors are required")
	}
	return InterfaceMessage{
		SiteID:    tp.SiteID,
		DeviceID:  tp.DeviceID,
		Timestamp: ts,
		IfIndex:   *p.IfIndex,
		InOctets:  *p.InOctets,
		OutOctets: *p.OutOctets,
		InErrors:  *p.InErrors,
		OutErrors: *p.OutErrors,
	}, nil
}

func checkOptionalIDs(tp TopicParts, siteID, deviceID string) error {
	if siteID != "" && siteID != tp.SiteID {
		return fmt.Errorf("site_id body %q does not match topic %q", siteID, tp.SiteID)
	}
	if deviceID != "" && deviceID != tp.DeviceID {
		return fmt.Errorf("device_id body %q does not match topic %q", deviceID, tp.DeviceID)
	}
	return nil
}

func parseTimestamp(raw string) (time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return time.Time{}, fmt.Errorf("timestamp is required")
	}
	ts, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		// Also accept RFC3339Nano from collectors.
		ts, err = time.Parse(time.RFC3339Nano, raw)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid timestamp: %w", err)
		}
	}
	return ts.UTC(), nil
}
