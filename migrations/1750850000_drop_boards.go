package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

// drop boards (plan 089) — the boards surface and the board_compose/board_add_card
// tools were retired in favor of the single-page chat+sidebar + in-conversation
// artifacts. Owner decision: drop the table completely; do not preserve data.
func init() {
	m.Register(dropBoardsUp, dropBoardsDown)
}

func dropBoardsUp(app core.App) error {
	col, err := app.FindCollectionByNameOrId("boards")
	if err != nil {
		return nil // already gone — idempotent
	}
	return app.Delete(col)
}

// dropBoardsDown recreates the boards collection (schema mirrored from
// 1750740000_boards.go) for reversibility — data is NOT restored.
func dropBoardsDown(app core.App) error {
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
