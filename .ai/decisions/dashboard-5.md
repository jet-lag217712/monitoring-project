## dashboard - 5

### Primary Service

Monitoring Dashboard

### Secondary Services

- Backend API
- Database

### Choice Made

Reuse `sites.location` as the shared site display-name alias. Operators rename from
the site detail page via `PATCH /api/sites/{siteId}` with `{ "location": "..." }`.
Empty / whitespace clears the alias (NULL) so projections fall back to `site_id`
through existing `LocationOrName`.

Collector string IDs (`sites.name` / API `site_id`) remain immutable and are shown
under the display title in smaller text. Ingestion continues to upsert only
`id`/`name`, so operator aliases are not overwritten.

`ogsd_api` receives `UPDATE` on `sites` so the API can persist aliases. Auth for
mutating requests stays the existing Google OIDC / appliance session + CSRF wrap.

### Alternatives Considered

- New `display_name` column separate from `location`
- Browser-local (`localStorage`) aliases only

### Trade-offs

- `location` continues to mean “friendly display label,” not a geographic address
- Any authenticated API user can rename; no finer-grained roles yet

### Status

Accepted
