package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// The 1750800000 migration rewrote each legacy path-based local model to the
// Ollama default tag without deduping, so a box with N legacy local models ends
// up with N identical rows (the model picker then shows N copies). Collapse
// duplicate chat_model rows per local provider to the oldest survivor; if a
// deleted row was the active model, repoint active_model to the survivor (same
// tag, so no behavior change). Fresh installs never create dupes.
func init() {
	m.Register(dedupLocalModelsUp, dedupLocalModelsDown)
}

func dedupLocalModelsUp(app core.App) error {
	providers, err := app.FindRecordsByFilter("llm_providers", "kind = 'local'", "", 0, 0)
	if err != nil {
		return nil // collection not yet created on this box
	}
	var settings *core.Record
	if s, err := app.FindFirstRecordByData("llm_settings", "key", "default"); err == nil {
		settings = s
	}
	for _, p := range providers {
		models, err := app.FindRecordsByFilter("llm_models", "provider = {:p}", "created", 0, 0, dbx.Params{"p": p.Id})
		if err != nil {
			return err
		}
		seen := map[string]string{} // chat_model -> survivor id
		for _, mdl := range models {
			chat := mdl.GetString("chat_model")
			survivor, dup := seen[chat]
			if !dup {
				seen[chat] = mdl.Id
				continue
			}
			if settings != nil && settings.GetString("active_model") == mdl.Id {
				settings.Set("active_model", survivor)
				if err := app.Save(settings); err != nil {
					return err
				}
			}
			if err := app.Delete(mdl); err != nil {
				return err
			}
		}
	}
	return nil
}

func dedupLocalModelsDown(app core.App) error {
	return nil // one-way cleanup; deleted duplicates cannot be restored
}
