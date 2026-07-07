# Backend API

## Plane Ownership

UI/UX Cloud Plane.

## Responsibilities

- Expose REST contracts for frontend clients.
- Read monitoring state and history from PostgreSQL.
- Translate database records into API responses.
- Enforce application access controls.

## Non-Responsibilities

- Polling SNMP devices.
- Processing telemetry transport messages.
- Writing monitoring samples.
- Rendering frontend views.
- Configuring monitored devices.
- Providing device console or management access.

## Deployment Boundary

The API runs in the UI/UX Cloud Plane and is the only frontend-facing service for monitoring data.

Approved flow:

```text
PostgreSQL -> Backend API -> UI/UX Cloud Plane
```
