# Database Architecture — v2

## Purpose

PostgreSQL is the local appliance system of record for inventory, telemetry, evaluated health evidence, collector operational state, and history. Only the Ingestion Service writes monitoring data; the Backend API exposes read-only contracts to frontend clients.

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

## Migration sequence

Phase 0 defines the migration order; SQL migrations are deferred to the v2
ingestion phase:

1. Seed v2 metric names and recognized units.
2. Add device identity, fingerprint, profile, capability, and interface metadata
   fields without removing v1 columns.
3. Add temperature/power component inventory and reading tables.
4. Add health current-state/history tables with reason, transition, threshold,
   policy, topology, and root-cause evidence.
5. Add collector current status and heartbeat history.
6. Add event-ID deduplication and observation-time indexes.
7. Backfill compatible current state, deploy ingestion support, then enable v2
   producers.

Each migration must be additive and reversible where practical. Existing v1
natural-key constraints remain in place throughout the compatibility window.

## Retention and indexing

Initial v2 retention is append-only: no automated deletion or archival job is
introduced until a separate measured-capacity decision is approved. All
time-series, component-reading, health-history, and heartbeat-history tables
must index their owning entity and observation time, for example
`(device_id, observed_at DESC)` and `(collector_id, observed_at DESC)`. Current
state tables require unique entity keys; event history requires a unique event
ID for v2 deduplication.

Partitioning or archival may be added later without changing the event contract.
The API needs efficient current site/device summaries, device/interface detail,
metric history, dependency impact, and collector status reads. Database roles
remain separated: administration/migrations, ingestion writes, and API
read-only access.

## Ownership

PostgreSQL owns authoritative inventory, telemetry, health, dependency, and
collector state/history. It does not own SNMP access, MQTT delivery, or local
collector buffering. The Ingestion Service is the only monitoring-data writer;
the Backend API uses read-only access.
