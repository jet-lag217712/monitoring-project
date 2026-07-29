# PostgreSQL Architecture — v2

PostgreSQL stores authoritative monitoring state and history inside the local appliance. Collectors and frontend clients never connect directly.

V2 requires migrations for enriched device/profile identity, interface metadata, component readings, health current state/history and dependency evidence, collector current status/history, plus seeded CPU, memory, temperature, and supported power metric types. All real-world timestamps use `TIMESTAMPTZ`; history tables remain append-oriented and current-state rows are updated only by valid, newer observations where ordering matters.

Ingestion performs `validate → deduplicate → transactional upsert/sample/history write → commit → MQTT ACK`. v2 uses `event_id` as the message idempotency key in addition to existing natural keys: `metric_samples` `(device_id, metric_type_id, collected_at)` and `interface_samples` `(interface_id, collected_at)`. Health and heartbeat histories must likewise reject duplicate event IDs.

Roles remain intentionally narrow: `ogsd_admin` applies migrations, `ogsd_ingestion` writes monitoring inventory/state/history and reads reference data, and `ogsd_api` is SELECT-only. New tables and indexes are introduced through [`database/migrations/`](../../database/migrations/), using additive rollout and production-like validation before collector v2 enablement.
