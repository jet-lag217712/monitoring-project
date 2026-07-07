# SNMP Collector Architecture

## Purpose

The SNMP Collector gathers network telemetry from monitored devices inside the Customer OOB Monitoring Plane and sends normalized telemetry to the UI/UX Cloud Plane through outbound-only secure transport.

## Plane Ownership

Plane: Customer OOB Monitoring Plane.

The collector is deployed in the customer environment so SNMP access to monitored devices stays local.

## Responsibilities

The collector is responsible for:

- Polling configured SNMP endpoints.
- Querying configured OIDs.
- Parsing SNMP responses.
- Normalizing device and interface telemetry.
- Buffering telemetry locally during transport interruptions.
- Publishing telemetry through outbound-only secure connections.

## Supported Model

Initial deployment:

- SNMPv2 monitoring.
- Network switches and routers.
- Read-only telemetry collection.

## Polling

Polling intervals should be configurable.

Examples:

- Device health checks.
- Interface telemetry.
- Resource utilization.

## Non-Responsibilities

The collector does not:

- Store telemetry permanently.
- Generate alerts.
- Serve dashboard requests.
- Host the Backend API.
- Host PostgreSQL.
- Configure monitored devices.
- Provide device console or management access.

## Deployment Boundary

The collector must not require inbound connectivity from the UI/UX Cloud Plane.

The approved flow is:

```text
SNMP Devices
    ↓
SNMP Collector
    ↓
Secure Outbound Telemetry Transport
    ↓
Cloud Ingestion
```
