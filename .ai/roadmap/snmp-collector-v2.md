# SNMP Collector v2 local roadmap

SNMP Collector v2 is the collector runtime inside the Equate appliance. It
observes local device reachability, emits versioned telemetry, and provides the
local operator TUI. It must remain useful during temporary MQTT or database
interruptions and must never become a remote management channel.

## Runtime contract

```text
SNMPv2c → poll/profile/filter/health → SQLite outbox
       → MQTT/TLS → local ingestion → PostgreSQL → API → UI
```

The collector owns polling-path evidence and local configuration. Ingestion
owns validation and persistence. PostgreSQL is authoritative for dashboard
state. The TUI is a client of the collector's Unix control socket.

## v2 event routes

```text
site/{site_id}/device/{device_id}/telemetry/v2/device
site/{site_id}/device/{device_id}/telemetry/v2/interface
site/{site_id}/device/{device_id}/telemetry/v2/health
site/{site_id}/collector/{collector_id}/telemetry/v2/heartbeat
```

Every event carries the shared envelope, event ID, observation/publication
timestamps, identity, and non-secret configuration revision. The schemas under
`docs/schemas/snmp-collector-v2/` are authoritative.

## Collection and health

- Core SNMPv2-MIB and IF-MIB identity/interface data are collected for every
  device.
- Supported vendor profiles add CPU, memory, temperature, and power readings.
- Unsupported readings are omitted rather than represented as zero.
- Interface filters run after inventory collection and before emission.
- A directly unreachable device becomes Critical after the configured failure
  threshold.
- A dependent device is Unknown only when every configured upstream path is
  unavailable.
- A responding device is Healthy or Warning according to the active temperature
  policy.

## Configuration and TUI

The setup TUI creates the local site layout. The day-2 TUI provides:

- inventory and device/interface views;
- reviewed, allowlisted CIDR discovery;
- temperature thresholds and upstream dependencies;
- interface filters and transport status;
- active revision, validation results, and reload.

Static YAML is read-only. The TUI writes only managed inventory with restrictive
permissions, fsync, atomic rename, explicit confirmation, and a secret-free
audit record.

## Validation and failure behavior

`collector validate -config` checks YAML, secret reference names, permissions,
inventory uniqueness, dependency cycles, profile names, interface expressions,
discovery bounds, and MQTT/TLS settings. Reload is transactional; an invalid
configuration retains the previous active snapshot.

MQTT outages do not stop polling. Events remain in SQLite and flush in order
after recovery. Database failures prevent ACK, allowing QoS 1 redelivery.
Readiness reports the active snapshot, buffer, and local publisher state.

## Delivery sequence

1. Establish schemas, reason codes, units, and ownership boundaries.
2. Implement merged static/managed inventory and atomic reload.
3. Implement bounded polling, profiles, filters, and reviewed discovery.
4. Implement local health/dependency correlation.
5. Emit v2 events and persist them transactionally.
6. Complete the TUI and local appliance integration.
7. Run OVA acceptance, outage drills, rollback, and restore validation.

## Deliberate exclusions

SNMPv3, SNMP writes, device console access, automatic enrollment, inferred
topology, remote TUI access, and secret-bearing telemetry remain out of scope.
