package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/equate/ogsd/services/ingestion-service/internal/transform"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrUnknownMetricType is returned when metric_types has no matching name.
var ErrUnknownMetricType = errors.New("unknown metric type")

// Result describes the outcome of a persist attempt.
type Result int

const (
	// ResultInserted means a new sample row was written.
	ResultInserted Result = iota
	// ResultDuplicate means the sample already existed (idempotent success).
	ResultDuplicate
)

// Store persists telemetry into PostgreSQL.
type Store struct {
	pool *pgxpool.Pool
}

// New wraps an existing connection pool.
func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// Open creates a pool from a PostgreSQL URL.
func Open(ctx context.Context, databaseURL string, maxConns, minConns int32, maxLifetime time.Duration) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}
	cfg.MaxConns = maxConns
	cfg.MinConns = minConns
	cfg.MaxConnLifetime = maxLifetime

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("connect database: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return &Store{pool: pool}, nil
}

// Close releases the pool.
func (s *Store) Close() {
	s.pool.Close()
}

// Pool exposes the underlying pool (tests / health checks).
func (s *Store) Pool() *pgxpool.Pool {
	return s.pool
}

// PersistDeviceSample upserts inventory and inserts a device metric sample.
func (s *Store) PersistDeviceSample(ctx context.Context, sample transform.DeviceSample) (Result, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := upsertSite(ctx, tx, sample.SiteUUID, sample.SiteName); err != nil {
		return 0, err
	}
	if err := upsertDevice(ctx, tx, sample.DeviceUUID, sample.SiteUUID, sample.DeviceHostname, sample.DeviceIPAddress, sample.CollectedAt); err != nil {
		return 0, err
	}

	metricTypeID, err := lookupMetricType(ctx, tx, sample.MetricName)
	if err != nil {
		return 0, err
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO metric_samples (device_id, metric_type_id, value, collected_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (device_id, metric_type_id, collected_at) DO NOTHING
	`, sample.DeviceUUID, metricTypeID, sample.Value, sample.CollectedAt)
	if err != nil {
		return 0, fmt.Errorf("insert metric_samples: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ResultDuplicate, nil
	}
	return ResultInserted, nil
}

// PersistInterfaceSample upserts inventory and inserts an interface sample.
func (s *Store) PersistInterfaceSample(ctx context.Context, sample transform.InterfaceSample) (Result, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx) //nolint:errcheck

	if err := upsertSite(ctx, tx, sample.SiteUUID, sample.SiteName); err != nil {
		return 0, err
	}
	if err := upsertDevice(ctx, tx, sample.DeviceUUID, sample.SiteUUID, sample.DeviceHostname, "", sample.CollectedAt); err != nil {
		return 0, err
	}
	ifaceID, err := upsertInterface(ctx, tx, sample.InterfaceUUID, sample.DeviceUUID, sample.IfIndex)
	if err != nil {
		return 0, err
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO interface_samples (
			interface_id, in_octets, out_octets, in_errors, out_errors, collected_at
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (interface_id, collected_at) DO NOTHING
	`, ifaceID, sample.InOctets, sample.OutOctets, sample.InErrors, sample.OutErrors, sample.CollectedAt)
	if err != nil {
		return 0, fmt.Errorf("insert interface_samples: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ResultDuplicate, nil
	}
	return ResultInserted, nil
}

func upsertSite(ctx context.Context, tx pgx.Tx, id uuid.UUID, name string) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO sites (id, name)
		VALUES ($1, $2)
		ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name
	`, id, name)
	if err != nil {
		return fmt.Errorf("upsert site: %w", err)
	}
	return nil
}

func upsertDevice(ctx context.Context, tx pgx.Tx, id, siteID uuid.UUID, hostname, ipAddress string, lastSeen time.Time) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO devices (
			id, site_id, hostname, ip_address, vendor, model, snmp_version, status, last_seen
		) VALUES (
		$1, $2, $3, COALESCE(NULLIF($4, ''), '0.0.0.0')::inet, 'unknown', 'unknown', '2c', 'online', $5
		)
		ON CONFLICT (id) DO UPDATE SET
			last_seen = EXCLUDED.last_seen,
		status = 'online',
		hostname = EXCLUDED.hostname,
		ip_address = CASE WHEN $4 <> '' THEN EXCLUDED.ip_address ELSE devices.ip_address END
	`, id, siteID, hostname, ipAddress, lastSeen)
	if err != nil {
		return fmt.Errorf("upsert device: %w", err)
	}
	return nil
}

// upsertInterface inserts or finds the interface row and returns its real id.
func upsertInterface(ctx context.Context, tx pgx.Tx, id, deviceID uuid.UUID, ifIndex int) (uuid.UUID, error) {
	var resolved uuid.UUID
	err := tx.QueryRow(ctx, `
		INSERT INTO interfaces (id, device_id, if_index, name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (device_id, if_index) DO UPDATE SET
			name = COALESCE(interfaces.name, EXCLUDED.name)
		RETURNING id
	`, id, deviceID, ifIndex, fmt.Sprintf("ifIndex-%d", ifIndex)).Scan(&resolved)
	if err != nil {
		return uuid.Nil, fmt.Errorf("upsert interface: %w", err)
	}
	return resolved, nil
}

func lookupMetricType(ctx context.Context, tx pgx.Tx, name string) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `SELECT id FROM metric_types WHERE name = $1`, name).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("%w: %s", ErrUnknownMetricType, name)
	}
	if err != nil {
		return uuid.Nil, fmt.Errorf("lookup metric_types: %w", err)
	}
	return id, nil
}
