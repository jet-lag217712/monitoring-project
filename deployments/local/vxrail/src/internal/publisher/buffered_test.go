package publisher

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/buffer"
	"github.com/equate/ogsd/services/snmp-collector/internal/events"
	"github.com/equate/ogsd/services/snmp-collector/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

type fakeMQTT struct {
	mu        sync.Mutex
	connected bool
	failOnce  bool
	published []struct {
		topic   string
		payload []byte
	}
}

func (f *fakeMQTT) IsConnected() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.connected
}

func (f *fakeMQTT) AwaitConnection(ctx context.Context) error {
	deadline := time.NewTimer(50 * time.Millisecond)
	defer deadline.Stop()
	for {
		if f.IsConnected() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline.C:
			return context.DeadlineExceeded
		case <-time.After(5 * time.Millisecond):
		}
	}
}

func (f *fakeMQTT) Publish(_ context.Context, topic string, payload []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failOnce {
		f.failOnce = false
		return context.DeadlineExceeded
	}
	cp := make([]byte, len(payload))
	copy(cp, payload)
	f.published = append(f.published, struct {
		topic   string
		payload []byte
	}{topic: topic, payload: cp})
	return nil
}

func (f *fakeMQTT) setConnected(v bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.connected = v
}

func (f *fakeMQTT) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.published)
}

func TestBufferedPublisherFlushAndMetrics(t *testing.T) {
	m := metrics.NewWithRegisterer(prometheus.NewRegistry())
	store, err := buffer.Open(buffer.Options{
		Path:          filepath.Join(t.TempDir(), "buffer.db"),
		MaxEntries:    100,
		BusyTimeoutMS: 5000,
		Metrics:       m,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	mqtt := &fakeMQTT{connected: true}
	bp := NewBufferedPublisher(store, mqtt, m, nil, 10, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bp.RunFlusher(ctx)

	ev := events.DeviceMetricEvent{
		SiteID:    "site-001",
		DeviceID:  "dev-001",
		Timestamp: time.Now().UTC(),
		Metric:    "uptime_seconds",
		Value:     42,
	}
	if err := bp.Publish(context.Background(), ev, ev); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if mqtt.count() == 2 && store.Depth() == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("published=%d depth=%d", mqtt.count(), store.Depth())
}

func TestBufferedPublisherRetriesAfterPublishFailure(t *testing.T) {
	m := metrics.NewWithRegisterer(prometheus.NewRegistry())
	store, err := buffer.Open(buffer.Options{
		Path:          filepath.Join(t.TempDir(), "buffer.db"),
		MaxEntries:    100,
		BusyTimeoutMS: 5000,
		Metrics:       m,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	mqtt := &fakeMQTT{connected: true, failOnce: true}
	bp := NewBufferedPublisher(store, mqtt, m, nil, 10, 50*time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go bp.RunFlusher(ctx)

	ev := events.DeviceMetricEvent{
		SiteID:    "site-001",
		DeviceID:  "dev-001",
		Timestamp: time.Now().UTC(),
		Metric:    "uptime_seconds",
		Value:     1,
	}
	if err := bp.Publish(context.Background(), ev); err != nil {
		t.Fatal(err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if mqtt.count() == 1 && store.Depth() == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("published=%d depth=%d", mqtt.count(), store.Depth())
}
