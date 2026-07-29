# Development VxRail-like collector fixture

This is a local Linux-VM fixture for testing multiple site collectors against a
GNS3 network. The supported customer product is the complete local appliance;
this directory intentionally omits the appliance services and should not be
used as a production installation.

## Start

From the repository root, sync the fixture to the test VM:

```bash
./deployments/development/vxrail/sync.sh --dry-run
./deployments/development/vxrail/sync.sh
```

On the VM:

```bash
cp .env.example .env
./bootstrap.sh
```

First boot creates shared local settings, the site manifest, per-site
configuration, managed inventory directories, and generated Compose files.

## TUI operations

Each site has an owner-only control socket. Run the TUI from the VM or from
inside its container:

```bash
collector tui -socket ./sites/site-001/run/control.sock -theme auto
docker compose -f docker-compose.yml -f docker-compose.sites.generated.yml \
  exec -it snmp-collector-site-001 \
  /collector tui -socket /run/snmp-collector/control.sock -theme auto
```

The TUI covers inventory, device details, discovery review, thresholds,
transport state, and configuration reload. It writes only the managed
inventory. Static YAML is read-only.

## Validation

```bash
./validate.sh
curl -fsS http://127.0.0.1:19090/healthz
```

Admin ports are local test ports. Control sockets are never published as TCP.
