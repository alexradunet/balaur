package migrations

import (
	"path/filepath"
	"strings"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Disable legacy local models whose chat_model is an Ollama tag (not an absolute
// .gguf path). V1 runs GGUF files in-process via the embedded Kronk engine, and
// Ollama was removed (plan 074), so a tag-based local model can no longer be
// served. Disabling it keeps an upgrading box from resolving the agent loop to a
// model nothing can run; if such a model was the active model, active_model is
// cleared so the box falls back to "no model" (the owner then installs a GGUF).
func init() {
	m.Register(kronkLocalModelsUp, func(core.App) error { return nil })
}

func kronkLocalModelsUp(app core.App) error {
	models, err := app.FindRecordsByFilter("llm_models", "enabled = true", "", 0, 0)
	if err != nil {
		return nil // collection absent on a fresh/older box — nothing to migrate
	}
	var settings *core.Record
	if s, err := app.FindFirstRecordByData("llm_settings", "key", "default"); err == nil {
		settings = s
	}
	for _, mdl := range models {
		if isGGUFModelPath(mdl.GetString("chat_model")) {
			continue
		}
		mdl.Set("enabled", false)
		if err := app.Save(mdl); err != nil {
			return err
		}
		if settings != nil && settings.GetString("active_model") == mdl.Id {
			settings.Set("active_model", "")
			if err := app.Save(settings); err != nil {
				return err
			}
		}
	}
	return nil
}

func isGGUFModelPath(s string) bool {
	return filepath.IsAbs(s) && strings.HasSuffix(strings.ToLower(s), ".gguf")
}
