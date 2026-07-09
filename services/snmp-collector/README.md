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

## Phase 1 status

Phase 1 implements polling, normalization, stdout publishing, and Prometheus metrics.
MQTT transport and SQLite buffering are Phase 2 — the `Publisher` interface and metric
names (`collector_mqtt_*`, `collector_buffer_*`) are registered now so Phase 2 is a swap.

## Build and run

```bash
cd services/snmp-collector
go test ./...
go run ./cmd/collector -config configs/collector.example.yaml
```

Admin endpoints (default `:9090`):

- `GET /metrics` — Prometheus scrape
- `GET /healthz` — liveness

Events are written as JSON lines to **stdout**. Logs go to **stderr**.

### Community secrets

Override per-device communities with env vars (do not commit real communities):

```bash
export SNMP_COMMUNITY_DEV_001=secret
```

Device IDs are uppercased and non-alphanumeric characters become `_`.

## Local validation with snmpsim

Requires Docker. On macOS without Docker Desktop, Colima works:

```bash
brew install docker docker-compose colima
colima start
# optional: export DOCKER_HOST=unix://$HOME/.colima/default/docker.sock
```

```bash
# Terminal 1 — simulated SNMP agent on UDP 1161
docker compose -f deployments/local/snmpsim/docker-compose.yaml up --build
```

**Docker Desktop / Linux:** the example config points at `127.0.0.1:1161`, so you can run the collector on the host:

```bash
cd services/snmp-collector
go run ./cmd/collector -config configs/collector.example.yaml
```

**Colima:** UDP port publish to the Mac host is unreliable. Run the collector on the same Docker network instead:

```bash
# from services/snmp-collector after building the image
docker build -t equate/snmp-collector:dev .
docker run --rm --network snmpsim_default -p 9090:9090 \
  -v "$PWD/configs/collector.example.yaml:/configs/collector.yaml:ro" \
  equate/snmp-collector:dev -config /configs/collector.yaml
```

When using the Docker-network path, set `devices[0].host` to `snmpsim` (the compose service name) instead of `127.0.0.1`.

Expect JSON device/interface events on stdout and `collector_poll_success_total`
increasing at `http://127.0.0.1:9090/metrics`.

## Container

```bash
cd services/snmp-collector
docker build -t equate/snmp-collector:dev .
docker run --rm -p 9090:9090 \
  -v "$PWD/configs/collector.example.yaml:/configs/collector.yaml:ro" \
  equate/snmp-collector:dev -config /configs/collector.yaml
```

Mount a real device config and ensure the container can reach SNMP targets on the network.
