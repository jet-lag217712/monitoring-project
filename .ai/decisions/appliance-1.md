## appliance - 1

### Primary Service

Deployments (On-Prem Appliance)

### Secondary Services

- Monitoring Dashboard
- Backend API
- Database
- Ingestion Service
- Local MQTT Transport
- SNMP Collector

### Choice Made

Support a full local appliance at `deployments/production/appliance/`. The
appliance runs nginx, Backend API, PostgreSQL, Mosquitto, Ingestion, and
generated per-site collector containers on one Debian 12 VM. It preserves the
existing Compose/MQTT pipeline and PostgreSQL system of record.

First boot creates local PAM-backed users, per-installation secrets, sites, and
generated collector services through the setup TUI. Day-2 changes use the
collector TUI over an owner-only Unix socket. Only TCP 80/443 is published.

Release preparation is architecture-parameterized: validate an ARM64 OVA in
VMware Fusion first, then qualify the AMD64 release on an x86 VMware-capable
host. The release path is offline bundle staging, VM configuration, OVA-safe
finalization, manual export, and clean re-import.

### Alternatives Considered

- A collector-only installation with application services elsewhere.
- Restoring the reverted ISO and in-process core implementation.
- Exposing MQTT, PostgreSQL, or service administration ports.
- Treating ARM64 validation as the AMD64 release.

### Pros

- Works after staging without Internet access.
- Reuses the tested transport, ingestion, persistence, API, and UI path.
- Gives operators one VM and one local account system to administer.
- Keeps internal services and control surfaces private.

### Cons

- Colocation creates one failure and maintenance domain.
- The customer owns VM capacity, backups, patching, and physical access.
- ARM64 validation does not replace AMD64 qualification.
- Local authentication and OVA handling add appliance operations.

### Status

Accepted
