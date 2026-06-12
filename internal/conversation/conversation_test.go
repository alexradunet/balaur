package conversation

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/storetest"
)

func TestMasterIsSingleton(t *testing.T) {
	app := storetest.NewApp(t)

	first, err := Master(app)
	if err != nil {
		t.Fatalf("Master: %v", err)
	}
	second, err := Master(app)
	if err != nil {
		t.Fatalf("Master again: %v", err)
	}
	if first.Id != second.Id {
		t.Fatalf("expected one master, got %s and %s", first.Id, second.Id)
	}
	if first.GetString("kind") != "master" || first.GetString("status") != "open" {
		t.Fatalf("unexpected master shape: kind=%s status=%s",
			first.GetString("kind"), first.GetString("status"))
	}
}

func TestAppendAndRecentTurnsRoundtrip(t *testing.T) {
	app := storetest.NewApp(t)
	master, _ := Master(app)

	turns := []struct {
		msg  llm.Message
		tool string
	}{
		{llm.Message{Role: "user", Content: "hello"}, ""},
		{llm.Message{Role: "assistant", Content: "", ToolCalls: []llm.ToolCall{{ID: "c1", Name: "recall", Args: "{}"}}}, ""},
		{llm.Message{Role: "tool", Content: "found nothing", ToolCallID: "c1"}, "recall"},
		{llm.Message{Role: "assistant", Content: "hi there"}, ""},
		{llm.Message{Role: "user", Content: "how are you"}, ""},
		{llm.Message{Role: "assistant", Content: "well, thank you"}, ""},
	}
	for _, tt := range turns {
		if err := Append(app, master.Id, tt.msg, tt.tool); err != nil {
			t.Fatalf("Append(%s): %v", tt.msg.Role, err)
		}
	}

	// RecentTurns: text turns only, chronological, no tool rounds, no
	// empty assistant turns.
	got, err := RecentTurns(app, master.Id, 10)
	if err != nil {
		t.Fatalf("RecentTurns: %v", err)
	}
	want := []llm.Message{
		{Role: "user", Content: "hello"},
		{Role: "assistant", Content: "hi there"},
		{Role: "user", Content: "how are you"},
		{Role: "assistant", Content: "well, thank you"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d turns, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Role != want[i].Role || got[i].Content != want[i].Content {
			t.Fatalf("turn %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	// Window limit keeps the most recent turns, still chronological.
	last2, _ := RecentTurns(app, master.Id, 2)
	if len(last2) != 2 || last2[0].Content != "how are you" || last2[1].Content != "well, thank you" {
		t.Fatalf("window wrong: %+v", last2)
	}

	// History keeps everything, including the tool round.
	hist, err := History(app, master.Id, 50)
	if err != nil {
		t.Fatalf("History: %v", err)
	}
	if len(hist) != 6 {
		t.Fatalf("history = %d records, want 6", len(hist))
	}
	if hist[2].GetString("role") != "tool" || hist[2].GetString("tool_name") != "recall" {
		t.Fatalf("tool round not preserved: role=%s tool=%s",
			hist[2].GetString("role"), hist[2].GetString("tool_name"))
	}
}

// seedHead creates an active head record in the test app.
func seedHead(t *testing.T, app core.App, name, status string) *core.Record {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("heads")
	if err != nil {
		t.Fatalf("heads collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("name", name)
	rec.Set("status", status)
	rec.SetEmail(fmt.Sprintf("head-%d@balaur.local", time.Now().UnixNano()))
	rec.SetRandomPassword()
	if err := app.Save(rec); err != nil {
		t.Fatalf("saving head: %v", err)
	}
	return rec
}

func TestForHead(t *testing.T) {
	app := storetest.NewApp(t)
	head := seedHead(t, app, "Scout", "active")

	// First call: creates the branch conversation.
	conv, err := ForHead(app, head)
	if err != nil {
		t.Fatalf("ForHead: %v", err)
	}
	if conv.GetString("kind") != "branch" {
		t.Errorf("kind = %q, want branch", conv.GetString("kind"))
	}
	if conv.GetString("status") != "open" {
		t.Errorf("status = %q, want open", conv.GetString("status"))
	}
	if conv.GetString("head") != head.Id {
		t.Errorf("head = %q, want %q", conv.GetString("head"), head.Id)
	}

	master, err := Master(app)
	if err != nil {
		t.Fatalf("Master: %v", err)
	}
	if conv.GetString("parent") != master.Id {
		t.Errorf("parent = %q, want master id %q", conv.GetString("parent"), master.Id)
	}

	// Second call: returns the same record (idempotent).
	conv2, err := ForHead(app, head)
	if err != nil {
		t.Fatalf("ForHead second: %v", err)
	}
	if conv.Id != conv2.Id {
		t.Errorf("ForHead created a second record: %s vs %s", conv.Id, conv2.Id)
	}

	// Exactly one branch conversation exists.
	branches, err := app.FindRecordsByFilter("conversations", "kind = 'branch'", "", 0, 0)
	if err != nil {
		t.Fatalf("listing branches: %v", err)
	}
	if len(branches) != 1 {
		t.Errorf("want 1 branch conversation, got %d", len(branches))
	}
}

func TestForHeadConcurrentCreate(t *testing.T) {
	app := storetest.NewApp(t)
	head := seedHead(t, app, "Concurrent", "active")

	const goroutines = 8
	ids := make([]string, goroutines)
	errs := make([]error, goroutines)

	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(i int) {
			defer wg.Done()
			conv, err := ForHead(app, head)
			if err != nil {
				errs[i] = err
				return
			}
			ids[i] = conv.Id
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d returned error: %v", i, err)
		}
	}

	// All goroutines must have returned the same conversation id.
	var firstID string
	for i, id := range ids {
		if id == "" {
			continue
		}
		if firstID == "" {
			firstID = id
		} else if id != firstID {
			t.Errorf("goroutine %d returned different id %s (want %s)", i, id, firstID)
		}
	}

	// Exactly one open branch conversation must exist afterward.
	open, err := app.FindRecordsByFilter("conversations",
		"kind = 'branch' && status = 'open' && head = {:head}",
		"", 0, 0, map[string]any{"head": head.Id})
	if err != nil {
		t.Fatalf("listing open branch conversations: %v", err)
	}
	if len(open) != 1 {
		t.Errorf("want exactly 1 open branch conversation, got %d", len(open))
	}
}
