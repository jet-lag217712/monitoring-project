# Dev cloud plane (Azure) — future / production path

This directory is **documentation only**. There is no Docker Compose here.

**Day-to-day testing** uses the Mac Compose stack ([`../../local/`](../../local/)) plus [`../../local/vxrail/`](../../local/vxrail/) on the Debian VM. Use this Azure runbook when you are ready to host the cloud plane in Azure.

The Azure cloud plane:

1. **PostgreSQL** — Azure Database for PostgreSQL Flexible Server (existing Terraform)
2. **One Azure Linux VM** — Mosquitto, ingestion, backend-api, and frontend as containers

The GNS3 collector ([`../vxrail/`](../vxrail/)) egresses MQTT/TLS to this Mosquitto endpoint (not Mac Mosquitto).

```text
GNS3 collector ──MQTT/TLS :8883──▶ Azure Linux VM (mosquitto + apps)
                                         │
                                         ▼
                               Azure Flexible Server (Postgres)
```

## Prerequisites

- Azure subscription; `az login` (run only with explicit approval in agent sessions)
- Terraform >= 1.5
- Ability to create a Linux VM and NSG rules
- This repo cloned on a machine that can build images (or build in CI and pull on the VM)

## 1. Provision PostgreSQL

Follow [`infrastructure/terraform/README.md`](../../../infrastructure/terraform/README.md).

```bash
cd infrastructure/terraform/environments/dev
cp terraform.tfvars.example terraform.tfvars
# Edit server_name, password, firewall rules (allow the Azure VM public IP)

terraform init
terraform plan
terraform apply
```

Migrate and bootstrap roles (from a host that can reach the Flexible Server):

```bash
export DATABASE_URL='postgres://ogsd_admin:<password>@<fqdn>:5432/ogsd?sslmode=require'
./infrastructure/script/migrate.sh up
OGSD_INGESTION_PASSWORD='...' OGSD_API_PASSWORD='...' \
  ./infrastructure/script/bootstrap-db-roles.sh
```

Note the FQDN and app role passwords for the VM containers below.

## 2. Create one Azure Linux VM

Suggested baseline:

- Ubuntu 22.04/24.04 LTS
- Size enough for four containers (e.g. `Standard_B2s` or larger)
- Public IP if the GNS3 lab reaches Azure over the internet; otherwise private IP + VPN/ExpressRoute

### NSG / firewall

| Direction | Port | Purpose |
|-----------|------|---------|
| Inbound | TCP `8883` | Mosquitto from VxRail/collector source IPs only |
| Inbound | TCP `80` (and/or `443`) | Browser UI |
| Inbound | TCP `22` | SSH (restrict to your admin IPs) |
| Outbound | TCP `5432` | Flexible Server (if not using private endpoint) |

Do **not** require inbound access from Azure into the GNS3/customer network.

Install Docker Engine + Compose plugin on the VM.

## 3. TLS certificates for Mosquitto

On a trusted machine (or the VM), generate or install server certs whose SAN matches the hostname/IP collectors will use in `MQTT_BROKER`.

Lab-only helper (not for production):

```bash
cd infrastructure/docker/mqtt-broker
MQTT_SERVER_CN=<azure-dns-or-ip> \
MQTT_SERVER_IP=<azure-public-ip> \
MQTT_SERVER_DNS=<optional-dns-name> \
  ./scripts/gen-dev-certs.sh
./scripts/gen-passwords.sh   # collector + ingestion users
```

Copy `certs/` and `passwords` to the VM (e.g. `/opt/ogsd/mqtt/`). Distribute `ca.crt` to the VxRail host as `deployments/dev/vxrail/certs/ca.crt`.

Prefer a real public CA or your org PKI for anything beyond a private lab.

## 4. Run app containers on the VM

Build images from the repo (on the VM or push to a registry):

| Image | Build context |
|-------|----------------|
| Mosquitto | `infrastructure/docker/mqtt-broker` |
| Ingestion | `services/ingestion-service` |
| Backend API | `services/backend-api` |
| Frontend | `frontend` (`VITE_API_BASE_URL=http://<azure-public-ip>` or your DNS) |

Example run pattern (adjust paths, passwords, and FQDN). Use a Docker network so containers can resolve each other by name; publish Mosquitto and the UI on the host.

```bash
docker network create ogsd-cloud

# Mosquitto — listen on all interfaces
docker run -d --name mosquitto --network ogsd-cloud --restart unless-stopped \
  -p 8883:8883 \
  -v /opt/ogsd/mqtt/certs:/mosquitto/certs:ro \
  -v /opt/ogsd/mqtt/passwords:/mosquitto/config/passwords:ro \
  -v mosquitto-data:/mosquitto/data \
  ogsd-mosquitto:local

# Ingestion — MQTT to mosquitto; DB to Flexible Server
docker run -d --name ingestion --network ogsd-cloud --restart unless-stopped \
  -p 9091:9091 \
  -e MQTT_BROKER=tls://mosquitto:8883 \
  -e MQTT_PASSWORD=<ingestion-mqtt-password> \
  -e DATABASE_URL='postgres://ogsd_ingestion:<password>@<pg-fqdn>:5432/ogsd?sslmode=require' \
  -v /opt/ogsd/configs/ingestion.yaml:/configs/ingestion.yaml:ro \
  -v /opt/ogsd/mqtt/certs/ca.crt:/certs/ca.crt:ro \
  ogsd-ingestion:local -config /configs/ingestion.yaml

# Backend API
docker run -d --name backend-api --network ogsd-cloud --restart unless-stopped \
  -p 8000:8000 -p 9092:9092 \
  -e DATABASE_URL='postgres://ogsd_api:<password>@<pg-fqdn>:5432/ogsd?sslmode=require' \
  -e GOOGLE_CLIENT_ID= \
  -v /opt/ogsd/configs/api.yaml:/configs/api.yaml:ro \
  ogsd-api:local -config /configs/api.yaml

# Frontend — nginx should proxy /api to backend-api:8000 (see deployments/local/nginx-frontend.conf)
docker run -d --name frontend --network ogsd-cloud --restart unless-stopped \
  -p 80:80 \
  -v /opt/ogsd/nginx-frontend.conf:/etc/nginx/conf.d/default.conf:ro \
  ogsd-frontend:local
```

### Config files on the VM

Mount YAML similar to [`deployments/local/configs/`](../../local/configs/):

- **ingestion.yaml** — `mqtt.tls.ca_file: /certs/ca.crt`; broker overridden by `MQTT_BROKER`
- **api.yaml** — set `cors_origins` to `http://<azure-public-ip>` (and HTTPS origin if used); lab may leave `auth.enabled: false`
- **nginx-frontend.conf** — `proxy_pass http://backend-api:8000` for `/api/`

### Environment summary

| Variable | Where | Purpose |
|----------|--------|---------|
| `DATABASE_URL` | ingestion, api | Flexible Server DSN (`sslmode=require`) |
| `MQTT_BROKER` | ingestion | `tls://mosquitto:8883` on the VM network |
| `MQTT_PASSWORD` | ingestion | Mosquitto `ingestion` user |
| `GOOGLE_CLIENT_ID` | api | Optional OIDC |
| `VITE_API_BASE_URL` | frontend **build** | Public UI origin (e.g. `http://x.x.x.x`) |

## 5. Startup order

1. Flexible Server up + migrations + roles  
2. Mosquitto on the VM  
3. Ingestion, backend-api, frontend  
4. VxRail collector (`../vxrail/bootstrap.sh`) with `MQTT_BROKER=tls://<azure-host>:8883`

## 6. Wire GNS3 / verify

```bash
# On Azure VM
curl -sf http://127.0.0.1:9091/healthz
curl -sf http://127.0.0.1:9092/healthz

# On VxRail host
# MQTT_BROKER=tls://<azure-host>:8883
# certs/ca.crt installed
./deployments/dev/vxrail/bootstrap.sh
curl -sf http://127.0.0.1:9090/healthz
```

Open `http://<azure-public-ip>/` in a browser.

## Shutdown

Stop containers on the VM (`docker stop ...`). Deprovision VM / Terraform only when intentionally tearing down the environment. Flexible Server data persists until the Terraform destroy.

## Gaps (explicit)

- No Terraform yet for the Linux VM, NSG, Mosquitto, or app containers
- Dev Mosquitto certs from `gen-dev-certs.sh` are lab-only
- Prefer a container registry and systemd/restart policy for anything shared beyond a personal lab
