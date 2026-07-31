## dashboard - 4

### Primary Service

Monitoring Dashboard

### Secondary Services

- Backend API

### Choice Made

Move global search into the fixed app header and open a full-page search
takeover when the operator focuses the field or types a query. Search covers:

- sites (name / location / ID)
- devices by hostname and IP address (`GET /api/search?q=`)

Selecting a site or device hit navigates through the existing
`useNetworkDashboard` selection state (no router). Escape / Clear / Close
dismisses the takeover and restores the prior view context after navigation
handlers clear the query.

The mid-page overview `SearchBar` is removed so there is a single entry point.

### Status

Accepted
