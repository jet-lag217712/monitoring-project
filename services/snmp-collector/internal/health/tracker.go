package health

import (
	"sort"
	"sync"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
)

// Tracker owns the committed failure ledger and post-cycle health evaluation.
type Tracker struct {
	mu      sync.RWMutex
	devices map[string]*DeviceHealth
}

// NewTracker creates an empty health tracker.
func NewTracker() *Tracker {
	return &Tracker{devices: make(map[string]*DeviceHealth)}
}

// Prune applies reload retention: remove devices absent from activeIDs.
// Retained devices keep failure count, terminal state, last success time, and evidence.
// New devices are created as unobserved on first ApplyBatch.
func (t *Tracker) Prune(activeIDs []string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.devices == nil {
		t.devices = make(map[string]*DeviceHealth)
	}
	active := make(map[string]struct{}, len(activeIDs))
	for _, id := range activeIDs {
		active[id] = struct{}{}
	}
	for id := range t.devices {
		if _, ok := active[id]; !ok {
			delete(t.devices, id)
		}
	}
}

// Device returns a copy of the committed health entry, if present.
func (t *Tracker) Device(id string) (DeviceHealth, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	entry, ok := t.devices[id]
	if !ok || entry == nil {
		return DeviceHealth{}, false
	}
	return cloneDeviceHealth(*entry), true
}

// Snapshot returns committed gauge inputs after correlation.
func (t *Tracker) Snapshot() Snapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()
	snap := Snapshot{
		DevicesByState: map[State]int{
			StateHealthy:  0,
			StateWarning:  0,
			StateCritical: 0,
			StateUnknown:  0,
		},
	}
	for _, entry := range t.devices {
		if entry == nil {
			continue
		}
		if !entry.HasState {
			if entry.FailureCount > 0 {
				snap.PendingFailures++
			}
			continue
		}
		snap.DevicesByState[entry.State]++
		if entry.State == StateUnknown {
			snap.DependencyImpacted++
		}
		if entry.FailureCount > 0 && entry.State != StateCritical && entry.State != StateUnknown {
			snap.PendingFailures++
		}
	}
	return snap
}

// ApplyBatch commits outcomes for a completed cycle and evaluates the full inventory.
// Events are returned sorted by device ID. TransitionReasserted is emitted when
// alerts_enabled (Administratively Ignored) changes without a state change.
func (t *Tracker) ApplyBatch(cfg *config.Config, outcomes []PollOutcome) []Event {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.devices == nil {
		t.devices = make(map[string]*DeviceHealth)
	}
	if cfg == nil {
		return nil
	}

	threshold := cfg.Health.FailureThreshold
	if threshold < 1 {
		threshold = 1
	}
	policyRevision := config.TemperaturePolicyRevision(cfg)

	index := buildDeviceIndex(cfg.Devices)
	for id := range index {
		if _, ok := t.devices[id]; !ok {
			t.devices[id] = &DeviceHealth{}
		}
	}

	for _, outcome := range outcomes {
		entry, ok := t.devices[outcome.DeviceID]
		if !ok {
			continue
		}
		entry.HasOutcome = true
		entry.LastOutcomeSuccess = outcome.Success
		entry.LastObservedAt = outcome.ObservedAt.UTC()
		entry.TemperatureC = cloneFloat(outcome.TemperatureC)
		if outcome.Success {
			entry.FailureCount = 0
			entry.LastSuccessAt = outcome.ObservedAt.UTC()
		} else {
			entry.FailureCount++
		}
	}

	order := topologicalOrder(index)
	var events []Event
	for _, id := range order {
		device := index[id]
		entry := t.devices[id]
		warningC := device.EffectiveTemperatureWarningC(cfg.Health.TemperatureWarningC)
		upstreams := normalizeIDs(device.UpstreamDeviceIDs)

		desired, ok := t.evaluateDevice(entry, upstreams, warningC, threshold)
		if !ok {
			continue
		}

		previous := previousStatePtr(entry)
		transition := classifyTransition(entry.HasState, entry.State, desired.state)
		ledgerReason := desired.reason
		eventReason := ledgerReason
		if transition == TransitionRecovered {
			eventReason = ReasonRecovered
		}

		alertsEnabled := device.AlertsEnabledOrDefault()
		policyChanged := entry.HasState && entry.HasAlertsEnabled && entry.AlertsEnabledPublished != alertsEnabled
		emit := !entry.HasState || entry.State != desired.state || entry.Reason != ledgerReason || policyChanged
		if policyChanged && entry.State == desired.state && entry.Reason == ledgerReason {
			transition = TransitionReasserted
		}

		entry.HasState = true
		entry.State = desired.state
		entry.Reason = ledgerReason
		entry.TemperatureC = cloneFloat(desired.temperatureC)
		entry.TemperatureWarningC = floatPtr(warningC)
		entry.UpstreamDeviceIDs = append([]string(nil), upstreams...)
		entry.UnavailableUpstream = append([]string(nil), desired.unavailable...)
		entry.RootCauseDeviceIDs = append([]string(nil), desired.rootCauses...)
		entry.AlertsEnabledPublished = alertsEnabled
		entry.HasAlertsEnabled = true

		if !emit {
			continue
		}

		observedAt := entry.LastObservedAt
		if observedAt.IsZero() {
			observedAt = entry.LastSuccessAt
		}

		events = append(events, Event{
			DeviceID:                     id,
			State:                        desired.state,
			Reason:                       eventReason,
			PreviousState:                previous,
			Transition:                   transition,
			FailureCount:                 entry.FailureCount,
			FailureThreshold:             threshold,
			TemperatureC:                 cloneFloat(desired.temperatureC),
			TemperatureWarningC:          floatPtr(warningC),
			TemperaturePolicyRevision:    policyRevision,
			UpstreamDeviceIDs:            append([]string(nil), upstreams...),
			UnavailableUpstreamDeviceIDs: append([]string(nil), desired.unavailable...),
			RootCauseDeviceIDs:           append([]string(nil), desired.rootCauses...),
			AlertsEnabled:                alertsEnabled,
			ObservedAt:                   observedAt,
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].DeviceID < events[j].DeviceID
	})
	return events
}

type evaluated struct {
	state        State
	reason       Reason
	temperatureC *float64
	unavailable  []string
	rootCauses   []string
}

func (t *Tracker) evaluateDevice(entry *DeviceHealth, upstreams []string, warningC float64, threshold int) (evaluated, bool) {
	if entry == nil || !entry.HasOutcome {
		return evaluated{}, false
	}

	if entry.LastOutcomeSuccess {
		return t.evaluateSuccess(entry, warningC), true
	}

	if entry.FailureCount < threshold {
		return evaluated{}, false
	}

	return t.evaluateFailure(entry, upstreams, threshold)
}

func (t *Tracker) evaluateSuccess(entry *DeviceHealth, warningC float64) evaluated {
	desired := evaluated{
		temperatureC: cloneFloat(entry.TemperatureC),
	}
	if entry.TemperatureC != nil && *entry.TemperatureC >= warningC {
		desired.state = StateWarning
		desired.reason = ReasonTemperatureThreshold
	} else {
		desired.state = StateHealthy
		desired.reason = ReasonReachable
	}
	return desired
}

func (t *Tracker) evaluateFailure(entry *DeviceHealth, upstreams []string, threshold int) (evaluated, bool) {
	if len(upstreams) == 0 {
		return evaluated{
			state:  StateCritical,
			reason: ReasonDirectUnreachable,
		}, true
	}

	anySuccess := false
	anyPending := false
	unavailable := make([]string, 0, len(upstreams))
	for _, upID := range upstreams {
		up := t.devices[upID]
		if up == nil || !up.HasOutcome {
			anyPending = true
			continue
		}
		if up.LastOutcomeSuccess {
			anySuccess = true
			continue
		}
		if up.FailureCount < threshold {
			anyPending = true
			continue
		}
		if !up.HasState || (up.State != StateCritical && up.State != StateUnknown) {
			anyPending = true
			continue
		}
		unavailable = append(unavailable, upID)
	}

	if anySuccess {
		return evaluated{
			state:  StateCritical,
			reason: ReasonDirectUnreachable,
		}, true
	}
	if anyPending || len(unavailable) < len(upstreams) {
		return evaluated{}, false
	}

	sort.Strings(unavailable)
	return evaluated{
		state:       StateUnknown,
		reason:      ReasonUpstreamUnreachable,
		unavailable: unavailable,
		rootCauses:  t.collectRootCauses(unavailable),
	}, true
}

func (t *Tracker) collectRootCauses(unavailable []string) []string {
	seen := make(map[string]struct{})
	var out []string
	var walk func(id string)
	walk = func(id string) {
		entry := t.devices[id]
		if entry == nil || !entry.HasState {
			return
		}
		switch entry.State {
		case StateCritical:
			if _, ok := seen[id]; ok {
				return
			}
			seen[id] = struct{}{}
			out = append(out, id)
		case StateUnknown:
			for _, up := range entry.UnavailableUpstream {
				walk(up)
			}
		}
	}
	for _, id := range unavailable {
		walk(id)
	}
	sort.Strings(out)
	return out
}

func classifyTransition(hasState bool, previous, next State) Transition {
	if !hasState {
		return TransitionInitial
	}
	if isRecovery(previous, next) {
		return TransitionRecovered
	}
	if previous != next {
		return TransitionEntered
	}
	// Reason-only changes still use entered; callers skip when reason also unchanged.
	return TransitionEntered
}

func isRecovery(previous, next State) bool {
	switch previous {
	case StateCritical, StateUnknown, StateWarning:
		return healthier(next, previous)
	default:
		return false
	}
}

func healthier(candidate, baseline State) bool {
	return stateRank(candidate) > stateRank(baseline)
}

func stateRank(state State) int {
	switch state {
	case StateHealthy:
		return 4
	case StateWarning:
		return 3
	case StateUnknown:
		return 2
	case StateCritical:
		return 1
	default:
		return 0
	}
}

func previousStatePtr(entry *DeviceHealth) *State {
	if entry == nil || !entry.HasState {
		return nil
	}
	state := entry.State
	return &state
}

func buildDeviceIndex(devices []config.DeviceConfig) map[string]config.DeviceConfig {
	index := make(map[string]config.DeviceConfig, len(devices))
	for _, device := range devices {
		index[device.ID] = device
	}
	return index
}

func topologicalOrder(index map[string]config.DeviceConfig) []string {
	ids := make([]string, 0, len(index))
	for id := range index {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	indegree := make(map[string]int, len(index))
	children := make(map[string][]string, len(index))
	for id, device := range index {
		_ = indegree[id]
		for _, up := range normalizeIDs(device.UpstreamDeviceIDs) {
			if _, ok := index[up]; !ok {
				continue
			}
			children[up] = append(children[up], id)
			indegree[id]++
		}
	}
	for id := range children {
		sort.Strings(children[id])
	}

	queue := make([]string, 0, len(index))
	for _, id := range ids {
		if indegree[id] == 0 {
			queue = append(queue, id)
		}
	}
	order := make([]string, 0, len(index))
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		order = append(order, id)
		for _, child := range children[id] {
			indegree[child]--
			if indegree[child] == 0 {
				queue = append(queue, child)
				sort.Strings(queue)
			}
		}
	}
	if len(order) < len(index) {
		seen := make(map[string]struct{}, len(order))
		for _, id := range order {
			seen[id] = struct{}{}
		}
		for _, id := range ids {
			if _, ok := seen[id]; !ok {
				order = append(order, id)
			}
		}
	}
	return order
}

func normalizeIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(ids))
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func cloneFloat(v *float64) *float64 {
	if v == nil {
		return nil
	}
	cp := *v
	return &cp
}

func floatPtr(v float64) *float64 {
	cp := v
	return &cp
}

func cloneDeviceHealth(in DeviceHealth) DeviceHealth {
	out := in
	out.TemperatureC = cloneFloat(in.TemperatureC)
	out.TemperatureWarningC = cloneFloat(in.TemperatureWarningC)
	out.UpstreamDeviceIDs = append([]string(nil), in.UpstreamDeviceIDs...)
	out.UnavailableUpstream = append([]string(nil), in.UnavailableUpstream...)
	out.RootCauseDeviceIDs = append([]string(nil), in.RootCauseDeviceIDs...)
	return out
}
