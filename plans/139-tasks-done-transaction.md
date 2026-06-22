# Plan 139: Make the recurring `tasks.Done` completion+advance atomic

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report. When done, update the status row for
> this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat 0c06da8..HEAD -- internal/tasks/tasks.go`
> Compare the "Current state" excerpt of `Done` against the live code; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `0c06da8`, 2026-06-22

## Why this matters

Completing a recurring task does two writes that are logically one operation: it
appends a `completion` entry (`addEntry`, its own `app.Save`), then mutates and
saves the task record (advance `due`, clear `nudged_at`/`snoozed_until`). If the
process dies between them, a completion entry exists but the task's `due` is still
in the past — the nudger re-fires it as overdue, and the derived streak counts a
completion for a task that still reads as due. Wrapping the entry write + task
mutation in a single transaction makes the pair all-or-nothing. (The
non-recurring branch is a single `app.Save` and is already atomic — leave it.)

This introduces the repo's first `app.RunInTransaction`; keep the audit and the
post-commit completion count OUTSIDE the transaction (audit must never fail the
operation; the count is a read after the fact).

## Current state

`internal/tasks/tasks.go` — `Done` (87-126), the recurring branch (104-125):
```go
	if err := addEntry(app, "completion", rec.Id, nil, rec.GetString("title"), now); err != nil {
		return DoneResult{}, err
	}
	anchor := rec.GetDateTime("due").Time().In(now.Location())
	if rec.GetBool("recur_from_done") && !calendarRule(rule) {
		anchor = now
	}
	next := Next(rule, anchor, now)
	rec.Set("due", next.UTC())
	rec.Set("nudged_at", "")
	rec.Set("snoozed_until", "")
	if err := app.Save(rec); err != nil {
		return DoneResult{}, fmt.Errorf("saving task: %w", err)
	}
	n, _ := app.CountRecords("entries", dbx.HashExp{"kind": "completion", "task": rec.Id})
	store.Audit(app, "tasks", "task.done", rec.Id, true, map[string]any{"next_due": next.UTC().Format(time.RFC3339)})
	return DoneResult{Recurring: true, NextDue: next, Completions: int(n)}, nil
```

`addEntry` (204-223) already takes `app core.App` as its first parameter, so it
works unchanged against a transaction app:
```go
func addEntry(app core.App, kind, taskID string, value map[string]any, text string, notedAt time.Time) error { … app.Save(rec) … }
```

`app.RunInTransaction(func(txApp core.App) error { … }) error` is the PocketBase
API (v0.39.3): the closure runs inside a DB transaction; returning a non-nil
error rolls it back.

## Commands you will need

| Purpose | Command                                  | Expected |
|---------|------------------------------------------|----------|
| Build   | `CGO_ENABLED=0 go build ./...`           | exit 0   |
| Tests   | `go test ./internal/tasks/`              | all pass |
| Race    | `go test -race ./internal/tasks/`        | all pass |
| Lint    | `make lint`                              | exit 0   |

## Steps

### Step 1: Wrap the completion entry + task save in one transaction

Replace the recurring branch's `addEntry(...) → … → app.Save(rec)` sequence so
both writes run inside `app.RunInTransaction`, threading the transaction app
into both `addEntry` and `Save`. Compute `anchor`/`next` and set the fields as
today, but perform the two writes inside the closure:
```go
	anchor := rec.GetDateTime("due").Time().In(now.Location())
	if rec.GetBool("recur_from_done") && !calendarRule(rule) {
		anchor = now
	}
	next := Next(rule, anchor, now)
	rec.Set("due", next.UTC())
	rec.Set("nudged_at", "")
	rec.Set("snoozed_until", "")
	if err := app.RunInTransaction(func(txApp core.App) error {
		if err := addEntry(txApp, "completion", rec.Id, nil, rec.GetString("title"), now); err != nil {
			return err
		}
		if err := txApp.Save(rec); err != nil {
			return fmt.Errorf("saving task: %w", err)
		}
		return nil
	}); err != nil {
		return DoneResult{}, err
	}
	n, _ := app.CountRecords("entries", dbx.HashExp{"kind": "completion", "task": rec.Id})
	store.Audit(app, "tasks", "task.done", rec.Id, true, map[string]any{"next_due": next.UTC().Format(time.RFC3339)})
	return DoneResult{Recurring: true, NextDue: next, Completions: int(n)}, nil
```
Keep the `CountRecords` and `store.Audit` OUTSIDE the transaction (after it
commits). Do NOT change the non-recurring branch (96-104).

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 2: Full gate

**Verify**: `go test ./internal/tasks/` → all pass; `go test -race ./internal/tasks/`
→ all pass; `make lint` → exit 0.

## Test plan

- The happy path must be unchanged: extend or add a test in
  `internal/tasks/tasks_test.go` (find the existing recurring-`Done` test —
  search for `Done(` with a recurring task) asserting that after `Done` on a
  recurring task: (a) exactly one new `completion` entry exists for the task,
  (b) the task's `due` advanced to `next` (in the future), (c) `nudged_at` and
  `snoozed_until` are cleared, (d) `DoneResult.Recurring` is true and
  `Completions` counts the entry. This proves the transactional path produces the
  same result as before.
- The crash-rollback case (process dies mid-transaction) cannot be unit-tested
  without a failure-injection seam; the atomicity is guaranteed by
  `RunInTransaction` by construction. Note this in the test comment — do NOT
  build a brittle failure-injection harness for it here.

## Done criteria

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go test -race ./internal/tasks/` passes, including the recurring-Done assertions
- [ ] `make lint` exits 0
- [ ] `grep -n "RunInTransaction" internal/tasks/tasks.go` returns a match in `Done`
- [ ] `store.Audit` and `CountRecords` for the recurring branch are OUTSIDE the transaction closure (verify by reading)
- [ ] The non-recurring branch (single `app.Save`) is unchanged
- [ ] Only `internal/tasks/tasks.go` (+ its test) and `plans/readme.md` modified
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report if:
- `core.App` does not expose `RunInTransaction(func(core.App) error) error` on
  this PocketBase version (`go doc github.com/pocketbase/pocketbase/core.App | grep -i RunInTransaction`) — report; do not hand-roll a transaction.
- Threading `txApp` into `addEntry` causes a type mismatch (it takes `core.App`;
  the tx app should satisfy it) — report the exact error.
- Existing recurring-`Done` tests fail in a way that suggests the transaction
  changed observable behavior beyond atomicity.

## Scope

**In scope**: `internal/tasks/tasks.go` (the recurring branch of `Done` only),
`internal/tasks/tasks_test.go`, `plans/readme.md` (status row).
**Out of scope**: the non-recurring `Done` branch; `Nudge` (its per-record
saves are documented best-effort catch-up — not this plan); `Snooze`/`Drop`;
`addEntry`'s signature (unchanged — it already takes `core.App`).

## Git workflow

- Branch off `origin/main`: `improve/139-tasks-done-transaction`.
- One commit; subject e.g. `fix(tasks): make recurring Done completion+advance atomic`.
- Do NOT push or open a PR.

## Maintenance notes

- This is the first `RunInTransaction` in `internal/` — if other multi-write
  domain operations need atomicity later (e.g. a multi-entry log op), this is the
  pattern to copy: writes inside the closure, audit + read-after-count outside.
- If `addEntry` ever gains side effects beyond the single `Save`, re-check that
  they belong inside the transaction.
