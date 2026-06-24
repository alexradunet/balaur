package migrations

import (
	"encoding/json"
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(upTasksToNodes, downTasksToNodes)
}

// upTasksToNodes folds the `tasks` collection into the nodes spine as type=task.
//
// Steps:
//  1. Register the "task" node_type with its property schema.
//  2. Migrate existing tasks rows → type=task nodes; preserve created/updated.
//  3. Remap entries.task from RelationField(tasks) → TextField (node id).
//  4. Drop the tasks collection.
func upTasksToNodes(app core.App) error {
	// ── Step 1: Register the "task" node type ────────────────────────────────
	nodeTypesCol, err := app.FindCollectionByNameOrId("node_types")
	if err != nil {
		return fmt.Errorf("tasks_to_nodes: node_types collection not found (plan 164 required): %w", err)
	}

	// Check plan 165 landed: node_types must have the properties field.
	if nodeTypesCol.Fields.GetByName("properties") == nil {
		return fmt.Errorf("tasks_to_nodes: node_types.properties field not found (plan 165 required)")
	}

	// Property schema for task nodes. Date-typed fields (due, snoozed_until,
	// nudged_at, done_at) are marked text because PocketBase DateField serializes
	// as "2006-01-02 15:04:05.000Z" — not RFC3339 — and ValidateProps would reject
	// them if marked PropDate. Text passes everything through.
	type propDef struct {
		Key      string   `json:"key"`
		Label    string   `json:"label"`
		Type     string   `json:"type"`
		Required bool     `json:"required"`
		Options  []string `json:"options,omitempty"`
	}
	taskProps := []propDef{
		{Key: "state", Label: "State", Type: "select", Required: true, Options: []string{"open", "done", "dropped"}},
		{Key: "due", Label: "Due", Type: "text"},
		{Key: "recur", Label: "Recurrence", Type: "text"},
		{Key: "recur_from_done", Label: "Recur from done", Type: "bool"},
		{Key: "snoozed_until", Label: "Snoozed until", Type: "text"},
		{Key: "nudged_at", Label: "Nudged at", Type: "text"},
		{Key: "done_at", Label: "Done at", Type: "text"},
		{Key: "source", Label: "Source", Type: "text"},
	}
	propsJSON, err := json.Marshal(taskProps)
	if err != nil {
		return fmt.Errorf("tasks_to_nodes: marshalling task property schema: %w", err)
	}

	taskTypeRow := core.NewRecord(nodeTypesCol)
	taskTypeRow.Set("name", "task")
	taskTypeRow.Set("label", "Task")
	taskTypeRow.Set("icon", "")
	taskTypeRow.Set("born_status", "active")
	taskTypeRow.Set("system", true)
	taskTypeRow.Set("properties", string(propsJSON))
	if err := app.Save(taskTypeRow); err != nil {
		return fmt.Errorf("tasks_to_nodes: saving task node_type: %w", err)
	}
	app.Logger().Info("tasks_to_nodes: registered task node_type")

	// ── Step 2: Migrate tasks rows → type=task nodes ─────────────────────────
	tasksCol, err := app.FindCollectionByNameOrId("tasks")
	if err != nil {
		// tasks collection missing — migration was already applied or never created.
		app.Logger().Warn("tasks_to_nodes: tasks collection not found — skipping data migration")
		return nil
	}
	nodesCol, err := app.FindCollectionByNameOrId("nodes")
	if err != nil {
		return fmt.Errorf("tasks_to_nodes: nodes collection not found: %w", err)
	}

	taskRecs, err := app.FindRecordsByFilter("tasks", "", "-created", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("tasks_to_nodes: loading tasks: %w", err)
	}
	app.Logger().Info("tasks_to_nodes: migrating tasks", "count", len(taskRecs))

	idMap := make(map[string]string, len(taskRecs)) // old task id → new node id

	for _, task := range taskRecs {
		props := map[string]any{
			"state": task.GetString("status"), // status → state
		}
		if due := task.GetString("due"); due != "" {
			props["due"] = due
		}
		if recur := task.GetString("recur"); recur != "" {
			props["recur"] = recur
		}
		props["recur_from_done"] = task.GetBool("recur_from_done")
		if su := task.GetString("snoozed_until"); su != "" {
			props["snoozed_until"] = su
		}
		if na := task.GetString("nudged_at"); na != "" {
			props["nudged_at"] = na
		}
		if da := task.GetString("done_at"); da != "" {
			props["done_at"] = da
		}
		if src := task.GetString("source"); src != "" {
			props["source"] = src
		}

		node := core.NewRecord(nodesCol)
		node.Set("type", "task")
		node.Set("title", task.GetString("title"))
		node.Set("body", task.GetString("notes"))
		node.Set("status", "active")
		node.Set("props", props)
		if err := app.Save(node); err != nil {
			return fmt.Errorf("tasks_to_nodes: saving node for task %q: %w", task.Id, err)
		}

		// Preserve original created/updated timestamps. AutodateField writes a
		// new timestamp on insert; patch them back with a raw SQL UPDATE.
		origCreated := task.GetString("created")
		origUpdated := task.GetString("updated")
		if origCreated != "" || origUpdated != "" {
			_, sqlErr := app.DB().NewQuery(
				"UPDATE nodes SET created={:c}, updated={:u} WHERE id={:id}",
			).Bind(map[string]any{
				"c":  origCreated,
				"u":  origUpdated,
				"id": node.Id,
			}).Execute()
			if sqlErr != nil {
				// Non-fatal: timestamp preservation is best-effort; ordering
				// on created is secondary to data correctness.
				app.Logger().Warn("tasks_to_nodes: could not preserve timestamps",
					"task_id", task.Id, "error", sqlErr)
			}
		}

		idMap[task.Id] = node.Id
	}

	// Count check: fail loud on data loss. Count only type=task nodes we
	// just created against the source tasks rows.
	taskNodeCount, err := app.FindRecordsByFilter("nodes", "type = 'task'", "", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("tasks_to_nodes: verifying task nodes: %w", err)
	}
	if len(taskNodeCount) != len(taskRecs) {
		return fmt.Errorf("tasks_to_nodes: data count mismatch: migrated %d tasks but found %d task nodes (data loss)", len(taskRecs), len(taskNodeCount))
	}
	app.Logger().Info("tasks_to_nodes: task migration verified", "migrated", len(taskRecs))

	// ── Step 3: Remap entries.task ─────────────────────────────────────────
	entriesCol, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		return fmt.Errorf("tasks_to_nodes: entries collection not found: %w", err)
	}

	// Change entries.task from RelationField(tasks) → TextField.
	// A TextField is simpler: streak/Done only do equality on the id value.
	entriesCol.Fields.RemoveByName("task")
	entriesCol.Fields.Add(&core.TextField{Name: "task", Max: 15})
	if err := app.Save(entriesCol); err != nil {
		return fmt.Errorf("tasks_to_nodes: converting entries.task to TextField: %w", err)
	}

	// Remap entry task ids to new node ids.
	entryRecs, err := app.FindRecordsByFilter("entries", "task != ''", "", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("tasks_to_nodes: loading entries with task: %w", err)
	}
	remapped := 0
	for _, entry := range entryRecs {
		oldID := entry.GetString("task")
		newID, ok := idMap[oldID]
		if !ok {
			app.Logger().Warn("tasks_to_nodes: entry task id not in idMap — leaving as-is",
				"entry_id", entry.Id, "task_id", oldID)
			continue
		}
		entry.Set("task", newID)
		if err := app.Save(entry); err != nil {
			return fmt.Errorf("tasks_to_nodes: remapping entry %q: %w", entry.Id, err)
		}
		remapped++
	}
	app.Logger().Info("tasks_to_nodes: remapped entry task ids", "count", remapped)

	// ── Step 4: Drop the tasks collection ────────────────────────────────────
	if err := app.Delete(tasksCol); err != nil {
		return fmt.Errorf("tasks_to_nodes: dropping tasks collection: %w", err)
	}
	app.Logger().Info("tasks_to_nodes: tasks collection dropped")

	return nil
}

// downTasksToNodes reverses the migration: recreate tasks, copy type=task nodes
// back, restore entries.task as a RelationField to tasks, then drop task nodes.
func downTasksToNodes(app core.App) error {
	// Only proceed if tasks is gone.
	if _, err := app.FindCollectionByNameOrId("tasks"); err == nil {
		app.Logger().Info("tasks_to_nodes down: tasks collection already exists — nothing to do")
		return nil
	}

	// Recreate the tasks collection (same schema as baseline init migration).
	tasks := core.NewBaseCollection("tasks")
	setOwnerRules(tasks, types.Pointer(ruleOwner))
	tasks.Fields.Add(
		&core.TextField{Name: "title", Required: true, Max: 300},
		&core.TextField{Name: "notes", Max: 5000},
		&core.SelectField{Name: "status", Required: true, Values: []string{"open", "done", "dropped"}},
		&core.DateField{Name: "due"},
		&core.TextField{Name: "recur", Max: 60},
		&core.BoolField{Name: "recur_from_done"},
		&core.DateField{Name: "snoozed_until"},
		&core.DateField{Name: "nudged_at"},
		&core.DateField{Name: "done_at"},
		&core.TextField{Name: "source", Max: 120},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	tasks.AddIndex("idx_tasks_due", false, "due", "")
	tasks.AddIndex("idx_tasks_nudge", false, "status, nudged_at, due", "")
	tasks.AddIndex("idx_tasks_done_at", false, "status, done_at", "")
	if err := app.Save(tasks); err != nil {
		return fmt.Errorf("tasks_to_nodes down: recreating tasks collection: %w", err)
	}

	// Copy type=task nodes back into tasks rows.
	taskNodes, err := app.FindRecordsByFilter("nodes", "type = 'task'", "", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("tasks_to_nodes down: loading task nodes: %w", err)
	}

	idMap := make(map[string]string, len(taskNodes)) // node id → task id

	for _, node := range taskNodes {
		props := map[string]any{}
		if raw, ok := node.Get("props").(map[string]any); ok {
			props = raw
		} else {
			_ = node.UnmarshalJSONField("props", &props)
		}
		getString := func(key string) string {
			if v, ok := props[key]; ok {
				if s, ok := v.(string); ok {
					return s
				}
			}
			return ""
		}

		rec := core.NewRecord(tasks)
		rec.Set("title", node.GetString("title"))
		rec.Set("notes", node.GetString("body"))
		rec.Set("status", getString("state"))
		if due := getString("due"); due != "" {
			rec.Set("due", due)
		}
		if recur := getString("recur"); recur != "" {
			rec.Set("recur", recur)
		}
		if rfd, ok := props["recur_from_done"].(bool); ok {
			rec.Set("recur_from_done", rfd)
		}
		if su := getString("snoozed_until"); su != "" {
			rec.Set("snoozed_until", su)
		}
		if na := getString("nudged_at"); na != "" {
			rec.Set("nudged_at", na)
		}
		if da := getString("done_at"); da != "" {
			rec.Set("done_at", da)
		}
		if src := getString("source"); src != "" {
			rec.Set("source", src)
		}
		if err := app.Save(rec); err != nil {
			return fmt.Errorf("tasks_to_nodes down: saving task from node %q: %w", node.Id, err)
		}

		// Best-effort: restore original timestamps.
		origCreated := node.GetString("created")
		origUpdated := node.GetString("updated")
		if origCreated != "" || origUpdated != "" {
			_, _ = app.DB().NewQuery(
				"UPDATE tasks SET created={:c}, updated={:u} WHERE id={:id}",
			).Bind(map[string]any{
				"c": origCreated, "u": origUpdated, "id": rec.Id,
			}).Execute()
		}

		idMap[node.Id] = rec.Id
	}

	// Restore entries.task as RelationField pointing at tasks.
	entriesCol, err := app.FindCollectionByNameOrId("entries")
	if err != nil {
		return fmt.Errorf("tasks_to_nodes down: entries collection not found: %w", err)
	}
	entriesCol.Fields.RemoveByName("task")
	entriesCol.Fields.Add(&core.RelationField{Name: "task", CollectionId: tasks.Id})
	if err := app.Save(entriesCol); err != nil {
		return fmt.Errorf("tasks_to_nodes down: restoring entries.task RelationField: %w", err)
	}

	// Remap entry task ids back to task row ids.
	entryRecs, err := app.FindRecordsByFilter("entries", "task != ''", "", 0, 0, nil)
	if err != nil {
		return fmt.Errorf("tasks_to_nodes down: loading entries: %w", err)
	}
	for _, entry := range entryRecs {
		oldNodeID := entry.GetString("task")
		newTaskID, ok := idMap[oldNodeID]
		if !ok {
			app.Logger().Warn("tasks_to_nodes down: entry node id not in idMap",
				"entry_id", entry.Id, "node_id", oldNodeID)
			continue
		}
		entry.Set("task", newTaskID)
		if err := app.Save(entry); err != nil {
			return fmt.Errorf("tasks_to_nodes down: remapping entry %q: %w", entry.Id, err)
		}
	}

	// Drop the task nodes.
	for _, node := range taskNodes {
		if err := app.Delete(node); err != nil {
			return fmt.Errorf("tasks_to_nodes down: deleting task node %q: %w", node.Id, err)
		}
	}

	// Remove the task node_type row.
	if nt, err := app.FindFirstRecordByFilter("node_types", "name = {:n}", dbx.Params{"n": "task"}); err == nil {
		_ = app.Delete(nt)
	}

	app.Logger().Info("tasks_to_nodes down: migration reversed")
	return nil
}
