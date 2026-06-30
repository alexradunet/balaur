# Plan 220: Bound `knowledge.FilterActive` — push ordering + a cap into the query instead of loading every active node and sorting in Go

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report — do not improvise. When done, update
> this plan's row in `plans/README.md` unless a reviewer told you they maintain
> the index.
>
> **Drift check (run first)**:
> `git diff --stat ef9f2df..HEAD -- internal/knowledge/knowledge.go internal/knowledge/search.go internal/nodes/nodes.go`
> If any changed since this plan was written, compare the "Current state"
> excerpts against the live code before proceeding; on a mismatch, STOP.

## Status
- **Priority**: P4 (low urgency at single-owner v1 scale; matters as the node count grows)
- **Effort**: M
- **Risk**: MEDIUM (ordering/semantics-preservation trap — read carefully)
- **Depends on**: none
- **Category**: perf / scale
- **Planned at**: commit `ef9f2df`, 2026-06-30

## Why this matters

`knowledge.FilterActive` powers the memory/skill **management** cards and search
(8 call sites in `internal/feature/knowledgecards/*`). It currently:
1. loads **every** active node of the kind (`nodes.ListByTypeStatus`, `limit 0`),
2. hydrates all of them,
3. filters by substring **in Go**,
4. for memories, **re-sorts by importance in Go**.

At one-owner v1 scale this is fine (hundreds of nodes). It does not scale: the
whole active set is read and hydrated on every card render regardless of how many
are shown. The hot context-injection path is already cached (`cache.go`:
`UpfrontMemories`/`ActiveSkills`), so this plan targets only the management/search
path.

**The trap**: `ListByTypeStatus` orders by `-created`, but `FilterActive`
re-sorts memories by `importance` *after* fetching. So you cannot simply pass a
`limit` to the existing query — a capped `-created` fetch would yield the newest
N, then importance-sort *those*, which is NOT the most-important N. Any cap MUST
order by the final sort key in SQL first. This is the whole risk of the plan.

## Current state

`internal/knowledge/knowledge.go:243` — `FilterActive`:

```go
func FilterActive(app core.App, kind Kind, query string) ([]*core.Record, error) {
	recs, err := nodes.ListByTypeStatus(app, string(kind), StatusActive)  // limit 0 = ALL
	if err != nil {
		return nil, err
	}
	hydrateAll(kind, recs)

	q := strings.ToLower(strings.TrimSpace(query))
	out := make([]*core.Record, 0, len(recs))
	for _, r := range recs {
		if q != "" && !matchesQuery(kind, r, q) {   // substring match, in Go
			continue
		}
		out = append(out, r)
	}
	// Memories order by importance desc, then newest; skills keep newest-first.
	if kind == Memory {
		sort.SliceStable(out, func(i, j int) bool {
			return out[i].GetInt("importance") > out[j].GetInt("importance")
		})
	}
	return out, nil
}
```

`matchesQuery` (knowledge.go:267) does a lowercase `strings.Contains` over
`title`, `content`, `when_to_use` (and `description` for skills).

`internal/nodes/nodes.go:229` — the unbounded source query (shared; other
callers depend on its current signature/order):

```go
func ListByTypeStatus(app core.App, typ, status string) ([]*core.Record, error) {
	return app.FindRecordsByFilter("nodes",
		"type = {:t} && status = {:s}", "-created", 0, 0,   // sort=-created, limit 0
		dbx.Params{"t": typ, "s": status})
}
```

`internal/knowledge/search.go:76` — `SearchActive` already has the FTS-or-
fallback shape and **caps to `limit` after** fetching (search.go:59-60); its
fallback also calls `ListByTypeStatus` unbounded (search.go:88). Read it as the
pattern for "bounded list with a limit cap" and keep this plan consistent with it.

## Commands you will need

| Purpose   | Command                                              | Expected         |
|-----------|------------------------------------------------------|------------------|
| Build     | `CGO_ENABLED=0 go build ./...`                       | exit 0           |
| Vet       | `go vet ./...`                                        | exit 0           |
| Test pkg  | `go test ./internal/knowledge/... ./internal/nodes/... -count=1` | PASS |
| Cards     | `go test ./internal/feature/knowledgecards/... -count=1` | PASS        |
| Full test | `go test ./... -count=1`                             | all pass         |
| gofmt     | `gofmt -l internal/knowledge internal/nodes`        | prints nothing   |

> Tests/commits must run with `TMPDIR=/home/alex/.cache/go-tmp` and `-count=1`.

## Scope

**In scope**:
- `internal/knowledge/knowledge.go` — `FilterActive`: accept/apply a bounded,
  correctly-ordered fetch.
- `internal/nodes/nodes.go` — add a bounded+ordered variant of `ListByTypeStatus`
  (do NOT change the existing function's signature; it has other callers).
- The matching `_test.go` (knowledge + nodes) — ordering + cap correctness.

**Out of scope** (do NOT touch):
- `cache.go` (`UpfrontMemories`/`ActiveSkills`) — already the cached hot path.
- `SearchActive`'s FTS path — only read it as the pattern.
- The 8 card call sites' EXTERNAL behavior — the rendered list must look the same
  for any realistic current dataset (below the cap). If you must change call
  sites, it is only to pass a cap/limit through; the displayed result for small
  datasets must be identical.
- Pushing the substring filter into SQL (`LIKE`) — DEFERRED (see Maintenance):
  matching SQLite `LIKE` semantics to `matchesQuery`'s exact lowercase
  `strings.Contains` over multiple fields is its own correctness risk. Keep the
  Go substring match in this plan; only the *fetch bound + order* moves to SQL.

## Git workflow
- Branch: `advisor/220-knowledge-list-bounds`
- Subject e.g. `perf(knowledge): bound FilterActive's active-node fetch with correct ordering`
- Do NOT push unless the operator instructed it.

## Steps

### Step 1: Add a bounded, correctly-ordered nodes query
In `internal/nodes/nodes.go`, add a sibling to `ListByTypeStatus` that takes a
`limit` and an `order` (or two purpose-named helpers). It must:
- order by the FINAL sort key in SQL (`-importance` for memories so the cap keeps
  the *most important*; `-created` for skills/newest-first), and
- apply the limit in the query.
Leave the existing `ListByTypeStatus` untouched (other callers rely on it).

> NOTE on the importance order: confirm `importance` is a stored, sortable field
> on the node record (it is read via `GetInt("importance")` in `FilterActive`).
> If it is a hydrated/computed prop rather than a real column, SQL ordering on it
> is impossible → STOP (see STOP conditions) and fall back to documenting the cap
> as deferred rather than shipping a wrong order.

**Verify**: `go build ./internal/nodes/...` → exit 0.

### Step 2: Apply the bound in `FilterActive`
Introduce a cap constant (e.g. `maxManagementList`, a generous value well above
any realistic current owner dataset — pick a number, document why). Two cases:
- **No query** (`query == ""`): fetch with the bounded+ordered query (correct
  order per kind) and cap. For memories this returns the most-important N already
  ordered; the Go importance re-sort becomes a no-op (keep it as a cheap
  stable-sort safety net or drop it — your call, but the result order must be
  identical to today for ≤cap datasets).
- **With query**: the substring match still needs to scan candidates the DB can't
  pre-filter. Keep the Go `matchesQuery` filter, but still bound the *fetch* with
  the cap + correct order so an unbounded full-collection load can't happen.
  `log` (structured, `app.Logger()`) a debug/info line when the cap truncates, so
  silent truncation is visible (AGENTS: "no silent caps").

For any dataset below the cap, the returned slice (membership AND order) MUST be
identical to the current implementation. That is the acceptance bar.

**Verify**: `go build ./... && go vet ./...` → exit 0.

### Step 3: Tests
In `internal/knowledge` (use `storetest`/`store` app helpers):
- **Order parity (memories)**: seed memories with mixed `importance` + `created`;
  assert `FilterActive(Memory, "")` returns them in the exact same order as today
  (importance desc) — proving the SQL order matches the old Go sort.
- **Cap correctness**: seed > cap memories with known importances; assert the
  returned set is the top-`cap` by importance, not the newest `cap`. (This is the
  trap; make the test explicit.)
- **Query path**: assert substring matching still returns the same records as
  before for a small dataset.
- **Skills**: assert newest-first order is preserved.
In `internal/nodes`: a direct test of the new bounded helper's order + limit.

**Verify**: `go test ./internal/knowledge/... ./internal/nodes/... ./internal/feature/knowledgecards/... -count=1` → PASS.

### Step 4: Full verification
- `gofmt -l internal/knowledge internal/nodes` → nothing
- `go vet ./...` → exit 0
- `go test ./... -count=1` → all pass

## Test plan
The decisive tests are **order parity** and **cap correctness** (Step 3) — they
prove the cap did not silently change which records the owner sees. Everything
else is regression coverage.

## Done criteria — ALL must hold
- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0; gofmt clean
- [ ] `FilterActive` no longer issues an unbounded (`limit 0`) fetch; it uses a capped, correctly-ordered query
- [ ] Existing `ListByTypeStatus` signature is unchanged (other callers unaffected)
- [ ] Test proves memory order parity with the old Go importance sort
- [ ] Test proves the cap keeps the top-N by importance (not newest-N)
- [ ] Truncation is logged (no silent cap)
- [ ] `go test ./... -count=1` exits 0
- [ ] Only `internal/knowledge/*`, `internal/nodes/nodes.go`, and `_test.go` changed
- [ ] `plans/README.md` row updated

## STOP conditions
Stop and report if:
- `importance` is NOT a stored sortable field (it's computed at hydrate time) —
  SQL can't order by it, so a capped fetch cannot preserve importance order.
  Report this; the safe fallback is to keep the unbounded fetch and document the
  scale limit as deferred, rather than ship a cap with the wrong top-N.
- The returned membership/order differs from the current implementation for any
  below-cap dataset — STOP; the management cards' displayed list must not change.
- Bounding the query path would drop substring matches that the current code
  finds (i.e. a match exists beyond the cap) and the cards rely on finding it —
  reconsider the cap size or report it; do not silently make search miss results.

## Maintenance notes
- DEFERRED: pushing the substring filter into SQL (`title/content/when_to_use
  LIKE`) would let the DB do the matching too, removing the Go scan entirely. It
  was left out here because matching SQLite `LIKE` semantics exactly to
  `matchesQuery`'s lowercase `strings.Contains` over multiple fields is a
  separate correctness exercise. Record it as future work if node counts grow.
- This and `SearchActive` (search.go) now both bound their fetches — keep them
  consistent if either's ordering/cap changes.
- Reviewer: the only thing that matters is that the owner sees the SAME records
  in the SAME order for realistic datasets; scrutinize the order-parity and
  top-N-by-importance tests, not the cap value itself.
