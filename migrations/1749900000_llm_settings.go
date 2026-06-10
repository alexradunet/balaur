package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(llmSettingsUp, llmSettingsDown)
}

func llmSettingsUp(app core.App) error {
	owner := types.Pointer(ruleOwner)
	settings := core.NewBaseCollection("llm_settings")
	setOwnerRules(settings, owner)
	settings.Fields.Add(
		&core.TextField{Name: "key", Required: true, Max: 40},
		&core.SelectField{Name: "provider", Required: true, Values: []string{"local", "synthetic", "remote"}},
		&core.TextField{Name: "model", Required: true, Max: 2000},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	settings.AddIndex("idx_llm_settings_key", true, "key", "")
	return app.Save(settings)
}

func llmSettingsDown(app core.App) error {
	settings, err := app.FindCollectionByNameOrId("llm_settings")
	if err != nil {
		return nil
	}
	return app.Delete(settings)
}
