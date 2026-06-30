# Plan 219: De-duplicate repeated `store.OwnerLocation` calls within a single function

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report — do not improvise. When done, update
> this plan's row in `plans/README.md` unless a reviewer told you they maintain
> the index.
>
> **Drift check (run first)**:
> `git diff --stat ef9f2df..HEAD -- internal/recap/compact.go internal/store/time.go`
> If either changed since this plan was written, compare the "Current state"
> excerpts against the live code before proceeding; on a mismatch, STOP.

## Status
- **Priority**: P4 (minor; single-owner v1 scale)
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: perf / cleanliness
- **Planned at**: commit `ef9f2df`, 2026-06-30

## Why this matters

`store.OwnerLocation(app)` does a DB read (`GetOwnerSetting` →
`FindFirstRecordByData`) **plus** `time.LoadLocation(name)` (which re-parses
zoneinfo — Go's `LoadLocation` does not cache) on **every** call. It is invoked
from 20+ sites. Almost all call it exactly once per operation, which is fine —
but a few functions call it **multiple times within the same call**, repeating
the identical DB read + tz parse for no benefit. The clean, low-risk win is to
resolve it once per function and reuse the `*time.Location`, exactly as most
existing call sites already do (e.g. `internal/nodes/day.go:34`,
`internal/web/recap.go:136`).

**Scope is deliberately narrow.** A process-wide memo was considered and
**rejected**: `OwnerLocation`'s own doc comment (`internal/store/time.go:17-21`)
documents per-call resolution as intentional ("Resolved per call: no global
state, and a dashboard edit takes effect on the next cron tick"), and AGENTS.md
forbids global mutable state. We keep that contract; we only stop calling the
function twice where once would do.

## Current state

`internal/store/time.go` — the per-call resolver (unchanged by this plan; shown
for context):

```go
func OwnerLocation(app core.App) *time.Location {
	name := GetOwnerSetting(app, "timezone", "")   // DB read every call
	if name == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(name)            // zoneinfo parse every call
	if err != nil {
		return time.Local
	}
	return loc
}
```

The offender — `internal/recap/compact.go` calls `OwnerLocation` up to **four**
times across `DraftToday`/`CommitToday`, including twice in a single `Sprintf`:

```go
// compact.go:43-44 (inside one Sprintf — two identical calls)
	label := fmt.Sprintf("Today, %s–%s", boundary.In(store.OwnerLocation(app)).Format("15:04"),
		now.In(store.OwnerLocation(app)).Format("15:04"))
// compact.go:65
	now = now.In(store.OwnerLocation(app))
// compact.go:92
	loc := store.OwnerLocation(app)
```

Authoritative list of all call sites (for the executor to audit for other
same-function repeats):

```
grep -rn "OwnerLocation(" internal --include=*.go | grep -v _test.go
```

As of this plan, the only function with a clear multi-call-per-invocation repeat
is in `compact.go` (the `Sprintf` double-call at :43-44, and separately :65/:92
which are in different functions — confirm). Most other sites call once and are
fine. **Verify the current call sites yourself** before editing — do not assume
this list is exhaustive at execution time.

## Commands you will need

| Purpose   | Command                                              | Expected         |
|-----------|------------------------------------------------------|------------------|
| Build     | `CGO_ENABLED=0 go build ./...`                       | exit 0           |
| Vet       | `go vet ./...`                                        | exit 0           |
| Test pkg  | `go test ./internal/recap/... -count=1`             | PASS             |
| Full test | `go test ./... -count=1`                             | all pass         |
| gofmt     | `gofmt -l internal/recap`                            | prints nothing   |

> Tests/commits must run with `TMPDIR=/home/alex/.cache/go-tmp` and `-count=1`.

## Scope

**In scope**:
- Any function that calls `store.OwnerLocation(app)` **more than once per
  invocation** — resolve it once into a local `loc` and reuse. Confirmed target:
  `internal/recap/compact.go`. Audit the full grep list for any other same-
  function repeats and fix those too (only true within-one-call repeats).

**Out of scope / explicitly NOT doing**:
- `internal/store/time.go` `OwnerLocation` itself — do NOT add a cache/memo. The
  per-call contract is documented and intentional.
- Single-call sites — leave them; one call per operation is correct.
- `conversation.Master` caching — considered and rejected: it is a singleton
  indexed lookup called ~once per turn/cron, and caching it risks staleness
  across reseed/reset (a new master record). Not worth the invalidation surface.
- Threading `loc` across package boundaries / function signatures — that is a
  larger refactor; this plan only collapses repeats *within* a function body.

## Git workflow
- Branch: `advisor/219-owner-location-dedup`
- Subject e.g. `perf(recap): resolve OwnerLocation once per compaction call`
- Do NOT push unless the operator instructed it.

## Steps

### Step 1: Audit call sites
Run the grep above. For each FUNCTION, count `OwnerLocation` calls. Mark only the
functions with ≥2 calls in one invocation. (Expected: `compact.go`.)

### Step 2: Collapse repeats
In each marked function, add `loc := store.OwnerLocation(app)` once near where
it is first needed and replace the repeated calls with `loc`. Behavior is
identical (same `*time.Location`), so no test should change meaning. Example for
the `Sprintf`:

```go
	loc := store.OwnerLocation(app)
	label := fmt.Sprintf("Today, %s–%s",
		boundary.In(loc).Format("15:04"), now.In(loc).Format("15:04"))
```

**Verify**: `go build ./... && go vet ./...` → exit 0.

### Step 3: Tests
No behavior change → existing `internal/recap` tests must still pass unchanged.
Do not add new tests unless an existing one lacks coverage of the touched
function (in which case add a minimal one). The win is fewer DB reads + tz
parses per call, which is not worth a dedicated assertion.

**Verify**: `go test ./internal/recap/... -count=1` → PASS.

### Step 4: Full verification
- `gofmt -l internal/recap` → nothing
- `go vet ./...` → exit 0
- `go test ./... -count=1` → all pass

## Done criteria — ALL must hold
- [ ] No function calls `store.OwnerLocation` more than once per invocation (verify by re-running the grep + reading each multi-hit function)
- [ ] `internal/store/time.go` `OwnerLocation` is UNCHANGED (no memo added)
- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0; `gofmt -l` clean
- [ ] `go test ./... -count=1` exits 0 with no test-meaning changes
- [ ] Only the offending call-site file(s) changed (`git status`; expected: `internal/recap/compact.go`)
- [ ] `plans/README.md` row updated

## STOP conditions
Stop and report if:
- The audit finds the repeats are now spread across functions/packages such that
  collapsing them would require changing function signatures — that is out of
  scope; report the larger refactor as deferred work instead.
- Any existing test changes meaning after the dedup — that means the calls were
  NOT equivalent (they ran at different times against a changing setting); STOP,
  because then the repeat was load-bearing.

## Maintenance notes
- This is a micro-optimization; its real value is consistency (most sites already
  resolve once). Reviewer can approve on the diff alone if tests stay green.
- If `OwnerLocation` ever shows up hot in a profile, the correct fix is to thread
  the resolved `*time.Location` down through the recap/turn pipeline (resolve
  once per request/cron tick) — NOT a global cache, to preserve the documented
  next-tick freshness contract. Record that as the next step, not this plan.
