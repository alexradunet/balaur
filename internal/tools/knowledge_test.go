package tools

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/recap"
	"github.com/alexradunet/balaur/internal/store"
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

// TestNodeListToolCapsOutput proves node_list stops at 50 entries and tells
// the model the listing is truncated (with the real total).
func TestNodeListToolCapsOutput(t *testing.T) {
	app := storetest.NewApp(t)
	for i := range 60 {
		if _, err := nodes.Create(app, "note", fmt.Sprintf("Note %02d", i), "", nodes.StatusActive, nil); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}
	tool := nodeListTool(app)

	out, err := tool.Execute(context.Background(), `{"type":"note"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	entries := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "- [") {
			entries++
		}
	}
	if entries != 50 {
		t.Fatalf("want exactly 50 entries, got %d", entries)
	}
	if !strings.Contains(out, "showing 50 of 60") {
		t.Fatalf("missing truncation note, got tail: %q", out[max(0, len(out)-120):])
	}
}

// TestNodeListToolSmallListNoTruncationNote proves an under-cap listing has
// no truncation line.
func TestNodeListToolSmallListNoTruncationNote(t *testing.T) {
	app := storetest.NewApp(t)
	for i := range 3 {
		if _, err := nodes.Create(app, "note", fmt.Sprintf("Note %d", i), "", nodes.StatusActive, nil); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}
	tool := nodeListTool(app)

	out, err := tool.Execute(context.Background(), `{"type":"note"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Count(out, "- [") != 3 {
		t.Fatalf("want 3 entries, got: %q", out)
	}
	if strings.Contains(out, "showing") {
		t.Fatalf("unexpected truncation note on a 3-item list: %q", out)
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

// TestNodeGetDayRecapFound proves node_get appends the day recap when a matching
// summary exists, routing through recap.Find rather than a raw "summaries" query.
func TestNodeGetDayRecapFound(t *testing.T) {
	app := storetest.NewApp(t)

	// Create a day node for 2026-01-15.
	dateKey := "2026-01-15"
	rec, err := nodes.Create(app, "day", "2026-01-15", "", nodes.StatusActive,
		map[string]any{"date": dateKey})
	if err != nil {
		t.Fatalf("Create day node: %v", err)
	}

	// Get the master conversation (created on first call).
	conv, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("Master: %v", err)
	}

	// Insert a matching day summary directly.
	loc := store.OwnerLocation(app)
	day, _ := time.ParseInLocation("2006-01-02", dateKey, loc)
	period := recap.Day(day)

	col, err := app.FindCollectionByNameOrId("summaries")
	if err != nil {
		t.Fatalf("find summaries collection: %v", err)
	}
	sum := core.NewRecord(col)
	sum.Set("conversation", conv.Id)
	sum.Set("period_type", period.Type)
	sum.Set("period_start", period.Start.UTC())
	sum.Set("period_end", period.End.UTC())
	sum.Set("content", "A productive day in January.")
	sum.Set("message_count", 2)
	if err := app.Save(sum); err != nil {
		t.Fatalf("save summary: %v", err)
	}

	tool := nodeGetTool(app)
	got, err := tool.Execute(context.Background(), `{"id":"`+rec.Id+`"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(got, "## Day recap") {
		t.Errorf("output missing '## Day recap'; got:\n%s", got)
	}
	if !strings.Contains(got, "A productive day in January.") {
		t.Errorf("output missing summary content; got:\n%s", got)
	}
}

// TestNodeGetDayRecapMissing proves node_get emits "No recap yet" when no
// matching day summary exists.
func TestNodeGetDayRecapMissing(t *testing.T) {
	app := storetest.NewApp(t)

	dateKey := "2026-01-16"
	rec, err := nodes.Create(app, "day", "2026-01-16", "", nodes.StatusActive,
		map[string]any{"date": dateKey})
	if err != nil {
		t.Fatalf("Create day node: %v", err)
	}

	tool := nodeGetTool(app)
	got, err := tool.Execute(context.Background(), `{"id":"`+rec.Id+`"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if !strings.Contains(got, "No recap yet for "+dateKey) {
		t.Errorf("output missing 'No recap yet for %s'; got:\n%s", dateKey, got)
	}
}
