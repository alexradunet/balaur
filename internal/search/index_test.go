package search

import (
	"path/filepath"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/storetest"
)

// openTestIndex creates a temporary on-disk index for tests.
func openTestIndex(t *testing.T) *Index {
	t.Helper()
	ix, err := Open(filepath.Join(t.TempDir(), "search.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { ix.Close() })
	return ix
}

// seedActiveMemory inserts an active memory directly via the PocketBase app
// (bypassing the knowledge lifecycle layer to avoid an import cycle — after
// Step 2, knowledge imports search, so search_test may not import knowledge).
func seedActiveMemory(t *testing.T, app core.App, title, content, category string, importance int) string {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("memories")
	if err != nil {
		t.Fatalf("find memories collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("title", title)
	rec.Set("content", content)
	rec.Set("category", category)
	rec.Set("importance", importance)
	rec.Set("when_to_use", "")
	rec.Set("source", "test")
	rec.Set("status", "active")
	if err := app.Save(rec); err != nil {
		t.Fatalf("save memory: %v", err)
	}
	return rec.Id
}

// seedProposedMemory inserts a proposed (non-active) memory.
func seedProposedMemory(t *testing.T, app core.App, title string) string {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("memories")
	if err != nil {
		t.Fatalf("find memories collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("title", title)
	rec.Set("content", "")
	rec.Set("category", "fact")
	rec.Set("importance", 1)
	rec.Set("when_to_use", "")
	rec.Set("source", "test")
	rec.Set("status", "proposed")
	if err := app.Save(rec); err != nil {
		t.Fatalf("save proposed memory: %v", err)
	}
	return rec.Id
}

func TestOpenCreatesSchema(t *testing.T) {
	ix := openTestIndex(t)
	// Insert and query to prove the schema exists.
	_, err := ix.db.Exec(`INSERT INTO memories_fts(id, title, content, when_to_use, category) VALUES (?, ?, ?, ?, ?)`,
		"id1", "flour bread", "", "", "")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	ids, err := ix.Query([]string{"flour"}, 10)
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(ids) != 1 || ids[0] != "id1" {
		t.Fatalf("unexpected ids: %v", ids)
	}
}

func TestRebuildAndQuery(t *testing.T) {
	app := storetest.NewApp(t)

	// Seed two active memories and one proposed (should not appear after Rebuild).
	_ = seedActiveMemory(t, app, "flour bread recipe", "knead the flour well", "fact", 2)
	_ = seedActiveMemory(t, app, "forest trail", "walked past the old mill", "preference", 2)
	_ = seedProposedMemory(t, app, "espresso budget")

	ix := openTestIndex(t)
	if err := ix.Rebuild(app); err != nil {
		t.Fatalf("Rebuild: %v", err)
	}

	// "flour" hits at least one active memory.
	ids, err := ix.Query([]string{"flour"}, 10)
	if err != nil {
		t.Fatalf("Query flour: %v", err)
	}
	if len(ids) == 0 {
		t.Fatal("expected at least one match for 'flour'")
	}

	// "espresso" (proposed only) must return zero ids after Rebuild.
	ids2, err := ix.Query([]string{"espresso"}, 10)
	if err != nil {
		t.Fatalf("Query espresso: %v", err)
	}
	if len(ids2) != 0 {
		t.Fatalf("proposed memory appeared in index: %v", ids2)
	}

	// Limit is respected.
	ids3, err := ix.Query([]string{"flour", "forest", "mill", "trail", "bread"}, 1)
	if err != nil {
		t.Fatalf("Query limit: %v", err)
	}
	if len(ids3) > 1 {
		t.Fatalf("limit not respected: got %d ids", len(ids3))
	}
}

func TestUpsertNonActive(t *testing.T) {
	app := storetest.NewApp(t)

	// Seed as active, upsert into index, then upsert with archived status.
	id := seedActiveMemory(t, app, "archivable memory", "some archivable content", "fact", 2)

	activeRec, err := app.FindRecordById("memories", id)
	if err != nil {
		t.Fatalf("find record: %v", err)
	}

	ix := openTestIndex(t)
	if err := ix.Upsert(activeRec); err != nil {
		t.Fatalf("Upsert active: %v", err)
	}
	idsActive, err := ix.Query([]string{"archivable"}, 10)
	if err != nil {
		t.Fatalf("Query before archive: %v", err)
	}
	if len(idsActive) == 0 {
		t.Fatal("expected active memory in index")
	}

	// Mutate the in-memory record to archived (without saving to PocketBase —
	// we only need the status field to drive Upsert's deletion branch).
	activeRec.Set("status", "archived")
	if err := ix.Upsert(activeRec); err != nil {
		t.Fatalf("Upsert archived: %v", err)
	}
	idsArchived, err := ix.Query([]string{"archivable"}, 10)
	if err != nil {
		t.Fatalf("Query after archive: %v", err)
	}
	if len(idsArchived) != 0 {
		t.Fatalf("archived memory still in index: %v", idsArchived)
	}
}

func TestQueryInjectionSafe(t *testing.T) {
	ix := openTestIndex(t)

	// Seed a benign record.
	_, err := ix.db.Exec(`INSERT INTO memories_fts(id, title, content, when_to_use, category) VALUES (?, ?, ?, ?, ?)`,
		"safe1", "target memory", "safe content", "", "")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	// Attempt to inject FTS5 operators via the term — must not error.
	// The term `flour" OR "x` has an embedded double-quote; our quoting
	// doubles it, so FTS5 sees it as a literal token, not an operator.
	ids, err := ix.Query([]string{`flour" OR "x`}, 10)
	if err != nil {
		t.Fatalf("injection term returned error: %v", err)
	}
	// Should match nothing (the injected token is a literal with a quote).
	_ = ids
}

func TestDeleteRemovesRecord(t *testing.T) {
	ix := openTestIndex(t)

	_, err := ix.db.Exec(`INSERT INTO memories_fts(id, title, content, when_to_use, category) VALUES (?, ?, ?, ?, ?)`,
		"del1", "delete me", "please remove", "", "")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := ix.Delete("del1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	ids, err := ix.Query([]string{"remove"}, 10)
	if err != nil {
		t.Fatalf("Query after delete: %v", err)
	}
	for _, id := range ids {
		if id == "del1" {
			t.Fatal("deleted record still returned by Query")
		}
	}
}

func TestQueryEmpty(t *testing.T) {
	ix := openTestIndex(t)
	ids, err := ix.Query(nil, 10)
	if err != nil {
		t.Fatalf("Query(nil): %v", err)
	}
	if len(ids) != 0 {
		t.Fatalf("empty query returned ids: %v", ids)
	}
	ids2, err := ix.Query([]string{}, 5)
	if err != nil {
		t.Fatalf("Query([]): %v", err)
	}
	if len(ids2) != 0 {
		t.Fatalf("empty query returned ids: %v", ids2)
	}
}
