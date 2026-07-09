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

// Poller schedules bounded concurrent device polls.
type Poller struct {
	cfg     *config.Config
	pub     publisher.Publisher
	metrics *metrics.Collector
	log     *slog.Logger
}

// New creates a Poller.
func New(cfg *config.Config, pub publisher.Publisher, m *metrics.Collector, log *slog.Logger) *Poller {
	if log == nil {
		log = slog.Default()
	}
	return &Poller{cfg: cfg, pub: pub, metrics: m, log: log}
}

// Run polls immediately, then on each poll_interval until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) {
	p.pollAll(ctx)

	ticker := time.NewTicker(p.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.pollAll(ctx)
		}
	}
}

func (p *Poller) pollAll(ctx context.Context) {
	sem := make(chan struct{}, p.cfg.MaxWorkers)
	var wg sync.WaitGroup

	for _, device := range p.cfg.Devices {
		if ctx.Err() != nil {
			break
		}
		device := device
		wg.Add(1)
		sem <- struct{}{}
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			p.pollDevice(ctx, device)
		}()
	}
	wg.Wait()
}

func (p *Poller) pollDevice(ctx context.Context, device config.DeviceConfig) {
	p.metrics.PollTotal.Inc()

	err := p.doPoll(ctx, device)
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

func (p *Poller) doPoll(ctx context.Context, device config.DeviceConfig) error {
	client, err := snmp.NewClient(device, p.cfg.SNMP)
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

	// vendor field is reserved for Phase 2+; empty means core-only.
	_ = device.Vendor

	evs := normalize.ToEvents(normalize.DeviceReading{
		SiteID:        p.cfg.SiteID,
		DeviceID:      device.ID,
		Timestamp:     time.Now().UTC(),
		UptimeSeconds: uptime,
		Interfaces:    ifaces,
	})

	publishCtx, cancel := context.WithTimeout(ctx, p.cfg.Publisher.Timeout)
	defer cancel()

	if err := p.pub.Publish(publishCtx, evs...); err != nil {
		return err
	}

	p.metrics.BufferEnqueueTotal.Add(float64(len(evs)))
	p.metrics.BufferFlushTotal.Inc()
	p.metrics.BufferDepth.Set(0)
	return nil
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
