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
	"github.com/equate/ogsd/services/snmp-collector/internal/metrics"
	"github.com/equate/ogsd/services/snmp-collector/internal/normalize"
	"github.com/equate/ogsd/services/snmp-collector/internal/publisher"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/core"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/filter"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/profile"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/readings"
	"github.com/equate/ogsd/services/snmp-collector/internal/snmp/vendors"
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

// Poller schedules bounded concurrent device polls.
type Poller struct {
	source   ConfigSource
	pub      publisher.Publisher
	metrics  *metrics.Collector
	log      *slog.Logger
	registry *profile.Registry
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
		source:   source,
		pub:      pub,
		metrics:  m,
		log:      log,
		registry: vendors.NewRegistry(),
	}
}

// Run polls due devices until ctx is cancelled. Each device has an independent
// schedule, while all work remains bounded by the active global worker limit.
func (p *Poller) Run(ctx context.Context) {
	nextDue := make(map[string]time.Time)
	firstCycle := true

	for {
		cfg := p.source.Current()
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

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case device, ok := <-jobs:
					if !ok {
						return
					}
					p.pollDevice(ctx, cfg, device)
				}
			}
		}()
	}

	for _, device := range devices {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return
		case jobs <- device:
		}
	}
	close(jobs)
	wg.Wait()
}

func (p *Poller) pollDevice(ctx context.Context, cfg *config.Config, device config.DeviceConfig) {
	p.metrics.PollTotal.Inc()
	started := time.Now()
	defer func() {
		p.metrics.PollDuration.Observe(time.Since(started).Seconds())
	}()

	err := p.doPoll(ctx, cfg, device)
	if err != nil {
		class := classifyError(err)
		p.metrics.PollFailureTotal.WithLabelValues(device.ID, class).Inc()
		p.log.Error("device poll failed",
			"device_id", device.ID,
			"host", device.Host,
			"error_class", class,
			"err", err,
		)
		return
	}
	p.metrics.PollSuccessTotal.Inc()
}

func (p *Poller) doPoll(ctx context.Context, cfg *config.Config, device config.DeviceConfig) error {
	client, err := snmp.NewClient(device, device.EffectiveSNMP(cfg.SNMP))
	if err != nil {
		return err
	}
	if err := client.Connect(ctx); err != nil {
		return err
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
		return err
	}

	interfaceCtx, cancelInterfaces := client.WithScaledTimeout(ctx, core.InterfacePollWalkBudget)
	ifaces, err := core.PollInterfaces(interfaceCtx, client)
	cancelInterfaces()
	if err != nil {
		return err
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
		return fmt.Errorf("compile interface filter: %w", err)
	}
	interfaceFilter.Apply(&result)
	p.metrics.InterfaceSelectionTotal.WithLabelValues(string(readings.Selected)).Add(float64(result.Filter.Selected))
	p.metrics.InterfaceSelectionTotal.WithLabelValues(string(readings.ExcludedDefault)).Add(float64(result.Filter.ExcludedDefault))
	p.metrics.InterfaceSelectionTotal.WithLabelValues(string(readings.ExcludedRule)).Add(float64(result.Filter.ExcludedRule))

	evs := normalize.ToEvents(result)

	publishCtx, cancel := context.WithTimeout(ctx, cfg.Publisher.Timeout)
	defer cancel()

	return p.pub.Publish(publishCtx, evs...)
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
