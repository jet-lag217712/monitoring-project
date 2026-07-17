# Rollback and database restore

## Collector image / compose rollback

1. Record the previous image digest or git SHA before upgrade.
2. Pin the prior image tag in Compose (or check out the prior compose revision).
3. `docker compose up -d` without wiping the `collector-state` volume.
4. Verify `/healthz`, `/readyz`, and API device visibility.

## Managed inventory rollback

Invalid overlays do not activate. To undo a bad managed write:

1. Replace `/var/lib/snmp-collector/managed-inventory.yaml` with a known-good copy (`0600`).
2. `config.reload` via TUI/control or `SIGHUP`.
3. Confirm active revision in TUI configuration view / `config.get`.

Static inventory is never written by the TUI; restore it from git if needed.

## PostgreSQL restore

1. Stop writers (ingestion) before restore when possible.
2. Restore from `pg_dump` / volume snapshot / Azure PITR per environment.
3. Re-run migrations only if restoring to an older schema baseline that still needs forward migrate.
4. Start ingestion, then collector.
5. Confirm API projections; delayed heartbeats must not overwrite newer collector status (observation-time ordering).

Migrate helper: [`infrastructure/script/migrate.sh`](../../infrastructure/script/migrate.sh).
