# Secure Outbound Telemetry Transport Architecture

## Purpose

Secure Outbound Telemetry Transport delivers monitoring events from SNMP collectors in the Customer OOB Monitoring Plane to cloud ingestion in the UI/UX Cloud Plane.

MQTT/TLS is the current implementation choice for this transport. It is not the product architecture and is not a system of record.

## Responsibilities

The transport layer is responsible for:

- Receiving telemetry from authenticated collectors over outbound-only secure connections.
- Delivering messages to cloud ingestion services.
- Decoupling collector polling from cloud processing.
- Protecting telemetry in transit.

## Communication Model

Publisher:

- SNMP Collector.

Consumer:

- Ingestion Service.

## Security

Production deployment should use:

- TLS.
- Collector authentication.
- Restricted route or topic permissions.
- No required inbound connectivity into the customer network.

## Scaling

Additional collectors can publish telemetry without changing downstream services.

Transport scaling must preserve PostgreSQL as the authoritative system of record.
