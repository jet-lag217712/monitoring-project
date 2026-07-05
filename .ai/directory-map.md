# Directory Structure of Project

## Project Structure
```
equate-ogsd/
├── .ai/
├── .github/
├── docs/
│   ├── architecture/
│   ├── diagrams/
│   ├── decisions/
│   └── system-design.pdf
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
│   │   ├── mqtt-broker/
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
│   ├── docker-compose/
│   ├── aws/
│   └── local/
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