# Responsive Overhaul + iPhone Timer Fix — Design

**Date:** 2026-08-09
**Status:** Approved (sections reviewed with Ryan)
**Branch:** multi-user

## Problem

1. On the Galaxy Z Fold 7 inner screen (~800×720 CSS px) every page functions but is
   cramped: words and controls cut off or smashed together. Root cause: ~800px sits just
   past Bootstrap's `md` breakpoint (768px), where pages split into half-width
   `col-md-*` columns **and** tables simultaneously add columns (e.g. `index.css`
   un-hides the today-table Note column and shrinks Name to 38% at exactly 768px while
   the card is squeezed into a ~376px half-column). Layout decisions are keyed to
   *viewport* width while content lives in *fractional* columns.
2. On iPhone Safari the rest-timer chip never appears when a strength set is marked
   complete, though the identical flow works on Android Chrome. Suspects, in order:
   - Chip is `position: fixed; bottom: 1rem` — hidden under iOS Safari's bottom tab
     bar, and under the software keyboard (iOS doesn't shrink the layout viewport).
   - Stale cache: `/fs/public` assets are served with **no** `Cache-Control`, `ETag`,
     or `Last-Modified` (embedded FS has zero ModTime), and no version params. iOS
     Safari caches heuristically or indefinitely.
   - User-cookie fallback: without an `ed_user` cookie the middleware silently uses
     the first user, whose timer setting may be off.

   The design fixes the first two outright; the third is a device-side check.

## Approach (chosen: container-query components)

Bootstrap 5 stays for **components** (navbar, cards, buttons, forms, accordion,
badges, utilities). Its viewport-breakpoint grid stops being the page layout
mechanism.

### 1. Layout system

- **Page level:** each page's content area becomes a CSS grid:
  `grid-template-columns: repeat(auto-fit, minmax(<min>, 1fr))` — columns form only
  when there is room, regardless of device or fold posture.
- **Card level:** card wrappers get `container-type: inline-size`; card internals
  adapt via `@container` rules keyed to the card's own width (today-table Note
  column, weight-card stat row, score-sheet header, date toolbar).
- **Shared stylesheet:** new `internal/web/public/css/app.css` loaded from
  `header.html` on every page. Holds the grid/container system plus shared component
  CSS (rest-timer chip, `.del-set-button`, `.exercise-button`, steppers). Fixes the
  current bug where `index.css` styles are silently missing on PRT / Equipment /
  Users / Plans pages. Page-specific rules stay in per-page CSS.
- **Fallback:** browsers without container-query support (pre-iOS 16 / pre-Chrome 105)
  get single-column stacked layout — usable, never broken.
- **Navbar:** `navbar-expand-sm` → `navbar-expand-lg` (8 links collapse below 992px).
- **Header shell fix:** `header.html` currently leaves `<head>`/`<body>` open and
  every page emits a second `</head>`/`<body>`. Normalize: header closes head and
  opens body once; page templates stop re-emitting them.

### 2. Page-by-page

- **index.html (home):**
  - Heatmap: drop hard `width: 770px` and the `direction: rtl` scroll hack; canvas
    container fluid within the card, JS scroll-to-current-week.
  - Today's-workout card: container-query treatment for column proportions and Note
    visibility; date toolbar becomes flex (fixed arrow buttons, flexible date input);
    library `hstack` rows get wrap allowance; bottom Save toolbar wraps.
  - Weight card: stat row (`display-6` number + BMI + button) wraps via container rule.
  - Score sheet header (shared partial): `flex-wrap`, title allowed to wrap.
- **stats.html:** auto-fit grid; add missing `table-responsive`; charts fluid.
- **weight.html:** auto-fit grid; the 4-element add-form `input-group` splits/wraps
  at narrow container widths.
- **prt_list.html:** keep 7-column table inside `table-responsive` (horizontal scroll
  when narrow); tighten per-cell badge rows.
- **template_form.html / plans.js:** rewrite JS-generated builder row (currently
  unprefixed `col-2`/`col`) container-aware.
- **config.html / exercise.html:** replace invalid `<form>`-inside-`<table>` markup
  with proper form rows; fixes config.html's unbalanced card divs.
- **users.html / equipment.html / prt_form.html:** already stack correctly at 800px;
  converted to the same grid system for consistency; `col-md-4` field rows become
  auto-fit so labels stop wrapping badly.

### 3. Timer fix + iOS hardening

- **Chip anchored top-right, just below the navbar** (immune to iOS bottom tab bar
  and keyboard). `viewport-fit=cover` added to the viewport meta;
  `env(safe-area-inset-*)` used in the offset; `-webkit-user-select` prefix added.
  Behavior unchanged otherwise (countdown, green flash, ding, tap-to-cancel).
- **Cache-busting:** all local `<script src>` / `<link href>` get `?v={{ .Version }}`.
  Version read hoisted from `configHandler` into shared middleware/package var so
  every page has it. New middleware on `/fs/` sets
  `Cache-Control: public, max-age=31536000, immutable` (safe because URLs are
  versioned). Route-level change confined to `internal/web/webgui.go` + middleware.
- **Robustness:** wrap the 5 unguarded `sessionStorage` sites (`index.js:218,709,728`
  + 2 more) in try/catch — iOS "Block All Cookies" currently aborts page init;
  fix UTC-date fallback (`new Date().toJSON()` → `toLocaleDateString('en-CA')`);
  null-guard `heatmap.js` / `weight-chart.js` initializers.
- **Known iOS limits (accepted):** no vibration API on iOS; WebAudio ding respects
  the silent switch. The visual flash is the reliable signal on iPhone.

## Explicitly out of scope

- No schema, data, or API changes. No new pages or features.
- No framework replacement; Bootstrap components remain.
- PRT list stays a table (no card-list re-presentation).

## Verification

1. Build via Docker (`golang:1.23` + gomod cache volume, per reference workflow),
   run the container locally.
2. Chrome device emulation across every page at: 360px phone, 390px iPhone,
   800×720 (Z Fold 7 inner), 768px tablet, 1200px+ desktop. Screenshot each;
   check for overflow, cramming, cut-off controls.
3. `curl -I` on `/fs/public/js/index.js` to confirm `Cache-Control`; view source to
   confirm `?v=` params.
4. Timer: emulated iPhone viewport — mark a strength set complete, confirm chip
   appears top-right; tap to cancel; let one expire.
5. Real-device pass by Ryan (Z Fold 7 + iPhone) before Unraid cutover, with backup
   first per playbook. Bump version before the commit that should produce the GHCR tag.
