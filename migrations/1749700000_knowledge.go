package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Knowledge lifecycle: memories and skills grow through a propose → approve
// flow. The model proposes; only the owner approves. Status values:
//
//	proposed  — created by the model, awaiting the owner's decision
//	active    — approved; eligible for context injection
//	archived  — kept for history, never injected
//	rejected  — dismissed proposal, kept so the model can learn what not
//	            to re-propose
var knowledgeStatuses = []string{"proposed", "active", "archived", "rejected"}

// memoryCategories keeps classification coarse on purpose (Pareto): enough
// to group and filter, not a taxonomy project.
var memoryCategories = []string{"fact", "preference", "person", "project", "context"}

func init() {
	m.Register(knowledgeUp, knowledgeDown)
}

func knowledgeUp(app core.App) error {
	memories, err := app.FindCollectionByNameOrId("memories")
	if err != nil {
		return err
	}
	memories.Fields.Add(
		&core.SelectField{Name: "status", Required: true, Values: knowledgeStatuses},
		&core.SelectField{Name: "category", Values: memoryCategories},
		&core.NumberField{Name: "importance", OnlyInt: true}, // 1..5, clamped in code
		&core.TextField{Name: "when_to_use", Max: 500},
		&core.DateField{Name: "last_used"},
		&core.NumberField{Name: "use_count", OnlyInt: true},
	)
	memories.AddIndex("idx_memories_status", false, "status", "")
	if err := app.Save(memories); err != nil {
		return err
	}

	skills, err := app.FindCollectionByNameOrId("skills")
	if err != nil {
		return err
	}
	skills.Fields.Add(
		&core.SelectField{Name: "status", Required: true, Values: knowledgeStatuses},
		&core.TextField{Name: "when_to_use", Max: 500},
		&core.DateField{Name: "last_used"},
		&core.NumberField{Name: "use_count", OnlyInt: true},
	)
	skills.AddIndex("idx_skills_status", false, "status", "")
	if err := app.Save(skills); err != nil {
		return err
	}

	// Backfill: anything created before the lifecycle existed was implicitly
	// owner-approved, so it becomes active.
	for _, name := range []string{"memories", "skills"} {
		records, err := app.FindAllRecords(name)
		if err != nil {
			return err
		}
		for _, rec := range records {
			if rec.GetString("status") == "" {
				rec.Set("status", "active")
				if err := app.Save(rec); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func knowledgeDown(app core.App) error {
	memories, err := app.FindCollectionByNameOrId("memories")
	if err != nil {
		return err
	}
	for _, f := range []string{"status", "category", "importance", "when_to_use", "last_used", "use_count"} {
		memories.Fields.RemoveByName(f)
	}
	if err := app.Save(memories); err != nil {
		return err
	}

	skills, err := app.FindCollectionByNameOrId("skills")
	if err != nil {
		return err
	}
	for _, f := range []string{"status", "when_to_use", "last_used", "use_count"} {
		skills.Fields.RemoveByName(f)
	}
	return app.Save(skills)
}
