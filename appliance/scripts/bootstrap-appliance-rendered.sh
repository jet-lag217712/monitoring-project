#!/usr/bin/env bash
# Bootstrap ephemeral rendered secrets and start the core appliance Compose stack.
# Sourced by configure-vm.sh (full install and --bootstrap-only first-boot).
set -euo pipefail

rand_secret() {
  openssl rand -base64 32 | tr -d '/+=' | head -c 32
}

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

sync_appliance_db_role_passwords() {
  local script="${RELEASE_DIR}/scripts/sync-db-role-passwords.sh"
  if [[ ! -f "${script}" ]]; then
    echo "sync_appliance_db_role_passwords: missing ${script}" >&2
    return 1
  fi
  EQUATE_RELEASE_DIR="${RELEASE_DIR}" COMPOSE_ENV="${COMPOSE_ENV}" bash "${script}"
}

bootstrap_appliance_rendered_and_stack() {
  local load_images="${LOAD_IMAGES:-1}"

  if [[ -z "${RELEASE_DIR:-}" || -z "${VERSION:-}" ]]; then
    echo "bootstrap_appliance_rendered_and_stack: RELEASE_DIR and VERSION are required" >&2
    return 1
  fi

  if [[ ! -f "${RELEASE_DIR}/release.env" ]]; then
    echo "release missing release.env: ${RELEASE_DIR}" >&2
    return 1
  fi

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

  render_mqtt_tls
  render_ui_tls
  render_mqtt_passwords
  fix_mqtt_runtime_permissions

  getent group "${APPLIANCE_GROUP}" >/dev/null 2>&1 || groupadd --system "${APPLIANCE_GROUP}"

  if [[ "${load_images}" == "1" ]]; then
    echo "loading container images..."
    for image_tar in "${RELEASE_DIR}/images/"*.tar; do
      docker load -i "${image_tar}"
    done
  fi

  install -m 0644 "${RELEASE_DIR}/scripts/equate-auth-broker.service" /etc/systemd/system/equate-auth-broker.service
  chmod 0755 "${RELEASE_DIR}/scripts/auth-broker.sh"
  systemctl daemon-reload
  if ! systemctl enable --now equate-auth-broker.service; then
    echo "equate-auth-broker failed to start; check: journalctl -xeu equate-auth-broker.service" >&2
    return 1
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

  sync_appliance_db_role_passwords

  compose up -d --remove-orphans

  cat >"${RUN_DIR}/rendered/installation.json" <<EOF
{"version":"${VERSION}","installed_at":"$(date -u +"%Y-%m-%dT%H:%M:%SZ")"}
EOF
  chmod 0600 "${RUN_DIR}/rendered/installation.json"
}

compose_from_release() {
  local release_dir="$1"
  shift
  (
    cd "${release_dir}"
    docker compose \
      --env-file "${COMPOSE_ENV}" \
      -f docker-compose.yml \
      -f docker-compose.sites.generated.yml \
      "$@"
  )
}

load_release_images() {
  local release_dir="$1"
  echo "loading container images..."
  for image_tar in "${release_dir}/images/"*.tar; do
    docker load -i "${image_tar}"
  done
}

copy_customer_artifacts() {
  local old_release="$1"
  local new_release="$2"
  local artifact
  for artifact in sites .env docker-compose.sites.generated.yml .setup-complete; do
    if [[ -e "${old_release}/${artifact}" ]]; then
      rm -rf "${new_release}/${artifact}"
      cp -a "${old_release}/${artifact}" "${new_release}/${artifact}"
    fi
  done
}

merge_compose_env_for_upgrade() {
  local old_env="$1"
  local new_release_env="$2"
  local out="$3"
  python3 - "${old_env}" "${new_release_env}" "${out}" <<'PY'
import sys

old_path, new_path, out_path = sys.argv[1:4]
update_keys = {"COMPOSE_PROJECT_NAME"}
new_values = {}
with open(new_path, encoding="utf-8") as handle:
    for raw in handle:
        line = raw.rstrip("\n")
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        if key.startswith("EQUATE_") or key == "COMPOSE_PROJECT_NAME":
            update_keys.add(key)
            new_values[key] = value

merged = []
seen = set()
with open(old_path, encoding="utf-8") as handle:
    for raw in handle:
        line = raw.rstrip("\n")
        if not line or line.startswith("#") or "=" not in line:
            merged.append(line)
            continue
        key, value = line.split("=", 1)
        if key in update_keys and key in new_values:
            value = new_values[key]
        merged.append(f"{key}={value}")
        seen.add(key)

for key in sorted(update_keys):
    if key not in seen and key in new_values:
        merged.append(f"{key}={new_values[key]}")

with open(out_path, "w", encoding="utf-8") as handle:
    handle.write("\n".join(merged) + "\n")
PY
  chmod 0600 "${out}"
}

backup_compose_env() {
  local version_label="$1"
  local backup_dir="${ETC_DIR}/upgrade-backup"
  install -d -m 0750 "${backup_dir}"
  cp -a "${COMPOSE_ENV}" "${backup_dir}/compose.env.${version_label}"
  chmod 0600 "${backup_dir}/compose.env.${version_label}"
}

save_upgrade_state() {
  local old_version="$1"
  local old_release="$2"
  local new_version="$3"
  local new_release="$4"
  install -d -m 0750 "${ETC_DIR}"
  python3 - "${ETC_DIR}/upgrade-state.json" "${old_version}" "${old_release}" "${new_version}" "${new_release}" <<'PY'
import json
import sys
from datetime import datetime, timezone

out_path, old_version, old_release, new_version, new_release = sys.argv[1:6]
payload = {
    "previous_version": old_version,
    "previous_release_dir": old_release,
    "target_version": new_version,
    "target_release_dir": new_release,
    "started_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
}
with open(out_path, "w", encoding="utf-8") as handle:
    json.dump(payload, handle, indent=2)
    handle.write("\n")
PY
  chmod 0600 "${ETC_DIR}/upgrade-state.json"
}

wait_for_postgres() {
  local release_dir="$1"
  local superuser="$2"
  local db="$3"
  for _ in $(seq 1 60); do
    if compose_from_release "${release_dir}" exec -T postgres pg_isready -U "${superuser}" -d "${db}" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
  done
  echo "postgres did not become ready" >&2
  return 1
}

wait_collector_health() {
  local port="$1"
  local timeout="${2:-120}"
  local elapsed=0
  while [[ "${elapsed}" -lt "${timeout}" ]]; do
    if curl -fsS "http://127.0.0.1:${port}/healthz" >/dev/null 2>&1; then
      return 0
    fi
    sleep 2
    elapsed=$((elapsed + 2))
  done
  echo "collector health check failed on port ${port}" >&2
  return 1
}

upgrade_appliance_release() {
  local old_release_dir="$1"
  local old_version="$2"
  local canary="${UPGRADE_CANARY:-0}"
  local host_arch bundle_arch superuser db
  local idx service port
  local backup_env state_file previous_version previous_release_dir current_release_dir

  if [[ -z "${RELEASE_DIR:-}" || -z "${VERSION:-}" ]]; then
    echo "upgrade_appliance_release: RELEASE_DIR and VERSION are required" >&2
    return 1
  fi
  if [[ ! -f "${RELEASE_DIR}/release.env" ]]; then
    echo "release missing release.env: ${RELEASE_DIR}" >&2
    return 1
  fi
  if [[ ! -f "${old_release_dir}/.setup-complete" ]]; then
    echo "current release is not configured (.setup-complete missing); run equate configure first" >&2
    return 1
  fi
  COMPOSE_ENV="${RUN_DIR}/rendered/compose.env"
  if [[ ! -f "${COMPOSE_ENV}" ]]; then
    echo "missing rendered compose env: ${COMPOSE_ENV}" >&2
    return 1
  fi
  if [[ "${old_version}" == "${VERSION}" ]]; then
    echo "already running version ${VERSION}" >&2
    return 1
  fi

  host_arch="$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')"
  bundle_arch="$(grep -E '^EQUATE_ARCH=' "${RELEASE_DIR}/release.env" | cut -d= -f2- | tr -d '[:space:]')"
  if [[ -n "${bundle_arch}" && "${bundle_arch}" != "${host_arch}" ]]; then
    echo "bundle architecture ${bundle_arch} does not match host ${host_arch}" >&2
    return 1
  fi

  echo "upgrading appliance ${old_version} -> ${VERSION}"
  echo "  from: ${old_release_dir}"
  echo "  to:   ${RELEASE_DIR}"

  copy_customer_artifacts "${old_release_dir}" "${RELEASE_DIR}"
  load_release_images "${RELEASE_DIR}"

  COMPOSE_ENV="${RUN_DIR}/rendered/compose.env"
  backup_compose_env "${old_version}"

  echo "stopping current stack..."
  compose_from_release "${old_release_dir}" down --remove-orphans 2>/dev/null || true

  merge_compose_env_for_upgrade "${COMPOSE_ENV}" "${RELEASE_DIR}/release.env" "${COMPOSE_ENV}.next"
  mv "${COMPOSE_ENV}.next" "${COMPOSE_ENV}"
  save_upgrade_state "${old_version}" "${old_release_dir}" "${VERSION}" "${RELEASE_DIR}"

  # shellcheck disable=SC1091
  source "${COMPOSE_ENV}"
  superuser="${POSTGRES_USER:-ogsd}"
  db="${POSTGRES_DB:-ogsd}"

  echo "running database migrations..."
  compose_from_release "${RELEASE_DIR}" run --rm migrate

  sync_appliance_db_role_passwords

  if [[ "${canary}" == "1" ]]; then
    echo "canary rollout: starting core services..."
    compose_from_release "${RELEASE_DIR}" up -d postgres mosquitto
    wait_for_postgres "${RELEASE_DIR}" "${superuser}" "${db}"
    compose_from_release "${RELEASE_DIR}" up -d ingestion backend-api frontend
    sleep 5

    if [[ -f "${RELEASE_DIR}/sites/manifest.yaml" ]]; then
      mapfile -t COLLECTOR_SERVICES < <(awk '/^    service_name: / {print $2}' "${RELEASE_DIR}/sites/manifest.yaml")
      mapfile -t ADMIN_PORTS < <(awk '/    admin_port: / {print $2}' "${RELEASE_DIR}/sites/manifest.yaml")
      for idx in "${!COLLECTOR_SERVICES[@]}"; do
        service="${COLLECTOR_SERVICES[$idx]}"
        port="${ADMIN_PORTS[$idx]:-$((19090 + idx))}"
        echo "canary rollout: starting collector ${service}..."
        compose_from_release "${RELEASE_DIR}" up -d "${service}"
        if ! wait_collector_health "${port}"; then
          echo "canary collector ${service} failed health check; run configure-vm.sh --rollback" >&2
          return 1
        fi
      done
    fi
  else
    echo "starting upgraded stack..."
    compose_from_release "${RELEASE_DIR}" up -d --remove-orphans
  fi

  install -m 0644 "${RELEASE_DIR}/scripts/equate-auth-broker.service" /etc/systemd/system/equate-auth-broker.service
  chmod 0755 "${RELEASE_DIR}/scripts/auth-broker.sh"
  systemctl daemon-reload
  systemctl restart equate-auth-broker.service

  cat >"${RUN_DIR}/rendered/installation.json" <<EOF
{"version":"${VERSION}","installed_at":"$(date -u +"%Y-%m-%dT%H:%M:%SZ")","upgraded_from":"${old_version}"}
EOF
  chmod 0600 "${RUN_DIR}/rendered/installation.json"
  echo "upgrade complete: ${old_version} -> ${VERSION}"
}

rollback_appliance_release() {
  local state_file="${ETC_DIR}/upgrade-state.json"
  if [[ ! -f "${state_file}" ]]; then
    echo "no upgrade state found at ${state_file}" >&2
    return 1
  fi

  local previous_version previous_release_dir current_release_dir
  previous_version="$(python3 - "${state_file}" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as handle:
    print(json.load(handle)["previous_version"])
PY
)"
  previous_release_dir="$(python3 - "${state_file}" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as handle:
    print(json.load(handle)["previous_release_dir"])
PY
)"
  current_release_dir="$(python3 - "${state_file}" <<'PY'
import json, sys
with open(sys.argv[1], encoding="utf-8") as handle:
    print(json.load(handle)["target_release_dir"])
PY
)"

  if [[ ! -d "${previous_release_dir}" ]]; then
    echo "previous release directory missing: ${previous_release_dir}" >&2
    return 1
  fi
  if [[ ! -f "${previous_release_dir}/release.env" ]]; then
    echo "previous release missing release.env: ${previous_release_dir}" >&2
    return 1
  fi

  COMPOSE_ENV="${RUN_DIR}/rendered/compose.env"
  local backup_env="${ETC_DIR}/upgrade-backup/compose.env.${previous_version}"
  if [[ ! -f "${COMPOSE_ENV}" ]]; then
    echo "missing rendered compose env: ${COMPOSE_ENV}" >&2
    return 1
  fi
  if [[ ! -f "${backup_env}" ]]; then
    echo "missing compose env backup: ${backup_env}" >&2
    return 1
  fi

  echo "rolling back to release ${previous_version} at ${previous_release_dir}"
  if [[ -n "${current_release_dir}" && -d "${current_release_dir}" ]]; then
    compose_from_release "${current_release_dir}" down --remove-orphans 2>/dev/null || true
  fi

  cp -a "${backup_env}" "${COMPOSE_ENV}"
  chmod 0600 "${COMPOSE_ENV}"

  RELEASE_DIR="${previous_release_dir}"
  VERSION="${previous_version}"
  # shellcheck disable=SC1091
  source "${COMPOSE_ENV}"
  compose_from_release "${RELEASE_DIR}" up -d postgres mosquitto
  wait_for_postgres "${RELEASE_DIR}" "${POSTGRES_USER:-ogsd}" "${POSTGRES_DB:-ogsd}"
  sync_appliance_db_role_passwords
  compose_from_release "${RELEASE_DIR}" up -d --remove-orphans

  install -m 0644 "${RELEASE_DIR}/scripts/equate-auth-broker.service" /etc/systemd/system/equate-auth-broker.service
  chmod 0755 "${RELEASE_DIR}/scripts/auth-broker.sh"
  systemctl daemon-reload
  systemctl restart equate-auth-broker.service

  rm -f "${state_file}"
  echo "rollback complete: active release is ${previous_version}"
}
