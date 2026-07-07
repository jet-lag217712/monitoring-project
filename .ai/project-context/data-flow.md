# Data Flow Architecture

## Purpose

Defines the complete telemetry lifecycle from monitored network devices to the UI/UX Cloud Plane.

## Approved Telemetry Pipeline

```text
SNMP Devices
    ↓
SNMP Collector
    ↓
Secure Outbound Telemetry Transport
    ↓
Cloud Ingestion
    ↓
PostgreSQL
    ↓
Backend API
    ↓
UI/UX Cloud Plane
```

## Customer OOB Monitoring Plane

The Customer OOB Monitoring Plane runs in the customer environment.

Responsibilities:

- Poll monitored devices using SNMP.
- Parse OID responses.
- Normalize telemetry into collector events.
- Buffer telemetry locally during transport interruptions.
- Initiate outbound-only secure telemetry connections.

The Customer OOB Monitoring Plane does not host PostgreSQL, cloud ingestion, the Backend API, or the UI/UX Cloud Plane.

## Secure Outbound Telemetry Transport

Secure Outbound Telemetry Transport delivers collector telemetry from the Customer OOB Monitoring Plane to cloud ingestion.

Responsibilities:

- Accept outbound collector connections.
- Protect telemetry in transit.
- Decouple device polling from cloud processing.
- Deliver telemetry to cloud ingestion.

MQTT/TLS is the current transport implementation. Transport is not storage and must not be treated as the source of monitoring state.

## UI/UX Cloud Plane

The UI/UX Cloud Plane owns cloud ingestion, PostgreSQL, Backend API, aggregation, visualization, and monitoring state.

Cloud Ingestion:

- Consumes telemetry from Secure Outbound Telemetry Transport.
- Validates payloads.
- Transforms telemetry.
- Writes monitoring state and history to PostgreSQL.

PostgreSQL:

- Stores inventory, current monitoring state, telemetry history, and alerts.
- Acts as the authoritative system of record.

Backend API:

- Provides application data to frontend clients.
- Handles application logic and API contracts.
- Reads monitoring state from PostgreSQL.

Frontend:

- Consumes data through the Backend API only.
- Displays network health, device state, interface telemetry, and alerts.

## Failure Handling

Device failures should create unhealthy device states after collector or ingestion validation.

Transport failures should trigger collector buffering and reconnect behavior.

Database failures should prevent acknowledged data loss through retry handling and controlled ingestion errors.

API failures should return controlled errors to clients.
