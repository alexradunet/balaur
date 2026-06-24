package migrations

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(upDayType, downDayType)
}

// upDayType registers the type=day node type and backfills on_day edges for
// all existing non-day nodes. Each node gets an edge to the type=day node
// representing its creation date in the owner's local timezone.
//
// Note: this migration intentionally does NOT import internal/nodes or
// internal/store to avoid the storetest→migrations→nodes import cycle.
// The DayNode resolve-or-create and AddEdge logic is inlined here.
func upDayType(app core.App) error {
	// ── Step 1: Register type=day ─────────────────────────────────────────────
	nodeTypesCol, err := app.FindCollectionByNameOrId("node_types")
	if err != nil {
		return fmt.Errorf("day_type: node_types collection not found (plan 164 required): %w", err)
	}
	if nodeTypesCol.Fields.GetByName("properties") == nil {
		return fmt.Errorf("day_type: node_types.properties field not found (plan 165 required)")
	}

	type propDef struct {
		Key      string `json:"key"`
		Label    string `json:"label"`
		Type     string `json:"type"`
		Required bool   `json:"required"`
	}
	propsJSON, err := json.Marshal([]propDef{
		{Key: "date", Label: "Date", Type: "text", Required: true},
	})
	if err != nil {
		return fmt.Errorf("day_type: marshalling day property schema: %w", err)
	}

	dayTypeRow := core.NewRecord(nodeTypesCol)
	dayTypeRow.Set("name", "day")
	dayTypeRow.Set("label", "Day")
	dayTypeRow.Set("icon", "")
	dayTypeRow.Set("born_status", "active")
	dayTypeRow.Set("system", true)
	dayTypeRow.Set("properties", string(propsJSON))
	if err := app.Save(dayTypeRow); err != nil {
		return fmt.Errorf("day_type: saving day node_type: %w", err)
	}
	app.Logger().Info("day_type: registered day node_type")

	// ── Step 2: Backfill on_day edges for existing nodes ──────────────────────
	// Resolve owner timezone from owner_settings (mirrors store.OwnerLocation).
	loc := migOwnerLocation(app)

	existing, err := app.FindRecordsByFilter("nodes", "type != 'day'", "", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("day_type: loading existing nodes: %w", err)
	}

	nodesCol, err := app.FindCollectionByNameOrId("nodes")
	if err != nil {
		return fmt.Errorf("day_type: finding nodes collection: %w", err)
	}
	edgesCol, err := app.FindCollectionByNameOrId("edges")
	if err != nil {
		return fmt.Errorf("day_type: finding edges collection: %w", err)
	}

	linked := 0
	for _, rec := range existing {
		t := rec.GetDateTime("created").Time().In(loc)
		key := t.Format("2006-01-02")

		dayNode, err := migResolveDayNode(app, nodesCol, key)
		if err != nil {
			app.Logger().Warn("day_type: backfill DayNode failed",
				"node_id", rec.Id, "err", err)
			continue
		}
		if _, err := migAddEdge(app, edgesCol, rec.Id, dayNode.Id, "on_day"); err != nil {
			app.Logger().Warn("day_type: backfill AddEdge failed",
				"node_id", rec.Id, "day_id", dayNode.Id, "err", err)
			continue
		}
		linked++
	}
	app.Logger().Info("day_type: backfilled on_day edges", "count", linked)
	return nil
}

// downDayType removes all on_day edges, all type=day nodes, and the day node_type row.
func downDayType(app core.App) error {
	onDayEdges, err := app.FindRecordsByFilter("edges", "type = {:t}", "", 0, 0,
		dbx.Params{"t": "on_day"})
	if err != nil {
		return fmt.Errorf("day_type down: loading on_day edges: %w", err)
	}
	for _, e := range onDayEdges {
		if err := app.Delete(e); err != nil {
			return fmt.Errorf("day_type down: deleting on_day edge %q: %w", e.Id, err)
		}
	}
	app.Logger().Info("day_type down: deleted on_day edges", "count", len(onDayEdges))

	dayNodes, err := app.FindRecordsByFilter("nodes", "type = 'day'", "", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("day_type down: loading day nodes: %w", err)
	}
	for _, n := range dayNodes {
		if err := app.Delete(n); err != nil {
			return fmt.Errorf("day_type down: deleting day node %q: %w", n.Id, err)
		}
	}
	app.Logger().Info("day_type down: deleted day nodes", "count", len(dayNodes))

	if nt, err := app.FindFirstRecordByFilter("node_types", "name = {:n}", dbx.Params{"n": "day"}); err == nil {
		_ = app.Delete(nt)
	}
	app.Logger().Info("day_type down: removed day node_type")
	return nil
}

// migOwnerLocation mirrors store.OwnerLocation without importing internal/store.
func migOwnerLocation(app core.App) *time.Location {
	rec, err := app.FindFirstRecordByFilter("owner_settings",
		"key = {:k}", dbx.Params{"k": "timezone"})
	if err != nil {
		return time.Local
	}
	name := rec.GetString("value")
	if name == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.Local
	}
	return loc
}

// migResolveDayNode resolves or creates the type=day node for the given date key.
func migResolveDayNode(app core.App, nodesCol *core.Collection, key string) (*core.Record, error) {
	rec, err := app.FindFirstRecordByFilter("nodes",
		"type = 'day' && status = 'active' && title = {:k}",
		dbx.Params{"k": key})
	if err == nil {
		return rec, nil
	}

	node := core.NewRecord(nodesCol)
	node.Set("type", "day")
	node.Set("title", key)
	node.Set("body", "")
	node.Set("status", "active")
	node.Set("props", map[string]any{"date": key})
	if err := app.Save(node); err != nil {
		return nil, fmt.Errorf("day_type: creating day node %q: %w", key, err)
	}
	return node, nil
}

// migAddEdge adds an edge idempotently (mirrors nodes.AddEdge without importing it).
func migAddEdge(app core.App, edgesCol *core.Collection, sourceID, targetID, edgeType string) (*core.Record, error) {
	e := core.NewRecord(edgesCol)
	e.Set("source", sourceID)
	e.Set("target", targetID)
	e.Set("type", edgeType)
	e.Set("context", "")
	if err := app.Save(e); err != nil {
		// Idempotent: unique-constraint violation means the edge already exists.
		if existing, ferr := app.FindFirstRecordByFilter("edges",
			"source = {:s} && target = {:t} && type = {:y}",
			dbx.Params{"s": sourceID, "t": targetID, "y": edgeType}); ferr == nil {
			return existing, nil
		}
		return nil, fmt.Errorf("day_type: saving edge: %w", err)
	}
	return e, nil
}
