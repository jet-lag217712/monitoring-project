# System Design — SNMP Collector v2

This is the maintainable source diagram for the current architecture. Generated exports are derived artifacts.

```mermaid
flowchart LR
    subgraph customer["Customer OOB Monitoring Plane"]
        inventory["Static + managed inventory"] --> collector["SNMP Collector v2"]
        devices["SNMPv2c devices"] -->|"independent polls"| collector
        tui["Local Bubble Tea TUI"] <-->|"Unix socket"| collector
        collector --> outbox["SQLite outbox"]
        collector --> local["metrics · healthz · readyz"]
    end

    subgraph cloud["UI/UX Cloud Plane"]
        broker["MQTT/TLS QoS 1"] --> ingestion["Ingestion"]
        ingestion --> postgres[("PostgreSQL\nsystem of record")]
        postgres --> api["Backend API"]
        api --> dashboard["React dashboard"]
    end

    outbox -->|"outbound-only v2 telemetry"| broker
```

The collector sends device, interface, health, and heartbeat telemetry. Dependency edges are local reachability evidence; they are not a complete physical-network graph. The TUI and status/control surface remain local-only.
