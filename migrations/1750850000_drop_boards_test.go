package migrations_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestBoardsCollectionDropped(t *testing.T) {
	app := storetest.NewApp(t) // runs all registered migrations
	if _, err := app.FindCollectionByNameOrId("boards"); err == nil {
		t.Fatal("boards collection still exists after drop migration")
	}
}
