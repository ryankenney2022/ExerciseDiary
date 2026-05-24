package db

import (
	"fmt"
	"log"

	"github.com/aceberg/ExerciseDiary/internal/models"
)

// SelectWorkoutsByUser - all workouts for a user, newest first
func SelectWorkoutsByUser(path string, userID int) []models.Workout {
	mu.Lock()
	dbx := connect(path)
	var ws []models.Workout
	err := dbx.Select(&ws, fmt.Sprintf(
		`SELECT * FROM workouts WHERE USER_ID = '%d' ORDER BY DATE DESC, ID DESC;`,
		userID,
	))
	mu.Unlock()
	if err != nil {
		log.Printf("WARN: SelectWorkoutsByUser: %v", err)
	}
	return ws
}

// GetWorkoutByUserDate - the workout for (user, date). Empty struct if none.
// When a user has multiple workouts on the same date (possible if name differs),
// returns the most recently created one.
func GetWorkoutByUserDate(path string, userID int, date string) models.Workout {
	mu.Lock()
	dbx := connect(path)
	var w models.Workout
	_ = dbx.Get(&w, fmt.Sprintf(
		`SELECT * FROM workouts WHERE USER_ID = '%d' AND DATE = '%s' ORDER BY ID DESC LIMIT 1;`,
		userID, date,
	))
	mu.Unlock()
	return w
}

// GetOrCreateTodayWorkout - returns the workout for (user, date), creating an
// unnamed empty one if it doesn't exist yet
func GetOrCreateTodayWorkout(path string, userID int, date string) models.Workout {
	w := GetWorkoutByUserDate(path, userID, date)
	if w.ID != 0 {
		return w
	}
	stmt := fmt.Sprintf(
		`INSERT INTO workouts (USER_ID, DATE, NAME, NOTE) VALUES ('%d','%s','','');`,
		userID, date,
	)
	exec(path, stmt)
	return GetWorkoutByUserDate(path, userID, date)
}

// UpdateWorkoutMeta - set name and note on an existing workout
func UpdateWorkoutMeta(path string, id int, name, note string) {
	stmt := fmt.Sprintf(
		`UPDATE workouts SET NAME = '%s', NOTE = '%s' WHERE ID = '%d';`,
		quoteStr(name), quoteStr(note), id,
	)
	exec(path, stmt)
}

// SelectSetsByUser - all sets owned by a user (joined via workouts)
func SelectSetsByUser(path string, userID int) []models.Set {
	mu.Lock()
	dbx := connect(path)
	var sets []models.Set
	err := dbx.Select(&sets, fmt.Sprintf(
		`SELECT sets.* FROM sets
		 JOIN workouts ON sets.WORKOUT_ID = workouts.ID
		 WHERE workouts.USER_ID = '%d'
		 ORDER BY sets.ID ASC;`,
		userID,
	))
	mu.Unlock()
	if err != nil {
		log.Printf("WARN: SelectSetsByUser: %v", err)
	}
	return sets
}

// SelectWeightByUser - all weight rows for a user
func SelectWeightByUser(path string, userID int) []models.BodyWeight {
	mu.Lock()
	dbx := connect(path)
	var w []models.BodyWeight
	err := dbx.Select(&w, fmt.Sprintf(
		`SELECT * FROM weight WHERE USER_ID = '%d' ORDER BY ID ASC;`,
		userID,
	))
	mu.Unlock()
	if err != nil {
		log.Printf("WARN: SelectWeightByUser: %v", err)
	}
	return w
}

// BulkDeleteSetsByUserDate - replace-flow helper, scoped to a user
func BulkDeleteSetsByUserDate(path string, userID int, date string) {
	stmt := fmt.Sprintf(
		`DELETE FROM sets WHERE DATE = '%s' AND WORKOUT_ID IN
		 (SELECT ID FROM workouts WHERE USER_ID = '%d');`,
		date, userID,
	)
	exec(path, stmt)
}
