# Plan 039: FTS5 memory recall — sidecar index, LIKE fallback

> **Executor instructions**: Follow this plan step by step; run every
> verification and confirm the expected result. STOP conditions are binding.
> Commit on branch `advisor/039-fts5-recall`. SKIP updating
> `plans/readme.md`. Audit every report claim against a tool result.
>
> **Drift check (run first)**: `git diff --stat 3ea002d..HEAD -- internal/search internal/knowledge internal/turn main.go go.mod`
> Any drift → compare excerpts; on mismatch, STOP.

## Status

- **Priority**: P2 · **Effort**: M–L · **Risk**: MED (new product dependency
  path; recall correctness affects every chat turn)
- **Depends on**: none (direction finding A, deferred since the first cycle)
- **Category**: direction · **Planned at**: commit `3ea002d`, 2026-06-12

## Why this matters

Recall today is "3 longest words, LIKE match" — it misses morphology, ranks
by importance only, and the docs have promised FTS5 for two cycles. The
`internal/search` spike already locked the driver:
`github.com/ncruces/go-sqlite3` (SQLite compiled to WASM via wazero — FTS5
included, CGO-free, single-binary story survives; `go.mod:7`, currently
test-only). The spike names two integration paths; **this plan takes the
"separate disposable index DB" path**, NOT the app-wide
`pocketbase.Config.DBConnect` driver swap — PocketBase keeps its default
driver, and the FTS index is a rebuildable sidecar file. AGENTS.md treats
app-wide PocketBase changes as repo-wide risk; the sidecar confines the new
driver to one package. **Embeddings stay out of scope** — `llm.Client.Embed`
keeps its zero callers; this plan is lexical recall only.

## Current state

- `internal/search/fts5_test.go:1-12` (the whole package today):

  ```go
  // PocketBase's default driver (modernc.org/sqlite) ships WITHOUT FTS5.
  // github.com/ncruces/go-sqlite3 (SQLite compiled to WASM, run via wazero)
  // includes FTS5 and stays CGO-free … When search lands, the driver moves
  // into the product via pocketbase.Config.DBConnect … or backs a separate
  // disposable index DB.
  ```

  The test proves `CREATE VIRTUAL TABLE … USING fts5` works on driver
  `"sqlite3"` registered by `ncruces/go-sqlite3/driver` + `/embed`.
- The recall seam — `internal/knowledge/knowledge.go:229-251` `SearchActive`:
  builds `title ~ {:q} || content ~ {:q} || when_to_use ~ {:q} || category ~ {:q}`
  clauses per term, `status = 'active'`, ordered `-importance,-created`,
  capped at `limit`. Its doc comment: "FTS5/embedding recall is roadmap (see
  internal/search) — the call site won't change."
- Term extraction — `internal/knowledge/context.go:89-113` `recallTerms`:
  words ≥ 4 chars, unique, 3 longest. Called by `BuildContext`
  (`context.go:45`, `SearchActive(app, recallTerms(userMessage), recallLimit)`);
  `BuildContext` is called from `internal/turn/turn.go:93` on every owner turn.
- `balaur memory recall` (CLI) also flows through `SearchActive` — it
  inherits whatever this plan does.
- App bootstrap: `main.go:32` `app := pocketbase.New()`. PocketBase exposes
  `app.Store()` (a concurrent map) — the sanctioned way to carry an
  app-scoped singleton without module globals (AGENTS.md forbids global
  derived state). Data dir: resolve via `app.DataDir()`.
- Memories change through the `memories` collection (knowledge package
  writes + owner edits via web). PocketBase record hooks
  (`OnRecordAfterCreateSuccess/AfterUpdateSuccess/AfterDeleteSuccess`,
  scoped to a collection name) are the single seam that sees every write —
  check the exact hook API for the pinned PocketBase version by reading
  existing hook usage in the repo (`grep -rn "OnRecord" internal/ main.go`)
  and PocketBase's source in the module cache if needed.

## Commands

| Purpose | Command | Expect |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tests | `go test ./...` | ok |
| Vet/fmt | `go vet ./...` / `gofmt -l .` | clean |
| Deps | `go mod tidy && git diff go.mod go.sum` | only expected moves |

## Scope

**In scope**: `internal/search/` (real package: index + tests; keep the
spike test), `internal/knowledge/knowledge.go` (+`knowledge_test.go`),
`main.go` (wire hooks + rebuild on serve), `go.mod`/`go.sum` (tidy),
`internal/self/knowledge.md`, `DESIGN.md` ledger + roadmap line, `README.md`
(only if it documents recall), `.gitignore` (the index file pattern if the
data dir pattern doesn't already cover it — `pb_data/` is ignored; the index
lives under it, so likely no change).
**Out of scope**: embeddings / `Embed()`; `recallTerms` semantics (keep
3-longest extraction — FTS ranking does the rest); `internal/turn`; the web
layer; any `pocketbase.Config.DBConnect` change.

## Git workflow

Branch `advisor/039-fts5-recall`; commit
`feat(search): FTS5 memory recall — sidecar index with LIKE fallback`.

## Steps

### Step 1: The index (`internal/search/index.go` + `index_test.go`)

```go
type Index struct { db *sql.DB }

// Open opens (creating if absent) the sidecar index at dir/search.db.
func Open(dir string) (*Index, error)
func (ix *Index) Close() error
// Rebuild drops and refills the whole index from the app's active memories.
// Idempotent; the file is disposable by design.
func (ix *Index) Rebuild(app core.App) error
func (ix *Index) Upsert(rec *core.Record) error  // active memories only; non-active = delete
func (ix *Index) Delete(id string) error
// Query returns memory ids ranked by bm25 for the given terms (OR semantics),
// capped at limit. Terms are sanitized: each term is quoted as a string
// token («"term"») so FTS5 query syntax in user text cannot inject operators.
func (ix *Index) Query(terms []string, limit int) ([]string, error)
```

Schema: `CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts USING
fts5(id UNINDEXED, title, content, when_to_use, category)`. Upsert =
delete-by-id + insert. Query:
`SELECT id FROM memories_fts WHERE memories_fts MATCH ? ORDER BY rank LIMIT ?`
with the match string built from quoted terms joined by ` OR `.
Driver imports exactly as the spike test does. The wazero runtime has a
startup cost — open ONCE per process (Step 3), never per query.

Tests (model after the spike's style): open in `t.TempDir()`; rebuild from a
seeded temp app (use the `internal/store` test helpers — check import
direction: search may import store for test helpers ONLY in _test.go files;
if that creates a cycle, seed via a minimal fake or take
`[]*core.Record`-shaped data — adapt and document); query ranking sanity
(a term hits title > a term buried in content is NOT asserted — assert
membership and limit, not bm25 internals); FTS-operator injection attempt
(`term" OR "x`) returns without error; non-active records absent after
Upsert of an archived record.

### Step 2: SearchActive consults the index, falls back to LIKE

In `internal/knowledge/knowledge.go`, at the top of `SearchActive`: fetch the
index from `app.Store()` (key, e.g. `"balaur.searchIndex"` — define the key
constant in `internal/search` and read it from knowledge; knowledge already
may import search… check direction: search must not import knowledge — keep
search dependency-free of domain packages; Rebuild needs memories access, so
Rebuild takes `app core.App` and queries the collection directly, which is
fine: search becomes a small domain-adjacent package). If the store value is
present and `Query` succeeds with ≥1 id: `FindRecordsByIds` then filter
`status == "active"` defensively and preserve the FTS rank order, cap at
limit, return. On ANY error, a nil index, or zero ids: fall through to the
existing LIKE body UNCHANGED (deterministic, offline-safe default —
AGENTS.md). Update the function's doc comment honestly.

Tests: with an index in the store, recall finds a stemmed/partial-word case
LIKE would miss (e.g. query term "flour" matching "flour," with punctuation —
pick a case FTS tokenization genuinely wins); with no index in the store,
behavior is byte-identical to today (existing tests must pass unmodified).

### Step 3: Wiring in main.go

In the serve bootstrap (find where `OnServe` / serve-event binding happens —
`web.Register` is mounted there): open the index at
`filepath.Join(app.DataDir(), "search.db")`… verify `app.DataDir()` is the
right call for the pinned PocketBase version (grep the repo / module). Put
it in `app.Store()`, `Rebuild` it (log a warn and continue WITHOUT the index
on any error — Balaur must boot even if the index is corrupt; delete+retry
once is allowed), and register the three record hooks on the `memories`
collection calling Upsert/Delete (errors logged, never fatal — the index is
eventually consistent via next boot's Rebuild). Close on app termination if
a hook exists for it (check `OnTerminate`).

**Verify**: `go test ./...` ok; `CGO_ENABLED=0 go build ./...` exit 0;
`go mod tidy` moves ncruces deps from test-only to product (inspect
`git diff go.mod` — expect no version changes, possibly indirect-flag
changes).

### Step 4: Docs truth

- `DESIGN.md` §3: move FTS5 recall from "Roadmap" to "True today" — phrase:
  "FTS5 memory recall (bm25-ranked sidecar index, rebuilt on boot, synced on
  write; LIKE fallback when the index is unavailable)". Roadmap keeps
  "embedding recall".
- `internal/self/knowledge.md`: same truth, one sentence.
- `README.md`: update only if it describes the LIKE recall (grep "recall").

**Verify**: `grep -n "FTS5" DESIGN.md internal/self/knowledge.md` → in
True-today/capabilities, not roadmap-only.

## Test plan

As embedded in Steps 1–2, plus: the full suite green; and an integration
test in `internal/knowledge` that goes seed-app → build index → Rebuild →
SearchActive returns the seeded memory by a content word (proving the whole
chain without the web layer).

## Done criteria

- [ ] `go test ./internal/search/... ./internal/knowledge/...` ok, including
      the injection and fallback tests
- [ ] With no index present, SearchActive's behavior and its existing tests
      are unchanged
- [ ] Boot survives a corrupt/missing index file (test: write garbage to
      search.db in a temp data dir, Open/Rebuild path logs + continues)
- [ ] `CGO_ENABLED=0 go build ./...` exit 0 (the wasm driver keeps it true)
- [ ] DESIGN.md ledger moved; knowledge.md updated
- [ ] All gates clean; only in-scope files (`git status`)

## STOP conditions

- The pinned PocketBase version's hook or Store API differs from what the
  plan assumes in a way that needs >20 lines of adaptation — report the real
  API shape first.
- The wazero/ncruces runtime measurably breaks the CGO-free build or adds
  >15MB to the binary — measure (`go build`, compare sizes) and report
  before committing.
- Any temptation to swap `pocketbase.Config.DBConnect` — out of scope.

## Maintenance notes

- The index is disposable: deleting `pb_data/search.db` is always safe
  (rebuilt on boot). Say so in knowledge.md.
- Embedding recall (the `Embed()` seam) would slot in as a SECOND ranking
  stage behind the same SearchActive seam — new plan when wanted.
- Reviewer: the FTS quoting in Query is the security surface (user text →
  match string); check the quoting escapes embedded `"` per FTS5 rules
  (double them).
