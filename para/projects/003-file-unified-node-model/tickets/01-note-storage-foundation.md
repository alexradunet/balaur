---
phase: ticket
status: done
project: 003-file-unified-node-model
ticket: 01
blocked-by: []
worker: "ab1ea38e-c898-4c0e-ba43-95de8c8ffac6"
branch: "003-ticket-01"
shared-blast-radius: true
---

# Ticket 01: Expand — note catalog, note repository, and contract suite

## What to build

The storage foundation for the file-unified node model (ADR-0004), as three NEW
files with zero changes to `app.js`. A **Note** is a path-identified canonical
`notes/*.md` file with no mandatory frontmatter and no `orbit-id`; its identity is
its path, and its kind (inbox/reference/AI) is an inert body marker. This ticket
makes notes creatable, placeable, updatable, deletable, and drainable through one
thin path-keyed repository, proven by a Node contract suite against the in-memory
adapters.

Create `storage/note-catalog.js`: a disposable synchronous projection in the
`widget-catalog.js` / `component-card-catalog.js` mold. Export `isNotePath(path)`
(`notes/` prefix + `.md` suffix) and `class NoteCatalog` with constructor `{vault}`
(throw `TypeError` without a vault) and methods `rebuild()`, `reconcile(_paths=[])`
(delegate to `rebuild()`), `getByPath(path)`, `getFallbackByPath(path)`, `notes()`,
`diagnostics()`. `rebuild()` lists `notes/` and `canvases/` in parallel; for each
note it records a case-fold-collision diagnostic (`NOTE_PATH_CASE_COLLISION` via
`caseFoldKey`), reads the body, and freezes `{path, hash, title, body, kind,
placements:[]}` into `_byPath`; on read/parse failure it freezes a fallback
`{path, title, body:"", diagnostic}` into `_fallbackByPath` with a
`NOTE_MALFORMED`/`NOTE_UNREADABLE` diagnostic and never throws out of `rebuild()`.
It then scans each canvas document (parse, `isCanvas()`; record `CANVAS_MALFORMED`
and skip on failure) and for every `file` node with `isNotePath(node.file)` pushes
`{canvasPath, nodeId}` onto that note's `placements`, else records
`NOTE_FILE_MISSING`. A note with zero placements is normal — no orphan diagnostic.
Title = first `# Heading` line else the path slug; kind = `"inbox"`/`"reference"`/
`"ai"` from the inert markers (`<!-- orbit:inbox -->`, `<!-- orbit:reference -->`,
`<!-- orbit:ai-card -->`) else `null`.

Create `storage/note-repository.js`: `class FileNoteRepository`, path-keyed,
combining the task/journal write discipline with the component-card path-scan
placement resolution. Constructor ports `{vault, index, indexer, catalog,
canvasPathFromId, now = () => new Date().toISOString()}`. Methods: `allocatePath(title)`
(slugify, probe `notes/<slug>.md` then `-2`, `-3`, … by case-fold key, return
`normalizePath(candidate)`; collision handling runs only at creation);
`createNote({title, body="", kind=null, color, canvasId, geometry, path})` — the
optional explicit `path` override (the task repository supports `input.path` the same
way) is REQUIRED here so Ticket 04's starter can seed notes at exact paths; derive the
title from the body's first heading when omitted, prepend the kind marker when `kind`
is set, write the `.md` with `{expectedHash:null}`, `indexer.indexFile(path, content,
{})`, `catalog.reconcile([path])`, then `addPlacement` if `canvasId`, and return
`{path, note, placement}`; `addPlacement(path, canvasId, geometry={})` cloned exactly
from `FileJournalRepository.addPlacement` (`journal-event-repository.js:64-88`) —
resolve the canvas path, `vault.stat` (throw `CANVAS_NOT_FOUND`), parse + `isCanvas()`
(throw `CANVAS_INVALID`), validate finite-integer geometry with positive width/height
(throw `CANVAS_GEOMETRY_INVALID`), allocate a unique node id (`geometry.id ||
node-${randomToken()}`, throw `CANVAS_ID_DUPLICATE` on clash), push `{id, type:"file",
file:path, x, y, width, height}` plus `color` when given, re-validate with
`isCanvas()`, write `JSON.stringify(doc,null,2)+"\n"` with `expectedHash: stat.hash`,
reindex the canvas, reconcile the catalog, return `{canvasId, nodeId, canvasPath,
path}`; `removePlacement(canvasId, nodeId)` cloned from
`FileTaskRepository.removePlacement` (`task-repository.js:151-167`) — filter the node
and its incident edges, return `{removed:false,…}` if nothing removed, else write,
reindex, reconcile, return `{removed:true, canvasId, nodeId}`; `updateNote(path, body)`
— `vault.stat` (throw `NOTE_NOT_FOUND`), full-body rewrite under `expectedHash:
stat.hash` (a note has no frontmatter, so this is trivially preservation-first),
reindex, reconcile, return `catalog.getByPath(path)`; `deleteNote(path)` — `vault.stat`
(throw `NOTE_NOT_FOUND`), resolve placements via `catalog.getByPath(path)?.placements`
(PATH SCAN; the placements index does NOT track notes), `removePlacement` each, then
`vault.remove(path, {expectedHash: stat.hash})`, `indexer.removeFile(path)`, reconcile,
return `{path, removedPlacements}`; and `replacePlacement(path, fromCanvasId, nodeId,
toCanvasId, geometry={})` — the DRAIN primitive: `removePlacement(fromCanvasId,
nodeId)`, and if removed `addPlacement(path, toCanvasId, geometry)`, return `{removed,
added: added||null, path}`. The source path is preserved so identity is unchanged. Use
`SchemaError`/`PathError` from `storage/vault-errors.js` with codes matching the
existing repositories, and a local `randomToken()` copied from the task repository.

Create `storage/note-repository.test.js`: a focused named suite (the
`component-card.test.js` precedent, not a `phaseN` number) using `node:test` +
`node:assert/strict`, with a `setup()` wiring `MemoryVault` + `MemoryIndex` +
`LifeIndexer` + `NoteCatalog` + `FileNoteRepository` and a `seedCanvases(vault)`
helper (copy the harness from `phase5.test.js:18-37`).

## Acceptance criteria

- [ ] `storage/note-catalog.js`, `storage/note-repository.js`, and `storage/note-repository.test.js` exist; `node --check` exits 0 for each.
- [ ] `createNote` writes a canonical `notes/<slug>.md`, indexes it as an UNTYPED source record (`index.getSourceFile(path)` has `entityType:null, parseStatus:"ok"`), and the body bytes match (marker + Markdown).
- [ ] `createNote` with a `canvasId` adds a standard `file`-node placement whose `file` equals the note path, and the canvas still passes `isCanvas()`.
- [ ] Collision allocation: two notes titled the same yield `notes/new-thought.md` then `notes/new-thought-2.md` (and a third `-3`); a case-fold collision (`New Thought` vs `new thought`) also allocates a suffix.
- [ ] `addPlacement` rejects a duplicate node id with `CANVAS_ID_DUPLICATE` and a missing canvas with `CANVAS_NOT_FOUND`.
- [ ] `removePlacement(canvasId, nodeId)` removes the node and its incident edges, keeps the canvas valid, and leaves the note file intact.
- [ ] `updateNote(path, body)` rewrites the body under the expected hash; a stale expected hash (a write behind the repository's back) rejects.
- [ ] `deleteNote(path)` removes every placement (place the note on two canvases first), then removes the file; the canvases stay valid.
- [ ] `replacePlacement(path, fromCanvas, nodeId, toCanvas, geometry)` (drain) removes the source placement and adds one for the SAME path on the target; the file is untouched and identity (path) is unchanged. `replacePlacement` is path-generic and path-keyed (project 002 consumes it).
- [ ] REGRESSION PIN (mandatory): after `createNote(..., {canvasId})`, `index.allPlacements()` and `index.placementsForEntity(<anything>)` contain NO row for the note path, yet `deleteNote` and `replacePlacement` still resolve the placement by scanning canvas documents (they work without the placements index).
- [ ] `createNote` accepts an optional explicit `path` override (for the Ticket 04 starter seed).
- [ ] `node --test storage/note-repository.test.js` passes in full.
- [ ] Notes gain NO `orbit-id` and NO mandatory frontmatter; `storage/life-indexer.js`, `storage/memory-index.js`, and `storage/canvas-validate.js` are byte-for-byte unchanged (`git diff --stat` shows none of them); no existing phase suite regresses.
- [ ] `git diff --check` exits 0; only the three new files are created.

## Blocked by

None — can start immediately.
