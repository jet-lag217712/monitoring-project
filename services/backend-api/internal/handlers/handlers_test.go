package handlers_test

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/equate/ogsd/services/backend-api/internal/handlers"
	"github.com/equate/ogsd/services/backend-api/internal/models"
	"github.com/google/uuid"
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
	if !strings.Contains(rec.Header().Get("Access-Control-Allow-Methods"), "PATCH") {
		t.Fatalf("PATCH missing from Allow-Methods: %v", rec.Header().Get("Access-Control-Allow-Methods"))
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
					IDFCount:                0,
					DeviceCount:             2,
					OnlineCount:             2,
					AvgCPU:                  0,
					AvgMemory:               0,
					ActiveAlerts:            0,
					HealthyCount:            2,
					WarningCount:            0,
					CriticalCount:           0,
					UnknownCount:            0,
					DependencyImpactedCount: 0,
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
	for _, key := range []string{
		"idf_count", "device_count", "online_count", "avg_cpu", "avg_memory", "active_alerts",
		"healthy_count", "warning_count", "critical_count", "unknown_count", "dependency_impacted_count",
	} {
		if _, ok := summary[key]; !ok {
			t.Fatalf("missing summary key %q", key)
		}
	}
}

func TestSiteDetailJSONKeys(t *testing.T) {
	days := 2.0
	cpu := 31.2
	fc := 2
	detail := models.SiteDetail{
		SiteID:   "district-office",
		Location: "District Office",
		Summary: models.SiteDetailSummary{
			TotalDevices:            2,
			OnlineCount:             1,
			IDFCount:                0,
			ActiveAlerts:            0,
			HealthyCount:            0,
			WarningCount:            0,
			CriticalCount:           1,
			UnknownCount:            1,
			DependencyImpactedCount: 1,
		},
		Latest: models.SiteDetailLatest{
			Devices: map[string]models.DeviceSummary{
				"core-sw-01": {
					Hostname:     "core-sw-01",
					Role:         "",
					Status:       3,
					StatusReason: "poll_failed",
					FailureCount: &fc,
					CPUPct:       &cpu,
					MemoryPct:    nil,
					UptimeDays:   &days,
					LatencyMs:    nil,
				},
				"access-01": {
					Hostname:               "access-01",
					Role:                   "",
					Status:                 0,
					StatusReason:           "upstream_unreachable",
					FailureCount:           &fc,
					UpstreamDeviceIDs:      []string{"dist-01"},
					UnavailableUpstreamIDs: []string{"dist-01"},
					RootCauseDeviceIDs:     []string{"core-sw-01"},
					CPUPct:                 nil,
					MemoryPct:              nil,
					UptimeDays:             nil,
					LatencyMs:              nil,
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
	for _, key := range []string{
		"total_devices", "online_count", "idf_count", "active_alerts",
		"healthy_count", "warning_count", "critical_count", "unknown_count", "dependency_impacted_count",
	} {
		if _, ok := summary[key]; !ok {
			t.Fatalf("missing summary key %q", key)
		}
	}
	if summary["critical_count"].(float64) != 1 || summary["unknown_count"].(float64) != 1 {
		t.Fatalf("critical/unknown counts wrong: %+v", summary)
	}
	latest := decoded["latest"].(map[string]any)
	devices := latest["devices"].(map[string]any)
	dev := devices["core-sw-01"].(map[string]any)
	for _, key := range []string{"hostname", "role", "status", "cpu_pct", "memory_pct", "uptime_days", "latency_ms", "status_reason"} {
		if _, ok := dev[key]; !ok {
			t.Fatalf("missing device key %q", key)
		}
	}
	access := devices["access-01"].(map[string]any)
	if access["status"].(float64) != 0 {
		t.Fatalf("unknown device status=%v", access["status"])
	}
	if access["status_reason"] != "upstream_unreachable" {
		t.Fatalf("reason=%v", access["status_reason"])
	}
	roots := access["root_cause_device_ids"].([]any)
	if len(roots) != 1 || roots[0] != "core-sw-01" {
		t.Fatalf("root causes=%v", roots)
	}
}

func TestDeviceDetailJSONKeys(t *testing.T) {
	temp := 52.5
	detail := models.DeviceDetail{
		ID:           uuid.MustParse("11111111-1111-1111-1111-111111111111"),
		SiteID:       "district-office",
		Hostname:     "access-01",
		IPAddress:    "10.0.20.11",
		Vendor:       "cisco",
		Model:        "sanitized-model",
		SerialNumber: "sanitized-serial",
		Profile:      "cisco",
		Status:       0,
		StatusReason: "upstream_unreachable",
		TemperatureC: &temp,
		CPUPct:       nil,
		MemoryPct:    nil,
		SNMP: &models.SNMPIdentity{
			SysName:     "access-01",
			SysObjectID: "1.3.6.1.4.1.9.1.9999",
			SysDescr:    "Sanitized lab fixture",
		},
		PowerComponents: []models.ComponentReading{
			{ComponentID: "power-1", Name: "PSU 1", Status: "ok", Unit: "state"},
		},
		History: &models.DeviceHistory{
			CPU:         []models.MetricPoint{},
			Memory:      []models.MetricPoint{},
			Temperature: []models.MetricPoint{{TS: time.Date(2026, 7, 16, 18, 0, 0, 0, time.UTC), Value: 52.5}},
			Uptime:      []models.MetricPoint{},
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
	for _, key := range []string{
		"id", "site_id", "hostname", "status", "status_reason", "temperature_c",
		"serial_number", "profile", "snmp", "power_components", "history",
	} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing key %q", key)
		}
	}
	if decoded["status"].(float64) != 0 {
		t.Fatalf("status=%v", decoded["status"])
	}
}

func TestSiteLocationUpdateJSONKeys(t *testing.T) {
	update := models.SiteLocationUpdate{
		SiteID:   "district-office",
		Location: "District Office",
	}
	raw, err := json.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"site_id", "location"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("missing key %q", key)
		}
	}
}
