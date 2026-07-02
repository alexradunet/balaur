# Plan 249: De-flake TestNudgeCatchesUpAfterDowntime across DST transitions (fixed clock, duration-based seeding)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 077318a..HEAD -- internal/tasks/nudge_test.go`
> If the file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tests
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

`TestNudgeCatchesUpAfterDowntime` anchors on the real `time.Now()`, seeds the
task's due time with **calendar** math (`now.AddDate(0, 0, -3)`), and asserts
the rendered lateness contains `"overdue 3d"`. But `Lateness` computes the day
count from the **absolute** duration (`int(now.Sub(due).Hours()/24)`). Across a
spring-forward DST transition the 3-calendar-day span is only 71 hours, so
`Lateness` renders `"overdue 2d"` and the test fails — for roughly three days
after every spring transition, in any DST zone. The box this repo's merge gate
runs on is in Europe/Bucharest (a DST zone), so the local `go test ./...` gate
goes red while a UTC CI run stays green. This plan pins the test (and the other
wall-clock anchors in the same file) to a fixed instant with duration-based
seeding, making `internal/tasks` deterministic regardless of when or where the
suite runs. **Zero production code changes** — `Lateness`'s duration semantics
are correct product behavior; only the test's seeding was calendar-based.

## Current state

Relevant files:

- `internal/tasks/nudge_test.go` — the nudge cron's test file; every test
  anchors on the real clock (`now := time.Now()` at lines 77, 110, 132, 149,
  165, 197, 241). Only the one at line 241 feeds a day-count assertion.
- `internal/tasks/nudge.go` — production code, **read-only for this plan**;
  contains `DueForNudge`, `Nudge`, `Lateness`.
- `internal/tasks/briefing_test.go` — sibling test file, **read-only**; already
  uses the fixed-clock pattern this plan mirrors.

The flaky test, verbatim (`internal/tasks/nudge_test.go:239-257`):

```go
func TestNudgeCatchesUpAfterDowntime(t *testing.T) {
	app := storetest.NewApp(t)
	now := time.Now()

	// Came due while the box slept three days — first tick picks it up once.
	if _, err := Create(app, CreateOpts{Title: "Renew ID", Due: now.AddDate(0, 0, -3)}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := Nudge(app, nil, now); err != nil {
		t.Fatalf("nudge: %v", err)
	}
	msgs := nudgeMessages(t, app)
	if len(msgs) != 1 {
		t.Fatalf("catch-up: %d messages, want 1", len(msgs))
	}
	if c := msgs[0].GetString("content"); !strings.Contains(c, "overdue 3d") {
		t.Errorf("lateness missing from catch-up nudge: %q", c)
	}
}
```

The production function whose semantics the assertion depends on
(`internal/tasks/nudge.go:148-161`) — duration-based, NOT calendar-based:

```go
// Lateness renders how a due time stands relative to now, in human terms.
func Lateness(due, now time.Time) string {
	due = due.In(now.Location())
	switch d := now.Sub(due); {
	case d < 2*time.Minute:
		return "due now"
	case d < time.Hour:
		return fmt.Sprintf("overdue %dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("overdue %dh", int(d.Hours()))
	default:
		return fmt.Sprintf("overdue %dd", int(d.Hours()/24))
	}
}
```

The mismatch: `now.AddDate(0, 0, -3)` spans 71h across spring-forward (73h
across fall-back), while `Lateness` divides the absolute duration by 24h. 71h
→ `int(71/24)` = 2 → `"overdue 2d"` → the `"overdue 3d"` assertion fails.
(Fall-back's 73h still yields 3, so only spring-forward breaks it.)

The fixed-clock pattern already established in this package
(`internal/tasks/briefing_test.go:46-51`):

```go
// at returns a fixed Wednesday (2026-06-24) at the given local hour — briefing
// tests anchor to this date so clock-sensitive assertions are deterministic
// regardless of when the suite runs.
func at(hour int) time.Time {
	return time.Date(2026, 6, 24, hour, 0, 0, 0, time.Local)
}
```

Why a fixed anchor is safe for the nudge tests (verified at plan time — the
executor need not re-derive this, only STOP if the code has drifted):

- `DueForNudge` (`internal/tasks/nudge.go:50-80`) uses only the `now` it is
  passed: `due.Before(now)`, `r.GetString("nudged_at") != ""` (a string
  presence check), and `suTime.After(now)`. It never reads the record's
  autodated `created` field or calls `time.Now()`.
- `Nudge` (`internal/tasks/nudge.go:88-134`) stamps
  `props["nudged_at"] = store.PBTime(now.UTC())` from the passed `now`.
- `Create` (`internal/tasks/tasks.go:30-60`) does no wall-clock validation for
  the non-recurring tasks these tests create (blank `Recur` passes `Due`
  through `normalizeRecur` untouched).
- `Snooze` (`internal/tasks/tasks.go:255-270`) stores `until` verbatim; no
  comparison against the real clock.
- `nudgeMessages` (`internal/tasks/nudge_test.go:55-62`) filters only on
  `origin = 'nudge'`, never on `created`.
- `grep -n "time.Now" internal/tasks/nudge.go` → no matches (production nudge
  code has no hidden clock).

The seven wall-clock anchors in `nudge_test.go`, and what each test asserts
(none besides line 241 has a day-count-sensitive assertion — all their other
time arithmetic is already `Add(<duration>)`, i.e. absolute):

| Line | Test | Assertion character |
|------|------|---------------------|
| 77   | `TestNudgeFiresOnceAndMarks` | message count, `nudged_at` non-zero, idempotency at `now+1m` |
| 110  | `TestNudgeBatchesIntoOneMessage` | one message contains both titles |
| 132  | `TestNudgeUsesComposedText` | message equals the faked model text |
| 149  | `TestNudgeFallsBackWhenModelFails` | fallback contains "Reminder" |
| 165  | `TestNudgeRespectsFutureAndSnooze` | zero fire, then fire at `now+61m` |
| 197  | `TestNudgeIdempotencyTwoTasks` | counts + `DueForNudge` empty at `now+1m` |
| 241  | `TestNudgeCatchesUpAfterDowntime` | **`"overdue 3d"` — the DST-sensitive one** |

Repo conventions that apply here (from `AGENTS.md`):

- "Tests use the standard `testing` package, table-driven where it helps
  readability" — no assertion frameworks. PocketBase-dependent tests use
  `storetest.NewApp(t)` (already in use throughout this file); the fake model
  is `internal/llmtest` (already in use at lines 138, 154) — never a real model.
- "Comments explain non-obvious intent, trade-offs, or constraints — never
  narrate what the code already says." The new helper's comment must say WHY
  the clock is fixed (DST), not what `time.Date` does.
- gofmt is enforced (`gofmt -l .` must print nothing).
- Surgical changes: touch only what the fix requires; do not "improve"
  adjacent tests' style.

Tour note: no file in `.tours/` anchors `internal/tasks/nudge_test.go`
(verified at plan time with `grep -rln "nudge_test" .tours/` → no matches), so
line shifts in this file cannot break a tour and `TestTours` is not required.

Self-knowledge note: `internal/self/knowledge.md` does **NOT** need updating —
this is a test-only determinism fix; no user-visible architecture or
capability changes.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Targeted tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/tasks/ -run TestNudge -count=1` | `ok`, exit 0 |
| Whole tasks package | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/tasks/ -count=1` | `ok`, exit 0 |
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all `ok` |
| Vet | `go vet ./...` | exit 0, no output |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |

(The host `/tmp` is a small tmpfs; the Go linker OOMs there — always set
`TMPDIR=$HOME/.cache/go-tmp` on `go test` invocations as shown.)

## Suggested executor toolkit

- If the `go-standards` skill is available, invoke it before Step 1 — it
  carries this repo's testing idioms (no `time.Sleep` synchronization, fake
  `llm.Client`, temp-dir apps).

## Scope

**In scope** (the only files you should modify):

- `internal/tasks/nudge_test.go`
- `plans/README.md` (status row only, at completion)

**Out of scope** (do NOT touch, even though they look related):

- `internal/tasks/nudge.go` — `Lateness`'s absolute-duration semantics are
  correct product behavior ("overdue 3d" means ≥72 real hours late); this plan
  must produce a **zero production diff**.
- `internal/tasks/briefing_test.go` — already deterministic via its `at()`
  helper; leave it and its helper comment alone.
- `internal/tasks/tasks_test.go`, `internal/tasks/streak_test.go` — they also
  anchor on `time.Now()`, but no assertion there is day-count-sensitive;
  sweeping them is deferred (see Maintenance notes).
- `internal/self/knowledge.md`, `.tours/` — not affected (see Current state).

## Git workflow

- The executor runs in an isolated git worktree branched from `origin/main`;
  branch name: `advisor/249-nudge-test-dst-deflake`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/`test`/`chore`);
  one commit per logical unit. Suggested single commit:
  `test(tasks): de-flake nudge tests across DST — fixed clock, duration seeding`.
- Stage with explicit pathspecs only (`git add internal/tasks/nudge_test.go`)
  — the main checkout is shared by parallel agents; never `git add -A`.
- **NEVER push.** The reviewer merges.

## Steps

### Step 1: Add a fixed-clock helper and fix the flaky test

In `internal/tasks/nudge_test.go`, add one helper near the top of the file
(after the imports, before `TestDueLine` is fine):

```go
// nudgeNow is the fixed anchor for the nudge tests, mirroring briefing_test's
// at(): anchoring on the real time.Now() made the "overdue 3d" assertion
// DST-sensitive — Lateness counts absolute 24h blocks, but calendar-day
// seeding (AddDate) spans only 71h across spring-forward. Nothing in the
// nudge path compares against the wall clock, so a fixed instant is safe.
func nudgeNow() time.Time {
	return time.Date(2026, 6, 24, 10, 0, 0, 0, time.UTC)
}
```

Then rewrite `TestNudgeCatchesUpAfterDowntime` (currently lines 239-257) to
use BOTH defenses — the fixed anchor AND duration-based seeding:

- `now := time.Now()` → `now := nudgeNow()`
- `Due: now.AddDate(0, 0, -3)` → `Due: now.Add(-72 * time.Hour)`, and update
  the comment above the `Create` call so it stays true, e.g.:
  `// Came due 72h (3 absolute days, matching Lateness' duration math) before`
  `// the first tick after downtime — picked up once.`
- Keep the `"overdue 3d"` assertion and everything else in the test unchanged
  (72h ≥ 24h → `Lateness` default branch → `int(72/24)` = 3, in every zone).

**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/tasks/ -run TestNudgeCatchesUpAfterDowntime -count=1 -v`
→ `--- PASS: TestNudgeCatchesUpAfterDowntime`, `ok`, exit 0.

### Step 2: Sweep the remaining six wall-clock anchors to the fixed clock

In the same file, replace `now := time.Now()` with `now := nudgeNow()` in each
of the six other tests (pre-plan lines 77, 110, 132, 149, 165, 197 — line
numbers will have shifted by the helper added in Step 1; match on the
`now := time.Now()` text inside each named test):

- `TestNudgeFiresOnceAndMarks`
- `TestNudgeBatchesIntoOneMessage`
- `TestNudgeUsesComposedText`
- `TestNudgeFallsBackWhenModelFails`
- `TestNudgeRespectsFutureAndSnooze`
- `TestNudgeIdempotencyTwoTasks`

This is a mechanical substitution: every other time expression in these tests
is already `now.Add(<duration>)` (absolute arithmetic), and no assertion in
them depends on the wall clock (see the table in Current state). Change
NOTHING else in these tests — no assertion, comment, or structure edits.

Escape hatch (expected NOT to trigger — see Current state's safety analysis):
if one of these six tests fails after the substitution because its logic
genuinely depends on the real clock (e.g. a comparison against a
PocketBase-autodated `created` field), revert that ONE test to
`now := time.Now()` and add a single comment line above it:
`// real clock kept: <reason>; DST-safe — no day-count assertion.`
Then note it in your completion report. If the **Step 1 test** cannot hold a
fixed clock, that is a STOP condition instead.

**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/tasks/ -run TestNudge -count=1`
→ `ok`, exit 0.

**Verify**: `grep -c "time.Now()" internal/tasks/nudge_test.go` → `0`
(or exactly the count of escape-hatch reverts, each carrying its comment).

### Step 3: Prove DST/zone safety with a TZ matrix

Run the whole `internal/tasks` package under three timezones (the `TZ` env
var re-points `time.Local` for the process):

**Verify** (all three commands → `ok`, exit 0):

```
TZ=UTC              TMPDIR=$HOME/.cache/go-tmp go test ./internal/tasks/ -count=1
TZ=Europe/Bucharest TMPDIR=$HOME/.cache/go-tmp go test ./internal/tasks/ -count=1
TZ=Pacific/Auckland TMPDIR=$HOME/.cache/go-tmp go test ./internal/tasks/ -count=1
```

### Step 4: Run the full gates

**Verify** (each → expected result from the Commands table):

1. `gofmt -l .` → empty output
2. `go vet ./...` → exit 0
3. `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output, exit 0
4. `CGO_ENABLED=0 go build ./...` → exit 0
5. `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0

Then commit (see Git workflow) with explicit pathspecs.

## Test plan

This plan IS a test change — no new test functions are added; the existing
seven nudge tests are made deterministic. Cases covered after the change:

- **The regression this plan fixes**: `TestNudgeCatchesUpAfterDowntime` now
  passes on every calendar date in every timezone, including the ~3 days after
  a spring-forward transition (Step 3's TZ matrix is the proof; the fixed
  anchor makes it independent of the run date too).
- **No behavior loss**: the same assertions still hold — catch-up fires once,
  content contains `"overdue 3d"`, batching/idempotency/snooze/fallback tests
  unchanged in what they check.
- Structural pattern: model the helper on `at()` in
  `internal/tasks/briefing_test.go:46-51` (quoted in Current state).

Verification: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/tasks/ -run TestNudge -count=1`
→ all 7 `TestNudge*` tests pass (add `-v` to count them).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./internal/tasks/ -run TestNudge -count=1` → exit 0
- [ ] TZ matrix (Step 3): all three `TZ=...` package runs → exit 0
- [ ] `grep -n "AddDate" internal/tasks/nudge_test.go` → no matches
- [ ] `grep -c "time.Now()" internal/tasks/nudge_test.go` → `0` (or each
      remaining occurrence sits directly under an escape-hatch comment per
      Step 2, reported at completion)
- [ ] **Zero production diff**: `git diff origin/main --name-only` lists only
      `internal/tasks/nudge_test.go` (plus `plans/README.md` if you maintain
      the index) — in particular NOT `internal/tasks/nudge.go`
- [ ] `gofmt -l .` → empty; `go vet ./...` → exit 0; staticcheck → exit 0;
      `CGO_ENABLED=0 go build ./...` → exit 0
- [ ] Full gate: `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0
- [ ] No files outside the in-scope list are modified (`git status --porcelain`)
- [ ] `plans/README.md` status row for 249 updated (unless the reviewer said
      they maintain the index)

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows `internal/tasks/nudge_test.go` changed since
  `077318a` AND the excerpts in "Current state" no longer match the live code
  (in particular: if `TestNudgeCatchesUpAfterDowntime` no longer seeds with
  `AddDate` or no longer asserts `"overdue 3d"`, the flake may already be
  fixed — report, don't re-fix).
- `TestNudgeCatchesUpAfterDowntime` fails with the fixed anchor from Step 1 —
  that means the nudge path DOES compare against the real wall clock somewhere,
  contradicting this plan's verified safety analysis; the fix would then be a
  different plan.
- The TZ matrix (Step 3) fails in some zone after Steps 1–2 — that indicates a
  real zone-handling bug in production code (`nudge.go` or below), which is
  out of scope here.
- The fix appears to require touching `internal/tasks/nudge.go`,
  `internal/tasks/briefing_test.go`, or any other out-of-scope file.
- More than one of the six swept tests needs the Step 2 escape hatch — a
  pattern of hidden wall-clock coupling means the safety analysis is stale.
- A step's verification fails twice after a reasonable fix attempt.

## Maintenance notes

- **Deferred sweep**: `internal/tasks/tasks_test.go` (14 `time.Now()` uses)
  and `internal/tasks/streak_test.go` (1) still anchor on the real clock.
  None of their assertions is day-count-sensitive today (they assert relative
  behavior, not rendered "overdue Nd" strings), so they were deliberately left
  out (surgical-change rule). If a test there ever asserts `Lateness`/`DueLine`
  output, anchor it on `nudgeNow()`-style fixed time with `Add(<duration>)`
  seeding.
- **For the reviewer**: the key thing to scrutinize is that the diff touches
  ONLY `nudge_test.go` (zero production diff) and that no assertion changed —
  every `strings.Contains`/count check must be byte-identical to before, except
  the comment above the `Create` call in `TestNudgeCatchesUpAfterDowntime`.
- **Interaction**: the package now has two fixed-clock helpers (`at()` in
  briefing_test.go, `nudgeNow()` in nudge_test.go). That duplication is one
  `time.Date` line and keeps each file self-contained; if the package's test
  clocks are ever consolidated, fold both into one shared `_test.go` helper —
  a cosmetic follow-up, not required.
- **Future rule of thumb** (why this class of flake happens): never combine
  calendar seeding (`AddDate`) with duration-based assertions (`Lateness`,
  `time.Sub`) around a real-clock anchor — pick absolute durations on both
  sides, or a fixed instant, or both (as here).
