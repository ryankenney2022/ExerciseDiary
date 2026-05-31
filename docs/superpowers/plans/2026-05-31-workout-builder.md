# Workout Builder Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a shared library of reusable workout plans ("templates") with prescribed sets/reps (never weight), that pre-populate today's workout via the existing plan-first model.

**Architecture:** Two new SQLite tables (`workout_templates`, `template_items`) added additively in `EnsureSchema`. A new DB layer (`templates.go`, unit-tested), new Gin handlers + routes under `/plans/`, two new HTML pages, and a new `plans.js`. "Apply" is client-side: a JSON endpoint returns a resolved plan, and JS calls the existing `addExercise(obj)` to inject planned (`COMPLETED=0`) rows into today's form — nothing persists until the user hits the normal Save.

**Tech Stack:** Go 1.x, Gin, sqlx + `modernc.org/sqlite` (pure-Go, no CGO), Go `html/template` (context-aware: structs inside `<script>` are JSON-encoded automatically), Bootstrap 5, SortableJS (already vendored via CDN in index.html), Chart.js (unused here).

**Spec:** `docs/superpowers/specs/2026-05-31-workout-builder-design.md`

**Conventions to follow (this codebase):**
- DB functions use `mu.Lock()` + `connect(path)` + `fmt.Sprintf` SQL with `quoteStr()` for user text. The project does **not** use parameterized queries — match the surrounding style.
- Errors in DB reads use `log.Printf("WARN: ...")`; writes use `check.IfError(err)`.
- Page handlers populate `models.GuiData`, then `c.HTML(200, "header.html", guiData)` followed by `c.HTML(200, "<page>.html", guiData)`.
- Page templates: `{{ define "x.html" }}<body>...{{ template "footer.html" }}{{ end }}`.
- Build with `make go-build` (`cd cmd/ExerciseDiary && CGO_ENABLED=0 go build`). Run locally with `make run`.

---

## File Structure

**New files:**
- `internal/db/templates.go` — CRUD for plans + items.
- `internal/db/templates_test.go` — unit tests for the DB layer (first tests in the repo).
- `internal/web/templates_page.go` — handlers: library, builder form, save, delete, JSON.
- `internal/web/templates/templates_page.html` — plan library page.
- `internal/web/templates/template_form.html` — plan builder page.
- `internal/web/public/js/plans.js` — builder UI (add/remove/reorder rows, reps↔time swap, prefill).

**Modified files:**
- `internal/models/models.go` — `WorkoutTemplate`, `TemplateItem`, GuiData fields.
- `internal/db/schema.go` — two tables + one index in `EnsureSchema`.
- `internal/web/webgui.go` — five routes.
- `internal/web/index.go` — load `Templates` for the home dropdown.
- `internal/web/templates/header.html` — "Plans" nav link.
- `internal/web/templates/index.html` — "Start from plan" dropdown, "Save as plan" button, apply-from-query bootstrap.
- `internal/web/public/js/index.js` — `applyPlan()` and `saveDayAsPlan()`.
- `internal/web/public/version` — version bump.

---

## Task 1: Data model (Go structs + GuiData fields)

**Files:**
- Modify: `internal/models/models.go`

- [ ] **Step 1: Add the two model structs**

Add after the `Workout` struct (around line 88) in `internal/models/models.go`:

```go
// WorkoutTemplate - a reusable, shared workout plan. Shared across all users
// (no owner column). Items hold prescribed exercises (sets/reps or duration,
// never weight). Items is populated by db.GetTemplate; it is not a DB column.
type WorkoutTemplate struct {
	ID        int    `db:"ID"`
	Name      string `db:"NAME"`
	Note      string `db:"NOTE"`
	CreatedAt string `db:"CREATED_AT"`
	Items     []TemplateItem
}

// TemplateItem - one prescribed exercise inside a WorkoutTemplate.
// TargetReps applies to MODE=reps exercises; TargetSeconds to MODE=timed or
// KIND=cardio. Zero = unset (apply falls back to a single blank row).
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

- [ ] **Step 2: Add GuiData fields**

In the `GuiData` struct (around line 157), add these fields before the closing brace:

```go
	Templates      []WorkoutTemplate // plan library (home dropdown + /plans/)
	TemplateCounts map[int]int       // template ID -> exercise count (library cards)
	OneTemplate    WorkoutTemplate   // for the builder form
```

- [ ] **Step 3: Build to verify it compiles**

Run: `cd cmd/ExerciseDiary && CGO_ENABLED=0 go build -o ../../tmp/ExerciseDiary . && cd ../..`
Expected: builds with no errors (the new types are unused so far, which is fine for a build).

- [ ] **Step 4: Commit**

```bash
git add internal/models/models.go
git commit -m "Workout templates: add WorkoutTemplate/TemplateItem models (mu.16)"
```

---

## Task 2: Schema (two new tables + index)

**Files:**
- Modify: `internal/db/schema.go`

- [ ] **Step 1: Add the tables and index in EnsureSchema**

In `internal/db/schema.go`, inside `EnsureSchema`, immediately after the `plates` `CREATE TABLE` block (after line 99) and before the `CREATE INDEX` calls, add:

```go
	exec(path, `CREATE TABLE IF NOT EXISTS workout_templates (
		"ID"         INTEGER PRIMARY KEY,
		"NAME"       TEXT DEFAULT '',
		"NOTE"       TEXT DEFAULT '',
		"CREATED_AT" TEXT DEFAULT ''
	);`)

	exec(path, `CREATE TABLE IF NOT EXISTS template_items (
		"ID"             INTEGER PRIMARY KEY,
		"TEMPLATE_ID"    INTEGER NOT NULL,
		"EXERCISE_ID"    INTEGER NOT NULL,
		"POSITION"       INTEGER DEFAULT 0,
		"TARGET_SETS"    INTEGER DEFAULT 0,
		"TARGET_REPS"    INTEGER DEFAULT 0,
		"TARGET_SECONDS" INTEGER DEFAULT 0,
		"NOTE"           TEXT DEFAULT ''
	);`)
```

- [ ] **Step 2: Add the index**

In the same function, alongside the other `CREATE INDEX IF NOT EXISTS` calls (after line 105), add:

```go
	exec(path, `CREATE INDEX IF NOT EXISTS idx_template_items_tpl ON template_items(TEMPLATE_ID);`)
```

- [ ] **Step 3: Build to verify**

Run: `cd cmd/ExerciseDiary && CGO_ENABLED=0 go build -o ../../tmp/ExerciseDiary . && cd ../..`
Expected: builds clean.

- [ ] **Step 4: Commit**

```bash
git add internal/db/schema.go
git commit -m "Workout templates: create workout_templates + template_items tables (mu.16)"
```

---

## Task 3: DB layer (`templates.go`) — TDD

This is the first test file in the repo. Tests use a temp-file SQLite DB (pure-Go driver, so no CGO needed). Run with `go test ./internal/db/`.

**Files:**
- Create: `internal/db/templates_test.go`
- Create: `internal/db/templates.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/db/templates_test.go`:

```go
package db

import (
	"path/filepath"
	"testing"

	"github.com/aceberg/ExerciseDiary/internal/models"
)

// newTestDB creates a throwaway SQLite file with the full schema applied.
func newTestDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	Create(path)       // base tables
	EnsureSchema(path) // added columns + template tables/index
	return path
}

func TestSaveAndGetTemplate(t *testing.T) {
	path := newTestDB(t)

	id := SaveTemplate(path,
		models.WorkoutTemplate{Name: "Push Day A", Note: "chest/tris"},
		[]models.TemplateItem{
			{ExerciseID: 1, TargetSets: 3, TargetReps: 5, Note: "slow eccentric"},
			{ExerciseID: 2, TargetSets: 3, TargetReps: 8},
		})
	if id == 0 {
		t.Fatal("expected non-zero template id")
	}

	got := GetTemplate(path, id)
	if got.Name != "Push Day A" || got.Note != "chest/tris" {
		t.Errorf("meta mismatch: %+v", got)
	}
	if len(got.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got.Items))
	}
	if got.Items[0].ExerciseID != 1 || got.Items[0].TargetSets != 3 || got.Items[0].TargetReps != 5 {
		t.Errorf("item0 mismatch: %+v", got.Items[0])
	}
	if got.Items[0].Note != "slow eccentric" {
		t.Errorf("item note not stored: %q", got.Items[0].Note)
	}
	if got.Items[0].Position != 0 || got.Items[1].Position != 1 {
		t.Errorf("positions not assigned by slice order: %+v", got.Items)
	}
}

func TestSaveTemplateReplacesItems(t *testing.T) {
	path := newTestDB(t)
	id := SaveTemplate(path, models.WorkoutTemplate{Name: "A"}, []models.TemplateItem{
		{ExerciseID: 1, TargetSets: 3, TargetReps: 5},
		{ExerciseID: 2, TargetSets: 3, TargetReps: 5},
	})
	SaveTemplate(path, models.WorkoutTemplate{ID: id, Name: "A v2"}, []models.TemplateItem{
		{ExerciseID: 9, TargetSets: 4, TargetReps: 10},
	})
	got := GetTemplate(path, id)
	if got.Name != "A v2" {
		t.Errorf("expected updated name, got %q", got.Name)
	}
	if len(got.Items) != 1 || got.Items[0].ExerciseID != 9 {
		t.Fatalf("expected items replaced, got %+v", got.Items)
	}
}

func TestDeleteTemplate(t *testing.T) {
	path := newTestDB(t)
	id := SaveTemplate(path, models.WorkoutTemplate{Name: "A"}, []models.TemplateItem{
		{ExerciseID: 1, TargetSets: 3, TargetReps: 5},
	})
	DeleteTemplate(path, id)
	if got := GetTemplate(path, id); got.ID != 0 {
		t.Errorf("expected template gone, got %+v", got)
	}
	if c := SelectTemplateCounts(path)[id]; c != 0 {
		t.Errorf("expected 0 items for deleted template, got %d", c)
	}
}

func TestSelectTemplatesNewestFirstAndCounts(t *testing.T) {
	path := newTestDB(t)
	id1 := SaveTemplate(path, models.WorkoutTemplate{Name: "First"}, nil)
	id2 := SaveTemplate(path, models.WorkoutTemplate{Name: "Second"}, []models.TemplateItem{
		{ExerciseID: 1, TargetSets: 3, TargetReps: 5},
		{ExerciseID: 2, TargetSets: 3, TargetReps: 5},
	})
	ts := SelectTemplates(path)
	if len(ts) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(ts))
	}
	if ts[0].ID != id2 || ts[1].ID != id1 {
		t.Errorf("expected newest first, got %d then %d", ts[0].ID, ts[1].ID)
	}
	counts := SelectTemplateCounts(path)
	if counts[id2] != 2 {
		t.Errorf("expected 2 items for id2, got %d", counts[id2])
	}
	if counts[id1] != 0 {
		t.Errorf("expected 0 items for id1, got %d", counts[id1])
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/db/ -run TestSaveAndGetTemplate -v`
Expected: FAIL — compile error / "undefined: SaveTemplate" (functions don't exist yet).

- [ ] **Step 3: Implement the DB layer**

Create `internal/db/templates.go`:

```go
package db

import (
	"fmt"
	"log"

	"github.com/aceberg/ExerciseDiary/internal/check"
	"github.com/aceberg/ExerciseDiary/internal/models"
)

// SelectTemplates - all plans, newest first (items NOT loaded).
func SelectTemplates(path string) []models.WorkoutTemplate {
	mu.Lock()
	dbx := connect(path)
	var ts []models.WorkoutTemplate
	err := dbx.Select(&ts, `SELECT * FROM workout_templates ORDER BY ID DESC;`)
	mu.Unlock()
	if err != nil {
		log.Printf("WARN: SelectTemplates: %v", err)
	}
	return ts
}

// GetTemplate - one plan with its items ordered by POSITION. Returns a
// zero-value struct (ID == 0) if the plan does not exist.
func GetTemplate(path string, id int) models.WorkoutTemplate {
	mu.Lock()
	dbx := connect(path)
	var t models.WorkoutTemplate
	_ = dbx.Get(&t, fmt.Sprintf(`SELECT * FROM workout_templates WHERE ID = '%d';`, id))
	err := dbx.Select(&t.Items, fmt.Sprintf(
		`SELECT * FROM template_items WHERE TEMPLATE_ID = '%d' ORDER BY POSITION ASC, ID ASC;`, id))
	mu.Unlock()
	if err != nil {
		log.Printf("WARN: GetTemplate items: %v", err)
	}
	return t
}

// SelectTemplateCounts - exercise count per template ID, for library cards.
func SelectTemplateCounts(path string) map[int]int {
	out := map[int]int{}
	mu.Lock()
	dbx := connect(path)
	rows, err := dbx.Queryx(`SELECT TEMPLATE_ID, COUNT(*) FROM template_items GROUP BY TEMPLATE_ID;`)
	mu.Unlock()
	if err != nil {
		log.Printf("WARN: SelectTemplateCounts: %v", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var tid, c int
		if err := rows.Scan(&tid, &c); err == nil {
			out[tid] = c
		}
	}
	return out
}

// SaveTemplate - insert (ID == 0) or update a plan, then full-replace its
// items from the provided slice. POSITION is assigned by slice order.
// Returns the template ID.
func SaveTemplate(path string, t models.WorkoutTemplate, items []models.TemplateItem) int {
	mu.Lock()
	defer mu.Unlock()
	dbx := connect(path)

	id := t.ID
	if id == 0 {
		res, err := dbx.Exec(fmt.Sprintf(
			`INSERT INTO workout_templates (NAME, NOTE, CREATED_AT) VALUES ('%s','%s','%s');`,
			quoteStr(t.Name), quoteStr(t.Note), quoteStr(t.CreatedAt)))
		check.IfError(err)
		lid, _ := res.LastInsertId()
		id = int(lid)
	} else {
		_, err := dbx.Exec(fmt.Sprintf(
			`UPDATE workout_templates SET NAME = '%s', NOTE = '%s' WHERE ID = '%d';`,
			quoteStr(t.Name), quoteStr(t.Note), id))
		check.IfError(err)
	}

	_, err := dbx.Exec(fmt.Sprintf(`DELETE FROM template_items WHERE TEMPLATE_ID = '%d';`, id))
	check.IfError(err)

	for pos, it := range items {
		_, err := dbx.Exec(fmt.Sprintf(
			`INSERT INTO template_items
			 (TEMPLATE_ID, EXERCISE_ID, POSITION, TARGET_SETS, TARGET_REPS, TARGET_SECONDS, NOTE)
			 VALUES ('%d','%d','%d','%d','%d','%d','%s');`,
			id, it.ExerciseID, pos, it.TargetSets, it.TargetReps, it.TargetSeconds, quoteStr(it.Note)))
		check.IfError(err)
	}
	return id
}

// DeleteTemplate - remove a plan and all of its items.
func DeleteTemplate(path string, id int) {
	exec(path, fmt.Sprintf(`DELETE FROM template_items WHERE TEMPLATE_ID = '%d';`, id))
	exec(path, fmt.Sprintf(`DELETE FROM workout_templates WHERE ID = '%d';`, id))
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/db/ -v`
Expected: PASS — all four tests (`TestSaveAndGetTemplate`, `TestSaveTemplateReplacesItems`, `TestDeleteTemplate`, `TestSelectTemplatesNewestFirstAndCounts`).

- [ ] **Step 5: Commit**

```bash
git add internal/db/templates.go internal/db/templates_test.go
git commit -m "Workout templates: DB layer + unit tests (mu.16)"
```

---

## Task 4: Library page (read-only list)

**Files:**
- Create: `internal/web/templates_page.go`
- Create: `internal/web/templates/templates_page.html`
- Modify: `internal/web/webgui.go` (route)
- Modify: `internal/web/templates/header.html` (nav link)

- [ ] **Step 1: Create the library handler**

Create `internal/web/templates_page.go`:

```go
package web

import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aceberg/ExerciseDiary/internal/db"
	"github.com/aceberg/ExerciseDiary/internal/models"
	"github.com/aceberg/ExerciseDiary/internal/prt"
)

// templatesHandler renders the shared plan library.
func templatesHandler(c *gin.Context) {
	user := currentUser(c)
	if user.ID == 0 {
		c.Redirect(http.StatusFound, "/users/")
		return
	}
	var guiData models.GuiData
	guiData.Config = appConfig
	guiData.Users = allUsers(c)
	guiData.CurrentUser = user
	guiData.Templates = db.SelectTemplates(appConfig.DBPath)
	guiData.TemplateCounts = db.SelectTemplateCounts(appConfig.DBPath)

	c.HTML(http.StatusOK, "header.html", guiData)
	c.HTML(http.StatusOK, "templates_page.html", guiData)
}
```

NOTE: the `sort`, `strconv`, `time`, and `prt` imports are unused until Task 5/6. Either add the remaining handlers now (recommended — do Tasks 5 & 6 handler steps together) or temporarily omit those imports. To keep tasks independently buildable, this step only imports what it uses: remove `sort`, `strconv`, `time`, `prt` from the import block for now and re-add them in Tasks 5/6.

For Step 1, use this import block:

```go
import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aceberg/ExerciseDiary/internal/db"
	"github.com/aceberg/ExerciseDiary/internal/models"
)
```

- [ ] **Step 2: Create the library page template**

Create `internal/web/templates/templates_page.html`:

```html
{{ define "templates_page.html" }}

<body>
<div class="container-lg mt-4">
  <div class="d-flex justify-content-between align-items-center mb-3">
    <h4 class="mb-0">Workout plans</h4>
    <a href="/plans/edit/new" class="btn btn-primary"><i class="bi bi-plus-lg"></i> New plan</a>
  </div>

  {{ if not .Templates }}
    <p class="text-muted">No plans yet. Create one with <strong>New plan</strong>, or build a workout and use <strong>Save as plan</strong> on the home page.</p>
  {{ end }}

  <div class="row">
    {{ range .Templates }}
    <div class="col-md-6 col-lg-4 mb-3">
      <div class="card border-primary h-100">
        <div class="card-header d-flex justify-content-between align-items-center">
          <span class="fw-bold">{{ if .Name }}{{ .Name }}{{ else }}(untitled){{ end }}</span>
          <span class="badge bg-secondary">{{ index $.TemplateCounts .ID }} ex</span>
        </div>
        <div class="card-body">
          {{ if .Note }}<p class="small text-muted mb-2">{{ .Note }}</p>{{ end }}
          <div class="d-flex gap-2 flex-wrap">
            <a href="/?apply={{ .ID }}" class="btn btn-sm btn-success"><i class="bi bi-clipboard-check"></i> Apply to today</a>
            <a href="/plans/edit/{{ .ID }}" class="btn btn-sm btn-outline-primary"><i class="bi bi-pencil"></i> Edit</a>
            <form action="/plans/del" method="post" onsubmit="return confirm('Delete this plan?');" class="d-inline">
              <input type="hidden" name="id" value="{{ .ID }}">
              <button type="submit" class="btn btn-sm btn-outline-danger"><i class="bi bi-trash"></i></button>
            </form>
          </div>
        </div>
      </div>
    </div>
    {{ end }}
  </div>
</div>

{{ template "footer.html" }}
{{ end }}
```

- [ ] **Step 3: Register the route**

In `internal/web/webgui.go`, in the GET block (after line 79, the equipment routes), add:

```go
	router.GET("/plans/", auth.Auth(&authConf), templatesHandler) // templates_page.go
```

- [ ] **Step 4: Add the nav link**

In `internal/web/templates/header.html`, after the Stats `<li>` (lines 70-72), add:

```html
              <li class="nav-item">
                <a class="nav-link active" href="/plans/">Plans</a>
              </li>
```

- [ ] **Step 5: Build, run, and verify manually**

Run: `cd cmd/ExerciseDiary && CGO_ENABLED=0 go build -o ../../tmp/ExerciseDiary . && cd ../..`
Expected: builds clean.

Then run `make run`, open the app, select a user, and click **Plans** in the nav.
Expected: the Plans page loads showing the empty-state message and a working **New plan** button (the builder 404s until Task 5 — that's expected).

- [ ] **Step 6: Commit**

```bash
git add internal/web/templates_page.go internal/web/templates/templates_page.html internal/web/webgui.go internal/web/templates/header.html
git commit -m "Workout templates: plan library page + nav link (mu.16)"
```

---

## Task 5: Builder page + save + delete + edit

**Files:**
- Modify: `internal/web/templates_page.go` (form, save, delete handlers + imports)
- Create: `internal/web/templates/template_form.html`
- Create: `internal/web/public/js/plans.js`
- Modify: `internal/web/webgui.go` (routes)

- [ ] **Step 1: Add the builder, save, and delete handlers**

In `internal/web/templates_page.go`, update the import block to:

```go
import (
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aceberg/ExerciseDiary/internal/db"
	"github.com/aceberg/ExerciseDiary/internal/models"
	"github.com/aceberg/ExerciseDiary/internal/prt"
)
```

Then append these three handlers:

```go
// templateFormHandler renders the builder for a new (id == "new") or existing plan.
func templateFormHandler(c *gin.Context) {
	user := currentUser(c)
	if user.ID == 0 {
		c.Redirect(http.StatusFound, "/users/")
		return
	}
	var guiData models.GuiData
	guiData.Config = appConfig
	guiData.Users = allUsers(c)
	guiData.CurrentUser = user
	guiData.ExData.Exs = db.SelectEx(appConfig.DBPath)
	sort.Slice(guiData.ExData.Exs, func(i, j int) bool {
		return guiData.ExData.Exs[i].Place < guiData.ExData.Exs[j].Place
	})

	if idStr := c.Param("id"); idStr != "new" {
		if id, err := strconv.Atoi(idStr); err == nil {
			guiData.OneTemplate = db.GetTemplate(appConfig.DBPath, id)
		}
	}

	c.HTML(http.StatusOK, "header.html", guiData)
	c.HTML(http.StatusOK, "template_form.html", guiData)
}

// saveTemplateHandler parses the builder form (parallel arrays, like set.go)
// and upserts the plan. Blank rows (no exercise chosen) are skipped.
func saveTemplateHandler(c *gin.Context) {
	user := currentUser(c)
	if user.ID == 0 {
		c.Redirect(http.StatusFound, "/users/")
		return
	}
	_ = c.PostFormMap("x") // force form parse (mirror set.go)
	form := c.Request.PostForm

	t := models.WorkoutTemplate{
		Name: firstOr(form["name"], ""),
		Note: firstOr(form["note"], ""),
	}
	if idStr := firstOr(form["template_id"], ""); idStr != "" {
		t.ID, _ = strconv.Atoi(idStr)
	}
	if t.ID == 0 {
		t.CreatedAt = time.Now().Format("2006-01-02 15:04:05")
	}

	exIDs := form["ex_id"]
	sets := form["target_sets"]
	reps := form["target_reps"]
	secs := form["target_seconds"]
	inotes := form["item_note"]

	var items []models.TemplateItem
	for i := 0; i < len(exIDs); i++ {
		exID, _ := strconv.Atoi(exIDs[i])
		if exID == 0 {
			continue // blank row
		}
		it := models.TemplateItem{ExerciseID: exID}
		if i < len(sets) {
			it.TargetSets, _ = strconv.Atoi(sets[i])
		}
		if i < len(reps) {
			it.TargetReps, _ = strconv.Atoi(reps[i])
		}
		if i < len(secs) {
			it.TargetSeconds = prt.ParseMMSS(secs[i]) // "" -> 0
		}
		if i < len(inotes) {
			it.Note = inotes[i]
		}
		items = append(items, it)
	}

	db.SaveTemplate(appConfig.DBPath, t, items)
	c.Redirect(http.StatusFound, "/plans/")
}

// deleteTemplateHandler removes a plan and its items.
func deleteTemplateHandler(c *gin.Context) {
	user := currentUser(c)
	if user.ID == 0 {
		c.Redirect(http.StatusFound, "/users/")
		return
	}
	if id, _ := strconv.Atoi(c.PostForm("id")); id != 0 {
		db.DeleteTemplate(appConfig.DBPath, id)
	}
	c.Redirect(http.StatusFound, "/plans/")
}
```

NOTE: `firstOr` already exists in `internal/web/set.go` — reuse it, do not redefine. `prt.ParseMMSS` already exists (used by `set.go`).

- [ ] **Step 2: Create the builder template**

Create `internal/web/templates/template_form.html`:

```html
{{ define "template_form.html" }}
<script src="https://cdn.jsdelivr.net/npm/sortablejs@1.15.6/Sortable.min.js"></script>

<body>
<div class="container-lg mt-4">
  <form action="/plans/" method="post" id="planForm">
    <input type="hidden" name="template_id" value="{{ .OneTemplate.ID }}">

    <div class="card border-primary mb-3">
      <div class="card-header">Plan details</div>
      <div class="card-body">
        <input name="name" id="planName" class="form-control mb-2" required
               placeholder="Plan name (e.g. Push Day A)" value="{{ .OneTemplate.Name }}">
        <textarea name="note" id="planNote" rows="2" class="form-control"
                  placeholder="Instructions / notes for this plan">{{ .OneTemplate.Note }}</textarea>
      </div>
    </div>

    <div class="card border-primary mb-3">
      <div class="card-header d-flex justify-content-between align-items-center">
        <span>Exercises</span>
        <button type="button" class="btn btn-sm btn-primary" onclick="addItemRow()">
          <i class="bi bi-plus-lg"></i> Add exercise
        </button>
      </div>
      <div class="card-body">
        <datalist id="exerciseOptions">
          {{ range .ExData.Exs }}<option value="{{ .Name }}"></option>{{ end }}
        </datalist>
        <div id="itemRows"></div>
        <p class="small text-muted mt-2 mb-0">Drag <i class="bi bi-grip-vertical"></i> to reorder. Weight is not prescribed — each person fills it in when they train.</p>
      </div>
    </div>

    <button type="submit" class="btn btn-success"><i class="bi bi-save"></i> Save plan</button>
    <a href="/plans/" class="btn btn-secondary">Cancel</a>
  </form>
</div>

<script>
  window.allExercises  = {{ .ExData.Exs }};
  window.templateItems = {{ .OneTemplate.Items }};
</script>
<script src="/fs/public/js/plans.js"></script>

{{ template "footer.html" }}
{{ end }}
```

- [ ] **Step 3: Create plans.js**

Create `internal/web/public/js/plans.js`:

```js
// plans.js — Workout plan builder. Renders editable item rows, swaps the
// target input between reps and mm:ss based on the chosen exercise's kind/mode,
// and supports drag-reorder. Submission posts parallel arrays:
//   ex_id[], target_sets[], target_reps[], target_seconds[], item_note[]
// Every row always contributes exactly one value to each array (hidden inputs
// keep them aligned), so the server can index them positionally.

(function () {
    let rowSeq = 0;

    function escAttr(s) {
        return String(s == null ? "" : s)
            .replace(/&/g, "&amp;").replace(/"/g, "&quot;")
            .replace(/</g, "&lt;").replace(/>/g, "&gt;");
    }
    function exByName(name) {
        for (const e of (window.allExercises || [])) if (e.Name === name) return e;
        return null;
    }
    function exByID(id) {
        for (const e of (window.allExercises || [])) if (e.ID === id) return e;
        return null;
    }
    function isTimed(ex) {
        return !!ex && (ex.Kind === "cardio" || ex.Mode === "timed");
    }
    function secondsToMMSS(sec) {
        sec = parseInt(sec, 10) || 0;
        if (sec <= 0) return "";
        const m = Math.floor(sec / 60), s = sec % 60;
        return m + ":" + String(s).padStart(2, "0");
    }

    // targetInputHTML returns the target widget plus a hidden counterpart so
    // both target_reps[] and target_seconds[] arrays stay length-aligned.
    function targetInputHTML(timed, prefill) {
        prefill = prefill || {};
        if (timed) {
            return '<input name="target_seconds" class="form-control" placeholder="mm:ss" value="' + secondsToMMSS(prefill.TargetSeconds) + '">' +
                   '<input type="hidden" name="target_reps" value="0">';
        }
        return '<input name="target_reps" type="number" min="0" class="form-control" placeholder="Reps" value="' + (prefill.TargetReps || "") + '">' +
               '<input type="hidden" name="target_seconds" value="">';
    }

    // addItemRow appends one builder row. prefill is optional:
    // {ExerciseID, TargetSets, TargetReps, TargetSeconds, Note}.
    window.addItemRow = function (prefill) {
        prefill = prefill || {};
        rowSeq += 1;
        const rid = "item-" + rowSeq;
        const ex = prefill.ExerciseID ? exByID(prefill.ExerciseID) : null;
        const exName = ex ? ex.Name : "";
        const timed = isTimed(ex);

        const row = document.createElement("div");
        row.className = "row g-2 align-items-center mb-2 plan-item-row";
        row.id = rid;
        row.innerHTML =
            '<div class="col-auto plan-drag" style="cursor:grab" title="Drag to reorder"><i class="bi bi-grip-vertical"></i></div>' +
            '<input type="hidden" name="ex_id" value="' + (ex ? ex.ID : "") + '">' +
            '<div class="col"><input list="exerciseOptions" class="form-control plan-ex-name" placeholder="Exercise" value="' + escAttr(exName) + '"></div>' +
            '<div class="col-2"><input name="target_sets" type="number" min="0" class="form-control" placeholder="Sets" value="' + (prefill.TargetSets || "") + '"></div>' +
            '<div class="col-2 plan-target">' + targetInputHTML(timed, prefill) + '</div>' +
            '<div class="col"><input name="item_note" class="form-control" placeholder="Cue / note" value="' + escAttr(prefill.Note) + '"></div>' +
            '<div class="col-auto"><button type="button" class="btn del-set-button" title="Remove"><i class="bi bi-x-lg"></i></button></div>';
        document.getElementById("itemRows").appendChild(row);

        // Resolve hidden ex_id and swap target widget when the exercise changes.
        const nameInput = row.querySelector(".plan-ex-name");
        nameInput.addEventListener("change", function () {
            const picked = exByName(this.value);
            row.querySelector('input[name="ex_id"]').value = picked ? picked.ID : "";
            row.querySelector(".plan-target").innerHTML = targetInputHTML(isTimed(picked), {});
        });

        // Remove button.
        row.querySelector(".del-set-button").addEventListener("click", function () {
            row.remove();
        });
    };

    document.addEventListener("DOMContentLoaded", function () {
        const itemRows = document.getElementById("itemRows");
        if (!itemRows) return;

        // Prefill source: "save as plan" stash (sessionStorage) wins, else the
        // existing template's items, else a single empty row.
        let items = window.templateItems || [];
        const stash = sessionStorage.getItem("planFromDay");
        if (stash) {
            try { items = JSON.parse(stash); } catch (e) { items = []; }
            sessionStorage.removeItem("planFromDay");
            const nm = sessionStorage.getItem("planFromDayName");
            if (nm) {
                const nameEl = document.getElementById("planName");
                if (nameEl) nameEl.value = nm;
                sessionStorage.removeItem("planFromDayName");
            }
        }

        if (!items.length) {
            addItemRow();
        } else {
            items.forEach(it => addItemRow(it));
        }

        if (window.Sortable) {
            new Sortable(itemRows, { handle: ".plan-drag", animation: 150 });
        }
    });
})();
```

- [ ] **Step 4: Register the routes**

In `internal/web/webgui.go`, add the GET route alongside the other `/plans/` GET (after the `/plans/` line from Task 4):

```go
	router.GET("/plans/edit/:id", auth.Auth(&authConf), templateFormHandler) // templates_page.go
```

And in the POST block (after line 91, the equipment POST), add:

```go
	router.POST("/plans/", auth.Auth(&authConf), saveTemplateHandler)    // templates_page.go
	router.POST("/plans/del", auth.Auth(&authConf), deleteTemplateHandler) // templates_page.go
```

- [ ] **Step 5: Build, run, verify manually**

Run: `cd cmd/ExerciseDiary && CGO_ENABLED=0 go build -o ../../tmp/ExerciseDiary . && cd ../..`
Expected: builds clean.

Then `make run` and verify:
1. **Plans → New plan** opens the builder with one empty row.
2. Add 2-3 exercises; pick a **reps** exercise (shows a Reps box) and a **timed** exercise like a plank (shows an mm:ss box). Confirm the box swaps when you change the exercise.
3. Set sets/reps, drag to reorder, set a name, **Save plan** → redirects to the library showing the new card with the right exercise count.
4. **Edit** the plan → builder loads with the saved rows in order, reps/time boxes correct → change something, save → persists.
5. **Delete** a plan → confirm dialog → card disappears.

- [ ] **Step 6: Commit**

```bash
git add internal/web/templates_page.go internal/web/templates/template_form.html internal/web/public/js/plans.js internal/web/webgui.go
git commit -m "Workout templates: builder page, save/edit/delete (mu.16)"
```

---

## Task 6: Apply a plan to today

**Files:**
- Modify: `internal/web/templates_page.go` (JSON endpoint)
- Modify: `internal/web/webgui.go` (route)
- Modify: `internal/web/index.go` (load Templates)
- Modify: `internal/web/templates/index.html` (dropdown + apply-from-query bootstrap)
- Modify: `internal/web/public/js/index.js` (`applyPlan`)

- [ ] **Step 1: Add the JSON endpoint**

In `internal/web/templates_page.go`, append:

```go
// templateJSONHandler returns a resolved plan for client-side apply. Items are
// joined to the current exercise list (Name/Kind/Mode/Color/Group); items whose
// exercise no longer exists are dropped.
func templateJSONHandler(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	t := db.GetTemplate(appConfig.DBPath, id)

	byID := map[int]models.Exercise{}
	for _, e := range db.SelectEx(appConfig.DBPath) {
		byID[e.ID] = e
	}

	type applyItem struct {
		Name          string `json:"Name"`
		Group         string `json:"Group"`
		Kind          string `json:"Kind"`
		Mode          string `json:"Mode"`
		Color         string `json:"Color"`
		TargetSets    int    `json:"TargetSets"`
		TargetReps    int    `json:"TargetReps"`
		TargetSeconds int    `json:"TargetSeconds"`
		Note          string `json:"Note"`
	}
	out := []applyItem{}
	for _, it := range t.Items {
		e, ok := byID[it.ExerciseID]
		if !ok {
			continue
		}
		out = append(out, applyItem{
			Name: e.Name, Group: e.Group, Kind: e.Kind, Mode: e.Mode, Color: e.Color,
			TargetSets: it.TargetSets, TargetReps: it.TargetReps,
			TargetSeconds: it.TargetSeconds, Note: it.Note,
		})
	}

	c.JSON(http.StatusOK, gin.H{"name": t.Name, "note": t.Note, "items": out})
}
```

- [ ] **Step 2: Register the JSON route**

In `internal/web/webgui.go`, alongside the other `/plans/` GET routes:

```go
	router.GET("/plans/json/:id", auth.Auth(&authConf), templateJSONHandler) // templates_page.go
```

- [ ] **Step 3: Load templates on the home page**

In `internal/web/index.go`, inside `indexHandler`, after the `guiData.TodayWorkout = ...` line (line 35), add:

```go
	guiData.Templates = db.SelectTemplates(appConfig.DBPath)
```

- [ ] **Step 4: Add `applyPlan` to index.js**

In `internal/web/public/js/index.js`, add this function near the other `window.*` helpers (e.g. after `addExercise`, around line 95):

```js
// applyPlan(id): fetch a saved plan and inject its planned rows into today's
// form via addExercise(). Skips exercises already present (append + skip-dup).
// Nothing persists until the user hits Save (identical to manual entry).
window.applyPlan = function (id) {
    fetch('/plans/json/' + id)
        .then(r => r.json())
        .then(plan => {
            const present = new Set();
            document.querySelectorAll('#todayEx tr[data-name]').forEach(tr => {
                if (tr.dataset.name) present.add(tr.dataset.name);
            });

            (plan.items || []).forEach(it => {
                if (present.has(it.Name)) return;
                present.add(it.Name);
                const sets = it.TargetSets > 0 ? it.TargetSets : 1;
                for (let s = 0; s < sets; s++) {
                    addExercise({
                        Name: it.Name, Group: it.Group, Kind: it.Kind, Mode: it.Mode, Color: it.Color,
                        Reps: it.TargetReps || 0,
                        DurationSeconds: it.TargetSeconds || 0,
                        Note: it.Note || "",
                        Completed: 0
                    });
                }
            });

            const nameEl = document.getElementById('workoutName');
            const noteEl = document.getElementById('workoutNote');
            if (nameEl && !nameEl.value && plan.name) nameEl.value = plan.name;
            if (noteEl && !noteEl.value && plan.note) noteEl.value = plan.note;
        })
        .catch(e => console.error('applyPlan failed', e));
};
```

- [ ] **Step 5: Add the "Start from plan" dropdown + apply-from-query bootstrap**

In `internal/web/templates/index.html`, add the dropdown just above the `workout_name` input (before line 257). It only renders if plans exist:

```html
                    {{ if .Templates }}
                    <div class="dropdown mb-2">
                      <button class="btn btn-outline-primary btn-sm dropdown-toggle" type="button" data-bs-toggle="dropdown" aria-expanded="false">
                        <i class="bi bi-clipboard-check"></i> Start from plan
                      </button>
                      <ul class="dropdown-menu">
                        {{ range .Templates }}
                        <li><a class="dropdown-item" href="#" onclick="applyPlan({{ .ID }}); return false;">{{ if .Name }}{{ .Name }}{{ else }}(untitled){{ end }}</a></li>
                        {{ end }}
                      </ul>
                    </div>
                    {{ end }}
```

Then, in the bottom `<script>` block, immediately after `setFormDate(window.allSets);` (line ~311), add:

```html
  // If arriving from a library "Apply to today" link (?apply=ID), apply it now
  // that today's existing rows are rendered (so skip-duplicate sees them).
  (function () {
    const p = new URLSearchParams(window.location.search).get('apply');
    if (p && typeof window.applyPlan === 'function') {
      window.applyPlan(p);
      history.replaceState({}, '', '/');
    }
  })();
```

- [ ] **Step 6: Build, run, verify manually**

Run: `cd cmd/ExerciseDiary && CGO_ENABLED=0 go build -o ../../tmp/ExerciseDiary . && cd ../..`
Expected: builds clean.

Then `make run` and verify:
1. On the home page (with at least one plan saved), **Start from plan → <plan>** injects the prescribed number of planned rows (greyed/uncompleted), reps pre-filled, weight blank; timed exercises show their mm:ss target.
2. Apply the **same** plan again → no duplicate rows appear (skip-dup works).
3. Add one of the plan's exercises manually first, then apply → only the *other* exercises get added.
4. From **Plans → Apply to today** on a card → lands on home with rows staged and the URL cleaned to `/`.
5. Fill in weights, mark sets complete, **Save** → reloads with everything persisted.

- [ ] **Step 7: Commit**

```bash
git add internal/web/templates_page.go internal/web/webgui.go internal/web/index.go internal/web/templates/index.html internal/web/public/js/index.js
git commit -m "Workout templates: apply plan to today (home dropdown + library button) (mu.16)"
```

---

## Task 7: Save this day as a plan

**Files:**
- Modify: `internal/web/templates/index.html` (button)
- Modify: `internal/web/public/js/index.js` (`saveDayAsPlan` + helpers)

- [ ] **Step 1: Add `saveDayAsPlan` and helpers to index.js**

In `internal/web/public/js/index.js`, add near `applyPlan`:

```js
// mmssToSeconds parses "m:ss" or plain seconds into integer seconds.
function mmssToSeconds(v) {
    v = String(v || "").trim();
    if (!v) return 0;
    if (v.indexOf(":") === -1) return parseInt(v, 10) || 0;
    const parts = v.split(":");
    const m = parseInt(parts[0], 10) || 0;
    const s = parseInt(parts[1], 10) || 0;
    return m * 60 + s;
}

// modeValue returns the most common value in an array (0 if empty).
function modeValue(arr) {
    if (!arr || !arr.length) return 0;
    const counts = {};
    let best = arr[0], bestN = 0;
    for (const v of arr) {
        counts[v] = (counts[v] || 0) + 1;
        if (counts[v] > bestN) { bestN = counts[v]; best = v; }
    }
    return best;
}

function exObjForName(name) {
    for (const e of (window.allExercises || [])) if (e.Name === name) return e;
    return null;
}

// saveDayAsPlan snapshots today's distinct exercises into a new plan: one item
// per exercise, TargetSets = number of rows, TargetReps/TargetSeconds = the
// most common value among them. Weight is ignored. Hands off to the builder
// (pre-filled via sessionStorage) so the user can tweak before saving.
window.saveDayAsPlan = function () {
    const order = [];
    const map = {};
    document.querySelectorAll('#todayEx tr[data-name]').forEach(tr => {
        if (tr.id && (tr.id.startsWith('hist-') || tr.id.startsWith('adv-'))) return;
        const name = tr.dataset.name;
        if (!name) return;
        if (!map[name]) { map[name] = { count: 0, reps: [], secs: [] }; order.push(name); }
        map[name].count += 1;
        const repsEl = tr.querySelector('input[name="reps"]');
        if (repsEl && repsEl.value) map[name].reps.push(parseInt(repsEl.value, 10) || 0);
        const durEl = tr.querySelector('input[name="duration_mmss"]');
        if (durEl && durEl.value) map[name].secs.push(mmssToSeconds(durEl.value));
    });

    if (!order.length) { alert('Add some exercises first.'); return; }

    const planName = prompt('Name this plan:', (document.getElementById('workoutName') || {}).value || '');
    if (planName === null) return; // cancelled

    const items = order.map(name => {
        const e = exObjForName(name);
        const m = map[name];
        return {
            ExerciseID: e ? e.ID : 0,
            TargetSets: m.count,
            TargetReps: modeValue(m.reps),
            TargetSeconds: modeValue(m.secs),
            Note: ""
        };
    }).filter(it => it.ExerciseID !== 0);

    try {
        sessionStorage.setItem('planFromDay', JSON.stringify(items));
        sessionStorage.setItem('planFromDayName', planName);
    } catch (e) {}
    window.location.href = '/plans/edit/new';
};
```

- [ ] **Step 2: Add the "Save as plan" button**

In `internal/web/templates/index.html`, near the existing workout Save button (the form's submit area in the today column), add a button. It must be `type="button"` so it doesn't submit the workout form:

```html
                    <button type="button" class="btn btn-outline-secondary btn-sm" onclick="saveDayAsPlan()">
                      <i class="bi bi-bookmark-plus"></i> Save as plan
                    </button>
```

(Place it next to the existing save/submit control in the today-workout card. Locate the today form's button row and add this beside it.)

- [ ] **Step 3: Build, run, verify manually**

Run: `cd cmd/ExerciseDiary && CGO_ENABLED=0 go build -o ../../tmp/ExerciseDiary . && cd ../..`
Expected: builds clean.

Then `make run` and verify:
1. Build a day with, say, Bench ×3 rows (reps 5) and a Plank ×2 rows (60s) → **Save as plan** → prompt for name → builder opens pre-filled: Bench with sets=3 reps=5, Plank with sets=2 and a 1:00 mm:ss target.
2. Tweak and **Save plan** → appears in the library.
3. Empty day → **Save as plan** → shows the "Add some exercises first" alert.

- [ ] **Step 4: Commit**

```bash
git add internal/web/templates/index.html internal/web/public/js/index.js
git commit -m "Workout templates: save current day as a plan (mu.16)"
```

---

## Task 8: Version bump + full verification

**Files:**
- Modify: `internal/web/public/version`

- [ ] **Step 1: Bump the version**

Read `internal/web/public/version` (currently `VERSION=0.1.9-mu.15`) and bump to:

```
VERSION=0.1.9-mu.16
```

- [ ] **Step 2: Run the DB tests + full build**

Run: `go test ./internal/db/ -v`
Expected: all template tests PASS.

Run: `cd cmd/ExerciseDiary && CGO_ENABLED=0 go build -o ../../tmp/ExerciseDiary . && cd ../..`
Expected: builds clean.

Run: `make fmt` (gofmt the new files) — expect no diffs after, or stage the formatting.

- [ ] **Step 3: End-to-end manual verification (cross-user)**

`make run`, then:
1. As **Ryan**: build "Push Day A" (Bench 3×5, OHP 3×8, Plank 2×60s), save.
2. As Ryan: home → **Start from plan → Push Day A** → rows appear, enter weights, mark complete, Save.
3. Switch to **Akane** (user dropdown): the same plan is visible in **Plans** (shared library) → Apply to today → she logs her own (different) weights.
4. Edit the plan, delete the plan — both reflect immediately.
5. On a phone-width viewport, confirm the builder rows and the "Start from plan" dropdown are usable (mobile-first check).

- [ ] **Step 4: Commit**

```bash
git add internal/web/public/version
git commit -m "Bump version to 0.1.9-mu.16 (Workout Builder)"
```

- [ ] **Step 5: Deploy (optional, when ready)**

Per project workflow, deploying is a separate gated step (backup before cutover). Use the `unraid-app-deploy` skill when you want to push to GHCR + Unraid. Do not deploy without explicit go-ahead.

---

## Self-Review

**Spec coverage:**
- Data model (2 tables, no owner, target reps/seconds) → Tasks 1-2. ✓
- Shared library, anyone edit/delete → Task 4 (list) + Task 5 (edit/delete). ✓
- Builder with reps↔time swap + drag reorder → Task 5. ✓
- Apply = append + skip-dup, client-side via addExercise, persist on Save → Task 6. ✓
- Apply from both home dropdown and library button → Task 6. ✓
- Save day as plan (distinct exercises, sets=count, reps/seconds=mode) → Task 7. ✓
- Empty-name guard → builder uses `required` on the name input (Task 5, Step 2) — blocks save with a browser hint, matching the spec decision. ✓
- Deleted-exercise handling → JSON endpoint drops missing exercises (Task 6); builder shows the row only if the exercise resolves. Note: a deleted exercise's item is silently dropped from the builder too (its `addItemRow` receives an unresolved ExerciseID → blank name row). Acceptable; the user can re-pick or remove. ✓
- Additive migration + version bump → Tasks 2 & 8. ✓

**Placeholder scan:** No TBD/TODO; all steps contain concrete code/commands. One NOTE in Task 6 Step 3 corrects an intermediate function name — the final instruction is to use `db.SelectTemplates`. ✓

**Type consistency:**
- `SaveTemplate(path, WorkoutTemplate, []TemplateItem) int`, `GetTemplate(path, id) WorkoutTemplate`, `SelectTemplates(path)`, `SelectTemplateCounts(path) map[int]int`, `DeleteTemplate(path, id)` — used consistently across Tasks 3-6. ✓
- JSON apply keys are PascalCase (`Name`, `TargetReps`, ...) matching `addExercise(obj)` expectations and `applyPlan` reads. Plan-level keys `name`/`note` are lowercase (gin.H) and read as `plan.name`/`plan.note`. ✓
- Form field names (`ex_id`, `target_sets`, `target_reps`, `target_seconds`, `item_note`, `template_id`, `name`, `note`) match between `plans.js` (Task 5) and `saveTemplateHandler` (Task 5). ✓
- `firstOr` and `prt.ParseMMSS` reused, not redefined. ✓
