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

	exec(path, `UPDATE exercises SET VIDEO_URL = '' WHERE VIDEO_URL IS NULL;`)
	exec(path, `UPDATE sets SET NOTE = '' WHERE NOTE IS NULL;`)

	exec(path, `CREATE INDEX IF NOT EXISTS idx_workouts_user_date ON workouts(USER_ID, DATE);`)
	exec(path, `CREATE INDEX IF NOT EXISTS idx_sets_workout ON sets(WORKOUT_ID);`)
	exec(path, `CREATE INDEX IF NOT EXISTS idx_weight_user ON weight(USER_ID);`)
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
