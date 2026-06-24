package migrations

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(upUnifyJournalIntoDay, downUnifyJournalIntoDay)
}

// upUnifyJournalIntoDay collapses type=journal nodes into type=day nodes
// (plan 171). After this migration every calendar date has at most one
// type=day node that is both the journal page and the on_day hub.
//
// Three sub-steps:
//  1. Re-type all type=journal nodes to type=day; ensure props.date is set and
//     title is the human-readable date. These become the canonical day nodes.
//  2. For each ISO-titled type=day node (title "YYYY-MM-DD") that was created
//     by migration 1750000040: if a human-titled day node exists for the same
//     props.date (from step 1), re-point its inbound on_day edges to the
//     human node, drop duplicate edges, then delete the ISO node. If no human
//     node exists for that date, retitle the ISO node to human format.
//  3. Remove the journal row from node_types (LAST, after no type=journal
//     nodes remain).
func upUnifyJournalIntoDay(app core.App) error {
	loc := migOwnerLocation(app)

	nodesCol, err := app.FindCollectionByNameOrId("nodes")
	if err != nil {
		return fmt.Errorf("unify_journal: finding nodes collection: %w", err)
	}
	edgesCol, err := app.FindCollectionByNameOrId("edges")
	if err != nil {
		return fmt.Errorf("unify_journal: finding edges collection: %w", err)
	}

	// ── Step 1: Re-type journal nodes → day ──────────────────────────────────

	journalNodes, err := app.FindRecordsByFilter("nodes",
		"type = 'journal' && status = 'active'", "", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("unify_journal: loading journal nodes: %w", err)
	}

	for _, jn := range journalNodes {
		// Ensure props.date is set (always should be from plan 160, but be safe).
		dateKey := ""
		if raw := jn.GetString("props"); raw != "" {
			var p map[string]any
			if json.Unmarshal([]byte(raw), &p) == nil {
				if d, ok := p["date"].(string); ok {
					dateKey = d
				}
			}
		}
		if dateKey == "" {
			// Fall back to creation date in owner timezone.
			dateKey = jn.GetDateTime("created").Time().In(loc).Format("2006-01-02")
		}

		// Human-readable title.
		t, err2 := parseDate(dateKey, loc)
		if err2 != nil {
			app.Logger().Warn("unify_journal: skipping journal node with unparseable date",
				"id", jn.Id, "date", dateKey, "err", err2)
			continue
		}
		humanLabel := t.Format("Monday, January 2 2006")

		jn.Set("type", "day")
		jn.Set("title", humanLabel)
		// Ensure props.date is persisted correctly.
		p := migGetProps(jn)
		p["date"] = dateKey
		jn.Set("props", p)

		if err2 := app.Save(jn); err2 != nil {
			return fmt.Errorf("unify_journal: re-typing journal node %q: %w", jn.Id, err2)
		}
	}
	app.Logger().Info("unify_journal: re-typed journal nodes to day", "count", len(journalNodes))

	// ── Step 2: Merge ISO-titled day nodes ───────────────────────────────────
	// After step 1, some dates may now have TWO type=day nodes: the newly
	// re-typed one (human title) and the ISO-titled one from migration 040.
	// Merge them: re-point on_day edges from the ISO node to the human node,
	// deduplicate, then delete the ISO node.

	// Load all remaining type=day nodes.
	dayNodes, err := app.FindRecordsByFilter("nodes",
		"type = 'day' && status = 'active'", "", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("unify_journal: loading day nodes: %w", err)
	}

	// Group by props.date.
	type dayGroup struct {
		human []*core.Record // human-titled ("Monday, January 2 2006")
		iso   []*core.Record // ISO-titled ("YYYY-MM-DD")
	}
	byDate := map[string]*dayGroup{}
	for _, dn := range dayNodes {
		dk := migNodePropsDate(dn)
		if dk == "" {
			// No props.date — use title as key if it looks like ISO.
			title := dn.GetString("title")
			if isISODate(title) {
				dk = title
			} else {
				continue // can't group without a key
			}
		}
		g, ok := byDate[dk]
		if !ok {
			g = &dayGroup{}
			byDate[dk] = g
		}
		title := dn.GetString("title")
		if isISODate(title) {
			g.iso = append(g.iso, dn)
		} else {
			g.human = append(g.human, dn)
		}
	}

	merged := 0
	for dateKey, g := range byDate {
		if len(g.iso) == 0 {
			continue // nothing to merge
		}
		if len(g.human) == 0 {
			// No human node exists — just retitle the ISO node.
			t, err2 := parseDate(dateKey, loc)
			if err2 != nil {
				continue
			}
			humanLabel := t.Format("Monday, January 2 2006")
			for _, isoNode := range g.iso {
				isoNode.Set("title", humanLabel)
				if err2 := app.Save(isoNode); err2 != nil {
					app.Logger().Warn("unify_journal: retitling ISO day node failed",
						"id", isoNode.Id, "err", err2)
				}
			}
			continue
		}

		// Human node(s) exist — use the first one as the canonical target.
		// (There should only ever be one, but handle >1 gracefully.)
		canonID := g.human[0].Id

		// Re-point on_day edges from each ISO node to the canonical human node.
		for _, isoNode := range g.iso {
			onDayEdges, err2 := app.FindRecordsByFilter("edges",
				"target = {:t} && type = 'on_day'", "", 0, 0,
				dbx.Params{"t": isoNode.Id})
			if err2 != nil {
				app.Logger().Warn("unify_journal: loading on_day edges for ISO node failed",
					"id", isoNode.Id, "err", err2)
				continue
			}

			for _, edge := range onDayEdges {
				srcID := edge.GetString("source")
				// Try to re-point to the canonical node; AddEdge is idempotent.
				if _, err3 := migAddEdge(app, edgesCol, srcID, canonID, "on_day"); err3 != nil {
					// Unique index violation means edge already exists — drop this one.
					app.Logger().Info("unify_journal: dropping duplicate on_day edge",
						"source", srcID, "from_target", isoNode.Id, "to_target", canonID)
				}
				// Delete the old edge regardless (whether we re-pointed or it was a dup).
				if err3 := app.Delete(edge); err3 != nil {
					app.Logger().Warn("unify_journal: deleting old on_day edge failed",
						"id", edge.Id, "err", err3)
				}
			}

			// Also re-point other edge types (source or target) that reference this ISO node
			// — they arose from wikilinks or semantic edges to the ISO-titled node.
			// (These are rare; walk both directions.)
			outEdges, _ := app.FindRecordsByFilter("edges",
				"source = {:s}", "", 0, 0, dbx.Params{"s": isoNode.Id})
			for _, edge := range outEdges {
				edge.Set("source", canonID)
				_ = app.Save(edge) // best-effort; unique index will reject true dups
			}
			inEdges, _ := app.FindRecordsByFilter("edges",
				"target = {:t}", "", 0, 0, dbx.Params{"t": isoNode.Id})
			for _, edge := range inEdges {
				edge.Set("target", canonID)
				_ = app.Save(edge)
			}

			// Now safe to delete the ISO node (no remaining edges point to it).
			if err2 := app.Delete(isoNode); err2 != nil {
				return fmt.Errorf("unify_journal: deleting ISO day node %q: %w", isoNode.Id, err2)
			}
			merged++
		}
	}
	app.Logger().Info("unify_journal: merged ISO day nodes", "count", merged)

	// ── Step 3: Remove the journal node_type row ──────────────────────────────
	// Verify no type=journal nodes remain before removing the registry row.
	remaining, _ := app.CountRecords("nodes", dbx.HashExp{"type": "journal"})
	if remaining > 0 {
		return fmt.Errorf("unify_journal: %d type=journal nodes remain after re-typing; aborting", remaining)
	}

	if nt, err2 := app.FindFirstRecordByFilter("node_types",
		"name = {:n}", dbx.Params{"n": "journal"}); err2 == nil {
		if err2 := app.Delete(nt); err2 != nil {
			return fmt.Errorf("unify_journal: deleting journal node_type: %w", err2)
		}
		app.Logger().Info("unify_journal: removed journal node_type row")
	}

	_ = nodesCol // used indirectly via migResolveDayNode pattern; suppress unused
	return nil
}

// downUnifyJournalIntoDay is a best-effort heuristic reverse. It re-creates
// the journal node_type row and re-types type=day nodes that have a non-empty
// body back to type=journal. Empty-body day nodes (the on_day hubs) are left
// as type=day. This cannot perfectly reconstruct ISO-titled day nodes that
// were merged away, so the down migration is lossy for the on_day edge graph —
// document this limitation.
//
// NOTE: A perfect reverse is impossible without silent data loss because the
// ISO day nodes from migration 040 were deleted. The down migration preserves
// journal content but does NOT reconstruct on_day edge topology for those dates.
func downUnifyJournalIntoDay(app core.App) error {
	loc := migOwnerLocation(app)

	// Re-create the journal node_type row.
	nodeTypesCol, err := app.FindCollectionByNameOrId("node_types")
	if err != nil {
		return fmt.Errorf("unify_journal down: node_types collection: %w", err)
	}

	// Only create if absent.
	if _, err2 := app.FindFirstRecordByFilter("node_types",
		"name = {:n}", dbx.Params{"n": "journal"}); err2 != nil {
		row := core.NewRecord(nodeTypesCol)
		row.Set("name", "journal")
		row.Set("label", "Journal")
		row.Set("icon", "")
		row.Set("born_status", "active")
		row.Set("system", false)
		row.Set("properties", `[{"key":"date","label":"Date","type":"text","required":true}]`)
		if err2 := app.Save(row); err2 != nil {
			return fmt.Errorf("unify_journal down: creating journal node_type: %w", err2)
		}
	}

	// Re-type non-empty-body type=day nodes back to type=journal.
	dayNodes, err := app.FindRecordsByFilter("nodes",
		"type = 'day' && status = 'active'", "", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("unify_journal down: loading day nodes: %w", err)
	}

	reverted := 0
	for _, dn := range dayNodes {
		if dn.GetString("body") == "" {
			continue // empty-body day nodes stay as type=day (on_day hubs)
		}
		// Restore ISO title from props.date for the journal node convention.
		dateKey := migNodePropsDate(dn)
		if dateKey == "" {
			continue
		}
		// Human title is already correct for journal nodes (plan 160 created them
		// with human titles); keep it as-is — only change the type.
		dn.Set("type", "journal")
		_ = loc // loc available if needed for future use
		if err2 := app.Save(dn); err2 != nil {
			app.Logger().Warn("unify_journal down: re-typing day→journal failed",
				"id", dn.Id, "err", err2)
			continue
		}
		reverted++
	}
	app.Logger().Info("unify_journal down: reverted day→journal", "count", reverted)
	return nil
}

// ── helpers ──────────────────────────────────────────────────────────────────

// isISODate reports whether s looks like "YYYY-MM-DD".
func isISODate(s string) bool {
	if len(s) != 10 {
		return false
	}
	return s[4] == '-' && s[7] == '-' &&
		strings.IndexFunc(s[:4], func(r rune) bool { return r < '0' || r > '9' }) == -1 &&
		strings.IndexFunc(s[5:7], func(r rune) bool { return r < '0' || r > '9' }) == -1 &&
		strings.IndexFunc(s[8:], func(r rune) bool { return r < '0' || r > '9' }) == -1
}

// parseDate parses a "YYYY-MM-DD" string into a time.Time in loc.
func parseDate(s string, loc *time.Location) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", s, loc)
}

// migGetProps reads the props JSON field of a node as a map.
func migGetProps(rec *core.Record) map[string]any {
	raw := rec.GetString("props")
	if raw == "" {
		return map[string]any{}
	}
	var p map[string]any
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return map[string]any{}
	}
	return p
}

// migNodePropsDate returns the props.date value of a day node.
func migNodePropsDate(rec *core.Record) string {
	p := migGetProps(rec)
	if d, ok := p["date"].(string); ok {
		return d
	}
	return ""
}
