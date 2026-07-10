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
│   ├── local/                 # Mac cloud Compose + local/vxrail (Debian collector → Mac MQTT)
│   │   ├── up.sh / down.sh
│   │   ├── docker-compose.yml
│   │   ├── vxrail/            # Collector → Mac Mosquitto (day-to-day test)
│   │   └── snmpsim/           # Optional standalone fixture (not default stack)
│   └── dev/                   # Azure dual-plane (later)
│       ├── vxrail/            # Collector → Azure Mosquitto
│       └── cloud/             # Azure runbook (no Compose)
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

## Local testing environments

- **Day-to-day:** Mac [`deployments/local/`](../deployments/local/) Compose (Mosquitto, Postgres, apps) + [`deployments/local/vxrail/`](../deployments/local/vxrail/) on the Debian VM (GNS3 + collector → Mac MQTT)
- **Azure path (later):** [`deployments/dev/`](../deployments/dev/) — VxRail → Azure Mosquitto + Azure cloud runbook

Do not add phase-named stacks under `deployments/local/` (for example `phase2/`). Extend `local/` or `dev/` instead.
