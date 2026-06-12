# Plan 032: Board drag + resize with persisted layout (no compaction)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report — do not improvise. Commit on branch
> `advisor/032-board-drag-resize`. SKIP updating `plans/readme.md` (reviewer
> maintains it). Audit every report claim against a tool result.
>
> **Drift check (run first)**: `git diff --stat 90bc397..HEAD -- internal/web/boards.go internal/cards web/templates/boards.html web/static`
> Any change → compare "Current state" excerpts; on mismatch, STOP.

## Status

- **Priority**: P2 · **Effort**: M–L · **Risk**: MED (first real client-side
  state JS; layout persistence)
- **Depends on**: 029, 030 (DONE, merged)
- **Category**: direction · **Planned at**: commit `90bc397`, 2026-06-12

## Why this matters

Boards shipped with server-defined layouts (plan 029). The owner now wants the
mockup's direct manipulation: drag a card to move it, grab its corner to
resize, and have the arrangement persist per board. Owner decisions
(2026-06-12): pointer drag + corner resize with 12-column snap, layout
persisted to PocketBase on drop; **no auto-compaction** in v1 — gaps stay
where you leave them.

## Current state

- `internal/web/boards.go`: `type boardCard = cards.Card` (line 22);
  `boardCardView` (line 40) carries `{Type, Query, W, Idx, Label…}` — read the
  actual struct; `boardCardViewsOf` (line 47) resolves specs. Board records:
  collection `boards`, fields `name` text, `cards` json
  (`[{"type":…,"params":{…}}]`), `sort` number.
- `internal/cards/cards.go`: `Card{Type string; Params map[string]string}`,
  `ValidateCards([]Card) ([]Card, error)`, `Spec{Type, Label, Icon, W int,
  Params}` — `W` is the default 12-col span. There is no per-type default
  height yet.
- `web/templates/boards.html` `board_grid` define renders:

  ```html
  <div class="board-grid" id="board-grid">
    {{range .Current.Cards}}
    <div class="board-slot" style="grid-column: span {{.W}}">
      <div class="board-slot-inner" hx-get="/ui/cards/{{.Type}}{{.Query}}"
           hx-trigger="load" hx-swap="innerHTML">…</div>
      <form … action="/ui/boards/{{$.Current.ID}}/cards/{{.Idx}}/remove" …>✕</form>
    </div>
    {{end}}
  ```

- `web/static/basm.css` ends with `/* ── Boards ── */` then `/* ── Candle ── */`
  sections; `.board-grid { display:grid; grid-template-columns:repeat(12,1fr); gap:16px }`.
- The mockup (`Balaur_ds/Balaur App.dc.html`, NOT in worktrees — reference
  only) used per-card `{x,y,w,h}` on a free grid with ~10px row units and
  per-type default heights (11–34 rows).
- JS conventions: `web/static/basm.js` (118 lines) — plain functions, no
  framework, `defer` loaded from `page_head`. Boards-specific JS belongs in a
  NEW file `web/static/board.js`, loaded only by boards.html.

## Commands

| Purpose | Command | Expect |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tests | `go test ./...` | ok |
| Vet/fmt | `go vet ./...` / `gofmt -l .` | clean |

## Scope

**In scope**: `internal/cards/cards.go` (+test) — layout fields + default H;
`internal/web/boards.go` (+`boards_test.go`) — layout route + view fields;
`internal/web/web.go` (one route); `web/templates/boards.html`;
`web/static/board.js` (new); `web/static/basm.css` (extend the existing
Boards section only); `internal/self/knowledge.md`, `DESIGN.md` ledger
(one clause: boards are draggable/resizable, layout persisted).

**Out of scope**: compaction/auto-pack; touch-screen long-press affordances
beyond what pointer events give for free; `basm.js`; chat/tasks files; any
new collection (layout lives in the existing `cards` json).

## Git workflow

Branch `advisor/032-board-drag-resize`; commit
`feat(boards): drag + resize with persisted layout`.

## Steps

### Step 1: Layout model in `internal/cards`

Extend `Card` with optional layout (omitempty keeps existing JSON and the
tools' args unchanged):

```go
type Card struct {
    Type   string            `json:"type"`
    Params map[string]string `json:"params,omitempty"`
    X      int               `json:"x,omitempty"` // 0-based col, 0..11
    Y      int               `json:"y,omitempty"` // 0-based row unit
    W      int               `json:"w,omitempty"` // col span 1..12; 0 = spec default
    H      int               `json:"h,omitempty"` // row-unit span; 0 = spec default
}
```

`ValidateCards` additionally clamps: X to 0..11, W to 0..12, Y to 0..500,
H to 0..120, and X+W ≤ 12 (shrink W). Add `H int` to `Spec` with per-type
defaults (row unit = 10px; pick from the mockup's proportions): today 16,
quests 30, calendar 26, timeline 26, journal 18, measure 12, lines 12,
memory 20, skills 14, heads 16. Tests: clamping table; JSON round-trip omits
zero layout.

**Verify**: `go test ./internal/cards/...` → ok.

### Step 2: Server render + layout endpoint

- `boardCardView` gains `X, Y, W, H` (resolved: card value if >0 else spec
  default; for cards with no stored layout, the server assigns sequential
  flowing positions exactly like today's behavior — simplest: when ALL cards
  of a board have zero X/Y/H, render the legacy `grid-column: span W` flow
  (no grid-row), so existing boards look unchanged until first drag).
- `board_grid` template: when a card has explicit layout, emit
  `style="grid-column: {{.X1}} / span {{.W}}; grid-row: {{.Y1}} / span {{.H}}"`
  (precompute X1=X+1, Y1=Y+1 in Go — keep template logic dumb). Add a drag
  grip element `<div class="board-slot-grip" title="drag to move">⠿</div>` and
  a resize handle `<div class="board-slot-resize" aria-hidden="true"></div>`
  to each slot, and `data-idx="{{.Idx}}"` on the slot.
- New route `POST /ui/boards/{id}/layout` (register next to the other board
  routes): form field `layout` = JSON `[{"idx":0,"x":0,"y":0,"w":4,"h":16},…]`.
  Handler: load record, decode existing cards, apply x/y/w/h by idx (REJECT
  with 400 if an idx is out of bounds or the entry count mismatches the
  board's card count — type/params are NEVER touched by this endpoint), run
  `cards.ValidateCards`, save. Respond 204 No Content (the client already
  shows the result; no re-render needed).
- `.board-grid` CSS (extend the Boards section): when the board has explicit
  layout the container needs `grid-auto-rows: 10px;` — set class
  `board-grid-free` from Go in that case; `.board-slot { position:relative }`,
  `.board-slot-inner { height:100%; overflow:auto }`, grip top-left (mono,
  `cursor:grab`), resize handle 14px bottom-right corner
  (`cursor:nwse-resize`), `.board-slot.dragging { opacity:.7; z-index:3 }`.

**Verify**: `go test ./internal/web/...` → ok (incl. new handler tests:
happy path persists and survives reload; idx out of bounds → 400; count
mismatch → 400; type/params unchanged after layout post).

### Step 3: `web/static/board.js` (~150 lines, vanilla)

Loaded via `<script src="/static/board.js" defer></script>` in boards.html
only. Behavior:

- On `pointerdown` on `.board-slot-grip`: capture pointer; compute the grid
  cell size from `#board-grid` (`(gridWidth - 11*gap)/12` per column; row =
  10px + 0 gap share — read computed gap); on `pointermove` set the slot's
  inline `grid-column`/`grid-row` from the pointer cell (clamp X to keep
  X+W ≤ 12, Y ≥ 0); add `.dragging`.
- On `pointerdown` on `.board-slot-resize`: same loop but adjusting W
  (1..12-X) and H (min 6).
- On `pointerup` for either: serialize ALL slots (`data-idx`, current
  computed column/row start/span — store the working values in
  `dataset.x/y/w/h` during the interaction so no CSS parsing is needed) and
  `fetch('/ui/boards/{id}/layout', {method:'POST', body: new URLSearchParams({layout: JSON.stringify(list)})})`.
  Board id from `data-board-id` on `#board-grid` (add it in the template).
  On non-OK response: `location.reload()` (server state wins).
- First drag on a legacy flowing board: before starting, snapshot every
  slot's CURRENT resolved grid position (`getComputedStyle` grid-column/row
  resolution is unreliable for auto-flow — instead compute from
  `offsetLeft/offsetTop` against cell size) and pin all slots to explicit
  coordinates, adding `board-grid-free` to the container. Document this
  function (`pinAllSlots`) — it is the migration moment from flow to free.
- No drag on `(pointer: coarse)`? No — pointer events work for touch; just
  ensure the grip uses `touch-action: none` (CSS).

**Verify**: `CGO_ENABLED=0 go build ./...` (embed picks up board.js — confirm
with the existing embed-FS asset test pattern: add `static/board.js` to
`web/embed_assets_test.go`'s list); `go test ./web/...` → ok.

### Step 4: Docs + ledger

knowledge.md: one sentence (boards drag/resize, persisted). DESIGN.md "True
today" boards clause: append "; cards drag and resize (pointer + 12-col
snap), layout persisted per board". Note in both that compaction is not
implemented (honesty).

**Verify**: `grep -n "resize" internal/self/knowledge.md DESIGN.md` → ≥1 each.

## Test plan

Go tests as in Steps 1–2 (clamping, layout endpoint contract, legacy-flow
default render, free render emits grid-row). JS is untested by automation —
state that plainly in the report; the reviewer does the manual pass.

## Done criteria

- [ ] Layout survives reload (handler test asserts persisted values render)
- [ ] `POST /ui/boards/{id}/layout` cannot alter type/params (test proves)
- [ ] Legacy boards (no layout) render exactly as before (test: no `grid-row`)
- [ ] `web/embed_assets_test.go` covers `static/board.js`
- [ ] All gates clean; no out-of-scope files (`git status`)

## STOP conditions

- The `cards` json shape on existing records resists in-place extension.
- The flow→free pinning approach (offsetLeft/offsetTop) proves unworkable in
  reasoning — do not invent a different persistence model; report.
- You want a JS framework or >250 lines of JS.

## Maintenance notes

- Compaction (mockup gPack) is the known follow-up; the dataset.x/y/w/h
  working-state design leaves room for it.
- Reviewer: manual pass — drag, resize, reload, second board untouched,
  remove-card after layout still bounds-correct (remove re-renders grid from
  server; layout idxs shift — the server rewrites cards array on remove, so
  stored layout moves with the entries; verify a remove after drag doesn't
  scramble positions).
