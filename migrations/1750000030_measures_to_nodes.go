package migrations

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(upMeasuresToNodes, downMeasuresToNodes)
}

// upMeasuresToNodes folds measure entries into the nodes spine as type=measure.
//
// Steps:
//  1. Register the "measure" node_type with its property schema.
//  2. Migrate measure rows (kind != 'completion' && kind != 'journal') → type=measure nodes.
//  3. Assert the migrated count equals the source count.
//  4. Delete the migrated measure rows from entries (completions stay).
func upMeasuresToNodes(app core.App) error {
	// ── Step 1: Register the "measure" node type ─────────────────────────────
	nodeTypesCol, err := app.FindCollectionByNameOrId("node_types")
	if err != nil {
		return fmt.Errorf("measures_to_nodes: node_types collection not found (plan 164 required): %w", err)
	}
	if nodeTypesCol.Fields.GetByName("properties") == nil {
		return fmt.Errorf("measures_to_nodes: node_types.properties field not found (plan 165 required)")
	}

	// Property schema for measure nodes.
	// noted_at is TEXT (not date) because PocketBase DateField serializes as
	// "2006-01-02 15:04:05.000Z" — not RFC3339 — and ValidateProps would reject
	// real values if the type were PropDate. Text passes everything through.
	type propDef struct {
		Key      string `json:"key"`
		Label    string `json:"label"`
		Type     string `json:"type"`
		Required bool   `json:"required"`
	}
	measureProps := []propDef{
		{Key: "kind", Label: "Kind", Type: "text", Required: true},
		{Key: "value_num", Label: "Value", Type: "number", Required: false},
		{Key: "unit", Label: "Unit", Type: "text", Required: false},
		{Key: "noted_at", Label: "Noted at", Type: "text", Required: true},
		{Key: "seed", Label: "Seed", Type: "bool", Required: false},
	}
	propsJSON, err := json.Marshal(measureProps)
	if err != nil {
		return fmt.Errorf("measures_to_nodes: marshalling measure property schema: %w", err)
	}

	measureTypeRow := core.NewRecord(nodeTypesCol)
	measureTypeRow.Set("name", "measure")
	measureTypeRow.Set("label", "Measure")
	measureTypeRow.Set("icon", "")
	measureTypeRow.Set("born_status", "active")
	measureTypeRow.Set("system", true)
	measureTypeRow.Set("properties", string(propsJSON))
	if err := app.Save(measureTypeRow); err != nil {
		return fmt.Errorf("measures_to_nodes: saving measure node_type: %w", err)
	}
	app.Logger().Info("measures_to_nodes: registered measure node_type")

	// ── Step 2: Migrate measure entries → type=measure nodes ─────────────────
	// Load all entries that are NOT completions and NOT journal entries.
	measureEntries, err := app.FindRecordsByFilter("entries",
		"kind != 'completion' && kind != 'journal'", "", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("measures_to_nodes: loading measure entries: %w", err)
	}
	sourceCount := len(measureEntries)
	app.Logger().Info("measures_to_nodes: migrating measure entries", "count", sourceCount)

	nodesCol, err := app.FindCollectionByNameOrId("nodes")
	if err != nil {
		return fmt.Errorf("measures_to_nodes: nodes collection not found: %w", err)
	}

	migratedIDs := make([]string, 0, sourceCount)
	for _, entry := range measureEntries {
		kind := entry.GetString("kind")
		notedAt := entry.GetDateTime("noted_at").Time().UTC()

		// Title: "<kind> <date>" e.g. "weight 2026-06-24"
		title := kind + " " + notedAt.Format("2006-01-02")

		// Body: the text field
		body := strings.TrimSpace(entry.GetString("text"))

		// Props: merge kind, value_num, unit, noted_at; plus any extras from value JSON.
		props := map[string]any{
			"kind":     kind,
			"noted_at": fmtMeasureTime(notedAt),
		}
		if v := entry.GetFloat("value_num"); v != 0 {
			props["value_num"] = v
		}
		if u := entry.GetString("unit"); u != "" {
			props["unit"] = u
		}
		// Merge extras from the value JSON field (e.g. {"seed":true}).
		var extras map[string]any
		if err := entry.UnmarshalJSONField("value", &extras); err == nil {
			for k, v := range extras {
				if _, exists := props[k]; !exists {
					props[k] = v
				}
			}
		}

		node := core.NewRecord(nodesCol)
		node.Set("type", "measure")
		node.Set("title", title)
		node.Set("body", body)
		node.Set("status", "active")
		node.Set("props", props)
		if err := app.Save(node); err != nil {
			return fmt.Errorf("measures_to_nodes: saving measure node for entry %s: %w", entry.Id, err)
		}
		migratedIDs = append(migratedIDs, entry.Id)
	}

	// ── Step 3: Assert count equality ────────────────────────────────────────
	migratedCount := len(migratedIDs)
	if migratedCount != sourceCount {
		return fmt.Errorf("measures_to_nodes: count mismatch: source=%d migrated=%d — data loss detected",
			sourceCount, migratedCount)
	}
	app.Logger().Info("measures_to_nodes: migration count verified", "count", migratedCount)

	// ── Step 4: Delete migrated measure rows (keep completions) ───────────────
	for _, entryID := range migratedIDs {
		entry, err := app.FindRecordById("entries", entryID)
		if err != nil {
			return fmt.Errorf("measures_to_nodes: reloading entry %s for deletion: %w", entryID, err)
		}
		if err := app.Delete(entry); err != nil {
			return fmt.Errorf("measures_to_nodes: deleting entry %s: %w", entryID, err)
		}
	}
	app.Logger().Info("measures_to_nodes: deleted source measure entries", "count", len(migratedIDs))

	return nil
}

// downMeasuresToNodes reverses the migration: recreates measure entries from
// type=measure nodes, then deletes those nodes. Also removes the measure node_type.
func downMeasuresToNodes(app core.App) error {
	// Load all active measure nodes.
	measureNodes, err := app.FindRecordsByFilter("nodes",
		"type = 'measure' && status = 'active'", "", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("measures_to_nodes down: loading measure nodes: %w", err)
	}

	entriesCol, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		return fmt.Errorf("measures_to_nodes down: entries collection not found: %w", err)
	}

	nodeIDsToDelete := make([]string, 0, len(measureNodes))
	for _, node := range measureNodes {
		props := map[string]any{}
		if err := node.UnmarshalJSONField("props", &props); err == nil {
			// ignore unmarshal error; props stays empty
		}
		getString := func(key string) string {
			if v, ok := props[key]; ok {
				if s, ok := v.(string); ok {
					return s
				}
			}
			return ""
		}
		getFloat := func(key string) float64 {
			if v, ok := props[key]; ok {
				if f, ok := v.(float64); ok {
					return f
				}
			}
			return 0
		}

		entry := core.NewRecord(entriesCol)
		entry.Set("kind", getString("kind"))
		entry.Set("text", node.GetString("body"))
		if v := getFloat("value_num"); v != 0 {
			entry.Set("value_num", v)
		}
		if u := getString("unit"); u != "" {
			entry.Set("unit", u)
		}
		// Reconstruct noted_at from the stored PB datetime string.
		if notedAtStr := getString("noted_at"); notedAtStr != "" {
			if t, err := time.Parse("2006-01-02 15:04:05.000Z", notedAtStr); err == nil {
				entry.Set("noted_at", t)
			}
		}
		// Reconstruct value JSON extras (e.g. seed marker).
		extras := map[string]any{}
		for k, v := range props {
			if k != "kind" && k != "value_num" && k != "unit" && k != "noted_at" {
				extras[k] = v
			}
		}
		if len(extras) > 0 {
			entry.Set("value", extras)
		}
		if err := app.Save(entry); err != nil {
			return fmt.Errorf("measures_to_nodes down: recreating entry for node %s: %w", node.Id, err)
		}
		nodeIDsToDelete = append(nodeIDsToDelete, node.Id)
	}

	// Delete the measure nodes.
	for _, id := range nodeIDsToDelete {
		node, err := app.FindRecordById("nodes", id)
		if err != nil {
			return fmt.Errorf("measures_to_nodes down: reloading node %s: %w", id, err)
		}
		if err := app.Delete(node); err != nil {
			return fmt.Errorf("measures_to_nodes down: deleting node %s: %w", id, err)
		}
	}

	// Remove the measure node_type row.
	if rec, err := app.FindFirstRecordByData("node_types", "name", "measure"); err == nil {
		if err := app.Delete(rec); err != nil {
			return fmt.Errorf("measures_to_nodes down: deleting measure node_type: %w", err)
		}
	}

	app.Logger().Info("measures_to_nodes down: reversed", "nodes_deleted", len(nodeIDsToDelete))
	return nil
}

// fmtMeasureTime formats a time as the PocketBase date string "2006-01-02 15:04:05.000Z".
// Stored in props.noted_at so hydrated rec.GetDateTime("noted_at") works.
func fmtMeasureTime(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05.000Z")
}
