# Development integration fixture

This directory supports developer testing of the local service graph before an
offline appliance release. It is not a customer deployment profile. The
production target remains the single-VM appliance in
[`../production/appliance/`](../production/appliance/).

The fixture can run the application services in one local Compose project and
the collector in a VxRail-like Linux VM connected to a GNS3 lab. Both sides are
local test infrastructure; no remote service is part of the product path.

## Application side

```bash
cp deployments/development/.env.example deployments/development/.env
./deployments/development/up.sh
./deployments/development/validate.sh
./deployments/development/smoke.sh
```

## Collector VM side

```bash
./deployments/development/vxrail/sync.sh
# On the Linux VM:
./bootstrap.sh
```

The setup TUI creates the local site manifest and collector artifacts. Run the
day-2 collector TUI inside the collector container when the VM bind mount does
not allow the host terminal to open the Unix socket:

```bash
docker compose -f docker-compose.yml -f docker-compose.sites.generated.yml \
  exec -it snmp-collector-site-001 \
  /collector tui -socket /run/snmp-collector/control.sock -theme auto
```

Use [`../runbooks/field-acceptance-gns3.md`](../runbooks/field-acceptance-gns3.md)
for the manual test checklist.
