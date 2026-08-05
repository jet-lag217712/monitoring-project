package retention

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/equate/ogsd/services/ingestion-service/internal/config"
	"github.com/equate/ogsd/services/ingestion-service/internal/metrics"
)

// Target is a table/column pair pruned by retention.
type Target struct {
	Table  string
	Column string
}

// Targets are the allowlisted history tables pruned by the retention job.
var Targets = []Target{
	{Table: "metric_samples", Column: "collected_at"},
	{Table: "interface_samples", Column: "collected_at"},
	{Table: "device_temperature_readings", Column: "observed_at"},
	{Table: "device_power_readings", Column: "observed_at"},
	{Table: "device_health_history", Column: "observed_at"},
	{Table: "collector_heartbeat_history", Column: "observed_at"},
	{Table: "ingested_events", Column: "observed_at"},
	{Table: "alerts", Column: "created_at"},
}

// Deleter removes rows older than a cutoff in batches.
type Deleter interface {
	DeleteOlderThan(ctx context.Context, table, column string, cutoff time.Time, batchSize int) (int64, error)
}

// Runner periodically prunes retained history.
type Runner struct {
	deleter Deleter
	cfg     config.RetentionConfig
	metrics *metrics.Ingestion
	log     *slog.Logger
	now     func() time.Time
}

// New creates a retention runner.
func New(deleter Deleter, cfg config.RetentionConfig, m *metrics.Ingestion, log *slog.Logger) *Runner {
	return &Runner{
		deleter: deleter,
		cfg:     cfg,
		metrics: m,
		log:     log,
		now:     time.Now,
	}
}

// SetNow overrides the clock used for cutoff calculation (tests).
func (r *Runner) SetNow(fn func() time.Time) {
	if fn != nil {
		r.now = fn
	}
}

// Cutoff returns the timestamp before which rows should be deleted.
func Cutoff(now time.Time, days int) time.Time {
	return now.UTC().AddDate(0, 0, -days)
}

// Run starts the retention loop until ctx is cancelled. It runs once immediately,
// then on each interval tick.
func (r *Runner) Run(ctx context.Context) {
	if err := r.RunOnce(ctx); err != nil && ctx.Err() == nil {
		r.log.Error("retention run failed", "err", err)
	}

	ticker := time.NewTicker(r.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := r.RunOnce(ctx); err != nil && ctx.Err() == nil {
				r.log.Error("retention run failed", "err", err)
			}
		}
	}
}

// RunOnce prunes all target tables in batches.
func (r *Runner) RunOnce(ctx context.Context) error {
	start := r.now()
	cutoff := Cutoff(start, r.cfg.Days)
	r.log.Info("retention starting", "cutoff", cutoff, "days", r.cfg.Days, "batch_size", r.cfg.BatchSize)

	var total int64
	for _, target := range Targets {
		if err := ctx.Err(); err != nil {
			return err
		}
		deleted, err := r.pruneTable(ctx, target, cutoff)
		if err != nil {
			if r.metrics != nil {
				r.metrics.RetentionErrors.Inc()
			}
			return fmt.Errorf("prune %s: %w", target.Table, err)
		}
		total += deleted
		if deleted > 0 {
			r.log.Info("retention pruned table", "table", target.Table, "deleted", deleted)
		}
	}

	elapsed := r.now().Sub(start).Seconds()
	if r.metrics != nil {
		r.metrics.RetentionRunDuration.Observe(elapsed)
	}
	r.log.Info("retention finished", "deleted", total, "duration_seconds", elapsed)
	return nil
}

func (r *Runner) pruneTable(ctx context.Context, target Target, cutoff time.Time) (int64, error) {
	var total int64
	for {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		n, err := r.deleter.DeleteOlderThan(ctx, target.Table, target.Column, cutoff, r.cfg.BatchSize)
		if err != nil {
			return total, err
		}
		if n == 0 {
			return total, nil
		}
		total += n
		if r.metrics != nil {
			r.metrics.RetentionDeleted.WithLabelValues(target.Table).Add(float64(n))
		}
		if n < int64(r.cfg.BatchSize) {
			return total, nil
		}
	}
}
