package web

import (
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/tools"
)

// A refresh-marked tool result loaded from history shows the plain text and
// never leaks the raw NUL marker (the live refresh has no meaning on reload).
func TestMessageViewsStripsRefreshMarker(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}

	col, err := app.FindCollectionByNameOrId("messages")
	if err != nil {
		t.Fatalf("messages collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("role", "tool")
	rec.Set("tool_name", "task_done")
	rec.Set("content", tools.MarkRefresh([]string{"today"}, `Done: "Buy milk".`))

	views := h.messageViews([]*core.Record{rec})
	if len(views) != 1 {
		t.Fatalf("want 1 view, got %d", len(views))
	}
	got := views[0].Content
	if strings.Contains(got, "balaur-refresh") || strings.Contains(got, "\x00") {
		t.Fatalf("raw marker leaked into history: %q", got)
	}
	if !strings.Contains(got, `Done: "Buy milk".`) {
		t.Fatalf("plain text missing from history: %q", got)
	}
}
