# Ingestion Service Architecture — v2

## Purpose and ownership

The Ingestion Service consumes authenticated MQTT/TLS telemetry and owns the transactional persistence of API-facing monitoring state in PostgreSQL. It never polls SNMP, evaluates local reachability, exposes collector administration, or serves dashboard requests.

## V2 contract handling

The service consumes the versioned device, interface, health, and collector-heartbeat routes defined in [`contracts.md`](contracts.md). It cross-checks topic/body identity, accepts only recognized schema versions and units, validates timestamps and state transitions, and rejects malformed or unsupported messages without crashing. v1 routes are retained during the explicit migration window; v2 is enabled only after its migrations and handler compatibility tests are deployed.

The durable processing pipeline is:

```text
MQTT receive → validate → deduplicate → transaction → MQTT acknowledge
```

For a new event the single transaction upserts required inventory/current state, writes history and samples, and commits before acknowledgment. Invalid or non-retryable unsupported events are acknowledged and logged as rejected. Duplicates are acknowledged without changing state. Database failure is not acknowledged, so QoS 1 redelivery occurs. `event_id` is the primary v2 deduplication key; natural sample uniqueness remains an additional safeguard.

## Persistence model

Ingestion persists enriched device identity/profile/fingerprint and interface metadata; normalized device and interface samples; individual temperature and power components; health current state and history; topology evidence; current collector operational state; and heartbeat history. It must preserve the actual collector observation time.

Health state uses `healthy`, `warning`, `critical`, and `unknown`. State/history records retain reason, failure count, temperature threshold and policy revision when relevant, configured/unavailable upstream IDs, and root causes. The collector supplies this local evidence; ingestion persists and exposes it rather than independently guessing a cascade.

Current collector status is updated only when the incoming heartbeat's `observed_at` is newer than the stored observation time. Delayed outbox delivery therefore cannot regress a collector’s apparent status.

## IDs, security, and observability

String site/device identifiers are deterministically mapped to UUID v5 entity keys. Interfaces remain unique by `(device_id, if_index)`. Ingestion credentials are restricted to its database write responsibilities and telemetry connections are TLS-authenticated.

Structured logs include processing result, route/event identity, and safe site/device/collector IDs; they never include credentials, certificates, raw payloads, or secrets. Prometheus metrics retain receive/accept/reject/dedup/database/processing/MQTT coverage and distinguish v2 validation or handler failures by bounded reason.
