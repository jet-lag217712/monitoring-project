package store

import (
	"context"
	"fmt"

	"github.com/equate/ogsd/services/ingestion-service/internal/transform"
	"github.com/equate/ogsd/services/ingestion-service/internal/validate"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// PersistDeviceTelemetry upserts enriched device inventory, samples, and components.
func (s *Store) PersistDeviceTelemetry(ctx context.Context, sample transform.DeviceTelemetrySample) (Result, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	dup, err := insertIngestedEvent(ctx, tx, sample.EventID, "device_telemetry", sample.SiteName, sample.CollectorID, sample.DeviceID, sample.ObservedAt)
	if err != nil {
		return 0, err
	}
	if dup {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit: %w", err)
		}
		return ResultDuplicate, nil
	}

	if err := upsertSite(ctx, tx, sample.SiteUUID, sample.SiteName); err != nil {
		return 0, err
	}
	if err := upsertDeviceV2(ctx, tx, sample); err != nil {
		return 0, err
	}

	metrics := []struct {
		name  string
		value *float64
	}{
		{"uptime_seconds", floatPtr(sample.UptimeSeconds)},
		{"cpu_utilization_pct", sample.CPUUtilizationPct},
		{"memory_utilization_pct", sample.MemoryUtilizationPct},
		{"primary_temperature_c", sample.PrimaryTemperatureC},
	}
	for _, m := range metrics {
		if m.value == nil {
			continue
		}
		if err := insertMetricSample(ctx, tx, sample.DeviceUUID, m.name, *m.value, sample.ObservedAt); err != nil {
			return 0, err
		}
	}

	for _, c := range sample.TemperatureComponents {
		if err := upsertTempComponent(ctx, tx, sample.DeviceUUID, c); err != nil {
			return 0, err
		}
		if err := insertTempReading(ctx, tx, sample.DeviceUUID, sample.EventID, c); err != nil {
			return 0, err
		}
	}
	for _, c := range sample.PowerComponents {
		if err := upsertPowerComponent(ctx, tx, sample.DeviceUUID, c); err != nil {
			return 0, err
		}
		if err := insertPowerReading(ctx, tx, sample.DeviceUUID, sample.EventID, c); err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return ResultInserted, nil
}

// PersistInterfaceTelemetry upserts interface metadata and counters.
func (s *Store) PersistInterfaceTelemetry(ctx context.Context, sample transform.InterfaceTelemetrySample) (Result, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	dup, err := insertIngestedEvent(ctx, tx, sample.EventID, "interface_telemetry", sample.SiteName, sample.CollectorID, sample.DeviceID, sample.ObservedAt)
	if err != nil {
		return 0, err
	}
	if dup {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit: %w", err)
		}
		return ResultDuplicate, nil
	}

	if err := upsertSite(ctx, tx, sample.SiteUUID, sample.SiteName); err != nil {
		return 0, err
	}
	if err := upsertDevice(ctx, tx, sample.DeviceUUID, sample.SiteUUID, sample.DeviceID, "", sample.ObservedAt); err != nil {
		return 0, err
	}
	ifaceID, err := upsertInterfaceV2(ctx, tx, sample)
	if err != nil {
		return 0, err
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO interface_samples (
			interface_id, in_octets, out_octets, in_errors, out_errors, in_discards, out_discards, collected_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (interface_id, collected_at) DO NOTHING
	`, ifaceID, sample.InOctets, sample.OutOctets, sample.InErrors, sample.OutErrors, sample.InDiscards, sample.OutDiscards, sample.ObservedAt)
	if err != nil {
		return 0, fmt.Errorf("insert interface_samples: %w", err)
	}
	_ = tag

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return ResultInserted, nil
}

// PersistHealth upserts current health and appends history.
func (s *Store) PersistHealth(ctx context.Context, sample transform.HealthSample) (Result, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	dup, err := insertIngestedEvent(ctx, tx, sample.EventID, "health_state", sample.SiteName, sample.CollectorID, sample.DeviceID, sample.ObservedAt)
	if err != nil {
		return 0, err
	}
	if dup {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit: %w", err)
		}
		return ResultDuplicate, nil
	}

	if err := upsertSite(ctx, tx, sample.SiteUUID, sample.SiteName); err != nil {
		return 0, err
	}
	if err := upsertDevice(ctx, tx, sample.DeviceUUID, sample.SiteUUID, sample.DeviceID, "", sample.ObservedAt); err != nil {
		return 0, err
	}

	upstream := nonNilStrings(sample.UpstreamDeviceIDs)
	unavailable := nonNilStrings(sample.UnavailableUpstreamDeviceIDs)
	rootCause := nonNilStrings(sample.RootCauseDeviceIDs)

	_, err = tx.Exec(ctx, `
		INSERT INTO device_health_history (
			device_id, site_id, state, reason, transition, previous_state,
			failure_count, failure_threshold, temperature_c, temperature_warning_c,
			temperature_policy_revision, upstream_device_ids, unavailable_upstream_device_ids,
			root_cause_device_ids, alerts_enabled, observed_at, event_id, config_revision
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18
		)
	`, sample.DeviceUUID, sample.SiteUUID, sample.State, sample.Reason, sample.Transition, sample.PreviousState,
		sample.FailureCount, sample.FailureThreshold, sample.TemperatureC, sample.TemperatureWarningC,
		sample.TemperaturePolicyRevision, upstream, unavailable,
		rootCause, sample.AlertsEnabled, sample.ObservedAt, sample.EventID, sample.ConfigRevision)
	if err != nil {
		return 0, fmt.Errorf("insert device_health_history: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO device_health_current (
			device_id, site_id, state, reason, transition, previous_state,
			failure_count, failure_threshold, temperature_c, temperature_warning_c,
			temperature_policy_revision, upstream_device_ids, unavailable_upstream_device_ids,
			root_cause_device_ids, alerts_enabled, observed_at, event_id, config_revision, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,NOW()
		)
		ON CONFLICT (device_id) DO UPDATE SET
			site_id = EXCLUDED.site_id,
			state = EXCLUDED.state,
			reason = EXCLUDED.reason,
			transition = EXCLUDED.transition,
			previous_state = EXCLUDED.previous_state,
			failure_count = EXCLUDED.failure_count,
			failure_threshold = EXCLUDED.failure_threshold,
			temperature_c = EXCLUDED.temperature_c,
			temperature_warning_c = EXCLUDED.temperature_warning_c,
			temperature_policy_revision = EXCLUDED.temperature_policy_revision,
			upstream_device_ids = EXCLUDED.upstream_device_ids,
			unavailable_upstream_device_ids = EXCLUDED.unavailable_upstream_device_ids,
			root_cause_device_ids = EXCLUDED.root_cause_device_ids,
			alerts_enabled = EXCLUDED.alerts_enabled,
			observed_at = EXCLUDED.observed_at,
			event_id = EXCLUDED.event_id,
			config_revision = EXCLUDED.config_revision,
			updated_at = NOW()
		WHERE device_health_current.observed_at <= EXCLUDED.observed_at
	`, sample.DeviceUUID, sample.SiteUUID, sample.State, sample.Reason, sample.Transition, sample.PreviousState,
		sample.FailureCount, sample.FailureThreshold, sample.TemperatureC, sample.TemperatureWarningC,
		sample.TemperaturePolicyRevision, upstream, unavailable,
		rootCause, sample.AlertsEnabled, sample.ObservedAt, sample.EventID, sample.ConfigRevision)
	if err != nil {
		return 0, fmt.Errorf("upsert device_health_current: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return ResultInserted, nil
}

// PersistHeartbeat appends heartbeat history and updates current status when newer.
func (s *Store) PersistHeartbeat(ctx context.Context, sample transform.HeartbeatSample) (Result, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	dup, err := insertIngestedEvent(ctx, tx, sample.EventID, "collector_heartbeat", sample.SiteName, sample.CollectorID, "", sample.ObservedAt)
	if err != nil {
		return 0, err
	}
	if dup {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit: %w", err)
		}
		return ResultDuplicate, nil
	}

	if err := upsertSite(ctx, tx, sample.SiteUUID, sample.SiteName); err != nil {
		return 0, err
	}
	if err := upsertCollector(ctx, tx, sample.CollectorUUID, sample.SiteUUID, sample.CollectorID); err != nil {
		return 0, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO collector_heartbeat_history (
			collector_uuid, site_id, collector_id, hostname, version, git_commit, build_time,
			uptime_seconds, sqlite_queue_depth, memory_usage_bytes, goroutine_count,
			observed_at, event_id, config_revision
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14
		)
	`, sample.CollectorUUID, sample.SiteUUID, sample.CollectorID, sample.Hostname, sample.Version, sample.GitCommit, sample.BuildTime,
		sample.UptimeSeconds, sample.SQLiteQueueDepth, sample.MemoryUsageBytes, sample.GoroutineCount,
		sample.ObservedAt, sample.EventID, sample.ConfigRevision)
	if err != nil {
		return 0, fmt.Errorf("insert collector_heartbeat_history: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO collector_status_current (
			collector_uuid, site_id, collector_id, hostname, version, git_commit, build_time,
			uptime_seconds, sqlite_queue_depth, memory_usage_bytes, goroutine_count,
			observed_at, event_id, config_revision, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,NOW()
		)
		ON CONFLICT (collector_uuid) DO UPDATE SET
			site_id = EXCLUDED.site_id,
			collector_id = EXCLUDED.collector_id,
			hostname = EXCLUDED.hostname,
			version = EXCLUDED.version,
			git_commit = EXCLUDED.git_commit,
			build_time = EXCLUDED.build_time,
			uptime_seconds = EXCLUDED.uptime_seconds,
			sqlite_queue_depth = EXCLUDED.sqlite_queue_depth,
			memory_usage_bytes = EXCLUDED.memory_usage_bytes,
			goroutine_count = EXCLUDED.goroutine_count,
			observed_at = EXCLUDED.observed_at,
			event_id = EXCLUDED.event_id,
			config_revision = EXCLUDED.config_revision,
			updated_at = NOW()
		WHERE collector_status_current.observed_at < EXCLUDED.observed_at
	`, sample.CollectorUUID, sample.SiteUUID, sample.CollectorID, sample.Hostname, sample.Version, sample.GitCommit, sample.BuildTime,
		sample.UptimeSeconds, sample.SQLiteQueueDepth, sample.MemoryUsageBytes, sample.GoroutineCount,
		sample.ObservedAt, sample.EventID, sample.ConfigRevision)
	if err != nil {
		return 0, fmt.Errorf("upsert collector_status_current: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return ResultInserted, nil
}

func insertIngestedEvent(ctx context.Context, tx pgx.Tx, eventID uuid.UUID, eventType, siteID, collectorID, deviceID string, observedAt interface{}) (duplicate bool, err error) {
	var collector *string
	if collectorID != "" {
		collector = &collectorID
	}
	var device *string
	if deviceID != "" {
		device = &deviceID
	}
	// ON CONFLICT DO NOTHING keeps the transaction usable on QoS 1 redelivery.
	// A plain unique-violation abort would make the later Commit fail with
	// "commit unexpectedly resulted in rollback".
	tag, err := tx.Exec(ctx, `
		INSERT INTO ingested_events (event_id, event_type, site_id, collector_id, device_id, observed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (event_id) DO NOTHING
	`, eventID, eventType, siteID, collector, device, observedAt)
	if err != nil {
		return false, fmt.Errorf("insert ingested_events: %w", err)
	}
	return tag.RowsAffected() == 0, nil
}

func upsertDeviceV2(ctx context.Context, tx pgx.Tx, sample transform.DeviceTelemetrySample) error {
	vendor := sample.Vendor
	if vendor == "" {
		vendor = "unknown"
	}
	model := sample.Model
	if model == "" {
		model = "unknown"
	}
	_, err := tx.Exec(ctx, `
		INSERT INTO devices (
			id, site_id, hostname, ip_address, vendor, model, snmp_version, status, last_seen,
			serial, sys_object_id, sys_name, sys_descr, profile_name, capabilities,
			collector_id, config_revision, last_observed_at, role, inventory_device_id
		) VALUES (
			$1, $2, $3, COALESCE(NULLIF($4, ''), '0.0.0.0')::inet, $5, $6, $7, 'online', $8,
			NULLIF($9, ''), $10, $11, $12, $13, $14, $15, $16, $8, NULLIF($17, ''), NULLIF($18, '')
		)
		ON CONFLICT (id) DO UPDATE SET
			last_seen = CASE WHEN devices.last_observed_at < EXCLUDED.last_observed_at THEN EXCLUDED.last_seen ELSE devices.last_seen END,
			status = CASE WHEN devices.last_observed_at < EXCLUDED.last_observed_at THEN 'online' ELSE devices.status END,
			hostname = CASE WHEN devices.last_observed_at < EXCLUDED.last_observed_at THEN EXCLUDED.hostname ELSE devices.hostname END,
			ip_address = CASE
				WHEN devices.last_observed_at < EXCLUDED.last_observed_at AND $4 <> '' THEN EXCLUDED.ip_address
				ELSE devices.ip_address
			END,
			role = CASE
				WHEN devices.last_observed_at < EXCLUDED.last_observed_at AND NULLIF($17, '') IS NOT NULL THEN EXCLUDED.role
				ELSE devices.role
			END,
			inventory_device_id = CASE
				WHEN devices.last_observed_at < EXCLUDED.last_observed_at AND NULLIF($18, '') IS NOT NULL THEN EXCLUDED.inventory_device_id
				ELSE devices.inventory_device_id
			END,
			vendor = CASE
				WHEN devices.last_observed_at < EXCLUDED.last_observed_at AND $5 <> 'unknown' THEN EXCLUDED.vendor
				ELSE devices.vendor
			END,
			model = CASE
				WHEN devices.last_observed_at < EXCLUDED.last_observed_at AND $6 <> 'unknown' THEN EXCLUDED.model
				ELSE devices.model
			END,
			snmp_version = CASE WHEN devices.last_observed_at < EXCLUDED.last_observed_at THEN EXCLUDED.snmp_version ELSE devices.snmp_version END,
			serial = CASE WHEN devices.last_observed_at < EXCLUDED.last_observed_at THEN COALESCE(EXCLUDED.serial, devices.serial) ELSE devices.serial END,
			sys_object_id = CASE WHEN devices.last_observed_at < EXCLUDED.last_observed_at THEN EXCLUDED.sys_object_id ELSE devices.sys_object_id END,
			sys_name = CASE WHEN devices.last_observed_at < EXCLUDED.last_observed_at THEN EXCLUDED.sys_name ELSE devices.sys_name END,
			sys_descr = CASE WHEN devices.last_observed_at < EXCLUDED.last_observed_at THEN EXCLUDED.sys_descr ELSE devices.sys_descr END,
			profile_name = CASE WHEN devices.last_observed_at < EXCLUDED.last_observed_at THEN EXCLUDED.profile_name ELSE devices.profile_name END,
			capabilities = CASE WHEN devices.last_observed_at < EXCLUDED.last_observed_at THEN EXCLUDED.capabilities ELSE devices.capabilities END,
			collector_id = CASE WHEN devices.last_observed_at < EXCLUDED.last_observed_at THEN EXCLUDED.collector_id ELSE devices.collector_id END,
			config_revision = CASE WHEN devices.last_observed_at < EXCLUDED.last_observed_at THEN EXCLUDED.config_revision ELSE devices.config_revision END,
			last_observed_at = GREATEST(devices.last_observed_at, EXCLUDED.last_observed_at)
	`, sample.DeviceUUID, sample.SiteUUID, sample.Hostname, sample.ManagementAddress, vendor, model, sample.SNMPVersion, sample.ObservedAt,
		sample.Serial, sample.SysObjectID, sample.SysName, sample.SysDescr, sample.ProfileName, sample.Capabilities,
		sample.CollectorID, sample.ConfigRevision, sample.Role, sample.DeviceID)
	if err != nil {
		return fmt.Errorf("upsert device v2: %w", err)
	}
	return nil
}

func upsertInterfaceV2(ctx context.Context, tx pgx.Tx, sample transform.InterfaceTelemetrySample) (uuid.UUID, error) {
	var resolved uuid.UUID
	err := tx.QueryRow(ctx, `
		INSERT INTO interfaces (
			id, device_id, if_index, name, description, admin_status, oper_status, speed_bps,
			if_alias, if_type, last_observed_at
		) VALUES (
			$1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8, NULLIF($5, ''), NULLIF($9, ''), $10
		)
		ON CONFLICT (device_id, if_index) DO UPDATE SET
			name = CASE WHEN interfaces.last_observed_at < EXCLUDED.last_observed_at THEN EXCLUDED.name ELSE interfaces.name END,
			description = CASE WHEN interfaces.last_observed_at < EXCLUDED.last_observed_at THEN COALESCE(EXCLUDED.description, interfaces.description) ELSE interfaces.description END,
			admin_status = CASE WHEN interfaces.last_observed_at < EXCLUDED.last_observed_at THEN EXCLUDED.admin_status ELSE interfaces.admin_status END,
			oper_status = CASE WHEN interfaces.last_observed_at < EXCLUDED.last_observed_at THEN EXCLUDED.oper_status ELSE interfaces.oper_status END,
			speed_bps = CASE WHEN interfaces.last_observed_at < EXCLUDED.last_observed_at THEN EXCLUDED.speed_bps ELSE interfaces.speed_bps END,
			if_alias = CASE WHEN interfaces.last_observed_at < EXCLUDED.last_observed_at THEN EXCLUDED.if_alias ELSE interfaces.if_alias END,
			if_type = CASE WHEN interfaces.last_observed_at < EXCLUDED.last_observed_at THEN EXCLUDED.if_type ELSE interfaces.if_type END,
			last_observed_at = GREATEST(interfaces.last_observed_at, EXCLUDED.last_observed_at)
		RETURNING id
	`, sample.InterfaceUUID, sample.DeviceUUID, sample.IfIndex, sample.Name, sample.Alias,
		sample.AdminStatus, sample.OperStatus, sample.SpeedBps, sample.Type, sample.ObservedAt).Scan(&resolved)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert interface v2: %w", err)
	}
	return resolved, nil
}

func upsertCollector(ctx context.Context, tx pgx.Tx, id, siteID uuid.UUID, collectorID string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO collectors (id, site_id, collector_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (site_id, collector_id) DO NOTHING
	`, id, siteID, collectorID)
	if err != nil {
		return fmt.Errorf("upsert collector: %w", err)
	}
	return nil
}

func insertMetricSample(ctx context.Context, tx pgx.Tx, deviceID uuid.UUID, metricName string, value float64, observedAt interface{}) error {
	metricTypeID, err := lookupMetricType(ctx, tx, metricName)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO metric_samples (device_id, metric_type_id, value, collected_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (device_id, metric_type_id, collected_at) DO NOTHING
	`, deviceID, metricTypeID, value, observedAt)
	if err != nil {
		return fmt.Errorf("insert metric_samples %s: %w", metricName, err)
	}
	return nil
}

func upsertTempComponent(ctx context.Context, tx pgx.Tx, deviceID uuid.UUID, c validate.ComponentReading) error {
	id := transform.ComponentUUID(deviceID, "temperature", c.ComponentID)
	_, err := tx.Exec(ctx, `
		INSERT INTO device_temperature_components (id, device_id, component_id, name, component_index)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (device_id, component_id) DO UPDATE SET
			name = EXCLUDED.name,
			component_index = EXCLUDED.component_index
	`, id, deviceID, c.ComponentID, c.Name, c.Index)
	if err != nil {
		return fmt.Errorf("upsert temperature component: %w", err)
	}
	return nil
}

func insertTempReading(ctx context.Context, tx pgx.Tx, deviceID, eventID uuid.UUID, c validate.ComponentReading) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO device_temperature_readings (
			device_id, component_id, value, unit, status, observed_at, event_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (device_id, component_id, observed_at) DO NOTHING
	`, deviceID, c.ComponentID, c.Value, c.Unit, c.Status, c.ObservedAt, eventID)
	if err != nil {
		return fmt.Errorf("insert temperature reading: %w", err)
	}
	return nil
}

func upsertPowerComponent(ctx context.Context, tx pgx.Tx, deviceID uuid.UUID, c validate.ComponentReading) error {
	id := transform.ComponentUUID(deviceID, "power", c.ComponentID)
	_, err := tx.Exec(ctx, `
		INSERT INTO device_power_components (id, device_id, component_id, name, component_index)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (device_id, component_id) DO UPDATE SET
			name = EXCLUDED.name,
			component_index = EXCLUDED.component_index
	`, id, deviceID, c.ComponentID, c.Name, c.Index)
	if err != nil {
		return fmt.Errorf("upsert power component: %w", err)
	}
	return nil
}

func insertPowerReading(ctx context.Context, tx pgx.Tx, deviceID, eventID uuid.UUID, c validate.ComponentReading) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO device_power_readings (
			device_id, component_id, value, unit, status, observed_at, event_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (device_id, component_id, observed_at) DO NOTHING
	`, deviceID, c.ComponentID, c.Value, c.Unit, c.Status, c.ObservedAt, eventID)
	if err != nil {
		return fmt.Errorf("insert power reading: %w", err)
	}
	return nil
}

func floatPtr(v float64) *float64 { return &v }

// nonNilStrings copies in and always returns a non-nil slice so pgx encodes
// TEXT[] as '{}' instead of NULL. append([]string(nil), empty...) yields nil.
func nonNilStrings(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	return out
}
