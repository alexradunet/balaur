package life

import (
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/storetest"
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

	// Seed one done task inside the day
	taskColl, _ := app.FindCollectionByNameOrId("tasks")
	taskInside := core.NewRecord(taskColl)
	taskInside.Set("title", "Test task")
	taskInside.Set("status", "done")
	taskInside.Set("done_at", evening)
	if err := app.Save(taskInside); err != nil {
		t.Fatalf("task save: %v", err)
	}

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
