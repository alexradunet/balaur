package seed

import (
	"testing"

	"github.com/pocketbase/dbx"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/storetest"
	"github.com/alexradunet/balaur/internal/tasks"
)

// total sums the fields that are strictly symmetric between Run and Reset.
// Journal entries seeded by world.go are deleted by Reset's combined
// note+journal delete (counted in Notes), so Journal is intentionally excluded
// from total() and verified separately. Edges cascade-delete with nodes and
// are also excluded.
func total(r *Result) int {
	return r.Messages + r.Tasks + r.Memories + r.Skills + r.Notes +
		r.LifeEntries + r.Summaries + r.Heads +
		r.People + r.Places + r.Books + r.Ideas
}

func TestRunSeedsAllCollections(t *testing.T) {
	app := storetest.NewApp(t)

	res, err := Run(app)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// Every base collection should gain at least one record.
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

	// World catalog nodes must all be present.
	worldChecks := map[string]int{
		"people": res.People, "places": res.Places,
		"books": res.Books, "ideas": res.Ideas,
		"journal": res.Journal,
	}
	for name, n := range worldChecks {
		if n <= 0 {
			t.Errorf("world %s: seeded %d, want > 0", name, n)
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

	// Connectivity: edges must exist and day nodes must be created.
	edgeCount, err2 := app.CountRecords("edges", nil)
	if err2 != nil {
		t.Fatalf("counting edges: %v", err2)
	}
	if int(edgeCount) < 100 {
		t.Errorf("edges = %d, want > 100 (graph not connected enough)", edgeCount)
	}

	dayNodes, err3 := app.FindRecordsByFilter("nodes", "type = 'day' && status = 'active'", "", 0, 0, nil)
	if err3 != nil {
		t.Fatalf("counting day nodes: %v", err3)
	}
	if len(dayNodes) == 0 {
		t.Errorf("day nodes = 0, want > 0")
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
	// Journal and Edges from second run must also be zero.
	if second.Journal != 0 {
		t.Errorf("second Run journal = %d, want 0", second.Journal)
	}
	if second.Edges != 0 {
		t.Errorf("second Run edges = %d, want 0", second.Edges)
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
	// plan 171: journal entries are type=day nodes (source=Marker); Reset counts
	// them in removed.Journal (not Notes). Verify symmetry directly.
	if removed.Notes != first.Notes {
		t.Errorf("Reset notes = %d, want %d", removed.Notes, first.Notes)
	}
	if removed.Journal != first.Journal {
		t.Errorf("Reset journal = %d, want %d (first.Journal=%d)", removed.Journal, first.Journal, first.Journal)
	}
	// total() excludes Journal and Edges (cascade-delete); verify the rest matches.
	if total(removed) != total(first) {
		t.Fatalf("Reset removed %d records (total), seeded %d", total(removed), total(first))
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

	// No world catalog nodes remain.
	for _, typ := range []string{"person", "place", "book", "idea"} {
		recs, _ := app.FindRecordsByFilter("nodes",
			"type = {:t} && props.source = {:m}", "", 0, 0,
			dbx.Params{"t": typ, "m": Marker})
		if len(recs) != 0 {
			t.Errorf("seeded %s nodes remain after Reset: %d", typ, len(recs))
		}
	}

	// No seed day nodes remain (either props.seed=true or props.source=Marker).
	allDayNodes, _ := app.FindRecordsByFilter("nodes", "type = 'day' && status = 'active'", "", 0, 0, nil)
	for _, r := range allDayNodes {
		p := nodes.Props(r)
		if b, ok := p["seed"].(bool); ok && b {
			t.Errorf("hub seed day node %s remains after Reset", r.Id)
		}
		if s, ok := p["source"].(string); ok && s == Marker {
			t.Errorf("journal seed day node %s remains after Reset", r.Id)
		}
	}

	// No edges remain (all seeded nodes are gone; edges cascade-delete).
	edgeCount, _ := app.CountRecords("edges", nil)
	if int(edgeCount) != 0 {
		t.Errorf("edges remain after Reset: %d", edgeCount)
	}

	// Reseeding after a reset works and restores the full footprint.
	again, err := Run(app)
	if err != nil {
		t.Fatalf("reseed: %v", err)
	}
	if total(again) != total(first) {
		t.Errorf("reseed total = %d, want %d", total(again), total(first))
	}
	if again.Journal != first.Journal {
		t.Errorf("reseed journal = %d, want %d", again.Journal, first.Journal)
	}
}
