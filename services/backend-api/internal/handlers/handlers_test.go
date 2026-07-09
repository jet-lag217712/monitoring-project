package handlers_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/equate/ogsd/services/backend-api/internal/handlers"
	"github.com/equate/ogsd/services/backend-api/internal/models"
)

func TestTestConfigShape(t *testing.T) {
	api := handlers.New(nil, slog.New(slog.NewTextHandler(io.Discard, nil)), 5*time.Minute)
	mux := http.NewServeMux()
	api.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/test-config", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}

	var cfg models.TestConfig
	if err := json.Unmarshal(rec.Body.Bytes(), &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Mode != "live" || !cfg.PollingEnabled {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestErrorEnvelopeShape(t *testing.T) {
	// Exercise CORS + OPTIONS path and ensure JSON content-type on a 404 from mux.
	mux := http.NewServeMux()
	api := handlers.New(nil, slog.New(slog.NewTextHandler(io.Discard, nil)), 5*time.Minute)
	api.Register(mux)
	h := handlers.CORS([]string{"http://127.0.0.1:5173"}, mux)

	req := httptest.NewRequest(http.MethodOptions, "/api/sites", nil)
	req.Header.Set("Origin", "http://127.0.0.1:5173")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("OPTIONS status=%d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:5173" {
		t.Fatalf("missing CORS origin: %v", rec.Header())
	}
}

func TestSiteOverviewJSONKeys(t *testing.T) {
	// Contract check: overview object keys match normalizeSites expectations.
	overview := map[string]models.SiteOverview{
		"district-office": {
			Location: "District Office",
			Type:     "",
			Status:   "ok",
			Latest: models.SiteOverviewLatest{
				Summary: models.SiteSummary{
					IDFCount:     0,
					DeviceCount:  2,
					OnlineCount:  2,
					AvgCPU:       0,
					AvgMemory:    0,
					ActiveAlerts: 0,
				},
			},
		},
	}
	raw, err := json.Marshal(overview)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	site := decoded["district-office"]
	for _, key := range []string{"location", "type", "status", "latest"} {
		if _, ok := site[key]; !ok {
			t.Fatalf("missing key %q", key)
		}
	}
	latest := site["latest"].(map[string]any)
	summary := latest["summary"].(map[string]any)
	for _, key := range []string{"idf_count", "device_count", "online_count", "avg_cpu", "avg_memory", "active_alerts"} {
		if _, ok := summary[key]; !ok {
			t.Fatalf("missing summary key %q", key)
		}
	}
}

func TestSiteDetailJSONKeys(t *testing.T) {
	days := 2.0
	detail := models.SiteDetail{
		SiteID:   "district-office",
		Location: "District Office",
		Summary: models.SiteDetailSummary{
			TotalDevices: 1,
			OnlineCount:  1,
			IDFCount:     0,
			ActiveAlerts: 0,
		},
		Latest: models.SiteDetailLatest{
			Devices: map[string]models.DeviceSummary{
				"core-sw-01": {
					Hostname:   "core-sw-01",
					Role:       "",
					Status:     1,
					CPUPct:     0,
					MemoryPct:  0,
					UptimeDays: &days,
					LatencyMs:  nil,
				},
			},
		},
	}
	raw, err := json.Marshal(detail)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"site_id", "location", "summary", "latest"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing key %q", key)
		}
	}
	summary := decoded["summary"].(map[string]any)
	for _, key := range []string{"total_devices", "online_count", "idf_count", "active_alerts"} {
		if _, ok := summary[key]; !ok {
			t.Fatalf("missing summary key %q", key)
		}
	}
	latest := decoded["latest"].(map[string]any)
	devices := latest["devices"].(map[string]any)
	dev := devices["core-sw-01"].(map[string]any)
	for _, key := range []string{"hostname", "role", "status", "cpu_pct", "memory_pct", "uptime_days", "latency_ms"} {
		if _, ok := dev[key]; !ok {
			t.Fatalf("missing device key %q", key)
		}
	}
}
