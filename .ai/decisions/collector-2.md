## collector - 2

### Primary Service

SNMP Collector

### Choice Made

Phase 1 uses a strict environment-reference configuration for SNMP
communities. Device inventory entries contain `community_env`, never a
plaintext `community`; the collector resolves the value only when creating an
SNMP session. Validation checks the reference name without requiring or
printing the secret value.

The static deployment inventory is authoritative when a managed inventory
entry has the same device ID. Unique managed entries are appended. Duplicate
IDs within either source and duplicate host/IP identities in the active merged
inventory are rejected. The managed file contains only a `devices` list and is
written with restrictive permissions through a temporary-file, fsync, and
atomic-rename flow. A configured but missing managed file represents an empty
managed source.

Configuration reloads use immutable snapshots. `SIGHUP` parses and validates
the complete configuration before swapping inventory and polling policy.
Existing polls retain their captured snapshot. Site and collector identity,
admin, publisher, MQTT, and buffer settings are startup-only during Phase 1.

### Status

Accepted — Phase 1 implementation decision.
