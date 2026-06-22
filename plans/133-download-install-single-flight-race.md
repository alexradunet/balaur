# Plan 133: Fix the download/install single-flight check-then-act race

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in "STOP conditions" occurs, stop and report. When done, update the
> status row for this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat b61e060..HEAD -- internal/web/models.go`

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `b61e060`, 2026-06-21

## Why this matters

The model-download and runtime-install handlers each guard against a second
concurrent run with a check-then-act sequence: `GetOk(key)` to test, then a
separate `Set(key, cancel)`. Two near-simultaneous owner POSTs (a double-click, a
retried request, two tabs) can both pass the `GetOk` check before either `Set`s,
launching two concurrent writers to the same `<file>.part` (interleaved writes →
corrupt bytes, and the loser's checksum verify fails) or two concurrent runtime
installs racing the same staging→rename. This is the same TOCTOU class that plan
120 fixed in `store.SetOwnerSetting`; the atomic primitive
(`Store().GetOrSet`, which runs its setFunc once under the write lock) already
exists. Single-owner trust makes it unlikely, but a double-submit is the
realistic trigger.

## Current state

`internal/web/models.go` — two identical non-atomic guards:

`downloadOfficialModel` (213–224):
```go
if _, ok := h.app.Store().GetOk(downloadStoreKey); ok {
    return h.modelsPanel(e, "")
}
ctx, cancel := context.WithCancel(e.Request.Context())
h.app.Store().Set(downloadStoreKey, cancel)
defer func() {
    h.app.Store().Remove(downloadStoreKey)
    cancel()
}()
```

`installRuntime` (513–523): the same pattern on `runtimeInstallStoreKey`.

The stored value is the `context.CancelFunc` itself — the cancel handler
(`POST /ui/model/download/cancel` → `cancelDownload`, web.go:207) retrieves it
and calls it. So the fix must keep storing the cancel func (not a wrapper),
while claiming the slot atomically.

`h.app.Store()` is a `*store.Store[string, any]`; its
`GetOrSet(key string, setFunc func() any) any` runs `setFunc` only if the key is
absent, under the store's write lock (`tools/store/store.go:176`).

## Commands you will need

| Purpose | Command                                  | Expected |
|---------|------------------------------------------|----------|
| Build   | `CGO_ENABLED=0 go build ./...`           | exit 0   |
| Tests   | `go test ./internal/web/`                | all pass |
| Race    | `go test -race -run TestClaimInFlight -count=3 ./internal/web/` | all pass, no race |
| Format  | `gofmt -l internal/web/`                 | empty    |

## Steps

### Step 1: Add an atomic `claimInFlight` helper

In `internal/web/models.go`, add:
```go
// claimInFlight atomically claims a single-flight slot under key, storing
// cancel as the in-flight token (so cancelDownload can find and call it).
// It returns true iff this caller won the slot; a loser must cancel its own
// context and bail. GetOrSet runs setFunc only when the key is absent, under
// the store's write lock — so exactly one concurrent caller wins.
func claimInFlight(app core.App, key string, cancel context.CancelFunc) bool {
	won := false
	app.Store().GetOrSet(key, func() any {
		won = true
		return cancel
	})
	return won
}
```

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 2: Use it in both handlers

Replace the check-then-act block in `downloadOfficialModel`:
```go
ctx, cancel := context.WithCancel(e.Request.Context())
if !claimInFlight(h.app, downloadStoreKey, cancel) {
    cancel()
    return h.modelsPanel(e, "")
}
defer func() {
    h.app.Store().Remove(downloadStoreKey)
    cancel()
}()
```
Do the same in `installRuntime` with `runtimeInstallStoreKey`. Leave the
`Remove`+`cancel` defers as-is (only the winner reaches them).

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0;
`grep -n "GetOk(downloadStoreKey)\|GetOk(runtimeInstallStoreKey)" internal/web/models.go`
returns nothing.

### Step 3: Concurrency test the helper

Add `TestClaimInFlightSingleWinner` in `internal/web/models_test.go` (or
`handlers_test.go`): create a test app (`newWebApp(t)`), spawn N (e.g. 20)
goroutines that each call `claimInFlight(app, "test-key", cancel)` with their own
`cancel` (a no-op `func(){}` is fine, or a real `context.WithCancel`), collect
the results, and assert exactly ONE returned `true`. Use a `sync.WaitGroup` and
guard the winner count with a mutex or an atomic. Then `app.Store().Remove("test-key")`.

**Verify**: `go test -race -run TestClaimInFlight -count=3 ./internal/web/` → all
pass, no data race reported.

### Step 4: Full gate

**Verify**: `go test ./...` → all pass; `gofmt -l internal/web/` → empty;
`git diff --check` → clean.

## Test plan

- `TestClaimInFlightSingleWinner`: under `-race`, 20 concurrent claims on one key
  yield exactly one winner — the property that prevents two concurrent
  download/install writers.
- Existing model/handler tests stay green (the happy path — a single claim —
  behaves identically to the old `GetOk`/`Set`).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `grep -n "GetOk(downloadStoreKey)\|GetOk(runtimeInstallStoreKey)" internal/web/models.go` returns nothing
- [ ] `go test -race -run TestClaimInFlight -count=3 ./internal/web/` passes, no race
- [ ] `go test ./...` passes
- [ ] `gofmt -l internal/web/` empty; `git diff --check` clean
- [ ] Only `internal/web/models.go`, the test file, and `plans/readme.md` modified
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:
- `cancelDownload` (or any reader of `downloadStoreKey`/`runtimeInstallStoreKey`)
  expects a value type other than `context.CancelFunc` — the helper must keep
  storing the cancel func so those readers keep working.
- `Store().GetOrSet` is not available on this PocketBase version (`go doc
  github.com/pocketbase/pocketbase/tools/store.Store.GetOrSet` errors) — report;
  do not hand-roll a mutex without checking.
- The race test reports more than one winner (the helper is not atomic as
  written) — report rather than loosening the assertion.

## Scope

**In scope**: `internal/web/models.go`, `internal/web/models_test.go` (or
`handlers_test.go`), `plans/readme.md` (status row).

**Out of scope**: the `defer Remove`/`cancel` cleanup (already correct — plan 086
fixed the Set-nil-vs-Remove bug); `cancelDownload`'s logic (it keeps reading the
stored cancel func unchanged); the actual download/install SSE bodies.

## Git workflow

- Branch off `origin/main`: `improve/133-download-install-single-flight-race`.
- One commit; conventional subject, e.g.
  `fix(web): make download/install single-flight claim atomic (GetOrSet)`.
- Do NOT push or open a PR unless the operator instructs it.

## Maintenance notes

- Any future "only one of these may run at a time" guard on `app.Store()` should
  use `claimInFlight` (or `GetOrSet` directly), never `GetOk` + `Set` — that gap
  is the bug this plan closes.
- The helper stores the cancel func as the token deliberately, so the existing
  cancel endpoint keeps working without change.
