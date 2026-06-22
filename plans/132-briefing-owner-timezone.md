# Plan 132: Fire the morning briefing in the owner's configured timezone

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in "STOP conditions" occurs, stop and report. When done, update the
> status row for this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat b61e060..HEAD -- main.go internal/tasks/briefing.go`
> Compare the "Current state" excerpt against the live `main.go`; on a mismatch,
> treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `b61e060`, 2026-06-21

## Why this matters

The owner can pin a timezone in `owner_settings` (`timezone` key, resolved by
`store.OwnerLocation`), and the embedded `tzdata` import exists specifically to
honor it on minimal hosts. The hourly recap cron does this correctly:
`registerRecap` passes `time.Now().In(store.OwnerLocation(app))`. But
`registerBriefing` passes a bare `time.Now()` (host-local). The briefing's hour
gate (`now.Hour() < hour`) and its once-per-day idempotency boundary
(`BriefedToday` computes "local midnight" via `now.Location()`) therefore run in
the *host* zone, not the owner's. For an owner whose pinned zone differs from the
host OS, the briefing fires at the wrong wall-clock hour, and the "since local
midnight" window can be off by the UTC-offset delta — producing a briefing a few
hours early/late or, at a day boundary, a duplicate or a skipped day. Recap got
this right; briefing diverged. The fix is to make the call site mirror recap.

## Current state

`main.go` — `registerRecap` (correct) vs `registerBriefing` (the bug):
```go
// registerRecap (line ~135):
if err := recap.EnsureSummaries(ctx, app, client, master.Id, time.Now().In(store.OwnerLocation(app))); err != nil { … }

// registerBriefing (line ~196):
if err := tasks.Briefing(app, client, time.Now(), hour); err != nil { … }   // <-- bare time.Now()
```
`store` is already imported in `main.go` (used by `registerRecap`).

`internal/tasks/briefing.go` is already zone-correct *given the `now` it
receives*: `Briefing` (45) gates on `now.Hour() < hour` (46) and `BriefedToday`
(33) computes midnight with `now.Location()` (34). So the ONLY change needed is
the caller passing the owner zone — `briefing.go` itself is not modified.

`registerNudge` (line ~164) also passes a bare `time.Now()`, but the nudger's
firing is UTC-correct (DB comparisons use `store.PBTime`); only its rendered
`Lateness` relative-time string is cosmetically in the host zone. Fixing it too
keeps all three crons consistent (optional Step 2).

## Commands you will need

| Purpose | Command                          | Expected |
|---------|----------------------------------|----------|
| Build   | `CGO_ENABLED=0 go build ./...`   | exit 0   |
| Tests   | `go test ./internal/tasks/`      | all pass |
| Format  | `gofmt -l .`                     | empty    |

## Steps

### Step 1: Pass the owner zone to the briefing

In `registerBriefing` in `main.go`, change the `Briefing` call to resolve the
owner zone exactly as `registerRecap` does:
```go
now := time.Now().In(store.OwnerLocation(app))
if err := tasks.Briefing(app, client, now, hour); err != nil {
    app.Logger().Warn("briefing: run stopped", "error", err)
}
```

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0;
`grep -n "Briefing(app, client, time.Now().In(store.OwnerLocation(app))" main.go`
returns a match (or the equivalent `now :=` form above is present).

### Step 2 (optional, same theme): owner zone for the nudge's rendered time

In `registerNudge`, likewise resolve `now := time.Now().In(store.OwnerLocation(app))`
and pass it to `tasks.Nudge`. This only changes the rendered relative-time
string; firing is unchanged. Skip if you want the tightest possible diff.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 3: Regression test the zone-sensitivity of the briefing boundary

Add a test in `internal/tasks/briefing_test.go` (create if absent, else append)
that pins the briefing math to the passed `now`'s zone. Suggested shape, using
the store test-app helper the package's other tests use:
- Seed an `origin=briefing` message at a UTC instant that is "today" in one zone
  but "yesterday" in another.
- Assert `BriefedToday(app, instant.In(zoneA))` and
  `BriefedToday(app, instant.In(zoneB))` differ as the midnight boundary dictates
  — proving the boundary follows the passed zone (which is what the Step 1 fix
  now feeds correctly).

If a clean seam to seed a briefing message in a test does not already exist in
this package's tests, instead add a focused unit test of `BriefedToday` that
asserts the midnight boundary is computed in `now.Location()` (e.g. two `now`
values at the same instant but different zones can yield different boundaries).

**Verify**: `go test ./internal/tasks/` → all pass, including the new test.

## Test plan

- New test guards that the briefing's once-per-day boundary is computed in the
  zone of the `now` it is given (the invariant the Step 1 caller fix relies on).
- Existing `internal/tasks` tests stay green (briefing.go logic unchanged).
- The `main.go` one-liner itself is verified by reading + the grep in Step 1
  (main wiring is not unit-tested in this repo).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0
- [ ] `grep -n "tasks.Briefing(app, client, time.Now()," main.go` returns NOTHING
      (the bare `time.Now()` call is gone)
- [ ] `grep -n "store.OwnerLocation(app)" main.go` shows it used in BOTH the recap
      and briefing registrations
- [ ] `go test ./internal/tasks/` passes, including the new test
- [ ] `gofmt -l .` empty; `git diff --check` clean
- [ ] Only `main.go`, `internal/tasks/briefing_test.go`, and `plans/readme.md`
      modified (`git status`) — `internal/tasks/briefing.go` is NOT modified
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:
- The "Current state" excerpt of `registerBriefing` doesn't match `main.go` (drift).
- The fix appears to require editing `internal/tasks/briefing.go` (it should not —
  that code is already zone-correct).
- You cannot find a clean way to seed/test a briefing message — fall back to the
  `BriefedToday` unit test described in Step 3 and note it.

## Scope

**In scope**: `main.go` (the `registerBriefing`, and optionally `registerNudge`,
call sites), `internal/tasks/briefing_test.go`, `plans/readme.md` (status row).

**Out of scope**: `internal/tasks/briefing.go` logic (already correct);
`registerRecap` (already correct); the broader question of task-PARSING timezone
(`internal/tools/tasks.go` uses host `time.Local` — that is a separate product
decision, NOT this bug).

## Git workflow

- Branch off `origin/main`: `improve/132-briefing-owner-timezone`.
- One commit; conventional subject, e.g.
  `fix(briefing): fire in the owner's configured timezone (mirror recap)`.
- Do NOT push or open a PR unless the operator instructs it.

## Maintenance notes

- Any future cron added to `main.go` that does wall-clock or per-day math must
  resolve `time.Now().In(store.OwnerLocation(app))`, not bare `time.Now()` — the
  recap and (now) briefing registrations are the pattern to copy.
- The separate task-parsing-timezone question (reminders parsed in host
  `time.Local`) is deliberately out of scope here; if the owner wants reminders
  to honor the pinned zone too, that is a follow-up that threads the zone through
  `tools.ParseDue`/`fmtDue`/`turn.nowLine`.
