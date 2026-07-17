package buffer

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/equate/ogsd/services/snmp-collector/internal/metrics"
	"github.com/prometheus/client_golang/prometheus"
)

func testMetrics(t *testing.T) *metrics.Collector {
	t.Helper()
	return metrics.NewWithRegisterer(prometheus.NewRegistry())
}

func openTestStore(t *testing.T, maxEntries int) *Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "buffer.db")
	s, err := Open(Options{
		Path:          path,
		MaxEntries:    maxEntries,
		BusyTimeoutMS: 5000,
		Metrics:       testMetrics(t),
	})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestEnqueuePeekDeleteFIFO(t *testing.T) {
	s := openTestStore(t, 0)
	ctx := context.Background()

	if err := s.Enqueue(ctx, "topic/a", []byte(`{"n":1}`)); err != nil {
		t.Fatal(err)
	}
	if err := s.Enqueue(ctx, "topic/b", []byte(`{"n":2}`)); err != nil {
		t.Fatal(err)
	}
	if s.Depth() != 2 {
		t.Fatalf("depth=%d, want 2", s.Depth())
	}

	rows, err := s.PeekOldest(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("len=%d", len(rows))
	}
	if rows[0].Topic != "topic/a" || string(rows[0].Payload) != `{"n":1}` {
		t.Fatalf("first=%+v", rows[0])
	}
	if rows[1].Topic != "topic/b" {
		t.Fatalf("second=%+v", rows[1])
	}

	if err := s.Delete(ctx, rows[0].ID); err != nil {
		t.Fatal(err)
	}
	if s.Depth() != 1 {
		t.Fatalf("depth=%d, want 1", s.Depth())
	}

	rows, err = s.PeekOldest(ctx, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].Topic != "topic/b" {
		t.Fatalf("remaining=%+v", rows)
	}
}

func TestEnqueueBatchAtomic(t *testing.T) {
	s := openTestStore(t, 2)
	ctx := context.Background()

	err := s.EnqueueBatch(ctx, []PendingEvent{
		{Topic: "t1", Payload: []byte("a")},
		{Topic: "t2", Payload: []byte("b")},
		{Topic: "t3", Payload: []byte("c")},
	})
	if !errors.Is(err, ErrBufferFull) {
		t.Fatalf("err=%v, want ErrBufferFull", err)
	}
	if s.Depth() != 0 {
		t.Fatalf("depth=%d, want 0 after failed batch", s.Depth())
	}

	if err := s.EnqueueBatch(ctx, []PendingEvent{
		{Topic: "t1", Payload: []byte("a")},
		{Topic: "t2", Payload: []byte("b")},
	}); err != nil {
		t.Fatal(err)
	}
	if s.Depth() != 2 {
		t.Fatalf("depth=%d, want 2", s.Depth())
	}
}

func TestBufferFull(t *testing.T) {
	s := openTestStore(t, 2)
	ctx := context.Background()

	if err := s.Enqueue(ctx, "t1", []byte("a")); err != nil {
		t.Fatal(err)
	}
	if err := s.Enqueue(ctx, "t2", []byte("b")); err != nil {
		t.Fatal(err)
	}
	err := s.Enqueue(ctx, "t3", []byte("c"))
	if !errors.Is(err, ErrBufferFull) {
		t.Fatalf("err=%v, want ErrBufferFull", err)
	}
	if s.Depth() != 2 {
		t.Fatalf("depth=%d", s.Depth())
	}
}

func TestWakeCoalesced(t *testing.T) {
	s := openTestStore(t, 0)
	ctx := context.Background()

	if err := s.Enqueue(ctx, "t1", []byte("a")); err != nil {
		t.Fatal(err)
	}
	if err := s.Enqueue(ctx, "t2", []byte("b")); err != nil {
		t.Fatal(err)
	}

	select {
	case <-s.Wake():
	default:
		t.Fatal("expected wake signal")
	}
	// Second signal should have been coalesced (channel buffer 1).
	select {
	case <-s.Wake():
		t.Fatal("unexpected second wake without new enqueue")
	default:
	}
}

func TestReopenBootstrapsDepth(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "buffer.db")
	m := testMetrics(t)

	s1, err := Open(Options{Path: path, MaxEntries: 0, BusyTimeoutMS: 5000, Metrics: m})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := s1.Enqueue(ctx, "t1", []byte("a")); err != nil {
		t.Fatal(err)
	}
	if err := s1.Enqueue(ctx, "t2", []byte("b")); err != nil {
		t.Fatal(err)
	}
	if err := s1.Close(); err != nil {
		t.Fatal(err)
	}

	s2, err := Open(Options{Path: path, MaxEntries: 0, BusyTimeoutMS: 5000, Metrics: m})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s2.Close() })
	if s2.Depth() != 2 {
		t.Fatalf("depth=%d, want 2", s2.Depth())
	}
}
