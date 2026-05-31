package web

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/aceberg/ExerciseDiary/internal/db"
	"github.com/aceberg/ExerciseDiary/internal/models"
)

// templatesHandler renders the shared plan library.
func templatesHandler(c *gin.Context) {
	user := currentUser(c)
	if user.ID == 0 {
		c.Redirect(http.StatusFound, "/users/")
		return
	}
	var guiData models.GuiData
	guiData.Config = appConfig
	guiData.Users = allUsers(c)
	guiData.CurrentUser = user
	guiData.Templates = db.SelectTemplates(appConfig.DBPath)
	guiData.TemplateCounts = db.SelectTemplateCounts(appConfig.DBPath)

	c.HTML(http.StatusOK, "header.html", guiData)
	c.HTML(http.StatusOK, "templates_page.html", guiData)
}
