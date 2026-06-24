# Plan 188: Collapse the duplicated FTS5+fallback skeleton in SearchActive / SearchAllActive

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 12a48bf..HEAD -- internal/knowledge/knowledge.go internal/knowledge/knowledge_test.go`
> If either in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.
>
> **This plan is OPPORTUNISTIC.** It is worth doing ONLY if the shared helper
> is genuinely simpler than the two duplicated copies. If extracting the helper
> makes the code uglier than the duplication it removes (see STOP conditions),
> STOP, revert, and report "not worth it". Do not force it.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `12a48bf`, 2026-06-24

## Why this matters

`SearchActive` and `SearchAllActive` in `internal/knowledge/knowledge.go` are
two ~55-line functions that share the identical control skeleton: pull the FTS5
index from `app.Store()`, run an FTS query, hydrate the returned ids from the
`nodes` collection, drop anything that fails the **consent filter**
(`status=active`), rank the survivors by the FTS id order, cap to `limit`,
and — when the index is absent or returns nothing — fall back to a deterministic
substring scan. They differ on only four axes (the FTS query method, the keep
predicate, whether records are hydrated, and the fallback source). A change to
ranking or to the consent filter must today be made twice and can silently
drift between the two. Centralizing the skeleton into one helper means there is
one place that enforces `status=active` and one place that enforces id-order
ranking. This is pure tech-debt cleanup — **behavior must stay byte-identical**;
the existing tests are the contract.

## Current state

Files:

- `internal/knowledge/knowledge.go` — the memory/skill layer over the `nodes`
  spine. `SearchActive` (memory-scoped recall) lives at lines 443–501;
  `SearchAllActive` (cross-type search) at lines 510–566. Both are in scope.
- `internal/knowledge/knowledge_test.go` — the contract. `TestSearchActive*`
  and `TestSearchAllActiveCrossType` cover both functions across the FTS-path
  and the no-index fallback path. Must pass UNCHANGED.

Package context (`internal/knowledge/knowledge.go:18-51`) — these are the
symbols the helper will use; do NOT re-import or redeclare them:

```go
package knowledge

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/nodes"
	"github.com/alexradunet/balaur/internal/search"
	"github.com/alexradunet/balaur/internal/store"
)

// Kind is the node TYPE a lifecycle call operates on.
type Kind string

const (
	Memory Kind = "memory"
	Skill  Kind = "skill"
)
```

`StatusActive` is `nodes.StatusActive` (`knowledge.go:48`). `hydrate`,
`hydrateAll`, and `matchesQuery` are existing unexported helpers in the same
file (`hydrate` at line 72, `hydrateAll` at line 100, `matchesQuery` at line
427). The FTS index API: `search.StoreKey` (a string const), `(*search.Index).
QueryKind(terms []string, kind string, limit int) ([]string, error)` and
`(*search.Index).Query(terms []string, limit int) ([]string, error)` — both
return ranked node ids.

**`SearchActive` — verbatim, `internal/knowledge/knowledge.go:440-501`:**

```go
// SearchActive finds active memories matching any of the given terms. When a
// FTS5 sidecar index is available it is bm25-ranked; otherwise it falls back to
// a deterministic substring scan over the active memory nodes.
func SearchActive(app core.App, terms []string, limit int) ([]*core.Record, error) {
	// --- FTS5 fast path ---
	if raw, ok := app.Store().GetOk(search.StoreKey); ok {
		if ix, ok := raw.(*search.Index); ok && ix != nil {
			ids, err := ix.QueryKind(terms, string(Memory), limit)
			if err == nil && len(ids) > 0 {
				recs, err := app.FindRecordsByIds("nodes", ids)
				if err == nil {
					var active []*core.Record
					for _, r := range recs {
						if r.GetString("type") == string(Memory) && r.GetString("status") == StatusActive {
							active = append(active, hydrate(Memory, r))
						}
					}
					if len(active) > 0 {
						order := make(map[string]int, len(ids))
						for i, id := range ids {
							order[id] = i
						}
						sort.Slice(active, func(i, j int) bool {
							return order[active[i].Id] < order[active[j].Id]
						})
						if len(active) > limit {
							active = active[:limit]
						}
						return active, nil
					}
				}
			}
		}
	}

	// --- substring fallback over active memory nodes ---
	recs, err := nodes.ListByTypeStatus(app, string(Memory), StatusActive)
	if err != nil {
		return nil, err
	}
	hydrateAll(Memory, recs)
	var matched []*core.Record
	for _, r := range recs {
		for _, t := range terms {
			t = strings.ToLower(strings.TrimSpace(t))
			if t == "" {
				continue
			}
			if matchesQuery(Memory, r, t) {
				matched = append(matched, r)
				break
			}
		}
	}
	sort.SliceStable(matched, func(i, j int) bool {
		return matched[i].GetInt("importance") > matched[j].GetInt("importance")
	})
	if limit > 0 && len(matched) > limit {
		matched = matched[:limit]
	}
	return matched, nil
}
```

**`SearchAllActive` — verbatim, `internal/knowledge/knowledge.go:503-566`:**

```go
// SearchAllActive is the cross-type search surface: it returns active nodes of
// ANY type matching the terms, ranked by bm25 when the FTS5 sidecar is
// available. Unlike SearchActive (which stays memory-scoped and hydrates memory
// aliases for context/recall callers), this returns RAW node records — the
// caller renders each hit by its node `type`. A node that is not active is never
// returned (the consent filter). When the index is unavailable it falls back to
// a deterministic substring scan over active nodes' title/body.
func SearchAllActive(app core.App, terms []string, limit int) ([]*core.Record, error) {
	// --- FTS5 fast path ---
	if raw, ok := app.Store().GetOk(search.StoreKey); ok {
		if ix, ok := raw.(*search.Index); ok && ix != nil {
			ids, err := ix.Query(terms, limit)
			if err == nil && len(ids) > 0 {
				recs, err := app.FindRecordsByIds("nodes", ids)
				if err == nil {
					var active []*core.Record
					for _, r := range recs {
						if r.GetString("status") == StatusActive {
							active = append(active, r)
						}
					}
					if len(active) > 0 {
						order := make(map[string]int, len(ids))
						for i, id := range ids {
							order[id] = i
						}
						sort.Slice(active, func(i, j int) bool {
							return order[active[i].Id] < order[active[j].Id]
						})
						if limit > 0 && len(active) > limit {
							active = active[:limit]
						}
						return active, nil
					}
				}
			}
		}
	}

	// --- substring fallback over all active nodes ---
	recs, err := app.FindRecordsByFilter(
		"nodes", "status = 'active'", "-updated,-created", 0, 0, nil)
	if err != nil {
		return nil, err
	}
	var matched []*core.Record
	for _, r := range recs {
		for _, t := range terms {
			t = strings.ToLower(strings.TrimSpace(t))
			if t == "" {
				continue
			}
			if strings.Contains(strings.ToLower(r.GetString("title")), t) ||
				strings.Contains(strings.ToLower(r.GetString("body")), t) {
				matched = append(matched, r)
				break
			}
		}
	}
	if limit > 0 && len(matched) > limit {
		matched = matched[:limit]
	}
	return matched, nil
}
```

### The four difference axes (the ONLY things that may differ)

| Axis | `SearchActive` | `SearchAllActive` |
|------|----------------|-------------------|
| FTS query | `ix.QueryKind(terms, string(Memory), limit)` | `ix.Query(terms, limit)` |
| keep predicate (FTS path) | `type==Memory && status==active`, then `hydrate(Memory, r)` | `status==active`, record kept raw |
| FTS `limit` cap guard | `if len(active) > limit` (no `limit > 0` guard) | `if limit > 0 && len(active) > limit` |
| fallback source + filter | `nodes.ListByTypeStatus(Memory, active)` → `hydrateAll` → `matchesQuery(Memory, …)` → **sort by importance desc** → `if limit > 0 && len > limit` cap | `FindRecordsByFilter("nodes","status='active'","-updated,-created")` → `Contains(title)‖Contains(body)` → **no extra sort** → `if limit > 0 && len > limit` cap |

**Three subtle differences you MUST preserve** (they are the whole reason this
is opportunistic — if a "shared core" can't hold them cleanly, STOP):

1. **FTS-path limit guard differs.** `SearchActive` caps with bare
   `if len(active) > limit`; `SearchAllActive` caps with
   `if limit > 0 && len(active) > limit`. These are NOT the same when
   `limit <= 0`. All real callers pass `limit > 0` (8, 10, 20, or `recallLimit`),
   so the FTS branch behaves identically in practice — but to keep behavior
   byte-identical you must NOT silently "normalize" one to the other inside a
   shared FTS core unless you can prove both call sites pass `limit > 0`. The
   safest path (see Step 1) leaves the FTS skeleton shared but routes the
   per-axis logic through closures; the limit guard `if limit > 0 && len > limit`
   is the SAFER form (it is the more conservative cap) and matches
   `SearchAllActive` — adopt that form in the shared core and confirm via the
   tests that `SearchActive` is unaffected (all its callers pass `limit > 0`).
2. **Fallback ranking differs.** `SearchActive`'s substring fallback sorts
   matched records by `importance` desc (`sort.SliceStable`);
   `SearchAllActive`'s fallback does NOT sort (it relies on the
   `-updated,-created` order from the DB query). Keep this difference — it
   belongs in the per-caller fallback closure, NOT the shared core.
3. **Fallback match predicate differs.** `SearchActive` uses
   `matchesQuery(Memory, r, t)` (checks title/content/when_to_use, lowercased);
   `SearchAllActive` checks only `title`/`body` substring. Keep both.

## Commands you will need

| Purpose            | Command                              | Expected on success        |
|--------------------|--------------------------------------|----------------------------|
| Build (CGO-free)   | `CGO_ENABLED=0 go build ./...`       | exit 0, no output          |
| Test (this pkg)    | `go test ./internal/knowledge/`      | `ok` — all pass            |
| Test (all)         | `go test ./...`                      | all packages `ok`          |
| Vet                | `go vet ./...`                       | exit 0, no output          |
| Format check       | `gofmt -l .`                         | empty output               |
| Whitespace check   | `git diff --check`                   | exit 0, no output          |
| Drift check        | `git diff --stat 12a48bf..HEAD -- internal/knowledge/knowledge.go internal/knowledge/knowledge_test.go` | see drift-check note below |

A `PostToolUse` hook runs `gofmt -w` on every Go file you edit, so formatting
stays clean automatically; still run `gofmt -l .` before finishing as the gate.

## Suggested executor toolkit

- Invoke the `go-standards` skill before writing the helper — it carries the
  repo's Go idioms (error wrapping with `%w`, early return, `slices`/`maps`,
  table-driven tests, no assertion frameworks). The helper here is small, but
  match those conventions.

## Scope

**In scope** (the only files you may modify):

- `internal/knowledge/knowledge.go` — extract the helper, rewrite the two
  callers to delegate.
- `internal/knowledge/knowledge_test.go` — must pass UNCHANGED. Add a NEW test
  case ONLY if Step 3 surfaces a newly-centralized edge that the existing
  tests do not cover (none is expected). Do not rewrite or delete existing
  tests.

**Out of scope** (do NOT touch, even though they look related):

- Ranking or consent semantics — behavior must stay byte-identical. This plan
  does not change what is returned, only where the code lives.
- The FTS index itself (`internal/search/`) — out of scope entirely.
- The callers of these two functions (`internal/cli/knowledge.go`,
  `internal/feature/graphcards/related.go`, `internal/knowledge/context.go`,
  `internal/tools/knowledge.go`). Their signatures do NOT change, so they must
  NOT be edited. If you find yourself needing to edit a caller, the public
  signatures changed — that is a STOP condition.
- `internal/self/knowledge.md` — this is an internal refactor with no
  architecture/capability change, so it does NOT need updating.
- `plans/README.md` — update only your own status row (per executor
  instructions); do not restructure the file.

## Git workflow

- You are likely running in an ephemeral worktree off `origin/main`. If you are
  on `main` directly, create a branch first: `git checkout -b refactor/188-dedup-search-skeleton`.
- One commit is enough for this small change. Conventional-commit subject, e.g.:
  `refactor(knowledge): share one FTS5+fallback skeleton across the two searches (188)`
- Do NOT push or open a PR unless the operator explicitly instructs it. The repo
  lands on `main` with no PR gate, but pushing is gated on a green full suite
  (`go test ./...`) and the operator's go-ahead.

## Steps

### Step 1: Add one unexported helper that holds the shared skeleton

In `internal/knowledge/knowledge.go`, add a single unexported function ABOVE
`SearchActive` (i.e. before line 440, after `matchesQuery` which ends at line
438). Parameterize it by the four difference axes as function values. A clean
shape:

```go
// searchActiveNodes is the shared FTS5-or-fallback skeleton behind SearchActive
// and SearchAllActive. It runs the FTS query, keeps only records that pass the
// `keep` predicate (the consent status=active filter lives there), ranks the
// survivors by the FTS id order, caps to limit, and returns them. When no index
// is present or the FTS path yields nothing, it delegates to `fallback`, which
// owns its own source query, match predicate, ordering, and limit cap.
//
//   - query   returns the FTS-ranked node ids for `terms`.
//   - keep    decides whether an FTS-hydrated record is returned, and may mutate
//     it (e.g. hydrate memory aliases). Return (record, true) to keep.
//   - fallback yields the already-matched, already-ordered, already-capped
//     substring-scan result when the FTS path does not produce hits.
func searchActiveNodes(
	app core.App,
	terms []string,
	limit int,
	query func(ix *search.Index) ([]string, error),
	keep func(r *core.Record) (*core.Record, bool),
	fallback func() ([]*core.Record, error),
) ([]*core.Record, error) {
	// --- FTS5 fast path ---
	if raw, ok := app.Store().GetOk(search.StoreKey); ok {
		if ix, ok := raw.(*search.Index); ok && ix != nil {
			ids, err := query(ix)
			if err == nil && len(ids) > 0 {
				recs, err := app.FindRecordsByIds("nodes", ids)
				if err == nil {
					var active []*core.Record
					for _, r := range recs {
						if kept, ok := keep(r); ok {
							active = append(active, kept)
						}
					}
					if len(active) > 0 {
						order := make(map[string]int, len(ids))
						for i, id := range ids {
							order[id] = i
						}
						sort.Slice(active, func(i, j int) bool {
							return order[active[i].Id] < order[active[j].Id]
						})
						if limit > 0 && len(active) > limit {
							active = active[:limit]
						}
						return active, nil
					}
				}
			}
		}
	}

	// --- fallback (per-caller: source, match predicate, ordering, limit cap) ---
	return fallback()
}
```

Notes for this step:

- The FTS-path limit cap in the shared core uses the SAFER guard
  `if limit > 0 && len(active) > limit` (matching `SearchAllActive`). This is
  behavior-preserving for all real callers because every caller of
  `SearchActive`/`SearchAllActive` passes `limit > 0` (verified: callers pass
  8, 10, 20, and the `recallLimit` const — none passes `<= 0`). If you cannot
  satisfy yourself of that, see STOP conditions.
- The `keep` closure returns `(*core.Record, bool)` so `SearchActive` can both
  filter (`type==Memory && status==active`) AND hydrate in one place, while
  `SearchAllActive` filters (`status==active`) and returns the record raw.
- Do NOT put the substring fallback logic inside the shared core — each caller's
  fallback (source query, match predicate, ordering, limit cap) goes in its own
  `fallback` closure so the importance-sort-vs-no-sort difference stays local.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0 (the helper is unused at
this point, which would fail `staticcheck` U1000 but NOT `go build`; that is
fine because Steps 2–3 wire it in within the same commit. Do not run
`staticcheck` until after Step 3.)

### Step 2: Rewrite `SearchActive` to delegate

Replace the body of `SearchActive` (lines 443–501, keeping the doc comment at
440–442 unchanged) with a call to `searchActiveNodes`, moving its specifics into
the three closures. Target shape:

```go
func SearchActive(app core.App, terms []string, limit int) ([]*core.Record, error) {
	return searchActiveNodes(app, terms, limit,
		func(ix *search.Index) ([]string, error) {
			return ix.QueryKind(terms, string(Memory), limit)
		},
		func(r *core.Record) (*core.Record, bool) {
			if r.GetString("type") == string(Memory) && r.GetString("status") == StatusActive {
				return hydrate(Memory, r), true
			}
			return nil, false
		},
		func() ([]*core.Record, error) {
			recs, err := nodes.ListByTypeStatus(app, string(Memory), StatusActive)
			if err != nil {
				return nil, err
			}
			hydrateAll(Memory, recs)
			var matched []*core.Record
			for _, r := range recs {
				for _, t := range terms {
					t = strings.ToLower(strings.TrimSpace(t))
					if t == "" {
						continue
					}
					if matchesQuery(Memory, r, t) {
						matched = append(matched, r)
						break
					}
				}
			}
			sort.SliceStable(matched, func(i, j int) bool {
				return matched[i].GetInt("importance") > matched[j].GetInt("importance")
			})
			if limit > 0 && len(matched) > limit {
				matched = matched[:limit]
			}
			return matched, nil
		},
	)
}
```

The fallback closure is a verbatim move of the old substring-fallback block
(old lines 475–500), including the `sort.SliceStable` by importance desc — keep
it exactly.

**Verify**: `go test ./internal/knowledge/` → `ok`. The
`TestSearchActive*` tests (`TestSearchActiveOnlyFindsActive`,
`TestSearchActiveFTSPath`, `TestSearchActiveFallbackNoIndex`,
`TestSearchActiveIntegration`, `TestSearchActiveStaysMemoryOnly`) must all pass
unchanged.

### Step 3: Rewrite `SearchAllActive` to delegate

Replace the body of `SearchAllActive` (lines 510–566, keeping the doc comment at
503–509 unchanged) with a call to `searchActiveNodes`. Target shape:

```go
func SearchAllActive(app core.App, terms []string, limit int) ([]*core.Record, error) {
	return searchActiveNodes(app, terms, limit,
		func(ix *search.Index) ([]string, error) {
			return ix.Query(terms, limit)
		},
		func(r *core.Record) (*core.Record, bool) {
			if r.GetString("status") == StatusActive {
				return r, true
			}
			return nil, false
		},
		func() ([]*core.Record, error) {
			recs, err := app.FindRecordsByFilter(
				"nodes", "status = 'active'", "-updated,-created", 0, 0, nil)
			if err != nil {
				return nil, err
			}
			var matched []*core.Record
			for _, r := range recs {
				for _, t := range terms {
					t = strings.ToLower(strings.TrimSpace(t))
					if t == "" {
						continue
					}
					if strings.Contains(strings.ToLower(r.GetString("title")), t) ||
						strings.Contains(strings.ToLower(r.GetString("body")), t) {
						matched = append(matched, r)
						break
					}
				}
			}
			if limit > 0 && len(matched) > limit {
				matched = matched[:limit]
			}
			return matched, nil
		},
	)
}
```

The fallback closure is a verbatim move of the old block (old lines 542–565),
WITHOUT an importance sort — `SearchAllActive` relies on the `-updated,-created`
DB ordering. Do not add a sort.

**Verify**: `go test ./internal/knowledge/` → `ok`. `TestSearchAllActiveCrossType`
must pass unchanged.

### Step 4: Full validation sweep

Run the full gate. The helper is now used by both callers, so dead-code (U1000)
no longer fires.

**Verify** (all must pass):

- `gofmt -l .` → empty output
- `go vet ./...` → exit 0
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go test ./...` → all packages `ok`
- `git diff --check` → exit 0
- `git diff --stat` → only `internal/knowledge/knowledge.go` (and
  `plans/README.md` for your status row) appear changed. If
  `internal/knowledge/knowledge_test.go` shows changes you did not intend, or
  any other file appears, STOP.

## Test plan

- **No new tests are expected.** The existing tests in
  `internal/knowledge/knowledge_test.go` ARE the behavior contract and must pass
  unchanged: `TestSearchActiveOnlyFindsActive`, `TestSearchActiveFTSPath`,
  `TestSearchActiveFallbackNoIndex`, `TestSearchActiveIntegration`,
  `TestSearchActiveStaysMemoryOnly`, `TestSearchAllActiveCrossType`. They
  exercise both the FTS path (a real `search.Open` + `Rebuild` index put into
  `app.Store()`) and the no-index fallback path, for both functions.
- If Step 2 or 3 surfaces a centralized edge the existing tests do not cover
  (NOT expected), add a focused table-driven case modeled on
  `TestSearchAllActiveCrossType` (`internal/knowledge/knowledge_test.go:557`) —
  standard `testing`, no assertion framework, `storetest.NewApp(t)` for the app,
  `search.Open(filepath.Join(t.TempDir(), "search.db"))` for the index. Do not
  fake the model here; these tests do not touch `llm.Client`.
- Verification: `go test ./internal/knowledge/` → `ok`, then `go test ./...`
  → all `ok`.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `gofmt -l .` returns empty output
- [ ] `go vet ./...` exits 0
- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go test ./...` exits 0 with all packages `ok`; the existing
      `TestSearchActive*` and `TestSearchAllActiveCrossType` tests pass UNCHANGED
- [ ] `git diff --check` exits 0
- [ ] `SearchActive` and `SearchAllActive` both delegate to one shared
      `searchActiveNodes` helper; the FTS5→ids→filter→id-order-sort→limit
      skeleton appears exactly once (`grep -c "order := make(map\[string\]int" internal/knowledge/knowledge.go` → `1`)
- [ ] No file outside `internal/knowledge/knowledge.go` (and your
      `plans/README.md` status row) is modified (`git status`)
- [ ] `plans/README.md` status row for plan 188 updated

## STOP conditions

Stop and report back (do not improvise) if:

- **The drift check is non-empty** and the live `SearchActive`/`SearchAllActive`
  bodies differ from the verbatim excerpts in "Current state" — the codebase has
  changed since this plan was written. Re-read both functions; if the four
  difference axes or the three subtle differences no longer hold, do NOT proceed.
- **You cannot confirm every caller passes `limit > 0`.** The shared FTS-path
  cap adopts the `if limit > 0 && …` guard. If you find or suspect a caller that
  passes `limit <= 0` to `SearchActive` (the old code used a bare
  `if len(active) > limit` there), the behavior is NOT byte-identical — STOP and
  report rather than changing observable behavior.
- **The shared core ends up uglier than the duplication.** This plan is
  explicitly opportunistic. If holding the three subtle differences (FTS limit
  guard, importance-sort-vs-none fallback, `matchesQuery`-vs-`title/body`
  predicate) forces awkward flags, nil juggling, or a helper harder to read than
  the two original copies, STOP, `git checkout -- internal/knowledge/knowledge.go`,
  and report "not worth it — leaving the duplication". A failed opportunistic
  refactor is a valid, expected outcome here, not a failure to complete.
- **Any test changes behavior.** If an existing test fails and the only way to
  make it pass is to edit the test's assertions (rather than your new code),
  you have changed behavior — STOP. The tests are the contract.
- **A non-in-scope file needs editing** (e.g. a caller signature). The public
  signatures of `SearchActive`/`SearchAllActive` do not change; if you think they
  must, STOP.
- A step's verification fails twice after a reasonable fix attempt.

## Maintenance notes

For the human/agent who owns this code after the change lands:

- The consent filter (`status == StatusActive`) now lives in two places by
  design: the FTS-path `keep` closure of each caller, and each caller's fallback
  match loop. The shared core does NOT enforce it — it trusts `keep`. If a third
  search surface is added, pass it a `keep` closure that enforces the consent
  filter, and a `fallback` closure that does the same. The id-order ranking and
  the `limit` cap ARE centralized in `searchActiveNodes` — change ranking once,
  there.
- The two callers intentionally differ on fallback ordering: memory recall sorts
  by importance desc; cross-type search keeps DB `-updated,-created` order. Do
  not "unify" these — they are different product contracts (recall surfaces the
  most important memory first; cross-type search is recency-first).
- A reviewer should scrutinize: (1) that the FTS-path `limit` guard change from
  bare `len > limit` to `limit > 0 && len > limit` in `SearchActive`'s path is
  truly behavior-neutral (all callers pass `limit > 0`); (2) that the
  importance-sort in `SearchActive`'s fallback was preserved verbatim; (3) that
  `hydrate(Memory, r)` still happens on the FTS path only for memory hits.
- Deferred out of this plan: nothing. This is a self-contained dedup with no
  follow-up. If the FTS index ever returns scored results, revisit the id-order
  ranking in the shared core (it assumes the FTS layer already ranked the ids).
