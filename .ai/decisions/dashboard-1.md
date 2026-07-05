## dashboard - 1

### Primary Service
Monitoring Dashboard

### Secondary Services
- None

### Choice Made
Lift interface selection state out of `DeviceDetail` into `useNetworkDashboard`, keyed per device by interface `name` or `if_index` (not array index).

### Alternatives Considered
- Local `useState` inside `DeviceDetail` (default first interface on mount)
- Key selection by array index only

### Pros
- Selection survives demo-data polling refreshes without resetting to the first interface
- Per-device map allows returning to a device and restoring the last viewed interface
- `name` / `if_index` keys remain stable if mock interface ordering changes

### Cons
- Hook surface grows slightly (`selectedInterfaceByDevice`, `handleInterfaceSelect`)
- Requires `if_index` on mock interface records for a stable numeric key option

### Cost / Benefit
Small increase in hook complexity in exchange for predictable selection behavior and alignment with how device/site navigation is already centralized in `useNetworkDashboard`.

### Status
Accepted
