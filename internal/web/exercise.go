package web

import (
	"net/http"
	"regexp"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"github.com/aceberg/ExerciseDiary/internal/db"
	"github.com/aceberg/ExerciseDiary/internal/models"
)

func exerciseHandler(c *gin.Context) {
	var guiData models.GuiData
	var id int

	exData.Exs = db.SelectEx(appConfig.DBPath)

	guiData.Config = appConfig
	guiData.ExData = exData
	guiData.GroupMap = createGroupMap()
	guiData.Users = allUsers(c)
	guiData.CurrentUser = currentUser(c)
	guiData.ExGroups, guiData.ExPlaces = collectGroupsAndPlaces(exData.Exs)

	idStr, ok := c.GetQuery("id")

	if ok && (idStr != "new") {
		id, _ = strconv.Atoi(idStr)

		for _, oneEx := range exData.Exs {
			if oneEx.ID == id {
				guiData.OneEx = oneEx
				break
			}
		}
	}

	c.HTML(http.StatusOK, "header.html", guiData)
	c.HTML(http.StatusOK, "exercise.html", guiData)
}

func saveExerciseHandler(c *gin.Context) {
	var oneEx models.Exercise

	oneEx.Group = c.PostForm("group")
	oneEx.Place = c.PostForm("place")
	oneEx.Name = c.PostForm("name")
	oneEx.Descr = c.PostForm("descr")
	oneEx.Image = c.PostForm("image")
	oneEx.VideoURL = c.PostForm("video_url")
	oneEx.Kind = c.PostForm("kind")
	if oneEx.Kind != "cardio" {
		oneEx.Kind = "strength"
	}

	id := c.PostForm("id")
	weight := c.PostForm("weight")
	reps := c.PostForm("reps")

	oneEx.ID, _ = strconv.Atoi(id)
	oneEx.Weight, _ = decimal.NewFromString(weight)
	oneEx.Reps, _ = strconv.Atoi(reps)

	if oneEx.ID != 0 {
		db.DeleteEx(appConfig.DBPath, oneEx.ID)
	}

	db.InsertEx(appConfig.DBPath, oneEx)
	exData.Exs = db.SelectEx(appConfig.DBPath)

	c.Redirect(http.StatusFound, "/")
}

func deleteExerciseHandler(c *gin.Context) {

	idStr := c.PostForm("id")
	id, _ := strconv.Atoi(idStr)

	db.DeleteEx(appConfig.DBPath, id)
	exData.Exs = db.SelectEx(appConfig.DBPath)

	c.Redirect(http.StatusFound, "/")
}

// collectGroupsAndPlaces returns distinct, sorted lists of group names and
// place-in-group values across the exercise library. Used to power the
// datalist combo inputs on the exercise edit form.
func collectGroupsAndPlaces(exs []models.Exercise) (groups, places []string) {
	gset := map[string]struct{}{}
	pset := map[string]struct{}{}
	for _, e := range exs {
		if e.Group != "" {
			gset[e.Group] = struct{}{}
		}
		if e.Place != "" {
			pset[e.Place] = struct{}{}
		}
	}
	for g := range gset {
		groups = append(groups, g)
	}
	for p := range pset {
		places = append(places, p)
	}
	sort.Strings(groups)
	// Sort places numerically when possible so "2" comes before "10".
	sort.Slice(places, func(i, j int) bool {
		a, aerr := strconv.Atoi(places[i])
		b, berr := strconv.Atoi(places[j])
		if aerr == nil && berr == nil {
			return a < b
		}
		return places[i] < places[j]
	})
	return groups, places
}

var ytIDRe = regexp.MustCompile(`(?:youtube\.com/(?:watch\?v=|embed/|shorts/)|youtu\.be/)([A-Za-z0-9_-]{6,})`)

// YouTubeEmbed converts a YouTube watch / short / share URL to the embed form.
// Returns "" if the input doesn't look like a YouTube URL.
func YouTubeEmbed(raw string) string {
	if raw == "" {
		return ""
	}
	m := ytIDRe.FindStringSubmatch(raw)
	if len(m) < 2 {
		return ""
	}
	return "https://www.youtube.com/embed/" + m[1]
}
