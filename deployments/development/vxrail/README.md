# Development VxRail — on-site SNMP collector (OrbStack / GNS3 lab)

Runs only the SNMP collector on a lab Ubuntu VM. Cloud services run on the Mac via [`../`](../) (`deployments/development`).

## Prerequisites

- OrbStack/Ubuntu VM with Docker Engine + Compose
- Layer-2 reachability to GNS3 SNMP devices
- Outbound TCP to Mac Mosquitto `:8883` (GNS3 Cloud / host network as documented in the parent README)
- Lab CA certificate for Mosquitto trust (`certs/ca.crt`)
- Device inventory in [`configs/collector.yaml`](configs/collector.yaml)
- SNMP community values supplied through the `SNMP_COMMUNITY_*` environment references in `.env`
- MQTT username/password matching Mac Mosquitto ACL (`collector` publish-only)

## Layout

| File | Purpose |
|------|---------|
| [`docker-compose.yml`](docker-compose.yml) | Collector only + persistent SQLite state + control socket mount |
| [`.env.example`](.env.example) | `MQTT_BROKER`, passwords, ports |
| [`configs/collector.yaml`](configs/collector.yaml) | Site inventory template (v2 telemetry) |
| [`run/`](run/) | Host bind for Unix control socket (`control.sock`) |
| [`certs/`](certs/) | Place lab `ca.crt` only (no private keys) |

## Rollout order

1. Mac cloud plane healthy (`deployments/development/up.sh`) and Mosquitto reachable from the VM
2. Install CA, fill `.env` and the `SNMP_COMMUNITY_*` references, then review inventory
3. Ensure state volume is owned by UID `65532` (distroless nonroot) on first create
4. `collector validate -config /configs/collector.yaml` (or run the equivalent image command)
5. `docker compose up -d --build`
6. Verify `GET /healthz` and `GET /readyz` on collector admin port
7. Confirm v2 samples appear in Mac API/UI

## Verification

```bash
curl -fsS http://127.0.0.1:9090/healthz
curl -fsS http://127.0.0.1:9090/readyz
docker compose logs -f snmp-collector
# Local TUI against the mounted control socket:
# collector tui -socket ./run/control.sock
# On Mac: API shows site devices after first successful poll
```

## Notes

- State volume `collector-state` holds `/var/lib/snmp-collector` (buffer + managed inventory + audit)
- Control socket path is `/run/snmp-collector/control.sock` inside the container (`./run` on the host); never publish it as a TCP port
- Admin `:9090` is scrape/liveness only when published
- Do not commit community strings or lab passwords
- Field acceptance checklist: [`../../runbooks/field-acceptance-gns3.md`](../../runbooks/field-acceptance-gns3.md)
- systemd least-privilege example: [`services/snmp-collector/deployments/systemd/snmp-collector.service`](../../../services/snmp-collector/deployments/systemd/snmp-collector.service)
