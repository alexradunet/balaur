package web

// graph_test.go — buildGraphData (the /ui/graph.json data builder).
// Correctness pins:
//   - the focus + its active neighbors and the edges between them are returned
//   - a PROPOSED (non-active) neighbor never appears — the consent spine holds
//     because traversal goes through nodes.Outbound/Backlinks (active-filtered)

import (
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
	_ "github.com/alexradunet/balaur/migrations"
)

func TestBuildGraphData(t *testing.T) {
	app := newWebApp(t)

	mkNode := func(title, status string) *core.Record {
		t.Helper()
		coll, err := app.FindCollectionByNameOrId("nodes")
		if err != nil {
			t.Fatalf("nodes collection: %v", err)
		}
		r := core.NewRecord(coll)
		r.Set("type", "note")
		r.Set("title", title)
		r.Set("status", status)
		if err := app.Save(r); err != nil {
			t.Fatalf("save node %q: %v", title, err)
		}
		return r
	}

	focus := mkNode("Focus", nodes.StatusActive)
	active := mkNode("Active Neighbor", nodes.StatusActive)
	proposed := mkNode("Proposed Neighbor", nodes.StatusProposed)
	if _, err := nodes.AddEdge(app, focus.Id, active.Id, "links", ""); err != nil {
		t.Fatalf("edge focus→active: %v", err)
	}
	if _, err := nodes.AddEdge(app, focus.Id, proposed.Id, "links", ""); err != nil {
		t.Fatalf("edge focus→proposed: %v", err)
	}

	gd, err := buildGraphData(app, focus.Id, 2)
	if err != nil {
		t.Fatalf("buildGraphData: %v", err)
	}

	hasNode := func(id string) bool {
		for _, n := range gd.Nodes {
			if n.ID == id {
				return true
			}
		}
		return false
	}
	hasLink := func(s, tg string) bool {
		for _, l := range gd.Links {
			if l.Source == s && l.Target == tg {
				return true
			}
		}
		return false
	}

	if !hasNode(focus.Id) {
		t.Error("focus node missing from graph")
	}
	if !hasNode(active.Id) {
		t.Error("active neighbor missing from graph")
	}
	if hasNode(proposed.Id) {
		t.Error("consent breach: proposed neighbor leaked into the graph")
	}
	if !hasLink(focus.Id, active.Id) {
		t.Error("focus→active link missing")
	}
	if hasLink(focus.Id, proposed.Id) {
		t.Error("consent breach: link to proposed neighbor present")
	}
	if len(gd.Nodes) != 2 {
		t.Errorf("node count = %d, want 2 (focus + active)", len(gd.Nodes))
	}
}

// TestBuildGraphDataInactiveFocus: an inactive focus is an error (the handler
// turns it into a sanitized 404).
func TestBuildGraphDataInactiveFocus(t *testing.T) {
	app := newWebApp(t)
	coll, err := app.FindCollectionByNameOrId("nodes")
	if err != nil {
		t.Fatalf("nodes collection: %v", err)
	}
	r := core.NewRecord(coll)
	r.Set("type", "note")
	r.Set("title", "Hidden")
	r.Set("status", nodes.StatusProposed)
	if err := app.Save(r); err != nil {
		t.Fatalf("save node: %v", err)
	}
	if _, err := buildGraphData(app, r.Id, 2); err == nil {
		t.Error("expected error for a non-active focus node, got nil")
	}
}
