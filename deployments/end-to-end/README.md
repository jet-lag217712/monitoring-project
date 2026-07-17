# End-to-end profile

Self-hosted Docker Compose with **every** service in one project. Use this for quick validation on a client site (or any single host with Docker and reachable SNMP devices).

**No SNMP simulator.** Point [`configs/collector.yaml`](configs/collector.yaml) at real devices and set the referenced `SNMP_COMMUNITY_*` variables before acceptance.

## Services

| Service | Host port (default) |
|---------|---------------------|
| Frontend | `:80` |
| Backend API | `:8000` (admin `:9092`) |
| Ingestion | admin `:9091` |
| Mosquitto MQTT/TLS | `:8883` |
| PostgreSQL | `:5432` |
| SNMP collector | admin `:9090` |

## Quick start

```bash
# From repo root
cp deployments/end-to-end/.env.example deployments/end-to-end/.env
# Edit configs/collector.yaml with real device IPs for acceptance
./deployments/end-to-end/up.sh
./deployments/end-to-end/validate.sh
./deployments/end-to-end/smoke.sh          # synthetic MQTT → API (no SNMP required)
./deployments/end-to-end/acceptance.sh     # waits for real SNMP device telemetry
./deployments/end-to-end/down.sh           # add -v to wipe volumes
```

## Notes

- Builds from canonical sources under `services/` and `frontend/`.
- TLS certs and Mosquitto passwords are generated under `infrastructure/docker/mqtt-broker/` on first `up.sh`.
- Smoke proves MQTT → ingestion → PostgreSQL → API. Acceptance proves collector → MQTT using a live SNMP target.
