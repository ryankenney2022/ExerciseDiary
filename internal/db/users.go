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

// InsertUser - create a new user; no-op if name already exists
func InsertUser(path, name, color string) {
	if existing := GetUserByName(path, name); existing.ID != 0 {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	stmt := fmt.Sprintf(
		`INSERT INTO users (NAME, COLOR, CREATED_AT) VALUES ('%s','%s','%s');`,
		quoteStr(name), quoteStr(color), now,
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
