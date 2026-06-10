// Package search will hold Balaur's full-text recall. This spike test locks
// in the driver decision ahead of that work:
//
// PocketBase's default driver (modernc.org/sqlite) ships WITHOUT FTS5.
// github.com/ncruces/go-sqlite3 (SQLite compiled to WASM, run via wazero)
// includes FTS5 and stays CGO-free, so the single-binary story survives.
// When search lands, the driver moves into the product via
// pocketbase.Config.DBConnect — the officially documented pattern
// (https://pocketbase.io/docs/go-overview/#custom-sqlite-driver) — or backs
// a separate disposable index DB. Until then it is a test-only dependency
// and adds nothing to the shipped binary.
package search

import (
	"database/sql"
	"testing"

	_ "github.com/ncruces/go-sqlite3/driver" // registers driver "sqlite3"
	_ "github.com/ncruces/go-sqlite3/embed"  // embeds the SQLite WASM build
)

func TestFTS5Available(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE VIRTUAL TABLE docs USING fts5(title, body)`); err != nil {
		t.Fatalf("FTS5 unavailable: %v", err)
	}

	seed := []struct{ title, body string }{
		{"market list", "buy salt, flour and a sack of apples"},
		{"journal", "walked the old forest path past the mill"},
		{"recipe", "knead the flour, rest the dough an hour"},
	}
	for _, d := range seed {
		if _, err := db.Exec(`INSERT INTO docs VALUES (?, ?)`, d.title, d.body); err != nil {
			t.Fatalf("insert: %v", err)
		}
	}

	rows, err := db.Query(`SELECT title FROM docs WHERE docs MATCH ? ORDER BY rank`, "flour")
	if err != nil {
		t.Fatalf("match query: %v", err)
	}
	defer rows.Close()

	var titles []string
	for rows.Next() {
		var title string
		if err := rows.Scan(&title); err != nil {
			t.Fatalf("scan: %v", err)
		}
		titles = append(titles, title)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows: %v", err)
	}
	if len(titles) != 2 {
		t.Fatalf("expected 2 matches for 'flour', got %v", titles)
	}
}
