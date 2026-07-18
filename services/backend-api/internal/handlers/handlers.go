package handlers

import (
	"context"
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
		var counts derive.SiteHealthCounts
		var cpuVals, memVals []*float64
		for _, d := range devs {
			online := derive.DeviceOnline(d.Status, d.LastSeen, now, a.onlineThreshold)
			if online {
				onlineCount++
			}
			proj := projectDevice(d, online)
			counts.Accumulate(proj.Status, proj.UnavailableUpstreamIDs)
			cpuVals = append(cpuVals, d.CPUPct)
			memVals = append(memVals, d.MemoryPct)
		}

		loc := ""
		if site.Location != nil {
			loc = *site.Location
		}
		out[site.Name] = models.SiteOverview{
			Location: derive.LocationOrName(loc, site.Name),
			Type:     "",
			Status:   derive.SiteStatusFromHealth(counts),
			Latest: models.SiteOverviewLatest{
				Summary: models.SiteSummary{
					IDFCount:                0,
					DeviceCount:             deviceCount,
					OnlineCount:             onlineCount,
					AvgCPU:                  derive.AvgNullable(cpuVals),
					AvgMemory:               derive.AvgNullable(memVals),
					ActiveAlerts:            alertCounts[site.ID],
					HealthyCount:            counts.HealthyCount,
					WarningCount:            counts.WarningCount,
					CriticalCount:           counts.CriticalCount,
					UnknownCount:            counts.UnknownCount,
					DependencyImpactedCount: counts.DependencyImpactedCount,
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
	var counts derive.SiteHealthCounts
	for _, d := range devices {
		online := derive.DeviceOnline(d.Status, d.LastSeen, now, a.onlineThreshold)
		if online {
			onlineCount++
		}
		proj := projectDevice(d, online)
		counts.Accumulate(proj.Status, proj.UnavailableUpstreamIDs)
		key := derive.DeviceMapKey(d.IPAddress, d.Hostname)
		deviceMap[key] = toDeviceSummary(d, proj)
	}

	loc := ""
	if site.Location != nil {
		loc = *site.Location
	}
	writeJSON(w, http.StatusOK, models.SiteDetail{
		SiteID:   site.Name,
		Location: derive.LocationOrName(loc, site.Name),
		Summary: models.SiteDetailSummary{
			TotalDevices:            len(devices),
			OnlineCount:             onlineCount,
			IDFCount:                0,
			ActiveAlerts:            alertCounts[site.ID],
			HealthyCount:            counts.HealthyCount,
			WarningCount:            counts.WarningCount,
			CriticalCount:           counts.CriticalCount,
			UnknownCount:            counts.UnknownCount,
			DependencyImpactedCount: counts.DependencyImpactedCount,
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
		out = append(out, toDeviceSummary(d, projectDevice(d, online)))
	}
	writeJSON(w, http.StatusOK, out)
}

func (a *API) handleGetDevice(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("deviceId")
	siteName := r.URL.Query().Get("siteId")
	ctx := r.Context()
	d, err := a.store.GetDevice(ctx, deviceID, siteName)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}
	online := derive.DeviceOnline(d.Status, d.LastSeen, a.now(), a.onlineThreshold)
	proj := projectDevice(d, online)

	tempComponents, err := a.store.ListTemperatureComponents(ctx, d.ID)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}
	powerComponents, err := a.store.ListPowerComponents(ctx, d.ID)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	historyStart := a.now().Add(-store.DefaultHistoryWindow)
	history, err := a.loadDeviceHistory(ctx, d.ID, &historyStart)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	detail := models.DeviceDetail{
		ID:                     d.ID,
		SiteID:                 d.SiteName,
		Hostname:               d.Hostname,
		IPAddress:              derive.NormalizeIP(d.IPAddress),
		Vendor:                 d.Vendor,
		Model:                  d.Model,
		Status:                 proj.Status,
		StatusReason:           proj.StatusReason,
		FailureCount:           proj.FailureCount,
		UpstreamDeviceIDs:      emptyToNil(proj.UpstreamDeviceIDs),
		UnavailableUpstreamIDs: emptyToNil(proj.UnavailableUpstreamIDs),
		RootCauseDeviceIDs:     emptyToNil(proj.RootCauseDeviceIDs),
		Role:                   d.Role,
		CPUPct:                 d.CPUPct,
		MemoryPct:              d.MemoryPct,
		TemperatureC:           d.TemperatureC,
		UptimeDays:             derive.UptimeDays(d.UptimeSeconds),
		LatencyMs:              nil,
		LastSeen:               d.LastSeen,
		TemperatureComponents:  toComponents(tempComponents),
		PowerComponents:        toComponents(powerComponents),
		History:                history,
	}
	if d.Serial != nil {
		detail.SerialNumber = *d.Serial
	}
	if d.ProfileName != nil {
		detail.Profile = *d.ProfileName
	}
	if len(d.Capabilities) > 0 {
		detail.Capabilities = d.Capabilities
	}
	detail.SNMP = buildSNMP(d)

	writeJSON(w, http.StatusOK, detail)
}

func (a *API) loadDeviceHistory(ctx context.Context, deviceID uuid.UUID, start *time.Time) (*models.DeviceHistory, error) {
	cpu, err := a.store.ListMetrics(ctx, deviceID, "cpu_utilization_pct", start, nil)
	if err != nil {
		return nil, err
	}
	mem, err := a.store.ListMetrics(ctx, deviceID, "memory_utilization_pct", start, nil)
	if err != nil {
		return nil, err
	}
	temp, err := a.store.ListMetrics(ctx, deviceID, "primary_temperature_c", start, nil)
	if err != nil {
		return nil, err
	}
	uptime, err := a.store.ListMetrics(ctx, deviceID, "uptime_seconds", start, nil)
	if err != nil {
		return nil, err
	}
	return &models.DeviceHistory{
		CPU:         toMetricPoints(cpu),
		Memory:      toMetricPoints(mem),
		Temperature: toMetricPoints(temp),
		Uptime:      toMetricPoints(uptime),
	}, nil
}

func (a *API) handleListInterfaces(w http.ResponseWriter, r *http.Request) {
	deviceID := r.PathValue("deviceId")
	siteName := r.URL.Query().Get("siteId")
	ctx := r.Context()
	d, err := a.store.GetDevice(ctx, deviceID, siteName)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}
	ifaces, err := a.store.ListInterfaces(ctx, d.ID)
	if err != nil {
		a.writeStoreError(w, err)
		return
	}

	historyStart := a.now().Add(-store.DefaultHistoryWindow)
	out := make([]models.InterfaceInfo, 0, len(ifaces))
	for _, iface := range ifaces {
		info := models.InterfaceInfo{
			ID:          iface.ID,
			IfIndex:     iface.IfIndex,
			SpeedBps:    iface.SpeedBps,
			InOctets:    iface.InOctets,
			OutOctets:   iface.OutOctets,
			InErrors:    iface.InErrors,
			OutErrors:   iface.OutErrors,
			InDiscards:  iface.InDiscards,
			OutDiscards: iface.OutDiscards,
		}
		if iface.Name != nil {
			info.Name = *iface.Name
		}
		if iface.Description != nil {
			info.Description = *iface.Description
		}
		if iface.IfAlias != nil {
			info.IfAlias = *iface.IfAlias
		}
		if iface.IfType != nil {
			info.IfType = *iface.IfType
		}
		if iface.AdminStatus != nil {
			info.AdminStatus = *iface.AdminStatus
		}
		if iface.OperStatus != nil {
			info.OperStatus = *iface.OperStatus
		}
		traffic, err := a.store.ListInterfaceTrafficHistory(ctx, iface.ID, historyStart)
		if err != nil {
			a.writeStoreError(w, err)
			return
		}
		if len(traffic) > 0 {
			info.TrafficHistory = toMetricPoints(traffic)
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

	writeJSON(w, http.StatusOK, models.MetricsResponse{
		DeviceID: d.Hostname,
		Metric:   metric,
		Start:    start,
		End:      end,
		Points:   toMetricPoints(samples),
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

func projectDevice(d store.DeviceRow, online bool) derive.DeviceProjection {
	return derive.ProjectDeviceStatus(
		d.HealthState,
		d.HealthPresent,
		d.HealthReason,
		d.FailureCount,
		d.UpstreamIDs,
		d.UnavailableIDs,
		d.RootCauseIDs,
		online,
	)
}

func toDeviceSummary(d store.DeviceRow, proj derive.DeviceProjection) models.DeviceSummary {
	ip := derive.NormalizeIP(d.IPAddress)
	return models.DeviceSummary{
		DeviceID:               d.InventoryDeviceID,
		Hostname:               d.Hostname,
		IPAddress:              ip,
		Role:                   d.Role,
		Status:                 proj.Status,
		StatusReason:           proj.StatusReason,
		FailureCount:           proj.FailureCount,
		UpstreamDeviceIDs:      emptyToNil(proj.UpstreamDeviceIDs),
		UnavailableUpstreamIDs: emptyToNil(proj.UnavailableUpstreamIDs),
		RootCauseDeviceIDs:     emptyToNil(proj.RootCauseDeviceIDs),
		CPUPct:                 d.CPUPct,
		MemoryPct:              d.MemoryPct,
		UptimeDays:             derive.UptimeDays(d.UptimeSeconds),
		LatencyMs:              nil,
	}
}

func toComponents(rows []store.ComponentRow) []models.ComponentReading {
	if len(rows) == 0 {
		return nil
	}
	out := make([]models.ComponentReading, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.ComponentReading{
			ComponentID: r.ComponentID,
			Name:        r.Name,
			Index:       r.Index,
			Status:      r.Status,
			Value:       r.Value,
			Unit:        r.Unit,
		})
	}
	return out
}

func toMetricPoints(samples []store.MetricSampleRow) []models.MetricPoint {
	points := make([]models.MetricPoint, 0, len(samples))
	for _, s := range samples {
		points = append(points, models.MetricPoint{TS: s.CollectedAt.UTC(), Value: s.Value})
	}
	return points
}

func buildSNMP(d store.DeviceRow) *models.SNMPIdentity {
	snmp := &models.SNMPIdentity{}
	has := false
	if d.SysName != nil && *d.SysName != "" {
		snmp.SysName = *d.SysName
		has = true
	}
	if d.SysObjectID != nil && *d.SysObjectID != "" {
		snmp.SysObjectID = *d.SysObjectID
		has = true
	}
	if d.SysDescr != nil && *d.SysDescr != "" {
		snmp.SysDescr = *d.SysDescr
		has = true
	}
	if d.UptimeSeconds != nil {
		cs := int64(*d.UptimeSeconds * 100)
		snmp.SysUpTime = &cs
		has = true
	}
	if !has {
		return nil
	}
	return snmp
}

func emptyToNil(in []string) []string {
	if len(in) == 0 {
		return nil
	}
	return in
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
