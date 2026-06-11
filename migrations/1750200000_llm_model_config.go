package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(llmModelConfigUp, llmModelConfigDown)
}

func llmModelConfigUp(app core.App) error {
	owner := types.Pointer(ruleOwner)

	providers := core.NewBaseCollection("llm_providers")
	setOwnerRules(providers, owner)
	providers.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 120},
		&core.SelectField{Name: "kind", Required: true, Values: []string{"kronk", "openai"}},
		&core.TextField{Name: "base_url", Max: 2000},
		&core.TextField{Name: "api_key", Max: 10000},
		&core.BoolField{Name: "local"},
		&core.BoolField{Name: "enabled"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	providers.AddIndex("idx_llm_providers_name", true, "name", "")
	if err := app.Save(providers); err != nil {
		return err
	}

	models := core.NewBaseCollection("llm_models")
	setOwnerRules(models, owner)
	models.Fields.Add(
		&core.RelationField{Name: "provider", Required: true, CollectionId: providers.Id, CascadeDelete: true},
		&core.TextField{Name: "label", Required: true, Max: 200},
		&core.TextField{Name: "chat_model", Required: true, Max: 2000},
		&core.TextField{Name: "embed_model", Max: 2000},
		&core.BoolField{Name: "enabled"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	models.AddIndex("idx_llm_models_provider", false, "provider", "")
	if err := app.Save(models); err != nil {
		return err
	}

	if old, err := app.FindCollectionByNameOrId("llm_settings"); err == nil {
		if err := app.Delete(old); err != nil {
			return err
		}
	}
	settings := core.NewBaseCollection("llm_settings")
	setOwnerRules(settings, owner)
	settings.Fields.Add(
		&core.TextField{Name: "key", Required: true, Max: 40},
		&core.RelationField{Name: "active_model", CollectionId: models.Id},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	settings.AddIndex("idx_llm_settings_key", true, "key", "")
	return app.Save(settings)
}

func llmModelConfigDown(app core.App) error {
	for _, name := range []string{"llm_settings", "llm_models", "llm_providers"} {
		col, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			continue
		}
		if err := app.Delete(col); err != nil {
			return err
		}
	}
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
