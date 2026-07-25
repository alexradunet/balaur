---
phase: ticket
status: done
project: 003-file-unified-node-model
ticket: 04
blocked-by: [03]
worker: "7fc30117-f726-4029-8cf0-ff72ddb863d2"
branch: "003-ticket-04"
shared-blast-radius: true
---

# Ticket 04: Contract — text read-only + greenfield file-backed starter

## What to build

Finish the hard cut. First, contract the edit surface so `text` is read-only interop:
Balaur never authors or mutates a text node, but imported/external `text` nodes still
render. Second, regenerate the graph starter as a greenfield, fully file-backed space
so a fresh install demonstrates the unified model. This is plan Steps 9-10 in one
ticket (the final "hard cut" milestone).

Make `text` read-only in the UI (Step 9, `app.js`): remove the inspector's
text-editing textarea — delete the `else if(item.type==="text")fields.push({key:"text",
label:"Markdown",control:"textarea",value:item.text});` branch (`app.js:1171`) and
replace it with a read-only note, e.g. `else if(item.type==="text")notes.push({text:
"Imported text node · read-only. Balaur authors file-backed notes (ADR-0004)."});`, so a
selected foreign `text` node shows its content (rendered on the canvas) but offers no
edit field. The `text` render branch in `renderNodes` (`app.js:844-851`) STAYS so
imported documents remain visible. Remove the now-dead `aiTitle`/`aiPrompt` mutation of
`item.text` in `applyInspectorField` (`app.js:1258`); those fields are handled by the
`scope==="note"` branch from Ticket 02 (an AI operator is a note file-node).

Rewrite the graph starter as a greenfield file-backed space (Step 10, `app.js`): in
`createGraphStarterWorkspace()` (`app.js:48-99`), replace every inline `text` node
(`home-guide`, `inbox-guide`, `inbox-trip`, `projects-guide`, `cb-note`, `wiki-guide`,
`wiki-budget`, `wiki-subscriptions`, `archive-guide`, `archive-portfolio`) with a
standard `file` node referencing a stable `notes/*.md` path. Define those paths as
module-level consts beside `STARTER_TASK_PATH` (`app.js:46`), e.g. `const STARTER_NOTES
= { homeGuide:"notes/start-here-your-life-as-a-graph.md", inboxTrip:"notes/autumn-city-break-idea.md", … };`
(slug each title through the same convention). Keep every node's `id`, geometry, and
`color` (including the dormant `#6c757d` archive color), and keep every edge
(`e-inbox-filed`, `e-cb-partof`, `e-wiki-relates`) with its label and color — only the
node storage changes from `type:"text"` to `type:"file", file:<path>`. Tasks (`cb-task`
→ `STARTER_TASK_PATH`) and the journal (`wiki-journal` → `journalPath`) are unchanged.
In `seedGraphStarterEntities()` (`app.js:100-118`), seed each starter note file at the
exact path its `file` node references, via `noteRepository.createNote({path:STARTER_NOTES.x,
body:"…", kind:…})` — exactly the way the task is seeded at `STARTER_TASK_PATH` today,
using the explicit `path` override added in Ticket 01, wrapped in try/catch so
re-seeding is idempotent. The inbox capture carries the `<!-- balaur:inbox -->` marker;
the reference pages carry `<!-- balaur:reference -->`. There is no compatibility path and
no migration of the old inline starter (hard cut). Leave the legacy `demoCanvas`
(`app.js:24-42`) untouched — its text nodes now render read-only via the interop path.

## Acceptance criteria

- [ ] The inspector no longer offers a Markdown editing textarea for a `text` node; a selected foreign `text` node shows a read-only note instead, and its content still renders on the canvas (the `text` render branch stays).
- [ ] `grep -n 'item.text=' app.js` returns NO matches (no code path mutates a text node's content).
- [ ] `createGraphStarterWorkspace` builds Home + four hub canvases whose guide/example cards are standard `file` nodes referencing stable `notes/*.md` path consts (defined beside `STARTER_TASK_PATH`); every node `id`, geometry, and `color` (including the dormant `#6c757d` archive color) and every labelled edge (`e-inbox-filed`, `e-cb-partof`, `e-wiki-relates`) is preserved; the task (`STARTER_TASK_PATH`) and journal (`journalPath`) nodes are unchanged.
- [ ] `seedGraphStarterEntities` seeds each starter note file at the exact path its `file` node references via `noteRepository.createNote({path, body, kind})`, idempotently (try/catch); the inbox capture carries `<!-- balaur:inbox -->` and the reference pages carry `<!-- balaur:reference -->`.
- [ ] A fresh install (true first run via `createGraphStarterWorkspace`) builds a fully file-backed space; no interactive authoring path produces a `text` node (the only remaining `type:"text"` matches are the legacy `demoCanvas` and the read-only interop render path).
- [ ] The legacy `demoCanvas` (`app.js:24-42`) is untouched.
- [ ] `node --check app.js` exits 0.
- [ ] `storage/life-indexer.js`, `storage/memory-index.js`, `storage/canvas-validate.js`, and the other repositories remain unchanged.
- [ ] `git diff --check` exits 0; only `app.js` is modified in this ticket.

## Blocked by

Ticket 03 (Migrate — authoring templates, double-click, AI output, and local parser).
