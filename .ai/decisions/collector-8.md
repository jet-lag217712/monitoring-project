## collector - 8

### Primary Service

SNMP Collector

### Secondary Services

- None (local operator plane only)

### Choice Made

Replace the Phase 6 raw key/value Bubble Tea stub with an Equate-branded
operator TUI that mirrors the Network Dashboard visual language from
`frontend/src/index.css`.

#### Visual system

- Brand mark renders as salmon `//` plus the `Equate` wordmark (logo.svg is two
  diagonal salmon strokes).
- Light palette uses the dashboard tokens verbatim (`#f8f7f2` bg, `#0a0a0a` ink,
  `#e8735a` salmon, status greens/ambers/reds/slate).
- Dark palette is derived for dark terminals: inverted backgrounds/ink, unchanged
  salmon accent and status colors.
- Theme selection: `-theme auto|light|dark` (default `auto` via terminal
  background detection). `NO_COLOR` forces monochrome styles.

#### Interaction model

- Structured tables/panels replace `%v` map dumps.
- Tabs for inventory, device, discovery, thresholds, transport, and config.
- Auto-refresh every 5s, viewport scrolling, spinner while loading.
- Threshold edits use text input; dependency overlays are editable from the
  inventory view. Mutations still use revision-bound prepare/commit + reload.

#### Non-goals

- No control-protocol changes.
- No public HTTP management surface.
- No credential display.

### Alternatives Considered

- Keep the stub TUI and polish only the help line — rejected; operator UX at
  customer sites needs branded, readable views.
- Embed a web UI locally — rejected; collector-6 requires local Unix-socket
  Bubble Tea only.

### Pros

- Matches dashboard brand for field operators.
- Readable structured views without changing the control plane.
- Adaptive theme works on light and dark terminals.

### Cons

- Derived dark palette is not yet in `frontend/src/index.css` (dashboard remains
  light-only); TUI dark tokens must stay in sync manually if brand colors change.

### Cost / Benefit

Modest Charm/bubbles dependency growth for a large operator-experience gain at
customer deployments.

### Status

Accepted
