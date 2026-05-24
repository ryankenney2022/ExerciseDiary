package db

import (
	"fmt"
	"log"
	"os"
	"time"
)

// MigrateMultiUser runs once. If the users table is empty and any historical
// sets/weight rows exist, it claims them for the user named in the
// MIGRATION_DEFAULT_USER env var. Subsequent boots see a non-empty users table
// and skip out immediately.
func MigrateMultiUser(path string) {
	if userCount(path) > 0 {
		return
	}

	if countRows(path, "sets") == 0 && countRows(path, "weight") == 0 {
		return
	}

	defaultName := os.Getenv("MIGRATION_DEFAULT_USER")
	if defaultName == "" {
		log.Println("=================================================================")
		log.Println("FATAL: existing workout history found but no users defined.")
		log.Println("Set MIGRATION_DEFAULT_USER (e.g. 'Akane') to claim the existing")
		log.Println("sets and weight history for that user. The variable is only read")
		log.Println("once — after the migration succeeds you can remove it.")
		log.Println("=================================================================")
		os.Exit(1)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	insertUser := fmt.Sprintf(
		`INSERT INTO users (NAME, COLOR, CREATED_AT) VALUES ('%s','%s','%s');`,
		quoteStr(defaultName), "#0d6efd", now,
	)
	exec(path, insertUser)

	userID := userIDByName(path, defaultName)
	if userID == 0 {
		log.Fatalf("FATAL: migration could not resolve user id for %q after insert", defaultName)
	}

	dates := distinctSetDates(path)
	log.Printf("INFO: multi-user migration: assigning %d set-dates and weight history to user %q (id=%d)",
		len(dates), defaultName, userID)

	for _, date := range dates {
		insertW := fmt.Sprintf(
			`INSERT INTO workouts (USER_ID, DATE, NAME, NOTE) VALUES ('%d','%s','','');`,
			userID, date,
		)
		exec(path, insertW)

		workoutID := workoutIDByUserDate(path, userID, date)
		if workoutID == 0 {
			log.Printf("WARN: failed to resolve workout id for date %s, skipping", date)
			continue
		}

		updateSets := fmt.Sprintf(
			`UPDATE sets SET WORKOUT_ID = '%d' WHERE DATE = '%s' AND (WORKOUT_ID IS NULL OR WORKOUT_ID = 0);`,
			workoutID, date,
		)
		exec(path, updateSets)
	}

	updateWeight := fmt.Sprintf(
		`UPDATE weight SET USER_ID = '%d' WHERE USER_ID IS NULL OR USER_ID = 0;`,
		userID,
	)
	exec(path, updateWeight)

	log.Println("INFO: multi-user migration complete")
}

func userCount(path string) int {
	return countRows(path, "users")
}

func countRows(path, table string) int {
	mu.Lock()
	dbx := connect(path)
	var n int
	err := dbx.Get(&n, fmt.Sprintf(`SELECT COUNT(*) FROM %s;`, table))
	mu.Unlock()
	if err != nil {
		log.Printf("WARN: count(%s) failed: %v", table, err)
		return 0
	}
	return n
}

func userIDByName(path, name string) int {
	mu.Lock()
	dbx := connect(path)
	var id int
	err := dbx.Get(&id, fmt.Sprintf(`SELECT ID FROM users WHERE NAME = '%s';`, quoteStr(name)))
	mu.Unlock()
	if err != nil {
		return 0
	}
	return id
}

func workoutIDByUserDate(path string, userID int, date string) int {
	mu.Lock()
	dbx := connect(path)
	var id int
	err := dbx.Get(&id, fmt.Sprintf(
		`SELECT ID FROM workouts WHERE USER_ID = '%d' AND DATE = '%s' ORDER BY ID DESC LIMIT 1;`,
		userID, date,
	))
	mu.Unlock()
	if err != nil {
		return 0
	}
	return id
}

func distinctSetDates(path string) []string {
	mu.Lock()
	dbx := connect(path)
	var dates []string
	err := dbx.Select(&dates, `SELECT DISTINCT DATE FROM sets ORDER BY DATE ASC;`)
	mu.Unlock()
	if err != nil {
		log.Printf("WARN: distinct set dates failed: %v", err)
		return nil
	}
	return dates
}
