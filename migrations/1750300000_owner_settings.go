package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// owner_settings is a simple key/value store for UI preferences that are
// owned by the owner and have no business logic of their own.
// First consumer: "soul_avatar" = "male" | "female" (chat avatar choice).
func init() {
	m.Register(ownerSettingsUp, ownerSettingsDown)
}

func ownerSettingsUp(app core.App) error {
	owner := types.Pointer(ruleOwner)

	col := core.NewBaseCollection("owner_settings")
	setOwnerRules(col, owner)
	col.Fields.Add(
		&core.TextField{Name: "key", Required: true, Max: 80},
		&core.TextField{Name: "value", Max: 500},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	col.AddIndex("idx_owner_settings_key", true, "key", "")
	if err := app.Save(col); err != nil {
		return err
	}

	// Seed the default: soul avatar = male.
	rec := core.NewRecord(col)
	rec.Set("key", "soul_avatar")
	rec.Set("value", "male")
	return app.Save(rec)
}

func ownerSettingsDown(app core.App) error {
	col, err := app.FindCollectionByNameOrId("owner_settings")
	if err != nil {
		return nil
	}
	return app.Delete(col)
}
