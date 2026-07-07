# Monitoring Dashboard Architecture

## Purpose

The dashboard provides visualization of infrastructure telemetry from the UI/UX Cloud Plane.

## Plane Ownership

Plane: UI/UX Cloud Plane.

The dashboard is a frontend client. It does not run in the Customer OOB Monitoring Plane and does not access monitored devices directly.

## Responsibilities

The frontend is responsible for:

- Displaying site health.
- Showing device status.
- Rendering interface telemetry.
- Displaying alerts.
- Surfacing current monitoring state from Backend API responses.

The frontend does not:

- Poll SNMP.
- Access databases directly.
- Process telemetry transport traffic.
- Configure monitored devices.
- Provide device console or management access.

## Data Source

All data is retrieved through the Backend API.

PostgreSQL remains the system of record, but frontend clients must never query PostgreSQL directly.

## Main Views

- Site Overview.
- Device Details.
- Interface Details.
- Alert Dashboard.
