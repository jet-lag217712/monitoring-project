## collector - 10

### Primary Service

SNMP Collector

### Secondary Services

- Deployments (development / VxRail lab)

### Choice Made

Extend the development VxRail first-boot bootstrap to provision **N isolated collector
site containers** from one shared deployment directory, while reusing a single Mosquitto
broker/password and SNMP community configuration.

#### Shared at deploy root

- `.env` with `MQTT_BROKER`, `MQTT_PASSWORD`, `SNMP_COMMUNITY`, `SNMP_DISCOVERY_COMMUNITY`
- Mosquitto CA at `certs/ca.crt`
- Collector image build context

#### Per site (generated)

- `sites/<site-id>/configs/collector.yaml` with unique `site_id`, `collector.id`, `mqtt.client_id`
- `sites/<site-id>/managed/managed-inventory.yaml` with exactly one `discovery.allowed_cidrs`
- `sites/<site-id>/run/control.sock`
- Named Docker volume `collector-state-<site-id>`
- Compose service `snmp-collector-<site-id>` with host admin port `9090 + index - 1`
- `sites/manifest.yaml` and `docker-compose.sites.generated.yml`

#### Setup / bootstrap

- `collector setup` collects shared secrets, site count, and per-site **site id** plus CIDR values.
- Setup writes all generated artifacts, starts every site service, runs discovery/threshold
  per site, and writes `.setup-complete` only after all sites succeed.
- `./bootstrap.sh --reconfigure` clears `.setup-complete` and re-runs the wizard.
- Day-2 operator TUI remains one socket/service per site.

### Alternatives Considered

- Four independent vxrail directories — rejected; duplicates secrets and sync overhead.
- One collector with multiple CIDRs — rejected; it does not model distinct local `site_id` values.
- Writable static YAML from setup — rejected; collector-9 static/managed split preserved.

### Pros

- Exercises multi-site ingestion/dashboard paths in the development lab.
- Preserves collector-7 isolation primitives per process.
- Operators configure site count, per-site site ids, and CIDR scope in the branded setup TUI.

### Cons

- Generated compose/manifest must be regenerated when site topology changes.
- Setup orchestration is more complex than single-site bootstrap.

### Status

Accepted
