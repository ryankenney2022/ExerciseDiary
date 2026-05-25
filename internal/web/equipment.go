package web

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"

	"github.com/aceberg/ExerciseDiary/internal/db"
	"github.com/aceberg/ExerciseDiary/internal/models"
)

// equipmentHandler renders the per-user barbell + plate inventory page.
// First visit shows the equipment defaults (45-lb bar) plus a seed set of
// common plate denominations with zero counts, so the user just fills in
// "how many pairs of each do I own".
func equipmentHandler(c *gin.Context) {
	user := currentUser(c)
	if user.ID == 0 {
		c.Redirect(http.StatusFound, "/users/")
		return
	}

	var guiData models.GuiData
	guiData.Config = appConfig
	guiData.Users = allUsers(c)
	guiData.CurrentUser = user
	guiData.Equipment = db.GetEquipment(appConfig.DBPath, user.ID)
	guiData.Plates = db.SelectPlatesByUser(appConfig.DBPath, user.ID)

	// Seed the form with common denominations when the user has none yet, so
	// the page is immediately usable.
	if len(guiData.Plates) == 0 {
		seeds := []float64{45, 35, 25, 10, 5, 2.5}
		for _, w := range seeds {
			guiData.Plates = append(guiData.Plates, models.Plate{
				UserID:    user.ID,
				Weight:    decimal.NewFromFloat(w),
				PairCount: 0,
			})
		}
	}

	c.HTML(http.StatusOK, "header.html", guiData)
	c.HTML(http.StatusOK, "equipment.html", guiData)
}

// saveEquipmentHandler persists the barbell weight and replaces the plate
// inventory in one POST. Form schema:
//
//	barbell_weight=45
//	plate_weight[]=45  plate_pairs[]=2
//	plate_weight[]=25  plate_pairs[]=4
//	... (any number; pairs of zero are filtered out)
func saveEquipmentHandler(c *gin.Context) {
	user := currentUser(c)
	if user.ID == 0 {
		c.Redirect(http.StatusFound, "/users/")
		return
	}

	bw, _ := decimal.NewFromString(strings.TrimSpace(c.PostForm("barbell_weight")))
	if bw.Sign() < 0 {
		bw = decimal.Zero
	}
	db.UpsertEquipment(appConfig.DBPath, models.Equipment{
		UserID:        user.ID,
		BarbellWeight: bw,
		Unit:          "lb",
	})

	weights := c.PostFormArray("plate_weight")
	pairs := c.PostFormArray("plate_pairs")
	var plates []models.Plate
	for i, ws := range weights {
		w, _ := decimal.NewFromString(strings.TrimSpace(ws))
		if w.Sign() <= 0 {
			continue
		}
		n := 0
		if i < len(pairs) {
			n, _ = strconv.Atoi(strings.TrimSpace(pairs[i]))
		}
		if n < 0 {
			n = 0
		}
		plates = append(plates, models.Plate{
			UserID:    user.ID,
			Weight:    w,
			PairCount: n,
		})
	}
	db.ReplacePlates(appConfig.DBPath, user.ID, plates)

	c.Redirect(http.StatusFound, "/equipment/")
}

// suggestPlatesHandler returns a JSON plate-loading suggestion for a target
// total weight. Consumed by the workout-entry row's plate hint via fetch().
//
// Response shape (designed for direct rendering):
//
//	{
//	  "target": "150",
//	  "bar": "45",
//	  "unit": "lb",
//	  "side": [ {"w": "25", "n": 2}, {"w": "2.5", "n": 1} ],
//	  "remainder": "0",
//	  "achievable": "150"
//	}
func suggestPlatesHandler(c *gin.Context) {
	user := currentUser(c)
	if user.ID == 0 {
		c.JSON(http.StatusOK, gin.H{"side": []any{}, "remainder": "0"})
		return
	}

	target, _ := decimal.NewFromString(strings.TrimSpace(c.Query("weight")))
	eq := db.GetEquipment(appConfig.DBPath, user.ID)
	plates := db.SelectPlatesByUser(appConfig.DBPath, user.ID)

	s := computePlateSuggestion(target, eq, plates)

	side := make([]gin.H, 0, len(s.SidePlates))
	for _, pc := range s.SidePlates {
		side = append(side, gin.H{"w": pc.Weight.String(), "n": pc.Count})
	}
	c.JSON(http.StatusOK, gin.H{
		"target":     target.String(),
		"bar":        eq.BarbellWeight.String(),
		"unit":       eq.Unit,
		"side":       side,
		"remainder":  s.Remainder.String(),
		"achievable": s.Achievable.String(),
	})
}

// computePlateSuggestion runs the greedy descending plate allocation.
// Inputs are pre-sorted plates (callers use SelectPlatesByUser which already
// orders by WEIGHT DESC).
//
// Logic:
//   - Bail out if target <= bar weight (no plates needed).
//   - perSide = (target - bar) / 2.
//   - For each plate (desc weight): take min(floor(perSide / w), pair_count);
//     subtract w * count from perSide; record if count > 0.
//   - Remainder = whatever's left on one side. Achievable = bar + 2 * loaded.
func computePlateSuggestion(target decimal.Decimal, eq models.Equipment, plates []models.Plate) models.PlateSuggestion {
	s := models.PlateSuggestion{
		TargetWeight:  target,
		BarbellWeight: eq.BarbellWeight,
		Unit:          eq.Unit,
	}
	if target.LessThanOrEqual(eq.BarbellWeight) {
		s.Achievable = eq.BarbellWeight
		return s
	}
	two := decimal.NewFromInt(2)
	perSide := target.Sub(eq.BarbellWeight).Div(two)
	loaded := decimal.Zero

	for _, p := range plates {
		if p.Weight.Sign() <= 0 || p.PairCount <= 0 {
			continue
		}
		if perSide.LessThan(p.Weight) {
			continue
		}
		// max usable count by remaining weight
		ratio := perSide.Div(p.Weight).Floor()
		maxByWeight := int(ratio.IntPart())
		count := maxByWeight
		if count > p.PairCount {
			count = p.PairCount
		}
		if count <= 0 {
			continue
		}
		used := p.Weight.Mul(decimal.NewFromInt(int64(count)))
		perSide = perSide.Sub(used)
		loaded = loaded.Add(used)
		s.SidePlates = append(s.SidePlates, models.PlateCount{Weight: p.Weight, Count: count})
	}
	s.Remainder = perSide
	s.Achievable = eq.BarbellWeight.Add(loaded.Mul(two))
	return s
}
