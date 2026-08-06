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
  collector and volume, delete Postgres rows (no FK cascade), rewrite
  manifest + generated compose, delete host artifacts, then reconcile the
  stack and re-sync topology. Bare `equate sites` continues to list.
- When the site is already absent from the manifest (configure shrink or a
  prior partial delete), `equate sites delete` still cleans leftover
  collector artifacts and database rows so the dashboard cannot keep a
  ghost site.
- `scripts/sync-site-topology.sh` upserts configured sites and prunes
  `sites` rows (plus dependents) whose names are not in the manifest so
  the frontend site list stays aligned with `equate sites`.

### Alternatives Considered

- Requiring operators to skip through `equate configure --sites` to change
  thresholds or shrink the site list — rejected; easy to mutate CIDRs/inventory
  by mistake.
- Top-level `equate delete --site` — rejected in favor of `equate sites delete`
  next to `equate sites list`.
- Relying on upsert-only topology sync for intentional site removal —
  rejected; operators need an explicit destructive command with confirmation.
  Upsert-only sync also left orphaned site/device rows after configure shrink,
  which the dashboard continued to list.

### Trade-offs

Operators gain safe, single-purpose commands. Site delete is destructive and
requires confirmation (or `--yes`). Temperature apply fails fast if any site
collector control socket is unreachable. Sudoers must include `equate sites *`.
Topology sync pruning is irreversible for rows not named in the manifest;
manifest remains the source of truth for configured collectors.

### Status

Accepted
