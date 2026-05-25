package db

import (
	"fmt"
	"log"
	"time"

	"github.com/aceberg/ExerciseDiary/internal/models"
)

// SelectUsers - all users ordered by id (creation order)
func SelectUsers(path string) []models.User {
	mu.Lock()
	dbx := connect(path)
	var users []models.User
	err := dbx.Select(&users, `SELECT * FROM users ORDER BY ID ASC;`)
	mu.Unlock()
	if err != nil {
		log.Printf("WARN: SelectUsers: %v", err)
	}
	return users
}

// GetUserByID - single user, empty struct if not found
func GetUserByID(path string, id int) models.User {
	mu.Lock()
	dbx := connect(path)
	var u models.User
	_ = dbx.Get(&u, fmt.Sprintf(`SELECT * FROM users WHERE ID = '%d';`, id))
	mu.Unlock()
	return u
}

// GetUserByName - single user by name, empty struct if not found
func GetUserByName(path, name string) models.User {
	mu.Lock()
	dbx := connect(path)
	var u models.User
	_ = dbx.Get(&u, fmt.Sprintf(`SELECT * FROM users WHERE NAME = '%s';`, quoteStr(name)))
	mu.Unlock()
	return u
}

// InsertUser - create a new user; no-op if name already exists.
// Sex/DOB/Altitude/DistanceUnit can be empty and edited later.
func InsertUser(path string, u models.User) {
	if existing := GetUserByName(path, u.Name); existing.ID != 0 {
		return
	}
	if u.Altitude == "" {
		u.Altitude = "low"
	}
	if u.DistanceUnit == "" {
		u.DistanceUnit = "mi"
	}
	if u.RestTimerSeconds <= 0 {
		u.RestTimerSeconds = 90
	}
	now := time.Now().UTC().Format(time.RFC3339)
	stmt := fmt.Sprintf(
		`INSERT INTO users (NAME, COLOR, CREATED_AT, SEX, DOB, ALTITUDE, DISTANCE_UNIT, REST_TIMER_ON, REST_TIMER_SECONDS) VALUES ('%s','%s','%s','%s','%s','%s','%s','%d','%d');`,
		quoteStr(u.Name), quoteStr(u.Color), now,
		quoteStr(u.Sex), quoteStr(u.DOB), quoteStr(u.Altitude), quoteStr(u.DistanceUnit),
		u.RestTimerOn, u.RestTimerSeconds,
	)
	exec(path, stmt)
}

// UpdateUser - update an existing user's editable fields.
func UpdateUser(path string, u models.User) {
	if u.Altitude == "" {
		u.Altitude = "low"
	}
	if u.DistanceUnit == "" {
		u.DistanceUnit = "mi"
	}
	if u.RestTimerSeconds <= 0 {
		u.RestTimerSeconds = 90
	}
	stmt := fmt.Sprintf(
		`UPDATE users SET NAME = '%s', COLOR = '%s', SEX = '%s', DOB = '%s', ALTITUDE = '%s', DISTANCE_UNIT = '%s', REST_TIMER_ON = '%d', REST_TIMER_SECONDS = '%d' WHERE ID = '%d';`,
		quoteStr(u.Name), quoteStr(u.Color), quoteStr(u.Sex), quoteStr(u.DOB),
		quoteStr(u.Altitude), quoteStr(u.DistanceUnit),
		u.RestTimerOn, u.RestTimerSeconds,
		u.ID,
	)
	exec(path, stmt)
}

// DeleteUser - remove a user only if they have no sets, workouts, or weight rows
func DeleteUser(path string, id int) error {
	if countRowsWhere(path, "sets", fmt.Sprintf("WORKOUT_ID IN (SELECT ID FROM workouts WHERE USER_ID = '%d')", id)) > 0 {
		return fmt.Errorf("user has sets")
	}
	if countRowsWhere(path, "weight", fmt.Sprintf("USER_ID = '%d'", id)) > 0 {
		return fmt.Errorf("user has weight history")
	}
	exec(path, fmt.Sprintf(`DELETE FROM workouts WHERE USER_ID = '%d';`, id))
	exec(path, fmt.Sprintf(`DELETE FROM users WHERE ID = '%d';`, id))
	return nil
}

func countRowsWhere(path, table, where string) int {
	mu.Lock()
	dbx := connect(path)
	var n int
	err := dbx.Get(&n, fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE %s;`, table, where))
	mu.Unlock()
	if err != nil {
		return 0
	}
	return n
}
