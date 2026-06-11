package tasks

import (
	"math"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Streaks are derived at read time from completion entries — never stored
// (one source of truth). The definition stays explainable: completions on
// distinct local days, counted backward while each gap fits the rule's
// period; a habit whose last completion is more than one period old has
// lapsed and reads 0.

// periodDays is the rule's expected gap between completions.
func periodDays(r Rule) int {
	switch r.Kind {
	case "daily":
		return 1
	case "every":
		return r.N
	case "weekly":
		return 7
	case "monthly":
		return 31
	}
	return 0
}

// noonOf anchors a date at local noon — date arithmetic DST cannot wobble
// (transitions happen in the small hours).
func noonOf(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 12, 0, 0, 0, t.Location())
}

// daysBetween counts calendar days from a to b in their location.
func daysBetween(a, b time.Time) int {
	return int(math.Round(noonOf(b).Sub(noonOf(a)).Hours() / 24))
}

// CompletionDays returns the distinct local days with a completion for the
// task, ascending, noon-anchored.
func CompletionDays(app core.App, taskID string, loc *time.Location) ([]time.Time, error) {
	recs, err := app.FindRecordsByFilter("entries",
		"kind = 'completion' && task = {:t}", "noted_at", 0, 0, dbx.Params{"t": taskID})
	if err != nil {
		return nil, err
	}
	var days []time.Time
	for _, r := range recs {
		d := noonOf(r.GetDateTime("noted_at").Time().In(loc))
		if len(days) == 0 || !days[len(days)-1].Equal(d) {
			days = append(days, d)
		}
	}
	return days, nil
}

// Streak counts the live run: starting from the latest completion day,
// completions whose gaps fit the rule's period. 0 when there is no rule,
// no completions, or the habit has lapsed relative to today.
func Streak(r Rule, days []time.Time, today time.Time) int {
	period := periodDays(r)
	if period == 0 || len(days) == 0 {
		return 0
	}
	if daysBetween(days[len(days)-1], today) > period {
		return 0
	}
	streak := 1
	for i := len(days) - 2; i >= 0; i-- {
		if daysBetween(days[i], days[i+1]) > period {
			break
		}
		streak++
	}
	return streak
}

// StreakFor loads completions and computes the live streak for one task.
// Errors read as 0 — a missing streak must never block a briefing.
func StreakFor(app core.App, rec *core.Record, now time.Time) int {
	rule, err := Parse(rec.GetString("recur"))
	if err != nil || rule.IsZero() {
		return 0
	}
	days, err := CompletionDays(app, rec.Id, now.Location())
	if err != nil {
		return 0
	}
	return Streak(rule, days, now)
}
