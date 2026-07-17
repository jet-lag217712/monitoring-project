# Ingestion Service Architecture — v2

## Purpose and ownership

The Ingestion Service consumes authenticated MQTT/TLS telemetry and owns the transactional persistence of API-facing monitoring state in PostgreSQL. It never polls SNMP, evaluates local reachability, exposes collector administration, or serves dashboard requests.

## V2 contract handling

The service consumes the versioned device, interface, health, and collector-heartbeat routes defined in [`contracts.md`](contracts.md). The machine-readable definitions are in [`docs/schemas/snmp-collector-v2/`](../schemas/snmp-collector-v2/). It cross-checks topic/body identity, accepts only recognized schema versions and units, validates timestamps and state transitions, and rejects malformed or unsupported messages without crashing. v1 routes are retained during the explicit migration window; v2 is enabled only after its migrations and handler compatibility tests are deployed.

The durable processing pipeline is:

```text
MQTT receive → validate → deduplicate → transaction → MQTT acknowledge
```

For a new event the single transaction upserts required inventory/current state, writes history and samples, and commits before acknowledgment. Invalid or non-retryable unsupported events are acknowledged and logged as rejected. Duplicates are acknowledged without changing state. Database failure is not acknowledged, so QoS 1 redelivery occurs. `event_id` is the primary v2 deduplication key; natural sample uniqueness remains an additional safeguard.

## Persistence model

Ingestion persists enriched device identity/profile/fingerprint and interface metadata; normalized device and interface samples; individual temperature and power components; health current state and history; topology evidence; current collector operational state; and heartbeat history. It must preserve the actual collector observation time.

Health state uses `healthy`, `warning`, `critical`, and `unknown`. State/history records retain reason, failure count, temperature threshold and policy revision when relevant, configured/unavailable upstream IDs, and root causes. The collector supplies this local evidence; ingestion persists and exposes it rather than independently guessing a cascade.

The accepted public reasons are `reachable`, `temperature_threshold`,
`direct_unreachable`, `upstream_unreachable`, and `recovered`. A pending poll
failure below threshold is not a terminal state transition. The collector's
previous state remains authoritative while the pending count is retained as
evidence. A dependent is stored as `unknown` only when all configured upstream
paths are unavailable; a responding upstream makes the dependent's failure
direct evidence after the configured threshold.

Current collector status is updated only when the incoming heartbeat's `observed_at` is newer than the stored observation time. Delayed outbox delivery therefore cannot regress a collector’s apparent status.

## Ownership boundary

Ingestion owns schema validation, route/body checks, event-ID and natural-key
deduplication, transaction boundaries, database writes, and ACK decisions. It
does not poll devices, infer topology, recalculate health, expose collector
controls, or render frontend responses. The collector owns local polling-path
evidence; PostgreSQL owns durable state; the Backend API owns read projections.

## IDs, security, and observability

String site/device identifiers are deterministically mapped to UUID v5 entity keys. Interfaces remain unique by `(device_id, if_index)`. Ingestion credentials are restricted to its database write responsibilities and telemetry connections are TLS-authenticated.

Structured logs include processing result, route/event identity, and safe site/device/collector IDs; they never include credentials, certificates, raw payloads, or secrets. Prometheus metrics retain receive/accept/reject/dedup/database/processing/MQTT coverage and distinguish v2 validation or handler failures by bounded reason.

## Compatibility test plan

Before v2 producers are enabled, compatibility tests must cover schema-version
and event-type rejection, topic/body identity mismatch, unit and timestamp
validation, duplicate event delivery, stale heartbeat ordering, invalid health
transitions, transaction-before-ACK behavior, and mixed v1/v2 consumption.
