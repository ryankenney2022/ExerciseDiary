package db

import (
	"fmt"
	"log"

	"github.com/shopspring/decimal"

	"github.com/aceberg/ExerciseDiary/internal/models"
)

// cardioSeed describes one exercise we'd like to exist out-of-the-box for the
// PRT-eligible cardio events. Idempotent: skipped per-name if already in DB.
type cardioSeed struct {
	Name  string
	Descr string
}

var cardioSeeds = []cardioSeed{
	{"1.5-Mile Run", "Navy PRT 1.5-mile run / walk event"},
	{"2000-Meter Row", "Navy PRT 2 km rower event (Concept 2 standard)"},
	{"500-Yard Swim", "Navy PRT 500-yard swim event"},
	{"450-Meter Swim", "Navy PRT 450-meter swim event"},
	{"Treadmill", "Navy PRT 1.5-mile treadmill event"},
	{"Stationary Bike (Life Fitness)", "Navy PRT stationary bike event - Life Fitness Inc. models"},
	{"Stationary Bike (Other)", "Navy PRT stationary bike event - non-Life Fitness models"},
}

const cardioSeedGroup = "Cardio (PRT)"

// SeedCardioExercises adds canonical PRT cardio exercises if they're missing.
// Existing rows with the same name are left untouched.
func SeedCardioExercises(path string) {
	existing := SelectEx(path)
	have := map[string]bool{}
	for _, e := range existing {
		have[e.Name] = true
	}

	added := 0
	for i, s := range cardioSeeds {
		if have[s.Name] {
			continue
		}
		ex := models.Exercise{
			Group:  cardioSeedGroup,
			Place:  fmt.Sprintf("%d", i+1),
			Name:   s.Name,
			Descr:  s.Descr,
			Kind:   "cardio",
			Weight: decimal.Zero,
			Reps:   0,
			Color:  "",
		}
		InsertEx(path, ex)
		added++
	}
	if added > 0 {
		log.Printf("INFO: seeded %d canonical PRT cardio exercises into group %q", added, cardioSeedGroup)
	}
}