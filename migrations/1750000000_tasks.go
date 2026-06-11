package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// Tasks and the life log. tasks holds commitments — one-offs and recurring
// habits/chores; the nudger (next slice) fires on due, with nudged_at as the
// fired-state so firing is idempotent and restart-safe. entries is the
// append-friendly life log shared by habit completions and, in later slices,
// health tracking (weight, workouts, achievements) and journaling. Streaks,
// trends and PRs are derived from entries at read time, never stored.
func init() {
	m.Register(tasksUp, tasksDown)
}

var taskStatuses = []string{"open", "done", "dropped"}

// entryKinds stays coarse on purpose (Pareto): enough to group, chart, and
// assemble a day page — not a taxonomy project.
var entryKinds = []string{"completion", "weight", "workout", "achievement", "journal", "note"}

func tasksUp(app core.App) error {
	owner := types.Pointer(ruleOwner)

	tasks := core.NewBaseCollection("tasks")
	setOwnerRules(tasks, owner)
	tasks.Fields.Add(
		&core.TextField{Name: "title", Required: true, Max: 300},
		&core.TextField{Name: "notes", Max: 5000},
		&core.SelectField{Name: "status", Required: true, Values: taskStatuses},
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
	tasks.AddIndex("idx_tasks_status", false, "status", "")
	tasks.AddIndex("idx_tasks_due", false, "due", "")
	if err := app.Save(tasks); err != nil {
		return err
	}

	entries := core.NewBaseCollection("entries")
	setOwnerRules(entries, owner)
	entries.Fields.Add(
		&core.SelectField{Name: "kind", Required: true, Values: entryKinds},
		&core.RelationField{Name: "task", CollectionId: tasks.Id},
		&core.JSONField{Name: "value"},
		&core.TextField{Name: "text", Max: 5000},
		&core.DateField{Name: "noted_at", Required: true},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	entries.AddIndex("idx_entries_kind_noted", false, "kind, noted_at", "")
	entries.AddIndex("idx_entries_task", false, "task", "")
	if err := app.Save(entries); err != nil {
		return err
	}

	// Agent-initiated messages (nudges, briefings — next slices) become
	// distinguishable from chat turns; briefing idempotency derives from it.
	messages, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		return err
	}
	messages.Fields.Add(&core.TextField{Name: "origin", Max: 30})
	return app.Save(messages)
}

func tasksDown(app core.App) error {
	for _, name := range []string{"entries", "tasks"} {
		if col, err := app.FindCollectionByNameOrId(name); err == nil {
			if err := app.Delete(col); err != nil {
				return err
			}
		}
	}
	messages, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		return nil
	}
	messages.Fields.RemoveByName("origin")
	return app.Save(messages)
}
