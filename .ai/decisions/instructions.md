# Decision record instructions

Decision records document choices that affect Equate architecture or
implementation. Store them in `.ai/decisions/` with one file per decision.

## Naming

Use the most specific service prefix and an incrementing number:

```text
collector-11.md
dashboard-4.md
appliance-2.md
database-1.md
```

## Required format

```markdown
## collector - 11

### Primary Service
SNMP Collector

### Secondary Services
- Ingestion Service
- Local appliance deployment

### Choice Made
Short statement of the accepted local design.

### Alternatives Considered
- Alternative one
- Alternative two

### Trade-offs
Operational and technical consequences.

### Status
Accepted
```

Record local appliance decisions, TUI behavior, service boundaries, data
contracts, and security controls. Do not introduce external deployment options or
remote configuration assumptions into a decision record.
