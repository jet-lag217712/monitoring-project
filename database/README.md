# Equate database

PostgreSQL is the local appliance system of record for sites, devices,
interfaces, telemetry, health evidence, alerts, and collector status history.
Only the Ingestion Service writes monitoring data; the Backend API uses a
read-only account.

## Layout

| Path | Purpose |
|---|---|
| `schema/` | Human-readable table definitions |
| `seed/` | Reference seed SQL |
| `migrations/` | Versioned golang-migrate files |

Schema changes go through new migrations under `migrations/`; update the
matching `schema/` description in the same change.

## Local migration

The appliance runs migrations after PostgreSQL is healthy. For source testing:

```bash
export DATABASE_URL=postgres://ogsd:ogsd@127.0.0.1:5432/ogsd?sslmode=disable
./infrastructure/script/migrate.sh up
./infrastructure/script/migrate.sh version
```

Role bootstrap is performed by the local installation workflow:

```bash
export OGSD_INGESTION_PASSWORD=replace-locally
export OGSD_API_PASSWORD=replace-locally
./infrastructure/script/bootstrap-db-roles.sh
```

The appliance generates actual credentials per installation. Never commit
passwords or copy runtime database files into the repository.

## Data guarantees

- `event_id` deduplicates v2 telemetry during MQTT QoS 1 redelivery.
- Natural keys remain defense in depth for metric and interface samples.
- Observation timestamps control current-state ordering.
- Health history and collector heartbeat history remain append-oriented.
- Component readings preserve individual temperature and power sensors.
