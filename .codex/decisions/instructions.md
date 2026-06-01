# Historical Instructions

## Purpose

Document architectural and implementation decisions that impact the system.

Store all decisions in:

```text
.codex/decisions/
```
---

## Naming Convention

Create a new file for every decision:

```text
{service}-{decision-number}.            md
```

Examples:

```text
ingestion-1.md
database-2.md
backendapi-1.md
```

---

## Service Mapping

| Service                    | Prefix       |
| -------------------------- | ------------ |
| SNMP Collector             | collector    |
| Monitoring Dashboard       | dashboard    |
| Backend API                | backendapi   |
| Ingestion Service          | ingestion    |
| Database                   | database     |
| Deployments (AWS)          | awsdeploy    |
| Deployments (Docker)       | dockerdeploy |
| Infrastructure (Docker)    | dockerinfra  |
| Infrastructure (AWS, Prod) | awsprodinfra |
| Infrastructure (AWS, Dev)  | awsdevinfra  |

Use the most specific category available.

---

## Required Format

```markdown
## ingestion - 1

### Primary Service
Ingestion Service

### Secondary Services
- Deployments (AWS)

### Choice Made
Docker-based EC2 Deployment

### Alternatives Considered
- AWS Lambda
- ECS Fargate

### Pros
- No cold starts
- Predictable cost

### Cons
- Requires instance management

### Cost / Benefit
Cheaper and simpler than Lambda for a continuously active workload.

### Status
Accepted
```

---

## Notes

* Create a new file for every decision.
* Do not modify historical decisions.
* If a decision changes, create a new record that supersedes the previous one.
* Keep entries concise and focused on the reasoning behind the choice.
