package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(upNodeTypes, downNodeTypes)
}

// upNodeTypes creates the node_types registry, seeds the eight built-in types,
// and converts nodes.type from a closed SelectField to an open TextField.
func upNodeTypes(app core.App) error {
	// (a) Create the node_types registry collection.
	col := core.NewBaseCollection("node_types")
	setOwnerRules(col, types.Pointer(ruleOwner))
	col.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 60},
		&core.TextField{Name: "label", Max: 120},
		&core.TextField{Name: "icon", Max: 20},
		&core.SelectField{Name: "born_status", Required: true, MaxSelect: 1, Values: []string{"active", "proposed"}},
		&core.BoolField{Name: "system"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	col.AddIndex("idx_node_types_name", true, "name", "")
	if err := app.Save(col); err != nil {
		return fmt.Errorf("saving node_types collection: %w", err)
	}

	// (b) Seed the eight built-in types.
	type seed struct {
		name       string
		label      string
		icon       string
		bornStatus string
	}
	seeds := []seed{
		{"note", "Note", "", "active"},
		{"memory", "Memory", "", "proposed"},
		{"skill", "Skill", "", "proposed"},
		{"journal", "Journal", "", "active"},
		{"person", "Person", "", "active"},
		{"book", "Book", "", "active"},
		{"idea", "Idea", "", "active"},
		{"place", "Place", "", "active"},
	}
	for _, s := range seeds {
		row := core.NewRecord(col)
		row.Set("name", s.name)
		row.Set("label", s.label)
		row.Set("icon", s.icon)
		row.Set("born_status", s.bornStatus)
		row.Set("system", true)
		if err := app.Save(row); err != nil {
			return fmt.Errorf("seeding node_types %q: %w", s.name, err)
		}
	}

	// (c) Convert nodes.type from SelectField → TextField, preserving data.
	nodesCol, err := app.FindCollectionByNameOrId("nodes")
	if err != nil {
		return fmt.Errorf("finding nodes collection: %w", err)
	}

	// Count existing records per type before the schema change.
	existing, err := app.FindRecordsByFilter("nodes", "", "", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("reading nodes before migration: %w", err)
	}
	before := map[string]int{}
	for _, r := range existing {
		before[r.GetString("type")]++
	}
	app.Logger().Info("node_types migration: type counts before", "counts", before)

	// Replace the SelectField with a TextField (same SQLite column name — no data loss).
	nodesCol.Fields.RemoveByName("type")
	nodesCol.Fields.Add(&core.TextField{Name: "type", Required: true, Max: 60})
	if err := app.Save(nodesCol); err != nil {
		return fmt.Errorf("converting nodes.type to TextField: %w", err)
	}

	// Verify counts are unchanged.
	after, err := app.FindRecordsByFilter("nodes", "", "", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("reading nodes after migration: %w", err)
	}
	counts := map[string]int{}
	for _, r := range after {
		counts[r.GetString("type")]++
	}
	app.Logger().Info("node_types migration: type counts after", "counts", counts)
	for typ, n := range before {
		if counts[typ] != n {
			return fmt.Errorf("nodes.type migration: count mismatch for %q: before=%d after=%d (data loss)", typ, n, counts[typ])
		}
	}

	return nil
}

// downNodeTypes removes the node_types collection (nodes.type stays TextField).
func downNodeTypes(app core.App) error {
	col, err := app.FindCollectionByNameOrId("node_types")
	if err != nil {
		return nil // already gone
	}
	return app.Delete(col)
}
