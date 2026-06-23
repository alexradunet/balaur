package nodes_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/storetest"
)

func TestCreateAndProps(t *testing.T) {
	app := storetest.NewApp(t)

	rec, err := nodes.Create(app, "memory", "Likes tea", "Black, no sugar.", nodes.StatusProposed,
		map[string]any{"category": "preference", "importance": 5})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rec.GetString("type") != "memory" || rec.GetString("status") != nodes.StatusProposed {
		t.Fatalf("unexpected type/status: %q/%q", rec.GetString("type"), rec.GetString("status"))
	}
	if got := nodes.PropString(rec, "category"); got != "preference" {
		t.Errorf("PropString(category) = %q, want preference", got)
	}
	if got := nodes.PropInt(rec, "importance"); got != 5 {
		t.Errorf("PropInt(importance) = %d, want 5", got)
	}

	// The create was audited.
	if n, _ := app.CountRecords("audit_log"); n == 0 {
		t.Error("expected an audit row after Create")
	}
}

func TestTransitionLifecycle(t *testing.T) {
	app := storetest.NewApp(t)
	rec, err := nodes.Create(app, "memory", "x", "", nodes.StatusProposed, nil)
	if err != nil {
		t.Fatal(err)
	}

	// proposed → archived is forbidden.
	if _, err := nodes.Transition(app, rec.Id, nodes.StatusArchived); err == nil {
		t.Error("proposed → archived should be rejected")
	}
	// proposed → active is allowed.
	if _, err := nodes.Transition(app, rec.Id, nodes.StatusActive); err != nil {
		t.Errorf("proposed → active: %v", err)
	}
}

func TestAddEdgeIdempotent(t *testing.T) {
	app := storetest.NewApp(t)
	a, _ := nodes.Create(app, "note", "A", "", nodes.StatusActive, nil)
	b, _ := nodes.Create(app, "note", "B", "", nodes.StatusActive, nil)

	e1, err := nodes.AddEdge(app, a.Id, b.Id, "", "")
	if err != nil {
		t.Fatalf("AddEdge: %v", err)
	}
	if e1.GetString("type") != nodes.DefaultEdgeType {
		t.Errorf("default edge type = %q, want %q", e1.GetString("type"), nodes.DefaultEdgeType)
	}
	// Second identical add returns the same edge, not an error.
	e2, err := nodes.AddEdge(app, a.Id, b.Id, "", "")
	if err != nil {
		t.Fatalf("AddEdge (dup): %v", err)
	}
	if e2.Id != e1.Id {
		t.Errorf("duplicate edge created: %q != %q", e2.Id, e1.Id)
	}
	if n, _ := app.CountRecords("edges"); n != 1 {
		t.Errorf("edge count = %d, want 1", n)
	}
}

func TestTraversalExcludesNonActive(t *testing.T) {
	app := storetest.NewApp(t)
	hub, _ := nodes.Create(app, "note", "Hub", "", nodes.StatusActive, nil)
	activeN, _ := nodes.Create(app, "note", "Active", "", nodes.StatusActive, nil)
	proposed, _ := nodes.Create(app, "memory", "Proposed", "", nodes.StatusProposed, nil)

	// hub → active, hub → proposed (outbound); proposed → hub (inbound).
	if _, err := nodes.AddEdge(app, hub.Id, activeN.Id, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := nodes.AddEdge(app, hub.Id, proposed.Id, "", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := nodes.AddEdge(app, proposed.Id, hub.Id, "", ""); err != nil {
		t.Fatal(err)
	}

	out, err := nodes.Outbound(app, hub.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Id != activeN.Id {
		t.Errorf("Outbound returned %d nodes (want 1 active); proposed must be excluded", len(out))
	}
	back, err := nodes.Backlinks(app, hub.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(back) != 0 {
		t.Errorf("Backlinks returned %d nodes; the only inbound source is proposed and must be excluded", len(back))
	}
	nb, err := nodes.Neighborhood(app, hub.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(nb) != 1 || nb[0].Id != activeN.Id {
		t.Errorf("Neighborhood returned %d nodes (want 1 active)", len(nb))
	}
}

func TestDropCascadesEdges(t *testing.T) {
	app := storetest.NewApp(t)
	a, _ := nodes.Create(app, "note", "A", "", nodes.StatusActive, nil)
	b, _ := nodes.Create(app, "note", "B", "", nodes.StatusActive, nil)
	if _, err := nodes.AddEdge(app, a.Id, b.Id, "", ""); err != nil {
		t.Fatal(err)
	}
	if err := nodes.Drop(app, a.Id); err != nil {
		t.Fatalf("Drop: %v", err)
	}
	if n, _ := app.CountRecords("edges"); n != 0 {
		t.Errorf("edges after cascade delete = %d, want 0", n)
	}
}
