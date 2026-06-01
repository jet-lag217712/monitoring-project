# Service Contracts

## MQTT Contracts

### Device Metric

Publisher:
SNMP Collector

Consumer:
Ingestion Service

Topic:
```text
site/{site_id}/device/{device_id}/metric/device
```

Payload:
```text
{
"timestamp": "RFC3339",
"metric": "cpu_utilization",
"value": 42.5
}
```

### Interface Metric

Publisher:
SNMP Collector

Consumer:
Ingestion Service

Topic:
```text
site/{site_id}/device/{device_id}/metric/interface
```

Payload:
```text
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
```text
{
...
}
```

## Database Ownership

Ingestion Service:
Write


Backend API:
Read

Dashboard:
None