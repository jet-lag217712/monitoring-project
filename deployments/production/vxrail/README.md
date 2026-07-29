# VxRail collector fixture

This directory is a collector-only validation fixture for a VxRail-type VM. It
is not a customer deployment model. Customer installations use the complete
local appliance in [`../appliance/`](../appliance/), which keeps MQTT,
ingestion, PostgreSQL, API, dashboard, and collectors on the same VM.

Use this fixture only when testing the collector container, SNMP reachability,
SQLite buffering, or the local TUI independently of the packaged appliance.
The supported operator workflow is:

```bash
cp .env.example .env
# Supply local test secrets; never commit .env.
collector validate -config ./configs/collector.yaml
docker compose up -d --build
collector tui -socket ./run/control.sock -theme auto
```

The control socket is local-only. Configure devices and discovery through the
TUI-managed inventory workflow; do not expose the socket or add a remote
management path.
