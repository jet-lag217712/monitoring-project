# Service Boundaries

### SNMP Collector
* Owns SNMP polling

### MQTT Broker
* Owns message routing

### Ingestion Service
* Owns MQTT contracts
* Owns database writes

### Backend API
* Owns REST contracts
* Owns database reads

### Dashboard
* Owns presentation