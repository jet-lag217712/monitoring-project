# Ingestion Service Architecture

## Purpose

The Ingestion Service is responsible for consuming monitoring data from the MQTT Broker, validating incoming messages, transforming payloads into the platform's internal data model, and persisting monitoring data into PostgreSQL.

The service acts as the bridge between the messaging layer and the storage layer.

The Ingestion Service is the only service authorized to write monitoring data into PostgreSQL.

---

## Responsibilities

The Ingestion Service is responsible for:

* Subscribing to MQTT topics
* Receiving monitoring messages
* Validating payload structure
* Validating required fields
* Transforming incoming data
* Creating and updating inventory records
* Storing metric samples
* Storing interface samples
* Generating ingestion logs
* Rejecting malformed messages

The Ingestion Service is not responsible for:

* SNMP polling
* MQTT routing
* User authentication
* Dashboard rendering
* Alert presentation

---

## Dependencies

### Upstream Dependencies

* MQTT Broker

### Downstream Dependencies

* PostgreSQL Database

---

## Data Flow

```text
SNMP Collector
↓
MQTT Broker
↓
Ingestion Service
↓
PostgreSQL
```

The Ingestion Service consumes messages from MQTT and converts them into database records.

No other service should write monitoring data directly into PostgreSQL.

---

## MQTT Topic Structure

Topics should follow a predictable hierarchy.

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
The topic structure provides routing context while the payload contains measurement data.

---

## Payload Structure

All payloads should be JSON.

Example:
```text
{
"timestamp": "2026-06-01T18:00:00Z",
"site_id": "hub",
"device_id": "rtr-01",
"metric": "cpu_utilization",
"value": 42.5
}
```
Required fields:

* timestamp
* site_id
* device_id
* metric
* value

Messages missing required fields are rejected.

---

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
This allows remote collectors to introduce devices without manual database updates.

---

## Interface Discovery

Interface records should be created automatically when interface metrics are received.

Workflow:

Interface Metric Received
↓
Interface Exists?
↓
No
↓
Create Interface Record
↓
Store Sample

Interfaces are uniquely identified by:

* device_id
* if_index

---

## Validation Rules

Every message must pass validation before persistence.

Validation checks include:

* Valid JSON
* Valid timestamp
* Required fields present
* Numeric metric values
* Known device identifier
* Valid MQTT topic format

Invalid messages are rejected and logged.

The service should never crash due to malformed payloads.

---

## Database Writes

### Device Updates

The service updates:

* last_seen
* status

when valid metrics are received.

### Metric Samples

Device-level metrics are stored within:
```text
metric_samples
```
Examples:

* CPU utilization
* Memory utilization
* Uptime

### Interface Samples

Interface-level metrics are stored within:
```text
interface_samples
```
Examples:

* Bandwidth counters
* Error counters
* Discard counters

---

## Error Handling

All processing failures should be logged.

Categories:

* Invalid Payload
* Validation Failure
* Database Failure
* MQTT Failure

Failed messages should not terminate the service.

The service should continue processing subsequent messages.

---

## Logging

The service produces structured JSON logs.

Each processed message should include:

* Timestamp
* Site ID
* Device ID
* Metric Name
* Processing Result

Example Results:

* accepted
* rejected
* database_error

---

## Availability Requirements

The service must be stateless.

Multiple ingestion instances may consume from the same MQTT Broker.

No monitoring state should be stored locally.

Temporary restarts must not result in data corruption.

---

## Security

All MQTT communication occurs over TLS.

The Ingestion Service accepts messages only from the MQTT Broker.

Database access is restricted to a dedicated ingestion account with write permissions.

Direct public access is prohibited.

---

## Future Enhancements

Potential future capabilities include:

* MQTT dead-letter queues
* Batch database writes
* Message deduplication
* Alert generation
* Metric enrichment
* High-volume ingestion optimization
* Stream processing

---

## Deployment Model

The Ingestion Service runs as a Docker container within the AWS private subnet.

Network Flow:
```
SNMP Collector
↓
MQTT Broker
↓
Ingestion Service
↓
PostgreSQL
```
The service maintains outbound connections to PostgreSQL and inbound subscriptions to MQTT topics.

No direct access from the Monitoring Dashboard is permitted.
