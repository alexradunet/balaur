# Plan 196: Move the plan-183 test-only compute counter out of `internal/knowledge` service code

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat e06346d..HEAD -- internal/knowledge/cache.go internal/knowledge/cache_test.go`
> If either in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `e06346d`, 2026-06-25

## Why this matters

Plan 183 (cache the upfront-memory / active-skill sets) left a package-level,
test-only seam in **service** code: `var contextCacheComputes atomic.Int64` in
`internal/knowledge/cache.go`, incremented on every cache MISS inside
`loadContextCache`, but read by exactly one test
(`TestCacheWarmReadDoesNotRescan`) to prove a warm read does not re-scan.
`AGENTS.md` forbids global mutable state ("No global mutable state. Pass
`core.App`, config structs, and loggers explicitly."). The plan-183 reviewer
accepted it as a transparent, justified test seam but flagged it for cleanup.

This plan removes the mutable counter from the production hot path **without
weakening the warm-read-no-rescan guarantee**. After this lands, the normal
(non-test) build has no package-level mutable counter in the cache code, and the
warm-read regression test still fails the moment the `GetOk` fast-path is
removed. Net effect: same guarantee, no global-mutable-state smell, smaller
production surface.

## Current state

Files in scope, with their role:

- `internal/knowledge/cache.go` — the context-cache singleton (upfront memories
  + active skills) read by `BuildContext` each turn. Holds the offending counter
  and its increment site inside `loadContextCache`.
- `internal/knowledge/cache_test.go` — same-package (`package knowledge`) tests;
  `TestCacheWarmReadDoesNotRescan` is the sole reader of the counter.

### The offending counter (`internal/knowledge/cache.go`)

Imports (lines 3–10) — note `sync/atomic` exists ONLY for this counter:

```go
import (
	"sort"
	"sync/atomic"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
)
```

The counter declaration (lines 16–20):

```go
// contextCacheComputes counts how many times loadContextCache ran the full
// nodes.ListByTypeStatus scan (a cache miss). It is the deterministic signal the
// warm-cache test asserts on: a warm read must NOT increment it. Test-only seam,
// kept tiny and unexported; service code never reads it.
var contextCacheComputes atomic.Int64
```

The increment site — inside `loadContextCache`, ONLY on a miss (line 96, in
context of lines 89–114):

```go
func loadContextCache(app core.App) (*contextCache, error) {
	if raw, ok := app.Store().GetOk(contextCacheKey); ok {
		if c, ok := raw.(*contextCache); ok && c != nil {
			return c, nil
		}
	}

	contextCacheComputes.Add(1) // miss: the scan runs (see the warm-cache test)
	upfront, err := computeUpfront(app, upfrontLimit)
	if err != nil {
		return &contextCache{}, err
	}
	skills, err := computeActiveSkills(app)
	if err != nil {
		return &contextCache{upfront: upfront}, err
	}

	fresh := &contextCache{upfront: upfront, skills: skills}
	// GetOrSet stores exactly once under the store write lock; a concurrent
	// miss that already populated the key wins, and we return its value.
	stored := app.Store().GetOrSet(contextCacheKey, func() any { return fresh })
	if c, ok := stored.(*contextCache); ok && c != nil {
		return c, nil
	}
	return fresh, nil
}
```

The `GetOk` fast-path (lines 90–94) is the warm-read short-circuit the
regression test protects: a warm cache returns BEFORE the increment/compute.
Removing it makes `loadContextCache` fall through to the miss path on every call,
which is exactly the bug `TestCacheWarmReadDoesNotRescan` must catch.

### The sole reader (`internal/knowledge/cache_test.go`)

`TestCacheWarmReadDoesNotRescan` (lines 254–297) reads `contextCacheComputes`
three times via `.Load()` and uses deltas to assert miss-count. Verbatim
excerpt of the load/assert sites:

```go
func TestCacheWarmReadDoesNotRescan(t *testing.T) {
	app := storetest.NewApp(t)
	id := activeMemory(t, app, "stale-probe", 5)

	start := contextCacheComputes.Load()
	if !contains(mustUpfront(t, app), id) { // miss: warms the cache, computes once
		t.Fatal("memory missing before warm")
	}
	if got := contextCacheComputes.Load() - start; got != 1 {
		t.Fatalf("first (cold) read computed %d times, want 1", got)
	}

	// ... raw hook-free demotion via app.Save (unchanged) ...

	warm := contextCacheComputes.Load()
	if !contains(mustUpfront(t, app), id) {
		t.Fatal("warm read reflected a hook-free change — it re-scanned instead of serving the snapshot")
	}
	_ = mustUpfront(t, app) // another warm read
	if got := contextCacheComputes.Load() - warm; got != 0 {
		t.Fatalf("warm reads computed %d times, want 0 — the scan re-ran on a warm cache", got)
	}

	// Now invalidate explicitly and confirm the next read recomputes ...
	invalidateContextCache(app)
	if contains(mustUpfront(t, app), id) {
		t.Fatal("after invalidation the demoted memory is still injected — recompute did not run")
	}
	if got := contextCacheComputes.Load() - warm; got != 1 {
		t.Fatalf("post-invalidation reads computed %d times, want exactly 1", got)
	}
}
```

The test's doc comment (lines 245–253) also names `contextCacheComputes` ("the
deterministic compute counter (incremented only on a cache MISS)") — update that
prose to the new mechanism name as part of this plan.

### Why the chosen fix is safe — the callers do not see the seam

`loadContextCache` is called only by `UpfrontMemories` and `ActiveSkills`, which
live in `internal/knowledge/context.go` and are themselves called by
`BuildContext` (context.go lines 37 and 64). This plan does NOT change
`loadContextCache`'s signature, so nothing ripples to those callers. Confirmed
call sites (`internal/knowledge/context.go`):

```
25:	upfrontLimit = 12
37:	upfront, err := UpfrontMemories(app, upfrontLimit)
64:	skills, err := ActiveSkills(app)
```

### Repo conventions that apply here

- **No global mutable state** (`AGENTS.md`): this is the rule being satisfied.
  A package-level FUNCTION variable that defaults to a no-op and is overridden
  only by the test is the accepted "test seam" shape — it is not mutable
  application state (no shared counter in the production build; the prod default
  never mutates anything).
- **Same-package tests** override unexported symbols directly. `cache_test.go`
  is `package knowledge` (line 1), so it can reassign an unexported
  package-level `var` declared in `cache.go` with no `export_test.go` needed.
  Exemplar of same-package test files in this repo: `internal/cli/export_test.go`
  (declared `package cli`, line 1) reaches unexported helpers.
- **gofmt is law**; `go vet` / `staticcheck` gate CI. `staticcheck` flags unused
  package-level symbols (U1000) — after this change there must be **no** unused
  symbol and **no** leftover unused import.
- **Suckless / KISS**: smallest correct change. Do not introduce a new struct
  field, a new constructor parameter, or a callback threaded through
  `UpfrontMemories`/`ActiveSkills`. The seam stays a single package-level
  function var.

## Commands you will need

Set `TMPDIR` before any `go` command (the repo's tmpfs `/tmp` overflows the
linker — see the repo memory note "tmpfs /tmp breaks go link"):

```
export TMPDIR=/home/alex/.cache/go-tmp
```

| Purpose            | Command                                                                 | Expected on success                          |
|--------------------|-------------------------------------------------------------------------|----------------------------------------------|
| Build (CGO-free)   | `CGO_ENABLED=0 go build ./...`                                          | exit 0, no output                            |
| Package tests      | `go test ./internal/knowledge/`                                        | `ok  ...internal/knowledge`                  |
| Targeted tests     | `go test ./internal/knowledge/ -run 'TestCacheWarmReadDoesNotRescan\|TestCacheReturnedRecordIsThrowaway' -v` | both PASS                                     |
| Vet                | `go vet ./internal/knowledge/`                                         | exit 0, no output                            |
| Staticcheck        | `staticcheck ./internal/knowledge/`                                    | exit 0, no output (no U1000/unused)          |
| Format check       | `gofmt -l internal/knowledge/cache.go internal/knowledge/cache_test.go` | empty output                                 |
| Whitespace check   | `git diff --check`                                                     | no output                                    |

`staticcheck` is at `/home/alex/go/bin/staticcheck`; if `staticcheck` is not on
`PATH`, invoke it by that absolute path.

## Scope

**In scope** (the ONLY files you may modify):
- `internal/knowledge/cache.go`
- `internal/knowledge/cache_test.go`

**Out of scope** (do NOT touch, even though they look related):
- `internal/knowledge/context.go` — `UpfrontMemories` / `ActiveSkills` /
  `BuildContext`. The fix must not change `loadContextCache`'s signature, so
  these callers stay untouched. If you find yourself editing this file, STOP.
- The cache logic itself: `computeUpfront`, `computeActiveSkills`, the
  `GetOk`/`GetOrSet` flow, `invalidateContextCache`, `RegisterCacheInvalidation`,
  the `copyForRead` race fix, and the `contextCache` concurrency invariant — all
  correct, all out of scope. Do NOT "improve" them.
- The other cache tests (`TestCacheParity`, `TestCacheAddInvalidates`,
  `TestCacheDropInvalidates`, `TestCacheArchiveInvalidates`,
  `TestCacheEditInvalidates`, `TestCacheSkillInvalidates`,
  `TestCacheTouchDoesNotInvalidate`, `TestCacheReturnedRecordIsThrowaway`). They
  must keep passing; do not rewrite them.

## Git workflow

- Branch: `advisor/196-cache-compute-counter-cleanup` (or the executor
  worktree's branch off `origin/main`).
- One commit; conventional-commit subject, e.g.
  `refactor(knowledge): move test-only cache-miss counter out of service code`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

Order matters: change production code and test together in step 1 (the test
overrides the new hook), then re-verify, then run the fail-on-revert proof
(step 3) and restore.

### Step 1: Replace the atomic counter with a no-op miss hook in `cache.go`

In `internal/knowledge/cache.go`:

1. **Drop the `sync/atomic` import** (lines 3–10). The new hook needs no atomic
   type. Resulting import block:

   ```go
   import (
   	"sort"

   	"github.com/pocketbase/pocketbase/core"

   	"github.com/alexradunet/balaur/internal/nodes"
   )
   ```

2. **Replace the counter declaration** (lines 16–20) with a package-level
   function variable that defaults to a no-op. The production build never
   mutates anything through it; only the test overrides it.

   ```go
   // onContextCacheMiss is a test-only seam fired once per cache MISS inside
   // loadContextCache (a miss runs the full nodes.ListByTypeStatus scan). It
   // defaults to a no-op so the production hot path carries no counter and no
   // shared mutable state; cache_test.go overrides it to assert the warm-read
   // guarantee (a warm read must NOT fire it). Keep it unexported and miss-only.
   var onContextCacheMiss = func() {}
   ```

3. **Replace the increment site** in `loadContextCache` (line 96). Change:

   ```go
   	contextCacheComputes.Add(1) // miss: the scan runs (see the warm-cache test)
   ```

   to:

   ```go
   	onContextCacheMiss() // miss: the scan runs (see the warm-cache test)
   ```

   Leave the rest of `loadContextCache` — the `GetOk` fast-path, both computes,
   the `GetOrSet` store — exactly as-is.

**Verify** (after step 2 lands too — the test references the new symbol):
`gofmt -l internal/knowledge/cache.go` → empty.

### Step 2: Rewire `TestCacheWarmReadDoesNotRescan` onto the miss hook

In `internal/knowledge/cache_test.go`, change `TestCacheWarmReadDoesNotRescan`
(lines 254–297) to drive a local counter through the `onContextCacheMiss` hook
instead of reading the removed `contextCacheComputes` global. Since the test is
same-package (`package knowledge`), it assigns the package var directly.

Target shape — install a local counter at the top of the test and restore the
no-op on cleanup, then replace each `contextCacheComputes.Load()` read with the
local counter's value:

```go
func TestCacheWarmReadDoesNotRescan(t *testing.T) {
	app := storetest.NewApp(t)
	id := activeMemory(t, app, "stale-probe", 5)

	// Drive the production miss hook to count cache misses for this test only.
	// Restore the no-op so other tests in the package are unaffected.
	var computes int
	prev := onContextCacheMiss
	onContextCacheMiss = func() { computes++ }
	t.Cleanup(func() { onContextCacheMiss = prev })

	start := computes
	if !contains(mustUpfront(t, app), id) { // miss: warms the cache, computes once
		t.Fatal("memory missing before warm")
	}
	if got := computes - start; got != 1 {
		t.Fatalf("first (cold) read computed %d times, want 1", got)
	}

	// Raw, hook-free demotion: load by id and app.Save directly. This bypasses
	// every invalidation seam, so a warm read MUST still serve the stale snapshot.
	rec, err := app.FindRecordById("nodes", id)
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	props := nodes.Props(rec)
	props["importance"] = 1
	rec.Set("props", props)
	if err := app.Save(rec); err != nil {
		t.Fatalf("raw save: %v", err)
	}

	warm := computes
	if !contains(mustUpfront(t, app), id) {
		t.Fatal("warm read reflected a hook-free change — it re-scanned instead of serving the snapshot")
	}
	_ = mustUpfront(t, app) // another warm read
	if got := computes - warm; got != 0 {
		t.Fatalf("warm reads computed %d times, want 0 — the scan re-ran on a warm cache", got)
	}

	// Now invalidate explicitly and confirm the next read recomputes (and now
	// reflects the demotion: importance 1 < 4, so it drops out of upfront).
	invalidateContextCache(app)
	if contains(mustUpfront(t, app), id) {
		t.Fatal("after invalidation the demoted memory is still injected — recompute did not run")
	}
	if got := computes - warm; got != 1 {
		t.Fatalf("post-invalidation reads computed %d times, want exactly 1", got)
	}
}
```

Notes:
- The hook-free `app.Save` demotion block (find → set props → save) is unchanged;
  keep it verbatim.
- Restoring via `t.Cleanup` (not a bare `defer`) matches the repo's testing
  idioms and keeps the package var clean for sibling tests, which is important
  because the hook is process-global within the package.
- Also update the test's doc comment (lines 245–253) so it no longer says
  "compute counter (incremented only on a cache MISS)". Reword to name the new
  mechanism, e.g.: "We install a local counter via the `onContextCacheMiss`
  hook (fired only on a cache MISS) and assert the first read computes once,
  warm reads compute zero more times, and only an invalidation recomputes.
  Reverting `loadContextCache` to compute unconditionally makes the warm-read
  assertion FAIL." Do not change the test name.

**Verify**:
```
export TMPDIR=/home/alex/.cache/go-tmp
gofmt -l internal/knowledge/cache.go internal/knowledge/cache_test.go   # empty
go vet ./internal/knowledge/                                            # exit 0
staticcheck ./internal/knowledge/                                       # exit 0, no U1000
go test ./internal/knowledge/                                           # ok
```
Expect: build/vet/staticcheck clean (no "contextCacheComputes" unused, no
unused `sync/atomic` import) and the full `internal/knowledge` suite green,
including `TestCacheWarmReadDoesNotRescan` and
`TestCacheReturnedRecordIsThrowaway`.

### Step 3: Prove the warm-read guarantee still fails-on-revert, then restore

This is mandatory evidence the seam still does its job. Do NOT commit while the
fast-path is removed.

1. In `internal/knowledge/cache.go`, TEMPORARILY remove the `GetOk` fast-path at
   the top of `loadContextCache` (lines 90–94) — the block:

   ```go
   	if raw, ok := app.Store().GetOk(contextCacheKey); ok {
   		if c, ok := raw.(*contextCache); ok && c != nil {
   			return c, nil
   		}
   	}
   ```

   With it gone, every call falls through to the miss path and fires
   `onContextCacheMiss()`.

2. Run only the warm-read test:
   ```
   go test ./internal/knowledge/ -run TestCacheWarmReadDoesNotRescan
   ```
   **Expected: FAIL** — the warm-read delta assertion (`want 0`) trips because
   the warm reads now recompute. If it PASSES, the seam is not wired correctly;
   treat that as a STOP condition (the rewrite did not preserve the guarantee).

3. **Restore** the `GetOk` fast-path exactly as it was (lines 90–94). Re-run:
   ```
   go test ./internal/knowledge/
   ```
   **Expected: ok** (all green again).

Record in your report that you performed the fail-on-revert proof and that the
test FAILED with the fast-path removed and PASSED once restored.

### Step 4: Final full-suite gate before declaring done

```
export TMPDIR=/home/alex/.cache/go-tmp
gofmt -l internal/knowledge/cache.go internal/knowledge/cache_test.go   # empty
git diff --check                                                        # no output
go vet ./internal/knowledge/                                            # exit 0
staticcheck ./internal/knowledge/                                       # exit 0
CGO_ENABLED=0 go build ./...                                            # exit 0
go test ./internal/knowledge/                                           # ok
go test ./...                                                           # all pass
```

`go test ./...` is the repo's push gate (never push red). If the tmpfs link bug
surfaces ("No space left on device") despite `TMPDIR`, confirm `TMPDIR` is
exported in the SAME shell and re-run; if it persists, STOP and report (do not
hand-edit anything else).

## Test plan

No new test files. The change is verified by EXISTING tests plus the
fail-on-revert proof:

- `TestCacheWarmReadDoesNotRescan` — rewired in step 2 to assert via the
  `onContextCacheMiss` hook instead of the removed global; still the core
  warm-read-no-rescan guard. Must PASS normally and FAIL when the `GetOk`
  fast-path is removed (step 3).
- `TestCacheReturnedRecordIsThrowaway` — unchanged; must keep passing (data-race
  guard, named in the brief's DONE criteria).
- The remaining six cache tests and the rest of `internal/knowledge` — must keep
  passing.

Structural pattern to follow for the test-local counter + `t.Cleanup` restore:
the override-and-restore idiom is standard table-free Go; mirror the cleanup
discipline already used across `internal/knowledge` tests (`storetest.NewApp(t)`,
`t.Helper()`, `t.Fatalf`). No assertion framework, no `time.Sleep`.

Verification: `go test ./internal/knowledge/` → `ok`, including the two named
tests; plus `go test ./...` → all pass.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -n "contextCacheComputes" internal/knowledge/` returns NO matches
      (the global counter is gone from both files).
- [ ] `grep -n "sync/atomic" internal/knowledge/cache.go` returns NO matches
      (the now-unused import is removed).
- [ ] `grep -n "onContextCacheMiss" internal/knowledge/cache.go` shows the
      no-op default declaration AND the call site in `loadContextCache`.
- [ ] `loadContextCache`'s signature is unchanged
      (`func loadContextCache(app core.App) (*contextCache, error)`), and
      `internal/knowledge/context.go` is NOT modified (`git status` shows only
      `cache.go` and `cache_test.go` changed).
- [ ] `gofmt -l internal/knowledge/cache.go internal/knowledge/cache_test.go`
      is empty.
- [ ] `go vet ./internal/knowledge/` exits 0.
- [ ] `staticcheck ./internal/knowledge/` exits 0 (no U1000 unused symbol, no
      unused import).
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go test ./internal/knowledge/` is `ok`, including
      `TestCacheWarmReadDoesNotRescan` and `TestCacheReturnedRecordIsThrowaway`.
- [ ] `go test ./...` all pass.
- [ ] Fail-on-revert proof performed and reported: warm-read test FAILS with the
      `GetOk` fast-path removed, PASSES once restored; fast-path restored
      verbatim.
- [ ] `git diff --check` is clean.
- [ ] `plans/README.md` status row for plan 196 updated.

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows `cache.go` or `cache_test.go` changed since `e06346d`,
  or the "Current state" excerpts (the counter decl at lines 16–20, the
  increment at line 96, the `GetOk` fast-path at lines 90–94, or the test reads)
  no longer match the live code.
- Removing the global appears to FORCE a signature change to `loadContextCache`,
  `UpfrontMemories`, or `ActiveSkills`, or any edit to
  `internal/knowledge/context.go`. The brief's escape hatch: keep the change
  package-local and minimal (function-var seam only); if you cannot, STOP and
  report.
- In step 3, the warm-read test PASSES with the `GetOk` fast-path removed — the
  seam is not actually proving the guarantee; STOP rather than commit a weakened
  regression test.
- Any verification command fails twice after a reasonable fix attempt.
- The fix appears to require touching any file outside the two in-scope files.
- `internal/self/knowledge.md` is NOT in scope: this is an internal test-seam
  refactor with no change to capability or architecture, so do not update it.

## Maintenance notes

For whoever owns this code next:

- The seam is now `onContextCacheMiss`, a package-level `func()` defaulting to a
  no-op, fired once per cache MISS inside `loadContextCache`. It exists purely so
  `TestCacheWarmReadDoesNotRescan` can prove a warm read does not re-scan. If a
  future change adds a SECOND compute path (e.g. a partial/incremental refresh),
  decide deliberately whether that path also counts as a "miss" for the hook, or
  the warm-read test's delta math will drift.
- Because the hook is a process-global package var, any test that overrides it
  MUST restore it (use `t.Cleanup`), or it leaks into sibling tests. Today only
  one test touches it; keep that invariant.
- A reviewer should confirm: no `sync/atomic` left in `cache.go`, no
  `contextCacheComputes` anywhere, `loadContextCache`'s signature and the
  `GetOk`/`GetOrSet` flow untouched, and the fail-on-revert proof was run.
- Deferred out of this plan (intentionally): nothing. This is the complete
  cleanup the plan-183 reviewer flagged.
