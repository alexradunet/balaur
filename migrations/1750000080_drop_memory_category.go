package migrations

import (
	"encoding/json"
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(upDropMemoryCategory, downDropMemoryCategory)
}

// upDropMemoryCategory collapses the memory category axis. The closed
// {fact, preference, person, project, context} select on the `memory` node type
// is removed from the node_types schema, and props.category is stripped from
// EVERY memory node across ALL statuses (proposed/active/archived/rejected) so
// pending proposals and history are not skipped. Nothing in the runtime
// branches on category — it was a display/navigation label only — so this is
// behavior-safe; only the label is lost (the down migration is lossy).
func upDropMemoryCategory(app core.App) error {
	// (a) Drop the category prop from the memory node_type schema.
	if nt, err := app.FindFirstRecordByFilter("node_types",
		"name = {:n}", dbx.Params{"n": "memory"}); err == nil {
		if raw := nt.GetString("properties"); raw != "" {
			var defs []map[string]any
			if err := json.Unmarshal([]byte(raw), &defs); err != nil {
				return fmt.Errorf("drop_memory_category: parsing memory schema: %w", err)
			}
			kept := make([]map[string]any, 0, len(defs))
			for _, d := range defs {
				if k, _ := d["key"].(string); k == "category" {
					continue
				}
				kept = append(kept, d)
			}
			out, err := json.Marshal(kept)
			if err != nil {
				return fmt.Errorf("drop_memory_category: marshalling memory schema: %w", err)
			}
			nt.Set("properties", string(out))
			if err := app.Save(nt); err != nil {
				return fmt.Errorf("drop_memory_category: saving memory schema: %w", err)
			}
		}
	} else {
		app.Logger().Warn("drop_memory_category: memory node_type not found, skipping schema edit")
	}

	// (b) Strip props.category from every memory node, ALL statuses.
	recs, err := app.FindRecordsByFilter("nodes", "type = 'memory'", "", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("drop_memory_category: loading memory nodes: %w", err)
	}
	before := len(recs)
	rewritten := 0
	for _, r := range recs {
		p := migGetProps(r)
		if _, ok := p["category"]; !ok {
			continue
		}
		delete(p, "category")
		r.Set("props", p)
		if err := app.Save(r); err != nil {
			return fmt.Errorf("drop_memory_category: rewriting node %q: %w", r.Id, err)
		}
		rewritten++
	}

	// (c) Invariant guard: no memory node lost, and no category key remains.
	after, err := app.CountRecords("nodes", dbx.HashExp{"type": "memory"})
	if err != nil {
		return fmt.Errorf("drop_memory_category: counting memory nodes: %w", err)
	}
	if int(after) != before {
		return fmt.Errorf("drop_memory_category: memory node count changed (before=%d after=%d)", before, after)
	}
	check, err := app.FindRecordsByFilter("nodes", "type = 'memory'", "", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("drop_memory_category: re-loading memory nodes: %w", err)
	}
	for _, r := range check {
		if _, ok := migGetProps(r)["category"]; ok {
			return fmt.Errorf("drop_memory_category: node %q still has props.category after rewrite", r.Id)
		}
	}
	app.Logger().Info("drop_memory_category: stripped category from memory nodes",
		"total", before, "rewritten", rewritten)
	return nil
}

// downDropMemoryCategory is LOSSY. It restores the category select to the memory
// node_type schema so new proposals can carry a category again, but it CANNOT
// reconstruct the per-row category values — up deleted them and they are
// unrecoverable. Existing memories come back category-less.
func downDropMemoryCategory(app core.App) error {
	nt, err := app.FindFirstRecordByFilter("node_types",
		"name = {:n}", dbx.Params{"n": "memory"})
	if err != nil {
		return nil // memory type gone — nothing to restore
	}
	var defs []map[string]any
	if raw := nt.GetString("properties"); raw != "" {
		if err := json.Unmarshal([]byte(raw), &defs); err != nil {
			return fmt.Errorf("drop_memory_category down: parsing memory schema: %w", err)
		}
	}
	for _, d := range defs {
		if k, _ := d["key"].(string); k == "category" {
			return nil // already present
		}
	}
	defs = append([]map[string]any{{
		"key":     "category",
		"label":   "Category",
		"type":    "select",
		"options": []string{"fact", "preference", "person", "project", "context"},
	}}, defs...)
	out, err := json.Marshal(defs)
	if err != nil {
		return fmt.Errorf("drop_memory_category down: marshalling memory schema: %w", err)
	}
	nt.Set("properties", string(out))
	if err := app.Save(nt); err != nil {
		return fmt.Errorf("drop_memory_category down: saving memory schema: %w", err)
	}
	return nil
}
