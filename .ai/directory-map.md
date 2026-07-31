# Equate repository map

This repository is for one product: the local Equate monitoring appliance.
The supported production shape is one VMware-compatible VM with all Equate
services running locally. The `/.ai/` directory is the canonical source for
architecture, decisions, standards, and roadmap documentation.

```text
monitoring-dashboard/
├── .ai/                         canonical project guidance
├── deployments/production/
│   └── appliance/               supported local appliance Compose runtime
├── deployments/update-channel/  Azure Blob channel manifest examples/schema
├── deployments/end-to-end/      single-host source validation
├── deployments/development/     developer integration fixture
├── deployments/runbooks/        operator procedures
├── appliance/scripts/           release, VM, OVA, .eqa package/publish scripts
├── appliance/keys/              update-signing public key (no private keys)
├── services/
│   ├── snmp-collector/           polling, TUI, outbox, equate upgrade, health evidence
│   ├── ingestion-service/        validation and PostgreSQL writes
│   └── backend-api/              read-only REST contracts
├── frontend/                    local dashboard
├── infrastructure/docker/       local Mosquitto and PostgreSQL images
├── database/                    schema, seeds, and migrations
├── docs/                        architecture, contracts, schemas, and release docs
└── remote-server/               GNS3/VMware laboratory network fixtures
```

## `.ai` structure

```text
.ai/
├── project-context/              current product architecture and boundaries
├── standards/                    implementation and security standards
├── decisions/                    accepted design decisions
├── roadmap/                      implementation sequencing
└── directory-map.md              this map
```

## Supported runtime boundaries

The appliance contains per-site collectors, local MQTT/TLS, ingestion,
PostgreSQL, Backend API, nginx, and the React dashboard. Collectors reach
monitored SNMP networks through the VM route table. The TUI reaches each
collector through an owner-only Unix socket. Only nginx publishes 80/443.

The old split deployment directories and retired infrastructure files are not
product architecture. Do not add documentation that presents them as customer
installation options.
