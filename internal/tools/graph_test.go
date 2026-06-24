package tools

import (
	"context"
	"strings"
	"testing"

	"github.com/pocketbase/dbx"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/storetest"
)

// --- node_link ---

func TestNodeLinkCreatesEdge(t *testing.T) {
	app := storetest.NewApp(t)
	a, _ := nodes.Create(app, "note", "Alpha", "", nodes.StatusActive, nil)
	b, _ := nodes.Create(app, "note", "Beta", "", nodes.StatusActive, nil)

	tool := nodeLinkTool(app)
	got, err := tool.Execute(context.Background(),
		`{"source":"`+a.Id+`","target":"`+b.Id+`","relation":"relates_to"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(got, "Alpha") || !strings.Contains(got, "Beta") {
		t.Errorf("unexpected result: %q", got)
	}
	if n, _ := app.CountRecords("edges"); n != 1 {
		t.Errorf("edges count = %d, want 1", n)
	}
}

func TestNodeLinkIdempotent(t *testing.T) {
	app := storetest.NewApp(t)
	a, _ := nodes.Create(app, "note", "A", "", nodes.StatusActive, nil)
	b, _ := nodes.Create(app, "note", "B", "", nodes.StatusActive, nil)

	tool := nodeLinkTool(app)
	args := `{"source":"` + a.Id + `","target":"` + b.Id + `","relation":"relates_to"}`

	if _, err := tool.Execute(context.Background(), args); err != nil {
		t.Fatalf("first call: %v", err)
	}
	if _, err := tool.Execute(context.Background(), args); err != nil {
		t.Fatalf("second (idempotent) call: %v", err)
	}
	if n, _ := app.CountRecords("edges"); n != 1 {
		t.Errorf("edges count = %d after two identical calls, want 1", n)
	}
}

func TestNodeLinkRejectsMissingTarget(t *testing.T) {
	app := storetest.NewApp(t)
	a, _ := nodes.Create(app, "note", "A", "", nodes.StatusActive, nil)

	tool := nodeLinkTool(app)
	_, err := tool.Execute(context.Background(),
		`{"source":"`+a.Id+`","target":"nonexistentid","relation":"relates_to"}`)
	if err == nil {
		t.Fatal("expected error for missing target, got nil")
	}
}

func TestNodeLinkRejectsProposedTarget(t *testing.T) {
	app := storetest.NewApp(t)
	a, _ := nodes.Create(app, "note", "A", "", nodes.StatusActive, nil)
	p, _ := nodes.Create(app, "note", "Proposed", "", nodes.StatusProposed, nil)

	tool := nodeLinkTool(app)
	_, err := tool.Execute(context.Background(),
		`{"source":"`+a.Id+`","target":"`+p.Id+`","relation":"relates_to"}`)
	if err == nil {
		t.Fatal("expected error for proposed target, got nil")
	}
	if !strings.Contains(err.Error(), "not active") {
		t.Errorf("error should mention 'not active', got: %v", err)
	}
}

// --- node_related ---

func TestNodeRelatedBothDirections(t *testing.T) {
	app := storetest.NewApp(t)
	hub, _ := nodes.Create(app, "note", "Hub", "", nodes.StatusActive, nil)
	out1, _ := nodes.Create(app, "note", "Out1", "", nodes.StatusActive, nil)
	in1, _ := nodes.Create(app, "note", "In1", "", nodes.StatusActive, nil)

	nodes.AddEdge(app, hub.Id, out1.Id, "relates_to", "")
	nodes.AddEdge(app, in1.Id, hub.Id, "relates_to", "")

	tool := nodeRelatedTool(app)
	got, err := tool.Execute(context.Background(), `{"id":"`+hub.Id+`","direction":"both"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(got, "Out1") || !strings.Contains(got, "In1") {
		t.Errorf("expected both neighbours, got: %q", got)
	}
}

func TestNodeRelatedOutboundOnly(t *testing.T) {
	app := storetest.NewApp(t)
	hub, _ := nodes.Create(app, "note", "Hub", "", nodes.StatusActive, nil)
	out1, _ := nodes.Create(app, "note", "Out1", "", nodes.StatusActive, nil)
	in1, _ := nodes.Create(app, "note", "In1", "", nodes.StatusActive, nil)

	nodes.AddEdge(app, hub.Id, out1.Id, "relates_to", "")
	nodes.AddEdge(app, in1.Id, hub.Id, "relates_to", "")

	tool := nodeRelatedTool(app)
	got, err := tool.Execute(context.Background(), `{"id":"`+hub.Id+`","direction":"out"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(got, "Out1") {
		t.Errorf("expected Out1 in outbound result, got: %q", got)
	}
	if strings.Contains(got, "In1") {
		t.Errorf("In1 should not appear in outbound-only result, got: %q", got)
	}
}

func TestNodeRelatedEmptyResult(t *testing.T) {
	app := storetest.NewApp(t)
	solo, _ := nodes.Create(app, "note", "Alone", "", nodes.StatusActive, nil)

	tool := nodeRelatedTool(app)
	got, err := tool.Execute(context.Background(), `{"id":"`+solo.Id+`"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got != "No related active nodes." {
		t.Errorf("expected empty message, got: %q", got)
	}
}

// --- node_query ---

func TestNodeQueryToolByType(t *testing.T) {
	app := storetest.NewApp(t)
	nodes.Create(app, "note", "Note1", "", nodes.StatusActive, nil)
	nodes.Create(app, "note", "Note2", "", nodes.StatusActive, nil)
	// proposed must not appear
	nodes.Create(app, "note", "Proposed", "", nodes.StatusProposed, nil)

	tool := nodeQueryTool(app)
	got, err := tool.Execute(context.Background(), `{"type":"note"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(got, "Note1") || !strings.Contains(got, "Note2") {
		t.Errorf("expected both notes, got: %q", got)
	}
	if strings.Contains(got, "Proposed") {
		t.Errorf("proposed node must not appear, got: %q", got)
	}
}

func TestNodeQueryToolNeverReturnsProposed(t *testing.T) {
	app := storetest.NewApp(t)
	nodes.Create(app, "note", "Active", "", nodes.StatusActive, nil)
	nodes.Create(app, "note", "Proposed", "", nodes.StatusProposed, nil)

	tool := nodeQueryTool(app)
	got, err := tool.Execute(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Contains(got, "Proposed") {
		t.Errorf("proposed node must not appear in query results, got: %q", got)
	}
}

func TestNodeQueryToolLimit(t *testing.T) {
	app := storetest.NewApp(t)
	for i := range 5 {
		nodes.Create(app, "note", "N"+string(rune('A'+i)), "", nodes.StatusActive, nil)
	}

	tool := nodeQueryTool(app)
	got, err := tool.Execute(context.Background(), `{"type":"note","limit":2}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// Each result line contains "(id <id>)"; count them.
	lines := strings.Split(strings.TrimSpace(got), "\n")
	if len(lines) != 2 {
		t.Errorf("expected 2 result lines (limit=2), got %d: %q", len(lines), got)
	}
}

// --- node_schema ---

func TestNodeSchemaToolListsTypes(t *testing.T) {
	app := storetest.NewApp(t)

	tool := nodeSchemaTool(app)
	got, err := tool.Execute(context.Background(), `{}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// The test DB (storetest) seeds all registered types; at minimum "note" must appear.
	if !strings.Contains(got, "note") {
		t.Errorf("expected 'note' in schema output, got: %q", got)
	}
}

func TestNodeSchemaToolSingleType(t *testing.T) {
	app := storetest.NewApp(t)

	// memory has a schema (category, importance) seeded in migrations.
	tool := nodeSchemaTool(app)
	got, err := tool.Execute(context.Background(), `{"type":"memory"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// Should report props from the live registry, not a hardcoded map.
	if !strings.Contains(got, "memory") {
		t.Errorf("expected 'memory' in output, got: %q", got)
	}
}

// --- node_get enrichment ---

func TestNodeGetIncludesPropsAndLinkSummary(t *testing.T) {
	app := storetest.NewApp(t)
	a, _ := nodes.Create(app, "memory", "Cat fact", "Cats are great.", nodes.StatusActive,
		map[string]any{"category": "fact", "importance": 5})
	b, _ := nodes.Create(app, "note", "B", "", nodes.StatusActive, nil)
	nodes.AddEdge(app, a.Id, b.Id, "relates_to", "")

	tool := nodeGetTool(app)
	got, err := tool.Execute(context.Background(), `{"id":"`+a.Id+`"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(got, "Cat fact") {
		t.Errorf("missing title in output: %q", got)
	}
	// Props should appear.
	if !strings.Contains(got, "category") {
		t.Errorf("missing props in output: %q", got)
	}
	// Link summary.
	if !strings.Contains(got, "outbound") {
		t.Errorf("missing link summary in output: %q", got)
	}
}

// --- KnowledgeTools includes graph tools ---

func TestKnowledgeToolsIncludesGraphVerbs(t *testing.T) {
	app := storetest.NewApp(t)
	ts := KnowledgeTools(app)
	want := map[string]bool{
		"node_link":    false,
		"node_related": false,
		"node_query":   false,
		"node_schema":  false,
	}
	for _, tool := range ts {
		if _, ok := want[tool.Spec.Name]; ok {
			want[tool.Spec.Name] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("KnowledgeTools missing tool %q", name)
		}
	}
}

// --- node_link defaults to relates_to ---

func TestNodeLinkDefaultRelation(t *testing.T) {
	app := storetest.NewApp(t)
	a, _ := nodes.Create(app, "note", "A", "", nodes.StatusActive, nil)
	b, _ := nodes.Create(app, "note", "B", "", nodes.StatusActive, nil)

	tool := nodeLinkTool(app)
	// No relation specified — should default to relates_to.
	_, err := tool.Execute(context.Background(),
		`{"source":"`+a.Id+`","target":"`+b.Id+`"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	edge, err := app.FindFirstRecordByFilter("edges",
		"source = {:s} && target = {:t}",
		dbx.Params{"s": a.Id, "t": b.Id})
	if err != nil {
		t.Fatalf("FindFirstRecordByFilter: %v", err)
	}
	if edge.GetString("type") != "relates_to" {
		t.Errorf("default relation = %q, want relates_to", edge.GetString("type"))
	}
}
