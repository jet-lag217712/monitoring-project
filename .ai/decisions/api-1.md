## api - 1

### Primary Service

Backend API

### Secondary Services

- Monitoring Dashboard
- PostgreSQL
- SNMP Collector setup (manifest authoring only)

### Choice Made

Cross-collector site reachability dependencies are modeled as
`upstream_site_ids` on the `sites` table, configured in the multi-site
manifest during `collector setup`, and synced into PostgreSQL by
`scripts/sync-site-topology.sh`.

Collectors remain unchanged. Device-level `upstream_device_ids` continue to
describe intra-collector reachability only. The backend API evaluates a
second site DAG at read time and projects:

- site-level unavailable upstreams and root-cause site IDs;
- optional device overlay from direct `critical` to `unknown` with reason
  `upstream_site_unreachable` when a downstream site is dependency-impacted.

Persisted collector health rows are not rewritten by the API.

Upstream site unavailable when configured hub devices are direct-critical, or
when any direct-critical device exists if no hubs are configured. Downstream
site dependency impact requires all configured upstream sites to be unavailable
and a strict majority of site devices to show failure evidence (direct
critical, or Unknown from an intra-collector / nested upstream outage).

Nested campus chains are `core → MDF → IDF`. Setup naming and API evaluation
complete empty IDF/MDF `upstream_site_ids` to that chain so an IDF outage
walks `IDF → MDF → core` and keeps the core as the only Critical root cause.

### Alternatives Considered

- Federate collectors with cross-site device references — rejected; violates
  collector-10 isolation and expands SNMP polling scope.
- Evaluate site dependencies in the dashboard — rejected; dashboard-3 requires
  API-side projection only.
- Auto-infer site topology from CDP — rejected; remains operator-defined.

### Trade-offs

- Topology changes require manifest edit + topology sync (or reconfigure).
- Site overlay is a read projection; historical DB health rows may differ from
  API responses during upstream outages.
- No visual topology graph in this iteration.

### Status

Accepted
