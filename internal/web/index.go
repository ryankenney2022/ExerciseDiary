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
