# Architecture

## Purpose

Equate OGSD is a cloud-native Out-of-Band (OOB) network management and monitoring platform for K-12 network infrastructure. The system monitors district network devices through distributed SNMP collectors deployed within customer environments and presents operational visibility through the UI/UX Cloud Plane.

The platform separates monitoring collection from the user experience layer. The on-premises OOB plane is responsible for collecting and securely transmitting telemetry, while the UI/UX Cloud Plane provides centralized visualization, API access, and operational workflows.

This document provides a high-level architectural overview for AI agents. Detailed implementation, networking, and service ownership information is documented elsewhere.

## High-Level Data Flow

```text
SNMP Devices
    ↓
SNMP Collector (On-Premises OOB Plane)
    ↓ MQTT/TLS Outbound Connection
Message Transport Layer
    ↓
Ingestion Service
    ↓
PostgreSQL
    ↓
Backend API
    ↓
UI/UX Cloud Plane
```

## Core Components

### SNMP Collector

Runs within the district OOB environment.

Responsibilities:

* Poll network devices using SNMPv2.
* Translate device telemetry into normalized monitoring events.
* Buffer telemetry locally during connectivity interruptions.
* Publish telemetry through outbound secure connections.
* Operate without requiring inbound access from the cloud platform.

### Message Transport Layer

Provides secure telemetry delivery between customer environments and the cloud platform.

Responsibilities:

* Receive telemetry from distributed collectors.
* Decouple device monitoring from cloud processing.
* Provide reliable message delivery between planes.

### Ingestion Service

Runs as part of the cloud backend.

Responsibilities:

* Validate incoming telemetry.
* Reject malformed or unauthorized payloads.
* Normalize monitoring data.
* Persist operational state into the database.

### PostgreSQL

Authoritative system of record.

Responsibilities:

* Store device inventory.
* Store telemetry history and current monitoring state.
* Support backend queries for the UI/UX Cloud Plane.

### Backend API

Provides the application interface between stored monitoring data and the UI/UX Cloud Plane.

Responsibilities:

* Provide stable contracts for frontend consumption.
* Enforce application logic and access controls.
* Abstract database implementation details from the frontend.

### UI/UX Cloud Plane

The centralized operational interface for users.

Responsibilities:

* Display network health, device status, and telemetry.
* Provide site and device-level visibility.
* Present alerts, historical metrics, and operational dashboards.
* Consume monitoring data exclusively through the Backend API.

## System of Record

PostgreSQL is the authoritative source of system state.

Services may cache data for performance but must not treat caches, message queues, local files, or in-memory state as authoritative.

The UI/UX Cloud Plane receives operational state through the Backend API, which reads from PostgreSQL.

## Design Principles

* The SNMP Collector is deployable within customer OOB environments.
* Cloud communication uses outbound-only collector connections.
* The UI/UX Cloud Plane is independent from individual customer deployments.
* Message transport is a delivery mechanism, not a persistence layer.
* PostgreSQL is the source of truth.
* Services communicate through well-defined contracts.
* Components are independently deployable.
* Runtime services are containerized.
* Prefer operational simplicity over premature optimization.

## Non-Goals

The system is not intended to:

* Configure network devices.
* Provision network infrastructure.
* Function as an SD-WAN controller.
* Perform packet capture or packet analysis.
* Replace existing network management systems.

## Related Documents

* `network-topology.md`
* `service-boundaries.md`
* `docs/system-design.pdf`
* `docs/architecture/*`
