# SNMP Collector v2 JSON Schemas

These Draft 2020-12 schemas are the machine-readable contract for v2 MQTT
events. They are documentation and compatibility artifacts in Phase 0; no
runtime producer or consumer uses them yet.

## Event schemas

| Schema | Event type | Route |
|---|---|---|
| `device-event.schema.json` | `device_telemetry` | `site/{site_id}/device/{device_id}/telemetry/v2/device` |
| `interface-event.schema.json` | `interface_telemetry` | `site/{site_id}/device/{device_id}/telemetry/v2/interface` |
| `health-event.schema.json` | `health_state` | `site/{site_id}/device/{device_id}/telemetry/v2/health` |
| `heartbeat-event.schema.json` | `collector_heartbeat` | `site/{site_id}/collector/{collector_id}/telemetry/v2/heartbeat` |

All event schemas compose `event-envelope.schema.json`. The envelope carries
identity, event UUID, observation/publication timestamps, and a non-secret
configuration revision. Topic IDs are authoritative and ingestion must
cross-check them against the envelope.

## Compatibility rules

- `schema_version` is currently the literal string `2.0`.
- Additive fields require a compatibility review; incompatible changes require
  a new schema version and route policy.
- Unknown schema versions, event types, units, enum values, and extra fields are
  rejected by v2 consumers.
- Examples contain only sanitized values and must remain safe to publish in
  documentation or tests.
- Schemas must never permit SNMP communities, TLS material, environment values,
  process arguments, filesystem paths, raw memory, or payload bodies outside the
  defined event data.

## Health model

Public states are `healthy`, `warning`, `critical`, and `unknown`. Public reason
codes are `reachable`, `temperature_threshold`, `direct_unreachable`,
`upstream_unreachable`, and `recovered`. A pending failure does not create a
terminal transition; its count is evidence attached to the prior state.

See [`collector-1.md`](../../../.ai/decisions/collector-1.md) and the service
contract documentation for the transition and ownership model.
