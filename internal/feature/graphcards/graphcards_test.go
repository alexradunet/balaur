package graphcards

// graphcards_test.go — the layering marker, the related-computation test (with
// the consent regression: a proposed neighbor never appears), and the SVG render
// smoke test. Internal test package (package graphcards) so it can drive the
// unexported buildRelated/buildGraph directly, the way the live cards do.

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/storetest"
)

// TestNoWebImports is a compile-time fact, mirroring the other feature packages:
// a feature package must never import internal/web (the layering law, spec §4.1).
func TestNoWebImports(t *testing.T) {
	t.Log("compile-time verified: internal/feature/graphcards has no internal/web imports")
}

func TestRelatedComputation(t *testing.T) {
	app := storetest.NewApp(t)

	a, err := nodes.Create(app, "note", "Greenhouse plan", "", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	b, err := nodes.Create(app, "note", "Seed list", "", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("create B: %v", err)
	}
	c, err := nodes.Create(app, "note", "Spring tasks", "", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("create C: %v", err)
	}
	// Use note (empty schema) — this test is about consent graph filtering, not memory props.
	d, err := nodes.Create(app, "note", "Draft idea", "", nodes.StatusProposed, nil)
	if err != nil {
		t.Fatalf("create D: %v", err)
	}

	// B → A (backlink of A), A → C (outbound of A), D → A (proposed backlink).
	if _, err := nodes.AddEdge(app, b.Id, a.Id, "links", ""); err != nil {
		t.Fatalf("edge B→A: %v", err)
	}
	if _, err := nodes.AddEdge(app, a.Id, c.Id, "links", ""); err != nil {
		t.Fatalf("edge A→C: %v", err)
	}
	if _, err := nodes.AddEdge(app, d.Id, a.Id, "links", ""); err != nil {
		t.Fatalf("edge D→A: %v", err)
	}

	v := buildRelated(app, map[string]string{"id": a.Id})

	got := map[string]string{} // id → rel
	for _, row := range v.Rows {
		got[row.ID] = row.Rel
	}
	if _, ok := got[b.Id]; !ok {
		t.Errorf("backlink node B (%s) missing from related list; rows=%+v", b.Id, v.Rows)
	}
	if _, ok := got[c.Id]; !ok {
		t.Errorf("outbound node C (%s) missing from related list; rows=%+v", c.Id, v.Rows)
	}
	if _, ok := got[d.Id]; ok {
		t.Errorf("proposed node D (%s) leaked into related list (consent filter broken)", d.Id)
	}
	if _, ok := got[a.Id]; ok {
		t.Errorf("focus node A (%s) appears in its own related list", a.Id)
	}
}

func TestGraphCardRendersSVG(t *testing.T) {
	app := storetest.NewApp(t)

	a, err := nodes.Create(app, "note", "Greenhouse plan", "", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("create A: %v", err)
	}
	b, err := nodes.Create(app, "note", "Seed list", "", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("create B: %v", err)
	}
	// A proposed neighbor that must never reach the SVG.
	// Use note (empty schema) — this test is about consent graph filtering, not memory props.
	d, err := nodes.Create(app, "note", "Secret draft", "", nodes.StatusProposed, nil)
	if err != nil {
		t.Fatalf("create D: %v", err)
	}
	if _, err := nodes.AddEdge(app, a.Id, b.Id, "links", ""); err != nil {
		t.Fatalf("edge A→B: %v", err)
	}
	if _, err := nodes.AddEdge(app, d.Id, a.Id, "links", ""); err != nil {
		t.Fatalf("edge D→A: %v", err)
	}

	var sb strings.Builder
	if err := GraphCard(buildGraph(app, map[string]string{"id": a.Id})).Render(&sb); err != nil {
		t.Fatalf("render: %v", err)
	}
	out := sb.String()
	for _, want := range []string{"<svg", "<circle", "<line"} {
		if !strings.Contains(out, want) {
			t.Errorf("graph SVG missing %q\n%s", want, out)
		}
	}
	if !strings.Contains(out, "Greenhouse plan") {
		t.Errorf("focus title absent from graph SVG\n%s", out)
	}
	if strings.Contains(out, "Secret draft") {
		t.Errorf("proposed node title leaked into graph SVG (consent filter broken)\n%s", out)
	}

	// Empty neighborhood: the focus dot renders (one circle), no edge line.
	lone, err := nodes.Create(app, "note", "Lonely node", "", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("create lone: %v", err)
	}
	var eb strings.Builder
	if err := GraphCard(buildGraph(app, map[string]string{"id": lone.Id})).Render(&eb); err != nil {
		t.Fatalf("render lone: %v", err)
	}
	empty := eb.String()
	if !strings.Contains(empty, "<svg") || !strings.Contains(empty, "<circle") {
		t.Errorf("empty-neighborhood graph missing svg/circle\n%s", empty)
	}
	if strings.Contains(empty, "<line") {
		t.Errorf("empty-neighborhood graph drew an edge line\n%s", empty)
	}
}
