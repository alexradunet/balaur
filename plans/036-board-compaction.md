# Plan 036: Board auto-compaction — cards pack upward after move/resize

> **Executor instructions**: Follow this plan step by step; run every
> verification and confirm the expected result. STOP conditions are binding.
> Commit on branch `advisor/036-board-compaction`. SKIP updating
> `plans/readme.md`. Audit every report claim against a tool result.
>
> **Drift check (run first)**: `git diff --stat 83ccb1e..HEAD -- web/static/board.js internal/web/boards.go internal/cards web/templates/boards.html`
> Any drift → compare excerpts; on mismatch, STOP.

## Status

- **Priority**: P3 · **Effort**: M · **Risk**: MED (pure-JS geometry; the
  server contract is already safe)
- **Depends on**: 032 (DONE, merged) · **Category**: direction
- **Planned at**: commit `83ccb1e`, 2026-06-12

## Why this matters

Plan 032 shipped drag/resize without compaction; the owner now wants the
mockup's behavior (its `gPack`): after every move or resize, cards pack
upward into free space so the board stays dense. The reference algorithm is
in `Balaur_ds/Balaur App.dc.html` (now committed — search `gPack`): sort
cards by (y, then x), then for each card move it up row-by-row while no
other card overlaps.

## Current state

- `web/static/board.js` (242 lines): pointer drag via `.board-slot-grip`,
  resize via `.board-slot-resize`; working state in `dataset.x/y/w/h`;
  `pinAllSlots()` migrates flow→free; on pointerup `serializeLayout()` +
  `fetch POST /ui/boards/{id}/layout` (form field `layout` =
  `[{"idx","x","y","w","h"},…]`); `location.reload()` on non-OK. Grid: 12
  cols, 10px row units, gap from computed style.
- Server (`internal/web/boards.go` `boardsLayout`): validates idx
  bounds/count, never touches type/params, `cards.ValidateCards` clamps —
  compaction needs NO server change.
- The mockup's `gPack` (read it in `Balaur_ds/Balaur App.dc.html`, around
  the `gPack =` definition): pure function over `[{x,y,w,h}]`.

## Commands

| Purpose | Command | Expect |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tests | `go test ./...` | ok |
| Vet/fmt | `go vet ./...` / `gofmt -l .` | clean |

## Scope

**In scope**: `web/static/board.js`; `internal/self/knowledge.md` + `DESIGN.md`
(drop the "no compaction" caveat both added in 032 — the honesty ledger must
say compaction exists now).
**Out of scope**: ALL Go code, templates, CSS, `internal/cards` — the server
contract already accepts any clamped layout.

## Git workflow

Branch `advisor/036-board-compaction`; commit
`feat(boards): auto-compaction — cards pack upward after move/resize`.

## Steps

### Step 1: `packLayout` in board.js

Add a pure function (placed near `serializeLayout`, same style):

```js
// packLayout compacts slots upward: sort by (y, x), then walk each card up
// one row unit at a time while it overlaps nothing already placed.
// Mirrors the mockup's gPack. Pure: takes/returns [{el,x,y,w,h}].
function packLayout(items) { … }
```

Overlap test: `ax < bx+bw && bx < ax+aw && ay < by+bh && by < ay+ah`.
Keep the dragged card's x and w; only y changes. The currently-dragged card
participates like any other (mockup behavior: everything packs).

### Step 2: Wire it

Call `packLayout` on every `pointerup` for both drag and resize, BEFORE
`serializeLayout`/POST; apply the packed y values to each slot's inline
`grid-row` and `dataset.y` so the visual result and the persisted payload
agree. Also pack once inside `pinAllSlots()` (first migration produces a
dense board). Do NOT pack live during pointermove (one pack per gesture —
keeps the drag target stable under the pointer).

### Step 3: Ledger truth

032 wrote "compaction is not implemented" into knowledge.md and DESIGN.md —
find those clauses and update them: cards now auto-pack upward after each
move/resize.

**Verify**: `grep -rn "compaction" internal/self/knowledge.md DESIGN.md` →
the remaining mentions say it exists (no "not implemented" phrasing).

## Test plan

JS has no automated harness (state that plainly). Mitigate: keep `packLayout`
pure and small; include in NOTES a hand-trace of one packing case
(3 cards, one gap) showing input→output coordinates. Go gates must stay
green (`go test ./...` — nothing should change).

## Done criteria

- [ ] `packLayout` exists, pure, called from both gesture ends + pinAllSlots
- [ ] No Go/CSS/template diffs (`git diff --stat` shows board.js + 2 docs only)
- [ ] Ledger clauses updated; all gates clean

## STOP conditions

- Compaction requires server changes or template changes — it must not;
  report instead.
- board.js would exceed ~350 lines total.

## Maintenance notes

- Reviewer: manual pass — drag a card into a gap (others rise), resize
  shorter (below packs up), reload persists packed state, legacy board's
  first drag produces a dense layout.
