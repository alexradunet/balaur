package web

import (
	"testing"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/tools/types"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/llm"
)

// TestDockDataTodayOnly verifies the home dock's inline transcript is TODAY
// only — older turns collapse into the recap telescope (HasRecap) rather than
// bleeding raw "further back" text into the scroll.
func TestDockDataTodayOnly(t *testing.T) {
	app := newWebApp(t)
	defer app.Cleanup()
	h := &handlers{app: app}

	conv, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master conversation: %v", err)
	}

	// One turn today (stays inline) + one backdated to yesterday (telescopes).
	for _, content := range []string{"today's question", "yesterday's question"} {
		if err := conversation.Append(app, conv.Id, llm.Message{Role: "user", Content: content}, ""); err != nil {
			t.Fatalf("append %q: %v", content, err)
		}
	}
	recs, err := app.FindRecordsByFilter("messages", "content = {:c}", "", 1, 0,
		dbx.Params{"c": "yesterday's question"})
	if err != nil || len(recs) == 0 {
		t.Fatalf("find yesterday msg: %v (n=%d)", err, len(recs))
	}
	yesterday := time.Now().AddDate(0, 0, -1)
	if _, err := app.DB().NewQuery("UPDATE messages SET created = {:at} WHERE id = {:id}").
		Bind(dbx.Params{"at": yesterday.UTC().Format(types.DefaultDateLayout), "id": recs[0].Id}).
		Execute(); err != nil {
		t.Fatalf("backdate: %v", err)
	}

	data, err := h.dockData()
	if err != nil {
		t.Fatalf("dockData: %v", err)
	}
	if len(data.History) != 1 {
		t.Fatalf("inline History = %d turns, want 1 (today only)", len(data.History))
	}
	if data.History[0].Content != "today's question" {
		t.Errorf("inline turn = %q, want today's", data.History[0].Content)
	}
	if !data.HasRecap {
		t.Error("HasRecap = false, want true (yesterday's turn predates today)")
	}
}
