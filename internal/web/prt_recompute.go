package web

import (
	"log"

	"github.com/aceberg/ExerciseDiary/internal/db"
)

// recomputeAllPRTScores walks every stored PRT test and rewrites its cached
// scores using the current embedded score tables. Cheap (a handful of rows in
// practice) and idempotent. Called at startup so users on an older image
// benefit from any table corrections without having to manually re-save each
// test.
//
// Tests with no snapshot sex/age (e.g. logged before the user filled out
// their profile) are left untouched.
func recomputeAllPRTScores(dbPath string) {
	users := db.SelectUsers(dbPath)
	changed := 0
	scanned := 0

	for _, u := range users {
		tests := db.SelectPRTByUser(dbPath, u.ID)
		for _, t := range tests {
			scanned++
			if t.SexAtTest == "" || t.AgeAtTest == 0 {
				continue
			}
			before := t
			computeAndStash(&t)
			if t.PushupScore != before.PushupScore ||
				t.PlankScore != before.PlankScore ||
				t.RunScore != before.RunScore ||
				t.OverallScore != before.OverallScore ||
				t.OverallCategory != before.OverallCategory {
				db.UpdatePRT(dbPath, t)
				changed++
			}
		}
	}

	if scanned > 0 {
		log.Printf("INFO: PRT recompute pass: %d tests scanned, %d updated", scanned, changed)
	}
}
