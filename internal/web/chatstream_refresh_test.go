package web

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/starfederation/datastar-go/datastar"

	"github.com/alexradunet/balaur/internal/agent"
	"github.com/alexradunet/balaur/internal/tools"
)

// refreshCard re-renders a card from live data and patches it by its
// ucard-{type} root id into the open stream.
func TestRefreshCardPatchesToday(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/ui/chat", nil)
	cs := &chatStream{sse: datastar.NewSSE(rec, req), h: h}

	cs.refreshCard("today")

	if body := rec.Body.String(); !strings.Contains(body, "ucard-today") {
		t.Fatalf("expected a patch carrying id ucard-today, got:\n%s", body)
	}
}

// A refresh-marked tool result morphs the tool row AND patches the named card.
func TestHandleToolResultRefreshRoutes(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app, tmpl: parseTemplates(t)}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/ui/chat", nil)
	cs := &chatStream{
		sse: datastar.NewSSE(rec, req), h: h,
		toolName: "task_done", toolID: "tool-x", toolBody: "tool-x-body",
	}

	cs.handleToolResult(agent.Event{
		Kind: "tool_result",
		Text: tools.MarkRefresh([]string{"today"}, `Done: "Buy milk".`),
	})

	body := rec.Body.String()
	if !strings.Contains(body, "ucard-today") {
		t.Fatalf("refresh did not patch the today card:\n%s", body)
	}
	if !strings.Contains(body, "Done") {
		t.Fatalf("tool-row text missing:\n%s", body)
	}
}
