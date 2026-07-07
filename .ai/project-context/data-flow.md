# Data Flow Architecture

## Purpose

Defines the complete telemetry lifecycle from network devices to cloud-facing services.

## Telemetry Pipeline

Arista Switches generate operational data.

SNMP Collector:
- Polls devices using SNMP.
- Parses OID responses.
- Normalizes telemetry.
- Publishes telemetry messages.

MQTT Broker:
- Provides asynchronous communication between collectors and ingestion.
- Buffers messages.
- Decouples polling from processing.

Ingestion Service:
- Consumes MQTT messages.
- Validates payloads.
- Transforms telemetry.
- Stores data in PostgreSQL.

Backend API:
- Provides application data to the dashboard.
- Handles business logic.
- Provides alert and device state information.

Cloud Services:
- Monitoring Dashboard consumes API data.
- Email Service handles notifications.

## Failure Handling

Device failures should create unhealthy device states.

MQTT failures should trigger reconnect behavior.

Database failures should prevent data loss through retry handling.

API failures should return controlled errors to clients.
""",

".ai/project-context/network-topology.md": """
# Network Topology

## Purpose

Defines network boundaries and communication paths.

## On-Prem Environment

The monitoring stack runs inside the district environment.

Components:
- SNMP Collector
- MQTT Broker
- Ingestion Service
- PostgreSQL
- Backend API

## Communication

SNMP:
- Collector communicates directly with network devices.

Internal Services:
- Docker networking is used for service communication.

Cloud Connectivity:
- Outbound HTTPS communication is preferred.
- No inbound access into the district network is required.

## Security Boundary

Telemetry collection and storage remain on-premises.

Only required application traffic leaves the environment.
""",

".ai/project-context/aws-deployment.md": """
# AWS Deployment

## Purpose

Defines cloud-hosted components.

## Cloud Responsibilities

AWS hosts:

- Monitoring Dashboard
- Email Service

AWS does not host:

- SNMP Collector
- MQTT Broker
- Ingestion Service
- PostgreSQL

## Connectivity

Cloud services communicate with the on-prem Backend API.

Security requirements:
- TLS encryption
- Authentication
- Restricted API access

## Deployment Model

The cloud layer is the presentation and notification layer.
The monitoring data pipeline remains on-premises.
""",

"docs/architecture/monitoring-dashboard.md": """
# Monitoring Dashboard Architecture

## Purpose

The dashboard provides visualization of infrastructure telemetry.

## Responsibilities

The frontend is responsible for:

- Displaying site health.
- Showing device status.
- Rendering interface metrics.
- Displaying alerts.

The frontend does not:

- Poll SNMP.
- Access databases directly.
- Process telemetry.

## Data Source

All data is retrieved through the Backend API.

## Main Views

- Site Overview
- Device Details
- Interface Metrics
- Alert Dashboard