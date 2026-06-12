package life

import (
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/store"
)

// DayData is everything a day page or `balaur day` needs, queried once.
type DayData struct {
	Journal []*core.Record // entries kind=journal, noted_at in day
	Logged  []*core.Record // entries other kinds, noted_at in day
	Done    []*core.Record // tasks done_at in day
	Recap   *core.Record   // day summary, nil when absent
}

// Day queries a full day's data: journal, logged entries, done tasks, and recap.
// The day boundary is [d 00:00, d+1 00:00) in the caller's location.
func Day(app core.App, conversationID string, d time.Time) (DayData, error) {
	ds, de := d, d.AddDate(0, 0, 1)
	data := DayData{}

	// Journal entries: kind='journal', noted_at in [ds, de)
	if recs, err := app.FindRecordsByFilter("entries",
		"kind = 'journal' && noted_at >= {:s} && noted_at < {:e}", "noted_at", 200, 0,
		dbx.Params{"s": store.PBTime(ds), "e": store.PBTime(de)}); err == nil {
		data.Journal = recs
	}

	// Logged entries: kind != 'completion' && kind != 'journal', noted_at in [ds, de)
	if recs, err := app.FindRecordsByFilter("entries",
		"kind != 'completion' && kind != 'journal' && noted_at >= {:s} && noted_at < {:e}",
		"noted_at", 200, 0,
		dbx.Params{"s": store.PBTime(ds), "e": store.PBTime(de)}); err == nil {
		data.Logged = recs
	}

	// Done tasks: status='done', done_at in [ds, de)
	// Also include completions (kind='completion') from entries
	if recs, err := app.FindRecordsByFilter("tasks",
		"status = 'done' && done_at >= {:s} && done_at < {:e}", "done_at", 200, 0,
		dbx.Params{"s": store.PBTime(ds), "e": store.PBTime(de)}); err == nil {
		data.Done = recs
	}
	if recs, err := app.FindRecordsByFilter("entries",
		"kind = 'completion' && noted_at >= {:s} && noted_at < {:e}", "noted_at", 200, 0,
		dbx.Params{"s": store.PBTime(ds), "e": store.PBTime(de)}); err == nil {
		data.Done = append(data.Done, recs...)
	}

	// Day recap, when available
	if rec := recap.Find(app, conversationID, recap.Day(d)); rec != nil {
		data.Recap = rec
	}

	return data, nil
}
