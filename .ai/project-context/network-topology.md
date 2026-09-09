# Local appliance network topology

The appliance VM has one customer-facing edge network and private container
networks. Collectors use the VM route table to reach configured SNMP networks;
they publish telemetry only to the local Mosquitto broker.

```text
Monitored SNMP networks
          │
          ▼
     collector containers ── Unix sockets ── local operator TUI
          │
          ▼
     private MQTT/TLS → ingestion → PostgreSQL → API
                                                   │
                                                   ▼
                                           nginx on 80/443
```

The `upstream_device_ids` graph is a local dependency DAG for reachability
correlation within one collector inventory. It is not a complete physical
topology, does not suppress child polling, and does not create a management
path.

Cross-collector site relationships use `sites.upstream_site_ids` in PostgreSQL,
configured in the multi-site manifest and evaluated at read time by the backend
API. See [api-1](../decisions/api-1.md).

## Validation fixtures (non-product)

| Path | Purpose |
|---|---|
| `deployments/production/appliance/` | Supported all-in-one customer VM |
| `remote-server/` | GNS3 SNMP lab topology for field acceptance |

The local appliance publishes only nginx. Admin ports, metrics, broker,
database, and control sockets stay on private networks or loopback.
