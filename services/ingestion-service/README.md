# Ingestion Service

## Plane Ownership

UI/UX Cloud Plane.

## Responsibilities

- Consume telemetry from Secure Outbound Telemetry Transport (MQTT/TLS).
- Validate telemetry payloads.
- Normalize collector string IDs to deterministic UUID v5 keys.
- Write monitoring state and history to PostgreSQL (idempotent).
- ACK MQTT messages only after commit (or safe reject/dedup).
- Reject malformed or unauthorized messages.

## Non-Responsibilities

- Polling SNMP devices.
- Hosting telemetry transport.
- Serving frontend requests.
- Rendering dashboard views.
- Configuring monitored devices.
- Providing device console or management access.

## Build and run

```bash
cd services/ingestion-service
go test ./...
export MQTT_PASSWORD=ingestion
export DATABASE_URL=postgres://ogsd_ingestion:ingestion@127.0.0.1:5432/ogsd?sslmode=disable
go run ./cmd/ingestion -config configs/ingestion.example.yaml
```

Admin endpoints (default `:9091`):

- `GET /metrics` — Prometheus scrape
- `GET /healthz` — liveness

## Local stack (`deployments/local/test-env`)

```bash
# From repo root
./deployments/local/test-env/up.sh
```

Full manual E2E runbook (C0–C9): [`deployments/local/test-env/README.md`](../../deployments/local/test-env/README.md).

## Testing

### Layer A — Unit (no Docker)

```bash
cd services/ingestion-service
go test ./... -count=1
```

### Layer B — Integration

Requires Mosquitto + Postgres from `deployments/local/test-env`:

```bash
./deployments/local/test-env/up.sh
export MQTT_PASSWORD=ingestion
export MQTT_BROKER=tls://127.0.0.1:8883
export MQTT_CA_FILE="$PWD/infrastructure/docker/mqtt-broker/certs/ca.crt"
export DATABASE_URL=postgres://ogsd_ingestion:ingestion@127.0.0.1:5432/ogsd?sslmode=disable
cd services/ingestion-service
go test -tags=integration ./tests/ -count=1 -v
```

## Metrics

| Metric | Meaning |
|--------|---------|
| `ingestion_messages_received_total` | MQTT messages received |
| `ingestion_messages_accepted_total` | Validated + persisted |
| `ingestion_messages_rejected_total` | Validation / unknown metric |
| `ingestion_messages_deduplicated_total` | Duplicates skipped |
| `ingestion_db_write_failure_total` | Transaction failures |
| `ingestion_processing_duration_seconds` | Receive → ACK decision |
| `ingestion_mqtt_connected` | 1 when connected |

## Container

```bash
cd services/ingestion-service
docker build -t equate/ingestion-service:dev .
```

## Deployment Boundary

Approved flow:

```text
Secure Outbound Telemetry Transport -> Cloud Ingestion -> PostgreSQL
```
