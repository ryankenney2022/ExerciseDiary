package db

import (
	"fmt"
	"log"

	"github.com/aceberg/ExerciseDiary/internal/check"
	"github.com/aceberg/ExerciseDiary/internal/models"
)

// SelectTemplates - all plans, newest first (items NOT loaded).
func SelectTemplates(path string) []models.WorkoutTemplate {
	mu.Lock()
	dbx := connect(path)
	var ts []models.WorkoutTemplate
	err := dbx.Select(&ts, `SELECT * FROM workout_templates ORDER BY ID DESC;`)
	mu.Unlock()
	if err != nil {
		log.Printf("WARN: SelectTemplates: %v", err)
	}
	return ts
}

// GetTemplate - one plan with its items ordered by POSITION. Returns a
// zero-value struct (ID == 0) if the plan does not exist.
func GetTemplate(path string, id int) models.WorkoutTemplate {
	mu.Lock()
	dbx := connect(path)
	var t models.WorkoutTemplate
	_ = dbx.Get(&t, fmt.Sprintf(`SELECT * FROM workout_templates WHERE ID = '%d';`, id))
	err := dbx.Select(&t.Items, fmt.Sprintf(
		`SELECT * FROM template_items WHERE TEMPLATE_ID = '%d' ORDER BY POSITION ASC, ID ASC;`, id))
	mu.Unlock()
	if err != nil {
		log.Printf("WARN: GetTemplate items: %v", err)
	}
	return t
}

// SelectTemplateCounts - exercise count per template ID, for library cards.
func SelectTemplateCounts(path string) map[int]int {
	out := map[int]int{}
	mu.Lock()
	dbx := connect(path)
	rows, err := dbx.Queryx(`SELECT TEMPLATE_ID, COUNT(*) FROM template_items GROUP BY TEMPLATE_ID;`)
	mu.Unlock()
	if err != nil {
		log.Printf("WARN: SelectTemplateCounts: %v", err)
		return out
	}
	defer rows.Close()
	for rows.Next() {
		var tid, c int
		if err := rows.Scan(&tid, &c); err == nil {
			out[tid] = c
		}
	}
	return out
}

// SaveTemplate - insert (ID == 0) or update a plan, then full-replace its
// items from the provided slice. POSITION is assigned by slice order.
// Returns the template ID.
func SaveTemplate(path string, t models.WorkoutTemplate, items []models.TemplateItem) int {
	mu.Lock()
	defer mu.Unlock()
	dbx := connect(path)

	id := t.ID
	if id == 0 {
		res, err := dbx.Exec(fmt.Sprintf(
			`INSERT INTO workout_templates (NAME, NOTE, CREATED_AT) VALUES ('%s','%s','%s');`,
			quoteStr(t.Name), quoteStr(t.Note), quoteStr(t.CreatedAt)))
		check.IfError(err)
		lid, _ := res.LastInsertId()
		id = int(lid)
	} else {
		_, err := dbx.Exec(fmt.Sprintf(
			`UPDATE workout_templates SET NAME = '%s', NOTE = '%s' WHERE ID = '%d';`,
			quoteStr(t.Name), quoteStr(t.Note), id))
		check.IfError(err)
	}

	_, err := dbx.Exec(fmt.Sprintf(`DELETE FROM template_items WHERE TEMPLATE_ID = '%d';`, id))
	check.IfError(err)

	for pos, it := range items {
		_, err := dbx.Exec(fmt.Sprintf(
			`INSERT INTO template_items
			 (TEMPLATE_ID, EXERCISE_ID, POSITION, TARGET_SETS, TARGET_REPS, TARGET_SECONDS, NOTE)
			 VALUES ('%d','%d','%d','%d','%d','%d','%s');`,
			id, it.ExerciseID, pos, it.TargetSets, it.TargetReps, it.TargetSeconds, quoteStr(it.Note)))
		check.IfError(err)
	}
	return id
}

// DeleteTemplate - remove a plan and all of its items.
func DeleteTemplate(path string, id int) {
	exec(path, fmt.Sprintf(`DELETE FROM template_items WHERE TEMPLATE_ID = '%d';`, id))
	exec(path, fmt.Sprintf(`DELETE FROM workout_templates WHERE ID = '%d';`, id))
}
