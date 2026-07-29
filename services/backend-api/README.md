# Equate Backend API

The Backend API is the read-only application boundary inside the local
appliance. It reads monitoring state from PostgreSQL and serves stable REST
contracts to nginx and the dashboard. It does not poll SNMP, consume MQTT,
write telemetry, or control collectors.

```text
PostgreSQL → Backend API → nginx → local dashboard
```

## Appliance authentication

Production configuration uses `auth.mode: appliance_local`. The API talks to
the host PAM broker through `/run/equate/auth.sock`; it does not mount
`/etc/shadow` and never logs passwords. Sessions are opaque, revocable, rate
limited, CSRF-protected, and tied to active local appliance users.

## Run locally

Start the local Compose validation stack first, then run the API with the
appliance configuration:

```bash
./deployments/end-to-end/up.sh
cd services/backend-api
go run ./cmd/api -config configs/api.example.yaml
```

The REST listener defaults to `http://127.0.0.1:8000`; administration and
metrics default to `http://127.0.0.1:9092`.

## API resources

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/sites` | Appliance site overview |
| GET | `/api/sites/{siteId}` | Site detail and latest devices |
| GET | `/api/sites/{siteId}/devices` | Devices for a site |
| GET | `/api/devices/{deviceId}` | Device detail |
| GET | `/api/devices/{deviceId}/interfaces` | Interface inventory |
| GET | `/api/devices/{deviceId}/metrics` | Metric history |
| GET | `/api/alerts` | Active alerts |

Status values remain compatible with the frontend: `0` Unknown, `1` Healthy,
`2` Warning, and `3` Critical. The API preserves the collector's reason and
dependency evidence; it does not infer health from missing metrics.

## Boundary rules

- The frontend reaches the API through nginx only.
- The API uses a read-only database account.
- SNMP communities, MQTT credentials, TLS material, filesystem paths, and
  TUI mutation operations are never returned in API responses.
- The API is private to the appliance and is not a remote management service.
