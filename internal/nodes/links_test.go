package nodes_test

import (
	"slices"
	"testing"

	"github.com/pocketbase/dbx"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/storetest"
)

func TestParseLinks(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{"empty", "", nil},
		{"empty target", "[[]]", nil},
		{"whitespace target", "[[   ]]", nil},
		{"single", "[[Alpha]]", []string{"Alpha"}},
		{"alias ignored", "[[Alpha|the first]]", []string{"Alpha"}},
		{"adjacent", "[[a]][[b]]", []string{"a", "b"}},
		{"dedup", "[[a]] and [[a]] again", []string{"a"}},
		{"case-insensitive dedup, first-seen wins", "[[Alpha]] [[alpha]]", []string{"Alpha"}},
		{"unicode", "[[Café]]", []string{"Café"}},
		// The [^\[\]|] class forbids ']' inside the target. For "[[a]b]]" the
		// target group matches "a" but the required closing "]]" is not at that
		// position ("]b" follows), and no other start position yields a valid
		// [[...]] — so the observed behavior is no match at all (nil).
		{"bracket inside is not a target", "[[a]b]]", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nodes.ParseLinks(tt.body)
			if !slices.Equal(got, tt.want) {
				t.Errorf("ParseLinks(%q) = %v, want %v", tt.body, got, tt.want)
			}
		})
	}
}

func TestSyncLinksResolveAndStub(t *testing.T) {
	app := storetest.NewApp(t)
	target, _ := nodes.Create(app, "note", "Target", "", nodes.StatusActive, nil)
	source, err := nodes.Create(app, "note", "Source", "see [[Target]] and [[Ghost]]", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("Create source: %v", err)
	}
	if err := nodes.SyncLinks(app, source); err != nil {
		t.Fatalf("SyncLinks: %v", err)
	}

	edges, err := app.FindRecordsByFilter("edges",
		"source = {:s} && type = {:t}", "", 0, 0,
		dbx.Params{"s": source.Id, "t": nodes.DefaultEdgeType})
	if err != nil {
		t.Fatalf("load edges: %v", err)
	}
	if len(edges) != 2 {
		t.Fatalf("edge count = %d, want 2", len(edges))
	}

	// "Ghost" became an active note stub.
	ghost, err := app.FindFirstRecordByFilter("nodes",
		"status = 'active' && title = {:t}", dbx.Params{"t": "Ghost"})
	if err != nil {
		t.Fatalf("Ghost stub not created: %v", err)
	}
	if ghost.GetString("type") != "note" {
		t.Errorf("Ghost stub type = %q, want note", ghost.GetString("type"))
	}

	// The edge set targets exactly {Target, Ghost}.
	gotTargets := map[string]bool{}
	for _, e := range edges {
		gotTargets[e.GetString("target")] = true
	}
	if !gotTargets[target.Id] || !gotTargets[ghost.Id] {
		t.Errorf("edge targets = %v, want {%s,%s}", gotTargets, target.Id, ghost.Id)
	}
}

func TestSyncLinksIdempotent(t *testing.T) {
	app := storetest.NewApp(t)
	source, _ := nodes.Create(app, "note", "Source", "see [[Target]] and [[Ghost]]", nodes.StatusActive, nil)
	if err := nodes.SyncLinks(app, source); err != nil {
		t.Fatal(err)
	}
	edges1, _ := app.CountRecords("edges")
	nodes1, _ := app.CountRecords("nodes")
	if err := nodes.SyncLinks(app, source); err != nil {
		t.Fatal(err)
	}
	edges2, _ := app.CountRecords("edges")
	nodes2, _ := app.CountRecords("nodes")
	if edges1 != edges2 {
		t.Errorf("edge count changed on re-sync: %d → %d", edges1, edges2)
	}
	if nodes1 != nodes2 {
		t.Errorf("node count changed on re-sync (extra stub): %d → %d", nodes1, nodes2)
	}
}

func TestSyncLinksRewrites(t *testing.T) {
	app := storetest.NewApp(t)
	a, _ := nodes.Create(app, "note", "A", "", nodes.StatusActive, nil)
	b, _ := nodes.Create(app, "note", "B", "", nodes.StatusActive, nil)
	source, _ := nodes.Create(app, "note", "Source", "[[A]]", nodes.StatusActive, nil)
	if err := nodes.SyncLinks(app, source); err != nil {
		t.Fatal(err)
	}
	// Change body to link B instead of A, re-sync.
	source.Set("body", "[[B]]")
	if err := nodes.SyncLinks(app, source); err != nil {
		t.Fatal(err)
	}
	edges, _ := app.FindRecordsByFilter("edges",
		"source = {:s} && type = {:t}", "", 0, 0,
		dbx.Params{"s": source.Id, "t": nodes.DefaultEdgeType})
	if len(edges) != 1 {
		t.Fatalf("edge count = %d, want 1 after rewrite", len(edges))
	}
	if edges[0].GetString("target") != b.Id {
		t.Errorf("edge target = %q, want B (%s); A (%s) edge should be gone",
			edges[0].GetString("target"), b.Id, a.Id)
	}
}

func TestSyncLinksNoSelfEdge(t *testing.T) {
	app := storetest.NewApp(t)
	self, _ := nodes.Create(app, "note", "Self", "[[Self]]", nodes.StatusActive, nil)
	if err := nodes.SyncLinks(app, self); err != nil {
		t.Fatal(err)
	}
	edges, _ := app.FindRecordsByFilter("edges",
		"source = {:s}", "", 0, 0, dbx.Params{"s": self.Id})
	if len(edges) != 0 {
		t.Errorf("self-link produced %d edges, want 0", len(edges))
	}
}

func TestBacklinksAndOutbound(t *testing.T) {
	app := storetest.NewApp(t)
	y, _ := nodes.Create(app, "note", "Y", "", nodes.StatusActive, nil)
	x, _ := nodes.Create(app, "note", "X", "[[Y]]", nodes.StatusActive, nil)
	if err := nodes.SyncLinks(app, x); err != nil {
		t.Fatal(err)
	}
	back, err := nodes.Backlinks(app, y.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(back) != 1 || back[0].Id != x.Id {
		t.Errorf("Backlinks(Y) = %d nodes, want [X]", len(back))
	}
	out, err := nodes.Outbound(app, x.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Id != y.Id {
		t.Errorf("Outbound(X) = %d nodes, want [Y]", len(out))
	}
}

func TestProposedNodesNeverResolve(t *testing.T) {
	app := storetest.NewApp(t)
	// Use type=note (empty schema) — this test is about link resolution, not memory props.
	hidden, _ := nodes.Create(app, "note", "Hidden", "", nodes.StatusProposed, nil)
	source, _ := nodes.Create(app, "note", "Source", "[[Hidden]]", nodes.StatusActive, nil)
	if err := nodes.SyncLinks(app, source); err != nil {
		t.Fatal(err)
	}
	// A NEW active stub "Hidden" must have been created; the proposed one is
	// never picked.
	stub, err := app.FindFirstRecordByFilter("nodes",
		"status = 'active' && title = {:t}", dbx.Params{"t": "Hidden"})
	if err != nil {
		t.Fatalf("active Hidden stub not created: %v", err)
	}
	if stub.Id == hidden.Id {
		t.Errorf("resolution picked the proposed node %s; it must stay out of the graph", hidden.Id)
	}
	edges, _ := app.FindRecordsByFilter("edges",
		"source = {:s}", "", 0, 0, dbx.Params{"s": source.Id})
	if len(edges) != 1 || edges[0].GetString("target") != stub.Id {
		t.Errorf("edge should target the active stub %s, not the proposed node", stub.Id)
	}
}
