package publisher

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/buffer"
	"github.com/equate/ogsd/services/snmp-collector/internal/events"
	"github.com/equate/ogsd/services/snmp-collector/internal/metrics"
)

type toggleConsumer struct {
	ack   atomic.Bool
	calls atomic.Int64
}

func (c *toggleConsumer) Consume(_ context.Context, _ string, _ []byte) bool {
	c.calls.Add(1)
	return c.ack.Load()
}

func TestInProcessPublisherRetainsEventsUntilConsumerAcknowledges(t *testing.T) {
	m := metrics.New()
	store, err := buffer.Open(buffer.Options{
		Path:    t.TempDir() + "/events.db",
		Metrics: m,
	})
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close() //nolint:errcheck

	consumer := &toggleConsumer{}
	pub, err := NewInProcessPublisher(store, consumer, nil, 10, time.Millisecond)
	if err != nil {
		t.Fatalf("new publisher: %v", err)
	}
	if err := pub.Publish(context.Background(), events.DeviceMetricEvent{
		SiteID: "site-001", DeviceID: "device-001", Timestamp: time.Now().UTC(), Metric: "uptime_seconds", Value: 1,
	}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go pub.Run(ctx)

	deadline := time.Now().Add(time.Second)
	for consumer.calls.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := store.Depth(); got != 1 {
		t.Fatalf("depth after failed consume = %d, want 1", got)
	}

	consumer.ack.Store(true)
	for store.Depth() != 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := store.Depth(); got != 0 {
		t.Fatalf("depth after acknowledged consume = %d, want 0", got)
	}
}
