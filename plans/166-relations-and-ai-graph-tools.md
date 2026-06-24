# Plan 166: First-class typed relations + AI link / traverse / query tools

> **Executor instructions**: Follow step by step; run every verification command
> and confirm its expected result before moving on. On a "STOP conditions" item,
> stop and report. When done, update this plan's status row in `plans/README.md`.
>
> **Drift check (run first)**:
> `git diff --stat 1c094a7..HEAD -- internal/nodes/ internal/tools/ internal/self/knowledge.md`
> Confirm plan 164 is DONE (needs the open `type` + registry). Plan 165 is
> helpful (property-aware querying) but not strictly required; if 165 is absent,
> implement the query verb on `title`/`type`/`status` only and note it.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: LOW (mostly additive — new tools + read paths; one small write verb)
- **Depends on**: `plans/164-object-type-registry.md` (165 recommended)
- **Category**: architecture / DX (AI capability)
- **Planned at**: commit `1c094a7`, 2026-06-24

## Why this matters

The graph exists (`edges`, backlinks, `[[wikilinks]]`) but the **model can't use
it as structure**. Today the AI's only graph affordances are full-text `search`
and implicit `[[wikilinks]]` buried in a node body — it cannot explicitly link
two objects, ask "what's related to X," or query "all books by author Y." That
is exactly the "easily leveraged by AI" gap. This plan exposes the existing
`internal/nodes` graph primitives (`AddEdge`, `Neighborhood`, `Backlinks`,
`Outbound`) as agent tools, adds a structured `node_query` verb (by type +
optional property filter), and gives edges a lightly-typed, owner-meaningful
relation vocabulary. After this, the model can build and walk the object graph
the way a Capacities/Anytype API client would.

## Current state

- `internal/nodes/nodes.go` already implements the graph primitives:
  - `AddEdge(app, sourceID, targetID, edgeType, context)` (line 163) —
    idempotent against the unique `(source,target,type)` index; defaults
    `edgeType` to `DefaultEdgeType = "links"` (line 36).
  - `Backlinks` (line 215), `Outbound` (line 228), `Neighborhood` (line 242) —
    all filter to `status=active` via `activeByIDs` (line 193). **The consent
    spine: proposed/rejected nodes never surface.**
  - `Get` (line 110), `ListByTypeStatus` (line 132).
- `internal/tools/knowledge.go` — the AI tool surface. `KnowledgeTools(app)`
  (line 21) returns the tool list. Existing node tools: `nodeWriteTool` (line
  ~255), `nodeListTool`, `nodeGetTool`, `nodeDropTool`. Helpers `obj(...)`,
  `str(...)` build JSON schemas; `agent.ToolSpecOf(name, desc, schema)` defines
  a tool; `agent.Tool{Spec, Execute}` is the shape. There is **no** link,
  related, or query tool today.
- `node_get` returns title/type/body but **not** props or links — the model
  can't see an object's structured fields or neighbours.
- Edges: `type` is free text (the `edges.type` field, migration line 112), with
  the unique index on `(source,target,type)`. There is no relation registry and
  no inverse-name concept; `[[wikilinks]]` all use type `"links"`.

### Conventions to match

- Tool definitions: copy the exact structure of `rememberTool`/`nodeWriteTool`
  in `internal/tools/knowledge.go` — `agent.ToolSpecOf`, `obj`/`str` helpers,
  `Execute: func(ctx, argsJSON) (string, error)` unmarshalling a local struct,
  returning a concise human/model-readable string. Match the terse,
  capability-honest tool descriptions already there.
- Tools are built per-request in `KnowledgeTools(app)`, so reading the registry
  inside a tool builder is fine.
- Audit owner-meaningful mutations after success (the `AddEdge` path already
  audits `edge.create` in `nodes.go:185`).
- Keep results small and deterministic. No new deps.

## Scope

**In scope**:
- `internal/nodes/nodes.go` — add `Edges(app, sourceID)` /
  `RelatedByQuery` read helpers IF needed (see steps); add a `Query` helper for
  type+prop filtering.
- `internal/nodes/relations.go` (create) — relation-type vocabulary + a small
  registry read (reuse 164's registry pattern, see Step 1).
- `internal/tools/graph.go` (create) — the new AI tools: `node_link`,
  `node_related`, `node_query`. (Putting them in a new file keeps
  `knowledge.go` from growing past the ~500-line smell threshold.)
- `internal/tools/graph_test.go` (create).
- `internal/tools/knowledge.go` — append the three new tools to the
  `KnowledgeTools` return slice; enrich `nodeGetTool` to show props + a link
  summary.
- `internal/self/knowledge.md` — document the new verbs.

**Out of scope**:
- Property schemas themselves (plan 165). This plan *reads* props for querying;
  it does not define schemas.
- Folding tasks/measures in (167/168).
- A graph UI — the read-only related view already exists (plan 163); this is the
  AI surface, not new visuals.
- Multi-hop / recursive traversal beyond 1 hop in `node_related` — keep it the
  1-hop `Neighborhood` for v1 (note deeper traversal as deferred). Plan 163's
  view owns deeper visualization.

## Relation vocabulary (keep it light)

Do NOT build a heavy relation-schema system. v1: a small curated set of
relation type names the tools accept (free-text still allowed, but these are the
suggested/validated set), with optional inverse labels for display:

| type | meaning | inverse (display) |
|---|---|---|
| `links` | generic association (the wikilink default) | linked from |
| `relates_to` | symmetric association | relates to |
| `part_of` | hierarchy / membership | has part |
| `about` | this is about that (e.g. note about a person) | referenced by |

Store these as a package-level slice in `internal/nodes/relations.go`
(`RelationTypes`), used to populate the `node_link` tool's `relation` enum and to
render inverse labels. `AddEdge` already accepts any string, so this is
guidance + UI nicety, not a hard constraint. Do **not** add a DB collection for
relations in v1 (note as deferred — promote to a registry like `node_types` only
if the owner wants to define custom relations).

**TWO defaults exist — make the split intentional, not accidental.** Wikilinks
write edges with type `links` (`nodes.DefaultEdgeType`, nodes.go:36); the
`node_link` tool defaults to `relates_to` (Step 3). That is fine, but it means
`links` and `relates_to` are two different "generic association" types in the
edge space. **Adopt this convention explicitly and document it in both the tool
description and `RelationTypes` comments:** `links` = wikilink-origin (the model
typed `[[X]]` in a body), `relates_to` = an explicit agent-asserted association.
Keeping them distinct is *useful* (you can tell how an edge arose and query on
it). Web research on LLM-friendly graph tools (the MCP knowledge-graph memory
server convention) favors **active-voice, subject-verb-object relation names** —
so prefer verb phrases the model emits naturally (`part_of`, `relates_to`,
`about`) and surface the `InverseLabel` so traversal reads as sentences. Do NOT
silently collapse `links` and `relates_to` into one; if you'd rather unify, that
is a deliberate decision to record here, not an executor improvisation.

## Steps

### Step 1: Add the relation vocabulary helper

Create `internal/nodes/relations.go` (package `nodes`):
- `var RelationTypes = []string{"links", "relates_to", "part_of", "about"}`
- `func InverseLabel(relType string) string` — returns the display inverse from
  the table above; falls back to "linked from" for unknown types.

**Verify**: `go build ./internal/nodes/` → exit 0.

### Step 2: Add a `Query` helper to `internal/nodes`

Add to `nodes.go` (or a `query.go` if you prefer; keep `nodes.go` under ~500
lines — it's currently 261):

```go
// QueryOpts narrows a node query. All fields optional.
type QueryOpts struct {
    Type      string            // exact type, "" = any
    PropMatch map[string]string // props key -> substring (AND across keys)
    Limit     int               // 0 = a sane default cap, e.g. 50
}

// Query returns active nodes matching opts, newest first. PropMatch is applied
// in Go over the active set (props is JSON; the listing is small). status is
// always active — the consent filter is non-negotiable.
func Query(app core.App, opts QueryOpts) ([]*core.Record, error)
```

Implementation: start from `ListByTypeStatus(app, opts.Type, StatusActive)` when
`Type != ""`, else `FindRecordsByFilter("nodes", "status='active'", "-updated,-created", 0, 0, nil)`;
then filter in Go by `PropMatch` using `nodes.PropString`; cap to `Limit`
(default 50). If 165 is not landed, `PropMatch` still works against whatever
keys exist in `props`.

**Verify**: `go test ./internal/nodes/ -run TestQuery -v` (write the test in the
Test plan) → pass.

### Step 3: Add the `node_link` tool

In `internal/tools/graph.go`, `GraphTools(app core.App) []agent.Tool` returning
the **four** tools (`node_link`, `node_related`, `node_query`, and
`node_schema` — Step 5b). `node_link`:
- Schema: `source` (id, required), `target` (id, required), `relation`
  (string enum from `nodes.RelationTypes`, default `relates_to`), `context`
  (optional free text — the "why" of the link, like Capacities' backlink
  context).
- Execute: resolve both ids exist via `nodes.Get` (return a clear error if
  either is missing or not active — links should not point at proposed/rejected
  nodes; check `status == "active"`). Then `nodes.AddEdge(app, source, target,
  relation, context)` (idempotent + audited already). Return e.g.
  `"Linked <source title> --relation--> <target title>."`.

> The model links by **id**, which it gets from `node_query`/`node_list`. Do not
> add title-based linking (ambiguous). Note this in the tool description.

### Step 4: Add the `node_related` tool

- Schema: `id` (required), `direction` (enum `["both","out","in"]`, default
  `both`).
- Execute: `nodes.Neighborhood` for `both`, `Outbound` for `out`, `Backlinks`
  for `in`. Render each neighbour as `- [<type>] <title> (id <id>)`. Empty →
  `"No related active nodes."`. (All paths already filter to active.)

### Step 5: Add the `node_query` tool

- Schema: `type` (optional string — suggest the registry types in the
  description), `match` (optional object: property-key → substring), `limit`
  (optional int).
- Execute: build `nodes.QueryOpts` and call `nodes.Query`. Render hits as
  `- [<type>] <title> (id <id>)`. Empty → `"No matching nodes."`.

### Step 5b: Add the `node_schema` introspection tool (HIGH value — do not skip)

This is the single highest-leverage AI affordance and the one the planned set was
missing. Both leading typed-object systems make the agent **read the schema
before writing** (Anytype exposes `list-types`/`list-properties`; Capacities
returns `propertyDefinitions` from `GET /space-info`). Without it the model
cannot know that a `book` wants `author`/`year`, so typed writes are guesswork.

- Tool name `node_schema` (or `list_types`). Schema: optional `type` (one type)
  — omitted returns all registered types.
- Execute: read the `node_types` registry (plan 164's collection; plan 165 added
  the `properties` column). Return, per type:
  `- <name> (<label>): props=[<key>:<type>[required], …]`. If plan 165 is not
  yet landed (no `properties` column), return just the type names/labels and note
  "property schemas pending plan 165".
- This closes the loop: after 167/168 fold tasks/measures in, the model can
  *discover* `task`/`measure` and their fields, not just that they exist.

> If plan 165 IS landed, the property list MUST come from the live
> `node_types.properties`, not a hardcoded map — the whole point is that the
> registry is the single source of truth.

### Step 6: Enrich `node_get` to show structure

In `internal/tools/knowledge.go`, `nodeGetTool` currently returns title/type/
body. Extend the output to also include:
- the node's `props` (as `key: value` lines, skipping empty) — so the model can
  read structured fields; and
- a one-line link summary using `nodes.Outbound`/`Backlinks` counts, e.g.
  `Links: 3 outbound, 2 backlinks`.

Keep it compact. Do not dump full neighbour bodies (that's `node_related`).

### Step 7: Register the new tools

In `KnowledgeTools(app)` (or wherever `ToolsForHead`/`Tools` assembles the verb
set — check `internal/turn/tools.go` so the tools actually reach the loop),
append `GraphTools(app)...`. Confirm the tools appear for the default head.

> Find where `KnowledgeTools` is consumed: `grep -rn "KnowledgeTools" internal/`.
> Add `GraphTools` alongside it in the same assembly so head tool-group filtering
> treats them as knowledge verbs. If heads filter by tool name groups, add the
> four new names (`node_link`, `node_related`, `node_query`, `node_schema`) to
> the same group as the existing `node_*` tools.

**Verify**: `go test ./internal/tools/ ./internal/turn/ -v` → pass.

### Step 8: Update `internal/self/knowledge.md`

Document the new capability under the knowledge/graph section: the model can now
`node_schema` to discover the registered object types and their property
schemas, `node_link` two objects with a typed relation + context, `node_related`
to walk 1-hop neighbours, and `node_query` to find objects by type and property —
all consent-filtered to active nodes.

**Verify**: `grep -n "node_link\|node_query\|node_related\|node_schema" internal/self/knowledge.md` → hits.

### Step 9: Full gate

**Verify**:
```
CGO_ENABLED=0 go build ./... && go vet ./... && go test ./... && gofmt -l internal/ && git diff --check
```
All exit 0; no `gofmt -l` output; `git diff --check` empty.

## Test plan

- `internal/nodes/` tests (use `storetest.NewApp(t)`; model after
  `internal/nodes/nodes_test.go` and `links_test.go`):
  - `Query` by type returns only that type, active only.
  - `Query` with `PropMatch` filters correctly; proposed nodes excluded.
  - `InverseLabel` returns expected strings incl. fallback.
- `internal/tools/graph_test.go` (model after `internal/tools/knowledge_test.go`):
  - `node_link` creates an edge (idempotent on second call — count stays 1);
    linking to a missing/proposed target errors.
  - `node_related` returns neighbours for both/out/in; empty case message.
  - `node_query` returns matching nodes; respects limit; never returns proposed.
  - `node_schema` lists registered types; after a type with a property schema
    exists (165), it reports that type's `props` (e.g. `book.author`).
- Verification: `go test ./...` → all pass including new tests.

## Done criteria

ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0
- [ ] `go test ./...` exits 0; new graph/query tests exist and pass
- [ ] The default head's tool set includes `node_link`, `node_related`,
      `node_query`, `node_schema` (a test or a `grep` of the assembled tool names
      confirms)
- [ ] `node_schema` returns property schemas from the live `node_types` registry
      (not a hardcoded map)
- [ ] `node_link` to a proposed/rejected/absent target returns an error (consent
      boundary holds)
- [ ] `node_get` output now includes props and a link-count summary
- [ ] `internal/tools/knowledge.go` stays under ~500 lines (new tools live in
      `graph.go`)
- [ ] `gofmt -l internal/` prints nothing
- [ ] `plans/README.md` status row for 166 updated

## STOP conditions

- Plan 164 not landed (open `type` / registry missing).
- `KnowledgeTools` is not where tools reach the loop and you can't find the
  assembly point (`internal/turn/tools.go`) — report the actual wiring rather
  than guessing.
- A verification fails twice after a reasonable fix.
- Drift-check mismatch with "Current state" excerpts.

## Maintenance notes

- **Deferred**: multi-hop traversal in `node_related` (v1 is 1-hop
  `Neighborhood`); a relation **registry** collection (v1 uses a curated slice);
  edge *properties* (edges carry only `context` today). Promote any of these
  only when a concrete need pulls it in.
- **Interaction**: plans 167/168 will create cross-type edges (task↔note,
  measure↔day). Those edges flow through `node_link`/`AddEdge` and appear in
  `node_related` automatically — no change needed here.
- **Reviewer should scrutinize**: that every new read/link path filters to
  `status=active` (the consent spine), and that `node_link` rejects non-active
  targets.
