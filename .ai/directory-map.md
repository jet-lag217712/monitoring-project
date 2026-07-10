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
│   ├── local/                 # Mac cloud Compose + local/vxrail (Debian → Mac MQTT)
│   │   ├── up.sh / down.sh
│   │   ├── docker-compose.yml
│   │   ├── vxrail/            # Collector → Mac Mosquitto (GNS3 day-to-day)
│   │   └── snmpsim/           # Optional standalone fixture (not default stack)
│   ├── local-physical/        # Pre-client E2E: Mac collector → physical SNMP
│   │   └── vxrail/            # Host go run → tls://127.0.0.1:8883 (reuses local cloud)
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

Three deployment plants:

- **`deployments/local/`** — Mac Compose (Mosquitto, Postgres, apps) + [`local/vxrail/`](../deployments/local/vxrail/) on Debian VM (GNS3 → Mac MQTT)
- **`deployments/local-physical/`** — same Mac cloud stack + Mac host collector against a **physical** network (pre-client E2E)
- **`deployments/dev/`** — Azure cloud runbook + VxRail → Azure MQTT

Do not add phase-named stacks under `deployments/local/` (for example `phase2/`). Extend `local/`, `local-physical/`, or `dev/` instead.
