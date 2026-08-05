## appliance - 5

### Primary Service

Deployments (On-Prem Appliance)

### Secondary Services

- SNMP Collector (`equate` CLI)
- Backend API / PostgreSQL (site row cleanup)

### Choice Made

Add day-2 CLI paths that avoid the full setup wizard:

- `equate configure --temperature <celsius>` applies the global temperature
  warning threshold to every site collector via the existing control-plane
  prepare → commit → reload path.
- `equate sites delete <site-id> [--yes]` removes one site: stop/remove the
  collector and volume, rewrite manifest + generated compose, delete host
  artifacts, delete Postgres rows (no FK cascade), then reconcile the stack
  and re-sync topology. Bare `equate sites` continues to list.

### Alternatives Considered

- Requiring operators to skip through `equate configure --sites` to change
  thresholds or shrink the site list — rejected; easy to mutate CIDRs/inventory
  by mistake.
- Top-level `equate delete --site` — rejected in favor of `equate sites delete`
  next to `equate sites list`.
- Relying on `sync-site-topology.sh` alone for site removal — rejected; it only
  upserts and leaves orphaned site/device rows.

### Trade-offs

Operators gain safe, single-purpose commands. Site delete is destructive and
requires confirmation (or `--yes`). Temperature apply fails fast if any site
collector control socket is unreachable. Sudoers must include `equate sites *`.

### Status

Accepted
