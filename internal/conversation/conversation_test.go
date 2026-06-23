package conversation

import (
	"testing"

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

// TestAppendOriginRecReturnsRecord verifies AppendOriginRec returns a non-nil
// record whose id and content round-trip through the DB.
// TestRecentTurnsExcludesRuntimeOrigins: a caught fabrication and the runtime's
// honesty note stay in the record but are barred from the context window, so a
// lie is never replayed to the model as a pattern.
func TestRecentTurnsExcludesRuntimeOrigins(t *testing.T) {
	app := storetest.NewApp(t)
	master, _ := Master(app)

	add := func(role, content, origin string) {
		if err := AppendOrigin(app, master.Id, llm.Message{Role: role, Content: content}, "", origin); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	add("user", "add a task to clean the dishes", "")
	add("assistant", "Task saved: clean the dishes", OriginUncommitted) // the lie
	add("assistant", "Runtime check: nothing was saved.", OriginCheck)  // the honesty note
	add("user", "thanks", "")
	add("assistant", "Anytime.", "")

	got, err := RecentTurns(app, master.Id, 10)
	if err != nil {
		t.Fatalf("RecentTurns: %v", err)
	}
	want := []llm.Message{
		{Role: "user", Content: "add a task to clean the dishes"},
		{Role: "user", Content: "thanks"},
		{Role: "assistant", Content: "Anytime."},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d turns, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Role != want[i].Role || got[i].Content != want[i].Content {
			t.Errorf("turn %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	// History keeps the full record — the owner still sees what was said.
	hist, _ := History(app, master.Id, 50)
	if len(hist) != 5 {
		t.Fatalf("history = %d records, want 5 (nothing dropped from the record)", len(hist))
	}
}

func TestAppendOriginRecReturnsRecord(t *testing.T) {
	app := storetest.NewApp(t)
	master, _ := Master(app)

	const content = "\x00balaur-uicard:quests?\nshowing the owner the Quests card"
	rec, err := AppendOriginRec(app, master.Id,
		llm.Message{Role: "tool", Content: content}, "quests", "")
	if err != nil {
		t.Fatalf("AppendOriginRec: %v", err)
	}
	if rec == nil {
		t.Fatal("AppendOriginRec returned nil record")
	}
	if rec.Id == "" {
		t.Error("returned record has empty id")
	}
	if got := rec.GetString("content"); got != content {
		t.Errorf("content = %q, want %q", got, content)
	}
	if got := rec.GetString("role"); got != "tool" {
		t.Errorf("role = %q, want %q", got, "tool")
	}
	// origin="" is the contract that sidesteps chatNudges (origin != '').
	if got := rec.GetString("origin"); got != "" {
		t.Errorf("origin = %q, want empty (sidesteps chatNudges)", got)
	}
	// Confirm it persisted: load from DB and compare id.
	loaded, err := app.FindRecordById("messages", rec.Id)
	if err != nil {
		t.Fatalf("FindRecordById after AppendOriginRec: %v", err)
	}
	if loaded.GetString("content") != content {
		t.Errorf("DB content = %q, want %q", loaded.GetString("content"), content)
	}
}
