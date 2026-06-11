package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// balaur-extensions: runtime tools written as single JS files under
// pb_extensions/. The collection is the consent ledger — an extension's
// code runs only while its row is active AND the file still hashes to the
// approved sha256. Any change re-proposes; nothing executes unapproved.
func init() {
	m.Register(extensionsUp, extensionsDown)
}

var extensionStatuses = []string{"proposed", "active", "disabled"}

func extensionsUp(app core.App) error {
	owner := types.Pointer(ruleOwner)

	ext := core.NewBaseCollection("extensions")
	setOwnerRules(ext, owner)
	ext.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 120},
		&core.TextField{Name: "description", Max: 1000},
		&core.TextField{Name: "path", Required: true, Max: 300},
		&core.TextField{Name: "sha256", Required: true, Max: 64},
		&core.SelectField{Name: "status", Required: true, Values: extensionStatuses},
		&core.JSONField{Name: "tools"},
		&core.TextField{Name: "source", Max: 120},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	ext.AddIndex("idx_extensions_name", true, "name", "")
	ext.AddIndex("idx_extensions_status", false, "status", "")
	return app.Save(ext)
}

func extensionsDown(app core.App) error {
	if col, err := app.FindCollectionByNameOrId("extensions"); err == nil {
		return app.Delete(col)
	}
	return nil
}
