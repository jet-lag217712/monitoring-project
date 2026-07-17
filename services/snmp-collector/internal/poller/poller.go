package poller

import (
	"context"
	"errors"
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
)

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
	source  ConfigSource
	pub     publisher.Publisher
	metrics *metrics.Collector
	log     *slog.Logger
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
	return &Poller{source: source, pub: pub, metrics: m, log: log}
}

// Run polls due devices until ctx is cancelled. Each device has an independent
// schedule, while all work remains bounded by the active global worker limit.
func (p *Poller) Run(ctx context.Context) {
	nextDue := make(map[string]time.Time)
	firstCycle := true

	for {
		cfg := p.source.Current()
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
	defer client.Close()

	pollCtx, cancel := client.WithTimeout(ctx)
	defer cancel()

	uptime, err := core.PollDevice(pollCtx, client)
	if err != nil {
		return err
	}

	ifaces, err := core.PollInterfaces(pollCtx, client)
	if err != nil {
		return err
	}

	evs := normalize.ToEvents(normalize.DeviceReading{
		SiteID:        cfg.SiteID,
		DeviceID:      device.ID,
		IPAddress:     device.Host,
		Timestamp:     time.Now().UTC(),
		UptimeSeconds: uptime,
		Interfaces:    ifaces,
	})

	publishCtx, cancel := context.WithTimeout(ctx, cfg.Publisher.Timeout)
	defer cancel()

	return p.pub.Publish(publishCtx, evs...)
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
