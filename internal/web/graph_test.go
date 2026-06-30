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

// TestBuildWholeGraphData: the unanchored whole-graph builder returns every
// active node and the edges among them, with per-type icons, and excludes
// proposed nodes (the consent spine).
func TestBuildWholeGraphData(t *testing.T) {
	app := newWebApp(t)

	mk := func(typ, title, status string) *core.Record {
		t.Helper()
		coll, err := app.FindCollectionByNameOrId("nodes")
		if err != nil {
			t.Fatalf("nodes collection: %v", err)
		}
		r := core.NewRecord(coll)
		r.Set("type", typ)
		r.Set("title", title)
		r.Set("status", status)
		if err := app.Save(r); err != nil {
			t.Fatalf("save node %q: %v", title, err)
		}
		return r
	}

	a := mk("note", "Alpha", nodes.StatusActive)
	b := mk("person", "Beta", nodes.StatusActive)
	proposed := mk("note", "Pending", nodes.StatusProposed)
	if _, err := nodes.AddEdge(app, a.Id, b.Id, "links", ""); err != nil {
		t.Fatalf("edge a→b: %v", err)
	}
	if _, err := nodes.AddEdge(app, a.Id, proposed.Id, "links", ""); err != nil {
		t.Fatalf("edge a→proposed: %v", err)
	}

	gd, err := buildWholeGraphData(app)
	if err != nil {
		t.Fatalf("buildWholeGraphData: %v", err)
	}

	byID := map[string]graphNode{}
	for _, n := range gd.Nodes {
		byID[n.ID] = n
	}
	if _, ok := byID[a.Id]; !ok {
		t.Error("active node Alpha missing from whole graph")
	}
	if _, ok := byID[proposed.Id]; ok {
		t.Error("consent breach: proposed node leaked into the whole graph")
	}
	if got := byID[a.Id].Icon; got != "📝" {
		t.Errorf("note icon = %q, want 📝", got)
	}
	if got := byID[b.Id].Icon; got != "👤" {
		t.Errorf("person icon = %q, want 👤", got)
	}
	hasLink := false
	for _, l := range gd.Links {
		if l.Source == a.Id && l.Target == b.Id {
			hasLink = true
		}
		if l.Target == proposed.Id {
			t.Error("consent breach: link to proposed node present in whole graph")
		}
	}
	if !hasLink {
		t.Error("a→b link missing from whole graph")
	}
	if gd.Links == nil {
		t.Error("Links must be non-nil ([] not null) — force-graph throws on null")
	}
}

// TestBuildGraphDataNoEdges: a node with no edges must return a non-nil empty
// Links slice. JSON `null` breaks force-graph (`null.some(...)`), which is the
// common case while the graph has no links yet.
func TestBuildGraphDataNoEdges(t *testing.T) {
	app := newWebApp(t)
	coll, err := app.FindCollectionByNameOrId("nodes")
	if err != nil {
		t.Fatalf("nodes collection: %v", err)
	}
	r := core.NewRecord(coll)
	r.Set("type", "note")
	r.Set("title", "Lonely")
	r.Set("status", nodes.StatusActive)
	if err := app.Save(r); err != nil {
		t.Fatalf("save node: %v", err)
	}
	gd, err := buildGraphData(app, r.Id, 2)
	if err != nil {
		t.Fatalf("buildGraphData: %v", err)
	}
	if gd.Links == nil {
		t.Error("Links must be non-nil ([] not null) for a node with no edges — force-graph throws on null")
	}
	if len(gd.Nodes) != 1 {
		t.Errorf("node count = %d, want 1 (lone focus)", len(gd.Nodes))
	}
}

// TestBuildGraphDataDepth2AndInbound: the batched BFS collects depth-2 outbound
// chains, inbound edges, excludes proposed nodes (consent spine), and drops
// dangling links (links whose endpoint was excluded by the consent filter).
func TestBuildGraphDataDepth2AndInbound(t *testing.T) {
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
	a := mkNode("A", nodes.StatusActive)   // depth-1 outbound
	b := mkNode("B", nodes.StatusActive)   // depth-2 outbound (via A)
	c := mkNode("C", nodes.StatusActive)   // depth-1 inbound
	p := mkNode("P", nodes.StatusProposed) // proposed — must be excluded

	if _, err := nodes.AddEdge(app, focus.Id, a.Id, "links", ""); err != nil {
		t.Fatalf("edge focus→A: %v", err)
	}
	if _, err := nodes.AddEdge(app, a.Id, b.Id, "links", ""); err != nil {
		t.Fatalf("edge A→B: %v", err)
	}
	if _, err := nodes.AddEdge(app, c.Id, focus.Id, "links", ""); err != nil {
		t.Fatalf("edge C→focus: %v", err)
	}
	// A → P: proposed node must not appear, and the A→P link must be dropped.
	if _, err := nodes.AddEdge(app, a.Id, p.Id, "links", ""); err != nil {
		t.Fatalf("edge A→P: %v", err)
	}

	gd, err := buildGraphData(app, focus.Id, 2)
	if err != nil {
		t.Fatalf("buildGraphData: %v", err)
	}

	byNodeID := map[string]bool{}
	for _, n := range gd.Nodes {
		byNodeID[n.ID] = true
	}
	hasLink := func(s, tg string) bool {
		for _, l := range gd.Links {
			if l.Source == s && l.Target == tg {
				return true
			}
		}
		return false
	}

	if !byNodeID[focus.Id] {
		t.Error("focus missing from graph")
	}
	if !byNodeID[a.Id] {
		t.Error("A (depth-1 outbound) missing from graph")
	}
	if !byNodeID[b.Id] {
		t.Error("B (depth-2 outbound via A) missing from graph")
	}
	if !byNodeID[c.Id] {
		t.Error("C (depth-1 inbound) missing from graph")
	}
	if byNodeID[p.Id] {
		t.Error("consent breach: proposed node P leaked into the graph")
	}
	if len(gd.Nodes) != 4 {
		t.Errorf("node count = %d, want 4 (focus, A, B, C)", len(gd.Nodes))
	}
	if !hasLink(focus.Id, a.Id) {
		t.Error("focus→A link missing")
	}
	if !hasLink(a.Id, b.Id) {
		t.Error("A→B link missing")
	}
	if !hasLink(c.Id, focus.Id) {
		t.Error("C→focus link missing")
	}
	if hasLink(a.Id, p.Id) {
		t.Error("dangling link A→P present (proposed endpoint not in graph)")
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
