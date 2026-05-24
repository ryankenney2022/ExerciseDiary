package web

import (
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"github.com/aceberg/ExerciseDiary/internal/db"
	"github.com/aceberg/ExerciseDiary/internal/models"
)

func addWeightHandler(c *gin.Context) {
	user := currentUser(c)
	if user.ID == 0 {
		c.Redirect(http.StatusFound, "/users/")
		return
	}

	var w models.BodyWeight

	w.UserID = user.ID
	w.Date = c.PostForm("date")
	weightStr := c.PostForm("weight")

	w.Weight, _ = decimal.NewFromString(weightStr)

	db.InsertW(appConfig.DBPath, w)

	back := "/"
	if ref := c.Request.Header.Get("Referer"); ref != "" {
		back = ref
	}
	c.Redirect(http.StatusFound, back)
}

func weightHandler(c *gin.Context) {
	var guiData models.GuiData

	user := currentUser(c)
	if user.ID == 0 {
		c.Redirect(http.StatusFound, "/users/")
		return
	}

	idStr, ok := c.GetQuery("del")
	if ok {
		id, _ := strconv.Atoi(idStr)
		db.DeleteW(appConfig.DBPath, id)
	}
	exData.Weight = db.SelectWeightByUser(appConfig.DBPath, user.ID)

	guiData.Config = appConfig
	guiData.ExData = exData
	guiData.Users = allUsers(c)
	guiData.CurrentUser = user

	// Sort weight by Date
	sort.Slice(guiData.ExData.Weight, func(i, j int) bool {
		return guiData.ExData.Weight[i].Date < guiData.ExData.Weight[j].Date
	})

	c.HTML(http.StatusOK, "header.html", guiData)
	c.HTML(http.StatusOK, "weight.html", guiData)
}
