package db

import (
	"fmt"

	"github.com/aceberg/ExerciseDiary/internal/models"
)

// Create - create table if not exists. New tables get the full multi-user schema;
// existing DBs are patched additively by EnsureSchema().
func Create(path string) {

	sqlStatement := `CREATE TABLE IF NOT EXISTS exercises (
		"ID"		INTEGER PRIMARY KEY,
		"GR"		TEXT,
		"PLACE"		TEXT,
		"NAME"		TEXT,
		"DESCR"		TEXT,
		"IMAGE"		TEXT,
		"VIDEO_URL"	TEXT,
		"COLOR"		TEXT,
		"WEIGHT"	INTEGER,
		"REPS"		INTEGER
	);`
	exec(path, sqlStatement)

	sqlStatement = `CREATE TABLE IF NOT EXISTS sets (
		"ID"		INTEGER PRIMARY KEY,
		"WORKOUT_ID" INTEGER,
		"DATE"		TEXT,
		"NAME"		TEXT,
		"COLOR"		TEXT,
		"WEIGHT"	INTEGER,
		"REPS"		INTEGER,
		"NOTE"		TEXT
	);`
	exec(path, sqlStatement)

	sqlStatement = `CREATE TABLE IF NOT EXISTS weight (
		"ID"		INTEGER PRIMARY KEY,
		"USER_ID"	INTEGER,
		"DATE"		TEXT,
		"WEIGHT"    INTEGER
	);`
	exec(path, sqlStatement)

	sqlStatement = `CREATE TABLE IF NOT EXISTS users (
		"ID"			INTEGER PRIMARY KEY,
		"NAME"			TEXT UNIQUE NOT NULL,
		"COLOR"			TEXT,
		"CREATED_AT"	TEXT
	);`
	exec(path, sqlStatement)

	sqlStatement = `CREATE TABLE IF NOT EXISTS workouts (
		"ID"		INTEGER PRIMARY KEY,
		"USER_ID"	INTEGER NOT NULL,
		"DATE"		TEXT NOT NULL,
		"NAME"		TEXT,
		"NOTE"		TEXT
	);`
	exec(path, sqlStatement)
}

// InsertEx - insert one exercise into DB
func InsertEx(path string, ex models.Exercise) {

	sqlStatement := `INSERT INTO exercises (GR, PLACE, NAME, DESCR, IMAGE, VIDEO_URL, COLOR, WEIGHT, REPS)
	VALUES ('%s','%s','%s','%s','%s','%s','%s','%v','%d');`

	ex.Group = quoteStr(ex.Group)
	ex.Name = quoteStr(ex.Name)
	ex.Descr = quoteStr(ex.Descr)
	ex.VideoURL = quoteStr(ex.VideoURL)

	sqlStatement = fmt.Sprintf(sqlStatement, ex.Group, ex.Place, ex.Name, ex.Descr, ex.Image, ex.VideoURL, ex.Color, ex.Weight, ex.Reps)

	exec(path, sqlStatement)
}

// InsertSet - insert one set into DB
func InsertSet(path string, ex models.Set) {

	sqlStatement := `INSERT INTO sets (WORKOUT_ID, DATE, NAME, COLOR, WEIGHT, REPS, NOTE)
	VALUES ('%d','%s','%s','%s','%v','%d','%s');`

	ex.Name = quoteStr(ex.Name)
	ex.Note = quoteStr(ex.Note)

	sqlStatement = fmt.Sprintf(sqlStatement, ex.WorkoutID, ex.Date, ex.Name, ex.Color, ex.Weight, ex.Reps, ex.Note)

	exec(path, sqlStatement)
}

// InsertW - insert weight
func InsertW(path string, ex models.BodyWeight) {

	sqlStatement := `INSERT INTO weight (USER_ID, DATE, WEIGHT)
	VALUES ('%d','%s','%v');`

	sqlStatement = fmt.Sprintf(sqlStatement, ex.UserID, ex.Date, ex.Weight)

	exec(path, sqlStatement)
}

// DeleteEx - delete one exercise
func DeleteEx(path string, id int) {

	sqlStatement := `DELETE FROM exercises WHERE ID='%d';`

	sqlStatement = fmt.Sprintf(sqlStatement, id)

	exec(path, sqlStatement)
}

// DeleteSet - delete one set
func DeleteSet(path string, id int) {

	sqlStatement := `DELETE FROM sets WHERE ID='%d';`

	sqlStatement = fmt.Sprintf(sqlStatement, id)

	exec(path, sqlStatement)
}

// DeleteW - delete weight
func DeleteW(path string, id int) {

	sqlStatement := `DELETE FROM weight WHERE ID='%d';`

	sqlStatement = fmt.Sprintf(sqlStatement, id)

	exec(path, sqlStatement)
}

// ClearEx - delete all exercises from table
func ClearEx(path string) {
	sqlStatement := `DELETE FROM exercises;`
	exec(path, sqlStatement)
}

// ClearSet - delete all sets from table
func ClearSet(path string) {
	sqlStatement := `DELETE FROM sets;`
	exec(path, sqlStatement)
}
