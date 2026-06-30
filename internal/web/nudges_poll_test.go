package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/llm"
)

// TestNudgePollFilter is a regression test for the chatNudges handler:
// honesty-check artifacts (origin "uncommitted"/"check") must not be returned
// by the poller — only agent-initiated origins "nudge" and "briefing" are
// surfaced.  Before the fix the filter was `origin != ”`, which leaked the
// runtime artifacts back into the chat after every turn where the honesty
// check fired.
func TestNudgePollFilter(t *testing.T) {
	app := newWebApp(t)
	defer app.Cleanup()

	conv, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master conversation: %v", err)
	}

	// Record cursor before appending so all test messages land after it.
	since := time.Now().Add(-time.Second).UnixMilli()

	type msg struct {
		content string
		origin  string
	}
	msgs := []msg{
		{"honesty check note", conversation.OriginCheck},
		{"uncommitted draft", conversation.OriginUncommitted},
		{"nudge hello", "nudge"},
		{"morning briefing", "briefing"},
	}
	for _, m := range msgs {
		if err := conversation.AppendOrigin(app, conv.Id,
			llm.Message{Role: "assistant", Content: m.content}, "", m.origin); err != nil {
			t.Fatalf("AppendOrigin origin=%q: %v", m.origin, err)
		}
	}

	// Build the router (same pattern as TestGuardRejectsNonLoopbackHost).
	baseRouter, err := apis.NewRouter(app)
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	se := &core.ServeEvent{App: app, Router: baseRouter}
	if err := app.OnServe().Trigger(se, func(e *core.ServeEvent) error { return nil }); err != nil {
		t.Fatalf("OnServe trigger: %v", err)
	}
	mux, err := se.Router.BuildMux()
	if err != nil {
		t.Fatalf("BuildMux: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/ui/chat/nudges?since=%d", since), nil)
	req.Host = "example.com" // newWebApp sets BALAUR_ALLOWED_HOSTS=example.com
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body:\n%s", w.Code, w.Body.String())
	}
	body := w.Body.String()

	// nudge and briefing must appear in the SSE response.
	for _, want := range []string{"nudge hello", "morning briefing"} {
		if !strings.Contains(body, want) {
			t.Errorf("response missing %q; body:\n%s", want, body)
		}
	}
	// Honesty-check artifacts must be absent.
	for _, notWant := range []string{"honesty check note", "uncommitted draft"} {
		if strings.Contains(body, notWant) {
			t.Errorf("response must not contain %q (honesty artifact leaked); body:\n%s", notWant, body)
		}
	}
}
