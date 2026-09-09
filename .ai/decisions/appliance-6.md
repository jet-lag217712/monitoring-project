## appliance - 6

### Primary Service

Deployments (On-Prem Appliance)

### Secondary Services

- SNMP Collector (`equate` CLI, setup TUI)
- GNS3 laboratory (`remote-server/`)

### Choice Made

Equate has two human workflows, both driven by the repository Makefile:

- Develop: `make appliance-bundle`, `make appliance-stage HOST=...`, `make appliance-configure` against an Equate-Appliance VM.
- Release: existing `appliance-package`, OVA, and Azure update-channel targets.

The customer/VM bundle copies only the runtime script allowlist in `appliance/scripts/runtime.manifest`. Operators on the VM use `equate`, not raw shell.

Removed unused Compose pathways (`deployments/end-to-end/`, `deployments/development/`, `deployments/production/cloud/`, `deployments/production/vxrail/`) and retired Terraform. The GNS3 lab stays under `remote-server/`. Collector setup accepts only the `appliance` profile.

### Alternatives Considered

- Keep Mac Docker development/end-to-end stacks as documented fixtures.
- Ship every `*.sh` file in the offline bundle.
- Retain `dev-vxrail` setup-profile aliases after deleting those trees.

### Trade-offs

Developers must stage onto a real appliance VM instead of a laptop Compose stack. Release packaging is smaller and matches the operator surface. Lab SNMP work uses GNS3 plus the appliance, not a split collector VM.

### Status

Accepted
