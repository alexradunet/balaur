---
phase: implement
status: done
project: 003-file-unified-node-model
ticket: 02
date: 2026-07-25
plan: plan.md
commit: 294636d
branch: 003-ticket-02
---

# Implementation: Ticket 02 — boot wiring, note render/edit, AI body-detection (plan Steps 4-6)

Scope is `app.js` only. Ticket 01's `storage/note-catalog.js` and
`storage/note-repository.js` (commit `4337949`) are the dependencies and were left
byte-for-byte unchanged. Work was done in dependency order (Step 4 → Step 6 → Step 5)
because the Step 5 render/inspector branches call the Step 6 `parseAICard`/`isAICard`
refactor; each step ended in a green `node --check` and was committed separately.

## Steps completed

- [x] Step 4: boot wiring — verified: `node --check app.js` exit 0.
  - Imports `NoteCatalog, isNotePath` and `FileNoteRepository` (app.js:16-17).
  - Globals `noteCatalog`, `noteRepository` (app.js:168-169) and `noteUpdateTimers` (app.js:173).
  - `configureLifeRuntime` constructs `noteCatalog` and `noteRepository` with the same
    `canvasPathFromId` resolver as the other repositories (app.js:331-335).
  - `noteCatalog.rebuild()` added to all five fan-outs: boot (app.js:359),
    `rebuildLifeIndex` as `noteCatalog?.rebuild()` (app.js:402), `loadGraphStarter`
    (app.js:421), import (app.js:1486), and the `applyCanvasOperations` reconcile
    (app.js:1619). Matching optional-chaining style per site, as the ticket specifies.
  - Boot-failure reset clears `noteCatalog = null; noteRepository = null;` (app.js:364).
- [x] Step 6: AI detection/parsing migration — verified: `node --check app.js` exit 0.
  - `parseAICard` split into `parseAICardBody(body)` (existing line logic on a body
    string) + thin `parseAICard(node)` reading `noteCatalog?.getByPath(node.file)?.body`
    (app.js:688-692). `buildAICardText` unchanged (app.js:724).
  - `isAICard` now detects a file-backed note via the catalog body (app.js:687).
  - `noteKind` reads kind from the catalog for a note file-node, returning
    inbox/reference/null (app.js:764); legacy text-node marker path retained for interop.
  - `nodeTitle` returns the catalog title for a note before the entity lookup (app.js:733).
  - `nodeSummary` derives the one-line summary from a note body with the same
    heading-first / first-non-marker convention (app.js:773-781).
  - `nodeAIContent` returns the catalog body for a note synchronously, before the
    entity/cache path (app.js:737).
  - `aiCardSignature` now signs `nodeAIContent(card)` instead of `card.text`
    (app.js:756). This was required by the hard constraint "no residual `node.text`
    reads on file nodes": after Step 6 an AI operator is a `file` node, so `card.text`
    was a residual file-node read. `aiCardSignature` is consumed by `applyInspectorField`
    and `deleteSelection` (both Step 5 scope). The change does not touch the AI security
    boundary, debouncing, queued reruns, or cycle detection.
- [x] Step 5: render + edit — verified: `node --check app.js` exit 0.
  - `renderNodes` note branch before the generic FILE fallback (app.js:951-962):
    resolves `getByPath || getFallbackByPath || {…diagnostic}`; a `note.diagnostic`
    renders a readable "NOTE · UNAVAILABLE" fallback (never empty, never writes);
    a body carrying `AI_CARD_MARKER` reuses the AI-operator card innerHTML reading
    title/prompt via `parseAICard`; otherwise kicker (from `noteKind` or `textMeta`)
    + `markdownToHTML(note.body)`. Synchronous catalog read, no per-card vault read.
  - Inspector (app.js:1226-1248): resolves `note`; titles it "Note"; `if(isAICard(item))`
    now drives the AI operator fields, which carry `scope:"note", notePath` so writes
    route to the note scope; a non-AI note gets `{key:"noteBody", scope:"note"}` textarea
    + `{intent:"delete-note", danger:true}` action; notes are excluded from the generic
    file-path/subpath block (`!note`) so editing cannot mutate the placement's `file`.
  - `applyInspectorField` note scope (app.js:1310-1318) mirrors the task discipline:
    `noteBodyFromInspector` computes the body (`noteBody` value, or `buildAICardText`
    against the current catalog body for `aiTitle`/`aiPrompt`); `scheduleNoteUpdate` /
    `flushNoteUpdate` debounce/flush into `updateNoteBody`, which guards on
    `canonicalWritable`, flushes pending workspace edits, calls `noteRepository.updateNote`,
    then `reloadCanvasDocuments([currentCanvasId])` and `renderNodes()` (app.js:578-608).
  - Inspector action dispatch forwards `notePath` (app.js:1211); `delete-note` wired to
    `deleteNoteEverywhere(path)` (app.js:1343), modeled on `deleteTaskEverywhere`:
    confirm, resolve affected canvases from `noteCatalog.getByPath(path).placements`
    via `canvasIdFromPath`, flush, `noteRepository.deleteNote`, reload, toast
    (app.js:1383-1392).
  - `deleteSelection` note branch removes only the placement via
    `noteRepository.removePlacement`, mirroring the task branch; the file stays
    (app.js:1406-1411).

## Files changed

- `app.js` — the only source file modified (40 lines net across three commits).

## Verification results

- `node --check app.js` → exit 0 (after each step and at HEAD).
- `node --test storage/phase1.test.js storage/phase5.test.js storage/phase8.test.js storage/phase-query.test.js storage/note-repository.test.js` → 91 pass, 0 fail.
- Full AGENTS.md §13 suite + note suite → **196 pass, 0 fail** (172 prior + 24 note-repository tests from ticket 01).
- `git diff --check` → exit 0.
- `git status --porcelain` → clean (all committed); only `app.js` modified by this ticket.
- Protected files unchanged: `git diff 4337949 -- storage/life-indexer.js storage/memory-index.js storage/canvas-validate.js storage/note-catalog.js storage/note-repository.js storage/task-repository.js storage/journal-event-repository.js storage/habit-repository.js storage/component-card-repository.js storage/widget-repository.js ai/generated-operations.js` → 0 lines.
- No residual `node.text` reads on file nodes: every remaining `.text` access is a guarded `type==="text"` interop read (nodeTitle/nodeAIContent/noteKind/nodeSummary/render text branch/graph-memory), a fetch/File/provider `.text()` call, the legacy inspector text-textarea mutation (reached only by interop text nodes; Step 9 removes it), or the AI output text node (Step 8 migrates it). AI operators (now file nodes) are read only through the catalog.
- AI security boundary unchanged: `ai/generated-operations.js` and the allowlist untouched; operators still propose allowlisted operations; output never executes host code.

## Commits (branch 003-ticket-02)

- `52208a1` app: wire note catalog and repository into boot and rebuild fan-out (Step 4)
- `48489a5` app: resolve AI operator detection and parsing from note catalog body (Step 6)
- `294636d` app: render and edit file-backed notes; note inspector scope and delete-everywhere (Step 5, HEAD)

## Browser-pending

Ticket 02 behaviors are UI-level and are NOT covered by the Node seam; they remain
browser-pending and must be verified later via the `browser-check` skill:

- A file-backed note renders its Markdown via the synchronous catalog projection.
- A note whose body carries the AI marker renders the AI-operator card UI.
- A missing/damaged note renders the readable fallback and is never overwritten.
- The inspector Markdown editor (`scope==="note"`) writes the `.md` (debounce/flush, reindex + reconcile + re-render) and never mutates the canvas document.
- "Delete note everywhere" and Delete-key placement removal behave as specified.

Note: at this intermediate point in the migration no authoring path produces file-backed
notes yet (Steps 7-10 are later tickets), so these render/edit/AI paths are dormant until
the authoring and starter migration lands; existing interop `text` nodes still render.

## Issues encountered

- The ticket's Step 6 function list did not name `aiCardSignature`, but its `card.text`
  read became a residual file-node read once `isAICard` moved to file nodes, violating
  the hard constraint. Fixed minimally by signing `nodeAIContent(card)` (the catalog
  body). This is change-detection infrastructure used by Step 5 functions; it does not
  alter the AI security boundary, debouncing, or cycle detection.
- The inspector's generic file-path/subpath editor (`{key:"file"}`) has no scope and would
  mutate a note placement's `file` reference on edit, breaking the placement and
  contradicting "editing never mutates the canvas document's content". Notes are excluded
  from that block (`!note`); they edit through `noteBody` only.
- `updateNoteBody` flushes pending workspace edits before `reloadCanvasDocuments`
  (mirroring `updateTask`), so the reload cannot discard unsaved in-memory canvas edits.
