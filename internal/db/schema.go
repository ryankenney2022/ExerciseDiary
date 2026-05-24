package db

import (
	"fmt"
	"log"
)

// EnsureSchema adds columns that were introduced after the original schema.
// Safe to run on every boot — checks PRAGMA table_info before each ALTER.
// New databases get the full schema from Create(); this only patches old ones.
func EnsureSchema(path string) {
	addColumnIfMissing(path, "exercises", "VIDEO_URL", "TEXT")
	addColumnIfMissing(path, "sets", "WORKOUT_ID", "INTEGER")
	addColumnIfMissing(path, "sets", "NOTE", "TEXT")
	addColumnIfMissing(path, "weight", "USER_ID", "INTEGER")
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
