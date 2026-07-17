## dashboard - 3

### Primary Service

Monitoring Dashboard

### Secondary Services

- Backend API
- Database (read-only projections)

### Choice Made

Phase 5 API health projection prefers `device_health_current` when present and
maps `healthy|warning|critical|unknown` to numeric compatibility
`1|2|3|0`. Devices without a health row keep the MVP online→`1` /
offline→`3` fallback so mixed v1/v2 rollout remains usable.

Site aggregates count Critical roots separately from Unknown /
dependency-impacted devices. Site string status is `alert` when any device is
Critical, otherwise `caution` when any Warning or Unknown exists, else `ok`.

Absent CPU/memory/temperature samples are JSON `null` (never fabricated zeros).
Device detail embeds a 24h `history` block for cpu/memory/temperature/uptime
while retaining `GET /metrics`.

The React Unknown badge uses a dedicated slate `.unknown` token/class. Unknown
is never styled as Healthy (green) or Critical (red). Demo scenario
`cascade-unknown` exercises Critical root vs Unknown dependent presentation.

### Status

Accepted — Phase 5 implementation decision.
