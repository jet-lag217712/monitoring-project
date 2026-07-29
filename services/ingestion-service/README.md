# Equate Ingestion Service

The Ingestion Service is the local appliance consumer between MQTT and
PostgreSQL. It validates versioned collector events, deduplicates them,
persists monitoring state transactionally, and acknowledges a message only
after the database decision is durable.

```text
local MQTT/TLS → validate → deduplicate → PostgreSQL transaction → ACK
```

## Responsibilities

- Validate v2 device, interface, health, and heartbeat events.
- Cross-check topic identifiers against envelope identifiers.
- Persist inventory, samples, components, health evidence, and collector
  status/history.
- Deduplicate by `event_id` with natural keys as defense in depth.
- Reject malformed or unauthorized messages without writing partial state.

It does not poll SNMP, serve the dashboard, or modify collector inventory.

## Local configuration

The appliance subscribes to:

```text
site/+/device/+/telemetry/v2/#
site/+/collector/+/telemetry/v2/heartbeat
```

The broker, CA, and database are local containers in the appliance Compose
project. Production credentials are generated per installation and loaded from
runtime environment files.

## Build and test

```bash
cd services/ingestion-service
go test ./...
go run ./cmd/ingestion -config configs/ingestion.example.yaml
```

For integration validation:

```bash
./deployments/end-to-end/up.sh
./deployments/end-to-end/smoke.sh
```

The default administration listener is `:9091` and exposes only liveness and
metrics. It is not published by the production appliance.

## Acknowledgement rule

```text
MQTT receive → validate → deduplicate → database transaction → ACK
```

Non-retryable invalid messages are acknowledged as rejected. Database failures
remain unacknowledged so MQTT QoS 1 redelivery can retry the event.
