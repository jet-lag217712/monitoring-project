#!/usr/bin/env bash
# Generate a local CA + Mosquitto server certificate for Phase 2 MQTT/TLS.
# Output is written under ./certs/ (gitignored). Do not use these certs in production.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
CERTS="${ROOT}/certs"
mkdir -p "${CERTS}"

CN="${MQTT_SERVER_CN:-localhost}"
DAYS="${MQTT_CERT_DAYS:-825}"

echo "Generating CA and server cert for CN=${CN} into ${CERTS}"

openssl genrsa -out "${CERTS}/ca.key" 4096
openssl req -x509 -new -nodes -key "${CERTS}/ca.key" -sha256 -days "${DAYS}" \
  -subj "/CN=Equate OGSD Dev CA" \
  -out "${CERTS}/ca.crt"

openssl genrsa -out "${CERTS}/server.key" 2048
openssl req -new -key "${CERTS}/server.key" \
  -subj "/CN=${CN}" \
  -out "${CERTS}/server.csr"

cat > "${CERTS}/server.ext" <<EOF
authorityKeyIdentifier=keyid,issuer
basicConstraints=CA:FALSE
keyUsage = digitalSignature, keyEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
DNS.1 = ${CN}
DNS.2 = localhost
DNS.3 = mosquitto
IP.1 = 127.0.0.1
EOF

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
