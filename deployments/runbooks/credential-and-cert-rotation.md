# Credential and certificate rotation

Never commit community strings, MQTT passwords, private keys, or TLS material.

## SNMP communities

1. Update the secret store / `.env` value for the device `community_env` name.
2. Restart or recreate the collector container/unit so the new environment is loaded.
3. Confirm polls succeed (`/readyz`, TUI device view, API device status).
4. Do not print communities in logs, audit files, MQTT payloads, or runbooks.

Static inventory keeps `community_env` references only.

## MQTT passwords

1. Update Mosquitto password file / secret injection for `collector` and `ingestion`.
2. Update matching `MQTT_PASSWORD` / profile `.env` values on cloud and collector hosts.
3. Rolling restart: Mosquitto → ingestion → collector.
4. Run v2 smoke: `./deployments/lib/smoke_mqtt_v2_to_api.sh` (with required env).

Local broker notes: [`infrastructure/docker/mqtt-broker/README.md`](../../infrastructure/docker/mqtt-broker/README.md).

## TLS CA / server certificates

1. Issue a new server certificate with SANs matching collector `MQTT_BROKER` hostnames.
2. Distribute the **public CA** (`ca.crt`) to collector and ingestion mounts.
3. Restart Mosquitto, then clients.
4. Never place private keys under tracked `certs/` directories intended for CA-only mounts.
