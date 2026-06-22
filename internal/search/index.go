// Package search provides the FTS5 memory recall index — a rebuildable
// sidecar SQLite database backed by the ncruces/go-sqlite3 wazero driver
// (CGO-free, FTS5 included). The index is disposable: deleting
// pb_data/search.db is always safe; it is rebuilt on the next boot.
package search

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	_ "github.com/ncruces/go-sqlite3/driver" // registers driver "sqlite3"
)

// StoreKey is the app.Store() key under which the *Index singleton lives.
// Read by internal/knowledge to consult the index without importing main.
const StoreKey = "balaur.searchIndex"

// Index is the FTS5 sidecar search index. Open once per process; never
// per query — the wazero runtime has a startup cost.
type Index struct {
	db *sql.DB
}

// Open opens (creating if absent) the sidecar index at the given path.
// Caller should use filepath.Join(app.DataDir(), "search.db").
func Open(path string) (*Index, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("search: open %s: %w", path, err)
	}
	if _, err := db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts
		USING fts5(id UNINDEXED, title, content, when_to_use, category)
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("search: create fts5 table: %w", err)
	}
	return &Index{db: db}, nil
}

// Close releases the underlying database connection.
func (ix *Index) Close() error {
	return ix.db.Close()
}

// Rebuild drops and refills the entire index from the app's active memories.
// Idempotent; safe to call on boot even after a partial write.
func (ix *Index) Rebuild(app core.App) error {
	tx, err := ix.db.Begin()
	if err != nil {
		return fmt.Errorf("search: rebuild begin tx: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM memories_fts`); err != nil {
		tx.Rollback()
		return fmt.Errorf("search: rebuild delete: %w", err)
	}

	recs, err := app.FindRecordsByFilter("memories", "status = 'active'", "", 0, 0, nil)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("search: rebuild fetch: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO memories_fts(id, title, content, when_to_use, category) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("search: rebuild prepare: %w", err)
	}
	defer stmt.Close()

	for _, r := range recs {
		if _, err := stmt.Exec(
			r.Id,
			r.GetString("title"),
			r.GetString("content"),
			r.GetString("when_to_use"),
			r.GetString("category"),
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("search: rebuild insert %s: %w", r.Id, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("search: rebuild commit: %w", err)
	}
	return nil
}

// Upsert inserts or replaces one record in the index. Non-active records
// (archived, rejected, proposed) are removed rather than indexed.
func (ix *Index) Upsert(rec *core.Record) error {
	// Always delete first so Upsert is truly idempotent.
	if err := ix.Delete(rec.Id); err != nil {
		return err
	}
	if rec.GetString("status") != "active" {
		return nil // non-active: deletion above is the right action
	}
	_, err := ix.db.Exec(
		`INSERT INTO memories_fts(id, title, content, when_to_use, category) VALUES (?, ?, ?, ?, ?)`,
		rec.Id,
		rec.GetString("title"),
		rec.GetString("content"),
		rec.GetString("when_to_use"),
		rec.GetString("category"),
	)
	if err != nil {
		return fmt.Errorf("search: upsert %s: %w", rec.Id, err)
	}
	return nil
}

// Delete removes a memory from the index by id. No-op if absent.
func (ix *Index) Delete(id string) error {
	_, err := ix.db.Exec(`DELETE FROM memories_fts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("search: delete %s: %w", id, err)
	}
	return nil
}

// Query returns memory ids ranked by bm25 for the given terms (OR semantics),
// capped at limit. Each term is double-quote-enclosed so FTS5 treats it as a
// string token, not an operator; embedded double-quotes are doubled per FTS5
// quoting rules so user text cannot inject operators.
func (ix *Index) Query(terms []string, limit int) ([]string, error) {
	if len(terms) == 0 {
		return nil, nil
	}
	// Build the FTS5 match string: "term1" OR "term2" …
	quoted := make([]string, 0, len(terms))
	for _, t := range terms {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		// Escape embedded double-quotes by doubling them (FTS5 spec §3.1).
		t = strings.ReplaceAll(t, `"`, `""`)
		quoted = append(quoted, `"`+t+`"`)
	}
	if len(quoted) == 0 {
		return nil, nil
	}
	matchExpr := strings.Join(quoted, " OR ")

	rows, err := ix.db.Query(
		`SELECT id FROM memories_fts WHERE memories_fts MATCH ? ORDER BY rank LIMIT ?`,
		matchExpr, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search: query: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("search: scan: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}
