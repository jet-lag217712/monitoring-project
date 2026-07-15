# Architecture

## Purpose

Equate OGSD is a two-plane network telemetry and monitoring platform for K-12 network infrastructure. The system monitors district network devices through distributed SNMP collectors deployed within customer environments and presents operational visibility through the UI/UX Cloud Plane.

The platform separates telemetry collection from the user experience layer. The customer-side OOB monitoring environment is responsible for collecting and securely transmitting telemetry, while the UI/UX Cloud Plane provides centralized visualization, API access, and operational workflows.

This document provides a high-level architectural overview for AI agents. Detailed implementation, networking, and service ownership information is documented elsewhere.

## High-Level Data Flow

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

## Core Components

### Customer OOB Monitoring Plane

Runs inside the customer environment.

Responsibilities:

* Reach monitored devices over SNMP.
* Host one or more SNMP collectors.
* Buffer telemetry locally during connectivity interruptions.
* Initiate outbound-only secure telemetry connections to the cloud plane.

Non-responsibilities:

* Hosting the Backend API.
* Hosting PostgreSQL.
* Hosting cloud ingestion.
* Serving UI/UX Cloud Plane requests.

### SNMP Collector

Runs within the customer OOB monitoring environment.

Responsibilities:

* Poll network devices using SNMPv2.
* Translate device telemetry into normalized monitoring events.
* Buffer telemetry locally during connectivity interruptions.
* Publish telemetry through outbound-only secure connections.
* Operate without requiring inbound access from the cloud platform.

### Secure Outbound Telemetry Transport

Provides secure telemetry delivery between customer environments and the cloud platform.

Responsibilities:

* Receive telemetry from distributed collectors.
* Decouple device monitoring from cloud processing.
* Provide reliable message delivery between planes.
* Treat MQTT/TLS as the current transport implementation, not the product architecture or a system of record.

### Cloud Ingestion

Runs as part of the cloud backend.

Responsibilities:

* Validate incoming telemetry.
* Reject malformed or unauthorized payloads.
* Normalize monitoring data.
* Persist operational state into the database.

### PostgreSQL

Authoritative system of record in the UI/UX Cloud Plane.

Responsibilities:

* Store device inventory.
* Store telemetry history and current monitoring state.
* Support backend queries for the UI/UX Cloud Plane.

### Backend API

Runs in the UI/UX Cloud Plane and provides the application interface between stored monitoring data and frontend clients.

Responsibilities:

* Provide stable contracts for frontend consumption.
* Enforce application logic and access controls.
* Abstract database implementation details from the frontend.

### UI/UX Cloud Plane

The centralized cloud plane for ingestion, storage, APIs, visualization, aggregation, and monitoring state.

Responsibilities:

* Accept telemetry from Secure Outbound Telemetry Transport.
* Persist monitoring state in PostgreSQL.
* Expose Backend API contracts for frontend clients.
* Display network health, device status, and telemetry.
* Provide site and device-level visibility.
* Present alerts, historical metrics, and operational dashboards.
* Consume monitoring data exclusively through the Backend API.

## System of Record

PostgreSQL is the authoritative source of system state.

Services may cache data for performance but must not treat caches, message queues, local files, or in-memory state as authoritative.

The UI/UX Cloud Plane receives operational state through the Backend API, which reads from PostgreSQL.

## Design Principles

* The Customer OOB Monitoring Plane contains monitored devices and SNMP collectors only.
* Cloud communication uses outbound-only collector connections.
* The UI/UX Cloud Plane owns ingestion, PostgreSQL, Backend API, and frontend experiences.
* Message transport is a delivery mechanism, not a persistence layer.
* PostgreSQL is the source of truth.
* Services communicate through well-defined contracts.
* Components are independently deployable.
* Runtime services are containerized.
* Prefer operational simplicity over premature optimization.

## Non-Goals

The system is not intended to:

* Configure network devices.
* Provide direct device console access.
* Provision network infrastructure.
* Function as an SD-WAN controller.
* Perform packet capture or packet analysis.
* Replace existing network management systems.

## Deployment profiles

Operational stacks live under `deployments/`:

* **end-to-end** — single Compose project with every service for client-site validation (real SNMP targets; no simulator).
* **development** — Mac cloud plane plus a separate OrbStack VM collector attached to GNS3 via a Cloud node.
* **production** — hybrid skeleton: Azure-hosted cloud services and an on-site VxRail collector (Terraform deferred).

See `deployments/README.md` and `network-topology.md` for plane boundaries and commands.

## Related Documents

* `network-topology.md`
* `service-boundaries.md`
* `docs/diagrams/system-design.md`
* `docs/architecture/*`
* `deployments/README.md`
