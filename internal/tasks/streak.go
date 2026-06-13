package tasks

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

// Streaks are derived at read time from completion entries — never stored
// (one source of truth). The definition stays explainable: completions on
// distinct local days, counted backward while each gap fits the rule's
// period; a habit whose last completion is more than one period old has
// lapsed and reads 0.

// periodDays is the fixed-length period for non-monthly rules.
func periodDays(r Rule) int {
	switch r.Kind {
	case "daily":
		return 1
	case "every":
		return r.N
	case "weekly":
		return 7
	}
	return 0
}

// allowedGapDays is the rule's maximum calendar-day gap from an anchor
// completion to the next one before the streak breaks: the distance to the
// rule's next occurrence after the anchor. Fixed-length kinds keep their
// constants; monthly is calendar-aware (Feb≠July, clamped day-of-month).
func allowedGapDays(r Rule, anchor time.Time) int {
	if r.Kind == "monthly" {
		next := monthlyOn(time.Date(anchor.Year(), anchor.Month()+1, 1, 12, 0, 0, 0, anchor.Location()), r.MonthDay)
		return daysBetween(anchor, next)
	}
	return periodDays(r)
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
	if (periodDays(r) == 0 && r.Kind != "monthly") || len(days) == 0 {
		return 0
	}
	lastDay := days[len(days)-1]
	if daysBetween(lastDay, today) > allowedGapDays(r, lastDay) {
		return 0
	}
	streak := 1
	for i := len(days) - 2; i >= 0; i-- {
		if daysBetween(days[i], days[i+1]) > allowedGapDays(r, days[i]) {
			break
		}
		streak++
	}
	return streak
}

// StreakFor loads completions and computes the live streak for one task.
// Errors read as 0 — a missing streak must never block a briefing.
func StreakFor(app core.App, rec *core.Record, now time.Time) int {
	return StreaksFor(app, []*core.Record{rec}, now)[rec.Id]
}

// StreaksFor computes live streaks for many tasks with ONE completions
// query (TodayBlock runs every turn — per-task queries were an N+1).
// Keyed by task id; tasks without a recurrence rule are absent. Errors
// read as an empty map — a missing streak must never block a briefing.
func StreaksFor(app core.App, recs []*core.Record, now time.Time) map[string]int {
	rules := make(map[string]Rule, len(recs))
	var ids []string
	for _, r := range recs {
		rule, err := Parse(r.GetString("recur"))
		if err != nil || rule.IsZero() {
			continue
		}
		rules[r.Id] = rule
		ids = append(ids, r.Id)
	}
	if len(ids) == 0 {
		return map[string]int{}
	}

	// kind = 'completion' && (task = {:t0} || task = {:t1} || …)
	params := dbx.Params{}
	conds := make([]string, len(ids))
	for i, id := range ids {
		key := fmt.Sprintf("t%d", i)
		conds[i] = fmt.Sprintf("task = {:%s}", key)
		params[key] = id
	}
	rows, err := app.FindRecordsByFilter("entries",
		"kind = 'completion' && ("+strings.Join(conds, " || ")+")",
		"noted_at", 0, 0, params)
	if err != nil {
		return map[string]int{}
	}

	// Same day-folding as CompletionDays, grouped per task: rows arrive
	// sorted by noted_at, so per-task subsequences stay ascending.
	days := make(map[string][]time.Time, len(ids))
	for _, r := range rows {
		id := r.GetString("task")
		d := noonOf(r.GetDateTime("noted_at").Time().In(now.Location()))
		ds := days[id]
		if len(ds) == 0 || !ds[len(ds)-1].Equal(d) {
			days[id] = append(ds, d)
		}
	}

	out := make(map[string]int, len(ids))
	for id, rule := range rules {
		out[id] = Streak(rule, days[id], now)
	}
	return out
}
