# Plan 066: drop the discarded per-kind life.Series scan from the lifelog tile

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 1f8f55e..HEAD -- internal/feature/lifecards/lifelog.go internal/feature/lifecards/lifelog_test.go`
> If either in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: perf
- **Planned at**: commit `1f8f55e`, 2026-06-15

## Why this matters

Every render of the lifelog tile fires one `life.Series` query per tracked
kind and then **throws the result away**. `life.Series` is a
`FindRecordsByFilter` over the `entries` table that returns up to 500 rows
(`internal/life/life.go:134-138`). The tile only ever displays `Kind`, `Unit`,
and `Count` — all three already come back from the single `GROUP BY` aggregate
in `life.Kinds` (`internal/life/life.go:103-131`). So for N tracked kinds the
tile issues N pointless ≤500-row scans on every render, and the lifelog tile
renders both on the board grid and again on the Part-B live refresh — doubling
the waste. This is a regression introduced by the gomponents migration: the
legacy `lifeOverview` in `internal/web/life.go` genuinely consumed `Series`
for sparklines and a "Recent" list, but the tile port kept the query and
dropped the consumers, leaving a comment that admits it
(`_ = recs // series fetched for consistency with legacy; tile shows count only`).
Removing the call deletes work whose output nothing reads.

## Current state

- `internal/feature/lifecards/lifelog.go` — renders the lifelog tile; the dead
  query lives in `buildLifelog` (lines 55-77).
- `internal/life/life.go` — owns `Kinds` (the aggregate that already supplies
  everything the tile needs) and `Series` (the wasted scan). **Not in scope**;
  other callers rely on both.

**The data builder as it exists today (`internal/feature/lifecards/lifelog.go:55-77`):**

```go
func buildLifelog(app core.App) LifelogView {
	now := time.Now()
	var kinds []LifeKindView
	ks, err := life.Kinds(app)
	if err == nil {
		for _, k := range ks {
			recs, err := life.Series(app, k.Kind, now.AddDate(0, 0, -lifeWindowDays))
			if err != nil {
				continue
			}
			_ = recs // series fetched for consistency with legacy; tile shows count only
			kinds = append(kinds, LifeKindView{
				Kind:  k.Kind,
				Unit:  k.Unit,
				Count: k.Count,
			})
		}
	}
	return LifelogView{
		Kinds:  kinds,
		Habits: buildLifelogHabits(app, now),
	}
}
```

**The view-model the tile actually consumes (`internal/feature/lifecards/lifelog.go:25-32`):**

```go
type LifeKindView struct {
	Kind  string
	Unit  string
	Count int
}
```

It uses only `Kind`, `Unit`, `Count` — every field is already populated by
`life.Kinds`. The `KindInfo` returned by `Kinds`
(`internal/life/life.go:92-99`, `:122-129`) carries exactly `Kind`, `Count`,
`Unit` (plus `Last`/`NumCount`, which the tile ignores).

**Two references the executor must NOT remove blindly:**

- `now` is still used **after** the loop, at line 75:
  `Habits: buildLifelogHabits(app, now)`. Keep `now := time.Now()`.
- `lifeWindowDays` is a **package-level const declared in a sibling file**,
  `internal/feature/lifecards/measure.go:25-28`:
  ```go
  const (
  	lifeWindowDays = 90
  	sparkW, sparkH = 240, 48
  )
  ```
  It is still consumed by `buildMeasure` at `measure.go:52`
  (`days := intParam(params, "days", lifeWindowDays)`). **Do NOT remove or
  move it.** After this change `lifelog.go` simply stops referencing it; that
  is fine — the const stays where it is for `measure.go`.

**Behavior edge that intentionally changes.** Today, a kind whose `life.Series`
call returns an error hits `if err != nil { continue }` and is **dropped from
the tile**. After this fix there is no `Series` call, so **every kind returned
by `life.Kinds` appears** in the tile. This is the correct behavior: a kind in
`Kinds` is a real tracker the owner logged; it should never have been hidden by
an unrelated `Series` error. Call this out in the commit message.

**Test conventions for this repo (inlined — the executor has not seen them):**

- Tests use the standard `testing` package, table-driven where it helps, **no
  assertion frameworks**, and **never hit a real model or daemon**.
- `storetest.NewApp(t)` returns a `core.App` backed by a temp-dir PocketBase
  instance with all migrations applied. Signature (verified at
  `internal/storetest/storetest.go:18`): `func NewApp(t *testing.T) core.App`.
- Seed life entries with `life.Log(app, life.LogOpts{Kind: ..., ValueNum: ...,
  Unit: ...})`. Real example from `internal/life/life_test.go:64-95`
  (`TestKindsInventory`):
  ```go
  app := storetest.NewApp(t)
  if _, err := Log(app, LogOpts{Kind: "weight", ValueNum: 82.5, Unit: "kg"}); err != nil {
  	t.Fatal(err)
  }
  if _, err := Log(app, LogOpts{Kind: "gratitude", Text: "the morning was quiet"}); err != nil {
  	t.Fatal(err)
  }
  kinds, err := Kinds(app)
  ```
- `buildLifelog` is **unexported**, so a test that calls it must be in the
  **internal** test package `package lifecards` (not `lifecards_test`). The
  existing `internal/feature/lifecards/lifelog_test.go` is `package
  lifecards_test` and only renders pre-built view-models — it cannot reach
  `buildLifelog`. You will add a second test file in `package lifecards`
  (see Step 3).

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Drift | `git diff --stat 1f8f55e..HEAD -- internal/feature/lifecards/lifelog.go internal/feature/lifecards/lifelog_test.go` | empty |
| Format check | `gofmt -l internal/feature/lifecards/` | prints nothing |
| Vet | `go vet ./internal/feature/lifecards/` | exit 0 |
| Package tests | `go test ./internal/feature/lifecards/` | all pass |
| All tests | `go test ./...` | all pass |
| Host build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Series gone from lifelog.go | `grep -c 'life.Series' internal/feature/lifecards/lifelog.go` | `0` |
| Whitespace | `git diff --check` | no output |

## Scope

**In scope** (modify only these):
- `internal/feature/lifecards/lifelog.go` — delete the `life.Series` call and
  the `if err != nil { continue }` guard; build `LifeKindView` directly from
  the `life.Kinds` entry.
- `internal/feature/lifecards/lifelog_test.go` — **OR** a new sibling file
  `internal/feature/lifecards/lifelog_builder_test.go` (preferred — the
  existing file is `package lifecards_test` and the new test must be `package
  lifecards`; see Step 3). Pick exactly one of these two; do not create both.

**Out of scope** (do NOT touch, even though they look related):
- `internal/life/life.go` — `Series` and `Kinds` stay exactly as they are.
  Other callers depend on them: `internal/feature/lifecards/lines.go:38`,
  `internal/feature/lifecards/measure.go:56`, `internal/web/life.go:49`,
  `internal/cli/life.go:89`, `internal/tools/life.go:133`. Do not "optimize"
  `Series` or fold it into `Kinds`.
- `internal/feature/lifecards/measure.go` — keep the `lifeWindowDays` const
  there; `buildMeasure` still uses it.
- `internal/web/life.go` — the legacy `lifeOverview`; it legitimately consumes
  `Series` for sparklines. Leave it.
- The `LifeKindView` struct shape and any rendered HTML — output must be
  byte-for-byte identical for any given set of kinds.

## Git workflow

- Branch: `improve/066-lifelog-dead-series-query`
- One commit; conventional-commit style, e.g.
  `perf(lifecards): drop discarded per-kind life.Series scan from lifelog tile`
  — and note in the body that errored-Series kinds now appear (intended).
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Remove the discarded `life.Series` call from `buildLifelog`

In `internal/feature/lifecards/lifelog.go`, replace the body of the `for _, k
:= range ks` loop so it no longer calls `life.Series`. Build each
`LifeKindView` directly from the `life.Kinds` entry. The whole function becomes:

```go
func buildLifelog(app core.App) LifelogView {
	now := time.Now()
	var kinds []LifeKindView
	ks, err := life.Kinds(app)
	if err == nil {
		for _, k := range ks {
			kinds = append(kinds, LifeKindView{
				Kind:  k.Kind,
				Unit:  k.Unit,
				Count: k.Count,
			})
		}
	}
	return LifelogView{
		Kinds:  kinds,
		Habits: buildLifelogHabits(app, now),
	}
}
```

Notes:
- Keep `now := time.Now()` — it is still passed to `buildLifelogHabits(app, now)`.
- Do NOT remove the `time` import; `time.Now()` still uses it.
- Do NOT touch `lifeWindowDays` (it lives in `measure.go` and is used there).
- The `_ = recs // ...` comment line is deleted along with the `Series` call.

**Verify**:
```
grep -c 'life.Series' internal/feature/lifecards/lifelog.go   # → 0
gofmt -l internal/feature/lifecards/lifelog.go                # → prints nothing
go build ./internal/feature/lifecards/                        # exit 0
```
If `go build` complains that `time` is now unused, STOP — that means something
in this function changed unexpectedly (`now` should still reference it). Report it.

### Step 2: Confirm `lifeWindowDays` is still referenced in the package

This is a guardrail, not an edit. Run:

```
grep -rn 'lifeWindowDays' internal/feature/lifecards/
```

**Verify**: at least one hit in `measure.go` (its declaration at `measure.go:25-28`
and its use at `measure.go:52`) and **zero** hits in `lifelog.go`. If `go vet`
or `go build` ever reports `lifeWindowDays declared and not used`, you removed
or broke the const — STOP and report; the const must stay in `measure.go`.

### Step 3: Add a builder test that proves every kind appears, independent of Series

`buildLifelog` is unexported, so the test must be in `package lifecards`. The
existing `internal/feature/lifecards/lifelog_test.go` is `package
lifecards_test` and only renders view-models — do not convert it. Create a new
file `internal/feature/lifecards/lifelog_builder_test.go`:

```go
package lifecards

import (
	"testing"

	"github.com/alexradunet/balaur/internal/life"
	"github.com/alexradunet/balaur/internal/storetest"
)

// TestBuildLifelogListsEveryKind seeds two trackers and asserts buildLifelog
// returns a LifeKindView for each with the correct Count — without depending on
// life.Series (the tile reads only the life.Kinds aggregate).
func TestBuildLifelogListsEveryKind(t *testing.T) {
	app := storetest.NewApp(t)
	// weight: two entries → Count 2. mood: one entry → Count 1.
	for _, o := range []life.LogOpts{
		{Kind: "weight", ValueNum: 82.5, Unit: "kg"},
		{Kind: "weight", ValueNum: 82.1, Unit: "kg"},
		{Kind: "mood", ValueNum: 7},
	} {
		if _, err := life.Log(app, o); err != nil {
			t.Fatalf("seed %s: %v", o.Kind, err)
		}
	}

	v := buildLifelog(app)

	got := map[string]int{}
	for _, k := range v.Kinds {
		got[k.Kind] = k.Count
	}
	if len(v.Kinds) != 2 {
		t.Fatalf("Kinds = %d (%v), want 2", len(v.Kinds), got)
	}
	if got["weight"] != 2 {
		t.Errorf("weight Count = %d, want 2", got["weight"])
	}
	if got["mood"] != 1 {
		t.Errorf("mood Count = %d, want 1", got["mood"])
	}
}
```

This test fails to even compile against the old code only if `buildLifelog`'s
signature changed; against the fixed code it passes and pins the new contract
(all `Kinds` entries surface, counts preserved). It deliberately does not
reference `life.Series`.

**Verify**:
```
gofmt -l internal/feature/lifecards/                 # → prints nothing
go vet ./internal/feature/lifecards/                 # exit 0
go test ./internal/feature/lifecards/                # all pass, incl. the new test
```

### Step 4: Full-tree verification

```
go vet ./...
go test ./...
CGO_ENABLED=0 go build ./...
git diff --check
```

**Verify**: vet clean; all tests pass (the existing render tests in
`lifelog_test.go` and `measure_test.go` are unaffected because the rendered
HTML for any given view-model is unchanged); CGO-free build exits 0; no
whitespace errors.

## Test plan

- **New test** in `internal/feature/lifecards/lifelog_builder_test.go`
  (`package lifecards`): seed 2 kinds (one with 2 entries, one with 1) via
  `life.Log`, call `buildLifelog(app)`, assert both kinds appear with the
  correct `Count`. This is the regression guard — it proves the tile lists
  every `Kinds` entry without any `Series` round-trip.
- **Structural pattern to copy**: seeding follows `TestKindsInventory` in
  `internal/life/life_test.go:64-95`; the `storetest.NewApp(t)` harness usage
  matches every test in that file.
- **Existing tests stay green**: the render tests in
  `internal/feature/lifecards/lifelog_test.go` and `measure_test.go` build
  view-models by hand and never call `buildLifelog`, so they are unaffected.
- Verification: `go test ./internal/feature/lifecards/` and `go test ./...` →
  all pass, including the new test.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -c 'life.Series' internal/feature/lifecards/lifelog.go` → `0`
- [ ] `buildLifelog` builds each `LifeKindView` directly from the `life.Kinds`
      entry; the `if err != nil { continue }` guard and the `_ = recs` comment
      are gone
- [ ] `now := time.Now()` is retained and still passed to `buildLifelogHabits`
- [ ] `lifeWindowDays` is unchanged in `internal/feature/lifecards/measure.go`
      and no longer referenced in `lifelog.go`
- [ ] A test in `package lifecards` seeds ≥2 kinds and asserts `buildLifelog`
      returns each with the right `Count` (new test exists and passes)
- [ ] `gofmt -l internal/feature/lifecards/` prints nothing
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` passes
- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `git diff --check` prints nothing
- [ ] `git status --porcelain` shows only `internal/feature/lifecards/lifelog.go`
      modified plus the one new `lifelog_builder_test.go` (or, if you chose to
      extend the existing file instead, only `lifelog.go` and `lifelog_test.go`)
- [ ] `plans/readme.md` status row for 066 updated (unless your reviewer maintains it)

## STOP conditions

Stop and report back (do not improvise) if:

- `buildLifelog` does not match the "Current state" excerpt
  (`internal/feature/lifecards/lifelog.go:55-77` has drifted since this plan
  was written).
- Removing the `life.Series` call leaves `time` unused, or any other reference
  in `lifelog.go` you didn't expect breaks the build — that means the function
  changed in ways this plan didn't account for; report the exact compiler error.
- `go vet`/`go build` reports `lifeWindowDays declared and not used` — the
  const in `measure.go` was removed or broken; it must stay (Step 2).
- Any existing test in `internal/feature/lifecards/` fails after the change
  (the rendered HTML for a given view-model must be identical — output did not
  change, only the data path).

## Maintenance notes

- This relies on `life.Kinds` continuing to return `Unit` (it does, via the
  `COALESCE(MAX(NULLIF(unit, '')), '')` column at `internal/life/life.go:114`).
  If a future change drops `Unit` from the aggregate, the tile's `Unit` would
  go blank — revisit `buildLifelog` then.
- If the lifelog **tile** ever needs sparklines or a recent-entries list (the
  data the legacy `lifeOverview` used `Series` for), reintroduce the `Series`
  call **and consume its result** — do not resurrect a fetch-and-discard.
- A reviewer should confirm the commit notes the intended behavior change
  (kinds whose `Series` errored are no longer hidden) and that no rendered HTML
  changed for a fixed view-model.
