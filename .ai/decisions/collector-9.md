## collector - 9

### Primary Service

SNMP Collector

### Secondary Services

- Deployments (Docker / VxRail)

### Choice Made

Expand managed-inventory discovery overlays and add a first-boot `collector setup`
wizard that runs before Docker Compose on customer/lab hosts.

#### Managed discovery

Managed inventory may now set `discovery.allowed_cidrs`, `community_env` (name
only), rate/burst, and optional targets/workers/retries/timeout. Static YAML
remains read-only in Compose (`:ro`). Runtime merges managed discovery onto the
static discovery defaults via `applyManagedPolicy`.

#### Control plane

New control methods orchestrate discovery without auto-enrollment:

- `discovery.policy.prepare` / `discovery.policy.commit`
- `discovery.scan.start`
- `discovery.candidates.list`
- `discovery.accept.prepare` / `discovery.accept.commit`

#### First-boot setup

`collector setup -dir <deploy>` writes `.env` and seed
`managed/managed-inventory.yaml`, starts Compose, waits for the control socket,
runs discovery accept + threshold apply, and writes `.setup-complete`.
`bootstrap.sh` invokes setup when the marker is absent.

### Alternatives Considered

- Writable static `collector.yaml` from the TUI — rejected; git/versioned static
  inventory stays authoritative for identity/MQTT.
- In-container setup only — rejected; secrets and managed seed must exist before
  the collector image starts.

### Pros

- Operators can CIDR-discover into managed inventory without editing static YAML.
- Lab/customer bootstrap is guided and branded.
- collector-6 prepare/commit/audit model is preserved.

### Cons

- Dark/setup flows add UI surface area beyond Phase 6 minimum.
- Docker bind-mount for managed file must exist on host before compose up.

### Status

Accepted
