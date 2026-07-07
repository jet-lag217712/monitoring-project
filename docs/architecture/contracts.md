# Service Contracts

## Telemetry Transport Contracts

The contract between SNMP collectors and cloud ingestion is implementation-neutral at the architecture level.

MQTT/TLS is the current transport implementation. When MQTT is used, routes may be expressed as topics, but transport remains delivery only and PostgreSQL remains the system of record.

### Device Metric

Publisher:

- SNMP Collector.

Consumer:

- Ingestion Service.

Route:

```text
site/{site_id}/device/{device_id}/metric/device
```

Payload:

```json
{
  "timestamp": "RFC3339",
  "metric": "cpu_utilization",
  "value": 42.5
}
```

### Interface Metric

Publisher:

- SNMP Collector.

Consumer:

- Ingestion Service.

Route:

```text
site/{site_id}/device/{device_id}/metric/interface
```

Payload:

```json
{
  "timestamp": "RFC3339",
  "if_index": 2,
  "in_octets": 123,
  "out_octets": 456,
  "in_errors": 0,
  "out_errors": 0
}
```

## REST Contracts

```text
GET /api/v1/devices
```

Response:

```json
{
  "devices": []
}
```

## Database Ownership

Ingestion Service:

- Writes monitoring data.

Backend API:

- Reads monitoring data for frontend clients.

Dashboard:

- No direct database access.
