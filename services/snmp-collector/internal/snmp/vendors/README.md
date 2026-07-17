# Vendor SNMP Packages

This directory holds vendor-specific OID collectors that extend the standard
`snmp/core` packages (IF-MIB + SNMPv2-MIB).

## Detection and extension pattern

1. Core polling reads `sysObjectID`, identity, and IF-MIB first.
2. The registry selects an exact object ID, the longest model-family prefix,
   a generic enterprise profile, or core-only fallback—in that order.
3. The selected profile adds only its declared capability readings.

`sysDescr` never selects a profile. Device `vendor` configuration remains
optional operator metadata and is not detection authority.

Profiles return optional vendor-neutral CPU, memory, temperature, and power
readings. Unsupported or unavailable fields are omitted rather than emitted as
zero. A profile collection failure preserves the successful core result.

## Phase 2 fixtures

Cisco and Arista packages use sanitized synthetic fixtures based on the
documented MIB mappings. Fixtures cover successful collection, missing
subtrees, `NoSuchObject`, timeout, and partial-table behavior. They contain no
live customer addresses, communities, or credentials.
