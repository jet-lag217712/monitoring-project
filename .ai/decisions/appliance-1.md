## appliance - 1

### Primary Service

Deployments (On-Prem Appliance)

### Secondary Services

- Monitoring Dashboard
- Backend API
- Database
- Ingestion Service
- Secure MQTT Transport
- SNMP Collector

### Choice Made

Support a full on-premises appliance at `deployments/production/appliance/`.
The appliance runs the existing UI/nginx, Backend API, PostgreSQL, Mosquitto,
Ingestion, and generated per-site collector containers on one Debian 12 VM.
It preserves the proven Compose/MQTT pipeline and PostgreSQL system of record.

This profile is the only supported exception to the physical two-plane
placement in `awsdeploy-1`. The planes are colocated, but their service
ownership, container networks, secrets, collector control sockets, durable
outboxes, and PostgreSQL authority remain isolated. Only TCP 80/443 may be
published from the VM; PostgreSQL, MQTT, service administration, and collector
control are internal or loopback-only.

The appliance uses local PAM-backed multi-user accounts created and managed
through the appliance TUI. The dashboard and restricted VM login use the same
accounts without exposing `/etc/shadow` to a container or granting users Docker
access.

Release preparation is architecture-parameterized: validate an ARM64 OVA in
VMware Fusion first, then build and qualify the AMD64 VMware client release on
an x86 VMware-capable host. The release path is SCP staging, VM configuration,
OVA-safe finalization, manual export, and clean re-import. GitHub publication
is deferred.

### Relationship To Existing Decisions

- Does not supersede `awsdeploy-1`; hybrid production remains the default
  two-plane model.
- Applies only to `deployments/production/appliance/`.
- Does not revive the reverted ISO or `equate-core` in-process architecture.

### Alternatives Considered

- Use the hybrid on-site collector plus cloud application profile only
- Restore the reverted ISO and `equate-core` implementation
- Expose MQTT, PostgreSQL, or service administration ports on the appliance
- Treat ARM64 Fusion output as the AMD64 client release
- Publish artifacts to GitHub before OVA re-import acceptance

### Pros

- Runs without a cloud dependency after the release bundle is staged
- Reuses the current tested transport, ingestion, persistence, API, and UI path
- Gives operators one VM and one local account system to administer
- Keeps internal services and control surfaces off the customer network

### Cons

- Colocation creates one failure and maintenance domain
- The customer owns VM capacity, backups, patching, and physical network access
- ARM64 validation does not replace AMD64 VMware qualification
- Local authentication and manual OVA release handling add appliance-specific operations

### Cost / Benefit

The appliance adds a supported local deployment without creating a second
application architecture. The operational cost of a single-VM failure domain
and manual VMware release process is accepted in exchange for an offline,
customer-controlled installation that retains existing service contracts.

### Status

Accepted
