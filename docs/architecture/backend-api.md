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

## Aggregation semantics

The API derives frontend-friendly aggregates from persisted health state. `unknown` means `upstream_unreachable` only when supported by recorded dependency evidence; it is visually distinct and is not an outage claim. CPU, memory, and power readings are displayed telemetry in v2, not independent API health rules. Device temperature policy is reflected through the persisted collector health result, not recomputed by the API.

## Operations

The API remains stateless and uses a read-only database account. Authentication, consistent JSON errors, structured request logs, and direct-PostgreSQL prohibition for frontend clients remain unchanged.
