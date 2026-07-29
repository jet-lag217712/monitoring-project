# Rollback and restore

## Appliance release rollback

1. Record the active release version, image digests, and configuration revision.
2. Stop the stack without deleting volumes.
3. Select the previous immutable release and run its Compose stack.
4. Verify `/healthz`, `/readyz`, the TUI configuration view, and dashboard data.

Never remove the PostgreSQL or collector-state volumes during a release
rollback.

## TUI-managed configuration rollback

Invalid managed edits do not activate. To undo a valid but unwanted change:

1. Restore the known-good managed inventory with mode `0600`.
2. Use the local TUI reload action or send `SIGHUP` to the collector.
3. Confirm the active revision and inventory in the TUI.

Static deployment YAML is never written by the TUI and should be restored from
the matching release source when necessary.

## PostgreSQL restore

1. Stop ingestion and other database writers.
2. Restore the approved local `pg_dump` or volume backup.
3. Apply forward migrations only when the restored database is behind the
   release schema.
4. Start PostgreSQL, ingestion, API, and collectors in that order.
5. Confirm current projections, observation-time ordering, and dashboard data.

Use [`infrastructure/script/migrate.sh`](../../infrastructure/script/migrate.sh)
for migrations. Document the release manifest, backup identifier, and any lost
telemetry before closing the incident.
