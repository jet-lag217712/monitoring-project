package store

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/equate/ogsd/services/backend-api/internal/derive"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// ErrAmbiguous is returned when a lookup matches more than one row.
var ErrAmbiguous = errors.New("ambiguous")

// Store provides read-only access to monitoring data.
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

// SiteRow is a sites table row.
type SiteRow struct {
	ID       uuid.UUID
	Name     string
	Location *string
}

// DeviceRow is a devices table row with optional latest uptime.
type DeviceRow struct {
	ID            uuid.UUID
	SiteID        uuid.UUID
	SiteName      string
	Hostname      string
	IPAddress     string
	Vendor        string
	Model         string
	Status        string
	LastSeen      *time.Time
	UptimeSeconds *float64
}

// InterfaceRow is an interfaces table row.
type InterfaceRow struct {
	ID          uuid.UUID
	DeviceID    uuid.UUID
	IfIndex     int
	Name        *string
	Description *string
	AdminStatus *string
	OperStatus  *string
	SpeedBps    *int64
}

// MetricSampleRow is one metric_samples point.
type MetricSampleRow struct {
	CollectedAt time.Time
	Value       float64
}

// AlertRow is an alerts table row.
type AlertRow struct {
	ID           uuid.UUID
	DeviceID     *uuid.UUID
	InterfaceID  *uuid.UUID
	SiteName     *string
	Severity     string
	AlertType    string
	Message      string
	Acknowledged bool
	CreatedAt    time.Time
}

// ListSites returns all sites.
func (s *Store) ListSites(ctx context.Context) ([]SiteRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, location
		FROM sites
		ORDER BY name
	`)
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}
	defer rows.Close()

	var out []SiteRow
	for rows.Next() {
		var r SiteRow
		if err := rows.Scan(&r.ID, &r.Name, &r.Location); err != nil {
			return nil, fmt.Errorf("scan site: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetSiteByName returns a site by collector string ID (sites.name).
func (s *Store) GetSiteByName(ctx context.Context, name string) (SiteRow, error) {
	var r SiteRow
	err := s.pool.QueryRow(ctx, `
		SELECT id, name, location
		FROM sites
		WHERE name = $1
	`, name).Scan(&r.ID, &r.Name, &r.Location)
	if errors.Is(err, pgx.ErrNoRows) {
		return SiteRow{}, ErrNotFound
	}
	if err != nil {
		return SiteRow{}, fmt.Errorf("get site: %w", err)
	}
	return r, nil
}

// ListDevicesBySite returns devices for a site UUID, with latest uptime_seconds.
func (s *Store) ListDevicesBySite(ctx context.Context, siteID uuid.UUID) ([]DeviceRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			d.id,
			d.site_id,
			s.name,
			d.hostname,
			d.ip_address::text,
			d.vendor,
			d.model,
			d.status,
			d.last_seen,
			u.value AS uptime_seconds
		FROM devices d
		JOIN sites s ON s.id = d.site_id
		LEFT JOIN LATERAL (
			SELECT ms.value
			FROM metric_samples ms
			JOIN metric_types mt ON mt.id = ms.metric_type_id
			WHERE ms.device_id = d.id AND mt.name = 'uptime_seconds'
			ORDER BY ms.collected_at DESC
			LIMIT 1
		) u ON true
		WHERE d.site_id = $1
		ORDER BY d.hostname
	`, siteID)
	if err != nil {
		return nil, fmt.Errorf("list devices by site: %w", err)
	}
	defer rows.Close()

	return scanDevices(rows)
}

// ListAllDevices returns every device with site name and latest uptime.
func (s *Store) ListAllDevices(ctx context.Context) ([]DeviceRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			d.id,
			d.site_id,
			s.name,
			d.hostname,
			d.ip_address::text,
			d.vendor,
			d.model,
			d.status,
			d.last_seen,
			u.value AS uptime_seconds
		FROM devices d
		JOIN sites s ON s.id = d.site_id
		LEFT JOIN LATERAL (
			SELECT ms.value
			FROM metric_samples ms
			JOIN metric_types mt ON mt.id = ms.metric_type_id
			WHERE ms.device_id = d.id AND mt.name = 'uptime_seconds'
			ORDER BY ms.collected_at DESC
			LIMIT 1
		) u ON true
		ORDER BY s.name, d.hostname
	`)
	if err != nil {
		return nil, fmt.Errorf("list all devices: %w", err)
	}
	defer rows.Close()

	return scanDevices(rows)
}

// GetDevice resolves a device by UUID, site-scoped collector ID, or globally unique hostname.
// siteName is the collector site ID (sites.name); when set, deviceID is resolved via the same
// deterministic UUID derivation used by ingestion.
func (s *Store) GetDevice(ctx context.Context, deviceID, siteName string) (DeviceRow, error) {
	if id, err := uuid.Parse(deviceID); err == nil {
		return s.getDeviceByID(ctx, id)
	}
	if siteName != "" {
		id := derive.DeviceUUID(siteName, deviceID)
		return s.getDeviceByID(ctx, id)
	}
	return s.getDeviceByHostnameUnique(ctx, deviceID)
}

func (s *Store) getDeviceByHostnameUnique(ctx context.Context, hostname string) (DeviceRow, error) {
	rows, err := s.pool.Query(ctx, deviceSelect+` WHERE d.hostname = $1 LIMIT 2`, hostname)
	if err != nil {
		return DeviceRow{}, fmt.Errorf("get device by hostname: %w", err)
	}
	defer rows.Close()

	var out []DeviceRow
	for rows.Next() {
		var r DeviceRow
		if err := rows.Scan(
			&r.ID, &r.SiteID, &r.SiteName, &r.Hostname, &r.IPAddress,
			&r.Vendor, &r.Model, &r.Status, &r.LastSeen, &r.UptimeSeconds,
		); err != nil {
			return DeviceRow{}, fmt.Errorf("scan device: %w", err)
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return DeviceRow{}, fmt.Errorf("get device by hostname: %w", err)
	}
	switch len(out) {
	case 0:
		return DeviceRow{}, ErrNotFound
	case 1:
		return out[0], nil
	default:
		return DeviceRow{}, ErrAmbiguous
	}
}

func (s *Store) getDeviceByID(ctx context.Context, id uuid.UUID) (DeviceRow, error) {
	row := s.pool.QueryRow(ctx, deviceSelect+` WHERE d.id = $1`, id)
	return scanDevice(row)
}

const deviceSelect = `
		SELECT
			d.id,
			d.site_id,
			s.name,
			d.hostname,
			d.ip_address::text,
			d.vendor,
			d.model,
			d.status,
			d.last_seen,
			u.value AS uptime_seconds
		FROM devices d
		JOIN sites s ON s.id = d.site_id
		LEFT JOIN LATERAL (
			SELECT ms.value
			FROM metric_samples ms
			JOIN metric_types mt ON mt.id = ms.metric_type_id
			WHERE ms.device_id = d.id AND mt.name = 'uptime_seconds'
			ORDER BY ms.collected_at DESC
			LIMIT 1
		) u ON true
`

// ListInterfaces returns interfaces for a device UUID.
func (s *Store) ListInterfaces(ctx context.Context, deviceID uuid.UUID) ([]InterfaceRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, device_id, if_index, name, description, admin_status, oper_status, speed_bps
		FROM interfaces
		WHERE device_id = $1
		ORDER BY if_index
	`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("list interfaces: %w", err)
	}
	defer rows.Close()

	var out []InterfaceRow
	for rows.Next() {
		var r InterfaceRow
		if err := rows.Scan(
			&r.ID, &r.DeviceID, &r.IfIndex, &r.Name, &r.Description,
			&r.AdminStatus, &r.OperStatus, &r.SpeedBps,
		); err != nil {
			return nil, fmt.Errorf("scan interface: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListMetrics returns time-series points for a device and metric name.
func (s *Store) ListMetrics(ctx context.Context, deviceID uuid.UUID, metric string, start, end *time.Time) ([]MetricSampleRow, error) {
	query := `
		SELECT ms.collected_at, ms.value
		FROM metric_samples ms
		JOIN metric_types mt ON mt.id = ms.metric_type_id
		WHERE ms.device_id = $1 AND mt.name = $2
	`
	args := []any{deviceID, metric}
	argN := 3
	if start != nil {
		query += fmt.Sprintf(" AND ms.collected_at >= $%d", argN)
		args = append(args, *start)
		argN++
	}
	if end != nil {
		query += fmt.Sprintf(" AND ms.collected_at <= $%d", argN)
		args = append(args, *end)
	}
	query += " ORDER BY ms.collected_at ASC"

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list metrics: %w", err)
	}
	defer rows.Close()

	var out []MetricSampleRow
	for rows.Next() {
		var r MetricSampleRow
		if err := rows.Scan(&r.CollectedAt, &r.Value); err != nil {
			return nil, fmt.Errorf("scan metric: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListActiveAlerts returns alerts with cleared_at IS NULL.
func (s *Store) ListActiveAlerts(ctx context.Context) ([]AlertRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			a.id,
			a.device_id,
			a.interface_id,
			s.name,
			a.severity,
			a.alert_type,
			a.message,
			a.acknowledged,
			a.created_at
		FROM alerts a
		LEFT JOIN devices d ON d.id = a.device_id
		LEFT JOIN sites s ON s.id = d.site_id
		WHERE a.cleared_at IS NULL
		ORDER BY a.created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list alerts: %w", err)
	}
	defer rows.Close()

	var out []AlertRow
	for rows.Next() {
		var r AlertRow
		if err := rows.Scan(
			&r.ID, &r.DeviceID, &r.InterfaceID, &r.SiteName,
			&r.Severity, &r.AlertType, &r.Message, &r.Acknowledged, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan alert: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CountActiveAlertsBySite returns active alert counts keyed by site UUID.
func (s *Store) CountActiveAlertsBySite(ctx context.Context) (map[uuid.UUID]int, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT d.site_id, COUNT(*)::int
		FROM alerts a
		JOIN devices d ON d.id = a.device_id
		WHERE a.cleared_at IS NULL
		GROUP BY d.site_id
	`)
	if err != nil {
		return nil, fmt.Errorf("count alerts by site: %w", err)
	}
	defer rows.Close()

	out := make(map[uuid.UUID]int)
	for rows.Next() {
		var siteID uuid.UUID
		var n int
		if err := rows.Scan(&siteID, &n); err != nil {
			return nil, fmt.Errorf("scan alert count: %w", err)
		}
		out[siteID] = n
	}
	return out, rows.Err()
}

func scanDevices(rows pgx.Rows) ([]DeviceRow, error) {
	var out []DeviceRow
	for rows.Next() {
		var r DeviceRow
		if err := rows.Scan(
			&r.ID, &r.SiteID, &r.SiteName, &r.Hostname, &r.IPAddress,
			&r.Vendor, &r.Model, &r.Status, &r.LastSeen, &r.UptimeSeconds,
		); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanDevice(row pgx.Row) (DeviceRow, error) {
	var r DeviceRow
	err := row.Scan(
		&r.ID, &r.SiteID, &r.SiteName, &r.Hostname, &r.IPAddress,
		&r.Vendor, &r.Model, &r.Status, &r.LastSeen, &r.UptimeSeconds,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return DeviceRow{}, ErrNotFound
	}
	if err != nil {
		return DeviceRow{}, fmt.Errorf("scan device: %w", err)
	}
	return r, nil
}
