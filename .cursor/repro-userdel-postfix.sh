#!/usr/bin/env bash
# Post-fix: verify prepare-ova userdel -f removes debian under active SSH.
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq >/dev/null
apt-get install -y -qq passwd procps python3 openssh-server sudo >/dev/null

id debian >/dev/null 2>&1 || useradd -m -s /bin/bash debian
echo 'debian ALL=(ALL) NOPASSWD:ALL' > /etc/sudoers.d/debian
chmod 440 /etc/sudoers.d/debian
mkdir -p /run/sshd /home/debian/.ssh /var/run/sshd
ssh-keygen -t ed25519 -N '' -f /root/testkey >/dev/null
cp /root/testkey.pub /home/debian/.ssh/authorized_keys
chown -R debian:debian /home/debian/.ssh
chmod 700 /home/debian/.ssh
chmod 600 /home/debian/.ssh/authorized_keys
printf '%s\n' 'PasswordAuthentication no' 'PermitRootLogin no' 'UsePAM yes' > /etc/ssh/sshd_config.d/ci-test.conf
/usr/sbin/sshd

ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
  -i /root/testkey debian@127.0.0.1 'sleep 120' >/tmp/bg-ssh.log 2>&1 &
sleep 1

# Mirror fixed prepare-ova loop
cat > /tmp/fixed-del.sh <<'EOS'
#!/usr/bin/env bash
set -euo pipefail
FAIL=0
report() { echo "prepare-ova: $*" >&2; FAIL=1; }
for build_user in debian packer; do
  if id "${build_user}" >/dev/null 2>&1; then
    _procs="$(ps -u "${build_user}" -o pid=,comm= 2>/dev/null | tr '\n' ';' | head -c 500 || true)"
    echo "==> debug prepare-ova userdel pre (hypothesis A): user=${build_user}; procs=${_procs}"
    set +e
    _out="$(userdel -f -r "${build_user}" 2>&1)"
    _rc=$?
    if [[ "${_rc}" -ne 0 ]]; then
      _out2="$(userdel -f "${build_user}" 2>&1)"
      _rc2=$?
      _out="${_out}; fallback: ${_out2}"
      _rc="${_rc2}"
    fi
    set -e
    _exists="$(id "${build_user}" >/dev/null 2>&1 && echo yes || echo no)"
    echo "==> debug prepare-ova userdel post (hypothesis A): user=${build_user}; rc=${_rc}; exists=${_exists}; out=${_out}"
    if id "${build_user}" >/dev/null 2>&1; then
      report "failed to remove CI build account ${build_user}: ${_out}"
    fi
  fi
done
if id debian >/dev/null 2>&1; then
  report "CI build account debian still present"
fi
echo "FAIL=${FAIL}"
exit "${FAIL}"
EOS
chmod +x /tmp/fixed-del.sh

set +e
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
  -i /root/testkey debian@127.0.0.1 'sudo bash /tmp/fixed-del.sh' | tee /tmp/try-out.txt
RC=${PIPESTATUS[0]}
set -e

python3 - <<PY
import json, time, re
text = open("/tmp/try-out.txt").read()
exists = "yes" if re.search(r"exists=yes", text) else "no"
fail = re.search(r"^FAIL=(\d+)$", text, re.M)
print(json.dumps({
  "sessionId": "887798",
  "hypothesisId": "A",
  "location": "docker-repro:post-fix",
  "message": "fixed userdel -f under SSH login",
  "data": {
    "ssh_rc": ${RC},
    "fail": fail.group(1) if fail else None,
    "debian_still_present_in_log": "CI build account debian still present" in text,
    "post_lines": [ln for ln in text.splitlines() if "userdel post" in ln or ln.startswith("FAIL=")],
  },
  "timestamp": int(time.time()*1000),
  "runId": "post-fix",
}))
PY
