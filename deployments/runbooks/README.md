# Deployment runbooks

Operational procedures for the local Equate appliance and its SNMP Collector v2 runtime.

| Runbook | Purpose |
|---------|---------|
| [install-and-validate.md](install-and-validate.md) | Appliance install, migrations, validation, and smoke |
| [inventory-and-discovery.md](inventory-and-discovery.md) | Static vs managed inventory, TUI/control, discovery |
| [credential-and-cert-rotation.md](credential-and-cert-rotation.md) | Communities, MQTT passwords, TLS CA |
| [queue-remediation.md](queue-remediation.md) | SQLite outbox depth, MQTT outage, corrupt DB |
| [rollback-and-restore.md](rollback-and-restore.md) | Image rollback, managed inventory, Postgres restore |
| [v2-cutover.md](v2-cutover.md) | V2-only production contract and emergency override |
| [field-acceptance-gns3.md](field-acceptance-gns3.md) | Manual GNS3/VMware lab acceptance checklist |

Decision record: [`.ai/decisions/collector-7.md`](../../.ai/decisions/collector-7.md).
