# Local testing (Mac + Debian VM)

Canonical day-to-day test environment:

| Host | Role | Path |
|------|------|------|
| **Mac** (Docker Compose) | Mosquitto, Postgres, ingestion, backend-api, frontend | this directory |
| **Debian VM** (VMware Fusion) | GNS3 + SNMP collector | [`vxrail/`](vxrail/) |

```text
Mac (deployments/local)
  mosquitto :8883, postgres, ingestion, api, frontend
        ▲
        │ MQTT/TLS
Debian VM + GNS3 (deployments/local/vxrail)
  snmp-collector → live C7200 SNMP
```

For Azure dual-plane (later), see [`../dev/`](../dev/).

## Quick start (Mac)

```bash
# From repo root
./deployments/local/up.sh
./deployments/local/down.sh        # keep volumes
./deployments/local/down.sh -v     # wipe volumes
```

Prerequisites: Docker Desktop. Migrations use `golang-migrate` or a Docker fallback.

Before first cert generation, set `MQTT_SERVER_IP` in `.env` to the Mac IP the Debian VM will use (so TLS SANs match).

## Services (Mac Compose)

| Service | Port | Role |
|---------|------|------|
| mosquitto | `8883` MQTT/TLS | Telemetry broker (bind reachable from VM) |
| postgres | `5432` | System of record |
| ingestion | admin `9091` | MQTT → Postgres |
| backend-api | `8000`, admin `9092` | REST API |
| frontend | `80` | UI (`/api` proxied to API) |

The SNMP collector is **not** in this Compose file. Run it on the Debian VM via [`vxrail/`](vxrail/).

## Collector (Debian VM)

1. Start Mac stack (`./up.sh`).
2. Copy `infrastructure/docker/mqtt-broker/certs/ca.crt` to `vxrail/certs/ca.crt` on the VM.
3. Set `MQTT_BROKER=tls://<mac-host-ip>:8883` in `vxrail/.env`.
4. `./deployments/local/vxrail/bootstrap.sh`

Details: [`vxrail/README.md`](vxrail/README.md). C7200 lab: [`remote-server/README.md`](../../remote-server/README.md).

## Integration tests (Mac)

```bash
export MQTT_PASSWORD=ingestion
export MQTT_BROKER=tls://127.0.0.1:8883
export MQTT_CA_FILE="$PWD/infrastructure/docker/mqtt-broker/certs/ca.crt"
export DATABASE_URL=postgres://ogsd_ingestion:ingestion@127.0.0.1:5432/ogsd?sslmode=disable
```

## Persistence

Named volumes `postgres-data` and `mosquitto-data` survive `down` unless you pass `-v`.

## Optional fixture

[`snmpsim/`](snmpsim/) is a standalone SNMP simulator image. It is **not** part of the default local stack (live GNS3 devices are used instead).
