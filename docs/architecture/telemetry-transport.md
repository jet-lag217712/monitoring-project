# Secure Outbound Telemetry Transport Architecture — v2

Secure Outbound Telemetry Transport carries collector telemetry from the Customer OOB Monitoring Plane to cloud ingestion. MQTT/TLS is the current implementation; transport is never the system of record.

Collectors make outbound TLS-authenticated MQTT connections, publish QoS 1 events through a durable SQLite outbox, and retain queued events across outages. The broker acknowledges delivery; collectors remove outbox records only after acknowledgment. Ingestion uses manual acknowledgment and acknowledges a new event only after its PostgreSQL transaction commits.

V2 carries versioned device, interface, health, and collector-heartbeat routes defined in [`contracts.md`](contracts.md). ACLs grant collectors publish-only access to their allowed routes and ingestion subscription access. Topic/body identities and schema versions are validated at ingestion. At-least-once delivery is expected, so `event_id` and natural sample keys make persistence idempotent.

The boundary remains outbound-only: no cloud service gains an inbound management path to the customer network. TLS, authentication, least-privilege topic permissions, and secret redaction are mandatory.
