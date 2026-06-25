# Plan 183: Cache the upfront-memory and active-skill sets so chat turns stop full-scanning the nodes collection

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 3d4963c..HEAD -- internal/knowledge/ internal/turn/turn.go internal/search/index.go main.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

> **⚠ REVISION (round 2) — REQUIRED, read before Step 1.** A first execution of
> this plan passed all functional criteria but introduced a **data race**: the
> per-turn `knowledge.Touch` (called from `internal/turn/turn.go` after the model
> reads context) does `app.Save(rec)` on records that may be the **cached
> pointers** returned by `UpfrontMemories` — so one turn mutates a shared cached
> record while another goroutine reads the cached set. This version MUST fix it:
> 1. **Invariant: a cached record is never passed to `app.Save`.** Make the cache
>    hand callers records that `Touch`/`app.Save` will never mutate — simplest
>    correct option: cache an immutable snapshot and have `UpfrontMemories` /
>    `ActiveSkills` return **fresh `*core.Record` copies** per call (copying a
>    dozen records is far cheaper than the full `ListByTypeStatus` scan + hydrate
>    this plan removes). (Alternatively, ensure the records fed to the per-turn
>    `Touch` are re-loaded by id, never the cached instances.)
> 2. **The regression test MUST FAIL when the fix is reverted (round-3 update).**
>    A `-race`-only test does NOT work here — `app.Save`'s internal locking masks
>    the window, so reverting the fix still passes under `-race`. Write a
>    DETERMINISTIC corruption assertion instead: warm the cache, capture an
>    upfront memory's `use_count` via `nodes.PropInt` on the record returned by
>    `UpfrontMemories`; call `Touch(app, Memory, thatReturnedRecord)`; then
>    re-read `UpfrontMemories` and assert the snapshot's `use_count` is UNCHANGED
>    (Touch mutated only the throwaway copy, never the cached snapshot). PROVE the
>    test is a real guard by temporarily reverting `copyForRead` to `return recs`
>    and confirming the new test FAILS, then restore (report that you did this).
>    Keep a `-race` test too (`CGO_ENABLED=1 go test -race ./internal/knowledge/
>    ./internal/turn/`) as a bonus, but the deterministic assertion is the guard.
> 3. Document the concurrency invariant in the cache's doc comment (not only the
>    selection/ordering reasoning).
> Treat this as both a Done criterion and a STOP condition: if you cannot
> guarantee the cache is never mutated via `Touch`, STOP and report instead of
> shipping the race. **Base note:** this plan now executes against `origin/main`
> (`ced5326`), whose `main.go` already carries plan 190's no-args launcher — put
> your cache-invalidation hook in the existing `registerSearchIndex` hook area
> and edit `main.go` additively; do NOT touch 190's launcher branch.

> **⚠ REVISION (round 4 — FINAL; this is the CORE perf requirement). Read before
> Step 1.** Prior rounds fixed the data race and the regression test, but the
> cache did NOT actually remove the scan from the hot path: `loadContextCache`
> called `computeUpfront`/`computeActiveSkills` UNCONDITIONALLY at the top of
> every call, then `GetOrSet` discarded the result on a warm cache — so the full
> `nodes.ListByTypeStatus` scan + hydrate STILL ran every turn, defeating the
> entire plan. Fix it:
> 1. `loadContextCache` MUST begin with a `GetOk` fast-path that returns WITHOUT
>    computing on a warm cache, and compute only on a miss:
>    `if raw, ok := app.Store().GetOk(contextCacheKey); ok { if c, ok := raw.(*contextCache); ok && c != nil { return c, nil } }`
>    then, ONLY on a miss, compute upfront + skills and store via `GetOrSet` on
>    FULL success (keep the existing "never store a partial/empty snapshot on a
>    compute error — return the fresh value WITHOUT storing" guard). Do NOT call
>    `computeUpfront`/`computeActiveSkills` before the `GetOk` check.
> 2. Add a DETERMINISTIC warm-cache assertion that the scan does NOT re-run on a
>    warm read: seed active memories, warm the cache (one `UpfrontMemories` call),
>    mutate a memory DIRECTLY via `app.Save` (bypassing the `Transition`/`UpdateFields`
>    invalidators — a raw update does not fire the delete-only hook), and assert a
>    second `UpfrontMemories` does NOT reflect the change (it served the snapshot,
>    not a fresh scan); then call the invalidation path and assert it now does.
> KEEP everything the prior round got right (the `copyForRead` race fix +
> `TestCacheReturnedRecordIsThrowaway`, the delete-only invalidation hook, the
> Memory/Skill invalidation in `Transition`/`UpdateFields`, the
> `computeUpfront`/`computeActiveSkills` parity extraction). Done criterion + STOP
> condition: if a warm turn still calls `nodes.ListByTypeStatus`, the plan is NOT
> done. **This is the final revision round** — if the warm-cache criterion still
> fails, STOP and report rather than improvising. Base: executes against
> `origin/main` (`3d4963c`).

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: none
- **Category**: perf
- **Planned at**: commit `12a48bf`, 2026-06-24

## Why this matters

Every chat turn runs `knowledge.BuildContext` on the latency-critical path
before the model call (`internal/turn/turn.go:96`). `BuildContext` calls
`UpfrontMemories` and `ActiveSkills`, and each of those issues a
`nodes.ListByTypeStatus` — a `type + status` filter scan over the whole `nodes`
collection — followed by per-record property hydration (`hydrateAll`). For an
owner with hundreds of memories that is a full-collection read plus per-record
prop deserialization on every single turn, before any token is generated.

These two sets change rarely (only when the owner approves/edits/archives a
memory or skill), but they are recomputed on every turn. Caching the
upfront-memory set and the active-skill set in `app.Store()` — and dropping the
cache only when an active memory/skill set actually changes — removes the scan
from the hot path while keeping the injected content correct. Correctness of
invalidation is non-negotiable: a stale cache that keeps or drops an injected
memory would make Balaur silently remember something the owner deleted (or
forget something they kept), eroding the consent and honesty pillars.

This plan deliberately does NOT change WHAT gets injected or the selection
semantics (importance ≥ 4, the limits, the name ordering). It only changes
WHERE the already-correct result comes from on a warm turn.

## Current state

### The two hot-path functions (the scan to cache)

`internal/knowledge/knowledge.go:568-602` — both call `nodes.ListByTypeStatus`
then `hydrateAll` on every invocation:

```go
568	// UpfrontMemories returns the highest-importance active memories that are always
569	// injected into context (tier 1 of the injection policy).
570	func UpfrontMemories(app core.App, limit int) ([]*core.Record, error) {
571		recs, err := nodes.ListByTypeStatus(app, string(Memory), StatusActive)
572		if err != nil {
573			return nil, err
574		}
575		hydrateAll(Memory, recs)
576		var out []*core.Record
577		for _, r := range recs {
578			if r.GetInt("importance") >= 4 {
579			out = append(out, r)
580			}
581		}
582		sort.SliceStable(out, func(i, j int) bool {
583			return out[i].GetInt("importance") > out[j].GetInt("importance")
584		})
585		if limit > 0 && len(out) > limit {
586			out = out[:limit]
587		}
588		return out, nil
589	}
590
591	// ActiveSkills returns active skills for the context index, ordered by name.
592	func ActiveSkills(app core.App) ([]*core.Record, error) {
593		recs, err := nodes.ListByTypeStatus(app, string(Skill), StatusActive)
594		if err != nil {
595			return nil, err
596		}
597		hydrateAll(Skill, recs)
598		sort.SliceStable(recs, func(i, j int) bool {
599			return recs[i].GetString("title") < recs[j].GetString("title")
600		})
601		return recs, nil
602	}
```

> NOTE: the indentation of lines 578-580 is reproduced as the file presents it
> in the line-numbered view; when you Read the file yourself, match the real
> file's gofmt indentation, not this excerpt's.

`nodes.ListByTypeStatus` (`internal/nodes/nodes.go:162-167`) is the scan:

```go
162	// ListByTypeStatus returns nodes of one type in one status, newest first.
163	func ListByTypeStatus(app core.App, typ, status string) ([]*core.Record, error) {
164		return app.FindRecordsByFilter("nodes",
165			"type = {:t} && status = {:s}", "-created", 0, 0,
166			dbx.Params{"t": typ, "s": status})
167	}
```

### The caller on the hot path

`internal/knowledge/context.go:32-74` — `BuildContext` calls `UpfrontMemories`
and `ActiveSkills` (and `SearchActive`, which is OUT OF SCOPE — leave it alone):

```go
32	func BuildContext(app core.App, userMessage string) (string, []*core.Record) {
...
37		upfront, err := UpfrontMemories(app, upfrontLimit)
...
64		skills, err := ActiveSkills(app)
65		if err == nil && len(skills) > 0 {
66			b.WriteString("\n## Skills you know (load with the `skill` tool before using)\n")
67			for _, s := range skills {
68				fmt.Fprintf(&b, "- %s — %s\n", s.GetString("name"), firstNonEmpty(
69					s.GetString("when_to_use"), s.GetString("description")))
70			}
71		}
72
73		return b.String(), used
74	}
```

`internal/turn/turn.go:96` runs it on every turn:

```go
96		knowledgeBlock, usedMemories := knowledge.BuildContext(app, userText)
```

### The Store cache pattern to mirror

`SearchActive` already reads a singleton out of `app.Store()` by a package-level
key (`internal/knowledge/knowledge.go:443-446`):

```go
443	func SearchActive(app core.App, terms []string, limit int) ([]*core.Record, error) {
444		// --- FTS5 fast path ---
445		if raw, ok := app.Store().GetOk(search.StoreKey); ok {
446			if ix, ok := raw.(*search.Index); ok && ix != nil {
```

`search.StoreKey` is a package const (`internal/search/index.go:20-22`):

```go
20	// StoreKey is the app.Store() key under which the *Index singleton lives.
21	// Read by internal/knowledge to consult the index without importing main.
22	const StoreKey = "balaur.searchIndex"
```

The **atomic Store primitive** to use is `app.Store().GetOrSet(key, setFunc)` —
it runs `setFunc` only when the key is absent, under the store's write lock, so
exactly one concurrent caller computes the value. The repo's exemplar is
`internal/web/models.go:180-187`:

```go
180	func claimInFlight(app core.App, key string, cancel context.CancelFunc) bool {
181		won := false
182		app.Store().GetOrSet(key, func() any {
183			won = true
184			return cancel
185		})
186		return won
187	}
```

To invalidate, **remove the key** (not Set-nil) — `internal/web/models.go:215`:
`h.app.Store().Remove(downloadStoreKey)`. AGENTS.md forbids the
`GetOk`+`Set` read-modify-write pattern for check-then-act; use `GetOrSet` to
populate and `Remove` to drop.

### THE INVALIDATION SEAM — the central design decision

The search FTS index keeps itself current via record hooks bound in
`main.go:registerSearchIndex` (`main.go:239-263`), NOT inside the search
package:

```go
239		upsertHook := func(e *core.RecordEvent) error {
...
247			return e.Next()
248		}
...
260		app.OnRecordAfterCreateSuccess("nodes").BindFunc(upsertHook)
261		app.OnRecordAfterUpdateSuccess("nodes").BindFunc(upsertHook)
262		app.OnRecordAfterDeleteSuccess("nodes").BindFunc(deleteHook)
```

**Do NOT piggyback this blanket Update hook for the cache.** Here is why, and
it is the load-bearing reason this plan exists. `internal/turn/turn.go:154-157`
runs, after every turn, for each memory that informed the reply:

```go
154		// Memories that informed this turn count as used.
155		for _, m := range usedMemories {
156			knowledge.Touch(app, knowledge.Memory, m)
157		}
```

`Touch` (`internal/knowledge/knowledge.go:380-390`) calls `app.Save(rec)` to
bump `use_count`/`last_used`:

```go
380	func Touch(app core.App, kind Kind, rec *core.Record) {
381		props := nodes.Props(rec)
382		props["use_count"] = nodes.PropInt(rec, "use_count") + 1
383		props["last_used"] = time.Now().UTC().Format(time.RFC3339)
384		rec.Set("props", props)
385		if err := app.Save(rec); err != nil {
...
```

That `app.Save` fires `OnRecordAfterUpdateSuccess("nodes")` **on every turn**.
If the cache invalidated on that hook, it would be dropped every turn — the
warm-cache path would never exist and the DONE criterion "no longer
ListByTypeStatus-scans per turn" would be impossible to meet. Worse, `Touch`
does NOT change the cached set: `UpfrontMemories` selects on `importance >= 4`
and sorts by importance then (for skills) title — none of which is `use_count`
or `last_used`. So a `Touch` is a *no-op for the cached set's membership and
order* and MUST NOT invalidate it.

**The seam this plan uses instead**: invalidate from the knowledge package's own
active-set-mutating functions, the only owner-initiated paths that change which
nodes are active or change a cached field (importance/title/body/status). These
are:

- `Transition` (`internal/knowledge/knowledge.go` — and the underlying
  `nodes.Transition` at `internal/nodes/nodes.go:169`): the only path that makes
  a node active or moves it out of active. **Confirm the exact `Transition`
  signature in the knowledge package by reading it** — callers use
  `Transition(app, kind, id, to)` (see `ApplyEdit` at
  `internal/knowledge/knowledge.go:333`: `Transition(app, kind, id, StatusArchived)`).
- `UpdateFields` (knowledge package): edits title/body/importance on an active
  node (changes the rendered upfront line or the importance≥4 membership).
- `ApplyEdit` (`internal/knowledge/knowledge.go:321`): routes to `Transition`
  (archive) or `UpdateFields` — so if those two invalidate, `ApplyEdit` is
  covered transitively; verify it adds no separate active-set write.
- A hard delete of a memory/skill node via `nodes.Drop`
  (`internal/nodes/nodes.go:149-159`). `nodes.Drop` lives in the `nodes`
  package, which knowledge imports — but invalidation logic belongs to
  knowledge. See Step 4 for how to cover the delete path without putting a
  knowledge dependency into `nodes`.

`Touch` and `ProposeMemory`/`ProposeSkill`/`ProposeEdit` deliberately do NOT
invalidate: proposals create `status=proposed` nodes (never in the active set),
and `ProposeEdit` parks props on an active node without changing any cached
field. Leaving them out is correct AND is what preserves the warm-cache win.

### Repo conventions to follow (with exemplars)

- **gofmt is law** (a PostToolUse hook + CI gofmt gate enforce it). Run
  `gofmt -l .` — it must print nothing.
- **Errors are values**: wrap with `fmt.Errorf("doing x: %w", err)`, return
  early, no panics. Exemplar: `nodes.ListByTypeStatus` above.
- **Structured logging only** via `app.Logger()` (`*slog.Logger`), key/value
  pairs — exemplar `main.go:243` `app.Logger().Warn("search: upsert failed", "id", e.Record.Id, "err", err)`. You will not need logging for this change.
- **Records ARE the domain model; knowledge owns its own PB reads/writes.**
  The cache and its invalidation live in `internal/knowledge`. Do NOT route
  through `internal/store` (that is cross-cutting only).
- **Atomic Store primitive**: `GetOrSet` to populate, `Remove` to invalidate —
  never `GetOk`+`Set`. Exemplar `internal/web/models.go:180-187,215`.
- **Tests**: standard `testing`, table-driven where it helps, NO assertion
  framework, NO `time.Sleep`. Use `storetest.NewApp(t)` for a PocketBase-backed
  app (it boots the full migration chain). Exemplar test file:
  `internal/knowledge/knowledge_test.go` (e.g. `TestBuildContext` at line 281,
  `TestSearchActiveFallbackNoIndex` at line 443).
- **No migration in this plan** (see Scope). The cache lives only in
  `app.Store()` (process memory), so it does not survive restart and needs no
  schema.

## Commands you will need

| Purpose           | Command                                   | Expected on success            |
|-------------------|-------------------------------------------|--------------------------------|
| Build (CGO-free)  | `CGO_ENABLED=0 go build ./...`            | exit 0, no output              |
| Test (this pkg)   | `go test ./internal/knowledge/`           | `ok`, all pass                 |
| Test (turn pkg)   | `go test ./internal/turn/`                | `ok`, all pass                 |
| Test (all)        | `go test ./...`                           | all packages `ok`              |
| Vet               | `go vet ./...`                            | exit 0, no output              |
| gofmt check       | `gofmt -l .`                              | prints nothing                 |
| Diff check        | `git diff --check`                        | prints nothing                 |
| Graph refresh     | `graphify update .`                       | regenerates graph, exit 0      |

Note on the test machine: if `go test ./...` fails to LINK with "No space left
on device", set `TMPDIR=/home/alex/.cache/go-tmp` (the repo's `/tmp` is a small
tmpfs). Per-package tests (`go test ./internal/knowledge/`) generally link fine
without it.

## Suggested executor toolkit

- Invoke the `go-standards` skill before writing Go here — for the `GetOrSet`
  atomic-store rule, the slog/error-wrapping idioms, and the table-driven /
  fake-`llm.Client` / `storetest.NewApp(t)` testing conventions.
- Before reading or grepping any source file, this repo MANDATES orienting with
  graphify first: `graphify query "<question>"`, `graphify explain "<concept>"`,
  or `graphify path "<A>" "<B>"`. Only read raw files after graphify orients you,
  or to confirm specific lines.

## Scope

**In scope** (the only files you should modify):

- `internal/knowledge/knowledge.go` — add the cache helpers + cached
  `UpfrontMemories`/`ActiveSkills`; call the invalidator from the active-set
  mutators.
- A new file `internal/knowledge/cache.go` (create) — the cache key, the
  cached-set struct/loader, and `invalidateContextCache(app)`. Keeping the cache
  in its own file keeps `knowledge.go` from growing past the ~500-line smell
  threshold; if you prefer, you MAY instead put these in `knowledge.go` — but a
  new `cache.go` is cleaner. Pick one; do not duplicate.
- `internal/knowledge/cache_test.go` (create) — invalidation + parity tests.
- `internal/knowledge/context.go` — only if `BuildContext` needs to call the
  cached entry points by a new name; if you keep the names `UpfrontMemories` and
  `ActiveSkills`, this file needs no edit. Prefer keeping the names.

**Out of scope** (do NOT touch, even though they look related):

- `main.go` and the `OnRecordAfter*` hooks in `registerSearchIndex` — the
  blanket Update hook fires on `Touch` every turn (see "THE INVALIDATION SEAM").
  Do not bind the cache to it.
- `internal/search/*` — the FTS index is a separate concern.
- `internal/turn/turn.go` — `Touch` must keep working unchanged; the cache must
  tolerate `Touch`'s `app.Save` without invalidating.
- `SearchActive` / `SearchAllActive` (`internal/knowledge/knowledge.go:443,510`)
  — the tier-2 recall path is NOT cached (it is query-dependent per turn).
- The selection semantics: importance ≥ 4, `upfrontLimit`/`recallLimit` (12/6 in
  `context.go:24-27`), the importance-desc and title-asc orderings. The cached
  result must be byte-identical to the uncached result.
- **A `nodes` schema migration / a new indexed `importance` column.** This is
  the heavier alternative this plan explicitly does NOT take (it needs an
  additive migration > `1750000080` and an `ORDER BY importance DESC LIMIT`
  rewrite). Record it as deferred (see Maintenance notes); do not build it.

## Git workflow

- Land-on-`main` repo (no PR gate). Executors typically run in a worktree off
  `origin/main`. Branch name if your harness needs one: `advisor/183-cache-upfront-context`.
- Conventional-commit subject, e.g.
  `perf(knowledge): cache upfront-memory + active-skill sets off the chat hot path`.
- Commit when the suite is green. Push ONLY if the operator told you to; gate any
  push on a green full `go test ./...`.

## Steps

### Step 1: Define the cached-set value and its Store key

Create `internal/knowledge/cache.go` with:

- A package const for the Store key, mirroring `search.StoreKey`'s style, e.g.
  `const contextCacheKey = "balaur.knowledge.contextCache"`.
- A small unexported struct holding both cached slices, e.g.
  ```go
  type contextCache struct {
      upfront []*core.Record // result of the uncached UpfrontMemories(app, upfrontLimit)
      skills  []*core.Record // result of the uncached ActiveSkills(app)
  }
  ```
  Cache the FULL upfront list at `upfrontLimit` (the value `BuildContext` asks
  for). `UpfrontMemories(app, limit)` callers pass `upfrontLimit` (12) on the hot
  path; if a caller passes a different `limit`, slice the cached `upfront` to
  that `limit` after reading from the cache (the cached slice is already sorted,
  so `out[:limit]` is correct). The non-hot-path callers in tests pass `10` —
  the slice-to-limit step keeps them correct.

Keep `core` imported as it already is in the package.

**Verify**: `CGO_ENABLED=0 go build ./internal/knowledge/` → exit 0 (the file
compiles even before it is wired in; unused symbols are fine at this point only
if referenced — if Go complains about unused, proceed to Step 2 which uses them).

### Step 2: Add the cache loader and the invalidator

In `cache.go`, add:

- `func loadContextCache(app core.App) *contextCache` — uses
  `app.Store().GetOrSet(contextCacheKey, func() any { ... })` to compute the
  cache exactly once when absent. Inside the `setFunc`, compute the two sets by
  the **existing uncached logic** (extract the current bodies of
  `UpfrontMemories`/`ActiveSkills` into private helpers, e.g.
  `computeUpfront(app, upfrontLimit)` and `computeActiveSkills(app)`, each doing
  the `nodes.ListByTypeStatus` + `hydrateAll` + filter/sort exactly as today).
  Return `*contextCache`. Because `GetOrSet`'s value type is `any`, type-assert
  the result back to `*contextCache`.
  - If either compute returns an error, do NOT cache a partial/empty value that
    could be mistaken for "no memories". Have `loadContextCache` return the
    computed cache only on success; on error, fall back to computing directly
    and returning a result WITHOUT storing it (so the next call retries). The
    simplest correct shape: compute both sets up front (handling errors), and
    only call `GetOrSet` to store when both succeed. Document this in a comment:
    a stored cache must always be a complete, correct snapshot.
- `func invalidateContextCache(app core.App)` — `app.Store().Remove(contextCacheKey)`.
  One line plus a doc comment explaining it is called after any active
  memory/skill set change (status/importance/title/body change or a
  delete), and deliberately NOT from `Touch` (use_count/last_used do not affect
  the cached set).

**Verify**: `CGO_ENABLED=0 go build ./internal/knowledge/` → exit 0.

### Step 3: Make `UpfrontMemories` and `ActiveSkills` read through the cache

Rewrite the two functions in `knowledge.go` to read from `loadContextCache`:

- `UpfrontMemories(app, limit)`: `c := loadContextCache(app)`; take `c.upfront`,
  and if `limit > 0 && len(c.upfront) > limit`, return `c.upfront[:limit]`, else
  return `c.upfront`. Preserve the `(... , error)` signature; return a nil error
  on the cache path. (If `loadContextCache` had to fall back due to a compute
  error, surface that error — see Step 2.)
- `ActiveSkills(app)`: `return loadContextCache(app).skills, nil`.
- Move the original scan/sort/filter bodies into the `computeUpfront` /
  `computeActiveSkills` helpers used by `loadContextCache` (Step 2). Do NOT
  duplicate the sort/filter logic — one source of truth (suckless rule).

The cached `*core.Record` values are shared across turns. `BuildContext` only
READS fields off them (`GetString`, `GetInt`) — it does not mutate. `Touch`
mutates a SEPARATE record instance loaded by `SearchActive`/recall, not these
cached pointers — but to be safe, confirm no in-scope code path writes to the
records returned by `UpfrontMemories`/`ActiveSkills`. If any does, that is a
STOP condition (a shared cached record must be treated read-only).

**Verify**: `go test ./internal/knowledge/` → all existing tests still pass
(this proves parity: `TestBuildContext`, `TestProposeAndApproveMemory`,
`TestFilterActive`, etc. exercise these functions).

### Step 4: Invalidate from every active-set mutation

Add a call to `invalidateContextCache(app)` AFTER the successful write in each
knowledge-package function that can change the active memory/skill set or a
cached field. Read each function before editing and place the call after the
`app.Save`/transition succeeds (never before — mirror the audit-after-write
rule):

1. `Transition` — in the knowledge package. **Read its real signature and body
   first.** It wraps `nodes.Transition`. Invalidate after a SUCCESSFUL
   transition, but ONLY when the kind is `Memory` or `Skill` AND the transition
   touches the active set (from-active or to-active). The simplest correct rule
   that is still cheap: invalidate whenever `to == StatusActive` OR the node's
   prior status was `StatusActive`. If determining the prior status is awkward,
   invalidating on ANY successful memory/skill `Transition` is acceptable
   (transitions are rare, owner-initiated).
2. `UpdateFields` — after a successful field edit on a `Memory`/`Skill` node
   whose status is `active` (an edit to a proposed node does not affect the
   cache, but invalidating anyway is harmless and rare; prefer the simple
   "invalidate after any successful UpdateFields on Memory/Skill").
3. `ApplyEdit` — verify it only calls `Transition`/`UpdateFields` (it does, per
   `knowledge.go:331-340`). If so, it is covered transitively; add NO extra
   invalidation. If you find a direct `app.Save` in it, invalidate there too.
4. The hard-delete path. `nodes.Drop` (`internal/nodes/nodes.go:149`) deletes
   any node and is called from the tools/CLI layer for memory/skill drops. The
   `nodes` package must NOT import `knowledge` (would be a dependency cycle —
   `knowledge` imports `nodes`). Cover deletes one of these two ways, in order
   of preference:
   - **(a) Bind a delete hook from a knowledge-package registration function.**
     Add `func RegisterCacheInvalidation(app core.App)` to `cache.go` that binds
     `app.OnRecordAfterDeleteSuccess("nodes").BindFunc(...)`; in the hook, if
     `e.Record.GetString("type")` is `"memory"` or `"skill"`, call
     `invalidateContextCache(app)`, then `return e.Next()`. Call
     `RegisterCacheInvalidation(app)` from `main.go` next to
     `registerSearchIndex(app)`. **BUT** binding a hook means editing `main.go`,
     which is out of scope above — so if you take (a), the one-line `main.go`
     wiring is the ONLY permitted `main.go` edit, and you MUST also bind it in
     tests (see Test plan) because `storetest.NewApp(t)` does not run `main.go`.
     A delete-only hook does NOT fire on `Touch` (Touch is an Update), so it is
     safe.
   - **(b) Invalidate at the knowledge/tool call sites that drop memory/skill
     nodes.** Find the callers of `nodes.Drop` for memory/skill (grep after a
     graphify orientation) and add `invalidateContextCache(app)` after a
     successful drop. This avoids touching `main.go` but may miss a future
     caller.

   **Recommendation**: take **(a)** — a delete hook is the single reliable seam
   for deletes and cannot be bypassed by a new caller, and it is provably
   Touch-safe because it is delete-only. Treat the one-line `main.go` wiring +
   the in-test binding as part of this plan. Document the choice in the commit.

**Verify**: `go test ./internal/knowledge/` and `go test ./internal/turn/` →
all pass.

### Step 5: Write the cache parity + invalidation tests

See Test plan for the exact cases. Create
`internal/knowledge/cache_test.go`.

**Verify**: `go test ./internal/knowledge/ -run Cache -v` → the new tests pass.

### Step 6: Full gates + graph refresh

Run the full suite and the lint gates.

**Verify**:
- `go vet ./...` → exit 0
- `gofmt -l .` → prints nothing
- `git diff --check` → prints nothing
- `go test ./...` → all packages `ok` (use `TMPDIR=/home/alex/.cache/go-tmp`
  if the link step reports "No space left on device")
- `CGO_ENABLED=0 go build ./...` → exit 0
- `graphify update .` → exit 0 (keeps the AST graph current; no API cost)

## Test plan

New file `internal/knowledge/cache_test.go`, modeled structurally on
`internal/knowledge/knowledge_test.go` (same imports: `storetest`, `nodes`,
`core`; same `storetest.NewApp(t)` setup; NO assertion framework; NO
`time.Sleep`). Note the real `Transition` signature is
`Transition(app, kind, id, status)` — e.g.
`Transition(app, Memory, rec.Id, StatusActive)` (see existing tests at
`knowledge_test.go:51,286,410`).

Cases (each its own `Test...` func or a subtest):

1. **Parity — cache returns the identical set to the uncached path.** Seed
   several active memories (mixed importance, some ≥ 4 some < 4) and active
   skills via `ProposeMemory`/`ProposeSkill` + `Transition(... , StatusActive)`.
   Capture `want := computeUpfront(app, upfrontLimit)` is NOT exported — so
   instead assert the cached `UpfrontMemories(app, upfrontLimit)` returns exactly
   the expected ids in the expected order (importance desc; for skills, title
   asc), matching what the old logic would produce. Assert
   `ActiveSkills(app)` returns skills in title order. Call each twice and assert
   the second call returns an equal-length, same-id, same-order slice (a warm
   read).

2. **Add-then-read shows the new memory (no stale omission).** Build the cache
   (call `UpfrontMemories`), then approve a NEW importance-5 memory
   (`ProposeMemory` + `Transition` to active). Assert the next
   `UpfrontMemories(app, upfrontLimit)` INCLUDES the new memory's id. This proves
   `Transition`-to-active invalidates.

3. **Drop-then-read hides the memory (no stale retention — the consent
   guarantee).** Approve an importance-5 memory, build the cache (read it, see
   the memory), then delete the node. Use the SAME delete path your Step-4
   choice covers:
   - If you took (a) the delete hook: in the test, after `storetest.NewApp(t)`,
     call `RegisterCacheInvalidation(app)` so the hook is bound (the test app
     does not run `main.go`). Then delete via `nodes.Drop(app, id)`.
   - If you took (b) call-site invalidation: delete via the same
     knowledge/tool function you instrumented.
   Assert the next `UpfrontMemories(app, upfrontLimit)` does NOT contain the
   dropped id. THIS IS THE NON-NEGOTIABLE CORRECTNESS TEST — a failure here means
   a deleted memory is still being injected.

4. **Archive-then-read hides the memory.** Approve an importance-5 memory, build
   the cache, then `Transition(app, Memory, id, StatusArchived)`. Assert the next
   `UpfrontMemories` excludes it (status moved out of active).

5. **Edit-then-read reflects the change.** Approve an importance-5 memory, build
   the cache, then `UpdateFields(app, Memory, id, map[string]string{"importance": "2"})`.
   Assert the next `UpfrontMemories(app, upfrontLimit)` EXCLUDES it (dropped
   below the ≥ 4 threshold). This proves an importance edit invalidates.

6. **Skills invalidate too.** Approve a skill, build the cache (see it in
   `ActiveSkills`), archive it via `Transition(app, Skill, id, StatusArchived)`,
   assert the next `ActiveSkills` excludes it.

7. **`Touch` does NOT invalidate (the warm-cache guarantee).** Approve an
   importance-5 memory, read `UpfrontMemories` to warm the cache, capture the
   returned record pointer's identity is irrelevant — instead, prove the cache
   was NOT recomputed after a `Touch`. The cleanest deterministic assertion
   WITHOUT timing: after warming, call
   `knowledge.Touch(app, knowledge.Memory, rec)` (which `app.Save`s the record),
   then assert `app.Store().GetOk(contextCacheKey)` STILL returns `ok == true`
   (the cache key is still present — `Touch` did not Remove it). Because
   `contextCacheKey` is unexported and the test is in `package knowledge`, it can
   reference it directly. This is the regression guard that the cache survives
   the per-turn `Touch`.

Verification: `go test ./internal/knowledge/` → all pass, including the ~7 new
cases. `go test ./internal/turn/` → all pass (Touch path unaffected).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go test ./internal/knowledge/` passes, including the new
      `cache_test.go` parity + invalidation tests (add/drop/archive/edit for
      memories, archive for skills, and the `Touch`-does-not-invalidate guard).
- [ ] `go test ./internal/turn/` passes (the per-turn `Touch` path is
      unaffected).
- [ ] `go test ./...` passes (use `TMPDIR=/home/alex/.cache/go-tmp` if linking
      runs out of space).
- [ ] `go vet ./...` exits 0; `gofmt -l .` prints nothing; `git diff --check`
      prints nothing.
- [ ] On a warm turn, `UpfrontMemories` and `ActiveSkills` do NOT call
      `nodes.ListByTypeStatus` — confirm by reading the final code: the scan is
      only reachable through the `loadContextCache` `setFunc`, which runs only
      when `contextCacheKey` is absent. The test in case 7 proves the key
      survives a `Touch`.
- [ ] Only in-scope files changed (`git status`): the new `internal/knowledge/cache.go`,
      `internal/knowledge/cache_test.go`, edits to `internal/knowledge/knowledge.go`,
      and — only if you took Step-4 option (a) — a single one-line wiring edit in
      `main.go`. No other files.
- [ ] `internal/self/knowledge.md` reviewed: this change is an internal
      perf optimization with no new capability or architecture surface, so it
      likely needs NO edit. If you added the `RegisterCacheInvalidation` boot
      step and judge it architecturally notable, add one line; otherwise leave it.
- [ ] `plans/README.md` status row for plan 183 updated (unless a reviewer told
      you they maintain the index).

## STOP conditions

Stop and report back (do not improvise) if:

- **No reliable, Touch-safe invalidation seam exists.** If you cannot cover the
  delete path with either a delete-only hook (option a) or call-site
  invalidation (option b) — or if the only available hook also fires on `Touch`
  (the per-turn `app.Save` at `internal/turn/turn.go:156`) so the cache would
  invalidate every turn — STOP. A stale memory cache silently keeps or drops
  injected memories, eroding the consent + honesty pillars. Correctness of
  invalidation is non-negotiable; a cache that cannot be invalidated reliably is
  worse than no cache, so do NOT ship a best-effort version.
- The live code at the "Current state" line numbers does not match the excerpts
  (drift since `12a48bf`) — re-derive against the real code and report the diff
  before proceeding.
- `Transition`'s real signature in the knowledge package differs from
  `Transition(app, kind, id, status)` in a way that changes how you call the
  invalidator — report it.
- You discover an in-scope code path that WRITES to the records returned by
  `UpfrontMemories`/`ActiveSkills` (the cached, shared pointers). A shared cached
  record must be read-only; report it rather than papering over with a deep copy.
- A step's verification fails twice after a reasonable fix attempt.
- The fix appears to require touching any out-of-scope file beyond the single
  permitted one-line `main.go` wiring (Step-4 option a).

## Maintenance notes

For the human/agent who owns this code after the change lands:

- **The cache is process-memory only** (`app.Store()`), so it is empty after a
  restart and rebuilt lazily on the first turn. It never persists, so there is
  no migration and no on-disk staleness to worry about.
- **Invalidation is the whole correctness story.** Any NEW code path that
  activates, archives, edits (title/body/importance), or deletes a memory/skill
  node MUST invalidate the context cache (`invalidateContextCache(app)` or, for
  deletes, be covered by the delete hook). If a future feature writes active
  memory/skill nodes outside the knowledge package's `Transition`/`UpdateFields`
  (or bypasses `nodes.Drop`), the cache will go stale. A reviewer should
  scrutinize exactly this: "does every new active-set mutation invalidate?"
- **`Touch` is intentionally NOT an invalidator.** It bumps `use_count` /
  `last_used`, neither of which is in the cached selection or ordering. If a
  future change makes `UpfrontMemories`/`ActiveSkills` order by recency/use,
  then `Touch` WOULD need to invalidate — revisit case 7's guard.
- **Deferred alternative (explicitly not taken here):** promote `importance` to
  a real indexed `nodes` column and let SQL do `WHERE type='memory' AND
  status='active' AND importance>=4 ORDER BY importance DESC LIMIT 12`. That
  removes the in-Go filter/sort and the full scan entirely, but it needs an
  additive migration (new file, unique 10-digit prefix > `1750000080`,
  registered via `m.Register(up, down)`; never edit the consolidated baseline
  `migrations/1749600000_init.go`) plus a backfill and a rewrite of the read
  path. It is the heavier, higher-value follow-up; this plan ships the low-risk
  cache first.
- A reviewer should also confirm the cached result is byte-identical to the old
  uncached `BuildContext` output (the parity test) — the injection content and
  ordering must not change.
