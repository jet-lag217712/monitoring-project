# Production cloud plane — Azure VM (skeleton)

Hosts Mosquitto, PostgreSQL (container or Azure Flexible Server), ingestion, backend API, and frontend.

**No development credentials.** Replace every `CHANGE_ME` / placeholder before deploy.

## Prerequisites (operator-owned)

- Azure Linux VM with Docker Engine + Compose plugin
- Network / NSG allowing:
  - Inbound TCP `8883` from collector source IPs only
  - Inbound TCP `80`/`443` for UI (as required)
  - Inbound TCP `22` from admin IPs
  - Outbound to Postgres if using Flexible Server
- Trusted TLS certificates for Mosquitto (SAN matches collector `MQTT_BROKER`)
- Secret injection for DB and MQTT passwords (do not commit)
- Image distribution strategy (build on CI / pull from registry — TBD)
- Authentication: set `GOOGLE_CLIENT_ID` and enable API auth in config
- Backups for Postgres volumes / Flexible Server
- Monitoring scrape of `/metrics` admin endpoints
- Rollback plan (previous image tags + compose revision)

Terraform is deferred; see `infrastructure/terraform/` only as a future Postgres option.

## Layout

| File | Purpose |
|------|---------|
| [`docker-compose.yml`](docker-compose.yml) | Five cloud services |
| [`.env.example`](.env.example) | Required variables (no secrets committed) |
| [`configs/`](configs/) | Ingestion + API YAML |
| [`nginx-frontend.conf`](nginx-frontend.conf) | Frontend `/api/` proxy |

## Rollout order

1. Provision VM + firewall
2. Place TLS material under `./certs/` (or mount from secret store)
3. Copy `.env.example` → `.env` and fill secrets
4. Start Postgres (and Mosquitto if co-located), then **run migrations + role bootstrap before app services**
5. `docker compose up -d` for ingestion, backend-api, and frontend (or pull pre-built images)
6. Verify health endpoints (below)
7. Start on-site collector ([`../vxrail/`](../vxrail/)) only after migrations succeed

See [`../../runbooks/install-and-validate.md`](../../runbooks/install-and-validate.md).

## Verification

```bash
curl -fsS http://127.0.0.1:9091/healthz   # ingestion
curl -fsS http://127.0.0.1:9092/healthz   # API admin
curl -fsS http://127.0.0.1:8000/api/test-config
curl -fsS http://127.0.0.1/               # frontend
```

From a host that can reach Mosquitto with the production CA, publish a test metric and confirm it appears via the API.

## Backup / restore (notes)

- Postgres container volume `postgres-data`: schedule `pg_dump` / volume snapshots
- Flexible Server: use Azure backup / PITR
- Mosquitto data volume is ephemeral for MQTT; do not treat as SoR
- Keep a copy of compose + configs + image digests for rollback

## Image strategy (placeholder)

Compose currently builds from the repo. Production should switch to pinned registry images, e.g.:

```yaml
image: ${REGISTRY}/ogsd-ingestion:${IMAGE_TAG}
```

Leave build contexts until that pipeline exists.
