# Install and validate the local appliance

The supported customer installation is the complete appliance in
[`../production/appliance/`](../production/appliance/). It runs on a
VMware-compatible VxRail-type VM and keeps collectors, MQTT, ingestion,
PostgreSQL, API, and dashboard local.

## Before first boot

- Verify the OVA checksum and architecture.
- Assign one virtual NIC and a route to every monitored SNMP network.
- Confirm the guest has the target CPU, memory, and disk capacity.
- Stage the release bundle offline; no runtime Internet access is required.
- Do not pre-create users, passwords, SNMP communities, or service keys.

## First boot

The console launches the setup TUI. Complete these steps in order:

1. Create the initial administrator and any additional local appliance users.
2. Set the appliance identity and local dashboard access.
3. Create named sites and their collector services.
4. Configure SNMP inventory and discovery allowlists.
5. Review discovery candidates and explicitly accept approved devices.
6. Set temperature thresholds, upstream dependencies, and interface filters.
7. Save, validate, reload, and start the generated Compose stack.

The TUI writes managed inventory and generated site artifacts. Static YAML is
read-only. A failed validation or reload leaves the last active snapshot in
place.

## Validation

```bash
./deployments/test.sh --quick
docker compose -f deployments/production/appliance/docker-compose.yml config
curl -fsS http://127.0.0.1:9090/healthz
curl -fsS http://127.0.0.1:9090/readyz
```

Run the collector checks from the appliance VM or its site container. Service
administration ports remain private to the VM; the dashboard is verified on
the published HTTPS endpoint.

## Acceptance

```bash
/usr/local/lib/equate/verify-appliance.sh
/usr/local/lib/equate/verify-ova-import.sh --configured
```

Confirm dashboard login with two local PAM-backed users, configure at least two
sites, review discovery before enrollment, view live device telemetry, reboot,
and verify that state and TUI-managed configuration persist.

For source-only testing use [`../end-to-end/`](../end-to-end/) and
`./deployments/test.sh --quick`.
