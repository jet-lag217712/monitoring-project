# AWS Deployment Notes

## Purpose

Defines one acceptable cloud deployment implementation for the UI/UX Cloud Plane.

AWS is a deployment choice, not the product architecture. Architecture is defined by the Customer OOB Monitoring Plane and the UI/UX Cloud Plane.

## Cloud Responsibilities

A cloud deployment may host:

- Frontend
- Backend API
- Cloud Ingestion
- PostgreSQL
- Secure Outbound Telemetry Transport endpoint
- Notification services

A cloud deployment does not host:

- SNMP Collector
- Customer monitored devices
- Customer OOB Monitoring Plane network access

## Connectivity

SNMP collectors initiate outbound-only secure telemetry connections from the Customer OOB Monitoring Plane to the UI/UX Cloud Plane.

Security requirements:
- TLS encryption
- Authentication
- Restricted telemetry and API access
- No required inbound cloud-initiated connections into the customer network

## Deployment Model

The cloud layer owns ingestion, storage, API access, aggregation, visualization, and notification workflows.

AWS-specific services, network constructs, and routing details belong in deployment runbooks or infrastructure notes, not architecture documents.
