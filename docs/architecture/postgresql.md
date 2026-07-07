# PostgreSQL Architecture

## Purpose

PostgreSQL stores authoritative monitoring state and historical telemetry in the UI/UX Cloud Plane.

## Responsibilities

PostgreSQL stores:

- Sites.
- Monitored devices.
- Interfaces.
- Metrics.
- Alerts.
- User data when application authentication is implemented.

## Design Goals

The database should support:

- Time-series queries.
- Device history.
- Dashboard queries.
- Alert generation inputs.
- Backend API contracts.

## Operational Requirements

Production deployment should include:

- Backups.
- Indexing.
- Migration management.
- Storage monitoring.
- Access controls separating ingestion writes from API reads.

## Deployment Boundary

PostgreSQL belongs to the UI/UX Cloud Plane and is the system of record.

Collectors and frontend clients must never connect directly to PostgreSQL.
