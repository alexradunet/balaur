# Plan 001: Serialize the nudge/recap/briefing background jobs so overlapping runs cannot double-fire

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat c4fce47..HEAD -- main.go internal/tasks/nudge.go internal/recap/generate.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `c4fce47`, 2026-06-12
- **Issue**: https://github.com/alexradunet/balaur/issues/16

## Why this matters

`main.go` wires three background jobs (recap, nudge, briefing). Each is
registered as a PocketBase cron AND fired once immediately via `go run()` at
serve start. PocketBase's cron runs every tick in its own goroutine
(`tools/cron/cron.go:196` in pocketbase v0.39.3 — `go func()` per job per
tick), so two instances of the same job can run concurrently:

1. The serve-start goroutine overlaps the first scheduled tick.
2. A slow run overlaps the next tick: `tasks.Nudge`'s optional model call has
   a 60-second timeout (`composeTimeout`, nudge.go:28) — exactly the cron
   interval (`* * * * *`).

Consequences, confirmed by reading the code:

- **Duplicate nudges**: `Nudge` SELECTs due tasks (`nudged_at = ''`), then
  composes (up to 60s), then appends the chat message, and only THEN marks
  `nudged_at`. Two overlapping runs both see the unmarked tasks and both
  post a reminder message. For a product whose identity is "verify, don't
  trust", duplicate reminders are trust damage.
- **Aborted recap catch-up**: `recap.ensureOne` is check-then-act
  (`Find` → slow LLM call → `save`) and the `summaries` collection has a
  UNIQUE index `idx_summaries_period` (migrations/1749800000_summaries.go:40).
  Two overlapping `EnsureSummaries` runs race; the loser's `app.Save` fails
  the unique constraint, which `EnsureSummaries` treats as a fatal error and
  aborts the rest of the catch-up (remaining weeks/months/years wait for the
  next hourly tick). It also wastes duplicate local-LLM inference.
- **Briefing**: same overlap shape (minute cron + 60s compose timeout in
  `composeBriefing`); the idempotency check `BriefedToday` runs before the
  slow compose, so two overlapping runs can both pass it.

The fix is one process-level guard per job: a `sync.Mutex` with `TryLock` —
an overlapping run simply skips. No schema change, no behavior change beyond
removing the duplicates.

## Current state

- `main.go` — the only file to modify. All three register functions share
  this shape (recap shown; nudge `:91-107` and briefing `:114-134` are
  analogous):

```go
// main.go:61-83
func registerRecap(app core.App) {
	if os.Getenv("BALAUR_RECAP") == "0" {
		return
	}
	var clients turn.ClientSource
	run := func() {
		client, err := clients.Active(app)
		if err != nil {
			return // no model configured; recap waits
		}
		...
		if err := recap.EnsureSummaries(ctx, app, client, master.Id, time.Now()); err != nil {
			app.Logger().Warn("recap: catch-up stopped", "error", err)
		}
	}
	app.Cron().MustAdd("recap", "0 * * * *", run)
	go run() // serve-start catch-up, off the serve path
}
```

- `internal/tasks/nudge.go:45-75` — `Nudge` reads due tasks, appends ONE
  message, then marks each task's `nudged_at`. Do NOT restructure this file;
  the guard lives in `main.go`.
- `internal/recap/generate.go:140-146` — `ensureOne`'s idempotency check
  (`Find(app, conversationID, p) != nil`) is safe under the mutex because
  only one run is in flight per process.
- Repo conventions (AGENTS.md): KISS — smallest correct change; no global
  mutable state (a function-scoped `sync.Mutex` captured by the two closures
  is not package-global state); `gofmt` is law.

## Commands you will need

| Purpose | Command | Expected on success |
|---|---|---|
| Format | `gofmt -l .` | empty output |
| Vet | `go vet ./...` | exit 0 |
| Tests | `go test ./...` | all packages ok |
| Build | `CGO_ENABLED=0 go build -o /tmp/balaur-test .` | exit 0 |

Sandbox note: if `go` commands fail with `x509: certificate signed by
unknown authority`, follow `docs/hyperagent-sandbox.md` (GOPROXY shim via
`scripts/goproxy-shim.py`; bound parallelism with `-p 2` / `-p 1`).

## Scope

**In scope** (the only files you should modify):
- `main.go`

**Out of scope** (do NOT touch, even though they look related):
- `internal/tasks/nudge.go`, `internal/tasks/briefing.go`,
  `internal/recap/generate.go` — the duplicate-claim window inside `Nudge`
  (message appended before marks are written) is a separate, smaller issue;
  plan 001 only removes concurrent runs. Changing the domain packages risks
  their tests.
- `internal/turn/models.go` (`ClientSource`) — its internal mutex is about
  client caching, not job overlap.
- Cron schedules — keep `0 * * * *` and `* * * * *` exactly as they are.

## Git workflow

- Branch: `advisor/001-serialize-background-jobs`
- Commit style (matches repo history): `fix(main): serialize background jobs with per-job TryLock guards` plus a body explaining the overlap windows. Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add a per-job mutex guard in each register function

In `main.go`, inside each of `registerRecap`, `registerNudge`,
`registerBriefing` (after the env-gate check, before `run` is defined), add:

```go
var mu sync.Mutex
```

and wrap the body of `run` so an in-flight run makes the new one skip:

```go
run := func() {
	if !mu.TryLock() {
		return // a previous run is still in flight; this tick skips
	}
	defer mu.Unlock()
	// ...existing body unchanged...
}
```

Add `"sync"` to the imports. The mutex is captured by both the cron closure
and the `go run()` call, serializing them. `TryLock` (Go 1.18+) skips
rather than queues — queuing minute ticks behind a slow model call would
build an unbounded backlog.

**Verify**: `gofmt -l .` → empty; `go vet ./...` → exit 0.

### Step 2: Confirm behavior is otherwise unchanged

`go build` and run the full suite. No test exercises `main.go` directly
(it is wire-up), so the suite passing proves you broke nothing downstream.

**Verify**: `go test ./...` → all ok. `CGO_ENABLED=0 go build -o /tmp/balaur-test .` → exit 0.

### Step 3: Manual smoke check of the guard

```bash
grep -c "mu.TryLock()" main.go
```

**Verify**: output is `3` (one guard per job).

## Test plan

No new automated tests in this plan: `main.go` is intentionally untestable
wire-up (AGENTS.md: "main.go stays thin"), and the domain packages are out
of scope. Coverage for this class of bug arrives with plan 006 (CI `-race`)
— note for the reviewer that `-race` on the existing suite plus the e2e
harness is the regression net here. If you want a belt-and-braces check,
run `go vet ./...` and read the three closures once more.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -c "mu.TryLock()" main.go` prints `3`
- [ ] `gofmt -l .` prints nothing
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0
- [ ] `CGO_ENABLED=0 go build -o /tmp/balaur-test .` exits 0
- [ ] `git diff --stat` shows changes ONLY in `main.go` (and `plans/README.md`)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- `main.go` no longer contains the three `registerRecap` / `registerNudge` /
  `registerBriefing` functions in the shape excerpted above.
- The Go version in `go.mod` is below 1.18 (no `TryLock`) — it is 1.26.4 at
  planning time, so this indicates major drift.
- You find an existing synchronization mechanism already added around these
  jobs (someone fixed it since `c4fce47`).

## Maintenance notes

- The guard serializes within ONE process. Two `balaur serve` processes on
  the same `pb_data` would still race; that is out of scope (SQLite locking
  makes that setup unsupported anyway).
- Residual (deliberately deferred): inside `tasks.Nudge`, the message is
  appended before `nudged_at` marks are written; if the process crashes
  between the two, the next run re-fires those nudges. Rare and self-healing,
  but if it ever matters, the fix is to mark first and append after.
- If a future change makes recap runs very long (e.g. year-one backlog on a
  slow model), skipped hourly ticks are correct behavior: the in-flight run
  is already doing the catch-up.
- Reviewer should scrutinize: that `defer mu.Unlock()` is inside the
  `TryLock` success path (unlocking an un-held mutex panics).
