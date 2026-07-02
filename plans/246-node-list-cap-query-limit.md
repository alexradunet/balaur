# Plan 246: Cap the node_list tool's output at 50 and push nodes.Query's limit into SQL when no prop filter applies

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- internal/nodes/query.go internal/nodes/query_test.go internal/tools/knowledge.go internal/tools/knowledge_test.go internal/self/knowledge.md .tours/17-the-tool-surface.tour .tours/06-memory-and-self-evolution.tour`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: plans/250-edges-or-filter-dedup.md — merge-friction only:
  both plans edit `internal/nodes/query.go`, so land 250 first to avoid a
  conflict. If that plan file does not exist, is REJECTED, or is already
  merged, proceed with this plan as written.
- **Category**: perf
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

The `node_list` agent tool renders **every** active node of a type — one line
per node, no cap — straight into the tool result that enters the model's
context mid-turn. Balaur runs local models with small context windows; with a
few hundred notes, one `node_list` call can blow the window or crowd out the
conversation. Separately, `nodes.Query` fetches **all** active nodes from
SQLite (SQL limit `0`) even when no prop filter applies, then truncates to the
cap in Go — the sort is already SQL-side (`-updated,-created`), so the limit
is trivially pushable in that branch. After this plan: `node_list` returns at
most 50 newest entries plus an explicit truncation line (so the model *knows*
it is looking at a slice and can narrow with `node_query` or `search`), and
`nodes.Query` bounds its DB fetch whenever it does not need the full set for
in-Go prop filtering.

## Current state

Relevant files (all paths repo-relative):

- `internal/nodes/query.go` — structured node search (`nodes.Query`) shared
  by the `node_query` tool (`internal/tools/graph.go:162`) and the graph
  card fallback (`internal/feature/graphcards/network.go:78`). Contains the
  unbounded fetch this plan bounds.
- `internal/nodes/nodes.go` — generic node access; `ListByTypeStatus`
  (lines 228–233) is the unbounded, `-created`-sorted loader that
  `nodes.Query`'s type branch currently delegates to. `ListByTypeStatus`
  itself stays **unchanged** — many callers (export, tasks, knowledge cache,
  CLI) rely on its full-set, no-limit contract.
- `internal/tools/knowledge.go` — the knowledge tool group; `nodeListTool`
  (lines 411–448) is the uncapped tool this plan caps.
- `internal/nodes/query_test.go`, `internal/tools/knowledge_test.go` —
  existing tests; new tests land here.
- `internal/self/knowledge.md` — the binary's self-description; one sentence
  (line 185) describes `node_list` and gets a parenthetical about the cap.
- `.tours/17-the-tool-surface.tour`, `.tours/06-memory-and-self-evolution.tour`
  — code tours with steps anchored to `internal/tools/knowledge.go` lines 23
  and 43 (tour 17) and line 43 (tour 06). Step 3 adds one import line to that
  file, shifting both anchors down by exactly one line — the tours must be
  repointed in the same change (`.tours/` are maintained artifacts;
  `tours_test.go` only catches missing files/out-of-range lines, so a
  one-line drift would silently mis-anchor the prose).

### `nodes.Query` today — `internal/nodes/query.go:20-52`

```go
func Query(app core.App, opts QueryOpts) ([]*core.Record, error) {
	cap := opts.Limit
	if cap <= 0 {
		cap = 50
	}

	var recs []*core.Record
	var err error
	if opts.Type != "" {
		recs, err = ListByTypeStatus(app, opts.Type, StatusActive)
	} else {
		recs, err = app.FindRecordsByFilter("nodes", "status = 'active'", "-updated,-created", 0, 0, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("query: loading nodes: %w", err)
	}

	// Filter by PropMatch (AND across all keys, substring match).
	if len(opts.PropMatch) > 0 {
		filtered := recs[:0]
		for _, r := range recs {
			if matchesProps(r, opts.PropMatch) {
				filtered = append(filtered, r)
			}
		}
		recs = filtered
	}

	if len(recs) > cap {
		recs = recs[:cap]
	}
	return recs, nil
}
```

Both branches fetch with SQL limit `0` (everything), then truncate in Go at
line 48–50. The `PropMatch` branch genuinely NEEDS the full fetch — props live
in a JSON column and are substring-matched in Go (`matchesProps`,
`internal/nodes/query.go:143-151`), so the DB cannot pre-limit without losing
matches. That constraint is settled: leave the full fetch when
`len(opts.PropMatch) > 0`. When `PropMatch` is empty, though, the SQL result
IS the final result set, so the cap can be the SQL limit.

The delegate the type branch uses today — `internal/nodes/nodes.go:228-233`:

```go
// ListByTypeStatus returns nodes of one type in one status, newest first.
func ListByTypeStatus(app core.App, typ, status string) ([]*core.Record, error) {
	return app.FindRecordsByFilter("nodes",
		"type = {:t} && status = {:s}", "-created", 0, 0,
		dbx.Params{"t": typ, "s": status})
}
```

Note the ordering difference: `ListByTypeStatus` sorts `-created`; the
no-type branch of `Query` sorts `-updated,-created`. Step 1 switches the type
branch to a direct `FindRecordsByFilter` with `-updated,-created`, so both
branches order identically. **This is a deliberate, acceptable ordering
change**: a recently-edited old node now sorts to the top of `node_list` /
`node_query` type queries. "Newest first" is the documented intent of both
tools, and most-recently-touched is the more useful reading of "newest".

### `nodeListTool` today — `internal/tools/knowledge.go:411-448`

```go
func nodeListTool(app core.App) agent.Tool {
	allTypes, err := nodes.TypeNames(app)
	if err != nil || len(allTypes) == 0 {
		app.Logger().Warn("node_list: could not load types from registry; falling back to [note]", "error", err)
		allTypes = []string{"note"}
	}
	return agent.Tool{
		Spec: agent.ToolSpecOf("node_list",
			"List active knowledge nodes of a given type (newest first).",
			obj(map[string]any{
				"type": map[string]any{"type": "string", "enum": allTypes, "description": "Node type to list (default note)."},
			}, "type")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("node_list: bad arguments: %w", err)
			}
			typ := args.Type
			if typ == "" {
				typ = "note"
			}
			recs, err := nodes.ListByTypeStatus(app, typ, nodes.StatusActive)
			if err != nil {
				return "", fmt.Errorf("node_list: %w", err)
			}
			if len(recs) == 0 {
				return fmt.Sprintf("No active %s nodes.", typ), nil
			}
			var b strings.Builder
			for _, r := range recs {
				fmt.Fprintf(&b, "- [%s] %s\n", r.Id, r.GetString("title"))
			}
			return b.String(), nil
		},
	}
}
```

No cap, no truncation signal. No existing test asserts `node_list` output
(verified: `grep -rn "node_list" internal/ --include="*_test.go"` returns
nothing; the `"No active skills yet."` / `"No active memories yet."`
assertions in `internal/feature/knowledgecards/*_test.go` belong to UI cards,
not this tool).

### Self-description sentence — `internal/self/knowledge.md:185`

```
  node_list, node_get, and node_drop list, read, and delete them. node_get now also
```

### Repo conventions that apply here

- Errors: `fmt.Errorf("doing x: %w", err)`, return early, no panics in
  library code.
- Structured logging only via `app.Logger()` (slog key/value pairs); no
  `fmt.Print*` in service code. `nodeListTool` already follows this at
  `internal/tools/knowledge.go:414`.
- No global mutable state; pass `core.App` explicitly (both functions
  already do).
- Tests: standard `testing` package, table-driven where it helps.
  PocketBase-dependent tests boot a temp-dir app via
  `storetest.NewApp(t)` (`internal/storetest/storetest.go:18`) — both test
  files in scope already use it. No `time.Sleep`-based synchronization.
- KISS/YAGNI: smallest correct change; no new config knob for the cap — a
  package constant is enough.
- The `nodes` collection has `&core.AutodateField{Name: "updated", OnCreate:
  true, OnUpdate: true}` (see `migrations/1749600000_init.go`), so
  `nodes.Update` bumps `updated` automatically — the ordering test in the
  Test plan relies on this.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Full test gate (the merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all pass |
| Targeted tests (nodes) | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/nodes/ -run TestQuery -count=1` | ok |
| Targeted tests (tools) | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/tools/ -run TestNodeList -count=1` | ok |
| Tours lint | `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` | ok |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |

The host `/tmp` is a small tmpfs and the Go linker OOMs there — always prefix
test runs with `TMPDIR=$HOME/.cache/go-tmp` as shown.

## Scope

**In scope** (the only files you may modify):

- `internal/nodes/query.go` — push the cap into SQL when `PropMatch` is empty
- `internal/nodes/query_test.go` — new cap/ordering tests
- `internal/tools/knowledge.go` — `nodeListTool` cap + truncation line
- `internal/tools/knowledge_test.go` — new `node_list` tests
- `internal/self/knowledge.md` — one-sentence self-description update
- `.tours/17-the-tool-surface.tour`, `.tours/06-memory-and-self-evolution.tour`
  — repoint anchors shifted by the added import line
- `plans/README.md` — status row only

**Out of scope** (do NOT touch, even though they look related):

- `internal/nodes/nodes.go` — `ListByTypeStatus` keeps its unbounded,
  `-created` contract; export, tasks, knowledge cache, CLI, and taskcards
  callers depend on the full set.
- The `PropMatch` branch's full fetch in `Query` — required for in-Go JSON
  prop filtering; do not attempt to pre-limit it.
- `internal/tools/graph.go` (`node_query`, `node_related`) — `node_query`
  already passes a limit through `QueryOpts`; it inherits the SQL bound with
  no edit.
- The `search` tool and `node_get` / `node_drop` tools in
  `internal/tools/knowledge.go` — unchanged.
- `internal/feature/graphcards/network.go` — inherits the SQL bound with no
  edit.

## Git workflow

- Executor runs in an isolated git worktree branched from `origin/main`;
  branch name `advisor/246-node-list-cap-query-limit`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/
  `test`/`chore`); e.g. `perf(nodes): push Query cap into SQL when no prop
  filter applies` is fine as `fix(nodes): …` or `refactor(nodes): …` — pick
  one and keep it conventional.
- Commit per logical unit **with explicit pathspecs** (the main checkout is
  shared by parallel agents — stage only your own files, e.g.
  `git add internal/nodes/query.go internal/nodes/query_test.go`).
- **NEVER push**; the reviewer merges.

## Steps

### Step 1: Bound the SQL fetch in `nodes.Query` when no prop filter applies

Edit `internal/nodes/query.go`, function `Query` (lines 20–52 today). Replace
the load section so that:

1. When `len(opts.PropMatch) == 0`, `cap` is passed as the SQL limit; when a
   prop filter exists, the SQL limit stays `0` (full fetch — the JSON prop
   substring match happens in Go and must see every candidate).
2. The `opts.Type != ""` branch stops delegating to `ListByTypeStatus` and
   calls `FindRecordsByFilter` directly with the same `-updated,-created`
   ordering as the no-type branch, so `ListByTypeStatus` keeps its no-limit,
   `-created` contract for its other callers.

Target shape (the `dbx` import already exists in this file — it is used by
`ActiveSubgraph`):

```go
	cap := opts.Limit
	if cap <= 0 {
		cap = 50
	}

	// With no prop filter the SQL result IS the final set, so the DB can
	// bound the fetch. PropMatch needs every candidate: props live in a JSON
	// column and are substring-matched in Go below.
	sqlLimit := 0
	if len(opts.PropMatch) == 0 {
		sqlLimit = cap
	}

	var recs []*core.Record
	var err error
	if opts.Type != "" {
		recs, err = app.FindRecordsByFilter("nodes",
			"type = {:t} && status = {:s}", "-updated,-created", sqlLimit, 0,
			dbx.Params{"t": opts.Type, "s": StatusActive})
	} else {
		recs, err = app.FindRecordsByFilter("nodes",
			"status = 'active'", "-updated,-created", sqlLimit, 0, nil)
	}
```

Keep everything after the error check unchanged: the `PropMatch` filtering
loop and the final `if len(recs) > cap { recs = recs[:cap] }` truncation (the
truncation is now a no-op in the no-PropMatch branches but still required
after in-Go filtering).

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0, and
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/nodes/ -count=1` → ok (all
existing tests — `TestQueryByType`, `TestQueryAnyType`, `TestQueryPropMatch`,
`TestQueryLimit` — still pass).

### Step 2: Add `nodes.Query` cap/ordering tests

In `internal/nodes/query_test.go` (package `nodes_test`; model after the
existing `TestQueryLimit` at lines 87–101), add:

```go
// TestQueryDefaultCapNewestFirst proves the default cap of 50 applies to the
// type branch and that ordering is -updated,-created: a just-edited old node
// must survive the cap.
func TestQueryDefaultCapNewestFirst(t *testing.T) {
	app := storetest.NewApp(t)

	first, err := nodes.Create(app, "note", "Oldest", "", nodes.StatusActive, nil)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	for i := range 59 {
		if _, err := nodes.Create(app, "note", fmt.Sprintf("N%02d", i), "", nodes.StatusActive, nil); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}
	// Bump the oldest node: its updated timestamp is now the newest, so
	// -updated,-created ordering must keep it inside the 50-record cap.
	body := "edited"
	if _, err := nodes.Update(app, first.Id, nil, &body, nil); err != nil {
		t.Fatalf("Update: %v", err)
	}

	recs, err := nodes.Query(app, nodes.QueryOpts{Type: "note"})
	if err != nil {
		t.Fatalf("Query: %v", err)
	}
	if len(recs) != 50 {
		t.Fatalf("want 50 (default cap), got %d", len(recs))
	}
	found := false
	for _, r := range recs {
		if r.Id == first.Id {
			found = true
		}
	}
	if !found {
		t.Error("just-edited node fell out of the cap — ordering is not -updated,-created")
	}
}
```

Add `"fmt"` to the test file's imports. Check the exact signatures before
writing: `nodes.Create(app core.App, typ, title, body, status string, props
map[string]any) (*core.Record, error)` (`internal/nodes/nodes.go:89`) and
`nodes.Update(app core.App, id string, title, body *string, props
map[string]any) (*core.Record, error)` (`internal/nodes/nodes.go:150`).

The PropMatch branch needs no new test — `TestQueryPropMatch` already covers
it and must stay green (its fetch behavior is unchanged).

**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/nodes/ -run TestQuery -count=1`
→ ok, including `TestQueryDefaultCapNewestFirst`.

### Step 3: Cap `nodeListTool` and append the truncation line

Edit `internal/tools/knowledge.go`:

1. Add `"github.com/pocketbase/dbx"` to the import block (place it in the
   second group, before `"github.com/pocketbase/pocketbase/core"`; gofmt
   keeps it there). **This adds exactly one line — Step 5 repoints the tour
   anchors.**
2. Add a package constant near `nodeListTool`:

   ```go
   // nodeListCap bounds node_list output: tool results enter the model
   // context mid-turn, and an uncapped listing can blow a small local window.
   const nodeListCap = 50
   ```

3. In `nodeListTool` (currently lines 411–448):
   - Change the spec description from
     `"List active knowledge nodes of a given type (newest first)."` to
     `"List active knowledge nodes of a given type (newest first, at most 50)."`
   - Replace the `nodes.ListByTypeStatus(app, typ, nodes.StatusActive)` call
     with `nodes.Query(app, nodes.QueryOpts{Type: typ, Limit: nodeListCap})`
     (keep the same error wrap `fmt.Errorf("node_list: %w", err)` and the
     `"No active %s nodes."` empty case verbatim).
   - After the listing loop, when the result is at the cap, count the real
     total and append the truncation line so the model knows it is looking at
     a slice:

     ```go
     if len(recs) == nodeListCap {
     	total, err := app.CountRecords("nodes",
     		dbx.HashExp{"type": typ, "status": nodes.StatusActive})
     	if err != nil {
     		app.Logger().Warn("node_list: counting nodes for truncation note", "error", err)
     	} else if total > int64(nodeListCap) {
     		fmt.Fprintf(&b, "…showing %d of %d — use node_query or search to narrow.\n", nodeListCap, total)
     	}
     }
     ```

     Notes: `app.CountRecords` returns `(int64, error)` and takes raw
     `dbx.Expression`s — `dbx.HashExp` on the plain `type`/`status` columns is
     the established pattern (see `internal/tasks/tasks.go:248`). Do NOT use a
     `props.*` key here (raw dbx cannot resolve JSON paths — see the comment
     at `internal/seed/seed.go:366-369`). The count runs only when the listing
     hit the cap, so the common small-list case stays one query. A count
     failure degrades to "no truncation note", never fails the tool — the
     listing itself already succeeded.

Behavior note (intended): `node_list` ordering changes from `-created` to
`-updated,-created` (Step 1's type branch). The tool's documented contract is
"newest first", which most-recently-touched still satisfies.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `gofmt -l .` → empty.

### Step 4: Add `node_list` tool tests

In `internal/tools/knowledge_test.go` (package `tools`; the file already
imports `context`, `strings`, `storetest`, and `nodes` — add `"fmt"` if not
present). Model after `TestNodeWriteToolCreatesActiveNode`
(`internal/tools/knowledge_test.go:52`): construct the tool directly and call
`tool.Execute`.

```go
// TestNodeListToolCapsOutput proves node_list stops at 50 entries and tells
// the model the listing is truncated (with the real total).
func TestNodeListToolCapsOutput(t *testing.T) {
	app := storetest.NewApp(t)
	for i := range 60 {
		if _, err := nodes.Create(app, "note", fmt.Sprintf("Note %02d", i), "", nodes.StatusActive, nil); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}
	tool := nodeListTool(app)

	out, err := tool.Execute(context.Background(), `{"type":"note"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	entries := 0
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "- [") {
			entries++
		}
	}
	if entries != 50 {
		t.Fatalf("want exactly 50 entries, got %d", entries)
	}
	if !strings.Contains(out, "showing 50 of 60") {
		t.Fatalf("missing truncation note, got tail: %q", out[max(0, len(out)-120):])
	}
}

// TestNodeListToolSmallListNoTruncationNote proves an under-cap listing has
// no truncation line.
func TestNodeListToolSmallListNoTruncationNote(t *testing.T) {
	app := storetest.NewApp(t)
	for i := range 3 {
		if _, err := nodes.Create(app, "note", fmt.Sprintf("Note %d", i), "", nodes.StatusActive, nil); err != nil {
			t.Fatalf("Create %d: %v", i, err)
		}
	}
	tool := nodeListTool(app)

	out, err := tool.Execute(context.Background(), `{"type":"note"}`)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if strings.Count(out, "- [") != 3 {
		t.Fatalf("want 3 entries, got: %q", out)
	}
	if strings.Contains(out, "showing") {
		t.Fatalf("unexpected truncation note on a 3-item list: %q", out)
	}
}
```

("note" is a built-in registered node type in the test app's schema — the
same fixture pattern the existing `internal/nodes/query_test.go` tests use.)

**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/tools/ -run TestNodeList -count=1`
→ ok, 2 new tests pass.

### Step 5: Repoint the shifted tour anchors

Step 3 added exactly one import line near the top of
`internal/tools/knowledge.go`, shifting every anchor below it down by one.
First confirm the shift: `git diff internal/tools/knowledge.go` must show
exactly **one** added line in the import block (plus the nodeListTool-area
changes, which sit below every anchored line). Then edit the two JSON tour
files, changing ONLY the `"line"` values of steps whose `"file"` is
`internal/tools/knowledge.go`:

- `.tours/17-the-tool-surface.tour`: the step at `"line": 23` (the
  `KnowledgeTools` comment block) → `24`; the step at `"line": 43` (the
  `ProposalMarker` const) → `44`.
- `.tours/06-memory-and-self-evolution.tour`: the step at `"line": 43`
  (`ProposalMarker`) → `44`.

Sanity-check the new anchors point at the same code they did before:
line 24 of the edited file must still be inside the `// KnowledgeTools gives
the model its memory and skill verbs…` comment block, and line 44 must still
sit on/next to `const ProposalMarker = "\x00balaur-proposal:"`. Also check
the tour step at `.tours/17-the-tool-surface.tour` line 23's prose still
holds ("It returns eleven tools") — this plan does not change the tool count,
so the prose needs no edit.

If the diff shows a shift other than exactly +1 (e.g. gofmt regrouped
imports), recompute the anchor lines from the live file instead of blindly
adding one.

**Verify**: `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok.

### Step 6: Update the self-description

`internal/self/knowledge.md` is the running binary's own description of its
capabilities; a tool-behavior change like this must land in the same change.
Edit line 185 from:

```
  node_list, node_get, and node_drop list, read, and delete them. node_get now also
```

to:

```
  node_list (newest 50, with a truncation note when more exist), node_get, and
  node_drop list, read, and delete them. node_get now also
```

(Reflow the surrounding paragraph only as far as needed; do not rewrite
neighbouring sentences.)

**Verify**: `grep -n "truncation note" internal/self/knowledge.md` → exactly
one match, on the node_list sentence.

### Step 7: Full gate

Run, in order:

1. `gofmt -l .` → empty output
2. `go vet ./...` → exit 0
3. `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output, exit 0
4. `CGO_ENABLED=0 go build ./...` → exit 0
5. `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0, all pass
6. `git status --porcelain` → only in-scope files (plus `plans/README.md`)

Commit per logical unit with explicit pathspecs, e.g.:

- `perf(nodes): push Query cap into SQL when no prop filter applies`
  (`internal/nodes/query.go`, `internal/nodes/query_test.go`)
- `perf(tools): cap node_list at 50 with a truncation note`
  (`internal/tools/knowledge.go`, `internal/tools/knowledge_test.go`,
  `internal/self/knowledge.md`, the two `.tours/*.tour` files)

## Test plan

New tests (all use `storetest.NewApp(t)`; no real model, no network, no
`time.Sleep`):

- `internal/nodes/query_test.go` — `TestQueryDefaultCapNewestFirst`: 60
  active notes, oldest one edited last → `Query{Type:"note"}` returns exactly
  50 AND includes the edited node (proves both the default cap on the type
  branch and the `-updated,-created` ordering). Model after the existing
  `TestQueryLimit`.
- `internal/tools/knowledge_test.go` — `TestNodeListToolCapsOutput`: 60
  active notes → exactly 50 `- [` entry lines plus a line containing
  `showing 50 of 60`. `TestNodeListToolSmallListNoTruncationNote`: 3 notes →
  3 entries, no `showing` substring. Model after
  `TestNodeWriteToolCreatesActiveNode` in the same file.

Existing tests that must stay green (they pin the unchanged behavior):

- `TestQueryByType`, `TestQueryAnyType`, `TestQueryPropMatch`,
  `TestQueryLimit` in `internal/nodes/query_test.go` — consent filter
  (active-only), PropMatch semantics, explicit Limit.
- The whole `internal/tools` package (other tools untouched).

Verification: `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0,
including the 3 new tests.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `gofmt -l .` → empty; `go vet ./...` → exit 0; staticcheck → no output
- [ ] `CGO_ENABLED=0 go build ./...` → exit 0
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0;
      `TestQueryDefaultCapNewestFirst`, `TestNodeListToolCapsOutput`, and
      `TestNodeListToolSmallListNoTruncationNote` exist and pass
- [ ] `grep -n "ListByTypeStatus" internal/nodes/query.go internal/tools/knowledge.go`
      → no matches (both call sites migrated)
- [ ] `grep -n "nodeListCap" internal/tools/knowledge.go` → at least 3
      matches (const + Query limit + truncation check)
- [ ] `grep -rn "func ListByTypeStatus" internal/nodes/nodes.go` → still
      present and byte-identical to the excerpt above
      (`git diff internal/nodes/nodes.go` → empty)
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok
- [ ] `git status --porcelain` shows changes ONLY in: `internal/nodes/query.go`,
      `internal/nodes/query_test.go`, `internal/tools/knowledge.go`,
      `internal/tools/knowledge_test.go`, `internal/self/knowledge.md`,
      `.tours/17-the-tool-surface.tour`,
      `.tours/06-memory-and-self-evolution.tour`, `plans/README.md`
- [ ] `plans/README.md` status row for 246 updated (unless the dispatching
      reviewer maintains the index)

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows `internal/nodes/query.go` changed since `077318a`
  and the `Query` function no longer matches the excerpt above — plan 250
  (or other parallel work) may have landed a conflicting edit; reconcile with
  the reviewer instead of merging by hand.
- Any test outside the two in-scope test files asserts `node_list` output
  verbatim or asserts `nodes.Query` result ordering/size in a way the cap or
  the `-updated,-created` type-branch ordering breaks (search first:
  `grep -rn "node_list\|nodes.Query(" --include="*_test.go" internal/`).
  Reconcile those tests explicitly in your report — do not silently rewrite
  them.
- The fix appears to require changing `ListByTypeStatus` itself or the
  PropMatch full-fetch — both are explicitly settled out of scope.
- `TestQueryDefaultCapNewestFirst` fails intermittently on the
  "just-edited node fell out of the cap" assertion — that would mean the
  `updated` autodate is not strictly later than the 60 creates on this host
  (millisecond-tie). Report it; do not add `time.Sleep` to paper over it.
- Step 5's `git diff` on `internal/tools/knowledge.go` shows the import block
  grew by anything other than exactly one line, AND recomputing anchors from
  the live file still leaves a tour step pointing at the wrong construct.
- A verification fails twice after a reasonable fix attempt.

## Maintenance notes

- **Ordering semantics are now uniform**: every `nodes.Query` branch sorts
  `-updated,-created`. `ListByTypeStatus` still sorts `-created` — if a
  future change consolidates them, its callers (export mirror determinism,
  task loaders, knowledge cache) must be re-checked against the ordering
  swap.
- The `PropMatch` branch still fetches ALL active rows and filters in Go —
  that is inherent to JSON-prop substring matching, not an oversight. If
  prop queries ever get slow at scale, the fix is a schema/index change
  (typed prop columns or FTS), not a limit push.
- `nodeListCap` (50) intentionally matches `nodes.Query`'s default cap and
  `node_query`'s documented default; if one moves, move them together or the
  truncation hint ("use node_query … to narrow") stops being an upgrade.
- Reviewer scrutiny: the truncation-note count query runs only when
  `len(recs) == nodeListCap` — confirm the boundary case (exactly 50 active
  nodes) emits no note (`total > int64(nodeListCap)` is false).
- Deferred (deliberately): no `limit` argument on the `node_list` tool spec —
  `node_query` already exposes one; adding a second knob duplicates surface
  (YAGNI).
