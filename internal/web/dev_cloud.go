package web

import (
	"os"
	"strings"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/store"
)

// bootstrapDevCloudModel is a DEVELOPMENT convenience: when BALAUR_MISTRAL_KEY
// is set it registers the curated Mistral cloud preset and selects it as the
// active model, so chat turns work during local testing (`make dev`, which
// sources the gitignored dev.env) without clicking through the Models page each
// run. The key is read from the env, stored in the hidden api_key field, and
// never logged; activation is audited by SetActiveLLMModel.
//
// It is a NO-OP when the key is absent — prod never sets it, so this never
// auto-enables the cloud path in production. Setting the key IS the owner's
// explicit, env-level opt-in to the consent-gated cloud path (AGENTS.md).
func bootstrapDevCloudModel(app core.App) {
	key := strings.TrimSpace(os.Getenv("BALAUR_MISTRAL_KEY"))
	if key == "" {
		return
	}
	preset, ok := llm.CloudPresetByKey("mistral")
	if !ok {
		return
	}
	modelID, err := store.SaveCloudModel(app, preset.Name, preset.BaseURL, key, preset.Label, preset.ChatModel, preset.EmbedModel)
	if err != nil {
		app.Logger().Error("dev cloud bootstrap: save Mistral model", "error", err)
		return
	}
	if err := store.SetActiveLLMModel(app, modelID, "dev-bootstrap"); err != nil {
		app.Logger().Error("dev cloud bootstrap: activate Mistral", "error", err)
		return
	}
	app.Logger().Info("dev cloud bootstrap: Mistral selected as active model", "model", preset.ChatModel)
}
