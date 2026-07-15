# SNMP Collector

## Plane Ownership

Customer OOB Monitoring Plane.

## Responsibilities

- Poll monitored devices using SNMP.
- Normalize device and interface telemetry.
- Buffer telemetry locally during connectivity interruptions.
- Publish telemetry through outbound-only secure transport.

## Non-Responsibilities

- Hosting PostgreSQL.
- Hosting the Backend API.
- Hosting cloud ingestion.
- Rendering UI workflows.
- Configuring monitored devices.
- Providing device console or management access.

## Deployment Boundary

The collector runs in the customer environment and initiates outbound-only telemetry connections to the UI/UX Cloud Plane.

Approved flow:

```text
SNMP Devices -> SNMP Collector -> Secure Outbound Telemetry Transport
```

## Phase 2 status

Phase 2 adds:

- Durable SQLite buffer (`internal/buffer`)
- MQTT/TLS publisher with event-driven flusher
- Prometheus buffer/MQTT metrics

`publisher.mode: stdout` remains available for snmpsim-only local work.

## Build and run

```bash
cd services/snmp-collector
go test ./...
go run ./cmd/collector -config configs/collector.example.yaml
```

Admin endpoints (default `:9090`):

- `GET /metrics` — Prometheus scrape
- `GET /healthz` — liveness

### Publisher modes

| Mode | Config | Behavior |
|------|--------|----------|
| `stdout` | `configs/collector.example.yaml` | JSON lines on stdout (Phase 1) |
| `mqtt` | `configs/collector.mqtt.example.yaml` | SQLite buffer → MQTT/TLS QoS 1 |

### MQTT mode

**Day-to-day GNS3 lab:** Mac cloud Compose + collector on the OrbStack Ubuntu VM:

```bash
# On Mac
./deployments/development/up.sh
./deployments/development/vxrail/sync.sh

# On VM (after setting MQTT_BROKER to the Mac host IP)
./bootstrap.sh
```

**Single-host client smoke (all services including collector):**

```bash
./deployments/end-to-end/up.sh
```

Or run Mosquitto/Postgres via development compose and the collector on the host with an example config:

```bash
./deployments/development/up.sh
cd services/snmp-collector
export MQTT_PASSWORD=secret
export MQTT_BROKER=tls://127.0.0.1:8883
go run ./cmd/collector -config configs/collector.mqtt.example.yaml
```

For production hybrid topology, see [`deployments/production/`](../../deployments/production/).

Details: [`deployments/development/README.md`](../../deployments/development/README.md), [`deployments/end-to-end/README.md`](../../deployments/end-to-end/README.md).

Buffer file defaults to `./data/buffer.db` (created automatically). Mount a persistent volume at that path in containers.

Optional env overrides:

| Variable | Purpose |
|----------|---------|
| `MQTT_PASSWORD` | Broker password (required in mqtt mode) |
| `MQTT_BROKER` | Override `mqtt.broker` |
| `MQTT_TLS_INSECURE=1` | Skip TLS verify (local/dev only) |

### Community secrets

Override per-device communities with env vars (do not commit real communities):

```bash
export SNMP_COMMUNITY_DEV_001=secret
```

Device IDs are uppercased and non-alphanumeric characters become `_`.

## Metrics (buffer / MQTT)

| Metric | Meaning |
|--------|---------|
| `collector_buffer_depth` | In-memory depth (bootstrapped from SQLite) |
| `collector_buffer_enqueue_total` | Rows inserted |
| `collector_buffer_flush_batches_total` | Flush loops that published ≥1 message |
| `collector_buffer_flushed_messages_total` | Messages published + deleted |
| `collector_mqtt_connected` | 1 when connected |
| `collector_mqtt_publish_total` | Successful publishes |
| `collector_mqtt_publish_failure_total` | Failed publishes |

## Local validation without deployment stacks

Deployment profiles do **not** include an SNMP simulator. For collector-only unit/dev work you may run any local SNMP agent (for example snmpsim) on UDP `1161` and point `configs/collector.example.yaml` at it:

```bash
cd services/snmp-collector
go run ./cmd/collector -config configs/collector.example.yaml
```

## Integration tests

Requires a running Mosquitto with certs and passwords (start `./deployments/development/up.sh` first):

```bash
export MQTT_PASSWORD='secret'
export MQTT_BROKER=tls://127.0.0.1:8883
go test -tags=integration ./tests/ -count=1 -v
```
## Container

```bash
cd services/snmp-collector
docker build -t equate/snmp-collector:dev .
docker run --rm -p 9090:9090 \
  -v "$PWD/configs/collector.example.yaml:/configs/collector.yaml:ro" \
  -v collector-data:/var/lib/collector \
  equate/snmp-collector:dev -config /configs/collector.yaml
```

For MQTT mode, mount CA certs, set `MQTT_PASSWORD`, and point `buffer.path` at a writable volume (e.g. `/var/lib/collector/buffer.db`).
