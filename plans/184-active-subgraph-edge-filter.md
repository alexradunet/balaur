# Plan 184: Scope ActiveSubgraph's edge query to the visible node set

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 12a48bf..HEAD -- internal/nodes/query.go internal/nodes/query_test.go`
> If either in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: perf
- **Planned at**: commit `12a48bf`, 2026-06-24

## Why this matters

`ActiveSubgraph` (internal/nodes/query.go) caps nodes at `limit` (default 50)
but then loads **every** edge in the `edges` table with an empty filter and
discards in Go any edge whose endpoints are not both in the capped node set.
As the knowledge graph grows, rendering an O(limit) subgraph costs an
O(total edges) table scan — paid on every graph-card view (the network card
calls `buildWholeGraphData` → `ActiveSubgraph` on each render, see
internal/web/graph.go:128-129). This plan scopes the edge query to the
already-capped node id set so the database returns only the candidate edges,
turning the scan into an indexed lookup bounded by the node cap. Behavior is
identical: the same nodes, the same edges, the same consent/no-dangle
guarantees — only the cost changes.

## Current state

Files:
- `internal/nodes/query.go` — the `nodes` package's structured queries.
  `ActiveSubgraph` (lines 64-89) is the function to change. It already imports
  `fmt` and `strings` but NOT `github.com/pocketbase/dbx` (you will add that).
- `internal/nodes/query_test.go` — package `nodes_test`. `TestActiveSubgraph`
  (lines 121-151) is the existing characterization test. You will extend it.

The function as it exists today (internal/nodes/query.go:53-89), verbatim:

```go
53	// Edge is one directed link between two active nodes, by id.
54	type Edge struct {
55		Source string
56		Target string
57	}
58
59	// ActiveSubgraph returns the whole active graph: up to limit active nodes
60	// (most-recently-updated first) and every edge whose BOTH endpoints are in that
61	// set. status=active is non-negotiable — proposed and rejected nodes are never
62	// returned and never reachable through an edge (the consent spine). Edges to a
63	// node beyond the cap are dropped so no endpoint dangles.
64	func ActiveSubgraph(app core.App, limit int) ([]*core.Record, []Edge, error) {
65		if limit <= 0 {
66			limit = 50
67		}
68		recs, err := app.FindRecordsByFilter("nodes", "status = 'active'", "-updated,-created", limit, 0, nil)
69		if err != nil {
70			return nil, nil, fmt.Errorf("active subgraph: loading nodes: %w", err)
71		}
72		in := make(map[string]bool, len(recs))
73		for _, r := range recs {
74			in[r.Id] = true
75		}
76
77		allEdges, err := app.FindRecordsByFilter("edges", "", "", 0, 0, nil)
78		if err != nil {
79			return nil, nil, fmt.Errorf("active subgraph: loading edges: %w", err)
80		}
81		edges := make([]Edge, 0, len(allEdges))
82		for _, e := range allEdges {
83		    s, t := e.GetString("source"), e.GetString("target")
84			if in[s] && in[t] {
85				edges = append(edges, Edge{Source: s, Target: t})
86			}
87		}
88		return recs, edges, nil
89	}
```

(Line 83 above is shown with the file's real leading-tab indentation; reproduce
the existing file's whitespace — gofmt will normalize anyway.)

The current import block (internal/nodes/query.go:1-8), verbatim:

```go
1	package nodes
2
3	import (
4		"fmt"
5		"strings"
6
7		"github.com/pocketbase/pocketbase/core"
8	)
```

The existing characterization test (internal/nodes/query_test.go:121-151),
verbatim — note it already proves the consent/no-dangle invariant (an edge
`a→p` to a proposed node is dropped):

```go
121	func TestActiveSubgraph(t *testing.T) {
122		app := storetest.NewApp(t)
123
124		a, _ := nodes.Create(app, "note", "Alpha", "", nodes.StatusActive, nil)
125		b, _ := nodes.Create(app, "note", "Beta", "", nodes.StatusActive, nil)
126		p, _ := nodes.Create(app, "note", "Pending", "", nodes.StatusProposed, nil)
127		if _, err := nodes.AddEdge(app, a.Id, b.Id, "links", ""); err != nil {
128			t.Fatalf("edge a→b: %v", err)
129		}
130		if _, err := nodes.AddEdge(app, a.Id, p.Id, "links", ""); err != nil {
131			t.Fatalf("edge a→p: %v", err)
132		}
133
134		recs, edges, err := nodes.ActiveSubgraph(app, 50)
135		if err != nil {
136			t.Fatalf("ActiveSubgraph: %v", err)
137		}
138		ids := map[string]bool{}
139		for _, r := range recs {
140			ids[r.Id] = true
141		}
142		if !ids[a.Id] || !ids[b.Id] {
143			t.Errorf("active nodes missing: %v", ids)
144		}
145		if ids[p.Id] {
146			t.Error("consent breach: proposed node returned by ActiveSubgraph")
147		}
148		if len(edges) != 1 || edges[0].Source != a.Id || edges[0].Target != b.Id {
149			t.Errorf("edges = %+v, want only a→b (edge to proposed node dropped)", edges)
150		}
151	}
```

### The invariant you must NOT break

The Go both-endpoints check (`if in[s] && in[t]`) is load-bearing and stays.
Reasoning: after this change the SQL filter is `source IN (set) OR target IN
(set)`. That OR will *also* match an edge whose source is in the set but whose
target is an out-of-set node (e.g. a node beyond the `limit` cap, or a proposed
node). Such an edge must still be dropped in Go so no endpoint dangles — that is
the documented consent/no-dangle invariant in the function's doc comment
(lines 59-63). **Keep both-endpoints filtering in Go; the SQL only narrows the
candidate set.** Do not weaken the filter to `AND` either: an edge `a→b` with
`a` in-set and `b` out-of-set is excluded by `a IN AND b IN`, but you need that
row returned-then-dropped only to confirm the Go check still works — practically
the OR is the correct, minimal candidate set and the Go check is the authority.

### Exemplar to copy: building an OR-of-equals IN-list with dbx.Params

This repo builds dynamic OR-of-equals filters from an id slice using indexed
`dbx.Params` keys. Copy this pattern. From `internal/tasks/streak.go:119-129`,
verbatim:

```go
119		// kind = 'completion' && (task = {:t0} || task = {:t1} || …)
120		params := dbx.Params{}
121		conds := make([]string, len(ids))
122		for i, id := range ids {
123			key := fmt.Sprintf("t%d", i)
124			conds[i] = fmt.Sprintf("task = {:%s}", key)
125			params[key] = id
126		}
127		rows, err := app.FindRecordsByFilter("entries",
128			"kind = 'completion' && ("+strings.Join(conds, " || ")+")",
129			"noted_at", 0, 0, params)
```

Key facts from this exemplar, all verified in this repo:
- The dbx import path is `github.com/pocketbase/dbx` (used in
  `internal/nodes/nodes.go:18` and `internal/tasks/streak.go:9`).
- PocketBase filter placeholders are `{:key}`, bound via `dbx.Params{"key": v}`.
- The filter operators are `||` (OR) and `&&` (AND), not SQL `OR`/`AND`.
- `FindRecordsByFilter` signature is
  `(collection, filter, sort, limit, offset, params)` — pass `0, 0` for
  limit/offset to fetch all matches (no cap), and the `dbx.Params` map last.

Another in-package reference for the placeholder + `dbx.Params` shape on the
`edges` collection: `internal/nodes/nodes.go:247`
(`app.FindRecordsByFilter("edges", "target = {:id}", "", 0, 0, dbx.Params{"id": id})`).

### Conventions that apply here

- gofmt is law (a PostToolUse hook rewrites Go files on save, and CI gofmt-gates).
- Errors are values: keep the existing `fmt.Errorf("active subgraph: …: %w", err)`
  wrapping style on the edge query.
- Tests: standard `testing`, table-free is fine here, NO assertion frameworks,
  NO `time.Sleep`. The package already uses `storetest.NewApp(t)` for a
  PocketBase-backed app (boots the full migration chain) — keep using it.
- KISS/YAGNI: smallest correct change. Do not chunk the id list (the node cap is
  ≤ ~maxGraphNodes; default 50 — well within any param limit). Do not add a new
  exported helper; inline the filter build inside `ActiveSubgraph`.

## Commands you will need

| Purpose        | Command                              | Expected on success      |
|----------------|--------------------------------------|--------------------------|
| Build          | `CGO_ENABLED=0 go build ./...`       | exit 0, no output        |
| Test (pkg)     | `go test ./internal/nodes/`          | `ok` / all pass          |
| Test (all)     | `go test ./...`                      | all pass                 |
| Vet            | `go vet ./...`                       | exit 0, no output        |
| Fmt check      | `gofmt -l .`                         | empty output             |
| Diff check     | `git diff --check`                   | empty output             |
| Confirm fix    | `grep -n 'FindRecordsByFilter("edges", "", ""' internal/nodes/query.go` | no matches |

## Scope

**In scope** (the only files you should modify):
- `internal/nodes/query.go` — `ActiveSubgraph` and its import block.
- `internal/nodes/query_test.go` — extend `TestActiveSubgraph`.

**Out of scope** (do NOT touch, even though they look related):
- `internal/web/graph.go` — the graph-card caller; it consumes `ActiveSubgraph`
  unchanged. Its behavior must not change.
- The `Edge` struct (lines 53-57) and its consumers — the return shape stays.
- The `status = 'active'` node filter on line 68 — the consent spine. Preserve
  it EXACTLY; this plan only changes the *edge* query.
- `Query` / `matchesProps` (the rest of query.go) — unrelated.
- Any other `FindRecordsByFilter("edges", …)` callsite — they already pass
  scoped filters (e.g. `internal/nodes/nodes.go:247,260`, `internal/nodes/links.go:110`).

## Git workflow

- Land-on-`main` repo; no PR gate. Executors typically run in a worktree off
  `origin/main`. If you are on `main` directly, branch first
  (e.g. `advisor/184-active-subgraph-edge-filter`).
- Conventional-commit subject, e.g.
  `perf(nodes): scope ActiveSubgraph edge query to the capped node set`.
- Commit or push ONLY if the operator asked. Gate any push on a green
  `go test ./...`.

## Steps

### Step 1: Add the characterization assertion (old behavior, before refactor)

Strengthen `TestActiveSubgraph` in `internal/nodes/query_test.go` so it pins the
two cases that the new SQL filter must keep handling: (a) an edge between two
*active, in-set* nodes is returned, and (b) an edge from an in-set node to an
*out-of-set* node is dropped (no dangle). The existing test already covers (a)
via `a→b` and a proposed-target drop via `a→p`. Add an explicit
"both endpoints out of set" case so the OR-filter's narrowing is also pinned.

Replace the body of `TestActiveSubgraph` (lines 121-151) so it creates an extra
pair of active nodes `c` and `d` that fall *outside* the cap, plus an edge
`c→d` between them, and calls `ActiveSubgraph` with a `limit` small enough to
exclude `c` and `d`. Use the most-recently-updated ordering (`-updated,-created`)
to your advantage: nodes created last sort first, so create `c` and `d` FIRST
(oldest) and `a`, `b` LAST so the small cap keeps `a`/`b` and drops `c`/`d`.

Target shape (adapt names/counts; keep the existing proposed-node assertions):

```go
func TestActiveSubgraph(t *testing.T) {
	app := storetest.NewApp(t)

	// Created first → oldest → sorted last by -updated,-created, so a small
	// cap drops them. An edge between two out-of-set nodes must not appear.
	c, _ := nodes.Create(app, "note", "Gamma", "", nodes.StatusActive, nil)
	d, _ := nodes.Create(app, "note", "Delta", "", nodes.StatusActive, nil)

	a, _ := nodes.Create(app, "note", "Alpha", "", nodes.StatusActive, nil)
	b, _ := nodes.Create(app, "note", "Beta", "", nodes.StatusActive, nil)
	p, _ := nodes.Create(app, "note", "Pending", "", nodes.StatusProposed, nil)

	if _, err := nodes.AddEdge(app, a.Id, b.Id, "links", ""); err != nil { // both in-set
		t.Fatalf("edge a→b: %v", err)
	}
	if _, err := nodes.AddEdge(app, a.Id, p.Id, "links", ""); err != nil { // target proposed → drop
		t.Fatalf("edge a→p: %v", err)
	}
	if _, err := nodes.AddEdge(app, a.Id, c.Id, "links", ""); err != nil { // target out-of-set → drop (no dangle)
		t.Fatalf("edge a→c: %v", err)
	}
	if _, err := nodes.AddEdge(app, c.Id, d.Id, "links", ""); err != nil { // both out-of-set → drop
		t.Fatalf("edge c→d: %v", err)
	}

	// limit=2 keeps the two newest active nodes (a, b); c and d fall outside.
	recs, edges, err := nodes.ActiveSubgraph(app, 2)
	if err != nil {
		t.Fatalf("ActiveSubgraph: %v", err)
	}
	ids := map[string]bool{}
	for _, r := range recs {
		ids[r.Id] = true
	}
	if !ids[a.Id] || !ids[b.Id] {
		t.Errorf("active in-cap nodes missing: %v", ids)
	}
	if ids[p.Id] {
		t.Error("consent breach: proposed node returned by ActiveSubgraph")
	}
	if ids[c.Id] || ids[d.Id] {
		t.Errorf("over-cap nodes returned: c=%v d=%v", ids[c.Id], ids[d.Id])
	}
	if len(edges) != 1 || edges[0].Source != a.Id || edges[0].Target != b.Id {
		t.Errorf("edges = %+v, want only a→b (dangling and out-of-set edges dropped)", edges)
	}
}
```

Note on ordering: `Create` sets `updated`/`created` at insert time; the newest
inserted nodes sort first under `-updated,-created`. If, on your machine, the
timestamp resolution is too coarse and `a`/`b` do not reliably win the cap over
`c`/`d` (a flaky `len(edges)`), that is the assumption-failure STOP condition
below — report it rather than papering over it with a sleep.

**Verify (before changing query.go — this MUST pass against the OLD code, since
the old full-scan + Go filter already yields exactly these results)**:
`go test ./internal/nodes/ -run TestActiveSubgraph` → `ok`, test passes.

### Step 2: Scope the edge query to the capped node set in query.go

In `internal/nodes/query.go`:

1. Add `"github.com/pocketbase/dbx"` to the import block (keep `fmt`, `strings`,
   `core`; gofmt will order/group it). Result:

   ```go
   import (
   	"fmt"
   	"strings"

   	"github.com/pocketbase/dbx"
   	"github.com/pocketbase/pocketbase/core"
   )
   ```

2. Replace the empty-filter edge load (current lines 77-87) with a scoped query
   built from the `in` node-id set, following the `streak.go` exemplar. Keep the
   Go both-endpoints check exactly. Handle the empty-set case (no nodes → no
   possible edges) by skipping the query entirely so you never emit a filter with
   zero conditions. Target shape:

   ```go
   	in := make(map[string]bool, len(recs))
   	ids := make([]string, 0, len(recs))
   	for _, r := range recs {
   		in[r.Id] = true
   		ids = append(ids, r.Id)
   	}

   	var edges []Edge
   	if len(ids) > 0 {
   		// Only edges touching a visible node can survive the both-endpoints
   		// check; let the DB narrow the candidates instead of scanning the
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
   ```

   Notes:
   - Reuse `id` as both `source` and `target` param value — the OR matches an
     edge if the id appears on *either* endpoint, which is the candidate set you
     want. Distinct keys per endpoint (`s%d` / `t%d`) keep params unique.
   - `var edges []Edge` (nil when there are no nodes) is fine: the caller in
     `internal/web/graph.go:141` ranges over it, and `range` over nil is a no-op.
     Do not change the return type.
   - Keep the node-loading block (line 68, `status = 'active'`) untouched.

**Verify**:
- `gofmt -l internal/nodes/query.go` → empty (no output).
- `CGO_ENABLED=0 go build ./...` → exit 0.
- `go test ./internal/nodes/ -run TestActiveSubgraph` → `ok` (the strengthened
  test from Step 1 now passes against the new scoped query — identical results).

### Step 3: Confirm the empty-filter edges query is gone and the suite is green

**Verify**:
- `grep -n 'FindRecordsByFilter("edges", "", ""' internal/nodes/query.go` →
  no matches (the empty-filter scan is eliminated).
- `go test ./internal/nodes/` → `ok`, all pass.
- `go test ./...` → all pass (confirms `internal/web` graph tests still pass —
  `TestBuildWholeGraphData` exercises this path through `buildWholeGraphData`).
- `go vet ./...` → exit 0.
- `git diff --check` → empty.
- `git status` → only `internal/nodes/query.go` and
  `internal/nodes/query_test.go` modified.

## Test plan

- Extend `TestActiveSubgraph` in `internal/nodes/query_test.go` (Step 1) to
  cover, in one PocketBase-backed run via `storetest.NewApp(t)`:
  - happy path: an edge between two active, in-cap nodes (`a→b`) is returned;
  - consent: a proposed-target edge (`a→p`) is dropped, and the proposed node is
    not in the node set;
  - no-dangle: an edge from an in-cap node to an out-of-cap node (`a→c`) is
    dropped (this is the case the new OR filter returns and Go must discard);
  - out-of-set: an edge between two out-of-cap nodes (`c→d`) never appears.
  Asserted result: exactly one edge, `a→b`; nodes `c`, `d`, `p` absent.
- Structural pattern to follow: the existing `TestActiveSubgraph` (it already
  uses `storetest.NewApp`, `nodes.Create`, `nodes.AddEdge`).
- Verification: `go test ./internal/nodes/` → all pass, including the
  strengthened `TestActiveSubgraph`. The same test passing against BOTH the old
  full-scan code (Step 1) and the new scoped code (Step 2) is the proof the
  refactor preserves behavior.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go test ./internal/nodes/` exits 0; the strengthened `TestActiveSubgraph`
      passes.
- [ ] `go test ./...` exits 0 (graph-card path unaffected).
- [ ] `go vet ./...` exits 0.
- [ ] `gofmt -l .` prints nothing.
- [ ] `git diff --check` prints nothing.
- [ ] `grep -n 'FindRecordsByFilter("edges", "", ""' internal/nodes/query.go`
      returns no matches.
- [ ] Only `internal/nodes/query.go` and `internal/nodes/query_test.go` are
      modified (`git status`).
- [ ] `plans/README.md` status row updated.

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows `internal/nodes/query.go` or `query_test.go` changed
  since commit `12a48bf`, and the live code no longer matches the "Current
  state" excerpts (especially if the empty-filter edge load on line 77 is
  already gone or rewritten — someone may have done this work).
- `TestActiveSubgraph` is flaky on the `len(edges)`/cap assertion because the
  `-updated,-created` ordering does not reliably keep `a`/`b` over `c`/`d` at
  `limit=2` (coarse timestamp resolution). Do NOT add `time.Sleep` to force
  ordering — report this so the test can be restructured to set the cap via a
  more deterministic seam.
- Making the test or build pass appears to require editing any file outside the
  two in-scope files (e.g. `internal/web/graph.go`, `nodes.go`, a migration).
- `go test ./...` fails in a package other than `internal/nodes` after the change
  (the edge return shape or values changed unexpectedly).
- A verification command fails twice after a reasonable fix attempt.

## Maintenance notes

For the human/agent who owns this code after the change lands:

- The Go both-endpoints check (`if in[s] && in[t]`) is the authority on which
  edges survive; the SQL OR-filter is only a candidate-narrowing optimization.
  If anyone later "simplifies" by trusting the SQL filter and dropping the Go
  check, dangling edges (in-set → out-of-set) will leak back in — that breaks
  the consent/no-dangle spine documented on the function. Keep both.
- The id list is intentionally un-chunked because the node cap (`maxGraphNodes`,
  used by `buildWholeGraphData`; default `limit` 50) bounds it well below any
  SQLite parameter limit (default ~999, and we emit 2 params per node). If
  `ActiveSubgraph` ever gains a much larger cap or an unbounded mode, revisit:
  chunk the id list into batches and union the results (see how other large
  IN-lists would need batching), or fall back to a join.
- A reviewer should scrutinize: (1) the `status = 'active'` node filter is
  untouched; (2) the both-endpoints Go check is unchanged; (3) the empty-node-set
  branch avoids emitting a zero-condition filter; (4) the strengthened test
  asserts the out-of-set drop, not just the proposed-node drop.
