# Frontend Architecture

This frontend is a Vite + React single-page dashboard. It does not use a routing library yet. Instead, the current screen is controlled by React state: the app shows the sites overview until a site is selected, then it shows that site's detail page.

## Where to Start

Start with `src/App.jsx`. It is intentionally small:

- It calls `useNetworkDashboard()` to get all dashboard state and actions.
- It renders `AppShell` for the page frame.
- It renders `DashboardPage` for the current dashboard screen.

After that, read `src/hooks/useNetworkDashboard.js`. That hook explains the data flow: load site data, fall back to the active demo scenario, track search text, track the selected site, and poll for updates.

## Folder Structure

### `src/config`

Configuration values live here. These files should be obvious places to change app-wide settings.

- `demo.js` contains `ACTIVE_DEMO`. This is the only value you need to edit to switch demo scenarios during development.
- `api.js` contains the API base URL, polling interval, and URL helper.

Keeping configuration out of components makes it easier to see which values are meant to be changed by developers.

### `src/data`

Mock data lives here. `mockData.js` defines the available demo scenarios and test config fallback data.

The dashboard currently supports these scenario IDs:

- `all-healthy`
- `two-caution`
- `one-red`

To change the active scenario, edit `src/config/demo.js`:

```js
export const ACTIVE_DEMO = 'two-caution'
```

The UI no longer exposes a scenario selector. Demo switching is now a developer configuration choice, not a runtime user action.

### `src/services`

API functions live here. `sitesApi.js` contains small functions for backend requests:

- `fetchSitesFromApi()`
- `fetchSiteDetailFromApi(siteId)`
- `fetchTestConfigFromApi()`

Future API calls should be added to this folder first. Components should not call `fetch()` directly. Keeping network code in services makes backend integration easier to test and change later.

### `src/hooks`

Stateful application logic lives here. `useNetworkDashboard.js` owns the dashboard's current state:

- site list
- alerts
- selected site
- selected site's detail data
- search query
- last updated time
- live/demo data mode
- polling behavior

This hook is the main state management layer. It keeps data loading and fallback logic out of the visual components.

### `src/utils`

Pure helper functions live here. `siteData.js` contains functions that transform or summarize data:

- normalize API/mock site responses into the shape used by components
- build alert banner data
- filter sites by search text
- calculate overview stats

These functions do not render UI and do not update React state.

### `src/layout`

Page-level layout components live here.

- `AppShell.jsx` renders the fixed navigation, alert banner, and main content area.
- `Nav.jsx` renders the top navigation bar.

Use this folder for components that frame the whole page rather than components that belong to a specific dashboard feature.

### `src/dashboard`

Dashboard-level screen composition lives here.

- `DashboardPage.jsx` decides whether to show the sites overview or the selected site detail.
- `PageHeader.jsx` renders reusable page header markup.
- `OverviewStats.jsx`, `StatCard.jsx`, and `LastUpdatedLabel.jsx` render the overview summary area.

This folder coordinates dashboard sections, but it avoids low-level table rows or individual device details.

### `src/sites`

Site-focused UI lives here.

- `SitesGrid.jsx` renders the all-sites overview content.
- `SiteCard.jsx` renders one site card.
- `SiteDetail.jsx` renders the selected site's detail page.

Use this folder for components whose main job is displaying campus/site information.

### `src/devices`

Device-focused UI lives here.

- `DeviceRow.jsx` renders one network device row inside the detail table.

Use this folder when adding components that describe individual switches, firewalls, wireless controllers, or other devices.

### `src/tables`

Table components live here.

- `DevicesTable.jsx` owns the table structure for device detail rows.

Use this folder for reusable table layouts. Row-specific rendering can stay closer to the feature, such as `src/devices`.

### `src/charts`

Chart components should live here when charts are added. There are no real charts yet, but this folder marks the intended home for future Recharts or utilization history components.

### `src/alerts`

Alert-specific UI lives here.

- `AlertBanner.jsx` renders the fixed banner shown when one or more sites need attention.

### `src/common`

Small reusable UI components live here. These components are intentionally generic and used across multiple feature folders.

Current examples:

- `BackButton.jsx`
- `LoadingSkeleton.jsx`
- `SearchBar.jsx`
- `StatusBadge.jsx`
- `UtilizationBar.jsx`

Only put a component in `common` if more than one feature can reasonably reuse it.

## Data Flow

1. `App.jsx` calls `useNetworkDashboard()`.
2. `useNetworkDashboard()` initializes state from the active demo scenario in `src/config/demo.js`.
3. The hook tries to load live data through `src/services/sitesApi.js`.
4. If live requests fail, the hook falls back to the active mock scenario in `src/data/mockData.js`.
5. The hook normalizes and summarizes data with helpers from `src/utils/siteData.js`.
6. `App.jsx` passes the dashboard state into `AppShell` and `DashboardPage`.
7. `DashboardPage` renders either `SitesGrid` or `SiteDetail`.
8. User actions, such as searching or clicking a site, call handlers from `useNetworkDashboard()`.

## Routing

There is no URL-based routing yet. The app uses `selectedSite` state:

- `selectedSite === null` means the all-sites overview is visible.
- `selectedSite !== null` means the site detail screen is visible.

If the app later needs URLs like `/sites/school-a`, add a router near `src/App.jsx` and keep screen components in `src/dashboard` or feature folders.

## Future API Calls

Add future backend request functions to `src/services`. Then call those service functions from a hook in `src/hooks`.

Avoid calling APIs directly from display components like `SiteCard` or `DevicesTable`. Display components should receive data through props and focus on rendering.

## Future Authentication

Authentication is not implemented yet. When it is added, a beginner-friendly structure would be:

- `src/auth` for login/logout components and auth-specific helpers
- `src/services/authApi.js` for auth network requests
- `src/hooks/useAuth.js` for current user/session state

If auth affects every request, add token/header logic inside the services layer rather than inside each component.

## Development Demo Configuration

The active demo scenario is configured in exactly one place:

```js
// src/config/demo.js
export const ACTIVE_DEMO = 'all-healthy'
```

Change that value to another scenario ID and restart or let Vite hot reload. No UI selector is rendered.
