package web

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/aceberg/ExerciseDiary/internal/db"
	"github.com/aceberg/ExerciseDiary/internal/models"
)

func usersHandler(c *gin.Context) {
	var guiData models.GuiData

	guiData.Config = appConfig
	guiData.Users = allUsers(c)
	guiData.CurrentUser = currentUser(c)

	c.HTML(http.StatusOK, "header.html", guiData)
	c.HTML(http.StatusOK, "users.html", guiData)
}

// saveUserHandler handles both "add" (when id is empty/0) and "edit" (when id > 0).
func saveUserHandler(c *gin.Context) {
	u := models.User{
		Name:         c.PostForm("name"),
		Color:        c.PostForm("color"),
		Sex:          c.PostForm("sex"),
		DOB:          c.PostForm("dob"),
		Altitude:     c.PostForm("altitude"),
		DistanceUnit: c.PostForm("distance_unit"),
	}

	if u.Name == "" {
		c.Redirect(http.StatusFound, "/users/")
		return
	}
	if u.Color == "" {
		u.Color = "#0d6efd"
	}
	if u.Altitude != "high" {
		u.Altitude = "low"
	}
	if u.Sex != "M" && u.Sex != "F" {
		u.Sex = ""
	}
	if u.DistanceUnit != "km" {
		u.DistanceUnit = "mi"
	}

	// Rest-timer fields. Checkbox sends "1" when checked, nothing when not.
	if c.PostForm("rest_timer_on") == "1" {
		u.RestTimerOn = 1
	}
	if s, err := strconv.Atoi(c.PostForm("rest_timer_seconds")); err == nil && s > 0 {
		u.RestTimerSeconds = s
	}

	idStr := c.PostForm("id")
	id, _ := strconv.Atoi(idStr)
	if id > 0 {
		u.ID = id
		db.UpdateUser(appConfig.DBPath, u)
	} else {
		db.InsertUser(appConfig.DBPath, u)
	}
	c.Redirect(http.StatusFound, "/users/")
}

func deleteUserHandler(c *gin.Context) {
	idStr := c.PostForm("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.Redirect(http.StatusFound, "/users/")
		return
	}

	_ = db.DeleteUser(appConfig.DBPath, id)
	c.Redirect(http.StatusFound, "/users/")
}
