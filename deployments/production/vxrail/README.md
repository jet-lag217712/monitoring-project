# Production VxRail — on-site SNMP collector (skeleton)

Runs only the SNMP collector on a customer VxRail Ubuntu VM. All other services live on the Azure cloud plane ([`../cloud/`](../cloud/)).

## Prerequisites

- Ubuntu VM on-site with Docker Engine + Compose
- Layer-2/L3 reachability to monitored SNMP devices
- Outbound TCP to Azure Mosquitto `:8883` (no inbound from cloud required)
- Production CA certificate for Mosquitto trust (`certs/ca.crt`)
- Device inventory in [`configs/collector.yaml`](configs/collector.yaml)
- SNMP community values supplied through the `SNMP_COMMUNITY_*` environment references in `.env`
- MQTT username/password matching cloud Mosquitto ACL (`collector` publish-only)

## Layout

| File | Purpose |
|------|---------|
| [`docker-compose.yml`](docker-compose.yml) | Collector only + persistent SQLite buffer |
| [`.env.example`](.env.example) | `MQTT_BROKER`, passwords, ports |
| [`configs/collector.yaml`](configs/collector.yaml) | Site inventory template |
| [`certs/`](certs/) | Place production `ca.crt` only (no private keys) |

## Rollout order

1. Cloud plane healthy and Mosquitto reachable from site
2. Install CA, fill `.env` and the `SNMP_COMMUNITY_*` references, then review inventory
3. `collector validate -config /configs/collector.yaml` (or run the equivalent image command)
4. `docker compose up -d --build` (or pull registry image)
5. Verify `GET /healthz` on collector admin port
5. Confirm samples appear in cloud API/UI

## Verification

```bash
curl -fsS http://127.0.0.1:9090/healthz
curl -fsS http://127.0.0.1:9090/readyz
docker compose logs -f snmp-collector
# Optional local TUI when admin.control_socket is mounted/available on the host
# collector tui -socket /run/snmp-collector/control.sock
# On cloud: API shows site devices after first successful poll
```

## Notes

- Buffer volume `collector-data` retains telemetry during MQTT outages
- Managed inventory is stored at `/data/managed-inventory.yaml` and is owner-only (`0600`)
- Control socket and audit log (when enabled) must remain owner-only (`0600`); never publish them
- Invalid managed writes/reloads leave the prior runtime snapshot active — replace the managed file and reload to roll back overlays
- Prefer registry image tags in production; build-from-source is a skeleton default
- Do not commit community strings or production passwords
- Least-privilege systemd example: [`services/snmp-collector/deployments/systemd/snmp-collector.service`](../../../services/snmp-collector/deployments/systemd/snmp-collector.service)