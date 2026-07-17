## collector - 5

### Primary Service

SNMP Collector

### Secondary Services

- Ingestion Service
- Database
- Secure Outbound Telemetry Transport

### Choice Made

Phase 4 dual-publish is controlled by `publisher.telemetry_version` with
allowed values `v1`, `v2`, and `both`. The original default was `both` so
existing v1 consumers remained fed while v2 routes were validated.
**Superseded by [`collector-7.md`](collector-7.md):** production default is
`v2`; `v1`/`both` are emergency/lab overrides only.

Envelope `config_revision` is a non-secret SHA-256 fingerprint of the active
configuration snapshot (site, collector identity, telemetry mode, health
defaults, and inventory identity/topology/threshold/poll overrides). Health
payload `temperature_policy_revision` remains the narrower
`TemperaturePolicyRevision` fingerprint from Phase 3.

Producer and consumer validation use hand-written Go contract checks that
mirror the Draft 2020-12 schemas under `docs/schemas/snmp-collector-v2/`. No
JSON Schema runtime dependency is introduced.

Optional device identity fields (`vendor`, `model`, `serial`) and optional
interface discards are omitted when not collected; values are never fabricated.
Temperature components without a numeric value are omitted because the device
schema requires a value. Power components may carry a null value.

Health MQTT publishing emits only the local `Tracker.ApplyBatch` transition set
(`initial`, `entered`, `recovered`). `reasserted` remains unused.

Heartbeat publishing runs on successful startup and on
`collector.heartbeat_interval`. SQLite outbox depth is sampled before the
heartbeat is enqueued so the heartbeat does not count itself. Build metadata
uses ldflags with an explicit `unknown` fallback. Current collector status is
updated by ingestion only when the incoming `observed_at` is newer than the
stored observation.

Additive migrations `000005`–`000010` seed v2 metrics, enrich inventory,
persist components/health/collector state, and add `ingested_events` for
`event_id` deduplication before v2 producers are enabled.

### Status

Accepted — Phase 4 implementation decision.
