# Local service boundaries

### SNMP Collector

- Owns SNMPv2c polling, vendor profile detection, interface selection, local
  health/dependency evaluation, inventory reload, discovery, SQLite outbox,
  and heartbeat creation.
- Exposes local Unix-socket/localhost operator tooling only.
- Does not own PostgreSQL, the API, dashboard workflows, or device writes.

### Local MQTT/TLS transport

- Delivers authenticated QoS 1 events from collectors to local ingestion.
- Does not own monitoring state, health decisions, or durable history.

### Ingestion Service

- Validates v2 events, deduplicates, persists monitoring data, and decides ACKs.
- Does not poll devices, serve frontend requests, or configure collectors.

### PostgreSQL

- Owns authoritative inventory, samples, health evidence, alerts, and collector
  status/history for the appliance.
- Is reachable only by the services that require it.

### Backend API

- Owns read-only REST contracts and adapters for the dashboard.
- Uses a read-only database account and never exposes secrets or control paths.

### Dashboard

- Reads only through the API and displays persisted evidence.
- Never accesses MQTT, PostgreSQL, SNMP, collector inventory, or TUI sockets.

### Local TUI and appliance broker

- The collector TUI performs approved local configuration over a Unix socket.
- The host PAM broker performs privileged local account operations.
- Neither surface is published as remote HTTP management.
