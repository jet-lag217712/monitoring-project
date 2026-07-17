package heartbeat

import (
	"context"
	"log/slog"
	"os"
	"runtime"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/metrics"
	"github.com/equate/ogsd/services/snmp-collector/internal/publisher"
	"github.com/equate/ogsd/services/snmp-collector/internal/telemetry"
)

// BuildInfo is release metadata injected at link time.
type BuildInfo struct {
	Version   string
	GitCommit string
	BuildTime string
}

// DepthFunc samples the durable outbox depth before enqueueing a heartbeat.
type DepthFunc func() (int64, error)

// ConfigSource supplies the active collector configuration.
type ConfigSource interface {
	Current() *config.Config
}

// Runner publishes startup and periodic collector heartbeats.
type Runner struct {
	source    ConfigSource
	pub       publisher.Publisher
	metrics   *metrics.Collector
	log       *slog.Logger
	depth     DepthFunc
	build     BuildInfo
	hostname  string
	startedAt time.Time
}

// New creates a heartbeat runner.
func New(source ConfigSource, pub publisher.Publisher, m *metrics.Collector, log *slog.Logger, depth DepthFunc, build BuildInfo) *Runner {
	if log == nil {
		log = slog.Default()
	}
	if depth == nil {
		depth = func() (int64, error) { return 0, nil }
	}
	hostname, err := os.Hostname()
	if err != nil || hostname == "" {
		hostname = "unknown"
	}
	return &Runner{
		source:    source,
		pub:       pub,
		metrics:   m,
		log:       log,
		depth:     depth,
		build:     build,
		hostname:  hostname,
		startedAt: time.Now().UTC(),
	}
}

// Run publishes an initial heartbeat then periodic heartbeats until ctx is done.
func (r *Runner) Run(ctx context.Context) {
	r.publishOnce(ctx)
	for {
		cfg := r.source.Current()
		interval := defaultInterval(cfg)
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
			r.publishOnce(ctx)
		}
	}
}

func defaultInterval(cfg *config.Config) time.Duration {
	if cfg == nil || cfg.Collector.HeartbeatInterval <= 0 {
		return 60 * time.Second
	}
	return cfg.Collector.HeartbeatInterval
}

func (r *Runner) publishOnce(ctx context.Context) {
	cfg := r.source.Current()
	if cfg == nil {
		return
	}
	mode := telemetry.ModeFromConfig(cfg)
	if !telemetry.ShouldPublishHeartbeat(mode) {
		return
	}

	started := time.Now()
	depth, err := r.depth()
	if err != nil {
		r.metrics.HeartbeatPublishFailure.Inc()
		r.log.Error("heartbeat depth sample failed", "err", err)
		return
	}

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	now := time.Now().UTC()
	ev := telemetry.Heartbeat(telemetry.Context{
		SiteID:         cfg.SiteID,
		CollectorID:    cfg.Collector.ID,
		ConfigRevision: config.ConfigRevision(cfg),
		EmittedAt:      now,
	}, telemetry.HeartbeatInput{
		Hostname:         r.hostname,
		Version:          r.build.Version,
		GitCommit:        r.build.GitCommit,
		BuildTime:        r.build.BuildTime,
		UptimeSeconds:    int64(now.Sub(r.startedAt).Seconds()),
		SQLiteQueueDepth: depth,
		MemoryUsageBytes: int64(memStats.Alloc),
		GoroutineCount:   runtime.NumGoroutine(),
		ObservedAt:       now,
	})

	publishCtx, cancel := context.WithTimeout(ctx, cfg.Publisher.Timeout)
	defer cancel()
	if err := r.pub.Publish(publishCtx, ev); err != nil {
		r.metrics.HeartbeatPublishFailure.Inc()
		r.metrics.HeartbeatDuration.Observe(time.Since(started).Seconds())
		r.log.Error("heartbeat publish failed", "err", err)
		return
	}
	r.metrics.HeartbeatPublishTotal.Inc()
	r.metrics.HeartbeatDuration.Observe(time.Since(started).Seconds())
	r.log.Info("heartbeat published",
		"collector_id", cfg.Collector.ID,
		"sqlite_queue_depth", depth,
		"uptime_seconds", int64(now.Sub(r.startedAt).Seconds()),
	)
}
