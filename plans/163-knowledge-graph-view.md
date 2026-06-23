# Plan 163: Ship a "see the network" surface — related-nodes list + a server-rendered 1-hop SVG graph card

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
> **Drift check (run first)**:
> `git diff --stat 72fd762..HEAD -- internal/cards/cards.go internal/feature/graphcards internal/feature/all internal/feature/storybook internal/web/home.go internal/self/knowledge.md internal/nodes`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: LOW
- **Depends on**: `plans/160-*.md` (nodes/edges schema + node cards + the `GET /ui/show/{type}` dispatcher), `plans/161-*.md` (the `edges` rows + the `internal/nodes` Backlinks/Outbound helpers). Soft-benefits from `plans/162-*.md` (FTS-similar; degrades gracefully when absent).
- **Category**: direction
- **Planned at**: commit `72fd762`, 2026-06-23
- **Issue**: —

## Why this matters

After 160 (nodes + edges) and 161 (`[[wikilinks]]` → edges + backlinks), Balaur
*has* a knowledge graph but the owner cannot **see** it. The value of a
second-brain is the connective tissue — "what links to this?", "what does this
touch?", "show me the neighborhood". This plan ships the cheapest slice that
makes the network visible: (1) a **related-nodes list** — Backlinks ∪ Outbound
(∪ FTS-similar when 162's index exists) for a focus node, and (2) a **1-hop SVG
graph** — the focus node and its direct neighbors, server-rendered as a
concentric SVG (Datastar only, no JS framework, no Node build). Both are
**read-only over edges 161 already maintains** — no new write path, pure upside
on a stable base. Everything heavier (force-directed/interactive/zoomable
graphs, whole-vault rendering, clustering) is explicitly deferred.

## Current state

This plan **adds** a new feature-card package `internal/feature/graphcards`
and two card types (`related`, `graph`). It **reuses** the existing card
registry, the `GET /ui/show/{type}` dispatcher, the feature self-registration
machinery, the storybook, and the FTS recall helper — none of which this plan
modifies beyond the small registration additions listed in Scope. The files
below are the exemplars to copy; do not invent new patterns.

### Architecture constraints this plan MUST honor (inlined from the LOCKED design)

- **`status=active` is mandatory on graph + related.** Both the related list and
  the graph card MUST surface ONLY `status=active` nodes. Proposed/rejected
  nodes never enter the graph, the related list, or any context that leaves the
  box. (This is the consent spine; un-approved proposals stay out.) Reach nodes
  through the 161 helpers, which already filter to active — do not write a raw
  query that could leak proposed rows.
- **Edges are node↔node only.** The `edges` collection's relation fields are
  named **`source`** and **`target`** (declared by plan 160; back-relation
  expand is `edges_via_target` for backlinks, `edges_via_source` for outbound).
  Use these exact names via 161's helpers — never invent a `/ui/notes/{id}`
  route or a differently-named relation.
- **The node/show route is whatever plan 160 registered** — the generic
  `GET /ui/show/{type}?id=...` dispatcher (see `internal/web/show.go` below).
  161 and 163 use 160's real route; do not invent a new one.
- **This plan READS only.** It does NOT touch `internal/search` (read FTS only
  through the existing recall helper), does NOT touch the migration, and does
  NOT redeclare nodes/edges. 160 owns the schema; 161 owns the edge-sync hook;
  162 owns all `internal/search` changes.

### The contract this plan consumes from plans 160 & 161

Plan 161 introduces (or extends) package **`internal/nodes`** with helpers that
return the active neighbors of a node. This plan calls them; it does not
re-implement edge traversal. The expected signatures (confirm against the live
161 code during the Drift check — if 161 named them differently, adapt the call
sites and STOP-report the rename):

```go
// internal/nodes (provided by plan 161)
// Backlinks returns active nodes that link TO the node id (via edges_via_target).
func Backlinks(app core.App, id string) ([]*core.Record, error)
// Outbound returns active nodes the node id links FROM (via edges_via_source).
func Outbound(app core.App, id string) ([]*core.Record, error)
```

If 161 exposes a single combined helper (e.g. `Neighbors`) instead of the pair,
use it; the related-computation in Step 2 is written to need only "active
backlinks" and "active outbound" sets. **If neither shape exists when you start,
that is a STOP condition** (161 has not landed — this plan hard-depends on it).

A node record (collection `nodes`, declared by plan 160) has at least:
`id`, `type` (text/select), `title` (text), `status` (select; filter to
`active`). The graph/related cards read `title` and `type` for labels and
`id` for the focus subject + links.

### Exemplar 1 — the card registry (where `related`/`graph` specs are added)

`internal/cards/cards.go:25-33` (the `Spec` shape):

```go
// Spec is the static description of one card type.
type Spec struct {
	Type   string      // "today", "quests", …
	Label  string      // "Today"
	Icon   string      // icon file stem under /static/icons
	W      int         // default grid span (of 12)
	H      int         // default height in row units (row unit = 10px)
	Params []ParamSpec // accepted query parameters
}
```

`internal/cards/cards.go:120-133` (a spec WITH a required param — the shape to
copy for the new `id`-required specs):

```go
		{
			Type:  "memory",
			Label: "Memory",
			Icon:  "tome",
			W:     4,
			H:     20,
			Params: []ParamSpec{
				{Name: "mode", Enum: []string{"summary", "manage"}, Doc: "summary (read-only) or manage (approve/archive inline)"},
				{Name: "category", Enum: []string{"fact", "preference", "person", "project", "context"}, Doc: "show one memory category (the Knowledge sidebar sub-items)"},
				{Name: "view", Enum: []string{"active", "proposed"}, Doc: "active (default — the category listing) or proposed (the Awaiting approval queue)"},
				{Name: "query", Doc: "optional search terms to filter active memories"},
				{Name: "limit", Doc: "number of memories to show (default 6, max 50)"},
			},
		},
```

`ParamSpec` (`internal/cards/cards.go:17-23`): `{Name, Required bool, Enum []string, Doc string}`.
`Validate` (`internal/cards/cards.go:296-353`) drops unknown keys, errors on a
missing required param, errors on a bad enum value, and clamps `limit`/`days`
to `[1,50]`/`[1,366]`. A free-string param like `id` is capped at
`maxParamLen` (256). So a `required` `id` param + a `limit` param need no
custom validation — the registry handles it.

### Exemplar 2 — the ui card registry (how a feature renders a card)

`internal/ui/registry.go` (the registration seam — excerpt **condensed**:
`LookupCard` is shown as a one-liner but the live file spreads it across lines
with doc comments; used here only as an API-shape reference, not an edit target):

```go
type CardSize int
const (
	Tile CardSize = iota
	Focus
)
type CardFunc func(size CardSize, params map[string]string) (g.Node, error)
func RegisterCard(typ string, fn CardFunc) { cardRegistry[typ] = fn }
func UnregisterCard(typ string) { delete(cardRegistry, typ) }
func LookupCard(typ string) (CardFunc, bool) { fn, ok := cardRegistry[typ]; return fn, ok }
```

The web dispatch (`internal/web/cards.go:121-133`, `cardSizeInto`) looks up the
registered `CardFunc` and renders it — a feature renderer that ignores the
`size` argument renders the same node for both Tile and Focus. The new cards
will ignore `size` (they have one surface).

### Exemplar 3 — a feature-card package's Register/Unregister + init

`internal/feature/knowledgecards/register.go` (the whole file — copy this shape):

```go
package knowledgecards
// ... package doc ...
import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/alexradunet/balaur/internal/feature"
	"github.com/alexradunet/balaur/internal/ui"
)
func Register(app core.App) {
	registerMemory(app)
	registerSkills(app)
}
func Unregister() {
	ui.UnregisterCard("memory")
	ui.UnregisterCard("skills")
}
func init() {
	feature.Add(feature.Funcs(Register, Unregister))
}
```

And the per-card registration body (`internal/feature/knowledgecards/memory.go:300-310`
— `func registerMemory` at line 300, `ui.RegisterCard("memory", …)` at line 301;
the body below is **illustrative — confirm against the live `registerMemory`
during the drift check**, not byte-exact):

```go
func registerMemory(app core.App) {
	ui.RegisterCard("memory", func(size ui.CardSize, params map[string]string) (g.Node, error) {
		if size == ui.Focus {
			return KnowledgeFocus(buildMemoryFocus(app, params)), nil
		}
		if params["mode"] == "manage" {
			return MemoryManageCard(buildMemoryManage(app)), nil
		}
		return MemoryCard(buildMemorySummary(app, params)), nil
	})
}
```

The feature layering law (from `knowledgecards/register.go` doc comment, line
1-8): a feature package imports `internal/ui`, `internal/feature`, its domain
package (here `internal/nodes`), gomponents, and `pocketbase/core` only —
**never `internal/web`** (spec §4.1). `internal/feature/knowledgecards/knowledgecards_test.go`
is a compile-time `TestNoWebImports` marker; add the same marker test.

### Exemplar 4 — the show route the cards are served by (do NOT modify)

`internal/web/show.go:25-48` — `GET /ui/show/{type}` validates params via
`cards.Validate`, then morphs `#panel-inner`. Because `related` and `graph` are
just registered card types with a validated `id` param, the existing dispatcher
serves `/ui/show/related?id=...` and `/ui/show/graph?id=...` for free once the
specs are registered. **You do not touch show.go.** (160's node-show route also
flows through this same dispatcher.)

### Exemplar 5 — the command palette ("/"-command navigation)

`internal/web/home.go:23-38` — `commandPaletteNode()` lists the panel launchers.
Each item is `{Label, Key, Icon, URL: "/ui/show/<type>?..."}`. The related/graph
cards are **focus-node-specific** (they need an `id`), so they do NOT get a
static palette item — the owner reaches them from a node's own card (a
"Related →" / "Graph →" footer link 160's node card adds, OR a link this plan
adds to the related/graph cards cross-referencing each other). Adding a
parameterless palette item that needs an `id` would 400. **Do not add a palette
item for related/graph.** (Verified: every existing palette URL is fully
parameterized or parameterless — none needs a runtime id.)

### Exemplar 6 — SVG rendered in gomponents

`internal/ui/sparkline.go:26-77` — the canonical way Balaur emits SVG from Go:
build `svgAttrs := []g.Node{...}` with `g.Attr(...)`, append child elements via
`g.El("path", g.Attr("d", ...), ...)` / `g.El("rect", ...)`, and return
`g.El("svg", svgAttrs...)`. Layout math (scaling points into the viewBox) is
plain Go (`math.Min/Max`, `strconv`). The graph card's concentric layout is the
same idea: compute x/y for each neighbor in Go, emit `<line>` + `<circle>` +
`<text>`. **No JS.** Numeric coordinates only — never interpolate node titles
into a path/attribute; titles go in `<text>` via `g.Text` (escaped). Use
`g.El("title", g.Text(label))` for hover tooltips, never an unescaped attribute.

### Exemplar 7 — the FTS recall helper (162, optional, degrades gracefully)

`internal/knowledge/knowledge.go:229-285` — `SearchActive(app, terms, limit)`
(signature `func SearchActive(app core.App, terms []string, limit int) ([]*core.Record, error)`
at line 229) takes the FTS5 fast path when `app.Store().GetOk(search.StoreKey)`
returns a live `*search.Index`, else falls through to a LIKE query, else returns
`(nil, nil)`. This is the EXISTING recall helper, but **today it searches the
`memories` collection, not `nodes`** — plan 162 re-points the node-search entry
point at `nodes`. So call whatever node-search helper 162 exposes, NOT
`SearchActive` directly. For "FTS-similar related
nodes", call the equivalent **node** search helper that 162 exposes (a unified
`knowledge_fts` over nodes; 162 owns it). **Do NOT import `internal/search`
directly and do NOT widen the FTS table** — read similar nodes only through the
helper 162 provides (or the existing `knowledge.SearchActive`-style entry point
that 162 re-points at nodes). If that helper is absent (162 not landed), the
related list MUST still render from Backlinks ∪ Outbound alone — skip the
FTS-similar section entirely, no error. This is the "degrades gracefully"
requirement; Step 2 makes it explicit.

### Conventions to match (Balaur law, inlined)

- gomponents: alias `g "maragu.dev/gomponents"`, `h "maragu.dev/gomponents/html"`
  (see `memory.go:12-14`). The click-morph links in this plan use the `g.Attr`
  string form `g.Attr("data-on:click__prevent", "@get('...')")` (Major-2 note in
  Step 2), so you do NOT need the `data "maragu.dev/gomponents-datastar"` alias —
  importing it unused would fail the build. User/model text (node titles) renders
  through escaping `g.Text` — **never** `g.Raw` for a node title. `g.Raw` is only
  for already-rendered trusted HTML.
- Errors are values: `fmt.Errorf("doing x: %w", err)`, return early, no panics
  in library code. Builders that hit the DB return `(View, error)` or swallow to
  an empty view the way `memory.go:229` does (`recs, _ := ...`) — match whichever
  the surrounding code uses; prefer surfacing the error to the card error strip.
- Structured logging via `app.Logger()` (slog) with key/value pairs — no
  `fmt.Print*`/`log.Printf`.
- Domain reads stay in the domain package: graph traversal lives in
  `internal/nodes` (161), the card package only composes views. Do NOT route new
  domain logic through `internal/store`.
- Storybook is the UI source of truth: any new card gets a story in the same
  change (`internal/feature/storybook/stories_cards.go` + registered in
  `story.go`'s `stories` slice).
- `internal/self/knowledge.md` is the running binary's self-description; update
  it when capabilities change (this plan adds a visible capability).

## Commands you will need

| Purpose | Command | Expected on success |
|---|---|---|
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test (all) | `go test ./...` | all pass |
| Test (env-scrubbed) | `env -u BALAUR_OS_ACCESS -u BALAUR_SOURCE -u BALAUR_MAX_STEPS go test ./...` | all pass |
| Test (this package) | `go test ./internal/feature/graphcards/... ./internal/cards/... ./internal/feature/storybook/...` | all pass |
| Format check | `gofmt -l internal/` | no output |
| Diff hygiene | `git diff --check` | no output |
| UI verify | `/verify` (run-balaur) → drive `http://127.0.0.1:8090/` | related list + graph render in the panel |

## Suggested executor toolkit

- Invoke the **`ui-development`** skill before writing the cards — it carries the
  Hearthwood tokens, the storybook-as-source-of-truth workflow, and the
  Datastar `@get`/SSE contract used by `/ui/show`.
- Invoke the **`go-standards`** skill for the error/logging/test idioms.
- Use **`run-balaur`** (`/verify`) for the browser check at the end.
- After code lands, run `graphify update .` (AST-only, no API cost) and minify
  `graphify-out/graph.json` before committing (the graph is committed minified).

## Scope

**In scope** (the only files you should create/modify):

- `internal/feature/graphcards/graphcards.go` (create) — package doc +
  `Register`/`Unregister`/`init`, and `registerRelated`/`registerGraph`.
- `internal/feature/graphcards/related.go` (create) — the related-nodes
  view-model, builder (Backlinks ∪ Outbound ∪ FTS-similar), and `RelatedCard`.
- `internal/feature/graphcards/graph.go` (create) — the 1-hop neighborhood
  view-model, concentric-layout builder, and `GraphCard` (server-rendered SVG).
- `internal/feature/graphcards/graphcards_test.go` (create) — `TestNoWebImports`
  marker, the related-computation test, and the SVG render smoke test.
- `internal/cards/cards.go` (modify) — add the `related` and `graph` specs to
  the `registry` slice in `init()`.
- `internal/feature/all/all.go` (modify) — add the blank import for graphcards.
- `internal/feature/storybook/stories_cards.go` (modify) — add `relatedStory()`
  and `graphStory()`.
- `internal/feature/storybook/story.go` (modify) — register the two new stories
  in the `stories` slice.
- `internal/self/knowledge.md` (modify) — one line documenting the related/graph
  surface.

**Out of scope** (do NOT touch, even though they look related):

- `internal/web/show.go`, `internal/web/cards.go` — the dispatcher already serves
  any registered card type; touching it means you've misread the design. STOP.
- The migration / `internal/nodes` schema / `migrations/*` — 160 owns the
  schema; 161 owns the edge-sync hook. This plan only READS.
- `internal/search/*` and the FTS table — 162 owns all search changes. Read
  similar nodes only through the helper 162 exposes; never widen the FTS table
  or import `internal/search` here.
- `internal/web/home.go` `commandPaletteNode` — do NOT add a related/graph
  palette item; both need a runtime `id` and a parameterless launcher would 400
  (see Exemplar 5). *(home.go is listed in the drift check only because the
  reviewer must confirm you did NOT change it.)*
- Force-directed / interactive / zoomable graph, physics, clustering,
  whole-vault rendering, Capacities typed-object graph, note↔task links — all
  DEFERRED (see Maintenance notes).

## Git workflow

- Branch: `improve/163-knowledge-graph-view` (executor worktrees base off
  `origin/main`).
- Commit per logical unit; conventional-commit subjects, e.g.
  `feat(graphcards): related-nodes list + 1-hop SVG graph card`.
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Register the `related` and `graph` card specs

In `internal/cards/cards.go`, inside `init()`'s `registry = []Spec{ ... }`
literal, add two specs (place them after the `skills` spec for tidiness — order
is cosmetic). Both take a **required** `id` (the focus node). `related` takes an
optional `limit`; `graph` takes an optional `limit` (max neighbors to draw).

```go
		{
			Type:  "related",
			Label: "Related",
			Icon:  "tome",
			W:     4,
			H:     20,
			Params: []ParamSpec{
				{Name: "id", Required: true, Doc: "the focus node id whose neighbors to list"},
				{Name: "limit", Doc: "max related nodes to show (default 12, max 50)"},
			},
		},
		{
			Type:  "graph",
			Label: "Graph",
			Icon:  "tome",
			W:     6,
			H:     24,
			Params: []ParamSpec{
				{Name: "id", Required: true, Doc: "the focus node id whose 1-hop neighborhood to draw"},
				{Name: "limit", Doc: "max neighbors to draw (default 12, max 24)"},
			},
		},
```

> Note: `Validate` clamps `limit` to `[1,50]`. The graph card additionally caps
> neighbors at 24 in its own builder (a dense SVG past ~24 nodes is unreadable —
> see Step 4). The registry clamp is the outer bound; the builder cap is the
> visual one. Use `scroll`/`tome`/`orb`/`quill`/`key`/`shield`/`hourglass`/`flame`
> icon stems only (the stems already in use across `cards.go`) — do NOT invent a
> stem; `tome` (the knowledge icon) is correct here.

**Verify**: `go test ./internal/cards/...` → all pass. Then
`CGO_ENABLED=0 go build ./internal/cards/...` → exit 0.

### Step 2: Build the related-nodes computation + card (`related.go`)

Create `internal/feature/graphcards/related.go`. The builder unions:

1. **Backlinks** — `nodes.Backlinks(app, id)` (active nodes linking TO id).
2. **Outbound** — `nodes.Outbound(app, id)` (active nodes id links FROM).
3. **FTS-similar** (OPTIONAL) — top results from 162's node-search helper for the
   focus node's `title`, EXCLUDING id itself and anything already in (1)/(2).
   **If 162's helper is absent or returns an error, skip this section entirely —
   no error, no log noise.** Guard the call so the related list always renders
   from edges alone.

De-duplicate by node id (a node that is both a backlink and outbound appears
once), drop the focus node itself, and cap the merged list to `limit`
(default 12). Each row carries `{ID, Title, Type, Rel}` where `Rel` is a short
label: `"backlink"`, `"links to"`, or `"similar"` (first source wins on a tie).

View-model + card shape (match `memory.go`'s row/list style — `ui.CardHead`,
`h.Ul(h.Class("ucard-list"))`, `h.Li(h.Class("ucard-row"))`, an
`ui.EmptyState` when empty). Each row links to the focus of that neighbor via
the 160 node-show route — **use 160's real route**; during the drift check,
read 160 to learn the exact path (it is `GET /ui/show/{type}?id=...`). The
node's `type` is the `{type}` segment and its id is the `id` param:

```go
// RelatedRow is one neighbor in the RelatedCard.
type RelatedRow struct {
	ID    string
	Title string
	Type  string // the node type, e.g. "note", "person"
	Rel   string // "backlink" | "links to" | "similar"
}

// RelatedView is the view-model for RelatedCard.
type RelatedView struct {
	FocusID    string
	FocusTitle string
	Rows       []RelatedRow
}

// RelatedCard lists the active nodes connected to the focus node.
func RelatedCard(v RelatedView) g.Node {
	// h.Article(h.Class("kcard ucard ucard-related"), h.ID("ucard-related"),
	//   ui.CardHead("/static/icons/tome.png", "Related",
	//     h.Span(h.Class("kcard-meta"), g.Text(v.FocusTitle))),
	//   relatedBody(v),
	//   h.Footer(h.Class("kcard-actions"),
	//     h.A(h.Href("/ui/show/graph?id="+v.FocusID),
	//       g.Attr("data-on:click__prevent", "@get('/ui/show/graph?id="+v.FocusID+"')"),
	//       g.Text("see graph →"))))
}
```

> **One attribute style only.** Use the verified house form
> `g.Attr("data-on:click__prevent", "@get('...')")` — NOT the
> `data.On(...)`/`data.ModifierPrevent` helper. The whole codebase uses the
> `g.Attr` string form for click-morph links (`memory.go:59`, `memory.go:185`),
> and the storybook escaping assertion in Step 6 (`&#39;` for the single quotes)
> only matches this form.

Each row's link uses the **node-show route from 160**. The card-footer
"see graph →" / "manage all →" links morph the panel via the
`h.A` + `h.Href` + `g.Attr("data-on:click__prevent", "@get(...)")` pattern —
see the footer link at `memory.go:59` and the meta link at `memory.go:185`
(both verified to use the `g.Attr` string form). Apply that same `h.A` pattern
to each related row so a click morphs the panel (the `/ui/show` door) instead of
a full navigation. Build the `@get` URL from the neighbor's type+id; HTML-escape
via `g.Text` for the visible title. (The plain per-row title link at
`memory.go:76` omits the Datastar attr; here we want the morph, so include the
`data-on:click__prevent` `@get` on the row.)

The builder signature and guard pattern:

```go
func buildRelated(app core.App, params map[string]string) RelatedView {
	id := params["id"]
	limit := ui.IntParam(params, "limit", 12)
	// 1+2: edges (these already filter to status=active in package nodes).
	back, _ := nodes.Backlinks(app, id)
	out, _ := nodes.Outbound(app, id)
	// merge+dedupe back then out (first source wins), cap to limit ...
	// 3: FTS-similar — OPTIONAL, guarded; skip entirely if the helper/index is absent.
	//    e.g. recs, err := <162 node-search helper>(app, focusTitleTerms, limit)
	//    if err == nil { append non-dupes, type/title from each rec } else { skip }
	// load focus title (FindRecordById("nodes", id)); empty view if not found/active.
}
```

`ui.IntParam(params, "limit", 12)` is the existing default helper
(`internal/ui/text.go:22`). Use `app.Logger()` only on a genuinely unexpected
error; a missing FTS index is expected and silent.

**Verify**: `CGO_ENABLED=0 go build ./internal/feature/graphcards/...` → exit 0
(after Step 5 wires the package). For now: `go vet ./internal/feature/graphcards/...`
once the file compiles alongside Steps 3–5.

### Step 3: Build the 1-hop SVG graph card (`graph.go`)

Create `internal/feature/graphcards/graph.go`. The card draws the focus node at
the center and its direct neighbors (Backlinks ∪ Outbound, active only) on a
single ring — a dead-simple **concentric/radial** layout computed in Go (no
physics, no force-direction). Cap neighbors at **24** (the visual cap; past that
an SVG ring is unreadable — say so in a comment).

Layout math (mirror `sparkline.go`'s "compute coords in Go, emit `g.El`"
pattern):

```go
const (
	graphW, graphH = 360, 360
	graphR         = 150 // ring radius
	nodeR          = 6   // neighbor dot radius
	focusR         = 9   // focus dot radius
	maxNeighbors   = 24  // visual cap; a denser ring is unreadable
)

// GraphNode is one drawn node.
type GraphNode struct {
	ID    string
	Title string
	Type  string
}

// GraphView is the view-model for GraphCard.
type GraphView struct {
	FocusID    string
	FocusTitle string
	Neighbors  []GraphNode
}

// GraphCard renders the focus node + its 1-hop neighbors as a concentric SVG.
// Edges are drawn from center to each neighbor; nodes are <circle> + <text>.
// Server-rendered, Datastar-only — NO physics, NO interactivity (deferred).
func GraphCard(v GraphView) g.Node {
	// center = (graphW/2, graphH/2)
	// for i, n := range v.Neighbors (capped to maxNeighbors):
	//   angle := 2*math.Pi*float64(i)/float64(len(neighbors))
	//   x := cx + graphR*math.Cos(angle); y := cy + graphR*math.Sin(angle)
	//   emit g.El("line", from center to (x,y), stroke var(--line))
	//   emit g.El("circle", cx=x cy=y r=nodeR fill var(--teal-ink)) with a
	//        g.El("title", g.Text(n.Title)) child for hover
	//   emit g.El("text", x=x y=y+labelOffset, g.Text(truncate(n.Title)))
	// then emit the focus circle (r=focusR) + its label LAST so it sits on top.
	// wrap all children in g.El("svg", viewBox "0 0 360 360", role="img", ...).
}
```

Rules:

- All coordinates are computed floats formatted with `strconv` — never
  interpolate a node title into a coordinate/path/attribute. Titles appear ONLY
  inside `g.El("text", g.Text(title))` and `g.El("title", g.Text(title))`
  (both escape). Truncate visible labels (e.g. 18 chars + "…") so they don't
  overflow; the full title lives in the `<title>` hover element.
- Empty neighborhood: still render the focus dot + label and a small caption
  ("No links yet") — an SVG with one node, not a blank/error.
- Wrap the SVG in a `kcard`/`ucard` article with `ui.CardHead("/static/icons/tome.png", "Graph", ...)`
  and a footer link "list related →" pointing at `/ui/show/related?id={focusID}`
  (the Datastar `@get` morph, mirroring related.go's "see graph →").

Make the neighbor dots clickable to re-focus: wrap each neighbor's
`<circle>`/`<text>` in an `<a>` (`g.El("a", g.Attr("href", showURL), ...)`)
with a `data-on:click__prevent` `@get` to that neighbor's node-show route — same
pattern as the row links, just on the SVG node. (SVG `<a>` is valid; keep it
simple — if Datastar-on-SVG misbehaves in the browser check, fall back to a
plain `href` and STOP-report, do not invent a workaround.)

**Verify**: `CGO_ENABLED=0 go build ./internal/feature/graphcards/...` → exit 0
(after Step 5). The render smoke test in Step 6 asserts the SVG structure.

### Step 4: Register the cards (`graphcards.go`)

Create `internal/feature/graphcards/graphcards.go` — the package doc + the
feature registration, copying `knowledgecards/register.go` exactly:

```go
// Package graphcards renders the "see the network" cards (related, graph) —
// a related-nodes list and a 1-hop server-rendered SVG graph — over the
// edges plan 161 maintains. Read-only; status=active only. Imports
// internal/ui, internal/feature, internal/nodes, gomponents, and
// pocketbase/core only — never internal/web (the layering law, spec §4.1).
package graphcards

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/alexradunet/balaur/internal/feature"
	"github.com/alexradunet/balaur/internal/ui"
)

func Register(app core.App) {
	ui.RegisterCard("related", func(_ ui.CardSize, params map[string]string) (g.Node, error) {
		return RelatedCard(buildRelated(app, params)), nil
	})
	ui.RegisterCard("graph", func(_ ui.CardSize, params map[string]string) (g.Node, error) {
		return GraphCard(buildGraph(app, params)), nil
	})
}

func Unregister() {
	ui.UnregisterCard("related")
	ui.UnregisterCard("graph")
}

func init() {
	feature.Add(feature.Funcs(Register, Unregister))
}
```

(Add the `g "maragu.dev/gomponents"` import; both renderers ignore `size`.)

**Verify**: `CGO_ENABLED=0 go build ./internal/feature/graphcards/...` → exit 0.

### Step 5: Blank-import graphcards into the feature aggregator

In `internal/feature/all/all.go`, add one import line (keep alphabetical order):

```go
	_ "github.com/alexradunet/balaur/internal/feature/graphcards"
```

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0. Then run the app's wiring
test: `go test ./internal/feature/...` → all pass (this confirms RegisterAll
picks up the new feature without panicking on a double registration).

### Step 6: Tests (`graphcards_test.go`)

Create `internal/feature/graphcards/graphcards_test.go` with package
`graphcards_test`. Three tests, modeled on
`internal/knowledge/knowledge_test.go:379` (the integration pattern) and
`internal/feature/knowledgecards/knowledgecards_test.go` (the no-web-imports
marker):

1. **`TestNoWebImports`** — the compile-time marker (copy verbatim from
   knowledgecards_test.go; only the package name + log string differ).
2. **`TestRelatedComputation`** — uses `storetest.NewApp(t)` to get a temp-dir
   app with the Balaur schema (nodes/edges from 160's baseline). **The helper
   lives in `internal/storetest`, NOT `internal/store`** — import
   `"github.com/alexradunet/balaur/internal/storetest"` and call
   `storetest.NewApp(t)` (the temp-dir app helper at
   `internal/storetest/storetest.go:18`, signature `func NewApp(t *testing.T) core.App`).
   The Balaur-law phrase "test helpers in internal/store" refers to fakes/seams
   in that package; the temp-app constructor itself is `storetest.NewApp`. Exact
   usage exemplar: `internal/knowledge/knowledge_test.go:380` (`app := storetest.NewApp(t)`
   with the import at line 12). Seed via the
   161 helpers / direct `app.Save` of `nodes` + `edges` records: a focus node A
   (active), a backlink node B→A (active), an outbound node A→C (active), and a
   **proposed** node D→A. Call `buildRelated(app, map[string]string{"id": A.Id})`
   and assert: B and C appear; D does **not** (status filter); the focus node A
   does not appear in its own list. (How to seed edges: read 161's edge-sync —
   either save an `edges` record `{source, target, type:"links"}` directly, or
   use a 161 helper if it exposes one. Save `nodes` rows with `status:"active"`
   / `"proposed"` and the required `title`/`type` fields per 160's schema; if a
   required field's name differs from this plan, read 160 and adapt.)
3. **`TestGraphCardRendersSVG`** — call
   `GraphCard(buildGraph(app, map[string]string{"id": A.Id}))`, render it to a
   string (`var b strings.Builder; node.Render(&b)`), and assert string-contains:
   `"<svg"`, `"<circle"`, `"<line"`, the focus title, and that the proposed
   node D's title is **absent**. Also assert an empty-neighborhood node renders
   `"<svg"` + the focus dot (one circle, no `<line>`).

Render-to-string pattern (the storybook + web tests use this; gomponents
`Render(w io.Writer)`):

```go
var b strings.Builder
if err := GraphCard(buildGraph(app, params)).Render(&b); err != nil { t.Fatal(err) }
out := b.String()
if !strings.Contains(out, "<svg") { t.Fatalf("no svg: %s", out) }
```

Note the escaping fact: Datastar single-quoted `@get('...')` attributes render
HTML-escaped as `&#39;` (seen across the web tests) — assert against the escaped
form if you assert on a `@get` URL.

**Verify**: `go test ./internal/feature/graphcards/...` → all pass (3 tests).
Then `env -u BALAUR_OS_ACCESS -u BALAUR_SOURCE -u BALAUR_MAX_STEPS go test ./...`
→ all pass.

### Step 7: Storybook stories for `related` and `graph`

In `internal/feature/storybook/stories_cards.go`, add `relatedStory()` and
`graphStory()` following `knowledgecardStory()` / `lifelogfocusStory()` (pass
hand-built view-models — stories never hit the DB). Add
`"github.com/alexradunet/balaur/internal/feature/graphcards"` to the file's
import block. Each story sets `ID`, `Group: "Cards"`, `Title`, a `Blurb`, 2–3
`Variants` (e.g. related: "with backlinks + outbound", "empty"; graph:
"3 neighbors", "empty neighborhood"), a `Props` table, and `Dos`/`Donts`
(do: "status=active only — proposals never appear"; don't: "add physics or
interactivity — the graph is a read-only 1-hop snapshot (deferred)").

Then in `internal/feature/storybook/story.go`, register both in the `stories`
slice (after `knowledgecardStory()`):

```go
	knowledgecardStory(),
	relatedStory(),
	graphStory(),
```

**Verify**: `go test ./internal/feature/storybook/...` → all pass (the
storybook `story_test.go` renders every story and asserts no nil/panic). Then
`gofmt -l internal/` → no output.

### Step 8: Update self-knowledge

In `internal/self/knowledge.md`, add one line under the knowledge/graph section
(read the file to find the right anchor) documenting that Balaur can now show a
node's **related nodes** (backlinks ∪ outbound ∪ FTS-similar) and a **1-hop SVG
graph** of its neighborhood, both read-only and `status=active`-only, served at
`/ui/show/related?id=` and `/ui/show/graph?id=`. Keep it to 1–2 sentences in the
existing voice.

**Verify**: `grep -n "related\|graph" internal/self/knowledge.md` → at least one
new line mentioning the related/graph surface.

### Step 9: Full gate + graph refresh

Run the full Balaur gate set and refresh the code graph.

**Verify** (all must pass):
- `gofmt -l internal/` → no output
- `go vet ./...` → exit 0
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go test ./...` → all pass
- `git diff --check` → no output
- `graphify update .` then minify `graphify-out/graph.json` (the graph is
  committed minified — see the project graphify rule).

## Test plan

- **New file** `internal/feature/graphcards/graphcards_test.go` (package
  `graphcards_test`):
  - `TestNoWebImports` — compile-time layering marker (copy from
    `internal/feature/knowledgecards/knowledgecards_test.go`).
  - `TestRelatedComputation` — happy path (backlink + outbound surface) AND the
    **consent regression** (a proposed neighbor is excluded) AND the focus node
    is not in its own list. Structural pattern:
    `internal/knowledge/knowledge_test.go:379` (`storetest.NewApp` + seed +
    assert).
  - `TestGraphCardRendersSVG` — the **SVG render smoke test**: asserts `<svg`,
    `<circle`, `<line`, focus title present, a proposed node's title absent, and
    the empty-neighborhood single-node render.
- **Existing tests that must stay green unchanged** (the behavior contract):
  `internal/cards/...` (the registry validates the new specs), and
  `internal/feature/storybook/story_test.go` (renders every story including the
  two new ones).
- Verification: `go test ./...` → all pass, including the 3 new graphcards tests
  and the 2 new stories rendered by `story_test.go`.
- **Browser check** (`/verify`, run-balaur): with a seeded node that has at least
  one `[[link]]`, open `/ui/show/related?id=<id>` and `/ui/show/graph?id=<id>` in
  the panel and confirm the list + the SVG render and the row/dot links re-focus.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0; the 3 new `graphcards` tests exist and pass
- [ ] `env -u BALAUR_OS_ACCESS -u BALAUR_SOURCE -u BALAUR_MAX_STEPS go test ./...` exits 0
- [ ] `gofmt -l internal/` produces no output
- [ ] `git diff --check` produces no output
- [ ] `grep -n '"related"' internal/cards/cards.go` AND `grep -n '"graph"' internal/cards/cards.go` each return a match (specs registered)
- [ ] `grep -n 'graphcards' internal/feature/all/all.go` returns a match (blank-imported)
- [ ] `grep -n 'relatedStory\|graphStory' internal/feature/storybook/story.go` returns matches (stories registered)
- [ ] `grep -rn 'internal/web' internal/feature/graphcards/*.go` returns NOTHING (layering law; test files included)
- [ ] `grep -rn 'internal/search' internal/feature/graphcards/*.go` returns NOTHING (162 owns search; read via the helper only)
- [ ] `git diff --stat 72fd762..HEAD -- internal/web/show.go internal/web/cards.go internal/search migrations` lists NO files (read-only boundary respected)
- [ ] `internal/self/knowledge.md` mentions the related/graph surface
- [ ] `plans/readme.md` status row for 163 updated

## STOP conditions

Stop and report back (do not improvise) if:

- **161 has not landed**: package `internal/nodes` has no `Backlinks`/`Outbound`
  (or combined `Neighbors`) helper, OR the `edges` collection's relation fields
  are not named `source`/`target`. This plan hard-depends on 161's edge layer;
  building a parallel traversal here would duplicate it and risk leaking
  proposed nodes.
- **160's node-show route differs** from `GET /ui/show/{type}?id=...`: read
  160's actual registered route and use it; if there is no generic node-show
  route at all, STOP — the row/dot links have no destination.
- The code at the "Current state" excerpts does not match the live files (the
  tree drifted since `72fd762`) — re-run the drift check and compare.
- A step's verification fails twice after a reasonable fix attempt.
- The fix appears to require editing `internal/web/show.go`, `internal/web/cards.go`,
  `internal/search/*`, or a migration — all out of scope. If you think you need
  to, you've misread the design.
- Datastar `@get` on an SVG `<a>` does not work in the browser check — fall back
  to a plain `href` for the SVG node links and report it; do not invent a JS
  workaround (no JS framework — Balaur law).

## Maintenance notes

For the human/agent who owns this after the change lands:

- **What a reviewer should scrutinize**: (1) the `status=active` filter is
  enforced on EVERY path into the related list and the graph — a proposed node
  must never appear; the regression test (`TestRelatedComputation` with a
  proposed neighbor) guards this, so confirm it actually fails when the filter is
  removed. (2) No node title is ever interpolated into an SVG coordinate, path,
  or attribute — titles go only through `g.Text` in `<text>`/`<title>`. (3) The
  FTS-similar section degrades to silence when 162's index is absent (no error,
  no log spam).
- **What future changes interact with this**: if 161 changes the
  `Backlinks`/`Outbound` signatures or the edge relation names, the builders here
  break — they are the only consumers in this package. If 162 renames its
  node-search helper, the optional FTS-similar call here must follow.
- **Deliberately deferred (name in "Known limitations & deferred work")**, do
  NOT build here:
  - **Interactive / force-directed / zoomable graph, physics, clustering** — 163
    ships a static 1-hop SVG snapshot only.
  - **Whole-vault graph rendering at scale** — capped at one hop, ≤24 neighbors;
    a paginated/zoomed full-vault canvas is a later UI concern over the same
    edges.
  - **Multi-hop traversal (recursive CTE)** — the LOCKED design names it for a
    later layer; v1 related/graph is 1-hop only.
  - **Capacities typed-object graph** (per-type schemas/templates) and
    **note↔task cross-layer links** (edges are node↔node only in v1).
  All of these are additive over the stable `nodes`/`edges` core — none requires
  reshaping what this plan ships.
