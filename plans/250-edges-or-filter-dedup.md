# Plan 250: Collapse the duplicated edges OR-filter builder in nodes/query.go

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 077318a..HEAD -- internal/nodes/query.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

`internal/nodes/query.go` contains two byte-near-identical 10-line builders of
the same SQL OR-filter (`source = {:sN} || target = {:tN}` over a list of node
ids): one inlined in `ActiveSubgraph`, one in `EdgesTouching` — whose own doc
comment admits "Uses the same OR-filter idiom as ActiveSubgraph". This is a
classic drift setup: a future change to the filter (a parameter-count cap, a
filter-syntax change after a PocketBase upgrade, an index hint) will land in
one copy and silently miss the other. The repository's own rules
(AGENTS.md, "KISS / YAGNI / SUCKLESS") say: "collapse duplicated boilerplate
into one shared helper; … one source of truth per concern." This plan makes
`ActiveSubgraph` delegate its candidate-edge fetch to `EdgesTouching` and
deletes the inline copy. Zero behavior change.

## Current state

Relevant files:

- `internal/nodes/query.go` — node/edge listing helpers; contains both copies
  of the builder (the only file this plan modifies).
- `internal/web/graph.go` — the two callers outside the package:
  `buildGraphData` calls `nodes.EdgesTouching` (line 69), `buildWholeGraphData`
  calls `nodes.ActiveSubgraph` (line 150). Read-only for this plan.
- `internal/nodes/query_test.go` — `TestActiveSubgraph` (line 121) covers the
  both-endpoints/consent behavior that must not change.
- `internal/web/graph_test.go` — `TestBuildGraphData`, `TestBuildWholeGraphData`,
  `TestBuildGraphDataNoEdges`, `TestBuildGraphDataDepth2AndInbound`,
  `TestBuildGraphDataInactiveFocus` exercise both functions through the web
  graph endpoints.

### Copy 1: the inline builder inside `ActiveSubgraph`

`internal/nodes/query.go:60-108` (the whole function, as it exists today):

```go
// ActiveSubgraph returns the whole active graph: up to limit active nodes
// (most-recently-updated first) and every edge whose BOTH endpoints are in that
// set. status=active is non-negotiable — proposed and rejected nodes are never
// returned and never reachable through an edge (the consent spine). Edges to a
// node beyond the cap are dropped so no endpoint dangles.
func ActiveSubgraph(app core.App, limit int) ([]*core.Record, []Edge, error) {
	if limit <= 0 {
		limit = 50
	}
	recs, err := app.FindRecordsByFilter("nodes", "status = 'active'", "-updated,-created", limit, 0, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("active subgraph: loading nodes: %w", err)
	}
	in := make(map[string]bool, len(recs))
	ids := make([]string, 0, len(recs))
	for _, r := range recs {
		in[r.Id] = true
		ids = append(ids, r.Id)
	}

	var edges []Edge
	if len(ids) > 0 {
		// Only edges touching a visible node can survive the both-endpoints
		// check, so let the DB narrow the candidates instead of scanning the
		// whole edges table. The Go check below is still the authority — an
		// edge from a visible node to an out-of-set node matches this OR but
		// must be dropped so no endpoint dangles (the consent/no-dangle spine).
		params := dbx.Params{}
		conds := make([]string, 0, len(ids))
		for i, id := range ids {
			sk, tk := fmt.Sprintf("s%d", i), fmt.Sprintf("t%d", i)
			conds = append(conds, fmt.Sprintf("source = {:%s}", sk), fmt.Sprintf("target = {:%s}", tk))
			params[sk], params[tk] = id, id
		}
		candidates, err := app.FindRecordsByFilter("edges",
			strings.Join(conds, " || "), "", 0, 0, params)
		if err != nil {
			return nil, nil, fmt.Errorf("active subgraph: loading edges: %w", err)
		}
		edges = make([]Edge, 0, len(candidates))
		for _, e := range candidates {
			s, t := e.GetString("source"), e.GetString("target")
			if in[s] && in[t] {
				edges = append(edges, Edge{Source: s, Target: t})
			}
		}
	}
	return recs, edges, nil
}
```

### Copy 2: `EdgesTouching`

`internal/nodes/query.go:110-132` (as it exists today):

```go
// EdgesTouching returns every record in the edges collection whose source or
// target is one of ids, in a single query. Uses the same OR-filter idiom as
// ActiveSubgraph. The caller is responsible for any further filtering (e.g.
// both-endpoints-active); an edge where only one endpoint is in ids is still
// returned.
func EdgesTouching(app core.App, ids []string) ([]*core.Record, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	params := dbx.Params{}
	conds := make([]string, 0, len(ids)*2)
	for i, id := range ids {
		sk, tk := fmt.Sprintf("s%d", i), fmt.Sprintf("t%d", i)
		conds = append(conds, fmt.Sprintf("source = {:%s}", sk), fmt.Sprintf("target = {:%s}", tk))
		params[sk], params[tk] = id, id
	}
	edges, err := app.FindRecordsByFilter("edges",
		strings.Join(conds, " || "), "", 0, 0, params)
	if err != nil {
		return nil, fmt.Errorf("edges touching: %w", err)
	}
	return edges, nil
}
```

### Why delegation is behavior-preserving (verified facts)

- Both copies run the identical query: `FindRecordsByFilter("edges",
  strings.Join(conds, " || "), "", 0, 0, params)` — same collection, same
  filter shape, same empty sort, no limit. Both return `[]*core.Record`, and
  `ActiveSubgraph` already iterates those records with
  `e.GetString("source")` / `e.GetString("target")` — exactly the fields any
  edge record carries. No type mismatch.
- `ActiveSubgraph` only reaches the builder when `len(ids) > 0`, and
  `EdgesTouching` returns `(nil, nil)` for empty ids anyway. Keep the existing
  `if len(ids) > 0` guard so the `edges` slice stays `nil` (not an empty
  non-nil slice) when there are no active nodes — minimal diff, identical
  nil-ness.
- The only observable difference after delegation is the error string on a
  failed edge query: it becomes
  `"active subgraph: loading edges: edges touching: <cause>"` (one extra wrap
  layer). No test asserts on that exact string.
- The consent/no-dangle comment at `query.go:81-86` documents a real
  invariant (the Go both-endpoints check is the authority, not the SQL OR).
  It must survive the refactor, reworded to reference the delegation.

### Repo conventions that apply

- Errors: `fmt.Errorf("doing x: %w", err)`, return early, no panics in library
  code. The existing wraps in this file are the exemplar — keep them.
- KISS/YAGNI: smallest correct change. Do NOT introduce a third helper, an
  options struct, or any new exported symbol — the fix is pure deletion +
  delegation.
- Surgical changes: touch only the lines this refactor requires; do not
  reformat or "improve" `Query`, `ActiveByIDs`, or `matchesProps`.
- `internal/self/knowledge.md` update is NOT needed: this is an internal
  dedup with zero user-visible architecture or capability change.
- `.tours/` update is NOT needed: no tour anchors `internal/nodes/query.go`
  by file+line (verified: `grep -rn "nodes/query" .tours/` returns nothing;
  the only mention of `query.go` is prose in `00-orientation.tour` anchored
  to `internal/nodes/nodes.go:89`, which this plan does not touch).

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all pass |
| Targeted tests (nodes) | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/nodes/ -run TestActiveSubgraph -count=1` | ok, exit 0 |
| Targeted tests (web graph) | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run 'TestBuildGraphData|TestBuildWholeGraphData' -count=1` | ok, exit 0 |
| Vet | `go vet ./...` | exit 0, no output |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |

Note: the host `/tmp` is a small tmpfs and the Go linker OOMs there — always
prefix test runs with `TMPDIR=$HOME/.cache/go-tmp` as shown.

## Scope

**In scope** (the only file you should modify):
- `internal/nodes/query.go`

**Out of scope** (do NOT touch, even though they look related):
- `EdgesTouching`'s contract and doc comment beyond the one sentence named in
  Step 1 — in particular, it MUST keep returning edges where only ONE endpoint
  is in `ids`; `buildGraphData` in `internal/web/graph.go:69` relies on
  exactly that to discover new BFS neighbors.
- `internal/web/graph.go` — both callers keep working unchanged.
- `internal/nodes/query_test.go` and `internal/web/graph_test.go` — existing
  tests already pin the behavior; no test changes are needed or wanted.
- Any new exported symbol, options struct, or shared "buildOrFilter" helper —
  the delegation IS the dedup.

## Git workflow

- Work in an isolated git worktree branched from `origin/main`; branch name
  `advisor/250-edges-or-filter-dedup`.
- Conventional-commit subject (`feat`/`fix`/`docs`/`refactor`/`style`/`test`/`chore`);
  this change is one commit: `refactor(nodes): collapse duplicated edges OR-filter into EdgesTouching`.
- Stage with explicit pathspecs only (`git add internal/nodes/query.go`) — the
  main checkout is shared by parallel agents; never `git add -A`.
- **NEVER push.** The reviewer merges.

## Steps

### Step 1: Delegate ActiveSubgraph's candidate fetch to EdgesTouching

In `internal/nodes/query.go`, inside `ActiveSubgraph`, replace the inline
builder block (today's lines 87-98: the `params := dbx.Params{}` declaration
through the `app.FindRecordsByFilter("edges", ...)` call and its error check)
with a call to `EdgesTouching(app, ids)`, and reword the comment above it so
it still documents the invariant. Target shape of the `var edges []Edge` block
(the rest of the function is untouched):

```go
	var edges []Edge
	if len(ids) > 0 {
		// Only edges touching a visible node can survive the both-endpoints
		// check, so EdgesTouching lets the DB narrow the candidates instead
		// of scanning the whole edges table. The Go check below is still the
		// authority — EdgesTouching also returns edges where only ONE
		// endpoint is in ids, and those must be dropped so no endpoint
		// dangles (the consent/no-dangle spine).
		candidates, err := EdgesTouching(app, ids)
		if err != nil {
			return nil, nil, fmt.Errorf("active subgraph: loading edges: %w", err)
		}
		edges = make([]Edge, 0, len(candidates))
		for _, e := range candidates {
			s, t := e.GetString("source"), e.GetString("target")
			if in[s] && in[t] {
				edges = append(edges, Edge{Source: s, Target: t})
			}
		}
	}
	return recs, edges, nil
```

Keep the `if len(ids) > 0` guard (preserves `edges == nil` on an empty graph).
Keep the both-endpoints loop byte-identical. Do not change `EdgesTouching`'s
body; in its doc comment, update only the second sentence — "Uses the same
OR-filter idiom as ActiveSubgraph." is now wrong (ActiveSubgraph delegates
instead of duplicating), so replace that one sentence with:
`ActiveSubgraph delegates its candidate fetch here.` Leave every other
sentence of that comment intact.

Both `strings` and `dbx` imports remain used after the edit (`strings` by
`EdgesTouching` and `matchesProps`, `dbx` by `EdgesTouching`) — do not touch
the import block.

**Verify**:
- `gofmt -l .` → empty output
- `go vet ./...` → exit 0
- `CGO_ENABLED=0 go build ./...` → exit 0
- `grep -c 'source = {:' internal/nodes/query.go` → `1`
- `grep -c 'dbx.Params{}' internal/nodes/query.go` → `1`

### Step 2: Prove zero behavior change with the existing tests

Run the tests that pin both functions' behavior, then the full gate.

**Verify**:
- `TMPDIR=$HOME/.cache/go-tmp go test ./internal/nodes/ -run TestActiveSubgraph -count=1` → `ok`, exit 0
- `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run 'TestBuildGraphData|TestBuildWholeGraphData' -count=1` → `ok`, exit 0
- `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output, exit 0
- `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0, all packages pass

### Step 3: Commit

`git add internal/nodes/query.go` then commit with subject
`refactor(nodes): collapse duplicated edges OR-filter into EdgesTouching`.

**Verify**: `git status --porcelain` → only expected worktree noise, no
unstaged changes to tracked files; `git show --stat HEAD` → exactly 1 file
changed (`internal/nodes/query.go`).

## Test plan

No new tests. This is a pure dedup and the behavior is already pinned:

- `TestActiveSubgraph` (`internal/nodes/query_test.go:121-167`) is the direct
  regression net — it creates active/rejected/proposed nodes plus four edges
  and asserts exactly one edge (`a→b`, both endpoints active) survives, i.e.
  the both-endpoints authority check and the consent spine. This test failing
  after the change means the delegation altered behavior — a STOP condition.
- `TestBuildGraphData`, `TestBuildWholeGraphData`, `TestBuildGraphDataNoEdges`,
  `TestBuildGraphDataDepth2AndInbound`, `TestBuildGraphDataInactiveFocus`
  (`internal/web/graph_test.go`) cover both callers end-to-end through the web
  graph builders.
- Verification: the two targeted commands in Step 2, then the full
  `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` gate → all pass.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -c 'source = {:' internal/nodes/query.go` prints `1` (one builder
      left, inside `EdgesTouching`)
- [ ] `grep -c 'dbx.Params{}' internal/nodes/query.go` prints `1`
- [ ] `grep -n 'EdgesTouching(app, ids)' internal/nodes/query.go` shows a hit
      inside `ActiveSubgraph`
- [ ] `grep -c 'no endpoint dangles' internal/nodes/query.go` prints `1`
      (the consent/no-dangle comment survived)
- [ ] `gofmt -l .` → empty; `go vet ./...` → exit 0;
      `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output
- [ ] `CGO_ENABLED=0 go build ./...` → exit 0
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0
- [ ] `git diff --stat origin/main..HEAD` lists ONLY
      `internal/nodes/query.go` (plus `plans/README.md` if you maintain the
      index) — no out-of-scope files modified
- [ ] `plans/README.md` status row for 250 updated (unless the dispatching
      reviewer maintains the index)

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows `internal/nodes/query.go` changed since `077318a` and
  the live code no longer matches the "Current state" excerpts (e.g. someone
  already collapsed the builder, or the filter syntax changed).
- `TestActiveSubgraph` fails after Step 1 — the delegation was supposed to be
  behavior-preserving; a failure means an assumption in "Current state" is
  wrong (most likely a difference between the two queries you should not
  paper over).
- `ActiveSubgraph`'s edge iteration turns out to need a field or ordering that
  `EdgesTouching`'s return does not carry (it should only need
  `GetString("source")`/`GetString("target")` on `[]*core.Record` with no
  ordering dependence — if reality differs, stop).
- The fix appears to require touching `internal/web/graph.go`,
  `EdgesTouching`'s body, or any test file.
- Any verification fails twice after a reasonable fix attempt.

## Maintenance notes

- `EdgesTouching` is now the single source of truth for the edges OR-filter.
  Any future change to it (e.g. chunking the OR when `len(ids)` grows past a
  SQLite parameter cap, or swapping to an `IN (...)` filter) automatically
  serves both `ActiveSubgraph` and `buildGraphData` — which is the point. When
  making such a change, remember its two distinct consumers: `ActiveSubgraph`
  filters to both-endpoints-in-set afterward, while `buildGraphData`
  (`internal/web/graph.go`) depends on one-endpoint edges being returned.
- Reviewer focus: confirm the both-endpoints loop in `ActiveSubgraph` is
  byte-identical to before, the `if len(ids) > 0` guard survived (nil-vs-empty
  `edges` slice), and no new exported symbols appeared.
- Deferred (deliberately): no parameter-count cap on the OR filter. Both call
  sites are bounded today (`ActiveSubgraph` by `limit`, default 50 via
  `maxGraphNodes`; BFS frontiers similarly capped), so a cap is YAGNI until a
  caller passes unbounded ids.
