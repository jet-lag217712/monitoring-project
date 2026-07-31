# Credential and certificate rotation

Never commit SNMP communities, MQTT passwords, private keys, session secrets,
or appliance TLS material.

## SNMP communities

1. Open the local collector TUI for the affected site.
2. Update the local secret referenced by `community_env`; the inventory keeps
   only the environment-variable name.
3. Restart or recreate the affected collector so it loads the new value.
4. Confirm `/readyz`, the TUI device view, and a successful poll.

Do not print communities in logs, audit files, telemetry, or diagnostics.

## MQTT credentials

1. Generate new local broker credentials for `collector` and/or `ingestion`.
2. Update the appliance runtime secret files.
3. Restart Mosquitto, ingestion, and collectors in that order.
4. Confirm the TUI transport view, `/readyz`, and a v2 smoke or live poll.

See [`infrastructure/docker/mqtt-broker/README.md`](../../infrastructure/docker/mqtt-broker/README.md).

## TLS certificates

1. Generate a new local broker or dashboard certificate with the appliance
   hostname/IP SANs.
2. Install only the public CA where clients need to trust the broker.
3. Keep private keys owner-only under the appliance runtime directory.
4. Restart the affected local services and validate the dashboard and MQTT
   readiness.

## Dashboard users

Use `equate users` or the appliance setup wizard user-management step to
create, delete, disable, enable, or reset local PAM-backed users. Never edit
`/etc/shadow` from a container or place passwords in Compose files.
