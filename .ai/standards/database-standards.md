# Database Standards

These standards are based on PostgreSQL documentation and the database patterns already present in this repository.

## Current Project Pattern

- Use PostgreSQL as the system of record.
- Keep the schema normalized for inventory data and append-only for time-series data.
- Match the existing naming style: lowercase `snake_case` table and column names.
- `sites`, `devices`, `interfaces`, `metric_types`, collector status, and health state are inventory or lifecycle tables.
- Device/interface samples, component readings, health history, and heartbeat history are append-oriented high-volume tables.
- Use UUID primary keys for entity tables unless there is a clear reason to use a different key type.
- Use `BIGSERIAL` or another sequence-backed key for large append-only sample tables when it is the better fit for insert-heavy workloads.

## Schema Design

- Put business rules in the database when they are stable and easy to express with constraints.
- Use `NOT NULL` for required fields.
- Use `UNIQUE` constraints for real identity rules.
- Use foreign keys for relationships between entities.
- Use `CHECK` constraints for row-level validation, not for cross-row or cross-table logic.
- Prefer `TIMESTAMPTZ` for timestamps that represent real events.
- Use domain-appropriate PostgreSQL types:
  - `UUID` for identifiers.
  - `INET` for IP addresses.
  - `BIGINT` for counters and byte totals.
  - `TEXT` for free-form descriptions and messages.
- Keep column types as narrow as practical.

## Table Conventions

- Inventory tables should have a clear primary key and required foreign keys.
- Append-only sample tables should be optimized for insert rate and time-range queries.
- Alert and lifecycle tables should preserve historical state instead of overwriting it blindly.
- For v2 health and collector state, retain the original observation timestamp and update a current-state projection only when ordering rules permit it; arrival time is not a substitute for observation time.
- Store component readings individually (name/index/value/unit/status); do not collapse a multi-sensor device into a fabricated scalar.
- Use `event_id` uniqueness for v2 telemetry alongside natural sample keys so QoS 1 redelivery is harmless.
- If a table tracks current status, make the meaning of default values explicit.
- Use naming that reflects the entity, not the implementation.

## Migrations

- Treat schema changes as migrations, not as manual edits to live databases.
- Keep migrations small and reversible when possible.
- Add constraints and indexes deliberately; do not assume they are free.
- Prefer additive changes first, then backfill, then tighten constraints.
- Validate schema changes against realistic data before merging.

## Indexing

- Add indexes to support known read paths, especially foreign keys and time-based lookups.
- Use composite indexes when queries filter and sort on the same columns.
- Match index order to the query pattern.
- Do not create indexes speculatively.
- Verify index value with `EXPLAIN` or `EXPLAIN ANALYZE` on realistic data.
- Revisit indexes when access patterns change.

## Querying

- Select only the columns you need.
- Filter as early as possible.
- Use parameterized queries for all application data access.
- Avoid N+1 query patterns.
- Use transactions for multi-step writes that must succeed or fail together.
- Keep long-running transactions out of hot paths when possible.

## Operations

- Run `ANALYZE` or equivalent maintenance when planner statistics are stale.
- Use real data when evaluating performance.
- Treat time-series growth as a first-class scaling concern.
- Enforce the 30-day default retention on high-volume sample/history tables via the ingestion retention job; leave inventory and current-state tables untouched.
- Plan for partitioning or archival when sample tables become large enough that delete-based retention is no longer enough.

## Existing Schema Notes

- The current schema already uses `sites(id UUID PRIMARY KEY, ...)`.
- The current schema already uses `devices(site_id UUID NOT NULL REFERENCES sites(id), ...)`.
- The current schema already uses `interfaces(device_id UUID NOT NULL REFERENCES devices(id), UNIQUE(device_id, if_index))`.
- The current schema already uses `metric_samples(device_id UUID NOT NULL REFERENCES devices(id), metric_type_id UUID NOT NULL REFERENCES metric_types(id), collected_at TIMESTAMPTZ NOT NULL)`.
- The current schema already uses `interface_samples(interface_id UUID NOT NULL REFERENCES interfaces(id), collected_at TIMESTAMPTZ NOT NULL)`.
- The current schema already uses dedicated indexes for device, interface, and time-based access.
- V2 migrations must add documented query indexes for health/dependency evidence, component history, collector heartbeat history, and current-state summaries.
- Preserve those patterns unless the workload proves they should change.

## References

- PostgreSQL documentation: https://www.postgresql.org/docs/current/index.html
- PostgreSQL constraints: https://www.postgresql.org/docs/current/ddl-constraints.html
- PostgreSQL indexes: https://www.postgresql.org/docs/current/indexes.html
- PostgreSQL query planning and index usage: https://www.postgresql.org/docs/current/indexes-examine.html
