package control

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/equate/ogsd/services/snmp-collector/internal/config"
)

func (s *Server) handle(ctx context.Context, req Request) (map[string]any, error) {
	select {
	case <-ctx.Done():
		return nil, newProtoError(CodeInternal, "request cancelled")
	default:
	}

	switch req.Method {
	case "status.summary":
		return s.handleStatusSummary()
	case "inventory.list":
		return s.handleInventoryList()
	case "device.get":
		return s.handleDeviceGet(req.Params)
	case "transport.get":
		return s.handleTransportGet()
	case "config.get":
		return s.handleConfigGet()
	case "discovery.status":
		return s.handleDiscoveryStatus()
	case "config.reload":
		return s.handleConfigReload()
	case "thresholds.prepare":
		return s.handleThresholdsPrepare(req.Params)
	case "thresholds.commit":
		return s.handleThresholdsCommit(req.Params)
	case "dependencies.prepare":
		return s.handleDependenciesPrepare(req.Params)
	case "dependencies.commit":
		return s.handleDependenciesCommit(req.Params)
	default:
		return nil, newProtoError(CodeMethodNotFound, "unknown method "+req.Method)
	}
}

func (s *Server) activeRevision() string {
	cfg := s.manager.Current()
	return config.ConfigRevision(cfg)
}

func (s *Server) handleStatusSummary() (map[string]any, error) {
	cfg := s.manager.Current()
	snap := s.status.Snapshot()
	healthSnap := s.health.Snapshot()
	stateCounts := map[string]int{}
	for state, count := range healthSnap.DevicesByState {
		stateCounts[string(state)] = count
	}
	lastPollIDs := make([]string, 0, len(snap.Devices))
	for id := range snap.Devices {
		lastPollIDs = append(lastPollIDs, id)
	}
	result := map[string]any{
		"site_id":              cfg.SiteID,
		"collector_id":         cfg.Collector.ID,
		"config_revision":      s.activeRevision(),
		"device_count":         len(cfg.Devices),
		"health_by_state":      stateCounts,
		"dependency_impacted":  healthSnap.DependencyImpacted,
		"pending_failures":     healthSnap.PendingFailures,
		"last_poll_device_ids": lastPollIDs,
	}
	if snap.Reload != nil {
		result["last_reload"] = snap.Reload
	}
	return result, nil
}

func (s *Server) handleInventoryList() (map[string]any, error) {
	cfg := s.manager.Current()
	rows := make([]map[string]any, 0, len(cfg.Devices))
	for _, device := range cfg.Devices {
		row := map[string]any{
			"id":                  device.ID,
			"host":                device.Host,
			"port":                device.Port,
			"community_env":       device.CommunityEnv,
			"upstream_device_ids": append([]string(nil), device.UpstreamDeviceIDs...),
		}
		if device.TemperatureWarningC != nil {
			row["temperature_warning_c"] = *device.TemperatureWarningC
		}
		if poll, ok := s.status.Device(device.ID); ok {
			row["last_poll"] = poll
		}
		if h, ok := s.health.Device(device.ID); ok && h.HasState {
			row["health"] = map[string]any{
				"state":         string(h.State),
				"reason":        string(h.Reason),
				"failure_count": h.FailureCount,
			}
		}
		rows = append(rows, row)
	}
	return map[string]any{
		"config_revision": s.activeRevision(),
		"devices":         rows,
	}, nil
}

func (s *Server) handleDeviceGet(params map[string]any) (map[string]any, error) {
	id, _ := params["device_id"].(string)
	if id == "" {
		return nil, newProtoError(CodeInvalidRequest, "device_id is required")
	}
	cfg := s.manager.Current()
	var device *config.DeviceConfig
	for i := range cfg.Devices {
		if cfg.Devices[i].ID == id {
			device = &cfg.Devices[i]
			break
		}
	}
	if device == nil {
		return nil, newProtoError(CodeNotFound, "device not found")
	}
	result := map[string]any{
		"id":                  device.ID,
		"host":                device.Host,
		"port":                device.Port,
		"community_env":       device.CommunityEnv,
		"version":             device.Version,
		"upstream_device_ids": append([]string(nil), device.UpstreamDeviceIDs...),
		"config_revision":     s.activeRevision(),
	}
	if device.TemperatureWarningC != nil {
		result["temperature_warning_c"] = *device.TemperatureWarningC
	} else {
		result["temperature_warning_c"] = cfg.Health.TemperatureWarningC
	}
	if poll, ok := s.status.Device(id); ok {
		result["last_poll"] = poll
	}
	if h, ok := s.health.Device(id); ok && h.HasState {
		result["health"] = map[string]any{
			"state":                           string(h.State),
			"reason":                          string(h.Reason),
			"failure_count":                   h.FailureCount,
			"upstream_device_ids":             append([]string(nil), h.UpstreamDeviceIDs...),
			"unavailable_upstream_device_ids": append([]string(nil), h.UnavailableUpstream...),
			"root_cause_device_ids":           append([]string(nil), h.RootCauseDeviceIDs...),
		}
	}
	return result, nil
}

func (s *Server) handleTransportGet() (map[string]any, error) {
	cfg := s.manager.Current()
	result := map[string]any{
		"publisher_mode":    cfg.Publisher.Mode,
		"telemetry_version": cfg.Publisher.TelemetryVersion,
		"config_revision":   s.activeRevision(),
	}
	if s.transport != nil {
		snap := s.transport.Snapshot()
		result["buffer_depth"] = snap.BufferDepth
		result["buffer_available"] = snap.BufferAvailable
		if snap.MQTTConnected != nil {
			result["mqtt_connected"] = *snap.MQTTConnected
		}
	}
	return result, nil
}

func (s *Server) handleConfigGet() (map[string]any, error) {
	cfg := s.manager.Current()
	result := map[string]any{
		"site_id":                     cfg.SiteID,
		"collector_id":                cfg.Collector.ID,
		"config_revision":             s.activeRevision(),
		"temperature_policy_revision": config.TemperaturePolicyRevision(cfg),
		"health": map[string]any{
			"temperature_warning_c": cfg.Health.TemperatureWarningC,
			"failure_threshold":     cfg.Health.FailureThreshold,
		},
		"discovery": map[string]any{
			"allowed_cidrs":         append([]string(nil), cfg.Discovery.AllowedCIDRs...),
			"max_probes_per_second": cfg.Discovery.MaxProbesPerSecond,
			"probe_burst":           cfg.Discovery.ProbeBurst,
			"max_targets":           cfg.Discovery.MaxTargets,
			"max_workers":           cfg.Discovery.MaxWorkers,
		},
		"admin": map[string]any{
			"listen":         cfg.Admin.Listen,
			"control_socket": cfg.Admin.ControlSocket,
		},
		"managed_path": cfg.ManagedInventoryPath(),
	}
	if snap := s.status.Snapshot(); snap.Reload != nil {
		result["last_reload"] = snap.Reload
	}
	return result, nil
}

func (s *Server) handleDiscoveryStatus() (map[string]any, error) {
	cfg := s.manager.Current()
	return map[string]any{
		"allowed_cidrs":         append([]string(nil), cfg.Discovery.AllowedCIDRs...),
		"max_probes_per_second": cfg.Discovery.MaxProbesPerSecond,
		"probe_burst":           cfg.Discovery.ProbeBurst,
		"max_targets":           cfg.Discovery.MaxTargets,
		"max_workers":           cfg.Discovery.MaxWorkers,
		"community_env":         cfg.Discovery.CommunityEnv,
		"note":                  "discovery runs via collector discover or future control orchestration; never auto-enrolls",
	}, nil
}

func (s *Server) handleConfigReload() (map[string]any, error) {
	err := s.manager.Reload()
	if err != nil {
		s.status.RecordReload(false, err.Error(), s.activeRevision())
		_ = s.audit.Record(AuditEntry{
			Action:  "config.reload",
			Success: false,
			Code:    CodeConfigReloadFailed,
			Message: err.Error(),
		})
		return nil, newProtoError(CodeConfigReloadFailed, err.Error())
	}
	rev := s.activeRevision()
	s.status.RecordReload(true, "", rev)
	s.status.SetRevision(rev)
	_ = s.audit.Record(AuditEntry{
		Action:   "config.reload",
		Success:  true,
		Revision: rev,
	})
	cfg := s.manager.Current()
	return map[string]any{
		"config_revision": rev,
		"device_count":    len(cfg.Devices),
	}, nil
}

func (s *Server) handleThresholdsPrepare(params map[string]any) (map[string]any, error) {
	if err := validateThresholdParams(params); err != nil {
		_ = s.audit.Record(AuditEntry{Action: "thresholds.prepare", Success: false, Code: CodeValidationFailed, Message: err.Error()})
		return nil, err
	}
	rev := s.activeRevision()
	item, err := s.pending.put("thresholds", rev, cloneParams(params))
	if err != nil {
		return nil, newProtoError(CodeInternal, err.Error())
	}
	_ = s.audit.Record(AuditEntry{Action: "thresholds.prepare", Success: true, Revision: rev})
	return map[string]any{
		"confirm_token": item.Token,
		"revision":      item.Revision,
		"expires_at":    item.ExpiresAt.Format(time.RFC3339Nano),
	}, nil
}

func (s *Server) handleThresholdsCommit(params map[string]any) (map[string]any, error) {
	token, _ := params["confirm_token"].(string)
	revision, _ := params["revision"].(string)
	if token == "" || revision == "" {
		_ = s.audit.Record(AuditEntry{Action: "thresholds.commit", Success: false, Code: CodeInvalidRequest, Message: "confirm_token and revision are required"})
		return nil, newProtoError(CodeInvalidRequest, "confirm_token and revision are required")
	}
	if revision != s.activeRevision() {
		_ = s.audit.Record(AuditEntry{Action: "thresholds.commit", Success: false, Code: CodeRevisionMismatch, Message: "active revision differs", Revision: s.activeRevision()})
		return nil, newProtoError(CodeRevisionMismatch, "configuration revision changed since prepare")
	}
	item, pe := s.pending.take(token, revision, "thresholds")
	if pe != nil {
		_ = s.audit.Record(AuditEntry{Action: "thresholds.commit", Success: false, Code: pe.Code, Message: pe.Message, Revision: revision})
		return nil, pe
	}
	if err := s.applyThresholdMutation(item.Payload); err != nil {
		_ = s.audit.Record(AuditEntry{Action: "thresholds.commit", Success: false, Code: CodeValidationFailed, Message: err.Error(), Revision: revision})
		return nil, err
	}
	_ = s.audit.Record(AuditEntry{Action: "thresholds.commit", Success: true, Revision: revision, Details: map[string]any{"written": true}})
	return map[string]any{
		"written":  true,
		"revision": revision,
		"note":     "call config.reload to activate",
	}, nil
}

func (s *Server) handleDependenciesPrepare(params map[string]any) (map[string]any, error) {
	if err := validateDependencyParams(params); err != nil {
		_ = s.audit.Record(AuditEntry{Action: "dependencies.prepare", Success: false, Code: CodeValidationFailed, Message: err.Error()})
		return nil, err
	}
	rev := s.activeRevision()
	item, err := s.pending.put("dependencies", rev, cloneParams(params))
	if err != nil {
		return nil, newProtoError(CodeInternal, err.Error())
	}
	_ = s.audit.Record(AuditEntry{Action: "dependencies.prepare", Success: true, Revision: rev})
	return map[string]any{
		"confirm_token": item.Token,
		"revision":      item.Revision,
		"expires_at":    item.ExpiresAt.Format(time.RFC3339Nano),
	}, nil
}

func (s *Server) handleDependenciesCommit(params map[string]any) (map[string]any, error) {
	token, _ := params["confirm_token"].(string)
	revision, _ := params["revision"].(string)
	if token == "" || revision == "" {
		_ = s.audit.Record(AuditEntry{Action: "dependencies.commit", Success: false, Code: CodeInvalidRequest, Message: "confirm_token and revision are required"})
		return nil, newProtoError(CodeInvalidRequest, "confirm_token and revision are required")
	}
	if revision != s.activeRevision() {
		_ = s.audit.Record(AuditEntry{Action: "dependencies.commit", Success: false, Code: CodeRevisionMismatch, Message: "active revision differs", Revision: s.activeRevision()})
		return nil, newProtoError(CodeRevisionMismatch, "configuration revision changed since prepare")
	}
	item, pe := s.pending.take(token, revision, "dependencies")
	if pe != nil {
		_ = s.audit.Record(AuditEntry{Action: "dependencies.commit", Success: false, Code: pe.Code, Message: pe.Message, Revision: revision})
		return nil, pe
	}
	if err := s.applyDependencyMutation(item.Payload); err != nil {
		_ = s.audit.Record(AuditEntry{Action: "dependencies.commit", Success: false, Code: CodeValidationFailed, Message: err.Error(), Revision: revision})
		return nil, err
	}
	_ = s.audit.Record(AuditEntry{Action: "dependencies.commit", Success: true, Revision: revision, Details: map[string]any{"written": true}})
	return map[string]any{
		"written":  true,
		"revision": revision,
		"note":     "call config.reload to activate",
	}, nil
}

func validateThresholdParams(params map[string]any) error {
	if _, ok := params["temperature_warning_c"]; ok {
		if _, err := asFloat(params["temperature_warning_c"]); err != nil {
			return newProtoError(CodeValidationFailed, "temperature_warning_c must be a number")
		}
	}
	if raw, ok := params["device_id"]; ok && raw != nil {
		id, _ := raw.(string)
		if id == "" {
			return newProtoError(CodeValidationFailed, "device_id must be a non-empty string when set")
		}
	}
	if params["temperature_warning_c"] == nil && params["device_id"] == nil {
		return newProtoError(CodeValidationFailed, "temperature_warning_c is required")
	}
	if params["temperature_warning_c"] == nil {
		return newProtoError(CodeValidationFailed, "temperature_warning_c is required")
	}
	return nil
}

func validateDependencyParams(params map[string]any) error {
	id, _ := params["device_id"].(string)
	if id == "" {
		return newProtoError(CodeValidationFailed, "device_id is required")
	}
	if _, ok := params["upstream_device_ids"]; !ok {
		return newProtoError(CodeValidationFailed, "upstream_device_ids is required")
	}
	if _, err := asStringSlice(params["upstream_device_ids"]); err != nil {
		return newProtoError(CodeValidationFailed, "upstream_device_ids must be a string array")
	}
	return nil
}

func (s *Server) applyThresholdMutation(params map[string]any) error {
	cfg := s.manager.Current()
	managedPath := cfg.ManagedInventoryPath()
	if managedPath == "" {
		return newProtoError(CodeValidationFailed, "inventory.managed_path is not configured")
	}
	value, err := asFloat(params["temperature_warning_c"])
	if err != nil {
		return newProtoError(CodeValidationFailed, err.Error())
	}
	deviceID, _ := params["device_id"].(string)
	if deviceID != "" {
		if err := s.validateDeviceInInventory(deviceID); err != nil {
			return err
		}
	}

	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()

	doc, err := config.ReadManagedDocument(managedPath)
	if err != nil {
		return newProtoError(CodeValidationFailed, err.Error())
	}
	if deviceID == "" {
		doc.Health.TemperatureWarningC = &value
	} else {
		found := false
		for i := range doc.Devices {
			if doc.Devices[i].ID == deviceID {
				v := value
				doc.Devices[i].TemperatureWarningC = &v
				found = true
				break
			}
		}
		if !found {
			doc.Devices = append(doc.Devices, config.DeviceConfig{
				ID:                  deviceID,
				TemperatureWarningC: &value,
			})
		}
	}
	if err := config.WriteManagedDocument(managedPath, doc); err != nil {
		return newProtoError(CodeValidationFailed, err.Error())
	}
	return nil
}

func (s *Server) applyDependencyMutation(params map[string]any) error {
	cfg := s.manager.Current()
	managedPath := cfg.ManagedInventoryPath()
	if managedPath == "" {
		return newProtoError(CodeValidationFailed, "inventory.managed_path is not configured")
	}
	deviceID, _ := params["device_id"].(string)
	upstreams, err := asStringSlice(params["upstream_device_ids"])
	if err != nil {
		return newProtoError(CodeValidationFailed, err.Error())
	}
	if err := s.validateDependencyTargets(deviceID, upstreams); err != nil {
		return err
	}

	s.mutationMu.Lock()
	defer s.mutationMu.Unlock()

	doc, err := config.ReadManagedDocument(managedPath)
	if err != nil {
		return newProtoError(CodeValidationFailed, err.Error())
	}
	found := false
	for i := range doc.Devices {
		if doc.Devices[i].ID == deviceID {
			doc.Devices[i].UpstreamDeviceIDs = upstreams
			found = true
			break
		}
	}
	if !found {
		doc.Devices = append(doc.Devices, config.DeviceConfig{
			ID:                deviceID,
			UpstreamDeviceIDs: upstreams,
		})
	}
	if err := config.WriteManagedDocument(managedPath, doc); err != nil {
		return newProtoError(CodeValidationFailed, err.Error())
	}
	return nil
}

func (s *Server) validateDeviceInInventory(deviceID string) error {
	for _, device := range s.manager.Current().Devices {
		if device.ID == deviceID {
			return nil
		}
	}
	return newProtoError(CodeValidationFailed, "device_id not found in active inventory")
}

func (s *Server) validateDependencyTargets(deviceID string, upstreams []string) error {
	if err := s.validateDeviceInInventory(deviceID); err != nil {
		return err
	}
	if err := s.manager.Current().ValidatePendingDependencyMutation(deviceID, upstreams); err != nil {
		return newProtoError(CodeValidationFailed, err.Error())
	}
	return nil
}

func cloneParams(params map[string]any) map[string]any {
	out := make(map[string]any, len(params))
	for k, v := range params {
		out[k] = v
	}
	return out
}

func asFloat(v any) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case json.Number:
		return n.Float64()
	default:
		return 0, fmt.Errorf("expected number")
	}
}

func asStringSlice(v any) ([]string, error) {
	switch list := v.(type) {
	case []string:
		return append([]string(nil), list...), nil
	case []any:
		out := make([]string, 0, len(list))
		for _, item := range list {
			s, ok := item.(string)
			if !ok {
				return nil, fmt.Errorf("expected string array")
			}
			out = append(out, s)
		}
		return out, nil
	case nil:
		return []string{}, nil
	default:
		return nil, fmt.Errorf("expected string array")
	}
}
