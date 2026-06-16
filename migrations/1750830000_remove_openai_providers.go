package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Drop the remote "OpenAI-compatible" provider path: for v1 Balaur runs LLMs
// exactly one way — local inference (provider kind "local"). Delete every
// kind="openai" provider (its models cascade away), clear the active model if it
// pointed at one, and tighten the kind enum to local-only. The now-dead base_url
// and api_key columns are left in place (deprecated) to keep this migration a
// pure data/enum change; dropping them is deferred (plan 074).
func init() {
	m.Register(removeOpenAIProvidersUp, removeOpenAIProvidersDown)
}

func removeOpenAIProvidersUp(app core.App) error {
	col, err := app.FindCollectionByNameOrId("llm_providers")
	if err != nil {
		return nil // collection absent on a fresh/older box; nothing to migrate
	}

	providers, err := app.FindRecordsByFilter("llm_providers", "kind = 'openai'", "", 0, 0)
	if err != nil {
		return err
	}
	if len(providers) > 0 {
		settings, _ := app.FindFirstRecordByData("llm_settings", "key", "default")
		for _, p := range providers {
			// Clear the active model if it belongs to this provider, before the
			// cascade delete removes the record and leaves a dangling relation.
			if settings != nil && settings.GetString("active_model") != "" {
				models, err := app.FindRecordsByFilter("llm_models", "provider = {:p}", "", 0, 0, dbx.Params{"p": p.Id})
				if err != nil {
					return err
				}
				for _, mdl := range models {
					if mdl.Id == settings.GetString("active_model") {
						settings.Set("active_model", "")
						if err := app.Save(settings); err != nil {
							return err
						}
					}
				}
			}
			if err := app.Delete(p); err != nil { // cascades to child llm_models
				return err
			}
		}
	}

	if f, ok := col.Fields.GetByName("kind").(*core.SelectField); ok {
		f.Values = []string{"local"}
		if err := app.Save(col); err != nil {
			return err
		}
	}
	return nil
}

func removeOpenAIProvidersDown(app core.App) error {
	col, err := app.FindCollectionByNameOrId("llm_providers")
	if err != nil {
		return nil
	}
	if f, ok := col.Fields.GetByName("kind").(*core.SelectField); ok {
		f.Values = []string{"local", "openai"}
		return app.Save(col)
	}
	return nil
}
