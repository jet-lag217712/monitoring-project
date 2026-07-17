## collector - 3

### Primary Service

SNMP Collector

### Choice Made

Phase 2 uses a strict stage-owned polling result. Core polling owns SNMPv2-MIB
identity and IF-MIB inventory/counters. Vendor profiles may only add optional
vendor-neutral readings, their static capability bit flags, and the selected
profile name. Interface filtering may only add per-interface selection
annotations and bounded aggregate counts. Normalization transforms the result
into the existing v1 events, and the publisher remains unchanged.

A successful core poll always produces a valid `DevicePollResult`. Profile
collection failures are observed independently and never discard or invalidate
core data. Unsupported profile readings are absent rather than represented by
fabricated zero values.

Profile selection uses `sysObjectID` only, in this order: exact object ID,
longest model-family prefix, generic enterprise profile, then core-only.
`sysDescr` may be logged when no profile matches, but never selects a profile.

Interface filtering annotates every interface as `Selected`,
`ExcludedDefault`, or `ExcludedRule`. Defaults exclude virtual IF-MIB types,
explicit includes can restore a default-excluded interface, and any matching
explicit exclusion wins. Only selected interfaces with collected counters are
bridged to v1 interface events.

Discovery is an operator-invoked CLI workflow isolated from the poll scheduler.
It scans only configured CIDR allowlists, applies a token bucket before every
SNMP probe, persists review candidates, and never auto-enrolls a device.
Export is non-activating; acceptance writes only through the existing atomic
managed-inventory path and requires full configuration validation. The
discovery community is referenced by `discovery.community_env` and resolved
only when discovery probes run.

### Status

Accepted — Phase 2 implementation decision.
