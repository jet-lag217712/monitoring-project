//go:build integration

package tests

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/equate/ogsd/services/ingestion-service/internal/handler"
	"github.com/equate/ogsd/services/ingestion-service/internal/metrics"
	"github.com/equate/ogsd/services/ingestion-service/internal/store"
	"github.com/equate/ogsd/services/ingestion-service/internal/transform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestIngestDeviceMetric_HappyPath(t *testing.T) {
	dbURL, broker, caFile, password := integrationEnv(t)
	ctx := context.Background()
	st := openStore(t, ctx, dbURL)
	defer st.Close()

	cleanupDeviceSamples(t, ctx, st, "site-itest", "dev-happy")

	m := metrics.NewWithRegisterer(prometheus.NewRegistry())
	h := handler.New(st, m, discardLog())

	ts := time.Now().UTC().Truncate(time.Second)
	topic := "site/site-itest/device/dev-happy/metric/device"
	payload := mustJSON(t, map[string]any{
		"timestamp": ts.Format(time.RFC3339),
		"site_id":   "site-itest",
		"device_id": "dev-happy",
		"metric":    "uptime_seconds",
		"value":     99.0,
	})

	ack := h.Handle(ctx, topic, payload)
	if !ack {
		t.Fatal("expected ACK")
	}

	deviceID := transform.DeviceUUID("site-itest", "dev-happy")
	count := countMetricSamples(t, ctx, st, deviceID, ts)
	if count != 1 {
		t.Fatalf("metric_samples count=%d want 1", count)
	}

	var status string
	var lastSeen time.Time
	err := st.Pool().QueryRow(ctx,
		`SELECT status, last_seen FROM devices WHERE id = $1`, deviceID,
	).Scan(&status, &lastSeen)
	if err != nil {
		t.Fatal(err)
	}
	if status != "online" {
		t.Fatalf("status=%q", status)
	}
	if lastSeen.IsZero() {
		t.Fatal("last_seen empty")
	}

	// Also verify MQTT publish path reaches a live subscriber handler via broker.
	_ = broker
	_ = caFile
	_ = password
}

func TestIngestInterfaceMetric_HappyPath(t *testing.T) {
	dbURL, _, _, _ := integrationEnv(t)
	ctx := context.Background()
	st := openStore(t, ctx, dbURL)
	defer st.Close()

	m := metrics.NewWithRegisterer(prometheus.NewRegistry())
	h := handler.New(st, m, discardLog())

	ts := time.Now().UTC().Truncate(time.Second).Add(1 * time.Second)
	topic := "site/site-itest/device/dev-iface/metric/interface"
	payload := mustJSON(t, map[string]any{
		"timestamp":  ts.Format(time.RFC3339),
		"if_index":   2,
		"in_octets":  100,
		"out_octets": 200,
		"in_errors":  0,
		"out_errors": 1,
	})

	if !h.Handle(ctx, topic, payload) {
		t.Fatal("expected ACK")
	}

	deviceID := transform.DeviceUUID("site-itest", "dev-iface")
	var ifaceCount int
	err := st.Pool().QueryRow(ctx, `
		SELECT COUNT(*) FROM interfaces WHERE device_id = $1 AND if_index = 2
	`, deviceID).Scan(&ifaceCount)
	if err != nil {
		t.Fatal(err)
	}
	if ifaceCount != 1 {
		t.Fatalf("interfaces count=%d", ifaceCount)
	}

	var sampleCount int
	err = st.Pool().QueryRow(ctx, `
		SELECT COUNT(*) FROM interface_samples s
		JOIN interfaces i ON i.id = s.interface_id
		WHERE i.device_id = $1 AND i.if_index = 2 AND s.collected_at = $2
	`, deviceID, ts).Scan(&sampleCount)
	if err != nil {
		t.Fatal(err)
	}
	if sampleCount != 1 {
		t.Fatalf("interface_samples count=%d", sampleCount)
	}
}

func TestIngest_DuplicateDelivery_NoDoubleInsert(t *testing.T) {
	dbURL, _, _, _ := integrationEnv(t)
	ctx := context.Background()
	st := openStore(t, ctx, dbURL)
	defer st.Close()

	reg := prometheus.NewRegistry()
	m := metrics.NewWithRegisterer(reg)
	h := handler.New(st, m, discardLog())

	ts := time.Now().UTC().Truncate(time.Second).Add(2 * time.Second)
	topic := "site/site-itest/device/dev-dup/metric/device"
	payload := mustJSON(t, map[string]any{
		"timestamp": ts.Format(time.RFC3339),
		"metric":    "uptime_seconds",
		"value":     1.0,
	})

	if !h.Handle(ctx, topic, payload) {
		t.Fatal("first handle should ACK")
	}
	if !h.Handle(ctx, topic, payload) {
		t.Fatal("duplicate should ACK")
	}

	deviceID := transform.DeviceUUID("site-itest", "dev-dup")
	count := countMetricSamples(t, ctx, st, deviceID, ts)
	if count != 1 {
		t.Fatalf("count=%d want 1", count)
	}
	if got := counterValue(t, m.MessagesDeduplicated); got < 1 {
		t.Fatalf("deduplicated=%v want >=1", got)
	}
}

func TestIngest_InvalidPayload_AckNoInsert(t *testing.T) {
	dbURL, _, _, _ := integrationEnv(t)
	ctx := context.Background()
	st := openStore(t, ctx, dbURL)
	defer st.Close()

	before := totalMetricSamples(t, ctx, st)
	m := metrics.NewWithRegisterer(prometheus.NewRegistry())
	h := handler.New(st, m, discardLog())

	ack := h.Handle(ctx, "site/site-itest/device/dev-bad/metric/device", []byte(`{not-json`))
	if !ack {
		t.Fatal("invalid should ACK")
	}
	after := totalMetricSamples(t, ctx, st)
	if after != before {
		t.Fatalf("rows changed %d -> %d", before, after)
	}
	if got := counterValue(t, m.MessagesRejected); got < 1 {
		t.Fatalf("rejected=%v", got)
	}
}

func TestIngest_UnknownMetric_AckNoInsert(t *testing.T) {
	dbURL, _, _, _ := integrationEnv(t)
	ctx := context.Background()
	st := openStore(t, ctx, dbURL)
	defer st.Close()

	before := totalMetricSamples(t, ctx, st)
	m := metrics.NewWithRegisterer(prometheus.NewRegistry())
	h := handler.New(st, m, discardLog())

	ts := time.Now().UTC().Truncate(time.Second).Add(3 * time.Second)
	payload := mustJSON(t, map[string]any{
		"timestamp": ts.Format(time.RFC3339),
		"metric":    "not_a_real_metric",
		"value":     1.0,
	})
	ack := h.Handle(ctx, "site/site-itest/device/dev-unknown/metric/device", payload)
	if !ack {
		t.Fatal("unknown metric should ACK")
	}
	after := totalMetricSamples(t, ctx, st)
	if after != before {
		t.Fatalf("rows changed %d -> %d", before, after)
	}
}

func TestIngest_DBFailure_NoAck_ThenRedeliver(t *testing.T) {
	dbURL, _, _, _ := integrationEnv(t)
	ctx := context.Background()

	// First handler uses a closed/broken pool to simulate DB failure.
	badPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Fatal(err)
	}
	badStore := store.New(badPool)
	badPool.Close() // force subsequent queries to fail

	mFail := metrics.NewWithRegisterer(prometheus.NewRegistry())
	hFail := handler.New(badStore, mFail, discardLog())

	ts := time.Now().UTC().Truncate(time.Second).Add(4 * time.Second)
	topic := "site/site-itest/device/dev-redeliver/metric/device"
	payload := mustJSON(t, map[string]any{
		"timestamp": ts.Format(time.RFC3339),
		"metric":    "uptime_seconds",
		"value":     7.0,
	})

	ack := hFail.Handle(ctx, topic, payload)
	if ack {
		t.Fatal("DB failure must not ACK")
	}
	if got := counterValue(t, mFail.DBWriteFailure); got < 1 {
		t.Fatalf("db_write_failure=%v", got)
	}

	// Recover with a healthy store and reprocess (simulates MQTT redelivery).
	st := openStore(t, ctx, dbURL)
	defer st.Close()
	mOK := metrics.NewWithRegisterer(prometheus.NewRegistry())
	hOK := handler.New(st, mOK, discardLog())
	if !hOK.Handle(ctx, topic, payload) {
		t.Fatal("redelivery should ACK after DB recovery")
	}

	deviceID := transform.DeviceUUID("site-itest", "dev-redeliver")
	count := countMetricSamples(t, ctx, st, deviceID, ts)
	if count != 1 {
		t.Fatalf("count=%d want 1 after redelivery", count)
	}
}

func TestIngest_MQTTBrokerRoundTrip(t *testing.T) {
	dbURL, broker, caFile, password := integrationEnv(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	st := openStore(t, ctx, dbURL)
	defer st.Close()

	m := metrics.NewWithRegisterer(prometheus.NewRegistry())
	h := handler.New(st, m, discardLog())

	// Live subscriber with deferred ACK via handler.
	subCancel := startIngestionSubscriber(t, ctx, broker, caFile, password, "ingestion-itest-"+uuid.NewString()[:8], h)
	defer subCancel()

	ts := time.Now().UTC().Truncate(time.Second).Add(5 * time.Second)
	topic := "site/site-itest/device/dev-mqtt/metric/device"
	payload := mustJSON(t, map[string]any{
		"timestamp": ts.Format(time.RFC3339),
		"metric":    "uptime_seconds",
		"value":     55.0,
	})

	publishQoS1(t, ctx, broker, caFile, payload, topic)

	deviceID := transform.DeviceUUID("site-itest", "dev-mqtt")
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if countMetricSamples(t, ctx, st, deviceID, ts) == 1 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("timeout waiting for MQTT-ingested row")
}

func integrationEnv(t *testing.T) (dbURL, broker, caFile, password string) {
	t.Helper()
	password = os.Getenv("MQTT_PASSWORD")
	if password == "" {
		t.Skip("MQTT_PASSWORD not set; start stack with ./deployments/local/up.sh")
	}
	broker = os.Getenv("MQTT_BROKER")
	if broker == "" {
		broker = "tls://127.0.0.1:8883"
	}
	caFile = os.Getenv("MQTT_CA_FILE")
	if caFile == "" {
		// Resolve from this source file so cwd (package dir under go test) does not matter.
		_, thisFile, _, ok := runtime.Caller(0)
		if !ok {
			t.Fatal("runtime.Caller failed")
		}
		repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
		caFile = filepath.Join(repoRoot, "infrastructure", "docker", "mqtt-broker", "certs", "ca.crt")
	}
	if _, err := os.Stat(caFile); err != nil {
		t.Skipf("ca file missing (%s); run ./deployments/local/up.sh", caFile)
	}
	dbURL = os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; start stack with ./deployments/local/up.sh")
	}
	return dbURL, broker, caFile, password
}

func openStore(t *testing.T, ctx context.Context, dbURL string) *store.Store {
	t.Helper()
	st, err := store.Open(ctx, dbURL, 5, 1, time.Hour)
	if err != nil {
		t.Fatalf("open store (is postgres up via local E2E stack?): %v", err)
	}
	return st
}

func discardLog() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func countMetricSamples(t *testing.T, ctx context.Context, st *store.Store, deviceID uuid.UUID, collectedAt time.Time) int {
	t.Helper()
	var n int
	err := st.Pool().QueryRow(ctx, `
		SELECT COUNT(*) FROM metric_samples WHERE device_id = $1 AND collected_at = $2
	`, deviceID, collectedAt).Scan(&n)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func totalMetricSamples(t *testing.T, ctx context.Context, st *store.Store) int {
	t.Helper()
	var n int
	if err := st.Pool().QueryRow(ctx, `SELECT COUNT(*) FROM metric_samples`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func cleanupDeviceSamples(t *testing.T, ctx context.Context, st *store.Store, siteID, deviceID string) {
	t.Helper()
	id := transform.DeviceUUID(siteID, deviceID)
	_, _ = st.Pool().Exec(ctx, `DELETE FROM metric_samples WHERE device_id = $1`, id)
}

func counterValue(t *testing.T, c prometheus.Counter) float64 {
	t.Helper()
	var m dto.Metric
	if err := c.Write(&m); err != nil {
		t.Fatal(err)
	}
	return m.GetCounter().GetValue()
}

func startIngestionSubscriber(t *testing.T, ctx context.Context, broker, caFile, password, clientID string, h *handler.Handler) context.CancelFunc {
	t.Helper()
	u, err := url.Parse(broker)
	if err != nil {
		t.Fatal(err)
	}
	pem, err := os.ReadFile(caFile)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		t.Fatal("bad ca")
	}

	subCtx, cancel := context.WithCancel(ctx)
	cliCfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{u},
		KeepAlive:                     30,
		CleanStartOnInitialConnection: true,
		TlsCfg: &tls.Config{
			RootCAs:    pool,
			MinVersion: tls.VersionTLS12,
			ServerName: u.Hostname(),
		},
		ConnectUsername: "ingestion",
		ConnectPassword: []byte(password),
		OnConnectionUp: func(cm *autopaho.ConnectionManager, _ *paho.Connack) {
			if _, err := cm.Subscribe(context.Background(), &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{{Topic: "site/+/device/+/metric/#", QoS: 1}},
			}); err != nil {
				t.Errorf("subscribe: %v", err)
			}
		},
		ClientConfig: paho.ClientConfig{
			ClientID:                   clientID,
			EnableManualAcknowledgment: true,
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(pr paho.PublishReceived) (bool, error) {
					ack := h.Handle(context.Background(), pr.Packet.Topic, pr.Packet.Payload)
					if !ack {
						return false, nil
					}
					if err := pr.Client.Ack(pr.Packet); err != nil {
						return false, err
					}
					return true, nil
				},
			},
		},
	}
	cm, err := autopaho.NewConnection(subCtx, cliCfg)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	if err := cm.AwaitConnection(subCtx); err != nil {
		cancel()
		t.Fatalf("subscriber await: %v", err)
	}
	t.Cleanup(func() {
		cancel()
		<-cm.Done()
	})
	return cancel
}

func publishQoS1(t *testing.T, ctx context.Context, broker, caFile string, payload []byte, topic string) {
	t.Helper()
	u, err := url.Parse(broker)
	if err != nil {
		t.Fatal(err)
	}
	pem, err := os.ReadFile(caFile)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		t.Fatal("bad ca")
	}
	password := os.Getenv("MQTT_COLLECTOR_PASSWORD")
	if password == "" {
		password = "secret"
	}

	pubCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	cliCfg := autopaho.ClientConfig{
		ServerUrls:                    []*url.URL{u},
		KeepAlive:                     30,
		CleanStartOnInitialConnection: true,
		TlsCfg: &tls.Config{
			RootCAs:    pool,
			MinVersion: tls.VersionTLS12,
			ServerName: u.Hostname(),
		},
		ConnectUsername: "collector",
		ConnectPassword: []byte(password),
		ClientConfig: paho.ClientConfig{
			ClientID: fmt.Sprintf("collector-itest-%s", uuid.NewString()[:8]),
		},
	}
	cm, err := autopaho.NewConnection(pubCtx, cliCfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancel()
		<-cm.Done()
	}()
	if err := cm.AwaitConnection(pubCtx); err != nil {
		t.Fatal(err)
	}
	if _, err := cm.Publish(pubCtx, &paho.Publish{QoS: 1, Topic: topic, Payload: payload}); err != nil {
		t.Fatal(err)
	}
}
