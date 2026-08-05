package store_test

import (
	"context"
	"testing"
	"time"

	"github.com/equate/ogsd/services/ingestion-service/internal/store"
)

func TestDeleteOlderThan_RejectsUnknownTarget(t *testing.T) {
	s := store.New(nil)
	_, err := s.DeleteOlderThan(context.Background(), "sites", "created_at", time.Now(), 100)
	if err == nil {
		t.Fatal("expected allowlist error")
	}
}

func TestDeleteOlderThan_RejectsColumnMismatch(t *testing.T) {
	s := store.New(nil)
	_, err := s.DeleteOlderThan(context.Background(), "metric_samples", "observed_at", time.Now(), 100)
	if err == nil {
		t.Fatal("expected allowlist error")
	}
}

func TestDeleteOlderThan_RejectsBadBatchSize(t *testing.T) {
	s := store.New(nil)
	_, err := s.DeleteOlderThan(context.Background(), "metric_samples", "collected_at", time.Now(), 0)
	if err == nil {
		t.Fatal("expected batch size error")
	}
}
