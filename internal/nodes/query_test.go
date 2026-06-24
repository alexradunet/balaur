package nodes_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/storetest"
)

func TestQueryByType(t *testing.T) {
	app := storetest.NewApp(t)

	nodes.Create(app, "note", "Note A", "", nodes.StatusActive, nil)
	nodes.Create(app, "note", "Note B", "", nodes.StatusActive, nil)
	// proposed note must NOT appear.
	nodes.Create(app, "note", "Proposed", "", nodes.StatusProposed, nil)
	// different type must NOT appear.
	nodes.Create(app, "memory", "Mem", "body", nodes.StatusActive,
		map[string]any{"category": "fact", "importance": 3})

	recs, err := nodes.Query(app, nodes.QueryOpts{Type: "note"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("want 2 active note nodes, got %d", len(recs))
	}
	for _, r := range recs {
		if r.GetString("type") != "note" {
			t.Errorf("unexpected type %q", r.GetString("type"))
		}
		if r.GetString("status") != nodes.StatusActive {
			t.Errorf("non-active node returned: status=%q", r.GetString("status"))
		}
	}
}

func TestQueryAnyType(t *testing.T) {
	app := storetest.NewApp(t)

	nodes.Create(app, "note", "N1", "", nodes.StatusActive, nil)
	nodes.Create(app, "memory", "M1", "b", nodes.StatusActive,
		map[string]any{"category": "fact", "importance": 3})
	// proposed must not appear.
	nodes.Create(app, "note", "NP", "", nodes.StatusProposed, nil)

	recs, err := nodes.Query(app, nodes.QueryOpts{})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(recs) != 2 {
		t.Fatalf("want 2 active nodes (any type), got %d", len(recs))
	}
	for _, r := range recs {
		if r.GetString("status") != nodes.StatusActive {
			t.Errorf("non-active node returned")
		}
	}
}

func TestQueryPropMatch(t *testing.T) {
	app := storetest.NewApp(t)

	nodes.Create(app, "memory", "About tea", "Black tea.", nodes.StatusActive,
		map[string]any{"category": "preference", "importance": 3})
	nodes.Create(app, "memory", "About coffee", "Dark roast.", nodes.StatusActive,
		map[string]any{"category": "fact", "importance": 2})
	// proposed — must not appear.
	nodes.Create(app, "memory", "Proposed pref", "p", nodes.StatusProposed,
		map[string]any{"category": "preference", "importance": 1})

	recs, err := nodes.Query(app, nodes.QueryOpts{
		Type:      "memory",
		PropMatch: map[string]string{"category": "prefer"},
	})
	if err != nil {
		t.Fatalf("Query PropMatch: %v", err)
	}
	if len(recs) != 1 {
		t.Fatalf("want 1 match, got %d", len(recs))
	}
	if nodes.PropString(recs[0], "category") != "preference" {
		t.Errorf("wrong record returned")
	}
}

func TestQueryLimit(t *testing.T) {
	app := storetest.NewApp(t)

	for i := range 5 {
		nodes.Create(app, "note", "N"+string(rune('A'+i)), "", nodes.StatusActive, nil)
	}

	recs, err := nodes.Query(app, nodes.QueryOpts{Type: "note", Limit: 3})
	if err != nil {
		t.Fatalf("Query Limit: %v", err)
	}
	if len(recs) != 3 {
		t.Fatalf("want 3 (limit), got %d", len(recs))
	}
}

func TestInverseLabel(t *testing.T) {
	cases := []struct {
		relType string
		want    string
	}{
		{"links", "linked from"},
		{"relates_to", "relates to"},
		{"part_of", "has part"},
		{"about", "referenced by"},
		{"unknown_type", "linked from"}, // fallback
	}
	for _, c := range cases {
		if got := nodes.InverseLabel(c.relType); got != c.want {
			t.Errorf("InverseLabel(%q) = %q, want %q", c.relType, got, c.want)
		}
	}
}
