# Production local appliance

The production product is a complete on-premises appliance for a
VxRail- or VMware-type virtual machine. Every Equate service runs locally:

```text
SNMP devices → per-site collectors → local MQTT/TLS → ingestion → PostgreSQL
                                                               ↓
                                                       API → nginx → UI
```

Use [`appliance/`](appliance/) for the runtime. The appliance setup TUI creates
the site manifest and generated collector services; operators then use the
collector TUI for inventory, discovery, thresholds, dependencies, transport,
and reload operations.

## Runtime requirements

- Debian 12 minimal guest or an equivalent supported appliance base
- VMware-compatible virtual machine with one virtual NIC
- Initial target: 4 vCPU, 8 GB RAM, 64 GB disk
- Layer-2/L3 reachability from the guest to the monitored SNMP networks
- No Internet connection required after the release bundle is staged

The VM publishes only TCP 80/443. Internal service ports and control sockets
remain private. Generated database, MQTT, TLS, and appliance credentials are
kept under runtime/state directories and are never supplied as release inputs.

## Source checkout

For a local source validation run:

```bash
cd deployments/production/appliance
./bootstrapper.sh
```

For customer installation, use the checksummed offline release and follow
[`docs/releases/appliance-ova.md`](../../docs/releases/appliance-ova.md).

## Operator entry points

```bash
# First boot or deliberate reconfiguration
./bootstrapper.sh --reconfigure
sudo equate configure              # full wizard
sudo equate configure --sites      # sites/SNMP/thresholds only
sudo equate configure --users      # user management only
sudo equate configure --temperature 80  # global temperature warning only

# Day-2 collector configuration, per site
equate view <site-id>
equate sites                       # list sites
sudo equate sites delete <site-id> # remove one site (type site-id to confirm, or --yes)

# Day-2 user administration
equate users list
equate users create <username>
```

The setup TUI must be completed before the stack is started. A failed
validation or reload leaves the last active configuration in place.
