package tools

import (
	"context"
	"strings"
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

	// remember proposes a type=memory node: content lands in body, importance
	// in props.
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

// TestNodeWriteToolPersistsProps proves node_write now threads typed props
// through to the saved node (the bug this plan fixes: props were hard-coded nil).
func TestNodeWriteToolPersistsProps(t *testing.T) {
	app := storetest.NewApp(t)
	tool := nodeWriteTool(app)

	if _, err := tool.Execute(context.Background(),
		`{"type":"book","title":"LHoD","body":"","props":{"author":"Le Guin","year":1969}}`); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	rec, err := app.FindFirstRecordByFilter("nodes",
		"type = {:t} && title = {:title}",
		dbx.Params{"t": "book", "title": "LHoD"})
	if err != nil {
		t.Fatalf("FindFirstRecordByFilter: %v", err)
	}
	if a := nodes.PropString(rec, "author"); a != "Le Guin" {
		t.Errorf("PropString(author) = %q, want Le Guin", a)
	}
}

// TestNodeWriteToolRejectsInvalidProps proves a type-mismatched prop is rejected
// and the error steers the model to node_schema.
func TestNodeWriteToolRejectsInvalidProps(t *testing.T) {
	app := storetest.NewApp(t)
	tool := nodeWriteTool(app)

	_, err := tool.Execute(context.Background(),
		`{"type":"book","title":"X","props":{"year":"nineteen"}}`)
	if err == nil {
		t.Fatal("expected an error for a string where the schema wants a number")
	}
	if !strings.Contains(err.Error(), "node_schema") {
		t.Errorf("error %q should steer the model to node_schema", err.Error())
	}
}

// TestNodeEditToolUpdatesProps proves node_edit changes title and props in place.
func TestNodeEditToolUpdatesProps(t *testing.T) {
	app := storetest.NewApp(t)
	rec, err := nodes.Create(app, "book", "Old", "", nodes.StatusActive,
		map[string]any{"author": "A", "year": 1969.0})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	tool := nodeEditTool(app)
	if _, err := tool.Execute(context.Background(),
		`{"id":"`+rec.Id+`","title":"New","props":{"author":"Z","year":2000}}`); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got, err := app.FindRecordById("nodes", rec.Id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if got.GetString("title") != "New" {
		t.Errorf("title = %q, want New", got.GetString("title"))
	}
	if a := nodes.PropString(got, "author"); a != "Z" {
		t.Errorf("PropString(author) = %q, want Z", a)
	}
}

// TestNodeEditToolRejectsMissingNode proves node_edit errors cleanly on a bad id.
func TestNodeEditToolRejectsMissingNode(t *testing.T) {
	app := storetest.NewApp(t)
	tool := nodeEditTool(app)
	if _, err := tool.Execute(context.Background(), `{"id":"nonexistentid","title":"x"}`); err == nil {
		t.Fatal("expected an error editing a node that does not exist")
	}
}

// TestKnowledgeToolsIncludesNodeEdit proves node_edit is registered in the
// model's knowledge verb set.
func TestKnowledgeToolsIncludesNodeEdit(t *testing.T) {
	app := storetest.NewApp(t)
	for _, tool := range KnowledgeTools(app) {
		if tool.Spec.Name == "node_edit" {
			return
		}
	}
	t.Fatal("KnowledgeTools missing tool \"node_edit\"")
}
