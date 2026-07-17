package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/equate/ogsd/services/ingestion-service/internal/transform"
	"github.com/equate/ogsd/services/ingestion-service/internal/validate"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

	_, err = tx.Exec(ctx, `
		INSERT INTO device_health_history (
			device_id, site_id, state, reason, transition, previous_state,
			failure_count, failure_threshold, temperature_c, temperature_warning_c,
			temperature_policy_revision, upstream_device_ids, unavailable_upstream_device_ids,
			root_cause_device_ids, observed_at, event_id, config_revision
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17
		)
	`, sample.DeviceUUID, sample.SiteUUID, sample.State, sample.Reason, sample.Transition, sample.PreviousState,
		sample.FailureCount, sample.FailureThreshold, sample.TemperatureC, sample.TemperatureWarningC,
		sample.TemperaturePolicyRevision, sample.UpstreamDeviceIDs, sample.UnavailableUpstreamDeviceIDs,
		sample.RootCauseDeviceIDs, sample.ObservedAt, sample.EventID, sample.ConfigRevision)
	if err != nil {
		return 0, fmt.Errorf("insert device_health_history: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO device_health_current (
			device_id, site_id, state, reason, transition, previous_state,
			failure_count, failure_threshold, temperature_c, temperature_warning_c,
			temperature_policy_revision, upstream_device_ids, unavailable_upstream_device_ids,
			root_cause_device_ids, observed_at, event_id, config_revision, updated_at
		) VALUES (
			$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,NOW()
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
			observed_at = EXCLUDED.observed_at,
			event_id = EXCLUDED.event_id,
			config_revision = EXCLUDED.config_revision,
			updated_at = NOW()
		WHERE device_health_current.observed_at <= EXCLUDED.observed_at
	`, sample.DeviceUUID, sample.SiteUUID, sample.State, sample.Reason, sample.Transition, sample.PreviousState,
		sample.FailureCount, sample.FailureThreshold, sample.TemperatureC, sample.TemperatureWarningC,
		sample.TemperaturePolicyRevision, sample.UpstreamDeviceIDs, sample.UnavailableUpstreamDeviceIDs,
		sample.RootCauseDeviceIDs, sample.ObservedAt, sample.EventID, sample.ConfigRevision)
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
	_, err = tx.Exec(ctx, `
		INSERT INTO ingested_events (event_id, event_type, site_id, collector_id, device_id, observed_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, eventID, eventType, siteID, collector, device, observedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return true, nil
		}
		return false, fmt.Errorf("insert ingested_events: %w", err)
	}
	return false, nil
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
			collector_id, config_revision, last_observed_at
		) VALUES (
			$1, $2, $3, '0.0.0.0'::inet, $4, $5, $6, 'online', $7,
			NULLIF($8, ''), $9, $10, $11, $12, $13, $14, $15, $7
		)
		ON CONFLICT (id) DO UPDATE SET
			last_seen = EXCLUDED.last_seen,
			status = 'online',
			hostname = EXCLUDED.hostname,
			vendor = CASE WHEN $4 <> 'unknown' THEN EXCLUDED.vendor ELSE devices.vendor END,
			model = CASE WHEN $5 <> 'unknown' THEN EXCLUDED.model ELSE devices.model END,
			snmp_version = EXCLUDED.snmp_version,
			serial = COALESCE(EXCLUDED.serial, devices.serial),
			sys_object_id = EXCLUDED.sys_object_id,
			sys_name = EXCLUDED.sys_name,
			sys_descr = EXCLUDED.sys_descr,
			profile_name = EXCLUDED.profile_name,
			capabilities = EXCLUDED.capabilities,
			collector_id = EXCLUDED.collector_id,
			config_revision = EXCLUDED.config_revision,
			last_observed_at = EXCLUDED.last_observed_at
	`, sample.DeviceUUID, sample.SiteUUID, sample.Hostname, vendor, model, sample.SNMPVersion, sample.ObservedAt,
		sample.Serial, sample.SysObjectID, sample.SysName, sample.SysDescr, sample.ProfileName, sample.Capabilities,
		sample.CollectorID, sample.ConfigRevision)
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
			name = EXCLUDED.name,
			description = COALESCE(EXCLUDED.description, interfaces.description),
			admin_status = EXCLUDED.admin_status,
			oper_status = EXCLUDED.oper_status,
			speed_bps = EXCLUDED.speed_bps,
			if_alias = EXCLUDED.if_alias,
			if_type = EXCLUDED.if_type,
			last_observed_at = EXCLUDED.last_observed_at
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
