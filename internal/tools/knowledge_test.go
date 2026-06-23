package tools

import (
	"context"
	"testing"

	"github.com/pocketbase/dbx"

	"github.com/alexradunet/balaur/internal/nodes"
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

	// remember proposes a type=memory node: content lands in body, category and
	// importance in props.
	rec, err := app.FindFirstRecordByFilter("nodes",
		"type = {:t} && title = {:title}",
		dbx.Params{"t": "memory", "title": "My name is Alex"})
	if err != nil {
		t.Fatalf("FindFirstRecordByFilter: %v", err)
	}
	if got := rec.GetString("body"); got != "My name is Alex" {
		t.Fatalf("body = %q, want %q", got, "My name is Alex")
	}
	if got := rec.GetString("status"); got != "proposed" {
		t.Fatalf("status = %q, want proposed", got)
	}
	if got := nodes.PropString(rec, "category"); got != "fact" {
		t.Fatalf("category = %q, want fact", got)
	}
	if got := nodes.PropInt(rec, "importance"); got != 3 {
		t.Fatalf("importance = %d, want 3", got)
	}
}

// TestNodeWriteToolCreatesActiveNode proves node_write creates an owner-authored
// node born active (not proposed) and audited.
func TestNodeWriteToolCreatesActiveNode(t *testing.T) {
	app := storetest.NewApp(t)
	tool := nodeWriteTool(app)

	if _, err := tool.Execute(context.Background(),
		`{"type":"note","title":"Garden plan","body":"south wall first"}`); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	rec, err := app.FindFirstRecordByFilter("nodes",
		"type = {:t} && title = {:title}",
		dbx.Params{"t": "note", "title": "Garden plan"})
	if err != nil {
		t.Fatalf("FindFirstRecordByFilter: %v", err)
	}
	if got := rec.GetString("status"); got != nodes.StatusActive {
		t.Fatalf("status = %q, want active (owner-authored)", got)
	}
	if got := rec.GetString("body"); got != "south wall first" {
		t.Fatalf("body = %q", got)
	}
}

func TestRememberToolRejectsBadJSON(t *testing.T) {
	app := storetest.NewApp(t)
	tool := rememberTool(app)
	if _, err := tool.Execute(context.Background(), `{bad json`); err == nil {
		t.Fatal("expected an error for malformed JSON args")
	}
}

func TestRememberToolRejectsEmptyTitle(t *testing.T) {
	app := storetest.NewApp(t)
	tool := rememberTool(app)
	if _, err := tool.Execute(context.Background(), `{"content":"x","category":"fact","importance":3}`); err == nil {
		t.Fatal("expected an error when the title is empty")
	}
}
