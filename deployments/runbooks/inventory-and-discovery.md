# Inventory and discovery

## Sources and precedence

1. **Static inventory** (`configs/collector.yaml`) — version-controlled, mounted read-only in Compose.
2. **Managed inventory** (`/var/lib/snmp-collector/managed-inventory.yaml`) — TUI/control writes only; overlays thresholds, upstreams, interface filters, and discovery rate/burst.
3. **Runtime snapshot** — loaded after validate/reload; never written back to static YAML.

Cross-source ID/IP collisions fail validation. Static-authoritative fields (host, port, `community_env`, collector identity, MQTT, buffer, admin listener, discovery CIDR allowlist) are not overlayable.

## Operator mutations

Use the local control socket (never HTTP mutation):

```bash
collector tui -socket ./run/control.sock
# or NDJSON against admin.control_socket
```

Mutations use prepare → confirm → commit with revision binding, then `config.reload`
(same path as `SIGHUP`). Invalid reloads leave the prior snapshot active.

## Discovery

- Operator-invoked only (`collector discover` or TUI discovery workflow).
- Allowlisted CIDRs + token-bucket rate limits.
- Never auto-enrolls devices; accept writes managed inventory then requires explicit reload.

See [`services/snmp-collector/README.md`](../../services/snmp-collector/README.md) and
[`.ai/decisions/collector-6.md`](../../.ai/decisions/collector-6.md).
