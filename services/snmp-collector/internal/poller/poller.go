package poller

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/health"
	"github.com/equate/ogsd/services/snmp-collector/internal/metrics"
	"github.com/equate/ogsd/services/snmp-collector/internal/publisher"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/core"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/filter"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/profile"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/readings"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/vendors"
	"github.com/equate/ogsd/services/snmp-collector/internal/telemetry"
)

// profileWalkBudget covers the largest vendor profile walk count in the registry.
const profileWalkBudget = 13

// ConfigSource supplies immutable collector configuration snapshots.
type ConfigSource interface {
	Current() *config.Config
}

type staticConfigSource struct {
	cfg *config.Config
}

func (s staticConfigSource) Current() *config.Config {
	return s.cfg
}

// Poller schedules bounded concurrent device polls and post-cycle health evaluation.
type Poller struct {
	source   ConfigSource
	pub      publisher.Publisher
	metrics  *metrics.Collector
	log      *slog.Logger
	registry *profile.Registry
	health   *health.Tracker

	mu            sync.Mutex
	activeDevices map[string]struct{}
}

// New creates a Poller using a fixed configuration snapshot.
func New(cfg *config.Config, pub publisher.Publisher, m *metrics.Collector, log *slog.Logger) *Poller {
	return NewWithConfigSource(staticConfigSource{cfg: cfg}, pub, m, log)
}

// NewWithConfigSource creates a Poller backed by a reloadable snapshot source.
func NewWithConfigSource(source ConfigSource, pub publisher.Publisher, m *metrics.Collector, log *slog.Logger) *Poller {
	if log == nil {
		log = slog.Default()
	}
	return &Poller{
		source:        source,
		pub:           pub,
		metrics:       m,
		log:           log,
		registry:      vendors.NewRegistry(),
		health:        health.NewTracker(),
		activeDevices: make(map[string]struct{}),
	}
}

// Tracker exposes the committed health ledger for tests.
func (p *Poller) Tracker() *health.Tracker {
	return p.health
}

// Run polls due devices until ctx is cancelled. Each device has an independent
// schedule, while all work remains bounded by the active global worker limit.
func (p *Poller) Run(ctx context.Context) {
	nextDue := make(map[string]time.Time)
	firstCycle := true

	for {
		cfg := p.source.Current()
		p.syncInventory(cfg)
		pruneNextDue(nextDue, cfg.Devices)
		now := time.Now()
		due := dueDevices(cfg, nextDue, firstCycle, now)
		if len(due) > 0 {
			p.pollAll(ctx, cfg, due)
			now = time.Now()
			for _, device := range due {
				nextDue[device.ID] = now.Add(device.EffectivePollInterval(cfg.PollInterval))
			}
			firstCycle = false
			if ctx.Err() != nil {
				return
			}
			continue
		}

		wait := nextDueWait(nextDue, now)
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

func (p *Poller) syncInventory(cfg *config.Config) {
	if cfg == nil {
		return
	}
	ids := make([]string, 0, len(cfg.Devices))
	next := make(map[string]struct{}, len(cfg.Devices))
	for _, device := range cfg.Devices {
		ids = append(ids, device.ID)
		next[device.ID] = struct{}{}
	}

	p.mu.Lock()
	changed := len(p.activeDevices) != len(next)
	if !changed {
		for id := range next {
			if _, ok := p.activeDevices[id]; !ok {
				changed = true
				break
			}
		}
	}
	if changed {
		p.health.Prune(ids)
		p.activeDevices = next
	}
	p.mu.Unlock()
}

func pruneNextDue(nextDue map[string]time.Time, devices []config.DeviceConfig) {
	active := make(map[string]struct{}, len(devices))
	for _, device := range devices {
		active[device.ID] = struct{}{}
	}
	for id := range nextDue {
		if _, ok := active[id]; !ok {
			delete(nextDue, id)
		}
	}
}

func dueDevices(cfg *config.Config, nextDue map[string]time.Time, firstCycle bool, now time.Time) []config.DeviceConfig {
	if firstCycle {
		return append([]config.DeviceConfig(nil), cfg.Devices...)
	}
	due := make([]config.DeviceConfig, 0, len(cfg.Devices))
	for _, device := range cfg.Devices {
		if scheduled, ok := nextDue[device.ID]; !ok || !scheduled.After(now) {
			due = append(due, device)
		}
	}
	return due
}

func nextDueWait(nextDue map[string]time.Time, now time.Time) time.Duration {
	wait := time.Hour
	for _, scheduled := range nextDue {
		if scheduled.Before(now) {
			return 0
		}
		if candidate := scheduled.Sub(now); candidate < wait {
			wait = candidate
		}
	}
	if wait <= 0 {
		return time.Millisecond
	}
	return wait
}

func (p *Poller) pollAll(ctx context.Context, cfg *config.Config, devices []config.DeviceConfig) {
	jobs := make(chan config.DeviceConfig)
	workerCount := cfg.MaxWorkers
	if workerCount > len(devices) {
		workerCount = len(devices)
	}
	if workerCount < 1 {
		return
	}

	outcomes := make([]health.PollOutcome, 0, len(devices))
	var outcomesMu sync.Mutex
	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for device := range jobs {
				if ctx.Err() != nil {
					return
				}
				outcome := p.pollDevice(ctx, cfg, device)
				outcomesMu.Lock()
				outcomes = append(outcomes, outcome)
				outcomesMu.Unlock()
			}
		}()
	}

	for _, device := range devices {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			// Incomplete cycle: discard any partial outcomes without health evaluation.
			return
		case jobs <- device:
		}
	}
	close(jobs)
	wg.Wait()

	outcomesMu.Lock()
	collected := append([]health.PollOutcome(nil), outcomes...)
	outcomesMu.Unlock()
	if len(collected) != len(devices) {
		// Cancellation interrupted workers before every due device finished.
		return
	}

	events := p.health.ApplyBatch(cfg, collected)
	p.observeHealth(events)
	p.publishHealth(ctx, cfg, events)
}

func (p *Poller) publishHealth(ctx context.Context, cfg *config.Config, healthEvents []health.Event) {
	mode := telemetry.ModeFromConfig(cfg)
	telCtx := telemetry.Context{
		SiteID:         cfg.SiteID,
		CollectorID:    cfg.Collector.ID,
		ConfigRevision: config.ConfigRevision(cfg),
		EmittedAt:      time.Now().UTC(),
	}
	evs := telemetry.HealthEvents(mode, telCtx, cfg.SiteID, healthEvents)
	if len(evs) == 0 {
		return
	}
	publishCtx, cancel := context.WithTimeout(ctx, cfg.Publisher.Timeout)
	defer cancel()
	if err := p.pub.Publish(publishCtx, evs...); err != nil {
		p.log.Error("health publish failed", "err", err, "events", len(evs))
	}
}

func (p *Poller) observeHealth(events []health.Event) {
	for _, ev := range events {
		p.metrics.ObserveHealthTransition(string(ev.State), string(ev.Reason), string(ev.Transition))
		p.log.Info("health transition",
			"device_id", ev.DeviceID,
			"state", string(ev.State),
			"reason", string(ev.Reason),
			"transition", string(ev.Transition),
			"failure_count", ev.FailureCount,
			"previous_state", previousStateString(ev.PreviousState),
		)
	}
	snap := p.health.Snapshot()
	byState := map[string]float64{
		"healthy":  float64(snap.DevicesByState[health.StateHealthy]),
		"warning":  float64(snap.DevicesByState[health.StateWarning]),
		"critical": float64(snap.DevicesByState[health.StateCritical]),
		"unknown":  float64(snap.DevicesByState[health.StateUnknown]),
	}
	p.metrics.SetHealthSnapshot(byState, float64(snap.DependencyImpacted), float64(snap.PendingFailures))
}

func previousStateString(state *health.State) string {
	if state == nil {
		return ""
	}
	return string(*state)
}

func (p *Poller) pollDevice(ctx context.Context, cfg *config.Config, device config.DeviceConfig) health.PollOutcome {
	p.metrics.PollTotal.Inc()
	started := time.Now()
	defer func() {
		p.metrics.PollDuration.Observe(time.Since(started).Seconds())
	}()

	outcome, err := p.doPoll(ctx, cfg, device)
	// ObservedAt is captured after poll completion inside doPoll.
	if err != nil {
		class := classifyError(err)
		p.metrics.PollFailureTotal.WithLabelValues(device.ID, class).Inc()
		p.log.Error("device poll failed",
			"device_id", device.ID,
			"host", device.Host,
			"error_class", class,
			"err", err,
		)
		if outcome.ObservedAt.IsZero() {
			outcome.ObservedAt = time.Now().UTC()
		}
		outcome.DeviceID = device.ID
		outcome.Success = false
		outcome.TemperatureC = nil
		return outcome
	}
	p.metrics.PollSuccessTotal.Inc()
	return outcome
}

func (p *Poller) doPoll(ctx context.Context, cfg *config.Config, device config.DeviceConfig) (health.PollOutcome, error) {
	outcome := health.PollOutcome{DeviceID: device.ID}

	client, err := snmp.NewClient(device, device.EffectiveSNMP(cfg.SNMP))
	if err != nil {
		outcome.ObservedAt = time.Now().UTC()
		return outcome, err
	}
	if err := client.Connect(ctx); err != nil {
		outcome.ObservedAt = time.Now().UTC()
		return outcome, err
	}
	defer func() {
		if err := client.Close(); err != nil {
			p.log.Warn("close SNMP client", "device_id", device.ID, "err", err)
		}
	}()

	identityCtx, cancelIdentity := client.WithTimeout(ctx)
	identity, err := core.PollDevice(identityCtx, client)
	cancelIdentity()
	if err != nil {
		outcome.ObservedAt = time.Now().UTC()
		return outcome, err
	}

	interfaceCtx, cancelInterfaces := client.WithScaledTimeout(ctx, core.InterfacePollWalkBudget)
	ifaces, err := core.PollInterfaces(interfaceCtx, client)
	cancelInterfaces()
	if err != nil {
		outcome.ObservedAt = time.Now().UTC()
		return outcome, err
	}

	result := readings.NewDevicePollResult(
		cfg.SiteID,
		device.ID,
		device.Host,
		time.Now().UTC(),
		identity,
		ifaces,
	)

	profileCtx, cancelProfile := client.WithScaledTimeout(ctx, profileWalkBudget)
	p.enrichProfile(profileCtx, client, &result)
	cancelProfile()

	interfaceFilter, err := filter.New(device.InterfaceFilters)
	if err != nil {
		outcome.ObservedAt = time.Now().UTC()
		return outcome, fmt.Errorf("compile interface filter: %w", err)
	}
	interfaceFilter.Apply(&result)
	p.metrics.InterfaceSelectionTotal.WithLabelValues(string(readings.Selected)).Add(float64(result.Filter.Selected))
	p.metrics.InterfaceSelectionTotal.WithLabelValues(string(readings.ExcludedDefault)).Add(float64(result.Filter.ExcludedDefault))
	p.metrics.InterfaceSelectionTotal.WithLabelValues(string(readings.ExcludedRule)).Add(float64(result.Filter.ExcludedRule))

	evs := telemetry.DeviceEvents(telemetry.ModeFromConfig(cfg), telemetry.Context{
		SiteID:         cfg.SiteID,
		CollectorID:    cfg.Collector.ID,
		ConfigRevision: config.ConfigRevision(cfg),
		EmittedAt:      time.Now().UTC(),
	}, result)

	publishCtx, cancel := context.WithTimeout(ctx, cfg.Publisher.Timeout)
	defer cancel()

	err = p.pub.Publish(publishCtx, evs...)
	// Capture observation time after the poll pipeline completes, before returning.
	outcome.ObservedAt = time.Now().UTC()
	if err != nil {
		return outcome, err
	}

	outcome.Success = true
	outcome.TemperatureC = health.PrimaryTemperaturePtr(result.Vendor.Temperatures)
	return outcome, nil
}

// enrichProfile mutates only VendorReadings. Core identity/interfaces stay valid
// even when every vendor walk fails.
func (p *Poller) enrichProfile(ctx context.Context, client profile.Client, result *readings.DevicePollResult) {
	matched, kind := p.registry.Match(result.Identity.SysObjectID)
	if matched == nil {
		result.Vendor = readings.VendorReadings{Profile: "core"}
		p.metrics.ProfileFallbackTotal.Inc()
		p.metrics.ProfileDetectionTotal.WithLabelValues("core", string(profile.MatchCore)).Inc()
		p.log.Debug("no vendor profile matched",
			"device_id", result.DeviceID,
			"sys_object_id", result.Identity.SysObjectID,
			"sys_descr", result.Identity.SysDescr,
		)
		return
	}

	p.metrics.ProfileDetectionTotal.WithLabelValues(matched.Name(), string(kind)).Inc()
	started := time.Now()
	vendor, err := matched.Collect(ctx, client)
	p.metrics.ProfileDuration.Observe(time.Since(started).Seconds())
	if vendor.Profile == "" {
		vendor.Profile = matched.Name()
	}
	if vendor.Capabilities == 0 {
		vendor.Capabilities = matched.Capabilities()
	}
	result.Vendor = vendor
	if err != nil {
		class := classifyError(err)
		p.metrics.ProfileCollectionFailure.WithLabelValues(matched.Name(), class).Inc()
		p.log.Warn("profile collection failed; retaining core readings",
			"device_id", result.DeviceID,
			"profile", matched.Name(),
			"match_kind", string(kind),
			"error_class", class,
			"err", err,
		)
	}
}

func classifyError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return metrics.ErrorClassTimeout
	}
	// gosnmp wraps timeouts as plain errors containing "timeout".
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline") {
		return metrics.ErrorClassTimeout
	}
	if strings.Contains(msg, "parse") || strings.Contains(msg, "unexpected type") || strings.Contains(msg, "unavailable") {
		return metrics.ErrorClassParse
	}
	return metrics.ErrorClassSNMP
}
