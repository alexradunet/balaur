package life

import (
	"testing"
	"time"

	"github.com/alexradunet/balaur/internal/storetest"
	"github.com/alexradunet/balaur/internal/tasks"
)

func TestDayDataBoundary(t *testing.T) {
	app := storetest.NewApp(t)

	// Create test entries and tasks on a specific day and adjacent days.
	// Use a fixed date for determinism.
	loc := time.UTC
	day := time.Date(2026, 6, 12, 0, 0, 0, 0, loc)
	morning := time.Date(2026, 6, 12, 9, 0, 0, 0, loc)
	evening := time.Date(2026, 6, 12, 18, 0, 0, 0, loc)
	nextDay := time.Date(2026, 6, 13, 0, 0, 1, 0, loc) // just after boundary

	// Seed one journal entry inside the day
	journalInside, err := JournalWrite(app, "morning reflection", morning)
	if err != nil {
		t.Fatalf("journal write inside: %v", err)
	}

	// Seed one logged entry (weight) inside the day
	loggedInside, err := Log(app, LogOpts{
		Kind: "weight", ValueNum: 70.5, Unit: "kg", NotedAt: evening,
	})
	if err != nil {
		t.Fatalf("log inside: %v", err)
	}

	// Seed one done task inside the day (done_at will be set to now by tasks.Done,
	// but we need it in the evening window — use tasks.Done with the evening time).
	taskRec, err := tasks.Create(app, tasks.CreateOpts{Title: "Test task"})
	if err != nil {
		t.Fatalf("task create: %v", err)
	}
	if _, err = tasks.Done(app, taskRec, evening); err != nil {
		t.Fatalf("task done: %v", err)
	}
	taskInside, err := app.FindRecordById("nodes", taskRec.Id)
	if err != nil {
		t.Fatalf("reload task: %v", err)
	}
	tasks.Hydrate(taskInside)

	// Seed one journal entry on next day (should not appear)
	journalOutside, err := JournalWrite(app, "next day note", nextDay)
	if err != nil {
		t.Fatalf("journal write outside: %v", err)
	}

	// Query the day
	data, err := Day(app, "", day)
	if err != nil {
		t.Fatalf("day query: %v", err)
	}

	// Verify journal membership: inside yes, outside no
	if len(data.Journal) != 1 {
		t.Fatalf("journal count = %d, want 1", len(data.Journal))
	}
	if data.Journal[0].Id != journalInside.Id {
		t.Fatalf("journal id = %q, want %q", data.Journal[0].Id, journalInside.Id)
	}

	// Verify logged membership: inside yes
	if len(data.Logged) != 1 {
		t.Fatalf("logged count = %d, want 1", len(data.Logged))
	}
	if data.Logged[0].Id != loggedInside.Id {
		t.Fatalf("logged id = %q, want %q", data.Logged[0].Id, loggedInside.Id)
	}

	// Verify done membership: task inside yes
	if len(data.Done) != 1 {
		t.Fatalf("done count = %d, want 1", len(data.Done))
	}
	if data.Done[0].Id != taskInside.Id {
		t.Fatalf("done id = %q, want %q", data.Done[0].Id, taskInside.Id)
	}

	// Verify the outside journal entry does not appear
	for _, rec := range data.Journal {
		if rec.Id == journalOutside.Id {
			t.Fatalf("outside journal entry %q appeared in day query", journalOutside.Id)
		}
	}
}

// TestRange verifies the period aggregator honours the half-open [start, end)
// window: a record at exactly start is included, at exactly end is excluded.
func TestRange(t *testing.T) {
	app := storetest.NewApp(t)
	loc := time.UTC
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, loc)
	end := time.Date(2026, 6, 8, 0, 0, 0, 0, loc) // exclusive

	// Logged measures: at start (incl), mid (incl), one second before start (excl).
	for _, lo := range []LogOpts{
		{Kind: "weight", ValueNum: 80, Unit: "kg", NotedAt: start},
		{Kind: "weight", ValueNum: 79, Unit: "kg", NotedAt: time.Date(2026, 6, 4, 9, 0, 0, 0, loc)},
		{Kind: "weight", ValueNum: 81, Unit: "kg", NotedAt: start.Add(-time.Second)},
	} {
		if _, err := Log(app, lo); err != nil {
			t.Fatalf("log %v: %v", lo.NotedAt, err)
		}
	}

	// Done tasks: one mid (incl), one at exactly end (excl).
	mkDone := func(title string, at time.Time) {
		rec, err := tasks.Create(app, tasks.CreateOpts{Title: title})
		if err != nil {
			t.Fatalf("task create: %v", err)
		}
		if _, err := tasks.Done(app, rec, at); err != nil {
			t.Fatalf("task done: %v", err)
		}
	}
	mkDone("inside", time.Date(2026, 6, 3, 12, 0, 0, 0, loc))
	mkDone("at-end", end)

	rd, err := Range(app, start, end)
	if err != nil {
		t.Fatalf("range: %v", err)
	}
	if len(rd.Logged) != 2 {
		t.Fatalf("logged = %d, want 2 (start+mid incl, before excl)", len(rd.Logged))
	}
	if len(rd.Done) != 1 {
		t.Fatalf("done = %d, want 1 (inside incl, at-end excl)", len(rd.Done))
	}
}
