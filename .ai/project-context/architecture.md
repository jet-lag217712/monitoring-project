# Architecture — SNMP Collector v2

## Purpose

Equate OGSD is a two-plane network telemetry and monitoring platform. Customer-side collectors observe network devices locally; the UI/UX Cloud Plane owns durable monitoring state, API contracts, and visualization. The agreed v2 design is defined in [`.ai/roadmap/snmp-collector-v2.md`](../roadmap/snmp-collector-v2.md).

## Data flow

```text
SNMPv2c devices → SNMP Collector v2 → SQLite outbox → MQTT/TLS → Ingestion → PostgreSQL → Backend API → Dashboard
```

The collector produces versioned device, interface, health, and heartbeat telemetry. MQTT is a delivery boundary, not a system of record; PostgreSQL is authoritative.

## Customer OOB Monitoring Plane

The customer plane contains monitored devices plus collector-local operational state: read-only static and TUI-managed inventory, the durable SQLite outbox, and a local Unix-socket status/control service used by the Bubble Tea TUI. Static inventory is authoritative for duplicate device IDs; unique managed entries are appended, and duplicate active host/IP identities are rejected. The collector polls every configured device independently through a bounded SNMPv2c worker pool, uses core SNMPv2-MIB/IF-MIB plus detected Cisco/Arista profiles, filters interfaces, and evaluates local temperature/dependency health evidence.

It has no inbound cloud management path. `/metrics` is scrape-only; `/healthz` is liveness; `/readyz` reports active configuration, buffer, and publisher readiness. The collector does not host PostgreSQL, the Backend API, cloud ingestion, or user-facing cloud workflows.

## UI/UX Cloud Plane

The cloud plane contains MQTT/TLS transport, Ingestion, PostgreSQL, the Backend API, and dashboard. Ingestion validates versioned routes/payloads, deduplicates, and transactionally persists inventory, samples, component readings, health evidence/history, and collector status/history. PostgreSQL supplies monitoring state to the read-only API; the dashboard consumes the API only.

## Principles and non-goals

- Customer-network access and reachability evidence remain local; the cloud persists rather than guesses them.
- Every collector-to-cloud connection is outbound-only, TLS-authenticated, QoS 1, and buffered locally.
- SNMP communities, TLS material, environment values, raw payloads, and local operator controls never reach cloud clients.
- v2 supports SNMPv2c read-only collection only. It excludes SNMPv3, configuration/device-console access, automatic enrollment, automatic dependency activation, and cloud-side credential editing.
- Dependency edges are a local reachability DAG, not a complete physical-network graph.
