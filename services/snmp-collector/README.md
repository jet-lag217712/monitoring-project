# Equate SNMP Collector

The collector is the local polling and operator-control component of the
Equate appliance. It polls SNMPv2c devices, evaluates local reachability and
temperature health, stores telemetry in a durable SQLite outbox, publishes to
the appliance's local MQTT/TLS broker, and exposes a Bubble Tea TUI through a
Unix socket.

```text
SNMP devices → collector → SQLite outbox → local MQTT/TLS → ingestion
                                ▲
                                └── collector tui (Unix socket)
```

The collector does not host PostgreSQL, the API, or the dashboard. It also does
not expose a public management HTTP endpoint or modify network devices.

## Appliance operation

The supported production runtime is generated under
[`deployments/production/appliance/`](../../deployments/production/appliance/).
First boot starts the setup TUI:

```bash
collector setup -dir deployments/production/appliance -theme auto -profile appliance
```

After a site is running, open its local operator TUI:

```bash
collector tui \
  -socket deployments/production/appliance/sites/site-001/run/control.sock \
  -theme auto
```

The TUI provides inventory, device/interface detail, discovery, thresholds,
transport, and configuration views. Mutations use prepare/confirm/commit,
write a secret-free audit record, and reload only after validation.

## Configuration ownership

- Static YAML is the protected source for identity, SNMP version, community
  environment-variable names, MQTT, buffer, admin, and discovery allowlists.
- The managed inventory is the only file written by the TUI. It contains
  thresholds, upstream dependencies, interface filters, and discovery policy.
- `collector validate -config <path>` validates the complete configuration.
- `SIGHUP`, `systemctl reload`, or the TUI reload action activates a valid
  snapshot. Invalid reloads retain the last active snapshot.

SNMP communities and MQTT passwords are supplied through environment references
and never appear in YAML, telemetry, logs, metrics, or audit entries.

## Local service surfaces

| Surface | Purpose |
|---|---|
| `GET /metrics` | Scrape-only metrics |
| `GET /healthz` | Process liveness |
| `GET /readyz` | Active configuration, buffer, and publisher readiness |
| Unix control socket | Local TUI/status/control protocol |

The default admin listener is `:9090` in the container. The control socket is
owner-only and must never be published as a TCP port.

## Health behavior

- A reachable device is Healthy unless the active temperature policy produces a
  Warning.
- A direct device failure becomes Critical after its configured consecutive
  failure threshold.
- A dependent device is Unknown only when every configured upstream path is
  unavailable; every device continues to be polled independently.
- Recovery returns the device to Healthy or Warning and clears the failure
  condition according to the active policy.

## Build and test

```bash
cd services/snmp-collector
go test ./...
go run ./cmd/collector validate -config configs/collector.docker-test.yaml
```

For a complete local smoke path, stage onto an Equate-Appliance VM
(`make appliance-stage`) and use the GNS3 lab under
[`remote-server/`](../../remote-server/). Operator recovery lives in
[`deployments/runbooks/`](../../deployments/runbooks/).
