// Package ingestion exposes the in-process ingestion boundary used by the
// single-node appliance. Transport adapters remain outside this package.
package ingestion

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	internalhandler "github.com/equate/ogsd/services/ingestion-service/internal/handler"
	"github.com/equate/ogsd/services/ingestion-service/internal/metrics"
	"github.com/equate/ogsd/services/ingestion-service/internal/store"
)

// Config supplies the database settings required by the in-process consumer.
type Config struct {
	DatabaseURL string
	MaxConns    int32
	MinConns    int32
	MaxLifetime time.Duration
}

// Runtime consumes already-local events and persists them transactionally.
// An event is acknowledged only after it is accepted, deduplicated, or safely
// rejected. Database failures return false so a durable dispatcher retries it.
type Runtime struct {
	store   *store.Store
	handler *internalhandler.Handler
}

// Open creates an in-process ingestion runtime.
func Open(ctx context.Context, cfg Config, log *slog.Logger) (*Runtime, error) {
	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("database URL is required")
	}
	if cfg.MaxConns <= 0 {
		cfg.MaxConns = 10
	}
	if cfg.MinConns < 0 {
		return nil, fmt.Errorf("minimum database connections must be non-negative")
	}
	if cfg.MaxLifetime <= 0 {
		cfg.MaxLifetime = time.Hour
	}

	s, err := store.Open(ctx, cfg.DatabaseURL, cfg.MaxConns, cfg.MinConns, cfg.MaxLifetime)
	if err != nil {
		return nil, err
	}
	return &Runtime{
		store:   s,
		handler: internalhandler.New(s, metrics.New(), log),
	}, nil
}

// Consume processes an event from a local durable queue. A false result tells
// the caller to retain the event for retry; it never exposes transport details.
func (r *Runtime) Consume(ctx context.Context, route string, payload []byte) bool {
	if r == nil || r.handler == nil {
		return false
	}
	return r.handler.Handle(ctx, route, payload)
}

// Close releases database resources.
func (r *Runtime) Close() {
	if r != nil && r.store != nil {
		r.store.Close()
	}
}
