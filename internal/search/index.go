// Package search provides the FTS5 knowledge recall index — a rebuildable
// sidecar SQLite database backed by the ncruces/go-sqlite3 wazero driver
// (CGO-free, FTS5 included). It indexes all active node types (note, memory,
// skill, day, and typed objects), keyed by kind. The index is disposable:
// deleting pb_data/search.db is always safe; it is rebuilt on the next boot.
package search

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"

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
		CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_fts
		USING fts5(id UNINDEXED, kind UNINDEXED, title, content, extra)
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

// Rebuild drops and refills the entire index from the app's active nodes (all
// types). Idempotent; safe to call on boot even after a partial write.
func (ix *Index) Rebuild(app core.App) error {
	tx, err := ix.db.Begin()
	if err != nil {
		return fmt.Errorf("search: rebuild begin tx: %w", err)
	}
	if _, err := tx.Exec(`DELETE FROM knowledge_fts`); err != nil {
		tx.Rollback()
		return fmt.Errorf("search: rebuild delete: %w", err)
	}

	recs, err := app.FindRecordsByFilter("nodes", "status = 'active'", "", 0, 0, nil)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("search: rebuild fetch: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO knowledge_fts(id, kind, title, content, extra) VALUES (?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("search: rebuild prepare: %w", err)
	}
	defer stmt.Close()

	for _, r := range recs {
		if _, err := stmt.Exec(
			r.Id,
			r.GetString("type"),
			r.GetString("title"),
			r.GetString("body"),
			nodeExtra(r),
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

// nodeExtra returns type-specific searchable text for a node's `extra` FTS
// column. v1 preserves the memory recall hint (when_to_use, stored in props by
// plan 160) so search parity with the prior memory-only index is not lost;
// other types contribute nothing extra yet.
func nodeExtra(r *core.Record) string {
	if r.GetString("type") == "memory" {
		return strings.TrimSpace(nodes.PropString(r, "when_to_use"))
	}
	return ""
}

// Upsert inserts or replaces one node in the index. The hooks bind to the whole
// `nodes` collection, so this fires for every node type; only active nodes
// belong in knowledge_fts. A node that is no longer active is deleted-then-
// skipped, keeping the index clean across status changes (the consent filter).
// content comes from the node body; extra carries type-specific searchable text.
func (ix *Index) Upsert(rec *core.Record) error {
	// Always delete first so Upsert is truly idempotent.
	if err := ix.Delete(rec.Id); err != nil {
		return err
	}
	if rec.GetString("status") != "active" {
		return nil // non-active: deletion above is the right action
	}
	_, err := ix.db.Exec(
		`INSERT INTO knowledge_fts(id, kind, title, content, extra) VALUES (?, ?, ?, ?, ?)`,
		rec.Id,
		rec.GetString("type"),
		rec.GetString("title"),
		rec.GetString("body"),
		nodeExtra(rec),
	)
	if err != nil {
		return fmt.Errorf("search: upsert %s: %w", rec.Id, err)
	}
	return nil
}

// Delete removes a node from the index by id. No-op if absent.
func (ix *Index) Delete(id string) error {
	_, err := ix.db.Exec(`DELETE FROM knowledge_fts WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("search: delete %s: %w", id, err)
	}
	return nil
}

// matchExpr builds the FTS5 match string "term1" OR "term2" … from terms.
// Each term is double-quote-enclosed so FTS5 treats it as a string token, not
// an operator; embedded double-quotes are doubled per FTS5 quoting rules
// (spec §3.1) so user text cannot inject operators. Returns "" when no usable
// term remains, signalling the caller to return no results.
func matchExpr(terms []string) string {
	quoted := make([]string, 0, len(terms))
	for _, t := range terms {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		t = strings.ReplaceAll(t, `"`, `""`)
		quoted = append(quoted, `"`+t+`"`)
	}
	return strings.Join(quoted, " OR ")
}

// scanIDs reads the id column off an FTS query result set.
func scanIDs(rows *sql.Rows) ([]string, error) {
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

// Query returns node ids of ANY kind ranked by bm25 for the given terms (OR
// semantics), capped at limit. This backs the cross-type `search` surface.
func (ix *Index) Query(terms []string, limit int) ([]string, error) {
	if len(terms) == 0 {
		return nil, nil
	}
	expr := matchExpr(terms)
	if expr == "" {
		return nil, nil
	}
	rows, err := ix.db.Query(
		`SELECT id FROM knowledge_fts WHERE knowledge_fts MATCH ? ORDER BY rank LIMIT ?`,
		expr, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search: query: %w", err)
	}
	return scanIDs(rows)
}

// QueryKind is Query narrowed to a single node kind (e.g. "memory"). It is the
// same bm25-ranked, injection-safe match with an added `kind = ?` filter so the
// memory-scoped recall path returns only memory hits.
func (ix *Index) QueryKind(terms []string, kind string, limit int) ([]string, error) {
	if len(terms) == 0 {
		return nil, nil
	}
	expr := matchExpr(terms)
	if expr == "" {
		return nil, nil
	}
	rows, err := ix.db.Query(
		`SELECT id FROM knowledge_fts WHERE knowledge_fts MATCH ? AND kind = ? ORDER BY rank LIMIT ?`,
		expr, kind, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search: query kind %q: %w", kind, err)
	}
	return scanIDs(rows)
}
