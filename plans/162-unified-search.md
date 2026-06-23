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
> `git diff --stat 0c85d0e..HEAD -- internal/search/index.go internal/search/index_test.go internal/knowledge/knowledge.go internal/tools/knowledge.go internal/cli/knowledge.go internal/cli/cli.go main.go internal/self/knowledge.md`
> Plans 160 AND 161 have ALREADY LANDED (HEAD `0c85d0e`). This plan's "Current
> state" excerpts below are now stamped at `0c85d0e` (POST-160/161) — they are the
> LIVE code, refreshed during reconcile. 160 already did the heavy lifting that
> earlier drafts of this plan attributed to 162: it repointed the FTS index from
> the `memories` collection to the `nodes` collection, kept it **memory-only**
> (an Upsert/Rebuild `type == "memory"` gate), reads `body` + props
> (`when_to_use`/`category` via `nodes.PropString`) off the node, and bound the
> FTS create/update/delete hooks in `main.go` to `"nodes"`. **The table is still
> named `memories_fts`.** So 162 is no longer a "repoint memories→nodes" job — it
> is a **WIDEN** job: generalize the already-nodes-pointed, memory-only index to
> ALL active node types. If 160 has somehow NOT landed (the `nodes` collection
> does not exist — see Step 0), that is a STOP condition: 162 cannot index a
> collection that does not exist yet.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: plans/160-*.md (the `nodes`+`edges` baseline migration — HARD dependency). Soft-benefits-from plans/161-*.md (wikilink edges) but does NOT require it.
- **Category**: migration
- **Planned at**: commit `0c85d0e`, 2026-06-23 (re-stamped post-160/161; originally drafted at `72fd762`)
- **Issue**: —

## Design conflict resolutions (advisor — AUTHORITATIVE; override the in-body conflict blocks)

The reconcile flagged five conflicts "for the advisor." All are DECIDED below; these OVERRIDE any "ADVISOR MUST DECIDE" block in Steps 2/5, the Test plan, and the CLI step. Follow these.

**1. Category stays indexed (Step 2 → option a).** The unified table is `knowledge_fts(id UNINDEXED, kind UNINDEXED, title, content, extra)`. Add a `nodeExtra(rec) string` helper: for a `memory` node, `extra` = `when_to_use` + " " + `category` (both from props) — this preserves what 160's live `memories_fts` already indexes (dropping `category` would be a REGRESSION, and `SearchActive`'s fallback still matches on category). For other node types, `extra` is type-specific or empty. DELETE any "category is no longer full-text indexed / is a browse facet" narrowing from the plan's Maintenance/Known-limitations.

**2. `recall` stays MEMORY-SCOPED — do NOT de-narrow `SearchActive` (OVERRIDES Step 5's "drop the type==Memory narrowing on both paths").** `SearchActive`'s three callers (`internal/turn/context.go` BuildContext, `internal/cli/knowledge.go` recall, `internal/tools/knowledge.go` recallTool) all consume hydrated *memory* aliases — returning mixed node types would feed them non-memory nodes hydrated as memory (a bug). So:
- The index gains a kind-aware `QueryKind(terms []string, kind string, limit int) ([]string, error)` (the existing `Query` MATCH/bm25/injection-safe-quoting body + a `kind = ?` filter) ALONGSIDE the now-all-kinds `Query(terms, limit)`.
- `SearchActive` changes ONLY its index call from `Query(...)` to `QueryKind(..., string(Memory), ...)` so recall returns the top-N **memories** (post-filtering an all-kinds top-N would silently drop memories crowded out below the cutoff by notes/journal). Keep its `type==Memory` re-filter, the `hydrate(Memory, …)` per hit, the `importance` sort, and the `ListByTypeStatus(Memory)` fallback EXACTLY as 160 shipped them. `SearchActive` is otherwise UNCHANGED.

**3. The new cross-type search is a SEPARATE function — no `nodes.ListByStatus`, `internal/nodes` stays out of scope (Step 5/Scope).** Add `SearchAllActive(app, terms) ([]*core.Record, error)` (next to `SearchActive`): call the index's all-kinds `Query`, map ids → nodes via `app.FindRecordsByIds("nodes", ids)`, defensively keep `status=="active"`, return RAW records (NO memory hydration). The `search` agent tool + results card render each hit by its node `type` via the existing card registry (e.g. `/ui/show/note?id=<id>` + a kind label). If a non-FTS fallback is wanted, inline `app.FindRecordsByFilter("nodes", "status='active'", …)` — do NOT add a lister to `internal/nodes`.

**4. Delete the inverted gate test (Test plan).** Remove `TestUpsertGatesNonMemoryNodes` (its premise — a `note` never indexes — is inverted once the `type!="memory"` Upsert gate is dropped). Replace with `TestRebuildMultiKind` (active note/journal/person nodes DO index, keyed by `kind`). KEEP `TestSearchConsentFilter` (proposed/rejected never indexed, both rebuild + live-flip paths). ADD a recall test asserting `SearchActive`/`QueryKind("memory")` still returns ONLY memory nodes (the no-regression guard for conflict 2).

**5. CLI verb is flat `balaur search <terms>` (CLI step).** Add a top-level `searchCmd` registered in `cli.go`'s `root.AddCommand` block, NOT a `balaur node …` group — it matches the `search` agent tool name and avoids the one-letter `node`/`note` collision (160 already shipped `noteCmd`). It prints the v1 JSON envelope of mixed-kind hits.

## Why this matters

Plan 160 already folded `memories`, `skills`, notes, typed objects, and journal
entries into a single `nodes` collection and removed the standalone
`memories`/`skills` collections. **160 also already touched `internal/search`**
(more than its dossier scope suggested): it repointed `Rebuild`/`Upsert` from the
removed `memories` collection to `nodes`, kept the index **memory-only** (a
`type == "memory"` gate in `Upsert`, a `type = 'memory' && status = 'active'`
filter in `Rebuild`), reads `body` for content and `nodes.PropString(r,
"when_to_use"/"category")` from props, and bound the FTS record hooks in
`main.go` to `"nodes"`. **The FTS table is still named `memories_fts` with its
original 5 columns `(id UNINDEXED, title, content, when_to_use, category)`.** So
the index already works on the new schema — for memories only.

This plan is what makes full-text recall span **all** knowledge types: it
**widens** the existing memory-only nodes index into a unified one — renames
`memories_fts(id, title, content, when_to_use, category)` →
`knowledge_fts(id UNINDEXED, kind UNINDEXED, title, content, extra)`, sets
`kind = the node type`, and **removes 160's `type == "memory"` gate** so EVERY
`status=active` node indexes. It keeps the strict `status=active` consent filter
so agent-proposed-but-unapproved knowledge never surfaces (the consent spine),
**rewrites the recall resolution path so a search id maps to a `nodes` record of
ANY type** (160 already resolves ids against `nodes`, but re-filters to
`type == memory` — 162 drops that type filter), fixes the `recall` tool to read
node `type`/`body` instead of `category`/`content`, and adds a cross-type
`search` agent tool + a `balaur node search` CLI command so the owner and the
model can "find anything" across notes, memories, skills, journal, and typed
objects in one query.

**Ownership boundary with 160 (read this — it is the crux of this plan).** 160
removed the `memories`/`skills` collections, folded them into `nodes`, AND already
repointed `internal/search` (Rebuild/Upsert/Delete/Query source + the `main.go`
hooks) and `SearchActive`'s id resolution onto `nodes` — memory-scoped. What 160
did NOT do, and 162 owns:
- **Widen the FTS schema** (`memories_fts` → `knowledge_fts`, add `kind`/`extra`,
  drop the dedicated `when_to_use`/`category` columns) and **remove the
  memory-only gate** so all active node types index (Steps 1–4).
- **Generalize `SearchActive`** from memory-scoped to cross-type: the LIVE code
  (`internal/knowledge/knowledge.go:302-360`) resolves ids against `nodes` but
  then re-filters `r.GetString("type") == string(Memory)` on the FTS path and
  scans only `type=memory` nodes in the fallback (`nodes.ListByTypeStatus(app,
  string(Memory), …)`). 162 removes the `type == memory` narrowing so mixed-type
  hits return (Step 5).
- **Generalize `recallTool`** (`internal/tools/knowledge.go:119-149`): it reads
  `m.GetString("category")`/`m.GetString("content")` (legacy hydrated aliases —
  these still resolve via `hydrate(Memory, …)`, but only because `SearchActive`
  hydrates as Memory) and calls `knowledge.Touch(app, knowledge.Memory, m)`. Once
  `SearchActive` returns mixed types, those memory-only reads are wrong; 162
  reads `type`/`body` and drops the Memory-kind Touch (Step 6).

**Reconcile, do not collide**: 160 already did the collection repoint and the
`body`/props field reads — do NOT redo them; build on that state. The done
criteria assert the final state: table is `knowledge_fts`, no `type == "memory"`
gate in `index.go`, no `r.GetString("type") == string(Memory)` re-filter inside
`SearchActive`, and no `GetString("category")`/`GetString("content")`/
`knowledge.Memory` on the recall path.

The win when this lands: one search surface over all knowledge, the consent
filter is enforced at the index boundary (proposals are never indexed, never
returned, never leave the box), and the deterministic substring fallback (a Go
scan over active nodes — 160 replaced the old `dbx`-LIKE filter with this) still
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
  rebuilds it, and binds the create/update/delete record hooks. **160 already
  repointed these hooks to the `nodes` collection** (`main.go:255-257`); 162 does
  NOT touch the hook bindings. The only `main.go` edit left is the stale doc
  comment ("rebuilds it from active memories") — widen the wording to "active
  nodes". **Note: 161 added a SECOND pair of `nodes` create/update hooks
  (`registerGraphLinks`, `main.go:266-275`) for wikilink edges — leave it
  untouched.**
- `internal/knowledge/knowledge.go` — `SearchActive` (`:302-360`) consults the
  index (FTS5 fast path) and falls back to a substring scan. **160 already
  repointed it to `nodes`** (`FindRecordsByIds("nodes", ids)` on the FTS path),
  but it remains **memory-scoped**: the FTS path re-filters
  `r.GetString("type") == string(Memory)` and hydrates as Memory, and the
  fallback only scans `nodes.ListByTypeStatus(app, string(Memory), …)`. **IN
  SCOPE for the `SearchActive` body only** (Step 5): drop the `type == memory`
  narrowing on both paths so search returns mixed-type hits. Do NOT touch the rest
  of this file (160 owns the lifecycle/propose/transition/hydrate/Touch code).
- `internal/tools/knowledge.go` — agent tools (`remember`, `recall`, `skill`,
  `propose_skill`, plus 160's `node_write`/`node_list`/`node_get`/`node_drop`) via
  `KnowledgeTools`. This plan ADDS a `search` tool here AND fixes `recallTool`
  (Step 6): its body reads `m.GetString("category")`/`m.GetString("content")`
  (legacy hydrated aliases valid only while results are memory nodes) and calls
  `knowledge.Touch(app, knowledge.Memory, m)`.
- `internal/cli/knowledge.go` + `internal/cli/cli.go` — CLI subcommands. The repo
  uses PARENT commands (`memoryCmd`/`skillCmd`) with subcommands, registered via
  `root.AddCommand(...)` in `cli.go:54`. This plan ADDS a `node` parent command
  with a `search` subcommand (`balaur node search <terms…>`), registered the same
  way, mirroring the JSON-envelope pattern of the existing commands.
- `internal/self/knowledge.md` — the running binary's self-description; updated
  to describe unified search over nodes.

### Excerpt: the FTS table + Rebuild source, LIVE at `0c85d0e` (`internal/search/index.go:36-92`)

160 left the table named `memories_fts` with the original 5 columns, sourced from
`nodes` filtered to `type='memory'`, reading `body` for content and props
(`nodes.PropString`) for `when_to_use`/`category`. (Note the import added by 160:
`"github.com/alexradunet/balaur/internal/nodes"` at `index.go:14`.)

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
		tx.Rollback()
		return fmt.Errorf("search: rebuild delete: %w", err)
	}

	recs, err := app.FindRecordsByFilter("nodes", "type = 'memory' && status = 'active'", "", 0, 0, nil)
	...
	stmt, err := tx.Prepare(`INSERT INTO memories_fts(id, title, content, when_to_use, category) VALUES (?, ?, ?, ?, ?)`)
	...
	for _, r := range recs {
		if _, err := stmt.Exec(
			r.Id,
			r.GetString("title"),
			r.GetString("body"),
			nodes.PropString(r, "when_to_use"),
			nodes.PropString(r, "category"),
		); err != nil {
			...
		}
	}
	...
}
```

**162's job here**: rename the table to `knowledge_fts(id, kind, title, content,
extra)`, set source filter to `"status = 'active'"` (all types, drop
`type = 'memory'`), carry `kind = r.GetString("type")`, keep `content ←
r.GetString("body")`, and route `when_to_use`/`category` through the `extra`
column via `nodeExtra` (see the design-conflict note in Step 2 — 160 stores both
in props, so `extra` must read them from props, not flat fields).

### Excerpt: Upsert + Query, LIVE at `0c85d0e` (`internal/search/index.go:99-160`)

160 added a `type != "memory"` gate (the memory-only narrowing 162 removes),
kept the `status != "active"` consent early-return, and reads `body` + props.

```go
func (ix *Index) Upsert(rec *core.Record) error {
	// Always delete first so Upsert is truly idempotent.
	if err := ix.Delete(rec.Id); err != nil {
		return err
	}
	if rec.GetString("type") != "memory" {
		return nil // non-memory node: deletion above is the right action
	}
	if rec.GetString("status") != "active" {
		return nil // non-active: deletion above is the right action
	}
	_, err := ix.db.Exec(
		`INSERT INTO memories_fts(id, title, content, when_to_use, category) VALUES (?, ?, ?, ?, ?)`,
		rec.Id,
		rec.GetString("title"),
		rec.GetString("body"),
		nodes.PropString(rec, "when_to_use"),
		nodes.PropString(rec, "category"),
	)
	...
}
```

**162's job here**: REMOVE the `if rec.GetString("type") != "memory" { return nil }`
gate (so every active node type indexes), KEEP the `status != "active"`
consent early-return EXACTLY, switch the INSERT to
`knowledge_fts(id, kind, title, content, extra)` with `kind = rec.GetString("type")`,
`content = rec.GetString("body")`, `extra = nodeExtra(rec)`.

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

### Excerpt: the record hooks in `main.go`, LIVE at `0c85d0e` (`main.go:234-257`)

**160 already repointed these three bindings to `"nodes"`** — 162 leaves them
byte-identical (the upsert/delete closures already key off `e.Record.Id` and the
index's own type/status logic). No hook edit is needed in 162.

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

The ONLY `main.go` change 162 makes is the stale doc comment on
`registerSearchIndex` (`main.go:199-203`): "rebuilds it from active memories" →
"active nodes". **Do NOT touch `registerGraphLinks` (`main.go:266-275`, added by
161)** — it binds a SECOND pair of `nodes` create/update hooks (`syncHook`) for
wikilink edges and is out of 162's scope.

### Excerpt: `registerSearchIndex` registration + doc comment, LIVE at `0c85d0e` (`main.go:57-58`, `main.go:199-204`)

```go
		registerSearchIndex(se.App)
		registerGraphLinks(se.App)
```

```go
// registerSearchIndex opens the FTS5 sidecar index at pb_data/search.db,
// puts it in app.Store(), and rebuilds it from active memories. On any
// error Balaur boots without the index — LIKE fallback keeps recall live.
// A corrupt file is deleted and one retry is attempted before giving up.
// Record hooks keep the index eventually consistent between boots.
func registerSearchIndex(app core.App) {
```

(162 widens line 200 "active memories" → "active nodes" only; the rest of the
comment stays.)

### Excerpt: `SearchActive`, LIVE at `0c85d0e` (`internal/knowledge/knowledge.go:302-360`)

**160 already rewrote this.** The FTS path resolves ids against `nodes`, but then
re-filters `r.GetString("type") == string(Memory)` and hydrates as Memory; the
fallback is a Go substring scan over `nodes.ListByTypeStatus(app, string(Memory),
…)` (NOT a `FindRecordsByFilter` LIKE clause anymore — the `dbx`-param LIKE
fallback from `72fd762` is gone). 162's only job is to remove the
memory-type narrowing on BOTH paths.

```go
func SearchActive(app core.App, terms []string, limit int) ([]*core.Record, error) {
	// --- FTS5 fast path ---
	if raw, ok := app.Store().GetOk(search.StoreKey); ok {
		if ix, ok := raw.(*search.Index); ok && ix != nil {
			ids, err := ix.Query(terms, limit)
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
			if matchesQuery(Memory, r, t) || strings.Contains(strings.ToLower(r.GetString("category")), t) {
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

**Both paths are memory-scoped**: the FTS path re-filters
`r.GetString("type") == string(Memory)` and hydrates each hit as Memory; the
fallback only lists `type=memory` nodes and uses memory-only `matchesQuery`/
`category`. Step 5 generalizes both — see the **DECIDED per-type-hydrate** code
there: this is NOT a `string(Memory)` → `"nodes"` repoint (160 did that); it is
removing the type narrowing while KEEPING a type-aware hydrate (memory/skill keep
their aliases for the existing `BuildContext`/CLI callers, other types pass raw),
and replacing the memory-only `importance` sort with a node-generic newest-first
order.

### Excerpt: the `recall` agent tool, LIVE at `0c85d0e` (`internal/tools/knowledge.go:119-149`)

The body still reads memory-only hydrated aliases (`category`/`content`) and
Touches the Memory kind — the two reads 162 must generalize.

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
			if err != nil {
				return "", err
			}
			if len(recs) == 0 {
				return "No approved memories match.", nil
			}
			var b strings.Builder
			for _, m := range recs {
				fmt.Fprintf(&b, "- [%s] %s: %s\n",
					m.GetString("category"), m.GetString("title"), m.GetString("content"))
				knowledge.Touch(app, knowledge.Memory, m)
			}
			return b.String(), nil
		},
	}
}
```

`KnowledgeTools` (`internal/tools/knowledge.go:21-32`) returns the slice; **160
already added four node tools** (`node_write`/`node_list`/`node_get`/`node_drop`).
162 inserts `searchTool(app)` into this LIVE list (do NOT drop the node tools):

```go
func KnowledgeTools(app core.App) []agent.Tool {
	return []agent.Tool{
		rememberTool(app),
		recallTool(app),
		skillTool(app),
		proposeSkillTool(app),
		nodeWriteTool(app),
		nodeListTool(app),
		nodeGetTool(app),
		nodeDropTool(app),
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
- **`nodes` field names (CONFIRMED against live 160 schema)**: `type` (select),
  `title` (text), `body` (text/markdown), `status` (select), `props` (json). The
  long markdown text IS `body` (confirmed in the live `index.go` Rebuild/Upsert
  and `knowledge.go` hydrate). **Memory-specific fields `when_to_use` and
  `category` are NOT flat columns — 160 stores them inside `props`**, read via the
  `nodes.PropString(rec, "when_to_use")` helper (`internal/nodes`). Any code that
  needs them (e.g. `nodeExtra`) must read from props, NOT `rec.GetString("when_to_use")`.
  The legacy flat names (`content`/`category`/`when_to_use`/`importance`) appear
  only as read-only ALIASES that `knowledge.hydrate(Memory, rec)` Sets on a record
  in memory — they are valid ONLY after a Memory hydrate, never on a raw node.

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

- `internal/search/index.go` — rename `memories_fts` → `knowledge_fts`; widen
  columns to `(id, kind, title, content, extra)`; **drop 160's `type = 'memory'`
  Rebuild filter and `type != "memory"` Upsert gate** so the source becomes all
  `nodes` filtered to `status='active'`; carry `kind` (the node `type`) and
  `extra` (`nodeExtra`) through Upsert/Rebuild; update the table name in `Delete`
  and `Query`. (Rebuild/Upsert already source `nodes` and read `body`+props — 160
  did that; 162 only un-narrows + renames + adds `kind`/`extra`.)
- `internal/search/index_test.go` — rename the raw `memories_fts` literals to
  `knowledge_fts`; **DELETE/replace `TestUpsertGatesNonMemoryNodes`** (its
  memory-only-gate premise is inverted by 162); add multi-kind rebuild/query/upsert
  tests + a consent-filter test (a proposed node is NEVER returned) + a memory
  recall-hint parity test.
- `main.go` — **doc-comment only** ("active memories" → "active nodes" on
  `registerSearchIndex`). The three hooks already bind `"nodes"` (160). Do NOT
  touch `registerGraphLinks` (161).
- `internal/knowledge/knowledge.go` — **`SearchActive` body only**: it already
  resolves ids against `"nodes"` (160); 162 drops the `type == Memory` re-filter on
  the FTS path and the memory-only `ListByTypeStatus`/`importance` sort on the
  fallback path, hydrating PER TYPE (memory/skill keep their aliases, other types
  pass raw) so search returns mixed types (Step 5). **DECIDED**: the "all active
  nodes" lister the fallback needs is an INLINE
  `app.FindRecordsByFilter("nodes", "status = 'active'", "-updated,-created", 0, 0, nil)`
  inside `SearchActive` — NOT a new `nodes.ListByStatus` helper. This keeps the
  whole change inside `knowledge.go` and inside this Scope list; `internal/nodes`
  stays untouched (out of scope). Do NOT touch any other function in `knowledge.go`
  (hydrate/Touch/lifecycle are 160's).
- `internal/tools/knowledge.go` — add a `searchTool` (cross-type search) into the
  LIVE `KnowledgeTools` list (which already holds the four `node_*` tools); **also
  fix `recallTool`** to read `type`/`body` off node records and to drop the
  `Touch(app, knowledge.Memory, m)` call (Touch itself still works but is
  memory-specific; recall on mixed types should not Touch as Memory).
- `internal/cli/knowledge.go` — add a `nodeCmd(app)` parent command with a
  `search` subcommand (`balaur node search <terms…>`) exposing the cross-type
  search as the standard JSON envelope. (DECIDED: ship `node search` — see Step 7.)
- `internal/cli/cli.go` — register `nodeCmd(app)` in the LIVE `root.AddCommand(...)`
  list (`cli.go:54-72`), alongside `memoryCmd`/`skillCmd`/`noteCmd`.
- `internal/self/knowledge.md` — one-line/paragraph update: unified FTS over
  active nodes (all knowledge types), consent-filtered.

**Out of scope** (do NOT touch — reason given):

- `migrations/` — plan **160 owns the `nodes`/`edges` baseline migration**.
  162 reads `nodes`; it never declares or alters the collection. Touching
  migrations here collides with 160.
- `internal/knowledge/knowledge.go` — **everything EXCEPT the `SearchActive`
  body**. The lifecycle/propose/transition/hydrate/Touch helpers are 160's and
  already correctly nodes-backed; 162 edits ONLY `SearchActive`'s memory-type
  narrowing (Step 5). If `SearchActive` no
  longer routes through `search.Index.Query` at all, that is a STOP condition.
- `internal/search/fts5_test.go` — the driver spike; unrelated, leave byte-identical.
- `internal/nodes/` — **out of scope. Do NOT add `nodes.ListByStatus` or any other
  helper here.** Step 5's "all active nodes" fallback is satisfied by an INLINE
  `app.FindRecordsByFilter("nodes", "status = 'active'", "-updated,-created", 0, 0, nil)`
  inside `SearchActive` (decided). Touching `internal/nodes` would trip the
  out-of-scope STOP condition for no benefit.
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

In `Rebuild`, change the `DELETE`, the source-query FILTER, the prepared
`INSERT`, and the per-record `Exec` to the unified table. **160's live source is
already `FindRecordsByFilter("nodes", "type = 'memory' && status = 'active'", …)`
— 162 only drops the `type = 'memory'` clause** so all active types index:
filter becomes `"status = 'active'"`. `kind` is the node's `type`; `content` ←
node `body`; `extra` ← `nodeExtra(r)` (which reads `when_to_use`/`category` from
props — see the helper below). This keeps the recall parity the live
`memories_fts` had (it indexed `when_to_use` and `category`):

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
searchable text into the `extra` column. **CONFIRMED against live 160 code**: the
memory recall hint `when_to_use` AND `category` live in `props`, read via
`nodes.PropString` (the `internal/nodes` package is already imported in
`index.go`). The current `memories_fts` indexes BOTH `when_to_use` AND `category`
as searchable columns (live `index.go:81-82` / `115-116`), so to keep FULL parity
`nodeExtra` should fold BOTH into `extra` — otherwise 162 narrows search vs. 160's
live behavior (a deliberate decision; see the design-conflict note below):

```go
// nodeExtra returns type-specific searchable text for a node's `extra` FTS
// column. v1 preserves the old memory recall hints (when_to_use + category,
// both stored in props by plan 160) so search parity with the memory-only
// memories_fts is not lost; other types contribute nothing extra yet.
func nodeExtra(r *core.Record) string {
	if r.GetString("type") == "memory" {
		return strings.TrimSpace(
			nodes.PropString(r, "when_to_use") + " " + nodes.PropString(r, "category"))
	}
	return ""
}
```

(`content` column ← node `body` field; `kind` column ← node `type` field;
`extra` column ← `nodeExtra(r)`.)

> **DECIDED (no longer open): `nodeExtra` packs BOTH `when_to_use` AND
> `category`.** Rationale: the LIVE `memories_fts` 160 shipped indexes `category`
> as a searchable column, and the live `SearchActive` fallback matches on
> `category` (`strings.Contains(... r.GetString("category") ...)`). Dropping
> `category` from the unified index would be a REGRESSION vs. shipped behavior, so
> the helper above (folding both fields, read from `props` via `nodes.PropString`)
> is the instruction — use it verbatim. Do NOT narrow `extra` to `when_to_use`
> only. Consequence: the `SearchActive` fallback's `category` substring match
> stays for memory hits (Step 5 keeps it), so the FTS and fallback paths agree.

**Verify**: `grep -n '"nodes"\|knowledge_fts\|memories_fts\|type = .memory.' internal/search/index.go`
shows `nodes` + `knowledge_fts`, NO `memories_fts`, and NO
`type = 'memory'` clause in the Rebuild filter (162 dropped it).

### Step 3: Remove the memory-only gate; switch `Upsert`, `Delete`, `Query` to the unified table

`Upsert`: **REMOVE 160's `if rec.GetString("type") != "memory" { return nil }`
gate** (live `index.go:104-106`) so all active node types index. Keep the
delete-first idempotency and the `status != "active"` early return EXACTLY; change
the INSERT to the unified columns and read `kind`/`body`/`extra`:

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

### Step 4: Fix only the stale doc comment in `main.go` (hooks already bind `nodes`)

**160 already repointed the three FTS bindings to `"nodes"` (live
`main.go:255-257`) — DO NOT touch them.** The only edit here is the stale doc
comment on `registerSearchIndex` (live `main.go:199-203`): line 200 reads
"rebuilds it from active memories" → change to "rebuilds it from active nodes".
Leave the rest of the comment, the `openAndRebuild` retry logic, and the
`upsertHook`/`deleteHook` closures byte-identical.

**DO NOT touch `registerGraphLinks` (`main.go:266-275`, added by 161).** It binds
a SECOND `nodes` create/update hook pair for wikilink edges — that is 161's, not
162's. 162 makes zero functional `main.go` changes; only the comment.

**Verify**: `grep -n 'OnRecordAfter.*Success("nodes")' main.go` shows the three
search bindings PLUS 161's two graph bindings (five total) — all `("nodes")`, zero
`("memories")`. `grep -n 'active nodes' main.go` shows the updated comment. Then
`CGO_ENABLED=0 go build ./...` → exit 0.

### Step 5: De-narrow `SearchActive` from memory-only to cross-type

This is the change that makes "find anything" actually return mixed-type hits.
**160 already did the id→record repoint** (`FindRecordsByIds("nodes", ids)`), so
this is NOT a `string(Memory)`→`"nodes"` repoint. The live function (excerpt
above, `knowledge.go:302-360`) resolves against `nodes` but stays memory-scoped in
THREE places 162 must change:

1. **FTS path type re-filter** — drop `r.GetString("type") == string(Memory)` so
   any active node passes; and the per-hit hydrate.
2. **Fallback source** — `nodes.ListByTypeStatus(app, string(Memory), …)` lists
   only memory nodes; widen it to all active nodes.
3. **Fallback matcher / sort** — `matchesQuery(Memory, …)`, the `category`
   substring, and the `importance` sort are memory-only concepts.

> **DECIDED (no longer open): hydrate PER node type — keep the hydrate, make it
> type-aware. Do NOT switch to raw records.** Reason the "stop hydrating / return
> raw" option is REJECTED: `SearchActive` has THREE production callers, and two of
> them read hydrated memory aliases that only exist after `hydrate(Memory, r)`:
> (1) `internal/knowledge/context.go:47` (`BuildContext` → `writeMemoryLine` reads
> `content`/`category` into the system-prompt "recalled" block); (2)
> `internal/cli/knowledge.go:173` (`memory recall` → `memoryJSON` reads
> `content`/`category`/`importance`/`when_to_use`). Returning raw records would
> silently blank those fields — a real regression in context assembly and the CLI
> JSON. (`recallTool`, the 3rd caller, is fixed in Step 6 either way.) So the
> de-narrowing keeps memory hits hydrated and lets other types pass through raw:
> `switch r.GetString("type")` → `hydrate(Memory, r)` for memory (and
> `hydrate(Skill, r)` for skill), else append the raw record. The FTS path already
> preserves rank order; the fallback drops the memory-only `importance` sort for a
> node-generic newest-first order (see below).

Concretely — DECIDED per-type-hydrate path (apply exactly this):

```go
		// FTS path — before (memory-narrowed + Memory-only hydrate):
		for _, r := range recs {
			if r.GetString("type") == string(Memory) && r.GetString("status") == StatusActive {
				active = append(active, hydrate(Memory, r))
			}
		}
		// after — any active node passes; hydrate per type so the existing
		// memory callers (BuildContext, CLI memory recall) keep their aliases,
		// and non-memory types pass through raw:
		for _, r := range recs {
			if r.GetString("status") != StatusActive {
				continue
			}
			switch r.GetString("type") {
			case string(Memory):
				active = append(active, hydrate(Memory, r))
			case string(Skill):
				active = append(active, hydrate(Skill, r))
			default:
				active = append(active, r)
			}
		}
```

```go
		// fallback — before (memory-only list + matcher + importance sort):
		recs, err := nodes.ListByTypeStatus(app, string(Memory), StatusActive)
		...
		hydrateAll(Memory, recs)
		// ... matchesQuery(Memory, r, t) || strings.Contains(... category ...)
		// ... sort.SliceStable(... importance ...)
		// after — list ALL active nodes (inline FindRecordsByFilter — see DECIDED
		// note below), hydrate per type, and substring-match generically.
		recs, err := app.FindRecordsByFilter(
			"nodes", "status = 'active'", "-updated,-created", 0, 0, nil)
		if err != nil {
			return nil, err
		}
		var matched []*core.Record
		for _, r := range recs {
			// hydrate per type so memory hits still carry content/category aliases
			switch r.GetString("type") {
			case string(Memory):
				hydrate(Memory, r)
			case string(Skill):
				hydrate(Skill, r)
			}
			for _, t := range terms {
				t = strings.ToLower(strings.TrimSpace(t))
				if t == "" {
					continue
				}
				// match on the universal title/body, plus the memory category
				// alias (present only after a Memory hydrate) for parity with
				// the FTS extra column (Step 2 keeps category searchable).
				if strings.Contains(strings.ToLower(r.GetString("title")), t) ||
					strings.Contains(strings.ToLower(r.GetString("body")), t) ||
					strings.Contains(strings.ToLower(r.GetString("category")), t) {
					matched = append(matched, r)
					break
				}
			}
		}
		// FindRecordsByFilter already returns newest-first (-updated,-created);
		// drop the memory-only importance sort entirely.
		if limit > 0 && len(matched) > limit {
			matched = matched[:limit]
		}
		return matched, nil
```

> **DECIDED (no longer open): use the INLINE `app.FindRecordsByFilter("nodes",
> "status = 'active'", "-updated,-created", 0, 0, nil)` shown above — do NOT add a
> `nodes.ListByStatus` helper.** `internal/nodes` exposes only
> `ListByTypeStatus(type, status)` at `0c85d0e`; rather than add a type-agnostic
> lister (which would touch `internal/nodes`, an out-of-scope file, and could trip
> the STOP condition), inline the filter directly inside the `SearchActive` body.
> This keeps the entire change inside `knowledge.go`'s `SearchActive` and inside
> the declared Scope. The `-updated,-created` sort makes the result newest-first,
> which replaces the memory-only `importance` sort.

Touch NOTHING else in `knowledge.go` (the hydrate/Touch/lifecycle helpers are
160's).

**Verify**: inside `SearchActive`, `string(Memory)` must no longer appear in a
NARROWING position — i.e. there is no `if r.GetString("type") == string(Memory)`
filter, no `nodes.ListByTypeStatus(app, string(Memory), …)` call, and no
`importance` sort. (It MAY still appear in the per-type-hydrate `switch` —
`case string(Memory):` / `case string(Skill):` — that is the decided cross-type
path, not a narrow.) Concretely: `grep -n 'type.*==.*string(Memory)\|ListByTypeStatus(app, string(Memory)\|GetInt("importance")' internal/knowledge/knowledge.go`
shows none of these INSIDE `SearchActive`. Then `CGO_ENABLED=0 go build
./internal/knowledge/...` → exit 0, and a multi-type search test (Test plan)
returns a note AND a memory.

### Step 6: Fix `recall` and add the cross-type `search` agent tool

**First fix `recallTool`** (live `internal/tools/knowledge.go:119-149`). Its body
reads `m.GetString("category")` and `m.GetString("content")` — these are
memory-only hydrated ALIASES. After Step 5 `SearchActive` returns MIXED types
(memory hits stay hydrated, other types are raw), so a non-memory hit has no
`category`/`content` alias and these reads go empty for it. It also calls
`knowledge.Touch(app, knowledge.Memory, m)`, which records usage against the
Memory kind specifically and is wrong for a non-memory hit. Read the universal
node fields `type`/`body` and drop the Memory-kind Touch:

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

(Dropping the `Touch` call is the KISS fix. NOTE — live `knowledge.Touch(app,
kind, rec)` (`knowledge.go:234-244`) DOES still work: it bumps `use_count` in
props and saves the node, taking a `Kind` only to drive the post-save
`hydrate(kind, rec)`. So it is not broken; it is just memory-specific. The CLI
recall path already runs without Touch. Drop it from `recallTool` rather than
calling `Touch(app, knowledge.Memory, m)` on a possibly-non-memory hit. There is
no node-wide `Touch(app, rec)` helper today, so do NOT invent one — just remove
the call.)

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

Register it in `KnowledgeTools` — **insert into the LIVE list** (160 already added
the four `node_*` tools; do not drop them):

```go
func KnowledgeTools(app core.App) []agent.Tool {
	return []agent.Tool{
		rememberTool(app),
		recallTool(app),
		searchTool(app),
		skillTool(app),
		proposeSkillTool(app),
		nodeWriteTool(app),
		nodeListTool(app),
		nodeGetTool(app),
		nodeDropTool(app),
	}
}
```

> **Important**: 162 owns `SearchActive`'s cross-type de-narrowing (Step 5), so it
> stays the single helper both `recall` and `search` call. **CONFIRMED: 160 did
> NOT rename it** — it is still `knowledge.SearchActive(app, terms, limit)` and
> still routes through `search.Index.Query` with a defensive `status == StatusActive`
> re-filter (live `knowledge.go:302-360`). Reuse it; do NOT re-implement index
> querying inside `tools`.

**Verify**: `CGO_ENABLED=0 go build ./internal/tools/...` → exit 0; then
`grep -n 'searchTool\|"search"' internal/tools/knowledge.go` shows the tool and
its registration, and
`grep -n 'GetString("category")\|GetString("content")\|knowledge.Memory' internal/tools/knowledge.go`
shows NONE of those on the recall path (the `remember` tool may still build a
`MemoryProposal`, which is 160's to keep — only `recallTool`'s field reads must
be gone).

### Step 7: Add the `node search` CLI subcommand

The repo's CLI uses PARENT commands with subcommands (live: `memoryCmd`
`knowledge.go:111`, `skillCmd` `:225`, and 160's `noteCmd` `:294`, each a
container), registered via `root.AddCommand(...)` in `internal/cli/cli.go:54-72`.
Match that shape: add a `nodeCmd(app)` parent in `internal/cli/knowledge.go` with
one `search <terms...>` subcommand. The surface is `balaur node search <terms…>`.
Mirror the `run(app, cliKind, …)` envelope pattern used by `statusListCmd`.

> **NAMING — DECIDED: ship `node search` as drafted.** 160 already shipped a
> `note` parent command (`noteCmd`, `cli.go:59`); a new `node` parent sits one
> letter from `note`, but the `node search` surface mirrors 160's `node_*` agent
> tools and the `nodes` collection, so it is the consistent choice — use it. (A
> flat top-level `balaur search` verb is a viable alternative but is NOT what this
> plan ships; if a future UX pass wants it, that is a deliberate rename, not part
> of 162.) Build exactly the `nodeCmd`/`nodeSearchCmd` below.

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
`root.AddCommand(...)` call (live `cli.go:54-72`), alongside `memoryCmd`/
`skillCmd`/`noteCmd`. The LIVE block (insert `nodeCmd(app)` after `skillCmd(app)`):

```go
	root.AddCommand(
		chatCmd(app),
		taskCmd(app),
		memoryCmd(app),
		skillCmd(app),
		nodeCmd(app),
		noteCmd(app),
		lifeCmd(app),
		journalCmd(app),
		dayCmd(app),
		recapCmd(app),
		historyCmd(app),
		auditCmd(app),
		verifyCmd(app),
		modelCmd(app),
		selfCmd(app),
		extCmd(app),
		doctorCmd(app),
		seedCmd(app),
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

In `internal/self/knowledge.md`, update the sentences that describe search/recall
(the file does NOT contain the literal `memories_fts`, so do not chase that token
— fix the prose). The LIVE wording (refreshed at `0c85d0e`):

- Live lines 79-81: `search (FTS5 sidecar index — bm25-ranked recall rebuilt on
  boot, synced on write; pb_data/search.db is disposable and safe to delete)`.
  Widen it to say the index spans **active nodes of all knowledge types**
  (note/memory/skill/journal/typed-objects), consent-filtered (proposed/rejected
  nodes are never indexed), and name the table `knowledge_fts`.
- Live line 114: `recall searches approved memory nodes;` (note: 160 already
  changed "approved memories" → "approved memory nodes" — the OLD literal in the
  earlier plan draft is gone). This is the real stale sentence now. Replace it so
  it reads: recall and the cross-type `search` tool query approved **nodes (all
  knowledge types)** via the unified `knowledge_fts` index, with a deterministic
  substring fallback when the sidecar is unavailable. (Live line 100 already says
  "Traversal and search filter to status=active" — that stays correct; you may
  extend it to name the `knowledge_fts` table.)

Mention the `search` agent tool and the `balaur node search` CLI verb.

**Verify**: `grep -in 'knowledge_fts\|active nodes\|all knowledge types' internal/self/knowledge.md`
returns at least one match (the NEW wording is present). Also confirm the stale
sentence is gone: `grep -in 'recall searches approved memory nodes' internal/self/knowledge.md`
returns nothing (it was reworded to cover all node types).

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

All tests live in `internal/search/index_test.go`. **160 already rewrote the seed
helpers to target `nodes`** — the live helpers are `seedMemoryNode(t, app, title,
content, category, importance, status)` (Sets `type=memory`, `title`, `body`,
`status`, and a `props` map of `category`/`importance`/`source`), plus the
convenience wrappers `seedActiveMemory` and `seedProposedMemory`. They already
compile against the live `memories_fts` (5-col). **What breaks under 162 is the
SIX raw FTS literals** — they still embed `INSERT INTO memories_fts(id, title,
content, when_to_use, category) VALUES (?, ?, ?, ?, ?)` (live lines 60, 193, 213
have inline inserts; the `Rebuild`/`Upsert` tests go through the renamed code) and
the literal table name `memories_fts`. Those must become `knowledge_fts(id, kind,
title, content, extra)`. **Also note: 160 added `TestUpsertGatesNonMemoryNodes`
(live `index_test.go:153-187`) — it asserts a `note` node NEVER lands in the
index. 162 REMOVES the memory-only gate, so this test's premise inverts: a note
node now SHOULD index. This test must be rewritten or deleted (see Step B).**

**Step A — generalize the seed helpers.** The live `seedMemoryNode`/
`seedActiveMemory`/`seedProposedMemory` are memory-specific (they hard-code
`type=memory` and a memory props map). For the multi-kind tests, ADD a generic
`seedActiveNode`/`seedProposedNode` ALONGSIDE them (keep the memory helpers — the
parity test still needs a memory node WITH props):

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

(The live `seedMemoryNode` stores `category`/`when_to_use` in `props` — keep it
for `TestExtraPreservesMemoryRecallHint`, which needs a memory node whose props
carry the recall hint that `nodeExtra` reads.)

(The live `nodes` collection accepts `type`/`title`/`body`/`status` via `Save`
with no extra required fields — `seedMemoryNode` proves it. `props` is optional.)

**Step B — rewrite the raw FTS literals AND fix the inverted gate test
(mandatory).** Three tests embed literal `INSERT INTO memories_fts(id, title,
content, when_to_use, category) VALUES (?, ?, ?, ?, ?)` (live: `TestOpenCreatesSchema`
`index_test.go:60`, `TestQueryInjectionSafe` `:193`, `TestDeleteRemovesRecord`
`:213`). Update each to `INSERT INTO knowledge_fts(id, kind, title, content, extra)
VALUES (?, ?, ?, ?, ?)` (supply a `kind`, e.g. `"note"`, and `""` for `extra`).
`TestRebuildAndQuery` and `TestUpsertNonActive` seed through `seedActiveMemory`/
`seedMemoryNode` and exercise the renamed Rebuild/Upsert — they keep working once
the code is renamed, but ADD multi-kind coverage (case 1 below). `TestQueryEmpty`
only calls `Query`, needs no change.

**CRITICAL — `TestUpsertGatesNonMemoryNodes` (live `index_test.go:153-187`) now
asserts the WRONG behavior.** It Upserts an active `note` node and asserts it does
NOT appear (`len(ids) != 0` → fail). 162 removes the memory-only gate, so the note
SHOULD index. **Delete this test** (its job — proving the gate — no longer exists)
and replace it with `TestRebuildMultiKind` / a generic upsert-any-type assertion
(case 1 below). The memory half of it (body+props.when_to_use searchable) is
preserved by `TestExtraPreservesMemoryRecallHint` (case 4).

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
   **CONFIRMED: 160 keeps the recall hint in `props.when_to_use`** (read by
   `nodeExtra` via `nodes.PropString`). Use the live `seedMemoryNode` helper, which
   already writes a `props` map — extend it (or add a variant) so `props.when_to_use`
   carries a unique term NOT present in title/body; `Rebuild`; assert
   `Query(["that-hint-term"])` returns the node id. No `t.Skip` is needed — the
   field exists. Category parity is DECIDED-in (Step 2 keeps `category` in
   `nodeExtra`), so ALSO assert a second case: a unique `props.category` term NOT
   present in title/body must match too.
5. **A `recall`/`search` integration test** — check `internal/knowledge/knowledge_test.go`
   for existing `TestSearchActive*` coverage. Since 160 made `SearchActive`
   nodes-backed but memory-scoped, the existing tests assert hydrated memory
   behavior. Because Step 5 KEEPS per-type hydration for memory hits, those
   existing memory assertions should STILL PASS unchanged — do not rewrite them to
   expect raw records. After Step 5's de-narrowing, ADD a case that seeds a memory
   AND a note, both active, and asserts BOTH are returned (proving cross-type; the
   note hit is raw so assert on its `type`/`title`/`body`, the memory hit is
   hydrated so its `content`/`category` aliases still resolve), plus a proposed
   node that is never returned. Confirm green.

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
- [ ] `grep -n 'OnRecordAfter.*Success("nodes")' main.go` shows FIVE bindings
      (3 search from `registerSearchIndex` + 2 graph from 161's
      `registerGraphLinks`); `grep -n 'OnRecordAfter.*Success("memories")' main.go`
      returns nothing. (162 changes none of these — they already bind `nodes`.)
- [ ] `grep -n 'searchTool\|"search"' internal/tools/knowledge.go` shows the new
      agent tool defined and registered in `KnowledgeTools` (alongside the four
      `node_*` tools 160 added — those must remain).
- [ ] `recall` no longer reads memory-only fields/kind:
      `grep -n 'GetString("category")\|GetString("content")\|knowledge.Memory' internal/tools/knowledge.go`
      shows none of these inside `recallTool` (the `remember` tool's
      `MemoryProposal`/`MemoryProposal` fields are 160's to keep).
- [ ] `SearchActive` is no longer memory-NARROWED (but it MAY still hydrate
      per-type): inside `SearchActive` there is no `if r.GetString("type") ==
      string(Memory)` filter, no `nodes.ListByTypeStatus(app, string(Memory), …)`
      call, and no `GetInt("importance")` sort —
      `grep -n 'type.*==.*string(Memory)\|ListByTypeStatus(app, string(Memory)\|GetInt("importance")' internal/knowledge/knowledge.go`
      shows none of these in the `SearchActive` body. (`case string(Memory):` /
      `case string(Skill):` in the per-type-hydrate switch is EXPECTED — that is the
      decided cross-type path. 160 already resolves ids against `"nodes"`; 162 drops
      the narrowing on both the FTS and fallback paths.)
- [ ] `grep -n 'nodeCmd\|nodeSearchCmd' internal/cli/knowledge.go` shows the
      parent + `search` subcommand defined; `grep -n 'nodeCmd(app)' internal/cli/cli.go`
      shows it registered in `root.AddCommand(...)`.
- [ ] `grep -in 'knowledge_fts\|active nodes\|all knowledge types' internal/self/knowledge.md`
      matches (new wording present); `grep -in 'recall searches approved memory nodes' internal/self/knowledge.md`
      returns nothing (live sentence reworded to cover all node types).
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
- `internal/knowledge`'s `SearchActive` is GONE entirely or no longer routes
  through `search.Index.Query` (Step 5/6) — the index integration contract changed.
  (CONFIRMED at `0c85d0e`: it exists, routes through `Query`, resolves ids against
  `"nodes"`, but re-filters to `type == Memory`. That is NOT a stop — Step 5 is
  exactly the de-narrowing fix.)
- The "Current state" excerpts in `internal/search/index.go` do not resemble the
  live code AS REFRESHED HERE (the table is `memories_fts`, nodes-backed,
  memory-only-gated, props-read via `nodes.PropString`; the quoting/`ORDER BY
  rank` block in `Query` is byte-identical to the pre-160 baseline). If someone has
  since renamed the table to `knowledge_fts` (162 already partly landed?) or
  changed the quoting block, the tree drifted further — reconcile before editing.
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
  In v1 the only branch populated is `memory` → its recall hint `when_to_use`
  (and, per the Step-2 design decision, `category`), both read from `props` via
  `nodes.PropString`. This preserves parity with the memory-only `memories_fts`
  160 shipped, whose `when_to_use` AND `category` columns were both searchable. The
  column exists precisely so future types (a person's aliases, a book's author, a
  flattened `props` blob) can be indexed WITHOUT another FTS schema migration: add
  a branch to `nodeExtra`, do not add columns. **The `category`-parity decision is
  DECIDED: `nodeExtra` keeps category** (see Step 2). The `SearchActive` fallback's
  `category` substring match is kept in tandem (Step 5), so the FTS `extra` column
  and the fallback path stay in agreement.
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
    121). Do not revive. Lexical FTS5 + substring fallback is the v1 contract.
  - **Per-type ranking weights / bm25 tuning** — `ORDER BY rank` is the v1
    default; tune only with a measured relevance complaint in hand.
  - **`category` indexing — DECIDED: KEEP.** The LIVE `memories_fts` (160) indexes
    the memory `category` (from props) as a searchable column, and the live
    `SearchActive` fallback matches on it. Dropping it would be a REGRESSION vs.
    shipped behavior, so `nodeExtra` folds `category` (with `when_to_use`) into
    `extra`, and the fallback keeps its `category` substring match — the two paths
    agree. Not deferred; this is the implemented behavior.
  - **`recall` vs `search`**: after Step 5/6 both call the same cross-type
    `SearchActive`, so `recall` now returns mixed node types too — its
    description still frames it as a memory-recall aid, but it is no longer
    collection-scoped. `search` is the explicit cross-type verb. If a future
    cleanup wants to merge or re-narrow them, that is a deliberate UX decision,
    not a bug — keep both until then.
  - **Recall-hint parity is intact.** CONFIRMED: 160 carries the memory recall
    hint in `props.when_to_use` (read by `nodeExtra` via `nodes.PropString`), so
    the parity test runs and asserts (no skip). This bullet's earlier "if 160
    dropped when_to_use" caveat no longer applies.
- A reviewer should scrutinize: that NO `proposed` node can ever reach the index
  (the spine), that `Query`'s injection-safe quoting is byte-identical to before,
  and that `git status` shows zero `migrations/` changes.
