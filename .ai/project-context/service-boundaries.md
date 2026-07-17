# Service Boundaries — SNMP Collector v2

### SNMP Collector

* Customer OOB Monitoring Plane.
* Owns SNMPv2c polling, profile detection/collection, interface selection, local health/dependency evaluation, local inventory/reload, SQLite outbox, and heartbeat creation.
* Exposes local Unix-socket/localhost operator tooling only. Does not own cloud persistence, public APIs, PostgreSQL, or dashboard workflows.

### Secure Outbound Telemetry Transport

* Boundary between planes.
* Delivers authenticated MQTT/TLS QoS 1 events; does not own monitoring state or storage.

### Ingestion Service

* UI/UX Cloud Plane.
* Owns v1/v2 contract handling during migration, validation, deduplication, and transactional monitoring writes.
* Persists collector-supplied health evidence, samples/components, and observation-time-ordered collector status. Does not poll SNMP or serve frontend requests.

### Backend API

* UI/UX Cloud Plane.
* Owns read-only REST contracts and adapters for health, dependency impact, telemetry, components, and collector status. Does not process transport or write monitoring samples.

### Dashboard

* UI/UX Cloud Plane.
* Owns presentation and reads only through the API. It displays the persisted health evidence without independently applying collector rules, and never accesses transport, PostgreSQL, or collectors directly.
