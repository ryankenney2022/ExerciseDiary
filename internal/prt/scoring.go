// Package prt provides Navy Physical Readiness Test scoring.
//
// Score tables are extracted from Guide-5A (DEC 2025) and embedded at build
// time. See prt-data/parse_prt.py for the extraction script and source PDF.
package prt

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

//go:embed score_tables.json
var rawTables []byte

// Event names used throughout the package and stored in DB strings.
const (
	EventPushups = "pushups"
	EventPlank   = "plank"
	EventRun     = "run"
)

// Result of a single-event scoring lookup.
type Result struct {
	Score    int    // 0-100; 0 = failure
	Category string // "Outstanding", "Excellent", "Good", "Satisfactory", "Probationary", or "Failure"
	Level    string // "High" / "Medium" / "Low" / "" (Probationary, Failure)
	Pass     bool   // true if score >= probationary threshold for the event
}

// Label returns "Category Level" with a fallback for Probationary/Failure.
func (r Result) Label() string {
	if r.Category == "" {
		return "—"
	}
	if r.Level == "" {
		return r.Category
	}
	return r.Category + " " + r.Level
}

type tableRow struct {
	Score    int    `json:"score"`
	Category string `json:"category"`
	Level    string `json:"level"`
	Raw      int    `json:"raw"`
}

type scoreTable struct {
	Sex      string     `json:"sex"`
	AgeMin   int        `json:"age_min"`
	AgeMax   int        `json:"age_max"`
	Altitude string     `json:"altitude"`
	Event    string     `json:"event"`
	Rows     []tableRow `json:"rows"`
}

var tables []scoreTable

func init() {
	if err := json.Unmarshal(rawTables, &tables); err != nil {
		log.Fatalf("prt: failed to parse embedded score tables: %v", err)
	}
}

// AgeOnDate returns the user's age in whole years on the given test date.
// Returns 0 if dob can't be parsed.
func AgeOnDate(dob, date string) int {
	d, err := time.Parse("2006-01-02", dob)
	if err != nil {
		return 0
	}
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return 0
	}
	years := t.Year() - d.Year()
	if t.YearDay() < d.YearDay() {
		years--
	}
	if years < 0 {
		return 0
	}
	return years
}

// Score looks up the score for a single event.
//   - sex: "M" or "F"
//   - age: years
//   - altitude: "low" (<5000 ft) or "high" (>=5000 ft)
//   - event: EventPushups, EventPlank, or EventRun
//   - raw: count for pushups; seconds for plank/run
func Score(sex string, age int, altitude, event string, raw int) Result {
	tbl := findTable(sex, age, altitude, event)
	if tbl == nil || len(tbl.Rows) == 0 {
		return Result{}
	}

	// Rows are sorted highest score first. The first row whose threshold is
	// met (or beaten) by `raw` wins. Pushups & plank are higher-is-better;
	// run is lower-is-better.
	for _, r := range tbl.Rows {
		var qualifies bool
		if event == EventRun {
			qualifies = raw > 0 && raw <= r.Raw
		} else {
			qualifies = raw >= r.Raw
		}
		if qualifies {
			return Result{
				Score:    r.Score,
				Category: strings.TrimSpace(r.Category),
				Level:    strings.TrimSpace(r.Level),
				Pass:     true,
			}
		}
	}

	// No qualifying row → failure for this event.
	return Result{Score: 0, Category: "Failure", Pass: false}
}

func findTable(sex string, age int, altitude, event string) *scoreTable {
	sex = strings.ToUpper(strings.TrimSpace(sex))
	for i := range tables {
		t := &tables[i]
		if t.Sex != sex || t.Altitude != altitude || t.Event != event {
			continue
		}
		if age >= t.AgeMin && age <= t.AgeMax {
			return t
		}
	}
	return nil
}

// Categories ranked worst → best. Used to compute overall category as the
// lowest event category (the limiting factor).
var categoryRank = map[string]int{
	"Failure":      0,
	"Probationary": 1,
	"Satisfactory": 2,
	"Good":         3,
	"Excellent":    4,
	"Outstanding":  5,
}

// Overall computes the test-level score and category from the three event
// results. The Navy rule: overall_score = mean of points; overall_category =
// the lowest event category. Any event that's a Failure makes the whole test
// a Failure. Skips events whose Raw is 0 (not done).
//
// Returns score=-1 if no events were scored.
func Overall(events ...Result) (score int, category string, pass bool) {
	scored := 0
	sum := 0
	minCat := -1
	hasFail := false
	for _, e := range events {
		if e.Category == "" {
			continue // event not done
		}
		scored++
		sum += e.Score
		rank, ok := categoryRank[e.Category]
		if !ok {
			continue
		}
		if minCat == -1 || rank < minCat {
			minCat = rank
		}
		if e.Category == "Failure" {
			hasFail = true
		}
	}
	if scored == 0 {
		return -1, "", false
	}
	score = sum / scored
	switch {
	case hasFail:
		category = "Failure"
	case minCat >= 0:
		for name, rank := range categoryRank {
			if rank == minCat {
				category = name
				break
			}
		}
	}
	pass = !hasFail && minCat >= categoryRank["Probationary"]
	return score, category, pass
}

// ParseMMSS converts "mm:ss" or "m:ss" into total seconds. Returns 0 on
// empty / invalid input.
func ParseMMSS(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	var m, sec int
	if _, err := fmt.Sscanf(s, "%d:%d", &m, &sec); err != nil {
		return 0
	}
	if sec < 0 || sec >= 60 || m < 0 {
		return 0
	}
	return m*60 + sec
}

// FormatMMSS converts seconds back to "mm:ss". Returns "" for 0.
func FormatMMSS(secs int) string {
	if secs <= 0 {
		return ""
	}
	return fmt.Sprintf("%d:%02d", secs/60, secs%60)
}
