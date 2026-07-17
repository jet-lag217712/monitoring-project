//go:build integration

package tests

import (
	"context"
	"testing"
	"time"

	"github.com/equate/ogsd/services/ingestion-service/internal/handler"
	"github.com/equate/ogsd/services/ingestion-service/internal/metrics"
	"github.com/equate/ogsd/services/ingestion-service/internal/store"
	"github.com/equate/ogsd/services/ingestion-service/internal/transform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

func TestIngestV2_DeviceTelemetry_HappyPath(t *testing.T) {
	dbURL, _, _, _ := integrationEnv(t)
	ctx := context.Background()
	st := openStore(t, ctx, dbURL)
	defer st.Close()

	eventID := uuid.New()
	observedAt := time.Now().UTC().Truncate(time.Second)
	topic := "site/site-itest/device/dev-v2/telemetry/v2/device"
	payload := mustJSON(t, map[string]any{
		"schema_version":  "2.0",
		"event_id":        eventID.String(),
		"event_type":      "device_telemetry",
		"site_id":         "site-itest",
		"collector_id":    "collector-itest",
		"device_id":       "dev-v2",
		"observed_at":     observedAt.Format(time.RFC3339),
		"emitted_at":      observedAt.Add(time.Second).Format(time.RFC3339),
		"config_revision": "revision-itest",
		"payload": map[string]any{
			"identity": map[string]any{
				"hostname":      "dev-v2",
				"sys_object_id": "1.2.3",
				"sys_name":      "dev-v2",
				"sys_descr":     "integration",
				"snmp_version":  "2c",
			},
			"profile": map[string]any{
				"name":         "core",
				"capabilities": []string{},
			},
			"readings": map[string]any{
				"uptime_seconds": 42,
			},
			"temperature_components": []any{},
			"power_components":       []any{},
		},
	})

	m := metrics.NewWithRegisterer(prometheus.NewRegistry())
	h := handler.New(st, m, discardLog())
	if !h.Handle(ctx, topic, payload) {
		t.Fatal("expected ACK")
	}

	deviceID := transform.DeviceUUID("site-itest", "dev-v2")
	if countMetricSamples(t, ctx, st, deviceID, observedAt) < 1 {
		t.Fatal("expected uptime metric sample")
	}
	if !eventIngested(t, ctx, st, eventID) {
		t.Fatal("expected ingested_events row")
	}
}

func TestIngestV2_EventIDDedup(t *testing.T) {
	dbURL, _, _, _ := integrationEnv(t)
	ctx := context.Background()
	st := openStore(t, ctx, dbURL)
	defer st.Close()

	eventID := uuid.New()
	observedAt := time.Now().UTC().Truncate(time.Second).Add(2 * time.Second)
	topic := "site/site-itest/collector/collector-itest/telemetry/v2/heartbeat"
	payload := heartbeatPayload(t, eventID, "site-itest", "collector-itest", observedAt)

	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegisterer(reg)
	h := handler.New(st, m, discardLog())
	if !h.Handle(ctx, topic, payload) {
		t.Fatal("first should ACK")
	}
	if !h.Handle(ctx, topic, payload) {
		t.Fatal("duplicate should ACK")
	}
	if got := counterValue(t, m.MessagesDeduplicated); got < 1 {
		t.Fatalf("deduplicated=%v", got)
	}

	var historyCount int
	err := st.Pool().QueryRow(ctx, `
		SELECT COUNT(*) FROM collector_heartbeat_history WHERE event_id = $1
	`, eventID).Scan(&historyCount)
	if err != nil {
		t.Fatal(err)
	}
	if historyCount != 1 {
		t.Fatalf("history count=%d want 1", historyCount)
	}
}

func TestIngestV2_StaleHeartbeat_DoesNotOverwrite(t *testing.T) {
	dbURL, _, _, _ := integrationEnv(t)
	ctx := context.Background()
	st := openStore(t, ctx, dbURL)
	defer st.Close()

	collectorID := "collector-stale"
	newer := time.Now().UTC().Truncate(time.Second)
	older := newer.Add(-2 * time.Minute)
	topic := "site/site-itest/collector/" + collectorID + "/telemetry/v2/heartbeat"

	h := handler.New(st, metrics.NewWithRegisterer(prometheus.NewRegistry()), discardLog())
	if !h.Handle(ctx, topic, heartbeatPayload(t, uuid.New(), "site-itest", collectorID, newer)) {
		t.Fatal("newer should ACK")
	}
	if !h.Handle(ctx, topic, heartbeatPayload(t, uuid.New(), "site-itest", collectorID, older)) {
		t.Fatal("older should ACK (history retained)")
	}

	collectorUUID := transform.CollectorUUID("site-itest", collectorID)
	var observedAt time.Time
	var uptime int64
	err := st.Pool().QueryRow(ctx, `
		SELECT observed_at, uptime_seconds FROM collector_status_current WHERE collector_uuid = $1
	`, collectorUUID).Scan(&observedAt, &uptime)
	if err != nil {
		t.Fatal(err)
	}
	if !observedAt.Equal(newer) {
		t.Fatalf("current observed_at=%s want %s", observedAt, newer)
	}
}

func TestIngestV2_HealthHappyPath(t *testing.T) {
	dbURL, _, _, _ := integrationEnv(t)
	ctx := context.Background()
	st := openStore(t, ctx, dbURL)
	defer st.Close()

	eventID := uuid.New()
	observedAt := time.Now().UTC().Truncate(time.Second)
	topic := "site/site-itest/device/dev-health/telemetry/v2/health"
	payload := mustJSON(t, map[string]any{
		"schema_version":  "2.0",
		"event_id":        eventID.String(),
		"event_type":      "health_state",
		"site_id":         "site-itest",
		"collector_id":    "collector-itest",
		"device_id":       "dev-health",
		"observed_at":     observedAt.Format(time.RFC3339),
		"emitted_at":      observedAt.Add(time.Second).Format(time.RFC3339),
		"config_revision": "revision-itest",
		"payload": map[string]any{
			"state":                           "critical",
			"reason":                          "direct_unreachable",
			"transition":                      "entered",
			"failure_count":                   2,
			"failure_threshold":               2,
			"upstream_device_ids":             []string{},
			"unavailable_upstream_device_ids": []string{},
			"root_cause_device_ids":           []string{},
		},
	})

	h := handler.New(st, metrics.NewWithRegisterer(prometheus.NewRegistry()), discardLog())
	if !h.Handle(ctx, topic, payload) {
		t.Fatal("expected ACK")
	}

	deviceID := transform.DeviceUUID("site-itest", "dev-health")
	var state string
	err := st.Pool().QueryRow(ctx, `SELECT state FROM device_health_current WHERE device_id = $1`, deviceID).Scan(&state)
	if err != nil {
		t.Fatal(err)
	}
	if state != "critical" {
		t.Fatalf("state=%s", state)
	}
}

func TestIngestV2_Malformed_AckRejected(t *testing.T) {
	dbURL, _, _, _ := integrationEnv(t)
	ctx := context.Background()
	st := openStore(t, ctx, dbURL)
	defer st.Close()

	m := metrics.NewWithRegisterer(prometheus.NewRegistry())
	h := handler.New(st, m, discardLog())
	ack := h.Handle(ctx, "site/site-itest/device/dev-bad/telemetry/v2/device", []byte(`{"schema_version":"9.9"}`))
	if !ack {
		t.Fatal("malformed v2 should ACK as rejected")
	}
	if got := counterValue(t, m.MessagesRejected); got < 1 {
		t.Fatalf("rejected=%v", got)
	}
}

func TestIngestV2_DBFailure_NoAck(t *testing.T) {
	dbURL, _, _, _ := integrationEnv(t)
	ctx := context.Background()

	badPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	badStore := store.New(badPool)
	badPool.Close()

	m := metrics.NewWithRegisterer(prometheus.NewRegistry())
	h := handler.New(badStore, m, discardLog())
	observedAt := time.Now().UTC().Truncate(time.Second)
	payload := heartbeatPayload(t, uuid.New(), "site-itest", "collector-dbfail", observedAt)
	ack := h.Handle(ctx, "site/site-itest/collector/collector-dbfail/telemetry/v2/heartbeat", payload)
	if ack {
		t.Fatal("db failure must not ACK")
	}
}

func TestIngest_MixedV1AndV2(t *testing.T) {
	dbURL, _, _, _ := integrationEnv(t)
	ctx := context.Background()
	st := openStore(t, ctx, dbURL)
	defer st.Close()

	h := handler.New(st, metrics.NewWithRegisterer(prometheus.NewRegistry()), discardLog())
	ts := time.Now().UTC().Truncate(time.Second).Add(10 * time.Second)

	v1 := mustJSON(t, map[string]any{
		"timestamp": ts.Format(time.RFC3339),
		"metric":    "uptime_seconds",
		"value":     11.0,
	})
	if !h.Handle(ctx, "site/site-itest/device/dev-mixed/metric/device", v1) {
		t.Fatal("v1 should ACK")
	}

	eventID := uuid.New()
	v2 := mustJSON(t, map[string]any{
		"schema_version":  "2.0",
		"event_id":        eventID.String(),
		"event_type":      "device_telemetry",
		"site_id":         "site-itest",
		"collector_id":    "collector-itest",
		"device_id":       "dev-mixed",
		"observed_at":     ts.Add(time.Second).Format(time.RFC3339),
		"emitted_at":      ts.Add(2 * time.Second).Format(time.RFC3339),
		"config_revision": "revision-itest",
		"payload": map[string]any{
			"identity": map[string]any{
				"hostname": "dev-mixed", "sys_object_id": "1.2.3", "sys_name": "dev-mixed",
				"sys_descr": "mixed", "snmp_version": "2c",
			},
			"profile":                map[string]any{"name": "core", "capabilities": []string{}},
			"readings":               map[string]any{"uptime_seconds": 12},
			"temperature_components": []any{},
			"power_components":       []any{},
		},
	})
	if !h.Handle(ctx, "site/site-itest/device/dev-mixed/telemetry/v2/device", v2) {
		t.Fatal("v2 should ACK")
	}
	if !eventIngested(t, ctx, st, eventID) {
		t.Fatal("expected v2 event_id ledger entry")
	}
}

func heartbeatPayload(t *testing.T, eventID uuid.UUID, siteID, collectorID string, observedAt time.Time) []byte {
	t.Helper()
	return mustJSON(t, map[string]any{
		"schema_version":  "2.0",
		"event_id":        eventID.String(),
		"event_type":      "collector_heartbeat",
		"site_id":         siteID,
		"collector_id":    collectorID,
		"observed_at":     observedAt.Format(time.RFC3339),
		"emitted_at":      observedAt.Add(time.Second).Format(time.RFC3339),
		"config_revision": "revision-itest",
		"payload": map[string]any{
			"hostname":           "collector-host",
			"version":            "unknown",
			"git_commit":         "unknown",
			"build_time":         "unknown",
			"uptime_seconds":     int64(observedAt.Unix() % 100000),
			"sqlite_queue_depth": 0,
			"memory_usage_bytes": 1000,
			"goroutine_count":    5,
		},
	})
}

func eventIngested(t *testing.T, ctx context.Context, st *store.Store, eventID uuid.UUID) bool {
	t.Helper()
	var n int
	err := st.Pool().QueryRow(ctx, `SELECT COUNT(*) FROM ingested_events WHERE event_id = $1`, eventID).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n == 1
}
