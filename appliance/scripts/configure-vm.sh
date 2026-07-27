#!/usr/bin/env bash
# Install a staged appliance release bundle on a Debian 12 VM.
#
# Usage (on the VM as root):
#   bash configure-vm.sh --bundle /tmp/equate-staging/bundle --version 1.0.0
set -euo pipefail

BUNDLE_DIR=""
VERSION=""
EQUATE_ROOT="/opt/equate"
ETC_DIR="/etc/equate"
VAR_DIR="/var/lib/equate"
RUN_DIR="/run/equate"
APPLIANCE_GROUP="equate-appliance"
# eclipse-mosquitto:2 runs as this UID and must read mounted TLS/password files.
MOSQUITTO_UID=1883

usage() {
  cat <<'EOF'
usage: configure-vm.sh --bundle <path> --version <semver>

Installs the immutable release under /opt/equate/releases/<version>, renders
installation secrets under /run/equate, and starts the production Compose stack.
EOF
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --bundle)
      BUNDLE_DIR="${2:-}"
      shift 2
      ;;
    --version)
      VERSION="${2:-}"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      usage >&2
      exit 1
      ;;
  esac
done

if [[ "$(id -u)" -ne 0 ]]; then
  echo "configure-vm.sh must run as root" >&2
  exit 1
fi

if [[ -z "${BUNDLE_DIR}" || -z "${VERSION}" ]]; then
  usage >&2
  exit 1
fi

if [[ ! -f "${BUNDLE_DIR}/release.env" ]]; then
  echo "bundle missing release.env: ${BUNDLE_DIR}" >&2
  exit 1
fi

# Callers may delete the release directory while cd'd into it; use a stable cwd.
cd /

require_cmd() {
  local cmd
  for cmd in "$@"; do
    if ! command -v "${cmd}" >/dev/null 2>&1; then
      echo "required command not found: ${cmd}" >&2
      exit 1
    fi
  done
}

require_cmd docker openssl install

if ! docker compose version >/dev/null 2>&1; then
  echo "docker compose plugin is required" >&2
  exit 1
fi

install_host_packages() {
  if command -v apt-get >/dev/null 2>&1; then
    echo "installing host packages (pamtester, python3)..."
    export DEBIAN_FRONTEND=noninteractive
    apt-get update -qq
    apt-get install -y -qq pamtester python3
  else
    require_cmd pamtester python3
  fi
}

install_host_packages

rand_secret() {
  openssl rand -base64 32 | tr -d '/+=' | head -c 32
}

RELEASE_DIR="${EQUATE_ROOT}/releases/${VERSION}"
CURRENT_LINK="${EQUATE_ROOT}/current"

echo "installing release ${VERSION} to ${RELEASE_DIR}"
install -d -m 0755 "${EQUATE_ROOT}/releases"
rm -rf "${RELEASE_DIR}"
cp -a "${BUNDLE_DIR}" "${RELEASE_DIR}"
ln -sfn "${RELEASE_DIR}" "${CURRENT_LINK}"

# Regenerate ephemeral secrets each install; remove stale dirs (e.g. mqtt/passwords as a folder).
rm -rf "${RUN_DIR}/rendered"

install -d -m 0750 "${ETC_DIR}/configs" "${ETC_DIR}/sites"
install -d -m 0750 "${VAR_DIR}/postgres" "${VAR_DIR}/mosquitto"
install -d -m 0755 "${RUN_DIR}/rendered/mqtt" "${RUN_DIR}/rendered/mqtt/certs" "${RUN_DIR}/rendered/certificates"

if [[ ! -f "${ETC_DIR}/configs/api.yaml" ]]; then
  cp "${RELEASE_DIR}/configs/api.yaml" "${ETC_DIR}/configs/api.yaml"
  cp "${RELEASE_DIR}/configs/ingestion.yaml" "${ETC_DIR}/configs/ingestion.yaml"
  cp "${RELEASE_DIR}/configs/collector.yaml" "${ETC_DIR}/configs/collector.yaml"
fi

POSTGRES_SUPERUSER="ogsd"
POSTGRES_DB="ogsd"
POSTGRES_PASSWORD="$(rand_secret)"
INGESTION_DB_PASSWORD="$(rand_secret)"
API_DB_PASSWORD="$(rand_secret)"
MQTT_COLLECTOR_PASSWORD="$(rand_secret)"
MQTT_INGESTION_PASSWORD="$(rand_secret)"

cat >"${RUN_DIR}/rendered/postgres.env" <<EOF
POSTGRES_USER=${POSTGRES_SUPERUSER}
POSTGRES_PASSWORD=${POSTGRES_PASSWORD}
POSTGRES_DB=${POSTGRES_DB}
EOF
chmod 0600 "${RUN_DIR}/rendered/postgres.env"

ADMIN_DATABASE_URL="postgres://${POSTGRES_SUPERUSER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable"
cat >"${RUN_DIR}/rendered/database-admin.env" <<EOF
DATABASE_URL=${ADMIN_DATABASE_URL}
EOF
chmod 0600 "${RUN_DIR}/rendered/database-admin.env"

cat >"${RUN_DIR}/rendered/ingestion.env" <<EOF
MQTT_PASSWORD=${MQTT_INGESTION_PASSWORD}
DATABASE_URL=postgres://ogsd_ingestion:${INGESTION_DB_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable
EOF
chmod 0600 "${RUN_DIR}/rendered/ingestion.env"

cat >"${RUN_DIR}/rendered/backend-api.env" <<EOF
DATABASE_URL=postgres://ogsd_api:${API_DB_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable
EOF
chmod 0600 "${RUN_DIR}/rendered/backend-api.env"

cat >"${RUN_DIR}/rendered/collector.env" <<EOF
MQTT_COLLECTOR_PASSWORD=${MQTT_COLLECTOR_PASSWORD}
SNMP_COMMUNITY=CHANGE_ME
SNMP_DISCOVERY_COMMUNITY=CHANGE_ME
EOF
chmod 0600 "${RUN_DIR}/rendered/collector.env"

render_mqtt_tls() {
  local certs="${RUN_DIR}/rendered/mqtt/certs"
  local cn="equate-appliance"
  openssl genrsa -out "${certs}/ca.key" 4096
  openssl req -x509 -new -nodes -key "${certs}/ca.key" -sha256 -days 825 \
    -subj "/CN=Equate Appliance CA" -out "${certs}/ca.crt"
  openssl genrsa -out "${certs}/server.key" 2048
  openssl req -new -key "${certs}/server.key" -subj "/CN=${cn}" -out "${certs}/server.csr"
  cat >"${certs}/server.ext" <<EXT
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${cn}
DNS.2 = mosquitto
DNS.3 = localhost
IP.1 = 127.0.0.1
EXT
  openssl x509 -req -in "${certs}/server.csr" -CA "${certs}/ca.crt" -CAkey "${certs}/ca.key" \
    -CAcreateserial -out "${certs}/server.crt" -days 825 -sha256 -extfile "${certs}/server.ext"
  rm -f "${certs}/server.csr" "${certs}/server.ext" "${certs}/ca.srl"
  chmod 600 "${certs}/ca.key"
  chmod 644 "${certs}/ca.crt" "${certs}/server.crt"
  chown "${MOSQUITTO_UID}:${MOSQUITTO_UID}" "${certs}/ca.crt" "${certs}/server.crt" "${certs}/server.key"
  chmod 640 "${certs}/server.key"
}

fix_mqtt_runtime_permissions() {
  local mqtt_dir="${RUN_DIR}/rendered/mqtt"
  local certs="${mqtt_dir}/certs"
  local passwords="${mqtt_dir}/passwords"
  chmod 755 "${mqtt_dir}" "${certs}"
  if [[ -f "${certs}/server.key" ]]; then
    chown "${MOSQUITTO_UID}:${MOSQUITTO_UID}" "${certs}/ca.crt" "${certs}/server.crt" "${certs}/server.key"
    chmod 644 "${certs}/ca.crt" "${certs}/server.crt"
    chmod 640 "${certs}/server.key"
    chmod 600 "${certs}/ca.key"
    chown root:root "${certs}/ca.key"
  fi
  if [[ -f "${passwords}" ]]; then
    chown "${MOSQUITTO_UID}:${MOSQUITTO_UID}" "${passwords}"
    chmod 640 "${passwords}"
  fi
}

render_ui_tls() {
  local certs="${RUN_DIR}/rendered/certificates"
  local cn="equate-appliance"
  openssl genrsa -out "${certs}/tls.key" 2048
  openssl req -x509 -new -nodes -key "${certs}/tls.key" -sha256 -days 825 \
    -subj "/CN=${cn}" -out "${certs}/tls.crt"
  chmod 600 "${certs}/tls.key"
}

render_mqtt_passwords() {
  local out="${RUN_DIR}/rendered/mqtt/passwords"
  install -d -m 0700 "$(dirname "${out}")"
  if [[ -d "${out}" ]]; then
    rm -rf "${out}"
  fi
  if command -v mosquitto_passwd >/dev/null 2>&1; then
    mosquitto_passwd -b -c "${out}" collector "${MQTT_COLLECTOR_PASSWORD}"
    mosquitto_passwd -b "${out}" ingestion "${MQTT_INGESTION_PASSWORD}"
  else
    docker run --rm -v "${RUN_DIR}/rendered/mqtt:/work" eclipse-mosquitto:2 \
      sh -c "mosquitto_passwd -b -c /work/passwords collector '${MQTT_COLLECTOR_PASSWORD}' && mosquitto_passwd -b /work/passwords ingestion '${MQTT_INGESTION_PASSWORD}'"
  fi
  chmod 640 "${out}"
  chown "${MOSQUITTO_UID}:${MOSQUITTO_UID}" "${out}"
}

render_mqtt_tls
render_ui_tls
render_mqtt_passwords
fix_mqtt_runtime_permissions

getent group "${APPLIANCE_GROUP}" >/dev/null 2>&1 || groupadd --system "${APPLIANCE_GROUP}"

echo "loading container images..."
for image_tar in "${RELEASE_DIR}/images/"*.tar; do
  docker load -i "${image_tar}"
done

install -m 0644 "${RELEASE_DIR}/scripts/equate-auth-broker.service" /etc/systemd/system/equate-auth-broker.service
chmod 0755 "${RELEASE_DIR}/scripts/auth-broker.sh"
systemctl daemon-reload
if ! systemctl enable --now equate-auth-broker.service; then
  echo "equate-auth-broker failed to start; check: journalctl -xeu equate-auth-broker.service" >&2
  exit 1
fi

COMPOSE_ENV="${RUN_DIR}/rendered/compose.env"
cat "${RELEASE_DIR}/release.env" >"${COMPOSE_ENV}"
# shellcheck disable=SC1091
source "${RUN_DIR}/rendered/postgres.env"
{
  echo "POSTGRES_USER=${POSTGRES_USER}"
  echo "POSTGRES_PASSWORD=${POSTGRES_PASSWORD}"
  echo "POSTGRES_DB=${POSTGRES_DB}"
  echo "OGSD_INGESTION_USER=ogsd_ingestion"
  echo "OGSD_INGESTION_PASSWORD=${INGESTION_DB_PASSWORD}"
  echo "OGSD_API_USER=ogsd_api"
  echo "OGSD_API_PASSWORD=${API_DB_PASSWORD}"
  echo "MQTT_INGESTION_PASSWORD=${MQTT_INGESTION_PASSWORD}"
  echo "MQTT_COLLECTOR_PASSWORD=${MQTT_COLLECTOR_PASSWORD}"
  echo "MQTT_PASSWORD=${MQTT_COLLECTOR_PASSWORD}"
  echo "MQTT_BROKER=tls://mosquitto:8883"
  echo "SNMP_COMMUNITY=CHANGE_ME"
  echo "SNMP_DISCOVERY_COMMUNITY=CHANGE_ME"
} >>"${COMPOSE_ENV}"
chmod 0600 "${COMPOSE_ENV}"

compose() {
  (
    cd "${RELEASE_DIR}"
    docker compose \
      --env-file "${COMPOSE_ENV}" \
      -f docker-compose.yml \
      -f docker-compose.sites.generated.yml \
      "$@"
  )
}

echo "resetting postgres data for fresh install..."
docker compose -p equate-appliance down 2>/dev/null || true
rm -rf "${VAR_DIR}/postgres"
install -d -m 0750 "${VAR_DIR}/postgres"

echo "starting core stack..."
compose up -d postgres mosquitto
for _ in $(seq 1 60); do
  if compose exec -T postgres pg_isready -U "${POSTGRES_SUPERUSER}" -d "${POSTGRES_DB}" >/dev/null 2>&1; then
    break
  fi
  sleep 2
done

echo "running database migrations..."
compose run --rm migrate

sql_escape() {
  printf "%s" "${1//\'/\'\'}"
}

bootstrap_role_passwords() {
  compose exec -T postgres psql -U "${POSTGRES_SUPERUSER}" -d "${POSTGRES_DB}" -v ON_ERROR_STOP=1 \
    -c "ALTER ROLE ogsd_ingestion WITH PASSWORD '$(sql_escape "${INGESTION_DB_PASSWORD}")';" \
    -c "ALTER ROLE ogsd_api WITH PASSWORD '$(sql_escape "${API_DB_PASSWORD}")';"
}

bootstrap_role_passwords

compose up -d --remove-orphans

cat >"${RUN_DIR}/rendered/installation.json" <<EOF
{"version":"${VERSION}","installed_at":"$(date -u +"%Y-%m-%dT%H:%M:%SZ")"}
EOF
chmod 0600 "${RUN_DIR}/rendered/installation.json"

if [[ -x "${RELEASE_DIR}/bin/equate" ]]; then
  install -m 0755 "${RELEASE_DIR}/bin/equate" /usr/local/bin/equate
  install -m 0755 "${RELEASE_DIR}/bin/collector" /usr/local/bin/collector
fi
install -d -m 0755 /etc/equate /opt/equate/scripts "${RELEASE_DIR}/scripts"
printf '%s\n' "${RELEASE_DIR}" > /etc/equate/deploy-dir
chmod 0644 /etc/equate/deploy-dir
if [[ -f "${RELEASE_DIR}/scripts/manage-users.sh" ]]; then
  install -m 0755 "${RELEASE_DIR}/scripts/manage-users.sh" /opt/equate/scripts/manage-users.sh
fi

cat <<EOF

Appliance release ${VERSION} installed.

  Release: ${RELEASE_DIR}
  Config:  ${ETC_DIR}
  State:   ${VAR_DIR}
  Secrets: ${RUN_DIR}/rendered (ephemeral; back up /etc/equate and ${VAR_DIR})

Next steps:
  1. Run first-boot setup: sudo equate configure
  2. Create appliance users and configure sites in the setup wizard.
  3. Open https://<vm-ip>/ and sign in with a local appliance user.
  4. Replace self-signed TLS under ${RUN_DIR}/rendered/certificates when ready.
EOF
