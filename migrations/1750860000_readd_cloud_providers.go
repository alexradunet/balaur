package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Re-add the opt-in OpenAI-compatible cloud provider path: widen the
// llm_providers.kind enum from local-only back to {local, openai}. Plan 074
// (1750830000_remove_openai_providers) deleted the kind=openai records but
// deliberately left the base_url and api_key columns in place, so this is a pure
// enum change — no field add. Cloud is never the default and never auto-selected;
// activation is consent-gated in the web layer. See plans/118.
func init() {
	m.Register(readdCloudProvidersUp, readdCloudProvidersDown)
}

func readdCloudProvidersUp(app core.App) error {
	col, err := app.FindCollectionByNameOrId("llm_providers")
	if err != nil {
		return nil // collection absent on a fresh/older box; nothing to migrate
	}
	if f, ok := col.Fields.GetByName("kind").(*core.SelectField); ok {
		f.Values = []string{"local", "openai"}
		return app.Save(col)
	}
	return nil
}

func readdCloudProvidersDown(app core.App) error {
	col, err := app.FindCollectionByNameOrId("llm_providers")
	if err != nil {
		return nil
	}
	// Refuse to tighten the enum while openai records still exist — dropping the
	// value would orphan them. The owner must delete cloud models first.
	recs, err := app.FindRecordsByFilter("llm_providers", "kind = 'openai'", "", 1, 0)
	if err != nil {
		return err
	}
	if len(recs) > 0 {
		return nil
	}
	if f, ok := col.Fields.GetByName("kind").(*core.SelectField); ok {
		f.Values = []string{"local"}
		return app.Save(col)
	}
	return nil
}
