package life

import (
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/tasks"
)

// DayData is everything a day page or `balaur day` needs, queried once.
type DayData struct {
	Journal []*core.Record // type=day node for the day (props.date); at most one
	Logged  []*core.Record // entries other kinds, noted_at in day
	Done    []*core.Record // tasks done_at in day
	Recap   *core.Record   // day summary, nil when absent
}

// dayKey is the props.date format (ISO "YYYY-MM-DD").
const dayKey = "2006-01-02"

// Day queries a full day's data: journal (the day node), logged entries, done
// tasks, and recap. The day boundary is [d 00:00, d+1 00:00) in the caller's
// location.
func Day(app core.App, conversationID string, d time.Time) (DayData, error) {
	ds, de := d, d.AddDate(0, 0, 1)
	data := DayData{}

	// Journal: the day's type=day node, keyed by props.date (plan 171).
	// At most one per date; return as a slice to keep DayData.Journal compatible.
	dateStr := ds.Format(dayKey)
	recs, err := app.FindRecordsByFilter("nodes",
		"type = 'day' && status = 'active' && props.date = {:d}", "-created", 1, 0,
		dbx.Params{"d": dateStr})
	if err != nil {
		return data, fmt.Errorf("day journal query: %w", err)
	}
	// Only include the day node in Journal if it has a non-empty body (i.e.
	// the owner actually wrote something that day).
	for _, r := range recs {
		if r.GetString("body") != "" {
			data.Journal = append(data.Journal, r)
		}
	}

	// Logged measures + done tasks/completions over [ds, de). Shared with the
	// period-node aggregator (Range) so the day and period lenses never drift.
	rd, err := Range(app, ds, de)
	if err != nil {
		return data, err
	}
	data.Logged, data.Done = rd.Logged, rd.Done

	// Day recap, when available
	if rec := recap.Find(app, conversationID, recap.Day(d)); rec != nil {
		data.Recap = rec
	}

	return data, nil
}

// RangeData is what was done and logged across an arbitrary [start, end) span —
// the period-node generalisation of DayData's Done/Logged. It carries no
// journal/recap: those are period-type-specific and resolved by the caller.
type RangeData struct {
	Done   []*core.Record // tasks done_at + completion entries noted_at in range
	Logged []*core.Record // measure entries noted_at in range
}

// Range aggregates done tasks/completions and logged measures over [start, end)
// in the caller's location — the data behind a week/month/quarter/year node.
func Range(app core.App, start, end time.Time) (RangeData, error) {
	data := RangeData{}

	// Logged measures: type=measure nodes whose noted_at falls in [start, end).
	logged, err := listMeasuresInRange(app, start, end)
	if err != nil {
		return data, fmt.Errorf("range logged query: %w", err)
	}
	data.Logged = logged

	// Done tasks: completed task nodes with done_at in range. tasks owns the
	// done-task rule (see tasks.DoneBetween) so this aggregator doesn't re-derive it.
	done, err := tasks.DoneBetween(app, start, end)
	if err != nil {
		return data, fmt.Errorf("range done-tasks query: %w", err)
	}
	data.Done = done

	// Completion entries in range. Limit 0 (unlimited): a month/quarter/year can
	// hold far more than a single day's worth.
	recs, err := app.FindRecordsByFilter("entries",
		"kind = 'completion' && noted_at >= {:s} && noted_at < {:e}", "noted_at", 0, 0,
		dbx.Params{"s": store.PBTime(start), "e": store.PBTime(end)})
	if err != nil {
		return data, fmt.Errorf("range completions query: %w", err)
	}
	data.Done = append(data.Done, recs...)

	return data, nil
}
