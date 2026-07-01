# Frontend Refactor Change Log

This document explains the refactor without requiring a git diff.

## Changed Files

### `src/App.jsx`

What changed:

- Removed polling, API fallback, search filtering, selected-site state, and demo selector state from the component.
- Kept `App` as the small application entry component.
- Added `useNetworkDashboard()`, `AppShell`, and `DashboardPage`.

Why it changed:

- `App.jsx` had become the main monolith. Moving state and UI composition into focused files makes the application easier for a beginner React developer to follow.

### `src/index.css`

What changed:

- Removed `.mode-strip` and `.mode-chip` styles.
- Kept existing colors, spacing, layout, card, table, nav, alert, and responsive styles unchanged.

Why it changed:

- The runtime demo selector was removed from the UI, so its unused styles were removed too.

## Created Files

### `docs/frontend-architecture.md`

What it contains:

- A beginner-friendly explanation of the frontend architecture, folder responsibilities, data flow, demo configuration, future API location, and future auth location.

Why it exists:

- Documentation was the most important requirement for this refactor.

### `docs/frontend-refactor.md`

What it contains:

- This file-by-file change log.

Why it exists:

- It explains the refactor at a practical level without requiring the reader to inspect a diff.

### `src/config/api.js`

What it contains:

- `API_BASE_URL`
- `POLL_INTERVAL_MS`
- `apiUrl(path)`

Why it exists:

- API configuration is now separate from UI and state logic.

### `src/config/demo.js`

What it contains:

- `ACTIVE_DEMO`

Why it exists:

- This is the single obvious place to change the active development demo scenario.

### `src/data/mockData.js`

What it contains:

- Base mock site and device data.
- Demo scenarios: `all-healthy`, `two-caution`, and `one-red`.
- `mockTestConfig`.

Why it exists:

- Mock data is now separated from application state and UI components.

### `src/utils/siteData.js`

What it contains:

- `normalizeSites(data)`
- `buildAlerts(list)`
- `filterSitesBySearch(sites, searchQuery)`
- `getSiteStats(sites)`

Why it exists:

- Data transformation and summary logic can now be read and tested separately from React components.

### `src/services/sitesApi.js`

What it contains:

- Small API request functions for sites, site detail, and test config.

Why it exists:

- Components and hooks no longer need to know raw endpoint paths or `fetch()` details.

### `src/hooks/useNetworkDashboard.js`

What it contains:

- Dashboard state.
- Polling behavior.
- Live API loading.
- Active demo fallback.
- Search state.
- Selected-site navigation state.

Why it exists:

- This centralizes state management while keeping UI components mostly presentational.

### `src/layout/AppShell.jsx`

What it contains:

- The app frame: nav, alert banner, and main content region.

Why it exists:

- Layout concerns are now separate from dashboard screen logic.

### `src/layout/Nav.jsx`

What it contains:

- The fixed top navigation bar.

Why it exists:

- The original `Nav.jsx` was moved into the `layout` folder because it frames the page.

### `src/alerts/AlertBanner.jsx`

What it contains:

- The fixed alert banner.

Why it exists:

- Alert UI now has a clear home instead of living at the root of `src`.

### `src/dashboard/DashboardPage.jsx`

What it contains:

- The simple state-based screen switch between the sites overview and site detail.

Why it exists:

- It makes the app's current "routing" behavior easy to find.

### `src/dashboard/PageHeader.jsx`

What it contains:

- Reusable page header markup for eyebrow, title, subtitle, and optional right-side content.

Why it exists:

- Header JSX was extracted from `SitesGrid` to keep the grid component smaller.

### `src/dashboard/LastUpdatedLabel.jsx`

What it contains:

- The small "Updated ..." label.

Why it exists:

- This keeps timestamp formatting markup out of the larger overview component.

### `src/dashboard/StatCard.jsx`

What it contains:

- One summary statistic card.

Why it exists:

- The previous `StatCard` helper lived inside `SitesGrid`. Extracting it makes it reusable and easier to locate.

### `src/dashboard/OverviewStats.jsx`

What it contains:

- The four-card dashboard stat strip.

Why it exists:

- Summary stats are dashboard-level UI, separate from the sites grid.

### `src/common/SearchBar.jsx`

What it contains:

- The reusable search input used on the sites overview.

Why it exists:

- Search input JSX is isolated from the grid layout.

### `src/common/StatusBadge.jsx`

What it contains:

- `SiteStatusBadge`
- `DeviceStatusBadge`

Why it exists:

- Both site cards and device rows use status badges. One shared file prevents duplicated badge logic.

### `src/common/UtilizationBar.jsx`

What it contains:

- `MiniBar`
- `UtilizationBar`
- shared utilization color classification logic

Why it exists:

- CPU/utilization bars were duplicated across site cards and detail rows.

### `src/common/BackButton.jsx`

What it contains:

- The reusable "All Sites" back button.

Why it exists:

- It keeps navigation button markup out of `SiteDetail`.

### `src/common/LoadingSkeleton.jsx`

What it contains:

- The loading skeleton used while site detail data is empty.

Why it exists:

- Loading UI now has a small named component.

### `src/sites/SitesGrid.jsx`

What it contains:

- The all-sites overview content: header, stats, search, cards, and empty state.

Why it exists:

- Site overview UI now lives in a `sites` feature folder.

### `src/sites/SiteCard.jsx`

What it contains:

- One clickable site card.

Why it exists:

- The original root-level file was moved into the `sites` folder and now reuses common badge/bar components.

### `src/sites/SiteDetail.jsx`

What it contains:

- The selected site's detail page header and device table.

Why it exists:

- The original root-level file was moved into the `sites` folder and split so table/row details are not embedded in the page component.

### `src/tables/DevicesTable.jsx`

What it contains:

- The device table structure and empty-table state.

Why it exists:

- Table markup is now separated from the site detail page.

### `src/devices/DeviceRow.jsx`

What it contains:

- One network device row.

Why it exists:

- Device-specific rendering now has a clear file instead of being inline inside `SiteDetail`.

### `src/charts/README.md`

What it contains:

- A note that future chart components belong in `src/charts`.

Why it exists:

- The project has chart-related intent but no chart implementation yet. This creates a clear home without inventing unnecessary chart abstractions.

## Deleted Files

### `src/Nav.jsx`

Why it was removed:

- Replaced by `src/layout/Nav.jsx` so layout components live together.

### `src/AlertBanner.jsx`

Why it was removed:

- Replaced by `src/alerts/AlertBanner.jsx` so alert UI has a clear feature folder.

### `src/SitesGrid.jsx`

Why it was removed:

- Replaced by `src/sites/SitesGrid.jsx` and smaller dashboard/common components.
- The runtime demo selector was removed from this UI.

### `src/SiteCard.jsx`

Why it was removed:

- Replaced by `src/sites/SiteCard.jsx` with shared badge and utilization components.

### `src/SiteDetail.jsx`

Why it was removed:

- Replaced by `src/sites/SiteDetail.jsx`, `src/tables/DevicesTable.jsx`, and `src/devices/DeviceRow.jsx`.

### `src/mockData.js`

Why it was removed:

- Replaced by `src/data/mockData.js` so mock data is separated from root-level source files.
