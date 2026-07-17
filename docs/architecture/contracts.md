# Service Contracts — SNMP Collector v2

## Status and ownership

This is the canonical v2 telemetry contract. It implements the agreed roadmap in [`.ai/roadmap/snmp-collector-v2.md`](../../.ai/roadmap/snmp-collector-v2.md). During migration, ingestion accepts the documented v1 routes while v2 producers and consumers are rolled out; new collection uses the v2 routes below.

MQTT/TLS is the delivery mechanism. PostgreSQL is the system of record. Every v2 message is QoS 1, durably queued in the collector's SQLite outbox, and may be delivered more than once.

## Routes

```text
site/{site_id}/device/{device_id}/telemetry/v2/device
site/{site_id}/device/{device_id}/telemetry/v2/interface
site/{site_id}/device/{device_id}/telemetry/v2/health
site/{site_id}/collector/{collector_id}/telemetry/v2/heartbeat
```

Ingestion validates the route shape and cross-checks every route identifier against the body. It rejects unknown schema versions, invalid IDs, malformed or stale timestamps, unknown metric units, invalid health transitions, and any route/body mismatch.

## Shared envelope

Every v2 payload contains these fields:

```json
{
  "schema_version": "2.0",
  "event_id": "018f...",
  "site_id": "site-001",
  "collector_id": "collector-west-01",
  "observed_at": "2026-07-16T18:00:00Z",
  "emitted_at": "2026-07-16T18:00:02Z",
  "config_revision": "sha256:..."
}
```

`event_id` is a stable event UUID generated before durable enqueue. `observed_at` is when the reading or state was observed; `emitted_at` is when the collector formed the message. `config_revision` is non-secret. Device and interface payloads also include `device_id`; heartbeat payloads include `collector_id` and do not use a device route.

## Device telemetry

The device payload contains normalized identity and scalar readings, plus component data rather than fabricated aggregate sensor values.

```json
{
  "schema_version": "2.0",
  "event_id": "018f...",
  "site_id": "site-001",
  "device_id": "dist-01",
  "collector_id": "collector-west-01",
  "observed_at": "2026-07-16T18:00:00Z",
  "emitted_at": "2026-07-16T18:00:02Z",
  "config_revision": "sha256:...",
  "identity": {
    "hostname": "dist-01",
    "sys_object_id": "1.3.6.1.4.1...",
    "sys_descr": "...",
    "vendor": "cisco",
    "model": "...",
    "serial": "...",
    "profile": "cisco",
    "capabilities": ["cpu", "memory", "temperature", "power"]
  },
  "readings": {
    "uptime_seconds": 12345,
    "cpu_utilization_pct": 34.2,
    "memory_utilization_pct": 61.0,
    "primary_temperature_c": 52.5
  },
  "temperature_components": [
    {"name": "inlet", "index": "1", "value": 52.5, "unit": "celsius", "status": "ok", "observed_at": "2026-07-16T18:00:00Z"}
  ],
  "power_components": [
    {"name": "PSU 1", "index": "1", "value": 1, "unit": "state", "status": "ok", "observed_at": "2026-07-16T18:00:00Z"}
  ]
}
```

Core SNMPv2-MIB and IF-MIB data are available for every supported device. Cisco and Arista profiles add capabilities only when their fingerprinted mapping succeeds. Unsupported devices use the core profile and omit unavailable readings; they never publish guessed zero values. Component source OIDs are available only to local operator diagnostics, never to cloud clients.

## Interface telemetry

One interface payload carries its identity, selected IF-MIB metadata, counters, errors, speed, and administrative/operational state.

```json
{
  "schema_version": "2.0",
  "event_id": "018f...",
  "site_id": "site-001",
  "device_id": "dist-01",
  "collector_id": "collector-west-01",
  "observed_at": "2026-07-16T18:00:00Z",
  "emitted_at": "2026-07-16T18:00:02Z",
  "config_revision": "sha256:...",
  "interface": {
    "if_index": 2,
    "name": "GigabitEthernet1/0/1",
    "alias": "AP-Uplink",
    "type": "ethernetCsmacd",
    "admin_status": "up",
    "oper_status": "up",
    "speed_bps": 1000000000
  },
  "counters": {
    "in_octets": 123,
    "out_octets": 456,
    "in_errors": 0,
    "out_errors": 0
  }
}
```

Only interfaces selected after configured filtering are emitted. The collector records selection and exclusion counts/reasons locally without high-cardinality metric labels.

## Health telemetry

Health is evaluated by the collector because it alone observes the polling path and locally managed dependency policy. The valid states are `healthy`, `warning`, `critical`, and `unknown`.

```json
{
  "schema_version": "2.0",
  "event_id": "018f...",
  "site_id": "site-001",
  "device_id": "access-01",
  "collector_id": "collector-west-01",
  "observed_at": "2026-07-16T18:00:00Z",
  "emitted_at": "2026-07-16T18:00:02Z",
  "config_revision": "sha256:...",
  "state": "unknown",
  "reason": "upstream_unreachable",
  "failure_count": 2,
  "temperature_warning_c": 65,
  "temperature_policy_revision": "policy:7",
  "upstream_device_ids": ["dist-01", "dist-02"],
  "unavailable_upstream_device_ids": ["dist-01", "dist-02"],
  "root_cause_device_ids": ["core-01"]
}
```

`warning` is a reachable device at or over its temperature threshold. A direct poll failure becomes `critical` only after the configured consecutive-failure threshold (default two). A failed dependent becomes `unknown` with `upstream_unreachable` only when every configured upstream is Critical or already upstream-unreachable. CPU, memory, and power do not change v2 health. Pending failures retain the prior terminal state and are recorded as evidence, not as a new state transition.

## Collector heartbeat

The collector emits a heartbeat on initial successful startup and at the configured interval (default 60 seconds). It uses the shared envelope and adds:

```json
{
  "collector_id": "collector-west-01",
  "hostname": "collector-host",
  "version": "v2.0.0",
  "git_commit": "abc123",
  "build_time": "2026-07-16T12:00:00Z",
  "uptime_seconds": 3600,
  "sqlite_queue_depth": 14,
  "memory_usage_bytes": 12345678,
  "goroutine_count": 42
}
```

Local builds use `unknown` for unavailable build metadata. A delayed heartbeat remains historical telemetry but cannot overwrite a current collector-status row with a newer `observed_at`. Heartbeats must not include arguments, paths, environment values, credentials, raw memory, or payload content.

## Delivery, idempotency, and migration

The collector enqueues before publication and removes an outbox item only after broker acknowledgment. Ingestion processes each message as `validate → deduplicate → upsert inventory/state and samples → commit → MQTT acknowledge`. Invalid or permanently unsupported messages are acknowledged and recorded as rejected; transaction failures are not acknowledged.

`event_id` provides v2 message deduplication. Natural keys continue to protect sample inserts: device metrics use device, metric type, and observation time; interface samples use interface and observation time. State/history writes are idempotent by event ID. Ingestion is deployed and verified before v2 collection is enabled; v1 routes remain supported through the documented migration window.

## REST compatibility

The API remains read-only for cloud clients. Numeric device status remains compatible: `0` unknown, `1` healthy, `2` warning, `3` critical. Responses that expose health must include `status_reason`, `upstream_device_ids`, `unavailable_upstream_device_ids`, and `root_cause_device_ids` when applicable.

No route or payload may contain SNMP community strings, TLS material, environment values, or other secrets.
