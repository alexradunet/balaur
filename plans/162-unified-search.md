# Plan 162: Unified full-text search across active nodes

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` (lowercase on disk) — unless a reviewer dispatched you
> and told you they maintain the index.
>
> **Sandbox note**: in a TLS-intercepting sandbox (Hyperagent), Go commands
> need the GOPROXY shim — see `docs/hyperagent-sandbox.md`. GOSUMDB stays on;
> never weaken checksum verification to work around a proxy.
>
> **Drift check (run first)**:
> `git diff --stat 72fd762..HEAD -- internal/search/index.go internal/search/index_test.go internal/knowledge/knowledge.go internal/tools/knowledge.go internal/cli/knowledge.go internal/cli/cli.go main.go internal/self/knowledge.md`
> Plan 160 is a hard dependency and WILL have changed several of these paths
> (it removes the `memories`/`skills` collections and folds them into `nodes`).
> That is expected. Before touching anything, read the live `internal/search/index.go`,
> `internal/tools/knowledge.go`, and `main.go` and reconcile them against the
> "Current state" excerpts below — the excerpts are stamped at commit `72fd762`
> (BEFORE 160). If 160 has NOT landed (the `nodes` collection does not exist —
> see Step 0), that is a STOP condition: 162 cannot index a collection that does
> not exist yet.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: plans/160-*.md (the `nodes`+`edges` baseline migration — HARD dependency). Soft-benefits-from plans/161-*.md (wikilink edges) but does NOT require it.
- **Category**: migration
- **Planned at**: commit `72fd762`, 2026-06-23
- **Issue**: —

## Why this matters

Plan 160 folds `memories`, `skills`, notes, typed objects, and journal entries
into a single `nodes` collection and **removes the standalone `memories`/`skills`
collections** — but the dossier deliberately scopes 160 to NOT touch
`internal/search`. The current FTS5 index (`internal/search/index.go`) indexes a
table named `memories_fts` sourced from the `memories` collection. **The moment
160 lands, that collection is gone, so the index code in this package references
a collection that no longer exists** — `Rebuild` and the `memories` record hooks
in `main.go` break. This plan is what makes full-text recall work again on the
new schema: it generalizes the index from one collection (`memories`) to one
collection of many *types* (`nodes`, kind = the node `type`), filters strictly
to `status=active` so agent-proposed-but-unapproved knowledge never surfaces
(the consent spine), repoints the write hooks to `nodes`, **rewrites the recall
resolution path so a search id maps to a `nodes` record of ANY type (not the
removed `memories` collection)**, fixes the `recall` tool to read node
`type`/`body` instead of the gone `category`/`content`, and adds a cross-type
`search` agent tool + a `balaur node search` CLI command so the owner and the
model can "find anything" across notes, memories, skills, journal, and typed
objects in one query.

**Ownership boundary with 160 (read this — it is the crux of this plan).** Plan
160 removes the `memories`/`skills` collections and folds them into `nodes`. The
dossier (`plans/160-163-knowledge-graph-design-dossier.md:83-93`, item 160)
assigns 160 the *collection removal* and the `internal/search` write **hooks**.
It does NOT assign 160 the cross-type rewrite of `SearchActive`'s
id→record resolution (`FindRecordsByIds(string(Memory), ids)` →
`FindRecordsByIds("nodes", ids)`) nor the `recall` tool's field reads — and the
LIVE code (`internal/knowledge/knowledge.go:229-285`,
`internal/tools/knowledge.go:108-138`) is hard-coded to the `memories`
collection and `category`/`content` fields in BOTH the FTS path and the LIKE
fallback. **That cross-type rewrite is 162's job** (it is what makes "find
anything" actually return mixed-type hits), so `internal/knowledge/knowledge.go`
(the `SearchActive` body only) and `recallTool` are IN SCOPE here. Step 5 owns
the `SearchActive` rewrite; Step 6 owns the `recall` fix. **Reconcile, do not
collide**: if 160 already rewrote `SearchActive` to resolve against `nodes`
(read it first per the drift check), keep 160's version and only confirm it in
Step 5; do not duplicate. If 160 did NOT, 162 makes the change. Either way the
done criteria below assert the final state: no `FindRecordsByIds(string(Memory)`
and no `GetString("category")`/`GetString("content")` on the recall path.

The win when this lands: one search surface over all knowledge, the consent
filter is enforced at the index boundary (proposals are never indexed, never
returned, never leave the box), and the deterministic LIKE fallback still
guarantees offline recall when the sidecar is unavailable.

## Current state

`search.db` is a disposable sidecar SQLite database, rebuilt from PocketBase on
every boot (`registerSearchIndex` in `main.go`). **There is NO data migration in
this plan** — widening the FTS schema just changes what the next boot rebuilds.
Deleting `pb_data/search.db` is always safe.

Relevant files, each with its role:

- `internal/search/index.go` — the FTS5 sidecar index: `Open` (creates the
  table), `Rebuild` (refills from active records), `Upsert`/`Delete` (per-record
  hooks), `Query` (bm25-ranked, injection-safe). **This plan owns all changes
  to this package.**
- `internal/search/index_test.go` — table/helper tests for the index. Extended
  here for multi-kind rebuild/query/upsert + a consent-filter test.
- `internal/search/fts5_test.go` — the driver spike test; **do not touch** (it
  only proves the ncruces/wazero FTS5 driver works).
- `main.go` — `registerSearchIndex` opens the index, stores it in `app.Store()`,
  rebuilds it, and binds the create/update/delete record hooks. The hooks
  currently bind to the `memories` collection; this plan repoints them to `nodes`.
- `internal/knowledge/knowledge.go` — `SearchActive` consults the index (FTS5
  fast path) and falls back to LIKE. **IN SCOPE for the `SearchActive` body
  only** (Step 5): its id→record resolution and LIKE fallback are hard-coded to
  the `memories` collection (`FindRecordsByIds(string(Memory), ids)` and a
  `content`/`category` LIKE clause) and must be repointed to `nodes`/`type`/`body`
  so search returns mixed-type hits. Do NOT touch the rest of this file (160 owns
  the lifecycle/propose/transition code that references the removed collections).
- `internal/tools/knowledge.go` — agent tools (`remember`, `recall`, `skill`,
  `propose_skill`) via `KnowledgeTools`. This plan ADDS a `search` tool here AND
  fixes `recallTool` (Step 6), whose body still reads the removed `category`/
  `content` fields and calls `Touch(app, knowledge.Memory, m)`.
- `internal/cli/knowledge.go` + `internal/cli/cli.go` — CLI subcommands. The repo
  uses PARENT commands (`memoryCmd`/`skillCmd`) with subcommands, registered via
  `root.AddCommand(...)` in `cli.go:54`. This plan ADDS a `node` parent command
  with a `search` subcommand (`balaur node search <terms…>`), registered the same
  way, mirroring the JSON-envelope pattern of the existing commands.
- `internal/self/knowledge.md` — the running binary's self-description; updated
  to describe unified search over nodes.

### Excerpt: the FTS table + Rebuild source, today (`internal/search/index.go:34-90`, at `72fd762`)

```go
	if _, err := db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS memories_fts
		USING fts5(id UNINDEXED, title, content, when_to_use, category)
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("search: create fts5 table: %w", err)
	}
	return &Index{db: db}, nil
}
```

```go
func (ix *Index) Rebuild(app core.App) error {
	tx, err := ix.db.Begin()
	...
	if _, err := tx.Exec(`DELETE FROM memories_fts`); err != nil {
		...
	}

	recs, err := app.FindRecordsByFilter("memories", "status = 'active'", "", 0, 0, nil)
	...
	stmt, err := tx.Prepare(`INSERT INTO memories_fts(id, title, content, when_to_use, category) VALUES (?, ?, ?, ?, ?)`)
	...
	for _, r := range recs {
		if _, err := stmt.Exec(
			r.Id,
			r.GetString("title"),
			r.GetString("content"),
			r.GetString("when_to_use"),
			r.GetString("category"),
		); err != nil {
			...
		}
	}
	...
}
```

### Excerpt: Upsert + Query, today (`internal/search/index.go:94-167`, at `72fd762`)

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
		`INSERT INTO memories_fts(id, title, content, when_to_use, category) VALUES (?, ?, ?, ?, ?)`,
		rec.Id,
		rec.GetString("title"),
		rec.GetString("content"),
		rec.GetString("when_to_use"),
		rec.GetString("category"),
	)
	...
}
```

```go
func (ix *Index) Query(terms []string, limit int) ([]string, error) {
	if len(terms) == 0 {
		return nil, nil
	}
	quoted := make([]string, 0, len(terms))
	for _, t := range terms {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		// Escape embedded double-quotes by doubling them (FTS5 spec §3.1).
		t = strings.ReplaceAll(t, `"`, `""`)
		quoted = append(quoted, `"`+t+`"`)
	}
	if len(quoted) == 0 {
		return nil, nil
	}
	matchExpr := strings.Join(quoted, " OR ")

	rows, err := ix.db.Query(
		`SELECT id FROM memories_fts WHERE memories_fts MATCH ? ORDER BY rank LIMIT ?`,
		matchExpr, limit,
	)
	...
}
```

**`Query`'s body is the injection-safe quoting + bm25 `ORDER BY rank` contract.
Preserve it EXACTLY — only the table name in the `SELECT … FROM <table> … MATCH
<table>` line changes.**

### Excerpt: the record hooks in `main.go` (`main.go:232-256`, at `72fd762`)

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

	app.OnRecordAfterCreateSuccess("memories").BindFunc(upsertHook)
	app.OnRecordAfterUpdateSuccess("memories").BindFunc(upsertHook)
	app.OnRecordAfterDeleteSuccess("memories").BindFunc(deleteHook)
```

### Excerpt: `SearchActive`'s memory-only id resolution + LIKE fallback (`internal/knowledge/knowledge.go:229-285`, at `72fd762`)

```go
func SearchActive(app core.App, terms []string, limit int) ([]*core.Record, error) {
	// --- FTS5 fast path ---
	if raw, ok := app.Store().GetOk(search.StoreKey); ok {
		if ix, ok := raw.(*search.Index); ok && ix != nil {
			ids, err := ix.Query(terms, limit)
			if err == nil && len(ids) > 0 {
				recs, err := app.FindRecordsByIds(string(Memory), ids)
				if err == nil {
					// Filter defensively: only active (the index may lag briefly).
					var active []*core.Record
					for _, r := range recs {
						if r.GetString("status") == StatusActive {
							active = append(active, r)
						}
					}
					...
				}
			}
		}
	}

	// --- LIKE fallback (unchanged) ---
	params := dbx.Params{}
	var clauses []string
	for i, t := range terms {
		...
		clauses = append(clauses,
			fmt.Sprintf("title ~ {:%[1]s} || content ~ {:%[1]s} || when_to_use ~ {:%[1]s} || category ~ {:%[1]s}", key))
	}
	if len(clauses) == 0 {
		return nil, nil
	}
	return app.FindRecordsByFilter(
		string(Memory),
		"status = 'active' && ("+strings.Join(clauses, " || ")+")",
		"-importance,-created", limit, 0,
		params,
	)
}
```

**Both the FTS path (`FindRecordsByIds(string(Memory), ids)`) and the LIKE
fallback (`FindRecordsByFilter(string(Memory), …)` with `content`/`category`)
target the `memories` collection. Step 5 repoints both to `nodes`/`type`/`body`.**

### Excerpt: the `recall` agent tool, today (`internal/tools/knowledge.go:108-138`, at `72fd762`)

```go
func recallTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("recall",
			"Search your approved memories for terms the automatic context may have missed.",
			obj(map[string]any{
				"terms": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "1-3 search terms."},
			}, "terms")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Terms []string `json:"terms"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("recall: bad arguments: %w", err)
			}
			recs, err := knowledge.SearchActive(app, args.Terms, 8)
			...
		},
	}
}
```

`KnowledgeTools` (`internal/tools/knowledge.go:19-26`) returns the slice; add the
new tool there:

```go
func KnowledgeTools(app core.App) []agent.Tool {
	return []agent.Tool{
		rememberTool(app),
		recallTool(app),
		skillTool(app),
		proposeSkillTool(app),
	}
}
```

### Conventions to honor (inlined — the executor has not read these docs)

- **The unified FTS table is fixed by the locked design** (from the design
  dossier and the cross-plan SHARED-NAMES contract): exactly one FTS5 table
  named `knowledge_fts` with columns `(id UNINDEXED, kind UNINDEXED, title,
  content, extra)`. `kind` = the node `type` (note/memory/skill/journal/person/…).
  Index ONLY `status=active` nodes. Use these exact column names so 163 (graph)
  and any future search consumer agree.
- **Consent filter is mandatory and non-negotiable**: the index source set is
  `nodes` filtered to `status = 'active'`. A node whose status is anything else
  (`proposed`, `rejected`, `archived`) must NEVER be indexed and must be Deleted
  from the index if its status flips away from active. This is the spine that
  lets the agent propose knowledge without poisoning search/context. Search must
  never surface a proposed node.
- **Keep the driver + quoting + ranking EXACTLY**: the ncruces/go-sqlite3 wazero
  FTS5 driver (imported as `_ "github.com/ncruces/go-sqlite3/driver"`), the
  double-quote-doubling injection guard in `Query`, and `ORDER BY rank` (bm25)
  are unchanged. You are widening the table and the source set, not rewriting the
  engine.
- **Errors are values** (AGENTS.md): wrap with `fmt.Errorf("…: %w", err)`,
  return early, no panics in library code. Match the existing `search:` error
  prefixes.
- **Structured logging only**: `app.Logger().Warn(...)` with key/value pairs, as
  the existing hooks already do. No `fmt.Print*`/`log.Printf` in service code.
- **CLI envelope pattern**: CLI subcommands return a value through
  `run(app, cliKind, func(...) (any, error))` and render records to
  `map[string]any` (see `memoryJSON` / `statusListCmd` in
  `internal/cli/knowledge.go:13-92`). Match that shape; do not hand-roll JSON.
- **Agent tool pattern**: a tool is an `agent.Tool{Spec: agent.ToolSpecOf(name,
  desc, schema), Execute: func(ctx, argsJSON) (string, error)}`; build the schema
  with the `obj(...)`/`str(...)` helpers in `internal/tools`. Match `recallTool`.
- **`nodes` field names (from plan 160, the SHARED contract)**: `type` (select),
  `title` (text), `body` (text/markdown), `status` (select), `props` (json).
  This plan reads `type`/`title`/`body`/`status` off node records. **If 160's
  field for the long markdown text is NOT named `body`, that is a STOP condition
  — re-read 160 and use 160's real field name; do not guess.**

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test (search pkg) | `go test ./internal/search/...` | all pass |
| Test (tools pkg) | `go test ./internal/tools/...` | all pass |
| Test (knowledge pkg) | `go test ./internal/knowledge/...` | all pass |
| Test (all) | `go test ./...` | all pass |
| Test (env-scrubbed) | `env -u BALAUR_OS_ACCESS -u BALAUR_SOURCE -u BALAUR_MAX_STEPS go test ./...` | all pass |
| Format | `gofmt -l internal/search internal/tools internal/cli internal/knowledge main.go` | no output |
| Diff hygiene | `git diff --check` | no output |
| Confirm nodes schema (Step 0) | `grep -rn "\"nodes\"" migrations/` | at least one match (the 160 baseline) |

## Suggested executor toolkit

- Use the `go-standards` skill when writing the new Go (error `%w` wrapping,
  slog logging, table-driven tests, fake seams).
- Recon rule: `graphify-out/graph.json` exists — orient with
  `graphify query "<question>"` / `graphify explain "<concept>"` before reading
  source, then Read the exact files to confirm line numbers (160 will have moved
  them).
- Inspect live records if needed with the `pocketbase-api` skill (e.g. confirm
  the `nodes` collection schema and that proposed nodes exist to test against).

## Scope

**In scope** (the only files you should modify or create):

- `internal/search/index.go` — generalize `memories_fts` → `knowledge_fts`;
  widen columns to `(id, kind, title, content, extra)`; switch the Rebuild source
  from the `memories` collection to `nodes` filtered to `status='active'`; carry
  `kind` (the node `type`) through Upsert/Rebuild; update the table name in
  `Delete` and `Query`.
- `internal/search/index_test.go` — extend with multi-kind rebuild/query/upsert
  tests + a consent-filter test (a proposed node is NEVER returned).
- `main.go` — repoint the three record hooks from `"memories"` to `"nodes"`
  (the `upsertHook`/`deleteHook` bodies are unchanged); update the comment on
  `registerSearchIndex` to say "active nodes" instead of "active memories".
- `internal/knowledge/knowledge.go` — **`SearchActive` body only**: repoint the
  FTS id resolution `FindRecordsByIds(string(Memory), ids)` →
  `FindRecordsByIds("nodes", ids)` and rewrite the LIKE fallback to query
  `nodes` with `title`/`body` (dropping the `content`/`category` clause). Do NOT
  touch any other function in this file.
- `internal/tools/knowledge.go` — add a `searchTool` (cross-type search) and
  include it in `KnowledgeTools`; **also fix `recallTool`** to read `type`/`body`
  off node records and to stop calling `Touch(app, knowledge.Memory, m)` against
  the removed `Memory` kind.
- `internal/cli/knowledge.go` — add a `nodeCmd(app)` parent command with a
  `search` subcommand (`balaur node search <terms…>`) exposing the cross-type
  search as the standard JSON envelope.
- `internal/cli/cli.go` — register `nodeCmd(app)` in the `root.AddCommand(...)`
  list (`cli.go:54`), alongside `memoryCmd(app)`/`skillCmd(app)`.
- `internal/self/knowledge.md` — one-line/paragraph update: unified FTS over
  active nodes (all knowledge types), consent-filtered.

**Out of scope** (do NOT touch — reason given):

- `migrations/` — plan **160 owns the `nodes`/`edges` baseline migration**.
  162 reads `nodes`; it never declares or alters the collection. Touching
  migrations here collides with 160.
- `internal/knowledge/knowledge.go` — **everything EXCEPT the `SearchActive`
  body**. The lifecycle/propose/transition/Touch helpers that reference the
  removed `memories`/`skills` collections are 160's to fix; 162 edits ONLY
  `SearchActive`'s two collection references (Step 5). If `SearchActive` no
  longer routes through `search.Index.Query` at all, that is a STOP condition.
- `internal/search/fts5_test.go` — the driver spike; unrelated, leave byte-identical.
- The web UI / a dedicated search results *card* / a `/search` web route —
  **DEFERRED to Known limitations.** A web results card must reuse 160's
  per-type node cards and 160's generic `GET /ui/show/{type}?id=…` route; their
  exact Go signatures are owned by 160 and not knowable from this plan without
  guessing. The agent `search` tool + the CLI `search` command deliver the
  end-to-end "find anything" slice deterministically and offline. Building the
  web card belongs to a follow-up once 160's card API is fixed. Do NOT invent a
  `/ui/notes/{id}` or `/search` route.
- Semantic / embedding / vector search — **REJECTED** (plans 073 and 121 deleted
  the embedding-rerank spike). Do NOT revive it. `kronk.Embed` stays unused here.
- bm25 ranking tuning / custom weights — out of scope; `ORDER BY rank` stays.
- The graph / multi-hop traversal — plan **163** owns it.

## Git workflow

- Branch: `improve/162-unified-search` (executor worktrees base off
  `origin/main`, per AGENTS.md — ensure 160 (and ideally 161) are merged to
  `main` first).
- Commit per logical unit; conventional-commit subjects (e.g.
  `feat(search): unified FTS over active nodes`, `feat(tools): cross-type search tool`).
- Do NOT push or open a PR unless the operator instructs it. Gate any push on a
  green full `go test ./...`.

## Steps

### Step 0: Confirm the dependency (160) has landed

162 indexes the `nodes` collection. Confirm it exists before doing anything.

```
grep -rn '"nodes"' migrations/
```

**Verify**: at least one match (160's baseline migration declares the `nodes`
collection). If there are **zero** matches, the `nodes` collection does not
exist — **STOP and report**: 160 has not landed and 162 cannot proceed.

Also confirm the node text/field names you will read. Open 160's baseline
migration and the live `nodes` collection schema and confirm the field that holds
the long markdown text is named `body` and the status field is `status` with an
`active` value. If 160 named them differently, use 160's real names everywhere
below.

**Verify**: `grep -rn '"body"\|"status"\|"type"' migrations/ | head` shows the
`nodes` field definitions. (Reading: confirm `body`, `status`, `type` exist on
`nodes`. STOP if `body` is absent and no equivalent long-text field is found.)

### Step 1: Generalize the FTS table in `Open`

In `internal/search/index.go`, change the `CREATE VIRTUAL TABLE` in `Open` from
`memories_fts (id UNINDEXED, title, content, when_to_use, category)` to the
unified table:

```go
	if _, err := db.Exec(`
		CREATE VIRTUAL TABLE IF NOT EXISTS knowledge_fts
		USING fts5(id UNINDEXED, kind UNINDEXED, title, content, extra)
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("search: create fts5 table: %w", err)
	}
```

Also update the package doc comment at the top of the file: replace "FTS5 memory
recall index" wording with "FTS5 knowledge recall index" (it indexes all node
types now), keeping the "disposable / rebuilt on next boot" sentence verbatim.

**Verify**: `grep -n 'knowledge_fts\|memories_fts' internal/search/index.go`
shows `knowledge_fts` and NO remaining `memories_fts`. (The build will fail until
later steps — that is expected; this step is byte-checked by grep, not compile.)

### Step 2: Switch `Rebuild` to source active nodes with `kind`

In `Rebuild`, change the `DELETE`, the source query, the prepared `INSERT`, and
the per-record `Exec` to the unified table. The source is the `nodes` collection
filtered to active; `kind` is the node's `type`; `content` ← node `body`. The
`extra` column carries any type-specific searchable text so recall parity is NOT
lost — concretely, a `memory` node's old recall hint lived in a `when_to_use`
field, so pack it into `extra` for memory kind via the `nodeExtra` helper below
(this avoids the silent regression the old `memories_fts` had, which indexed
`when_to_use` and `category` as columns):

```go
	if _, err := tx.Exec(`DELETE FROM knowledge_fts`); err != nil {
		tx.Rollback()
		return fmt.Errorf("search: rebuild delete: %w", err)
	}

	recs, err := app.FindRecordsByFilter("nodes", "status = 'active'", "", 0, 0, nil)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("search: rebuild fetch: %w", err)
	}

	stmt, err := tx.Prepare(`INSERT INTO knowledge_fts(id, kind, title, content, extra) VALUES (?, ?, ?, ?, ?)`)
	...
	defer stmt.Close()

	for _, r := range recs {
		if _, err := stmt.Exec(
			r.Id,
			r.GetString("type"),
			r.GetString("title"),
			r.GetString("body"),
			nodeExtra(r),
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("search: rebuild insert %s: %w", r.Id, err)
		}
	}
```

Add the small `nodeExtra` helper in the same file. It packs type-specific
searchable text into the `extra` column so memory recall hints
(`when_to_use`) stay findable. KISS — one helper, the only branch that matters
in v1 (memory's recall hint, if 160 kept that field on the node or in `props`):

```go
// nodeExtra returns type-specific searchable text for a node's `extra` FTS
// column. v1 preserves the old memory recall hint (when_to_use) so search
// parity with the removed memories_fts is not lost; other types contribute
// nothing extra yet.
func nodeExtra(r *core.Record) string {
	if r.GetString("type") == "memory" {
		// 160 may store the recall hint as a flat field or inside props.
		// Read whichever 160 actually kept; default to "" if neither exists.
		return r.GetString("when_to_use")
	}
	return ""
}
```

(`content` column ← node `body` field; `kind` column ← node `type` field;
`extra` column ← `nodeExtra(r)`. If 160's long-text field is not `body`, use
160's real name — see Step 0. **If 160 dropped `when_to_use` entirely (e.g.
folded it into `body` or `props` with a different key), read 160's node schema
and either pull the right key or, if no recall-hint field survives, leave
`nodeExtra` returning `""` and note the parity loss in Known limitations — do
not guess a field name.**)

**Verify**: `grep -n '"nodes"\|knowledge_fts\|"memories"' internal/search/index.go`
shows `nodes` + `knowledge_fts`, and `"memories"` returns NO matches.

### Step 3: Switch `Upsert` and `Delete` and `Query` to the unified table

`Upsert`: keep the delete-first idempotency and the `status != "active"` early
return EXACTLY; only change the INSERT to the unified columns and read `type`/`body`:

```go
	_, err := ix.db.Exec(
		`INSERT INTO knowledge_fts(id, kind, title, content, extra) VALUES (?, ?, ?, ?, ?)`,
		rec.Id,
		rec.GetString("type"),
		rec.GetString("title"),
		rec.GetString("body"),
		nodeExtra(rec),
	)
```

(Use the same `nodeExtra(rec)` helper from Step 2 so the live-upsert path and the
rebuild path index identical text — recall parity must hold on both paths.)

`Delete`: change `DELETE FROM memories_fts WHERE id = ?` → `DELETE FROM
knowledge_fts WHERE id = ?`.

`Query`: change ONLY the table name in the one SQL line — the quoting loop,
`matchExpr`, `ORDER BY rank`, and `LIMIT` stay byte-identical:

```go
	rows, err := ix.db.Query(
		`SELECT id FROM knowledge_fts WHERE knowledge_fts MATCH ? ORDER BY rank LIMIT ?`,
		matchExpr, limit,
	)
```

**Verify**: `grep -n 'memories_fts' internal/search/index.go` returns NOTHING;
then `CGO_ENABLED=0 go build ./internal/search/...` → exit 0.

### Step 4: Repoint the record hooks in `main.go` to `nodes`

In `registerSearchIndex` (`main.go`), change the three bindings at the bottom
from `"memories"` to `"nodes"`. The `upsertHook`/`deleteHook` closures are
unchanged (they already key off `e.Record.Id` and the index's own status logic):

```go
	app.OnRecordAfterCreateSuccess("nodes").BindFunc(upsertHook)
	app.OnRecordAfterUpdateSuccess("nodes").BindFunc(upsertHook)
	app.OnRecordAfterDeleteSuccess("nodes").BindFunc(deleteHook)
```

Also update the `registerSearchIndex` doc comment: replace "rebuilds it from
active memories" with "rebuilds it from active nodes" and "Record hooks keep the
index eventually consistent" stays.

**Verify**: `grep -n 'OnRecordAfter.*Success("memories")\|OnRecordAfter.*Success("nodes")' main.go`
shows three `("nodes")` bindings and zero `("memories")`. Then
`CGO_ENABLED=0 go build ./...` → exit 0.

### Step 5: Make `SearchActive` resolve ids against `nodes` (cross-type)

This is the change that makes "find anything" actually return mixed-type hits.
**First reconcile with 160**: read the live `SearchActive` in
`internal/knowledge/knowledge.go`. The index `Query` now returns ids of mixed
node types (note/skill/journal/person/memory), so the id→record resolution and
the LIKE fallback must target `nodes`, not the removed `memories` collection.

- **If 160 already rewrote `SearchActive` to resolve against `nodes`** (FTS path
  does `FindRecordsByIds("nodes", ids)` and the LIKE fallback queries `nodes`
  with `title`/`body`): keep 160's version, change nothing, and skip to the
  verify. Do NOT duplicate.
- **If 160 left it memory-only** (the `72fd762` excerpt above — FTS path does
  `FindRecordsByIds(string(Memory), ids)`, LIKE fallback does
  `FindRecordsByFilter(string(Memory), …)` with a `content`/`category` clause):
  make these two edits, and nothing else in the function.

FTS path — change the id resolution from the `memories` collection to `nodes`:

```go
		// before:
		recs, err := app.FindRecordsByIds(string(Memory), ids)
		// after:
		recs, err := app.FindRecordsByIds("nodes", ids)
```

LIKE fallback — query `nodes` on `title`/`body` (the node text fields), dropping
the memory-only `content`/`category` clause:

```go
		// before (per-term clause):
		clauses = append(clauses,
			fmt.Sprintf("title ~ {:%[1]s} || content ~ {:%[1]s} || when_to_use ~ {:%[1]s} || category ~ {:%[1]s}", key))
		// after:
		clauses = append(clauses,
			fmt.Sprintf("title ~ {:%[1]s} || body ~ {:%[1]s}", key))

		// before (the final query target):
		return app.FindRecordsByFilter(
			string(Memory),
			"status = 'active' && ("+strings.Join(clauses, " || ")+")",
			"-importance,-created", limit, 0,
			params,
		)
		// after — query nodes; if 160 removed `importance` from the node sort
		// surface, use "-updated,-created" (a node has no importance column):
		return app.FindRecordsByFilter(
			"nodes",
			"status = 'active' && ("+strings.Join(clauses, " || ")+")",
			"-updated,-created", limit, 0,
			params,
		)
```

Keep the defensive `status == StatusActive` re-filter and the FTS rank-order
sort EXACTLY as they are — only the collection name and the LIKE field list
change. Touch NOTHING else in `knowledge.go`.

**Verify**: `grep -n 'FindRecordsByIds\|FindRecordsByFilter' internal/knowledge/knowledge.go`
on the `SearchActive` lines shows `"nodes"` (not `string(Memory)`); then
`grep -n 'string(Memory)' internal/knowledge/knowledge.go` shows no occurrence
inside `SearchActive`. `CGO_ENABLED=0 go build ./internal/knowledge/...` → exit 0.
If 160 already rewrote it, the grep already shows `"nodes"` and you made no edit —
that is the success case too.

### Step 6: Fix `recall` and add the cross-type `search` agent tool

**First fix `recallTool`** (`internal/tools/knowledge.go:108-138`). Its body
reads `m.GetString("category")` and `m.GetString("content")` (both removed by
160) and calls `knowledge.Touch(app, knowledge.Memory, m)` (the `Memory` kind no
longer maps to a collection). `SearchActive` now returns `nodes` records, so read
`type`/`body` and stop touching against the gone kind:

```go
	// before:
	for _, m := range recs {
		fmt.Fprintf(&b, "- [%s] %s: %s\n",
			m.GetString("category"), m.GetString("title"), m.GetString("content"))
		knowledge.Touch(app, knowledge.Memory, m)
	}
	// after — node records carry type/body; recall stays read-only (no Touch,
	// matching the CLI `memory recall` which already skips Touch to avoid
	// skewing usage stats on an inspection path):
	for _, m := range recs {
		fmt.Fprintf(&b, "- [%s] %s: %s\n",
			m.GetString("type"), m.GetString("title"), m.GetString("body"))
	}
```

(Dropping the `Touch` call is the KISS fix: `Touch` takes a `knowledge.Kind`
that no longer corresponds to a collection after 160, and the CLI recall path
already runs without it. If 160 kept a node-wide `Touch(app, rec)` helper,
calling it is fine but optional — do NOT reintroduce `knowledge.Memory`.)

**Then add `searchTool`** and include it in `KnowledgeTools`. It searches ALL
active node types (not just memories) and returns a `[kind] title: snippet` list
with the node id, so the model can cite a hit. Reuse `knowledge.SearchActive`
(rewritten in Step 5 to query the unified index and return active node records).

```go
func searchTool(app core.App) agent.Tool {
	return agent.Tool{
		Spec: agent.ToolSpecOf("search",
			"Full-text search across ALL your approved knowledge — notes, memories, "+
				"skills, journal entries, and typed objects. Returns mixed-type hits "+
				"ranked by relevance. Proposed/unapproved knowledge is never returned.",
			obj(map[string]any{
				"terms": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "1-4 search terms (OR semantics)."},
			}, "terms")),
		Execute: func(ctx context.Context, argsJSON string) (string, error) {
			var args struct {
				Terms []string `json:"terms"`
			}
			if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
				return "", fmt.Errorf("search: bad arguments: %w", err)
			}
			recs, err := knowledge.SearchActive(app, args.Terms, 10)
			if err != nil {
				return "", err
			}
			if len(recs) == 0 {
				return "No approved knowledge matches.", nil
			}
			var b strings.Builder
			for _, r := range recs {
				fmt.Fprintf(&b, "- [%s] %s: %s\n",
					r.GetString("type"), r.GetString("title"), snippet(r.GetString("body")))
			}
			return b.String(), nil
		},
	}
}
```

Add a tiny `snippet` helper in the same file (truncate `body` to ~160 runes,
single line) — or inline a `strings`-based truncation; keep it KISS:

```go
// snippet returns a short single-line preview of node body text for search hits.
func snippet(s string) string {
	s = strings.ReplaceAll(strings.TrimSpace(s), "\n", " ")
	if len([]rune(s)) > 160 {
		return string([]rune(s)[:160]) + "…"
	}
	return s
}
```

Register it in `KnowledgeTools`:

```go
func KnowledgeTools(app core.App) []agent.Tool {
	return []agent.Tool{
		rememberTool(app),
		recallTool(app),
		searchTool(app),
		skillTool(app),
		proposeSkillTool(app),
	}
}
```

> **Important**: 162 owns `SearchActive`'s cross-type rewrite (Step 5), so it
> stays the single helper both `recall` and `search` call. If 160 renamed it
> (e.g. to `Search`, dropping the `Memory` kind argument), use 160's name. If no
> node-wide search helper exists in `internal/knowledge` at all, **STOP and
> report** — do NOT re-implement index querying inside `tools` (that would
> duplicate the index integration and bypass the active-status defensive
> re-filter `SearchActive` performs).

**Verify**: `CGO_ENABLED=0 go build ./internal/tools/...` → exit 0; then
`grep -n 'searchTool\|"search"' internal/tools/knowledge.go` shows the tool and
its registration, and
`grep -n 'GetString("category")\|GetString("content")\|knowledge.Memory' internal/tools/knowledge.go`
shows NONE of those on the recall path (the `remember` tool may still build a
`MemoryProposal`, which is 160's to keep — only `recallTool`'s field reads must
be gone).

### Step 7: Add the `node search` CLI subcommand

The repo's CLI uses PARENT commands with subcommands (`memoryCmd`/`skillCmd` are
containers; `recall`/`show` are subcommands), registered via `root.AddCommand(...)`
in `internal/cli/cli.go:54`. Match that shape: add a `nodeCmd(app)` parent in
`internal/cli/knowledge.go` with one `search <terms...>` subcommand. The surface
is `balaur node search <terms…>` (the brief's name), NOT a flat top-level verb.
Mirror the `run(app, cliKind, …)` envelope pattern used by `statusListCmd`.

```go
func nodeCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "node",
		Short: "Query knowledge nodes — full-text search across all types, deterministic, no model",
	}
	cmd.AddCommand(nodeSearchCmd(app))
	return cmd
}

func nodeSearchCmd(app core.App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "search <terms...>",
		Short: "Full-text search across all active knowledge nodes (the model's search tool)",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.RunE = run(app, "node.search", func(cmd *cobra.Command, args []string) (any, error) {
		recs, err := knowledge.SearchActive(app, args, 20)
		if err != nil {
			return nil, err
		}
		out := make([]map[string]any, 0, len(recs))
		for _, r := range recs {
			out = append(out, map[string]any{
				"id":     r.Id,
				"type":   r.GetString("type"),
				"title":  r.GetString("title"),
				"body":   r.GetString("body"),
				"status": r.GetString("status"),
			})
		}
		return out, nil
	})
	return cmd
}
```

Register `nodeCmd(app)` in `internal/cli/cli.go` inside the existing
`root.AddCommand(...)` call (`cli.go:54`), alongside `memoryCmd(app)` and
`skillCmd(app)`:

```go
	root.AddCommand(
		chatCmd(app),
		taskCmd(app),
		memoryCmd(app),
		skillCmd(app),
		nodeCmd(app),
		lifeCmd(app),
		...
	)
```

**Verify** (grep is the primary check — it confirms wiring without booting
inference): `grep -n 'nodeCmd\|nodeSearchCmd' internal/cli/knowledge.go` shows
both functions defined, and `grep -n 'nodeCmd(app)' internal/cli/cli.go` shows the
registration. Then `CGO_ENABLED=0 go build ./...` → exit 0.
(Optional, only if a model/native lib is available: `go run . node search --help
2>&1 | grep -i 'Full-text search'` → one match. `go run .` boots the full app
and may hang in a sandbox, so do NOT rely on it — the grep + build is the
contract.)

### Step 8: Update self-knowledge

In `internal/self/knowledge.md`, update the two sentences that describe
search/recall (the file does NOT contain the literal `memories_fts`, so do not
chase that token — fix the prose):

- Line ~79: `search (FTS5 sidecar index — bm25-ranked recall …)` — widen it to
  say the index spans **active nodes of all knowledge types**
  (note/memory/skill/journal/typed-objects), consent-filtered (proposed/rejected
  nodes are never indexed), and name the table `knowledge_fts`.
- Line ~103: `recall searches approved memories` — this is the real stale
  sentence. Replace it with something like: recall and the cross-type `search`
  tool query approved **nodes** (all knowledge types) via the unified
  `knowledge_fts` index, with a deterministic LIKE fallback when the sidecar is
  unavailable.

Mention the `search` agent tool and the `balaur node search` CLI verb.

**Verify**: `grep -in 'knowledge_fts\|active nodes\|all knowledge types' internal/self/knowledge.md`
returns at least one match (the NEW wording is present — this is the real check,
since the old `memories_fts` token was never in this file). Also confirm the old
sentence is gone: `grep -in 'recall searches approved memories' internal/self/knowledge.md`
returns nothing.

### Step 9: Full gate

Run the whole suite and the format/diff checks.

**Verify**:
- `gofmt -l internal/search internal/tools internal/cli internal/knowledge main.go internal/self` → no output
- `go vet ./...` → exit 0
- `go test ./...` → all pass
- `env -u BALAUR_OS_ACCESS -u BALAUR_SOURCE -u BALAUR_MAX_STEPS go test ./...` → all pass
- `CGO_ENABLED=0 go build ./...` → exit 0
- `git diff --check` → no output

### Step 10: Refresh the graph

After code changes land, keep the knowledge graph current (AST-only, no API cost).

```
graphify update .
```

`graph.json` is committed **minified** — if `graphify update` emits a
pretty-printed file, minify it before staging (see the repo's graphify note). Do
NOT stage `graph.json` if it would balloon the diff; if unsure, leave the graph
update out of the commit and report it.

**Verify**: `git diff --stat graphify-out/graph.json` shows a reasonable change
(not a 100k+-line reformat). If it shows a massive reformat, the file was not
re-minified — fix or drop it from the commit.

## Test plan

All tests live in `internal/search/index_test.go`, modeled on the existing
table tests. **The existing helpers and EVERY raw FTS literal in this file WILL
fail to compile against the new schema** — the helpers call
`FindCollectionByNameOrId("memories")` and `Set` `category`/`importance`/
`when_to_use`/`source` (all gone post-160), and six tests embed
`INSERT INTO memories_fts(id, title, content, when_to_use, category)` literals.
This rewrite is **mandatory, not optional** — the file is in scope and the
suite is red until it is done.

**Step A — replace the seed helpers (mandatory).** Drop
`seedActiveMemory`/`seedProposedMemory` and add `seedActiveNode`/`seedProposedNode`
that target `nodes` and Set ONLY the fields 160 declares (`type`, `title`, `body`,
`status`). If 160 left a node-seeding helper in `internal/knowledge`/`internal/seed`,
prefer reusing it; otherwise inline this:

```go
// seedActiveNode inserts an active node of the given type directly via the
// PocketBase app (bypassing the knowledge lifecycle to avoid an import cycle).
func seedActiveNode(t *testing.T, app core.App, kind, title, body string) string {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("nodes")
	if err != nil {
		t.Fatalf("find nodes collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("type", kind)
	rec.Set("title", title)
	rec.Set("body", body)
	rec.Set("status", "active")
	if err := app.Save(rec); err != nil {
		t.Fatalf("save node: %v", err)
	}
	return rec.Id
}

// seedProposedNode inserts a proposed (non-active) node.
func seedProposedNode(t *testing.T, app core.App, kind, title, body string) string {
	t.Helper()
	col, err := app.FindCollectionByNameOrId("nodes")
	if err != nil {
		t.Fatalf("find nodes collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("type", kind)
	rec.Set("title", title)
	rec.Set("body", body)
	rec.Set("status", "proposed")
	if err := app.Save(rec); err != nil {
		t.Fatalf("save proposed node: %v", err)
	}
	return rec.Id
}
```

(If 160 made `body` non-settable here, or `nodes` requires `props`, read 160's
schema and Set what its validators require — but never re-add `category`/
`importance`/`when_to_use` columns; they are gone.)

**Step B — rewrite the six raw FTS literals (mandatory).** These tests embed
`INSERT INTO memories_fts(id, title, content, when_to_use, category) VALUES (?, ?, ?, ?, ?)`
and break against the 5-column `knowledge_fts`. Update each to
`INSERT INTO knowledge_fts(id, kind, title, content, extra) VALUES (?, ?, ?, ?, ?)`
(supply a `kind`, e.g. `"note"`, where a literal value is needed):
`TestOpenCreatesSchema`, `TestQueryInjectionSafe`, `TestDeleteRemovesRecord`
(these three have literal `memories_fts` inserts), plus `TestRebuildAndQuery`,
`TestUpsertNonActive`, and `TestQueryEmpty` need their memory seeding switched to
the `seedActiveNode`/`seedProposedNode` helpers (`TestQueryEmpty` only calls
`Query`, so it needs no seed change — confirm and leave it). The injection-safety
and limit behaviors are unchanged; these tests stay the contract.

**New/updated test cases (each a `Test…` func):**

1. **`TestRebuildMultiKind`** — seed active nodes of several `type`s (a `note`, a
   `memory`, a `skill`, a `journal`) with `seedActiveNode`, rebuild, and assert a
   term unique to each returns its id. Proves the index is type-agnostic.
   (Replaces/extends `TestRebuildAndQuery`.)
2. **`TestSearchConsentFilter`** (THE consent test) — seed one active node and
   one **proposed** node whose body contains a unique term; `Rebuild`; assert
   `Query(["that-term"])` returns ONLY the active node's id and NEVER the proposed
   node's id. Then `Upsert` the proposed record directly (fetch it with
   `app.FindRecordById("nodes", id)`) and assert it is still absent (the
   `status != "active"` branch deletes it). This is the spine test: a proposed
   node is never returned by search.
3. **`TestUpsertNonActive`** — keep the existing intent on the `nodes`/`type`/
   `body` shape: seed active node, fetch via `FindRecordById("nodes", id)`,
   Upsert, confirm present; flip in-memory status to `archived`, Upsert again,
   confirm removed.
4. **`TestExtraPreservesMemoryRecallHint`** (parity test for the regression fix) —
   seed a `memory` node whose `when_to_use`/recall-hint text (whatever field
   `nodeExtra` reads) contains a unique term NOT present in title/body; `Rebuild`;
   assert `Query(["that-hint-term"])` returns the node id. If 160 dropped the
   recall-hint field entirely so `nodeExtra` returns `""`, SKIP this test with
   `t.Skip` and note it in Known limitations — do not assert on a field that does
   not exist.
5. **A `recall`/`search` integration test** — if `internal/knowledge` has a
   `SearchActive` test (`internal/knowledge/knowledge_test.go`
   `TestSearchActive*`), update its fixtures to seed `nodes` and assert a known
   active node is recalled and a proposed one is not. If 160 already updated those
   tests, leave them and only confirm green.

Structural pattern to copy: the existing `internal/search/index_test.go`
table/helper style (`openTestIndex`, `storetest.NewApp(t)`, plain `testing`,
no assertion framework).

**Verification**: `go test ./internal/search/... ./internal/tools/... ./internal/knowledge/...`
→ all pass, including the new `TestRebuildMultiKind` and `TestSearchConsentFilter`.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -rn 'memories_fts' internal/ main.go` returns NOTHING (the old table
      name is fully gone).
- [ ] `grep -n 'knowledge_fts' internal/search/index.go` shows the unified table
      in `Open`, `Rebuild`, `Upsert`, `Delete`, and `Query` (5 sites).
- [ ] `grep -n 'OnRecordAfter.*Success("nodes")' main.go` shows three bindings;
      `grep -n 'OnRecordAfter.*Success("memories")' main.go` returns nothing.
- [ ] `grep -n 'searchTool\|"search"' internal/tools/knowledge.go` shows the new
      agent tool defined and registered in `KnowledgeTools`.
- [ ] `recall` no longer reads the removed memory fields/kind:
      `grep -n 'GetString("category")\|GetString("content")\|knowledge.Memory' internal/tools/knowledge.go`
      shows none of these inside `recallTool` (the `remember` tool's
      `MemoryProposal` is 160's to keep).
- [ ] `SearchActive` resolves against `nodes`, not the removed collection:
      `grep -n 'FindRecordsByIds("nodes"\|FindRecordsByFilter(' internal/knowledge/knowledge.go`
      on the `SearchActive` lines shows `"nodes"`, and `string(Memory)` does NOT
      appear inside `SearchActive`.
- [ ] `grep -n 'nodeCmd\|nodeSearchCmd' internal/cli/knowledge.go` shows the
      parent + `search` subcommand defined; `grep -n 'nodeCmd(app)' internal/cli/cli.go`
      shows it registered in `root.AddCommand(...)`.
- [ ] `grep -in 'knowledge_fts\|active nodes\|all knowledge types' internal/self/knowledge.md`
      matches (new wording present); `grep -in 'recall searches approved memories' internal/self/knowledge.md`
      returns nothing (old sentence replaced).
- [ ] `go test ./...` exits 0; `TestRebuildMultiKind` and `TestSearchConsentFilter`
      exist and pass.
- [ ] `env -u BALAUR_OS_ACCESS -u BALAUR_SOURCE -u BALAUR_MAX_STEPS go test ./...`
      exits 0.
- [ ] `gofmt -l internal/search internal/tools internal/cli internal/knowledge main.go internal/self`
      is empty.
- [ ] `go vet ./...` exits 0; `CGO_ENABLED=0 go build ./...` exits 0;
      `git diff --check` is empty.
- [ ] `git status` shows only in-scope files modified (no `migrations/` changes).
- [ ] `plans/readme.md` status row for 162 updated.

## STOP conditions

Stop and report back (do not improvise) if:

- **160 has not landed**: `grep -rn '"nodes"' migrations/` returns nothing — there
  is no `nodes` collection to index (Step 0).
- The `nodes` long-text field is NOT named `body` and no equivalent long-text
  field is found in 160's schema (Step 0) — you cannot guess which field holds
  the markdown content.
- `internal/knowledge`'s cross-type search helper (`SearchActive` or 160's
  rename) is GONE entirely, or no longer routes through `search.Index.Query`
  (Step 5/6) — the index integration contract changed under this plan. (If it
  exists but still resolves against the removed `memories` collection, that is
  NOT a stop — Step 5 is exactly the fix; rewrite it to `nodes`.)
- The "Current state" excerpts in `internal/search/index.go` do not resemble the
  live code even after accounting for 160's collection rename (e.g. someone
  already renamed the table, or the quoting/`ORDER BY rank` block changed) — the
  tree drifted; reconcile before editing.
- Any step's verification fails twice after a reasonable fix attempt.
- A fix appears to require touching an out-of-scope file (especially
  `migrations/`, or any `internal/knowledge` function OTHER than the
  `SearchActive` body — the lifecycle/propose/transition/Touch helpers are 160's).
- A `go get`/`go mod` step fails with "certificate signed by unknown authority"
  in a sandbox — apply the GOPROXY shim per `docs/hyperagent-sandbox.md`; never
  weaken `GOSUMDB`.

## Maintenance notes

For the human/agent who owns this code after the change lands:

- **The `extra` column carries type-specific searchable text via `nodeExtra`.**
  In v1 the only branch populated is `memory` → its recall hint (`when_to_use`),
  which preserves parity with the removed `memories_fts` (whose `when_to_use`
  column was searchable). The column exists precisely so future types (a person's
  aliases, a book's author, a flattened `props` blob) can be indexed WITHOUT
  another FTS schema migration: add a branch to `nodeExtra`, do not add columns.
  NOTE: the removed `memories_fts` also indexed `category` as a column; it is NOT
  packed into `extra` because category is a short enum the owner browses by facet,
  not free text worth full-text matching — this is a deliberate, documented narrow
  (see Known limitations), not an accidental drop.
- **The consent filter lives in two places, and both must stay**: `Rebuild`'s
  `status = 'active'` WHERE clause AND `Upsert`'s `status != "active"` early
  return. A node that flips to proposed/rejected/archived is removed from the
  index by the update hook → Upsert → Delete branch. A reviewer should verify
  `TestSearchConsentFilter` covers both the rebuild path and the live-flip path.
- **`search.db` is disposable**: there is no migration, no versioning. If the FTS
  schema ever changes again, just bump nothing — the next boot's `Rebuild`
  rewrites the table. The `Open` uses `CREATE … IF NOT EXISTS`, so on a schema
  change you must delete a stale `search.db` (registerSearchIndex already
  delete-and-retries on a corrupt/rebuild failure).
- **Deferred (named here so it is tracked, not lost)**:
  - **Web search results card + `/search` route** — reuse 160's per-type node
    cards and 160's generic `GET /ui/show/{type}?id=…` dispatcher to render mixed
    hits in the browser. Deferred because it depends on 160's exact card API.
    This plan ships search via the agent tool + CLI; the web card is a thin
    follow-up.
  - **Semantic / embedding / vector search** — explicitly REJECTED (plans 073,
    121). Do not revive. Lexical FTS5 + LIKE fallback is the v1 contract.
  - **Per-type ranking weights / bm25 tuning** — `ORDER BY rank` is the v1
    default; tune only with a measured relevance complaint in hand.
  - **`category` is no longer full-text indexed.** The old `memories_fts` indexed
    the memory `category` enum as a column; the unified index does not (see the
    `extra` maintenance note — category is a browse facet, not free text). A
    memory whose ONLY matching term was its category string is no longer matched
    by FTS. This is an accepted, deliberate narrow for v1; if it bites, pack
    category into `extra` for memory kind in `nodeExtra` (one line, no schema
    change).
  - **`recall` vs `search`**: after Step 5/6 both call the same cross-type
    `SearchActive`, so `recall` now returns mixed node types too — its
    description still frames it as a memory-recall aid, but it is no longer
    collection-scoped. `search` is the explicit cross-type verb. If a future
    cleanup wants to merge or re-narrow them, that is a deliberate UX decision,
    not a bug — keep both until then.
  - **If 160 dropped the memory recall-hint field** (`when_to_use` not carried
    onto the node and not in `props`), `nodeExtra` returns `""` for memory and the
    parity test is skipped — recall-hint search is then unrecoverable without a
    schema add. Flagged so it is a conscious choice, not silent loss.
- A reviewer should scrutinize: that NO `proposed` node can ever reach the index
  (the spine), that `Query`'s injection-safe quoting is byte-identical to before,
  and that `git status` shows zero `migrations/` changes.
