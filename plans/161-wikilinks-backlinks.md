# Plan 161: Wikilinks & backlinks on the nodes+edges graph

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
> **Drift check (run first)**: `git diff --stat 6ab038a..HEAD -- internal/nodes internal/feature/knowledgecards internal/ui/chat/markdown.go internal/ui/chat/message.go main.go internal/feature/storybook/stories_cards.go internal/self/knowledge.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: MED
- **Depends on**: plans/160-nodes-edges-spine.md (the `nodes` + `edges`
  collections, the `note` card type, and the `GET /ui/show/{type}` node route
  it registers). **160 has MERGED** (HEAD `6ab038a`, "merge: 160 — greenfield
  nodes+edges knowledge spine") — its `internal/nodes` package, `note` card,
  and the `GET /ui/show/note` route are live. This plan's "Current state"
  excerpts below have been refreshed against that post-160 tree.
- **Category**: direction
- **Planned at**: commit `6ab038a`, 2026-06-23 (reconciled against post-160 main)

## Design conflict resolutions (advisor — AUTHORITATIVE; override the in-body option menus)

The reconcile pass flagged two design conflicts "for the advisor to choose." Both are now DECIDED. These OVERRIDE the option menus in **Step 5** ("DESIGN CONFLICT … the advisor must resolve") and **Step 8** (the backlinks-fixture "resolve this before coding" note) — follow these, do not re-deliberate:

**1. The note-card body renders clickable `[[links]]` (Step 5 → option b).** The owner chose: links must be clickable where notes are read. So the note card renders its body through `chat.RenderMarkdownLinked` (reversing 160's "escaped `g.Text`, no goldmark in knowledgecards" deferral). Concretely:
- `internal/feature/knowledgecards` imports `internal/ui/chat` (a leaf UI package — no import cycle) and calls `chat.RenderMarkdownLinked(body, resolver)`.
- Build the resolver in `buildNote` (it has the app): `func(title string)(string,bool){ rec,err := app.FindFirstRecordByFilter("nodes","status='active' && title={:t}",dbx.Params{"t":title}); if err==nil {return rec.Id,true}; return "",false }`.
- Keep `NoteCard(v NoteView)` app-free + storybook-renderable: **pre-render in `buildNote`** into a new `NoteView` field (e.g. `BodyNode g.Node = chat.RenderMarkdownLinked(rec.GetString("body"), resolver)`); `NoteCard` emits `v.BodyNode` in place of the `g.Text(v.Body)` div. **Update 160's existing `notecardStory()` variants** (and any other `NoteView{Body:…}` literal) to the new field — a plain `g.Text("…")` BodyNode is fine in stories (no app/resolver needed there).
- **Revise `note.go`'s package doc** (the "keep goldmark out of knowledgecards / defer markdown" comment) to state the body now renders linked markdown via the shared chat renderer — a deliberate 161 decision superseding 160's deferral.
- Keep `renderMarkdown` (plain chat) byte-identical; only ADD `RenderMarkdownLinked`.
- **Add a Maintenance note**: `RenderMarkdownLinked` is now used by BOTH the chat bubble and the note card; a future cleanup may promote it from `internal/ui/chat` to a shared `internal/ui` atom (the reconcile's option c) so cards don't import the chat package — DEFERRED, not in 161.

**2. The backlinks panel uses a view-model (Step 8 → option i).** `LinkedFrom` takes `[]BacklinkView` where `type BacklinkView struct { ID, Title string }` — NOT `[]*core.Record`. `buildNote` maps `nodes.Backlinks(app, rec.Id)` records → `[]BacklinkView` (`.Id`, `.GetString("title")`). Storybook passes plain `BacklinkView{ID:"n1",Title:"…"}` literals. Update Step 6's `LinkedFrom` signature + Step 8's fixture accordingly. This keeps the card/story layer free of a `*core.Record` dependency.

## Why this matters

Plan 160 lands the `nodes` + `edges` knowledge graph as data, but nothing yet
*creates* edges from what the owner writes. Wikilinks are the connective tissue:
the owner (and Balaur) type `[[Another Note]]` in a node body, and the graph
grows links automatically — resolving to an existing node by title, or creating
a stub node on the spot (LogSeq's create-on-link). Backlinks ("Linked from")
then make the graph navigable: open a node, see everything that points at it.
Without this, `nodes`/`edges` is an inert schema; with it, Balaur has a real
personal-knowledge-management spine. The Pareto slice here is one regexp parser,
one save hook that mirrors the existing FTS index hook, two query helpers, a
render extension to the existing markdown renderer, and a "Linked from" panel in
the node card — no graph view, no search, no block refs (all deferred).

## Current state

The facts the executor needs, inlined. Re-read each file before editing — line
numbers are leads, not facts.

### What plan 160 provides (SHIPPED — verified verbatim against `internal/nodes/nodes.go` at HEAD `6ab038a`)

160 shipped the `nodes` and `edges` collections, `internal/nodes/nodes.go`, and
the `note` card. The shared contract, confirmed against the live code:

- **`nodes`** collection fields: `type` (select), `title` (text, required),
  `body` (text, markdown), `status` (select: `active|proposed|archived|rejected`),
  `props` (json), `created`/`updated` (autodate). Status constants are real
  (`internal/nodes/nodes.go:26-31`): `StatusProposed = "proposed"`,
  `StatusActive = "active"`, `StatusArchived = "archived"`, `StatusRejected =
  "rejected"`.
- **`edges`** collection fields: `source` (relation→nodes), `target`
  (relation→nodes), `type` (text), `context` (text, optional). The relation
  field names ARE `source`/`target` (confirmed — `AddEdge` sets
  `rec.Set("source", …)`/`rec.Set("target", …)` at `nodes.go:172-173`, and
  `Backlinks`/`Outbound` filter `target = {:id}`/`source = {:id}`). Unique index
  on `(source,target,type)` (`idx_edges_unique`) and an index on `target`
  (`idx_edges_target`) for backlinks (`migrations/1749600000_init.go:110-117`).
  Both `source` and `target` have `CascadeDelete: true` (lines 110-111) — so a
  deleted node's edges are removed automatically; **no delete hook is needed in
  161** (confirmed — see the Step 4 cascade note). The PocketBase auto back-relation
  expands `edges_via_target`/`edges_via_source` DO exist (migration comment, line
  106), **but 160's `Backlinks`/`Outbound` do NOT use them — they query the `edges`
  collection directly (`FindRecordsByFilter("edges", …)`, `nodes.go:216`/`229`)
  then load the active nodes by id via the unexported `activeByIDs`
  (`nodes.go:193-212`). Call the helpers; do not hand-roll an expand.**
  The default edge type is the exported `nodes.DefaultEdgeType = "links"`
  (`nodes.go:36`).
- **The node route** is the generic `GET /ui/show/{type}?id=...` dispatcher (the
  existing card system — `internal/web/show.go`, `uiShow` at line 19, looks up
  the spec via `cards.Get(typ)` at line 31). 160 registered a `note` card type
  (`registerNote` in `internal/feature/knowledgecards/note.go:77-81`) so
  `GET /ui/show/note?id=<nodeId>` works.
- **`internal/nodes` is 160's domain package — REUSE it, do NOT redeclare.**
  `internal/nodes/nodes.go` (260 lines) ALREADY owns these, with the exact live
  signatures (copy them verbatim — they all filter to `status=active` where it
  matters):
  - `func AddEdge(app core.App, sourceID, targetID, edgeType, context string) (*core.Record, error)`
    (`nodes.go:163`) — defaults `edgeType` to `DefaultEdgeType` ("links") when
    empty (`nodes.go:164-166`); **idempotent against the `(source,target,type)`
    unique index** — on a save error it find-and-returns the existing edge
    (`nodes.go:176-184`).
  - `func Backlinks(app core.App, id string) ([]*core.Record, error)`
    (`nodes.go:215`) — inbound active nodes (queries `edges` on `target = {:id}`,
    then `activeByIDs`).
  - `func Outbound(app core.App, id string) ([]*core.Record, error)`
    (`nodes.go:228`) — outbound active nodes (queries `edges` on
    `source = {:id}`, then `activeByIDs`).
  - `func Neighborhood(app core.App, id string) ([]*core.Record, error)`
    (`nodes.go:242`) — the 1-hop set (backlinks ∪ outbound, active, deduped).
  - `func Create(app core.App, typ, title, body, status string, props map[string]any) (*core.Record, error)`
    (`nodes.go:85`) — requires a non-empty title, audits `node.create`.
  - `func Get(app core.App, id string) (*core.Record, error)` (`nodes.go:110`).
  - Status constants `nodes.StatusActive`/`StatusProposed`/`StatusArchived`/
    `StatusRejected` (`nodes.go:26-31`) and `nodes.DefaultEdgeType` (`nodes.go:36`).

  **161 MUST import `internal/nodes` and call these — it must NOT redeclare
  `Backlinks`/`Outbound`/`AddEdge` or the status constants** (two sources of
  truth violates SUCKLESS and the ownership boundary; raw `app.Save(edge)`
  bypasses 160's unique-index idempotency at `nodes.go:176-184`). 161's *new*
  code (the `[[ ]]` parser, the resolve-or-create-stub step, the `SyncLinks`
  body→edge sync, and the save hook) lands **inside `internal/nodes`** alongside
  160's helpers — same package, one source of truth. The live signatures above
  match this plan's assumptions exactly — no mechanical rename is needed.
- **`/ui/show/note?id=<id>` is the generic node viewer.** The shipped `note`
  card (`internal/feature/knowledgecards/note.go`) takes only an `id` param
  (`buildNote` reads `params["id"]` at line 61, loads the node, and maps it to a
  `NoteView`); the rendered kind derives from `rec.GetString("type")`
  (`buildNote` sets `Type: rec.GetString("type")` at line 68, and `NoteCard`
  falls back to `"note"` only when `Type == ""`, lines 37-39). So
  `/ui/show/note?id=<id>` renders a node of **any** type (note, memory, skill,
  person, book, journal, …) by id — the executor does NOT pass a `type` param.
  **CONFIRMED: the route does not require a `type` param** (the STOP condition
  below about a required `type` param does not fire).
  **Every wikilink chip and every backlink chip in this plan therefore uses the
  href `/ui/show/note?id=<id>` regardless of the target node's type.** Never emit
  `/ui/show/node?id=...` (there is no `node` card — it 404s) and never invent
  `/ui/notes/{id}`.

> **VERIFIED at HEAD `6ab038a`**: 160's relation field names ARE `source`/`target`
> (`internal/nodes/nodes.go:172-173`, `216`, `229`), and the `note` card does NOT
> require a `type` param (`buildNote` reads only `params["id"]`,
> `internal/feature/knowledgecards/note.go:60-72`). Both STOP-condition hedges
> below are therefore satisfied — proceed with `source`/`target` and the
> `/ui/show/note?id=<id>` href as written. (Kept as STOP conditions only in case a
> later change drifts them.)

### The FTS index hook — the EXACT pattern to mirror for the link-sync hook

`main.go:202-256` — `registerSearchIndex` opens the sidecar index and binds
record hooks. **Post-160, the FTS hooks bind `"nodes"` (NOT `"memories"` — 160
repointed them when memories folded into the unified `nodes` collection).** The
hook-registration tail (verbatim, `main.go:232-255`):

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

**These FTS hooks already bind `"nodes"` — so `OnRecordAfterCreateSuccess("nodes")`
and `OnRecordAfterUpdateSuccess("nodes")` ALREADY appear once each in `main.go`
before this plan adds anything. Your `registerGraphLinks` hook adds a SECOND
binding on those same events; PocketBase composes them (each `BindFunc` runs and
calls `e.Next()`), so the link-sync hook does NOT replace the FTS hook. Do NOT
edit `registerSearchIndex` — it is 162's territory; just add a sibling
`registerGraphLinks` (Step 4).** (Out of scope: this `registerSearchIndex` body
is plan 162's; leave the `internal/search` index alone.)

It is wired into `OnServe` at `main.go:48-58` (verbatim):

```go
	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		registerKronkEngine(se.App)
		if err := web.Register(se); err != nil {
			return err
		}
		registerRecap(se.App)
		registerNudge(se.App)
		registerBriefing(se.App)
		registerSearchIndex(se.App)
		return se.Next()
	})
```

You will add `registerGraphLinks(se.App)` to this block (after
`registerSearchIndex`) and define it as a sibling of `registerSearchIndex`.

### The markdown renderer to EXTEND (plan 158) — do NOT add a second renderer

`internal/ui/chat/markdown.go` (whole file, 35 lines; verbatim — the `var (md/
mdSane)` block is lines 16-19 and `renderMarkdown` is lines 28-34, func at 28):

```go
package chat

import (
	"bytes"

	"github.com/microcosm-cc/bluemonday"
	"github.com/yuin/goldmark"
	g "maragu.dev/gomponents"
)

// md converts assistant Markdown to HTML. Default goldmark config (no
// html.WithUnsafe) escapes raw HTML blocks, so model-emitted <script> never
// renders; the bluemonday pass below then strips dangerous link schemes
// (javascript:, data:) and any stray markup goldmark passed through. Built once
// and reused — both values are concurrency-safe after construction.
var (
	md     = goldmark.New()
	mdSane = bluemonday.UGCPolicy()
)

// renderMarkdown turns assistant Markdown into a trusted, sanitized HTML node.
// On any error it falls back to escaped plain text — a render failure must never
// blank or unescape the bubble.
//
// ponytail: re-renders the whole accumulated buffer on every streamed token.
// Fine for a local single-owner app with short replies; revisit only if a
// measurement shows it on the hot path.
func renderMarkdown(s string) g.Node {
	var buf bytes.Buffer
	if err := md.Convert([]byte(s), &buf); err != nil {
		return g.Text(s)
	}
	return g.Raw(mdSane.Sanitize(buf.String()))
}
```

Key facts about this renderer (verified):
- Default `goldmark.New()` (no `html.WithUnsafe`) treats `[[Title]]` as ordinary
  text and passes it through **literally** to the output (goldmark has no
  wikilink syntax). So in the rendered HTML the literal substring `[[Title]]`
  survives goldmark.
- `bluemonday.UGCPolicy()` (used as `mdSane`) **allows `<a href>` anchors** with
  `http/https/mailto` and **relative** URLs (UGCPolicy calls `AllowStandardURLs`
  + `AllowRelativeURLs(true)` via `AllowAttrs("href").OnElements("a")`). A
  relative href like `/ui/show/note?id=abc123` survives sanitization. **You will
  verify this empirically in Step 5 — do not trust this paragraph; the test is
  the source of truth.**

### How the renderer is called and tested

`internal/ui/chat/message.go:83-105` — `messageBody` calls `renderMarkdown(content)`
when `markdown` is true (balaur turns + `MessageBody`). The node card from 160
will render its body through markdown too (160 renders the node body; 161 makes
that body's `[[links]]` clickable by rendering through the *same* renderer).

`internal/ui/chat/message_test.go:49-66` — `TestMessageBalaurMarkdown` is the
structural template for a render test: render a node, assert the produced HTML
contains the expected fragments and does NOT contain the raw markup. The render
helper is `render(t, node)` → `uitest.Render(t, node)`
(`internal/ui/chat/helpers_test.go:13`).

### The consent rule (MANDATORY — graph + search filter to status=active)

From `internal/nodes/nodes.go:6-10` (the consent-boundary doc) and the LOCKED
architecture: graph traversal and resolution **must filter to `status=active`**.
Resolve-by-title and backlinks queries must only see and create active nodes;
they must never surface `proposed`/`rejected` nodes. The resolve-by-title
pattern to copy is `LoadSkill` (`internal/knowledge/knowledge.go:398-405`) — note
that post-160 it queries the `nodes` collection by `type`+`status`+`title`, not
an old `skills` collection by `name`:

```go
	rec, err := app.FindFirstRecordByFilter("nodes",
		"type = {:t} && status = {:s} && title = {:name}",
		dbx.Params{"t": string(Skill), "s": StatusActive, "name": name})
```

161's `resolveOrCreateStub` resolves a wikilink target by title across ALL node
types (a `[[Title]]` may point at any kind of node), so drop the `type` clause and
keep only `status` + `title`:
`app.FindFirstRecordByFilter("nodes", "status = 'active' && title = {:title}", dbx.Params{"title": title})`.

### The card show route + dispatcher (160 extends this; 161 reads it)

`internal/web/show.go:25-48` — `uiShow` handles `GET /ui/show/{type}`: looks up
the card spec via `cards.Get(typ)`, validates params, renders the card into the
panel. 160 registers the `note` card so `GET /ui/show/note?id=<id>` works (and
renders a node of any type — see the generic-node-viewer contract above). **Use
this exact route form (`/ui/show/note?id=<id>`) for every chip href; there is no
`node` card (`/ui/show/node` 404s), and never invent `/ui/notes/{id}`.**

### Storybook story conventions

`internal/feature/storybook/story.go:27-51` — a `Story` has `ID/Group/Title/Blurb/
Variants/Props/Dos/Donts`; each `Variant` is `{Label string, Node g.Node}`.
`internal/feature/storybook/story_test.go:35-46` (`TestAllStoriesRender`) renders
every registered story — your new/extended story must render non-empty. **160
SHIPPED the node-card story: `notecardStory()` in
`internal/feature/storybook/stories_cards.go:131-156`, registered in the `stories`
slice at `story.go:94`** (the slice is `var stories = []Story{` at `story.go:53`).
Its variants are `"note"`, `"typed object (person)"`, and `"not found"`, built
directly from `knowledgecards.NoteCard(knowledgecards.NoteView{…})`. **161 EXTENDS
this story** — add a backlinks variant. Because `NoteCard`/`NoteView` carries no
backlinks slot yet (see the note-card section below), the cleanest extension is a
variant that renders the `LinkedFrom` panel (Step 6) alongside a `NoteCard`, e.g.
`g.Group([]g.Node{knowledgecards.NoteCard(...), knowledgecards.LinkedFrom(fixtureNodes)})`,
where `fixtureNodes` is a non-empty `[]*core.Record` (see the fixture warning in
Step 8). Stories are registered in the `stories` slice in `story.go:53` (and the
node story is already there at `story.go:94`).

### Conventions to match (Balaur law — the executor has not read AGENTS.md)

- New domain logic talks to PocketBase **directly in its own package** — do NOT
  route through `internal/store`. 161's new graph code lands in `internal/nodes`
  (160's domain package), which owns its own reads/writes (records-as-domain-
  model). `store` is cross-cutting only. Do NOT create a parallel `internal/graph`
  package — that duplicates 160's `Backlinks`/`Outbound`/`AddEdge`.
- Errors are values: `fmt.Errorf("doing x: %w", err)`, return early, no panics in
  library code.
- Structured logging via `app.Logger()` (a `*slog.Logger`) with key/value pairs.
  No `fmt.Print*`/`log.*` in service code.
- gomponents: alias `h "maragu.dev/gomponents/html"` and `g "maragu.dev/gomponents"`.
  User/model text renders through escaping `g.Text`; `g.Raw` only for
  already-sanitized HTML (the markdown renderer's output is the only legitimate
  `g.Raw` here — it has already passed bluemonday).
- Audit (`store.Audit`) STRICTLY AFTER a successful write. Stub-node creation is
  an owner-initiated side effect of saving the owner's own text — audit it.
- Tests: standard `testing`, table-driven where it helps, no assertion
  frameworks. PocketBase-dependent tests use `storetest.NewApp(t)`
  (`internal/storetest/storetest.go:18`) for a temp-dir app with the schema.
- `gofmt` is law; a PostToolUse hook reformats edited Go files automatically.

## Commands you will need

| Purpose            | Command | Expected on success |
|--------------------|---------|---------------------|
| Build (CGO-free)   | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet                | `go vet ./...` | exit 0 |
| Test (package)     | `go test ./internal/nodes/... ./internal/ui/chat/... ./internal/feature/...` | all pass |
| Test (all)         | `go test ./...` | all pass |
| Test (env-scrubbed)| `env -u BALAUR_OS_ACCESS -u BALAUR_SOURCE -u BALAUR_MAX_STEPS go test ./...` | all pass |
| Format check       | `gofmt -l internal/nodes internal/ui/chat main.go internal/feature` | no output |
| Diff hygiene       | `git diff --check` | no output |
| Graph refresh      | `graphify update .` | updates graphify-out (then minify before commit) |

## Suggested executor toolkit

- Invoke the `go-standards` skill when writing the Go (error wrapping, slog,
  table-driven tests, the gomponents `g.Text`/`g.Raw` rule).
- Invoke the `ui-development` skill for the card render + storybook story work
  (Hearthwood tokens, storybook-as-source-of-truth).
- For the final browser check, the `run-balaur` / `/verify` skill drives
  `http://127.0.0.1:8090/`.

## Scope

**In scope** (the only files you should modify or create):

- `internal/nodes/links.go` (create) — 161's NEW graph code added to 160's
  domain package: the `[[ ]]` parser (`ParseLinks`), resolve-or-create-stub
  (`resolveOrCreateStub`), and the body→edge sync (`SyncLinks`). It REUSES 160's
  `nodes.AddEdge`, `nodes.Backlinks`, `nodes.Outbound` (same package — call them
  directly, do NOT redeclare them). A separate file keeps 161's diff legible
  inside the package 160 created.
- `internal/nodes/links_test.go` (create) — parser table tests + resolution/stub/
  backlinks tests (temp-dir app). These live in the same `nodes` package so they
  can call 160's `Backlinks`/`Outbound` and 161's `SyncLinks`/`ParseLinks`.
- `main.go` — add `registerGraphLinks` (sibling of `registerSearchIndex`) and
  call it in `OnServe`; it calls `nodes.SyncLinks`.
- `internal/ui/chat/markdown.go` — add an exported `RenderMarkdownLinked(s string,
  resolve func(title string) (id string, ok bool)) g.Node` that runs the existing
  pipeline then substitutes resolved `[[Title]]` chips (href
  `/ui/show/note?id=<id>`); keep `renderMarkdown` working unchanged for plain chat
  (success path byte-identical, error path still `g.Text(s)`).
- `internal/ui/chat/markdown_test.go` (create) — wikilink render tests.
- `internal/feature/knowledgecards/*` — add the `LinkedFrom` backlinks panel to
  160's node (`note`) renderer (160's note card lives in `note.go` — a feature
  card package that already imports `pocketbase/core` and `internal/ui`; see
  Step 6). 160 added NO backlinks slot to `NoteView`/`NoteCard`, so you ADD a
  minimal "Linked from" section (thread the backlinks list in via a `NoteView`
  field or render it in the `registerNote` closure — Step 6). Adds an
  `internal/nodes` import for `nodes.Backlinks`. (See the Step 5 DESIGN CONFLICT
  about whether to also markdown-render the note body — do not silently wire it.)
- `internal/feature/storybook/stories_cards.go` — extend the node story (added by
  160) with a "with backlinks" variant, OR add a small wikilink/backlinks story.
- `internal/self/knowledge.md` — add wikilinks/backlinks to the knowledge-layer
  description.
- `plans/readme.md` — status row update (per executor instructions).

**Out of scope** (do NOT touch — another plan or boundary owns it):

- `migrations/*` and the `nodes`/`edges` schema — **160 owns the migration.**
  161 only reads the collections. If you find yourself editing a migration,
  STOP — you've misread the dependency.
- `internal/search/*` and `registerSearchIndex` — **162 owns all search/FTS
  changes.** Do not add nodes to the FTS index here; do not touch `index.go`.
- `internal/feature/knowledgecards/*` node (`note`) card *renderer* internals
  beyond adding the `LinkedFrom` panel call (Step 6) — **160 owns the node card
  component.** 160 added NO backlinks slot, so add a minimal "Linked from" section
  via the helper (a `NoteView` field or the `registerNote` closure), but do not
  restructure `NoteCard` or its edit form. Whether to ALSO markdown-render the
  note body (so `[[links]]` in the body become chips) is a DESIGN CONFLICT the
  advisor must resolve (Step 5) — 160 deliberately renders the body as escaped
  `g.Text` and keeps goldmark out of `knowledgecards`; do not flip that silently.
- The graph view / force-directed UI — **163 owns it.**
- `internal/knowledge/*`, `internal/life/*` (journal) — the memory/skill/journal
  fold-in is 160's foundation change, not 161's.
- Block refs `((...))`, LogSeq queries, typed relations, note↔task cross-layer
  links — DEFERRED (see Maintenance notes). Edges stay node↔node, type `"links"`.

> **Scope trap**: the word "links" appears as the edge `type` default (160) AND as
> a CSS/UI concept. You only ever write edges with `type="links"`; do not invent
> other edge types in this plan.

## Git workflow

- Branch: `improve/161-wikilinks-backlinks` (executor worktrees base off
  `origin/main`; ensure 160 is merged first).
- Commit per logical unit; conventional-commit subjects, e.g.
  `feat(graph): [[wikilink]] parser + edge-sync hook + backlinks`.
- Do NOT push or open a PR unless the operator instructed it. End commit messages
  with the `Co-Authored-By` trailer the repo uses.

## Steps

### Step 1: Add the `[[ ]]` parser to `internal/nodes` (160's package)

Create `internal/nodes/links.go` **inside 160's `internal/nodes` package** (do
NOT create a new `internal/graph` package — 160 already owns the graph helpers;
161 adds the wikilink layer to the same package and reuses them). Start with the
file-level doc comment and the parser. The parser is one regexp; it returns the
distinct link **targets** (the title before any `|alias`), in first-seen order,
de-duplicated.

> The package-level doc comment belongs to 160's `nodes.go`. `links.go` opens
> with `package nodes` and a plain file comment (NOT a second `// Package nodes`
> doc — two doc comments on one package is a vet/lint smell).

Target shape:

```go
// links.go (plan 161): turns [[wikilinks]] in node bodies into "links" edges
// between nodes. The parser, the resolve-or-create-stub step, and the idempotent
// edge sync run on node save. It REUSES this package's AddEdge/Backlinks/Outbound
// (plan 160) — it does NOT redeclare them. All operations filter strictly to
// status=active nodes — proposed/rejected nodes never enter the graph (the
// consent boundary).
package nodes

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/store"
)

// wikilinkRe matches [[Target]] and [[Target|alias]]. The target is group 1
// (everything up to an optional pipe); the alias (group 2) is display-only and
// does not affect resolution. Non-greedy inner match so adjacent links
// [[a]][[b]] split correctly; [^\[\]|] forbids brackets/pipes inside the target
// so nested [[ ]] cannot swallow a closing pair.
var wikilinkRe = regexp.MustCompile(`\[\[([^\[\]|]+?)(?:\|([^\[\]]*))?\]\]`)

// ParseLinks returns the distinct link targets in a body, in first-seen order,
// trimmed and de-duplicated case-insensitively-by-trimmed-title. Empty targets
// (e.g. "[[]]" or "[[  ]]") are skipped.
//
// Case-folding policy (deliberate, document it): the case-insensitive dedup here
// only collapses duplicate links WITHIN one body so we never write two edges for
// "[[Alpha]] [[alpha]]" — it picks the first-seen casing as the resolution title.
// Resolution itself (resolveOrCreateStub, Step 2) matches by EXACT title
// (`title = {:title}`), so the stored casing is what resolves. This is fine for
// the Pareto slice; if case-collision across distinct nodes ("Alpha" vs "alpha")
// ever matters, make both ends consistent then — not now.
func ParseLinks(body string) []string {
	matches := wikilinkRe.FindAllStringSubmatch(body, -1)
	var out []string
	seen := map[string]bool{}
	for _, m := range matches {
		title := strings.TrimSpace(m[1])
		if title == "" {
			continue
		}
		key := strings.ToLower(title)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, title)
	}
	return out
}
```

**Verify**: `CGO_ENABLED=0 go build ./internal/nodes/...` → exit 0.

### Step 2: Add resolve-or-create-stub + idempotent edge sync

Append to `internal/nodes/links.go`. `resolveOrCreateStub` finds an active node
by title or creates a stub (`type=note`, `status=active`, empty body) — LogSeq
create-on-link. `SyncLinks` deletes this source node's prior `type="links"` edges
and inserts the current set, idempotently, **writing each edge through 160's
`AddEdge` (NOT raw `app.Save(edge)`)** so the `(source,target,type)` unique-index
idempotency 160 built is honored.

Target shape:

```go
// resolveOrCreateStub returns the id of the active node titled `title`, creating
// a stub note node if none exists. Stubs are born active (owner-authored content
// links are trusted; the stub is a placeholder the owner can flesh out later).
func resolveOrCreateStub(app core.App, title string) (string, error) {
	rec, err := app.FindFirstRecordByFilter("nodes",
		"status = 'active' && title = {:title}", dbx.Params{"title": title})
	if err == nil {
		return rec.Id, nil
	}
	col, err := app.FindCollectionByNameOrId("nodes")
	if err != nil {
		return "", fmt.Errorf("nodes: find nodes collection: %w", err)
	}
	stub := core.NewRecord(col)
	stub.Set("type", "note")
	stub.Set("title", title)
	stub.Set("body", "")
	stub.Set("status", "active")
	if err := app.Save(stub); err != nil {
		return "", fmt.Errorf("nodes: create stub node %q: %w", title, err)
	}
	store.Audit(app, "owner", "graph.stub", "nodes/"+stub.Id, true,
		map[string]any{"title": title})
	return stub.Id, nil
}

// SyncLinks parses the source node's body, resolves each [[link]] to an active
// node id (creating stubs), and rewrites this source's "links" edges to exactly
// that set. Idempotent: calling twice with the same body is a no-op after the
// first (the delete clears the old set; AddEdge is idempotent against the
// (source,target,type) unique index). A node never links to itself (self-links
// are dropped).
func SyncLinks(app core.App, source *core.Record) error {
	titles := ParseLinks(source.GetString("body"))

	// Resolve to target ids (dedup again post-resolution: two titles may map to
	// the same node).
	wantTargets := map[string]bool{}
	for _, t := range titles {
		id, err := resolveOrCreateStub(app, t)
		if err != nil {
			return err
		}
		if id == source.Id {
			continue // no self-edge
		}
		wantTargets[id] = true
	}

	// Full-replace: delete this source's existing "links" edges, then re-insert
	// the wanted set through AddEdge (160), which is idempotent against the
	// (source,target,type) unique index. (Simplest correct; edge count per node
	// is small — see Maintenance notes for the diff-based optimization.)
	old, err := app.FindRecordsByFilter("edges",
		"source = {:src} && type = {:t}", "", 0, 0,
		dbx.Params{"src": source.Id, "t": "links"})
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
		if _, err := AddEdge(app, source.Id, tgt, "links", ""); err != nil {
			return fmt.Errorf("nodes: create edge %s→%s: %w", source.Id, tgt, err)
		}
	}
	return nil
}
```

`"fmt"` is already in Step 1's import block. **Use 160's real relation field
names** for `source`/`target` if they differ, and **160's real `AddEdge`
signature** if it differs from `AddEdge(app, sourceID, targetID, edgeType,
context string)`.

**Verify**: `CGO_ENABLED=0 go build ./internal/nodes/...` → exit 0;
`go vet ./internal/nodes/...` → exit 0.

### Step 3: Backlinks/Outbound are 160's — confirm, do NOT redeclare

There is **no new traversal code in 161**. `nodes.Backlinks(app, id)`
(`internal/nodes/nodes.go:215`) and `nodes.Outbound(app, id)` (`nodes.go:228`)
already exist in 160's `internal/nodes` package, both filtered to `status=active`
via the unexported `activeByIDs` (`nodes.go:193-212`). The node card (Step 6) and
the tests (Step 7) call those directly. **Do NOT add a second `Backlinks`/`Outbound`
to `links.go`** — that is the duplication this plan was repaired to remove.

Confirm they exist with the right shape before relying on them:

```sh
grep -n "func Backlinks\|func Outbound" internal/nodes/*.go   # 160's helpers — expect one each
```

If either is missing (160 named them differently, or did not ship them), STOP and
report — 160 owns the traversal helpers and 161 must not invent a parallel set.

**Verify**: `grep -c "func Backlinks" internal/nodes/*.go` → reports exactly one
declaration across the package (160's), NOT two.

### Step 4: Wire the save hook in `main.go` (mirror `registerSearchIndex`)

In `main.go`, add `registerGraphLinks(se.App)` to the `OnServe` block right after
`registerSearchIndex(se.App)` (the block at `main.go:48-58`; `registerSearchIndex`
is the last call before `return se.Next()`). Define `registerGraphLinks` as a
sibling function (place it after `registerSearchIndex`, which now ends at
`main.go:256`):

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

The `nodes` import (`"github.com/alexradunet/balaur/internal/nodes"`) is **NOT yet
in `main.go` at HEAD `6ab038a`** — its current internal imports are `cli`,
`conversation`, `kronk`, `llm`, `recap`, `search`, `store`, `tasks`, `turn`, `web`
(`main.go:19-28`). Add `"github.com/alexradunet/balaur/internal/nodes"` to that
import block (gofmt will sort it). Re-check with `grep -n 'internal/nodes' main.go`
first in case a parallel change added it.

> **Hook-collision note (CONFIRMED post-160)**: `registerSearchIndex` ALREADY
> binds the FTS upsert/delete hooks to `"nodes"` (`main.go:253-255` — 160
> repointed them from `"memories"`). So `OnRecordAfterCreateSuccess("nodes")` and
> `OnRecordAfterUpdateSuccess("nodes")` each already appear ONCE in `main.go`
> before this plan. Multiple `BindFunc` on the same `OnRecordAfter*Success("nodes")`
> event compose — each runs and calls `e.Next()` — so adding the link-sync hook
> does NOT replace the FTS hook (and `registerGraphLinks` runs AFTER
> `registerSearchIndex` in `OnServe`, so the FTS upsert fires first, then the
> link-sync). Keep the `e.Next()` tail. The done-criteria `grep` below expects
> exactly ONE link-sync binding from THIS plan, not one binding total on `"nodes"`
> (there will be two create-bindings and two update-bindings total: FTS + links).

> **Re-entrancy note**: `SyncLinks` calls `app.Save` on stub nodes and edge
> records. Saving a stub node re-fires `OnRecordAfterCreateSuccess("nodes")` →
> `SyncLinks(stub)`. That is **safe and terminating**: the stub's body is empty,
> so `ParseLinks("")` is empty, so the recursive sync deletes zero edges and
> inserts zero edges and returns. `SyncLinks` never re-saves the source node — it
> only writes `edges` rows and stub `nodes` rows — so the update hook is never
> re-fired for the source node itself. Edge saves are on the `edges` collection,
> which has no hook. Do NOT add guards for this — the empty-body base case already
> terminates it. (If you observe infinite recursion, STOP — it means 160 created
> stubs with a non-empty body, which violates this plan's assumption.)

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0;
`go vet ./...` → exit 0.

### Step 5: Extend the markdown renderer to render resolved `[[links]]` as chips

In `internal/ui/chat/markdown.go`, add an exported `RenderMarkdownLinked`. It runs
the existing goldmark+bluemonday pipeline, then substitutes `[[Title]]` /
`[[Title|alias]]` occurrences in the **sanitized HTML string** with an anchor
chip. Because goldmark passes `[[...]]` through literally and bluemonday escapes
the inner text but leaves the brackets, the substitution operates on already-safe
HTML; the replacement anchor uses a **relative** href that bluemonday already
permits. Keep `renderMarkdown` (no resolver) working unchanged — it must NOT
substitute, so plain chat is untouched.

Target shape (use the SAME regexp pattern as the parser). **Do NOT import
`internal/nodes` into `internal/ui/chat`**: `internal/ui/chat` imports only
`internal/ui` (verified — `toolrow.go:7` is the package's only internal import; no
domain package is imported anywhere in `internal/ui/chat`, and it does NOT import
`pocketbase/core`). Pulling in `internal/nodes` would create a NEW UI→domain
dependency, violating the package-boundary law. Instead **inline a local copy of
the regexp** (`linkChipRe` below), kept in lockstep with `nodes.wikilinkRe` per
the maintenance note. This duplicated-regexp path is the ONLY path — there is no
import fallback:

```go
// linkChipRe matches the literal [[Title]] / [[Title|alias]] text that survives
// goldmark+bluemonday. Kept in lockstep with internal/nodes wikilinkRe.
var linkChipRe = regexp.MustCompile(`\[\[([^\[\]|]+?)(?:\|([^\[\]]*))?\]\]`)

// RenderMarkdownLinked renders node Markdown like renderMarkdown, then turns each
// [[wikilink]] into a clickable chip. resolve maps a link title to the target
// node's id; ok=false means unresolved (after plan 161's save hook, unresolved
// links should be rare — a stub is created on save — but render must still
// degrade gracefully to a non-link span). The display text is the alias when
// present, else the title. Every chip links to the 160 generic node viewer
// /ui/show/note?id=<id> — that route renders a node of any type by id (it derives
// the type from the record), so no type param is needed; never /ui/show/node
// (no such card) or /ui/notes/{id}.
func RenderMarkdownLinked(s string, resolve func(title string) (id string, ok bool)) g.Node {
	base := renderMarkdownString(s) // sanitized HTML string
	out := linkChipRe.ReplaceAllStringFunc(base, func(m string) string {
		sub := linkChipRe.FindStringSubmatch(m)
		title := strings.TrimSpace(sub[1])
		display := title
		if len(sub) > 2 && strings.TrimSpace(sub[2]) != "" {
			display = strings.TrimSpace(sub[2])
		}
		display = html.EscapeString(display)
		if title == "" {
			return m
		}
		id, ok := resolve(title)
		if !ok || id == "" {
			return `<span class="wikilink wikilink-unresolved">` + display + `</span>`
		}
		href := "/ui/show/note?id=" + html.EscapeString(id)
		return `<a class="wikilink" href="` + href + `">` + display + `</a>`
	})
	return g.Raw(out)
}
```

Refactor `renderMarkdown` to share the string-producing core so there is one
goldmark+bluemonday pipeline (SUCKLESS — one source of truth). `renderMarkdownString`
returns `("", false)` on a goldmark convert error so `renderMarkdown` can keep its
**exact original** error behavior — `g.Text(s)`, the gomponents escaping path the
AGENTS.md rule blesses — rather than silently switching the error fallback to
`g.Raw(html.EscapeString(s))` (a different trust posture). The success path is byte
-identical to the original (`g.Raw(mdSane.Sanitize(...))`), so plain chat is
unchanged:

```go
// renderMarkdownString runs the goldmark+bluemonday pipeline and returns the
// sanitized HTML. ok=false signals a goldmark convert error so callers reproduce
// the original escaped-plain-text fallback (g.Text) instead of trusting raw HTML.
func renderMarkdownString(s string) (string, bool) {
	var buf bytes.Buffer
	if err := md.Convert([]byte(s), &buf); err != nil {
		return "", false
	}
	return mdSane.Sanitize(buf.String()), true
}

func renderMarkdown(s string) g.Node {
	out, ok := renderMarkdownString(s)
	if !ok {
		return g.Text(s) // unchanged from the original error fallback
	}
	return g.Raw(out)
}
```

`RenderMarkdownLinked` (above) must adapt to this two-value return: change
`base := renderMarkdownString(s)` to `base, ok := renderMarkdownString(s)`, and on
`!ok` return `g.Text(s)` too (a convert error has no `[[links]]` to substitute —
degrade to the same escaped plain text). Add a test that a goldmark-convert-error
input still escapes (see Test plan `TestRenderMarkdownLinkedConvertErrorEscapes`).

Add imports `"regexp"` and `"strings"` to `markdown.go` (and `"html"`, used by
`RenderMarkdownLinked` for the display-text and id escaping).

> **Why the post-pass substitution is sound (and what NOT to assert)** — two
> facts make it work, both verified empirically against goldmark + bluemonday
> `UGCPolicy()` (the live `md`/`mdSane`):
>   1. **`[[Foo]]` survives goldmark+bluemonday literally.**
>      `renderMarkdownString("[[Foo]]")` returns `"<p>[[Foo]]</p>\n"` — goldmark
>      has no wikilink syntax, and bluemonday escapes inner text but leaves the
>      `[[ ]]` brackets. So `linkChipRe` still matches in the sanitized string.
>   2. **A RELATIVE href survives sanitize.** `mdSane.Sanitize` keeps
>      `href="/ui/show/note?id=abc123"` (UGCPolicy calls `AllowRelativeURLs(true)`)
>      — it adds `rel="nofollow"` but does NOT drop the href.
>
> **Do NOT assert that `class="wikilink"` survives `mdSane.Sanitize` — it does
> not.** Sanitizing `<a class="wikilink" href="/ui/show/note?id=abc">Foo</a>`
> yields `<a href="/ui/show/note?id=abc" rel="nofollow">Foo</a>` (UGCPolicy never
> calls `AllowStyling()`, so `class` is not in its allowlist). That is irrelevant
> here: **`RenderMarkdownLinked` injects the chip into the ALREADY-sanitized
> string and returns `g.Raw(out)` WITHOUT re-sanitizing**, so the chip's `class`
> is intentionally preserved — it never passes through bluemonday. The chip is
> trusted output we construct ourselves (fixed href shape + HTML-escaped display
> text), not model text. So the only empirical assertions Step 7 makes are (1) and
> (2) above. If EITHER fails — `renderMarkdownString("[[Foo]]")` does not contain
> literal `[[Foo]]`, OR the relative href is stripped by `mdSane.Sanitize` — STOP
> and report; the substitution strategy is then invalid and needs a goldmark AST
> extension (out of this plan's Pareto slice). Do NOT STOP on `class` being
> stripped by a direct sanitize call — that is expected and not on the real path.

**DESIGN CONFLICT — read before wiring the body render (the advisor must resolve
this).** The shipped note card does NOT render its body through markdown at all:
`NoteCard` renders the body as escaped plain text — `g.If(v.Body != "",
h.Div(h.Class("kcard-body"), g.Text(v.Body)))` (`internal/feature/knowledgecards/note.go:46`).
Its package doc is explicit and deliberate (`note.go:3-9`): *"Body is rendered as
escaped text … but knowledgecards must not import goldmark to stay within the
layering law, so the markdown-render pass is deferred to the chat bubble path."*
So there is **no existing `renderMarkdown`/`chat.MessageBody` call in the card to
"switch"** — Step 5's "switch that ONE call to `RenderMarkdownLinked`" assumed a
markdown render that 160 deliberately did NOT add. And making `knowledgecards`
call `chat.RenderMarkdownLinked` means `knowledgecards` would import
`internal/ui/chat` (it currently imports only `internal/ui`,
`internal/feature`, `pocketbase/core`, and gomponents — `note.go:11-18`,
`register.go:1-7`). That is not a compile cycle (`internal/ui/chat` imports no
balaur package — it is a leaf UI package), but it DOES contradict the note card's
stated layering choice to keep goldmark out of knowledgecards.

**Do NOT silently wire `chat.RenderMarkdownLinked` into the note card.** Options
for the advisor to pick from (do not choose unilaterally):
  (a) **Scope-trim**: keep the note body as escaped `g.Text` (160's choice) and
      ship wikilink chips ONLY where markdown already renders (the chat bubble
      path via `RenderMarkdownLinked`), plus the "Linked from" backlinks panel on
      the card (Step 6 — that panel is pure gomponents, no markdown, no new
      import, so it is unaffected). Clickable `[[links]]` inside the note card
      body then become a follow-up once the card has a markdown render seam.
  (b) **Add the import deliberately**: have `knowledgecards` import
      `internal/ui/chat` and render the body through `RenderMarkdownLinked`,
      explicitly revising note.go's "no goldmark in knowledgecards" doc comment —
      a real layering decision the advisor should bless, not the executor.
  (c) **Move the renderer**: relocate `RenderMarkdownLinked` to `internal/ui`
      (which knowledgecards already imports) — but that drags goldmark/bluemonday
      into the `ui` atom package, a bigger layering change.

If the advisor picks (a) or (b), the resolver shape is unchanged: it closes over
the app and resolves a title to an active node id (the `resolveOrCreateStub` read
half, without the create — just `FindFirstRecordByFilter("nodes", "status =
'active' && title = {:title}", …)` returning `(rec.Id, true)` on hit, `("", false)`
on miss; the chip needs only the id because `/ui/show/note?id=<id>` derives the
type from the record). Keep any body-render change to a single call — do not
restructure the card.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0;
`gofmt -l internal/ui/chat` → no output.

### Step 6: Add the "Linked from" backlinks panel to the node card

160 owns the node card, which lives in `internal/feature/knowledgecards/note.go`
(the `note` renderer registered via `ui.RegisterCard("note", …)` in `registerNote`
at `note.go:77-81`; `Register` calls it at `register.go:21`). The card is
`NoteCard(v NoteView)` (`note.go:32-57`); the loader is `buildNote(app, params)`
(`note.go:60-73`). **There is NO backlinks slot in `NoteView`/`NoteCard` yet** —
160 did not add one (it noted only that this is "the first /ui/show/note surface
the route plans 161/163 build on", `note.go:5`). So you ADD the panel, not fill a
slot. **Put `LinkedFrom` in this `knowledgecards` package**, NOT in
`internal/ui/chat` or `internal/ui`: `LinkedFrom` takes `[]*core.Record`, which
means importing `pocketbase/core`; `internal/ui/chat` imports ONLY `internal/ui`
and does NOT import `pocketbase/core` (verified — it imports no balaur package at
all), so putting a `[]*core.Record` helper there would introduce a forbidden new
UI-atom→`core` dependency. The `knowledgecards` package already imports
`pocketbase/core` (`note.go:12`) and `internal/ui` (`note.go:17`), so it is the
correct home (it's where 160's note card already reads records via `buildNote`).

Each chip anchor is wrapped in its own `<li>` inline — there is no `wrapLis`
helper anywhere in the repo; do NOT call one. The chip title renders through
escaping `g.Text` (no XSS) and the `wikilink` class via `h.Class` — rendered
directly as a gomponent, NOT through bluemonday (this is our own trusted markup,
not model text). Minimal shape (in `internal/feature/knowledgecards`, with the
package's existing `h "maragu.dev/gomponents/html"` and `g "maragu.dev/gomponents"`
aliases):

```go
// LinkedFrom renders the backlinks panel: a "Linked from" section listing the
// nodes that wikilink to this node. Empty list renders nothing (no empty box).
// The argument is named `backlinks` (not `nodes`) so it never reads as a
// reference to the `nodes` domain package.
func LinkedFrom(backlinks []*core.Record) g.Node {
	if len(backlinks) == 0 {
		return nil
	}
	var items []g.Node
	for _, n := range backlinks {
		items = append(items, h.Li(h.A(
			h.Class("wikilink"),
			h.Href("/ui/show/note?id="+n.Id),
			g.Text(n.GetString("title")), // escaping path — no XSS
		)))
	}
	return h.Section(h.Class("node-backlinks"),
		h.H3(g.Text("Linked from")),
		h.Ul(g.Group(items)),
	)
}
```

**Wiring it in (note the real shapes):** the card is `NoteCard(v NoteView)` and the
loader is `buildNote(app, params) NoteView` (`note.go:32,60`); `NoteView` has no
record field and `NoteCard` takes no `core.App`, so the backlinks list has to be
threaded in. Two minimal options (KISS — pick the smaller):
  - add a `Backlinks []*core.Record` field to `NoteView`, have `buildNote` set it
    via `nodes.Backlinks(app, rec.Id)` (160's status=active-filtered helper —
    `knowledgecards` imports `internal/nodes`; no cycle), and have `NoteCard` emit
    `LinkedFrom(v.Backlinks)` after the body `Div`; or
  - keep `NoteView`/`NoteCard` untouched and render the panel in the
    `registerNote` closure: `g.Group([]g.Node{NoteCard(buildNote(app, params)),
    LinkedFrom(nodes.Backlinks-result)})` (the closure already has `app` and
    `params`). Either way, **`knowledgecards` gains an `internal/nodes` import** —
    that is allowed (nodes is a domain package; chat/ui stay out of it). **Audit
    nothing here** — this is a read.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 7: Tests — parser, resolution/stub, backlinks, render

Create `internal/nodes/links_test.go` and `internal/ui/chat/markdown_test.go`.
See Test plan for the case list. Then extend the storybook node story (Step 8).

**Verify**: `go test ./internal/nodes/... ./internal/ui/chat/...` → all pass.

### Step 8: Extend the storybook node story with a backlinks variant

160 SHIPPED the node story: `notecardStory()` at
`internal/feature/storybook/stories_cards.go:131-156`, registered in the `stories`
slice at `story.go:94`. **Extend `notecardStory()`** — add a `Variant` showing the
node card with a populated "Linked from" panel (a fixture of 2-3 backlink nodes).
Its current variants build directly from `knowledgecards.NoteCard(NoteView{…})`
(lines 136-138), so the new variant wraps a `NoteCard` plus the `LinkedFrom` panel:
`g.Group([]g.Node{knowledgecards.NoteCard(NoteView{…}), knowledgecards.LinkedFrom(fixtureBacklinks)})`.
(`stories_cards.go` already imports `g "maragu.dev/gomponents"` and
`knowledgecards`.)

> **The backlinks fixture MUST be non-empty AND is a `[]*core.Record`.**
> `LinkedFrom` returns `nil` on an empty list (renders nothing), and
> `TestAllStoriesRender` (`story_test.go:35-46`) asserts every variant renders
> **non-empty** — an empty fixture would render nil and fail the suite. But
> `LinkedFrom` takes `[]*core.Record`, and a `*core.Record` cannot be built from a
> plain struct literal — `core.NewRecord(collection)` needs a real collection (an
> app). `stories_cards.go` is a pure render file (no app). **Resolve this before
> coding** (advisor note): either (i) give `LinkedFrom` a tiny view-model input
> (e.g. `LinkedFrom(items []BacklinkView)` where `BacklinkView{ID, Title string}`)
> so the card layer never depends on `*core.Record` and the story can pass plain
> structs — cleaner layering, and it drops the `pocketbase/core` import from the
> panel signature; the `buildNote`/`registerNote` wiring maps
> `nodes.Backlinks(...)` records to `[]BacklinkView`; or (ii) keep the
> `[]*core.Record` signature and add the backlinks variant in a story file that
> can mint records via `storetest` (heavier; storybook stories are normally
> app-free). **Prefer (i)** — it keeps storybook fixtures plain and avoids a
> `core.Record` dependency leaking into the story. If (i) is chosen, update the
> `LinkedFrom` code in Step 6 to take `[]BacklinkView` and have the card handler do
> the record→view mapping. Use 2-3 fixture nodes either way.

**Verify**: `go test ./internal/feature/storybook/...` → all pass
(`TestAllStoriesRender` renders the new variant non-empty).

### Step 9: Update `internal/self/knowledge.md`

160 already rewrote `internal/self/knowledge.md` onto the unified spine: the
"Data lives in PocketBase collections" paragraph (`knowledge.md:88-98`) now
describes `nodes`/`edges`, status-based consent, and active-only traversal; the
Knowledge capabilities bullet (`knowledge.md:109-115`) ends with the note card
line ("The note card (/ui/show/note?id=…) renders a node's title + body with an
inline edit form."). Add one or two sentences to ONE of those spots stating that
node bodies support `[[wikilinks]]`, which create node→node `links` edges on save
(resolving by title or creating a stub node), and that the node card shows
backlinks ("Linked from"). The cleanest seam: extend the existing `edges` sentence
in the Data paragraph (~`knowledge.md:90-94`) or append to the note-card line at
`knowledge.md:115`. Keep it terse and accurate — a stale self-description makes
Balaur lie about
itself.

**Verify**: `grep -n "wikilink\|Linked from\|backlink" internal/self/knowledge.md`
→ at least one match.

### Step 10: Full gate + graph refresh

Run the full suite and the gate set; refresh the graph.

**Verify**:
- `gofmt -l internal/nodes internal/ui/chat main.go internal/feature internal/self` → no output
- `go vet ./...` → exit 0
- `env -u BALAUR_OS_ACCESS -u BALAUR_SOURCE -u BALAUR_MAX_STEPS go test ./...` → all pass
- `CGO_ENABLED=0 go build ./...` → exit 0
- `git diff --check` → no output
- `graphify update .` → completes (minify graph.json before committing)

## Test plan

New tests:

- `internal/nodes/links_test.go` (package `nodes` — in-package, so it calls
  `SyncLinks`/`ParseLinks`/`Backlinks`/`Outbound` bare, no package qualifier):
  - **`TestParseLinks`** (table-driven, pure — no app): cases:
    - empty body → nil; `"[[]]"` and `"[[   ]]"` → nil (empty target skipped);
    - `"[[Alpha]]"` → `["Alpha"]`;
    - alias `"[[Alpha|the first]]"` → `["Alpha"]` (alias ignored for resolution);
    - adjacent `"[[a]][[b]]"` → `["a","b"]`;
    - duplicates `"[[a]] and [[a]] again"` → `["a"]` (deduped);
    - case-insensitive dup `"[[Alpha]] [[alpha]]"` → `["Alpha"]` (first-seen wins);
    - unicode `"[[Café]]"` → `["Café"]`;
    - bracket inside is not a target `"[[a]b]]"` — assert the actual behavior you
      observe (document it in the test); the `[^\[\]|]` class forbids `]` inside.
  - **`TestSyncLinksResolveAndStub`** (temp-dir app via `storetest.NewApp(t)`):
    create an active node "Target"; create a source node whose body is
    `"see [[Target]] and [[Ghost]]"`; call `SyncLinks(app, source)`; assert
    two `type="links"` edges exist from source; assert "Ghost" now exists as an
    active `type=note` stub node; assert the edge to it points at the stub's id.
  - **`TestSyncLinksIdempotent`**: call `SyncLinks` twice with the same body;
    assert edge count is unchanged (no duplicates) and no extra stub created.
  - **`TestSyncLinksRewrites`**: sync body `"[[A]]"`, then change body to
    `"[[B]]"` and re-sync; assert the A-edge is gone and a B-edge exists.
  - **`TestSyncLinksNoSelfEdge`**: a node titled "Self" with body `"[[Self]]"` →
    zero edges (self-links dropped).
  - **`TestBacklinksAndOutbound`**: nodes X, Y; X body `"[[Y]]"`; sync; assert
    `Backlinks(app, Yid)` returns X and `Outbound(app, Xid)` returns Y. (These are
    160's helpers — this test exercises them through 161's `SyncLinks` to confirm
    the edge writes are visible to the traversal the node card reads.)
  - **`TestProposedNodesNeverResolve`** (consent): create a `status=proposed` node
    titled "Hidden"; sync a source body `"[[Hidden]]"`; assert resolution did NOT
    pick the proposed node (a NEW active stub "Hidden" was created instead) — the
    proposed node stays out of the graph.
- `internal/ui/chat/markdown_test.go` (package `chat_test`, model after
  `message_test.go`):
  - **`TestRenderMarkdownStringPassesWikilink`**: assert
    `chat`-rendered `[[Foo]]` survives as literal `[[Foo]]` through the base
    pipeline (the empirical check #1 from Step 5). Since `renderMarkdownString` is
    unexported, assert via the public path: `RenderMarkdownLinked("[[Foo]]", nil)`
    with a resolver returning `ok=false` produces the unresolved span, proving the
    `[[Foo]]` token reached the substitution intact. (Optionally also assert the
    relative href survives: a resolver returning `("x","true")` yields output
    containing `href="/ui/show/note?id=x"` — empirical check #2.)
  - **`TestRenderMarkdownLinkedResolved`**: resolver returns
    `("abc123", true)` for "Foo"; assert output contains
    `<a class="wikilink" href="/ui/show/note?id=abc123">Foo</a>` and does NOT
    contain `[[Foo]]`. (The `class="wikilink"` IS present here even though
    bluemonday strips `class` from anchors — because `RenderMarkdownLinked`
    injects this chip into the already-sanitized string and returns `g.Raw`
    without re-sanitizing. This asserts the post-sanitize injection, not that
    sanitize preserves `class`.)
  - **`TestRenderMarkdownLinkedAlias`**: `"[[Foo|the foo]]"` → chip display text is
    `the foo`, href still resolves "Foo".
  - **`TestRenderMarkdownLinkedUnresolved`**: resolver `ok=false` → output contains
    `wikilink-unresolved` span, no anchor.
  - **`TestRenderMarkdownLinkedNoInjection`**: body
    `"[[<script>alert(1)</script>]]"` resolver ok=false → output contains no raw
    `<script>` (escaped), proving the display text is HTML-escaped.
  - **`TestRenderMarkdownLinkedConvertErrorEscapes`** (error-fallback contract):
    feed an input that makes `md.Convert` error (or assert the documented behavior
    that on a convert error `RenderMarkdownLinked` returns `g.Text(s)` — escaped
    plain text, same as `renderMarkdown`); assert the output escapes `<>&` and
    contains no raw `<script>`. This proves the refactor preserved the original
    `g.Text(s)` error fallback rather than switching to raw HTML. (If you cannot
    force a goldmark convert error, the existing `TestMessageBalaurMarkdown` in
    `message_test.go` already covers the unchanged plain-chat success path; cite it
    and skip the forced-error case.)

Existing tests that must stay green **unchanged** (behavior contract):
- `internal/ui/chat/message_test.go` — `TestMessageBalaurMarkdown`,
  `TestMessageBalaur`, etc. (plain chat must not gain wikilink chips).
- `internal/feature/storybook/story_test.go` — `TestAllStoriesRender`,
  `TestStoriesUniqueAndLookup`.

Verification: `go test ./internal/nodes/... ./internal/ui/chat/... ./internal/feature/...`
→ all pass, including the new tests; then `go test ./...` → all pass.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `gofmt -l internal/nodes internal/ui/chat main.go internal/feature internal/self` produces no output
- [ ] `env -u BALAUR_OS_ACCESS -u BALAUR_SOURCE -u BALAUR_MAX_STEPS go test ./...` exits 0; new `TestParseLinks`, `TestSyncLinksResolveAndStub`, `TestBacklinksAndOutbound`, `TestProposedNodesNeverResolve`, and `TestRenderMarkdownLinkedResolved` exist and pass
- [ ] `git diff --check` produces no output
- [ ] `grep -n "registerGraphLinks" main.go` returns the definition AND the `OnServe` call (this plan's link-sync hook). Note: `OnRecordAfterCreateSuccess("nodes")` may now appear MORE than once in `main.go` (160 and/or 162 also bind `"nodes"` hooks); the link-sync binding is the one inside `registerGraphLinks`, and the hooks compose (each calls `e.Next()`)
- [ ] `grep -rn "ui/notes/" internal/` returns no matches (the wrong route was never invented)
- [ ] `grep -rn "/ui/show/node?" internal/` returns no matches (chips use `/ui/show/note`, the real 160 route — `/ui/show/node` has no card and 404s)
- [ ] `grep -rn "/ui/show/note?id=" internal/ui/chat internal/feature` returns matches (the chip + backlinks hrefs use the real 160 route)
- [ ] `grep -rn "wrapLis" internal/` returns no matches (no undefined helper — `<li>` wrapping is inline)
- [ ] `grep -rn "func Backlinks\|func Outbound" internal/nodes/links.go` returns NOTHING — 161 does NOT redeclare 160's traversal helpers (they stay in 160's `nodes.go`); `links.go` adds only `ParseLinks`/`resolveOrCreateStub`/`SyncLinks`
- [ ] `grep -rn "app.Save(edge\|core.NewRecord(.*edges" internal/nodes/links.go` returns NOTHING — `SyncLinks` writes edges through 160's idempotent `AddEdge`, never a raw edge save (which would bypass the unique-index idempotency)
- [ ] `internal/graph/` does NOT exist (`test ! -d internal/graph`) — 161 lands inside 160's `internal/nodes`, not a parallel package
- [ ] `internal/search/index.go` is unchanged (`git diff --stat` does not list it — 162 owns search)
- [ ] `migrations/` is unchanged (`git diff --stat` does not list it — 160 owns the schema)
- [ ] `grep -n "wikilink\|Linked from\|backlink" internal/self/knowledge.md` returns at least one match
- [ ] Storybook node story renders the backlinks variant (`go test ./internal/feature/storybook/...` passes)
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row for 161 updated

## STOP conditions

Stop and report back (do not improvise) if:

- The `nodes`/`edges` collections do not exist (a temp-app test fails with
  "missing collection"). 161 hard-depends on 160 — which IS merged at HEAD
  `6ab038a`, so this should not fire; if it does, the checkout is stale.
- 160's relation field names are NOT `source`/`target`, or the node route is NOT
  `GET /ui/show/note?id=...`, or 160's `note` card REQUIRES a `type` param to
  render non-note nodes. **VERIFIED post-160: they ARE `source`/`target`
  (`nodes.go:172-173`), the route IS `/ui/show/note?id=...`, and the `note` card
  does NOT require a `type` param (`note.go:60-72`)** — so this condition is
  already satisfied and should not fire. (Kept in case a later change drifts it.)
- The empirical render check fails: `renderMarkdownString("[[Foo]]")` does NOT
  contain literal `[[Foo]]`, OR a relative `href="/ui/show/note?id=x"` anchor is
  stripped by `mdSane.Sanitize`. The post-pass substitution strategy is then
  invalid and needs a goldmark AST extension — out of this plan's slice. (Do NOT
  STOP merely because `mdSane.Sanitize` strips `class="wikilink"` from an anchor
  fed through it directly — that is expected; the chip is injected post-sanitize
  and never re-sanitized, so its `class` is preserved by design.)
- The save hook recurses infinitely (stub creation re-fires the hook with a
  non-empty body) — means 160's stub assumption (empty body) is wrong.
- A step's verification fails twice after a reasonable fix attempt.
- The fix appears to require touching `internal/search/*`, `migrations/*`, or the
  graph view (163) — all out of scope.

## Maintenance notes

For the human/agent who owns this code after the change lands:

- **Deferred, by design (name in any roadmap, do NOT build here)**: block refs
  `((...))`, LogSeq-style queries, an interactive/force-directed graph view (plan
  163), unified FTS indexing of nodes (plan 162 — nodes are NOT added to the FTS
  index in this plan), typed relations (only `type="links"` exists), and
  note↔task cross-layer links (edges stay node↔node in v1).
- **Edge sync is full-replace per source** (delete all `type="links"` edges, then
  re-insert). Simple and correct while link counts per node are small. If a node
  ever holds hundreds of links and this shows up on a profile, switch to a diff
  (insert missing / delete removed) — but only with a measurement in hand.
- **Stubs are born active.** Revisit if the consent model changes so that
  agent-created stubs should be `proposed` — today the stub is a side effect of
  the *owner's* trusted save, so active is correct.
- **The render regexp `linkChipRe` is duplicated** from `internal/nodes`'s
  `wikilinkRe` to keep `internal/ui/chat` from importing the domain package
  `internal/nodes` — chat is a UI atom package and must not depend on a domain
  package (the package-boundary law; chat currently imports only `internal/ui` and
  does NOT import `pocketbase/core`). A reviewer should check the two regexps stay
  in lockstep; if a third consumer appears, the right move is a tiny shared regexp
  package both can import (NOT making chat import `internal/nodes`).
- **161 reuses 160's `nodes.Backlinks`/`Outbound`/`AddEdge`** — it does not
  redeclare them. The wikilink layer (`ParseLinks`, `resolveOrCreateStub`,
  `SyncLinks`) lives in `internal/nodes/links.go` alongside 160's helpers; edge
  writes go through `AddEdge` so the `(source,target,type)` unique-index
  idempotency holds. If a reviewer sees a second `Backlinks`/`Outbound` or a raw
  `app.Save(edge)` in `links.go`, that is the duplication this plan exists to
  avoid — remove it.
- **Reviewer scrutiny**: (1) the empirical render test proves bluemonday keeps the
  RELATIVE href (`/ui/show/note?id=…`) — without it the chips silently vanish;
  the chip's `class="wikilink"` is preserved NOT by sanitize (UGCPolicy strips
  `class`) but because the chip is injected POST-sanitize and the result is
  returned via `g.Raw` without re-sanitizing; (2) the consent filter
  (`status='active'`) is present in `resolveOrCreateStub` and in 160's
  `Backlinks`/`Outbound` — proposed nodes must never enter the graph; (3) the
  link-sync hook logs and continues on error (a bad parse must never block the
  owner's save); (4) the display text in chips is HTML-escaped (no XSS via a
  `[[<script>]]` title).
- **Cascade delete** (160) cleans a deleted node's edges, which is why there is no
  delete hook here. CONFIRMED: both `source` and `target` carry
  `CascadeDelete: true` (`migrations/1749600000_init.go:110-111`), so a deleted
  node's edges are removed by PocketBase automatically — no delete hook needed.
