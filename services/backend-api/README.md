# Backend API

## Plane Ownership

UI/UX Cloud Plane.

## Responsibilities

- Expose REST contracts for frontend clients.
- Read monitoring state and history from PostgreSQL.
- Translate database records into API responses.
- Enforce application access controls (OIDC in Phase 6).

## Non-Responsibilities

- Polling SNMP devices.
- Processing telemetry transport messages.
- Writing monitoring samples.
- Rendering frontend views.
- Configuring monitored devices.
- Providing device console or management access.
- Alert generation.

## Deployment Boundary

The API runs in the UI/UX Cloud Plane and is the only frontend-facing service for monitoring data.

Approved flow:

```text
PostgreSQL -> Backend API -> UI/UX Cloud Plane
```

## Run locally

Requires local Postgres from `deployments/local/test-env/` with roles bootstrapped
(`ogsd_api` SELECT-only).

```bash
export DATABASE_URL='postgres://ogsd_api:api@127.0.0.1:5432/ogsd?sslmode=disable'
cd services/backend-api
go run ./cmd/api -config configs/api.example.yaml
```

- REST API: `http://127.0.0.1:8000`
- Admin (`/healthz`, `/metrics`): `http://127.0.0.1:9092`

### MVP endpoints

| Method | Path | Notes |
|--------|------|-------|
| GET | `/api/sites` | Overview object keyed by collector site ID (`sites.name`) |
| GET | `/api/sites/{siteId}` | Site detail + `latest.devices` |
| GET | `/api/sites/{siteId}/devices` | Device list for site |
| GET | `/api/devices/{deviceId}` | Resolve by UUID, or collector ID with optional `?siteId=` |
| GET | `/api/devices/{deviceId}/interfaces` | IF-MIB inventory; optional `?siteId=` |
| GET | `/api/devices/{deviceId}/metrics` | Query: `start`, `end`, `metric` (default `uptime_seconds`); optional `?siteId=` |
| GET | `/api/alerts` | Active alerts (`cleared_at IS NULL`) |
| GET | `/api/test-config` | `{ "mode": "live", "polling_enabled": true }` |

REST prefix is `/api/...` (not `/api/v1/...`). Frontend and the MVP roadmap are canonical.

### Identifier conventions

- `{siteId}` = collector string ID stored in `sites.name`
- Site detail device map keys prefer real `ip_address`; fall back to `hostname` when IP is `0.0.0.0`
- `{deviceId}` = collector string ID (`devices.hostname`) with optional `?siteId=` (sites.name), or device UUID

### Honest defaults (MVP)

Fields without telemetry backing (`cpu_pct`, `memory_pct`, `latency_ms`, `role`, `type`, `idf_count`) return `0`, `null`, or `""`.
