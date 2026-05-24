package db

import (
	"fmt"
	"log"

	"github.com/aceberg/ExerciseDiary/internal/models"
)

// SelectPRTByUser - all PRT tests for a user, newest first.
func SelectPRTByUser(path string, userID int) []models.PRTTest {
	mu.Lock()
	dbx := connect(path)
	var tests []models.PRTTest
	err := dbx.Select(&tests, fmt.Sprintf(
		`SELECT * FROM prt_tests WHERE USER_ID = '%d' ORDER BY DATE DESC, ID DESC;`,
		userID,
	))
	mu.Unlock()
	if err != nil {
		log.Printf("WARN: SelectPRTByUser: %v", err)
	}
	return tests
}

// GetPRT - single test by ID; empty struct if not found.
func GetPRT(path string, id int) models.PRTTest {
	mu.Lock()
	dbx := connect(path)
	var t models.PRTTest
	_ = dbx.Get(&t, fmt.Sprintf(`SELECT * FROM prt_tests WHERE ID = '%d';`, id))
	mu.Unlock()
	return t
}

// GetLastPRTByUser - most recent test, empty struct if none.
func GetLastPRTByUser(path string, userID int) models.PRTTest {
	mu.Lock()
	dbx := connect(path)
	var t models.PRTTest
	_ = dbx.Get(&t, fmt.Sprintf(
		`SELECT * FROM prt_tests WHERE USER_ID = '%d' ORDER BY DATE DESC, ID DESC LIMIT 1;`,
		userID,
	))
	mu.Unlock()
	return t
}

// InsertPRT - insert a new test, returns the new ID.
func InsertPRT(path string, t models.PRTTest) int {
	stmt := fmt.Sprintf(`INSERT INTO prt_tests
		(USER_ID, DATE, NOTE, SEX_AT_TEST, AGE_AT_TEST, ALTITUDE_AT_TEST,
		 PUSHUPS, PLANK_SECONDS, RUN_SECONDS,
		 PUSHUP_SCORE, PLANK_SCORE, RUN_SCORE, OVERALL_SCORE, OVERALL_CATEGORY)
		VALUES ('%d','%s','%s','%s','%d','%s','%d','%d','%d','%d','%d','%d','%d','%s');`,
		t.UserID, t.Date, quoteStr(t.Note),
		quoteStr(t.SexAtTest), t.AgeAtTest, quoteStr(t.AltitudeAtTest),
		t.Pushups, t.PlankSeconds, t.RunSeconds,
		t.PushupScore, t.PlankScore, t.RunScore, t.OverallScore, quoteStr(t.OverallCategory),
	)
	exec(path, stmt)

	mu.Lock()
	dbx := connect(path)
	var id int
	_ = dbx.Get(&id, fmt.Sprintf(
		`SELECT ID FROM prt_tests WHERE USER_ID = '%d' AND DATE = '%s' ORDER BY ID DESC LIMIT 1;`,
		t.UserID, t.Date,
	))
	mu.Unlock()
	return id
}

// UpdatePRT - update an existing test's fields.
func UpdatePRT(path string, t models.PRTTest) {
	stmt := fmt.Sprintf(`UPDATE prt_tests SET
		DATE = '%s', NOTE = '%s',
		SEX_AT_TEST = '%s', AGE_AT_TEST = '%d', ALTITUDE_AT_TEST = '%s',
		PUSHUPS = '%d', PLANK_SECONDS = '%d', RUN_SECONDS = '%d',
		PUSHUP_SCORE = '%d', PLANK_SCORE = '%d', RUN_SCORE = '%d',
		OVERALL_SCORE = '%d', OVERALL_CATEGORY = '%s'
		WHERE ID = '%d';`,
		t.Date, quoteStr(t.Note),
		quoteStr(t.SexAtTest), t.AgeAtTest, quoteStr(t.AltitudeAtTest),
		t.Pushups, t.PlankSeconds, t.RunSeconds,
		t.PushupScore, t.PlankScore, t.RunScore, t.OverallScore, quoteStr(t.OverallCategory),
		t.ID,
	)
	exec(path, stmt)
}

// DeletePRT - remove a test by ID.
func DeletePRT(path string, id int) {
	exec(path, fmt.Sprintf(`DELETE FROM prt_tests WHERE ID = '%d';`, id))
}
