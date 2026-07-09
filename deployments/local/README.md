# Local deployments

## `test-env/` — canonical local test stack

All local integration and E2E testing uses [`test-env/`](test-env/).

```bash
./deployments/local/test-env/up.sh
./deployments/local/test-env/down.sh
```

Do not create new phase-named directories under `deployments/local/` for testing. Extend `test-env` instead.

## `snmpsim/`

Shared SNMP simulator image and sample data, built by the `test-env` compose stack.
