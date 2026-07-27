# Network Topology — SNMP Collector v2

## Customer OOB Monitoring Plane

The customer plane contains SNMPv2c devices and the collector with static/managed inventory, SQLite outbox, and a local Bubble Tea TUI/status-control socket. The collector independently reaches each configured device, then initiates outbound-only MQTT/TLS to the cloud. No cloud service, public HTTP endpoint, or dashboard client can manage devices, inventory, or credentials in this plane.

The configured `upstream_device_ids` graph is a local dependency DAG used to correlate reachability. It is not a complete physical topology, does not suppress child polling, and never creates a cloud-to-customer path.

## UI/UX Cloud Plane

MQTT/TLS transport, Ingestion, PostgreSQL, Backend API, and frontend run in the cloud plane. Ingestion persists telemetry and health evidence; PostgreSQL is the system of record; the frontend accesses it only through the API.

## Deployment profiles

| Profile | Customer / collector plane | Cloud / application plane |
|---|---|---|
| `deployments/end-to-end/` | Collector in the same Compose project | Same host |
| `deployments/development/` | OrbStack VM plus GNS3 Cloud | Mac Docker Compose |
| `deployments/production/` | On-site VxRail VM | Azure VM Compose |
| `deployments/production/appliance/` | Per-site collector containers on the appliance VM | Same VM (UI/nginx, API, PostgreSQL, Mosquitto, Ingestion) |

In hybrid profiles collectors always initiate MQTT/TLS to the remote Mosquitto
endpoint. In the appliance profile collectors publish to local Mosquitto on
private container networks; only nginx publishes TCP 80/443 from the VM. Do not
place a hybrid collector on a GNS3 Docker node; use a GNS3 Cloud adapter attached
to the VM bridge.

Appliance network and release details: [`appliance.md`](appliance.md),
[`docs/releases/appliance-ova.md`](../../docs/releases/appliance-ova.md).
