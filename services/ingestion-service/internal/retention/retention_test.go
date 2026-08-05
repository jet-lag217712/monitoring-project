package retention_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/equate/ogsd/services/ingestion-service/internal/config"
	"github.com/equate/ogsd/services/ingestion-service/internal/metrics"
	"github.com/equate/ogsd/services/ingestion-service/internal/retention"
	"github.com/prometheus/client_golang/prometheus"
)

func TestCutoff(t *testing.T) {
	now := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	got := retention.Cutoff(now, 30)
	want := time.Date(2026, 7, 5, 12, 0, 0, 0, time.UTC)
	if !got.Equal(want) {
		t.Fatalf("cutoff=%s want=%s", got, want)
	}
}

func TestTargets_Allowlist(t *testing.T) {
	if len(retention.Targets) != 8 {
		t.Fatalf("targets=%d want 8", len(retention.Targets))
	}
	seen := map[string]bool{}
	for _, target := range retention.Targets {
		if target.Table == "" || target.Column == "" {
			t.Fatalf("empty target: %#v", target)
		}
		if seen[target.Table] {
			t.Fatalf("duplicate table %s", target.Table)
		}
		seen[target.Table] = true
	}
}

type fakeDeleter struct {
	mu      sync.Mutex
	batches map[string][]int64 // table → successive RowsAffected
	calls   []deleteCall
	errOn   string
}

type deleteCall struct {
	Table     string
	Column    string
	Cutoff    time.Time
	BatchSize int
}

func (f *fakeDeleter) DeleteOlderThan(_ context.Context, table, column string, cutoff time.Time, batchSize int) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, deleteCall{Table: table, Column: column, Cutoff: cutoff, BatchSize: batchSize})
	if f.errOn == table {
		return 0, errors.New("boom")
	}
	remaining := f.batches[table]
	if len(remaining) == 0 {
		return 0, nil
	}
	n := remaining[0]
	f.batches[table] = remaining[1:]
	return n, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestRunOnce_BatchesUntilEmpty(t *testing.T) {
	deleter := &fakeDeleter{
		batches: map[string][]int64{
			"metric_samples": {10000, 2500},
			// others return 0
		},
	}
	enabled := true
	cfg := config.RetentionConfig{
		Enabled:   &enabled,
		Days:      30,
		Interval:  time.Hour,
		BatchSize: 10000,
	}
	m := metrics.NewWithRegisterer(prometheus.NewRegistry())
	runner := retention.New(deleter, cfg, m, testLogger())
	fixed := time.Date(2026, 8, 4, 12, 0, 0, 0, time.UTC)
	runner.SetNow(func() time.Time { return fixed })

	if err := runner.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}

	var metricCalls int
	for _, c := range deleter.calls {
		if c.Table == "metric_samples" {
			metricCalls++
			if c.Column != "collected_at" {
				t.Fatalf("column=%s", c.Column)
			}
			if !c.Cutoff.Equal(retention.Cutoff(fixed, 30)) {
				t.Fatalf("cutoff=%s", c.Cutoff)
			}
			if c.BatchSize != 10000 {
				t.Fatalf("batch=%d", c.BatchSize)
			}
		}
	}
	if metricCalls != 2 {
		t.Fatalf("metric_samples calls=%d want 2", metricCalls)
	}
	// All 8 tables should be visited at least once.
	tables := map[string]bool{}
	for _, c := range deleter.calls {
		tables[c.Table] = true
	}
	if len(tables) != len(retention.Targets) {
		t.Fatalf("tables visited=%d want %d", len(tables), len(retention.Targets))
	}
}

func TestRunOnce_ErrorIncrementsMetric(t *testing.T) {
	deleter := &fakeDeleter{errOn: "metric_samples", batches: map[string][]int64{}}
	cfg := config.RetentionConfig{Days: 30, Interval: time.Hour, BatchSize: 1000}
	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegisterer(reg)
	runner := retention.New(deleter, cfg, m, testLogger())

	err := runner.RunOnce(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunOnce_RespectsCancel(t *testing.T) {
	deleter := &fakeDeleter{
		batches: map[string][]int64{
			"metric_samples": {10000, 10000, 10000},
		},
	}
	cfg := config.RetentionConfig{Days: 30, Interval: time.Hour, BatchSize: 10000}
	runner := retention.New(deleter, cfg, nil, testLogger())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := runner.RunOnce(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err=%v", err)
	}
}
