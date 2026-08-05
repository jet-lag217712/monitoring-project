// Package health evaluates local device health and reachability correlation.
package health

import (
	"time"
)

// State is a terminal health state.
type State string

const (
	StateHealthy  State = "healthy"
	StateWarning  State = "warning"
	StateCritical State = "critical"
	StateUnknown  State = "unknown"
)

// Reason is the public health reason code.
type Reason string

const (
	ReasonReachable            Reason = "reachable"
	ReasonTemperatureThreshold Reason = "temperature_threshold"
	ReasonDirectUnreachable    Reason = "direct_unreachable"
	ReasonUpstreamUnreachable  Reason = "upstream_unreachable"
	ReasonRecovered            Reason = "recovered"
)

// Transition classifies how a health event relates to the prior terminal state.
// TransitionReasserted exists for schema alignment but is never emitted in Phase 3.
type Transition string

const (
	TransitionInitial    Transition = "initial"
	TransitionEntered    Transition = "entered"
	TransitionRecovered  Transition = "recovered"
	TransitionReasserted Transition = "reasserted"
)

// PollOutcome is the committed result of one device poll attempt.
type PollOutcome struct {
	DeviceID   string
	Success    bool
	ObservedAt time.Time
	// TemperatureC is set only when a valid primary temperature was observed.
	TemperatureC *float64
}

// Event is a local health transition. It intentionally omits MQTT envelope fields.
type Event struct {
	DeviceID                     string
	State                        State
	Reason                       Reason
	PreviousState                *State
	Transition                   Transition
	FailureCount                 int
	FailureThreshold             int
	TemperatureC                 *float64
	TemperatureWarningC          *float64
	TemperaturePolicyRevision    string
	UpstreamDeviceIDs            []string
	UnavailableUpstreamDeviceIDs []string
	RootCauseDeviceIDs           []string
	// AlertsEnabled is copied from the active device overlay at emit time.
	// false means Administratively Ignored (monitor without site-alert impact).
	AlertsEnabled bool
	ObservedAt    time.Time
}

// DeviceHealth is the committed per-device ledger entry.
type DeviceHealth struct {
	HasState            bool
	State               State
	Reason              Reason
	FailureCount        int
	LastSuccessAt       time.Time
	LastObservedAt      time.Time
	LastOutcomeSuccess  bool
	HasOutcome          bool
	TemperatureC        *float64
	TemperatureWarningC *float64
	UpstreamDeviceIDs   []string
	UnavailableUpstream []string
	RootCauseDeviceIDs  []string
	// AlertsEnabledPublished tracks the last alerts_enabled value emitted so
	// overlay toggles can force a TransitionReasserted publish.
	AlertsEnabledPublished bool
	HasAlertsEnabled       bool
}

// Snapshot is a read-only copy of committed tracker gauges.
type Snapshot struct {
	DevicesByState     map[State]int
	DependencyImpacted int
	PendingFailures    int
}
