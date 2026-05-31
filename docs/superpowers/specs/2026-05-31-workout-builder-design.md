# Workout Builder (reusable plans) — Design

**Date:** 2026-05-31
**Status:** Approved for planning
**Milestone:** multi-user
**Author:** Ryan (with Claude)

## Summary

Add a **Workout Builder**: a shared library of reusable workout plans
("templates"). A plan is a named, ordered list of exercises with prescribed
**sets** and **reps** (or hold duration for timed/cardio exercises), plus
workout-level and per-exercise notes/instructions. **Weight is never
prescribed** — it is filled in live per user, because the same plan is used by
multiple people of different capabilities.

When starting a workout, the user picks a plan and it pre-populates today's
workout with planned exercise rows (using the existing plan-first
`COMPLETED=0` model). The user then records weight/reps/notes and marks rows
complete as normal.

This is feature 1 of 3 from the iteration session. Features 2 (per-exercise
progress charts) and 3 (cross-platform rest-timer fix) are separate
spec → plan → build cycles.

## Goals

- Build and save reusable workout plans from scratch.
- Save an existing logged day as a new plan ("save as plan").
- Apply a plan to today's workout, pre-filling exercises + prescribed
  sets/reps/duration, leaving weight blank.
- Shared library: every user sees and can apply/edit/delete every plan.

## Non-goals (YAGNI)

- Per-user/private plans or ownership/permissions (pure shared library).
- Prescribed weight or per-user weight suggestions.
- Scheduling, calendars, multi-week programs, or progression logic.
- Plan versioning/history.

## Decisions (locked during brainstorming)

1. **What a plan prescribes:** hybrid — exercise list + per-exercise target
   sets/reps (or seconds), plus notes. **No weight.**
2. **Ownership:** shared library; no owner column. Anyone can apply, edit, or
   delete any plan (mirrors the shared exercises library).
3. **Apply semantics:** append, **skip exercises already present** on today's
   workout (idempotent if tapped twice).
4. **Creation paths:** dedicated builder screen **and** "save this day as a
   plan" shortcut.
5. **Apply entry points:** both the home screen ("Start from plan ▾") **and**
   a per-card "Apply to today" button on the library page.

## Data model

Two new tables, added in `internal/db/schema.go::EnsureSchema` using the same
`CREATE TABLE IF NOT EXISTS` pattern as `equipment`/`plates`.

### `workout_templates`
| Column | Type | Notes |
|---|---|---|
| `ID` | INTEGER PK | |
| `NAME` | TEXT DEFAULT '' | e.g. "Push Day A" |
| `NOTE` | TEXT DEFAULT '' | workout-level instructions/comments |
| `CREATED_AT` | TEXT DEFAULT '' | timestamp set on insert |

### `template_items`
| Column | Type | Notes |
|---|---|---|
| `ID` | INTEGER PK | |
| `TEMPLATE_ID` | INTEGER | FK → workout_templates.ID |
| `EXERCISE_ID` | INTEGER | FK → exercises.ID; resolved to Name/Color/Kind/Mode at apply time |
| `POSITION` | INTEGER DEFAULT 0 | ordering within the plan |
| `TARGET_SETS` | INTEGER DEFAULT 0 | prescribed # of sets; 0 = unset → applies as 1 row |
| `TARGET_REPS` | INTEGER DEFAULT 0 | prescribed reps for `mode=reps`; 0 = unset |
| `TARGET_SECONDS` | INTEGER DEFAULT 0 | prescribed hold/duration for `timed`/`cardio`; 0 = unset |
| `NOTE` | TEXT DEFAULT '' | per-exercise instruction/cue |

Index: `CREATE INDEX IF NOT EXISTS idx_template_items_tpl ON template_items(TEMPLATE_ID);`

**Target field selection:** the builder shows a **reps** input for exercises
with `MODE=reps` and a **mm:ss duration** input for `MODE=timed` or
`KIND=cardio`. `TARGET_SETS` always applies.

### Go models (`internal/models/models.go`)
```go
type WorkoutTemplate struct {
    ID        int    `db:"ID"`
    Name      string `db:"NAME"`
    Note      string `db:"NOTE"`
    CreatedAt string `db:"CREATED_AT"`
    Items     []TemplateItem // populated on detail load, not a db column
}

type TemplateItem struct {
    ID            int    `db:"ID"`
    TemplateID    int    `db:"TEMPLATE_ID"`
    ExerciseID    int    `db:"EXERCISE_ID"`
    Position      int    `db:"POSITION"`
    TargetSets    int    `db:"TARGET_SETS"`
    TargetReps    int    `db:"TARGET_REPS"`
    TargetSeconds int    `db:"TARGET_SECONDS"`
    Note          string `db:"NOTE"`
}
```

## Components

### DB layer — `internal/db/templates.go` (new)
- `SelectTemplates(path) []WorkoutTemplate` — all plans, newest first (no items).
- `GetTemplate(path, id) WorkoutTemplate` — plan + items (items ordered by POSITION).
- `SelectTemplateCounts(path) map[int]int` — exercise count per template, for library cards (or compute via a joined query).
- `SaveTemplate(path, WorkoutTemplate, []TemplateItem) int` — upsert plan, **full-replace** its items (delete existing items for the ID, reinsert from the submitted list), return template ID. Mirrors the day-form full-replace save in `set.go`.
- `DeleteTemplate(path, id)` — delete plan + its items.

Follow existing conventions in this package: `mu.Lock()`/`connect()`,
`quoteStr()` for user text, `fmt.Sprintf` SQL (consistent with the rest of the
file — note this codebase does not use parameterized queries).

### Web handlers — `internal/web/templates_page.go` (new)
- `GET  /plans/` → `templatesHandler` — render library page.
- `GET  /plans/edit/:id` → `templateFormHandler` — render builder (`id=new` or numeric). Passes exercise list for the pickers.
- `POST /plans/` → `saveTemplateHandler` — parse the builder form (parallel arrays like `set.go`), build items, `SaveTemplate`, redirect to `/plans/`.
- `POST /plans/del` → `deleteTemplateHandler` — `DeleteTemplate`, redirect to `/plans/`.
- `GET  /plans/json/:id` → `templateJSONHandler` — return a resolved plan as JSON for client-side apply (items joined to exercises → Name, Color, Kind, Mode, targets, note).

Register all routes in `internal/web/webgui.go` behind `auth.Auth(&authConf)`,
matching the existing block.

### GuiData additions (`internal/models/models.go`)
- `Templates []WorkoutTemplate` — for the library page and home dropdown.
- `TemplateCounts map[int]int` — exercise counts for library cards.
- `OneTemplate WorkoutTemplate` — for the builder.

The home dropdown needs the plan list, so `indexHandler` also loads
`SelectTemplates`.

### Templates (HTML)
- `internal/web/templates/templates_page.html` (new) — library: cards with
  name, exercise count, note, and buttons **Apply to today**, **Edit**,
  **Delete** (delete confirms), plus **＋ New plan**.
- `internal/web/templates/template_form.html` (new) — builder form:
  - Name + Note fields.
  - Reorderable exercise rows (reuse **SortableJS**, already loaded since
    mu.14). Each row: exercise datalist-combo picker (same pattern as the
    exercise form's Group/Place combos), target-sets input, conditional
    reps **or** mm:ss input, per-exercise note, remove button.
  - **＋ Add exercise** and **Save**.
- `internal/web/templates/header.html` — add **"Plans"** nav link (between
  Stats and Weight), `href="/plans/"`.
- `internal/web/templates/index.html` — add **"Start from plan ▾"** dropdown
  at the top of the today-workout column, listing `.Templates` by name; add a
  **"Save as plan"** button on the logged-day section.

### JS — `internal/web/public/js/plans.js` (new) + edits to `index.js`
- **Builder** (`plans.js`): add/remove/reorder rows; on exercise pick, swap the
  target input between reps and mm:ss based on the chosen exercise's
  kind/mode (exercise metadata embedded in the page).
- **Apply** (in `index.js` or `plans.js`): fetch `/plans/json/:id`, then for
  each item, **skip if the exercise name is already a row in today's form**;
  otherwise inject `TARGET_SETS` planned rows (≥1) via the existing
  row-adding code path — `COMPLETED=0`, weight blank, reps from `TARGET_REPS`
  (or duration from `TARGET_SECONDS`), row note from the item note. If today's
  workout name/note are empty, set them from the plan. **Nothing persists
  until the user hits Save** (identical to manual entry; the existing
  `POST /set/` full-replace then stores everything).
- **Save as plan** (client-side, no extra endpoint): JS reads today's rendered
  exercise rows, derives one item per distinct exercise (`TARGET_SETS` = number
  of that exercise's rows today; `TARGET_REPS`/`TARGET_SECONDS` = the mode/most
  common value among them; weight ignored), stashes the derived list (e.g. via
  `sessionStorage`), and opens the builder (`/plans/edit/?id=new`) which
  pre-fills from that stash. The user tweaks and saves through the normal
  `POST /plans/` path.

## Data flow

**Build a plan:** `/plans/edit/?id=new` → fill name/note + rows → `POST /plans/`
→ `SaveTemplate` (insert plan, insert items) → redirect `/plans/`.

**Apply a plan:** home dropdown or library "Apply to today" → `GET
/plans/json/:id` → JS injects planned rows into today's form (skip dups) →
user records weight/reps, marks complete → **Save** → `POST /set/`
full-replace persists planned + completed rows (existing flow, no changes to
`set.go` semantics).

**Save day as plan:** logged day → "Save as plan" → name prompt → builder
pre-filled from today's distinct exercises → user tweaks → `POST /plans/`.

## Error handling & edge cases

- **Empty plan name** → default to a placeholder ("Untitled plan") or block
  save with a hint; pick block-with-hint for clarity.
- **Plan with zero items** → allow save (user may be drafting) but the library
  card shows "0 exercises"; applying a 0-item plan is a no-op.
- **`TARGET_SETS = 0`** → apply injects exactly 1 planned row.
- **Exercise deleted after being added to a plan** → `GET /plans/json/:id`
  skips items whose `EXERCISE_ID` no longer resolves; builder shows such a row
  as "(missing exercise)" so the user can remove it.
- **Apply when exercise already on today's list** → that item is skipped
  (decision 3); other items still apply.
- **Apply twice** → second apply skips everything already present → no dup rows.
- **Mode mismatch** (e.g. plan made for a reps exercise whose mode later
  changed to timed) → apply uses the exercise's *current* kind/mode to decide
  which target to write; if the relevant target is 0, falls back to a single
  blank planned row.
- **Concurrent multi-user edits to the same plan** → last write wins
  (acceptable for a household app; consistent with the rest of the codebase).

## Testing

- **DB:** unit tests for `SaveTemplate` (insert + full-replace of items),
  `GetTemplate` (item ordering by POSITION), `DeleteTemplate` (cascades items),
  `SelectTemplates` ordering.
- **Schema:** `EnsureSchema` is idempotent — new tables/index created once,
  safe on re-boot (existing test pattern).
- **Apply logic (JS):** manual verification — apply to empty day (rows appear,
  weight blank, reps prefilled), apply twice (no dups), apply with one exercise
  already present (only that one skipped), timed/cardio target writes duration.
- **Save as plan:** verify distinct-exercise grouping and sets-count/mode-reps
  derivation from a multi-row day.
- **Manual cross-user:** Ryan builds "Push Day A", Akane applies it and logs
  her own weights; both see the same plan in the library.

## Migration / compatibility

- Additive only: two new tables + one index via `EnsureSchema`. No changes to
  existing tables, `sets` semantics, or `set.go` save flow.
- No backfill required.
- **Version bump** required before the commit that should produce a new GHCR
  tag (`internal/web/public/version`), per project workflow.

## Files touched

**New:** `internal/db/templates.go`, `internal/web/templates_page.go`,
`internal/web/templates/templates_page.html`,
`internal/web/templates/template_form.html`,
`internal/web/public/js/plans.js`.

**Edited:** `internal/db/schema.go` (tables+index),
`internal/models/models.go` (models + GuiData fields),
`internal/web/webgui.go` (routes), `internal/web/index.go` (load templates),
`internal/web/templates/header.html` (nav link),
`internal/web/templates/index.html` (dropdown + "save as plan"),
`internal/web/public/js/index.js` (apply/inject rows),
`internal/web/public/version` (bump).
