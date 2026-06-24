package migrations

import (
	"encoding/json"
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(upNodeTypeSchemas, downNodeTypeSchemas)
}

// upNodeTypeSchemas adds `properties` and `template` JSONFields to node_types,
// then backfills the built-in memory and skill type schemas so the registry
// documents reality (the knowledge package already writes these props shapes).
func upNodeTypeSchemas(app core.App) error {
	col, err := app.FindCollectionByNameOrId("node_types")
	if err != nil {
		return fmt.Errorf("node_type_schemas migration: node_types collection not found (plan 164 required): %w", err)
	}

	col.Fields.Add(
		&core.JSONField{Name: "properties", MaxSize: 65536},
		&core.JSONField{Name: "template", MaxSize: 65536},
	)
	if err := app.Save(col); err != nil {
		return fmt.Errorf("adding properties/template to node_types: %w", err)
	}

	// Backfill built-in schemas. These mirror exactly what ProposeMemory and
	// ProposeSkill in internal/knowledge write — the code is the source of truth.
	type propDef struct {
		Key      string   `json:"key"`
		Label    string   `json:"label"`
		Type     string   `json:"type"`
		Required bool     `json:"required"`
		Options  []string `json:"options,omitempty"`
	}
	type backfill struct {
		name  string
		props []propDef
	}

	backfills := []backfill{
		{
			name: "memory",
			props: []propDef{
				// category is a select but NOT required: ProposeMemory may write an empty
				// string when the caller omits it (e.g. in tests or CLI); code is the
				// source of truth for the schema shape.
				{Key: "category", Label: "Category", Type: "select",
					Options: []string{"fact", "preference", "person", "project", "context"}},
				// importance is always clamped to 1..5 by clampImportance, so it is
				// always present and non-zero; Required here documents that invariant.
				{Key: "importance", Label: "Importance", Type: "number", Required: true},
				{Key: "when_to_use", Label: "When to use", Type: "text"},
				{Key: "source", Label: "Source", Type: "text"},
			},
		},
		{
			name: "skill",
			props: []propDef{
				{Key: "description", Label: "Description", Type: "text"},
				{Key: "when_to_use", Label: "When to use", Type: "text"},
			},
		},
		// book gets a demonstration schema (optional, as noted in the plan).
		{
			name: "book",
			props: []propDef{
				{Key: "author", Label: "Author", Type: "text"},
				{Key: "year", Label: "Year", Type: "number"},
			},
		},
	}

	for _, bf := range backfills {
		raw, err := json.Marshal(bf.props)
		if err != nil {
			return fmt.Errorf("marshalling schema for %q: %w", bf.name, err)
		}
		rec, err := app.FindFirstRecordByData("node_types", "name", bf.name)
		if err != nil {
			// Missing built-in is non-fatal — log and continue.
			app.Logger().Warn("node_type_schemas: built-in type not found, skipping",
				"name", bf.name)
			continue
		}
		rec.Set("properties", string(raw))
		if err := app.Save(rec); err != nil {
			return fmt.Errorf("backfilling schema for %q: %w", bf.name, err)
		}
	}

	return nil
}

// downNodeTypeSchemas removes the properties and template fields (best-effort).
func downNodeTypeSchemas(app core.App) error {
	col, err := app.FindCollectionByNameOrId("node_types")
	if err != nil {
		return nil // already gone
	}
	col.Fields.RemoveByName("properties")
	col.Fields.RemoveByName("template")
	if err := app.Save(col); err != nil {
		return fmt.Errorf("removing properties/template from node_types: %w", err)
	}
	return nil
}
