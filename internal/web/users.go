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
	name := c.PostForm("name")
	color := c.PostForm("color")
	sex := c.PostForm("sex")
	dob := c.PostForm("dob")
	altitude := c.PostForm("altitude")
	distanceUnit := c.PostForm("distance_unit")
	idStr := c.PostForm("id")

	if name == "" {
		c.Redirect(http.StatusFound, "/users/")
		return
	}
	if color == "" {
		color = "#0d6efd"
	}
	if altitude != "high" {
		altitude = "low"
	}
	if sex != "M" && sex != "F" {
		sex = ""
	}
	if distanceUnit != "km" {
		distanceUnit = "mi"
	}

	id, _ := strconv.Atoi(idStr)
	if id > 0 {
		db.UpdateUser(appConfig.DBPath, id, name, color, sex, dob, altitude, distanceUnit)
	} else {
		db.InsertUser(appConfig.DBPath, name, color, sex, dob, altitude, distanceUnit)
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
