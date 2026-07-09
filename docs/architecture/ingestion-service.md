# Ingestion Service Architecture

## Purpose

The Ingestion Service validates telemetry delivered through Secure Outbound Telemetry Transport, transforms payloads into the platform data model, and persists monitoring data into PostgreSQL.

The service bridges telemetry delivery and durable storage in the UI/UX Cloud Plane.

## Plane Ownership

Plane: UI/UX Cloud Plane.

The Ingestion Service runs in the cloud plane with access to PostgreSQL. It is not deployed inside the Customer OOB Monitoring Plane.

## Responsibilities

The Ingestion Service is responsible for:

- Consuming telemetry delivered by Secure Outbound Telemetry Transport.
- Validating payload structure and required fields.
- Transforming collector string IDs into deterministic UUID primary keys (UUID v5).
- Creating and updating inventory records (auto-discovery).
- Storing metric samples and interface samples idempotently.
- Acknowledging MQTT messages only after successful persistence (or safe reject/dedup).
- Generating structured ingestion logs and Prometheus metrics.
- Rejecting malformed messages.

The Ingestion Service is not responsible for:

- SNMP polling.
- Telemetry transport routing.
- User authentication.
- Dashboard rendering.
- Alert presentation.
- Device configuration or console access.

## Dependencies

Upstream:

- Secure Outbound Telemetry Transport.

Downstream:

- PostgreSQL Database.

## Data Flow

```text
SNMP Collector
    ↓
Secure Outbound Telemetry Transport
    ↓
Ingestion Service
    ↓
PostgreSQL
```

No other service should write monitoring data directly into PostgreSQL.

MQTT/TLS is the current telemetry transport implementation. Implementation-specific topic names may be used by the transport, but architecture docs should describe the boundary as Secure Outbound Telemetry Transport.

## Transport Contract Shape

Topics match [`contracts.md`](contracts.md). The metric *kind* is in the topic path (`device` or `interface`), not a metric name.

```text
site/{site_id}/device/{device_id}/metric/device
site/{site_id}/device/{device_id}/metric/interface
```

Examples:

```text
site/site-001/device/dev-001/metric/device
site/site-001/device/dev-001/metric/interface
```

Topic path IDs are authoritative. The collector may also include `site_id` and `device_id` in the JSON body; when present they must match the topic.

## Payload Structure

All payloads are JSON. Shapes match the collector wire format and [`contracts.md`](contracts.md).

### Device metric

```json
{
  "timestamp": "2026-06-01T18:00:00Z",
  "site_id": "site-001",
  "device_id": "dev-001",
  "metric": "uptime_seconds",
  "value": 12345.0
}
```

Required body fields: `timestamp`, `metric`, `value`. Optional: `site_id`, `device_id` (cross-checked against topic when present).

### Interface metric

```json
{
  "timestamp": "2026-06-01T18:00:00Z",
  "site_id": "site-001",
  "device_id": "dev-001",
  "if_index": 2,
  "in_octets": 123,
  "out_octets": 456,
  "in_errors": 0,
  "out_errors": 0
}
```

Required body fields: `timestamp`, `if_index`, `in_octets`, `out_octets`, `in_errors`, `out_errors`. Optional: `site_id`, `device_id`.

## Idempotent Ingestion Pipeline

MQTT QoS 1 with **manual acknowledgment**. Never ACK a new message before the database transaction commits.

```text
MQTT Receive → Validate → Deduplicate → DB Commit → ACK
```

| Condition | Action |
|-----------|--------|
| Invalid messages | ACK immediately and log `rejected` (prevent poison-queue loops) |
| Duplicates | ACK without insert (idempotent success); log `deduplicated` |
| New messages | ACK **only** after transaction commits; log `accepted` |
| DB failures | No ACK; broker redelivers; log `database_error` |
| Unknown metric type | ACK and reject (non-retryable until seed data exists) |

Consumer settings: QoS 1, manual ACK, `Clean Session = false` (session recovery).

### Idempotency keys

| Message type | Natural key |
|--------------|-------------|
| Device metric | `(device_id, metric_type_id, collected_at)` |
| Interface metric | `(interface_id, collected_at)` |

Enforced by `UNIQUE` constraints plus `INSERT ... ON CONFLICT DO NOTHING`.

## Identifier Mapping

Collector uses string IDs (`site-001`, `dev-001`). PostgreSQL entity tables use UUID primary keys.

Ingestion derives deterministic **UUID v5** values from a fixed OGSD namespace:

- Site UUID from `site_id`
- Device UUID from `site_id` + `device_id`
- Interface UUID from device UUID + `if_index`

This enables stable auto-discovery without a separate ID registry.

## Device Discovery

The Ingestion Service automatically creates inventory records when previously unknown devices are received. Site → device upserts, sample insert, and `devices.last_seen` update run in a **single transaction**.

## Interface Discovery

Interface records are created automatically when interface telemetry is received. Interfaces are uniquely identified by `(device_id, if_index)`.

## Validation Rules

Every message must pass validation before persistence:

- Valid JSON.
- Valid RFC3339 timestamp.
- Required fields present for the message kind.
- Numeric values where required.
- Valid topic format (`.../metric/device` or `.../metric/interface`).
- Topic/body ID consistency when body IDs are present.

Invalid messages are rejected and logged. The service must never crash due to malformed payloads.

## Database Writes

Device updates include `last_seen` and `status`.

Device-level metrics are stored in `metric_samples`. Interface-level metrics are stored in `interface_samples`.

## Error Handling

All processing failures are logged. Categories: invalid payload, validation failure, database failure, transport failure. Failed messages must not terminate the service.

## Logging

Structured JSON logs. Each processed message includes timestamp, site ID, device ID, metric name (when applicable), and processing result:

- `accepted`
- `rejected`
- `deduplicated`
- `database_error`

## Observability

Prometheus metrics on the admin port (`/metrics`):

- `ingestion_messages_received_total`
- `ingestion_messages_accepted_total`
- `ingestion_messages_rejected_total`
- `ingestion_messages_deduplicated_total`
- `ingestion_db_write_failure_total`
- `ingestion_processing_duration_seconds`
- `ingestion_mqtt_connected`

## Availability Requirements

The service is stateless with respect to monitoring data (PostgreSQL is the system of record). Temporary restarts must not result in data corruption; MQTT session recovery plus idempotent inserts handle redelivery.

## Security

All telemetry transport communication must use TLS. The Ingestion Service accepts messages only from authenticated transport paths. Database access is restricted to a dedicated ingestion account with write permissions. Direct public access is prohibited.

## Future Enhancements

Potential future capabilities include:

- Dead-letter handling.
- Batch database writes.
- Alert generation.
- Metric enrichment.
- High-volume ingestion optimization.
- Stream processing.

## Deployment Boundary

The Ingestion Service runs in the UI/UX Cloud Plane.

Network flow:

```text
Secure Outbound Telemetry Transport
    ↓
Ingestion Service
    ↓
PostgreSQL
```

No direct access from the frontend is permitted.
