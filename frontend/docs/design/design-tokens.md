# Design Tokens — Oak Grove Network Dashboard

> **For AI agents:** This file is the authoritative reference for every raw design value used in the dashboard. When generating new pages or components, derive all visual decisions from this token set. Never invent ad-hoc hex values, font sizes, or spacing — always map to a token below.

---

## 1 · Identity

| Property | Value |
|---|---|
| Product name | **Network Dashboard** |
| Brand name (wordmark) | **Equate** |
| App `<title>` | `Equate Network Dashboard` |
| Aesthetic | Warm neutral minimalism · technical data density · calm authority |
| Audience | IT administrators monitoring K-12 school network infrastructure |

The visual language is intentionally restrained: warm off-white backgrounds, a muted ink palette for text, and a single coral/salmon accent color. The only moment of color emphasis comes from status indicators (green / amber / red).

---

## 2 · Color Palette

All colors are defined as CSS custom properties on `:root` in `src/index.css`.

### 2.1 Backgrounds

| Token | Hex | Usage |
|---|---|---|
| `--bg` | `#f8f7f2` | Page background, site cards, nav background |
| `--bg-2` | `#f6f4ee` | Stat cards, table header, secondary surfaces |
| `--bg-3` | `#e6e6e6` | Utilization bar track, skeleton base |

> **Warmth note:** The backgrounds carry a subtle warm/cream undertone, not pure white. This gives the UI a calm, physical quality. Do not substitute `#ffffff` or `#f5f5f5`.

### 2.2 Text / Ink

| Token | Hex | Usage |
|---|---|---|
| `--ink` | `#0a0a0a` | Primary text — headings, values, strong labels |
| `--ink-soft` | `#3d3d3d` | Secondary text — defined but unused directly, available for mid-weight copy |
| `--ink-muted` | `#7a7a7a` | De-emphasized text — metadata labels, timestamps, placeholder text, secondary table cells |

### 2.3 Borders

| Token | Hex | Usage |
|---|---|---|
| `--border` | `#eceae3` | Default border for cards, tables, nav |
| `--border-strong` | `#7a7a7a` | Emphasized borders (hover states, dashed outlines, separator dots in nav) |

### 2.4 Brand Accent (Salmon / Coral)

| Token | Hex | Usage |
|---|---|---|
| `--salmon` | `#e8735a` | Logo mark strokes, eyebrow text, search icon, hover arrow color, focus ring base |
| `--salmon-light` | `#fce8e3` | Accent backgrounds (defined, available for tints) |
| `--salmon-dark` | `#c75a42` | Darker salmon for hover/active states on accent elements |

> **Accent usage rule:** The salmon color appears sparingly — logo, eyebrow labels, interactive micro-details. It should never be used as a background for large areas or as body text.

### 2.5 Status Colors

| Token | Hex | Background token | Hex | Text treatment |
|---|---|---|---|---|
| `--status-ok` | `#22c55e` | `--status-ok-bg` | `#f0fdf4` | Text: `#15803d` |
| `--status-caution` | `#f59e0b` | `--status-caution-bg` | `#fffbeb` | Text: `#b45309` |
| `--status-alert` | `#ef4444` | `--status-alert-bg` | `#fef2f2` | Text: `#b91c1c` |
| `--status-unknown` | `#64748b` | `--status-unknown-bg` | `#f1f5f9` | Text: `#475569` |

Status colors follow a traffic-light semantic plus a distinct Unknown slate: green = healthy, amber = caution/warning, red = critical/alert, slate = upstream-unreachable / dependency impact. They are used exclusively for device and site health indicators — never decoratively.

**Alert banner specific overrides (not tokens, hardcoded in CSS):**
- Alert banner background: `#fef2f2`
- Alert banner border: `#fecaca`
- Alert banner text: `#b91c1c`
- Alert dot: `#ef4444`

### 2.6 Complete Token Map (Copy-Paste Reference)

```css
:root {
  --bg:            #f8f7f2;
  --bg-2:          #f6f4ee;
  --bg-3:          #e6e6e6;
  --ink:           #0a0a0a;
  --ink-soft:      #3d3d3d;
  --ink-muted:     #7a7a7a;
  --border:        #eceae3;
  --border-strong: #7a7a7a;
  --salmon:        #e8735a;
  --salmon-light:  #fce8e3;
  --salmon-dark:   #c75a42;
  --radius:        8px;
  --radius-lg:     16px;
  --status-ok:         #22c55e;
  --status-ok-bg:      #f0fdf4;
  --status-caution:    #f59e0b;
  --status-caution-bg: #fffbeb;
  --status-alert:      #ef4444;
  --status-alert-bg:   #fef2f2;
  --status-unknown:    #64748b;
  --status-unknown-bg: #f1f5f9;
}
```

---

## 3 · Typography

### 3.1 Type Families

Two font families are loaded from Google Fonts (`src/index.css` `@import`):

| Family | Classification | Weights loaded | Role |
|---|---|---|---|
| **Epilogue** | Geometric humanist sans-serif | 400, 500, 600, 700, 800 | Body, headings, UI text, navigation |
| **JetBrains Mono** | Monospaced | 400, 600, 700 | Technical labels, metric values, timestamps, IP addresses, table headers, uptime, latency |

```css
@import url('https://fonts.googleapis.com/css2?family=Epilogue:wght@400;500;600;700;800&family=JetBrains+Mono:wght@400;600;700&display=swap');
```

> **AI agent rule:** JetBrains Mono should be used wherever data feels "technical" — numbers that represent measurements (CPU %, memory %, latency ms, uptime days), IP addresses, labels that read like machine-readable identifiers, and all `text-transform: uppercase` metadata labels. Epilogue handles all human-readable text, headings, and prose.

### 3.2 Type Scale

| Role | Size | Weight | Family | Notes |
|---|---|---|---|---|
| Page title (h1) | `clamp(1.6rem, 2.5vw, 2.2rem)` | 800 | Epilogue | `letter-spacing: -0.035em`, `line-height: 1.1` |
| Site name (card heading) | `0.9rem` | 700 | Epilogue | `letter-spacing: -0.01em` |
| Subtitle / page-sub | `0.88rem` | 400 | Epilogue | `color: var(--ink-muted)`, `line-height: 1.6`, `margin-top: 6px` |
| Metric value (card) | `0.88rem` | 700 | Epilogue | `letter-spacing: -0.01em` |
| Stat value (overview strip) | `1.6rem` | 800 | Epilogue | `letter-spacing: -0.04em`, `line-height: 1` |
| Eyebrow label | `0.68rem` | 600 | JetBrains Mono | `letter-spacing: 0.08em`, `text-transform: uppercase`, `color: var(--salmon)` |
| Stat label | `0.68rem` | 600 | JetBrains Mono | `letter-spacing: 0.07em`, `text-transform: uppercase`, `color: var(--ink-muted)` |
| Metric label | `0.6rem` | 400 | JetBrains Mono | `letter-spacing: 0.06em`, `text-transform: uppercase`, `color: var(--ink-muted)` |
| Status badge | `0.62rem` | 700 | JetBrains Mono | `letter-spacing: 0.05em`, `text-transform: uppercase` |
| Site type (sub-label) | `0.65rem` | 400 | JetBrains Mono | `text-transform: uppercase`, `letter-spacing: 0.05em`, `color: var(--ink-muted)` |
| Table header | `0.62rem` | 700 | JetBrains Mono | `letter-spacing: 0.07em`, `text-transform: uppercase`, `color: var(--ink-muted)` |
| Table body text | `0.82rem` | 400 | Epilogue | General table cell text |
| Table mono values | `0.72rem`–`0.75rem` | 400 | JetBrains Mono | IP addresses, uptime, latency |
| Nav wordmark | `1.0rem` | 800 | Epilogue | `letter-spacing: -0.02em` |
| Nav right meta | `0.72rem` | 400 | JetBrains Mono | `letter-spacing: 0.04em`, `color: var(--ink-muted)` |
| Back button | `0.8rem` | 400 | Epilogue | `color: var(--ink-muted)` |
| IDF count (card footer) | `0.68rem` | 400 | JetBrains Mono | `color: var(--ink-muted)` |
| Alert banner | `0.82rem` | 500 | Epilogue | `color: #b91c1c` |
| Last updated timestamp | `0.68rem` | 400 | JetBrains Mono | `color: var(--ink-muted)` |
| Search placeholder | inherited | 400 | Epilogue | `color: var(--ink-muted)` |

### 3.3 Letter-Spacing Conventions

Negative letter-spacing (`-0.01em` to `-0.04em`) is applied to all display-weight headings and large values to tighten the spacing at high font weights — a common technique for Epilogue at 700-800 weight.

Positive letter-spacing (`0.04em` to `0.08em`) is applied to ALL uppercase/monospace labels to aid readability at small sizes. These always pair with `text-transform: uppercase`.

### 3.4 Line Height Conventions

| Context | `line-height` |
|---|---|
| Page titles, stat values | `1.0`–`1.1` (tight) |
| Subtitle / page-sub | `1.6` |
| Body prose (empty states, etc.) | `1.6` |
| Single-line labels | implicit / `1` |

---

## 4 · Spacing & Sizing

### 4.1 Border Radii

| Token | Value | Usage |
|---|---|---|
| `--radius` | `8px` | Stat cards, table wrap, skeleton, smaller elements |
| `--radius-lg` | `16px` | Site cards, table container, empty state, loading skeleton override |
| `100px` (pill) | hardcoded | Status badges (no overflow, always fully rounded) |
| `999px` (pill) | hardcoded | Search bar (full pill shape) |
| `2px` | hardcoded | Utilization bar and track |
| `50%` | hardcoded | All dots (status dot, badge dot, eyebrow dot, alert dot) |

### 4.2 Spacing Scale

These are the spacing values observed in use (all via CSS, no CSS vars for spacing):

| Usage | Value |
|---|---|
| Nav height | `60px` |
| Nav horizontal padding | `48px` (desktop), `20px` (≤900px) |
| Page content top padding | `88px` (no alert) / `124px` (alert banner visible) |
| Page content horizontal padding | `48px` (desktop), `20px` (≤900px) |
| Page content bottom padding | `48px` (desktop), `40px` (≤900px) |
| Max content width | `1280px` |
| Page header margin-bottom | `32px` |
| Stat strip margin-bottom | `32px` |
| Stat strip gap | `12px` |
| Search bar margin-bottom | `24px` |
| Sites grid gap | `16px` |
| Site card padding | `20px` |
| Site card header margin-bottom | `14px` |
| Site metrics gap | `8px` |
| Site metrics margin-bottom | `14px` |
| Mini bar margin-top | `12px` |
| Card footer padding-top | `12px` |
| Stat card padding | `16px 18px` |
| Stat label margin-bottom | `8px` |
| Table header padding | `10px 16px` |
| Table cell padding | `12px 16px` |
| Table body font size | `0.82rem` |
| Back button margin-bottom | `28px` |
| Site detail header margin-bottom | `28px` |
| Page eyebrow margin-bottom | `10px` |
| Alert banner padding | `10px 48px` |
| Empty state padding | `36px 24px` |
| Empty state title margin-top | `8px` |

### 4.3 Component Dimensions

| Component | Size |
|---|---|
| Logo mark (SVG) | `30px × 30px` |
| Nav logo gap | `10px` |
| Status badge padding | `3px 9px` |
| Badge dot | `5px × 5px` |
| Eyebrow dot | `5px × 5px` |
| Alert dot | `7px × 7px` |
| Mini bar height | `4px` |
| Search bar min-height | `40px` |
| Search bar padding | `0 16px` |
| Utilization bar inline width | `64px` |
| Loading skeleton height | `200px` |
| Sparkline placeholder | `100px wide × 32px tall` |

---

## 5 · Shadows & Elevation

The design uses shadows very sparingly — only on hover states.

| State | Shadow value |
|---|---|
| Default | None |
| Site card (hover) | `0 4px 16px rgba(0,0,0,0.06)` |

No drop shadows on the nav, stat cards, table, or other elements.

---

## 6 · Focus & Interactive States

### Search bar focus-within
```css
border-color: var(--border-strong);
box-shadow: 0 0 0 4px rgba(232, 115, 90, 0.12);
```
This creates a translucent salmon glow ring using `rgba` of `--salmon` at 12% opacity.

### Back button hover
```css
color: var(--ink);
```
Transitions from `--ink-muted` to `--ink`.

### Site card hover
```css
border-color: var(--border-strong);
box-shadow: 0 4px 16px rgba(0,0,0,0.06);
transform: translateY(-2px);
```

### Card arrow hover (child of .site-card:hover)
```css
color: var(--salmon);
transform: translateX(3px);
```

---

## 7 · Logo Mark

The logo is an SVG file at `assets/logo.svg`. It consists of two parallel diagonal lines (a stylized double-slash `//` motif):

```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 30 30" fill="none">
  <line x1="7"  y1="24" x2="17" y2="6"  stroke="#e8735a" stroke-width="3.5" stroke-linecap="round"/>
  <line x1="13" y1="24" x2="23" y2="6"  stroke="#e8735a" stroke-width="3.5" stroke-linecap="round"/>
</svg>
```

- Stroke color: `#e8735a` (same as `--salmon`)
- Stroke width: `3.5`
- Linecap: `round`
- ViewBox: `0 0 30 30`

The double-slash (`//`) is also used in the page title of the index.html and appears in font as a decorative element in the nav. In the `<title>` it appears as `Equate Network Dashboard`.

---

## 8 · Responsive Breakpoints

| Breakpoint | Rule | Changes |
|---|---|---|
| Default (desktop) | `> 900px` | Full padding, 4-column stat strip, auto-fill card grid |
| Tablet | `≤ 900px` | Nav and content padding reduced to `20px`, stat strip becomes 2-col, card grid becomes 1-col, nav-right gap `8px` |
| Mobile | `≤ 540px` | Nav wraps (`height: auto`, `min-height: 60px`, top/bottom padding `10px`), nav-right fills full width and wraps, stat strip becomes 1-col |
