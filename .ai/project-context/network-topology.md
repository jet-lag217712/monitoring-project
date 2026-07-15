# Network Topology

## Purpose

Defines network boundaries and communication paths.

## Customer OOB Monitoring Plane

The customer plane runs inside the district environment.

Components:
- Monitored network devices
- SNMP Collector

Responsibilities:
- Collector reaches monitored devices over SNMP.
- Collector buffers telemetry locally during connectivity interruptions.
- Collector initiates outbound-only secure telemetry connections.

The customer plane does not host:
- Ingestion Service
- PostgreSQL
- Backend API
- UI/UX Cloud Plane frontend

## UI/UX Cloud Plane

The cloud plane hosts the user-facing and stateful services.

Components:
- Secure Outbound Telemetry Transport endpoint
- Cloud Ingestion
- PostgreSQL
- Backend API
- Frontend

Responsibilities:
- Receive telemetry from outbound collector connections.
- Validate and persist telemetry.
- Store monitoring state in PostgreSQL.
- Serve UI/API workflows.

## Communication

SNMP:
- SNMP Collector communicates directly with monitored devices inside the Customer OOB Monitoring Plane.

Telemetry:
- SNMP Collector initiates outbound-only secure telemetry transport to the UI/UX Cloud Plane.
- No inbound access into the customer network is required from cloud services.

Application:
- Frontend clients communicate with the Backend API.
- Backend API reads from PostgreSQL.
- Cloud Ingestion writes to PostgreSQL.

## Security Boundary

Raw device access remains inside the Customer OOB Monitoring Plane.

Monitoring state and historical telemetry are stored in PostgreSQL in the UI/UX Cloud Plane.

## Deployment topology (profiles)

| Profile | Customer / collector plane | Cloud plane |
|---------|----------------------------|-------------|
| `deployments/end-to-end/` | Collector in the same Compose project as cloud services | Same host |
| `deployments/development/` | OrbStack Ubuntu VM + GNS3 Cloud (macvlan); sync via `vxrail/sync.sh` | Mac Docker Compose |
| `deployments/production/` | On-site VxRail Ubuntu VM | Azure VM Compose |

Collectors always initiate **outbound-only** MQTT/TLS to the cloud Mosquitto endpoint. Do not place the collector on a GNS3 Docker node; use a GNS3 Cloud adapter attached to the VM bridge.
