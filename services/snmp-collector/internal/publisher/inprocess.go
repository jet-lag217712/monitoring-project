package publisher

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/buffer"
	"github.com/equate/ogsd/services/snmp-collector/internal/events"
)

// LocalConsumer is the transport-independent ingestion contract. Returning
// false retains an event in the durable spool for a later retry.
type LocalConsumer interface {
	Consume(context.Context, string, []byte) bool
}

// InProcessPublisher persists typed events before handing them to an in-process
// consumer. It is the appliance transport: no broker or network hop is used.
type InProcessPublisher struct {
	store       *buffer.Store
	consumer    LocalConsumer
	log         *slog.Logger
	batchSize   int
	idleBackoff time.Duration
}

// NewInProcessPublisher creates a durable in-process dispatcher.
func NewInProcessPublisher(store *buffer.Store, consumer LocalConsumer, log *slog.Logger, batchSize int, idleBackoff time.Duration) (*InProcessPublisher, error) {
	if store == nil {
		return nil, fmt.Errorf("buffer store is required")
	}
	if consumer == nil {
		return nil, fmt.Errorf("local consumer is required")
	}
	if log == nil {
		log = slog.Default()
	}
	if batchSize <= 0 {
		batchSize = 50
	}
	if idleBackoff <= 0 {
		idleBackoff = 500 * time.Millisecond
	}
	return &InProcessPublisher{store: store, consumer: consumer, log: log, batchSize: batchSize, idleBackoff: idleBackoff}, nil
}

// Publish atomically appends events to the SQLite/WAL spool before dispatch.
func (p *InProcessPublisher) Publish(ctx context.Context, evs ...events.Event) error {
	batch := make([]buffer.PendingEvent, 0, len(evs))
	for _, ev := range evs {
		if err := ctx.Err(); err != nil {
			return err
		}
		payload, err := json.Marshal(ev)
		if err != nil {
			return fmt.Errorf("marshal event: %w", err)
		}
		batch = append(batch, buffer.PendingEvent{Topic: ev.Topic(), Payload: payload})
	}
	return p.store.EnqueueBatch(ctx, batch)
}

// Run drains the spool in order. Rows are deleted only after Consume returns
// true, preserving at-least-once delivery through database outages or crashes.
func (p *InProcessPublisher) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-p.store.Wake():
		case <-time.After(p.idleBackoff):
		}

		if err := p.dispatchOnce(ctx); err != nil && ctx.Err() == nil {
			p.log.Warn("in-process dispatch deferred", "err", err)
		}
	}
}

func (p *InProcessPublisher) dispatchOnce(ctx context.Context) error {
	rows, err := p.store.PeekOldest(ctx, p.batchSize)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !p.consumer.Consume(ctx, row.Topic, row.Payload) {
			return fmt.Errorf("consumer did not acknowledge queued event %d", row.ID)
		}
		if err := p.store.Delete(ctx, row.ID); err != nil {
			return err
		}
	}
	return nil
}

// Depth returns the current durable event backlog.
func (p *InProcessPublisher) Depth() int64 {
	if p == nil || p.store == nil {
		return 0
	}
	return p.store.Depth()
}
