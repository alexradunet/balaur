# Plan 100 — Panel responsive + sidebar chrome (re-scope of plan 091)

- **Status:** PLANNED (expand once **098** lands; **supersedes plan 091**, whose
  premise the canvas pivot invalidated)
- **Priority:** P2
- **Effort:** M
- **Risk:** MED
- **Depends on:** **098** (panel exists), soft-depends **099** (tabs)

## Why 091 is superseded

Plan 091 ("sidebar chrome: head/model switchers, theme/palette, recap into the
rail + responsive/a11y polish") was authored when **the left rail was the only
control surface and the chat was the only content surface**. The canvas pivot
(098) adds a right panel and changes what the rail is for. 091's premise — *where
the switchers and content live* — no longer holds, so 091 is retired and its
still-valid goals fold into this plan, re-grounded on the panel layout.

## Intent (settle details at expand time)

1. **Mobile/responsive done properly.** 098 ships a *minimal* mobile panel
   drawer (slide-in on summon, `✕`/scrim/Escape close). This plan gives it full
   accessible-drawer parity — reuse the product topnav drawer scaffolding in
   `basm.js` (lines ~226–271): `inert` when closed, focus-move-in, Tab focus
   trap, restore focus. Also wire a **mobile reveal for the left rail itself**
   (today it is `display:none` ≤720px with no burger on the live app-shell — a
   pre-existing gap 091 flagged).
2. **Sidebar chrome.** Head/model switchers + theme/palette + a recap affordance
   in the rail's brand/footer slots — but decide, now that a panel exists,
   whether any of these belong in the panel head instead (e.g. settings-ish
   switchers). Keep the rail as *navigation + identity*, the panel as *content*.
3. **Panel width / resize.** 098 sets a fixed `--w-panel` (480px). Decide whether
   to make it owner-resizable. The orphaned drag-to-resize grip + `--sidebar-w`
   plumbing in `basm.js` (~182–205, dead since the board IA) can be **revived for
   the panel** or **retired as dead code** — pick one and do it (don't leave it
   dead).

## Out of scope

- The panel/canvas mechanism itself (098), in-panel domain tabs (099).
