# Patterns, Motion & AI Agent Guidelines

> **For AI agents:** This file documents the dashboard's interaction patterns, animation system, status/alerting semantics, and explicit rules for generating new UI that fits the design. Read this file before writing any new component or page.

---

## 1 · Animation Catalogue

The dashboard uses four distinct CSS keyframe animations. All are defined in `src/index.css`. Use these animations consistently — do not invent new keyframes unless a genuinely new pattern is introduced.

### 1.1 `blink` — Alert Indicator

**Used on:** Alert dot in banner, status badge dot (alert state only)

```css
@keyframes blink {
  0%, 100% { opacity: 1; }
  50%       { opacity: 0.2; }
}
```

- Duration: `1.2s`
- Easing: `ease-in-out`
- Iteration: `infinite`
- Applied to: `.alert-dot`, `.alert .badge-dot`

**Semantic meaning:** Something requires immediate attention. Only use `blink` on red/alert status indicators. Never blink non-alert elements.

---

### 1.2 `pulse` — Ambient Activity Indicator

**Used on:** Eyebrow dot (`.eyebrow-dot`)

```css
@keyframes pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50%       { opacity: 0.4; transform: scale(0.7); }
}
```

- Duration: `2s`
- Easing: `ease-in-out`
- Iteration: `infinite`
- Applied to: `.eyebrow-dot`

**Semantic meaning:** The system is live/active. The eyebrow dot pulses on every page (both All Sites and Site Detail) to suggest real-time monitoring. This is slower and gentler than `blink` — it signals "running" not "warning".

**Note:** The pulse uses `transform: scale()` which requires no hardware-accelerated layer promotion beyond standard GPU compositing — it's lightweight.

---

### 1.3 `shimmer` — Loading Skeleton

**Used on:** `.skeleton` (loading placeholder)

```css
@keyframes shimmer {
  0%   { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
```

- Duration: `1.4s`
- Iteration: `infinite`
- Applied via: `background: linear-gradient(90deg, var(--bg-2) 25%, var(--bg-3) 50%, var(--bg-2) 75%); background-size: 200% 100%`

**Semantic meaning:** Data is loading. The shimmering gradient sweeps left-to-right. Used only for the site detail loading state.

---

### 1.4 Mini Bar Fill — `width` Transition

**Used on:** `.mini-bar` (utilization fill)

```css
transition: width 0.4s ease;
```

- Not a keyframe animation — a CSS property transition
- Triggers whenever the `width` style changes (i.e., when CPU data updates)
- Duration: `0.4s` — slightly longer than UI micro-interactions to feel like a "meter rising"

---

## 2 · Interaction Patterns

### 2.1 Hover Micro-Interactions

The dashboard uses four hover patterns. All use `0.15s ease` transitions.

#### Site Card Hover
```css
.site-card:hover {
  border-color: var(--border-strong);      /* border brightens */
  box-shadow: 0 4px 16px rgba(0,0,0,0.06); /* subtle lift shadow */
  transform: translateY(-2px);              /* 2px upward nudge */
}

.site-card:hover .card-arrow {
  color: var(--salmon);                    /* arrow turns salmon */
  transform: translateX(3px);              /* arrow slides right */
}
```

The combined effect: the card rises slightly, gains depth, and its arrow "points forward" in salmon. This clearly communicates clickability without heavy decoration.

#### Back Button Hover
```css
.back-btn:hover {
  color: var(--ink); /* lightens from muted to full ink */
}
```

#### Search Bar Focus
```css
.searchbar:focus-within {
  border-color: var(--border-strong);
  box-shadow: 0 0 0 4px rgba(232, 115, 90, 0.12);
}
```

The focus ring uses the salmon brand color at 12% opacity — visible without being jarring.

#### Table Row Hover
```css
.ups-table tr:hover td {
  background: var(--bg-2); /* row brightens slightly */
}
```

### 2.2 Keyboard Accessibility

Site cards implement keyboard interaction:
```jsx
<div
  role="button"
  tabIndex={0}
  onKeyDown={e => e.key === 'Enter' && onClick?.()}
>
```

All interactive elements must be reachable by keyboard. New interactive components should follow this same pattern.

---

## 3 · Status System

The dashboard has a three-tier status system applied to both sites and devices.

### 3.1 Site Status

| Status | Meaning | Visual treatment |
|---|---|---|
| `ok` | All devices healthy, no issues | Green badge (`● OK`), no left border accent on card |
| `caution` | Warning condition — device degraded or at-risk | Amber badge (`● Caution`), amber left border on card |
| `alert` | Critical condition — device down or in critical state | Red badge (`● Alert`), red left border on card, active alerts metric shown in red |

**CSS modifiers applied to `.site-card`:**
- `.site-card.status-ok` → no extra border
- `.site-card.status-caution` → `border-left: 3px solid var(--status-caution)`
- `.site-card.status-alert` → `border-left: 3px solid var(--status-alert)`

### 3.2 Device Status

Device status comes from the API as a numeric code (1, 2, 3):

| Code | Semantic | Badge class | Badge label |
|---|---|---|---|
| `1` | Healthy | `.status-badge.ok` | `Healthy` |
| `2` | Warning | `.status-badge.caution` | `Warning` |
| `3` | Critical | `.status-badge.alert` | `Critical` |
| unknown | Fallback | `.status-badge.ok` | `Unknown` |

### 3.3 Status Color Usage Rules

1. **Green only for health.** `--status-ok` / `#22c55e` must never be used decoratively. It means "healthy" or "normal".
2. **Red only for critical.** `--status-alert` / `#ef4444` must only indicate a problem requiring action.
3. **Amber for caution.** `--status-caution` / `#f59e0b` means "degraded but not critical".
4. **Status badges never appear without a dot.** The dot provides the color signal; the text reinforces it.
5. **Alert badges blink; ok/caution badges do not.**

### 3.4 Alert Banner Activation

When any site has an alert, `buildAlerts()` (in `src/utils/siteData.js`) generates alert objects with site name and message. The `AlertBanner` component renders when `alerts.length > 0`. Single alert shows the specific site + message; multiple alerts show a count summary.

---

## 4 · Data Presentation Conventions

### 4.1 Numeric Values

| Data type | Format | Typography |
|---|---|---|
| Percentages (CPU, memory) | `{n}%` | JetBrains Mono |
| Uptime | `{n} days` | JetBrains Mono |
| Latency | `{n} ms` | JetBrains Mono |
| IP addresses | `{a.b.c.d}` | JetBrains Mono, `color: var(--ink-muted)` |
| Device / site counts | `{n}` | Epilogue or JetBrains Mono depending on context |
| Online ratio | `{online} / {total}` | Epilogue bold |

### 4.2 Missing / Null Data

When a field is missing, `null`, or `undefined`, render an em-dash: `—`. Do not render `null`, `undefined`, `N/A`, or `0` for absent data.

### 4.3 Zero vs. Absent

| Condition | Display |
|---|---|
| Active alerts = 0 | Show `None` (not `0`) |
| Active alerts > 0 | Show the count, colored red when status is alert |
| Missing data | `—` |

### 4.4 Timestamps

The "Last Updated" timestamp appears in the page header's right slot:
- Format: `Updated {time}` (e.g., `Updated 4:43:26 PM`)
- Typography: JetBrains Mono, `0.68rem`, `color: var(--ink-muted)`
- Generated from `new Date().toLocaleTimeString()` in the hook

---

## 5 · Pattern: JetBrains Mono Inline Styles

Some components apply JetBrains Mono via inline `style` objects rather than CSS classes. This is intentional for small, context-specific overrides that don't warrant a new class:

```jsx
// Example from DeviceRow.jsx
const monoTextStyle = {
  fontFamily: "'JetBrains Mono', monospace",
  fontSize: '0.75rem',
}
```

When creating new components, prefer CSS classes for repeated patterns, but inline `style` is acceptable for one-off typographic overrides on small spans (timestamps, isolated values).

---

## 6 · CSS Variable Usage Rules

**Always use tokens for these properties:**
- All background colors → `var(--bg)`, `var(--bg-2)`, `var(--bg-3)`
- All text colors → `var(--ink)`, `var(--ink-soft)`, `var(--ink-muted)`
- All border colors → `var(--border)`, `var(--border-strong)`
- Status colors → `var(--status-ok)`, `var(--status-caution)`, `var(--status-alert)` and `-bg` variants
- Brand color → `var(--salmon)`, `var(--salmon-light)`, `var(--salmon-dark)`
- Border radius → `var(--radius)`, `var(--radius-lg)`

**Acceptable hardcoded values (consistent exceptions already in the codebase):**
- Alert banner colors: `#fef2f2`, `#fecaca`, `#b91c1c`, `#ef4444` — these mirror token values but are hardcoded in the banner styles
- Status badge text colors: `#15803d`, `#b45309`, `#b91c1c` — these are the dark variants used for text only
- Pill radius: `100px`, `999px` — utility values not semantic tokens
- Dot sizes: `5px`, `7px` — structural not semantic

**Never:**
- Use `#ffffff` or pure white anywhere — the design uses warm `#f8f7f2`
- Add drop shadows except on hovered site cards
- Use any font other than Epilogue or JetBrains Mono
- Apply `font-weight` other than: 400, 500, 600, 700, 800

---

## 7 · AI Agent Rules: Building New UI

When Composer 2.5 or any AI agent generates new pages, components, or UI patterns for this codebase, the following rules apply:

### Rule 1: Derive from tokens, not imagination
All colors, fonts, and radii must come from Section 2 and 3 of `00-design-tokens.md`. Do not guess hex values.

### Rule 2: Match the two-font discipline
- `Epilogue` for all prose, headings, button labels, UI copy
- `JetBrains Mono` for all technical values, table headers, metadata labels, anything uppercase and data-like

### Rule 3: Eyebrow + title pattern for new pages
Any new top-level page should open with:
```
• [PAGE TYPE IN UPPERCASE]    ← salmon eyebrow with animated dot
[Page Title Here]              ← Epilogue 800, clamp size
[Optional subtitle]            ← Epilogue 400, muted
```
This is the established visual hierarchy for page introductions.

### Rule 4: New data tables follow the DevicesTable pattern
All new tables must:
- Wrap in `.ups-table-wrap` (rounded container with border)
- Use `.ups-table` class (full-width, border-collapse: collapse)
- Use `.ups-table thead` with `bg-2` background
- Use JetBrains Mono uppercase headers at `0.62rem`
- Use `12px 16px` padding on cells
- Show row hover with `background: var(--bg-2)`
- Handle empty state with a centered muted message

### Rule 5: New cards follow the SiteCard pattern
New summary cards should:
- Use `background: var(--bg)` + `border: 1px solid var(--border)` + `border-radius: var(--radius-lg)` + `padding: 20px`
- Include a card header section (name/type on left, status/badge on right)
- Use a bottom footer with `border-top: 1px solid var(--border)` and `padding-top: 12px`
- Implement hover with `border-color: var(--border-strong)`, lift shadow, and `-2px` Y translate

### Rule 6: New status indicators reuse the status badge
Do not create custom status indicators. Always use `<SiteStatusBadge>` or `<DeviceStatusBadge>` from `src/common/StatusBadge.jsx`. If a new type of status is needed, add it to that file.

### Rule 7: Metrics labels always use JetBrains Mono uppercase
Any label for a data metric (CPU, memory, latency, uptime, count) should use:
```css
font-family: 'JetBrains Mono', monospace;
font-size: 0.6rem – 0.72rem;  /* depending on density */
text-transform: uppercase;
letter-spacing: 0.06em – 0.08em;
color: var(--ink-muted);
```

### Rule 8: Avoid decorative flourishes
The design's restraint is intentional. New UI should not add:
- Color gradients on backgrounds or cards
- Additional shadows (except hover lift on cards)
- Decorative icons beyond what exists
- Extra borders or dividers not already in the pattern language
- New color introductions outside the token set

### Rule 9: New API calls go in `src/services/`
Any new backend request function belongs in `src/services/`. New state and polling logic belongs in `src/hooks/`. Display components must not call `fetch()` directly.

### Rule 10: Charts belong in `src/charts/`
When utilization history charts or sparklines are added, they go in `src/charts/`. The folder exists as a placeholder for this intent. Recharts or a similar library is the expected choice.

---

## 8 · Visual Tone Reference

Use these adjectives to calibrate new design decisions against the existing aesthetic:

**The design IS:**
- Calm
- Technical and data-dense, but not cluttered
- Warm (cream backgrounds, not clinical white)
- Restrained (accent color appears sparingly)
- Precise (monospace labels reinforce machine-readability)
- Professional but not corporate

**The design is NOT:**
- Colorful or expressive
- Playful or consumer-facing
- Dark-mode (default is light; dark mode is not implemented)
- Shadow-heavy or elevated
- Icon-heavy
- Animated beyond small purposeful micro-interactions

---

## 9 · Relationship Between Eyebrow Dot Colors and Brand

The salmon/coral color (`#e8735a`) appears in exactly four places:
1. Logo mark strokes
2. Eyebrow label text + animated dot
3. Search bar icon (`⌕`)
4. Hover state of the card arrow and back button

Every other UI element uses neutral ink, muted ink, or status colors. This scarcity makes the salmon color meaningful — it only appears on elements that signal "identity" (logo, labels) or "navigate/interact here" (arrows, focus rings).

When adding new interactive elements, ask: should this element reinforce the salmon accent? Only use it if the element represents navigation, identity, or a search/action affordance.
