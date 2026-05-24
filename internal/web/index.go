package web

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/aceberg/ExerciseDiary/internal/db"
	"github.com/aceberg/ExerciseDiary/internal/models"
	"github.com/aceberg/ExerciseDiary/internal/prt"
)

func indexHandler(c *gin.Context) {
	var guiData models.GuiData

	user := currentUser(c)
	if user.ID == 0 {
		c.Redirect(http.StatusFound, "/users/")
		return
	}

	exData.Exs = db.SelectEx(appConfig.DBPath)
	exData.Sets = db.SelectSetsByUser(appConfig.DBPath, user.ID)
	exData.Weight = db.SelectWeightByUser(appConfig.DBPath, user.ID)

	guiData.Config = appConfig
	guiData.ExData = exData
	guiData.GroupMap = createGroupMap()
	guiData.HeatMap = generateHeatMap()
	guiData.Users = allUsers(c)
	guiData.CurrentUser = user
	guiData.TodayWorkout = db.GetWorkoutByUserDate(appConfig.DBPath, user.ID, time.Now().Format("2006-01-02"))
	guiData.LastPRT = db.GetLastPRTByUser(appConfig.DBPath, user.ID)
	if user.Sex != "" && user.DOB != "" {
		age := prt.AgeOnDate(user.DOB, time.Now().Format("2006-01-02"))
		guiData.ScoreSheet = prt.SheetFor(user.Sex, age, user.Altitude)
	}

	guiData.LastDone = lastDoneByName(exData.Sets)

	// Sort exercises by Place
	sort.Slice(guiData.ExData.Exs, func(i, j int) bool {
		return guiData.ExData.Exs[i].Place < guiData.ExData.Exs[j].Place
	})

	// Sort weight by Date
	sort.Slice(guiData.ExData.Weight, func(i, j int) bool {
		return guiData.ExData.Weight[i].Date < guiData.ExData.Weight[j].Date
	})

	c.HTML(http.StatusOK, "header.html", guiData)
	c.HTML(http.StatusOK, "index.html", guiData)
}

// lastDoneByName scans user-scoped sets and returns the most recent DATE for
// each exercise name. Empty map if there are no sets. Used to label exercises
// on the home accordion with when the user last did them.
func lastDoneByName(sets []models.Set) map[string]string {
	out := map[string]string{}
	for _, s := range sets {
		if s.Date == "" || s.Name == "" {
			continue
		}
		if cur, ok := out[s.Name]; !ok || s.Date > cur {
			out[s.Name] = s.Date
		}
	}
	return out
}

func createGroupMap() map[string]string {
	i := 0
	grMap := make(map[string]string)

	for _, ex := range exData.Exs {

		_, ok := grMap[ex.Group]
		if !ok {
			grMap[ex.Group] = "grID" + fmt.Sprintf("%d", i)
			i = i + 1
		}
	}
	return grMap
}
