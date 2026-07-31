## appliance - 4

### Primary Service

Deployments (On-Prem Appliance)

### Secondary Services

- SNMP Collector (`equate` CLI)
- Azure Blob Storage (static update channel hosting)

### Choice Made

Add an optional **connected update channel** for appliances that can reach an
HTTPS update host. Release engineering publishes signed `.eqa` packages (gzip'd
tars of the existing offline bundle) and a static channel `manifest.json` to
Azure Blob Storage. `equate upgrade` with no `--bundle` fetches the manifest,
compares versions, downloads the matching artifact, verifies SHA-256 and an
Ed25519 signature against an **embedded** public key, extracts to
`/tmp/equate-staging/bundle`, and delegates apply to the existing
`configure-vm.sh --upgrade` path. Air-gapped sites keep offline staging via
`--bundle` / `stage-release.sh`. Standard and NoAuth editions use separate
channels and never cross-update (`appliance-3`).

### Alternatives Considered

- Rebuild upgrade logic in Go — rejected; shell upgrade/rollback already
  preserves sites, secrets, migrations, and canary rollout.
- Unsigned HTTP downloads — rejected; fail closed on integrity/signature.
- Fetch trust keys from the update server — rejected; trust anchor must ship
  with `equate`.
- Application server for updates — rejected; static Blob + manifest is enough.
- Fold updates into the collector day-2 TUI — rejected; release lifecycle is
  host-level (`equate upgrade`).

### Trade-offs

Connected sites gain one-command patching. Operators must manage signing keys
and Blob ACLs. Air-gap remains supported but requires manual staging. A
compromised signing key could push malicious packages — treat the private key
as a production secret and rotate by shipping a new embedded public key.

### Status

Accepted
