#!/usr/bin/env bash
# Reproduce prepare-ova userdel failure when build user still has processes.
set -euo pipefail
LOG="$(cd "$(dirname "$0")" && pwd)/debug-887798.log"
: > "${LOG}"

docker run --rm debian:bookworm-slim bash -s <<'GUEST' | tee /tmp/ova-userdel-repro.out
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq >/dev/null
apt-get install -y -qq passwd procps python3 >/dev/null

id debian >/dev/null 2>&1 || useradd -m debian
su -s /bin/bash debian -c "sleep 60" &
sleep 0.3

echo "=== processes for debian ==="
ps -u debian -o pid,user,comm || true

echo "=== userdel -r (no force) ==="
set +e
OUT1=$(userdel -r debian 2>&1)
RC1=$?
set -e
EXISTS1=$(id debian >/dev/null 2>&1 && echo yes || echo no)
echo "RC1=${RC1} EXISTS1=${EXISTS1} OUT1=${OUT1}"

echo "=== current prepare-ova pattern (swallow errors) ==="
set +e
userdel -r debian 2>/dev/null || userdel debian 2>/dev/null || true
set -e
EXISTS_SWALLOW=$(id debian >/dev/null 2>&1 && echo yes || echo no)
echo "EXISTS_AFTER_SWALLOW=${EXISTS_SWALLOW}"

echo "=== userdel -f -r ==="
set +e
OUT2=$(userdel -f -r debian 2>&1)
RC2=$?
set -e
EXISTS2=$(id debian >/dev/null 2>&1 && echo yes || echo no)
echo "RC2=${RC2} EXISTS2=${EXISTS2} OUT2=${OUT2}"

python3 - <<PY
import json, time
print(json.dumps({
  "sessionId": "887798",
  "hypothesisId": "A",
  "location": "docker-repro:userdel",
  "message": "userdel while debian has process",
  "data": {
    "rc_noforce": ${RC1},
    "exists_noforce": "${EXISTS1}",
    "out_noforce": """${OUT1}""",
    "exists_after_swallow_pattern": "${EXISTS_SWALLOW}",
    "rc_force": ${RC2},
    "exists_force": "${EXISTS2}",
    "out_force": """${OUT2}""",
  },
  "timestamp": int(time.time() * 1000),
  "runId": "pre-fix",
}))
PY
GUEST

# Persist container NDJSON lines into session log
grep '^{"sessionId"' /tmp/ova-userdel-repro.out >> "${LOG}" || true
echo "wrote ${LOG}"
cat "${LOG}"
