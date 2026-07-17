# SNMP Collector deployments

## systemd (production host)

[`systemd/snmp-collector.service`](systemd/snmp-collector.service) is a least-privilege unit for a dedicated `snmp-collector` service user.

### Install sketch

1. Create user/group `snmp-collector` with no login shell.
2. Install the binary to `/usr/local/bin/collector`.
3. Place static config at `/etc/equate/collector.yaml` (root-owned, readable by the service user).
4. Create `/var/lib/snmp-collector` owned by `snmp-collector:snmp-collector` mode `0750`.
5. Point `inventory.managed_path`, `buffer.path`, and `admin.control_socket` at state/runtime directories:
   - managed inventory: `/var/lib/snmp-collector/managed-inventory.yaml` (`0600`)
   - audit log: `/var/lib/snmp-collector/managed-inventory.yaml.audit.log` (`0600`)
   - control socket: `/run/snmp-collector/control.sock` (`0600`, via `RuntimeDirectory=`)
6. Supply SNMP/MQTT secrets only through `/etc/equate/snmp-collector.env` (`community_env` / `password_env` names in YAML).
7. `systemctl enable --now snmp-collector`

### Operator workflow

```bash
collector validate -config /etc/equate/collector.yaml
collector tui -socket /run/snmp-collector/control.sock
# or: systemctl reload snmp-collector   # SIGHUP
```

### Rollback

- Invalid managed writes or reloads leave the prior runtime snapshot active.
- To discard managed overlays: replace or remove the managed inventory file, then `systemctl reload snmp-collector`.
- Static YAML is never modified by the TUI/control plane.

### Security notes

The unit sets `ProtectSystem=strict`, `PrivateTmp=true`, `NoNewPrivileges=true`, and an empty `CapabilityBoundingSet`. `ReadWritePaths` is limited to collector state and runtime directories. The control socket is local-only; do not expose it over TCP or HTTP.
