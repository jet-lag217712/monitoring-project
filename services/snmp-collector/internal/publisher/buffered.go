package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/buffer"
	"github.com/equate/ogsd/services/snmp-collector/internal/events"
	"github.com/equate/ogsd/services/snmp-collector/internal/metrics"
)

// mqttTransport is the MQTT surface used by the buffered flusher.
type mqttTransport interface {
	IsConnected() bool
	AwaitConnection(ctx context.Context) error
	Publish(ctx context.Context, topic string, payload []byte) error
}

// BufferedPublisher persists events to SQLite and flushes them over MQTT.
type BufferedPublisher struct {
	store       *buffer.Store
	mqtt        mqttTransport
	metrics     *metrics.Collector
	log         *slog.Logger
	batchSize   int
	idleBackoff time.Duration
}

// NewBufferedPublisher creates a publisher that enqueues locally and flushes in the background.
func NewBufferedPublisher(store *buffer.Store, mqtt mqttTransport, m *metrics.Collector, log *slog.Logger, batchSize int, idleBackoff time.Duration) *BufferedPublisher {
	if log == nil {
		log = slog.Default()
	}
	if batchSize <= 0 {
		batchSize = 50
	}
	if idleBackoff <= 0 {
		idleBackoff = 500 * time.Millisecond
	}
	return &BufferedPublisher{
		store:       store,
		mqtt:        mqtt,
		metrics:     m,
		log:         log,
		batchSize:   batchSize,
		idleBackoff: idleBackoff,
	}
}

// Publish marshals events and enqueues them for durable delivery.
func (p *BufferedPublisher) Publish(ctx context.Context, evs ...events.Event) error {
	for _, ev := range evs {
		if err := ctx.Err(); err != nil {
			return err
		}
		payload, err := json.Marshal(ev)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		if err := p.store.Enqueue(ctx, ev.Topic(), payload); err != nil {
			return err
		}
	}
	return nil
}

// RunFlusher drains the buffer to MQTT until ctx is cancelled.
func (p *BufferedPublisher) RunFlusher(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.store.Wake():
		case <-time.After(p.idleBackoff):
		}

		if !p.mqtt.IsConnected() {
			awaitCtx, cancel := context.WithTimeout(ctx, p.idleBackoff)
			_ = p.mqtt.AwaitConnection(awaitCtx)
			cancel()
			if !p.mqtt.IsConnected() {
				continue
			}
		}

		if err := p.flushOnce(ctx); err != nil {
			p.log.Error("buffer flush failed", "err", err)
		}
	}
}

// Drain attempts to flush remaining buffered messages until empty or ctx expires.
func (p *BufferedPublisher) Drain(ctx context.Context) error {
	for p.store.Depth() > 0 {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("drain incomplete, depth=%d: %w", p.store.Depth(), err)
		}
		if !p.mqtt.IsConnected() {
			if err := p.mqtt.AwaitConnection(ctx); err != nil {
				return fmt.Errorf("drain await connection, depth=%d: %w", p.store.Depth(), err)
			}
		}
		if err := p.flushOnce(ctx); err != nil {
			return err
		}
		if p.store.Depth() > 0 {
			select {
			case <-ctx.Done():
				return fmt.Errorf("drain incomplete, depth=%d: %w", p.store.Depth(), ctx.Err())
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
	return nil
}

func (p *BufferedPublisher) flushOnce(ctx context.Context) error {
	rows, err := p.store.PeekOldest(ctx, p.batchSize)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	published := 0
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := p.mqtt.Publish(ctx, row.Topic, row.Payload); err != nil {
			p.metrics.MQTTPublishFailure.Inc()
			return err
		}
		if err := p.store.Delete(ctx, row.ID); err != nil {
			return err
		}
		p.metrics.MQTTPublishTotal.Inc()
		p.metrics.BufferFlushedMessagesTotal.Inc()
		published++
	}
	if published > 0 {
		p.metrics.BufferFlushBatchesTotal.Inc()
	}
	return nil
}
