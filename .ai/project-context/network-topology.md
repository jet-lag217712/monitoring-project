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
