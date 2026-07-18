# Development VxRail — on-site SNMP collectors (OrbStack / GNS3 lab)

Runs one or more SNMP collector site containers on a lab Ubuntu VM. Cloud services run on the Mac via [`../`](../) (`deployments/development`).

## Prerequisites

- OrbStack/Ubuntu VM with Docker Engine + Compose
- Layer-2 reachability to GNS3 SNMP devices
- Outbound TCP to Mac Mosquitto `:8883` (GNS3 Cloud / host network as documented in the parent README)
- Lab CA certificate for Mosquitto trust (`certs/ca.crt`)
- Static template at [`configs/collector.yaml`](configs/collector.yaml)
- Shared SNMP community values in `.env` (`SNMP_COMMUNITY`, `SNMP_DISCOVERY_COMMUNITY`)
- MQTT username/password matching Mac Mosquitto ACL (`collector` publish-only)

## Layout

| File | Purpose |
|------|---------|
| [`docker-compose.yml`](docker-compose.yml) | Shared collector image anchor |
| [`docker-compose.sites.generated.yml`](docker-compose.sites.generated.yml) | Generated per-site services (after setup) |
| [`sites/manifest.yaml`](sites/manifest.yaml) | Generated site count, CIDRs, ports, identities |
| [`.env.example`](.env.example) | Shared `MQTT_BROKER`, passwords, SNMP communities |
| [`configs/collector.yaml`](configs/collector.yaml) | Static template copied per site at setup |
| [`sites/<site-id>/`](sites/) | Per-site config, managed inventory, control socket |
| [`certs/`](certs/) | Place lab `ca.crt` only (no private keys) |

## Rollout order

1. Mac cloud plane healthy (`deployments/development/up.sh`) and Mosquitto reachable from the VM
2. Install CA at `certs/ca.crt`
3. Run first-boot setup (writes shared `.env`, per-site artifacts, starts collectors, discovers devices):

```bash
./bootstrap.sh
# or from services/snmp-collector:
# go run ./cmd/collector setup -dir ../../deployments/development/vxrail -theme auto
```

The setup wizard asks for:
- Shared MQTT broker/password and SNMP communities
- Site container count (default `4`)
- Per-site **site id** (defaults `site-001`, `site-002`, …; operator-chosen identifiers used everywhere)
- One discovery CIDR per site (defaults `10.255.0.0/24`, `10.255.1.0/24`, …)

4. Verify `GET /healthz` on each site admin port (`19090`, `19091`, …)
5. Confirm v2 samples appear in Mac API/UI for each `site_id`

Subsequent starts skip the wizard when `.setup-complete` exists. Re-run the wizard with:

```bash
./bootstrap.sh --reconfigure
```

## Day-2 operator TUI

Each site has its own control socket and Compose service. On **Docker Desktop (Mac)**, run the TUI inside the site container:

```bash
cd deployments/development/vxrail
docker compose -f docker-compose.yml -f docker-compose.sites.generated.yml \
  exec -it snmp-collector-site-001 /collector tui \
  -socket /run/snmp-collector/control.sock -theme auto
```

On a **Linux VM** with `collector` on `PATH`:

```bash
collector tui -socket ./sites/site-001/run/control.sock -theme auto
```

Repeat for `site-002`, `site-003`, etc.

## Verification

```bash
curl -fsS http://127.0.0.1:19090/healthz   # site-001
curl -fsS http://127.0.0.1:19091/healthz   # site-002
docker compose -f docker-compose.yml -f docker-compose.sites.generated.yml ps

# On Mac: API shows each site after first successful poll
```

## Notes

- Shared secrets live only in `.env`; generated YAML never contains community strings or MQTT passwords
- Each site has its own state volume, managed inventory, and Unix control socket
- Admin ports are `19090 + index - 1` on the host (avoids development cloud admin ports `9091`/`9092`); control sockets are never published as TCP
- Do not commit `.env`, `sites/`, or `docker-compose.sites.generated.yml`
- Field acceptance checklist: [`../../runbooks/field-acceptance-gns3.md`](../../runbooks/field-acceptance-gns3.md)
- Multi-site lab decision: [`.ai/decisions/collector-10.md`](../../../.ai/decisions/collector-10.md)
