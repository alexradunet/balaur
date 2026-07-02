package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pocketbase/pocketbase/apis"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"

	"github.com/alexradunet/balaur/internal/conversation"
	"github.com/alexradunet/balaur/internal/llm"
	"github.com/alexradunet/balaur/internal/llmtest"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/turn"
	_ "github.com/alexradunet/balaur/migrations"
)

// messengerBlockingClient implements llm.Client for in-flight guard tests.
// ChatStream blocks until release is closed (or the context is cancelled),
// and signals started (once) when it is first entered — at that point the
// handler has already acquired the turn guard via turn.TryBegin.
type messengerBlockingClient struct {
	release chan struct{}
	started chan struct{} // closed once on first ChatStream entry
	once    sync.Once
}

func (bc *messengerBlockingClient) ChatStream(ctx context.Context, _ []llm.Message, _ []llm.ToolSpec) (<-chan llm.Chunk, error) {
	bc.once.Do(func() { close(bc.started) })
	ch := make(chan llm.Chunk, 1)
	go func() {
		select {
		case <-bc.release:
		case <-ctx.Done():
		}
		ch <- llm.Chunk{Content: "done", Done: true}
		close(ch)
	}()
	return ch, nil
}

func (bc *messengerBlockingClient) Embed(_ context.Context, _ []string) ([][]float32, error) {
	return nil, nil
}

// buildMessengerRouter constructs the full PocketBase router (same pattern
// as TestNudgePollFilter and TestGuardRejectsNonLoopbackHost) and returns the
// app and http.Handler.
func buildMessengerRouter(t *testing.T) (*tests.TestApp, http.Handler) {
	t.Helper()
	app := newWebApp(t)
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
	return app, mux
}

// postMessenger fires POST /api/messenger/turn. auth="" omits the header.
func postMessenger(mux http.Handler, host, auth, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/messenger/turn",
		strings.NewReader(body))
	req.Host = host
	req.Header.Set("Content-Type", "application/json")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	return w
}

// countMessages counts persisted messages for a conversation; used to verify
// a turn did (or did not) append to the master conversation.
func countMessages(app core.App, convID string) (int, error) {
	recs, err := app.FindRecordsByFilter(
		"messages", "conversation = {:id}", "-created", 0, 0,
		map[string]any{"id": convID},
	)
	if err != nil {
		return 0, err
	}
	return len(recs), nil
}

// TestMessengerDisabledByDefault: no messenger_token set → 403, no turn run.
func TestMessengerDisabledByDefault(t *testing.T) {
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

	w := postMessenger(mux, "example.com", "Bearer anysecret", `{"message":"hello"}`)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403; body: %s", w.Code, w.Body)
	}
	if !strings.Contains(w.Body.String(), "not enabled") {
		t.Errorf("body must mention 'not enabled'; got: %s", w.Body)
	}

	after, err := countMessages(app, master.Id)
	if err != nil {
		t.Fatalf("count after: %v", err)
	}
	if after != before {
		t.Errorf("no turn must run: message count changed %d→%d", before, after)
	}
}

// TestMessengerAuthRequired: token set, bad/missing header → 401, no turn.
func TestMessengerAuthRequired(t *testing.T) {
	const token = "super-secret-tok"
	cases := []struct {
		name string
		auth string
	}{
		{"no auth header", ""},
		{"wrong token", "Bearer wrong-token"},
		{"missing Bearer prefix", token},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			app, mux := buildMessengerRouter(t)
			defer app.Cleanup()
			if err := store.SetOwnerSetting(app, "messenger_token", token); err != nil {
				t.Fatalf("SetOwnerSetting: %v", err)
			}
			master, err := conversation.Master(app)
			if err != nil {
				t.Fatalf("master: %v", err)
			}
			before, err := countMessages(app, master.Id)
			if err != nil {
				t.Fatalf("count before: %v", err)
			}

			w := postMessenger(mux, "example.com", tc.auth, `{"message":"hi"}`)
			if w.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401; body: %s", w.Code, w.Body)
			}

			after, err := countMessages(app, master.Id)
			if err != nil {
				t.Fatalf("count after: %v", err)
			}
			if after != before {
				t.Errorf("turn must not run on auth failure: count %d→%d", before, after)
			}
		})
	}
}

// TestMessengerHappyPath: correct token + correct header + valid message →
// 200 {"reply":"..."} and master conversation grows (turn persisted).
func TestMessengerHappyPath(t *testing.T) {
	const token = "correct-token"
	app, mux := buildMessengerRouter(t)
	defer app.Cleanup()

	seedScriptedModel(t, app, llmtest.Text("Hello from messenger"))
	if err := store.SetOwnerSetting(app, "messenger_token", token); err != nil {
		t.Fatalf("SetOwnerSetting: %v", err)
	}

	master, err := conversation.Master(app)
	if err != nil {
		t.Fatalf("master: %v", err)
	}
	before, err := countMessages(app, master.Id)
	if err != nil {
		t.Fatalf("count before: %v", err)
	}

	w := postMessenger(mux, "example.com", "Bearer "+token, `{"message":"hi"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"reply"`) {
		t.Errorf("response missing 'reply' key; body: %s", body)
	}
	if !strings.Contains(body, "Hello from messenger") {
		t.Errorf("reply missing expected text; body: %s", body)
	}

	after, err := countMessages(app, master.Id)
	if err != nil {
		t.Fatalf("count after: %v", err)
	}
	if after <= before {
		t.Errorf("turn must persist messages: count %d→%d (no growth)", before, after)
	}
}

// TestMessengerLoopbackGuard: a non-loopback host not in BALAUR_ALLOWED_HOSTS
// is rejected with 403; a loopback IP passes.
//
// This exercises the host check, which is a DNS-rebinding defense (the same one
// guardLocalUI gives /ui/*), NOT network-layer loopback isolation. The Host
// header is attacker-spoofable, so on a 0.0.0.0-binding box (e.g. the prod
// NetBird mesh) a peer can pass this check with Host: localhost — the Bearer
// token (see TestMessengerAuthRequired) is the real gate. messengerTurn calls
// the shared isAllowedHost helper inline because guardLocalUI bypasses /api/*.
func TestMessengerLoopbackGuard(t *testing.T) {
	const token = "any-token"
	app, mux := buildMessengerRouter(t)
	defer app.Cleanup()
	if err := store.SetOwnerSetting(app, "messenger_token", token); err != nil {
		t.Fatalf("SetOwnerSetting: %v", err)
	}

	// "evil.test" is not in BALAUR_ALLOWED_HOSTS (newWebApp sets it to "example.com").
	w := postMessenger(mux, "evil.test", "Bearer "+token, `{"message":"hi"}`)
	if w.Code != http.StatusForbidden {
		t.Errorf("evil.test: expected 403, got %d; body: %s", w.Code, w.Body)
	}

	// 127.0.0.1 is a loopback address and must pass the host guard.
	// No model is seeded — the request reaches the "no active model" path;
	// any non-403 proves the guard let it through.
	w = postMessenger(mux, "127.0.0.1", "Bearer "+token, `{"message":"hi"}`)
	if w.Code == http.StatusForbidden {
		t.Errorf("127.0.0.1 (loopback): must not be blocked by host guard, got 403")
	}
}

// TestMessengerInFlight: a second POST while the first is mid-turn returns 429.
//
// The messengerBlockingClient signals `started` once ChatStream is entered,
// meaning the handler has already acquired the turn guard (turn.TryBegin) by
// that point. The second request is fired after that signal and must receive
// 429 via the shared cross-surface guard.
func TestMessengerInFlight(t *testing.T) {
	const token = "inflight-token"

	release := make(chan struct{})
	bc := &messengerBlockingClient{
		release: release,
		started: make(chan struct{}),
	}

	app, mux := buildMessengerRouter(t)
	defer app.Cleanup()

	turn.SetTestClient(app, bc)
	activateLocalModel(t, app)
	if err := store.SetOwnerSetting(app, "messenger_token", token); err != nil {
		t.Fatalf("SetOwnerSetting: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		postMessenger(mux, "example.com", "Bearer "+token, `{"message":"first"}`)
	}()

	// Wait until the handler has entered ChatStream (and thus holds the turn
	// in-flight guard, turn.TryBegin).
	<-bc.started

	// Second request while first is mid-turn.
	w := postMessenger(mux, "example.com", "Bearer "+token, `{"message":"second"}`)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected 429 while turn in flight, got %d; body: %s", w.Code, w.Body)
	}

	// Unblock the first turn and wait for it to finish.
	close(release)
	wg.Wait()
}

// TestMessengerAuthThrottleLockout: 5 consecutive bad tokens lock the
// endpoint out; the 6th (bad or good) gets 429 with a distinct body from
// the turn guard's "busy".
func TestMessengerAuthThrottleLockout(t *testing.T) {
	app, mux := buildMessengerRouter(t)
	defer app.Cleanup()
	if err := store.SetOwnerSetting(app, "messenger_token", "right-token"); err != nil {
		t.Fatalf("SetOwnerSetting: %v", err)
	}

	for i := 0; i < messengerMaxFailures; i++ {
		w := postMessenger(mux, "example.com", "Bearer wrong", `{"message":"hi"}`)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401; body: %s", i+1, w.Code, w.Body)
		}
	}

	w := postMessenger(mux, "example.com", "Bearer wrong", `{"message":"hi"}`)
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("6th bad attempt: status = %d, want 429; body: %s", w.Code, w.Body)
	}
	if !strings.Contains(w.Body.String(), "too many failed auth attempts") {
		t.Errorf("429 body must mention throttle, not the turn guard's busy body; got: %s", w.Body)
	}

	// Even the correct token is rejected during lockout — no model is
	// seeded, so if this ever ran a turn it would surface as a non-429
	// status instead.
	w = postMessenger(mux, "example.com", "Bearer right-token", `{"message":"hi"}`)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("correct token during lockout: status = %d, want 429; body: %s", w.Code, w.Body)
	}
}

// TestMessengerAuthThrottleSuccessResets: a successful auth resets the
// failure counter, so a later run of failures below the limit never trips
// the lockout.
func TestMessengerAuthThrottleSuccessResets(t *testing.T) {
	const token = "right-token"
	app, mux := buildMessengerRouter(t)
	defer app.Cleanup()
	seedScriptedModel(t, app, llmtest.Text("ok"))
	if err := store.SetOwnerSetting(app, "messenger_token", token); err != nil {
		t.Fatalf("SetOwnerSetting: %v", err)
	}

	for i := 0; i < messengerMaxFailures-1; i++ {
		w := postMessenger(mux, "example.com", "Bearer wrong", `{"message":"hi"}`)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("bad attempt %d: status = %d, want 401; body: %s", i+1, w.Code, w.Body)
		}
	}

	w := postMessenger(mux, "example.com", "Bearer "+token, `{"message":"hi"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("correct token: status = %d, want 200; body: %s", w.Code, w.Body)
	}

	for i := 0; i < messengerMaxFailures; i++ {
		w := postMessenger(mux, "example.com", "Bearer wrong", `{"message":"hi"}`)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("post-reset bad attempt %d: status = %d, want 401 (not 429); body: %s", i+1, w.Code, w.Body)
		}
	}
}

// TestAuthThrottleCooldown is a unit test on authThrottle using the fake
// clock seam — no time.Sleep.
func TestAuthThrottleCooldown(t *testing.T) {
	now := time.Unix(0, 0)
	th := &authThrottle{now: func() time.Time { return now }}

	if !th.allow() {
		t.Fatalf("fresh throttle: allow() = false, want true")
	}

	for i := 0; i < messengerMaxFailures; i++ {
		th.fail()
	}
	if th.allow() {
		t.Fatalf("after %d failures: allow() = true, want false", messengerMaxFailures)
	}

	now = now.Add(messengerCooldown)
	if !th.allow() {
		t.Fatalf("after cooldown: allow() = false, want true")
	}
	th.fail()
	if !th.allow() {
		t.Fatalf("after cooldown reset + 1 failure: allow() = false, want true (counter was reset)")
	}

	for i := 0; i < messengerMaxFailures; i++ {
		th.fail()
	}
	th.success()
	if !th.allow() {
		t.Fatalf("after success(): allow() = false, want true")
	}
}
