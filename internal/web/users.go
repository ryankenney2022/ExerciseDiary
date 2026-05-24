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

func saveUserHandler(c *gin.Context) {
	name := c.PostForm("name")
	color := c.PostForm("color")

	if name == "" {
		c.Redirect(http.StatusFound, "/users/")
		return
	}
	if color == "" {
		color = "#0d6efd"
	}

	db.InsertUser(appConfig.DBPath, name, color)
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
