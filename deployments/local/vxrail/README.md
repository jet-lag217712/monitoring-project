# Local VxRail (`ogsd-local-vxrail`)

SNMP collector for day-to-day testing on the **Debian VM** (VMware Fusion) with GNS3.

Publishes MQTT/TLS **outbound only** to Mosquitto on the **Mac** host ([`../`](../) Compose stack).

No snmpsim. Device inventory is the live C7200 lab ([`remote-server/README.md`](../../../remote-server/README.md)).

## Prerequisites

- Docker + Compose on the Debian VM
- Mac cloud plane already up (`../up.sh`) with Mosquitto on TCP `8883`
- Mac host IP reachable from the VM (Fusion shared/host network)
- `certs/ca.crt` — CA that signed the Mac Mosquitto server cert
- Collector source IP allowed by router SNMP ACL (see remote-server README)

## Quick start

On the **Mac** first:

```bash
./deployments/local/up.sh
```

On the **Debian VM**:

```bash
cd deployments/local/vxrail
cp .env.example .env
# MQTT_BROKER=tls://<mac-host-ip>:8883
# Place Mac CA at certs/ca.crt (or let bootstrap copy if same repo checkout has certs)
./bootstrap.sh
```

## Configuration

| Variable | Example | Notes |
|----------|---------|--------|
| `MQTT_BROKER` | `tls://192.168.64.1:8883` | Mac Mosquitto as seen from the VM |
| `MQTT_PASSWORD` | `secret` | Must match Mac Mosquitto `collector` user |

Edit [`configs/collector.yaml`](configs/collector.yaml) if loopback addresses or community differ.

## Health

```bash
curl -sf http://127.0.0.1:9090/healthz
docker compose --env-file .env logs -f snmp-collector
```

## Troubleshooting

| Symptom | Check |
|---------|--------|
| TLS verify failed | Wrong/missing `certs/ca.crt`; regenerate Mac certs with `MQTT_SERVER_IP=<mac-ip>` |
| Connection refused | Mac firewall; Mosquitto not published; wrong Mac IP from VM |
| SNMP timeouts | Source IP not in ACL; no route to Loopback0; wrong community |

## Azure instead of Mac?

Use [`../../dev/vxrail/`](../../dev/vxrail/) and point `MQTT_BROKER` at Azure Mosquitto ([`../../dev/cloud/`](../../dev/cloud/)).
