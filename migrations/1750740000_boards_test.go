package migrations_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestBoardsCollectionExists(t *testing.T) {
	app := storetest.NewApp(t)

	// Collection must exist.
	col, err := app.FindCollectionByNameOrId("boards")
	if err != nil {
		t.Fatalf("boards collection missing: %v", err)
	}

	// Required fields must be present.
	for _, field := range []string{"name", "cards", "sort", "created", "updated"} {
		if col.Fields.GetByName(field) == nil {
			t.Errorf("boards field %q missing", field)
		}
	}

	// Index must exist.
	const idxName = "idx_boards_sort"
	var name string
	err = app.DB().
		NewQuery("SELECT name FROM sqlite_master WHERE type='index' AND name={:n}").
		Bind(map[string]any{"n": idxName}).Row(&name)
	if err != nil || name != idxName {
		t.Errorf("index %s missing (err=%v)", idxName, err)
	}
}
