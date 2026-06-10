package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(initModelCollections, dropModelCollections)
}

func initModelCollections(app core.App) error {
	owner := types.Pointer(ruleOwner)

	models := core.NewBaseCollection("local_models")
	setOwnerRules(models, owner)
	models.Fields.Add(
		&core.TextField{Name: "key", Required: true, Max: 160},
		&core.TextField{Name: "name", Required: true, Max: 240},
		&core.TextField{Name: "family", Required: true, Max: 120},
		&core.SelectField{Name: "role", Required: true, Values: []string{"chat", "embedding"}},
		&core.NumberField{Name: "release_year", Required: true, OnlyInt: true},
		&core.BoolField{Name: "tool_support"},
		&core.URLField{Name: "source_page"},
		&core.URLField{Name: "download_url", Required: true},
		&core.TextField{Name: "license", Max: 120},
		&core.TextField{Name: "provenance", Max: 400},
		&core.SelectField{Name: "status", Required: true, Values: []string{"available", "downloading", "downloaded", "failed"}},
		&core.TextField{Name: "local_path", Max: 2000},
		&core.NumberField{Name: "size_bytes", OnlyInt: true},
		&core.NumberField{Name: "downloaded_bytes", OnlyInt: true},
		&core.TextField{Name: "sha256", Max: 64},
		&core.TextField{Name: "error", Max: 2000},
		&core.BoolField{Name: "active"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	models.AddIndex("idx_local_models_key", true, "key", "")
	models.AddIndex("idx_local_models_status", false, "status", "")
	if err := app.Save(models); err != nil {
		return err
	}

	settings := core.NewBaseCollection("model_settings")
	setOwnerRules(settings, owner)
	settings.Fields.Add(
		&core.TextField{Name: "key", Required: true, Max: 40},
		&core.SelectField{Name: "provider", Required: true, Values: []string{"env", "local", "remote"}},
		&core.RelationField{Name: "active_chat_model", CollectionId: models.Id},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	settings.AddIndex("idx_model_settings_key", true, "key", "")
	return app.Save(settings)
}

func dropModelCollections(app core.App) error {
	for _, name := range []string{"model_settings", "local_models"} {
		c, err := app.FindCollectionByNameOrId(name)
		if err != nil {
			continue
		}
		if err := app.Delete(c); err != nil {
			return err
		}
	}
	return nil
}
