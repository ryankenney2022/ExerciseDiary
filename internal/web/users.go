package web

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

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

	// Height: accept either total inches OR feet + inches; the form posts
	// both and we sum (a ft value of N adds N*12 to inches). Total is
	// clamped to a sane range so a fat-fingered entry doesn't poison BMI.
	feet, _ := strconv.Atoi(c.PostForm("height_feet"))
	inches, _ := strconv.Atoi(c.PostForm("height_inches"))
	total := feet*12 + inches
	if total < 0 {
		total = 0
	}
	if total > 108 { // 9 ft — anything beyond this is data error
		total = 108
	}
	u.HeightInches = total

	if c.PostForm("bmi_enabled") == "1" {
		u.BMIEnabled = 1
	}
	if g, err := decimal.NewFromString(c.PostForm("goal_weight")); err == nil && g.Sign() >= 0 {
		u.GoalWeight = g
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
