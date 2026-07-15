# Production profile (skeleton)

Hybrid deployment for real customer sites. **No Terraform in this profile** — infrastructure provisioning comes later. This tree is an implementation-ready Compose + runbook skeleton.

```text
Customer site (VxRail Ubuntu VM)          Azure VM (cloud plane)
─────────────────────────────────         ─────────────────────────
SNMP devices                              Mosquitto (MQTT/TLS :8883)
    │                                          ▲
    ▼                                          │ outbound TLS only
SNMP Collector ───────────────────────────────┘
                                              Ingestion → PostgreSQL
                                              Backend API → Frontend
```

| Plane | Path | Contents |
|-------|------|----------|
| Cloud | [`cloud/`](cloud/) | Azure-hosted Mosquitto, Postgres (or Flexible Server), ingestion, API, frontend |
| VxRail | [`vxrail/`](vxrail/) | On-site collector only |

## Promotion checklist

1. Prove path on [`../end-to-end/`](../end-to-end/) with real SNMP devices
2. Prove lab path on [`../development/`](../development/) (Mac + OrbStack/GNS3)
3. Fill production secrets, TLS, inventory, and image tags
4. Deploy Azure cloud plane first, then on-site collector
5. Run verification steps in each README

## Explicitly deferred

- Terraform / Azure resource provisioning
- CI image publish + registry wiring
- Production secret manager integration
- Automated DNS / TLS certificate issuance
