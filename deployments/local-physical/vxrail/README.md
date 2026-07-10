# Local-physical VxRail (Mac collector → physical LAN)

Pre-client E2E: run the SNMP collector **on the Mac** against **real physical devices**, publishing to Mosquitto from [`../../local/`](../../local/).

## Prerequisites

1. Cloud plane up: `./deployments/local/up.sh`
2. Mac can reach devices over SNMPv2c (UDP 161); Mac IP allowed on device ACLs
3. Go toolchain for host run (recommended), or Docker for the alternate Compose path
4. Edit [`configs/collector.yaml`](configs/collector.yaml) — replace `REPLACE_WITH_*` hosts

## Primary: host `go run` (recommended)

```bash
# From repo root — after local/up.sh
./deployments/local-physical/vxrail/bootstrap.sh --prepare-only

cd services/snmp-collector
set -a && source ../../deployments/local-physical/vxrail/.env && set +a
export MQTT_PASSWORD="${MQTT_PASSWORD:-secret}"
export MQTT_BROKER="${MQTT_BROKER:-tls://127.0.0.1:8883}"
go run ./cmd/collector -config ../../deployments/local-physical/vxrail/configs/collector.host.yaml
```

Or use the helper printed by `./bootstrap.sh` (default mode prepares certs and prints the exact `go run` command).

Admin: `http://127.0.0.1:9090/healthz`

## Secondary: Docker Compose

Compose is an alternate. On **Docker Desktop for Mac**, container networking to arbitrary LAN/SNMP targets is often unreliable — prefer host run.

On Linux with host networking:

```bash
cd deployments/local-physical/vxrail
cp .env.example .env
./bootstrap.sh --compose
```

`docker-compose.yml` uses `network_mode: host` (Linux). Do not rely on this on macOS Docker Desktop for physical SNMP.

## Configuration

| Variable | Default | Notes |
|----------|---------|--------|
| `MQTT_BROKER` | `tls://127.0.0.1:8883` | Mac local Mosquitto |
| `MQTT_PASSWORD` | `secret` | Mosquitto `collector` user |

Configs:

- [`configs/collector.host.yaml`](configs/collector.host.yaml) — host run (relative CA + buffer paths)
- [`configs/collector.yaml`](configs/collector.yaml) — container paths (`/certs`, `/data`)

Fill in real device `host` / `community` values. Override communities with `SNMP_COMMUNITY_<DEVICE_ID>` (e.g. `SNMP_COMMUNITY_CORE_SW1`).

## Troubleshooting

| Symptom | Check |
|---------|--------|
| SNMP timeouts | Mac route/ACL; wrong IP/community; firewall blocking UDP 161 |
| TLS errors | Run `./bootstrap.sh --prepare-only`; CA must match local Mosquitto |
| MQTT refused | `./deployments/local/up.sh` not running; port 8883 |
