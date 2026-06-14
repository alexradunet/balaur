package migrations

import (
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Local inference moved from a llamafile subprocess (chat_model = a file path)
// to Ollama (chat_model = a tag, e.g. "gemma4:e4b"). Rewrite legacy path-based
// local models to the default tag + dedicated embed tag so existing installs
// resolve a valid model instead of a permanently-"missing" file path.
func init() {
	m.Register(ollamaLocalModelsUp, ollamaLocalModelsDown)
}

func ollamaLocalModelsUp(app core.App) error {
	providers, err := app.FindRecordsByFilter("llm_providers", "kind = 'local'", "", 0, 0)
	if err != nil {
		return nil // collection not yet created on this box
	}
	for _, p := range providers {
		models, err := app.FindRecordsByFilter("llm_models", "provider = {:p}", "", 0, 0, dbx.Params{"p": p.Id})
		if err != nil {
			return err
		}
		for _, mdl := range models {
			chat := mdl.GetString("chat_model")
			if !strings.HasSuffix(chat, ".gguf") && !strings.HasSuffix(chat, ".llamafile") {
				continue // already a tag
			}
			mdl.Set("chat_model", "gemma4:e4b")
			mdl.Set("embed_model", "embeddinggemma")
			mdl.Set("label", "Local Gemma 4 E4B")
			if err := app.Save(mdl); err != nil {
				return err
			}
		}
	}
	return nil
}

func ollamaLocalModelsDown(app core.App) error {
	return nil // one-way data cleanup; nothing to restore
}
