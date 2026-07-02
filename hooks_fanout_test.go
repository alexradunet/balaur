package main

// Integration tests for the node-save hook fan-out skip (plan 244): a
// metadata-only save (knowledge.Touch use_count bumps, the task nudge mark)
// must not re-sync wikilink edges or re-index into FTS, while a save that
// genuinely changes an indexed/linked field must still do both.

import (
	"slices"
	"sort"
	"testing"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/knowledge"
	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/search"
	"github.com/alexradunet/balaur/internal/storetest"
)

// sourceEdgeIDs returns the sorted edge ids for source = id, type = "links".
func sourceEdgeIDs(t *testing.T, app core.App, id string) []string {
	t.Helper()
	edges, err := app.FindRecordsByFilter("edges",
		"source = {:s} && type = {:t}", "", 0, 0,
		dbx.Params{"s": id, "t": nodes.DefaultEdgeType})
	if err != nil {
		t.Fatalf("load edges: %v", err)
	}
	ids := make([]string, 0, len(edges))
	for _, e := range edges {
		ids = append(ids, e.Id)
	}
	sort.Strings(ids)
	return ids
}

// edgeCreateAuditCount counts audit_log rows with action = 'edge.create'.
func edgeCreateAuditCount(t *testing.T, app core.App) int {
	t.Helper()
	rows, err := app.FindRecordsByFilter("audit_log", "action = 'edge.create'", "", 0, 0, nil)
	if err != nil {
		t.Fatalf("count edge.create audit rows: %v", err)
	}
	return len(rows)
}

func TestTouchSkipsEdgeAndIndexFanout(t *testing.T) {
	app := storetest.NewApp(t)
	registerSearchIndex(app)
	registerGraphLinks(app)

	id, err := nodes.Create(app, "memory", "Touch source", "knows [[Alpha]] and [[Beta]]",
		nodes.StatusActive, map[string]any{"when_to_use": "greeting", "importance": 4})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	before := sourceEdgeIDs(t, app, id.Id)
	if len(before) != 2 {
		t.Fatalf("expected 2 edges after create, got %d: %v", len(before), before)
	}
	n0 := edgeCreateAuditCount(t, app)

	// Re-fetch fresh so Original() reflects DB state, mirroring production.
	rec, err := app.FindRecordById("nodes", id.Id)
	if err != nil {
		t.Fatalf("re-fetch: %v", err)
	}
	knowledge.Touch(app, knowledge.Memory, rec)

	after := sourceEdgeIDs(t, app, id.Id)
	if !slices.Equal(before, after) {
		t.Fatalf("edge ids churned: before=%v after=%v", before, after)
	}
	n1 := edgeCreateAuditCount(t, app)
	if n1 != n0 {
		t.Fatalf("phantom edge.create audit rows appeared: before=%d after=%d", n0, n1)
	}
}

func TestBodyEditResyncsEdges(t *testing.T) {
	app := storetest.NewApp(t)
	registerSearchIndex(app)
	registerGraphLinks(app)

	id, err := nodes.Create(app, "memory", "Body edit source", "knows [[Alpha]] and [[Beta]]",
		nodes.StatusActive, map[string]any{"when_to_use": "greeting", "importance": 4})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	n0 := edgeCreateAuditCount(t, app)

	rec, err := app.FindRecordById("nodes", id.Id)
	if err != nil {
		t.Fatalf("re-fetch: %v", err)
	}
	rec.Set("body", "knows [[Alpha]] and [[Beta]] and [[Gamma]]")
	if err := app.Save(rec); err != nil {
		t.Fatalf("save body edit: %v", err)
	}

	gamma, err := app.FindFirstRecordByFilter("nodes", "title = 'Gamma'")
	if err != nil {
		t.Fatalf("expected Gamma node to be created: %v", err)
	}
	edges, err := app.FindRecordsByFilter("edges",
		"source = {:s} && target = {:t} && type = {:y}", "", 0, 0,
		dbx.Params{"s": id.Id, "t": gamma.Id, "y": nodes.DefaultEdgeType})
	if err != nil {
		t.Fatalf("load edge to Gamma: %v", err)
	}
	if len(edges) != 1 {
		t.Fatalf("expected an edge to Gamma, got %d", len(edges))
	}
	n1 := edgeCreateAuditCount(t, app)
	if n1 <= n0 {
		t.Fatalf("expected edge.create audit rows to increase: before=%d after=%d", n0, n1)
	}
}

func TestStatusArchiveDropsFromIndex(t *testing.T) {
	app := storetest.NewApp(t)
	registerSearchIndex(app)
	registerGraphLinks(app)

	raw, ok := app.Store().GetOk(search.StoreKey)
	if !ok {
		t.Fatal("search index not registered in app.Store()")
	}
	ix, ok := raw.(*search.Index)
	if !ok || ix == nil {
		t.Fatal("search index has unexpected type or is nil")
	}

	id, err := nodes.Create(app, "note", "Archive source", "xylophonequartz lattice",
		nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	ids, err := ix.Query([]string{"xylophonequartz"}, 10)
	if err != nil {
		t.Fatalf("Query before archive: %v", err)
	}
	if len(ids) != 1 || ids[0] != id.Id {
		t.Fatalf("expected node in index before archive, got %v", ids)
	}

	if _, err := nodes.Transition(app, id.Id, nodes.StatusArchived, "node"); err != nil {
		t.Fatalf("Transition to archived: %v", err)
	}

	ids2, err := ix.Query([]string{"xylophonequartz"}, 10)
	if err != nil {
		t.Fatalf("Query after archive: %v", err)
	}
	if len(ids2) != 0 {
		t.Fatalf("expected node removed from index after archive, got %v", ids2)
	}
}

func TestTitleEditReindexes(t *testing.T) {
	app := storetest.NewApp(t)
	registerSearchIndex(app)
	registerGraphLinks(app)

	raw, ok := app.Store().GetOk(search.StoreKey)
	if !ok {
		t.Fatal("search index not registered in app.Store()")
	}
	ix, ok := raw.(*search.Index)
	if !ok || ix == nil {
		t.Fatal("search index has unexpected type or is nil")
	}

	id, err := nodes.Create(app, "note", "Old Title", "some body text",
		nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	rec, err := app.FindRecordById("nodes", id.Id)
	if err != nil {
		t.Fatalf("re-fetch: %v", err)
	}
	rec.Set("title", "Zanzibarunique")
	if err := app.Save(rec); err != nil {
		t.Fatalf("save title edit: %v", err)
	}

	ids, err := ix.Query([]string{"Zanzibarunique"}, 10)
	if err != nil {
		t.Fatalf("Query after title edit: %v", err)
	}
	if len(ids) != 1 || ids[0] != id.Id {
		t.Fatalf("expected re-indexed title to be searchable, got %v", ids)
	}
}
