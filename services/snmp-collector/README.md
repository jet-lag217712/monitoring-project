# SNMP Collector

## Plane Ownership

Customer OOB Monitoring Plane.

## Responsibilities

- Poll monitored devices using SNMP.
- Normalize device and interface telemetry.
- Buffer telemetry locally during connectivity interruptions.
- Publish telemetry through outbound-only secure transport.

## Non-Responsibilities

- Hosting PostgreSQL.
- Hosting the Backend API.
- Hosting cloud ingestion.
- Rendering UI workflows.
- Configuring monitored devices.
- Providing device console or management access.

## Deployment Boundary

The collector runs in the customer environment and initiates outbound-only telemetry connections to the UI/UX Cloud Plane.

Approved flow:

```text
SNMP Devices -> SNMP Collector -> Secure Outbound Telemetry Transport
```
