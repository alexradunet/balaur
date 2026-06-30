# Plan 215: Batch the graph-card BFS — replace the per-node `Outbound`/`Backlinks` N+1 with a level-batched edge fetch

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report — do not improvise. When done, update
> this plan's row in `plans/README.md` unless a reviewer told you they maintain
> the index.
>
> **Drift check (run first)**:
> `git diff --stat ef9f2df..HEAD -- internal/web/graph.go internal/nodes/nodes.go`
> If either file changed since this plan was written, compare the "Current state"
> excerpts against the live code before proceeding; on a mismatch, STOP.

## Status
- **Priority**: P2
- **Effort**: M
- **Risk**: LOW
- **Depends on**: none
- **Category**: perf
- **Planned at**: commit `ef9f2df`, 2026-06-30

## Why this matters

`GET /ui/graph.json?id=&depth=` (the focused graph card) walks the active
neighborhood with a depth-limited BFS that issues **4 database round-trips per
visited node**: one `edges` query + one `nodes` id-batch for outbound, and the
same pair again for inbound. At the `maxGraphNodes` cap of 150 and depth 2 that
is up to ~600 queries to render one card. The whole-graph sibling
(`buildWholeGraphData`) already avoids this by calling the batched
`nodes.ActiveSubgraph`; the focused builder did not get the same treatment. This
is a bounded, on-demand storm (not a steady-state drain), but every graph open
pays it and it scales with neighborhood size.

## Current state

`internal/web/graph.go` — the focused BFS (the N+1 is the inner loop, lines
61–88):

```go
// buildGraphData (graph.go:48)
	seen := map[string]*core.Record{focus.Id: focus}
	links := map[[2]string]bool{}
	frontier := []string{focus.Id}

	for d := 0; d < depth && len(seen) < maxGraphNodes; d++ {
		var next []string
		for _, id := range frontier {
			out, err := nodes.Outbound(app, id)          // <-- query per node
			if err != nil {
				return graphData{}, err
			}
			for _, n := range out {
				links[[2]string{id, n.Id}] = true
				if _, ok := seen[n.Id]; !ok && len(seen) < maxGraphNodes {
					seen[n.Id] = n
					next = append(next, n.Id)
				}
			}
			back, err := nodes.Backlinks(app, id)        // <-- query per node
			if err != nil {
				return graphData{}, err
			}
			for _, n := range back {
				links[[2]string{n.Id, id}] = true
				if _, ok := seen[n.Id]; !ok && len(seen) < maxGraphNodes {
					seen[n.Id] = n
					next = append(next, n.Id)
				}
			}
		}
		frontier = next
	}
```

The helpers it calls — `internal/nodes/nodes.go`:

```go
// activeByIDs (nodes.go:291) — one FindRecordsByIds, keeps only active, preserves order.
func activeByIDs(app core.App, ids []string) ([]*core.Record, error) { ... }

// Backlinks (nodes.go:313)
func Backlinks(app core.App, id string) ([]*core.Record, error) {
	edges, err := app.FindRecordsByFilter("edges", "target = {:id}", "", 0, 0, dbx.Params{"id": id})
	// ... collect source ids ... return activeByIDs(app, ids)
}

// Outbound (nodes.go:326)
func Outbound(app core.App, id string) ([]*core.Record, error) {
	edges, err := app.FindRecordsByFilter("edges", "source = {:id}", "", 0, 0, dbx.Params{"id": id})
	// ... collect target ids ... return activeByIDs(app, ids)
}
```

The batched EXEMPLAR to mirror — `internal/web/graph.go:128` `buildWholeGraphData`
already calls `nodes.ActiveSubgraph(app, maxGraphNodes)` (in `internal/nodes/nodes.go`),
which loads the whole active subgraph's nodes + edges in a small fixed number of
queries. **Open `nodes.ActiveSubgraph` and reuse its batched edge-query
technique** — it has already solved "load all edges touching a set of nodes in
one query" the PocketBase-idiomatic way (whatever filter form it uses for the IN
set, copy it exactly rather than inventing one).

The consent spine: `activeByIDs` keeps only `status == StatusActive` records
(`nodes.go:305`). Any batched replacement MUST preserve that filter — a proposed
or rejected node must never appear in the graph.

## Commands you will need

| Purpose   | Command                                              | Expected         |
|-----------|------------------------------------------------------|------------------|
| Build     | `CGO_ENABLED=0 go build ./...`                       | exit 0           |
| Vet       | `go vet ./...`                                        | exit 0           |
| Test pkg  | `go test ./internal/web/... ./internal/nodes/... -count=1` | PASS       |
| Full test | `go test ./... -count=1`                             | all pass         |
| gofmt     | `gofmt -l internal/web internal/nodes`              | prints nothing   |

> Tests/commits must run with `TMPDIR=/home/alex/.cache/go-tmp` (the default
> tmpfs `/tmp` OOMs the Go linker) and `-count=1` (the test cache can mask
> date-dependent failures).

## Scope

**In scope**:
- `internal/web/graph.go` (rewrite `buildGraphData`'s BFS to batch per level)
- `internal/nodes/nodes.go` (add the batched helper(s) the BFS needs — e.g. an exported "edges touching a set of ids" loader + exported active-by-ids, mirroring `ActiveSubgraph`)
- the matching `_test.go` (add a behavior-preserving graph test)

**Out of scope** (do NOT touch):
- `buildWholeGraphData` / `nodes.ActiveSubgraph` — already batched; only read them as the pattern.
- `Outbound`/`Backlinks` themselves — they have other callers (`nodes.Neighborhood`, node-card "links" summaries). Leave them; the BFS stops using them but they stay.
- The wire shape (`graphData`/`graphNode`/`graphLink`) and the JSON contract — unchanged; the client depends on it.
- The `maxGraphNodes` cap and the dangling-link drop logic (graph.go:101–109) — keep them.

## Git workflow
- Branch: `advisor/215-graph-bfs-batch`
- Conventional-commit subject, e.g. `perf(web): batch the graph BFS edge fetch (drop the per-node N+1)`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add a batched edge/neighbor loader in `internal/nodes`
Open `nodes.ActiveSubgraph` and copy its edge-query idiom. Add an exported helper
(name it for what it does, e.g. `EdgesTouching(app core.App, ids []string) ([]*core.Record, error)`)
that returns, in ONE `edges` query, every edge whose `source` OR `target` is in
`ids`. Also expose an exported active-by-ids batch loader (either export
`activeByIDs` as `ActiveByIDs`, or have the new helper return the active neighbor
records directly) — preserving the `status == StatusActive` filter and id-order
semantics of the existing `activeByIDs`.

**Verify**: `go build ./internal/nodes/...` → exit 0.

### Step 2: Rewrite `buildGraphData`'s BFS to batch per level
Replace the inner `for _, id := range frontier { Outbound; Backlinks }` with a
per-LEVEL batch:
1. One `EdgesTouching(app, frontier)` call.
2. From the returned edges, record each directed pair into `links` (source→target)
   and collect the set of neighbor ids (the endpoint not already in `seen`),
   respecting the `len(seen) < maxGraphNodes` cap.
3. One active-by-ids batch load of the new neighbor ids; add the active ones to
   `seen` and to `next`.
Keep the `seen`/`links` maps, the cap checks, and the final dangling-link drop
(graph.go:101–109) exactly as they are. The output `graphData` must be identical
for any given graph.

**Verify**: `go build ./internal/web/...` → exit 0; `go vet ./internal/web/...` → exit 0.

### Step 3: Behavior-preserving test
Add a test in `internal/web` (model it on existing `internal/web` tests using
`storetest.NewApp(t)`): build a small fixture — a focus node with a few outbound
+ inbound edges, one proposed/rejected neighbor (to prove the consent filter
still excludes it), and one neighbor just past a deliberately-low cap if feasible
— then call `buildGraphData` and assert the returned `Nodes` and `Links` sets
match the expected set (the same as the old per-node walk would produce). If you
can add a query-count seam cheaply, assert the batched version issues O(depth)
edge queries rather than O(nodes); otherwise correctness + the diff is the proof.

**Verify**: `go test ./internal/web/... ./internal/nodes/... -count=1` → PASS.

### Step 4: Full verification
- `gofmt -l internal/web internal/nodes` → prints nothing
- `go vet ./...` → exit 0
- `go test ./... -count=1` → all pass

## Test plan
- New `internal/web` test (above): correctness parity on a fixture incl. the
  consent-filter exclusion and the dangling-link drop.
- Existing `internal/nodes` and `internal/web` tests must stay green (the new
  `nodes` helper is additive; `Outbound`/`Backlinks` untouched).

## Done criteria — ALL must hold
- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0; `gofmt -l internal/web internal/nodes` prints nothing
- [ ] `grep -n "nodes.Outbound\|nodes.Backlinks" internal/web/graph.go` → no matches (the BFS no longer calls them per-node)
- [ ] `buildGraphData` issues a fixed number of queries per BFS level, not per node (visible in the diff)
- [ ] A graph test asserts the node+link set is unchanged AND that a proposed/rejected neighbor is excluded
- [ ] `go test ./... -count=1` exits 0
- [ ] Only `internal/web/graph.go`, `internal/nodes/nodes.go`, and a `_test.go` changed (`git status`)
- [ ] `plans/README.md` row updated

## STOP conditions
Stop and report if:
- `nodes.ActiveSubgraph` no longer exists or its edge-query technique can't be reused (the exemplar is gone) — report; do not invent an `IN` filter form that diverges from the rest of the package.
- The batched result differs from the per-node walk for your fixture (missing/extra node or link) — the dedup or cap logic was changed; do not ship a behavior change.
- Preserving the `status=active` consent filter in the batch path is not straightforward — STOP; the consent spine is non-negotiable.

## Maintenance notes
- After this, the focused (`buildGraphData`) and whole-graph (`buildWholeGraphData`)
  builders both batch — keep them consistent if either changes.
- `Outbound`/`Backlinks` remain the per-node helpers for node-card link summaries;
  this plan deliberately leaves them.
- Reviewer: confirm the consent filter (`status=active`) survives the batch and
  that the JSON wire shape is byte-identical for a sample graph.
