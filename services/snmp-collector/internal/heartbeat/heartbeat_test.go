package heartbeat_test

import (
	"context"
	"io"
	"log/slog"
	"sync"
	"testing"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/events"
	"github.com/equate/ogsd/services/snmp-collector/internal/heartbeat"
	"github.com/equate/ogsd/services/snmp-collector/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type staticSource struct{ cfg *config.Config }

func (s staticSource) Current() *config.Config { return s.cfg }

type capturePublisher struct {
	mu   sync.Mutex
	evs  []events.Event
	depthBefore int64
}

func (p *capturePublisher) Publish(_ context.Context, evs ...events.Event) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.evs = append(p.evs, evs...)
	return nil
}

func TestRunner_SamplesDepthBeforeEnqueue(t *testing.T) {
	cfg := &config.Config{
		SiteID: "site-001",
		Collector: config.CollectorConfig{
			ID:                "collector-1",
			HeartbeatInterval: time.Hour,
		},
		Publisher: config.PublisherConfig{
			Mode:             "stdout",
			Timeout:          time.Second,
			TelemetryVersion: "v2",
		},
	}
	pub := &capturePublisher{}
	var sampled bool
	depth := func() (int64, error) {
		sampled = true
		return 7, nil
	}
	m := metrics.NewWithRegisterer(prometheus.NewRegistry())
	r := heartbeat.New(staticSource{cfg}, pub, m, slog.New(slog.NewTextHandler(io.Discard, nil)), depth, heartbeat.BuildInfo{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go r.Run(ctx)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		pub.mu.Lock()
		n := len(pub.evs)
		pub.mu.Unlock()
		if n > 0 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	cancel()
	if !sampled {
		t.Fatal("expected depth sample before publish")
	}
	pub.mu.Lock()
	defer pub.mu.Unlock()
	if len(pub.evs) == 0 {
		t.Fatal("expected heartbeat event")
	}
	hb, ok := pub.evs[0].(events.HeartbeatEvent)
	if !ok {
		t.Fatalf("got %T", pub.evs[0])
	}
	if hb.Payload.SQLiteQueueDepth != 7 {
		t.Fatalf("depth=%d", hb.Payload.SQLiteQueueDepth)
	}
}

func TestRunner_SkipsV1Mode(t *testing.T) {
	cfg := &config.Config{
		SiteID: "site-001",
		Collector: config.CollectorConfig{
			ID:                "collector-1",
			HeartbeatInterval: time.Hour,
		},
		Publisher: config.PublisherConfig{
			Mode:             "stdout",
			Timeout:          time.Second,
			TelemetryVersion: "v1",
		},
	}
	pub := &capturePublisher{}
	m := metrics.NewWithRegisterer(prometheus.NewRegistry())
	r := heartbeat.New(staticSource{cfg}, pub, m, slog.New(slog.NewTextHandler(io.Discard, nil)), nil, heartbeat.BuildInfo{})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	r.Run(ctx)
	pub.mu.Lock()
	defer pub.mu.Unlock()
	if len(pub.evs) != 0 {
		t.Fatalf("v1 mode should not publish heartbeat, got %d", len(pub.evs))
	}
}
