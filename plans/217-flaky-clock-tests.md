# Plan 217: Anchor two wall-clock-fragile tests to a fixed clock (`TestSeedRangeContent`, the briefing `at()` tests)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command (always with `-count=1` so the test cache can't mask a
> date-dependent result) and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report. When done, update the status row for
> this plan in `plans/README.md` — unless a reviewer dispatched you and told you
> they maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat ef9f2df..HEAD -- internal/seed/seed_test.go internal/seed/seed.go internal/tasks/briefing_test.go`
> On any change, compare the "Current state" excerpts to the live code; mismatch = STOP.

## Status

- **Priority**: P2
- **Effort**: S (seed) + M (briefing)
- **Risk**: LOW (test-only)
- **Depends on**: none
- **Category**: tests
- **Planned at**: commit `ef9f2df`, 2026-06-30

## Why this matters

Two existing tests seed/assert against the **real wall clock**, so they pass most
days and fail on specific calendar boundaries — the exact defect family that
already bit this repo: `TestSeedSummariesPopulatesAllBandLevels` failed every
Monday until plan 211 added the `runAt(app, now)` seam and anchored it to a fixed
Wednesday. A test that's red 1 day in 7 (or once a month) erodes the green-gate
discipline the whole `/improve` loop relies on. This plan removes the last two
known clock-fragile tests by anchoring each to a single fixed clock.

The repo convention is already established: date-dependent tests use a fixed
`time.Date(2026, ...)` and (for seed) the `runAt(app, fixedNow)` seam; for
PocketBase `created` autodate, tests override it with
`SetRaw("created", t.UTC().Format("2006-01-02 15:04:05.000Z"))`.

## Current state

### A — `internal/seed/seed_test.go`: `TestSeedRangeContent` uses the real clock twice

```go
// internal/seed/seed_test.go (current)
func TestSeedRangeContent(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := Run(app); err != nil {          // Run() reads real time.Now() (seed.go:71)
		t.Fatalf("Run: %v", err)
	}

	now := time.Now()                            // SECOND independent real-clock read
	month := recap.Month(now.AddDate(0, -8, 0))  // month 8 back from "now"
	data, err := life.Range(app, month.Start, month.End)
	if err != nil {
		t.Fatalf("life.Range: %v", err)
	}
	if len(data.Done) == 0 { t.Errorf("...Done = 0, want ≥1") }
	if len(data.Logged) == 0 { t.Errorf("...Logged = 0, want ≥1") }
}
```

Two failure modes: (a) the two `time.Now()` reads can straddle a day/month
boundary, desyncing the seed placement from the asserted month; (b) the seed
places content via day/week offsets from its own `now`, so whether a measure
lands in the *calendar* month exactly 8 back depends on today's day-of-month — on
month-boundary run dates the window can come up empty.

The seam + the already-fixed sibling (the pattern to copy), same file:

```go
// internal/seed/seed.go:70-79
func Run(app core.App) (*Result, error) {
	return runAt(app, time.Now())
}
// runAt is Run with an injectable clock so date-dependent seeding ... is deterministic in tests.
func runAt(app core.App, now time.Time) (*Result, error) { ... }

// internal/seed/seed_test.go — TestSeedSummariesPopulatesAllBandLevels (already fixed, plan 211):
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC) // fixed Wednesday
	if _, err := runAt(app, now); err != nil { ... }
```

`seed_test.go` is `package seed` (it already calls the unexported `runAt`), so the
fix has no import changes.

### B — `internal/tasks/briefing_test.go`: `at()` reads the real clock; persisted `created` is real autodate

```go
// internal/tasks/briefing_test.go:48-51
// at returns today's date at the given local hour — briefing tests pin the
// clock inside the real today so created-timestamp comparisons hold.
func at(hour int) time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, time.Local)
}
```

`at()` is used as the `now` argument to `Briefing(app, client, now, hour)` in
`TestBriefingFiresOncePerDay` (line 53), `TestBriefingHourGateAndCatchUp` (82),
and other briefing tests. `Briefing` persists its message via PocketBase autodate
(real wall clock); `BriefedToday(app, now)` then compares that real `created`
against the `at()`-derived local midnight. If the suite executes across local
midnight, the two disagree and the idempotency assertions flip.

The established technique for controlling `created` is already in this file:

```go
// internal/tasks/briefing_test.go:229 (inside TestBriefedTodayZoneSensitivity)
	rec.SetRaw("created", seedUTC.Format("2006-01-02 15:04:05.000Z"))
```

`briefingMessages(t, app)` (briefing_test.go:37) returns the persisted
`origin='briefing'` message records — use it to reach the row(s) to rewrite.

> NOTE: simply hard-coding `at()` to a fixed date is NOT enough on its own — the
> persisted `created` would still be real-now, so `BriefedToday(fixed-at)` would
> compare a fixed day against a real-now `created` and break **every** run. The
> fix must anchor BOTH the logical `now` (`at()`) AND the persisted `created` to
> the same fixed clock.

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Seed pkg | `TMPDIR=/home/alex/.cache/go-tmp go test ./internal/seed/... -count=1` | PASS |
| Tasks pkg | `TMPDIR=/home/alex/.cache/go-tmp go test ./internal/tasks/... -count=1` | PASS |
| Full | `TMPDIR=/home/alex/.cache/go-tmp go test ./... -count=1` | all pass |
| Vet/fmt | `go vet ./...` ; `gofmt -l internal/seed internal/tasks` | exit 0 ; empty |

(Always `-count=1` — a plain `go test` may serve a stale cached PASS and hide the very date-dependence this plan removes.)

## Scope

**In scope**:
- `internal/seed/seed_test.go` (`TestSeedRangeContent` only)
- `internal/tasks/briefing_test.go` (`at()` + the briefing tests that use it)

**Out of scope** (do NOT touch):
- Production code: `internal/seed/seed.go`, `internal/tasks/briefing.go` — the
  `runAt` seam already exists; do NOT add a clock seam to `Briefing` (use the
  test-side `SetRaw` technique instead). If you conclude the briefing fix
  genuinely requires a production clock seam, STOP and report rather than change
  `briefing.go`.
- `TestSeedSummariesPopulatesAllBandLevels` (already fixed), and briefing tests
  that don't use `at()` for an assertion anchor (e.g. `TestBriefedTodayZoneSensitivity`
  already controls `created` explicitly — leave it).
- Do NOT weaken any assertion to make it pass; the fix is determinism, not laxity.

## Git workflow
- Branch: `advisor/217-flaky-clock-tests`
- Conventional-commit subject, e.g. `test(seed,tasks): anchor clock-fragile tests to a fixed date`
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Fix `TestSeedRangeContent` (seed)

Replace the real-clock reads with the `runAt` seam + a single fixed `now`,
mirroring `TestSeedSummariesPopulatesAllBandLevels`:

```go
func TestSeedRangeContent(t *testing.T) {
	app := storetest.NewApp(t)
	// Fixed Wednesday so seed placement and the asserted month derive from one
	// clock — deterministic regardless of the run date. Mirrors
	// TestSeedSummariesPopulatesAllBandLevels.
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	if _, err := runAt(app, now); err != nil {
		t.Fatalf("runAt: %v", err)
	}
	month := recap.Month(now.AddDate(0, -8, 0))
	data, err := life.Range(app, month.Start, month.End)
	// ... unchanged assertions ...
}
```

**Verify**: `TMPDIR=/home/alex/.cache/go-tmp go test ./internal/seed/ -run TestSeedRangeContent -count=1 -v` → PASS.

> If the test FAILS with `Done = 0` or `Logged = 0` at the fixed `now`, the seed
> does not populate the month 8 back from 2026-06-24 (Oct 2025). STOP and report
> — do NOT weaken the assertion. (A different fixed anchor known to fall inside a
> populated seed span would be the fix, chosen with the seed author's intent in
> mind.)

### Step 2: Anchor the briefing `at()` clock AND the persisted `created`

2a. Change `at()` to a fixed base date (a Wednesday, matching the repo's
convention) instead of `time.Now()`:

```go
// at returns a FIXED test day at the given local hour. Anchored (not time.Now())
// so created-timestamp comparisons are deterministic across local midnight; the
// briefing message's `created` is rewritten to the same clock (see briefingAt).
func at(hour int) time.Time {
	return time.Date(2026, 6, 24, hour, 0, 0, 0, time.Local)
}
```

2b. After each `Briefing(...)` call that is expected to PERSIST a message, rewrite
the persisted briefing message(s) `created` to the same fixed clock so
`BriefedToday` compares like-for-like. Add a small helper in the test file:

```go
// stampBriefingCreated rewrites every persisted briefing message's `created` to
// `when`, overriding PocketBase's real-clock autodate so idempotency comparisons
// are deterministic. Mirrors the SetRaw technique in TestBriefedTodayZoneSensitivity.
func stampBriefingCreated(t *testing.T, app core.App, when time.Time) {
	t.Helper()
	for _, rec := range briefingMessages(t, app) {
		rec.SetRaw("created", when.UTC().Format("2006-01-02 15:04:05.000Z"))
		if err := app.Save(rec); err != nil {
			t.Fatalf("stamping briefing created: %v", err)
		}
	}
}
```

Then in each affected test, call `stampBriefingCreated(t, app, at(<hour>))`
immediately after a `Briefing(...)` that persisted (and before the next
`Briefing`/assertion that depends on the day boundary). Walk
`TestBriefingFiresOncePerDay`, `TestBriefingHourGateAndCatchUp`, and any other
`at()`-using briefing test, and insert the stamp so every persisted message's
`created` lies on the fixed `at()` day for the tick that wrote it. The
"next day" assertions (`at(10).AddDate(0,0,1)`) then correctly see the prior
message as before the new midnight.

**Verify**: `TMPDIR=/home/alex/.cache/go-tmp go test ./internal/tasks/ -run TestBriefing -count=1 -v` → all briefing tests PASS.

> If a briefing test's logic can't be made deterministic with `at()`-anchoring +
> `stampBriefingCreated` alone (e.g. it asserts something that genuinely needs the
> real clock), STOP and report which test and why — do not change `briefing.go`.

### Step 3: Full verification

- `TMPDIR=/home/alex/.cache/go-tmp go test ./internal/seed/... ./internal/tasks/... -count=1` → PASS
- `grep -n "time.Now()" internal/seed/seed_test.go internal/tasks/briefing_test.go` → no match inside `TestSeedRangeContent` or `at()` (other tests may legitimately still use it — see STOP note)
- `go vet ./...` → exit 0 ; `gofmt -l internal/seed internal/tasks` → empty
- `TMPDIR=/home/alex/.cache/go-tmp go test ./... -count=1` → all pass

## Test plan
- No NEW test functions — this hardens two existing ones. Proof is the targeted
  `-count=1` runs passing and `time.Now()` no longer anchoring `TestSeedRangeContent`
  or `at()`.
- Pattern to follow: `TestSeedSummariesPopulatesAllBandLevels` (seed) and
  `TestBriefedTodayZoneSensitivity` (the `SetRaw("created", ...)` technique).

## Done criteria — ALL must hold
- [ ] `at()` in `briefing_test.go` no longer calls `time.Now()`
- [ ] `TestSeedRangeContent` uses `runAt(app, fixedNow)` and derives `month` from that same `fixedNow` (no `time.Now()`)
- [ ] `TMPDIR=/home/alex/.cache/go-tmp go test ./internal/seed/... ./internal/tasks/... -count=1` PASSES
- [ ] `TMPDIR=/home/alex/.cache/go-tmp go test ./... -count=1` exits 0
- [ ] `go vet ./...` exit 0 ; `gofmt -l internal/seed internal/tasks` empty
- [ ] No production (`.go` non-test) file modified — only the two `_test.go` files (`git status`)
- [ ] `plans/README.md` status row updated

## STOP conditions
- `TestSeedRangeContent` fails at the fixed anchor (`Done`/`Logged` = 0) — report; do not weaken the assertion.
- A briefing test needs real-clock behavior that `at()`-anchoring + `stampBriefingCreated` can't provide — report; do NOT add a seam to `briefing.go` in this plan.
- A run with `-count=1` reveals a THIRD clock-fragile test elsewhere — report it (don't expand scope here).

## Maintenance notes
- New date-dependent tests must use a fixed `time.Date(2026,...)` clock (and
  `runAt`/`SetRaw("created",...)` where seeding or autodate is involved) — never
  `time.Now()` for an assertion anchor.
- A reviewer should confirm no assertion was loosened to pass; the diff should be
  pure clock-anchoring + `created` stamping.
- If `Briefing` later gains a production clock seam for other reasons, the
  `stampBriefingCreated` helper can be retired in favor of it.
