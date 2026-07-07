# Monitoring Dashboard Architecture

## Purpose

The dashboard provides visualization of infrastructure telemetry.

## Responsibilities

The frontend is responsible for:

-   Displaying site health.
-   Showing device status.
-   Rendering interface metrics.
-   Displaying alerts.

The frontend does not:

-   Poll SNMP.
-   Access databases directly.
-   Process telemetry.

## Data Source

All data is retrieved through the Backend API.

## Main Views

-   Site Overview
-   Device Details
-   Interface Metrics
-   Alert Dashboard
