# Plan 211: Make the seed band-coverage summaries test deterministic via an injectable-now seam (fixes a Monday-only RED gate)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. Touch
> only the files listed as in scope. If any STOP condition occurs, stop and
> report. Commit your work in the worktree following the git workflow section.
> SKIP updating `plans/README.md` — your reviewer maintains the index. Audit
> every claim in your report against an actual tool result. Use the report format
> at the end.
>
> **Drift check (run first)**:
> `git diff --stat 79c6784..HEAD -- internal/seed/seed.go internal/seed/seed_test.go` — expect EMPTY.

## Status
- **Priority**: P1 (a RED full suite blocks every commit) | **Effort**: S | **Risk**: LOW
- **Depends on**: none
- **Category**: tests (test-quality / flakiness)
- **Planned at**: commit `79c6784`, 2026-06-29

## Why this matters

`internal/seed/TestSeedSummariesPopulatesAllBandLevels` **fails every Monday**, so a
fresh `go test ./...` is RED — which blocks the pre-commit hook and every merge
gate (the whole improve batch is stuck behind it). It was masked earlier by Go's
test cache; a `-count=1` run exposes it.

Root cause: the test calls `seed.Run(app)`, which seeds with the **real
`time.Now()`**. `recap.Bands` builds the "day" band as *days of the current ISO
week before today* — on a **Monday** that set is empty by design (the week just
started), so the seed writes no `day` summary and the test's "≥1 row per
period_type" assertion fails ("no day row found"). It passes Tue–Sun.

The sibling test `TestSeedPeriodsCoversAllBands` (same file) is already
deterministic — it uses a **fixed** `time.Date(2026, 6, 24, …)` (a Wednesday,
where all five bands are populated). This plan makes the summaries test
deterministic the same way: add a `runAt(app, now)` seam to `internal/seed` so
`Run` keeps real `time.Now()` in production, but the test anchors the same fixed
Wednesday.

This is a NEW finding outside the 199–210 modularity batch — it surfaced while
executing 204. Fixing it greens the gate so 204 (and 205+) can land.

## Current state

`internal/seed/seed.go` — `Run` (line 70) computes `now` once and threads it to
its sub-seeders:

```go
func Run(app core.App) (*Result, error) {
	now := time.Now()
	res := &Result{}

	n, err := seedMessages(app, now)
	// ... seedTasks(app, now) ... seedLife(app, now) ... seedSummaries(app, now) ...
	// ... seedWorld(app, now) ...
	return res, nil
}
```

`now` is used at lines 71, 74 (`seedMessages`), 80 (`seedTasks`), 100
(`seedLife`), 105 (`seedSummaries`), 116 (`seedWorld`). `seedSummaries(app, now)`
→ `seedPeriods(now)` (line 500) drives the band/summary generation. (The other
`time.Now()` in this file, at line ~211, is inside **`Reset`** for cleanup — NOT
in scope, leave it.)

The failing test — `internal/seed/seed_test.go:217`:
```go
func TestSeedSummariesPopulatesAllBandLevels(t *testing.T) {
	app := storetest.NewApp(t)
	if _, err := Run(app); err != nil {        // <-- real time.Now()
		t.Fatalf("Run: %v", err)
	}
	wantTypes := []string{"day", "week", "month", "quarter", "year"}
	// ... asserts ≥1 summaries row per type ...
}
```

The deterministic sibling to mirror — `internal/seed/seed_test.go:195`:
```go
func TestSeedPeriodsCoversAllBands(t *testing.T) {
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	periods := seedPeriods(now)
	// ...
}
```

`seed_test.go` is `package seed` (line 1) — it already calls the unexported
`seedPeriods`, so it can call an unexported `runAt`. `time` is already imported.

## Commands you will need

| Purpose   | Command                                              | Expected            |
|-----------|------------------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`                       | exit 0              |
| Vet       | `go vet ./...`                                        | exit 0              |
| Seed test | `go test ./internal/seed/ -count=1`                  | PASS (note `-count=1`!) |
| Full test | `go test ./... -count=1`                             | all pass            |
| gofmt     | `gofmt -l internal/seed`                             | prints nothing      |

> **CRITICAL: always use `-count=1`** when checking — a plain `go test` may serve
> a stale cached PASS and hide the real (date-dependent) result. If `go test ./...`
> hits "No space left on device", set `TMPDIR=/home/alex/.cache/go-tmp`.

## Scope

**In scope**:
- `internal/seed/seed.go` (add the `runAt(app, now)` seam; `Run` becomes a thin wrapper)
- `internal/seed/seed_test.go` (point `TestSeedSummariesPopulatesAllBandLevels` at `runAt` with the fixed Wednesday)

**Out of scope** (do NOT touch):
- `internal/recap` — the day-band-empty-on-Monday behavior is CORRECT production
  behavior; do not change it. The fix is in the TEST's determinism, not recap.
- The `time.Now()` in `Reset` (~line 211) — unrelated cleanup path.
- The other seed tests (`TestRunSeedsAllCollections`, `TestRunIsIdempotent`,
  `TestResetRemovesOnlySeededData`, `TestSeedRangeContent`) — they pass today and
  must keep calling `Run(app)` with the real clock (they don't assert day-band
  presence). Do NOT convert them.

## Git workflow
- Branch: `advisor/211-seed-summaries-test-deterministic`
- Conventional-commit subject, e.g. `test(seed): anchor band-coverage test to a fixed date via runAt seam`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add the `runAt(app, now)` seam

In `internal/seed/seed.go`, refactor `Run` into a thin wrapper over `runAt`:

```go
// Run seeds dummy data using the current time. Safe to call repeatedly.
func Run(app core.App) (*Result, error) {
	return runAt(app, time.Now())
}

// runAt is Run with an injectable clock so date-dependent seeding (recap bands,
// task due dates, measure timestamps) is deterministic in tests.
func runAt(app core.App, now time.Time) (*Result, error) {
	res := &Result{}

	// ... the ENTIRE current body of Run, VERBATIM, EXCEPT delete the
	//     `now := time.Now()` line (now is the parameter) ...

	return res, nil
}
```

Concretely: move everything currently inside `Run` (lines 72–137, i.e. `res :=
&Result{}` through `return res, nil`) into `runAt`, and delete the
`now := time.Now()` line (line 71) — `now` is now the parameter. Do not change
any other line of the body.

**Verify**:
- `grep -n "func runAt\|func Run\b" internal/seed/seed.go` → both present; `Run` is a 1-line wrapper
- `go build ./internal/seed/...` → exit 0
- `go vet ./internal/seed/...` → exit 0

### Step 2: Anchor the summaries test to the fixed Wednesday

In `internal/seed/seed_test.go`, in `TestSeedSummariesPopulatesAllBandLevels`,
replace the `Run(app)` call with `runAt` using the same fixed date the sibling
test uses:

```go
func TestSeedSummariesPopulatesAllBandLevels(t *testing.T) {
	app := storetest.NewApp(t)
	// Anchor to a fixed Wednesday so all five recap bands (incl. the day band,
	// which is empty on Mondays by design) are populated — deterministic
	// regardless of when the suite runs. Mirrors TestSeedPeriodsCoversAllBands.
	now := time.Date(2026, 6, 24, 12, 0, 0, 0, time.UTC)
	if _, err := runAt(app, now); err != nil {
		t.Fatalf("runAt: %v", err)
	}
	// ... rest of the test (the wantTypes loop + quarter/year guard) unchanged ...
}
```

Leave the rest of the test body exactly as is.

**Verify**:
- `grep -n "runAt(app, now)" internal/seed/seed_test.go` → one match (in this test)
- `go test ./internal/seed/ -run TestSeedSummariesPopulatesAllBandLevels -count=1 -v` → PASS

### Step 3: Full verification (with `-count=1`)

**Verify**:
- `gofmt -l internal/seed` → prints nothing
- `go vet ./...` → exit 0
- `go test ./internal/seed/ -count=1` → PASS (ALL seed tests, incl. the ones still on real `Run`)
- `go test ./... -count=1` → all pass (this proves the gate is GREEN uncached — the whole point)
- `CGO_ENABLED=0 go build ./...` → exit 0

The pre-commit hook (`make lint` → full suite) will now PASS, so your commit will
succeed. (Before this fix it was RED on Mondays.)

## Test plan
- No NEW test function — this fixes an existing one's determinism. The proof is
  `go test ./internal/seed/ -count=1` passing (it fails on `main` today).
- Do NOT weaken the assertion (still require ≥1 row for all 5 period types incl.
  `day`, and keep the quarter/year generic-text guard). The fix is the anchor
  date, not the assertion.

## Done criteria — ALL must hold
- [ ] `grep -n "func runAt" internal/seed/seed.go` returns one match; `Run` is a thin wrapper calling `runAt(app, time.Now())`
- [ ] `grep -n "time.Now()" internal/seed/seed.go` no longer matches inside `Run`'s old body (only the `Run` wrapper + the unrelated `Reset` call remain)
- [ ] `go test ./internal/seed/ -count=1` PASSES
- [ ] `go test ./... -count=1` exits 0 (gate green, uncached)
- [ ] `gofmt -l internal/seed` prints nothing; `go vet ./...` exits 0
- [ ] Only `internal/seed/seed.go` and `internal/seed/seed_test.go` modified (`git status`)

## STOP conditions
Stop and report (do not improvise) if:
- After the seam, a seed test OTHER than `TestSeedSummariesPopulatesAllBandLevels`
  fails under `-count=1` — that's a second clock-fragile test; report it, don't
  silently convert it.
- `go test ./... -count=1` still shows a RED package after the fix — report which
  (the seed fix should make it green; a different red package is a separate issue).
- You find `runAt` would need to change any sub-seeder's behavior (it should NOT —
  the body is verbatim, only `now` becomes a param).

## Maintenance notes
- `Run` keeps real `time.Now()` in production; only tests inject a fixed clock.
- If a future test needs deterministic seeding, call `runAt(app, fixedNow)`.
- The recap day-band is intentionally empty on Mondays — never "fix" that in
  `internal/recap` to satisfy a test.
