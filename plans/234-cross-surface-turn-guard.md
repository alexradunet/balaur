# Plan 234: Cross-surface turn in-flight guard — one turn at a time on the master conversation (web + CLI + messenger)

> **Follow-up (a) of plan 231; designed via a Phase-0 concurrency audit.** Today
> `turn.Run` has NO concurrency guard: two turns from different gateways (e.g. a
> `balaur chat` cron + an open web session, or web + messenger) both target the one
> master conversation and corrupt it — context-snapshot races, interleaved message
> writes, compaction skew. `messengerMu` only guards messenger-vs-messenger. This
> generalizes it to a single cross-surface guard: a second concurrent turn gets an
> immediate "busy", never a silent corruption.
>
> **Drift check (run first)**:
> `git diff --stat <BASE>..HEAD -- internal/turn/turn.go internal/web/chat.go internal/cli/chat.go internal/web/messenger.go`
> (BASE = the commit this is dispatched against.) Compare excerpts before editing; on real drift, STOP.

## Status
- **Priority**: P3 (correctness) / **Effort**: M / **Risk**: MEDIUM–HIGH (concurrency; touches all 3 turn gateways; a wrong lock could deadlock or stall web streaming)
- **Depends on**: 231 (messenger gateway) — merged
- **Category**: correctness / concurrency
- **Planned at**: dispatched against current `main`

## Why this matters (the audit's findings)

A Phase-0 concurrency audit of `internal/turn/turn.go` `Run` found:
- **No serialization exists.** The only mutex tied to turns is `messengerMu` (messenger-vs-messenger only). `turn.Run` is fully unguarded for concurrent calls.
- **All turns share one conversation.** Every gateway resolves `conversation.Master(app)` → the same record; `turn.Run` reads `RecentTurns` (turn.go:82) BEFORE persisting the user message (turn.go:86), then writes many messages (turn.go:136–163).
- **Concrete corruption from two concurrent turns:** (1) both read the same prior context before either persists → neither sees the other's user message; (2) their multi-message `AppendOrigin` writes interleave non-deterministically → garbled history on the next `RecentTurns`; (3) compaction-boundary skew.
- **Three callers of `turn.Run`:** `internal/web/chat.go:52` (SSE streaming), `internal/cli/chat.go:90` (buffered JSON), `internal/web/messenger.go:113` (buffered, already TryLock-guarded locally).

## The design (audit's recommendation — follow it exactly)

**A single process-wide in-flight guard, TryLock semantics, acquired by each gateway BEFORE its medium-specific setup. `turn.Run`'s signature does NOT change, and the guard is NOT inside `turn.Run`.**

Why not inside `turn.Run`: web opens the SSE stream and paints the user bubble (`cs.start()`) BEFORE calling `turn.Run`. A guard inside `Run` (or acquired after `cs.start`) would either stall a live SSE stream silently (blocking lock) or reject after the bubble is painted (orphaned bubble). So the guard must be acquired at the very top of each gateway, before any medium setup.

Why TryLock (not blocking): at single-owner v1, a second concurrent turn is always a race (two surfaces firing at once), never a work queue. Immediate "busy" is correct for all three media; a blocking lock would stall an SSE connection or block a bridge/CLI thread.

Why global (not per-conversation): every turn targets the one master conversation in v1, so a single lock is correct and simplest (KISS). Document that keying by conversation id is the future generalization if multiple turn-bearing conversations ever exist.

### New guard (in `internal/turn`, NOT inside `Run`)
Add a small `internal/turn/guard.go`:
```go
// TryBegin acquires the process-wide "a turn is in flight" guard. It returns an
// end func to release it and ok=true on success; ok=false means another turn is
// already running (the caller must reject with a medium-appropriate "busy").
// One master conversation in v1 → a single global guard; key by conversation id
// if multiple turn-bearing conversations are ever added.
func TryBegin() (end func(), ok bool)
```
Backed by a package-level `sync.Mutex` + `TryLock`. `end` calls `Unlock` exactly once (guard against double-call).

### Gateway wiring (acquire BEFORE setup; release on EVERY path; reject cleanly on busy)
- **`internal/web/chat.go` `handlers.chat()`** — call `end, ok := turn.TryBegin()` at the TOP, **before `newChatStream`/`cs.start()`**. If `!ok`: return a clean "busy" WITHOUT painting a user bubble or opening the stream — reuse the existing toast/error idiom (e.g. an `emitToast`-style note "One message is still being answered — try again in a moment") or a plain error response; NO `#chat` mutation. If `ok`: `defer end()`, then proceed exactly as today (`cs.start` → `turn.Run` → `cs.finish`).
- **`internal/cli/chat.go`** — `end, ok := turn.TryBegin()` before `turn.Run` (turn.go:90 site). `!ok` → return the JSON error envelope `{"error":"busy: a turn is already in progress"}` (non-zero exit). `ok` → `defer end()`.
- **`internal/web/messenger.go`** — REPLACE the local `messengerMu` (var at :53, TryLock at :90, Unlock defer at :93) with `turn.TryBegin()`: `end, ok := turn.TryBegin(); if !ok { return 429 busy }; defer end()`. Remove the now-unused `messengerMu` var and the `sync` import if it becomes unused. Keep the acquire AFTER auth/consent (guard the turn, not the auth) but before `turn.Run`.

## Commands you will need
| Purpose | Command | Expected |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Race | `TMPDIR=/home/alex/.cache/go-tmp CGO_ENABLED=1 go test -race ./internal/turn/... -count=1` | PASS (guard is concurrency code — race-test it) |
| Test | `go test ./internal/turn/... ./internal/web/... ./internal/cli/... -count=1` | PASS |
| Full | `go test ./... -count=1` | all pass |
| gofmt | `gofmt -l internal/turn internal/web internal/cli` | nothing |

> Prefix with `TMPDIR=/home/alex/.cache/go-tmp`; `-count=1`. Commit FOREGROUND. No `make vulncheck`. The race target uses CGO_ENABLED=1 (the ONLY CGO exception — for `-race`); the shipped build stays CGO-free.

## Scope
**In scope**:
- `internal/turn/guard.go` (new — `TryBegin`) + `internal/turn/guard_test.go`.
- `internal/web/chat.go` (acquire before cs.start; busy path).
- `internal/cli/chat.go` (acquire before turn.Run; busy path).
- `internal/web/messenger.go` (replace `messengerMu` with `turn.TryBegin`; update its comment — the messenger-local mutex is now the shared cross-surface guard).
- Tests in the touched packages.

**Out of scope** (do NOT touch):
- `turn.Run`'s signature/body — the guard is EXTERNAL to it.
- The cron single-appends (nudge/briefing/recap) — they post a single message, not a full turn; not in scope (a single atomic append does not race like a multi-message turn).
- Any behavior of a turn in the non-concurrent (normal) case — it must be byte-for-byte unchanged.

## Git workflow
- Branch: `advisor/234-cross-surface-turn-guard`
- Subject e.g. `fix(turn): cross-surface in-flight guard — one turn at a time on the master conversation`
- Do NOT push.

## Steps

### Step 1: `turn.TryBegin` + race-tested unit test
Add `internal/turn/guard.go` with `TryBegin() (end func(), ok bool)` over a package-level `sync.Mutex`/`TryLock`; `end` releases once (idempotent or documented single-call). Add `guard_test.go`: first `TryBegin` → ok; a second before release → !ok; after `end()` → ok again; a `-race` test firing N goroutines asserts exactly one holds at a time.
**Verify**: `go test ./internal/turn/... -count=1` and the `-race` target → PASS.

### Step 2: Wire the three gateways
Per "Gateway wiring". Web: acquire at the very top of `handlers.chat()`, before `cs.start()`; busy path paints NO bubble. CLI: before `turn.Run`. Messenger: replace `messengerMu` with `turn.TryBegin` (remove the old var/import if unused). Each `defer end()` on the ok path.
**Verify**: `go build ./... && go vet ./...` → exit 0; `gofmt -l internal/turn internal/web internal/cli` → clean.

### Step 3: Tests
- **Guard unit + race** (Step 1).
- **Messenger still rejects concurrent** with 429 (its existing `TestMessengerInFlight` should still pass, now via the shared guard — update it if the mechanism name changed, not the behavior).
- **CLI/web busy path** (best-effort, non-flaky): if a blocking fake client is already used by existing tests, add a case that a second call while one is in flight is rejected "busy". If deterministic concurrency is too flaky, assert the guard is acquired/released on the normal path (a turn runs, and afterward `TryBegin` succeeds again — proving release) and rely on the guard unit test for the concurrency proof.
- **No regression**: the existing web/cli/messenger turn tests pass unchanged (normal single-turn behavior identical).
**Verify**: `go test ./internal/turn/... ./internal/web/... ./internal/cli/... -count=1` → PASS.

### Step 4: Full verification
- `go test ./... -count=1` → all pass; the `-race` guard test → PASS
- `gofmt -l` clean; `go vet ./...` → exit 0

## Test plan
The guard unit `-race` test is the correctness core (exactly one holder). The gateway tests prove each surface rejects "busy" cleanly and releases on the normal path. The decisive property: a normal, non-concurrent turn behaves EXACTLY as before on all three surfaces.

## Done criteria — ALL must hold
- [ ] `turn.TryBegin` exists; a `-race` test proves exactly one turn holds the guard at a time and it's released after `end()`.
- [ ] Web `handlers.chat()` acquires BEFORE `cs.start()`; on busy it paints NO user bubble / opens no stream and returns a clean "busy" signal.
- [ ] CLI rejects a concurrent turn with a "busy" JSON error; messenger rejects with 429 via the shared guard (old `messengerMu` removed).
- [ ] `turn.Run`'s signature/body unchanged; normal single-turn behavior unchanged on all three surfaces (existing tests green).
- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0; `gofmt -l` clean; `go test ./... -count=1` green; the `-race` guard test passes.
- [ ] `plans/README.md` status row updated.

## STOP conditions
- If acquiring the guard before `cs.start()` in `web/chat.go` proves impossible without restructuring the SSE flow broadly, STOP and report — do NOT acquire after `cs.start` (that orphans the user bubble on busy) and do NOT use a blocking lock in the web path (it stalls the SSE stream silently).
- If removing `messengerMu` changes any of 231's four security constraints, STOP — the guard swap must preserve them exactly.
- If a global guard would wrongly serialize a legitimate NON-turn path (e.g. it turns out some cron DOES call `turn.Run`), report — the audit found only the 3 gateways call `turn.Run`; if a 4th caller exists, reconsider scope.

## Maintenance notes
- Generalizes `messengerMu` to a cross-surface guard. Keyed globally (one master conversation in v1); key by conversation id if multiple turn-bearing conversations are ever added.
- Reviewer (adversarial): verify (1) `end()` releases on EVERY return path incl. the honesty-check retry and error paths, (2) no deadlock (no nested `TryBegin`), (3) web busy path leaves no orphaned DOM/stream state, (4) the `-race` test actually exercises concurrency, (5) 231's messenger security constraints are intact after the `messengerMu` swap.
