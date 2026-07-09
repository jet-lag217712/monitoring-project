# Database

PostgreSQL is the system of record for OGSD monitoring data.

## Layout

| Path | Purpose |
|------|---------|
| `schema/` | Human-readable table definitions (source of truth for review) |
| `seed/` | Reference seed SQL (also applied via migrations) |
| `migrations/` | Versioned golang-migrate files applied to environments |

After the initial Phase 4 cutover, **schema changes go through new migrations** under `migrations/`. Update `schema/` in the same change for readability.

## Migrations

Requires [golang-migrate](https://github.com/golang-migrate/migrate) (`brew install golang-migrate`) or Docker.

```bash
export DATABASE_URL=postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable
./infrastructure/script/migrate.sh up
./infrastructure/script/migrate.sh version
```

Local `deployments/local/test-env/up.sh` runs migrate + role password bootstrap automatically.

### Migration versions

| Version | Contents |
|---------|----------|
| 1 | Tables + indexes |
| 2 | Dedup unique constraints |
| 3 | `uptime_seconds` metric type seed |
| 4 | Roles `ogsd_admin`, `ogsd_ingestion`, `ogsd_api` + grants |

### Idempotency keys

| Table | Unique constraint |
|-------|-------------------|
| `metric_samples` | `(device_id, metric_type_id, collected_at)` |
| `interface_samples` | `(interface_id, collected_at)` |

## Roles

| Role | Use |
|------|-----|
| `ogsd` | Local Docker superuser only (migrations bootstrap) |
| `ogsd_admin` | Migrations / DDL (Azure Flexible Server admin) |
| `ogsd_ingestion` | Ingestion writes (INSERT/UPDATE inventory; INSERT+SELECT samples for ON CONFLICT) |
| `ogsd_api` | Backend API reads |

Set passwords after migrate:

```bash
export DATABASE_URL=postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable
export OGSD_INGESTION_PASSWORD=ingestion
export OGSD_API_PASSWORD=api
./infrastructure/script/bootstrap-db-roles.sh
```

## Azure

Terraform module: `infrastructure/terraform/modules/postgresql/`  
Environments: `infrastructure/terraform/environments/{dev,prod}/`

After provisioning, run migrations with the admin DSN (`sslmode=require`), then bootstrap `ogsd_ingestion` / `ogsd_api` passwords.
