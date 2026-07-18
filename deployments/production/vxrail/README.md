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
| [`docker-compose.yml`](docker-compose.yml) | Collector only + persistent SQLite state + control socket mount |
| [`.env.example`](.env.example) | `MQTT_BROKER`, passwords, ports |
| [`configs/collector.yaml`](configs/collector.yaml) | Site inventory template (v2 telemetry) |
| [`managed/managed-inventory.yaml`](managed/managed-inventory.yaml) | Operator-managed discovery CIDRs + accepted devices (host bind) |
| [`run/`](run/) | Host bind for Unix control socket (`control.sock`) |
| [`certs/`](certs/) | Place production `ca.crt` only (no private keys) |

## Rollout order

1. Cloud plane healthy (migrations applied) and Mosquitto reachable from site
2. Install CA at `certs/ca.crt`
3. Run first-boot setup (writes `.env`, managed discovery policy, starts collector, discovers devices):

```bash
./bootstrap.sh
# or: collector setup -dir . -theme auto
```

4. `collector validate -config /configs/collector.yaml` (or run the equivalent image command)
5. Verify `GET /healthz` and `GET /readyz` on collector admin port
6. Confirm v2 samples appear in cloud API/UI

Subsequent starts skip the wizard when `.setup-complete` exists.

## Verification

```bash
curl -fsS http://127.0.0.1:9090/healthz
curl -fsS http://127.0.0.1:9090/readyz
docker compose logs -f snmp-collector

# Branded local TUI when the control socket is mounted on the host:
collector tui -socket ./run/control.sock -theme auto
# Force light or dark: -theme light | -theme dark

# On cloud: API shows site devices after first successful poll
```

## Notes

- State volume `collector-state` holds `/var/lib/snmp-collector` (buffer + managed inventory + audit)
- Control socket and audit log must remain owner-only (`0600`); never publish them as TCP ports
- Admin `:9090` is scrape/liveness only when published
- Invalid managed writes/reloads leave the prior runtime snapshot active — replace the managed file and reload to roll back overlays
- Prefer registry image tags in production; build-from-source is a skeleton default
- Do not commit community strings or production passwords
- Runbooks: [`../../runbooks/`](../../runbooks/)
- Least-privilege systemd example: [`services/snmp-collector/deployments/systemd/snmp-collector.service`](../../../services/snmp-collector/deployments/systemd/snmp-collector.service)
