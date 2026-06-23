# Plan 160: Greenfield knowledge spine — `nodes` + `edges`, memories/skills/journal folded into typed nodes

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` (lowercase on disk) — unless a reviewer dispatched you
> and told you they maintain the index.
>
> **Sandbox note**: in a TLS-intercepting sandbox (Hyperagent), Go commands
> need the GOPROXY shim — see `docs/hyperagent-sandbox.md`. GOSUMDB stays on;
> never weaken checksum verification.
>
> **This is a LARGE foundation plan.** The schema change is ATOMIC: removing
> `memories`/`skills` and splitting journal out of `entries` means the build
> breaks until *every* referencing call site is ported in this same plan. Do
> not split it across commits in a way that leaves `go build` red between
> logical units — order the steps so the tree compiles at each step boundary
> wherever possible (the migration + domain package land first; callers switch;
> old collections are removed last). If the surface explodes past the
> enumerated touch-points, see STOP conditions.
>
> **Drift check (run first)**:
> `git diff --stat 72fd762..HEAD -- migrations/ internal/knowledge/ internal/life/ internal/search/ internal/tools/knowledge.go internal/tools/journal.go internal/seed/ internal/self/ internal/cli/knowledge.go internal/cli/life.go internal/web/knowledge.go internal/web/cards.go internal/feature/knowledgecards/ internal/feature/journalcards/ internal/cards/cards.go main.go`
> If any in-scope file changed since this plan was written (commit `72fd762`),
> compare the "Current state" excerpts against the live code before proceeding;
> on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: L
- **Risk**: HIGH
- **Depends on**: none (this is the foundation; 161/162/163 depend on it)
- **Category**: migration
- **Planned at**: commit `72fd762`, 2026-06-23

## Why this matters

Balaur's knowledge today is two parallel collections (`memories`, `skills`)
plus a journal smuggled into the life-log `entries` collection as
`kind='journal'`. The owner wants the LogSeq/Capacities shape: **everything is
a typed, linkable node**. This plan rewrites the consolidated baseline schema
to a single `nodes` collection (note / memory / skill / journal / person /
book / idea / … — extensible) plus an `edges` collection (node↔node links),
carrying the existing consent lifecycle (`proposed → active → archived /
rejected`) onto node rows. It is the foundation: plan 161 adds `[[wikilinks]]`
and backlinks on top of `edges`, 162 unifies search over `nodes`, 163 adds
graph traversal — none can land until this is green. This is a **greenfield**
change: `pb_data/` is disposable, there is no data migration, the new baseline
migration IS the final schema.

## Current state

### The collections / packages this plan rewrites

- `migrations/1749600000_init.go` — the single consolidated baseline migration
  (plan 156). Creates all 14 collections in one `InitCollections`. **Rewrite**:
  add `nodes` + `edges`, remove `memories` + `skills`, slim `entries`'s journal
  role (entries stays; journal moves to `type=journal` nodes).
- `migrations/schema_test.go` — `TestSchemaBaseline` asserts the exact
  collection/field/index set. **Rewrite** to the new schema.
- `migrations/timestamp_uniqueness_test.go` — generic; only relevant if you add
  a new migration file (you should NOT — edit the existing baseline in place).
- `internal/knowledge/knowledge.go` — memory/skill domain: `ProposeMemory`,
  `ProposeSkill`, `Transition`, `UpdateFields`, `Touch`, `ListByStatus`,
  `FilterActive`, `SearchActive`, `UpfrontMemories`, `ActiveSkills`,
  `LoadSkill`. **Port** onto `nodes` (type+status filters; old fields → props).
- `internal/life/journal.go` — `JournalWrite`/`JournalDrop` write `kind=journal`
  rows in `entries`. **Port** `JournalWrite` to upsert a `type=journal` node
  (one per day, born active, verbatim).
- `internal/life/day.go` — `Day()` reads `entries` `kind='journal'` for the day
  page's `Journal` list. **Repoint** the journal read to `type=journal` nodes.
- `internal/search/index.go` + `main.go` (`registerSearchIndex`, lines 197-256)
  — FTS hooks bind to the `memories` collection. **Repoint the hook collection
  name to `nodes`** (the FTS *schema* stays plan 162's job — here only the
  collection name the hooks bind to and the rebuild query's collection/filter
  change so the binary boots; do NOT redesign the table — see Scope).
- `internal/tools/knowledge.go` + `internal/tools/journal.go` — the
  `remember`/`recall`/`skill`/`propose_skill`/`journal_write` tools.
  **Repoint** onto `nodes` via the ported domain package. Add
  `node_write`/`node_list`/`node_get`/`node_drop` for owner-authored notes +
  typed objects.
- `internal/seed/seed.go` — seeds memories/skills via the knowledge package and
  journal via life. **Repoint** to seed nodes.
- `internal/self/knowledge.md` — the running binary's self-description (data
  model + capabilities). **Rewrite** the collections list + consent model.
- `internal/cli/knowledge.go`, `internal/cli/life.go` — `balaur memory`,
  `balaur skill`, `balaur journal`, `balaur day`. **Repoint** onto nodes; add
  `balaur note`.
- `internal/web/knowledge.go`, `internal/web/cards.go`,
  `internal/feature/knowledgecards/*`, `internal/feature/journalcards/day.go`,
  `internal/cards/cards.go` — the cards + the `/ui/show/{type}` dispatch.
  **Repoint** memory/skill/journal cards to type-filtered active nodes; add a
  `note` card type so `/ui/show/note?id=...` works (the route 161/163 depend on).

### Verbatim excerpts (confirm you are looking at the right code)

**The baseline migration's collection list and the two collections to remove**
— `migrations/1749600000_init.go:25-30`:

```go
// collectionNames in dependency order (relations point left); dropped in reverse.
var collectionNames = []string{
	"heads", "conversations", "messages", "memories", "skills", "audit_log",
	"summaries", "tasks", "entries", "extensions",
	"llm_providers", "llm_models", "llm_settings", "owner_settings",
}
```

`migrations/1749600000_init.go:86-124` (the `memories` and `skills` blocks to
REMOVE — read them so you know exactly which fields map into `nodes.props`):

```go
	memories := core.NewBaseCollection("memories")
	setOwnerRules(memories, owner)
	memories.Fields.Add(
		&core.TextField{Name: "title", Required: true, Max: 300},
		&core.TextField{Name: "content", Max: 100000},
		&core.TextField{Name: "source", Max: 300},
		&core.SelectField{Name: "status", Required: true, Values: []string{"proposed", "active", "archived", "rejected"}},
		&core.SelectField{Name: "category", Values: []string{"fact", "preference", "person", "project", "context"}},
		&core.NumberField{Name: "importance", OnlyInt: true, Min: types.Pointer(1.0), Max: types.Pointer(5.0)},
		&core.TextField{Name: "when_to_use", Max: 500},
		&core.DateField{Name: "last_used"},
		&core.NumberField{Name: "use_count", OnlyInt: true},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	memories.AddIndex("idx_memories_status", false, "status", "")
	memories.AddIndex("idx_memories_status_importance", false, "status, importance", "")
	if err := app.Save(memories); err != nil {
		return err
	}

	skills := core.NewBaseCollection("skills")
	setOwnerRules(skills, owner)
	skills.Fields.Add(
		&core.TextField{Name: "name", Required: true, Max: 120},
		&core.TextField{Name: "description", Max: 2000},
		&core.TextField{Name: "content", Max: 100000},
		&core.SelectField{Name: "status", Required: true, Values: []string{"proposed", "active", "archived", "rejected"}},
		&core.TextField{Name: "when_to_use", Max: 500},
		&core.DateField{Name: "last_used"},
		&core.NumberField{Name: "use_count", OnlyInt: true},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	skills.AddIndex("idx_skills_name", true, "name", "")
	skills.AddIndex("idx_skills_status", false, "status", "")
	if err := app.Save(skills); err != nil {
		return err
	}
```

**The relation-field exemplar to copy for `edges.source`/`edges.target`** —
`migrations/1749600000_init.go:69-84` (messages → conversations,
`CascadeDelete: true`):

```go
	messages := core.NewBaseCollection("messages")
	setOwnerRules(messages, owner)
	messages.Fields.Add(
		&core.RelationField{Name: "conversation", Required: true, CollectionId: conversations.Id, CascadeDelete: true},
		&core.SelectField{Name: "role", Required: true, Values: []string{"system", "user", "assistant", "tool"}},
		...
	)
	messages.AddIndex("idx_messages_conv_created", false, "conversation, created", "")
```

**The unique-multi-column-index exemplar** — `migrations/1749600000_init.go:157`
(summaries unique period; copy this shape for the `edges` unique index):

```go
	summaries.AddIndex("idx_summaries_period", true, "conversation, period_type, period_start", "")
```

**`setOwnerRules`** — `migrations/1749600000_init.go:293-299` (apply to `nodes`
and `edges` exactly as the other base collections do):

```go
func setOwnerRules(c *core.Collection, owner *string) {
	c.ListRule = owner
	c.ViewRule = owner
	c.CreateRule = owner
	c.UpdateRule = owner
	c.DeleteRule = owner
}
```

**The consent lifecycle to preserve** — `internal/knowledge/knowledge.go:34-40`
and `:118-124`:

```go
const (
	StatusProposed = "proposed"
	StatusActive   = "active"
	StatusArchived = "archived"
	StatusRejected = "rejected"
)
```
```go
var validTransitions = map[string][]string{
	StatusProposed: {StatusActive, StatusRejected},
	StatusActive:   {StatusArchived},
	StatusArchived: {StatusActive},
	StatusRejected: {},
}
```

**The five memory categories to preserve as `props.category`** —
`internal/knowledge/knowledge.go:56` and the migration enum at
`1749600000_init.go:93`: `fact | preference | person | project | context`.

**Journal-in-entries (the thing being split out)** — `internal/life/journal.go:32-35`:

```go
	rec := core.NewRecord(col)
	rec.Set("kind", "journal")
	rec.Set("text", text)
	rec.Set("noted_at", notedAt.UTC())
```
and the day-page read in `internal/life/day.go:28-35`:

```go
	// Journal entries: kind='journal', noted_at in [ds, de)
	recs, err := app.FindRecordsByFilter("entries",
		"kind = 'journal' && noted_at >= {:s} && noted_at < {:e}", "noted_at", 200, 0,
		dbx.Params{"s": store.PBTime(ds), "e": store.PBTime(de)})
```

**The FTS hooks bound to `memories`** — `main.go:253-255`:

```go
	app.OnRecordAfterCreateSuccess("memories").BindFunc(upsertHook)
	app.OnRecordAfterUpdateSuccess("memories").BindFunc(upsertHook)
	app.OnRecordAfterDeleteSuccess("memories").BindFunc(deleteHook)
```
and the rebuild query — `internal/search/index.go:61`:

```go
	recs, err := app.FindRecordsByFilter("memories", "status = 'active'", "", 0, 0, nil)
```

**The `/ui/show/{type}` dispatcher already exists** — `internal/web/show.go:31`
(`cards.Get(typ)`), so registering a `note` card type in `internal/cards/cards.go`
(plus a feature renderer) makes `/ui/show/note?id=...` work for free. The card
registry shape to copy is `internal/cards/cards.go:120-133` (the `memory` spec
with `Params []ParamSpec`).

### LOCKED schema this plan must implement (verbatim from the design)

`nodes` collection fields:
- `type` (select; values include `note`, `memory`, `skill`, `journal`,
  `person`, `book`, `idea`, `place` — extensible; required)
- `title` (text, required, Max 300)
- `body` (text, markdown, Max 100000)
- `status` (select: `proposed|active|archived|rejected`, required)
- `props` (json — type-specific fields)
- `created` / `updated` (autodate)

`edges` collection fields:
- `source` (relation → `nodes`, single, `CascadeDelete: true`)
- `target` (relation → `nodes`, single, `CascadeDelete: true`)
- `type` (text, default `"links"`)
- `context` (text, optional)
- Index: **unique** `(source, target, type)`; non-unique index on `(target)`
  for backlinks.

**Relation field names are `source` and `target` — these exact names. Plans
161 and 163 use back-relation expand `edges_via_target` (backlinks) and
`edges_via_source` (outbound). Do NOT rename them.**

**TRUST / CONSENT (preserve exactly):** `note` + typed-objects + `journal` are
born `status=active` (owner-authored, trusted). `memory` + `skill` are born
`status=proposed` (agent-proposed, consent-gated) and become `active` on owner
approval. **Graph traversal AND search MUST filter to `status=active`** — never
surface proposed/rejected nodes.

### Repo conventions to honor (the executor has not read AGENTS.md)

- **Domain packages own their own PocketBase reads/writes** — `internal/nodes`,
  `internal/knowledge`, `internal/life` talk to PocketBase directly; do NOT
  route new domain logic through `internal/store` (store is cross-cutting only:
  audit/settings/LLM/time).
- **Audit STRICTLY AFTER a successful write**, never before: `store.Audit(app,
  actor, action, target, allowed, detail)` — see
  `internal/knowledge/knowledge.go:82-83`.
- **Errors are values**: wrap with `fmt.Errorf("doing x: %w", err)`, return
  early, no panics in library code.
- **Structured logging**: `app.Logger()` (slog) with key/value pairs.
- **gomponents**: alias `h "maragu.dev/gomponents/html"` (journalcards) OR the
  `hh`/dot-import convention already in the file you edit — match the file.
  User/model text renders through escaping `g.Text`; `g.Raw` only for
  already-rendered trusted HTML.
- **Storybook is the UI source of truth** — any new/changed card adds or updates
  its story in `internal/feature/storybook/` in the same change.
- **Self-knowledge** (`internal/self/knowledge.md`) must be updated in the same
  change when architecture/capabilities change.
- **`pb_data/` is disposable** here. After the migration changes, you MUST drop
  the dev DB before booting (`rm -rf pb_data`) — PocketBase records the baseline
  as applied on first boot and will NOT re-run it.

## Commands you will need

| Purpose | Command | Expected on success |
|---|---|---|
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test (all) | `go test ./...` | all pass |
| Test (env-scrubbed) | `env -u BALAUR_OS_ACCESS -u BALAUR_SOURCE -u BALAUR_MAX_STEPS go test ./...` | all pass |
| Format | `gofmt -l .` | no output |
| Diff hygiene | `git diff --check` | no output |
| Lint (workflow-touching) | `make lint` | clean (staticcheck + govulncheck + gofmt/vet) |
| Fresh DB + seed smoke | `rm -rf pb_data && go run . seed` (or `make run` then `balaur seed`) | seeds nodes, no error |
| UI verify | `/verify` (run-balaur) → drive http://127.0.0.1:8090/ | memory/skill/journal/note cards render from nodes |
| Graph refresh | `graphify update .` then minify `graphify-out/graph.json` before commit | graph current |

## Suggested executor toolkit

- `go-standards` skill — apply Balaur Go idioms (errors `%w`, slog,
  records-as-domain-model, audit-after-save, table-driven tests, fake
  `llm.Client`).
- `ui-development` skill — for the card + composer + storybook work in Step 8.
- `pocketbase-api` skill — to verify rows landed after `go run . seed`.
- `run-balaur` skill — to drive the browser verify.
- Reference: `internal/storetest/storetest.go` (temp-dir app for domain tests),
  `internal/feature/journalcards/day.go` (card renderer + `ui.RegisterCard`),
  `internal/cards/cards.go:120-144` (card spec registration).

## Scope

**In scope** (the only files you should modify or create):

- `migrations/1749600000_init.go` — add `nodes`+`edges`, remove
  `memories`+`skills`, keep `entries` (journal role removed).
- `migrations/schema_test.go` — rewrite `TestSchemaBaseline` to the new schema.
- `internal/nodes/nodes.go` (create) — the node domain package: CRUD by
  type+status; edge helpers (`AddEdge`, `Backlinks`, `Outbound`); 1-hop
  neighborhood read; audit-after-write.
- `internal/nodes/nodes_test.go` (create) — domain + edge tests.
- `internal/knowledge/knowledge.go` — port onto `nodes` (type=memory/skill).
- `internal/knowledge/knowledge_test.go` — update for the node-backed paths.
- `internal/life/journal.go` — `JournalWrite` upserts a `type=journal` node.
- `internal/life/journal_test.go` — update.
- `internal/life/day.go` — read journal from `type=journal` nodes.
- `internal/life/day_test.go` — update if it asserts the journal read.
- `internal/search/index.go` — repoint the rebuild collection name to `nodes`
  (collection + status filter only; FTS *table schema* stays as-is for plan 162).
- `main.go` — `registerSearchIndex`: bind hooks to `nodes`, not `memories`.
- `internal/tools/knowledge.go` — repoint remember/recall/skill/propose_skill;
  add `node_write`/`node_list`/`node_get`/`node_drop`.
- `internal/tools/journal.go` — repoint `journal_write` (still via `life`).
- `internal/tools/knowledge_test.go` — update.
- `internal/seed/seed.go` — seed nodes instead of memories/skills; seed a note.
- `internal/seed/seed_test.go` — update counts/markers.
- `internal/self/knowledge.md` — rewrite data-model + capabilities sections.
- `internal/self/self.go`, `internal/self/tool.go` — if they enumerate
  `memories`/`skills` collections in the capability inventory, update the list.
- `internal/cli/knowledge.go` — repoint memory/skill onto nodes; add `balaur note`.
- `internal/cli/life.go` — `journal write` / `day` read journal from nodes.
- `internal/cli/doctor.go` — if it counts `memories`/`skills` collections,
  repoint to `nodes` (verify with grep; only touch the collection-name strings).
- `internal/web/knowledge.go` — repoint card/grid/transition/edit to nodes.
- `internal/web/cards.go` — `proposalBody` loads `kind` from `nodes`.
- `internal/web/home.go` — the Knowledge sidebar / memory-category sub-items if
  they reference `memories`/`skills` collection strings (verify with grep).
- `internal/feature/knowledgecards/*.go` (memory.go, skills.go, register.go,
  knowledgefocus.go) — read type-filtered active nodes; build the note card.
- `internal/feature/journalcards/day.go` — journal counts from `type=journal`.
- `internal/cards/cards.go` — add a `note` card spec (`id` param) so
  `/ui/show/note?id=...` resolves; keep `memory`/`skills`/`day` specs.
- `internal/feature/storybook/stories_cards.go`,
  `internal/feature/storybook/stories_navigation.go` — add/update note +
  memory/skill stories from node fixtures.
- `internal/feature/knowledgecards/knowledgefocus_test.go`,
  `internal/web/knowledge_test.go`, `internal/web/knowledge_gomponents_test.go`,
  `internal/cards/cards_test.go`, `internal/web/cards_test.go`,
  `internal/tools/ui_test.go`, `internal/tools/choices_test.go`,
  `internal/cli/cli_test.go`, `internal/heads/heads_test.go`,
  `internal/web/heads_test.go`, `internal/web/journal_test.go`,
  `internal/search/index_test.go`, `internal/search/fts5_test.go` — update ONLY
  the assertions/fixtures that reference the removed collections or the
  journal-in-entries path. (Use the build/test failures as the worklist; do not
  pre-emptively rewrite tests that still pass.)

**Out of scope** (do NOT touch — separate plan owns it, or it is deferred):

- `internal/search/index.go` FTS *table schema* redesign (the unified
  `knowledge_fts(id, kind, title, content, extra)` table) — that is **plan 162**.
  Here you ONLY change the collection name the rebuild/hooks read (`memories` →
  `nodes`) and the active-status filter so the binary boots green. Do NOT add
  `kind`, do NOT rename the table, do NOT change `Upsert`/`Query` columns beyond
  what compiles against `nodes`. If repointing the existing `memories_fts`
  columns (`title, content, when_to_use, category`) to node fields is more than
  a field-name swap, leave `when_to_use`/`category` reading from `props` via a
  small accessor and STOP-think before expanding. **If this turns into a search
  redesign, STOP — that is 162's job.**
- `[[wikilinks]]` parsing + edges-on-save hook — **plan 161**.
- Graph traversal / recursive CTE / "related nodes" view — **plan 163**.
- `tasks` as nodes, note↔task cross-layer edges, life-log-measures as nodes,
  block refs `((…))`, per-type Capacities schemas, LogSeq queries — **DEFERRED**
  (name them in Maintenance notes; build none).
- `tasks`, `summaries`, `conversations`, `messages`, `heads`, `llm_*`,
  `owner_settings`, `extensions`, `audit_log` collections — **unchanged** in the
  migration. `entries` STAYS (life-log measures/lines); only its journal role
  moves out.
- A rich text editor / WYSIWYG for the note composer — use a `<textarea>` that
  `@post`s, reusing `ui.Composer` (NOT a rich editor).

**Scope traps:**
- `entries` is KEPT — do not delete it. Only `kind='journal'` rows move to nodes;
  measures/lines/completions stay.
- `proposalBody` in `internal/web/cards.go:216` has a comment "collection name
  == kind (`memories`/`skills`)" — after the port, `kind` is `nodes` and the
  record carries `type=memory|skill`. Update both the lookup AND the comment.
- `internal/cli/knowledge.go:36` builds `enabled` from
  `status == StatusActive` — preserve that derivation; skills no longer have an
  `enabled` field (they never did post-155).

## Git workflow

- Branch: `improve/160-nodes-edges-spine` (executor worktrees base off
  `origin/main`).
- Commit per logical phase (migration+test; nodes package; knowledge port;
  journal port; search/hooks; tools; cards/web; cli; seed; self/storybook), or
  one commit if you land it atomically. Conventional-commit subjects, e.g.
  `feat(nodes): consolidated nodes+edges baseline; fold memories/skills/journal`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

> Order rationale: the migration + `nodes` package land first (new path exists),
> then domain ports switch callers onto it, then the old collections are removed
> in the migration's final form, then UI/CLI/self/seed follow. Between the
> migration rewrite (Step 1) and the last caller port the tree may not compile —
> that is unavoidable for an atomic schema change; minimize the window by doing
> Steps 1-7 before running the full build.

### Step 1: Rewrite the baseline migration — add `nodes`+`edges`, remove `memories`+`skills`

In `migrations/1749600000_init.go`:

1. Edit `collectionNames` (line 26-30): replace `"memories", "skills"` with
   `"nodes", "edges"`. Order: `edges` references `nodes`, so list `nodes` BEFORE
   `edges` (relations point left; dropped in reverse). Suggested:
   `"heads", "conversations", "messages", "nodes", "edges", "audit_log", "summaries", "tasks", "entries", "extensions", "llm_providers", "llm_models", "llm_settings", "owner_settings"`.
2. Delete the `memories := ...` block (lines 86-105) and the `skills := ...`
   block (lines 107-124).
3. In their place, add the `nodes` collection then the `edges` collection
   (target shape — match the field/index API of the existing blocks):

```go
	// nodes: the unified knowledge spine. type decides the kind (note, memory,
	// skill, journal, person, book, idea, place, …); props holds type-specific
	// fields. Consent lives in status: note/journal/typed-objects are born
	// active (owner-authored); memory/skill are born proposed (agent-proposed)
	// and become active only on the owner's approval. Traversal and search
	// filter to status=active so proposals never leave the box.
	nodes := core.NewBaseCollection("nodes")
	setOwnerRules(nodes, owner)
	nodes.Fields.Add(
		&core.SelectField{Name: "type", Required: true, MaxSelect: 1, Values: []string{
			"note", "memory", "skill", "journal", "person", "book", "idea", "place",
		}},
		&core.TextField{Name: "title", Required: true, Max: 300},
		&core.TextField{Name: "body", Max: 100000},
		&core.SelectField{Name: "status", Required: true, MaxSelect: 1, Values: []string{"proposed", "active", "archived", "rejected"}},
		&core.JSONField{Name: "props"},
		&core.AutodateField{Name: "created", OnCreate: true},
		&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
	)
	nodes.AddIndex("idx_nodes_type_status", false, "type, status", "")
	nodes.AddIndex("idx_nodes_status", false, "status", "")
	if err := app.Save(nodes); err != nil {
		return err
	}

	// edges: node↔node links. source/target cascade-delete with their nodes
	// (PocketBase auto-cleans an edge when either endpoint is removed).
	// Back-relation expand gives backlinks for free: ?expand=edges_via_target
	// (inbound) and ?expand=edges_via_source (outbound). The unique index
	// (source,target,type) makes [[link]] edge sync idempotent (plan 161).
	edges := core.NewBaseCollection("edges")
	setOwnerRules(edges, owner)
	edges.Fields.Add(
		&core.RelationField{Name: "source", Required: true, CollectionId: nodes.Id, CascadeDelete: true, MaxSelect: 1},
		&core.RelationField{Name: "target", Required: true, CollectionId: nodes.Id, CascadeDelete: true, MaxSelect: 1},
		&core.TextField{Name: "type", Max: 60},
		&core.TextField{Name: "context", Max: 2000},
		&core.AutodateField{Name: "created", OnCreate: true},
	)
	edges.AddIndex("idx_edges_unique", true, "source, target, type", "")
	edges.AddIndex("idx_edges_target", false, "target", "")
	if err := app.Save(edges); err != nil {
		return err
	}
```

> Note on `edges.type` default `"links"`: PocketBase `TextField` has no schema
> default; set the default in the Go write path (`internal/nodes` `AddEdge`
> defaults `type` to `"links"` when empty), not in the migration. The LOCKED
> spec's `default "links"` is a write-side default.

4. `entries` block (lines 187-203) stays exactly as-is — journal rows simply
   stop being written there. No migration change for `entries`.

**Verify**:
`grep -n '"nodes"\|"edges"\|core.NewBaseCollection("nodes")\|core.NewBaseCollection("edges")' migrations/1749600000_init.go`
→ shows nodes+edges present; and
`grep -n 'NewBaseCollection("memories")\|NewBaseCollection("skills")' migrations/1749600000_init.go`
→ **no output** (both removed).

### Step 2: Rewrite `schema_test.go` for the new collection/field/index set

In `migrations/schema_test.go`, `TestSchemaBaseline`:

1. The collection list (lines 23-27): replace `"memories", "skills"` with
   `"nodes", "edges"`.
2. The "retired collections never created" check (lines 34-38): the list is
   currently `{"boards", "grants"}`. APPEND `"memories"` and `"skills"` to it —
   do NOT replace `boards`/`grants` (they must stay retired). Result:
   `{"boards", "grants", "memories", "skills"}`.
3. The `fieldCheck` table (lines 52-60): remove the `memories`/`skills` rows; add
   `{"nodes", []string{"type", "title", "body", "status", "props"}, []string{"content", "category", "name"}}`
   and `{"edges", []string{"source", "target", "type", "context"}, nil}`.
4. The index list (lines 85-95): remove `idx_memories_status`,
   `idx_memories_status_importance`, `idx_skills_name`, `idx_skills_status`; add
   `idx_nodes_type_status`, `idx_nodes_status`, `idx_edges_unique`,
   `idx_edges_target`.
5. Update the comment "All 14 app collections" if the count changes (it stays 14:
   −2 memories/skills, +2 nodes/edges).

**Verify**: `go test ./migrations/...` → `ok` (TestSchemaBaseline +
TestMigrationTimestampsAreUnique pass).

### Step 3: Create the `internal/nodes` domain package

Create `internal/nodes/nodes.go`. Records-as-domain-model; audit-after-write;
errors `%w`. Provide at least:

- Status constants (`StatusProposed/Active/Archived/Rejected`) and the same
  `validTransitions` map as `knowledge.go:118-124`. (Knowledge will re-use these
  via the nodes package or keep its own — your call, but ONE source of truth;
  prefer exporting from `nodes` and having `knowledge` reference them.)
- `Create(app, type, title, body, status string, props map[string]any) (*core.Record, error)`
  — writes a node, audits `node.create` after `app.Save`.
- `Get(app, id) (*core.Record, error)`, `Drop(app, id) error` (audited).
- `ListByTypeStatus(app, typ, status string) ([]*core.Record, error)` — newest
  first; the building block for active-only reads.
- `Transition(app, id, to string) (*core.Record, error)` — validates the
  lifecycle, audits.
- Edge helpers:
  - `AddEdge(app, sourceID, targetID, edgeType, context string) (*core.Record, error)`
    — defaults `edgeType` to `"links"` when empty; idempotent against the unique
    index (on a unique-constraint error, find-and-return the existing edge or
    treat as success — do NOT panic).
  - `Backlinks(app, id string) ([]*core.Record, error)` — inbound nodes via
    `?expand=edges_via_target` (use `app.ExpandRecord` or a filter on
    `edges.target = id` then load sources). Filter results to `status=active`.
  - `Outbound(app, id string) ([]*core.Record, error)` — via `edges_via_source`,
    filtered to `status=active`.
  - `Neighborhood(app, id string) ([]*core.Record, error)` — the 1-hop set
    (backlinks ∪ outbound), `status=active` only.

> The `status=active` filter on traversal is MANDATORY and load-bearing
> (consent): `Backlinks`/`Outbound`/`Neighborhood` must never return a proposed
> or rejected node.

**Verify**: `CGO_ENABLED=0 go build ./internal/nodes/...` → exit 0.

### Step 4: Port `internal/knowledge` onto nodes

> Stale comment to fix while porting: `knowledge.go:34` reads
> `// Statuses (mirrors migrations/1749700000_knowledge.go).` — that migration
> file no longer exists (consolidated into `1749600000_init.go` in plan 156), so
> the comment already lies. Since you are rewriting this file anyway, update it
> to reference the live baseline / the `nodes` status constants (e.g.
> `// Statuses for nodes (mirrors the nodes.status enum in migrations/1749600000_init.go).`).

In `internal/knowledge/knowledge.go`, rewrite the bodies (keep the exported
signatures stable where callers depend on them — `MemoryProposal`,
`SkillProposal`, `ProposeMemory`, `ProposeSkill`, `Transition`, `UpdateFields`,
`Touch`, `ListByStatus`, `FilterActive`, `SearchActive`, `UpfrontMemories`,
`ActiveSkills`, `LoadSkill`):

- `ProposeMemory`: create a `type=memory`, `status=proposed` node. Map fields →
  `props`: `category`, `importance` (clamped 1-5), `when_to_use`, `source`,
  `use_count`, `last_used`. `title` → node `title`; `content` → node `body`.
- `ProposeSkill`: create a `type=skill`, `status=proposed` node. `name` →
  `title`; `content` → `body`; `props`: `description`, `when_to_use`,
  `use_count`, `last_used`. (Skill name uniqueness was enforced by
  `idx_skills_name`; nodes have no per-type unique index — enforce uniqueness in
  the Go path if `LoadSkill` relies on it: `LoadSkill` should pick the first
  active `type=skill` node whose `title` matches.)
- `Kind` type (`Memory="memories"`, `Skill="skills"`): repurpose to the node
  TYPE (`Memory="memory"`, `Skill="skill"`) since callers pass it around (web
  `kindFromPath`, cli). Audit the every caller that does `string(kind)` as a
  collection name — they must now read the `nodes` collection and filter by
  `type`. Prefer: keep `Kind` as the node-type string and add a small
  `nodeType(Kind) string` helper; every read becomes
  `nodes.ListByTypeStatus(app, string(kind), status)`.
- `Transition`/`UpdateFields`: operate on the node row; whitelist writable props
  per kind (memory: title/body/category/importance/when_to_use; skill:
  title/body/description/when_to_use). Preserve the audit actions
  (`knowledge.propose`, `knowledge.<to>`, `knowledge.edit`).
- `SearchActive`/`UpfrontMemories`/`FilterActive`/`ActiveSkills`/`LoadSkill`:
  filter `type = 'memory'` (or `'skill'`) `&& status = 'active'`. The
  `category`/`importance`/`when_to_use` reads come from `props` (PocketBase
  `record.GetString` does NOT reach into json — read `props` via
  `record.Get("props")` → map, or a small accessor). The FTS fast path in
  `SearchActive` (lines 230-261) reads `app.FindRecordsByIds(string(Memory),
  ids)` — change the collection to `"nodes"` and keep the active filter.

**Verify**: `CGO_ENABLED=0 go build ./internal/knowledge/...` → exit 0 (after
Step 3's nodes package exists). `go test ./internal/knowledge/...` once Step 5+
land (the test file is updated in Step 9).

### Step 5: Port journal onto nodes

In `internal/life/journal.go`, `JournalWrite`: instead of an `entries`
`kind=journal` row, upsert a **`type=journal` node, one per day, born active,
verbatim**:

- Compute the day key from `notedAt` (owner location → `YYYY-MM-DD`).
- Find an existing active `type=journal` node for that day (store the day key in
  `props.date`, or use the node `title` = the day label). If found, append the
  new text to `body` (verbatim, separated by a blank line); else create one with
  `status=active`, `title` = the day label, `body` = the text,
  `props.date` = day key. Audit `journal.write` after save.
- `JournalDrop`: delete the journal node by id (audited), guarding it is
  `type=journal`.

In `internal/life/day.go`, `Day()`: replace the `entries` `kind='journal'` query
(lines 28-35) with a read of `type=journal` `status=active` nodes for the day
(filter on `props.date` = the day key, or `created`/`updated` in the window —
match whatever key `JournalWrite` writes). Keep `DayData.Journal []*core.Record`.

> Decide ONE journal-per-day key and use it in both `JournalWrite` and `Day()`.
> Recommended: `props.date` = `YYYY-MM-DD` in owner location; query with a
> `props ~ {:datekey}` LIKE filter or an exact `props.date` match (PocketBase
> filters can address json paths: `props.date = {:d}`).

**Verify**: `CGO_ENABLED=0 go build ./internal/life/...` → exit 0.

### Step 6: Repoint the search hooks + rebuild to `nodes` (NOT a redesign)

This step has THREE edits, not one. After the hooks bind to `nodes`, the
upsert/delete path fires for EVERY node type — `note`, `journal`, `person`, … —
not just memories, and `Upsert` reads node-absent fields. All three must change
together or the index breaks (empty memory content) and leaks (non-memory nodes
into `memories_fts`).

**(a) Hook bind** — `main.go` `registerSearchIndex` (lines 253-255): bind the
three hooks to `"nodes"` instead of `"memories"`.

**(b) Gate the upsert/delete path to `type=='memory'`** — once bound to `nodes`,
`upsertHook` (`main.go:232`) fires for every node type and calls `Upsert`
(`internal/search/index.go:94`). On a non-memory node the field reads below are
meaningless and would pollute `memories_fts`. Gate it: skip the index work when
`e.Record.GetString("type") != "memory"` — do this once, at the top of
`Upsert`/`Delete` in `index.go` (preferred — it keeps the hook closures untouched
and is the single source of truth), e.g. at the start of `Upsert`:
`if rec.GetString("type") != "memory" { return ix.Delete(rec.Id) }` (delete-then-skip
so a node that was a memory and changed type is removed). `Delete` itself is by id
only and is safe to leave ungated.

**(c) Repoint `Upsert`'s field reads to node shape** — `Upsert`
(`internal/search/index.go:94-114`) currently reads `rec.GetString("content")`,
`rec.GetString("when_to_use")`, `rec.GetString("category")`. On a node row
`content` is empty (the field is `body`) and `when_to_use`/`category` live inside
`props` json (PocketBase `GetString` does NOT reach into json). Change the reads:
`content` ← `rec.GetString("body")`; `when_to_use`/`category` ← read from `props`
via the SAME small accessor Step 4 / the Rebuild query use
(`rec.Get("props")` → map, `props["when_to_use"]`, `props["category"]`). `title`
stays `rec.GetString("title")`.

**(d) Rebuild** — `internal/search/index.go` `Rebuild` (line 61): read from
`"nodes"` with filter `type = 'memory' && status = 'active'` (memories are the
only node type the v1 FTS index covers — plan 162 unifies all types). The
`content`/`when_to_use`/`category` reads in the Rebuild loop (lines 78-80) change
the same way as (c): `content` ← `body`; `when_to_use`/`category` from `props`.

> This is still 162-safe: NO table or column rename, NO `kind` column. You are
> ONLY changing which records feed `memories_fts` (memory nodes only) and how the
> two prop columns + `body` are read off a node row. The `memories_fts` table and
> its five columns (`id, title, content, when_to_use, category`) stay intact —
> plan 162 owns the unified `knowledge_fts` redesign. If you find yourself
> renaming the table or adding a `kind` column, STOP — that is 162.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0 (the whole tree should now
compile once Steps 1-6 land; remaining failures point you to the caller files in
Steps 7-9). `grep -n 'OnRecordAfter.*("nodes")' main.go` → 3 matches.
`grep -n 'GetString("type") != "memory"\|GetString("body")' internal/search/index.go`
→ the gate + the `body` read are present.

### Step 7: Port the tools

In `internal/tools/knowledge.go`:
- `remember`/`recall`/`skill`/`propose_skill` keep their specs; they already call
  the `knowledge` package, so they work once Step 4 lands. The `MarkProposal`
  calls pass the kind string (`"memories"`/`"skills"`) — change to the node
  collection so `proposalBody` can load the record: pass `"nodes"` as the kind
  and the node id (and update `proposalBody` in Step 8 to read `nodes`).
- Add `node_write`, `node_list`, `node_get`, `node_drop` tools for
  owner-authored notes + typed objects (born `active`, owner-voiced, audited).
  `node_write` takes a `type` (default `note`), `title`, `body`, optional
  `props`. These call `internal/nodes`. Register them from `KnowledgeTools` (or
  a new `NodeTools`) and wire into the turn's tool set the same way
  `KnowledgeTools` is wired (find the call site:
  `grep -rn "KnowledgeTools\|JournalTools" internal/turn`).

In `internal/tools/journal.go`: `journal_write` still calls `life.JournalWrite`
— unchanged once Step 5 lands.

**Verify**: `CGO_ENABLED=0 go build ./internal/tools/...` → exit 0;
`grep -n 'node_write\|node_list\|node_get\|node_drop' internal/tools/knowledge.go`
→ 4 tool names present.

### Step 8: Port the web cards + add the `note` card + composer

- `internal/cards/cards.go`: add a `note` card spec. The `memory` spec
  (`cards.go:120-133`) has NO id-like single-record param (its params are
  `mode`/`category`/`view`/`query`/`limit`) — do NOT try to copy an `id` from
  it. Use these literal Params verbatim so no inference is needed (match the
  surrounding `ParamSpec` shape — `Name`, `Required`, `Doc`):
  ```go
  Params: []ParamSpec{
      {Name: "id", Required: true, Doc: "node id to show"},
      {Name: "type", Doc: "node type for typed-object render"},
  },
  ```
  Keep the existing `memory`, `skills`, `day` specs.
- `internal/feature/knowledgecards/*`: the memory/skills renderers read
  type-filtered active nodes (via the ported `knowledge` package — no direct
  collection strings). Add a `note` renderer (a `ui.RegisterCard("note", …)`)
  rendering the node's title + markdown body + a `ui.Composer` `<textarea>` that
  `@post`s to a node-write route for owner edits (reuse `ui.Composer`; NOT a
  rich editor). Register `note` from `Register` and unregister it.
- `internal/web/cards.go` `proposalBody` (lines 204-225): the knowledge branch
  loads `app.FindRecordById(kind, id)` where `kind` was `"memories"`/`"skills"`
  — change to read from `"nodes"` and render via the existing
  `renderCardHTML(knowledge.Kind, rec)` (the kind now derives from
  `rec.GetString("type")`). Update the comment on line 216.
- `internal/web/knowledge.go` `kindFromPath` (lines 57-65): map path values
  `memories`→`knowledge.Memory`, `skills`→`knowledge.Skill` (unchanged path
  strings; the Kind now means node-type). The reads inside
  `knowledgeGrid`/`knowledgeCard`/`knowledgeTransition`/`knowledgeEdit` go
  through the ported `knowledge` package.
- `internal/feature/journalcards/day.go` `buildDay` (lines 39-78): journal count
  comes from `life.Day(...).Journal` (already updated in Step 5) — no change if
  it reads `DayView.JournalN` from `len(dd.Journal)`. Confirm.
- Storybook: add a `note` story and refresh the memory/skill stories from node
  fixtures in `internal/feature/storybook/stories_cards.go`; update the Knowledge
  navigation story in `stories_navigation.go` if it lists collection types.
- `/`-command palette / sidebar: add a "note" palette item alongside memory/skill
  (wherever the palette items live — `grep -rn "card palette\|cardPaletteNode\|palette" internal/web internal/feature`).

**Verify**: `CGO_ENABLED=0 go build ./internal/web/... ./internal/feature/...` →
exit 0; `grep -n '"note"' internal/cards/cards.go` → the note spec present.
Then `/verify` (run-balaur): `/ui/show/note?id=<a seeded note id>` renders the
note; `/ui/show/memory` and `/ui/show/skills` still render from nodes.

### Step 9: Port CLI + seed + self-knowledge + remaining tests

- `internal/cli/knowledge.go`: memory/skill commands call the ported package;
  `skillJSON` `enabled` stays `status == StatusActive` (line 36). Add a
  `balaur note` command group (`note add --type --title --body`, `note list
  --type`, `note show <id>`, `note drop <id>`) over `internal/nodes`. Wire it
  into the root command next to `memoryCmd`/`skillCmd` (find the registration:
  `grep -n "memoryCmd\|skillCmd\|AddCommand" internal/cli`).
- `internal/cli/life.go`: `journal write` and `day` already go through `life`
  (Steps 5) — confirm the JSON shape for journal entries (now nodes) still
  serializes (`entryJSON` reads `text`; a journal node has `body`, not `text`).
  Add a journal-node JSON shaper or read `body`.
- `internal/seed/seed.go`: `seedMemories`/`seedSkills` call the ported knowledge
  package for the WRITES (work as-is once it writes nodes), but their
  IDEMPOTENCY GUARDS read collection strings directly and MUST be repointed —
  enumerate all four:
  - `seedMemories` guard (`seed.go:246`):
    `app.CountRecords("memories", dbx.HashExp{"source": Marker})`. The memory
    `source` marker now lives in `props.source`. **Do NOT port this to
    `CountRecords` with a `props.source` filter.** `CountRecords` takes a raw
    `dbx.Expression` and BYPASSES PocketBase's record-field resolver
    (`record_query.go` ~L461), so a `props.source` term is emitted as a literal
    column and the query errors — only the filter-STRING APIs
    (`FindRecordsByFilter` / `FindFirstRecordByFilter`) resolve a json path to
    `JSON_EXTRACT`. Repoint to the resolver-backed `FindFirstRecordByFilter`
    form — the SAME shape `seedSkills` already uses below — and treat a found
    record as "already seeded":
    `_, err := app.FindFirstRecordByFilter("nodes", "type = {:t} && props.source = {:m}", dbx.Params{"t": "memory", "m": Marker})`
    — `err == nil` → a marker node exists, skip; `errors.Is(err, sql.ErrNoRows)`
    → seed. (This is how Step 5 resolves `props.date` too: via a filter string,
    never via `CountRecords`.)
  - `seedSkills` guard (`seed.go:292`):
    `app.FindFirstRecordByFilter("skills", "name = {:n}", dbx.Params{"n": s.p.Name})`.
    A skill's `name` is now the node `title`. Repoint to:
    `app.FindFirstRecordByFilter("nodes", "type = {:t} && title = {:n}", dbx.Params{"t": "skill", "n": s.p.Name})`.
  - `Reset`'s memory delete: `del("memories", "source = …")` must become a
    `nodes` filter on the same json path:
    `del("nodes", "type = 'memory' && props.source = {:m}")` (or whatever
    `del` helper signature the file uses — match it; the load-bearing change is
    `memories`→`nodes` + `source`→`props.source`).
  - `Reset`'s skill delete (`seed.go:136`): the Reset routine ALSO deletes the
    seeded skills (the 4th collection-string read). Repoint it the SAME way —
    `del("skills", …)` → `del("nodes", "type = 'skill' && …")` — porting its
    filter with the skill `name`→node `title` mapping used in the `seedSkills`
    guard above. Run `grep -n '"skills"' internal/seed/seed.go` to find the exact
    call; if its filter shape is not a simple name/marker match, STOP and report
    rather than guessing.
  Then add a `seedNotes` that creates a couple of `type=note` nodes (born active)
  and a `type=journal` node, so the note/day cards have data. Update
  `internal/seed/seed_test.go` counts.

  > Pin: `props.source = {:m}` and `props.date = {:d}` are PocketBase json-path
  > filters — the SAME mechanism Step 5 relies on for the journal-per-day key.
  > If `props.source = {:m}` does NOT match a seeded memory node in a quick smoke
  > (`rm -rf pb_data && go run . seed` twice — second run must seed 0 new), STOP
  > and report rather than guessing a `props ~ '"source":...'` LIKE form.
- `internal/self/knowledge.md` (lines 88-90, 102-104, 109-111): rewrite the
  data-model collections list to `nodes, edges, …` (drop `memories, skills`);
  rewrite the Knowledge capability bullet to the consent-on-status node model;
  add the `node_write`/`node_list`/`node_get`/`node_drop` tools and the note
  card. Note journal is now a `type=journal` node.
- `internal/self/self.go` / `internal/self/tool.go`: if they hardcode the
  collection list for the capability inventory, update it (grep first).
- Remaining tests: run the full suite; fix ONLY the assertions/fixtures that
  reference removed collections or the journal-in-entries path. Do not rewrite
  passing tests.

**Verify**: `go test ./...` → all pass; `grep -rn '"memories"\|"skills"'
--include=*.go internal main.go migrations` → only appears in the
"retired collections must not exist" check in `schema_test.go` and in
path-string mappings (`kindFromPath` path values `memories`/`skills` are URL
paths, not collection names — those are allowed; verify each remaining hit is a
URL path or a retired-check, not a live collection read).

### Step 10: Refresh the graph + final gates

Run `graphify update .` (AST-only), then minify `graphify-out/graph.json` before
staging (the graph is committed minified — see the repo memory note). Run all
gates.

**Verify**: see Done criteria.

## Test plan

New / updated tests (model after `internal/storetest.NewApp(t)`
(`internal/storetest/storetest.go`, package `storetest`, `func NewApp` at
`storetest.go:18`) for temp-dir app instances; table-driven; fake `llm.Client`
for tool tests — never a real model):

- `internal/nodes/nodes_test.go` (create): `Create` writes the row + audits;
  `Transition` enforces `validTransitions` (proposed→active ok, proposed→archived
  rejected); `AddEdge` is idempotent against the unique index; **`Backlinks`/
  `Outbound`/`Neighborhood` return only `status=active` nodes (the consent
  filter) — add a proposed node and assert it is excluded**; cascade-delete: drop
  a node, assert its edges are gone (`app.CountRecords("edges", …) == 0`).
- `internal/knowledge/knowledge_test.go` (update): consent flow on nodes —
  `ProposeMemory` creates a `type=memory status=proposed` node;
  `SearchActive`/`UpfrontMemories` never return it; after `Transition` to active
  it is recalled. Five categories survive in `props.category`. Skill propose →
  approve → `LoadSkill` returns it; a proposed skill is not loadable.
- `internal/search/index_test.go` (update): the FTS index gate from Step 6 —
  upserting an active `type=note` (or `type=journal`) node does NOT land in
  `memories_fts` (assert a query for its title returns 0 hits); an active
  `type=memory` node DOES index and is found, with its `body`/`props.when_to_use`/
  `props.category` (not empty `content`) searchable. Reuse the existing
  `openTestIndex`/`TestRebuildAndQuery` shape (`index_test.go:13`, `:84`).
- `internal/life/journal_test.go` (update): `JournalWrite` creates ONE
  `type=journal` node for a day, born `active`, verbatim; a second write the same
  day appends to the same node; `Day()` returns it in `Journal`.
- `migrations/schema_test.go` (rewrite): the new collection/field/index
  assertions (Step 2); `memories`/`skills` are in the retired list.
- `internal/tools/knowledge_test.go` (update): `remember` proposes a node,
  `recall` finds an active node, `node_write` creates a `type=note status=active`
  node and audits.
- Card render tests (`internal/web/knowledge_test.go`,
  `internal/feature/knowledgecards/knowledgefocus_test.go`,
  `internal/cards/cards_test.go`): memory/skill/note cards render from node
  fixtures; the note card renders title + body + composer.
- CLI tests (`internal/cli/cli_test.go`): `balaur note add/list/show` and
  `balaur memory propose/approve` round-trip on nodes.

Existing tests that must pass UNCHANGED (behavior contracts not in scope):
`internal/tasks/*`, `internal/conversation/*`, `internal/recap/*`,
`internal/heads/*` (except where they reference seed counts/journal),
`internal/store/*`.

Verification: `go test ./...` → all pass, including the new
`internal/nodes` package tests.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0 (and `env -u BALAUR_OS_ACCESS -u BALAUR_SOURCE -u BALAUR_MAX_STEPS go test ./...` exits 0)
- [ ] `gofmt -l .` prints nothing
- [ ] `git diff --check` prints nothing
- [ ] `make lint` is clean (staticcheck + govulncheck + gofmt/vet)
- [ ] `grep -n 'NewBaseCollection("nodes")\|NewBaseCollection("edges")' migrations/1749600000_init.go` → both present
- [ ] `grep -n 'NewBaseCollection("memories")\|NewBaseCollection("skills")' migrations/1749600000_init.go` → no output
- [ ] `grep -rn 'OnRecordAfter.*("memories")' main.go` → no output; `grep -n 'OnRecordAfter.*("nodes")' main.go` → 3 matches
- [ ] `internal/search/index.go` `Upsert` gates non-memory nodes out (only `type='memory'` reaches `memories_fts`) and reads `body`/`props` (not `content`/top-level `when_to_use`/`category`) — `grep -n 'GetString("type") != "memory"\|GetString("body")' internal/search/index.go` → both present
- [ ] `internal/nodes/nodes.go` exists and `go test ./internal/nodes/...` passes, including a test asserting `Backlinks`/`Neighborhood` exclude a proposed node
- [ ] `grep -n '"note"' internal/cards/cards.go` → the note card spec present (so `/ui/show/note?id=...` resolves via the existing dispatcher)
- [ ] `rm -rf pb_data && go run . seed` exits 0 and seeds nodes (verify with `balaur memory list` / `balaur note list` or the pocketbase-api skill)
- [ ] `/verify` (run-balaur): memory, skills, day, and the new note card all render from nodes at http://127.0.0.1:8090/
- [ ] `internal/self/knowledge.md` data-model section lists `nodes, edges` and not `memories, skills`
- [ ] `graphify-out/graph.json` refreshed (minified) via `graphify update .`
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row for 160 updated

## STOP conditions

Stop and report back (do not improvise) if:

- The code at the "Current state" locations does not match the excerpts (the
  tree drifted since `72fd762`).
- **Removing `memories`/`skills` cascades into MORE than the enumerated
  port-surface touch-points** — an unexpected package, or the total touched file
  count exceeds ~25. Report the surface before proceeding.
- **The consolidated baseline cannot be safely rewritten in place** — e.g.
  another migration file references `memories`/`skills`, or PocketBase rejects
  the in-place edit because the baseline is already recorded as applied against a
  non-disposable DB. (`pb_data/` should be disposable here — if it is not, STOP.)
- Repointing the search index to `nodes` requires changing the FTS *table
  schema* (renaming the table, adding a `kind` column) — that is **plan 162's**
  job; STOP and report rather than doing 162's work here.
- A relation field named `source`/`target` cannot be created with
  `CascadeDelete: true` against `nodes` (PocketBase constraint) — STOP and report
  the exact field names PocketBase forces, so plans 161/163 can follow the same
  names.
- A step's verification fails twice after a reasonable fix attempt.
- The journal-per-day upsert key choice (Step 5) conflicts with how `Day()` or a
  recap reads journal — STOP and report rather than scattering two different day
  keys.

## Maintenance notes

For the human/agent who owns this after it lands:

- **This plan is at the edge of one-executor size.** It keeps the schema ATOMIC
  (the migration + all callers in one plan, as required — a half-ported tree does
  not build). If an executor cannot land it in one pass, the RECOMMENDED split is:
  (160a) migration + `schema_test.go` + `internal/nodes` package + the
  `internal/knowledge`/`internal/life` domain ports + search-hook repoint (the
  compile-critical core); then (160b) the UI cards/composer/storybook + CLI +
  seed + self-knowledge (the presentation surface). Both halves must still land
  before declaring 160 done — do NOT ship 160a alone (the binary's UI would read
  dead collections). Do not silently drop the UI/CLI/self scope.
- **Relation field names `source`/`target` and the back-relation expand names
  `edges_via_source`/`edges_via_target` are a contract** for plans 161 (wikilink
  edge sync, backlinks render) and 163 (traversal). If a PB constraint forced a
  rename, record the real names here.
- **The `status=active` filter on `Backlinks`/`Outbound`/`Neighborhood` is the
  consent spine** — a reviewer must confirm no traversal or search path can
  return a proposed/rejected node. This is the one place Balaur is stricter than
  LogSeq/Capacities, on purpose.
- **`edges.type` default `"links"` is a write-side default** (PB TextField has no
  schema default) — enforced in `nodes.AddEdge`, not the migration.
- **What a reviewer should scrutinize**: (1) every former `record.GetString`
  read of `category`/`importance`/`when_to_use`/`use_count` now reaches into
  `props` json correctly; (2) `proposalBody` and `kindFromPath` no longer treat
  the path `kind` as a collection name; (3) the seed idempotency guards
  (`seedMemories` find, `seedSkills` find, `Reset` memory delete, and `Reset`
  skill delete at `seed.go:136`) all read `nodes` on the `props.source`/`title`
  json paths, not the dropped collections; (4) the FTS rebuild reads `nodes` with
  `type='memory' && status='active'`, AND `Upsert`/the upsert hook gate
  non-memory nodes out of `memories_fts` and read `body`/`props` (not
  `content`/top-level `when_to_use`/`category`) — otherwise active notes/journal/
  person nodes leak into the memory index and memory content indexes empty.
- **Deliberately deferred to later plans** (named so nobody re-audits them as
  gaps): `[[wikilinks]]` + edges-on-save (161); unified `knowledge_fts` search
  over all node types (162); graph traversal + related-nodes view (163);
  tasks-as-nodes + note↔task cross-layer edges; life-log-measures-as-nodes;
  LogSeq block refs `((…))` and queries; Capacities per-type object
  schemas/templates; interactive/force-directed graph UI. Each is an ADDITION to
  the stable `nodes`/`edges` core, never a reshape — that is the design
  validation.
