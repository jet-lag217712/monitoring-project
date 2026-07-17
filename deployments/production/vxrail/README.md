# Production VxRail — on-site SNMP collector (skeleton)

Runs only the SNMP collector on a customer VxRail Ubuntu VM. All other services live on the Azure cloud plane ([`../cloud/`](../cloud/)).

## Prerequisites

- Ubuntu VM on-site with Docker Engine + Compose
- Layer-2/L3 reachability to monitored SNMP devices
- Outbound TCP to Azure Mosquitto `:8883` (no inbound from cloud required)
- Production CA certificate for Mosquitto trust (`certs/ca.crt`)
- Device inventory in [`configs/collector.yaml`](configs/collector.yaml)
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
2. Install CA + fill `.env` + inventory
3. `docker compose up -d --build` (or pull registry image)
4. Verify `GET /healthz` on collector admin port
5. Confirm samples appear in cloud API/UI

## Verification

```bash
curl -fsS http://127.0.0.1:9090/healthz
docker compose logs -f snmp-collector
# On cloud: API shows site devices after first successful poll
```

## Notes

- Buffer volume `collector-data` retains telemetry during MQTT outages
- Prefer registry image tags in production; build-from-source is a skeleton default
- Do not commit community strings or production passwords
