# Plan 239: Log, throttle, and audit the messenger gateway's credential events

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- internal/web/messenger.go internal/web/messenger_settings.go internal/web/messenger_test.go internal/web/messenger_settings_test.go internal/web/web.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: security
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

`POST /api/messenger/turn` is Balaur's only network surface gated by a bearer
credential, and by the handler's own documented security model the Bearer
token is the PRIMARY access control — the Host check is only a DNS-rebinding
defense, and production deliberately binds `0.0.0.0:8080` behind a NetBird
mesh ACL. Today a failed token attempt returns 401 with no log line, no rate
limit, and nothing in `audit_log`: an ACL-admitted peer (or anyone reachable
after ACL/firewall drift) can guess tokens at wire speed, invisibly.
Separately, setting/rotating/clearing that token via the settings UI writes
`owner_settings` with no audit entry — unlike every other capability-granting
mutation in the codebase (cloud consent, model download, runtime install, OS
tools all audit). After this plan lands: failed auth is logged (never the
token), repeated failures hit a short in-process lockout (brute-force
friction, honest about not being real rate limiting), and token set/clear
leaves an `audit_log` row recording only the state transition, never the
value.

## Current state

### Files

- `internal/web/messenger.go` — the `POST /api/messenger/turn` handler
  (`messengerTurn`); contains the silent, unthrottled 401 (lines 70–79).
- `internal/web/messenger_settings.go` — `saveMessengerToken`
  (`POST /ui/settings/messenger-token`); persists the token with no audit
  (lines 16–21).
- `internal/web/web.go` — route registration and the `handlers` struct
  (lines 252, 256–259); the struct is where per-instance throttle state
  must live (repo rule: no package-level mutable state).
- `internal/web/messenger_test.go` — endpoint tests; has the helpers
  `buildMessengerRouter`, `postMessenger`, `countMessages`, and the
  blocking-client pattern.
- `internal/web/messenger_settings_test.go` — settings-handler tests using
  `tests.ApiScenario`.
- `internal/store/audit.go` — `store.Audit` (the audit seam, read-only for
  this plan).

### The silent, unthrottled auth failure

`internal/web/messenger.go:70-79` (verified at `077318a`):

```go
	// 3. Token auth — constant-time comparison; the header value is never logged.
	authHeader := e.Request.Header.Get("Authorization")
	const prefix = "Bearer "
	var provided string
	if len(authHeader) > len(prefix) && authHeader[:len(prefix)] == prefix {
		provided = authHeader[len(prefix):]
	}
	if subtle.ConstantTimeCompare([]byte(provided), []byte(tok)) != 1 {
		return e.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
```

The comparison is constant-time (good), but the failure branch has no
`app.Logger()` call, no audit, and no rate limit. Nothing in the repo enables
PocketBase's built-in rate limiter, and the handler's own header comment
(`internal/web/messenger.go:10-19`) states that on a `0.0.0.0`-binding box
"the Bearer token … is the PRIMARY, effective access control, not the host
check."

Immediately after the token check the handler acquires the cross-surface turn
guard, `internal/web/messenger.go:83-87`:

```go
	end, ok := turn.TryBegin()
	if !ok {
		return e.JSON(http.StatusTooManyRequests, map[string]string{"error": "busy"})
	}
	defer end()
```

Note the existing 429 body is `"busy"` — the new throttle's 429 must use a
*different* body so tests (and bridge operators) can tell them apart.

### The unaudited token write

`internal/web/messenger_settings.go:16-21` (verified at `077318a`):

```go
func (h *handlers) saveMessengerToken(e *core.RequestEvent) error {
	token := strings.TrimSpace(e.Request.FormValue("messenger_token"))
	// Never log the token value — it is a secret.
	if err := store.SetOwnerSetting(h.app, "messenger_token", token); err != nil {
		return e.InternalServerError("saving messenger token", err)
	}
```

No `store.Audit` call anywhere in this file. Compare the exemplar for a
capability-granting owner mutation, `internal/web/models.go:178-181`:

```go
	if err := store.SetOwnerSetting(h.app, cloudAckKey(cfg.ProviderID), "1"); err != nil {
		return e.InternalServerError("saving consent", err)
	}
	store.Audit(h.app, "owner", "llm.cloud.consent", cfg.ProviderID, true, map[string]any{"provider": cfg.ProviderName})
```

### The audit seam

`internal/store/audit.go:14` (signature — last param is named `detail` and
lands in the `audit_log.detail` JSON field):

```go
func Audit(app core.App, actor, action, target string, allowed bool, detail map[string]any) {
```

### Where per-instance state lives

`internal/web/web.go:256-259` — the `handlers` struct; and `web.go:162` —
its single construction (one instance per serve, shared across all requests,
so throttle fields MUST be mutex-guarded):

```go
type handlers struct {
	app     core.App
	clients turn.ClientSource
}
```

```go
	h := &handlers{app: se.App, clients: turn.ClientSource{Engine: kronk.FromStore(se.App)}}
```

### Repo conventions that apply here

- **No global mutable state** — throttle state lives on `handlers` (or a
  small struct embedded in it), never at package level.
- **Structured logging only** via `h.app.Logger()` (`*slog.Logger`) with
  key/value pairs; no `fmt.Print*`. The token or Authorization header value
  must NEVER appear in a log line, an audit row, or an error message
  (constraint 4 in `messenger.go`'s header comment: "No secrets in
  output/logs").
- **Audit strictly AFTER the successful write** — the audit row must never
  record a mutation that did not persist. Match `models.go:178-181` above.
- **Errors**: `fmt.Errorf("doing x: %w", err)`, return early, no panics in
  library code (this plan adds no new error paths that need wrapping, but
  keep the rule in mind).
- **Tests**: std `testing` package, table-driven where it helps; web-handler
  tests use `newWebApp` (`internal/web/handlers_test.go:29`) and either the
  full-router pattern (`buildMessengerRouter`, `messenger_test.go:54-70`) or
  `tests.ApiScenario` with `AfterTestFunc` for post-request DB assertions
  (exemplar: `internal/web/models_test.go:63-68`). No `time.Sleep`-based
  synchronization — inject a clock seam instead.
- **KISS/YAGNI**: this throttle is documented brute-force *friction* at v1
  scale (one owner, one bridge), not real rate limiting. No per-IP maps, no
  token buckets, no config knobs.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all pass |
| Targeted tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run 'TestMessenger\|TestSaveMessengerToken\|TestAuthThrottle' -count=1` | ok, all pass |
| Vet | `go vet ./...` | exit 0 |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |

(Full `/tmp` is a small tmpfs on this host; the Go linker OOMs there —
always set `TMPDIR=$HOME/.cache/go-tmp` for test runs.)

The `\|` in the targeted-tests regex above is markdown-table escaping only.
Type the command with real `|` characters:

```sh
TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run 'TestMessenger|TestSaveMessengerToken|TestAuthThrottle' -count=1
```

A literal `\|` in Go's RE2 syntax matches zero tests while still printing
"ok" — a silent false pass, not the expected result.

## Suggested executor toolkit

- If the `go-standards` skill is available, invoke it before Step 1 — it
  covers this repo's slog, audit-after-save, and testing idioms.

## Scope

**In scope** (the only files you should modify):

- `internal/web/messenger.go`
- `internal/web/messenger_settings.go`
- `internal/web/messenger_test.go`
- `internal/web/messenger_settings_test.go`
- `internal/web/web.go` (only the `handlers` struct — add one field)

**Out of scope** (do NOT touch, even though they look related):

- A token "generate" button or any settings-UI change
  (`internal/feature/settingscards/*`) — deferred, separate concern.
- PocketBase's global rate limiter configuration — this plan ships a local,
  endpoint-specific throttle on purpose.
- `guardLocalUI`, `isAllowedHost`, or any redesign of the messenger auth
  model — the Host-check-vs-token primacy documented in
  `messenger.go:10-27` is a settled design (plan 231); do not revisit it.
- `internal/store/audit.go` — use it, don't change it.
- `internal/self/knowledge.md` — NOT needed for this change: the file does
  not describe the messenger gateway's auth mechanics (its only mention is a
  "Future gateways (messengers)" aside at line 72), and this plan hardens an
  existing capability rather than adding a user-visible one.
- `.tours/` — no tour anchors `internal/web/messenger*.go`; the three
  `web.go` anchors sit at lines 1, 64, and 138 (`00-orientation.tour`,
  `07-the-web-gateway.tour`), all BEFORE the `handlers` struct at line 256,
  so adding a struct field shifts no anchored line and falsifies no prose.

## Git workflow

- The executor runs in an isolated git worktree branched from `origin/main`.
- Branch: `advisor/239-messenger-auth-observability`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/
  `test`/`chore`); one commit per logical unit is fine, e.g.
  `fix(web): log, throttle, and audit messenger credential events`.
- Stage with explicit pathspecs only (the main checkout is shared by
  parallel agents — stage only your own files):
  `git add internal/web/messenger.go internal/web/messenger_settings.go internal/web/messenger_test.go internal/web/messenger_settings_test.go internal/web/web.go`
- NEVER push. The reviewer merges.

## Steps

### Step 1: Add the throttle type and failed-auth logging to `messenger.go`, and the throttle field to `handlers`

**1a.** In `internal/web/messenger.go`, add package-level constants and a
small mutex-guarded throttle type (constants are fine at package level; only
mutable state is banned). Place them above `messengerTurn`. Target shape:

```go
// Brute-force friction on the token check (v1 scale: one owner, one
// bridge). After messengerMaxFailures consecutive bad tokens the endpoint
// answers 429 until messengerCooldown passes; any successful auth resets
// the counter. This is deliberate friction, NOT real rate limiting — no
// per-IP tracking, no persistence across restarts.
const (
	messengerMaxFailures = 5
	messengerCooldown    = 30 * time.Second
)

// authThrottle holds the failure counter. It lives on the handlers struct
// (one instance per serve, shared across requests), so all access is
// mutex-guarded. The zero value is ready to use.
type authThrottle struct {
	mu       sync.Mutex
	failures int
	lastFail time.Time
	now      func() time.Time // test seam; nil means time.Now
}

func (t *authThrottle) clock() time.Time {
	if t.now != nil {
		return t.now()
	}
	return time.Now()
}

// allow reports whether an auth attempt may proceed. Cooldown expiry
// resets the counter so one stale failure cannot re-lock the endpoint.
func (t *authThrottle) allow() bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.failures < messengerMaxFailures {
		return true
	}
	if t.clock().Sub(t.lastFail) >= messengerCooldown {
		t.failures = 0
		return true
	}
	return false
}

func (t *authThrottle) fail() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failures++
	t.lastFail = t.clock()
}

func (t *authThrottle) success() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.failures = 0
}
```

Add `"strconv"` and `"sync"` to the import block (`"time"` is already
imported).

**1b.** In `internal/web/web.go`, add the field to the `handlers` struct at
lines 256–259 (the zero value is ready — the constructor at `web.go:162`
needs NO change):

```go
type handlers struct {
	app     core.App
	clients turn.ClientSource
	// messengerThrottle adds brute-force friction to the messenger token
	// check (messenger.go); per-instance, mutex-guarded — no package state.
	messengerThrottle authThrottle
}
```

**1c.** In `messengerTurn`, wire the throttle around the existing token
check. Insert the lockout check AFTER the consent gate (after the `tok == ""`
early return at line 66–68) and BEFORE reading the Authorization header:

```go
	// 3a. Brute-force friction: after messengerMaxFailures consecutive bad
	//     tokens, reject with 429 until messengerCooldown passes. The body
	//     differs from the turn guard's "busy" so callers can tell them apart.
	if !h.messengerThrottle.allow() {
		e.Response.Header().Set("Retry-After", strconv.Itoa(int(messengerCooldown/time.Second)))
		return e.JSON(http.StatusTooManyRequests, map[string]string{"error": "too many failed auth attempts"})
	}
```

Then change the failure branch (currently `messenger.go:77-79`) to record the
failure and log it — the log line carries the remote address ONLY, never the
token or header value:

```go
	if subtle.ConstantTimeCompare([]byte(provided), []byte(tok)) != 1 {
		h.messengerThrottle.fail()
		h.app.Logger().Warn("messenger: auth failed", "remote", e.Request.RemoteAddr)
		return e.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	}
	h.messengerThrottle.success()
```

**1d.** Extend the file-header security-model comment (constraint 4 area,
`messenger.go:26-27`) with one line noting the new behavior, e.g.:
`//  5. Failed auth is logged (remote addr only) and throttled: after 5` /
`//     consecutive bad tokens the endpoint answers 429 for a 30s cooldown.`

**Verify**:
`CGO_ENABLED=0 go build ./... && go vet ./...` → exit 0, no output.
`gofmt -l .` → empty output.

### Step 2: Audit the token write in `messenger_settings.go`

In `saveMessengerToken` (`internal/web/messenger_settings.go:16-21`), add the
audit call strictly AFTER the successful `store.SetOwnerSetting` (never
before — the audit log must not record a mutation that did not persist), and
record only the state transition, NEVER the token value:

```go
	if err := store.SetOwnerSetting(h.app, "messenger_token", token); err != nil {
		return e.InternalServerError("saving messenger token", err)
	}
	// Audit the state transition only — never the token value.
	state := "set"
	if token == "" {
		state = "cleared"
	}
	store.Audit(h.app, "owner", "messenger.token", "owner_settings/messenger_token", true, map[string]any{"state": state})
```

**Verify**:
`CGO_ENABLED=0 go build ./... && go vet ./...` → exit 0.
`grep -c "store.Audit" internal/web/messenger_settings.go` → `1`.

### Step 3: Throttle tests in `internal/web/messenger_test.go`

Append three tests, reusing the existing helpers `buildMessengerRouter`
(`messenger_test.go:54`), `postMessenger` (`:73`), `seedScriptedModel`
(`internal/web/fakeclient_test.go:17`), and `llmtest.Text`. No `time.Sleep`
anywhere — cooldown expiry is covered by the unit test's fake clock.

1. **`TestMessengerAuthThrottleLockout`** (integration, full mux, default
   cooldown — no clock needed because nothing waits for expiry):
   - `app, mux := buildMessengerRouter(t)`; `defer app.Cleanup()`;
     `store.SetOwnerSetting(app, "messenger_token", "right-token")`.
   - 5 × `postMessenger(mux, "example.com", "Bearer wrong", `{"message":"hi"}`)`
     → each returns 401.
   - 6th bad post → 429, body contains `too many failed auth attempts`
     (NOT `busy` — that is the turn guard's body).
   - A post with the CORRECT token during lockout → also 429 (proves the
     lockout gates the primary access control; no model is seeded, so if
     this ever ran a turn it would surface as a non-429 status).
2. **`TestMessengerAuthThrottleSuccessResets`** (integration):
   - Seed token + `seedScriptedModel(t, app, llmtest.Text("ok"))`.
   - 4 bad posts → 401 each (counter at 4, still below the limit of 5).
   - 1 correct post → 200 (and resets the counter).
   - 5 more bad posts → ALL 401, none 429 (without the reset, cumulative
     failures would be ≥5 and the 2nd of these would already be 429).
3. **`TestAuthThrottleCooldown`** (unit test on the `authThrottle` type,
   fake clock via the `now` seam):

   ```go
   now := time.Unix(0, 0)
   th := &authThrottle{now: func() time.Time { return now }}
   ```

   - Fresh throttle: `allow()` → true.
   - After `messengerMaxFailures` × `fail()`: `allow()` → false.
   - Advance `now = now.Add(messengerCooldown)`: `allow()` → true (cooldown
     expiry resets the counter), and a single subsequent `fail()` leaves
     `allow()` true (counter really was reset, not left at the limit).
   - `success()` after some `fail()`s: `allow()` stays true.

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run 'TestMessenger|TestAuthThrottle' -count=1` → ok, all pass (including the five pre-existing
`TestMessenger*` tests — `TestMessengerInFlight` must still pass: its
requests all use the correct token, so the throttle never trips there).

### Step 4: Audit tests in `internal/web/messenger_settings_test.go`

Extend the two existing scenarios with `AfterTestFunc` DB assertions —
pattern exemplar: `internal/web/models_test.go:63-68`
(`s.AfterTestFunc = func(tb testing.TB, a *tests.TestApp, _ *http.Response) {…}`;
the field's signature is `func(t testing.TB, app *tests.TestApp, res *http.Response)`).
Audit-row query pattern exemplar: `internal/ext/ext_test.go:107-113`
(`app.FindRecordsByFilter("audit_log", "action = '…'", "", 0, 0)`).

1. In **`TestSaveMessengerTokenSetsAndPatches`** (posts `secret42`), add an
   `AfterTestFunc` field to the `tests.ApiScenario` literal that asserts:
   - exactly ONE `audit_log` row with `action = 'messenger.token'`;
   - `GetString("actor") == "owner"`,
     `GetString("target") == "owner_settings/messenger_token"`,
     `GetBool("allowed") == true`;
   - marshal the whole record (`raw, err := json.Marshal(rec)`) and assert
     `strings.Contains(string(raw), `"state":"set"`)` AND
     `!strings.Contains(string(raw), "secret42")` — the raw record JSON must
     not contain the token string anywhere.
2. In **`TestSaveMessengerTokenClearsOnEmpty`** (posts an empty value), add
   an `AfterTestFunc` asserting exactly one `messenger.token` row whose
   marshaled JSON contains `"state":"cleared"`.

Add the needed imports (`encoding/json`, `net/http`, `testing` is present).
Leave `TestSaveMessengerTokenWithExistingSeeded` unchanged.

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run 'TestSaveMessengerToken' -count=1` → ok, all 3 pass.

### Step 5: Full gate

Run, in order:

1. `gofmt -l .` → empty output.
2. `go vet ./...` → exit 0.
3. `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output,
   exit 0 (watch for U1000 — every new method on `authThrottle` is called).
4. `CGO_ENABLED=0 go build ./...` → exit 0.
5. `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0, all pass.
6. `git diff --check` → no output.
7. (Belt-and-braces, since `web.go` changed) `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok.

## Test plan

- New tests, all in-scope files, no real model (fake `llm.Client` via
  `llmtest`/`seedScriptedModel` only), no `time.Sleep`:
  - `internal/web/messenger_test.go`:
    `TestMessengerAuthThrottleLockout` (5 bad → 401s; 6th → 429 with the
    `too many failed auth attempts` body; correct token during lockout →
    429), `TestMessengerAuthThrottleSuccessResets` (4 bad + 1 good + 5 bad →
    the last 5 are 401, not 429), `TestAuthThrottleCooldown` (fake-clock
    unit test: lockout, cooldown expiry resets, success resets).
  - `internal/web/messenger_settings_test.go`: `AfterTestFunc` additions to
    the two existing scenarios asserting the single `messenger.token`
    audit row (`state: set` / `state: cleared`) and that the token string
    never appears in the marshaled record.
- Structural patterns: full-router tests model `messenger_test.go:100-128`;
  `AfterTestFunc` assertions model `internal/web/models_test.go:63-68`;
  audit-row queries model `internal/ext/ext_test.go:107-113`.
- Regression guard: the five pre-existing `TestMessenger*` tests and the
  three `TestSaveMessengerToken*` tests must still pass unmodified in
  behavior.
- Verification: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run 'TestMessenger|TestSaveMessengerToken|TestAuthThrottle' -count=1` → ok, and
  the full gate `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `gofmt -l .` → empty; `go vet ./...` → exit 0;
      `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output;
      `CGO_ENABLED=0 go build ./...` → exit 0.
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0.
- [ ] `grep -c '"messenger: auth failed"' internal/web/messenger.go` → `1`
      (and the line contains no reference to `tok`, `provided`, or
      `authHeader`).
- [ ] `grep -c 'store.Audit' internal/web/messenger_settings.go` → `1`, with
      action `"messenger.token"` and detail carrying only a `state` key.
- [ ] `grep -c 'messengerThrottle authThrottle' internal/web/web.go` → `1`
      (the field declaration; a bare `messengerThrottle` grep would count 2
      because Step 1b's doc comment also names the field);
      `grep -rn 'var .*authThrottle' internal/web/messenger.go` →
      no package-level variable (state lives on `handlers` only).
- [ ] The three new tests plus the two `AfterTestFunc` assertions exist and
      pass: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run 'TestMessenger|TestSaveMessengerToken|TestAuthThrottle' -count=1 -v`
      shows `TestMessengerAuthThrottleLockout`,
      `TestMessengerAuthThrottleSuccessResets`, `TestAuthThrottleCooldown`
      all PASS.
- [ ] No token value appears in any log, audit, or error string:
      `grep -n 'Logger()' internal/web/messenger.go internal/web/messenger_settings.go`
      shows no line passing `tok`, `token`, `provided`, or `authHeader` as a
      log value.
- [ ] `git status --porcelain` lists ONLY the five in-scope files (plus
      `plans/README.md` if you update the index row).
- [ ] `plans/README.md` status row updated (skip if the reviewer said they
      maintain the index, or if the index does not track plan 239).

## STOP conditions

Stop and report back (do not improvise) if:

- The excerpts in "Current state" do not match the live files (drift since
  `077318a`) — in particular if `messengerTurn`'s auth branch or
  `saveMessengerToken` already contains a `Logger()` or `store.Audit` call.
- `grep -rn '"messenger.token"' internal/` (excluding this plan) already
  matches — an audit row for this path exists and this plan would
  double-audit; re-check before writing any code.
- The `handlers` struct at `internal/web/web.go:256` is no longer the
  per-serve instance shared across requests (e.g. it became per-request, or
  the messenger route no longer goes through `h.messengerTurn`) — the
  throttle design assumes one long-lived instance.
- `TestMessengerInFlight` starts failing after your change — that means the
  throttle is tripping on correct-token requests and the design needs
  review, not a workaround.
- A step's verification fails twice after a reasonable fix attempt.
- The fix appears to require touching an out-of-scope file (e.g. the
  settings card in `internal/feature/settingscards`, or
  `internal/store/audit.go`).

## Maintenance notes

- **The throttle is deliberately naive**: one global counter per process, no
  per-IP tracking, resets on restart. At v1 scale (one owner, one bridge,
  loopback/mesh-only exposure) that is the right cost. If the messenger
  gateway ever serves multiple bridges or moves past the NetBird ACL,
  revisit with PocketBase's rate limiter or a per-remote map — do not grow
  this one.
- **Reviewer scrutiny**: (1) the audit call sits strictly AFTER
  `SetOwnerSetting` succeeds; (2) no code path logs or audits the token or
  Authorization header; (3) the throttle's 429 body differs from the turn
  guard's `busy` body; (4) `success()` is called only after the
  constant-time compare passes.
- **Interaction**: the lockout answers 429 *before* the request body is
  parsed or a turn starts, so a locked-out endpoint does zero LLM work.
  A legitimate bridge with a stale token will lock itself out for 30s
  windows — the log line (`messenger: auth failed`) is how the owner
  diagnoses that.
- **Pre-existing drift, out of scope**: `internal/self/knowledge.md:72`
  still says "Future gateways (messengers) follow the same rule" even
  though the messenger gateway shipped (plan 231). Noted for a future docs
  pass; do not fix it in this plan.
- **Deferred**: a settings-UI "generate token" button (would remove the
  human-chosen-weak-token risk); auditing failed *auth attempts* (log-only
  today — an audit row per bad request would let a flood bloat
  `audit_log`).
