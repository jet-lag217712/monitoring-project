# Backend API Architecture — v2

## Purpose

The Backend API is the read-only UI/UX Cloud Plane interface over PostgreSQL monitoring state. It translates persisted v2 telemetry and health evidence into stable dashboard contracts; it does not poll SNMP, consume MQTT, modify collector inventory, or write telemetry samples.

## Resources

Existing site, device, interface, metric, and alert endpoints remain the resource model. Their response adapters now expose:

- Current device status plus numeric compatibility (`0` unknown, `1` healthy, `2` warning, `3` critical), explicit `status_reason`, failure count, and dependency/root-cause fields.
- Site summaries that distinguish warning count, direct critical count, and dependency-impacted count rather than counting Unknown dependents as Critical.
- Device identity/profile/fingerprint, vendor/model/serial, SNMP identity, CPU/memory/temperature histories, primary temperature, and component power/temperature data.
- Interface identity/metadata, admin/operational state, counters, errors, speed, and traffic history.
- Collector operational status and latest heartbeat metadata where administrative views need it.

All timestamps are UTC ISO-8601. API contracts must carry data already normalized/persisted by ingestion and must never expose SNMP communities, TLS material, collector paths, environment values, or operator-control actions.

## Response contract examples

### Device summary

Existing fields remain available. V2 adds explicit health evidence:

```json
{
  "hostname": "access-01",
  "role": "Access Switch",
  "status": 0,
  "status_reason": "upstream_unreachable",
  "failure_count": 2,
  "upstream_device_ids": ["dist-01", "dist-02"],
  "unavailable_upstream_device_ids": ["dist-01", "dist-02"],
  "root_cause_device_ids": ["core-01"],
  "cpu_pct": 31.2,
  "memory_pct": 48.1,
  "uptime_days": 112.4,
  "latency_ms": null
}
```

Numeric status compatibility is fixed: `0` Unknown, `1` Healthy, `2`
Warning, and `3` Critical. Unknown is not converted to Critical or Offline by
the API.

### Site summary

```json
{
  "total_devices": 12,
  "healthy_count": 8,
  "warning_count": 2,
  "critical_count": 1,
  "unknown_count": 1,
  "dependency_impacted_count": 1,
  "online_count": 10,
  "active_alerts": 1
}
```

`critical_count` counts direct Critical root devices. Devices recorded as
Unknown due to dependency impact are counted separately.

### Device detail and component data

Device detail retains existing identity fields and adds profile, SNMP identity,
health evidence, component readings, and histories:

```json
{
  "vendor": "cisco",
  "model": "sanitized-model",
  "serial_number": "sanitized-serial",
  "profile": "cisco",
  "temperature_c": 52.5,
  "power_components": [
    {"component_id": "power-1", "name": "PSU 1", "status": "ok", "value": 1, "unit": "state"}
  ],
  "snmp": {
    "sysName": "access-01",
    "sysObjectID": "1.3.6.1.4.1.9.1.9999",
    "sysDescr": "Sanitized lab fixture",
    "sysUpTime": 1234500
  },
  "history": {
    "cpu": [{"ts": "2026-07-16T18:00:00Z", "value": 31.2}],
    "memory": [{"ts": "2026-07-16T18:00:00Z", "value": 48.1}],
    "temperature": [{"ts": "2026-07-16T18:00:00Z", "value": 52.5}],
    "uptime": [{"ts": "2026-07-16T18:00:00Z", "value": 9711360}]
  }
}
```

## React adapter contract

The adapter preserves the current UI shape while mapping v2 API fields:

| API field | Existing UI field | Rule |
|---|---|---|
| `cpu_utilization_pct` or current `cpu_pct` | `cpu_pct` | Preserve numeric percent; absent data remains empty/unknown, not zero telemetry. |
| `memory_utilization_pct` or current `memory_pct` | `memory_pct` | Preserve numeric percent. |
| `primary_temperature_c` | `temperature_c` | Use documented primary temperature only for the summary; preserve components separately. |
| `status` | `status` | Preserve `0/1/2/3` numeric compatibility. |
| `status_reason` | `status_reason` | Preserve reason for badges, details, and diagnostics. |
| upstream/root-cause arrays | matching snake_case fields | Preserve IDs without inferring topology in React. |
| component readings | `temperature_components`, `power_components` | Render each component and status without collapsing it into a fabricated value. |

Unknown receives its own visual treatment and explanatory reason. The adapter
must not derive Critical from missing data, CPU, memory, power, or dependency
fields. API compatibility tests must cover existing fields plus the new v2
fields.

## Aggregation semantics

The API derives frontend-friendly aggregates from persisted health state. `unknown` means `upstream_unreachable` only when supported by recorded dependency evidence; it is visually distinct and is not an outage claim. CPU, memory, and power readings are displayed telemetry in v2, not independent API health rules. Device temperature policy is reflected through the persisted collector health result, not recomputed by the API.

## Operations

The API remains stateless and uses a read-only database account. Authentication, consistent JSON errors, structured request logs, and direct-PostgreSQL prohibition for frontend clients remain unchanged.
