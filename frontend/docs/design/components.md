# Component Design Specifications

> **For AI agents:** This file documents every UI component in the dashboard — its visual anatomy, CSS class names, states, and exact styling behavior. When building new components or extending existing ones, match these specifications exactly to maintain visual consistency.

---

## Component Index

1. [Nav](#1--nav)
2. [Alert Banner](#2--alert-banner)
3. [Page Header](#3--page-header)
4. [Stat Card & Stat Strip](#4--stat-card--stat-strip)
5. [Search Bar](#5--search-bar)
6. [Site Card](#6--site-card)
7. [Status Badge](#7--status-badge)
8. [Utilization Bar (MiniBar + UtilizationBar)](#8--utilization-bar)
9. [Sites Grid](#9--sites-grid)
10. [Empty State](#10--empty-state)
11. [Back Button](#11--back-button)
12. [Site Detail Header](#12--site-detail-header)
13. [Devices Table](#13--devices-table)
14. [Device Row](#14--device-row)
15. [Loading Skeleton](#15--loading-skeleton)
16. [Eyebrow Label](#16--eyebrow-label)

---

## 1 · Nav

**File:** `src/layout/Nav.jsx`  
**CSS class:** `nav` (HTML `<nav>` element)

### Visual Anatomy

```
┌─────────────────────────────────────────────────────────────┐
│ ⟋⟋  Network Dashboard               4 sites · 12 devices   │
└─────────────────────────────────────────────────────────────┘
   ↑                                   ↑
 .nav-logo                          .nav-right
 (.logo-mark + wordmark text)       (meta text, JetBrains Mono)
```

### Layout
- `position: fixed; top: 0; left: 0; right: 0; z-index: 100`
- `display: flex; align-items: center; justify-content: flex-start`
- `height: 60px`
- `padding: 0 48px` (desktop), `0 20px` (≤900px)
- `background: var(--bg)`
- `border-bottom: 1px solid var(--border)`

### `.nav-logo`
- `font-family: 'Epilogue', sans-serif; font-weight: 800; font-size: 1.0rem`
- `color: var(--ink); letter-spacing: -0.02em`
- `display: flex; align-items: center; gap: 10px`
- `cursor: pointer` (clicking navigates back to all-sites view)
- The logo mark wraps in `.logo-mark` (30×30px, `display: grid; place-items: center`)

### `.nav-right`
- `margin-left: auto` (pushes to far right)
- `font-family: 'JetBrains Mono', monospace; font-size: 0.72rem; color: var(--ink-muted); letter-spacing: 0.04em`
- Content format: `{n} sites · {n} network devices`
- The `·` separator uses class `.nav-sep` with `color: var(--border-strong)`

### Transitions
- `background` and `border-color` both transition `0.3s ease` (reserved for potential dark mode)

---

## 2 · Alert Banner

**File:** `src/alerts/AlertBanner.jsx`  
**CSS class:** `.alert-banner`

### Behavior
- Renders `null` when there are no active alerts
- When alerts are present, it renders a fixed banner immediately below the nav

### Layout
- `position: fixed; top: 60px; left: 0; right: 0; z-index: 99`
- `padding: 10px 48px` (desktop), `10px 20px` (≤900px)
- `display: flex; align-items: center; gap: 10px`
- `background: #fef2f2; border-bottom: 1px solid #fecaca`

### Typography
- `font-size: 0.82rem; font-weight: 500; color: #b91c1c`
- Strong "Alert:" prefix in bold

### Alert Dot
- Class: `.alert-dot`
- `7px × 7px; border-radius: 50%; background: #ef4444`
- Animated: `animation: blink 1.2s ease-in-out infinite`

### Effect on layout
When the banner is visible, `AppShell` overrides the page-content `padding-top` from `88px` to `124px` (via inline style) to prevent content from being hidden beneath both nav (60px) and banner (~40px).

---

## 3 · Page Header

**File:** `src/dashboard/PageHeader.jsx`  
**CSS class:** `.page-header`

### Visual Anatomy

```
┌─────────────────────────────────────────┬──────────────────┐
│ • NETWORK DASHBOARD                     │  right content   │
│ All Sites                               │  (e.g. timestamp)│
│ Subtitle text here                      │                  │
└─────────────────────────────────────────┴──────────────────┘
```

### Layout
- `.page-header`: `display: flex; align-items: flex-end; justify-content: space-between; gap: 20px; margin-bottom: 32px`
- `.page-header-left` contains the eyebrow, title, and subtitle

### Eyebrow (`.page-eyebrow`)
- `display: inline-flex; align-items: center; gap: 7px`
- `font-family: 'JetBrains Mono', monospace; font-size: 0.68rem; font-weight: 600`
- `letter-spacing: 0.08em; text-transform: uppercase; color: var(--salmon)`
- `margin-bottom: 10px`
- Contains an `.eyebrow-dot` (5×5px, salmon, animated pulse) followed by the eyebrow text

### Eyebrow Dot (`.eyebrow-dot`)
- `5px × 5px; border-radius: 50%; background: var(--salmon)`
- `animation: pulse 2s ease-in-out infinite`
- Pulse keyframes: scale + opacity (1→0.7, 1→0.4 at 50%)

### Title (`.page-title`, rendered as `<h1>`)
- `font-size: clamp(1.6rem, 2.5vw, 2.2rem); font-weight: 800`
- `letter-spacing: -0.035em; color: var(--ink); line-height: 1.1`

### Subtitle (`.page-sub`, rendered as `<p>`)
- `font-size: 0.88rem; color: var(--ink-muted); margin-top: 6px; line-height: 1.6`

### Right Content
Any React node passed as `rightContent` prop renders on the right side, aligned to `flex-end`. Currently used for the `<LastUpdatedLabel>`.

---

## 4 · Stat Card & Stat Strip

**Files:** `src/dashboard/StatCard.jsx`, `src/dashboard/OverviewStats.jsx`  
**CSS classes:** `.stat-strip`, `.stat-card`, `.stat-label`, `.stat-value`

### Stat Strip Layout
- `.stat-strip`: `display: grid; grid-template-columns: repeat(4, 1fr); gap: 12px; margin-bottom: 32px`
- Responsive: 2-col at ≤900px, 1-col at ≤540px

### Stat Card
- `.stat-card`: `background: var(--bg-2); border: 1px solid var(--border); border-radius: var(--radius); padding: 16px 18px`
- `transition: background 0.3s ease`

### Stat Label
- `.stat-label`: `font-family: 'JetBrains Mono', monospace; font-size: 0.68rem; font-weight: 600`
- `text-transform: uppercase; letter-spacing: 0.07em; color: var(--ink-muted); margin-bottom: 8px`

### Stat Value
- `.stat-value`: `font-size: 1.6rem; font-weight: 800; letter-spacing: -0.04em; color: var(--ink); line-height: 1`
- Color modifiers via `tone` prop:
  - `.stat-value.alert` → `color: var(--status-alert)` (red, used when count > 0)
  - `.stat-value.caution` → `color: var(--status-caution)` (amber, used when count > 0)
  - `.stat-value.ok` → `color: var(--status-ok)` (green)

### The Four Cards (from `OverviewStats.jsx`)
1. **Total sites** — value: site count, no tone modifier
2. **Network devices** — value: device count, no tone modifier
3. **Critical** — value: alert count, tone `'alert'` if > 0
4. **Caution** — value: caution count, tone `'caution'` if > 0

---

## 5 · Search Bar

**File:** `src/common/SearchBar.jsx`  
**CSS classes:** `.searchbar-row`, `.searchbar`, `.searchbar-icon`

### Visual Anatomy
```
┌─────────────────────────────────────────────────────────┐
│ ⌕  Search by site name, type, status, or ID            │
└─────────────────────────────────────────────────────────┘
```

### Layout
- `.searchbar-row`: `margin-bottom: 24px; width: 100%`
- `.searchbar`: `display: flex; align-items: center; gap: 10px; background: var(--bg)`
- `border: 1px solid var(--border); border-radius: 999px; padding: 0 16px; min-height: 40px`

### Focus State
When the input inside is focused, `.searchbar:focus-within` applies:
- `border-color: var(--border-strong)`
- `box-shadow: 0 0 0 4px rgba(232, 115, 90, 0.12)` (translucent salmon glow)

### Icon
- `.searchbar-icon`: `color: var(--salmon); font-size: 1rem; line-height: 1`
- Uses the Unicode character `⌕` (U+2315, TELEPHONE RECORDER)

### Input
- `width: 100%; border: none; background: transparent; color: var(--ink); font: inherit; outline: none`
- `::placeholder { color: var(--ink-muted) }`

### Transitions
- `border-color`, `box-shadow`, and `background` all `0.15s ease`

---

## 6 · Site Card

**File:** `src/sites/SiteCard.jsx`  
**CSS class:** `.site-card`

### Visual Anatomy

```
┌────────────────────────────────────────────┐
│ School A                         ● OK       │
│ MULTI-IDF CAMPUS                           │
│                                            │
│ AVG CPU    AVG MEMORY                      │
│ 27%        42%                             │
│                                            │
│ DEVICES ONLINE  ACTIVE ALERTS              │
│ 3 / 3           None                       │
│                                            │
│ ████░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░░    │  ← MiniBar
├────────────────────────────────────────────┤
│ 2 IDFs                                  →  │
└────────────────────────────────────────────┘
```

### Container
- `.site-card`: `display: flex; flex-direction: column; height: 100%`
- `background: var(--bg); border: 1px solid var(--border); border-radius: var(--radius-lg); padding: 20px`
- `cursor: pointer; position: relative; overflow: hidden`

### Hover State
- `border-color: var(--border-strong)`
- `box-shadow: 0 4px 16px rgba(0,0,0,0.06)`
- `transform: translateY(-2px)`
- Arrow inside footer: `color: var(--salmon); transform: translateX(3px)`

### Status Modifier Classes
- `.site-card.status-alert`: `border-left: 3px solid var(--status-alert)` (red left border)
- `.site-card.status-caution`: `border-left: 3px solid var(--status-caution)` (amber left border)
- `.site-card` (no modifier or `.status-ok`): no left border accent

### Card Header (`.site-card-header`)
- `display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 14px`
- Left: `.site-name` (name) + `.site-type` (type badge)
- Right: `<SiteStatusBadge>` (see Status Badge spec)

#### `.site-name`
- `font-size: 0.9rem; font-weight: 700; color: var(--ink); letter-spacing: -0.01em; line-height: 1.3`

#### `.site-type`
- `font-family: 'JetBrains Mono', monospace; font-size: 0.65rem; color: var(--ink-muted)`
- `text-transform: uppercase; letter-spacing: 0.05em; margin-top: 2px`

### Metrics Grid (`.site-metrics`)
- `display: grid; grid-template-columns: 1fr 1fr; gap: 8px; margin-bottom: 14px`
- Four metric items: Avg CPU, Avg Memory, Devices Online, Active Alerts

#### `.metric-label`
- `font-family: 'JetBrains Mono', monospace; font-size: 0.6rem; text-transform: uppercase`
- `letter-spacing: 0.06em; color: var(--ink-muted); margin-bottom: 2px`

#### `.metric-value`
- `font-size: 0.88rem; font-weight: 700; color: var(--ink); letter-spacing: -0.01em`
- Active alerts value gets `color: var(--status-alert)` when status is `'alert'`

### MiniBar
See [Utilization Bar](#8--utilization-bar).

### Card Footer (`.site-card-footer`)
- `display: flex; align-items: center; justify-content: space-between`
- `margin-top: auto; padding-top: 12px; border-top: 1px solid var(--border)`
- Left: `.ups-count` — IDF count in JetBrains Mono 0.68rem muted
- Right: `.card-arrow` — `→` character, transitions to salmon on parent hover

### Accessibility
- `role="button"; tabIndex={0}` — keyboard accessible via Enter key

---

## 7 · Status Badge

**File:** `src/common/StatusBadge.jsx`  
**CSS class:** `.status-badge`

### Visual Anatomy
```
┌──────────────┐
│ ● HEALTHY    │  ← pill shape, green bg/text
└──────────────┘
```

### Base Styles
- `.status-badge`: `display: inline-flex; align-items: center; gap: 5px`
- `padding: 3px 9px; border-radius: 100px; flex-shrink: 0`
- `font-family: 'JetBrains Mono', monospace; font-size: 0.62rem; font-weight: 700`
- `text-transform: uppercase; letter-spacing: 0.05em`

### Status Variants

| Class | Background | Text color |
|---|---|---|
| `.status-badge.ok` | `var(--status-ok-bg)` = `#f0fdf4` | `#15803d` (dark green) |
| `.status-badge.caution` | `var(--status-caution-bg)` = `#fffbeb` | `#b45309` (dark amber) |
| `.status-badge.alert` | `var(--status-alert-bg)` = `#fef2f2` | `#b91c1c` (dark red) |
| `.status-badge.unknown` | `var(--status-unknown-bg)` = `#f1f5f9` | `#475569` (slate) |

### Badge Dot (`.badge-dot`)
- `5px × 5px; border-radius: 50%; flex-shrink: 0`
- `.ok .badge-dot` → `background: var(--status-ok)` (bright green)
- `.caution .badge-dot` → `background: var(--status-caution)` (bright amber)
- `.alert .badge-dot` → `background: var(--status-alert)` + `animation: blink 1.2s infinite` (blinking red)
- `.unknown .badge-dot` → `background: var(--status-unknown)` (slate)

### Two Exported Components
- `<SiteStatusBadge status="ok|caution|alert" />` — maps to `OK`, `Caution`, `Alert` labels
- `<DeviceStatusBadge status={0|1|2|3} />` — maps to numeric API status codes:
  - `0` → class `unknown`, label `Unknown`
  - `1` → class `ok`, label `Healthy`
  - `2` → class `caution`, label `Warning`
  - `3` → class `alert`, label `Critical`
  - fallback → class `unknown`, label `Unknown`

---

## 8 · Utilization Bar

**File:** `src/common/UtilizationBar.jsx`

Two distinct variants exist: `MiniBar` (4px tall, full width of container) and `UtilizationBar` (inline with percentage text).

### Utilization Color Classification

```js
value > 90  → 'critical' (red)
value > 75  → 'low'      (amber)
else        → ''          (green, default)
```

### MiniBar (`.mini-bar-wrap` / `.mini-bar`)

Used inside site cards as a CPU utilization indicator.

- `.mini-bar-wrap`: `height: 4px; background: var(--bg-3); border-radius: 2px; overflow: hidden; margin-top: 12px`
- `.mini-bar`: `height: 100%; border-radius: 2px; transition: width 0.4s ease`
  - Default: `background: var(--status-ok)` (green)
  - `.mini-bar.low`: `background: var(--status-caution)` (amber)
  - `.mini-bar.critical`: `background: var(--status-alert)` (red)
- Width is set as `style={{ width: \`${value}%\` }}`

### UtilizationBar (inline table variant)

Used inside `DeviceRow` in the devices table.

```jsx
<div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
  <div className="mini-bar-wrap" style={{ width: 64 }}>
    <div className={`mini-bar ${utilClass}`} style={{ width: `${pct}%` }} />
  </div>
  <span style={{ fontFamily: "'JetBrains Mono', monospace", fontSize: '0.72rem' }}>
    {pct}%
  </span>
</div>
```

- Track fixed at `64px` wide
- Percentage label in JetBrains Mono 0.72rem

---

## 9 · Sites Grid

**File:** `src/sites/SitesGrid.jsx`  
**CSS class:** `.sites-grid`

### Layout
- `display: grid`
- `grid-template-columns: repeat(auto-fill, minmax(280px, 1fr))`
- `gap: 16px`
- Cards grow to fill available columns; minimum card width 280px

### Responsive
- At ≤900px: becomes 1-col (`grid-template-columns: 1fr`)

### Composition Order (top to bottom on all-sites page)
1. `<PageHeader>` — eyebrow + title + last updated timestamp
2. `<OverviewStats>` — 4-card stat strip
3. `<SearchBar>` — pill search input
4. `.sites-grid` with `<SiteCard>` instances — OR — `.empty-state` if no results

---

## 10 · Empty State

**CSS class:** `.empty-state`

Shown when search query matches no sites.

- `border: 1px dashed var(--border-strong)` (dashed, not solid)
- `border-radius: var(--radius-lg)`
- `padding: 36px 24px; background: var(--bg-2)`

### `.empty-state-title`
- `font-size: 1.1rem; font-weight: 800; letter-spacing: -0.03em; color: var(--ink)`

### `.empty-state-copy`
- `margin-top: 8px; color: var(--ink-muted); max-width: 38ch; line-height: 1.6`

---

## 11 · Back Button

**File:** `src/common/BackButton.jsx`  
**CSS class:** `.back-btn`

```
← All Sites
```

- `display: inline-flex; align-items: center; gap: 6px`
- `font-size: 0.8rem; color: var(--ink-muted); cursor: pointer`
- `border: none; background: none; font-family: 'Epilogue', sans-serif; padding: 0`
- `margin-bottom: 28px`
- Hover: `color: var(--ink)`
- Transition: `color 0.15s`
- Default label: `← All Sites`

---

## 12 · Site Detail Header

**File:** `src/sites/SiteDetail.jsx`  
**CSS class:** `.site-detail-header`

```
← All Sites

• SITE DETAIL
District Office                              [Alert badge if active]
3 devices · 3 online
```

### Layout
- `.site-detail-header`: `display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 28px; gap: 20px; flex-wrap: wrap`
- Left: eyebrow (`• SITE DETAIL`) + `<h1>` title + `<p>` device summary
- Right: optional `<SiteStatusBadge alert>` when `active_alerts > 0`

### Eyebrow (reuses `.page-eyebrow` pattern)
- `color: var(--salmon)` with animated `.eyebrow-dot`
- Text: `Site Detail`

---

## 13 · Devices Table

**File:** `src/tables/DevicesTable.jsx`  
**CSS classes:** `.ups-table-wrap`, `.ups-table`

### Table Container (`.ups-table-wrap`)
- `background: var(--bg); border: 1px solid var(--border); border-radius: var(--radius-lg); overflow: hidden`

### Table (`.ups-table`)
- `width: 100%; border-collapse: collapse; font-size: 0.82rem`

### Table Header
- `<thead>`: `background: var(--bg-2); border-bottom: 1px solid var(--border)`
- `<th>`: `padding: 10px 16px`
- `font-family: 'JetBrains Mono', monospace; font-size: 0.62rem; font-weight: 700`
- `text-transform: uppercase; letter-spacing: 0.07em; color: var(--ink-muted); text-align: left`

### Column Headers (left to right)
`IP Address` · `Hostname` · `Role` · `Status` · `CPU` · `Memory` · `Uptime` · `Latency`

### Table Body
- `<td>`: `padding: 12px 16px; border-bottom: 1px solid var(--border); color: var(--ink); vertical-align: middle`
- Last row: no bottom border (`tr:last-child td { border-bottom: none }`)
- Row hover: `tr:hover td { background: var(--bg-2) }`

### Empty State (no devices)
When device list is empty, a single row spanning all columns:
- `text-align: center; color: var(--ink-muted); padding: 32px 16px`
- Text: `No data yet — waiting for first poll`

---

## 14 · Device Row

**File:** `src/devices/DeviceRow.jsx`

### Column Data and Styling

| Column | Content | Style |
|---|---|---|
| IP Address | IP string | `.ups-ip`: JetBrains Mono 0.72rem, `color: var(--ink-muted)` |
| Hostname | hostname or `—` | `color: var(--ink-muted); font-size: 0.8rem` |
| Role | role string or `—` | `color: var(--ink-muted); font-size: 0.8rem` |
| Status | `<DeviceStatusBadge>` | See Status Badge spec |
| CPU | `<UtilizationBar pct={n} />` | 64px bar + JetBrains Mono pct label |
| Memory | `{n}%` | JetBrains Mono 0.75rem |
| Uptime | `{n} days` | JetBrains Mono 0.75rem |
| Latency | `{n} ms` | JetBrains Mono 0.75rem |

Missing data (`null`/`undefined`) renders as `—` em-dash.

---

## 15 · Loading Skeleton

**File:** `src/common/LoadingSkeleton.jsx`  
**CSS class:** `.skeleton`

Shown while site detail data loads.

- `background: linear-gradient(90deg, var(--bg-2) 25%, var(--bg-3) 50%, var(--bg-2) 75%)`
- `background-size: 200% 100%`
- `animation: shimmer 1.4s infinite`
- Default render: `height: 200px; border-radius: 16px`

Shimmer keyframes:
```css
@keyframes shimmer {
  0%   { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
```

---

## 16 · Eyebrow Label

Appears in `PageHeader` and at the top of site detail pages. Not a standalone component — rendered inline in JSX.

### Pattern
```jsx
<div className="page-eyebrow">
  <span className="eyebrow-dot" />
  LABEL TEXT
</div>
```

### Styles
- `.page-eyebrow`: `display: inline-flex; align-items: center; gap: 7px`
- `font-family: 'JetBrains Mono', monospace; font-size: 0.68rem; font-weight: 600`
- `letter-spacing: 0.08em; text-transform: uppercase; color: var(--salmon); margin-bottom: 10px`

### Eyebrow Dot
- `.eyebrow-dot`: `5px × 5px; border-radius: 50%; background: var(--salmon)`
- `animation: pulse 2s ease-in-out infinite`
- Pulse: oscillates scale (1→0.7) and opacity (1→0.4) at 50% keyframe

---

## Transition Summary

All interactive transitions across the UI:

| Element | Property | Duration | Easing |
|---|---|---|---|
| Site card (hover) | `border-color`, `box-shadow`, `transform` | `0.15s` | `ease` |
| Card arrow (hover) | `color`, `transform` | `0.15s` | `ease` |
| Back button (hover) | `color` | `0.15s` | `ease` |
| Search bar (focus) | `border-color`, `box-shadow`, `background` | `0.15s` | `ease` |
| Mini bar fill | `width` | `0.4s` | `ease` |
| Nav (color transitions) | `background`, `border-color`, `color` | `0.3s` | `ease` |
| Body (color transitions) | `background`, `color` | `0.3s` | `ease` |
| Alert banner | `transform` | `0.2s` | `ease` |
