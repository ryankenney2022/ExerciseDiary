package web

import (
	// "log"
	"net/http"
	"sort"

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

	c.HTML(http.StatusOK, "header.html", guiData)
	c.HTML(http.StatusOK, "stats.html", guiData)
}
