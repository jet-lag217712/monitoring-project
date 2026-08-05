package store

import (
	"context"
	"fmt"
	"time"
)

// allowedRetentionTargets maps table name → timestamp column for retention deletes.
// Identifiers are never taken from caller input outside this allowlist.
var allowedRetentionTargets = map[string]string{
	"metric_samples":              "collected_at",
	"interface_samples":           "collected_at",
	"device_temperature_readings": "observed_at",
	"device_power_readings":       "observed_at",
	"device_health_history":       "observed_at",
	"collector_heartbeat_history": "observed_at",
	"ingested_events":             "observed_at",
	"alerts":                      "created_at",
}

// DeleteOlderThan deletes up to batchSize rows from table where column is older than cutoff.
// table and column must match an allowlisted retention target.
func (s *Store) DeleteOlderThan(ctx context.Context, table, column string, cutoff time.Time, batchSize int) (int64, error) {
	expected, ok := allowedRetentionTargets[table]
	if !ok || expected != column {
		return 0, fmt.Errorf("retention target not allowlisted: %s.%s", table, column)
	}
	if batchSize <= 0 {
		return 0, fmt.Errorf("batch size must be positive")
	}

	// Identifiers are allowlisted above; values are parameterized.
	query := fmt.Sprintf(`
		DELETE FROM %s
		WHERE ctid IN (
			SELECT ctid FROM %s
			WHERE %s < $1
			LIMIT $2
		)`, table, table, column)

	tag, err := s.pool.Exec(ctx, query, cutoff, batchSize)
	if err != nil {
		return 0, fmt.Errorf("delete older than from %s: %w", table, err)
	}
	return tag.RowsAffected(), nil
}
