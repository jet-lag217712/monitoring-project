# Database Architecture — v2

## Purpose

PostgreSQL is the UI/UX Cloud Plane system of record for inventory, telemetry, evaluated health evidence, collector operational state, and history. Only the Ingestion Service writes monitoring data; the Backend API exposes read-only contracts to frontend clients.

## V2 data model

The core hierarchy remains Site → Device → Interface. v2 enriches it with:

- Device identity, vendor/model/serial, SNMP fingerprint, detected profile, and capabilities.
- Device and interface time-series samples, including uptime, CPU, memory, primary temperature, traffic, errors, counters, and status.
- Temperature and power component inventory/readings, preserving component name/index/unit/status instead of a fabricated single value.
- Current health and health history, including state, reason, observation time, failure count, temperature policy evidence, configured/unavailable upstream IDs, and root-cause IDs.
- Current collector status and heartbeat history, including configured identity, build metadata, runtime values, and SQLite queue depth.

Metric types include CPU, memory, temperature, and supported power readings with recognized units. Device health and collector status are current-state projections backed by history, not MQTT or collector-local state.

## Consistency and ordering

Ingestion writes each accepted event in a transaction, deduplicates v2 messages by event ID, and preserves natural sample uniqueness. Observation timestamps, not arrival order, govern current collector status. Valid state transition and topic/body validation occur before writes. Schema rollout is additive: deploy migrations and ingestion handling first, then enable v2 collectors while v1 consumption remains compatible for the migration period.

## Retention and access

Time-series, component-reading, health-history, and heartbeat-history tables require time-range indexes and a future retention/partitioning policy based on measured volume. The API needs efficient current site/device summaries, device/interface detail, metric history, dependency impact, and collector status reads. Database roles remain separated: administration/migrations, ingestion writes, and API read-only access.
