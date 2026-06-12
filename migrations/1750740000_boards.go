package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// boards — owner-composed dashboards of typed cards (plan 029).
// Each board is a named, ordered list of card references; the page renders
// a 12-column CSS grid of slots that lazy-load their card via HTMX.
func init() {
	m.Register(boardsUp, boardsDown)
}

func boardsUp(app core.App) error {
	owner := types.Pointer(ruleOwner)

	boards := core.NewBaseCollection("boards")
	setOwnerRules(boards, owner)
	boards.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 80},
		&core.JSONField{Name: "cards"},
		&core.NumberField{Name: "sort"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	boards.AddIndex("idx_boards_sort", false, "sort", "")
	return app.Save(boards)
}

func boardsDown(app core.App) error {
	col, err := app.FindCollectionByNameOrId("boards")
	if err != nil {
		return nil // already gone
	}
	return app.Delete(col)
}
