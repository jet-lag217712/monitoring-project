# Equate Monitoring Dashboard

React 19 + Vite frontend for the OGSD monitoring MVP.

## Local development

```bash
cp .env.example .env.local
# Set VITE_GOOGLE_CLIENT_ID to your Google OAuth Web Client ID
# Keep VITE_DEMO_ENABLED=true for mock fallback while iterating

npm install
npm run dev
```

Open `http://localhost:5173` (or the Vite URL). Sign in with Google, then the dashboard polls the Backend API.

In Google Cloud Console → Credentials → your OAuth Web client, add:

- Authorized JavaScript origins: `http://localhost:5173`, `http://127.0.0.1:5173`

## Environment

| Variable | Purpose |
|----------|---------|
| `VITE_API_BASE_URL` | Backend API origin (default `http://127.0.0.1:8000`) |
| `VITE_GOOGLE_CLIENT_ID` | Google OAuth Web Client ID (must match API `GOOGLE_CLIENT_ID`) |
| `VITE_DEMO_ENABLED` | `true` allows mock fallback on API failure; use `false` in production |

## Production image

```bash
docker build \
  --build-arg VITE_API_BASE_URL=https://api.example.com \
  --build-arg VITE_GOOGLE_CLIENT_ID="$GOOGLE_CLIENT_ID" \
  --build-arg VITE_DEMO_ENABLED=false \
  -t equate/frontend:dev .
```
