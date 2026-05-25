package web

import (
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aceberg/ExerciseDiary/internal/db"
	"github.com/aceberg/ExerciseDiary/internal/models"
)

func statsHandler(c *gin.Context) {
	var guiData models.GuiData

	user := currentUser(c)
	if user.ID == 0 {
		c.Redirect(http.StatusFound, "/users/")
		return
	}

	guiData.ExData.Sets = db.SelectSetsByUser(appConfig.DBPath, user.ID)
	guiData.Config = appConfig
	guiData.Users = allUsers(c)
	guiData.CurrentUser = user

	guiData.GroupMap = make(map[string]string)

	for _, ex := range guiData.ExData.Sets {
		_, ok := guiData.GroupMap[ex.Name]
		if !ok {
			guiData.GroupMap[ex.Name] = ex.Name
		}
	}

	// Sort Sets by Date
	sort.Slice(guiData.ExData.Sets, func(i, j int) bool {
		return guiData.ExData.Sets[i].Date < guiData.ExData.Sets[j].Date
	})

	guiData.CardioSummary = computeCardioSummary(guiData.ExData.Sets, user)

	c.HTML(http.StatusOK, "header.html", guiData)
	c.HTML(http.StatusOK, "stats.html", guiData)
}

// computeCardioSummary tallies the user's cardio activity from the last 7
// days (inclusive of today). A "cardio" set is any set with a positive
// DurationSeconds — we infer kind from data rather than re-joining exercises.
func computeCardioSummary(sets []models.Set, user models.User) models.CardioSummary {
	cutoff := time.Now().AddDate(0, 0, -6).Format("2006-01-02") // includes today + previous 6 days
	unit := user.DistanceUnit
	if unit == "" {
		unit = "mi"
	}
	out := models.CardioSummary{Unit: unit}
	for _, s := range sets {
		if s.DurationSeconds <= 0 {
			continue
		}
		if s.Date < cutoff {
			continue
		}
		out.TotalMinutes += s.DurationSeconds / 60
		out.TotalDist = out.TotalDist.Add(s.DistanceValue)
		out.SessionCount++
	}
	return out
}
