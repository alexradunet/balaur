# Plan 181: Stop a spurious nudge when a recurring task is completed before its due time

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 12a48bf..HEAD -- internal/tasks/tasks.go internal/tasks/tasks_test.go`
> If either in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition. (At planning time this diff was
> empty — the files were unchanged from 12a48bf.)

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `12a48bf`, 2026-06-24

## Why this matters

When the owner finishes a recurring habit *early* — e.g. an evening walk due
18:00, marked done at 15:00 the same day — Balaur re-nudges them at 18:00 for a
task they already completed. The bug is in `tasks.Done`: for a fixed/calendar
recurrence it computes the next occurrence as the first slot strictly after
`now` (the completion time). When the due is still in the future, that "next"
slot is the *current* due itself, unchanged — and because `Done` also clears
`nudged_at`, the nudger treats the still-pending occurrence as fresh and fires
again at the original due time. The fix advances to the first occurrence
strictly after the occurrence the owner just satisfied, i.e. strictly after
`max(due, now)`. After this lands, an early completion correctly schedules the
*next* day/week and never re-nudges the same occurrence; overdue completions
and interval-from-completion (`recur_from_done`) habits keep their current
behavior.

## Current state

Files:

- `internal/tasks/tasks.go` — the tasks **domain** package (PocketBase
  reads/writes for commitments). `Done` is the only function this plan edits.
  Note: this is NOT `internal/web/tasks.go` (the web gateway) — do not touch
  the web package.
- `internal/tasks/recur.go` — the pure recurrence primitive. `Next` and
  `calendarRule` live here. Read-only for this plan: do NOT change them.
- `internal/tasks/tasks_test.go` — the table/unit tests for the domain
  package. You add cases here.

### The bug, in `internal/tasks/tasks.go` `Done` (verbatim, lines 213–220)

```go
213		anchor := rec.GetDateTime("due").Time().In(now.Location())
214		// From-done anchoring is an interval concept; calendar-pattern rules
215		// keep their day-and-hour pattern even on records that predate the
216		// Create-time validation.
217		if rec.GetBool("recur_from_done") && !calendarRule(rule) {
218			anchor = now
219		}
220		next := Next(rule, anchor, now)
```

`Next` is called with `after = now`. When `anchor` (the due) is in the future,
`Next` returns the due unchanged (see below), so the just-satisfied occurrence
is re-scheduled and re-nudged.

### The primitive (do NOT change), `internal/tasks/recur.go` `Next` (verbatim, lines 78–96)

```go
78	// Next returns the first occurrence strictly after `after`, anchored on
79	// `due` for the wall-clock time of day. All math happens in due's Location
80	// (callers pass box-local times, the same convention recap uses for days);
81	// AddDate preserves the wall clock across DST.
82	//
83	// Skip-forward semantics (org-mode "++"): when `after` is far past `due`,
84	// the result lands once in the future — never a backlog of missed runs.
85	func Next(r Rule, due, after time.Time) time.Time {
86		switch r.Kind {
87		case "daily", "every":
88		step := 1
89		if r.Kind == "every" {
90		step = r.N
91		}
92		c := due
93		for !c.After(after) {
94		c = c.AddDate(0, 0, step)
95		}
96		return c
```

Signature confirmed: `Next(r Rule, due, after time.Time) time.Time` — returns
the first occurrence **strictly after `after`**, with `due` supplying the
wall-clock anchor. When `due.After(after)`, the `for !c.After(after)` loop body
never runs and `Next` returns `due` unchanged. That is the exact case this plan
fixes by passing a larger `after`.

### The helper this plan relies on (do NOT change), `internal/tasks/recur.go` `calendarRule` (verbatim, lines 123–128)

```go
123	// calendarRule reports whether the rule carries an intrinsic calendar
124	// pattern (specific weekdays / day of month) rather than an interval
125	// anchored on its due.
126	func calendarRule(r Rule) bool {
127		return r.Kind == "weekly" || r.Kind == "monthly"
128	}
```

`calendarRule` exists and returns true for `weekly`/`monthly`, false for
`daily`/`every`. The brief assumption holds — proceed.

### Why `recur_from_done` must keep `after = now`

In `Done`, the interval-from-completion path reassigns `anchor = now` (line
218). For that path the next occurrence must be exactly one interval step from
the completion instant, so `after` must stay `now`. `recur_from_done` is only
valid on interval rules — `normalizeRecur` (`internal/tasks/tasks.go:78-85`)
rejects `recur_from_done` on calendar rules at Create/Update time:

```go
78		if calendarRule(rule) {
79			if recurFromDone {
80				return due, fmt.Errorf("tasks: %s rules are calendar-anchored — recur_from_done applies to daily and every:<N>d habits", rule.Kind)
81			}
```

So the only records where `recur_from_done && !calendarRule(rule)` is true are
daily/every habits, and they take the `anchor = now` branch.

### Repo conventions that apply here (with exemplars)

- **Tests**: standard `testing`, table-driven where it reads well, NO assertion
  frameworks, NO `time.Sleep`. Pass an explicit `now time.Time` into `Done` and
  use fixed `time.Date(...)` values for determinism — exactly how
  `TestDoneCalendarRuleKeepsPattern` (`internal/tasks/tasks_test.go:96-128`) and
  `TestDoneRecurringFixedSchedule` (`internal/tasks/tasks_test.go:159-207`) do.
- **PocketBase-backed test app**: `storetest.NewApp(t)` boots a temp-dir app
  through the full migration chain — see every test in this file, e.g.
  `internal/tasks/tasks_test.go:15`.
- **Reload-and-rehydrate pattern**: records live in the `nodes` collection;
  to assert persisted state, reload with `app.FindRecordById("nodes", rec.Id)`
  then call `hydrate(got)` (the test-local helpers `hydrate`, `dehydrate`,
  `fmtTime`, and `nodes_Props` are already in this file — reuse them; do not
  redefine). Exemplar: `internal/tasks/tasks_test.go:192-202`.
- **errors as values / structured logging**: not changed by this plan — the
  edit is a 6-line arithmetic change with no new error paths or logging.
- **gofmt is law**: a PostToolUse hook + CI gate reformat Go on save; still run
  `gofmt -l .` before declaring done (must print nothing).

## Commands you will need

| Purpose          | Command                                  | Expected on success            |
|------------------|------------------------------------------|--------------------------------|
| Drift check      | `git diff --stat 12a48bf..HEAD -- internal/tasks/tasks.go internal/tasks/tasks_test.go` | empty (no in-scope drift) |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...`           | exit 0                         |
| Test (this pkg)  | `go test ./internal/tasks/`              | `ok ... internal/tasks`        |
| Test (all)       | `go test ./...`                          | all packages pass              |
| Vet              | `go vet ./...`                           | exit 0, no findings            |
| Format check     | `gofmt -l .`                             | prints nothing                 |
| Diff hygiene     | `git diff --check`                       | no whitespace errors           |

## Suggested executor toolkit

- Invoke the `go-standards` skill if available before editing — it covers this
  repo's testing idioms (table-driven, no `time.Sleep`, `storetest.NewApp`).

## Scope

**In scope** (the only files you may modify):

- `internal/tasks/tasks.go` — `Done` function only (the anchor/next block at
  lines 213–220).
- `internal/tasks/tasks_test.go` — add new test cases.

**Out of scope** (do NOT touch, even though they look related):

- `internal/tasks/recur.go` — `Next` and `calendarRule` are the primitives;
  changing `Next`'s strictly-after-`after` contract would ripple through
  `Create`/`Update`/`normalizeRecur`. The fix lives entirely in the caller.
- `tasks.Snooze`, `tasks.Update`, `tasks.Create`, `normalizeRecur` — unrelated
  verbs; leave them byte-for-byte unchanged.
- `internal/web/tasks.go` and `internal/tasks/nudge.go` — the gateway and the
  nudger consume `due`/`nudged_at`; they need no change once `Done` writes the
  correct next due.
- `internal/self/knowledge.md` — this is a behavior bugfix within an existing
  capability, not an architecture/capability change; do not edit it.

## Git workflow

- You are likely in an executor worktree based off `origin/main`. Land on the
  worktree branch; do NOT open a PR and do NOT push unless the operator told
  you to.
- Conventional-commit subject, e.g.:
  `fix(tasks): advance recurring due past the satisfied occurrence on early completion`
- Single commit for both the code and test change is fine (one logical unit).

## Steps

### Step 1: Fix the anchor/`after` computation in `Done`

In `internal/tasks/tasks.go`, replace the block at lines 213–220 (verbatim
above) with the version below. The change: introduce an `after` variable
defaulting to `now`; on the early-completion case (calendar/fixed rule whose
due is still in the future) bump `after` to the due so `Next` schedules the
slot strictly *after* the just-satisfied occurrence; keep `after = now` on the
`recur_from_done` interval path.

Replace exactly:

```go
	anchor := rec.GetDateTime("due").Time().In(now.Location())
	// From-done anchoring is an interval concept; calendar-pattern rules
	// keep their day-and-hour pattern even on records that predate the
	// Create-time validation.
	if rec.GetBool("recur_from_done") && !calendarRule(rule) {
		anchor = now
	}
	next := Next(rule, anchor, now)
```

with:

```go
	anchor := rec.GetDateTime("due").Time().In(now.Location())
	after := now
	// From-done anchoring is an interval concept; calendar-pattern rules
	// keep their day-and-hour pattern even on records that predate the
	// Create-time validation.
	if rec.GetBool("recur_from_done") && !calendarRule(rule) {
		anchor = now // interval-from-completion: next is one step from completion
	} else if anchor.After(now) {
		// Early completion: the owner satisfied the current (future) occurrence.
		// Advance strictly past it so the just-finished slot is not re-scheduled
		// (and, with nudged_at cleared below, never re-nudged) at its old due.
		after = anchor
	}
	next := Next(rule, anchor, after)
```

Notes for the executor:

- The `else if` is load-bearing: because the `recur_from_done` branch reassigns
  `anchor = now`, the future-due check must only run on the *other* branch,
  where `anchor` still holds the original due. Keep it as `else if`, not a
  second independent `if`.
- Everything after this block (`props["due"] = fmtTime(next.UTC())`,
  `delete(props, "nudged_at")`, the transaction, audit) is unchanged.

**Verify**:
- `gofmt -l internal/tasks/tasks.go` → prints nothing.
- `CGO_ENABLED=0 go build ./...` → exit 0.
- `go test ./internal/tasks/` → still `ok` (all existing tests, including
  `TestDoneRecurringFixedSchedule`, `TestDoneCalendarRuleKeepsPattern`,
  `TestDoneRecurFromDone`, must stay green — the existing tests already cover
  overdue and from-done regimes).

### Step 2: Add the early-completion regression tests

In `internal/tasks/tasks_test.go`, add a new test function modeled on
`TestDoneRecurringFixedSchedule` (lines 159–207) and
`TestDoneCalendarRuleKeepsPattern` (lines 96–128). Use a PocketBase app via
`storetest.NewApp(t)`, fixed `time.Date(...)` values, an explicit `now` passed
into `Done`, and the existing `hydrate`/`dehydrate`/`fmtTime`/`nodes_Props`
helpers. Cover these cases (all four must assert the resulting `res.NextDue`,
in `time.Local`, with `.After(now)` and the right wall clock/day):

1. **Daily, early same-day completion** — due today 09:00, `now` = today 08:00.
   Expect `NextDue` = tomorrow 09:00 (advances by one day; not the same-day
   09:00 slot). This is the bug the plan fixes; without Step 1 it would equal
   today 09:00 and fail.
2. **Weekly, early completion before the weekday** — a `weekly:wed` rule due on
   a Wednesday 18:00, `now` = that same Wednesday 15:00. Expect `NextDue` =
   the *following* Wednesday 18:00 (advances a full week), not the same-day
   18:00. Construct the due with a fixed date that is genuinely a Wednesday and
   set `now` to the same calendar day a few hours earlier. (Pick a real
   Wednesday, e.g. `2026-07-01` is a Wednesday — verify the weekday in your
   chosen date; the test asserts `NextDue.Weekday() == time.Wednesday` and a
   date 7 days after the due.)
3. **Overdue completion still skips forward (regression guard)** — daily due
   yesterday 09:00, `now` = today 10:00. Expect `NextDue` = tomorrow 09:00
   (unchanged behavior: `Next` skips past `now`). This guards that Step 1 did
   not regress the overdue path. (`TestDoneRecurringFixedSchedule` already
   asserts a yesterday-due daily lands after `now` at 09:00 — you may rely on
   it and keep this case as an explicit same-function assertion for clarity, or
   confirm the existing test still passes and omit a duplicate. Prefer adding
   the explicit case so the four regimes sit together.)
4. **`recur_from_done` interval unchanged** — `every:3d` with
   `RecurFromDone: true`, due 48h ago, `now` = now-ish fixed instant. Expect
   `NextDue` = `now.AddDate(0, 0, 3)`. (`TestDoneRecurFromDone` already covers
   this; you may rely on it rather than duplicating — but if you add a case,
   match its exact expectation.)

Constraints to honor so the tests are deterministic and correct:

- For the **weekly** case, remember `Create` snaps a calendar due forward to
  the next matching weekday via `normalizeRecur` (it calls `Next(rule, due,
  due)` when the due misses the pattern — see `internal/tasks/tasks.go:78-85`).
  So pass a `Due` that ALREADY lands on the rule's weekday at the intended wall
  clock; then the stored due is exactly what you set. Reload and `hydrate`, or
  read `res.NextDue` directly from `Done`'s return.
- Assert in `time.Local` (the file's convention — every existing case converts
  with `.In(time.Local)`), and assert `res.NextDue.After(now)` for every case.
- Do NOT introduce a fake clock package or `time.Sleep`; pass `now` explicitly,
  as the existing tests do.
- Reuse `nodes_Props`, `fmtTime`, `dehydrate`, `hydrate` from the file; do not
  redefine them.

**Verify**:
- `go test ./internal/tasks/ -run TestDone` → all `Done`-family tests pass,
  including the new early-completion function.
- `go test ./internal/tasks/` → `ok ... internal/tasks` (full package).
- `gofmt -l internal/tasks/tasks_test.go` → prints nothing.

### Step 3: Full-suite green and hygiene

**Verify**:
- `go vet ./...` → exit 0.
- `go test ./...` → all packages pass.
- `gofmt -l .` → prints nothing.
- `git diff --check` → no whitespace errors.
- `git status --porcelain` → only `internal/tasks/tasks.go` and
  `internal/tasks/tasks_test.go` modified (no other files).

## Test plan

- New test function in `internal/tasks/tasks_test.go`, modeled structurally on
  `TestDoneRecurringFixedSchedule` (lines 159–207). Cases:
  - daily early same-day completion → due advances to next day (the fix);
  - weekly early completion before the weekday → advances a full week;
  - overdue daily completion → still skips forward past `now` (regression
    guard);
  - `recur_from_done` interval → next = completion + interval (unchanged;
    covered by the existing `TestDoneRecurFromDone`, optionally re-asserted).
- Reuse the existing helpers (`storetest.NewApp`, `hydrate`, `dehydrate`,
  `fmtTime`, `nodes_Props`); no new helpers, no assertion framework, no
  `time.Sleep`.
- Verification: `go test ./internal/tasks/` → all pass, including the new
  early-completion case(s) and every pre-existing recurring-task test
  (`TestDoneRecurringFixedSchedule`, `TestDoneCalendarRuleKeepsPattern`,
  `TestDoneRecurFromDone`, `TestSnoozeAndDrop`, `TestUpdate`, …).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go test ./internal/tasks/` passes, including a new early-completion test
      asserting a daily task completed before its same-day due advances to the
      next day, and a weekly task completed before its weekday advances a full
      week.
- [ ] All pre-existing recurring-task tests still pass
      (`TestDoneRecurringFixedSchedule`, `TestDoneCalendarRuleKeepsPattern`,
      `TestDoneRecurFromDone`).
- [ ] `go test ./...` passes (full suite).
- [ ] `go vet ./...` exits 0.
- [ ] `gofmt -l .` prints nothing; `git diff --check` is clean.
- [ ] Only `internal/tasks/tasks.go` (the `Done` block) and
      `internal/tasks/tasks_test.go` are modified — `git status --porcelain`
      shows no other files.
- [ ] `plans/README.md` status row for plan 181 updated (unless a reviewer
      told you they maintain the index).

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows `internal/tasks/tasks.go` or
  `internal/tasks/tasks_test.go` changed since `12a48bf`, and the "Current
  state" excerpts no longer match the live code at the cited lines.
- `calendarRule` no longer exists in `internal/tasks/recur.go`, or `Next`'s
  signature is no longer `Next(r Rule, due, after time.Time) time.Time` with
  "first occurrence strictly after `after`" semantics — the fix depends on
  both. Re-read `internal/tasks/recur.go` and adapt before editing.
- After the fix, `TestDoneRecurFromDone` or `TestDoneCalendarRuleKeepsPattern`
  starts failing — that means the `else if` ordering is wrong (the future-due
  branch is stealing the `recur_from_done` path). Re-check that `anchor = now`
  is the `if` branch and `after = anchor` is the `else if`.
- A verification fails twice after a reasonable fix attempt.
- The fix appears to require editing any file outside the in-scope list.

## Maintenance notes

For the human/agent who owns this after it lands:

- The fix lives entirely in the `Done` caller; `Next`'s "strictly after
  `after`" contract is unchanged. If anyone later changes `Next` to be
  inclusive (first occurrence *at or after* `after`), revisit this block — the
  `after = anchor` bump would then schedule the satisfied occurrence again.
- The reviewer should confirm: (1) the `else if` (not a second `if`) so the
  `recur_from_done` interval path keeps `after = now`; (2) `anchor.After(now)`
  is compared against the *original* due (it is, because the comparison runs in
  the `else` branch before any reassignment); (3) `nudged_at` is still cleared
  unconditionally — the fix is about the new `due` being a genuinely future,
  not-yet-satisfied slot, so clearing the fired-state is still correct.
- Deferred / not in this plan: the nudger and web gateway are untouched; this
  plan does not add a "completed early" UI affordance, only stops the spurious
  re-nudge.
