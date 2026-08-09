# Responsive Overhaul + iPhone Timer Fix — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make every page usable at any width (phone 360px → Z Fold 7 inner 800×720 → desktop) via a container-query layout system, and fix the rest-timer chip that never appears on iPhone Safari.

**Architecture:** Bootstrap 5 stays for components; its viewport grid (`col-md-*`) is replaced by CSS grid page layouts + `@container` rules keyed to each card's real width. A new shared `app.css` is loaded on every page from `header.html`. Go side: version exposed as a template func for cache-busting, `Cache-Control` middleware on `/fs/`. The chip re-anchors top-right (immune to iOS bottom toolbar/keyboard).

**Tech Stack:** Go 1.23 / Gin / html-template, Bootstrap 5.3 (CDN), vanilla JS, CSS container queries (iOS 16+/Chrome 105+; older browsers gracefully get single-column).

**Spec:** `docs/superpowers/specs/2026-08-09-responsive-overhaul-design.md`

## Global Constraints

- Go is NOT installed on this machine. Build/test ONLY via Docker:
  `docker run --rm -v "C:\Users\rkenn\ClaudeCodeProjects\ExerciseDiary:/src" -v edgomod:/go/pkg/mod -w /src -e CGO_ENABLED=0 -e GOFLAGS=-mod=mod golang:1.23 sh -c "go build ./... && go test ./... && echo OK"`
- Runtime smoke test:
  `docker run -d --name ed-smoke -p 8851:8851 -v "C:\Users\rkenn\ClaudeCodeProjects\ExerciseDiary:/src" -v edgomod:/go/pkg/mod -w /src -e CGO_ENABLED=0 -e GOFLAGS=-mod=mod golang:1.23 sh -c "go run ./cmd/ExerciseDiary -d /tmp/eddata"`
  then browse http://localhost:8851. Create one user via the UI (Manage users) if pages look empty. `docker rm -f ed-smoke` when done. First `go run` may take ~1 min to compile.
- Visual verification widths (browser window / device emulation): **360×740** (phone), **390×844** (iPhone), **800×720** (Z Fold 7 inner — the key target), **1280×800** (desktop). "PASS" means: no horizontal page scroll, no overlapping/cut-off text or controls, tap targets not smashed together.
- Only gofmt Go files you touch (repo is not gofmt-clean; don't reformat others).
- No schema/data/API changes. No new features. Keep all element IDs used by JS (`weightCardCol`, `weightShowCol`, `matrix-chart`, `weight-chart`, `todayEx`, etc.) unless a task explicitly renames one.
- Template note: pages render as TWO sequential writes: `header.html` then `<page>.html` (see `internal/web/index.go:55-56`). There is no block/inheritance system.
- Version bump (`internal/web/public/version`) happens ONLY in the final task — it drives the GHCR tag.
- Commit after every task. Commit messages end with:
  `Co-Authored-By: Claude Fable 5 <noreply@anthropic.com>` and `Claude-Session: https://claude.ai/code/session_017sZqzKAVRTWyoSS8BwXHhP`

---

### Task 1: Go infrastructure — version template func + Cache-Control on /fs/

**Files:**
- Create: `internal/web/version.go`
- Create: `internal/web/version_test.go`
- Modify: `internal/web/webgui.go` (funcMap ~line 49, middleware before StaticFS line 62)
- Modify: `internal/web/config.go:26-29` (replace inline version read)

**Interfaces:**
- Produces: template func `ver` (returns e.g. `0.1.9-mu.16`) usable in ALL templates as `{{ ver }}`; `cacheControlMiddleware() gin.HandlerFunc`; package var `appVersion string`.
- Consumes: existing `pubFS` embed (`internal/web/const-var.go:29`), `check.IfError`.

- [ ] **Step 1: Write the failing test** — `internal/web/version_test.go`:

```go
package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestParseVersion(t *testing.T) {
	got := parseVersion([]byte("VERSION=0.1.9-mu.16\n"))
	if got != "0.1.9-mu.16" {
		t.Fatalf("parseVersion = %q, want %q", got, "0.1.9-mu.16")
	}
}

func TestCacheControlMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(cacheControlMiddleware())
	r.GET("/fs/x", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	r.GET("/other", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/fs/x", nil))
	if cc := w.Header().Get("Cache-Control"); !strings.Contains(cc, "immutable") {
		t.Fatalf("/fs/ Cache-Control = %q, want immutable", cc)
	}

	w = httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/other", nil))
	if cc := w.Header().Get("Cache-Control"); cc != "" {
		t.Fatalf("/other Cache-Control = %q, want empty", cc)
	}
}
```

- [ ] **Step 2: Run tests, verify FAIL** (Docker build command from Global Constraints). Expected: compile error — `parseVersion` / `cacheControlMiddleware` undefined.

- [ ] **Step 3: Implement** — `internal/web/version.go`:

```go
package web

import (
	"strings"

	"github.com/gin-gonic/gin"
)

// appVersion is parsed once at startup from the embedded public/version file
// ("VERSION=x.y.z"). Exposed to all templates via the "ver" func for
// cache-busting static asset URLs.
var appVersion string

func parseVersion(b []byte) string {
	return strings.TrimPrefix(strings.TrimSpace(string(b)), "VERSION=")
}

// cacheControlMiddleware marks /fs/ static assets immutable. Safe because
// every local asset URL carries a ?v={{ ver }} param that changes each release.
func cacheControlMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if strings.HasPrefix(c.Request.URL.Path, "/fs/") {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
		}
		c.Next()
	}
}
```

In `webgui.go` inside `Gui()`, before the funcMap (~line 48):

```go
	verFile, err := pubFS.ReadFile("public/version")
	check.IfError(err)
	appVersion = parseVersion(verFile)
```

Add to funcMap: `"ver": func() string { return appVersion },`
Immediately before the `router.StaticFS("/fs/", ...)` line add: `router.Use(cacheControlMiddleware())`
In `config.go:26-29` replace the four lines (file read through `guiData.Version = version[8:]`) with `guiData.Version = appVersion` (drop the now-unused `check` import if nothing else uses it — it does: keep imports compiling).

- [ ] **Step 4: Run tests, verify PASS** (same Docker command). Expected: `OK`.
- [ ] **Step 5: Commit** — `git add -A && git commit` — message: `Cache-Control on /fs/ + version exposed as template func (infra for cache busting)`

---

### Task 2: Header shell normalization, app.css scaffold, cache-busted asset URLs, navbar

**Files:**
- Create: `internal/web/public/css/app.css`
- Modify: `internal/web/templates/header.html`
- Modify: ALL page templates (strip duplicate `</head>`/`<body>`, add `?v={{ ver }}`): `index.html:2,9-10`, `stats.html:2-6`, `weight.html:2-5`, `config.html:3-4`, `equipment.html:3,74`, `exercise.html:3`, `users.html:3`, `prt_list.html:3`, `prt_form.html:3`, `templates_page.html:3`, `template_form.html:2-4,44`, `login.html:2-3`
- Modify: `internal/web/public/css/index.css` (remove moved rules)

**Interfaces:**
- Consumes: `{{ ver }}` from Task 1.
- Produces: `app.css` loaded on EVERY page (from header.html); shared classes `.del-set-button`, `.exercise-button`, `.add-exercise-button` now global. Header emits exactly one `</head><body>`.

- [ ] **Step 1: header.html changes**
  - Line 8 viewport meta → `<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">`
  - After the theme/bootstrap links (both CDN and NodePath branches), add:
    `<link rel="stylesheet" href="/fs/public/css/app.css?v={{ ver }}">`
  - Line 30: `navbar-expand-sm` → `navbar-expand-lg`; line 41-42: the `d-none d-sm-inline` / `d-inline d-sm-none` username spans → single `<span class="ms-1">…</span>`; line 66: `mb-sm-0` → `mb-lg-0`.
- [ ] **Step 2: Strip the duplicate shell from every page template.** header.html already ends with `</head>` + `<body>` + navbar. In each page template listed above, DELETE the stray `</head>` line and the stray `<body>` line (line refs above; verify by eye — some pages have only one of the two). Per-page `<script>`/`<link>` tags stay where they are (top of body — valid HTML5).
- [ ] **Step 3: Version-param every local asset.** In every page template, each `/fs/public/js/*.js` and `/fs/public/css/*.css` URL gets `?v={{ ver }}` appended (e.g. `<script src="/fs/public/js/index.js?v={{ ver }}"></script>`). CDN URLs and favicon untouched. Files: index.html (5 js + 1 css), stats.html (2 js + 1 css), weight.html (2 js + 1 css), config.html (1 css), equipment.html (1 js), template_form.html (1 js).
- [ ] **Step 4: Create app.css** — move these rules VERBATIM out of `index.css:1-29` into it (delete from index.css): the `.del-set-button` block (incl. `:hover` and `.text-danger`/`.text-success` overrides), `.exercise-button`, `.add-exercise-button`. Top-of-file comment: `/* app.css — shared layout system + components, loaded on every page (header.html). */`
- [ ] **Step 5: Build + smoke test.** Docker build (expect OK), run smoke container, then in a browser check `/`, `/prt/`, `/plans/`, `/config/`: view-source shows exactly one `</head>` and one `<body>`, all local assets carry `?v=0.1.9-mu.16`, delete/edit icon buttons on the PRT page are now styled (they weren't before). `curl -I http://localhost:8851/fs/public/css/app.css` shows the immutable Cache-Control header.
- [ ] **Step 6: Commit** — `Normalize header shell, shared app.css on all pages, cache-busted asset URLs`

---

### Task 3: Layout primitives in app.css

**Files:**
- Modify: `internal/web/public/css/app.css` (append)

**Interfaces:**
- Produces (consumed by Tasks 5-10): `.page-cq` (container root, goes on each page's `container-lg` div), `.grid-2`, `.grid-8-4`, `.grid-4-8`, `.grid-cards`, `.form-grid`, `.card-cq`.

- [ ] **Step 1: Append to app.css:**

```css
/* ---- Layout system ----------------------------------------------------
   Pages: put .page-cq on the page's .container-lg, then use the .grid-*
   classes instead of Bootstrap rows/cols. Grids are single-column by
   default and gain columns via @container queries on the page container —
   i.e. based on actual available width, not viewport width. Browsers
   without container-query support simply keep the single-column layout. */
.page-cq {
    container-type: inline-size;
    container-name: page;
}
.grid-2, .grid-8-4, .grid-4-8 {
    display: grid;
    grid-template-columns: 1fr;
    gap: 1.5rem;
    align-items: start;
    margin-bottom: 1.5rem;
}
.grid-2 > *, .grid-8-4 > *, .grid-4-8 > *, .grid-cards > * { min-width: 0; }
@container page (min-width: 700px) {
    .grid-2   { grid-template-columns: 1fr 1fr; }
    .grid-8-4 { grid-template-columns: 2fr 1fr; }
    .grid-4-8 { grid-template-columns: 1fr 2fr; }
}
/* Card gallery (plans library): as many ~300px columns as fit. */
.grid-cards {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(min(100%, 300px), 1fr));
    gap: 1rem;
    margin-bottom: 1.5rem;
}
/* Form field rows (replaces col-md-4 triplets): fields wrap naturally. */
.form-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(100%, 160px), 1fr));
    gap: 0.75rem 1rem;
}
/* Card whose INTERNALS use @container rules (today table, weight card…). */
.card-cq { container-type: inline-size; }
```

- [ ] **Step 2: Build + quick check.** Docker build; smoke-run; pages unchanged visually (classes not yet used anywhere). No console errors.
- [ ] **Step 3: Commit** — `app.css layout primitives: container-query grids and form-grid`

---

### Task 4: Rest-timer chip fix + iOS JS hardening

**Files:**
- Modify: `internal/web/public/css/app.css` (chip rules move here, re-anchored)
- Modify: `internal/web/public/css/index.css:105-131` (delete old chip rules + keyframes)
- Modify: `internal/web/public/js/index.js` (sessionStorage guards ~lines 218, 709, 728; UTC date ~726-731)
- Modify: `internal/web/public/js/heatmap.js:19`, `internal/web/public/js/weight-chart.js` (null guards)

**Interfaces:**
- Consumes: `.rest-timer-chip` element created by `rest-timer.js:23-28` (unchanged).
- Produces: helpers `ssGet(key)`, `ssSet(key, value)` in index.js (top of file), used anywhere index.js touches sessionStorage.

- [ ] **Step 1: Move + re-anchor chip CSS.** Delete `index.css:105-131` (the two `.rest-timer-chip` rules + `@keyframes restTimerFlash`). Append to app.css:

```css
/* Floating rest-timer chip — pinned TOP-right, just below the navbar.
   Top-anchored on purpose: iOS Safari's bottom tab bar and the software
   keyboard both overlay a bottom-anchored fixed element (the original bug:
   chip was invisible on iPhone). Click to cancel; green flash when done. */
.rest-timer-chip {
    position: fixed;
    top: calc(env(safe-area-inset-top, 0px) + 4.5rem);
    right: calc(env(safe-area-inset-right, 0px) + 1rem);
    z-index: 1080;
    background: var(--bs-primary);
    color: var(--bs-light);
    padding: 0.5rem 0.9rem;
    border-radius: 999px;
    font-size: 1.05rem;
    font-weight: 600;
    box-shadow: 0 4px 12px rgba(0,0,0,0.35);
    cursor: pointer;
    -webkit-user-select: none;
    user-select: none;
    transition: background 0.25s ease;
}
.rest-timer-chip .bi { vertical-align: -2px; margin-right: 0.25rem; }
.rest-timer-chip.rest-timer-done {
    background: var(--bs-success);
    animation: restTimerFlash 0.6s ease-in-out 0s 2;
}
@keyframes restTimerFlash {
    0%, 100% { transform: scale(1); }
    50%      { transform: scale(1.08); }
}
```

- [ ] **Step 2: sessionStorage guards in index.js.** Add near the top (after the escape helpers):

```js
// iOS Safari with "Block All Cookies" throws on ANY sessionStorage access;
// an unguarded call here used to abort page init entirely.
function ssGet(key) { try { return window.sessionStorage.getItem(key); } catch (e) { return null; } }
function ssSet(key, val) { try { window.sessionStorage.setItem(key, val); } catch (e) {} }
```

Replace every direct `sessionStorage.getItem(...)` / `sessionStorage.setItem(...)` in index.js with `ssGet(...)` / `ssSet(...)` (grep finds them: ~lines 218, 709, 728 — verify none are missed with `grep -n "sessionStorage" internal/web/public/js/*.js`; the `localStorage` sites are already try/catch-wrapped, leave them).

- [ ] **Step 3: UTC date fallback fix.** In `setFormDate` (~index.js:726-731) replace `new Date().toJSON().slice(0,10)` with `new Date().toLocaleDateString('en-CA')` (matches the existing correct usage at index.js:838).
- [ ] **Step 4: Chart null guards.** `heatmap.js` `makeChart`: first line `const el = document.getElementById('matrix-chart'); if (!el) return;` then use `el.getContext('2d')`. Same pattern in `weight-chart.js` `weightChart()` for its `getElementById(id)`.
- [ ] **Step 5: Verify.** Build + smoke-run. In browser at 390×844: enable rest timer for the user (Manage users), add + complete a strength set on Home → chip appears TOP-right below navbar, counts down, flashes green, tap cancels. Simulate storage failure: in DevTools console run `Object.defineProperty(window, 'sessionStorage', {get(){throw new Error('blocked')}})` then reload — page must still render rows (no init abort).
- [ ] **Step 6: Commit** — `Rest timer chip: top-anchor + safe-area (iOS fix); guard sessionStorage; UTC date + chart guards`

---

### Task 5: Home page (index.html) re-layout

**Files:**
- Modify: `internal/web/templates/index.html:44-292`
- Modify: `internal/web/templates/score_sheet.html:4-23` (shared header wrap)
- Modify: `internal/web/public/css/index.css` (todayex `@media` → `@container`; `.horiz-scroll`)
- Modify: `internal/web/public/js/index.js` (note-cell class in 2 strength renderers + thead; heatmap scroll)
- Modify: `internal/web/public/js/heatmap.js` (scroll-to-end after render)

**Interfaces:**
- Consumes: `.page-cq`, `.grid-8-4`, `.grid-2`, `.card-cq` from Task 3.
- Produces: today-table note cells use class `todayex-note` ONLY (no `d-none d-md-table-cell`) — thead `index.html:278` and both strength renderers (`index.js` ~331-376 and ~382-430) must agree.

- [ ] **Step 1: Page shell.** `index.html:44` → `<div class="container-lg mt-4 page-cq">`. Row 1 (`:45-97`): replace `<div class="row">` with `<div class="grid-8-4">`; the three children lose their `col-md-8 mb-4` / `col-md-4 mb-4` / `col-md-4 mb-4 d-none` classes → `<div>`, `<div id="weightCardCol">`, `<div class="d-none" id="weightShowCol">` (keep both ids — `index.js:965-981` toggles them). Row 2 (`:99`): `row` → nothing; `col-md-6 mb-4` → plain `div` (full width). Row 3 (`:143`): `row` → `grid-2`; both `col-md` children → plain `div`s.
- [ ] **Step 2: Heatmap fluid.** `index.html:50` `style="height: 100px; width: 770px;"` → `style="height: 100px; min-width: 700px;"`. In `index.css` delete the `@media (max-width: 1399px)` block and add unconditionally:

```css
.horiz-scroll { overflow-x: auto; }
```

In `heatmap.js`, at the end of `makeChart` (after the Chart is created) add:

```js
    // Open scrolled to the most recent weeks (right edge).
    const scroller = el.closest('.horiz-scroll');
    if (scroller) scroller.scrollLeft = scroller.scrollWidth;
```

- [ ] **Step 3: Weight card internals.** `index.html:65` remove `style="overflow-x: auto;"`; `:66` `d-flex align-items-center mb-2` → `d-flex align-items-center flex-wrap gap-2 mb-2` (remove `me-3` from `:67` and `:71`, keep `ms-auto` on the button wrapper).
- [ ] **Step 4: Today card + table container queries.** Give the today-workout card (`index.html:236` area, the `card` inside the second `grid-2` child) class `card-cq`. Date toolbar (`:239-251`): replace the `row`/`col-md-2`/`col-md hstack`/`col-md-2` structure with:

```html
<div class="d-flex align-items-center gap-2">
  <button type="button" class="btn del-set-button" onclick="shiftDate(-1)"><i class="bi bi-arrow-left-square"></i></button>
  <input name="date" type="date" class="form-control flex-grow-1" id="realDate" style="min-width: 0;">
  <button type="button" class="btn del-set-button" onclick="shiftDate(1)"><i class="bi bi-arrow-right-square"></i></button>
</div>
```

(Copy the EXACT existing button handlers/ids/icons from the current markup — the snippet shows the shape; preserve whatever onclick names index.html:239-251 actually uses.) Bottom toolbar (`:284-287`): add `flex-wrap gap-2` to its container.
In the thead (`:278`) change the Note `<th>` class from `d-none d-md-table-cell todayex-note` to `todayex-note` only. In `index.js` strength + timed-strength renderers, same change on the note `<td>` (search `d-md-table-cell`).
In `index.css` replace the `:42-52` width block with:

```css
/* Today's-sets column sizing, keyed to the CARD's width (container query),
   not the viewport — the card may be a half column on wide screens. */
.table-todayex .todayex-name   { width: 50%; }
.table-todayex .todayex-weight { width: 22%; }
.table-todayex .todayex-reps   { width: 22%; }
.table-todayex .todayex-del    { width:  6%; }
.table-todayex .todayex-note   { display: none; }
@container (min-width: 620px) {
    .table-todayex .todayex-name   { width: 38%; }
    .table-todayex .todayex-weight { width: 18%; }
    .table-todayex .todayex-reps   { width: 18%; }
    .table-todayex .todayex-note   { display: table-cell; width: 20%; }
    .table-todayex .todayex-del    { width:  6%; }
}
```

- [ ] **Step 5: Library accordion.** `index.html:162-229`: remove the nested `row` + `col-md-11`/`col-md-1` split — make it a `d-flex gap-2` with the accordion list `flex-grow-1 min-width-0` and the add-group button `flex-shrink-0`. In `index.css`, `.exercise-button { width: 70% }` → `.exercise-button { flex: 1 1 auto; min-width: 0; text-align: left; color: var(--bs-body-color); }` and `.add-exercise-button { width: 9% }` → `.add-exercise-button { flex: 0 0 auto; }` (these moved to app.css in Task 2 — edit them there).
- [ ] **Step 6: Score sheet header.** `score_sheet.html:4-23`: outer header `d-flex justify-content-between align-items-center` → add `flex-wrap gap-2`; the long title span gets `class="me-auto"` with `style="min-width: 0;"`.
- [ ] **Step 7: Verify at all four widths** (Global Constraints). Key checks at 800×720: heatmap+weight side by side or stacked WITHOUT squeeze; library and today cards STACKED full-width (container 776px < 700+700); today table shows Note column when full-width; date input not clipped; nothing overflows. At 1280: two columns return; today card (~half width, <620cq) correctly HIDES Note. At 360/390: single column, 50/22/22/6 table, no horizontal scroll.
- [ ] **Step 8: Commit** — `Home page: container-query layout (grid-8-4/grid-2, fluid heatmap, today-table @container)`

---

### Task 6: Stats + Weight pages

**Files:**
- Modify: `internal/web/templates/stats.html:7-76`
- Modify: `internal/web/templates/weight.html:6-48`

**Interfaces:** Consumes `.page-cq`, `.grid-4-8`, `.card-cq`.

- [ ] **Step 1: stats.html.** `:7` add `page-cq` to the container div. `:31` `row` → `grid-4-8`; `col-md-4`/`col-md-8` children → plain `div`s. Wrap the exercise table (`:45-54`) in `<div class="table-responsive">…</div>`. Remove the `col-1 col-4 col-3 col-3` classes from its `<th>`s (they're grid classes on table cells — replace with nothing; the browser sizes 4 columns fine).
- [ ] **Step 2: weight.html.** `:6` add `page-cq`. `:7` `row` → `grid-4-8`; children lose `col-md-*`. Same `<th>` class cleanup (`:14-17`). Replace the 4-element `input-group` add-form (`:41-47`) with a wrapping flex row, preserving all existing `name`/`id` attributes and button handlers:

```html
<div class="d-flex flex-wrap gap-2 align-items-center">
  <input name="date" type="date" class="form-control" style="flex: 1 1 10rem; min-width: 8rem;" ...existing attrs...>
  <input name="weight" type="number" step="any" min="0" class="form-control" style="flex: 1 1 7rem; min-width: 6rem;" ...existing attrs...>
  <div class="d-flex gap-2">…the two existing buttons…</div>
</div>
```

- [ ] **Step 3: Verify** both pages at the four widths: 800×720 → stacked full-width (table above charts on stats); no cramped pickers; charts resize; weight form wraps at 360 instead of smashing.
- [ ] **Step 4: Commit** — `Stats + Weight pages on container-query grid`

---

### Task 7: PRT list + PRT form

**Files:**
- Modify: `internal/web/templates/prt_list.html:4-80`
- Modify: `internal/web/templates/prt_form.html:4-120`

**Interfaces:** Consumes `.page-cq`, `.grid-8-4`, `.form-grid`.

- [ ] **Step 1: prt_list.html.** `:4` add `page-cq`. Table already has `table-responsive` — keep. In the score cells (`:38`, `:44`, `:50`) the badge wrapper `<div class="mt-1">` → `<div class="mt-1 text-nowrap">`; add `text-nowrap` to the Date cell. Header row (`:5-8`) add `flex-wrap gap-2`.
- [ ] **Step 2: prt_form.html.** `:4` add `page-cq`. `:5` `row` → `grid-8-4`; `col-lg-7`/`col-lg-5` → plain `div`s. The three `col-md-4` input wrappers (`:23,29,35`): replace their parent `row g-3` with `<div class="form-grid">` and drop the `col-md-4` classes.
- [ ] **Step 3: Verify** at the four widths; at 360 the PRT table horizontal-scrolls inside its card without breaking the page; form fields wrap cleanly.
- [ ] **Step 4: Commit** — `PRT pages on container-query grid`

---

### Task 8: Plans library + builder

**Files:**
- Modify: `internal/web/templates/templates_page.html:4-30`
- Modify: `internal/web/templates/template_form.html:5-45`
- Modify: `internal/web/public/js/plans.js:56-66`
- Modify: `internal/web/public/css/app.css` (append `.plan-row` rules)

**Interfaces:**
- Consumes: `.page-cq`, `.grid-cards`.
- Produces: builder row markup class `plan-row` — inputs keep their existing `name` attributes (`name`, `sets`, `target`, `note`) and the drag-handle/remove-button classes that `plans.js` Sortable init + submit serializer rely on (read `plans.js` fully before editing; preserve every selector it queries).

- [ ] **Step 1: templates_page.html.** `:4` add `page-cq`. `:15-16`: replace the `row` + `col-md-6 col-lg-4 mb-3` card wrappers with `<div class="grid-cards">` and plain `div` children. Card header (`:18-21`): add `flex-wrap gap-1`, plan-name span gets `text-truncate` with `style="min-width: 0;"`, badge gets `flex-shrink-0`.
- [ ] **Step 2: Builder row → flex.** In `plans.js:56-66` replace the `row g-2 align-items-center` + `col-*` markup with (adapt to the EXACT current inner markup — only the wrapper classes change):

```js
row.className = 'plan-row';
// children in order: drag handle (span), name input, sets input, target input, note input, remove button
```

Append to app.css:

```css
/* Plan-builder row: wraps on narrow screens instead of crushing col-2 cells. */
.plan-row { display: flex; flex-wrap: wrap; gap: 0.5rem; align-items: center; }
.plan-row .plan-name   { flex: 2 1 12rem; min-width: 0; }
.plan-row .plan-sets,
.plan-row .plan-target { flex: 0 1 5rem; min-width: 4rem; }
.plan-row .plan-note   { flex: 1 1 8rem; min-width: 0; }
```

Give the four inputs those classes in the JS-generated markup (keep existing `name=`/placeholder attrs and the handle/remove classes).
- [ ] **Step 3: Verify.** Plans library: cards 1-up at 360, 2-up at 800, 3+-up at 1280. Builder (`/plans/edit/new`): add rows, drag to reorder still works, save a plan and re-open it (round-trip proves the serializer still finds its inputs). At 360 the row wraps to two lines instead of crushing.
- [ ] **Step 4: Commit** — `Plans library grid-cards + wrapping builder rows`

---

### Task 9: Config + Exercise — replace form-in-table markup

**Files:**
- Modify: `internal/web/templates/config.html:5-116`
- Modify: `internal/web/templates/exercise.html:4-110`
- Modify: `internal/web/public/css/config.css` (tooltip clamp)

**Interfaces:** Consumes `.page-cq`, `.grid-2`. Form field `name=` attributes and the POST endpoints are UNCHANGED — this is markup-only.

- [ ] **Step 1: config.html.** `:5` add `page-cq`; `:6` `row` → `grid-2`; the two `col-md` → plain `div`s. Inside each card, replace the `<table>`-with-interleaved-`<form>`s (invalid HTML — browsers hoist the forms) with one `<form>` per card containing stacked groups; every input keeps its exact `name` and current value template expression:

```html
<form action="/config/" method="post">
  <div class="mb-2">
    <label class="form-label mb-1">Host {{/* keep the existing .tip-button markup inline after the label text */}}</label>
    <input name="host" type="text" class="form-control" value="{{ .Config.Host }}">
  </div>
  <!-- …one block per existing row, same order… -->
  <button type="submit" class="btn btn-primary mt-2">Save</button>
</form>
```

While restructuring, fix the unbalanced card/div nesting around `:60-63` and `:109-116` (the About card currently never closes before the column ends — the rewrite makes nesting explicit and balanced).
- [ ] **Step 2: exercise.html.** Same treatment: `:4` add `page-cq`; `:5` `row` → `grid-2`; left column's label/input `<table>` (`:10-92`) becomes stacked `.mb-2` form groups inside the single main `<form>`; the separate delete `<form>` (`:84-90`) moves OUT of the table to after the main form, `style="float: right;"` → wrap both action buttons in `<div class="d-flex justify-content-between mt-2">`. Right column (image + video) unchanged.
- [ ] **Step 3: Tooltip clamp.** In `config.css` `.tip-button[mytitle]:focus::after, .tip-button[mytitle]:hover::after` rule: `max-width: 18em;` → `max-width: min(18em, 70vw); left: 0;` so it can't overflow the viewport on narrow screens.
- [ ] **Step 4: Verify.** Both pages at four widths. CRITICAL: save config changes and save an exercise edit — confirm POSTs still work (field names unchanged). Delete button still deletes (test on a throwaway exercise). Tip tooltips readable at 360.
- [ ] **Step 5: Commit** — `Config + Exercise: valid form markup, container-query grid, tooltip clamp`

---

### Task 10: Users + Equipment consistency pass

**Files:**
- Modify: `internal/web/templates/users.html:4-100` (+ the "new user" form ~:150-190)
- Modify: `internal/web/templates/equipment.html:4-70`

**Interfaces:** Consumes `.page-cq`, `.grid-8-4`, `.form-grid`.

- [ ] **Step 1: users.html.** `:4` add `page-cq`; `:5` `row` → `grid-8-4`; `col-lg-7`/`col-lg-5` → plain `div`s. In each user edit form and the new-user form: the `row g-3` with `col-md-6/-4/-12` children → `<div class="form-grid">`, children lose `col-*` classes; the two `col-md-12` children (if any hold full-width controls like the color row) get `style="grid-column: 1 / -1;"`. Accordion header (`:15-21`): add `flex-wrap` to its inner layout, badges `flex-shrink-0`.
- [ ] **Step 2: equipment.html.** `:4` add `page-cq`; `:5` `row` → `grid-8-4`; children → plain `div`s. Plates table: wrap in `table-responsive`, drop the inline `style="width:9em"` / `style="width:3em"` from the `<th>`s (`:28-29`) — replace with `class="text-nowrap"` on the first `<th>`. NOTE: `equipment.html:79-83` inline JS also generates plate rows — keep its markup consistent with the template rows.
- [ ] **Step 3: Verify** at four widths. Save a user edit (all fields round-trip), add/remove a plate row, "Try it" suggestion still works.
- [ ] **Step 4: Commit** — `Users + Equipment on container-query grid`

---

### Task 11: Full sweep verification + version bump

**Files:**
- Modify: `internal/web/public/version` (`VERSION=0.1.9-mu.16` → `VERSION=0.1.9-mu.17`)
- Modify: anything small the sweep finds

**Interfaces:** none new.

- [ ] **Step 1: Full matrix sweep.** Smoke-run; visit EVERY page (`/`, `/stats/`, `/weight/`, `/prt/`, `/prt/new`, `/equipment/`, `/plans/`, `/plans/edit/new`, `/exercise/?id=new`, `/config/`, `/users/`, `/login/`) at 360×740, 390×844, 800×720, 1280×800. Screenshot each; check PASS criteria from Global Constraints. Exercise the interactive flows once each: add strength/timed/cardio set, complete a set (timer chip), heatmap scroll position, weight add, plan apply-to-today.
- [ ] **Step 2: Fix anything found** (small CSS/markup nits only — anything structural goes back to its task's file set with a fresh commit).
- [ ] **Step 3: Cache-bust proof.** View source on `/`: all `?v=` params read `0.1.9-mu.16` (pre-bump); after Step 4's bump + restart they read `0.1.9-mu.17` — confirming Safari-stale-cache can't survive a release.
- [ ] **Step 4: Bump version** file to `VERSION=0.1.9-mu.17`.
- [ ] **Step 5: Final Docker build + test** (`go build ./... && go test ./...` → OK).
- [ ] **Step 6: Commit** — `Bump version to 0.1.9-mu.17 (responsive overhaul + iPhone rest-timer fix)`

---

## Self-Review Notes

- Spec coverage: layout system → T3; shared CSS everywhere → T2; navbar → T2; shell fix → T2; per-page treatments → T5-T10; chip + safe-area + viewport-fit → T4 (meta in T2); cache-busting + Cache-Control → T1-T2; sessionStorage/UTC/chart guards → T4; verification matrix → per-task + T11; version bump last → T11. No schema changes anywhere. ✔
- Real-device pass (Ryan: Z Fold 7 + iPhone) and Unraid cutover happen AFTER this plan, per the deploy playbook — deliberately out of plan scope.
- Known judgment calls an implementer may tune: the `700px` page-container column threshold and `620px` today-table threshold; adjust ±40px if the 800×720 sweep shows a better break.
