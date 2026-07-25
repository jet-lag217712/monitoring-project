# SNMP Collector

Customer OOB Monitoring Plane service. Polls SNMP devices on-site, buffers
telemetry in SQLite, publishes outbound-only over MQTT/TLS to the UI/UX Cloud
Plane, and exposes a local Equate-branded Bubble Tea TUI over a Unix control
socket.

**Does not:** host PostgreSQL, the Backend API, cloud ingestion, or the React
dashboard. Does not store plaintext SNMP communities or MQTT passwords in YAML.

```text
SNMP Devices → SNMP Collector → MQTT/TLS (outbound) → Cloud Plane
                     ↑
              collector tui (Unix socket, local only)
```

---

## Prerequisites

| Requirement | Notes |
|-------------|--------|
| Go **1.25+** | From `go.mod`; needed for local `go run` / build |
| Docker Engine + Compose | For VxRail lab/production paths |
| Network | L2/L3 to SNMP devices; outbound TCP to MQTT broker `:8883` |
| CA certificate | Mosquitto trust (`certs/ca.crt`) for MQTT/TLS paths |
| Secrets via env | `SNMP_COMMUNITY_*`, `MQTT_PASSWORD` — never in YAML or git |

---

## Path A — Local stdout (fastest smoke)

Use this to prove the binary starts and polls without MQTT.

```bash
cd services/snmp-collector

# 1. Community for the example device (name must match community_env in YAML)
export SNMP_COMMUNITY_DEV_001=REPLACE_ME_LOCALLY

# 2. Enable the control socket for the TUI
mkdir -p run
cp configs/collector.example.yaml /tmp/collector.local.yaml
# Set admin.control_socket (example uses empty string by default):
#   admin:
#     listen: ":9090"
#     control_socket: "./run/control.sock"
# Or with a one-liner:
python3 - <<'PY'
from pathlib import Path
p = Path("/tmp/collector.local.yaml")
text = Path("configs/collector.example.yaml").read_text()
text = text.replace('control_socket: ""', 'control_socket: "./run/control.sock"')
p.write_text(text)
print("wrote", p)
PY

# 3. Validate (does not need secret values — only env var names)
go run ./cmd/collector validate -config /tmp/collector.local.yaml

# 4. Run
go run ./cmd/collector -config /tmp/collector.local.yaml
```

In another terminal:

```bash
curl -fsS http://127.0.0.1:9090/healthz
curl -fsS http://127.0.0.1:9090/readyz

cd services/snmp-collector
go run ./cmd/collector tui -socket ./run/control.sock -theme auto
```

`publisher.mode: stdout` prints JSON lines. Point `devices[].host`/`port` at a
real agent (default `127.0.0.1:161`) or a local simulator on `1161`.

---

## Path B — MQTT lab (Mac cloud + VxRail VM)

Day-to-day GNS3 / OrbStack lab: cloud plane on the Mac, collector on the Ubuntu VM.

### On the Mac

```bash
# From repo root
./deployments/development/up.sh
./deployments/development/vxrail/sync.sh
```

### On the VM

```bash
cd deployments/development/vxrail   # or the synced remote dir

# First boot: Equate setup wizard (shared env → site count/CIDRs → compose → scan → thresholds)
./bootstrap.sh
# Writes shared .env, sites/manifest.yaml, per-site configs/managed inventory, .setup-complete

# Reconfigure site count or CIDRs:
./bootstrap.sh --reconfigure

# Day-2 operator TUI (per site):
collector tui -socket ./sites/site-001/run/control.sock -theme auto
docker compose -f docker-compose.yml -f docker-compose.sites.generated.yml \
  exec -it snmp-collector-site-001 /collector tui -socket /run/snmp-collector/control.sock -theme auto
```

Confirm devices appear in the Mac API/UI after the first successful poll.

Details: [`deployments/development/vxrail/README.md`](../../deployments/development/vxrail/README.md),
[`deployments/development/README.md`](../../deployments/development/README.md).

---

## Path C — Customer site (production VxRail)

On-site Ubuntu VM; cloud plane in Azure. Compose skeleton:
[`deployments/production/vxrail/`](../../deployments/production/).

```bash
cd deployments/production/vxrail

# 1. Prerequisites
# - Docker Engine + Compose on the site VM
# - Reachability to SNMP devices
# - Outbound TCP to Azure Mosquitto :8883
# - Production CA at certs/ca.crt

# 2. Secrets
cp .env.example .env
# Edit .env:
#   MQTT_BROKER=tls://<azure-broker>:8883
#   MQTT_PASSWORD=<production password>
#   SNMP_COMMUNITY_CORE_SWITCH=<community>
# Never commit .env

# 3. Inventory and TLS
# Edit configs/collector.yaml (site_id, devices, community_env, mqtt, buffer paths)
# Ensure admin.control_socket is /run/snmp-collector/control.sock
# Place production CA: certs/ca.crt

# 4. State volume must be writable by UID 65532
# Compose uses user: "65532:65532" and volume collector-state → /var/lib/snmp-collector

# 5. Validate (host binary or one-shot container)
collector validate -config ./configs/collector.yaml
# Or:
# docker compose run --rm --entrypoint /collector snmp-collector validate -config /configs/collector.yaml

# 6. Start (prefer registry image tags in real production)
docker compose up -d --build

# 7. Verify
curl -fsS http://127.0.0.1:${COLLECTOR_ADMIN_PORT:-9090}/healthz
curl -fsS http://127.0.0.1:${COLLECTOR_ADMIN_PORT:-9090}/readyz
docker compose logs -f snmp-collector

# 8. Operator TUI (socket mounted at ./run)
collector tui -socket ./run/control.sock -theme auto
```

Confirm v2 telemetry in the cloud API/UI. Runbooks:
[`deployments/runbooks/`](../../deployments/runbooks/),
field acceptance: [`field-acceptance-gns3.md`](../../deployments/runbooks/field-acceptance-gns3.md).

---

## Path D — systemd host install

Least-privilege unit: [`deployments/systemd/snmp-collector.service`](deployments/systemd/snmp-collector.service).

1. Create user/group `snmp-collector` (no login shell).
2. Install binary to `/usr/local/bin/collector`.
3. Place static config at `/etc/equate/collector.yaml` (root-owned, readable by service user).
4. Create `/var/lib/snmp-collector` owned `snmp-collector:snmp-collector` mode `0750`.
5. Point managed inventory, buffer, and control socket at state/runtime dirs:
   - managed: `/var/lib/snmp-collector/managed-inventory.yaml` (`0600`)
   - audit: `/var/lib/snmp-collector/managed-inventory.yaml.audit.log` (`0600`)
   - socket: `/run/snmp-collector/control.sock` (`0600`, via `RuntimeDirectory=`)
   - `publisher.telemetry_version: v2`
6. Secrets only in `/etc/equate/snmp-collector.env` (`community_env` / `password_env` names in YAML).
7. `systemctl enable --now snmp-collector`

```bash
collector validate -config /etc/equate/collector.yaml
collector tui -socket /run/snmp-collector/control.sock -theme auto
# or: systemctl reload snmp-collector   # SIGHUP
```

Full notes: [`deployments/README.md`](deployments/README.md).

---

## Operator TUI

Equate-branded Bubble Tea client of the collector control socket. Local OS access
only — never publish the socket over TCP or HTTP.

```bash
collector tui -socket /run/snmp-collector/control.sock -theme auto
collector setup -dir deployments/development/vxrail -theme auto   # first-boot wizard
# -theme light | dark | auto (default)
# NO_COLOR=1 disables ANSI color
```

| Key | Action |
|-----|--------|
| `1`–`6` | Inventory, Device, Discovery, Thresholds, Transport, Config |
| `tab` / `←` `→` | Switch views |
| `r` | Refresh |
| `R` | `config.reload` |
| `t` | Edit temperature threshold (text input → prepare → `y`/`n` commit) |
| `S` / `A` / `e` | Discovery: scan / accept successful / edit CIDR policy |
| `d` | Edit device upstream dependencies (device/inventory views) |
| `n` / `p` | Next/previous device (device view) |
| `↑` `↓` | Scroll |
| `q` | Quit |

Branding: salmon `//` mark + **Equate** wordmark, dashboard status colors
(ok / caution / alert / unknown), adaptive light/dark from terminal background.
Decision: [`.ai/decisions/collector-8.md`](../../.ai/decisions/collector-8.md).

---

## Secrets and inventory

- Each device sets `community_env` to an environment variable name; the collector
  resolves the value only when opening an SNMP session.
- MQTT uses `password_env` (typically `MQTT_PASSWORD`).
- **Static YAML** is read-only for the TUI (host, port, version, `community_env`,
  identity, MQTT, buffer, discovery CIDRs).
- **Managed inventory** (`inventory.managed_path`) is the only file the TUI
  writes: thresholds, upstreams, interface filters, discovery rate/burst.
  Mode `0600`; missing file = empty overlays.
- Reload: `SIGHUP`, `systemctl reload`, or TUI `R` / post-commit reload.
  Invalid reloads keep the prior runtime snapshot.

```bash
export SNMP_COMMUNITY_DEV_001=REPLACE_ME_LOCALLY
export MQTT_PASSWORD=secret
```

---

## Admin endpoints and control plane

Default admin listen `:9090` (scrape/liveness only when published):

| Endpoint | Purpose |
|----------|---------|
| `GET /metrics` | Prometheus scrape |
| `GET /healthz` | Liveness |
| `GET /readyz` | Ready: active config, buffer available, usable publisher (MQTT connected in mqtt mode) |

Control socket (when `admin.control_socket` is set), protocol version `1`:

| Kind | Methods |
|------|---------|
| Status | `status.summary`, `inventory.list`, `device.get`, `transport.get`, `config.get`, `discovery.status` |
| Mutations | `thresholds` / `dependencies` prepare+commit (revision-bound), then `config.reload` |

Audit log sits beside the managed inventory (`*.audit.log`), secret-free.

---

## Publisher modes

| Mode | Config | Behavior |
|------|--------|----------|
| `stdout` | `configs/collector.example.yaml` | JSON lines on stdout |
| `mqtt` | `configs/collector.mqtt.example.yaml` | SQLite outbox → MQTT/TLS QoS 1 |

| `publisher.telemetry_version` | Behavior |
|-------------------------------|----------|
| `v1` | Flat metric routes only |
| `v2` | Enveloped telemetry + heartbeat (production default) |
| `both` | Dual-publish (lab/emergency only) |

Optional env overrides: `MQTT_PASSWORD`, `MQTT_BROKER`, `MQTT_TLS_INSECURE=1` (dev only).

Single-host end-to-end smoke (all services): `./deployments/end-to-end/up.sh`.

---

## Discovery

```bash
export SNMP_DISCOVERY_COMMUNITY=REPLACE_ME_LOCALLY
go run ./cmd/collector discover -config configs/collector.example.yaml -output discovery-candidates.json
# Review, then:
#   go run ./cmd/collector discover export -from reviewed.json -to discovery-export.yaml
#   go run ./cmd/collector discover accept -config configs/collector.example.yaml -from reviewed.json
# Accept writes managed inventory only; reload to activate (TUI R or SIGHUP).
```

Discovery never auto-enrolls devices and never exceeds configured probe rate/burst.

---

## Container (standalone)

```bash
cd services/snmp-collector
docker build -t equate/snmp-collector:dev .
docker run --rm -p 9090:9090 \
  -v "$PWD/configs/collector.example.yaml:/configs/collector.yaml:ro" \
  -v collector-data:/var/lib/collector \
  equate/snmp-collector:dev -config /configs/collector.yaml
```

For MQTT: mount CA certs, set `MQTT_PASSWORD`, and point `buffer.path` at a
writable volume.

---

## Troubleshooting

| Symptom | Likely cause / fix |
|---------|-------------------|
| TUI: dial control socket failed | Collector not running, wrong `-socket` path, or `admin.control_socket` empty |
| TUI/setup on **Docker Desktop (Mac)**: socket file exists but dial refused | Bind-mounted Unix sockets are not dialable from the macOS host; use `docker compose -f docker-compose.yml -f docker-compose.sites.generated.yml exec -it snmp-collector-site-001 /collector tui -socket /run/snmp-collector/control.sock -theme auto`. First-boot `collector setup` auto-falls back to `docker compose exec … rpc` for review/threshold steps after rebuild. |
| TUI: permission denied | Socket must be readable by your user; compose mount `./run` |
| Log: `chmod control socket: … invalid argument` then restart loop | Fixed in current builds: Docker Desktop/macOS bind mounts reject socket chmod — collector now warns and keeps listening. Rebuild/restart the image if you still see a fatal chmod exit |
| `/readyz` fails in mqtt mode | MQTT disconnected, bad CA/password/broker, or buffer unavailable |
| Reload rejected | Validation failed — prior snapshot stays active; fix managed/static YAML and retry |
| Managed write fails | Path missing or not `0600` / not writable by collector UID (`65532` in compose) |
| No devices in cloud UI | Check polls in TUI device view, MQTT connected in transport view, cloud ingestion |

---

## Build and test

```bash
cd services/snmp-collector
go test ./...
go test -tags=integration ./tests/ -count=1 -v   # needs ./deployments/development/up.sh + MQTT_PASSWORD
```

---

## Appendix — phase capability map

| Phase | Capability |
|-------|------------|
| 1 | Static + managed inventory, `community_env`, validate, SIGHUP reload |
| 2 | Core/IF-MIB poll, Cisco/Arista profiles, `collector discover` |
| 3 | Health DAG, dependency impact, `/readyz` |
| 4 | MQTT v2 telemetry + heartbeats, durable outbox |
| 6 | Unix control protocol + TUI |
| 8 | Equate-branded adaptive TUI (this README’s operator experience) |

Architecture: [`.ai/roadmap/snmp-collector-v2.md`](../../.ai/roadmap/snmp-collector-v2.md),
[`docs/architecture/snmp-collector.md`](../../docs/architecture/snmp-collector.md).
