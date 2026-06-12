package migrations_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestAPIKeyIsHiddenFromREST(t *testing.T) {
	app := storetest.NewApp(t)
	col, err := app.FindCollectionByNameOrId("llm_providers")
	if err != nil {
		t.Fatalf("llm_providers collection: %v", err)
	}
	field := col.Fields.GetByName("api_key")
	if field == nil {
		t.Fatal("api_key field not found")
	}
	if !field.GetHidden() {
		t.Error("api_key field must be hidden (GetHidden() == true)")
	}
}
