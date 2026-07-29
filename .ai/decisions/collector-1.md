## collector - 1

### Primary Service

SNMP Collector

### Secondary Services

- Ingestion Service
- Database
- Backend API
- Monitoring Dashboard
- Secure Outbound Telemetry Transport

### Choice Made

Phase 0 establishes a versioned, JSON Schema Draft 2020-12 contract for SNMP
Collector v2 telemetry. Events use a shared envelope with a UUID event ID,
observation and emission timestamps, site/collector/device identity, and a
non-secret configuration revision. Event-specific data is contained under
`payload`.

The schemas are stored under
`docs/schemas/snmp-collector-v2/` and define device, interface, health, and
collector-heartbeat events. MQTT routes remain versioned and preserve v1 route
consumption during migration.

### Health taxonomy

The collector owns local health evaluation because it observes both the SNMP
polling result and the configured dependency DAG. The public states are:

- `healthy`: reachable and below the temperature threshold.
- `warning`: reachable and at or above the temperature threshold.
- `critical`: directly unreachable after the consecutive-failure threshold.
- `unknown`: unreachable because every configured upstream path is unavailable.

The public reason codes are `reachable`, `temperature_threshold`,
`direct_unreachable`, `upstream_unreachable`, and `recovered`. A failure below
the threshold does not create a terminal state transition; it retains the prior
state and records the pending count as evidence.

### Health transition model

The state machine is deterministic:

```text
unobserved --success/below threshold--> healthy
unobserved --success/at threshold----> warning
healthy    --temperature threshold---> warning
warning    --temperature recovery----> healthy
any state  --direct failure threshold-> critical
any state  --all upstream paths fail--> unknown
critical   --successful poll----------> healthy or warning
unknown    --successful poll----------> healthy or warning
```

The collector polls every device independently. A failed device with at least
one successfully polled configured upstream is direct failure evidence and may
become Critical after the threshold. A dependent becomes Unknown only when all
configured upstream paths are unavailable. Root-cause IDs and unavailable
upstream IDs are retained in the health event. CPU, memory, and power never
drive v2 health state.

### Ownership boundaries

| Component | Owns | Does not own |
|---|---|---|
| SNMP Collector | Polling, profile detection, normalization, local health, event creation, SQLite outbox | PostgreSQL persistence, public API, dashboard behavior |
| MQTT/TLS transport | Authentication, delivery, QoS 1, reconnect, transport buffering | Schema meaning, health decisions, persistence |
| Ingestion Service | Validation, topic/body checks, deduplication, database transactions, ACK decisions | SNMP polling, dashboard presentation |
| PostgreSQL | Authoritative inventory, samples, health, dependency, collector state/history | SNMP access or message delivery |
| Backend API | Read-only projections and compatibility responses | Polling or re-evaluating collector health |
| React dashboard | API adaptation and visual treatment | Direct collector/database access or health inference |
| Local TUI/control socket | Local operator status and approved control actions | Public management paths or secret exposure |
| Contract schemas | Versioned producer/consumer interface | Runtime ownership of event semantics |

### Migration and compatibility

The migration is additive:

1. Seed recognized v2 metrics and units.
2. Add device/profile/interface/component persistence.
3. Add health current state/history and dependency evidence.
4. Add collector status and heartbeat history.
5. Add event-ID deduplication and observation-time indexes.
6. Deploy ingestion support before enabling v2 producers.
7. Run v1/v2 compatibility and dual-publish validation.
8. Cut over consumers, then deprecate v1 publishing through a separately
   reviewed change.

V2 deduplicates by `event_id` and retains natural sample keys as defense in
   depth. Observation time, not arrival time, controls current-state updates.
MQTT messages are acknowledged only after successful persistence; invalid
non-retryable messages are acknowledged as rejected, while database failures
are redelivered.

Initial v2 retention is append-only with no automated deletion or archival.
Time-based indexes are required; partitioning or archival requires a later
measured-capacity decision.

### Vendor evidence boundary

Official Cisco and Arista MIB documentation is sufficient for the Phase 0
mapping matrix. Sanitized lab captures are a required gate before Phase 2
profiles are implemented. No live communities, credentials, certificates, or
device-sensitive captures may be committed.

### Compatibility tests planned

- Collector event serialization, route/envelope validation, and redaction.
- Ingestion schema, enum, timestamp, identity, ordering, deduplication, and ACK
  behavior.
- Database migration constraints and observation-time indexes.
- API status compatibility and health/dependency fields.
- React adapter mappings and explicit Unknown presentation.
- Fixture-driven Cisco and Arista profile mappings before Phase 2 completion.

### Status

Accepted — Phase 0 design decision.
