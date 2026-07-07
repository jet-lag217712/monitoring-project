# Service Boundaries

### SNMP Collector
* Plane: Customer OOB Monitoring Plane
* Owns SNMP polling of monitored devices
* Owns local telemetry buffering before outbound delivery
* Does not host APIs, PostgreSQL, or user-facing workflows

### Secure Outbound Telemetry Transport
* Plane: boundary between Customer OOB Monitoring Plane and UI/UX Cloud Plane
* Owns secure telemetry delivery between planes
* Treats MQTT/TLS as the current transport implementation
* Does not own monitoring state or storage

### Ingestion Service
* Plane: UI/UX Cloud Plane
* Owns telemetry transport contracts
* Owns database writes
* Owns payload validation and normalization
* Does not poll SNMP or serve frontend requests

### Backend API
* Plane: UI/UX Cloud Plane
* Owns REST contracts
* Owns database reads
* Does not poll SNMP, process telemetry transport, or write monitoring samples

### Dashboard
* Plane: UI/UX Cloud Plane
* Owns presentation
* Reads monitoring data only through the Backend API
* Does not access PostgreSQL, transport, or collectors directly
