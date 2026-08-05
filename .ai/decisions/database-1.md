## database - 1

### Primary Service

PostgreSQL / Ingestion Service

### Secondary Services

- On-prem appliance deployment
- Backend API (read-only consumers of retained history)

### Choice Made

Retain high-volume time-series and history for 30 days by default. The
ingestion service runs a background retention job that batched-deletes rows
older than the configured cutoff from:

- `metric_samples`, `interface_samples`
- `device_temperature_readings`, `device_power_readings`
- `device_health_history`, `collector_heartbeat_history`
- `ingested_events`, `alerts`

Inventory and current-state projections are never pruned (`sites`, `devices`,
`interfaces`, component inventory, `collectors`, `device_health_current`,
`collector_status_current`, `metric_types`). Retention is configurable via
ingestion YAML (`retention.days`, `retention.interval`, `retention.batch_size`,
`retention.enabled`). Disk reclaim relies on PostgreSQL autovacuum.

### Alternatives Considered

- Continue append-only until measured capacity proves a problem — rejected;
  250+ devices make unbounded growth an appliance risk before that measurement
  can land in production.
- `pg_cron` or a separate cleanup container — rejected; appliance Postgres is
  stock `postgres:16-alpine` without cron, and ingestion already owns
  monitoring-data writes.
- Partitioning / archival instead of delete — deferred; can be added later
  without changing the event contract.

### Trade-offs

Operators lose history older than the retention window. Batched deletes avoid
long locks but temporarily increase write I/O and dead tuples until
autovacuum catches up. `ogsd_ingestion` gains DELETE on history tables only.

### Status

Accepted
