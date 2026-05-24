package web

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"github.com/aceberg/ExerciseDiary/internal/db"
	"github.com/aceberg/ExerciseDiary/internal/models"
)

func setHandler(c *gin.Context) {
	user := currentUser(c)
	if user.ID == 0 {
		c.Redirect(http.StatusFound, "/users/")
		return
	}

	_ = c.PostFormMap("sets")
	formMap := c.Request.PostForm

	dates := formMap["date"]
	if len(dates) == 0 || dates[0] == "" {
		c.Redirect(http.StatusFound, "/")
		return
	}
	date := dates[0]

	workout := db.GetOrCreateTodayWorkout(appConfig.DBPath, user.ID, date)

	workoutName := firstOr(formMap["workout_name"], "")
	workoutNote := firstOr(formMap["workout_note"], "")
	db.UpdateWorkoutMeta(appConfig.DBPath, workout.ID, workoutName, workoutNote)

	db.BulkDeleteSetsByUserDate(appConfig.DBPath, user.ID, date)

	names := formMap["name"]
	weights := formMap["weight"]
	reps := formMap["reps"]
	notes := formMap["note"]

	var oneSet models.Set
	var formData []models.Set
	for i := 0; i < len(names); i++ {
		oneSet = models.Set{}
		oneSet.WorkoutID = workout.ID
		oneSet.Date = date
		oneSet.Name = names[i]
		if i < len(weights) {
			oneSet.Weight, _ = decimal.NewFromString(weights[i])
		}
		if i < len(reps) {
			oneSet.Reps, _ = strconv.Atoi(reps[i])
		}
		if i < len(notes) {
			oneSet.Note = notes[i]
		}
		formData = append(formData, oneSet)
	}

	db.BulkAddSets(appConfig.DBPath, formData)
	exData.Sets = db.SelectSetsByUser(appConfig.DBPath, user.ID)

	c.Redirect(http.StatusFound, "/")
}

func firstOr(vals []string, fallback string) string {
	if len(vals) == 0 {
		return fallback
	}
	return vals[0]
}
