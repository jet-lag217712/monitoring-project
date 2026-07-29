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

// DefaultHistoryWindow is the lookback used for embedded device history.
const DefaultHistoryWindow = 24 * time.Hour

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

// DeviceRow is a devices table row with optional latest metrics and health.
type DeviceRow struct {
	ID             uuid.UUID
	SiteID         uuid.UUID
	SiteName       string
	Hostname       string
	IPAddress      string
	Role               string
	InventoryDeviceID  string
	Vendor             string
	Model          string
	Serial         *string
	SysObjectID    *string
	SysName        *string
	SysDescr       *string
	ProfileName    *string
	Capabilities   []string
	Status         string
	LastSeen       *time.Time
	UptimeSeconds  *float64
	CPUPct         *float64
	MemoryPct      *float64
	TemperatureC   *float64
	HealthPresent  bool
	HealthState    string
	HealthReason   string
	FailureCount   int
	UpstreamIDs    []string
	UnavailableIDs []string
	RootCauseIDs   []string
}

// InterfaceRow is an interfaces table row with optional latest sample.
type InterfaceRow struct {
	ID          uuid.UUID
	DeviceID    uuid.UUID
	IfIndex     int
	Name        *string
	Description *string
	IfAlias     *string
	IfType      *string
	AdminStatus *string
	OperStatus  *string
	SpeedBps    *int64
	InOctets    *int64
	OutOctets   *int64
	InErrors    *int64
	OutErrors   *int64
	InDiscards  *int64
	OutDiscards *int64
}

// MetricSampleRow is one metric_samples point.
type MetricSampleRow struct {
	CollectedAt time.Time
	Value       float64
}

// InterfaceTrafficSampleRow is one interface_samples traffic point.
type InterfaceTrafficSampleRow struct {
	CollectedAt time.Time
	InOctets    float64
	OutOctets   float64
}

// ComponentRow is a temperature or power component with latest reading.
type ComponentRow struct {
	ComponentID string
	Name        string
	Index       string
	Value       *float64
	Unit        string
	Status      string
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

// ListDevicesBySite returns devices for a site UUID with health and latest scalars.
func (s *Store) ListDevicesBySite(ctx context.Context, siteID uuid.UUID) ([]DeviceRow, error) {
	rows, err := s.pool.Query(ctx, deviceSelect+` WHERE d.site_id = $1 ORDER BY d.hostname`, siteID)
	if err != nil {
		return nil, fmt.Errorf("list devices by site: %w", err)
	}
	defer rows.Close()

	return scanDevices(rows)
}

// ListAllDevices returns every device with site name, health, and latest scalars.
func (s *Store) ListAllDevices(ctx context.Context) ([]DeviceRow, error) {
	rows, err := s.pool.Query(ctx, deviceSelect+` ORDER BY s.name, d.hostname`)
	if err != nil {
		return nil, fmt.Errorf("list all devices: %w", err)
	}
	defer rows.Close()

	return scanDevices(rows)
}

// GetDevice resolves a device by UUID, site-scoped collector ID, or globally unique hostname.
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

	out, err := scanDevices(rows)
	if err != nil {
		return DeviceRow{}, err
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
			COALESCE(d.role, ''),
			COALESCE(d.inventory_device_id, ''),
			COALESCE(d.vendor, ''),
			COALESCE(d.model, ''),
			d.serial,
			d.sys_object_id,
			d.sys_name,
			d.sys_descr,
			d.profile_name,
			COALESCE(d.capabilities, '{}'),
			d.status,
			d.last_seen,
			u.value AS uptime_seconds,
			cpu.value AS cpu_pct,
			mem.value AS memory_pct,
			COALESCE(temp.value, h.temperature_c) AS temperature_c,
			(h.device_id IS NOT NULL) AS health_present,
			COALESCE(h.state, ''),
			COALESCE(h.reason, ''),
			COALESCE(h.failure_count, 0),
			COALESCE(h.upstream_device_ids, '{}'),
			COALESCE(h.unavailable_upstream_device_ids, '{}'),
			COALESCE(h.root_cause_device_ids, '{}')
		FROM devices d
		JOIN sites s ON s.id = d.site_id
		LEFT JOIN device_health_current h ON h.device_id = d.id
		LEFT JOIN LATERAL (
			SELECT ms.value
			FROM metric_samples ms
			JOIN metric_types mt ON mt.id = ms.metric_type_id
			WHERE ms.device_id = d.id AND mt.name = 'uptime_seconds'
			ORDER BY ms.collected_at DESC
			LIMIT 1
		) u ON true
		LEFT JOIN LATERAL (
			SELECT ms.value
			FROM metric_samples ms
			JOIN metric_types mt ON mt.id = ms.metric_type_id
			WHERE ms.device_id = d.id AND mt.name = 'cpu_utilization_pct'
			ORDER BY ms.collected_at DESC
			LIMIT 1
		) cpu ON true
		LEFT JOIN LATERAL (
			SELECT ms.value
			FROM metric_samples ms
			JOIN metric_types mt ON mt.id = ms.metric_type_id
			WHERE ms.device_id = d.id AND mt.name = 'memory_utilization_pct'
			ORDER BY ms.collected_at DESC
			LIMIT 1
		) mem ON true
		LEFT JOIN LATERAL (
			SELECT ms.value
			FROM metric_samples ms
			JOIN metric_types mt ON mt.id = ms.metric_type_id
			WHERE ms.device_id = d.id AND mt.name = 'primary_temperature_c'
			ORDER BY ms.collected_at DESC
			LIMIT 1
		) temp ON true
`

// ListInterfaces returns interfaces for a device UUID with latest counters.
func (s *Store) ListInterfaces(ctx context.Context, deviceID uuid.UUID) ([]InterfaceRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			i.id,
			i.device_id,
			i.if_index,
			i.name,
			i.description,
			i.if_alias,
			i.if_type,
			i.admin_status,
			i.oper_status,
			i.speed_bps,
			samp.in_octets,
			samp.out_octets,
			samp.in_errors,
			samp.out_errors,
			samp.in_discards,
			samp.out_discards
		FROM interfaces i
		LEFT JOIN LATERAL (
			SELECT in_octets, out_octets, in_errors, out_errors, in_discards, out_discards
			FROM interface_samples
			WHERE interface_id = i.id
			ORDER BY collected_at DESC
			LIMIT 1
		) samp ON true
		WHERE i.device_id = $1
		ORDER BY i.if_index
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
			&r.IfAlias, &r.IfType, &r.AdminStatus, &r.OperStatus, &r.SpeedBps,
			&r.InOctets, &r.OutOctets, &r.InErrors, &r.OutErrors, &r.InDiscards, &r.OutDiscards,
		); err != nil {
			return nil, fmt.Errorf("scan interface: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListInterfaceTrafficHistory returns recent in/out octet samples for an interface.
func (s *Store) ListInterfaceTrafficHistory(ctx context.Context, interfaceID uuid.UUID, start time.Time) ([]InterfaceTrafficSampleRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			collected_at,
			COALESCE(in_octets, 0)::double precision,
			COALESCE(out_octets, 0)::double precision
		FROM interface_samples
		WHERE interface_id = $1 AND collected_at >= $2
		ORDER BY collected_at ASC
	`, interfaceID, start)
	if err != nil {
		return nil, fmt.Errorf("list interface traffic: %w", err)
	}
	defer rows.Close()

	var out []InterfaceTrafficSampleRow
	for rows.Next() {
		var r InterfaceTrafficSampleRow
		if err := rows.Scan(&r.CollectedAt, &r.InOctets, &r.OutOctets); err != nil {
			return nil, fmt.Errorf("scan interface traffic: %w", err)
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

// ListTemperatureComponents returns inventory + latest reading per component.
func (s *Store) ListTemperatureComponents(ctx context.Context, deviceID uuid.UUID) ([]ComponentRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			c.component_id,
			c.name,
			c.component_index,
			r.value,
			COALESCE(r.unit, ''),
			COALESCE(r.status, '')
		FROM device_temperature_components c
		LEFT JOIN LATERAL (
			SELECT value, unit, status
			FROM device_temperature_readings
			WHERE device_id = c.device_id AND component_id = c.component_id
			ORDER BY observed_at DESC
			LIMIT 1
		) r ON true
		WHERE c.device_id = $1
		ORDER BY c.component_index, c.component_id
	`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("list temperature components: %w", err)
	}
	defer rows.Close()
	return scanComponents(rows)
}

// ListPowerComponents returns inventory + latest reading per component.
func (s *Store) ListPowerComponents(ctx context.Context, deviceID uuid.UUID) ([]ComponentRow, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			c.component_id,
			c.name,
			c.component_index,
			r.value,
			COALESCE(r.unit, ''),
			COALESCE(r.status, '')
		FROM device_power_components c
		LEFT JOIN LATERAL (
			SELECT value, unit, status
			FROM device_power_readings
			WHERE device_id = c.device_id AND component_id = c.component_id
			ORDER BY observed_at DESC
			LIMIT 1
		) r ON true
		WHERE c.device_id = $1
		ORDER BY c.component_index, c.component_id
	`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("list power components: %w", err)
	}
	defer rows.Close()
	return scanComponents(rows)
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

func scanComponents(rows pgx.Rows) ([]ComponentRow, error) {
	var out []ComponentRow
	for rows.Next() {
		var r ComponentRow
		if err := rows.Scan(&r.ComponentID, &r.Name, &r.Index, &r.Value, &r.Unit, &r.Status); err != nil {
			return nil, fmt.Errorf("scan component: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanDevices(rows pgx.Rows) ([]DeviceRow, error) {
	var out []DeviceRow
	for rows.Next() {
		r, err := scanDeviceFields(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func scanDevice(row pgx.Row) (DeviceRow, error) {
	r, err := scanDeviceFields(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return DeviceRow{}, ErrNotFound
	}
	if err != nil {
		return DeviceRow{}, err
	}
	return r, nil
}

type scannable interface {
	Scan(dest ...any) error
}

func scanDeviceFields(row scannable) (DeviceRow, error) {
	var r DeviceRow
	err := row.Scan(
		&r.ID, &r.SiteID, &r.SiteName, &r.Hostname, &r.IPAddress, &r.Role, &r.InventoryDeviceID,
		&r.Vendor, &r.Model, &r.Serial, &r.SysObjectID, &r.SysName, &r.SysDescr,
		&r.ProfileName, &r.Capabilities, &r.Status, &r.LastSeen, &r.UptimeSeconds,
		&r.CPUPct, &r.MemoryPct, &r.TemperatureC,
		&r.HealthPresent, &r.HealthState, &r.HealthReason, &r.FailureCount,
		&r.UpstreamIDs, &r.UnavailableIDs, &r.RootCauseIDs,
	)
	if err != nil {
		return DeviceRow{}, fmt.Errorf("scan device: %w", err)
	}
	return r, nil
}
