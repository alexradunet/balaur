package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/llmtest"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
)

// TestChatBusyToast: while a turn is in flight (the test itself holds the
// cross-surface guard), POST /ui/chat must deliver only a warn toast over a
// minimal SSE stream — no user bubble, no #chat mutation, and no persisted
// message. Pins the busy branch at internal/web/chat.go:42-50.
func TestChatBusyToast(t *testing.T) {
	app, mux := buildMessengerRouter(t)
	defer app.Cleanup()

	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master conversation: %v", err)
	}
	before, err := countMessages(app, master.Id)
	if err != nil {
		t.Fatalf("count before: %v", err)
	}

	// Simulate an in-flight turn: hold the process-wide guard for the
	// duration of the request. No model is needed — the handler rejects
	// before resolving a client.
	end, ok := turn.TryBegin()
	if !ok {
		t.Fatal("turn guard unexpectedly already held")
	}
	defer end()

	req := httptest.NewRequest(http.MethodPost, "/ui/chat",
		strings.NewReader("message=hello"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("busy SSE must open with 200, got %d; body: %s", w.Code, w.Body)
	}
	body := w.Body.String()
	if !strings.Contains(body, "One message is still being answered") {
		t.Errorf("busy response missing the toast text; body: %s", body)
	}
	if !strings.Contains(body, "toast-region") {
		t.Errorf("toast must target #toast-region; body: %s", body)
	}
	if strings.Contains(body, "cmsg cmsg-user") {
		t.Errorf("busy response must not paint a user bubble; body: %s", body)
	}

	after, err := countMessages(app, master.Id)
	if err != nil {
		t.Fatalf("count after: %v", err)
	}
	if after != before {
		t.Errorf("busy rejection must persist nothing: message count %d→%d", before, after)
	}
}

// TestMessengerBadBodyReleasesGuard: the messenger handler acquires the
// cross-surface guard BEFORE parsing the body (messenger.go steps 4→5), so a
// 400 must release it via the defer — otherwise one malformed bridge request
// would wedge every surface busy forever. A valid follow-up request must
// therefore run a full turn (200), never 429.
func TestMessengerBadBodyReleasesGuard(t *testing.T) {
	const token = "release-check-token"
	cases := []struct {
		name string
		body string
	}{
		{"malformed json", `{`},
		{"empty message", `{}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, mux := buildMessengerRouter(t)
			defer app.Cleanup()
			seedScriptedModel(t, app, llmtest.Text("guard was released"))
			if err := store.SetOwnerSetting(app, "messenger_token", token); err != nil {
				t.Fatalf("SetOwnerSetting: %v", err)
			}

			w := postMessenger(mux, "example.com", "Bearer "+token, tc.body)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("bad body: status = %d, want 400; body: %s", w.Code, w.Body)
			}

			// Immediately-following valid request: 429 here means the 400
			// branch leaked the guard.
			w2 := postMessenger(mux, "example.com", "Bearer "+token, `{"message":"hi"}`)
			if w2.Code == http.StatusTooManyRequests {
				t.Fatalf("guard not released after 400: follow-up got 429; body: %s", w2.Body)
			}
			if w2.Code != http.StatusOK {
				t.Fatalf("follow-up: status = %d, want 200; body: %s", w2.Code, w2.Body)
			}
			if !strings.Contains(w2.Body.String(), "guard was released") {
				t.Errorf("follow-up reply missing scripted text; body: %s", w2.Body)
			}
		})
	}
}

// TestMessengerIPv6LoopbackHost: Host "[::1]:8090" must pass the host guard —
// net.SplitHostPort strips the brackets/port and isAllowedHost accepts the
// loopback IP. No model is seeded, so any non-403 proves the guard let the
// request through (same technique as TestMessengerLoopbackGuard).
func TestMessengerIPv6LoopbackHost(t *testing.T) {
	const token = "v6-token"
	app, mux := buildMessengerRouter(t)
	defer app.Cleanup()
	if err := store.SetOwnerSetting(app, "messenger_token", token); err != nil {
		t.Fatalf("SetOwnerSetting: %v", err)
	}

	w := postMessenger(mux, "[::1]:8090", "Bearer "+token, `{"message":"hi"}`)
	if w.Code == http.StatusForbidden {
		t.Errorf("[::1]:8090 (IPv6 loopback): must not be blocked by host guard, got 403; body: %s", w.Body)
	}
}
