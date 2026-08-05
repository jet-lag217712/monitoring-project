## dashboard - 4

### Primary Service

Monitoring Dashboard

### Secondary Services

- Backend API

### Choice Made

Move global search into the fixed app header and open a translucent results
popover under the header when the operator focuses the field or types a query.
The current page (sites grid / site detail / device detail) stays mounted and
remains visible through the dimmed backdrop. Search covers:

- sites (name / location / ID)
- devices by hostname and IP address (`GET /api/search?q=`)

Selecting a site or device hit navigates through the existing
`useNetworkDashboard` selection state (no router). Escape / X / backdrop click /
Clear dismisses the popover and restores the prior view context after navigation
handlers clear the query.

The mid-page overview `SearchBar` is removed so there is a single entry point
in the nav.

### Status

Accepted
