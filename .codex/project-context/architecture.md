# Architecture

## Purpose

Equate OGSD is a cloud-hosted SNMP monitoring platform for K-12 network infrastructure. The system collects SNMP telemetry from district network devices, securely transports telemetry to AWS, stores normalized state in PostgreSQL, and exposes monitoring data through a web dashboard.

This document provides a high-level architectural overview for AI agents. Detailed implementation, networking, and service ownership information is documented elsewhere.

## High-Level Data Flow

```text
SNMP Devices
    ↓
SNMP Collector
    ↓ MQTT/TLS
MQTT Broker
    ↓
Ingestion Service
    ↓
PostgreSQL
    ↓
Backend API
    ↓
Monitoring Dashboard
```

## Core Components

### SNMP Collector

Runs within the district environment.

Responsibilities:

* Poll network devices via SNMPv2.
* Buffer telemetry locally.
* Publish telemetry to AWS through MQTT over TLS.
* Operate without inbound connectivity requirements.

### MQTT Broker

Runs in AWS.

Responsibilities:

* Receive telemetry from collectors.
* Decouple collection from ingestion.
* Route messages to downstream consumers.

### Ingestion Service

Runs in AWS.

Responsibilities:

* Validate incoming telemetry.
* Reject malformed payloads.
* Transform raw telemetry into normalized records.
* Persist data to PostgreSQL.

### PostgreSQL

Authoritative system of record.

Responsibilities:

* Store device inventory.
* Store telemetry and monitoring state.
* Support dashboard and API queries.

### Backend API

Runs in AWS.

Responsibilities:

* Provide read-oriented access to monitoring data.
* Expose stable contracts for frontend consumption.
* Enforce business logic and data access rules.

### Monitoring Dashboard

React-based frontend.

Responsibilities:

* Display monitoring information.
* Visualize device health and status.
* Consume data exclusively through the Backend API.

## System of Record

PostgreSQL is the authoritative source of system state.

Services may cache data for performance but must not treat caches, MQTT messages, local files, or in-memory state as authoritative.

Dashboard data must originate from the Backend API, which reads from PostgreSQL.

## Design Principles

* The SNMP Collector is deployable on-premises.
* AWS receives telemetry through outbound collector connections only.
* MQTT is a transport mechanism, not a persistence layer.
* PostgreSQL is the source of truth.
* Services communicate through well-defined contracts.
* Components are independently deployable.
* All runtime services are containerized.
* Prefer simplicity over premature optimization.

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
