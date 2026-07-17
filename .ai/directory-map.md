# Directory Structure of Project

## Project Structure
```
monitoring-dashboard/
├── .ai/
├── .github/
├── docs/
│   ├── architecture/
│   └── diagrams/
│       └── system-design.md
│
├── services/
│   ├── snmp-collector/
│   │   ├── cmd/
│   │   ├── internal/          # config, inventory, health DAG, discovery, TUI/socket,
│   │   │                      # SQLite outbox, SNMP core + vendors/{cisco,arista}
│   │   ├── configs/
│   │   ├── data/
│   │   ├── deployments/
│   │   ├── tests/
│   │   ├── Dockerfile
│   │   ├── go.mod
│   │   └── README.md
│   │
│   ├── ingestion-service/     # v1/v2 validation, dedup, transactional persistence
│   │   ├── cmd/
│   │   ├── internal/
│   │   ├── configs/
│   │   ├── tests/
│   │   ├── Dockerfile
│   │   └── README.md
│   │
│   └── backend-api/
│       ├── cmd/
│       ├── internal/
│       ├── configs/
│       ├── tests/
│       ├── Dockerfile
│       └── README.md
│
├── frontend/
│   ├── src/
│   ├── assets/
│   ├── docs/
│   ├── Dockerfile
│   ├── index.html
│   ├── package.json
│   └── vite.config.js
│
├── infrastructure/
│   ├── terraform/
│   │   ├── environments/
│   │   │   ├── dev/
│   │   │   └── prod/
│   │   └── modules/
│   │
│   ├── docker/
│   │   ├── mqtt-broker/
│   │   ├── postgres/
│   │   └── local-dev/
│   └── script/
│
├── database/
│   ├── migrations/
│   ├── schema/
│   └── seed/
│
├── deployments/
│   ├── end-to-end/            # Single-host: all services including collector
│   ├── development/           # Mac cloud Compose + development/vxrail (OrbStack/GNS3)
│   │   ├── up.sh / down.sh
│   │   ├── docker-compose.yml
│   │   └── vxrail/            # Collector on VM → Mac Mosquitto (GNS3 Cloud)
│   ├── production/            # Hybrid skeleton (no Terraform yet)
│   │   ├── cloud/             # Azure VM Compose
│   │   └── vxrail/            # On-site collector → Azure Mosquitto
│   ├── lib/                   # Shared shell helpers
│   └── test.sh                # Aggregate validation runner
│
├── remote-server/
│   └── configurations/
│
├── AGENTS.md
├── README.md
└── .gitignore
```

## .ai Structure

```
.ai/
├── project-context/
│   ├── architecture.md
│   ├── aws-deployment.md
│   ├── data-flow.md
│   ├── monitoring-requirements.md
│   ├── network-topology.md
│   └── service-boundaries.md
│
├── standards/
│   ├── coding-standards.md
│   ├── golang-standards.md
│   ├── react-standards.md
│   ├── api-standards.md
│   ├── database-standards.md
│   └── security-standards.md
│
├── decisions/
│   ├── awsdeploy-1.md
│   ├── collector-1.md
│   ├── collector-2.md
│   ├── collector-3.md
│   ├── collector-4.md
│   ├── dashboard-1.md
│   ├── dashboard-2.md
│   └── instructions.md
│
├── roadmap/
│   ├── mvp-implementation-plan.md  # historical baseline
│   └── snmp-collector-v2.md        # current roadmap authority
└── directory-map.md
```

## Deployment profiles

Three deployment profiles under [`deployments/`](../deployments/):

- **`deployments/end-to-end/`** — one Compose project with every service (including collector) for client-site smoke; no SNMP simulator
- **`deployments/development/`** — Mac cloud Compose (Mosquitto, Postgres, apps) + [`development/vxrail/`](../deployments/development/vxrail/) on OrbStack Ubuntu VM (GNS3 Cloud → Mac MQTT)
- **`deployments/production/`** — hybrid skeleton: Azure cloud Compose + on-site VxRail collector (Terraform deferred)

Do not add phase-named stacks under these profiles. Extend `end-to-end/`, `development/`, or `production/` instead.

## SNMP Collector v2 documentation

The v2 roadmap is [`.ai/roadmap/snmp-collector-v2.md`](roadmap/snmp-collector-v2.md). The versioned wire contract is [`docs/architecture/contracts.md`](../docs/architecture/contracts.md). Collector implementation keeps core MIB work under `services/snmp-collector/internal/snmp/core/`, vendor profiles under `internal/snmp/vendors/{cisco,arista}/`, and separates runtime configuration/inventory, discovery, health dependency evaluation, local TUI/control, and durable outbox concerns.
