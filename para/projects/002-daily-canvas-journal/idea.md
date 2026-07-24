---
phase: idea
status: grilled
project: 002-daily-canvas-journal
date: 2026-07-24
depends-on: [001-jd-to-para, 003-file-unified-node-model]
---

# Idea: Daily journal as a zoomable daily-canvas hierarchy

Raised during 001 implementation. The user wants the daily journal to be a
**daily canvas** (one canvas per day), organized so you **zoom through
year → month → week → day** to reach a given day, rather than a textarea panel
in the Today view (which is what 001's Step 5 builds). This follow-up replaces
that Step-5 panel.

## Why it fits
- More canvas-native than a sidebar form; aligns with the product thesis
  "canvas as the main operating layer." A day-canvas is a richer daily driver
  (that day's journal node + tasks + notes + links) than a single textarea.
- The journal stays a canonical `.md` (`journal/YYYY/YYYY-MM-DD.md`, existing
  `FileJournalRepository`); the day-canvas is the *spatial container* holding
  that day's journal file-node. Reconciles cleanly with the file-canonical model.

## Grounded feasibility (verified in the engine, 2026-07-24)
- **Zoom-to-enter already exists** (`app.js:1055`): hover a portal and scroll
  past **220%** to enter the sub-canvas. So "zoom to drill down" through a
  year→month→week→day canvas hierarchy works with current machinery.
- **Per-canvas camera persists** (sidecar stores a `camera` per canvas, restored
  on entry), so each level remembers its own zoom/position.
- **Canvases load eagerly at boot** (`loadWorkspace` reads every `.canvas` into
  memory; AGENTS.md mandates an in-memory working set). A pre-built
  year→month→week→day tree is ~430 canvases/year, all loaded at startup. So the
  design MUST **create day/month canvases on demand** (only when journaled /
  navigated to), not pre-build the tree.

## Decisions made (2026-07-24, with user)
- **Zoom-to-navigate across nested day-canvases for v1** (feasible now).
- **True semantic zoom deferred** (one continuous space where zoom level changes
  rendered granularity: year grid → month cards → week rows → day cells). The
  engine has no precedent (zoom is a plain CSS scale); treat as a later ambition.
  Zoom-to-navigate gets ~90% of the feel.
- **On-demand canvas creation** to avoid the eager-load volume problem.

## Open questions (for the grill/plan)
- Exact hierarchy: year→month→week→day, or skip the week level (year→month→day)?
  Weeks add ~52 canvases/year; is the week granularity worth it?
- Where does the time hierarchy hang off the graph? A Journal/Calendar hub in the
  001 model (Inbox · Projects · Wiki · Archive), or a dedicated time index?
- How is "today" reached quickly (a Today-view "open today's day-canvas" shortcut
  may survive from 001's Step 5)?
- Day-canvas creation trigger: explicit "journal today" action, or auto-create on
  first edit?
- What does a day-canvas contain by default (journal node + task placements +
  free notes)? How do its tasks relate to the global Today projection?
- Reuse of 001 Step 5: which parts survive (date nav, place-on-canvas, journal
  repo placement) vs. get replaced (the textarea editor)?

## Relationship to 001
001 Step 5 builds a Today-view daily-note panel (textarea + prev/Today/next +
place-on-canvas). This follow-up supersedes the textarea editor with "open
today's day-canvas," likely keeping a Today shortcut + the journal repo
placement. Sequenced after 001 lands.

## Status (2026-07-24) — grilled, reframed, now depends on 003

Grilling this idea surfaced a bigger foundation it needs: a **file-unified node
model** (every content node is a `file` node; notes become `.md` files). That is
now project **003-file-unified-node-model**, and 002 is sequenced *after* it
(003 must land first). The shared decisions were grilled together and live in
`../003-file-unified-node-model/grill-2026-07-24.md`.

Settled for 002 (see 003's grill for rationale):

- **Hierarchy:** `year → month → day` (week level skipped; a week view, if ever
  wanted, is a visual grouping inside the month canvas, not a canvas level).
- **Position:** the daily canvas is the **top-level front door**, replacing the
  Inbox hub's role; it supersedes the single `wiki-journal` node in Wiki.
- **Behavior:** a **draining queue** — capture notes + tasks into today's canvas,
  then physically re-place them out to project/Wiki/Archive canvases. This only
  works because 003 makes notes file-backed (a note re-places exactly like a
  task). **Inbox survives as a state** (a node still in today's canvas with no
  outgoing filing edge), not a place; the Inbox hub canvas is retired.
- **Creation:** year/month/day canvases are created **on demand**, never
  pre-built (canvases eager-load at boot).

Still open for 002's own spec (deferred from the grill): where the time-tree
hangs off Home and the "open today" fast path; day-canvas default contents and
how its tasks relate to the global Today projection; which parts of 001 Step 5
survive vs are replaced; zoom-to-navigate UX. True semantic zoom stays a later
ambition, not v1.
