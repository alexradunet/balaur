package migrations

import (
	"fmt"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(hideApiKeyUp, hideApiKeyDown)
}

func hideApiKeyUp(app core.App) error {
	col, err := app.FindCollectionByNameOrId("llm_providers")
	if err != nil {
		return fmt.Errorf("llm_providers: %w", err)
	}
	field := col.Fields.GetByName("api_key")
	if field != nil {
		field.SetHidden(true)
	}
	return app.Save(col)
}

func hideApiKeyDown(app core.App) error {
	col, err := app.FindCollectionByNameOrId("llm_providers")
	if err != nil {
		return nil
	}
	field := col.Fields.GetByName("api_key")
	if field != nil {
		field.SetHidden(false)
	}
	return app.Save(col)
}
