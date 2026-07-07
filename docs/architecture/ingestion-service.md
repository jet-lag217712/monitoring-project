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
- Validating payload structure.
- Validating required fields.
- Transforming incoming data.
- Creating and updating inventory records.
- Storing metric samples.
- Storing interface samples.
- Generating ingestion logs.
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

Transport routes should follow a predictable hierarchy when the current MQTT implementation is used.

Format:

```text
site/{siteId}/device/{deviceId}/metric/{metricName}
```

Examples:

```text
site/hub/device/rtr-01/metric/cpu_utilization
site/hub/device/rtr-01/metric/memory_utilization
site/remote-01/device/rtr-05/metric/interface_traffic
```

The route provides delivery context while the payload contains measurement data.

## Payload Structure

All payloads should be JSON.

Example:

```json
{
  "timestamp": "2026-06-01T18:00:00Z",
  "site_id": "hub",
  "device_id": "rtr-01",
  "metric": "cpu_utilization",
  "value": 42.5
}
```

Required fields:

- `timestamp`
- `site_id`
- `device_id`
- `metric`
- `value`

Messages missing required fields are rejected.

## Device Discovery

The Ingestion Service automatically creates inventory records when previously unknown devices are received.

Workflow:

```text
Message Received
    ↓
Device Exists?
    ↓
No
    ↓
Create Device Record
    ↓
Store Metric
```

This allows remote collectors to introduce monitored devices without manual database updates.

## Interface Discovery

Interface records should be created automatically when interface telemetry is received.

Workflow:

```text
Interface Telemetry Received
    ↓
Interface Exists?
    ↓
No
    ↓
Create Interface Record
    ↓
Store Sample
```

Interfaces are uniquely identified by:

- `device_id`
- `if_index`

## Validation Rules

Every message must pass validation before persistence.

Validation checks include:

- Valid JSON.
- Valid timestamp.
- Required fields present.
- Numeric metric values where required.
- Known or discoverable device identifier.
- Valid transport route format when route metadata is present.

Invalid messages are rejected and logged.

The service should never crash due to malformed payloads.

## Database Writes

Device updates include:

- `last_seen`
- `status`

Device-level metrics are stored in:

```text
metric_samples
```

Interface-level metrics are stored in:

```text
interface_samples
```

## Error Handling

All processing failures should be logged.

Categories:

- Invalid payload.
- Validation failure.
- Database failure.
- Transport failure.

Failed messages should not terminate the service.

## Logging

The service produces structured JSON logs.

Each processed message should include:

- Timestamp.
- Site ID.
- Device ID.
- Metric name.
- Processing result.

Example results:

- `accepted`
- `rejected`
- `database_error`

## Availability Requirements

The service must be stateless.

Multiple ingestion instances may consume from the same transport implementation when supported by deployment configuration.

No monitoring state should be stored locally.

Temporary restarts must not result in data corruption.

## Security

All telemetry transport communication must use TLS.

The Ingestion Service accepts messages only from authenticated transport paths.

Database access is restricted to a dedicated ingestion account with write permissions.

Direct public access is prohibited.

## Future Enhancements

Potential future capabilities include:

- Dead-letter handling.
- Batch database writes.
- Message deduplication.
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
