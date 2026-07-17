// Package status provides an operator-facing runtime cache for the local
// control plane. It is not a metrics database and does not retain history,
// OID dumps, or time-series counter data.
package status

import (
	"sync"
	"time"
)

// PollResultClass summarizes the last poll outcome for operator display.
type PollResultClass string

const (
	PollSuccess PollResultClass = "success"
	PollFailure PollResultClass = "failure"
)

// InterfaceSummary is a bounded aggregate of interface selection.
type InterfaceSummary struct {
	Selected        int `json:"selected"`
	ExcludedDefault int `json:"excluded_default"`
	ExcludedRule    int `json:"excluded_rule"`
}

// ComponentSummary is a bounded aggregate of vendor component readings.
type ComponentSummary struct {
	Profile           string   `json:"profile,omitempty"`
	TemperatureCount  int      `json:"temperature_count"`
	PowerCount        int      `json:"power_count"`
	HasCPU            bool     `json:"has_cpu"`
	HasMemory         bool     `json:"has_memory"`
	PrimaryTempC      *float64 `json:"primary_temp_c,omitempty"`
	CPUUtilizationPct *float64 `json:"cpu_utilization_pct,omitempty"`
	MemUtilizationPct *float64 `json:"memory_utilization_pct,omitempty"`
}

// DevicePoll is the last-poll operator cache entry for one device.
type DevicePoll struct {
	DeviceID    string           `json:"device_id"`
	ObservedAt  time.Time        `json:"observed_at"`
	Result      PollResultClass  `json:"result"`
	ErrorClass  string           `json:"error_class,omitempty"`
	SysName     string           `json:"sys_name,omitempty"`
	SysObjectID string           `json:"sys_object_id,omitempty"`
	Interfaces  InterfaceSummary `json:"interfaces"`
	Components  ComponentSummary `json:"components"`
}

// ReloadStatus records the most recent configuration reload attempt.
type ReloadStatus struct {
	Success    bool      `json:"success"`
	Message    string    `json:"message,omitempty"`
	Revision   string    `json:"revision,omitempty"`
	ObservedAt time.Time `json:"observed_at"`
}

// TransportSnapshot is a point-in-time transport view for operators.
type TransportSnapshot struct {
	PublisherMode   string `json:"publisher_mode"`
	MQTTConnected   *bool  `json:"mqtt_connected,omitempty"`
	BufferDepth     int64  `json:"buffer_depth"`
	BufferAvailable bool   `json:"buffer_available"`
}

// Snapshot is the full operator status view.
type Snapshot struct {
	ConfigRevision string                `json:"config_revision"`
	Reload         *ReloadStatus         `json:"reload,omitempty"`
	Devices        map[string]DevicePoll `json:"devices"`
}

// Store is a concurrent operator state cache.
type Store struct {
	mu       sync.RWMutex
	devices  map[string]DevicePoll
	reload   *ReloadStatus
	revision string
}

// New creates an empty operator status store.
func New() *Store {
	return &Store{devices: make(map[string]DevicePoll)}
}

// SetRevision records the active configuration revision fingerprint.
func (s *Store) SetRevision(revision string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.revision = revision
}

// RecordReload stores the latest reload outcome.
func (s *Store) RecordReload(success bool, message, revision string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reload = &ReloadStatus{
		Success:    success,
		Message:    message,
		Revision:   revision,
		ObservedAt: time.Now().UTC(),
	}
	if success && revision != "" {
		s.revision = revision
	}
}

// RecordPoll upserts the last-poll summary for a device.
func (s *Store) RecordPoll(poll DevicePoll) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.devices == nil {
		s.devices = make(map[string]DevicePoll)
	}
	poll.ObservedAt = poll.ObservedAt.UTC()
	s.devices[poll.DeviceID] = poll
}

// Device returns a copy of one device's last-poll entry.
func (s *Store) Device(id string) (DevicePoll, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	poll, ok := s.devices[id]
	return poll, ok
}

// Snapshot returns a deep copy of the operator cache.
func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := Snapshot{
		ConfigRevision: s.revision,
		Devices:        make(map[string]DevicePoll, len(s.devices)),
	}
	if s.reload != nil {
		reload := *s.reload
		out.Reload = &reload
	}
	for id, poll := range s.devices {
		out.Devices[id] = poll
	}
	return out
}

// Prune removes last-poll entries for device IDs that are no longer configured.
func (s *Store) Prune(activeIDs map[string]struct{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id := range s.devices {
		if _, ok := activeIDs[id]; !ok {
			delete(s.devices, id)
		}
	}
}
