package nodes_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/store"
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
	// Use note (empty schema) — this test is about lifecycle transitions, not memory props.
	rec, err := nodes.Create(app, "note", "x", "", nodes.StatusProposed, nil)
	if err != nil {
		t.Fatal(err)
	}

	// proposed → archived is forbidden.
	if _, err := nodes.Transition(app, rec.Id, nodes.StatusArchived, "node"); err == nil {
		t.Error("proposed → archived should be rejected")
	}
	// proposed → active is allowed.
	if _, err := nodes.Transition(app, rec.Id, nodes.StatusActive, "node"); err != nil {
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
	// Use note (empty schema) — this test is about graph traversal, not memory props.
	proposed, _ := nodes.Create(app, "note", "Proposed", "", nodes.StatusProposed, nil)

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

func strptr(s string) *string { return &s }

func TestUpdateValidatesAndAudits(t *testing.T) {
	app := storetest.NewApp(t)
	rec, err := nodes.Create(app, "book", "Old title", "", nodes.StatusActive,
		map[string]any{"author": "A", "year": 1969.0})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	got, err := nodes.Update(app, rec.Id, strptr("New title"), nil,
		map[string]any{"author": "B", "year": 1970.0})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if got.GetString("title") != "New title" {
		t.Errorf("title = %q, want New title", got.GetString("title"))
	}
	if a := nodes.PropString(got, "author"); a != "B" {
		t.Errorf("PropString(author) = %q, want B", a)
	}

	audits, _ := store.ListAudit(app, "node.update", "owner", 10)
	if len(audits) == 0 {
		t.Fatal("expected a node.update audit row")
	}
}

func TestUpdateRejectsNonActive(t *testing.T) {
	app := storetest.NewApp(t)
	rec, err := nodes.Create(app, "note", "Draft", "", nodes.StatusProposed, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_, err = nodes.Update(app, rec.Id, strptr("New"), nil, nil)
	if err == nil || !strings.Contains(err.Error(), "not active") {
		t.Fatalf("Update on a proposed node: err = %v, want a 'not active' error", err)
	}
}

func TestUpdateRejectsInvalidProps(t *testing.T) {
	app := storetest.NewApp(t)
	rec, err := nodes.Create(app, "book", "X", "", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	_, err = nodes.Update(app, rec.Id, nil, nil, map[string]any{"year": "not-a-number"})
	if err == nil || !strings.Contains(err.Error(), "invalid props") {
		t.Fatalf("Update with a bad prop type: err = %v, want an 'invalid props' error", err)
	}
}

// TestUpdateRejectsConsentGatedTypes is the consent backstop: memory and skill
// are NOT owner-authored, so they must never be editable in place via Update —
// those changes go through propose_edit. (A memory/skill node is born proposed,
// but even an active one must be refused by the type guard.)
func TestUpdateRejectsConsentGatedTypes(t *testing.T) {
	app := storetest.NewApp(t)
	// memory's importance prop is Required, so supply valid props at create time;
	// skill has no required props. Both are consent-gated (born proposed), so
	// create them ACTIVE here to prove the type guard refuses them even when the
	// status backstop would not.
	cases := []struct {
		typ   string
		props map[string]any
	}{
		{"memory", map[string]any{"importance": 3.0}},
		{"skill", nil},
	}
	for _, tc := range cases {
		rec, err := nodes.Create(app, tc.typ, "Gated "+tc.typ, "body", nodes.StatusActive, tc.props)
		if err != nil {
			t.Fatalf("Create %s: %v", tc.typ, err)
		}
		_, err = nodes.Update(app, rec.Id, strptr("Hacked"), nil, nil)
		if err == nil || !strings.Contains(err.Error(), "not owner-authored") {
			t.Fatalf("Update on a %s node: err = %v, want a 'not owner-authored' rejection", tc.typ, err)
		}
	}
}
