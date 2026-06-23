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
> **Drift check (run first)**: `git diff --stat 72fd762..HEAD -- internal/nodes internal/feature/knowledgecards internal/ui/chat/markdown.go internal/ui/chat/message.go main.go internal/feature/storybook/stories_cards.go internal/self/knowledge.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: MED
- **Depends on**: plans/160-nodes-edges-spine.md (the `nodes` + `edges`
  collections, the `note` card type, and the `GET /ui/show/{type}` node route
  it registers). **161 must NOT be executed before 160 is merged** — it reads
  collections and a route that only 160 creates.
- **Category**: direction
- **Planned at**: commit `72fd762`, 2026-06-23

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

### What plan 160 provides (DO NOT redeclare — read 160's plan/diff to confirm names)

160 declares the `nodes` and `edges` collections and the `note` card. The shared
contract every plan in this set agrees on:

- **`nodes`** collection fields: `type` (select), `title` (text, required),
  `body` (text, markdown), `status` (select: `active|proposed|archived|rejected`),
  `props` (json), `created`/`updated` (autodate).
- **`edges`** collection fields: `source` (relation→nodes, single, cascade),
  `target` (relation→nodes, single, cascade), `type` (text, default `"links"`),
  `context` (text, optional). Unique index on `(source,target,type)`; index on
  `target` for backlinks. **Back-relation expand names are `edges_via_source`
  (outbound) and `edges_via_target` (backlinks).**
- **The node route** is the generic `GET /ui/show/{type}?id=...` dispatcher (the
  existing card system — see `internal/web/show.go` below). 160 registers a
  `note` card type whose URL form is `GET /ui/show/note?id=<nodeId>`.
- **`internal/nodes` is 160's domain package — REUSE it, do NOT redeclare.** 160
  Step 3 (`plans/160-nodes-edges-spine.md:537-564`) creates `internal/nodes`,
  which ALREADY owns, all `status=active`-filtered:
  - `nodes.AddEdge(app core.App, sourceID, targetID, edgeType, context string) (*core.Record, error)`
    — defaults `edgeType` to `"links"` when empty; **idempotent against the
    `(source,target,type)` unique index** (on a unique-constraint hit it
    find-and-returns the existing edge — see 160 line 554-557, 844).
  - `nodes.Backlinks(app core.App, id string) ([]*core.Record, error)` — inbound
    active nodes (via `edges_via_target`).
  - `nodes.Outbound(app core.App, id string) ([]*core.Record, error)` — outbound
    active nodes (via `edges_via_source`).
  - `nodes.Neighborhood(app core.App, id string) ([]*core.Record, error)` — the
    1-hop set.
  - Status constants `nodes.StatusActive`/`StatusProposed`/`StatusArchived`/
    `StatusRejected` and `nodes.Create(app, type, title, body, status, props)`.

  **161 MUST import `internal/nodes` and call these — it must NOT redeclare
  `Backlinks`/`Outbound`/`AddEdge` or the status constants** (two sources of
  truth violates SUCKLESS and the ownership boundary; raw `app.Save(edge)`
  bypasses 160's unique-index idempotency). 161's *new* code (the `[[ ]]`
  parser, the resolve-or-create-stub step, the `SyncLinks` body→edge sync, and
  the save hook) lands **inside `internal/nodes`** alongside 160's helpers — same
  package, one source of truth. See the "FTS index hook" mirror below for the
  same "160 provides X, reuse it" posture. If 160's helper signatures differ from
  the shapes above, **160 wins** — use 160's real signatures and treat a
  mechanical-rename-impossible mismatch as a STOP condition.
- **`/ui/show/note?id=<id>` is the generic node viewer.** Per 160's Step 8
  (`plans/160-nodes-edges-spine.md:680-714`), the `note` card spec takes an `id`
  param and its renderer derives the node kind from `rec.GetString("type")` (160
  line 695-696: "the kind now derives from `rec.GetString("type")`"). So
  `/ui/show/note?id=<id>` renders a node of **any** type (note, memory, skill,
  person, book, journal, …) by id — the executor does NOT pass a `type` param.
  **Every wikilink chip and every backlink chip in this plan therefore uses the
  href `/ui/show/note?id=<id>` regardless of the target node's type.** Never emit
  `/ui/show/node?id=...` (there is no `node` card — it 404s) and never invent
  `/ui/notes/{id}`.

> **If 160's actual field names differ** (e.g. it had to name the relation fields
> something other than `source`/`target` for a PocketBase constraint), 160's plan
> says so and **160 wins** — use 160's real names everywhere in this plan. Treat a
> name mismatch as a STOP condition and report it; do not guess.
>
> **If 160's `note` card actually REQUIRES a `type` param** (i.e.
> `/ui/show/note?id=<id>` without `type` does not render a non-note node), STOP
> and report: the chips must then pass `&type=<rec.type>` and the resolver/helper
> must carry each node's type. Do not guess — confirm against 160's real route
> registration before changing the href shape.

### The FTS index hook — the EXACT pattern to mirror for the link-sync hook

`main.go:202-256` — `registerSearchIndex` opens the sidecar index and binds
record hooks. The hook-registration tail (verbatim, `main.go:232-256`):

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

From `internal/knowledge/knowledge.go:34-40` (statuses) and the LOCKED
architecture: graph traversal and resolution **must filter to `status=active`**.
Resolve-by-title and backlinks queries must only see and create active nodes;
they must never surface `proposed`/`rejected` nodes. The resolve-by-title
pattern to copy is `LoadSkill` (`internal/knowledge/knowledge.go:302-311`):

```go
	rec, err := app.FindFirstRecordByFilter(string(Skill),
		"status = 'active' && name = {:name}",
		dbx.Params{"name": name})
```

Use `app.FindFirstRecordByFilter("nodes", "status = 'active' && title = {:title}", dbx.Params{"title": title})`.

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
every registered story — your new/extended story must render non-empty. The node
card story belongs to 160; **161 EXTENDS that story** (adds a "with backlinks"
variant) if 160 created it, or adds the backlinks panel as a documented variant.
Stories are registered in the `stories` slice in `story.go:53-111`.

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
  160's node (`note`) renderer (160's note card lives here — it's a feature card
  package that already imports `pocketbase/core`; see Step 6). Fill 160's
  backlinks slot if present, else add a minimal "Linked from" section.
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
  component.** If 160 already added a backlinks slot, fill it; if not, add a
  minimal "Linked from" section via the helper, but do not restructure the card.
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

There is **no new traversal code in 161**. `nodes.Backlinks(app, id)` and
`nodes.Outbound(app, id)` already exist in 160's `internal/nodes` package
(`plans/160-nodes-edges-spine.md:558-562`), both filtered to `status=active`. The
node card (Step 6) and the tests (Step 7) call those directly. **Do NOT add a
second `Backlinks`/`Outbound` to `links.go`** — that is the duplication this plan
was repaired to remove.

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
`registerSearchIndex(se.App)` (the block at `main.go:48-58`). Define
`registerGraphLinks` as a sibling function (place it near `registerSearchIndex`,
~`main.go:256`):

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

The `nodes` import (`"github.com/alexradunet/balaur/internal/nodes"`) is almost
certainly already in `main.go` after 160 (160 wires `internal/nodes` tools/CLI).
If `grep -n 'internal/nodes' main.go` shows it, do not re-add it; if not, add it.

> **Hook-collision note**: 160 may already bind hooks to `"nodes"` (and plan 162
> binds the FTS upsert/delete to `"nodes"`). Multiple `BindFunc` on the same
> `OnRecordAfter*Success("nodes")` event compose — each runs and calls `e.Next()`
> — so adding the link-sync hook does NOT replace 160's/162's. Keep the
> `e.Next()` tail. The done-criteria `grep` below expects exactly ONE
> link-sync binding from THIS plan, not one binding total on `"nodes"`.

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

160's node card render must call `chat.RenderMarkdownLinked(body, resolver)`
instead of plain `renderMarkdown`/`g.Text` for the node body, where `resolver`
closes over the app and resolves a title to an active node id (the
`resolveOrCreateStub` read half, without the create — just
`FindFirstRecordByFilter("nodes", "status = 'active' && title = {:title}", …)`
returning `(rec.Id, true)` on hit, `("", false)` on miss; the chip needs only the
id because `/ui/show/note?id=<id>` derives the type from the record). If 160's card
already renders the body through `chat.MessageBody`/`renderMarkdown`, switch that
ONE call to `RenderMarkdownLinked`; otherwise add the call in the card body slot.
Keep the change to a single render call — do not restructure the card.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0;
`gofmt -l internal/ui/chat` → no output.

### Step 6: Add the "Linked from" backlinks panel to the node card

160 owns the node card, which lives in `internal/feature/knowledgecards/*` (the
`note` renderer registered via `ui.RegisterCard("note", …)` — see
`plans/160-nodes-edges-spine.md:728-737`). **Put `LinkedFrom` in that feature
card package**, NOT in `internal/ui/chat` or `internal/ui`: `LinkedFrom` takes
`[]*core.Record`, which means importing `pocketbase/core`; `internal/ui/chat`
imports ONLY `internal/ui` and does NOT import `pocketbase/core` (verified), so
putting a `[]*core.Record` helper there would introduce a forbidden new
UI-atom→`core` dependency. The `knowledgecards` feature package already imports
`pocketbase/core`, so it is the correct home (it's where 160's node card already
reads records). Fill 160's backlinks slot if present.

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

The card handler fetches the list with `nodes.Backlinks(app, nodeId)` (160's
status=active-filtered helper — import `internal/nodes`) and passes it to
`LinkedFrom`. **Audit nothing here** — this is a read.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 7: Tests — parser, resolution/stub, backlinks, render

Create `internal/nodes/links_test.go` and `internal/ui/chat/markdown_test.go`.
See Test plan for the case list. Then extend the storybook node story (Step 8).

**Verify**: `go test ./internal/nodes/... ./internal/ui/chat/...` → all pass.

### Step 8: Extend the storybook node story with a backlinks variant

In `internal/feature/storybook/stories_cards.go`, find the node story 160 added
(grep for the node card story builder) and add a `Variant` showing the node card
with a populated "Linked from" panel (a fixture of 2-3 backlink nodes), plus one
`[[wikilink]]` rendered as a chip in the body. If 160 did NOT add a node story,
add a small `wikilinkStory()` builder and register it in the `stories` slice
(`story.go:53-111`) demonstrating: a resolved chip, an unresolved chip, and a
"Linked from" list.

> **The backlinks fixture MUST be non-empty.** `LinkedFrom` returns `nil` on an
> empty list (renders nothing), and `TestAllStoriesRender`
> (`story_test.go:35-46`) asserts every variant renders **non-empty** — an empty
> backlinks fixture would render nil and fail the suite. Use 2-3 fixture nodes.

**Verify**: `go test ./internal/feature/storybook/...` → all pass
(`TestAllStoriesRender` renders the new variant non-empty).

### Step 9: Update `internal/self/knowledge.md`

Add one or two sentences to the knowledge-layer description (around the existing
search/knowledge lines, `knowledge.md:76-104`) stating that node bodies support
`[[wikilinks]]`, which create node→node `links` edges on save (resolving by title
or creating a stub node), and that the node card shows backlinks ("Linked from").
Keep it terse and accurate — a stale self-description makes Balaur lie about
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

- 160 is not merged, or the `nodes`/`edges` collections do not exist (a temp-app
  test fails with "missing collection"). 161 hard-depends on 160.
- 160's relation field names are NOT `source`/`target`, or the node route is NOT
  `GET /ui/show/note?id=...`, or 160's `note` card REQUIRES a `type` param to
  render non-note nodes (so `/ui/show/note?id=<id>` alone does not render a memory/
  skill/person node). Use 160's real names/route — but if they conflict with this
  plan's helper code in a way you cannot mechanically rename, STOP and report.
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
  delete hook here. If 160 did NOT set cascade on the edge relations, add a
  delete hook that removes edges where `source` OR `target` equals the deleted id
  — and tell 160's owner the schema is missing cascade.
