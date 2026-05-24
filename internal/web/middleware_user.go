package web

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/aceberg/ExerciseDiary/internal/db"
	"github.com/aceberg/ExerciseDiary/internal/models"
)

const userCookie = "ed_user"

// userMiddleware resolves the active user identity from the ed_user cookie
// and stashes it in the gin context as "user". If the cookie is missing or
// invalid, the first registered user is used. If no users exist yet, leaves
// an empty user in context and lets the handler decide (most pages handle
// this by redirecting to /users).
func userMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		users := db.SelectUsers(appConfig.DBPath)

		var current models.User
		if cookie, err := c.Cookie(userCookie); err == nil {
			if id, convErr := strconv.Atoi(cookie); convErr == nil {
				for _, u := range users {
					if u.ID == id {
						current = u
						break
					}
				}
			}
		}

		if current.ID == 0 && len(users) > 0 {
			current = users[0]
		}

		c.Set("user", current)
		c.Set("users", users)
		c.Next()
	}
}

func currentUser(c *gin.Context) models.User {
	if v, ok := c.Get("user"); ok {
		if u, ok := v.(models.User); ok {
			return u
		}
	}
	return models.User{}
}

func allUsers(c *gin.Context) []models.User {
	if v, ok := c.Get("users"); ok {
		if us, ok := v.([]models.User); ok {
			return us
		}
	}
	return nil
}

// switchUserHandler - POST /user/switch with form field "id". Sets cookie, redirects back.
func switchUserHandler(c *gin.Context) {
	idStr := c.PostForm("id")
	id, err := strconv.Atoi(idStr)
	if err == nil && id > 0 {
		c.SetCookie(userCookie, idStr, 60*60*24*365, "/", "", false, false)
	}

	back := "/"
	if ref := c.Request.Header.Get("Referer"); ref != "" {
		back = ref
	}
	c.Redirect(http.StatusFound, back)
}
