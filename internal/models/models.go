package models

import (
	"github.com/shopspring/decimal"

	"github.com/aceberg/ExerciseDiary/internal/auth"
	"github.com/aceberg/ExerciseDiary/internal/prt"
)

// Conf - web gui config
type Conf struct {
	Host      string
	Port      string
	Theme     string
	Color     string
	Icon      string
	DBPath    string
	DirPath   string
	ConfPath  string
	NodePath  string
	HeatColor string
	PageStep  int
	Auth      bool
}

// User - one profile (no auth boundary; identity hint only).
// Sex/DOB/Altitude are used for Navy PRT scoring; safe to leave blank for
// users who only log workouts.
type User struct {
	ID           int    `db:"ID"`
	Name         string `db:"NAME"`
	Color        string `db:"COLOR"`
	CreatedAt    string `db:"CREATED_AT"`
	Sex          string `db:"SEX"`           // "M" or "F"; "" if unset
	DOB          string `db:"DOB"`           // YYYY-MM-DD; "" if unset
	Altitude     string `db:"ALTITUDE"`      // "low" (<5000 ft) or "high"
	DistanceUnit string `db:"DISTANCE_UNIT"` // "mi" (default) or "km" - for cardio entries
}

// PRTTest - one Navy Physical Readiness Test session for a user.
// Each event is optional (individual logging is supported); cached scores are
// recomputed on every save/edit.
type PRTTest struct {
	ID     int    `db:"ID"`
	UserID int    `db:"USER_ID"`
	Date   string `db:"DATE"`
	Note   string `db:"NOTE"`

	// Snapshots locked at save time so scores stay stable if profile changes.
	SexAtTest      string `db:"SEX_AT_TEST"`
	AgeAtTest      int    `db:"AGE_AT_TEST"`
	AltitudeAtTest string `db:"ALTITUDE_AT_TEST"`

	// Raw inputs (0 = not done).
	Pushups      int `db:"PUSHUPS"`
	PlankSeconds int `db:"PLANK_SECONDS"`
	RunSeconds   int `db:"RUN_SECONDS"`

	// Cached computed scores. 0 here means "not done" *only when paired with a
	// 0 raw input*; a real 0 (= failure with raw > 0) is also possible.
	PushupScore     int    `db:"PUSHUP_SCORE"`
	PlankScore      int    `db:"PLANK_SCORE"`
	RunScore        int    `db:"RUN_SCORE"`
	OverallScore    int    `db:"OVERALL_SCORE"`
	OverallCategory string `db:"OVERALL_CATEGORY"`
}

// Workout - sets grouped by (user, date) with optional name and note
type Workout struct {
	ID     int    `db:"ID"`
	UserID int    `db:"USER_ID"`
	Date   string `db:"DATE"`
	Name   string `db:"NAME"`
	Note   string `db:"NOTE"`
}

// Exercise - one exercise (shared across users).
// Kind = "strength" (Weight/Reps apply) or "cardio" (Duration/Distance apply).
type Exercise struct {
	ID       int             `db:"ID"`
	Group    string          `db:"GR"`
	Place    string          `db:"PLACE"`
	Name     string          `db:"NAME"`
	Descr    string          `db:"DESCR"`
	Image    string          `db:"IMAGE"`
	VideoURL string          `db:"VIDEO_URL"`
	Color    string          `db:"COLOR"`
	Weight   decimal.Decimal `db:"WEIGHT"`
	Reps     int             `db:"REPS"`
	Kind     string          `db:"KIND"` // "strength" or "cardio"
}

// Set - one logged entry inside a workout. Most fields are optional:
//   - Strength entries use Weight + Reps; cardio fields stay at 0/""
//   - Cardio entries use DurationSeconds + DistanceValue (+ optional HR/cal/equipment);
//     Weight + Reps stay at 0
type Set struct {
	ID              int             `db:"ID"`
	WorkoutID       int             `db:"WORKOUT_ID"`
	Date            string          `db:"DATE"`
	Name            string          `db:"NAME"`
	Color           string          `db:"COLOR"`
	Weight          decimal.Decimal `db:"WEIGHT"`
	Reps            int             `db:"REPS"`
	Note            string          `db:"NOTE"`
	DurationSeconds int             `db:"DURATION_SECONDS"`
	DistanceValue   decimal.Decimal `db:"DISTANCE_VALUE"`
	AvgHR           int             `db:"AVG_HR"`
	MaxHR           int             `db:"MAX_HR"`
	Calories        int             `db:"CALORIES"`
	Equipment       string          `db:"EQUIPMENT"`
}

// AllExData - all sets and exercises
type AllExData struct {
	Exs    []Exercise
	Sets   []Set
	Weight []BodyWeight
}

// HeatMapData - data for HeatMap
type HeatMapData struct {
	X string
	Y string
	D string
	V int
}

// BodyWeight - store weight
type BodyWeight struct {
	ID     int             `db:"ID"`
	UserID int             `db:"USER_ID"`
	Date   string          `db:"DATE"`
	Weight decimal.Decimal `db:"WEIGHT"`
}

// GuiData - web gui data
type GuiData struct {
	Config       Conf
	Themes       []string
	ExData       AllExData
	GroupMap     map[string]string
	OneEx        Exercise
	HeatMap      []HeatMapData
	Version      string
	Auth         auth.Conf
	Users        []User
	CurrentUser  User
	TodayWorkout Workout
	PRTTests     []PRTTest
	OnePRT       PRTTest
	LastPRT      PRTTest // most recent test for the current user (for index summary card)
	ScoreSheet   *prt.Sheet // standards table for current user's bracket (nil if unknown)
	ExGroups     []string // distinct, sorted group names across all exercises
	ExPlaces     []string // distinct, sorted "place in group" values across all exercises
	LastDone     map[string]string // exercise name -> most recent DATE the current user logged it
	CardioSummary CardioSummary // 7-day cardio totals for the current user (Stats page)
}

// CardioSummary aggregates a user's cardio activity over a recent window.
type CardioSummary struct {
	TotalMinutes int
	TotalDist    decimal.Decimal
	Unit         string
	SessionCount int
}
