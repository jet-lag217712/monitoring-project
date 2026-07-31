#!/usr/bin/env bash
# Generate Ed25519 key material for Equate .eqa update signing.
#
# Usage:
#   ./appliance/scripts/generate-update-keys.sh
#   ./appliance/scripts/generate-update-keys.sh --out-dir /secure/equate-keys
#
# Writes:
#   <out-dir>/equate-updates.pub   (commit this; keep in sync with EmbeddedPublicKeyHex)
#   <out-dir>/equate-updates.priv  (NEVER commit; set EQUATE_UPDATE_SIGNING_KEY to this path)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
OUT_DIR="${ROOT}/appliance/keys"

while [[ $# -gt 0 ]]; do
  case "$1" in
    --out-dir)
      OUT_DIR="${2:-}"
      shift 2
      ;;
    -h|--help)
      sed -n '2,12p' "$0"
      exit 0
      ;;
    *)
      echo "unknown argument: $1" >&2
      exit 1
      ;;
  esac
done

mkdir -p "${OUT_DIR}"
PUB="${OUT_DIR}/equate-updates.pub"
PRIV="${OUT_DIR}/equate-updates.priv"

TMP="$(mktemp -d)"
trap 'rm -rf "${TMP}"' EXIT
cat >"${TMP}/gen.go" <<'EOF'
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
)

func main() {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Args[1], []byte(hex.EncodeToString(pub)+"\n"), 0o644); err != nil {
		panic(err)
	}
	if err := os.WriteFile(os.Args[2], []byte(hex.EncodeToString(priv)+"\n"), 0o600); err != nil {
		panic(err)
	}
	fmt.Printf("public=%s\n", hex.EncodeToString(pub))
}
EOF

go run "${TMP}/gen.go" "${PUB}" "${PRIV}"
echo "wrote ${PUB}"
echo "wrote ${PRIV} (keep secret)"
echo
echo "Next: update EmbeddedPublicKeyHex in services/snmp-collector/internal/update/verify.go"
echo "to match $(tr -d '[:space:]' < "${PUB}")"
