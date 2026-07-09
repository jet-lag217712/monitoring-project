# PostgreSQL Architecture

## Purpose

PostgreSQL stores authoritative monitoring state and historical telemetry in the UI/UX Cloud Plane.

## Responsibilities

PostgreSQL stores:

- Sites.
- Monitored devices.
- Interfaces.
- Metrics.
- Alerts.
- User data when application authentication is implemented.

## Design Goals

The database should support:

- Time-series queries.
- Device history.
- Dashboard queries.
- Alert generation inputs.
- Backend API contracts.

## Operational Requirements

Production deployment should include:

- Backups.
- Indexing.
- Migration management.
- Storage monitoring.
- Access controls separating ingestion writes from API reads.

## Migrations and Roles

Schema changes are applied with [golang-migrate](https://github.com/golang-migrate/migrate) from [`database/migrations/`](../../database/migrations/). See [`database/README.md`](../../database/README.md).

| Role | Purpose |
|------|---------|
| `ogsd_admin` | DDL / migrations (Azure Flexible Server administrator) |
| `ogsd_ingestion` | INSERT/UPDATE for auto-discovery; INSERT+SELECT on samples (ON CONFLICT); SELECT on reference tables |
| `ogsd_api` | SELECT only |

Idempotency is enforced at the database:

- `metric_samples`: `UNIQUE (device_id, metric_type_id, collected_at)`
- `interface_samples`: `UNIQUE (interface_id, collected_at)`

Azure provisioning: [`infrastructure/terraform/modules/postgresql/`](../../infrastructure/terraform/modules/postgresql/).

## Deployment Boundary

PostgreSQL belongs to the UI/UX Cloud Plane and is the system of record.

Collectors and frontend clients must never connect directly to PostgreSQL.
