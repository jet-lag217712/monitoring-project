#!/usr/bin/env bash
# Host-side PAM authentication broker for appliance_local API auth.
# Listens on a permissioned Unix socket and speaks newline-delimited JSON.
#
# Request:
#   {"operation":"authenticate","username":"alice","password":"..."}
#   {"operation":"account_status","username":"alice"}
#
# Response:
#   {"ok":true,"username":"alice"}
#   {"ok":false,"error":"invalid credentials"}
set -euo pipefail

APPLIANCE_GROUP="${EQUATE_APPLIANCE_GROUP:-equate-appliance}"
SOCKET_PATH="${EQUATE_AUTH_SOCKET:-/run/equate/auth.sock}"
API_GID="${EQUATE_API_GID:-65532}"

if [[ "$(id -u)" -ne 0 ]]; then
  echo "auth-broker must run as root" >&2
  exit 1
fi

if ! command -v python3 >/dev/null 2>&1; then
  echo "python3 is required" >&2
  exit 1
fi

if ! command -v pamtester >/dev/null 2>&1; then
  echo "pamtester is required (install libpam-modules and pamtester)" >&2
  exit 1
fi

SOCKET_DIR="$(dirname "${SOCKET_PATH}")"
install -d -m 0750 -o root -g "${API_GID}" "${SOCKET_DIR}"
rm -f "${SOCKET_PATH}"

exec python3 - "${SOCKET_PATH}" "${APPLIANCE_GROUP}" "${API_GID}" <<'PY'
import grp
import json
import os
import pwd
import socket
import subprocess
import sys

SOCKET_PATH, APPLIANCE_GROUP, API_GID = sys.argv[1:4]
API_GID = int(API_GID)


def write_response(conn, ok, username="", error=""):
    payload = {"ok": bool(ok), "username": username}
    if error:
        payload["error"] = error
    if ok and error:
        raise RuntimeError("inconsistent broker response")
    conn.sendall((json.dumps(payload, separators=(",", ":")) + "\n").encode("utf-8"))


def in_appliance_group(username: str) -> bool:
    try:
        group = grp.getgrnam(APPLIANCE_GROUP)
        user = pwd.getpwnam(username)
    except KeyError:
        return False
    return group.gr_gid in os.getgrouplist(username, user.pw_gid)


def account_locked(username: str) -> bool:
    proc = subprocess.run(
        ["passwd", "-S", username],
        capture_output=True,
        text=True,
        check=False,
    )
    if proc.returncode != 0:
        return True
    parts = proc.stdout.split()
    if len(parts) < 2:
        return True
    return parts[1] in {"L", "LK", "N", "NP"}


def authenticate(username: str, password: str) -> bool:
    if not in_appliance_group(username):
        return False
    proc = subprocess.run(
        ["pamtester", "login", username, "authenticate"],
        input=password,
        capture_output=True,
        text=True,
        check=False,
    )
    return proc.returncode == 0


def account_status(username: str) -> bool:
    if not in_appliance_group(username):
        return False
    if account_locked(username):
        return False
    return True


def handle_request(raw: str):
  try:
      req = json.loads(raw)
  except json.JSONDecodeError:
      return False, "", "invalid json"
  op = req.get("operation")
  username = req.get("username", "")
  if not isinstance(username, str) or not username:
      return False, "", "username is required"
  if op == "authenticate":
      password = req.get("password", "")
      if not isinstance(password, str) or not password:
          return False, "", "password is required"
      if authenticate(username, password):
          return True, username, ""
      return False, username, "invalid credentials"
  if op == "account_status":
      if account_status(username):
          return True, username, ""
      return False, username, "account unavailable"
  return False, "", "unsupported operation"


server = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
server.bind(SOCKET_PATH)
os.chown(SOCKET_PATH, 0, API_GID)
os.chmod(SOCKET_PATH, 0o660)
server.listen(32)

while True:
    conn, _ = server.accept()
    with conn:
        try:
            data = b""
            while not data.endswith(b"\n"):
                chunk = conn.recv(4096)
                if not chunk:
                    break
                data += chunk
                if len(data) > 4096:
                    raise ValueError("request too large")
            if not data:
                continue
            ok, username, error = handle_request(data.decode("utf-8").strip())
            write_response(conn, ok, username, error)
        except Exception as exc:  # noqa: BLE001 - broker must stay alive
            write_response(conn, False, "", str(exc))
PY
