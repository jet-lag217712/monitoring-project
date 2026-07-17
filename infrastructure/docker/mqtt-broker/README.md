# Mosquitto MQTT/TLS Broker (local)

Local Eclipse Mosquitto broker for Phase 2 collector transport testing.

## Prerequisites

- Docker (or Colima)
- OpenSSL
- `mosquitto_passwd` (optional; Docker fallback documented below)

## 1. Generate TLS certificates

```bash
cd infrastructure/docker/mqtt-broker
chmod +x scripts/gen-dev-certs.sh scripts/gen-passwords.sh
./scripts/gen-dev-certs.sh
```

Optional SAN overrides (useful for Azure / GNS3 collector TLS):

```bash
MQTT_SERVER_CN=cloud.lab \
MQTT_SERVER_DNS=mosquitto,cloud.lab \
MQTT_SERVER_IP=20.x.x.x \
  ./scripts/gen-dev-certs.sh
```

Creates gitignored files under `certs/`:

- `ca.crt` — trust this in the collector (`mqtt.tls.ca_file`)
- `server.crt` / `server.key` — presented by Mosquitto

For local/dev stacks, prefer [`deployments/development/up.sh`](../../../deployments/development/up.sh), which generates certs and starts Mosquitto with the Mac cloud stack. Collector on the OrbStack VM: [`deployments/development/vxrail/`](../../../deployments/development/vxrail/). Single-host all-in-one: [`deployments/end-to-end/`](../../../deployments/end-to-end/). Production Mosquitto: [`deployments/production/cloud/`](../../../deployments/production/cloud/).

## 2. Create broker users

```bash
./scripts/gen-passwords.sh
# prompts for the collector password
# ingestion password defaults to "ingestion" (override with MQTT_INGESTION_PASSWORD)
```

Docker-only alternative (no local `mosquitto_passwd`):

```bash
docker run --rm -it -v "$PWD:/work" eclipse-mosquitto:2 \
  mosquitto_passwd -c /work/passwords collector
docker run --rm -it -v "$PWD:/work" eclipse-mosquitto:2 \
  mosquitto_passwd /work/passwords ingestion
```

## 3. Start the broker

```bash
docker compose up --build -d
```

Broker listens on `tls://127.0.0.1:8883`.

## 4. Point the collector at Mosquitto

```bash
export MQTT_PASSWORD='<collector password from step 2>'
# from this directory (mqtt-broker), collector is three levels up:
cd ../../../services/snmp-collector
go run ./cmd/collector -config configs/collector.mqtt.example.yaml
```

## ACL model

| User | Permission | Topic |
|------|------------|-------|
| `collector` | write | `site/+/device/+/metric/#` |
| `collector` | write | `site/+/device/+/telemetry/v2/#` |
| `collector` | write | `site/+/collector/+/telemetry/v2/heartbeat` |
| `ingestion` | read | `site/+/device/+/metric/#` |
| `ingestion` | read | `site/+/device/+/telemetry/v2/#` |
| `ingestion` | read | `site/+/collector/+/telemetry/v2/heartbeat` |

Anonymous access is disabled. TLS is required on port 8883.

## Credential rotation (local)

1. Re-run `mosquitto_passwd` for the affected user.
2. Update `MQTT_PASSWORD` (or ingestion secret) on clients.
3. Restart Mosquitto (`docker compose restart`) and collectors.

## Production notes

- Do not reuse these self-signed certs.
- Issue per-site collector credentials; never commit passwords or private keys.
- Prefer a real CA and secret store (vault / env injection) in Azure deployments (Phase 7).

## Smoke subscribe

```bash
mosquitto_sub -h 127.0.0.1 -p 8883 \
  --cafile certs/ca.crt \
  -u ingestion -P ingestion \
  -t 'site/+/device/+/metric/#' -t 'site/+/device/+/telemetry/v2/#' \
  -t 'site/+/collector/+/telemetry/v2/heartbeat' -v
```
