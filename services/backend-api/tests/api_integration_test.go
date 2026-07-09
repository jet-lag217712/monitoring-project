//go:build integration

package tests

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/equate/ogsd/services/backend-api/internal/handlers"
	"github.com/equate/ogsd/services/backend-api/internal/models"
	"github.com/equate/ogsd/services/backend-api/internal/store"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAPI_SitesAndDetailShapes(t *testing.T) {
	dbURL := integrationDBURL(t)
	ctx := context.Background()

	adminURL := os.Getenv("DATABASE_ADMIN_URL")
	if adminURL == "" {
		// Local test-env superuser (migrations / seed only).
		adminURL = "postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable"
	}

	seedSite(t, ctx, adminURL)
	defer cleanupSeed(t, ctx, adminURL)

	st, err := store.Open(ctx, dbURL, 5, 1, time.Hour)
	if err != nil {
		t.Fatalf("open store as ogsd_api: %v", err)
	}
	defer st.Close()

	api := handlers.New(st, slog.New(slog.NewTextHandler(io.Discard, nil)), 5*time.Minute)
	mux := http.NewServeMux()
	api.Register(mux)

	t.Run("list sites", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/sites", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var out map[string]models.SiteOverview
		if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		site, ok := out["api-itest-site"]
		if !ok {
			t.Fatalf("missing site key; got %#v", out)
		}
		if site.Status != "ok" {
			t.Fatalf("status=%q", site.Status)
		}
		if site.Latest.Summary.DeviceCount != 1 {
			t.Fatalf("device_count=%d", site.Latest.Summary.DeviceCount)
		}
		if site.Latest.Summary.OnlineCount != 1 {
			t.Fatalf("online_count=%d", site.Latest.Summary.OnlineCount)
		}
	})

	t.Run("site detail", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/sites/api-itest-site", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var detail models.SiteDetail
		if err := json.Unmarshal(rec.Body.Bytes(), &detail); err != nil {
			t.Fatal(err)
		}
		if detail.SiteID != "api-itest-site" {
			t.Fatalf("site_id=%q", detail.SiteID)
		}
		dev, ok := detail.Latest.Devices["api-itest-device"]
		if !ok {
			t.Fatalf("expected hostname key; devices=%#v", detail.Latest.Devices)
		}
		if dev.Hostname != "api-itest-device" || dev.Status != 1 {
			t.Fatalf("device=%+v", dev)
		}
		if dev.UptimeDays == nil || *dev.UptimeDays != 1.0 {
			t.Fatalf("uptime_days=%v want 1.0", dev.UptimeDays)
		}
	})

	t.Run("device by hostname", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/devices/api-itest-device", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("device by site-scoped collector id", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/devices/api-itest-device?siteId=api-itest-site", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
	})

	t.Run("ambiguous device without siteId", func(t *testing.T) {
		seedDuplicateDevice(t, ctx, adminURL)
		defer cleanupDuplicateDevice(t, ctx, adminURL)

		req := httptest.NewRequest(http.MethodGet, "/api/devices/api-itest-device", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusBadRequest {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var errBody models.APIError
		if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
			t.Fatal(err)
		}
		if errBody.Error.Code != "VALIDATION_ERROR" {
			t.Fatalf("code=%q", errBody.Error.Code)
		}
	})

	t.Run("device metrics", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/devices/api-itest-device/metrics?metric=uptime_seconds", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
		}
		var resp models.MetricsResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatal(err)
		}
		if len(resp.Points) < 1 {
			t.Fatal("expected metric points")
		}
	})

	t.Run("unknown site 404", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/sites/does-not-exist", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("status=%d", rec.Code)
		}
		var errBody models.APIError
		if err := json.Unmarshal(rec.Body.Bytes(), &errBody); err != nil {
			t.Fatal(err)
		}
		if errBody.Error.Code != "RESOURCE_NOT_FOUND" {
			t.Fatalf("code=%q", errBody.Error.Code)
		}
	})

	t.Run("alerts empty ok", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/alerts", nil)
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status=%d", rec.Code)
		}
		var alerts []models.AlertInfo
		if err := json.Unmarshal(rec.Body.Bytes(), &alerts); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("ogsd_api cannot write", func(t *testing.T) {
		_, err := st.Pool().Exec(ctx, `INSERT INTO sites (id, name) VALUES ($1, $2)`, uuid.New(), "should-fail")
		if err == nil {
			t.Fatal("expected write to fail for ogsd_api")
		}
	})
}

func integrationDBURL(t *testing.T) string {
	t.Helper()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set; start stack with ./deployments/local/test-env/up.sh")
	}
	return dbURL
}

func seedDuplicateDevice(t *testing.T, ctx context.Context, adminURL string) {
	t.Helper()
	pool, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatalf("admin connect: %v", err)
	}
	defer pool.Close()

	siteID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeee0003")
	deviceID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeee0004")
	now := time.Now().UTC()

	_, err = pool.Exec(ctx, `
		INSERT INTO sites (id, name, location)
		VALUES ($1, 'api-itest-site-2', 'API Integration Site 2')
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, location = EXCLUDED.location
	`, siteID)
	if err != nil {
		t.Fatalf("seed duplicate site: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO devices (id, site_id, hostname, ip_address, vendor, model, snmp_version, status, last_seen)
		VALUES ($1, $2, 'api-itest-device', '0.0.0.0', 'unknown', 'unknown', '2c', 'online', $3)
		ON CONFLICT (id) DO UPDATE SET
			hostname = EXCLUDED.hostname,
			status = EXCLUDED.status,
			last_seen = EXCLUDED.last_seen
	`, deviceID, siteID, now)
	if err != nil {
		t.Fatalf("seed duplicate device: %v", err)
	}
}

func cleanupDuplicateDevice(t *testing.T, ctx context.Context, adminURL string) {
	t.Helper()
	pool, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Logf("cleanup connect: %v", err)
		return
	}
	defer pool.Close()

	deviceID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeee0004")
	siteID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeee0003")
	_, _ = pool.Exec(ctx, `DELETE FROM devices WHERE id = $1`, deviceID)
	_, _ = pool.Exec(ctx, `DELETE FROM sites WHERE id = $1`, siteID)
}

func seedSite(t *testing.T, ctx context.Context, adminURL string) {
	t.Helper()
	pool, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Fatalf("admin connect: %v", err)
	}
	defer pool.Close()

	siteID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeee0001")
	deviceID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeee0002")
	now := time.Now().UTC()

	_, err = pool.Exec(ctx, `
		INSERT INTO sites (id, name, location)
		VALUES ($1, 'api-itest-site', 'API Integration Site')
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, location = EXCLUDED.location
	`, siteID)
	if err != nil {
		t.Fatalf("seed site: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO devices (id, site_id, hostname, ip_address, vendor, model, snmp_version, status, last_seen)
		VALUES ($1, $2, 'api-itest-device', '0.0.0.0', 'unknown', 'unknown', '2c', 'online', $3)
		ON CONFLICT (id) DO UPDATE SET
			hostname = EXCLUDED.hostname,
			status = EXCLUDED.status,
			last_seen = EXCLUDED.last_seen
	`, deviceID, siteID, now)
	if err != nil {
		t.Fatalf("seed device: %v", err)
	}

	var metricTypeID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT id FROM metric_types WHERE name = 'uptime_seconds'`).Scan(&metricTypeID)
	if err != nil {
		t.Fatalf("lookup metric type: %v", err)
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO metric_samples (device_id, metric_type_id, value, collected_at)
		VALUES ($1, $2, 86400, $3)
		ON CONFLICT DO NOTHING
	`, deviceID, metricTypeID, now)
	if err != nil {
		t.Fatalf("seed metric: %v", err)
	}
}

func cleanupSeed(t *testing.T, ctx context.Context, adminURL string) {
	t.Helper()
	pool, err := pgxpool.New(ctx, adminURL)
	if err != nil {
		t.Logf("cleanup connect: %v", err)
		return
	}
	defer pool.Close()

	deviceID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeee0002")
	siteID := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeee0001")
	_, _ = pool.Exec(ctx, `DELETE FROM metric_samples WHERE device_id = $1`, deviceID)
	_, _ = pool.Exec(ctx, `DELETE FROM devices WHERE id = $1`, deviceID)
	_, _ = pool.Exec(ctx, `DELETE FROM sites WHERE id = $1`, siteID)
}
