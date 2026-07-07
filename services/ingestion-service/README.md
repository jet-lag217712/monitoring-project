# Ingestion Service

## Plane Ownership

UI/UX Cloud Plane.

## Responsibilities

- Consume telemetry from Secure Outbound Telemetry Transport.
- Validate telemetry payloads.
- Normalize telemetry into platform records.
- Write monitoring state and history to PostgreSQL.
- Reject malformed or unauthorized messages.

## Non-Responsibilities

- Polling SNMP devices.
- Hosting telemetry transport.
- Serving frontend requests.
- Rendering dashboard views.
- Configuring monitored devices.
- Providing device console or management access.

## Deployment Boundary

The service runs in the UI/UX Cloud Plane with write access to PostgreSQL.

MQTT/TLS is the current transport implementation; transport is delivery only and is not the system of record.

Approved flow:

```text
Secure Outbound Telemetry Transport -> Cloud Ingestion -> PostgreSQL
```
