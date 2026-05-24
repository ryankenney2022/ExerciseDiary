package models

import (
	"github.com/shopspring/decimal"

	"github.com/aceberg/ExerciseDiary/internal/auth"
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

// User - one profile (no auth boundary; identity hint only)
type User struct {
	ID        int    `db:"ID"`
	Name      string `db:"NAME"`
	Color     string `db:"COLOR"`
	CreatedAt string `db:"CREATED_AT"`
}

// Workout - sets grouped by (user, date) with optional name and note
type Workout struct {
	ID     int    `db:"ID"`
	UserID int    `db:"USER_ID"`
	Date   string `db:"DATE"`
	Name   string `db:"NAME"`
	Note   string `db:"NOTE"`
}

// Exercise - one exercise (shared across users)
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
}

// Set - one set
type Set struct {
	ID        int             `db:"ID"`
	WorkoutID int             `db:"WORKOUT_ID"`
	Date      string          `db:"DATE"`
	Name      string          `db:"NAME"`
	Color     string          `db:"COLOR"`
	Weight    decimal.Decimal `db:"WEIGHT"`
	Reps      int             `db:"REPS"`
	Note      string          `db:"NOTE"`
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
}
