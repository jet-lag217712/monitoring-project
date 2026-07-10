# Dev VxRail (`ogsd-dev-vxrail`)

SNMP collector for the **Azure** dual-plane path. Same live C7200 inventory as local VxRail; MQTT target is **Azure** Mosquitto.

For day-to-day testing against Mac Mosquitto, use [`../../local/vxrail/`](../../local/vxrail/) instead.

## Prerequisites

- Docker + Compose on the GNS3 / Debian host
- Azure cloud plane reachable ([`../cloud/README.md`](../cloud/README.md))
- `certs/ca.crt` — CA that signed the Azure Mosquitto server certificate
- Collector source IP allowed by router SNMP ACL

## Quick start

```bash
cd deployments/dev/vxrail
cp .env.example .env
# MQTT_BROKER=tls://<azure-public-or-private-ip>:8883
# Place Azure CA at certs/ca.crt
./bootstrap.sh
```

## Configuration

| Variable | Example | Notes |
|----------|---------|--------|
| `MQTT_BROKER` | `tls://20.x.x.x:8883` | Azure Mosquitto |
| `MQTT_PASSWORD` | `secret` | Must match Azure Mosquitto `collector` user |

Edit [`configs/collector.yaml`](configs/collector.yaml) if inventory differs.

## Health

```bash
curl -sf http://127.0.0.1:9090/healthz
docker compose --env-file .env logs -f snmp-collector
```

## Troubleshooting

| Symptom | Check |
|---------|--------|
| TLS verify failed | Wrong/missing `certs/ca.crt`; SAN must match host in `MQTT_BROKER` |
| Connection refused | Azure NSG / firewall; Mosquitto not listening on `0.0.0.0:8883` |
| SNMP timeouts | Source IP not in ACL; no route to loopbacks; wrong community |
