package handlers

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/equate/ogsd/services/backend-api/internal/derive"
	"github.com/equate/ogsd/services/backend-api/internal/models"
	"github.com/equate/ogsd/services/backend-api/internal/store"
	"github.com/google/uuid"
)

// API serves the read-only monitoring REST endpoints.
type API struct {
	store           *store.Store
	log             *slog.Logger
	onlineThreshold time.Duration
	now             func() time.Time
}

// New creates an API handler set.
func New(s *store.Store, log *slog.Logger, onlineThreshold time.Duration) *API {
	return &API{
		store:           s,
		log:             log,
		onlineThreshold: onlineThreshold,
		now:             time.Now,
	}
}

// Register mounts all MVP routes on mux.
func (a *API) Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/sites", a.handleListSites)
	mux.HandleFunc("GET /api/sites/{siteId}", a.handleGetSite)
	mux.HandleFunc("GET /api/sites/{siteId}/devices", a.handleListSiteDevices)
	mux.HandleFunc("GET /api/devices/{deviceId}", a.handleGetDevice)
	mux.HandleFunc("GET /api/devices/{deviceId}/interfaces", a.handleListInterfaces)
	mux.HandleFunc("GET /api/devices/{deviceId}/metrics", a.handleListMetrics)
	mux.HandleFunc("GET /api/alerts", a.handleListAlerts)
	mux.HandleFunc("GET /api/test-config", a.handleTestConfig)
}

func (a *API) handleListSites(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sites, err := a.store.ListSites(ctx)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}
	devices, err := a.store.ListAllDevices(ctx)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}
	alertCounts, err := a.store.CountActiveAlertsBySite(ctx)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	now := a.now()
	bySite := make(map[uuid.UUID][]store.DeviceRow)
	for _, d := range devices {
		bySite[d.SiteID] = append(bySite[d.SiteID], d)
	}

	out := make(map[string]models.SiteOverview, len(sites))
	for _, site := range sites {
		devs := bySite[site.ID]
		deviceCount := len(devs)
		onlineCount := 0
		anyOffline := false
		for _, d := range devs {
			online := derive.DeviceOnline(d.Status, d.LastSeen, now, a.onlineThreshold)
			if online {
				onlineCount++
			} else {
				anyOffline = true
			}
		}
		if deviceCount == 0 {
			anyOffline = false
		}

		loc := ""
		if site.Location != nil {
			loc = *site.Location
		}
		out[site.Name] = models.SiteOverview{
			Location: derive.LocationOrName(loc, site.Name),
			Type:     "",
			Status:   derive.SiteStatus(anyOffline),
			Latest: models.SiteOverviewLatest{
				Summary: models.SiteSummary{
					IDFCount:     0,
					DeviceCount:  deviceCount,
					OnlineCount:  onlineCount,
					AvgCPU:       0,
					AvgMemory:    0,
					ActiveAlerts: alertCounts[site.ID],
				},
			},
		}
	}

	writeJSON(w, http.StatusOK, out)
}

func (a *API) handleGetSite(w http.ResponseWriter, r *http.Request) {
	siteID := r.PathValue("siteId")
	site, err := a.store.GetSiteByName(r.Context(), siteID)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}
	devices, err := a.store.ListDevicesBySite(r.Context(), site.ID)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}
	alertCounts, err := a.store.CountActiveAlertsBySite(r.Context())
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	now := a.now()
	deviceMap := make(map[string]models.DeviceSummary, len(devices))
	onlineCount := 0
	for _, d := range devices {
		online := derive.DeviceOnline(d.Status, d.LastSeen, now, a.onlineThreshold)
		if online {
			onlineCount++
		}
		key := derive.DeviceMapKey(d.IPAddress, d.Hostname)
		deviceMap[key] = toDeviceSummary(d, online)
	}

	loc := ""
	if site.Location != nil {
		loc = *site.Location
	}
	writeJSON(w, http.StatusOK, models.SiteDetail{
		SiteID:   site.Name,
		Location: derive.LocationOrName(loc, site.Name),
		Summary: models.SiteDetailSummary{
			TotalDevices: len(devices),
			OnlineCount:  onlineCount,
			IDFCount:     0,
			ActiveAlerts: alertCounts[site.ID],
		},
		Latest: models.SiteDetailLatest{Devices: deviceMap},
	})
}

func (a *API) handleListSiteDevices(w http.ResponseWriter, r *http.Request) {
	siteID := r.PathValue("siteId")
	site, err := a.store.GetSiteByName(r.Context(), siteID)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}
	devices, err := a.store.ListDevicesBySite(r.Context(), site.ID)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	now := a.now()
	out := make([]models.DeviceSummary, 0, len(devices))
	for _, d := range devices {
		online := derive.DeviceOnline(d.Status, d.LastSeen, now, a.onlineThreshold)
		out = append(out, toDeviceSummary(d, online))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("deviceId")
	siteName := r.URL.Query().Get("siteId")
	d, err := a.store.GetDevice(r.Context(), deviceID, siteName)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}
	online := derive.DeviceOnline(d.Status, d.LastSeen, a.now(), a.onlineThreshold)
	writeJSON(w, http.StatusOK, models.DeviceDetail{
		ID:         d.ID,
		SiteID:     d.SiteName,
		Hostname:   d.Hostname,
		IPAddress:  derive.NormalizeIP(d.IPAddress),
		Vendor:     d.Vendor,
		Model:      d.Model,
		Status:     derive.DeviceStatusCode(online),
		Role:       "",
		CPUPct:     0,
		MemoryPct:  0,
		UptimeDays: derive.UptimeDays(d.UptimeSeconds),
		LatencyMs:  nil,
		LastSeen:   d.LastSeen,
	})
}

func (a *API) handleListInterfaces(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("deviceId")
	siteName := r.URL.Query().Get("siteId")
	d, err := a.store.GetDevice(r.Context(), deviceID, siteName)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}
	ifaces, err := a.store.ListInterfaces(r.Context(), d.ID)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	out := make([]models.InterfaceInfo, 0, len(ifaces))
	for _, iface := range ifaces {
		info := models.InterfaceInfo{
			ID:       iface.ID,
			IfIndex:  iface.IfIndex,
			SpeedBps: iface.SpeedBps,
		}
		if iface.Name != nil {
			info.Name = *iface.Name
		}
		if iface.Description != nil {
			info.Description = *iface.Description
		}
		if iface.AdminStatus != nil {
			info.AdminStatus = *iface.AdminStatus
		}
		if iface.OperStatus != nil {
			info.OperStatus = *iface.OperStatus
		}
		out = append(out, info)
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) handleListMetrics(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("deviceId")
	siteName := r.URL.Query().Get("siteId")
	d, err := a.store.GetDevice(r.Context(), deviceID, siteName)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	metric := r.URL.Query().Get("metric")
	if metric == "" {
		metric = "uptime_seconds"
	}

	var start, end *time.Time
	if v := r.URL.Query().Get("start"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid start timestamp; use RFC3339")
			return
		}
		start = &t
	}
	if v := r.URL.Query().Get("end"); v != "" {
		t, err := time.Parse(time.RFC3339, v)
		if err != nil {
			writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "invalid end timestamp; use RFC3339")
			return
		}
		end = &t
	}

	samples, err := a.store.ListMetrics(r.Context(), d.ID, metric, start, end)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	points := make([]models.MetricPoint, 0, len(samples))
	for _, s := range samples {
		points = append(points, models.MetricPoint{TS: s.CollectedAt.UTC(), Value: s.Value})
	}

	writeJSON(w, http.StatusOK, models.MetricsResponse{
		DeviceID: d.Hostname,
		Metric:   metric,
		Start:    start,
		End:      end,
		Points:   points,
	})
}

func (a *API) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	rows, err := a.store.ListActiveAlerts(r.Context())
	if err != nil {
		a.writeStoreError(w, err)
		return
	}
	out := make([]models.AlertInfo, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.AlertInfo{
			ID:           row.ID,
			DeviceID:     row.DeviceID,
			InterfaceID:  row.InterfaceID,
			Severity:     row.Severity,
			AlertType:    row.AlertType,
			Message:      row.Message,
			Acknowledged: row.Acknowledged,
			CreatedAt:    row.CreatedAt.UTC(),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) handleTestConfig(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, models.TestConfig{
		Mode:           "live",
		PollingEnabled: true,
	})
}

func toDeviceSummary(d store.DeviceRow, online bool) models.DeviceSummary {
	return models.DeviceSummary{
		Hostname:   d.Hostname,
		Role:       "",
		Status:     derive.DeviceStatusCode(online),
		CPUPct:     0,
		MemoryPct:  0,
		UptimeDays: derive.UptimeDays(d.UptimeSeconds),
		LatencyMs:  nil,
	}
}

func (a *API) writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "RESOURCE_NOT_FOUND", "resource not found")
		return
	}
	if errors.Is(err, store.ErrAmbiguous) {
		writeError(w, http.StatusBadRequest, "VALIDATION_ERROR", "device id is ambiguous across sites; pass siteId query parameter")
		return
	}
	a.log.Error("database error", "err", err)
	writeError(w, http.StatusInternalServerError, "DATABASE_ERROR", "database error")
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(v)
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, models.APIError{
		Error: models.ErrorBody{Code: code, Message: message},
	})
}
