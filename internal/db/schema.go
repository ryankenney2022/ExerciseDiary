package db

import (
	"fmt"
	"log"
)

// EnsureSchema adds columns that were introduced after the original schema and
// then creates the indexes (which depend on those columns existing). Safe to
// run on every boot — checks PRAGMA table_info before each ALTER and uses
// CREATE INDEX IF NOT EXISTS.
//
// TEXT columns use DEFAULT '' so that pre-existing rows are backfilled with an
// empty string (SQLite applies the default to existing rows when the column is
// added). The trailing UPDATEs are a safety net for DBs that were patched by
// an earlier build before the DEFAULT clause was added.
func EnsureSchema(path string) {
	addColumnIfMissing(path, "exercises", "VIDEO_URL", "TEXT DEFAULT ''")
	addColumnIfMissing(path, "sets", "WORKOUT_ID", "INTEGER")
	addColumnIfMissing(path, "sets", "NOTE", "TEXT DEFAULT ''")
	addColumnIfMissing(path, "weight", "USER_ID", "INTEGER")

	addColumnIfMissing(path, "users", "SEX", "TEXT DEFAULT ''")
	addColumnIfMissing(path, "users", "DOB", "TEXT DEFAULT ''")
	addColumnIfMissing(path, "users", "ALTITUDE", "TEXT DEFAULT 'low'")
	addColumnIfMissing(path, "users", "DISTANCE_UNIT", "TEXT DEFAULT 'mi'")

	addColumnIfMissing(path, "exercises", "KIND", "TEXT DEFAULT 'strength'")

	addColumnIfMissing(path, "sets", "DURATION_SECONDS", "INTEGER DEFAULT 0")
	addColumnIfMissing(path, "sets", "DISTANCE_VALUE", "REAL DEFAULT 0")
	addColumnIfMissing(path, "sets", "AVG_HR", "INTEGER DEFAULT 0")
	addColumnIfMissing(path, "sets", "MAX_HR", "INTEGER DEFAULT 0")
	addColumnIfMissing(path, "sets", "CALORIES", "INTEGER DEFAULT 0")
	addColumnIfMissing(path, "sets", "EQUIPMENT", "TEXT DEFAULT ''")

	exec(path, `UPDATE exercises SET VIDEO_URL = '' WHERE VIDEO_URL IS NULL;`)
	exec(path, `UPDATE sets SET NOTE = '' WHERE NOTE IS NULL;`)
	exec(path, `UPDATE users SET SEX = '' WHERE SEX IS NULL;`)
	exec(path, `UPDATE users SET DOB = '' WHERE DOB IS NULL;`)
	exec(path, `UPDATE users SET ALTITUDE = 'low' WHERE ALTITUDE IS NULL OR ALTITUDE = '';`)
	exec(path, `UPDATE users SET DISTANCE_UNIT = 'mi' WHERE DISTANCE_UNIT IS NULL OR DISTANCE_UNIT = '';`)
	exec(path, `UPDATE exercises SET KIND = 'strength' WHERE KIND IS NULL OR KIND = '';`)
	exec(path, `UPDATE sets SET EQUIPMENT = '' WHERE EQUIPMENT IS NULL;`)

	exec(path, `CREATE TABLE IF NOT EXISTS prt_tests (
		"ID"                INTEGER PRIMARY KEY,
		"USER_ID"           INTEGER NOT NULL,
		"DATE"              TEXT NOT NULL,
		"NOTE"              TEXT DEFAULT '',
		"SEX_AT_TEST"       TEXT DEFAULT '',
		"AGE_AT_TEST"       INTEGER DEFAULT 0,
		"ALTITUDE_AT_TEST"  TEXT DEFAULT 'low',
		"PUSHUPS"           INTEGER DEFAULT 0,
		"PLANK_SECONDS"     INTEGER DEFAULT 0,
		"RUN_SECONDS"       INTEGER DEFAULT 0,
		"PUSHUP_SCORE"      INTEGER DEFAULT 0,
		"PLANK_SCORE"       INTEGER DEFAULT 0,
		"RUN_SCORE"         INTEGER DEFAULT 0,
		"OVERALL_SCORE"     INTEGER DEFAULT 0,
		"OVERALL_CATEGORY"  TEXT DEFAULT ''
	);`)

	exec(path, `CREATE INDEX IF NOT EXISTS idx_workouts_user_date ON workouts(USER_ID, DATE);`)
	exec(path, `CREATE INDEX IF NOT EXISTS idx_sets_workout ON sets(WORKOUT_ID);`)
	exec(path, `CREATE INDEX IF NOT EXISTS idx_weight_user ON weight(USER_ID);`)
	exec(path, `CREATE INDEX IF NOT EXISTS idx_prt_user_date ON prt_tests(USER_ID, DATE DESC);`)
}

func addColumnIfMissing(path, table, column, colType string) {
	if hasColumn(path, table, column) {
		return
	}

	stmt := fmt.Sprintf(`ALTER TABLE %s ADD COLUMN "%s" %s;`, table, column, colType)
	log.Printf("INFO: schema upgrade: %s", stmt)
	exec(path, stmt)
}

func hasColumn(path, table, column string) bool {
	mu.Lock()
	dbx := connect(path)
	rows, err := dbx.Queryx(fmt.Sprintf(`PRAGMA table_info(%s);`, table))
	mu.Unlock()

	if err != nil {
		log.Printf("WARN: PRAGMA table_info(%s) failed: %v", table, err)
		return false
	}
	defer rows.Close()

	for rows.Next() {
		row := map[string]interface{}{}
		if err := rows.MapScan(row); err != nil {
			continue
		}
		if name, ok := row["name"].(string); ok && name == column {
			return true
		}
	}
	return false
}
