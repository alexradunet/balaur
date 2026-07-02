# Plan 244: Skip wikilink re-sync and FTS re-index on node saves that didn't change the indexed/linked content

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- main.go internal/search/index.go internal/search/index_test.go hooks_fanout_test.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: MED
- **Depends on**: none
- **Category**: perf
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

Every chat turn marks each memory that informed it as "used" via
`knowledge.Touch`, which does a full `app.Save` on the node even though only
`props.use_count` and `props.last_used` changed (up to 18 saves per turn: 12
upfront + 6 recalled memories). Two hooks in `main.go` are bound to
`OnRecordAfterUpdateSuccess("nodes")` and fire on every one of those saves:
the FTS upsert (always a DELETE+INSERT into `pb_data/search.db`) and the
wikilink edge sync (`nodes.SyncLinks`), which FULL-REPLACES the node's `links`
edges — deleting every edge and re-creating it, writing one `edge.create`
audit row per edge each time. The same fan-out fires for every task node the
minute-cron nudge marks with `props.nudged_at`. The latency cost is small
today; the real harms are the **unbounded phantom `edge.create` rows in the
owner's audit ledger** for edges that never logically changed (a transparency
cost — the audit log is supposed to be a truthful record of mutations) and
the constant churn of edge record ids. After this plan, a save that did not
change the body skips the edge re-sync entirely, and a save that did not
change any field the FTS index stores skips the re-index — no phantom audit
rows, no edge-id churn, no wasted FTS writes.

## Current state

### The per-turn Touch fan-out

`internal/turn/turn.go:154-157` — every used memory is Touched after each turn:

```go
	// Memories that informed this turn count as used.
	for _, m := range usedMemories {
		knowledge.Touch(app, knowledge.Memory, m)
	}
```

`internal/knowledge/context.go:25-26` — the caps (so up to 12+6 = 18 Touches
per turn):

```go
	upfrontLimit = 12
	recallLimit  = 6
```

`internal/knowledge/knowledge.go:220-230` — Touch changes only props metadata
but does a full save (do NOT change this function — the hooks are the fix):

```go
func Touch(app core.App, kind Kind, rec *core.Record) {
	props := nodes.Props(rec)
	props["use_count"] = nodes.PropInt(rec, "use_count") + 1
	props["last_used"] = time.Now().UTC().Format(time.RFC3339)
	rec.Set("props", props)
	if err := app.Save(rec); err != nil {
		app.Logger().Warn("knowledge touch failed", "kind", string(kind), "id", rec.Id, "err", err)
		return
	}
	hydrate(kind, rec)
}
```

`internal/tasks/nudge.go:115-124` — the minute-cron nudge marks each fired
task with `props.nudged_at` via `txApp.Save(rec)` (same hook fan-out, out of
scope to change; it benefits automatically).

### The two hooks that fire on every node update

`main.go:281-304` — inside `registerSearchIndex`; the upsert hook is bound to
create AND update:

```go
	upsertHook := func(e *core.RecordEvent) error {
		if raw, ok := app.Store().GetOk(search.StoreKey); ok {
			if idx, ok := raw.(*search.Index); ok && idx != nil {
				if err := idx.Upsert(e.Record); err != nil {
					app.Logger().Warn("search: upsert failed", "id", e.Record.Id, "err", err)
				}
			}
		}
		return e.Next()
	}
	deleteHook := func(e *core.RecordEvent) error {
		if raw, ok := app.Store().GetOk(search.StoreKey); ok {
			if idx, ok := raw.(*search.Index); ok && idx != nil {
				if err := idx.Delete(e.Record.Id); err != nil {
					app.Logger().Warn("search: delete failed", "id", e.Record.Id, "err", err)
				}
			}
		}
		return e.Next()
	}

	app.OnRecordAfterCreateSuccess("nodes").BindFunc(upsertHook)
	app.OnRecordAfterUpdateSuccess("nodes").BindFunc(upsertHook)
	app.OnRecordAfterDeleteSuccess("nodes").BindFunc(deleteHook)
```

`main.go:324-339` — `registerGraphLinks`, same shape:

```go
// registerGraphLinks keeps node→node "links" edges in sync with [[wikilinks]]
// in node bodies. On every node create/update it re-parses the body and rewrites
// that node's link edges (creating stub nodes for unresolved titles). Cascade
// delete on the edges relations (plan 160) cleans a deleted node's edges, so no
// delete hook is needed here. A sync failure is logged, never fatal — a bad
// parse must not block the owner's save.
func registerGraphLinks(app core.App) {
	syncHook := func(e *core.RecordEvent) error {
		if err := nodes.SyncLinks(app, e.Record); err != nil {
			app.Logger().Warn("graph: link sync failed", "id", e.Record.Id, "err", err)
		}
		return e.Next()
	}
	app.OnRecordAfterCreateSuccess("nodes").BindFunc(syncHook)
	app.OnRecordAfterUpdateSuccess("nodes").BindFunc(syncHook)
}
```

### What each hook actually does per invocation

`internal/nodes/links.go:106-127` — SyncLinks full-replaces the edge set
(deletes all, re-creates all; do NOT modify this file):

```go
	// Full-replace: delete this source's existing "links" edges, then re-insert
	// the wanted set through AddEdge (160), which is idempotent against the
	// (source,target,type) unique index. (Simplest correct; edge count per node
	// is small — see Maintenance notes for the diff-based optimization.)
	old, err := app.FindRecordsByFilter("edges",
		"source = {:src} && type = {:t}", "", 0, 0,
		dbx.Params{"src": source.Id, "t": DefaultEdgeType})
	if err != nil {
		return fmt.Errorf("nodes: load existing edges: %w", err)
	}
	for _, e := range old {
		if err := app.Delete(e); err != nil {
			return fmt.Errorf("nodes: delete stale edge %s: %w", e.Id, err)
		}
	}
	for tgt := range wantTargets {
		// AddEdge defaults the type to "links" on empty and is idempotent against
		// the unique index (160). In-package call — do NOT redeclare it here.
		if _, err := AddEdge(app, source.Id, tgt, DefaultEdgeType, ""); err != nil {
			return fmt.Errorf("nodes: create edge %s→%s: %w", source.Id, tgt, err)
		}
	}
```

`internal/nodes/nodes.go:283-284` — every re-created edge writes a phantom
audit row:

```go
	store.Audit(app, "owner", "edge.create", "edges/"+rec.Id, true,
		map[string]any{"source": sourceID, "target": targetID, "type": edgeType})
```

`internal/search/index.go:111-131` — Upsert is always DELETE+INSERT, and the
status gate lives INSIDE it (a status flip active→archived must still reach
this delete path — the skip predicate must never swallow a status change):

```go
func (ix *Index) Upsert(rec *core.Record) error {
	// Always delete first so Upsert is truly idempotent.
	if err := ix.Delete(rec.Id); err != nil {
		return err
	}
	if rec.GetString("status") != "active" {
		return nil // non-active: deletion above is the right action
	}
	_, err := ix.db.Exec(
		`INSERT INTO knowledge_fts(id, kind, title, content, extra) VALUES (?, ?, ?, ?, ?)`,
		rec.Id,
		rec.GetString("type"),
		rec.GetString("title"),
		rec.GetString("body"),
		nodeExtra(rec),
	)
```

`internal/search/index.go:99-104` — the fifth indexed value, `extra`, comes
from props (so the FTS skip predicate must compare it too, not raw props):

```go
func nodeExtra(r *core.Record) string {
	if r.GetString("type") == "memory" {
		return strings.TrimSpace(nodes.PropString(r, "when_to_use"))
	}
	return ""
}
```

So the full set of values the FTS index stores per node is: `type`, `title`,
`body`, `nodeExtra(rec)` — plus `status`, which gates membership. `use_count`,
`last_used`, `nudged_at` and all other props are NOT indexed.

### The key PocketBase fact this plan stands on (verified at v0.39.3)

`go.mod` pins `github.com/pocketbase/pocketbase v0.39.3`. In
`~/go/pkg/mod/github.com/pocketbase/pocketbase@v0.39.3/core/record_model.go`:

- `Record.Original()` "returns a shallow copy of the current record model
  populated with its ORIGINAL db data state (aka. right after PostScan())".
- `originalData` is refreshed ONLY by `PostScan()` (which runs when a record
  is scanned from the DB, i.e. on fetch). Nothing in the save/update flow
  refreshes it — the `IgnoreUnchangedFields` doc comment states it outright:
  "if you have performed save on the same Record instance multiple times you
  may have to refetch it, so that m.Original() could reflect the last saved
  change."
- `Record.Clone()` is built on `Original()` and preserves `originalData`, so
  the per-turn memory clones handed out by the knowledge context cache
  (`internal/knowledge/cache.go`, `copyForRead` → `Clone`) carry the
  as-fetched values too.
- `GetRaw` falls back to `originalData` when the key is not in the working
  data, so `e.Record.Original().GetString("body")` and
  `nodes.PropString(original, "when_to_use")` both read the pre-save values
  (this is the same record shape `Rebuild` already handles at
  `internal/search/index.go:64` — freshly scanned records hold everything in
  `originalData`).

Therefore, inside an `OnRecordAfterUpdateSuccess` hook,
`e.Record.Original()` carries the record's **pre-save (as-fetched) field
values** and `e.Record` the just-saved values — exactly the pair the skip
predicates need. This was verified against the module cache source; Step 4's
integration tests re-verify it behaviorally (if the assumption were wrong,
the "must NOT skip" tests would fail).

### Edge-id consumers (why churn-free ids are safe to rely on)

All `edges` readers query by `source`/`target`/`type` filters, never by
pinned edge ids: `internal/nodes/nodes.go:314,327` (Backlinks/Outbound),
`internal/nodes/query.go:94,126`, `internal/nodes/links.go:110`. Nothing
persists an edge id elsewhere, so nothing depends on the current churn.

### Conventions that apply here

- Errors: `fmt.Errorf("doing x: %w", err)`, return early, no panics in
  library code.
- Structured logging only via `app.Logger()` (slog key/value); no
  `fmt.Print*` in service code. The existing hooks already follow this —
  match them.
- No global mutable state; pass `core.App` explicitly.
- Tests: std `testing` package, table-driven where it helps.
  PocketBase-dependent tests boot a temp app via
  `storetest.NewApp(t)` (`internal/storetest/storetest.go:18`) — see
  `internal/search/index_test.go:28-48` (`seedMemoryNode`) for the seeding
  pattern to model. No `time.Sleep`-based synchronization.
- KISS/YAGNI: smallest correct change — two skip guards and one small
  exported predicate, nothing more.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all pass |
| Targeted search tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/search/ -run TestIndexedFieldsChanged -count=1` | ok |
| Targeted tests (general form) | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/<pkg>/ -run <Name> -count=1` | ok |
| Integration tests (exact command in Step 4 — regexp contains pipes, so it lives outside this table) | see Step 4 | ok, 4 tests pass |
| Tours lint (main.go/index.go are tour-anchored) | `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` | ok |
| Vet | `go vet ./...` | exit 0 |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |

(The host `/tmp` is a small tmpfs; the Go linker OOMs there — always set
`TMPDIR=$HOME/.cache/go-tmp` for test runs.)

## Suggested executor toolkit

- If the `go-standards` skill is available, invoke it before writing the Go
  in Steps 1–4 (error wrapping, table-driven test idioms, PocketBase
  patterns).

## Scope

**In scope** (the only files you may modify/create):
- `main.go` — gate the two `OnRecordAfterUpdateSuccess("nodes")` bindings.
- `internal/search/index.go` — add exported `IndexedFieldsChanged` (append at
  END of file — see Step 1 for why placement matters).
- `internal/search/index_test.go` — unit test for the new predicate.
- `hooks_fanout_test.go` (create, repo root, `package main`) — integration
  tests.
- `plans/README.md` — status row only, at the very end.

**Out of scope** (do NOT touch, even though they look related):
- `internal/knowledge/knowledge.go` — `Touch` itself stays a full save; the
  per-turn context cache (plan 183) already bounds reads. Do NOT batch,
  debounce, or remove Touch.
- `internal/nodes/links.go` / `internal/nodes/nodes.go` — the full-replace in
  `SyncLinks` and the `edge.create` audit in `AddEdge` are correct for saves
  whose body DID change; the fix is to not call SyncLinks pointlessly. (A
  diff-based reconcile inside SyncLinks is the documented fallback if a STOP
  condition fires — but do not do both.)
- `internal/tasks/nudge.go` — the nudge marking logic; it benefits from the
  hook guards automatically.
- Auditing of REAL edge changes — an actual body edit must keep producing
  `edge.create` rows.
- `internal/self/knowledge.md` — its prose ("synced on write" at line 93-94,
  "[[wikilinks]]: on save, each link becomes a node→node `links` edge" at
  lines 116-118) stays true: sync is still maintained on every
  content-changing save; only no-op work is skipped. No capability changes.
  Explicitly NOT needed.
- `.tours/*.tour` — see Step 5: all anchors on `main.go` are at lines ≤ 226
  and all edits land at lines ≥ ~300; `internal/search/index.go` anchors are
  at lines 1, 54, 176 and the new function is appended after line 213. No
  anchor shifts, so no tour edits — verified by running TestTours.

## Git workflow

- The executor runs in an isolated git worktree branched from `origin/main`.
- Branch: `advisor/244-node-save-hook-fanout-skip`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/
  `test`/`chore`); commit per logical unit with explicit pathspecs (the main
  checkout is shared by parallel agents — stage only your own files), e.g.:
  - `perf(hooks): skip FTS re-index and wikilink re-sync on metadata-only node saves`
  - `test(hooks): cover fan-out skip and must-not-skip paths`
- **NEVER push.** The reviewer merges.

## Steps

### Step 1: Add `search.IndexedFieldsChanged` and its unit test

In `internal/search/index.go`, append at the **very end of the file** (after
`QueryKind`, which currently ends at line 213 — appending avoids shifting the
tour anchors at index.go lines 1, 54, and 176):

```go
// IndexedFieldsChanged reports whether saving `after` over `before` touched
// anything the FTS index stores (kind=type, title, content=body, extra) or the
// status gate that decides index membership. Metadata-only saves — props like
// use_count/last_used (knowledge.Touch) or nudged_at (the task nudge mark) —
// return false, letting the update hook skip the DELETE+INSERT entirely.
// before is expected to be e.Record.Original() (the pre-save field values).
func IndexedFieldsChanged(before, after *core.Record) bool {
	return before.GetString("type") != after.GetString("type") ||
		before.GetString("title") != after.GetString("title") ||
		before.GetString("body") != after.GetString("body") ||
		before.GetString("status") != after.GetString("status") ||
		nodeExtra(before) != nodeExtra(after)
}
```

Note `nodeExtra` is already in this file (line 99) — that is why the
predicate lives in `internal/search` and not in `main.go`: `extra` is an
indexed value and `nodeExtra` is unexported.

In `internal/search/index_test.go`, add a table-driven
`TestIndexedFieldsChanged`. Build record pairs with
`storetest.NewApp(t)` + `core.NewRecord(col)` on the `nodes` collection
(model on `seedMemoryNode`, index_test.go:28-48, but no save is needed —
just `Set` fields on two in-memory records). Cases (name each):

1. identical type/title/body/status/props → `false`
2. props-only change (`use_count` bumped, `last_used` set; same
   `when_to_use`) on a `memory` node → `false` (the Touch case)
3. title differs → `true`
4. body differs → `true`
5. status `active` vs `archived` → `true` (the index-membership gate)
6. type differs → `true`
7. memory `when_to_use` prop differs → `true` (the extra column)

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/search/ -run TestIndexedFieldsChanged -count=1`
→ `ok`, and `TMPDIR=$HOME/.cache/go-tmp go test ./internal/search/ -count=1`
→ `ok` (no existing test broken).

### Step 2: Gate the FTS update binding in `main.go`

In `registerSearchIndex` (main.go, bindings currently at lines 302-304),
replace ONLY the update binding — create and delete bindings stay
unconditional:

```go
	app.OnRecordAfterCreateSuccess("nodes").BindFunc(upsertHook)
	app.OnRecordAfterUpdateSuccess("nodes").BindFunc(func(e *core.RecordEvent) error {
		// Metadata-only saves (knowledge.Touch use_count bumps, task nudged_at
		// marks) change props the index does not store — skip the DELETE+INSERT.
		// Original() carries the as-fetched (pre-save) field values; PocketBase
		// only refreshes it on fetch (PostScan), never on save. A status flip
		// (e.g. active→archived) changes an indexed gate, so it always passes
		// through to Upsert's delete path.
		if !search.IndexedFieldsChanged(e.Record.Original(), e.Record) {
			return e.Next()
		}
		return upsertHook(e)
	})
	app.OnRecordAfterDeleteSuccess("nodes").BindFunc(deleteHook)
```

`main.go` already imports `internal/search` and `core` — no import changes.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 3: Gate the graph-links update binding in `main.go`

In `registerGraphLinks` (main.go:330-339), keep `syncHook` as-is, keep the
create binding unconditional, and wrap only the update binding. Also update
the function's doc comment (its current second sentence — "On every node
create/update it re-parses the body and rewrites that node's link edges" —
becomes false):

```go
// registerGraphLinks keeps node→node "links" edges in sync with [[wikilinks]]
// in node bodies. On node create, and on any update that changed the body, it
// re-parses the body and rewrites that node's link edges (creating stub nodes
// for unresolved titles); body-unchanged updates (metadata-only saves like
// knowledge.Touch or the task nudge mark) are skipped — the link set derives
// from the body alone, and the full-replace would otherwise delete+recreate
// every edge and write a phantom edge.create audit row per edge. Cascade
// delete on the edges relations (plan 160) cleans a deleted node's edges, so
// no delete hook is needed here. A sync failure is logged, never fatal — a
// bad parse must not block the owner's save.
func registerGraphLinks(app core.App) {
	syncHook := func(e *core.RecordEvent) error {
		if err := nodes.SyncLinks(app, e.Record); err != nil {
			app.Logger().Warn("graph: link sync failed", "id", e.Record.Id, "err", err)
		}
		return e.Next()
	}
	app.OnRecordAfterCreateSuccess("nodes").BindFunc(syncHook)
	app.OnRecordAfterUpdateSuccess("nodes").BindFunc(func(e *core.RecordEvent) error {
		if e.Record.GetString("body") == e.Record.Original().GetString("body") {
			return e.Next()
		}
		return syncHook(e)
	})
}
```

(Body-only comparison is exactly right here: `SyncLinks` computes the wanted
edge set from `source.GetString("body")` alone — the source node's status,
title, and props never enter the computation; see links.go:89-104.)

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0 and `go vet ./...` →
exit 0.

### Step 4: Integration tests for skip and must-not-skip paths

Create `hooks_fanout_test.go` in the repo root, `package main` (the package
already has `tours_test.go` in package main). It exercises the real
registered hooks through `storetest.NewApp(t)`:

Shared setup helper (per test): `app := storetest.NewApp(t)`, then
`registerSearchIndex(app)` and `registerGraphLinks(app)` (both are unexported
main-package functions — callable from this test). `registerSearchIndex`
creates `search.db` inside the test app's temp data dir and puts the index in
`app.Store()` under `search.StoreKey`. Fetch it where needed:

```go
raw, ok := app.Store().GetOk(search.StoreKey)
// assert ok; ix := raw.(*search.Index)
```

Seed nodes directly on the `nodes` collection (model on
`internal/search/index_test.go:28-48` — `core.NewRecord(col)`, `Set` type/
title/body/status/props, `app.Save`). IMPORTANT: before every UPDATE under
test, re-fetch the record fresh (`app.FindRecordById("nodes", id)`) so
`Original()` reflects the DB state — that mirrors production (turn-time
memories are fresh `Clone`s; the nudge cron re-lists records each tick).

Four tests:

1. **`TestTouchSkipsEdgeAndIndexFanout`** — the skip path (the regression this
   plan exists for):
   - Seed an active `memory` node with body `"knows [[Alpha]] and [[Beta]]"`
     and props `{"when_to_use": "greeting", "importance": 4}`. The create
     hook fires `SyncLinks`, creating two stub nodes and two edges.
   - Collect the sorted edge ids for `source = <node id>` from the `edges`
     collection, and `n0 :=` count of `audit_log` rows with
     `action = 'edge.create'`
     (`app.FindRecordsByFilter("audit_log", "action = 'edge.create'", "", 0, 0, nil)`).
   - Re-fetch the memory node fresh, then call
     `knowledge.Touch(app, knowledge.Memory, rec)`.
   - Assert: the sorted edge ids for the source are IDENTICAL (same ids, not
     just same count — id churn is the bug), and the `edge.create` audit
     count is still exactly `n0` (zero phantom rows).
2. **`TestBodyEditResyncsEdges`** — must NOT skip on a real body edit:
   - Continue from the same seeding shape; re-fetch fresh, `Set("body",
     "knows [[Alpha]] and [[Beta]] and [[Gamma]]")`, `app.Save`.
   - Assert: a node titled `Gamma` now exists, an edge from the source to it
     exists, and the `edge.create` audit count increased (> n0).
3. **`TestStatusArchiveDropsFromIndex`** — the status flip must still reach
   Upsert's delete path (the riskiest regression of the FTS guard):
   - Seed an active `note` node with a unique body term, e.g.
     `"xylophonequartz lattice"`. Assert `ix.Query([]string{"xylophonequartz"}, 10)`
     returns its id.
   - Archive it via `nodes.Transition(app, id, "archived", "node")` (the real
     owner path — validates the lifecycle and saves; `Transition` re-fetches
     internally so `Original()` is the pre-save state).
   - Assert the same `ix.Query` no longer returns the id.
4. **`TestTitleEditReindexes`** — title-only edit must re-index:
   - Seed an active `note` titled `"Old Title"` with any body; re-fetch
     fresh, `Set("title", "Zanzibarunique")`, `app.Save`.
   - Assert `ix.Query([]string{"Zanzibarunique"}, 10)` returns the id.

Imports the test will need: `internal/storetest`, `internal/knowledge`,
`internal/nodes`, `internal/search`, `github.com/pocketbase/pocketbase/core`.
Use plain `t.Fatalf` assertions (no assertion frameworks), and never a real
model — nothing here touches `llm.Client`.

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test . -run 'TestTouchSkips|TestBodyEdit|TestStatusArchive|TestTitleEdit' -count=1`
→ `ok`, 4 tests pass.

Sanity check the skip is real: temporarily `git stash` the main.go changes
is NOT needed — instead confirm test 1 FAILS against the old behavior by
reasoning is insufficient; run this one-shot check: with the new code in
place, test 1 passing while tests 2–4 also pass demonstrates both directions.
If test 1 passes but you suspect the guard never engages, add a temporary
`t.Log` in the test on the audit count before/after — do not leave debug
logging in the final code.

### Step 5: Full gates

Run, in order:

1. `gofmt -l .` → empty output
2. `go vet ./...` → exit 0
3. `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output, exit 0
4. `CGO_ENABLED=0 go build ./...` → exit 0
5. `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → `ok`
   (main.go and index.go are tour-anchored: `.tours/09-recall-and-search.tour`
   anchors index.go at lines 1/54/176 and main.go at 226;
   `.tours/19-bootstrapping.tour` anchors main.go at 47/75/126/136/226. All
   plan edits are at main.go ≥ ~300 and index.go > 213, so nothing shifts —
   this run confirms it.)
6. `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0, all pass
7. `git diff --check` → no output
8. `git status --porcelain` → only the in-scope files listed under Scope

**Verify**: all eight commands produce the stated results.

## Test plan

- New unit test `TestIndexedFieldsChanged` in
  `internal/search/index_test.go` — 7 table cases listed in Step 1 (happy
  no-change, the Touch props-only case, and one case per indexed field +
  status + extra). Model the record construction on `seedMemoryNode` in the
  same file (index_test.go:28-48).
- New integration tests in `hooks_fanout_test.go` (package main) — 4 tests
  listed in Step 4: the skip path (edge ids stable + zero phantom
  `edge.create` audit rows under `knowledge.Touch`), and the three
  must-not-skip paths (body edit re-syncs edges + audits; status
  active→archived leaves the FTS index; title edit re-indexes).
- Existing suites must stay green — especially `./internal/search/` (the
  Upsert/Rebuild/consent tests) and the full-repo gate.
- Verification:
  `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0 with the 5 new
  test functions passing (1 unit + 4 integration).

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `gofmt -l .` → empty; `go vet ./...` → exit 0; staticcheck → no output;
      `CGO_ENABLED=0 go build ./...` → exit 0
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0
- [ ] `grep -n "IndexedFieldsChanged" internal/search/index.go main.go` →
      one definition in index.go, one call in main.go
- [ ] `grep -c "OnRecordAfterUpdateSuccess" main.go` → `2` (both now gated:
      the search one calls `search.IndexedFieldsChanged`, the links one
      compares `Original().GetString("body")`)
- [ ] `grep -n "Original()" main.go` → at least 2 hits (one per gated hook)
- [ ] New tests exist and pass:
      `TMPDIR=$HOME/.cache/go-tmp go test . -run 'TestTouchSkips|TestBodyEdit|TestStatusArchive|TestTitleEdit' -count=1` → ok, and
      `TMPDIR=$HOME/.cache/go-tmp go test ./internal/search/ -run TestIndexedFieldsChanged -count=1` → ok
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok
- [ ] `git status --porcelain` shows changes ONLY in: `main.go`,
      `internal/search/index.go`, `internal/search/index_test.go`,
      `hooks_fanout_test.go`, `plans/README.md`
- [ ] `plans/README.md` status row for 244 updated

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows in-scope files changed since `077318a` and the
  "Current state" excerpts no longer match the live code.
- **The key assumption fails behaviorally**: in Step 4, test 2, 3, or 4 fails
  because the guard skipped a real change — i.e. `e.Record.Original()` does
  NOT carry pre-save values inside `OnRecordAfterUpdateSuccess` on the pinned
  PocketBase v0.39.3. Report it; the documented fallback (a NEW plan, not an
  improvisation) is the diff-based reconcile inside `nodes.SyncLinks`
  (compare `wantTargets` against the loaded `old` edges and only
  delete/create the difference — links.go:106-109 already names it), which
  achieves the same observable outcome for edges without `Original()`.
- Test 1 fails because edge ids changed or phantom `edge.create` rows
  appeared even with the guard in place (would mean some other path re-syncs
  — e.g. a second hook binding you did not account for). Do not widen the
  guard; report.
- `registerSearchIndex(app)` cannot run under `tests.NewTestApp` (e.g. the
  FTS5 sidecar fails to open in the sandboxed test env) after one reasonable
  fix attempt. Do not stub the index with fakes; report.
- You find a code path that saves the SAME in-memory `*core.Record` node
  instance twice with a body change between the saves (grep for repeated
  `app.Save` on one node record variable) — the guard would miss a
  changed-then-reverted body on such a path. None exists today
  (`knowledge.Update`, `nodes.Update`, `nodes.Transition`, `tasks` all fetch
  fresh per operation); if one appeared, report instead of landing.
- Any step's verification fails twice after a reasonable fix attempt.
- The fix appears to require touching an out-of-scope file (especially
  `internal/nodes/links.go` or `internal/knowledge/knowledge.go`).

## Maintenance notes

- **The guard's contract**: skipping is only correct while (a) the wikilink
  edge set derives from `body` alone (if `SyncLinks` ever starts reading
  another field — e.g. per-type link rules from props — the body-equality
  guard in `registerGraphLinks` must widen with it), and (b)
  `search.IndexedFieldsChanged` compares every value `Upsert` writes. If a
  new column is added to `knowledge_fts` (or `nodeExtra` learns a new node
  type), update `IndexedFieldsChanged` in the same change — the unit test's
  table is the checklist.
- **PocketBase upgrades**: the guard leans on `Record.Original()` holding
  as-fetched values in after-update hooks (v0.39.3 behavior). The Step 4
  integration tests are the tripwire — if a future PocketBase bump changes
  `Original()` semantics, tests 2–4 fail loudly rather than silently
  staling the index. Keep them.
- **Same-instance double-save caveat**: `Original()` reflects fetch time, not
  last-save time (PocketBase documents this on `IgnoreUnchangedFields`). A
  path that saved one in-memory node record twice — body changed, then
  reverted — would skip a needed re-sync. No such path exists; the FTS side
  additionally self-heals on every boot (`Rebuild`). A reviewer should
  scrutinize exactly this: that both guards compare against `Original()` on
  freshly fetched records only.
- **Explicitly deferred**: diff-based reconcile inside `SyncLinks`
  (links.go:106-109 names it) — with metadata-only saves no longer reaching
  SyncLinks, the remaining full-replace only runs on genuine body edits,
  where the per-node edge count is small; not worth the extra code now.
  Also deferred: batching/removing `knowledge.Touch` itself (out of scope
  here; the context cache from plan 183 already bounds the read side).
- **Audit-ledger effect**: after this lands, `edge.create` rows in
  `audit_log` mean a link genuinely appeared. Anyone analyzing historical
  audit data should know rows before this change include phantom re-creates.
