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

## Current v2 status

Phase 1 adds strict runtime configuration and inventory foundations:

- static YAML plus optional managed inventory merging (static entries win on duplicate IDs)
- `community_env` references instead of plaintext SNMP communities
- dependency, interface-filter, temperature-policy, discovery-policy, and polling validation
- atomic managed-inventory writes and transactional `SIGHUP` reloads
- `collector validate -config …` for secret-value-independent CI/operator validation

Phase 2 adds polling, profiles, and isolated discovery:

- core SNMPv2-MIB identity + IF-MIB inventory/counters (including `ifLastChange`)
- stage-owned `DevicePollResult` pipeline: Core → Profile → Filter → Normalize → Publisher (v1/v2/both)
- Cisco/Arista `sysObjectID` detection with static capability flags and fixture-tested enrichment
- interface filter annotations (`selected` / `excluded_default` / `excluded_rule`)
- operator-invoked `collector discover` with token-bucket rate limiting; never auto-enrolls devices

Phase 3 adds health and reachability correlation:

- consecutive failure ledger and post-cycle temperature / dependency-DAG evaluation
- local health transitions (`initial` / `entered` / `recovered` only)
- health, dependency-impacted, pending-failure, and readiness Prometheus metrics
- `GET /readyz` (config + buffer + usable publisher; MQTT must be connected in mqtt mode)

Phase 4 adds MQTT v2 telemetry and heartbeat publishing:

- enveloped v2 device, interface, health, and heartbeat events on versioned routes
- `publisher.telemetry_version: v1 | v2 | both` (default `v2`; `v1`/`both` emergency/lab only)
- durable outbox reuse for all event families; `event_id` allocated before enqueue
- startup + periodic heartbeats with build metadata (`unknown` fallback) and outbox depth sampled before enqueue
- heartbeat publish success/failure/duration metrics

Phase 6 adds the local operator control plane and Bubble Tea TUI:

- Unix NDJSON control socket (`admin.control_socket`) with protocol version `1`, stable error codes, size/timeout limits
- revision-bound prepare/commit mutations for thresholds and dependencies; secret-free audit log
- managed overlays for temperature, upstreams, interface filters, and discovery rate/burst
- `collector tui` client with inventory, device, discovery, thresholds, transport, and configuration views
- systemd least-privilege unit under `deployments/systemd/`

The managed inventory file contains only policy overlays and a `devices` list. A configured but
missing file is treated as empty. Existing managed files must be owner-only
(`0600`). Publisher, MQTT, buffer, admin, site, and collector identity settings
remain startup-only.

## Existing transport status

The collector retains:

- Durable SQLite buffer (`internal/buffer`)
- MQTT/TLS publisher with event-driven flusher
- Prometheus buffer/MQTT metrics

`publisher.mode: stdout` remains available for snmpsim-only local work.

## Build and run

```bash
cd services/snmp-collector
go test ./...
go run ./cmd/collector -config configs/collector.example.yaml

# Validate without requiring secret values.
go run ./cmd/collector validate -config configs/collector.example.yaml

# Local operator TUI (requires a running collector with admin.control_socket set).
go run ./cmd/collector tui -socket /run/snmp-collector/control.sock

# Operator-invoked discovery (isolated from the poll scheduler).
export SNMP_DISCOVERY_COMMUNITY=REPLACE_ME_LOCALLY
go run ./cmd/collector discover -config configs/collector.example.yaml -output discovery-candidates.json
# Review candidates, then either:
#   go run ./cmd/collector discover export -from reviewed.json -to discovery-export.yaml
#   go run ./cmd/collector discover accept -config configs/collector.example.yaml -from reviewed.json
# Accept writes managed inventory only; send SIGHUP or use TUI/control config.reload to activate.
```

Admin endpoints (default `:9090`):

- `GET /metrics` — Prometheus scrape
- `GET /healthz` — liveness
- `GET /readyz` — readiness (active config, buffer available, usable publisher)

Control plane (when `admin.control_socket` is set):

- Versioned NDJSON over a Unix socket (`0600`)
- Status methods: `status.summary`, `inventory.list`, `device.get`, `transport.get`, `config.get`, `discovery.status`
- Mutations: revision-bound `thresholds`/`dependencies` prepare+commit, then `config.reload`
- Audit log beside the managed inventory (`*.audit.log`), secret-free

See [`deployments/`](deployments/) for the systemd unit and permission/rollback notes.

### Discovery

`collector discover` scans only `discovery.allowed_cidrs`, resolves
`discovery.community_env` at probe time, and applies the configured token bucket
before every SNMP probe. It never schedules polling and never auto-enrolls
devices. Export produces reviewable YAML; accept uses the atomic managed-
inventory writer and requires a subsequent reload to become active.

### Publisher modes

| Mode | Config | Behavior |
|------|--------|----------|
| `stdout` | `configs/collector.example.yaml` | JSON lines on stdout |
| `mqtt` | `configs/collector.mqtt.example.yaml` | SQLite buffer → MQTT/TLS QoS 1 |

`publisher.telemetry_version` controls which families are emitted:

| Value | Behavior |
|-------|----------|
| `v1` | Flat `metric/device` and `metric/interface` only |
| `v2` | Enveloped `telemetry/v2/{device,interface,health}` + heartbeat |
| `both` | Dual-publish v1 and v2 (emergency/lab override; unsupported in deployments) |

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

### SNMP community environment references

Each device names the environment variable that supplies its SNMP community.
The collector resolves the value only when it creates an SNMP session:

```bash
export SNMP_COMMUNITY_DEV_001=REPLACE_ME_LOCALLY
```

Do not place the community value in static YAML, managed inventory, logs, or
deployment files. Validation checks only the environment variable name.

### Reload

Send `SIGHUP` to reload inventory and polling-policy changes without restarting:

```bash
kill -HUP "$(pidof collector)"
```

The complete configuration is parsed and validated before activation. Invalid
reloads leave the current polling snapshot unchanged. In-flight polls continue
using the snapshot they started with.

## Metrics (buffer / MQTT / Phase 2)

| Metric | Meaning |
|--------|---------|
| `collector_buffer_depth` | In-memory depth (bootstrapped from SQLite) |
| `collector_buffer_enqueue_total` | Rows inserted |
| `collector_buffer_flush_batches_total` | Flush loops that published ≥1 message |
| `collector_buffer_flushed_messages_total` | Messages published + deleted |
| `collector_mqtt_connected` | 1 when connected |
| `collector_mqtt_publish_total` | Successful publishes |
| `collector_mqtt_publish_failure_total` | Failed publishes |
| `collector_config_reload_success_total` | Successful SIGHUP reloads |
| `collector_config_reload_failure_total` | Failed SIGHUP reloads |
| `collector_profile_detection_total` | Profile matches by name and match kind |
| `collector_profile_fallback_total` | Core-only fallbacks |
| `collector_profile_collection_failure_total` | Vendor enrichment failures |
| `collector_profile_duration_seconds` | Vendor collection duration |
| `collector_interface_selection_total` | Interfaces by filter outcome |
| `collector_discovery_attempts_total` | Discovery probe attempts |
| `collector_discovery_candidates_total` | Successful discovery candidates |
| `collector_discovery_errors_total` | Failed discovery probes |
| `collector_discovery_rate_limit_waits_total` | Probes delayed by the token bucket |

## Local validation without deployment stacks

Deployment profiles do **not** include an SNMP simulator. For collector-only unit/dev work you may run any local SNMP agent (for example snmpsim) on UDP `1161` and point `configs/collector.example.yaml` at it:

```bash
cd services/snmp-collector
export SNMP_COMMUNITY_DEV_001=REPLACE_ME_LOCALLY
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
