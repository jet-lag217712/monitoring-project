#!/usr/bin/env python3
"""Runtime probe for Windows->Mac->GNS3(eth2 .129) path. Writes NDJSON debug logs."""
import json, os, subprocess, time, urllib.request

LOG = "/Users/jeetlad/Projects/Equate/monitoring-dashboard/.cursor/debug-2afd95.log"
INGEST = "http://127.0.0.1:7535/ingest/67222a7b-79e8-4cfd-9a12-c85ccde20fea"
SESSION = "2afd95"
RUN = os.environ.get("DEBUG_RUN_ID", "pre-fix")


def log(hypothesis_id, location, message, data=None):
    # #region agent log
    payload = {
        "sessionId": SESSION,
        "runId": RUN,
        "hypothesisId": hypothesis_id,
        "location": location,
        "message": message,
        "data": data or {},
        "timestamp": int(time.time() * 1000),
    }
    line = json.dumps(payload, separators=(",", ":"))
    os.makedirs(os.path.dirname(LOG), exist_ok=True)
    with open(LOG, "a") as f:
        f.write(line + "\n")
    try:
        req = urllib.request.Request(
            INGEST,
            data=line.encode(),
            headers={"Content-Type": "application/json", "X-Debug-Session-Id": SESSION},
            method="POST",
        )
        urllib.request.urlopen(req, timeout=1).read()
    except Exception:
        pass
    # #endregion


def sh(cmd):
    p = subprocess.run(cmd, shell=True, capture_output=True, text=True)
    return p.returncode, (p.stdout or "") + (p.stderr or "")


def ping(host, src=None, count=2, wait_ms=1000):
    cmd = f"/sbin/ping -c {count} -W {wait_ms}"
    if src:
        cmd += f" -S {src}"
    cmd += f" {host}"
    rc, out = sh(cmd)
    ok = rc == 0 and "bytes from" in out
    loss = "100.0% packet loss" in out or "100% packet loss" in out
    return {
        "ok": ok,
        "loss": loss,
        "rc": rc,
        "summary": out.strip().splitlines()[-2:] if out.strip() else [],
        "out_tail": out.strip()[-300:],
    }


# H-A: Mac routes/forwarding for eth2 next-hop .129
_, fwd = sh("/usr/sbin/sysctl -n net.inet.ip.forwarding")
_, routes = sh("/usr/sbin/netstat -rn -f inet | /usr/bin/grep -E '10\\.255|192\\.168\\.103'")
_, route_get = sh("/sbin/route -n get 10.255.0.1")
gw_is_129 = "192.168.103.129" in route_get
log(
    "A",
    "debug-probe-gns3.py:routes",
    "forwarding and lab routes via eth2",
    {
        "forwarding": fwd.strip(),
        "routes": routes.strip().splitlines(),
        "route_get_10_255_0_1": route_get.strip(),
        "next_hop_is_eth2_129": gw_is_129,
    },
)

# H-B: eth2 (.129) reachability; DO-CORE (.2) on segment but NOT used as Mac next-hop
log(
    "B",
    "debug-probe-gns3.py:eth2",
    "eth2 and DO-CORE L2 reachability",
    {
        "ping_eth2_129": ping("192.168.103.129", count=1),
        "ping_docore_2": ping("192.168.103.2", count=1),
    },
)

# H-C: Mac->lab via eth2 next-hop (native egress source)
lab_targets = ["10.255.0.1", "10.255.1.1", "10.255.2.1", "10.255.3.1"]
native = {t: ping(t, count=1) for t in lab_targets}
log("C", "debug-probe-gns3.py:native", "Mac native pings to lab nets", native)

# H-D: simulate LAN client source (Wi-Fi IP) — return path / NAT for Windows-like traffic
wifi_src = {t: ping(t, src="10.0.0.125", count=1) for t in lab_targets}
log(
    "D",
    "debug-probe-gns3.py:wifi-src",
    "pings sourced from Wi-Fi IP (LAN-client simulation)",
    wifi_src,
)

# H-E: pf NAT anchor present for LAN->lab return path
rc, nat = sh("/sbin/pfctl -a gns3.lan.gateway -s nat 2>&1")
_, anchors = sh("/sbin/pfctl -s Anchors 2>&1 | /usr/bin/grep gns3 || true")
_, info = sh("/sbin/pfctl -s info 2>&1 | /usr/bin/head -15")
log(
    "E",
    "debug-probe-gns3.py:pf",
    "pf NAT/anchor status",
    {
        "nat_rc": rc,
        "nat": nat.strip().splitlines()[:20],
        "anchors": anchors.strip(),
        "info_head": info.strip().splitlines()[:15],
        "nat_mentions_10_255": any("10.255" in line for line in nat.splitlines()),
    },
)

# --- Duplicate ICMP diagnosis (user report) ---
import re

def ping_dup_analysis(host, count=5):
    rc, out = sh(f"/sbin/ping -c {count} -W 1000 {host}")
    replies = []
    for line in out.splitlines():
        # 64 bytes from 10.255.0.1: icmp_seq=0 ttl=255 time=15.308 ms
        # ... (DUP!)
        m = re.search(
            r"bytes from ([0-9.]+): icmp_seq=(\d+) ttl=(\d+) time=([0-9.]+) ms( \(DUP!\))?",
            line,
        )
        if m:
            replies.append(
                {
                    "from": m.group(1),
                    "seq": int(m.group(2)),
                    "ttl": int(m.group(3)),
                    "time_ms": float(m.group(4)),
                    "dup": bool(m.group(5)),
                }
            )
    dups = [r for r in replies if r["dup"]]
    by_seq = {}
    for r in replies:
        by_seq.setdefault(r["seq"], []).append(r)
    multi = {str(k): v for k, v in by_seq.items() if len(v) > 1}
    ttls = sorted({r["ttl"] for r in replies})
    return {
        "host": host,
        "rc": rc,
        "reply_count": len(replies),
        "dup_count": len(dups),
        "unique_ttls": ttls,
        "multi_reply_seqs": multi,
        "raw_tail": out.strip()[-800:],
    }


# H-F: bridge100 has multiple vmenet members (L2 loop / duplicate frames)
_, ifconfig_b100 = sh("/sbin/ifconfig bridge100")
_, ifconfig_all_vm = sh("/sbin/ifconfig -a | /usr/bin/grep -E '^[a-z]|inet |member:'")
members = re.findall(r"member:\s+(\S+)", ifconfig_b100)
log(
    "F",
    "debug-probe-gns3.py:bridge",
    "bridge100 membership (possible L2 dup path)",
    {
        "bridge100": ifconfig_b100.strip(),
        "members": members,
        "member_count": len(members),
        "multi_member_bridge": len(members) > 1,
    },
)

# H-G: ping duplicates to loopbacks / eth2 with TTL fingerprinting
dup_targets = ["10.255.0.1", "10.255.1.1", "10.255.2.1", "10.255.3.1", "192.168.103.129", "192.168.103.2"]
dup_results = {t: ping_dup_analysis(t, count=4) for t in dup_targets}
log(
    "G",
    "debug-probe-gns3.py:dups",
    "ICMP duplicate analysis with TTL fingerprint",
    dup_results,
)

# H-H: ICMP redirects suggesting dual next-hops on shared L2
_, ping_redir = sh("/sbin/ping -c 3 -W 1000 10.255.0.1")
redirects = [ln for ln in ping_redir.splitlines() if "Redirect" in ln or "DUP" in ln]
log(
    "H",
    "debug-probe-gns3.py:redirects",
    "ICMP redirects / DUP lines during lab ping",
    {"lines": redirects, "full_tail": ping_redir.strip()[-600:]},
)

# H-I: compare next-hop .129 vs direct .2 (evidence only; do not change routes)
_, get129 = sh("/sbin/route -n get 10.255.0.1")
log(
    "I",
    "debug-probe-gns3.py:nexthop",
    "current next-hop for lab (must remain eth2 .129 per user)",
    {"route_get": get129.strip(), "uses_129": "192.168.103.129" in get129},
)

summary = {
    "runId": RUN,
    "forwarding": fwd.strip(),
    "next_hop_is_eth2_129": gw_is_129,
    "native_ok": all(v["ok"] for v in native.values()),
    "wifi_src_ok": all(v["ok"] for v in wifi_src.values()),
    "pf_nat_ok": any("10.255" in line for line in nat.splitlines()),
    "bridge_members": members,
    "dup_counts": {t: dup_results[t]["dup_count"] for t in dup_targets},
    "dup_ttls": {t: dup_results[t]["unique_ttls"] for t in dup_targets},
}
print(json.dumps(summary, indent=2))
