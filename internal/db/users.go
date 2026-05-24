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
// Sex/DOB/Altitude can be empty and edited later.
func InsertUser(path, name, color, sex, dob, altitude string) {
	if existing := GetUserByName(path, name); existing.ID != 0 {
		return
	}
	if altitude == "" {
		altitude = "low"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	stmt := fmt.Sprintf(
		`INSERT INTO users (NAME, COLOR, CREATED_AT, SEX, DOB, ALTITUDE) VALUES ('%s','%s','%s','%s','%s','%s');`,
		quoteStr(name), quoteStr(color), now,
		quoteStr(sex), quoteStr(dob), quoteStr(altitude),
	)
	exec(path, stmt)
}

// UpdateUser - update an existing user's editable fields.
func UpdateUser(path string, id int, name, color, sex, dob, altitude string) {
	if altitude == "" {
		altitude = "low"
	}
	stmt := fmt.Sprintf(
		`UPDATE users SET NAME = '%s', COLOR = '%s', SEX = '%s', DOB = '%s', ALTITUDE = '%s' WHERE ID = '%d';`,
		quoteStr(name), quoteStr(color), quoteStr(sex), quoteStr(dob), quoteStr(altitude), id,
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
