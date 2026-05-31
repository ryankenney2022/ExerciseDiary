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
