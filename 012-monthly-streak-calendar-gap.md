# Plan 012: Make monthly habit streaks calendar-aware instead of 31-day approximated

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on.
> If anything in "STOP conditions" occurs, stop and report. When done,
> update this plan's row in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat c4fce47..HEAD -- internal/tasks/streak.go internal/tasks/streak_test.go internal/tasks/recur.go`
> On drift, re-verify the excerpts below.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW (display-only value; recurrence scheduling untouched)
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `c4fce47`, 2026-06-12
- **Issue**: https://github.com/alexradunet/balaur/issues/27

## Why this matters

Streaks are derived with a fixed "period days" per rule kind
(`internal/tasks/streak.go:17-30`), and `monthly` is hardcoded to 31. Two
wrong behaviors follow, both verified arithmetically against the code:

- **False continuity across a skipped month**: `monthly:31`, completed
  Mar 31, next completed May 1 — the April occurrence (Apr 30, via clamp)
  was missed entirely, but `daysBetween = 31` is not `> 31`, so the streak
  reads unbroken.
- **Premature lapse on long gaps that are still within the rule**: rules
  whose next occurrence is more than 31 days away can never exist for
  monthly (max gap 31), but completions late in a 28-day February window
  produce gaps the 31-day constant mis-scores relative to the actual next
  due date.

The repo already has the correct primitive: `monthlyOn` in
`internal/tasks/recur.go:151-153` places a day-of-month with clamping. The
fix replaces the constant with "days until the rule's next calendar
occurrence after the anchor completion" — for monthly only. Daily,
`every:N`, and weekly semantics are documented and tested as-is
(streak.go's header: "a habit whose last completion is more than one period
old has lapsed") and MUST NOT change.

## Current state

- `internal/tasks/streak.go:17-30` — the constant table:

```go
func periodDays(r Rule) int {
	switch r.Kind {
	case "daily":
		return 1
	case "every":
		return r.N
	case "weekly":
		return 7
	case "monthly":
		return 31
	}
	return 0
}
```

- `streak.go:64-80` — both uses of the period: the lapse check
  (`daysBetween(days[len(days)-1], today) > period`) and the consecutive
  gap check (`daysBetween(days[i], days[i+1]) > period`).
- `internal/tasks/recur.go:151-153`:

```go
// monthlyOn places day-of-month `day` in t's month at t's wall-clock time,
```

  (read the full function plus `Next` at recur.go:84-128 before coding —
  `Next`'s `monthly` branch shows the intended advance-and-clamp pattern.)
- Test conventions: `internal/tasks/streak_test.go` — pinned dates via the
  local `day(y, m, d)` helper (noon-anchored, `time.Local`), table-style
  assertions with explanatory messages. Extend THIS file.
- `daysBetween` (streak.go:38-41) — noon-anchored calendar-day difference;
  keep using it.

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Focused | `go test ./internal/tasks/ -run TestStreak -v` | all streak tests pass |
| Gates | `gofmt -l .` / `go vet ./...` / `go test ./...` | clean / 0 / ok |

Sandbox note: TLS failures → `docs/hyperagent-sandbox.md`.

## Scope

**In scope**:
- `internal/tasks/streak.go`
- `internal/tasks/streak_test.go`

**Out of scope** (do NOT touch):
- `internal/tasks/recur.go` — `Next`, `Matches`, `monthlyOn` are scheduling
  logic with their own tests; you only CALL `monthlyOn`.
- Weekly/daily/every semantics (see above).
- Where streaks are displayed (`briefing.go` dayLine, `web/life.go`) — the
  function signature must not change.

## Git workflow

- Branch: `advisor/012-monthly-streak-calendar`
- Commit style: `fix(tasks): calendar-aware lapse gap for monthly streaks`. No push/PR unless instructed.

## Steps

### Step 1: Replace the constant with an anchored gap function

In `streak.go`, change the period machinery to compute the allowed gap FROM
a given anchor completion day:

```go
// allowedGapDays is the rule's maximum calendar-day gap from an anchor
// completion to the next one before the streak breaks: the distance to the
// rule's next occurrence after the anchor. Fixed-length kinds keep their
// constants; monthly is calendar-aware (Feb≠July, clamped day-of-month).
func allowedGapDays(r Rule, anchor time.Time) int {
	if r.Kind == "monthly" {
		next := monthlyOn(anchor.AddDate(0, 1, 0), r.MonthDay)
		return daysBetween(anchor, next)
	}
	return periodDays(r)
}
```

(Confirm `monthlyOn(anchor.AddDate(0,1,0), r.MonthDay)` lands on the NEXT
month's occurrence including clamping — check against `Next`'s monthly
branch; if `AddDate` month-overflow (Jan 31 + 1 month = Mar 3) breaks the
anchor month, normalize to the first of the next month before calling
`monthlyOn`, e.g. `time.Date(anchor.Year(), anchor.Month()+1, 1, 12, 0, 0, 0, anchor.Location())`.
Pick whichever produces Apr 30 for anchor Mar 31 / MonthDay 31 — the unit
test in Step 2 decides.)

Update `Streak` to use it at both gap sites:

```go
	if daysBetween(days[len(days)-1], today) > allowedGapDays(r, days[len(days)-1]) {
		return 0
	}
	...
		if daysBetween(days[i], days[i+1]) > allowedGapDays(r, days[i]) {
			break
		}
```

Keep `periodDays` for the fixed kinds (it is still the `== 0` no-rule
guard); preserve the `period == 0 → 0` early-out using
`periodDays(r) == 0` so kind validation is unchanged.

**Verify**: `go vet ./internal/tasks/` → 0.

### Step 2: Tests that pin the corrected semantics

Add `TestStreakMonthlyCalendarAware` to `streak_test.go` using the existing
`day()` helper:

| Case | Rule | Completions | Today | Expect |
|---|---|---|---|---|
| month-end clamp survives | monthly:31 | Jan 31, Feb 28, Mar 31 (2026) | Mar 31 | 3 |
| skipped April lapses | monthly:31 | Mar 31 | May 1 | 0 |
| alive through next due | monthly:15 | Jan 15 | Feb 15 | 1 |
| lapses day after next due | monthly:15 | Jan 15 | Feb 16 | 0 |
| consecutive across short month | monthly:15 | Jan 15, Feb 15, Mar 15 | Mar 15 | 3 |

Work each expected value out against `allowedGapDays` BY HAND in the test
comments (e.g. anchor Jan 31 → next Feb 28 → allowed 28; gap Jan 31→Feb 28
is 28 → alive). Also re-run the existing `TestStreakDaily` /
weekly / every tests UNCHANGED — they must pass without edits (proof the
fixed kinds kept their semantics).

**Verify**: `go test ./internal/tasks/ -run TestStreak -v` → all pass,
including every pre-existing case untouched.

### Step 3: Full gates

**Verify**: `gofmt -l .` empty; `go test ./...` ok.

## Test plan

Step 2's table IS the plan; model formatting on the existing
`TestStreakDaily` (streak_test.go:12-27). The hand-derived comments are
mandatory — they are what makes the next reader trust the clamp math.

## Done criteria

- [ ] `grep -n "case \"monthly\":" internal/tasks/streak.go` shows the 31 constant is no longer the monthly lapse source (`allowedGapDays` present)
- [ ] `go test ./internal/tasks/ -run TestStreak -v` → old cases unchanged + ≥ 5 new monthly cases pass
- [ ] `go test ./...` exit 0; `gofmt -l .` empty
- [ ] Diff confined to `streak.go` + `streak_test.go` (plus `plans/README.md`)
- [ ] `plans/README.md` status row updated

## STOP conditions

- Any EXISTING streak test needs editing to pass — that means you changed
  daily/weekly/every semantics; revert and re-read Step 1.
- `monthlyOn` clamping disagrees with both anchor-normalization options in
  Step 1 (neither yields Apr 30 for Mar 31 + monthly:31) — report with the
  actual outputs; the recur.go primitive may have different semantics than
  planned.

## Maintenance notes

- Weekly multi-day rules (`weekly:mon,thu`) keep the documented 7-day slack
  — a future refinement could anchor them on `Next` the same way, but it is
  a SEMANTIC change to a documented behavior; needs an owner decision, not
  a drive-by.
- `StreakFor` performance (one query per task per render) is untouched
  here; if the perf cluster ever batches `CompletionDays`, `Streak`'s pure
  function shape (rule + days + today) makes that free.
