package db

import (
	"path/filepath"
	"testing"

	"github.com/aceberg/ExerciseDiary/internal/models"
)

// newTestDB creates a throwaway SQLite file with the full schema applied.
func newTestDB(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	Create(path)       // base tables
	EnsureSchema(path) // added columns + template tables/index
	return path
}

func TestSaveAndGetTemplate(t *testing.T) {
	path := newTestDB(t)

	id := SaveTemplate(path,
		models.WorkoutTemplate{Name: "Push Day A", Note: "chest/tris"},
		[]models.TemplateItem{
			{ExerciseID: 1, TargetSets: 3, TargetReps: 5, Note: "slow eccentric"},
			{ExerciseID: 2, TargetSets: 3, TargetReps: 8},
		})
	if id == 0 {
		t.Fatal("expected non-zero template id")
	}

	got := GetTemplate(path, id)
	if got.Name != "Push Day A" || got.Note != "chest/tris" {
		t.Errorf("meta mismatch: %+v", got)
	}
	if len(got.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got.Items))
	}
	if got.Items[0].ExerciseID != 1 || got.Items[0].TargetSets != 3 || got.Items[0].TargetReps != 5 {
		t.Errorf("item0 mismatch: %+v", got.Items[0])
	}
	if got.Items[0].Note != "slow eccentric" {
		t.Errorf("item note not stored: %q", got.Items[0].Note)
	}
	if got.Items[0].Position != 0 || got.Items[1].Position != 1 {
		t.Errorf("positions not assigned by slice order: %+v", got.Items)
	}
}

func TestSaveTemplateReplacesItems(t *testing.T) {
	path := newTestDB(t)
	id := SaveTemplate(path, models.WorkoutTemplate{Name: "A"}, []models.TemplateItem{
		{ExerciseID: 1, TargetSets: 3, TargetReps: 5},
		{ExerciseID: 2, TargetSets: 3, TargetReps: 5},
	})
	SaveTemplate(path, models.WorkoutTemplate{ID: id, Name: "A v2"}, []models.TemplateItem{
		{ExerciseID: 9, TargetSets: 4, TargetReps: 10},
	})
	got := GetTemplate(path, id)
	if got.Name != "A v2" {
		t.Errorf("expected updated name, got %q", got.Name)
	}
	if len(got.Items) != 1 || got.Items[0].ExerciseID != 9 {
		t.Fatalf("expected items replaced, got %+v", got.Items)
	}
}

func TestDeleteTemplate(t *testing.T) {
	path := newTestDB(t)
	id := SaveTemplate(path, models.WorkoutTemplate{Name: "A"}, []models.TemplateItem{
		{ExerciseID: 1, TargetSets: 3, TargetReps: 5},
	})
	DeleteTemplate(path, id)
	if got := GetTemplate(path, id); got.ID != 0 {
		t.Errorf("expected template gone, got %+v", got)
	}
	if c := SelectTemplateCounts(path)[id]; c != 0 {
		t.Errorf("expected 0 items for deleted template, got %d", c)
	}
}

func TestSelectTemplatesNewestFirstAndCounts(t *testing.T) {
	path := newTestDB(t)
	id1 := SaveTemplate(path, models.WorkoutTemplate{Name: "First"}, nil)
	id2 := SaveTemplate(path, models.WorkoutTemplate{Name: "Second"}, []models.TemplateItem{
		{ExerciseID: 1, TargetSets: 3, TargetReps: 5},
		{ExerciseID: 2, TargetSets: 3, TargetReps: 5},
	})
	ts := SelectTemplates(path)
	if len(ts) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(ts))
	}
	if ts[0].ID != id2 || ts[1].ID != id1 {
		t.Errorf("expected newest first, got %d then %d", ts[0].ID, ts[1].ID)
	}
	counts := SelectTemplateCounts(path)
	if counts[id2] != 2 {
		t.Errorf("expected 2 items for id2, got %d", counts[id2])
	}
	if counts[id1] != 0 {
		t.Errorf("expected 0 items for id1, got %d", counts[id1])
	}
}
