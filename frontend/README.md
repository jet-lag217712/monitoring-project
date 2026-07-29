# Equate Monitoring Dashboard

React 19 + Vite frontend for the local Equate appliance. The browser talks to
nginx, nginx proxies approved requests to the local Backend API, and the API
reads PostgreSQL. The frontend never connects directly to collectors, MQTT, or
PostgreSQL.

## Local development

```bash
cp .env.example .env.local
# Keep VITE_DEMO_ENABLED=true for mock fallback while iterating.
npm install
npm run dev
```

Open the Vite URL. The appliance authentication mode is local and is backed by
the host PAM broker in a production build. Development-only mock mode is
explicitly indicated in the UI and must not be used for an appliance release.

## Environment

| Variable | Purpose |
|---|---|
| `VITE_API_BASE_URL` | API origin; appliance default is `/api` |
| `VITE_AUTH_MODE` | Use `appliance_local` for the packaged appliance |
| `VITE_DEMO_ENABLED` | `true` enables mock fallback for development only |

## Production image

```bash
docker build \
  --build-arg VITE_API_BASE_URL=/api \
  --build-arg VITE_AUTH_MODE=appliance_local \
  --build-arg VITE_DEMO_ENABLED=false \
  -t equate/frontend:local .
```

The production appliance serves the built frontend from nginx over its local
HTTPS endpoint. No external identity provider or remote dashboard is part of
the supported appliance configuration.
