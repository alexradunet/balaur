# Plan 247: Pin the untested busy and error branches of the three turn gateways

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- internal/web/gateway_busy_test.go internal/cli/cli_test.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition. Also compare the excerpts from the
> behavior-under-test files (`internal/web/chat.go`, `internal/web/messenger.go`,
> `internal/cli/chat.go`, `internal/turn/guard.go`) — they are read-only for
> this plan but the tests assert their exact strings and ordering.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: plans/239-messenger-auth-observability.md (merge-friction only:
  239 extends `internal/web/messenger_test.go` and adds an auth throttle to
  `internal/web/messenger.go`; land 239 first. This plan deliberately creates a
  NEW test file instead of editing `messenger_test.go`, and all its messenger
  requests use a VALID token, so 239's failed-auth throttle never triggers.)
- **Category**: tests
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

Balaur runs exactly one turn at a time on the master conversation across all
three gateways (web, CLI, messenger) via a process-wide guard,
`turn.TryBegin()`. Every gateway has a "busy" rejection branch and the
messenger gateway has error branches — and today several of them have ZERO
test coverage: the web busy toast, the CLI busy error, the messenger 400
bad-body path, and the IPv6 loopback Host form. The most dangerous gap is the
messenger 400 path: the handler acquires the guard BEFORE parsing the body,
and release rests solely on a single `defer end()`. If a refactor ever
replaced that defer with per-path `end()` calls and missed the 400 branch, one
malformed bridge request would wedge ALL surfaces (web, CLI, messenger) busy
forever — and the current suite would stay green. This plan adds four small
tests that pin those branches. It is a tests-only change: zero production diff.

## Current state

### The guard primitive (read-only for this plan)

`internal/turn/guard.go:24-30` — a process-wide `sync.Mutex` with `TryLock`;
`end` is idempotent via `sync.Once`:

```go
func TryBegin() (end func(), ok bool) {
	if !turnMu.TryLock() {
		return nil, false
	}
	var once sync.Once
	return func() { once.Do(turnMu.Unlock) }, true
}
```

`internal/turn/guard_test.go` already covers the primitive itself
(first-succeeds, second-rejected, succeeds-after-end, end-idempotent, race).
Do NOT touch it — this plan tests the *gateway branches*, not the mutex.

### Web gateway busy branch — untested

`internal/web/chat.go:40-48` — busy is rejected before any medium setup: a
minimal SSE stream delivers only a toast; no user bubble, no `#chat` mutation:

```go
	end, ok := turn.TryBegin()
	if !ok {
		// Open a minimal SSE connection just to deliver the toast; no #chat
		// mutation, no user bubble.
		sse := datastar.NewSSE(e.Response, e.Request)
		emitToast(sse, "warn", "One message is still being answered — try again in a moment.")
		return nil
	}
	defer end()
```

The toast string `One message is still being answered` appears in no
`_test.go` file (verified with grep at the planned-at commit). The toast is
rendered by `internal/web/toast.go:15-20`:

```go
func emitToast(sse *datastar.ServerSentEventGenerator, tone, msg string) {
	_ = sse.PatchElements(
		renderNodeHTML(ui.Toast(ui.ToastProps{Tone: tone}, g.Text(msg))),
		datastar.WithSelectorID("toast-region"), datastar.WithModeAppend(),
	)
}
```

so the SSE body contains both the toast text and the `toast-region` selector.
The happy-path user bubble carries class `cmsg cmsg-user` (see
`internal/web/handlers_test.go:244`, `ExpectedContent: []string{..., "cmsg cmsg-user", ...}`)
— its ABSENCE is the "no user bubble" assertion.

### CLI gateway busy branch — untested

`internal/cli/chat.go:60-71` — note the client resolves BEFORE the guard, so a
busy test must still inject a fake client or it fails earlier with a
model-resolution error:

```go
	cmd.RunE = run(app, "chat", func(cmd *cobra.Command, args []string) (any, error) {
		client, err := chatClients(app)
		if err != nil {
			return nil, err
		}
		// Cross-surface in-flight guard: one turn at a time on the master
		// conversation (web + CLI + messenger share the same guard).
		end, ok := turn.TryBegin()
		if !ok {
			return nil, errors.New("busy: a turn is already in progress")
		}
		defer end()
```

The error string `busy: a turn is already in progress` appears in no
`_test.go` file. The failure contract is a v1 error envelope on stderr —
`internal/cli/cli.go:119-123`:

```go
func failJSON(cmd *cobra.Command, err error) error {
	exitCode.Store(1)
	_ = emit(cmd.ErrOrStderr(), "error", map[string]string{"error": err.Error()})
	return err
}
```

### Messenger gateway: guard acquired BEFORE body parse; 400 branch untested

`internal/web/messenger.go:81-95` — the load-bearing ordering. The guard
(step 4) is taken before body parse (step 5); release is ONLY the `defer end()`
at line 87. No test proves a 400 frees the guard:

```go
	// 4. Cross-surface in-flight guard — one turn at a time on the master
	//    conversation (shared with the web and CLI gateways via turn.TryBegin).
	end, ok := turn.TryBegin()
	if !ok {
		return e.JSON(http.StatusTooManyRequests, map[string]string{"error": "busy"})
	}
	defer end()

	// 5. Parse body.
	var body struct {
		Message string `json:"message"`
	}
	if err := json.NewDecoder(e.Request.Body).Decode(&body); err != nil || body.Message == "" {
		return e.BadRequestError("message is required", nil)
	}
```

The only in-flight test today, `TestMessengerInFlight`
(`internal/web/messenger_test.go:253`), covers messenger-vs-messenger 429 on
the HAPPY path — it never exercises the 400 branch or proves release after it.

### Messenger host guard: IPv6 loopback Host form unexercised

`internal/web/messenger.go:55-61` splits the port off `Host` and delegates:

```go
	host := e.Request.Host
	if hh, _, err := net.SplitHostPort(host); err == nil {
		host = hh
	}
	if !isAllowedHost(host) {
		return e.ForbiddenError("host not allowed", nil)
	}
```

`internal/web/web.go:105-111` — the loopback pass:

```go
func isAllowedHost(host string) bool {
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil && ip.IsLoopback() {
		return true
	}
```

For `Host: [::1]:8090`, `net.SplitHostPort` yields `::1`, which
`net.ParseIP(...).IsLoopback()` accepts. `TestMessengerLoopbackGuard`
(`internal/web/messenger_test.go:224-245`) covers `127.0.0.1` and `evil.test`
but NOT the bracketed IPv6 form — the only path that exercises the
`SplitHostPort` branch with a loopback IPv6.

### Test helpers to reuse (all in package `web` or `cli` — same package as the new tests)

- `newWebApp(t)` — `internal/web/handlers_test.go:29-52`: temp-dir
  `tests.TestApp` with routes registered and
  `t.Setenv("BALAUR_ALLOWED_HOSTS", "example.com")` (httptest requests default
  to Host `example.com`).
- `buildMessengerRouter(t)` — `internal/web/messenger_test.go:54-70`: builds
  the full PocketBase mux by triggering `OnServe`; returns `(app, mux)`.
- `postMessenger(mux, host, auth, body)` — `internal/web/messenger_test.go:73-84`:
  fires `POST /api/messenger/turn` with a controlled `req.Host`; `auth=""`
  omits the header.
- `countMessages(app, convID)` — `internal/web/messenger_test.go:88-97`:
  counts persisted `messages` rows for a conversation.
- `seedScriptedModel(tb, app, replies...)` — `internal/web/fakeclient_test.go:17-21`:
  injects the shared scripted fake `llm.Client` via `turn.SetTestClient` and
  activates a local model record. Tests never hit a real model.
- `conversation.Master(app)` — returns the master conversation record (see its
  use at `internal/web/messenger_test.go:104-108`).
- CLI: `executeEnvelope(t, cmd, args...)` — `internal/cli/cli_test.go:24-59`:
  runs a cobra command, returns the v1 envelope; on failure it parses the
  error envelope from stderr and returns it with the exec error.
- CLI: `withScriptedClient(t, c)` — `internal/cli/cli_test.go:249-254`:

```go
func withScriptedClient(t *testing.T, c llm.Client) {
	t.Helper()
	prev := chatClients
	chatClients = func(core.App) (llm.Client, error) { return c, nil }
	t.Cleanup(func() { chatClients = prev })
}
```

- CLI: `storetest.NewApp(t)` — `internal/storetest/storetest.go:18-26`:
  temp-dir migrated test app with cleanup.
- `llmtest.New(llmtest.Text("..."))` — the scripted fake client
  (`internal/llmtest`); one `llmtest.Text` reply serves one plain-text turn
  (exemplar: `TestMessengerHappyPath`, `internal/web/messenger_test.go:180`).

### Conventions that apply

- "Tests use the standard `testing` package, table-driven where it helps
  readability, run with `go test ./...`. No assertion frameworks." (AGENTS.md)
- "Fake the `llm.Client` interface — tests never hit a real model." (AGENTS.md)
- No `time.Sleep`-based synchronization. None is needed here: every request in
  this plan is synchronous (the guard is held by the test goroutine itself, not
  by a background turn), so no goroutines or channels are required — unlike
  `TestMessengerInFlight`, which needed a blocking client.
- Do NOT call `t.Parallel()` in any new test: `turn.TryBegin` is a
  process-wide mutex, and a parallel sibling test in the same package would
  race on it. (Separate packages run as separate processes under
  `go test ./...`, so `web` and `cli` tests cannot interfere with each other.)

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Full test gate (the merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0 |
| Targeted web tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run 'TestChatBusyToast\|TestMessengerBadBodyReleasesGuard\|TestMessengerIPv6LoopbackHost' -count=1` | ok, all pass |
| Targeted CLI test | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/cli/ -run TestChatBusyGuard -count=1` | ok |
| Vet | `go vet ./...` | exit 0 |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |

(The host `/tmp` is a small tmpfs; the Go linker OOMs there — always set
`TMPDIR=$HOME/.cache/go-tmp` as shown. `-count=1` bypasses the test cache.)

## Suggested executor toolkit

- If the `go-standards` skill is available, invoke it before writing the tests
  — it carries this repo's testing idioms (table-driven, fake `llm.Client`,
  no-sleep synchronization).

## Scope

**In scope** (the only files you should modify):

- `internal/web/gateway_busy_test.go` (create — new file, so plan 239's
  additions to `messenger_test.go` never conflict)
- `internal/cli/cli_test.go` (append one test + one import)

**Out of scope** (do NOT touch, even though they look related):

- `internal/web/chat.go`, `internal/web/messenger.go`, `internal/cli/chat.go`,
  `internal/turn/guard.go` — this is a tests-only plan; zero production diff.
- `internal/turn/guard_test.go` — already covers the primitive (including a
  `-race` exercise); duplicating it adds nothing.
- `internal/web/messenger_test.go` and `internal/web/messenger_settings_test.go`
  — plan 239 extends them; keep this plan's diff disjoint.
- `internal/self/knowledge.md` — no capability or architecture changes; a
  tests-only change does not alter the binary's self-description.
- `.tours/` — no tour anchors any in-scope file (`.tours/10-the-cli-api.tour`
  anchors `internal/cli/chat.go`, which this plan does not modify; new tests
  are APPENDED to `cli_test.go`, so no anchored line shifts).

## Git workflow

- The executor runs in an isolated git worktree branched from `origin/main`;
  branch name `advisor/247-gateway-busy-error-path-tests`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/`test`/`chore`);
  use `test:` here, e.g. `test(web,cli): pin gateway busy + messenger error branches`.
- Commit per logical unit with explicit pathspecs (the main checkout is shared
  by parallel agents — stage ONLY your own files):
  `git add internal/web/gateway_busy_test.go internal/cli/cli_test.go`.
- NEVER push; the reviewer merges.

## Steps

### Step 1: Create `internal/web/gateway_busy_test.go` with the three web tests

New file, package `web`. It reuses helpers already defined in the package
(`buildMessengerRouter`, `postMessenger`, `countMessages`, `seedScriptedModel`)
— do not redefine them. Target shape (adjust only if a compile error demands
it, e.g. an unused import):

```go
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
// message. Pins the busy branch at internal/web/chat.go:40-47.
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
```

Notes for this step:

- `httptest.NewRequest` defaults Host to `example.com`, which `newWebApp`
  (called inside `buildMessengerRouter`) allowlists via
  `BALAUR_ALLOWED_HOSTS=example.com` — so the `/ui/chat` POST passes the
  DNS-rebinding guard. A POST without `Origin`/`Sec-Fetch-Site` headers passes
  the CSRF rules (see the "no CSRF headers allowed (CLI/curl/harness)" case in
  `TestOriginGuard`, `internal/web/handlers_test.go:477-481`).
- `httptest.ResponseRecorder` implements `http.Flusher`, which the datastar
  SSE writer needs — the existing SSE-streaming chat tests already rely on
  this via `tests.ApiScenario`.

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run 'TestChatBusyToast|TestMessengerBadBodyReleasesGuard|TestMessengerIPv6LoopbackHost' -count=1`
→ `ok`, 3 top-level tests pass (4 including subtests).

### Step 2: Append `TestChatBusyGuard` to `internal/cli/cli_test.go`

Add `"github.com/alexradunet/balaur/internal/turn"` to the import block, then
append at the end of the file:

```go
// TestChatBusyGuard: while a turn is in flight (the test holds the
// cross-surface guard), `balaur chat` must fail with the busy error envelope
// and persist nothing. The fake client is still required because chatCmd
// resolves the model BEFORE checking the guard (internal/cli/chat.go:61-71).
func TestChatBusyGuard(t *testing.T) {
	app := storetest.NewApp(t)
	withScriptedClient(t, llmtest.New(llmtest.Text("must never be reached")))

	end, ok := turn.TryBegin()
	if !ok {
		t.Fatal("turn guard unexpectedly already held")
	}
	defer end()

	env, err := executeEnvelope(t, chatCmd(app), "hello")
	if err == nil {
		t.Fatal("chat must fail while a turn is in flight")
	}
	data, _ := env["data"].(map[string]any)
	if data == nil {
		t.Fatalf("error envelope missing data object: %v", env)
	}
	if got, _ := data["error"].(string); got != "busy: a turn is already in progress" {
		t.Errorf("error = %q, want %q", got, "busy: a turn is already in progress")
	}

	// The rejection happens before turn.Run — nothing may be persisted.
	recs, findErr := app.FindAllRecords("messages")
	if findErr != nil {
		t.Fatalf("FindAllRecords(messages): %v", findErr)
	}
	if len(recs) != 0 {
		t.Errorf("busy rejection must persist no messages, found %d", len(recs))
	}
}
```

`executeEnvelope` (`internal/cli/cli_test.go:24-59`) already validates the
error envelope's `v:1` and `kind:"error"` shape on the failure path, so the
test only asserts the payload. If `app.FindAllRecords` does not exist under
the vendored PocketBase version (it does at the planned-at commit), fall back
to `app.FindRecordsByFilter("messages", "role != ''", "", 0, 0)` — mirror
`countMessages` in `internal/web/messenger_test.go:88-97`.

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/cli/ -run TestChatBusyGuard -count=1`
→ `ok`.

### Step 3: Full gate

Run, in order:

1. `gofmt -l .` → empty output.
2. `go vet ./...` → exit 0.
3. `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output, exit 0.
4. `CGO_ENABLED=0 go build ./...` → exit 0.
5. `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0. In
   particular the pre-existing `TestMessengerInFlight` and `TestTryBegin*`
   tests must still pass (the new tests always release the guard via
   `defer end()`, so no cross-test wedging is possible even on failure).

**Verify**: all five commands succeed with the expected outputs above.

## Test plan

New tests (all in this plan — it IS the test plan):

- `internal/web/gateway_busy_test.go` (create):
  - `TestChatBusyToast` — web busy branch: 200 SSE containing the toast text
    targeting `#toast-region`; no `cmsg cmsg-user` bubble; message count on
    the master conversation unchanged.
  - `TestMessengerBadBodyReleasesGuard` — table-driven over `{` (malformed
    JSON) and `{}` (empty message): each → 400, then an immediate valid
    request → 200 (explicitly NOT 429), proving the `defer end()` released
    the process-wide guard after the 400.
  - `TestMessengerIPv6LoopbackHost` — `Host: [::1]:8090` with a valid token →
    not 403 (the `SplitHostPort` + loopback branch of `isAllowedHost`).
- `internal/cli/cli_test.go` (extend):
  - `TestChatBusyGuard` — CLI busy branch: error envelope with
    `data.error == "busy: a turn is already in progress"`; zero `messages`
    rows persisted.

Structural patterns to model after: `TestMessengerHappyPath` and
`TestMessengerLoopbackGuard` (`internal/web/messenger_test.go:175-245`) for
the mux-driven web tests; `TestVerifyFlagsUnbackedClaim`
(`internal/cli/cli_test.go:297-325`) for the CLI envelope test.

Verification:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ ./internal/cli/ -count=1`
→ ok for both packages, including the 4 new top-level tests.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run 'TestChatBusyToast|TestMessengerBadBodyReleasesGuard|TestMessengerIPv6LoopbackHost' -count=1` → ok.
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./internal/cli/ -run TestChatBusyGuard -count=1` → ok.
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0.
- [ ] `gofmt -l .` → empty; `go vet ./...` → exit 0;
      `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output;
      `CGO_ENABLED=0 go build ./...` → exit 0.
- [ ] Zero production diff: `git diff --name-only $(git merge-base origin/main HEAD)..HEAD`
      lists ONLY `internal/web/gateway_busy_test.go`, `internal/cli/cli_test.go`,
      and (if you maintain the index) `plans/README.md` — every code path is a
      `_test.go` file.
- [ ] `grep -c "One message is still being answered" internal/web/gateway_busy_test.go` → ≥ 1, and
      `grep -c "busy: a turn is already in progress" internal/cli/cli_test.go` → ≥ 1
      (the previously untested strings are now pinned).
- [ ] No files outside the in-scope list are modified (`git status --porcelain`
      in the worktree shows nothing unexpected).
- [ ] `plans/README.md` status row updated (unless the dispatching reviewer
      said they maintain the index).

## STOP conditions

Stop and report back (do not improvise) if:

- The excerpts in "Current state" no longer match the live code — in
  particular: the toast text in `internal/web/chat.go`, the busy error string
  in `internal/cli/chat.go`, or the guard-before-body-parse ordering in
  `internal/web/messenger.go` (plan 239 adds an auth throttle to that file and
  will shift line numbers — shifted lines alone are fine; changed
  strings/ordering are a STOP).
- `TestChatBusyToast` cannot observe the toast SSE through the mux (e.g. the
  datastar SSE writer errors against `httptest.NewRecorder`). Do not add a
  production seam to work around it — that is out of scope; report instead.
- The follow-up request in `TestMessengerBadBodyReleasesGuard` genuinely
  returns 429 — that means the 400 branch really does leak the guard (the
  exact production bug these tests exist to catch). Fixing
  `internal/web/messenger.go` is out of scope for this tests-only plan:
  report the finding.
- A messenger request with a VALID token gets throttled/rejected by plan 239's
  auth-throttle changes (should be impossible — the throttle counts failed
  auths only), or `TestMessengerInFlight` starts failing after your change.
- A step's verification fails twice after a reasonable fix attempt.
- The change appears to require touching any out-of-scope file.

## Maintenance notes

- These tests pin two owner-facing strings: the web toast
  `"One message is still being answered"` and the CLI error
  `"busy: a turn is already in progress"`. Rewording either in production
  requires updating the corresponding test in the same commit — that is the
  point: the copy is part of the busy contract.
- `TestMessengerBadBodyReleasesGuard` is the regression tripwire for the
  guard-release invariant. A reviewer of any future `internal/web/messenger.go`
  refactor (e.g. replacing `defer end()` with per-branch releases, or moving
  body parsing before the guard) should confirm this test still encodes the
  intended ordering.
- If the guard is ever keyed by conversation id instead of being process-wide
  (the roadmap note in `internal/turn/guard.go:22-23`), all four tests need
  re-derivation: the test would have to hold the MASTER conversation's key.
- Deferred (deliberately): a web-vs-messenger cross-surface concurrency test
  with a blocking client (the `messengerBlockingClient` pattern). The
  same-goroutine hold used here proves the same guard is consulted with far
  less machinery; add the concurrent variant only if a real cross-surface race
  regression ever appears.
