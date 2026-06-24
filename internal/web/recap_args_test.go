package web

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/llm"
)

// TestMessageViewsArgsOnReload verifies the read-side correlation: a tool call's
// arguments are persisted on the assistant record's tool_payload, and on reload
// messageViews threads them onto the following tool-result row (which carries no
// args of its own), so the reloaded transcript shows the same args fold as a live
// turn.
func TestMessageViewsArgsOnReload(t *testing.T) {
	app := newWebApp(t)
	defer app.Cleanup()
	h := &handlers{app: app}

	conv, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master conversation: %v", err)
	}

	// The assistant turn that requested the call (empty content → skipped on
	// reload, but its tool_payload must still feed the queue).
	if err := conversation.Append(app, conv.Id, llm.Message{
		Role: "assistant",
		ToolCalls: []llm.ToolCall{
			{ID: "tc1", Name: "task_add", Args: `{"title":"water the tomatoes"}`},
		},
	}, ""); err != nil {
		t.Fatalf("append assistant tool-call: %v", err)
	}
	// The matching tool-result row (no args persisted on this record).
	if err := conversation.Append(app, conv.Id, llm.Message{
		Role: "tool", Content: "added task", ToolCallID: "tc1",
	}, "task_add"); err != nil {
		t.Fatalf("append tool result: %v", err)
	}

	recs, err := conversation.History(app, conv.Id, 60)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	views := h.messageViews(recs)

	var tool *messageView
	for i := range views {
		if views[i].Role == "tool" {
			tool = &views[i]
			break
		}
	}
	if tool == nil {
		t.Fatal("no tool view produced from history")
	}
	if !strings.Contains(tool.Args, "water the tomatoes") || !strings.Contains(tool.Args, `"title"`) {
		t.Errorf("tool view Args missing the persisted arguments: %q", tool.Args)
	}

	// End to end: the rendered tool row carries the collapsed args fold.
	out := renderNodeHTML(h.renderMessages(views))
	if !strings.Contains(out, `class="tool-args"`) {
		t.Errorf("reloaded tool row missing the args fold:\n%s", out)
	}
	if !strings.Contains(out, "water the tomatoes") {
		t.Errorf("reloaded tool row missing the argument value:\n%s", out)
	}
}
