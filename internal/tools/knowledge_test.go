package tools

import (
	"context"
	"testing"

	"github.com/pocketbase/dbx"

	"github.com/alexradunet/balaur/internal/storetest"
)

func TestRememberToolAcceptsStringFallback(t *testing.T) {
	app := storetest.NewApp(t)
	tool := rememberTool(app)

	got, err := tool.Execute(context.Background(), `"My name is Alex"`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if got == "" {
		t.Fatal("expected non-empty tool response")
	}

	rec, err := app.FindFirstRecordByFilter("memories", "title = {:title}", dbx.Params{"title": "My name is Alex"})
	if err != nil {
		t.Fatalf("FindFirstRecordByFilter: %v", err)
	}
	if got := rec.GetString("content"); got != "My name is Alex" {
		t.Fatalf("content = %q, want %q", got, "My name is Alex")
	}
	if got := rec.GetString("status"); got != "proposed" {
		t.Fatalf("status = %q, want proposed", got)
	}
	if got := rec.GetString("category"); got != "fact" {
		t.Fatalf("category = %q, want fact", got)
	}
	if got := rec.GetInt("importance"); got != 3 {
		t.Fatalf("importance = %d, want 3", got)
	}
}
