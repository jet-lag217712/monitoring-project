#!/usr/bin/env bash
# Reproduce userdel failure when debian has an active SSH/utmp login (CI pattern).
set -euo pipefail
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq >/dev/null
apt-get install -y -qq passwd procps python3 openssh-server sudo >/dev/null

id debian >/dev/null 2>&1 || useradd -m -s /bin/bash debian
echo 'debian:debian' | chpasswd
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

# Keep an SSH login open so debian is "logged in" in utmp
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
  -i /root/testkey debian@127.0.0.1 'sleep 120' >/tmp/bg-ssh.log 2>&1 &
sleep 1

echo "=== who / utmp ==="
who || true

cat > /tmp/try-del.sh <<'EOS'
#!/usr/bin/env bash
set +e
echo "SUDO_USER=${SUDO_USER:-}"
echo "whoami=$(whoami)"
who || true

echo "=== userdel -r (no force) ==="
OUT=$(userdel -r debian 2>&1)
RC=$?
echo "RC=${RC}"
echo "OUT=${OUT}"
EXISTS=$(id debian >/dev/null 2>&1 && echo yes || echo no)
echo "EXISTS_AFTER_NOFORCE=${EXISTS}"

echo "=== prepare-ova swallow pattern ==="
userdel -r debian 2>/dev/null || userdel debian 2>/dev/null || true
EXISTS2=$(id debian >/dev/null 2>&1 && echo yes || echo no)
echo "EXISTS_AFTER_SWALLOW=${EXISTS2}"

echo "=== proposed fix userdel -f -r ==="
OUT3=$(userdel -f -r debian 2>&1)
RC3=$?
EXISTS3=$(id debian >/dev/null 2>&1 && echo yes || echo no)
echo "RC_FORCE=${RC3}"
echo "OUT_FORCE=${OUT3}"
echo "EXISTS_AFTER_FORCE=${EXISTS3}"
EOS
chmod +x /tmp/try-del.sh

echo "=== run via ssh+sudo (CI pattern) ==="
ssh -o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null \
  -i /root/testkey debian@127.0.0.1 'sudo bash /tmp/try-del.sh' | tee /tmp/try-out.txt

python3 - <<'PY'
import json, time, re
text = open("/tmp/try-out.txt").read()
def grab(key):
    m = re.search(rf"^{key}=(.*)$", text, re.M)
    return m.group(1).strip() if m else None
data = {
    "sudo_user": grab("SUDO_USER"),
    "whoami": grab("whoami"),
    "rc_noforce": grab("RC"),
    "out_noforce": grab("OUT"),
    "exists_after_noforce": grab("EXISTS_AFTER_NOFORCE"),
    "exists_after_swallow": grab("EXISTS_AFTER_SWALLOW"),
    "rc_force": grab("RC_FORCE"),
    "out_force": grab("OUT_FORCE"),
    "exists_after_force": grab("EXISTS_AFTER_FORCE"),
    "who": "\n".join([ln for ln in text.splitlines() if "pts/" in ln or "debian" in ln.lower()][:10]),
}
print(json.dumps({
    "sessionId": "887798",
    "hypothesisId": "A",
    "location": "docker-repro:ssh-login-userdel",
    "message": "userdel while debian SSH login active",
    "data": data,
    "timestamp": int(time.time() * 1000),
    "runId": "pre-fix",
}))
# Hypothesis B: home dir busy
print(json.dumps({
    "sessionId": "887798",
    "hypothesisId": "B",
    "location": "docker-repro:ssh-login-userdel",
    "message": "noforce failure reason classification",
    "data": {
        "out": data.get("out_noforce"),
        "looks_like_logged_in": bool(data.get("out_noforce") and "logged" in (data.get("out_noforce") or "").lower()),
        "looks_like_busy": bool(data.get("out_noforce") and ("busy" in (data.get("out_noforce") or "").lower() or "in use" in (data.get("out_noforce") or "").lower())),
    },
    "timestamp": int(time.time() * 1000),
    "runId": "pre-fix",
}))
PY
