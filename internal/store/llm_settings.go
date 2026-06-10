package store

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
)

const llmSettingsKey = "default"

type LLMChoice struct {
	Provider string
	Model    string
}

func ActiveLLMChoice(app core.App) (LLMChoice, bool, error) {
	recs, err := app.FindRecordsByFilter(
		"llm_settings",
		"key = {:key}",
		"", 1, 0,
		dbx.Params{"key": llmSettingsKey},
	)
	if err != nil {
		return LLMChoice{}, false, err
	}
	if len(recs) == 0 {
		return LLMChoice{}, false, nil
	}
	return LLMChoice{
		Provider: recs[0].GetString("provider"),
		Model:    recs[0].GetString("model"),
	}, true, nil
}

func SaveLLMChoice(app core.App, choice LLMChoice) error {
	col, err := app.FindCollectionByNameOrId("llm_settings")
	if err != nil {
		return err
	}
	rec, err := app.FindFirstRecordByData("llm_settings", "key", llmSettingsKey)
	if err != nil {
		rec = core.NewRecord(col)
		rec.Set("key", llmSettingsKey)
	}
	rec.Set("provider", choice.Provider)
	rec.Set("model", choice.Model)
	return app.Save(rec)
}
