# Install and validate

## Choose a profile

| Profile | Use when |
|---------|----------|
| [`../end-to-end/`](../end-to-end/) | Single-host smoke with all services |
| [`../development/`](../development/) | Mac cloud + OrbStack VM collector |
| [`../production/`](../production/) | Azure cloud + on-site VxRail |

Do not create phase-named stacks.

## Migrations before collector

Always apply database migrations (and role bootstrap) **before** enabling the
collector or expecting API projections:

- `end-to-end/up.sh` and `development/up.sh` do this automatically after Postgres is healthy.
- Production cloud: start Postgres, run `infrastructure/script/migrate.sh` + role bootstrap, then bring up ingestion/API/frontend, then start [`../production/vxrail/`](../production/vxrail/).

## Validate

```bash
./deployments/end-to-end/validate.sh
./deployments/development/validate.sh
./deployments/development/vxrail/validate.sh
./deployments/production/cloud/validate.sh
./deployments/production/vxrail/validate.sh
# or
./deployments/test.sh --quick
```

## Non-root state volumes

Collector images run as UID `65532` (distroless `nonroot`). On first create:

```bash
docker run --rm -v <project>_collector-state:/var/lib/snmp-collector busybox:1.36 \
  chown -R 65532:65532 /var/lib/snmp-collector
chown -R 65532:65532 deployments/*/run deployments/*/vxrail/run 2>/dev/null || true
```

`end-to-end/up.sh` and `development/vxrail/bootstrap.sh` attempt this automatically.

## Smoke

```bash
./deployments/end-to-end/smoke.sh      # v2 MQTT → API
./deployments/development/smoke.sh    # cloud plane only
```

Optional MQTT outage drill:

```bash
./deployments/lib/mqtt_outage_drill.sh deployments/end-to-end
```
