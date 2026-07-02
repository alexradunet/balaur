package bridge

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testWait = 5 * time.Second

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

// fakeTelegram serves scripted getUpdates batches (empty once exhausted) and
// records every getUpdates offset and every sendMessage payload onto
// channels so tests can synchronize without time.Sleep.
type fakeTelegram struct {
	mu      sync.Mutex
	batches [][]tgUpdate
	idx     int

	polls chan string      // offset query param on each getUpdates call
	sent  chan sentMessage // decoded sendMessage bodies
}

type sentMessage struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

func newFakeTelegram(batches [][]tgUpdate) *fakeTelegram {
	return &fakeTelegram{
		batches: batches,
		polls:   make(chan string, 256),
		sent:    make(chan sentMessage, 16),
	}
}

func (f *fakeTelegram) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getUpdates"):
			f.mu.Lock()
			var result []tgUpdate
			if f.idx < len(f.batches) {
				result = f.batches[f.idx]
				f.idx++
			}
			f.mu.Unlock()
			select {
			case f.polls <- r.URL.Query().Get("offset"):
			default:
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(tgUpdatesResponse{OK: true, Result: result})
		case strings.HasSuffix(r.URL.Path, "/sendMessage"):
			var msg sentMessage
			_ = json.NewDecoder(r.Body).Decode(&msg)
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]bool{"ok": true})
			f.sent <- msg
		default:
			http.NotFound(w, r)
		}
	}
}

// waitForPoll blocks for the next getUpdates offset. If want is non-empty it
// fails the test on a mismatch.
func waitForPoll(t *testing.T, tg *fakeTelegram, want string) string {
	t.Helper()
	select {
	case v := <-tg.polls:
		if want != "" && v != want {
			t.Fatalf("getUpdates offset = %q, want %q", v, want)
		}
		return v
	case <-time.After(testWait):
		t.Fatal("timed out waiting for a getUpdates poll")
		return ""
	}
}

func waitForSend(t *testing.T, tg *fakeTelegram) sentMessage {
	t.Helper()
	select {
	case msg := <-tg.sent:
		return msg
	case <-time.After(testWait):
		t.Fatal("timed out waiting for sendMessage")
		return sentMessage{}
	}
}

func waitForRunDone(t *testing.T, done <-chan error) {
	t.Helper()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run returned error: %v", err)
		}
	case <-time.After(testWait):
		t.Fatal("Run did not return after ctx cancel")
	}
}

// balaurCall is one decoded request to the fake Balaur messenger endpoint.
type balaurCall struct {
	auth string
	body string
}

func waitForBalaurCall(t *testing.T, calls <-chan balaurCall) balaurCall {
	t.Helper()
	select {
	case c := <-calls:
		return c
	case <-time.After(testWait):
		t.Fatal("timed out waiting for a Balaur call")
		return balaurCall{}
	}
}

func baseConfig(tgURL, balaurURL string, allowed ...int64) Config {
	return Config{
		BotToken:        testToken,
		MessengerToken:  "secret-messenger-token",
		BalaurURL:       balaurURL,
		TelegramBaseURL: tgURL,
		AllowedChatIDs:  allowed,
		RetryBase:       time.Millisecond,
	}
}

const testToken = "TESTBOTTOKEN"

func TestBridgeHappyPath(t *testing.T) {
	tg := newFakeTelegram([][]tgUpdate{
		{{UpdateID: 1, Message: &tgMessage{Chat: tgChat{ID: 42}, Text: "hi"}}},
	})
	tgSrv := httptest.NewServer(tg.handler())
	defer tgSrv.Close()

	calls := make(chan balaurCall, 4)
	balaurSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]string{"reply": "Hello from Balaur"})
		calls <- balaurCall{auth: r.Header.Get("Authorization"), body: string(b)}
	}))
	defer balaurSrv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := baseConfig(tgSrv.URL, balaurSrv.URL, 42)

	done := make(chan error, 1)
	go func() { done <- Run(ctx, cfg, testLogger()) }()

	waitForPoll(t, tg, "0")
	call := waitForBalaurCall(t, calls)
	if call.auth != "Bearer secret-messenger-token" {
		t.Errorf("Authorization = %q, want Bearer secret-messenger-token", call.auth)
	}
	if call.body != `{"message":"hi"}` {
		t.Errorf("turn body = %q, want {\"message\":\"hi\"}", call.body)
	}

	msg := waitForSend(t, tg)
	if msg.ChatID != 42 || msg.Text != "Hello from Balaur" {
		t.Errorf("sendMessage = %+v, want {ChatID:42 Text:\"Hello from Balaur\"}", msg)
	}

	waitForPoll(t, tg, "2")

	cancel()
	waitForRunDone(t, done)
}

func TestBridgeBusyRetry(t *testing.T) {
	tg := newFakeTelegram([][]tgUpdate{
		{{UpdateID: 9, Message: &tgMessage{Chat: tgChat{ID: 7}, Text: "hi again"}}},
	})
	tgSrv := httptest.NewServer(tg.handler())
	defer tgSrv.Close()

	var attempts int32
	balaurSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&attempts, 1)
		w.Header().Set("Content-Type", "application/json")
		if n == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_ = json.NewEncoder(w).Encode(map[string]string{"error": "busy"})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"reply": "ok now"})
	}))
	defer balaurSrv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := baseConfig(tgSrv.URL, balaurSrv.URL, 7)

	done := make(chan error, 1)
	go func() { done <- Run(ctx, cfg, testLogger()) }()

	msg := waitForSend(t, tg)
	if msg.Text != "ok now" {
		t.Errorf("delivered text = %q, want %q", msg.Text, "ok now")
	}
	if got := atomic.LoadInt32(&attempts); got != 2 {
		t.Errorf("Balaur POSTs = %d, want 2", got)
	}

	cancel()
	waitForRunDone(t, done)
}

func TestBridgeAllowlistRejects(t *testing.T) {
	tg := newFakeTelegram([][]tgUpdate{
		{{UpdateID: 5, Message: &tgMessage{Chat: tgChat{ID: 999}, Text: "hi"}}},
	})
	tgSrv := httptest.NewServer(tg.handler())
	defer tgSrv.Close()

	var balaurHits int32
	balaurSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&balaurHits, 1)
	}))
	defer balaurSrv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := baseConfig(tgSrv.URL, balaurSrv.URL, 1) // 999 not allowlisted

	done := make(chan error, 1)
	go func() { done <- Run(ctx, cfg, testLogger()) }()

	waitForPoll(t, tg, "0")
	waitForPoll(t, tg, "6") // offset advances past the rejected update without a Balaur call

	cancel()
	waitForRunDone(t, done)

	if got := atomic.LoadInt32(&balaurHits); got != 0 {
		t.Errorf("Balaur hit %d times, want 0 (sender not allowlisted)", got)
	}
}

func TestBridgeEmptyAllowlistFails(t *testing.T) {
	var tgHits, balaurHits int32
	tgSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&tgHits, 1)
	}))
	defer tgSrv.Close()
	balaurSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&balaurHits, 1)
	}))
	defer balaurSrv.Close()

	cfg := baseConfig(tgSrv.URL, balaurSrv.URL) // no allowed chat ids

	err := Run(context.Background(), cfg, testLogger())
	if err == nil {
		t.Fatal("Run with empty AllowedChatIDs = nil error, want non-nil")
	}
	if !strings.Contains(err.Error(), "allowlist") {
		t.Errorf("error = %q, want it to mention 'allowlist'", err.Error())
	}
	if got := atomic.LoadInt32(&tgHits); got != 0 {
		t.Errorf("Telegram fake hit %d times, want 0", got)
	}
	if got := atomic.LoadInt32(&balaurHits); got != 0 {
		t.Errorf("Balaur fake hit %d times, want 0", got)
	}
}

// hangingTelegram simulates a real long-poll: getUpdates blocks until the
// request context is cancelled (or a safety timeout), signalling started
// once so the test can synchronize on "the bridge is now polling".
type hangingTelegram struct {
	started chan struct{}
	once    sync.Once
}

func (h *hangingTelegram) handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.once.Do(func() { close(h.started) })
		select {
		case <-r.Context().Done():
		case <-time.After(testWait):
		}
	}
}

func TestBridgeGracefulShutdown(t *testing.T) {
	hang := &hangingTelegram{started: make(chan struct{})}
	tgSrv := httptest.NewServer(hang.handler())
	defer tgSrv.Close()

	balaurSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected call to Balaur endpoint during shutdown test")
	}))
	defer balaurSrv.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cfg := baseConfig(tgSrv.URL, balaurSrv.URL, 1)

	done := make(chan error, 1)
	go func() { done <- Run(ctx, cfg, testLogger()) }()

	select {
	case <-hang.started:
	case <-time.After(testWait):
		t.Fatal("bridge never started polling")
	}

	cancel()
	waitForRunDone(t, done)
}
