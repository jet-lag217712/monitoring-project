# Service Contracts — SNMP Collector v2

## Status and ownership

This document defines the v2 telemetry contract described by the [SNMP
Collector v2 roadmap](../../.ai/roadmap/snmp-collector-v2.md). The machine-
readable source is [`docs/schemas/snmp-collector-v2/`](../schemas/snmp-collector-v2/).
The Phase 0 ownership and transition decision is recorded in
[`collector-1.md`](../../.ai/decisions/collector-1.md).

The SNMP Collector owns polling-path evidence and local health evaluation.
Ingestion owns validation, deduplication, persistence, and MQTT ACK decisions.
PostgreSQL is the system of record. MQTT/TLS is only the authenticated,
outbound delivery mechanism.

## Routes and compatibility

**Production contract is v2 only** ([`collector-7.md`](../../.ai/decisions/collector-7.md)).
Deployment profiles publish and subscribe to versioned v2 routes:

```text
site/{site_id}/device/{device_id}/telemetry/v2/device
site/{site_id}/device/{device_id}/telemetry/v2/interface
site/{site_id}/device/{device_id}/telemetry/v2/health
site/{site_id}/collector/{collector_id}/telemetry/v2/heartbeat
```

Legacy v1 routes are deprecated and unsupported for deployment:

```text
site/{site_id}/device/{device_id}/metric/device
site/{site_id}/device/{device_id}/metric/interface
```

Ingestion and collector code may still accept or emit v1 when explicitly
configured (`publisher.telemetry_version: v1` or `both`) for emergency/lab
use only. Route identifiers remain authoritative: ingestion cross-checks them
against envelope identifiers and rejects mismatch, malformed IDs, unknown
schema versions, unsupported units, invalid transitions, stale timestamps, and
unknown event types.

## Formal event envelope

Every v2 event has this envelope:

```json
{
  "schema_version": "2.0",
  "event_id": "018f3e2c-7a9d-7b20-8f63-1e2d3c4b5a60",
  "event_type": "device_telemetry",
  "site_id": "site-001",
  "collector_id": "collector-west-01",
  "device_id": "dist-01",
  "observed_at": "2026-07-16T18:00:00Z",
  "emitted_at": "2026-07-16T18:00:02Z",
  "config_revision": "revision-2026-07-16-001",
  "payload": {}
}
```

`event_id` is generated before durable enqueue and is unique per observation.
`observed_at` controls ordering and current-state updates; `emitted_at` records
publication timing and cannot overwrite a newer observation. `config_revision`
is non-secret. Device, interface, and health events require `device_id`; a
heartbeat does not.

All event schemas use JSON Schema Draft 2020-12, closed event shapes, and the
literal `schema_version` `2.0`. Additive changes require compatibility review;
incompatible changes require a new schema version and rollout decision.

## Metric units

| Metric or component unit | Meaning |
|---|---|
| `seconds` | Device uptime in seconds. |
| `percent` | CPU or memory utilization from 0 through 100. |
| `celsius` | Temperature value in degrees Celsius. |
| `state` | A power component state value; status remains authoritative. |
| `watts`, `volts`, `amps` | Numeric power-supply readings in the named unit. |
| `octets` | Interface byte counters. |
| `count` | Interface errors, discards, or other integer counters. |

The collector omits unavailable vendor readings. It never emits fabricated zero
values to represent an unsupported OID.

## Device telemetry

The device event contains normalized identity, detected profile/capabilities,
core uptime, optional vendor scalar readings, and individual temperature/power
components. The complete shape is defined by
[`device-event.schema.json`](../schemas/snmp-collector-v2/device-event.schema.json).

```json
{
  "schema_version": "2.0",
  "event_id": "018f3e2c-7a9d-7b20-8f63-1e2d3c4b5a60",
  "event_type": "device_telemetry",
  "site_id": "site-001",
  "collector_id": "collector-west-01",
  "device_id": "dist-01",
  "observed_at": "2026-07-16T18:00:00Z",
  "emitted_at": "2026-07-16T18:00:02Z",
  "config_revision": "revision-2026-07-16-001",
  "payload": {
    "identity": {
      "hostname": "dist-01",
      "sys_object_id": "1.3.6.1.4.1.9.1.9999",
      "sys_name": "dist-01",
      "sys_descr": "Sanitized lab fixture",
      "vendor": "cisco",
      "model": "sanitized-model",
      "serial": "sanitized-serial",
      "snmp_version": "2c"
    },
    "profile": {
      "name": "cisco",
      "capabilities": ["cpu", "memory", "temperature", "power"]
    },
    "readings": {
      "uptime_seconds": 12345,
      "cpu_utilization_pct": 34.2,
      "memory_utilization_pct": 61,
      "primary_temperature_c": 52.5
    },
    "temperature_components": [],
    "power_components": []
  }
}
```

## Interface telemetry

An interface event carries one selected interface, its metadata, status, speed,
and counters. Filtering occurs before emission. The complete shape is defined
by [`interface-event.schema.json`](../schemas/snmp-collector-v2/interface-event.schema.json).

```json
{
  "schema_version": "2.0",
  "event_id": "018f3e2c-7a9d-7b20-8f63-1e2d3c4b5a61",
  "event_type": "interface_telemetry",
  "site_id": "site-001",
  "collector_id": "collector-west-01",
  "device_id": "dist-01",
  "observed_at": "2026-07-16T18:00:00Z",
  "emitted_at": "2026-07-16T18:00:02Z",
  "config_revision": "revision-2026-07-16-001",
  "payload": {
    "interface": {
      "if_index": 2,
      "name": "GigabitEthernet1/0/1",
      "alias": "Sanitized uplink",
      "type": "ethernetCsmacd",
      "admin_status": "up",
      "oper_status": "up",
      "speed_bps": 1000000000
    },
    "counters": {
      "in_octets": 123,
      "out_octets": 456,
      "in_errors": 0,
      "out_errors": 0,
      "in_discards": 0,
      "out_discards": 0
    }
  }
}
```

## Health state and reason taxonomy

| State | Meaning |
|---|---|
| `healthy` | Device responded and temperature is below its active threshold. |
| `warning` | Device responded and temperature is at or above its active threshold. |
| `critical` | Device is directly unreachable after the consecutive-failure threshold. |
| `unknown` | Every configured upstream path is unavailable, so dependent reachability is unknown. |

Public reason codes are:

- `reachable`
- `temperature_threshold`
- `direct_unreachable`
- `upstream_unreachable`
- `upstream_site_unreachable`
- `recovered`

Pending failures do not create a new terminal state. They retain the prior
state and are represented by failure-count evidence and operational metrics.

## Health transition model

```text
unobserved --success/below threshold--> healthy
unobserved --success/at threshold----> warning
healthy    --temperature threshold---> warning
warning    --temperature recovery----> healthy
any state  --direct failure threshold-> critical
any state  --all upstream paths fail--> unknown
critical   --successful poll----------> healthy or warning
unknown    --successful poll----------> healthy or warning
```

Every device is polled independently. A failed device with at least one
successfully polled configured upstream is direct failure evidence. A dependent
becomes `unknown` only when all configured upstream paths are unavailable.
Health evidence includes previous state, transition type, failure count,
threshold, temperature policy, upstream IDs, unavailable upstream IDs, and root
cause IDs. CPU, memory, and power never drive v2 health state.

The complete shape is defined by
[`health-event.schema.json`](../schemas/snmp-collector-v2/health-event.schema.json).

## Collector heartbeat

The collector publishes an initial successful-startup heartbeat and a periodic
heartbeat using the same envelope. The complete shape is defined by
[`heartbeat-event.schema.json`](../schemas/snmp-collector-v2/heartbeat-event.schema.json).

The payload includes `hostname`, `version`, `git_commit`, `build_time`,
`uptime_seconds`, `sqlite_queue_depth`, `memory_usage_bytes`, and
`goroutine_count`. Local builds use `unknown` for unavailable build metadata.
Delayed heartbeats remain historical telemetry but cannot overwrite a newer
collector-status observation.

Heartbeats must not contain process arguments, filesystem paths, environment
values, credentials, raw memory, or payload bodies.

## Ownership boundaries

| Component | Owns | Does not own |
|---|---|---|
| SNMP Collector | Polling, profile detection, normalization, local health, event creation, SQLite outbox | PostgreSQL persistence, public API, dashboard behavior |
| MQTT/TLS transport | Authentication, delivery, QoS 1, reconnect, transport buffering | Schema meaning, health decisions, persistence |
| Ingestion Service | Validation, topic/body checks, deduplication, database transactions, ACK decisions | SNMP polling, dashboard presentation |
| PostgreSQL | Authoritative inventory, samples, health, dependency, collector state/history | SNMP access or message delivery |
| Backend API | Read-only projections and compatibility responses | Polling or re-evaluating collector health |
| React dashboard | API adaptation and visual treatment | Direct collector/database access or health inference |
| Local TUI/control socket | Local operator status and approved control actions | Public management paths or secret exposure |
| Contract schemas | Versioned producer/consumer interface | Runtime ownership of event semantics |

## Delivery, idempotency, and migration

The collector durably enqueues before publication and removes an outbox row
only after broker success. Ingestion processes each event as:

```text
MQTT receive → validate → deduplicate → transaction → MQTT acknowledge
```

Invalid non-retryable messages are acknowledged and logged as rejected.
Duplicates are acknowledged without changing state. Database failures are not
acknowledged so QoS 1 redelivery occurs. V2 uses `event_id` deduplication while
retaining natural sample keys: device metrics use device, metric type, and
observation time; interface samples use interface and observation time.

Production deployments use v2-only publishing and ingestion topics
([`collector-7.md`](../../.ai/decisions/collector-7.md)). The earlier Phase 4
dual-publish migration window is closed because no production workload ever
depended on v1 routes. Emergency dual-publish remains a documented override,
not a deployment default. High-volume time-series and history are retained for
30 days by default via the ingestion retention job
([`database-1.md`](../../.ai/decisions/database-1.md)). Time-based indexes are
required; partitioning or archival may be added later without changing the
event contract.

## Security

No route or payload may contain SNMP communities, TLS material, environment
values, credentials, filesystem paths, process arguments, raw memory, or other
secrets.
