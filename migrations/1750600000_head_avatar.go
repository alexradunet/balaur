package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

// headAvatarUp adds the balaur_avatar field to the heads collection.
// This lets each sub-head carry its own Balaur personality (balaur-01…balaur-16).
// Empty value means "inherit from the owner's balaur_avatar setting".
func init() {
	m.Register(headAvatarUp, headAvatarDown)
}

func headAvatarUp(app core.App) error {
	col, err := app.FindCollectionByNameOrId("heads")
	if err != nil {
		return err
	}
	col.Fields.Add(&core.TextField{Name: "balaur_avatar", Max: 20})
	return app.Save(col)
}

func headAvatarDown(app core.App) error {
	col, err := app.FindCollectionByNameOrId("heads")
	if err != nil {
		return nil
	}
	col.Fields.RemoveByName("balaur_avatar")
	return app.Save(col)
}
