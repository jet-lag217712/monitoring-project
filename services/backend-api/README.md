# Backend API

## Plane Ownership

UI/UX Cloud Plane.

## Responsibilities

- Expose REST contracts for frontend clients.
- Read monitoring state and history from PostgreSQL.
- Translate database records into API responses.
- Enforce application access controls (Google OIDC).


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

Requires local Postgres from `./deployments/development/up.sh` (Mac cloud stack) with roles bootstrapped
(`ogsd_api` SELECT-only).

```bash
export DATABASE_URL='postgres://ogsd_api:api@127.0.0.1:5432/ogsd?sslmode=disable'
export GOOGLE_CLIENT_ID='your-google-oauth-web-client-id.apps.googleusercontent.com'
cd services/backend-api
go run ./cmd/api -config configs/api.example.yaml
```

With `auth.enabled: true` (default in `configs/api.example.yaml`), every `/api/*` request requires:

```http
Authorization: Bearer <Google ID token>
```

Set `auth.enabled: false` only for local unauthenticated debugging.

- REST API: `http://127.0.0.1:8000`
- Admin (`/healthz`, `/metrics`): `http://127.0.0.1:9092` (unauthenticated)


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

## Honest defaults (MVP + v2)

- Absent telemetry (`cpu_pct`, `memory_pct`, `temperature_c`, `latency_ms`) is JSON `null`, never fabricated zeros.
- Device `status` uses numeric compatibility: `0` unknown, `1` healthy, `2` warning, `3` critical.
- When `device_health_current` exists, status/reason/dependency fields come from that projection. Without a health row, MVP online→`1` / offline→`3` fallback remains.
- Site summaries expose `healthy_count`, `warning_count`, `critical_count`, `unknown_count`, and `dependency_impacted_count`. Unknown dependents are never counted as Critical.
- Device detail embeds `history.{cpu,memory,temperature,uptime}` (24h window) plus temperature/power components and SNMP identity when persisted.
- Fields without inventory backing (`role`, `type`, `idf_count`) return `0` or `""`.
