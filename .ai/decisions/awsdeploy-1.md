## awsdeploy - 1

### Primary Service
Deployments (AWS)

### Secondary Services
- SNMP Collector
- Ingestion Service
- Database
- Backend API
- Monitoring Dashboard

### Choice Made
Use a two-plane deployment model: Customer OOB Monitoring Plane on-premises and UI/UX Cloud Plane in cloud.

### Supersedes
Any earlier guidance that placed Backend API, PostgreSQL, or cloud ingestion inside the customer environment as the product architecture.

### Alternatives Considered
- Full monitoring stack inside the customer environment
- Cloud presentation layer communicating with an on-prem Backend API

### Pros
- Keeps device SNMP access local to the customer environment
- Requires only outbound-only collector telemetry connections
- Centralizes ingestion, PostgreSQL, Backend API, aggregation, visualization, and monitoring state in the UI/UX Cloud Plane
- Treats AWS and MQTT as implementation choices rather than architecture boundaries

### Cons
- Requires secure telemetry transport between planes
- Requires collector buffering for transport outages

### Cost / Benefit
The model preserves customer network isolation while giving the cloud plane a single authoritative PostgreSQL-backed monitoring state for UI and API workflows.

### Status
Accepted
