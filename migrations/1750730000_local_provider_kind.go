package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// Rename the local-inference provider kind from "kronk" to "local". In-process
// kronk/llama.cpp is gone; a local GGUF is now served by a llamafile
// subprocess and reached over the OpenAI-compatible API. The data model only
// needs to know it is local, not which engine serves it.
func init() {
	m.Register(localProviderKindUp, localProviderKindDown)
}

func localProviderKindUp(app core.App) error {
	col, err := app.FindCollectionByNameOrId("llm_providers")
	if err != nil {
		return nil // collection not yet created on this box; nothing to migrate
	}
	if f, ok := col.Fields.GetByName("kind").(*core.SelectField); ok {
		f.Values = []string{"local", "openai"}
		if err := app.Save(col); err != nil {
			return err
		}
	}
	recs, err := app.FindRecordsByFilter("llm_providers", "kind = 'kronk'", "", 0, 0)
	if err != nil {
		return err
	}
	for _, r := range recs {
		r.Set("kind", "local")
		if r.GetString("name") == "Local Kronk" {
			r.Set("name", "Local model")
		}
		if err := app.Save(r); err != nil {
			return err
		}
	}
	return nil
}

func localProviderKindDown(app core.App) error {
	col, err := app.FindCollectionByNameOrId("llm_providers")
	if err != nil {
		return nil
	}
	if f, ok := col.Fields.GetByName("kind").(*core.SelectField); ok {
		f.Values = []string{"kronk", "openai"}
		if err := app.Save(col); err != nil {
			return err
		}
	}
	recs, err := app.FindRecordsByFilter("llm_providers", "kind = 'local'", "", 0, 0)
	if err != nil {
		return err
	}
	for _, r := range recs {
		r.Set("kind", "kronk")
		if r.GetString("name") == "Local model" {
			r.Set("name", "Local Kronk")
		}
		if err := app.Save(r); err != nil {
			return err
		}
	}
	return nil
}
