# Monitoring Requirements

## Purpose

Define what the OGSD Monitoring Dashboard must show, how it should behave, and which data it must consume from the backend.

The dashboard is a read-only presentation layer for district network health. It must not own source-of-truth monitoring state.

## Current Product Shape

The current frontend exposes these monitoring views:

- All sites overview
- Site detail view
- Alert banner
- Demo mode fallback

The example statistics in `frontend/monitoring-dashboard/src/mockData.js` are hard coded and should be treated as representative UI fixtures, not authoritative monitoring data.

## Monitoring Scope

The MVP monitoring experience focuses on these questions:

- How many sites are being monitored?
- Which sites need attention?
- Which devices are unhealthy?
- What is the current utilization and responsiveness of key devices?
- What changed since the last poll?

The system should surface site-level health first, then allow drill-down to device-level detail.

## Frontend Reference Model

The current dashboard fixtures define the minimum shape the live API should support or derive.

Site overview records currently use these fields:

- `location`
- `type`
- `status`
- `idf_count`
- `device_count`
- `online_count`
- `avg_cpu`
- `avg_memory`
- `active_alerts`

Site detail records currently use these fields:

- `summary.total_devices`
- `summary.online_count`
- `summary.idf_count`
- `summary.active_alerts`
- `latest.devices`

Device detail records currently use these fields:

- `hostname`
- `role`
- `status`
- `cpu_pct`
- `memory_pct`
- `uptime_days`
- `latency_ms`

## Required Overview Data

The all-sites view must be able to render these values:

- Total sites
- Total monitored devices
- Critical site count
- Caution site count
- Last updated time
- Data mode indicator, such as live or demo

Each site card must be able to render these values:

- Site name or location
- Site type
- Site status
- IDF count
- Device count
- Online device count
- Average CPU utilization
- Average memory utilization
- Active alert count

## Required Site Detail Data

The site detail view must be able to render these values:

- Site name or location
- Total devices
- Online device count
- IDF count
- Active alert count

Each device row must be able to render these values:

- IP address
- Hostname
- Role
- Device status
- CPU utilization
- Memory utilization
- Uptime
- Latency

## Status Model

The dashboard must support the following site status values:

- `ok`
- `caution`
- `alert`

The dashboard must support the following device status values:

- `1` for healthy
- `2` for warning
- `3` for critical

Status labels, badges, and banner copy must remain consistent across the overview and detail screens.

## Data Requirements

The dashboard must receive monitoring data from the Backend API only.

The frontend should not query PostgreSQL, MQTT, or the SNMP collector directly.

The backend response must provide enough normalized data for the frontend to:

- render the overview
- render the site detail table
- derive alert counts
- filter sites locally by name, type, status, or identifier

## Refresh Requirements

The dashboard currently polls on a 5 second interval. That behavior should remain the default unless there is a documented reason to change it.

Polling should update:

- site inventory data
- site detail data when a site is selected
- test or mode configuration used by the UI

## Degradation Requirements

If the live API is unavailable, the dashboard should continue to render using demo data instead of failing into a blank state.

The UI must make the current data source visible to the user, so demo data is not mistaken for live telemetry.

## UX Requirements

The monitoring UI should prioritize fast scanning of health state.

Requirements:

- Use clear color coding for healthy, caution, and critical states.
- Show an alert banner when at least one site needs attention.
- Keep the overview searchable by site attributes.
- Keep site detail readable in a dense table layout.
- Preserve keyboard access for interactive site cards and navigation controls.
- Keep loading and empty states explicit.

## Non-Functional Requirements

Monitoring presentation should follow these standards:

- React components should stay small and predictable.
- Derived metrics should be computed from current state rather than stored redundantly.
- UI state should remain easy to reason about during polling and mode changes.
- Presentation logic should not depend on hidden backend side effects.

## Security And Ownership

Monitoring data is read-only from the dashboard perspective.

The dashboard must not expose secrets, raw credentials, or direct infrastructure access paths.

All authoritative state must live in PostgreSQL and be surfaced through the Backend API.

## Out Of Scope

The MVP monitoring dashboard does not need to:

- configure network devices
- edit inventory directly
- acknowledge alerts
- perform SNMP polling
- process MQTT traffic
- manage user permissions beyond future auth integration

## Open Questions

- Which metrics will be backed by live API responses in the first backend pass?
- Should demo mode remain a permanent fallback or only a development aid?
- Should alert thresholds be computed in the backend, the database, or the frontend?
- Should the overview include trend information, or remain strictly point-in-time?
- Should device status remain numeric in the API, or should the backend normalize it to strings for the frontend?
