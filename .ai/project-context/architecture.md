# Architecture — SNMP Collector v2

## Purpose

Equate OGSD is a two-plane network telemetry and monitoring platform.
Customer-side collectors observe network devices locally; the UI/UX Plane owns
durable monitoring state, API contracts, and visualization. The default
production placement separates the customer and cloud planes. The full
on-premises appliance is the only supported placement exception: it colocates
both planes on one Debian 12 VM while retaining their logical boundaries. The
agreed v2 design is defined in
[`.ai/roadmap/snmp-collector-v2.md`](../roadmap/snmp-collector-v2.md), and the
appliance exception is defined in [`appliance.md`](appliance.md). OVA build and
verification runbooks live in
[`docs/releases/appliance-ova.md`](../../docs/releases/appliance-ova.md).

## Data flow

```text
SNMPv2c devices → SNMP Collector v2 → SQLite outbox → MQTT/TLS → Ingestion → PostgreSQL → Backend API → Dashboard
```

The collector produces versioned device, interface, health, and heartbeat telemetry. MQTT is a delivery boundary, not a system of record; PostgreSQL is authoritative.

## Customer OOB Monitoring Plane

The customer plane contains monitored devices plus collector-local operational state: read-only static and TUI-managed inventory, the durable SQLite outbox, and a local Unix-socket status/control service used by the Bubble Tea TUI. Static inventory is authoritative for duplicate device IDs; unique managed entries are appended, and duplicate active host/IP identities are rejected. The collector polls every configured device independently through a bounded SNMPv2c worker pool, uses core SNMPv2-MIB/IF-MIB plus detected Cisco/Arista profiles, filters interfaces, and evaluates local temperature/dependency health evidence.

It has no inbound cloud management path. `/metrics` is scrape-only; `/healthz`
is liveness; `/readyz` reports active configuration, buffer, and publisher
readiness. A collector does not own or directly expose PostgreSQL, the Backend
API, Ingestion, or user-facing workflows. In the appliance profile those
services are separate containers on the same VM and communicate only through
the defined internal service boundaries.

## UI/UX Plane

The UI/UX Plane contains MQTT transport, Ingestion, PostgreSQL, the Backend API,
and dashboard. Ingestion validates versioned routes/payloads, deduplicates, and
transactionally persists inventory, samples, component readings, health
evidence/history, and collector status/history. PostgreSQL supplies monitoring
state to the read-only API; the dashboard consumes the API only.

In hybrid profiles this plane is cloud-hosted and collector transport crosses
the site boundary with TLS authentication. In
`deployments/production/appliance/` the same flow is local to private container
networks. Only nginx publishes TCP 80/443; PostgreSQL, MQTT, administration
ports, metrics, and collector control sockets remain internal or loopback-only.
Local PAM-backed users authenticate to the restricted VM appliance interface
and dashboard as defined in [`appliance.md`](appliance.md). Post-install and
re-import checks are automated by
[`appliance/scripts/verify-appliance.sh`](../../appliance/scripts/verify-appliance.sh)
and [`appliance/scripts/verify-ova-import.sh`](../../appliance/scripts/verify-ova-import.sh).

## Principles and non-goals

- Customer-network access and reachability evidence remain local; the cloud persists rather than guesses them.
- In hybrid profiles, every collector-to-cloud connection is outbound-only,
  TLS-authenticated, QoS 1, and buffered locally. The appliance retains QoS 1
  and durable buffering across its internal MQTT boundary.
- SNMP communities, TLS material, environment values, raw payloads, and local operator controls never reach cloud clients.
- v2 supports SNMPv2c read-only collection only. It excludes SNMPv3, configuration/device-console access, automatic enrollment, automatic dependency activation, and cloud-side credential editing.
- Dependency edges are a local reachability DAG, not a complete physical-network graph.
- Physical colocation is allowed only for the production appliance profile; it
  does not relax service ownership, reviewed discovery, secret handling,
  container isolation, or PostgreSQL authority, and it does not revive the
  reverted ISO/`equate-core` architecture.
