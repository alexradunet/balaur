package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// summaries: the recap telescope. One row per (conversation, period_type,
// period_start): days summarise messages, weeks summarise days, months
// summarise days, quarters summarise months, years summarise quarters.
// Derived data — regenerable from messages at any time, hence no consent
// lifecycle; generation is audited and switchable instead (BALAUR_RECAP).
func init() {
	m.Register(summariesUp, summariesDown)
}

var summaryPeriods = []string{"day", "week", "month", "quarter", "year"}

func summariesUp(app core.App) error {
	conversations, err := app.FindCollectionByNameOrId("conversations")
	if err != nil {
		return err
	}

	col := core.NewBaseCollection("summaries")
	col.ListRule = types.Pointer(ruleOwner)
	col.ViewRule = types.Pointer(ruleOwner)
	col.Fields.Add(
		&core.RelationField{Name: "conversation", Required: true, CollectionId: conversations.Id, CascadeDelete: true},
		&core.SelectField{Name: "period_type", Required: true, Values: summaryPeriods},
		&core.DateField{Name: "period_start", Required: true},
		&core.DateField{Name: "period_end", Required: true},
		&core.TextField{Name: "content", Max: 20000},
		&core.NumberField{Name: "message_count", OnlyInt: true},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	// Idempotency anchor: one summary per conversation+granularity+period.
	col.AddIndex("idx_summaries_period", true, "conversation, period_type, period_start", "")
	return app.Save(col)
}

func summariesDown(app core.App) error {
	col, err := app.FindCollectionByNameOrId("summaries")
	if err != nil {
		return nil
	}
	return app.Delete(col)
}
