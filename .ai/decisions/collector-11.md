## collector - 11

### Primary Service

SNMP Collector

### Secondary Services

- Ingestion Service
- Backend API
- Monitoring Dashboard

### Choice Made

Operators can mark a device as **Administratively Ignored** from the local TUI.
The device remains in inventory and continues to be polled and health-evaluated,
but its health does not drive site alert aggregates.

#### Configuration

Managed overlay field `alerts_enabled` on a device:

- omitted / `true`: normal alerting
- `false`: Administratively Ignored

Static identity fields remain non-overlayable. The managed inventory file is
still the only file the TUI writes. Mutation follows the existing
prepare/commit/reload workflow via `device.alerting.prepare` and
`device.alerting.commit`.

#### Health contract

`alerts_enabled` is included on v2 `health_state` payloads. When the overlay
toggles without a health state change, the tracker emits
`transition: reasserted` so PostgreSQL receives the updated flag on the next
poll cycle.

#### Dashboard impact

Ingestion persists `alerts_enabled` on `device_health_current` /
`device_health_history`. The Backend API exposes
`alerts_enabled` / `administratively_ignored` on device projections and
excludes ignored devices from critical/warning/unknown site aggregates so
they do not flip site status to `alert` or `caution`.

### Status

Accepted
