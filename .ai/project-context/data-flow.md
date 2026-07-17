# Data Flow Architecture — SNMP Collector v2

## Pipeline

```text
SNMPv2c devices → collector poll/profile/filter/health evaluation → SQLite outbox
→ MQTT/TLS QoS 1 → ingestion validate/deduplicate/transaction → PostgreSQL → API → dashboard
```

The collector writes a durable event before publish and removes it only after broker acknowledgment. Ingestion uses manual acknowledgment: `receive → validate → deduplicate → upsert state/samples → commit → ACK`. Invalid or permanently unsupported messages are acknowledged as rejected; transaction failures are left unacknowledged for redelivery.

## Collector observations

Each cycle independently polls every device. Core SNMPv2-MIB and IF-MIB data are collected for all profiles; Cisco and Arista profiles add fixture-tested CPU, memory, temperature, and power data. Device/interface/health events carry the v2 shared envelope, identity, observation time, non-secret configuration revision, and event ID. Heartbeats add collector build/runtime/outbox information.

Health is evaluated locally: successful polls are Healthy or temperature-policy Warning; a direct failure reaches Critical after the consecutive-failure threshold; a failed dependent is Unknown only when all configured upstream paths are unavailable. Pending correlation retains prior terminal state. Ingestion persists the stated reason, failure count, policy, unavailable upstreams, and root causes rather than inferring a cascade.

## Failure behavior

MQTT outages do not stop polling; events accumulate in SQLite and flush in order when delivery returns. Database failure prevents an ACK, so QoS 1 redelivery plus idempotency protect persistence. A delayed heartbeat is history only when its observation time is older than current collector status. API failure returns controlled errors; dashboard demo mode remains visibly distinct from live data.
