# Vendor SNMP Packages

This directory holds vendor-specific OID collectors that extend the standard
`snmp/core` packages (IF-MIB + SNMPv2-MIB).

## Extension pattern

1. Create a package named after the vendor, e.g. `cisco/`, `juniper/`.
2. Export a poll function that accepts a context and SNMP client and returns
   additional device metrics (CPU, memory, etc.).
3. Select the package from device config via the `vendor` field:

```yaml
devices:
  - id: "sw-01"
    host: "10.0.0.1"
    vendor: "cisco"
```

When `vendor` is empty or no matching package exists, the collector falls back
to core-only metrics (`uptime_seconds` + IF-MIB interface counters).

## Phase 1 status

No vendor implementations yet. Core MIB polling is sufficient for MVP.
