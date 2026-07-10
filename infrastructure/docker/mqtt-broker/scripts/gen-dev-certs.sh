# Generate a local CA + Mosquitto server certificate for MQTT/TLS lab use.
# Output is written under ./certs/ (gitignored). Do not use these certs in production.
#
# Optional env:
#   MQTT_SERVER_CN   — certificate CN (default: localhost)
#   MQTT_SERVER_DNS  — comma-separated extra DNS SANs (default: host.docker.internal)
#   MQTT_SERVER_IP   — comma-separated extra IP SANs (e.g. GNS3 cloud VM IP)
#   MQTT_CERT_DAYS   — validity days (default: 825)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CERTS="${ROOT}/certs"
mkdir -p "${CERTS}"

CN="${MQTT_SERVER_CN:-localhost}"
DAYS="${MQTT_CERT_DAYS:-825}"
# Comma-separated extra DNS names (e.g. host.docker.internal,cloud.lab).
EXTRA_DNS="${MQTT_SERVER_DNS:-host.docker.internal}"
# Comma-separated extra IP addresses (e.g. GNS3 cloud VM IP).
EXTRA_IPS="${MQTT_SERVER_IP:-}"

echo "Generating CA and server cert for CN=${CN} into ${CERTS}"

openssl genrsa -out "${CERTS}/ca.key" 4096
openssl req -x509 -new -nodes -key "${CERTS}/ca.key" -sha256 -days "${DAYS}" \
  -subj "/CN=Equate OGSD Dev CA" \
  -out "${CERTS}/ca.crt"

openssl genrsa -out "${CERTS}/server.key" 2048
openssl req -new -key "${CERTS}/server.key" \
  -subj "/CN=${CN}" \
  -out "${CERTS}/server.csr"

{
  echo "authorityKeyIdentifier=keyid,issuer"
  echo "basicConstraints=CA:FALSE"
  echo "keyUsage = digitalSignature, keyEncipherment"
  echo "extendedKeyUsage = serverAuth"
  echo "subjectAltName = @alt_names"
  echo ""
  echo "[alt_names]"
  dns_i=1
  echo "DNS.${dns_i} = ${CN}"
  dns_i=$((dns_i + 1))
  for name in localhost mosquitto; do
    if [[ "${name}" != "${CN}" ]]; then
      echo "DNS.${dns_i} = ${name}"
      dns_i=$((dns_i + 1))
    fi
  done
  if [[ -n "${EXTRA_DNS}" ]]; then
    IFS=',' read -ra DNS_ARR <<< "${EXTRA_DNS}"
    for name in "${DNS_ARR[@]}"; do
      name="$(echo "${name}" | xargs)"
      [[ -z "${name}" ]] && continue
      echo "DNS.${dns_i} = ${name}"
      dns_i=$((dns_i + 1))
    done
  fi
  ip_i=1
  echo "IP.${ip_i} = 127.0.0.1"
  ip_i=$((ip_i + 1))
  if [[ -n "${EXTRA_IPS}" ]]; then
    IFS=',' read -ra IP_ARR <<< "${EXTRA_IPS}"
    for ip in "${IP_ARR[@]}"; do
      ip="$(echo "${ip}" | xargs)"
      [[ -z "${ip}" ]] && continue
      echo "IP.${ip_i} = ${ip}"
      ip_i=$((ip_i + 1))
    done
  fi
} > "${CERTS}/server.ext"

openssl x509 -req -in "${CERTS}/server.csr" -CA "${CERTS}/ca.crt" -CAkey "${CERTS}/ca.key" \
  -CAcreateserial -out "${CERTS}/server.crt" -days "${DAYS}" -sha256 \
  -extfile "${CERTS}/server.ext"

chmod 600 "${CERTS}/ca.key" "${CERTS}/server.key"
rm -f "${CERTS}/server.csr" "${CERTS}/server.ext" "${CERTS}/ca.srl"

echo "Wrote:"
echo "  ${CERTS}/ca.crt"
echo "  ${CERTS}/server.crt"
echo "  ${CERTS}/server.key"
echo "Mount ca.crt into the collector as mqtt.tls.ca_file."
