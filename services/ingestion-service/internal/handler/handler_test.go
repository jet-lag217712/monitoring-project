package handler_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/equate/ogsd/services/ingestion-service/internal/handler"
	"github.com/equate/ogsd/services/ingestion-service/internal/metrics"
	"github.com/equate/ogsd/services/ingestion-service/internal/store"
	"github.com/equate/ogsd/services/ingestion-service/internal/transform"
	"github.com/prometheus/client_golang/prometheus"
)

type fakeStore struct {
	deviceResult store.Result
	deviceErr    error
	ifaceResult  store.Result
	ifaceErr     error
}

func (f *fakeStore) PersistDeviceSample(context.Context, transform.DeviceSample) (store.Result, error) {
	return f.deviceResult, f.deviceErr
}

func (f *fakeStore) PersistInterfaceSample(context.Context, transform.InterfaceSample) (store.Result, error) {
	return f.ifaceResult, f.ifaceErr
}

func newHandler(t *testing.T, s *fakeStore) *handler.Handler {
	t.Helper()
	m := metrics.NewWithRegisterer(prometheus.NewRegistry())
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return handler.New(s, m, log)
}

func TestHandle_RejectACKs(t *testing.T) {
	h := newHandler(t, &fakeStore{})
	ack := h.Handle(context.Background(), "site/a/device/b/metric/device", []byte(`{`))
	if !ack {
		t.Fatal("invalid payload should ACK")
	}
}

func TestHandle_DuplicateACKs(t *testing.T) {
	h := newHandler(t, &fakeStore{deviceResult: store.ResultDuplicate})
	payload := []byte(`{"timestamp":"2026-06-01T18:00:00Z","metric":"uptime_seconds","value":1}`)
	ack := h.Handle(context.Background(), "site/site-001/device/dev-001/metric/device", payload)
	if !ack {
		t.Fatal("duplicate should ACK")
	}
}

func TestHandle_DBErrorNoACK(t *testing.T) {
	h := newHandler(t, &fakeStore{deviceErr: errors.New("db down")})
	payload := []byte(`{"timestamp":"2026-06-01T18:00:00Z","metric":"uptime_seconds","value":1}`)
	ack := h.Handle(context.Background(), "site/site-001/device/dev-001/metric/device", payload)
	if ack {
		t.Fatal("db error must not ACK")
	}
}

func TestHandle_InsertACKs(t *testing.T) {
	h := newHandler(t, &fakeStore{deviceResult: store.ResultInserted})
	payload := []byte(`{"timestamp":"2026-06-01T18:00:00Z","metric":"uptime_seconds","value":1}`)
	ack := h.Handle(context.Background(), "site/site-001/device/dev-001/metric/device", payload)
	if !ack {
		t.Fatal("insert should ACK")
	}
}

func TestHandle_UnknownMetricACKs(t *testing.T) {
	h := newHandler(t, &fakeStore{deviceErr: store.ErrUnknownMetricType})
	payload := []byte(`{"timestamp":"2026-06-01T18:00:00Z","metric":"nope","value":1}`)
	ack := h.Handle(context.Background(), "site/site-001/device/dev-001/metric/device", payload)
	if !ack {
		t.Fatal("unknown metric should ACK")
	}
}
