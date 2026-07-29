# Equate appliance architecture

Equate is a local, on-premises network monitoring appliance. The supported
production placement is a single Debian 12 VM suitable for a VxRail- or
VMware-type installation. The appliance is packaged as an offline OVA and
continues operating without Internet access after staging.

## Runtime flow

```text
SNMPv2c devices → per-site collectors → SQLite outboxes → local MQTT/TLS
→ Ingestion → PostgreSQL → Backend API → nginx → dashboard
```

PostgreSQL is the monitoring system of record. MQTT is a local delivery
boundary. Each collector's SQLite database is a durable outbox that lets
polling continue when the broker is temporarily unavailable.

## Service ownership

- The collector owns SNMP polling, vendor profiles, inventory overlays, local
  health/dependency evaluation, discovery, the outbox, and the local TUI.
- MQTT owns authenticated QoS 1 delivery between local containers.
- Ingestion owns event validation, deduplication, transactional persistence,
  and ACK decisions.
- PostgreSQL owns durable inventory, samples, health history, and collector
  status/history.
- The Backend API owns read-only REST projections.
- The dashboard owns presentation and never connects directly to services or
  storage.
- The appliance host owns local PAM authentication and privileged user
  operations through the permissioned broker socket.

## Configuration

First boot uses `collector setup` in the appliance TUI to create users, sites,
generated collector services, and per-installation secrets. Day-2 configuration
uses `collector tui` over a local Unix socket. Static YAML is protected and
read-only; the TUI writes only managed inventory and reloads validated
snapshots.

## Security boundary

Only TCP 80/443 is published. PostgreSQL, MQTT, administration endpoints,
metrics, and collector control sockets remain private. The appliance does not
provide a remote management plane. Secrets never appear in logs, telemetry,
metrics, audit files, release manifests, or API responses.
