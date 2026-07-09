package buffer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/equate/ogsd/services/snmp-collector/internal/metrics"
	_ "modernc.org/sqlite"
)

// ErrBufferFull is returned when Enqueue would exceed MaxEntries.
var ErrBufferFull = errors.New("buffer full")

// Row is a buffered telemetry event awaiting MQTT delivery.
type Row struct {
	ID      int64
	Topic   string
	Payload []byte
}

// Options configures a Store.
type Options struct {
	Path          string
	MaxEntries    int // 0 = unlimited
	BusyTimeoutMS int
	Metrics       *metrics.Collector
}

// Store is a durable FIFO buffer backed by SQLite.
type Store struct {
	db      *sql.DB
	mu      sync.Mutex
	depth   atomic.Int64
	max     int
	wake    chan struct{}
	metrics *metrics.Collector
}

// Open creates or opens a SQLite buffer database.
func Open(opts Options) (*Store, error) {
	if opts.Path == "" {
		return nil, fmt.Errorf("buffer path is required")
	}
	if opts.BusyTimeoutMS <= 0 {
		opts.BusyTimeoutMS = 5000
	}
	if opts.Metrics == nil {
		return nil, fmt.Errorf("metrics collector is required")
	}

	dir := filepath.Dir(opts.Path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create buffer dir: %w", err)
		}
	}

	db, err := sql.Open("sqlite", opts.Path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	// Single writer; serialize via Store.mu for depth consistency.
	db.SetMaxOpenConns(1)

	s := &Store{
		db:      db,
		max:     opts.MaxEntries,
		wake:    make(chan struct{}, 1),
		metrics: opts.Metrics,
	}

	if err := s.init(opts.BusyTimeoutMS); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) init(busyTimeoutMS int) error {
	if _, err := s.db.Exec(fmt.Sprintf("PRAGMA busy_timeout=%d", busyTimeoutMS)); err != nil {
		return fmt.Errorf("pragma busy_timeout: %w", err)
	}
	if _, err := s.db.Exec(`PRAGMA journal_mode=WAL`); err != nil {
		return fmt.Errorf("pragma journal_mode: %w", err)
	}
	if _, err := s.db.Exec(`
CREATE TABLE IF NOT EXISTS pending_events (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    topic      TEXT NOT NULL,
    payload    BLOB NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX IF NOT EXISTS idx_pending_events_id ON pending_events (id ASC);
`); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}

	var n int64
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM pending_events`).Scan(&n); err != nil {
		return fmt.Errorf("bootstrap depth: %w", err)
	}
	s.depth.Store(n)
	s.metrics.SetBufferDepth(n)
	return nil
}

// Depth returns the in-memory buffer depth (O(1)).
func (s *Store) Depth() int64 {
	return s.depth.Load()
}

// Wake returns a coalesced signal channel that fires when new rows are enqueued.
func (s *Store) Wake() <-chan struct{} {
	return s.wake
}

// Enqueue persists a telemetry event and wakes the flusher.
func (s *Store) Enqueue(ctx context.Context, topic string, payload []byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if topic == "" {
		return fmt.Errorf("topic is required")
	}
	if len(payload) == 0 {
		return fmt.Errorf("payload is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.max > 0 && s.depth.Load() >= int64(s.max) {
		return ErrBufferFull
	}

	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO pending_events (topic, payload) VALUES (?, ?)`,
		topic, payload,
	); err != nil {
		return fmt.Errorf("insert pending event: %w", err)
	}

	n := s.depth.Add(1)
	s.metrics.SetBufferDepth(n)
	s.metrics.BufferEnqueueTotal.Inc()
	s.signal()
	return nil
}

// PeekOldest returns up to limit oldest buffered rows without removing them.
func (s *Store) PeekOldest(ctx context.Context, limit int) ([]Row, error) {
	if limit <= 0 {
		return nil, nil
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, topic, payload FROM pending_events ORDER BY id ASC LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("peek oldest: %w", err)
	}
	defer rows.Close()

	out := make([]Row, 0, limit)
	for rows.Next() {
		var r Row
		if err := rows.Scan(&r.ID, &r.Topic, &r.Payload); err != nil {
			return nil, fmt.Errorf("scan pending event: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate pending events: %w", err)
	}
	return out, nil
}

// Delete removes a row after successful MQTT delivery.
func (s *Store) Delete(ctx context.Context, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.ExecContext(ctx, `DELETE FROM pending_events WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete pending event: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return nil
	}
	depth := s.depth.Add(-1)
	if depth < 0 {
		depth = 0
		s.depth.Store(0)
	}
	s.metrics.SetBufferDepth(depth)
	return nil
}

// Close closes the underlying database.
func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) signal() {
	select {
	case s.wake <- struct{}{}:
	default:
	}
}
