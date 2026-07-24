## appliance - 1

### Primary Service

Equate Appliance

### Secondary Services

- SNMP Collector
- Ingestion Service
- Backend API
- Monitoring Dashboard
- PostgreSQL

### Choice Made

The production appliance is a self-contained, single-node Debian 12 appliance
installed from a fully offline, UEFI-only ISO. Its
immutable release contains exactly five Compose services: UI, API, PostgreSQL,
`equate-core` (logical SNMP Poller and Ingestion), and migration. It does not
ship Mosquitto, broker certificates, MQTT credentials, or an MQTT management
surface.

`equate-core` uses a typed in-process Go event interface. Its SQLite/WAL spool
records an event before dispatch and retains it until the PostgreSQL ingestion
transaction acknowledges it. MQTT, NATS, and Redis Streams are optional future
transport adapters; they remain outside production appliance artifacts.

Appliance configuration and Equate configuration are separate. Appliance
Manager owns `/etc/equate/appliance.yaml`; Equate Core validates and owns
`/etc/equate/application.yaml` through local Unix-socket control paths.
Secrets are encrypted in `/etc/equate/secrets/*.age` and rendered only into
read-only runtime material under `/run/equate/rendered/`.

Every release is independent under `/opt/equate/releases/<version>/` and
`current` is the sole activation point. Packages from online channels, approved
mirrors, and virtual media use an identical signed `.eqa` verification and
staging flow.

### Alternatives Considered

- Keep the cloud/MQTT production topology for appliance deployments.
- Bundle a customer-managed broker into the appliance.
- Put mutable customer settings in the Compose release directory.
- Continue exporting a preinstalled VMware OVA.

### Pros

- Supports completely disconnected monitoring without a broker dependency.
- Retains durable at-least-once delivery and PostgreSQL deduplication behavior.
- Makes update and rollback a controlled symlink switch over immutable bundles.
- Keeps customer secrets and operational controls off public HTTP ports.
- Makes client installation independent of host-specific VMware build tooling.

### Cons

- The appliance is intentionally single-node and is not a distributed worker
  topology.
- External scale-out requires a separately designed transport adapter and
  deployment mode.
- Installation deliberately erases the first non-removable virtual disk after
  an explicit boot-menu selection.

### Supersedes

For appliance production only, this decision supersedes the cloud-plane/MQTT
assumptions in `awsdeploy-*`, `collector-1`, and the cloud deployment context.
Those records remain applicable to the historical and development cloud
profiles. Collector safety guarantees are unchanged: SNMPv2c only, bounded
polling, managed inventory, explicit discovery, rate limits, review before
enrollment, and PostgreSQL as the system of record.

### Status

Accepted
