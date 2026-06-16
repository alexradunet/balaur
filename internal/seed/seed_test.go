package seed

import (
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/storetest"
)

// total sums every collection count in a Result — the seed's footprint.
func total(r *Result) int {
	return r.Messages + r.Tasks + r.Memories + r.Skills +
		r.LifeEntries + r.Summaries + r.Boards + r.Heads
}

func TestRunSeedsAllCollections(t *testing.T) {
	app := storetest.NewApp(t)

	res, err := Run(app)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Every collection should gain at least one record.
	checks := map[string]int{
		"messages": res.Messages, "tasks": res.Tasks, "memories": res.Memories,
		"skills": res.Skills, "life_entries": res.LifeEntries,
		"summaries": res.Summaries, "boards": res.Boards, "heads": res.Heads,
	}
	for name, n := range checks {
		if n <= 0 {
			t.Errorf("%s: seeded %d records, want > 0", name, n)
		}
	}

	// Spot-check that records actually landed and carry the marker.
	if n, _ := app.CountRecords("messages", dbx.HashExp{"origin": Marker}); int(n) != res.Messages {
		t.Errorf("marked messages = %d, reported %d", n, res.Messages)
	}
	if n, _ := app.CountRecords("tasks", dbx.HashExp{"source": Marker}); int(n) != res.Tasks {
		t.Errorf("marked tasks = %d, reported %d", n, res.Tasks)
	}
}

func TestRunIsIdempotent(t *testing.T) {
	app := storetest.NewApp(t)

	if _, err := Run(app); err != nil {
		t.Fatalf("first Run: %v", err)
	}
	second, err := Run(app)
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if got := total(second); got != 0 {
		t.Fatalf("second Run created %d records, want 0 (idempotent)", got)
	}
}

func TestResetRemovesOnlySeededData(t *testing.T) {
	app := storetest.NewApp(t)

	// A real (non-seeded) task must survive a reset.
	col, err := app.FindCollectionByNameOrId("tasks")
	if err != nil {
		t.Fatalf("tasks collection: %v", err)
	}
	real := core.NewRecord(col)
	real.Set("title", "real task")
	real.Set("status", "open")
	real.Set("source", "owner")
	if err := app.Save(real); err != nil {
		t.Fatalf("saving real task: %v", err)
	}

	first, err := Run(app)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	removed, err := Reset(app)
	if err != nil {
		t.Fatalf("Reset: %v", err)
	}
	if total(removed) != total(first) {
		t.Fatalf("Reset removed %d records, seeded %d", total(removed), total(first))
	}

	// The real task is untouched; seeded tasks are gone.
	if _, err := app.FindRecordById("tasks", real.Id); err != nil {
		t.Errorf("real task was deleted by Reset: %v", err)
	}
	if n, _ := app.CountRecords("tasks", dbx.HashExp{"source": Marker}); n != 0 {
		t.Errorf("seeded tasks remain after Reset: %d", n)
	}

	// Reseeding after a reset works and restores the full footprint.
	again, err := Run(app)
	if err != nil {
		t.Fatalf("reseed: %v", err)
	}
	if total(again) != total(first) {
		t.Errorf("reseed created %d records, want %d", total(again), total(first))
	}
}
