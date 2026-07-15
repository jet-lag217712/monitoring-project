# Development profile

Day-to-day lab workflow:

1. **Mac (cloud plane)** — Frontend, PostgreSQL, Ingestion, Mosquitto, Backend API via one Compose project
2. **OrbStack Ubuntu VM (collector plane)** — SNMP collector attached to GNS3 via a **Cloud** node (not a GNS3 Docker node)

```text
GNS3 devices ──SNMP──▶ collector (VM macvlan 10.254.254.2)
                              │ MQTT/TLS :8883
                              ▼
                     Mac Mosquitto → ingestion → Postgres → API → UI
```

## Cloud plane (Mac)

```bash
cp deployments/development/.env.example deployments/development/.env
# Set MQTT_SERVER_IP to the Mac IP visible from the OrbStack VM BEFORE first cert gen
./deployments/development/up.sh
./deployments/development/validate.sh
./deployments/development/smoke.sh
./deployments/development/down.sh   # -v wipes volumes
```

| Service | Host port (default) |
|---------|---------------------|
| Frontend | `:80` |
| Backend API | `:8000` / admin `:9092` |
| Ingestion | admin `:9091` |
| Mosquitto | `:8883` |
| PostgreSQL | `:5432` |

## Collector plane (OrbStack VM)

```bash
# On Mac — push source + package + public CA
./deployments/development/vxrail/sync.sh --dry-run
./deployments/development/vxrail/sync.sh

# On VM
cd /home/gns3/ogsd-vxrail   # or VXRAIL_REMOTE_DIR
cp -n .env.example .env
# Set MQTT_BROKER=tls://<mac-host-ip>:8883
sudo ./setup-gns3-bridge.sh
./bootstrap.sh
```

Configure SSH target in [`vxrail/.env`](vxrail/.env.example): `VXRAIL_SSH_HOST`, `VXRAIL_SSH_USER`, `VXRAIL_REMOTE_DIR`.

### GNS3 wiring

1. `setup-gns3-bridge.sh` creates IP-less bridge `br-gns3-vxrail`
2. GNS3 **Cloud** adapter binds to that bridge
3. Cloud → lab uplink (e.g. DO-CORE GigabitEthernet6/0)
4. Collector container sits on macvlan `10.254.254.2/30`

Do **not** use a GNS3 Docker node for the collector.

## Updating collector code

```bash
# After editing services/snmp-collector/
./deployments/development/vxrail/sync.sh
ssh gns3@192.168.x.x 'cd /home/gns3/ogsd-vxrail && ./bootstrap.sh'
```

`sync.sh` uses tar-over-SSH (works without `rsync` on the VM). Prefer an SSH key so you are not prompted for a password.
