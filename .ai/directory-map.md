# Directory Structure of Project

## Project Structure
```
equate-ogsd/
├── .ai/
├── .github/
├── docs/
│   ├── architecture/
│   ├── decisions/
│   └── diagrams/
│       └── system-design.md
│
├── services/
│   ├── snmp-collector/
│   │   ├── cmd/
│   │   ├── internal/
│   │   ├── configs/
│   │   ├── deployments/
│   │   ├── tests/
│   │   ├── Dockerfile
│   │   ├── go.mod
│   │   └── README.md
│   │
│   ├── ingestion-service/
│   │   ├── cmd/
│   │   ├── internal/
│   │   ├── configs/
│   │   ├── deployments/
│   │   ├── tests/
│   │   ├── Dockerfile
│   │   └── README.md
│   │
│   └── backend-api/
│       ├── cmd/
│       ├── internal/
│       ├── configs/
│       ├── deployments/
│       ├── tests/
│       ├── Dockerfile
│       └── README.md
│
├── frontend/
│   └── monitoring-dashboard/
│       ├── src/
│       ├── assets/
│       ├── Dockerfile
│       ├── index.html
│       ├── package.json
│       └── vite.config.js
│
├── infrastructure/
│   ├── terraform/
│   │   ├── environments/
│   │   │   ├── dev/
│   │   │   └── prod/
│   │   └── modules/
│   │
│   ├── docker/
│   │   ├── telemetry-transport/
│   │   ├── postgres/
│   │   └── local-dev/
│   │
│   └── scripts/
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
├── remote-servers/
│   ├── configs/
│   └── keys/
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
│   └── instructions.md
│
├── prompts/
│   ├── backend-engineer.md
│   ├── frontend-engineer.md
│   └── reviewer.md
│
├── roadmap/
│   ├── mvp.md
│   ├── phase-2.md
│   └── backlog.md
└── directory-map.md
```

## Deployment profiles

Three deployment profiles under [`deployments/`](../deployments/):

- **`deployments/end-to-end/`** — one Compose project with every service (including collector) for client-site smoke; no SNMP simulator
- **`deployments/development/`** — Mac cloud Compose (Mosquitto, Postgres, apps) + [`development/vxrail/`](../deployments/development/vxrail/) on OrbStack Ubuntu VM (GNS3 Cloud → Mac MQTT)
- **`deployments/production/`** — hybrid skeleton: Azure cloud Compose + on-site VxRail collector (Terraform deferred)

Do not add phase-named stacks under these profiles. Extend `end-to-end/`, `development/`, or `production/` instead.
