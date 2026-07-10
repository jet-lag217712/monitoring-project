# Dev deployments (GNS3 + Azure)

Path for running the **cloud plane on Azure** while the collector stays on-prem / GNS3.

Day-to-day testing (Mac Compose + Debian VM) lives under [`../local/`](../local/) — use that first.

| Plane | Location | Artifact |
|-------|----------|----------|
| On-prem (VxRail) | GNS3 / Debian VM | [`vxrail/`](vxrail/) Compose — collector → **Azure** Mosquitto |
| Cloud | Azure | [`cloud/`](cloud/) **runbook** — one Linux VM + Terraform Postgres |

```text
GNS3 / Debian VM                      Azure
┌─────────────────┐                   ┌──────────────────────────────┐
│ snmp-collector  │── MQTT/TLS ─────▶│ Linux VM: mosquitto,          │
│ live C7200 SNMP │   egress only     │   ingestion, api, frontend   │
└─────────────────┘                   │ Flexible Server: PostgreSQL  │
                                      └──────────────────────────────┘
```

## Order of operations

1. Deploy Azure cloud plane — [`cloud/README.md`](cloud/README.md)
2. Point collector at Azure — [`vxrail/README.md`](vxrail/README.md)
3. Confirm SNMP to C7200s and MQTT egress

## Related

- Local testing (Mac + Debian): [`../local/README.md`](../local/README.md)
- C7200 lab: [`remote-server/README.md`](../../remote-server/README.md)
- Postgres Terraform: [`infrastructure/terraform/README.md`](../../infrastructure/terraform/README.md)
