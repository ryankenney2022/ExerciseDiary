package web

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/aceberg/ExerciseDiary/internal/db"
	"github.com/aceberg/ExerciseDiary/internal/models"
	"github.com/aceberg/ExerciseDiary/internal/prt"
)

// prtListHandler - GET /prt/
func prtListHandler(c *gin.Context) {
	user := currentUser(c)
	if user.ID == 0 {
		c.Redirect(http.StatusFound, "/users/")
		return
	}

	var guiData models.GuiData
	guiData.Config = appConfig
	guiData.Users = allUsers(c)
	guiData.CurrentUser = user
	guiData.PRTTests = db.SelectPRTByUser(appConfig.DBPath, user.ID)

	c.HTML(http.StatusOK, "header.html", guiData)
	c.HTML(http.StatusOK, "prt_list.html", guiData)
}

// prtFormHandler - GET /prt/new or /prt/edit/:id
func prtFormHandler(c *gin.Context) {
	user := currentUser(c)
	if user.ID == 0 {
		c.Redirect(http.StatusFound, "/users/")
		return
	}

	var guiData models.GuiData
	guiData.Config = appConfig
	guiData.Users = allUsers(c)
	guiData.CurrentUser = user

	if idStr := c.Param("id"); idStr != "" {
		if id, err := strconv.Atoi(idStr); err == nil {
			guiData.OnePRT = db.GetPRT(appConfig.DBPath, id)
		}
	}

	c.HTML(http.StatusOK, "header.html", guiData)
	c.HTML(http.StatusOK, "prt_form.html", guiData)
}

// prtSaveHandler - POST /prt/
func prtSaveHandler(c *gin.Context) {
	user := currentUser(c)
	if user.ID == 0 {
		c.Redirect(http.StatusFound, "/users/")
		return
	}

	idStr := c.PostForm("id")
	id, _ := strconv.Atoi(idStr)

	date := c.PostForm("date")
	if date == "" {
		c.Redirect(http.StatusFound, "/prt/")
		return
	}

	pushups, _ := strconv.Atoi(c.PostForm("pushups"))
	plankSeconds := prt.ParseMMSS(c.PostForm("plank_mmss"))
	runSeconds := prt.ParseMMSS(c.PostForm("run_mmss"))
	note := c.PostForm("note")

	// Snapshot user attributes at test date so the score is stable later.
	t := models.PRTTest{
		ID:             id,
		UserID:         user.ID,
		Date:           date,
		Note:           note,
		SexAtTest:      user.Sex,
		AgeAtTest:      prt.AgeOnDate(user.DOB, date),
		AltitudeAtTest: user.Altitude,
		Pushups:        pushups,
		PlankSeconds:   plankSeconds,
		RunSeconds:     runSeconds,
	}

	computeAndStash(&t)

	if id > 0 {
		db.UpdatePRT(appConfig.DBPath, t)
	} else {
		db.InsertPRT(appConfig.DBPath, t)
	}
	c.Redirect(http.StatusFound, "/prt/")
}

// prtDeleteHandler - POST /prtdel/
func prtDeleteHandler(c *gin.Context) {
	idStr := c.PostForm("id")
	if id, err := strconv.Atoi(idStr); err == nil && id > 0 {
		db.DeletePRT(appConfig.DBPath, id)
	}
	c.Redirect(http.StatusFound, "/prt/")
}

// computeAndStash fills the cached score fields on a PRTTest in-place using
// the snapshot attributes already set on the test.
func computeAndStash(t *models.PRTTest) {
	if t.SexAtTest == "" || t.AgeAtTest == 0 {
		// Can't score without profile info; leave scores at 0.
		t.PushupScore, t.PlankScore, t.RunScore = 0, 0, 0
		t.OverallScore, t.OverallCategory = 0, ""
		return
	}

	results := []prt.Result{}

	if t.Pushups > 0 {
		r := prt.Score(t.SexAtTest, t.AgeAtTest, t.AltitudeAtTest, prt.EventPushups, t.Pushups)
		t.PushupScore = r.Score
		results = append(results, r)
	}
	if t.PlankSeconds > 0 {
		r := prt.Score(t.SexAtTest, t.AgeAtTest, t.AltitudeAtTest, prt.EventPlank, t.PlankSeconds)
		t.PlankScore = r.Score
		results = append(results, r)
	}
	if t.RunSeconds > 0 {
		r := prt.Score(t.SexAtTest, t.AgeAtTest, t.AltitudeAtTest, prt.EventRun, t.RunSeconds)
		t.RunScore = r.Score
		results = append(results, r)
	}

	overall, cat, _ := prt.Overall(results...)
	if overall < 0 {
		overall = 0
	}
	t.OverallScore = overall
	t.OverallCategory = cat
}

// prtEventLabel - template helper: looks up event score → label for display.
func prtEventLabel(sex string, age int, altitude, event string, raw int) string {
	if sex == "" || age == 0 || raw == 0 {
		return ""
	}
	r := prt.Score(sex, age, altitude, event, raw)
	return r.Label()
}

// prtFormatSeconds - template helper for mm:ss display.
func prtFormatSeconds(secs int) string { return prt.FormatMMSS(secs) }
