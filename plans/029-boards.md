# Plan 029: Boards — owner-composed dashboards of typed cards (server-defined layout)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 9c77f42..HEAD -- internal/web web/templates migrations internal/cards web/templates/layout.html`
> Plans 025–028 being DONE is expected drift. Anything else in migrations/ or
> layout.html → compare excerpts; on mismatch, STOP.

## Status

- **Priority**: P2
- **Effort**: M–L
- **Risk**: MED (new collection + new page surface)
- **Depends on**: plans/028-typed-card-registry.md (hard)
- **Category**: direction
- **Planned at**: commit `9c77f42`, 2026-06-12

## Why this matters

The Hearthwood design's home mode is the **board**: a grid of cards the owner
(or, in plan 030, Balaur) composes — Study, Quest Log, Self, Balaur, plus
custom boards. With the typed card registry (plan 028) shipped, a board is
just a named, ordered list of `{type, params}` references; the page renders a
CSS grid of slots that each lazy-load their card via
`hx-get="/ui/cards/{type}?…"`. Layout is **server-defined** in v1: grid spans
come from each card type's default width — no drag, no resize, no client
layout state (owner decision, 2026-06-12; drag/resize is an explicit
follow-up).

## Current state

- **Migrations convention**: Go files in `migrations/` named
  `<unix-ts>_<slug>.go`, timestamps unique and strictly increasing
  (AGENTS.md hard rule; `migrations/timestamp_uniqueness_test.go` enforces).
  Highest existing: `1750730000_local_provider_kind.go`. **This plan claims
  `1750740000`.** Look at any recent migration (e.g.
  `migrations/1750720000_conversation_indexes.go`) for the
  `m.Register(up, down)` pattern, and at an older collection-creating
  migration for the collection + API-rules pattern (the `tasks` collection's
  migration is the closest exemplar — find it with
  `grep -l '"tasks"' migrations/*.go`).
- **Rule-boundary rule** (AGENTS.md): Go-side `app.Save`/`Find*` bypass
  collection API rules; that is fine here — board mutations come only from
  owner-facing web handlers behind `guardLocalUI`, same trust level as the
  existing `/ui/profile/*` and `/ui/tasks/*` handlers. Set the collection's
  REST API rules to owner-only anyway (copy the rule pattern the `tasks`
  migration uses), so heads' REST tokens cannot touch boards.
- **Topbar nav** — `web/templates/layout.html:23-30`:

  ```html
  <nav>
    <a href="/tasks">Tasks</a>
    <a href="/life">Life</a>
    <a href="/memory">Memory</a>
    <a href="/heads">Heads</a>
    <a href="/settings">Settings</a>
    …
  ```

- **Registry interface** (from plan 028): `internal/cards` exposes
  `All() []Spec`, `Get(typ)`, `Validate(typ, params)`; `Spec.W` is the
  default 12-col span; `GET /ui/cards/{type}?…` renders one card;
  `GET /ui/cards` renders the palette index.
- **Page idioms**: full pages embed `{{template "page_head" .}}` +
  `{{template "topbar" .}}` (see `web/templates/life.html` for a simple
  exemplar); fragments swap with `hx-target`/`hx-swap` and stable ids
  (`tcard-{id}` pattern); tabs use `.k-tabs`/`.k-tab`/`.k-tab-active`
  (see `web/templates/knowledge.html`).
- **Mockup reference** (for naming and defaults only — its drag engine and
  localStorage persistence are explicitly NOT ported): default boards in
  `Balaur App.dc.html:161-…` are Study, Quest Log, Self, Balaur; "compose a
  board from one ask" exists at lines 440-457 (that part lands in plan 030).

## Commands you will need

| Purpose   | Command                          | Expected on success |
|-----------|----------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`   | exit 0              |
| Tests     | `go test ./...`                  | ok (incl. migration timestamp test) |
| Vet/fmt   | `go vet ./...` / `gofmt -l .`    | clean               |

## Scope

**In scope**:
- `migrations/1750740000_boards.go` (+ its `_test.go`)
- `internal/web/boards.go`, `internal/web/boards_test.go` (create)
- `web/templates/boards.html` (create)
- `internal/web/web.go` (routes)
- `web/templates/layout.html` (nav link)
- `web/static/basm.css` (ONLY a small `## Boards` appendix section: the
  12-column grid + slot spans; ~25 lines)
- `internal/self/knowledge.md`, `DESIGN.md` honesty ledger

**Out of scope**:
- Drag/move/resize and any client-side layout persistence (explicit follow-up).
- Agent tools that create boards (plan 030).
- Making `/boards` the home route — `/` stays the chat.

## Git workflow

- Branch: `advisor/029-boards`
- Commit style: `feat(boards): owner-composed dashboards of typed cards`

## Steps

### Step 1: Migration — the `boards` collection

`migrations/1750740000_boards.go`: base collection `boards` with fields:
- `name` (text, required, max ~80)
- `cards` (json) — `[{"type":"quests","params":{"status":"open"}}, …]`
- `sort` (number) — board ordering in the tab strip

API rules: owner-only on all five rules (copy the exact rule expressions from
the `tasks` collection migration). Down migration deletes the collection.
Write the companion `_test.go` following the pattern of
`migrations/1750720000_conversation_indexes_test.go` (temp app, run
migrations, assert collection + fields exist).

**Verify**: `go test ./migrations/...` → ok (including
`timestamp_uniqueness_test.go`).

### Step 2: Board storage helpers + default boards

In `internal/web/boards.go` (boards are a web-surface concern; no other
package reads them — keep it in the gateway per the "domain packages own their
reads" rule, the domain here IS the web UI):

```go
type boardCard struct {
    Type   string            `json:"type"`
    Params map[string]string `json:"params,omitempty"`
}
```

`ensureDefaultBoards(app)`: if the collection is empty, create (sort order):
1. **Study** — today, quests(status=open,limit=8), calendar
2. **Quest log** — quests(status=open,limit=20), calendar
3. **Self** — journal(limit=5), timeline(days=14)
4. **Balaur** — memory(limit=6), skills(limit=6), heads

Called lazily from the boards page handler (never from a migration — defaults
are content, not schema; and never overwrite owner edits: only when count==0).
Every card entry must pass `cards.Validate` before save — one shared
`validateBoardCards([]boardCard) error` used by create/add handlers too.

**Verify**: unit test — fresh app, call handler, four boards exist; call
again, still four.

### Step 3: Routes + page

Register in `web.go`:

```
GET  /boards                        → redirect to first board (by sort)
GET  /boards/{id}                   → full page
POST /ui/boards                     → create (form: name) → redirect/render new board
POST /ui/boards/{id}/rename         → form: name → re-render board header frag
POST /ui/boards/{id}/delete         → guard: refuse to delete the last board → redirect /boards
POST /ui/boards/{id}/cards/add      → form: type + param fields → re-render board grid frag
POST /ui/boards/{id}/cards/{idx}/remove → re-render board grid frag
```

(`{idx}` = position in the cards array — boards are small; index addressing is
the KISS choice. Validate bounds.)

`web/templates/boards.html` — full page:
- Tab strip of boards (`.k-tabs`; active = `.k-tab-active`), a `+ board`
  ghost-button form.
- `board_grid` define (the HTMX re-render target, id `board-grid`):
  a `.board-grid` div (CSS: `display:grid; grid-template-columns:repeat(12,1fr); gap:16px`)
  where each card slot is

  ```html
  <div class="board-slot" style="grid-column: span {{.W}}"
       hx-get="/ui/cards/{{.Type}}{{.Query}}" hx-trigger="load" hx-swap="innerHTML">
    <div class="k-empty">…</div>
  </div>
  ```

  `.W` from `cards.Get(type).Spec.W` (8 or 4), `.Query` the URL-encoded params
  (build in Go, html/template will escape correctly in the attribute).
  Each slot gets a small `✕` remove form (`.btn-ghost .btn-sm`) posting to the
  remove route, and the grid ends with an "add a card" `<details>` fold listing
  the palette (`range` over `cards.All()`: label, icon, param inputs for each
  ParamSpec — text inputs are fine, `Doc` as placeholder).
- Board header: name + rename `<details>` fold + delete form (confirm via
  `hx-confirm` or a plain submit — match how destructive actions are done
  elsewhere; knowledge dismiss uses plain forms, do the same).
- Add `<a href="/boards">Boards</a>` to the topbar nav in layout.html,
  FIRST in the list (it is the mockup's primary mode).
- basm.css appendix: `.board-grid`, `.board-slot` (position relative, min
  height), `.board-slot > .ucard { height: 100% }`, responsive collapse to
  single column under 720px (match the existing media-query style at the
  bottom of basm.css).

**Verify**:
```sh
make run &
curl -sL http://127.0.0.1:8090/boards | grep -c "board-grid"   # ≥1
curl -s  http://127.0.0.1:8090/boards | head -1                 # redirect followed above; raw GET → 30x
```
plus handler tests (test plan).

### Step 4: Docs

- knowledge.md: boards capability paragraph (what they are, that Balaur can
  list types; plan 030 adds compose).
- DESIGN.md "True today": "boards — owner-composed dashboards of typed cards
  at /boards (server-defined layout; Study/Quest log/Self/Balaur defaults)".

**Verify**: `grep -n "boards" internal/self/knowledge.md DESIGN.md` → ≥1 each.

## Test plan

In `internal/web/boards_test.go`, model after `handlers_test.go` (temp-dir
app + HTTP harness):
- GET /boards with empty collection → defaults created, redirect to Study.
- GET /boards/{id} → 200, contains `board-grid` and one `hx-get="/ui/cards/`
  per card.
- POST create/rename/add/remove happy paths; add with an invalid type → 400 or
  error fragment (assert `cards.Validate` is enforced); remove with
  out-of-bounds idx → 400, board unchanged.
- Delete refuses when only one board remains.
- Migration test from Step 1.

Verification: `go test ./...` → all pass.

## Done criteria

- [ ] Migration `1750740000_boards.go` exists; timestamp test green
- [ ] Four default boards appear on first visit; idempotent
- [ ] Every mutation handler validates card entries via `cards.Validate`
      (`grep -c "Validate" internal/web/boards.go` → ≥2)
- [ ] Boards nav link present; `/boards` renders with slots lazy-loading cards
- [ ] `go test ./...`, vet, fmt, CGO-free build clean; `git diff --check` clean
- [ ] knowledge.md + DESIGN.md updated; `plans/readme.md` row updated

## STOP conditions

- A migration with timestamp ≥ `1750740000` already exists (someone claimed
  the number — take the next free one and note it in your report AND in
  `plans/readme.md`'s dependency notes).
- Plan 028's registry API differs from the interface described here.
- The `tasks` migration's rule pattern can't be located or doesn't translate
  to a base collection — report rather than inventing rule expressions.
- You feel the need for client-side JS beyond what HTMX attributes give you —
  that's the drag/resize follow-up trying to sneak in; stop at server-defined.

## Maintenance notes

- Follow-up (owner-deferred): drag/move/resize with layout persisted per
  board (add `x,y,w,h` to the cards JSON entries — schema already a json blob,
  so no migration needed; plus a vanilla-JS grid engine, ~200 lines — spec it
  as its own plan when wanted).
- Plan 030 writes board records from a tool: it must reuse
  `validateBoardCards` — if 030 lands first in a worktree, coordinate; the
  shared helper belongs wherever 030's tool can import it WITHOUT importing
  `internal/web` (move it into `internal/cards` as `ValidateCards([]Card)` in
  that case — flag this in review).
- Reviewer: board names render via html/template escaping (model/owner input);
  the remove-by-index forms must re-read the record inside the handler (no
  TOCTOU between render and post is worth defending beyond bounds-checking,
  single-owner box).
