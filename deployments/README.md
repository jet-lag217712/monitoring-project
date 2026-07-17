# Deployments

Three deployment profiles. Pick one based on what you are doing:

| Profile | When to use |
|---------|-------------|
| [`end-to-end/`](end-to-end/) | Single-host client-site smoke: **all** services in one Compose project (includes collector). No SNMP simulator. |
| [`development/`](development/) | Day-to-day lab: Mac cloud plane + OrbStack Ubuntu VM collector (GNS3 **Cloud** node). |
| [`production/`](production/) | Hybrid skeleton: Azure cloud plane + on-site VxRail collector. No Terraform yet. |

```text
SNMP devices → Collector → MQTT/TLS → Ingestion → PostgreSQL → Backend API → Frontend
```

## Decision guide

1. **Quick client demo / single machine with real SNMP?** → `end-to-end/`
2. **Developing with GNS3 on OrbStack?** → `development/` (+ `development/vxrail/`)
3. **Preparing a real customer hybrid deploy?** → `production/` (fill secrets; Terraform later)

## Commands

```bash
# End-to-end (all services)
./deployments/end-to-end/up.sh
./deployments/end-to-end/smoke.sh          # v2 MQTT → API
./deployments/end-to-end/acceptance.sh     # real SNMP required
./deployments/end-to-end/down.sh

# Development cloud plane (Mac)
./deployments/development/up.sh
./deployments/development/smoke.sh
./deployments/development/vxrail/sync.sh
./deployments/development/down.sh

# Aggregate checks
./deployments/test.sh --quick
./deployments/test.sh --with-smoke
```

## Runbooks

See [`runbooks/`](runbooks/) for install, inventory, credential rotation, queue
remediation, rollback/restore, V2 cutover, and GNS3 field acceptance.

## Port map (defaults)

| Port | Service |
|------|---------|
| 80 | Frontend |
| 8000 | Backend API REST |
| 9092 | Backend API admin |
| 9091 | Ingestion admin |
| 9090 | Collector admin |
| 8883 | Mosquitto MQTT/TLS |
| 5432 | PostgreSQL |

## Configuration ownership

| Concern | Owner |
|---------|--------|
| Service source code | `services/*`, `frontend/` |
| MQTT broker image / cert scripts | `infrastructure/docker/mqtt-broker/` |
| DB migrations / roles | `database/migrations/`, `infrastructure/script/` |
| Compose, env, per-profile YAML | `deployments/<profile>/` |
| Collector inventory | `deployments/*/configs/collector.yaml` or `*/vxrail/configs/` |

Never copy Go service trees into `deployments/`. The development VM sync places a **runtime** snapshot under `vxrail/src/` on the remote host only.

## Promotion checklist

1. `end-to-end` smoke + on-site acceptance with real devices
2. `development` cloud smoke + VM collector → GNS3
3. Fill `production` secrets, TLS, inventory, image tags
4. Deploy Azure cloud plane, then on-site collector
5. Verify healthz + UI; document rollback image digests

## Testing

| Layer | How |
|-------|-----|
| Validate compose/config | `*/validate.sh` or `./deployments/test.sh --quick` |
| Cloud pipeline smoke | Synthetic **v2** MQTT → API (`smoke.sh`) |
| MQTT outage drill | `./deployments/lib/mqtt_outage_drill.sh deployments/end-to-end` |
| Real SNMP acceptance | `end-to-end/acceptance.sh` + [`runbooks/field-acceptance-gns3.md`](runbooks/field-acceptance-gns3.md) |
| CI | [`.github/workflows/deployments.yml`](../.github/workflows/deployments.yml) |

CI covers unit tests, compose validation, image builds, and cloud smoke. Real SNMP / GNS3 remain manual release gates.
