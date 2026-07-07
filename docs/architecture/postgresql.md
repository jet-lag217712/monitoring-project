# PostgreSQL Architecture

## Purpose

PostgreSQL stores monitoring state and historical telemetry.

## Responsibilities

Stores:

-   Sites
-   Devices
-   Interfaces
-   Metrics
-   Alerts
-   User data

## Design Goals

The database should support:

-   Time-series queries.
-   Device history.
-   Dashboard queries.
-   Alert generation.

## Operational Requirements

Production deployment should include:

-   Backups
-   Indexing
-   Migration management
-   Storage monitoring
