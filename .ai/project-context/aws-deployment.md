# AWS Deployment Notes — SNMP Collector v2

AWS is a possible UI/UX Cloud Plane implementation, not the product architecture. It may host MQTT/TLS transport, Ingestion, PostgreSQL, Backend API, frontend, and notification services. It never hosts customer SNMP devices, the collector, its local inventory/outbox, or its TUI/control surface.

Collectors establish outbound-only TLS-authenticated MQTT QoS 1 connections from customer networks. Cloud connectivity must not expose inbound access to the customer plane. Transport ACLs are least privilege; ingestion validates v2 messages before transactional persistence; cloud logs, APIs, and dashboards never surface collector secrets or local operator controls.
