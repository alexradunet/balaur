package seed

import (
	"testing"

	"github.com/pocketbase/dbx"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/storetest"
	"github.com/alexradunet/balaur/internal/tasks"
)

// total sums every collection count in a Result — the seed's footprint.
func total(r *Result) int {
	return r.Messages + r.Tasks + r.Memories + r.Skills + r.Notes +
		r.LifeEntries + r.Summaries + r.Heads
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
		"skills": res.Skills, "notes": res.Notes, "life_entries": res.LifeEntries,
		"summaries": res.Summaries, "heads": res.Heads,
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
	// Tasks are now type=task nodes; count matching nodes.
	if n, _ := app.CountRecords("nodes", dbx.HashExp{"type": "task", "status": nodes.StatusActive}); int(n) < res.Tasks {
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

	// A real (non-seeded) task node must survive a reset.
	real, err := tasks.Create(app, tasks.CreateOpts{Title: "real task", Source: "owner"})
	if err != nil {
		t.Fatalf("creating real task: %v", err)
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

	// The real task node is untouched; seeded task nodes are gone.
	if _, err := app.FindRecordById("nodes", real.Id); err != nil {
		t.Errorf("real task was deleted by Reset: %v", err)
	}
	// No nodes with type=task and props.source=Marker should remain.
	remaining, _ := app.FindRecordsByFilter("nodes",
		"type = {:t} && props.source = {:m}", "", 0, 0,
		dbx.Params{"t": "task", "m": Marker})
	if len(remaining) != 0 {
		t.Errorf("seeded tasks remain after Reset: %d", len(remaining))
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
