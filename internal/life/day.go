package life

import (
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
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

	// Logged measures: type=measure nodes whose noted_at falls in [ds, de).
	// noted_at lives in props (JSON) so we filter in Go after loading.
	logged, err := listMeasuresInRange(app, ds, de)
	if err != nil {
		return data, fmt.Errorf("day logged query: %w", err)
	}
	data.Logged = logged

	// Done tasks: type=task nodes with props.state='done' and done_at in [ds, de).
	// done_at lives in props so we filter in Go after loading active task nodes.
	if all, err2 := nodes.ListByTypeStatus(app, "task", nodes.StatusActive); err2 == nil {
		for _, r := range all {
			tasks.Hydrate(r)
			if r.GetString("status") != "done" {
				continue
			}
			doneAt := r.GetDateTime("done_at").Time()
			if doneAt.IsZero() || doneAt.Before(ds) || !doneAt.Before(de) {
				continue
			}
			data.Done = append(data.Done, r)
		}
	}

	recs, err = app.FindRecordsByFilter("entries",
		"kind = 'completion' && noted_at >= {:s} && noted_at < {:e}", "noted_at", 200, 0,
		dbx.Params{"s": store.PBTime(ds), "e": store.PBTime(de)})
	if err != nil {
		return data, fmt.Errorf("day completions query: %w", err)
	}
	data.Done = append(data.Done, recs...)

	// Day recap, when available
	if rec := recap.Find(app, conversationID, recap.Day(d)); rec != nil {
		data.Recap = rec
	}

	return data, nil
}
