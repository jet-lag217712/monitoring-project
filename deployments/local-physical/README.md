# Local-physical plant (pre-client E2E)

End-to-end testing on a **physical** network before a client site. The Mac runs both the cloud plane and the SNMP collector.

| Plant | Purpose |
|-------|---------|
| [`../local/`](../local/) | Mac cloud Compose + Debian VM / GNS3 collector |
| **`local-physical/`** (this plant) | Mac cloud + Mac collector → **physical** SNMP devices |
| [`../dev/`](../dev/) | Azure cloud + collector → Azure MQTT |

```text
Mac
├── deployments/local Compose
│     mosquitto :8883, postgres, ingestion, api, frontend
└── deployments/local-physical/vxrail
      snmp-collector (host go run — recommended)
         ├── SNMP ──▶ physical network devices
         └── MQTT ──▶ tls://127.0.0.1:8883
```

Cloud Compose is **not** duplicated here. Reuse [`../local/`](../local/).

## Startup order

1. Start cloud plane on the Mac:

```bash
./deployments/local/up.sh
```

2. Edit physical device inventory in [`vxrail/configs/collector.yaml`](vxrail/configs/collector.yaml) (replace placeholder hosts/communities).

3. Ensure the Mac’s LAN IP is allowed on device SNMP ACLs and can route to those devices.

4. Start the collector — see [`vxrail/README.md`](vxrail/README.md) (host `go run` is primary).

## E2E checklist

- [ ] `curl -sf http://127.0.0.1:9091/healthz` (ingestion)
- [ ] `curl -sf http://127.0.0.1:9092/healthz` (API admin)
- [ ] `curl -sf http://127.0.0.1:9090/healthz` (collector, after start)
- [ ] Collector logs show SNMP polls succeeding (no timeout storms)
- [ ] Ingestion metrics show MQTT connected / messages accepted
- [ ] UI at `http://127.0.0.1/` shows sites/devices after data arrives
- [ ] `psql` or API returns recent metric samples for a polled device

## Networking notes

- Prefer running the collector **on the Mac host** (not Docker) so SNMP UDP uses the real LAN stack.
- Mosquitto is on `127.0.0.1:8883` from the collector’s perspective.
- Do not commit real site communities or production IPs; use env overrides (`SNMP_COMMUNITY_<DEVICE_ID>`) for secrets.
