# Plan 011: Close the test gaps on the security and consent boundaries (heads, VM, SSE) with one shared LLM fake

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on.
> If anything in "STOP conditions" occurs, stop and report. When done,
> update this plan's row in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat c4fce47..HEAD -- internal/heads/ internal/ext/ internal/llm/ internal/turn/turn_test.go internal/cli/cli_test.go internal/tasks/nudge_test.go internal/recap/generate_test.go`
> On drift, re-verify the excerpts below.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW (tests + one test-helper package; one production-visible addition: none)
- **Depends on**: none (coordinate with plan 006 — its `-race` run benefits from these tests; no file overlap)
- **Category**: tests
- **Planned at**: commit `c4fce47`, 2026-06-12
- **Issue**: https://github.com/alexradunet/balaur/issues/26

## Why this matters

AGENTS.md's central security claim is "tests must prove out-of-scope access
fails." The existing proofs are good but partial, and two other
trust-critical layers are untested:

1. **Heads**: `internal/heads/heads_test.go` covers ungranted access, write
   without write-grant, merge revocation, and token resolution — but NOT
   head expiry, NOT grant-level expiry, NOT cross-head isolation, NOT
   `Revoke` (only `Merge`). The expiry checks are real code paths
   (`scoped.go:40-41` head `expires`; `scoped.go:53-54` per-grant
   `expires`) with zero coverage.
2. **The goja VM cap**: `internal/ext/vm.go:144-156` interrupts a handler on
   context cancellation or the 30s `invokeTimeout`. No test exercises
   either interrupt; a regression here lets a rogue extension hold the turn
   pipeline hostage. (`ext_test.go` covers consent/pinning/discovery well —
   the RUNTIME cap is the gap.)
3. **The OpenAI SSE parser** (`internal/llm/openai.go:112+`): the only path
   every remote-model tool call rides through. `kronk_test.go` tests env
   and deadline logic only; no test drives the stream parser with real
   wire bytes (multi-fragment tool calls, mid-stream disconnect).
4. **Four divergent fake LLM clients** exist — `turn_test.go:21-52`
   (`fakeClient`, scriptable, mutex-guarded), `cli_test.go:227+`
   (`scriptedClient`), `tasks/nudge_test.go:17-34` (minimal `fakeClient`),
   `recap/generate_test.go:20-38` (`echoClient`). Any `llm.Client`
   interface change means four edits, and the richer fakes' capabilities
   aren't available where the simpler ones live. One shared
   `internal/llmtest` package ends that.

## Current state

- `internal/llm/llm.go` — the interface (verify exact methods before
  porting; at c4fce47: `ChatStream(ctx, msgs, tools) (<-chan Chunk, error)`
  and `Embed(ctx, texts) ([][]float32, error)`; `Chunk` at llm.go:32,
  `ToolCall` at :18, `Message` at :10).
- The richest fake, to be promoted (verbatim from
  `internal/turn/turn_test.go:21-52`):

```go
type fakeClient struct {
	mu      sync.Mutex
	replies []fakeReply
	calls   int
}

type fakeReply struct {
	text  string
	calls []llm.ToolCall
}

func (f *fakeClient) ChatStream(ctx context.Context, msgs []llm.Message, tools []llm.ToolSpec) (<-chan llm.Chunk, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls++
	ch := make(chan llm.Chunk, 2)
	if len(f.replies) == 0 {
		ch <- llm.Chunk{Done: true}
		close(ch)
		return ch, nil
	}
	r := f.replies[0]
	f.replies = f.replies[1:]
	if r.text != "" {
		ch <- llm.Chunk{Content: r.text}
	}
	ch <- llm.Chunk{Done: true, ToolCalls: r.calls}
	close(ch)
	return ch, nil
}

func (f *fakeClient) Embed(...) ... { return nil, nil }
```

- Heads API (`internal/heads/heads.go`): `Spawn(app, name, purpose string,
  ttl time.Duration, grants []Grant, opts ...SpawnOption) (*core.Record, string, error)`
  (:44), `Merge(app, headID)` (:127), `Revoke(app, headID)` (:132),
  `Resolve(app, token)` (:110). Test seeding pattern: `heads_test.go:17-56`
  (`newApp`, `seedMemory`, grant construction — read it; reuse it).
- Scoped checks (`internal/heads/scoped.go:36-61`): head `status` must be
  `active`; head `expires` (when set) must be future; per-grant `expires`
  (when set) must be future; grant `read`/`write` booleans per mode. Audit
  rows are written for every decision (`allow`, :27-34).
- VM invoke (`internal/ext/vm.go:144-163`): `done` channel + goroutine
  selecting `ctx.Done()` / `time.After(invokeTimeout)` / `done`;
  `vm.Interrupt(...)` on the first two. `invokeTimeout` is a package const
  (30s) — your cancellation test uses a SHORT `context.WithTimeout`, never
  the 30s path. Extension fixtures: see how `ext_test.go` writes `.js`
  files into a temp `pb_extensions` dir and approves them
  (`TestApprovePinsServesAndAudits`, :82).
- SSE wire shape accepted by the parser (mirror of `scripts/fake-model.py`
  and the fragment-assembly behavior): events are
  `data: {json}\n\n` lines, text deltas in
  `choices[0].delta.content`, tool-call fragments in
  `choices[0].delta.tool_calls[]` with `index`-keyed accumulation, the
  stream ends with `data: [DONE]`. Read `openai.go:112-197` once before
  writing the cases — your tests must assert against what the code actually
  accumulates (`Chunk.ToolCalls` on the Done chunk).

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Focused | `go test ./internal/llmtest/... ./internal/heads/ ./internal/ext/ ./internal/llm/ -v` | new tests pass |
| Race-check the fake | `CGO_ENABLED=1 go test -race ./internal/llmtest/... ./internal/turn/...` | ok |
| Full gates | `gofmt -l .` / `go vet ./...` / `go test ./...` | clean / 0 / ok |

Sandbox note: TLS failures → `docs/hyperagent-sandbox.md`; use `-p 1` if
memory-tight.

## Scope

**In scope**:
- `internal/llmtest/llmtest.go` (create — test-helper package, mirrors the
  `storetest` precedent)
- `internal/turn/turn_test.go`, `internal/cli/cli_test.go`,
  `internal/tasks/nudge_test.go`, `internal/tasks/briefing_test.go` (if it
  has its own fake), `internal/recap/generate_test.go` — switch to llmtest
- `internal/heads/heads_test.go` (new cases)
- `internal/ext/ext_test.go` (VM cancellation case)
- `internal/llm/openai_test.go` (create — SSE parser cases)

**Out of scope** (do NOT touch):
- ANY production file. If a new test exposes a real bug (e.g. grant expiry
  not enforced), mark the test `t.Skip("BUG: ...")` with the evidence and
  report — fixing is a separate change.
- `internal/llm/kronk_test.go` — keep as-is.
- The `invokeTimeout` constant — do not shorten it for testability; the ctx
  path covers the interrupt mechanism.

## Git workflow

- Branch: `advisor/011-boundary-tests`; commits per area:
  `test(llmtest): shared scripted llm.Client fake`,
  `test(heads): expiry, cross-head isolation, revoke`,
  `test(ext): context cancellation interrupts a running handler`,
  `test(llm): OpenAI SSE parser wire cases`. No push/PR unless instructed.

## Steps

### Step 1: internal/llmtest

Create `internal/llmtest/llmtest.go`, package doc modeled on
`internal/storetest/storetest.go`'s ("shared test fake so every package's
tests script the model the same way; tests never hit a real model —
AGENTS.md"). Port the turn_test fake verbatim with exported names:
`type Reply struct { Text string; Calls []llm.ToolCall }`,
`type ScriptedClient struct { ... Calls int }`,
`func New(replies ...Reply) *ScriptedClient`,
plus `func Text(s string) Reply` and `func ToolCall(id, name, args string) Reply`
conveniences. Keep the mutex; keep `Embed` returning `nil, nil`.

**Verify**: `go vet ./internal/llmtest/` → 0.

### Step 2: Migrate the four fakes

Replace each package's local fake with `llmtest.New(...)`:
- `turn_test.go` — drop `fakeClient`/`fakeReply`, map each
  `fakeReply{text: ..., calls: ...}` to `llmtest.Reply{Text:, Calls:}`;
  anything reading `client.calls` uses `client.Calls`.
- `cli_test.go` — `scriptedClient` likewise (it is swapped in via the
  `chatClients` package var, `cli/chat.go:21` — keep that injection point).
- `nudge_test.go` / `briefing_test.go` — their minimal fakes become
  `llmtest.New(llmtest.Text("..."))` (or an error-returning knob: add
  `Err error` to `ScriptedClient` ONLY if one of these tests needs it —
  read them first; the nudge fake has an `err` field at :17-34).
- `recap/generate_test.go` — `echoClient` echoes input; if its behavior is
  load-bearing (summaries derived from prompts), add an optional
  `Respond func(msgs []llm.Message) string` hook to `ScriptedClient`
  instead of forcing scripted replies. Keep the hook nil-default.

**Verify**: `go test ./internal/turn/ ./internal/cli/ ./internal/tasks/ ./internal/recap/ -p 1` → all pass with zero behavior changes.

### Step 3: Heads boundary cases

Add to `heads_test.go` (reusing `newApp`/`seedMemory`):
- `TestExpiredHeadDeniesAccess` — `Spawn` with `ttl = -time.Minute` (or set
  the head's `expires` to the past after spawn, matching how `Spawn`
  records ttl — read Spawn:44-108 first); `AsHead(...).Records("memories", ...)`
  → error wrapping `ErrDenied`; assert an `audit_log` row with
  `allowed=false` exists for the attempt.
- `TestExpiredGrantDeniesAccess` — active head, grant with past `expires`
  → denied.
- `TestCrossHeadIsolation` — head A granted `memories`, head B granted
  `skills`; A reading `skills` → denied; B reading `memories` → denied;
  each head's own target → allowed.
- `TestRevokeClosesHead` — `Revoke(app, headID)` → subsequent access
  denied AND `Resolve(app, token)` fails; assert the revoke audit action
  (read `Revoke`/`closeHead` in heads.go:127+ for the exact action string).

**Verify**: `go test ./internal/heads/ -v` → all (old + 4 new) pass.

### Step 4: VM cancellation

Add to `ext_test.go`: `TestContextCancellationInterruptsHandler` — approve
an extension whose handler is an infinite loop
(`handler: function () { for (;;) {} }` — follow the fixture style of
`TestApprovePinsServesAndAudits`), call the tool's `Execute` with
`context.WithTimeout(ctx, 200*time.Millisecond)`, assert it returns within
~2s (wrap in a `time.After` watchdog in the test) with an error mentioning
the interrupt. This proves `vm.Interrupt` actually stops a busy loop.

**Verify**: `go test ./internal/ext/ -run TestContextCancellation -v` →
passes in < 5s.

### Step 5: SSE parser cases

Create `internal/llm/openai_test.go` driving `OpenAIClient.ChatStream`
against `httptest.NewServer` responses (Content-Type `text/event-stream`):
1. text delta + `[DONE]` → collected text matches.
2. tool call split across THREE `data:` events (same `index`, `arguments`
   fragmented as `{"tit`, `le":"X`, `"}`) → final Done chunk's ToolCalls[0]
   has name + reassembled args.
3. two interleaved tool calls (`index` 0 and 1) → both assembled, in order.
4. connection closed WITHOUT `[DONE]` → a Chunk with non-nil Err arrives
   (read openai.go's scanner-error path first to assert the right shape).
Use `llm.Collect` where it simplifies (it exists — grep `func Collect`
in `internal/llm/`).

**Verify**: `go test ./internal/llm/ -v` → kronk tests + 4 new SSE cases pass.

### Step 6: Full gates + race

**Verify**: `gofmt -l .` empty; `go vet ./...` 0; `go test -p 1 ./...` ok;
`CGO_ENABLED=1 go test -race ./internal/llmtest/... ./internal/turn/... ./internal/llm/` ok.

## Test plan

This plan IS tests; the structural patterns to follow are named per step
(heads_test seeding, ext_test fixtures, table-driven style per
`recur_test.go`). Net new: ≥ 9 test functions + 1 helper package.

## Done criteria

- [ ] `internal/llmtest/llmtest.go` exists; `grep -rln "type fakeClient\|type scriptedClient\|type echoClient" internal/ | grep -v llmtest` → no matches
- [ ] 4 new heads tests, 1 new ext test, ≥ 4 SSE cases — all passing (`go test ./... -p 1` exit 0)
- [ ] Race run (Step 6) exit 0
- [ ] No production files changed: `git diff --name-only | grep -v _test.go | grep -v llmtest | grep -v plans/` → empty
- [ ] `plans/README.md` status row updated

## STOP conditions

- A new boundary test FAILS against current code (e.g. expired grants are
  honored) — that is a real vulnerability finding: mark `t.Skip("BUG: …")`,
  stop, and report with the failing assertion. Do not fix scoped.go here.
- `Spawn`'s ttl semantics don't allow constructing an expired head (e.g. it
  clamps negatives) — set the record's `expires` field directly via the app
  and note it; if that's also impossible, report.
- The interleaved-tool-call SSE case fails because the parser only supports
  a single call — verify against `fake-model.py` (which only ever sends
  index 0) and downgrade case 3 to a skip-with-comment rather than
  "fixing" the parser.

## Maintenance notes

- `llmtest.ScriptedClient` is now the ONLY fake to update when `llm.Client`
  grows a method. If plan/decision ever splits `Embed` off the interface
  (see DEBT notes in the audit), llmtest is the single test-side touchpoint.
- Plan 004's web harness uses a real `httptest` SSE server rather than
  llmtest (it must exercise the wire path); both are correct — don't
  consolidate them.
- The heads tests added here are the regression net the sub-head chat
  feature (roadmap) will rely on; extend them with route-level cases when
  that ships.
