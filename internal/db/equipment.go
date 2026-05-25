package db

import (
	"fmt"
	"log"

	"github.com/shopspring/decimal"

	"github.com/aceberg/ExerciseDiary/internal/models"
)

// GetEquipment returns the equipment row for a user. If none exists, returns
// a zero-value struct with the default barbell weight (45 lb) so callers can
// still render a sensible form. The caller decides whether to persist it.
func GetEquipment(path string, userID int) models.Equipment {
	mu.Lock()
	dbx := connect(path)
	var e models.Equipment
	err := dbx.Get(&e, fmt.Sprintf(
		`SELECT * FROM equipment WHERE USER_ID = '%d' LIMIT 1;`, userID,
	))
	mu.Unlock()
	if err != nil {
		// No row yet — return defaults; not an error worth logging.
		return models.Equipment{
			UserID:        userID,
			BarbellWeight: decimal.NewFromInt(45),
			Unit:          "lb",
		}
	}
	if e.Unit == "" {
		e.Unit = "lb"
	}
	return e
}

// UpsertEquipment writes the user's barbell + unit. Uses INSERT OR REPLACE
// since USER_ID is the primary key.
func UpsertEquipment(path string, e models.Equipment) {
	stmt := fmt.Sprintf(
		`INSERT OR REPLACE INTO equipment (USER_ID, BARBELL_WEIGHT, UNIT) VALUES ('%d', %s, '%s');`,
		e.UserID, e.BarbellWeight.String(), quoteStr(e.Unit),
	)
	exec(path, stmt)
}

// SelectPlatesByUser returns a user's plates ordered by weight descending so
// the greedy calculator can iterate in the natural order.
func SelectPlatesByUser(path string, userID int) []models.Plate {
	mu.Lock()
	dbx := connect(path)
	var ps []models.Plate
	err := dbx.Select(&ps, fmt.Sprintf(
		`SELECT * FROM plates WHERE USER_ID = '%d' ORDER BY WEIGHT DESC;`, userID,
	))
	mu.Unlock()
	if err != nil {
		log.Printf("WARN: SelectPlatesByUser: %v", err)
	}
	return ps
}

// ReplacePlates wipes and rewrites a user's plate inventory atomically enough
// for our purposes (single-user-per-write app). Plates with PairCount == 0
// AND Weight == 0 are skipped (empty form rows).
func ReplacePlates(path string, userID int, plates []models.Plate) {
	exec(path, fmt.Sprintf(`DELETE FROM plates WHERE USER_ID = '%d';`, userID))
	for _, p := range plates {
		if p.Weight.Sign() <= 0 {
			continue
		}
		if p.PairCount < 0 {
			p.PairCount = 0
		}
		stmt := fmt.Sprintf(
			`INSERT INTO plates (USER_ID, WEIGHT, PAIR_COUNT) VALUES ('%d', %s, '%d');`,
			userID, p.Weight.String(), p.PairCount,
		)
		exec(path, stmt)
	}
}
