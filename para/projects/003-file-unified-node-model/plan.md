---
phase: plan
status: done
project: 003-file-unified-node-model
date: 2026-07-24
---

# Plan 003: File-unified node model — every content node is a file-backed note

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, report your results to the lead:
> diff summary, verification evidence, and any STOP conditions encountered.
> The human-steered lead owns push and pull-request decisions. The worker
> commits and reports only.
>
> **Pre-edit identity check (mandatory before any file edit)**:
> 1. `pwd` matches the assigned worktree path.
> 2. `git branch --show-current` is non-main (this work is on branch `rugged-cougar`).
> 3. `git worktree list` confirms this worktree's identity.
> 4. `git status --short` is clean except for the two already-present items
>    (`M CONTEXT.md` and `?? docs/adr/0004-file-unified-node-model.md`).
> Stop and report if any check fails.
>
> **Drift check (run first)**: `git diff --stat 9ab2c1d..HEAD -- app.js storage/ index.html styles/ AGENTS.md docs/`
> If any in-scope file changed since this plan was written (commit `9ab2c1d`),
> compare the "Current state" excerpts against the live code before proceeding;
> on a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1 (foundational; project 002-daily-canvas-journal is sequenced after this and depends on the drain primitive)
- **Effort**: L
- **Risk**: MED (touches every authoring path, the AI operator/output storage shape, rendering, the inspector, and the starter; the canvas validator and entity repositories must not regress)
- **Depends on**: none (ADR-0004 and the reconciled `CONTEXT.md` already exist on this branch)
- **Category**: migration
- **Planned at**: commit `9ab2c1d`, 2026-07-24

## Why this matters

Balaur runs two content-node models at once. Inline `text` nodes carry their
content inside the canvas JSON: no stable identity, no canonical file, no
placements. `file` nodes reference canonical `.md`/`.html`/`.canvas` files and
get placements, repositories, and indexing. Notes, inbox captures, reference
pages, goals, and AI operator cards are the only inline holdouts. Because an
inline note has no file to re-place, it cannot **drain** (re-place from one
canvas to another), which blocks the daily-canvas draining queue (project 002),
and it violates the file-canonical thesis (ADR-0001): the same "note" idea is
sometimes durable data and sometimes canvas-embedded pixels depending on how it
was created.

This plan unifies on one model: every content node is a standard JSON Canvas
`file` node referencing a canonical `notes/*.md` file; the canvas document is
only space plus edges. `group` (layout) and `link` (external URL) stay. `text`
stays a valid, rendered, **read-only** node type for imported/external canvases
(JSON Canvas 1.0 interop), but Balaur's own authoring never produces `text`
nodes. This is ADR-0004 (already written and accepted on this branch), which
reverses AGENTS.md §4.2 ("inbox notes and reference pages are standard text
nodes").

## Current state

Read these before editing. Line numbers are against commit `9ab2c1d`.

### Vocabulary (from the reconciled `CONTEXT.md` — use these terms in names/comments)

- **Note**: a path-identified canonical `notes/*.md` file, no mandatory
  frontmatter or `orbit-id`; identity is the path. Kind (inbox/reference/AI) is
  an inert body marker. Placed by zero or more standard `file` nodes.
- **Placement**: a standard JSON Canvas `file` node referencing a canonical
  file; its node ID identifies that spatial occurrence only.
- **Drain**: re-place a path-bound node from one canvas to another (remove the
  placement here, add a placement for the same path there), leaving the file and
  its identity unchanged.
- **Projection**: disposable runtime data derived from canonical files
  (`MemoryIndex`, preloaded catalogs). May be purged and rebuilt; never a second
  owner.
- **Text node (interop)**: a standard `text` node Balaur renders read-only for
  imported/external documents but never authors.

### The decision (ADR-0004, already on this branch at `docs/adr/0004-file-unified-node-model.md`)

Every content node is a standard `file` node referencing a canonical
`.md`/`.html`/`.canvas`; the canvas is only space plus edges. Notes are
path-identified `notes/*.md` with no mandatory `orbit-id`/frontmatter; kind is
an inert body marker; the indexer treats them as valid untyped Markdown. `text`
remains valid/rendered/read-only for interop. Migration is a hard cut:
regenerate the starter; no auto-rewrite of existing inline text nodes.

### Storage layer (the seams you build on)

- `storage/task-repository.js` — `FileTaskRepository`, the closest template for
  the note repository's write discipline: constructor ports
  `{vault, index, indexer, canvasPathFromId, now}`; canonical-file-first writes;
  `addPlacement`/`removePlacement` operate on the canvas document and re-validate
  with `isCanvas()` before writing with the canvas's expected hash; reindex via
  `indexer.indexFile(path, content, {})` after every write (see
  `task-repository.js:114-176`). **Id-keyed** (uses `index.taskById` /
  `index.placementsForEntity`).
- `storage/journal-event-repository.js` — `FileJournalRepository`, the
  **path-keyed** analog. `addPlacement(localDate, canvasId, geometry)` pushes a
  `{type:"file", file:<path>}` node and re-validates (see
  `journal-event-repository.js:64-88`). Its `updateJournal` reads the expected
  hash from `vault.stat(path).hash` (line 56-62) — the simplest precondition
  pattern; reuse it for note updates.
- `storage/component-card-repository.js` — `FileComponentCardRepository`, the
  precedent for **path-scan placement resolution** and delete-everywhere: it
  takes a `catalog` port and resolves placements from `catalog.getById(id).placements`
  (built by scanning canvas documents), removes each, then removes the file
  (see `component-card-repository.js:230-256`). The note repository uses the same
  shape because note placements are NOT in the disposable placements index.
- `storage/widget-catalog.js` / `storage/component-card-catalog.js` — the
  **projection mold** for the note content catalog: constructor `{vault}`;
  `rebuild()` lists the entity dir + `canvases/`, reads bodies, scans canvas
  file-nodes for placements, records case-fold-collision and missing-file
  diagnostics, and exposes `getByPath` / `getFallbackByPath` / `diagnostics()`
  (see `widget-catalog.js:31-110`). Render reads bodies from this projection
  synchronously; it never does an async vault read per card (AGENTS.md §5).
- `storage/vault-path.js` — `slugify(title)` (line 60-72, never empty, falls back
  to `"untitled"`, byte-bounded), `normalizePath(input)` (line 104-138, enforces
  component bounds / forbidden chars / device names / traversal), `caseFoldKey`
  / `samePathFold` (line 44-57) for collision detection, `assertSafePath`
  (line 142-176). All note path generation goes through these.
- `storage/life-indexer.js` — `buildSourceRecord` already classifies a
  frontmatter-less `.md` as a valid **untyped** source record
  (`entityType: null, parseStatus: "ok"`); proven by `storage/phase3.test.js:43-46`
  (`notes/readme.md`). `extractCanvasPlacements` (line 137-152) **skips** file-nodes
  whose path is not an entity dir (`entityTypeFromPath("notes/x.md") === null`),
  so note placements are never added to the placements index — exactly the
  behavior to preserve and pin with a regression test. **No indexer change is
  needed or allowed.**
- `storage/canvas-validate.js` — `isCanvas()` accepts `text`, `file`, `link`,
  `group`; rejects a `file` node with a `subpath` that does not start with `#`
  (proven by `storage/phase4.test.js:297`). The validator does not change.
- `storage/memory-vault.js` / `storage/memory-index.js` — the deterministic Node
  test adapters. `MemoryIndex.getSourceFile(path)` returns the source record
  (with `contentHash`); `allPlacements()` / `placementsForEntity(id)` read the
  placements index.

### App layer (`app.js`, 1831 lines — do NOT mass-format; extend surgically)

Authoring paths that currently produce `type:"text"` (all must become file-backed):

- `addNode(kind, point)` — `app.js:1103-1124`. The `presets` object
  (`note/inbox/reference/goal/habit/project/ai`) all set `type:"text"`
  (`app.js:1108-1114`). `widget`/`group` are non-text; `subcanvas`/`task` return early.
- double-click on empty background — `app.js:1074-1081` calls `addNode("note", …)`.
  The geometry hit-test `nodeAtClientPoint` (`app.js:1019-1025`) prevents spawning
  over a card; keep it.
- note tool on pointerdown — `app.js:1071` calls `addNode("note", p)`.
- AI one-shot note — `createAINote`, `app.js:1743` pushes a `type:"text"` node.
- AI operator output — `runAICard`, `app.js:1759-1760` creates/reuses a
  `type:"text"` output node connected by the reserved `AI output` edge.
- local intent parser — `runLocalAssistant`, `app.js:1655` adds a goal/habit/
  project/note as a `type:"text"` node via `applyCanvasOperations([{type:"node.add",node}])`.

AI operator detection/parsing (must move from `node.text` to the resolved note body):

- `AI_CARD_MARKER="<!-- orbit:ai-card -->"` — `app.js:676`.
- `isAICard(node)` — `app.js:677` (`node?.type==="text"&&node.text.includes(...)`).
- `parseAICard(node)` / `buildAICardText(title,prompt)` — `app.js:678-682`.
- `nodeTitle` / `nodeSummary` / `nodeAIContent` / `noteKind` / `textMeta` —
  `app.js:691-731` (all branch on `node.type==="text"`).
- AI context preload `preloadAIFileInputs` / `aiFileContentCache` — `app.js:697-712`
  (already resolves file bodies for `file` nodes; notes flow through the catalog).

Rendering (`renderNodes`, `app.js:805-905`):

- text-node branch — `app.js:838-851` (AI-card UI, else markdown via
  `markdownToHTML`; kicker from `noteKind`/`textMeta`).
- file-node branches — component card (`app.js:852-877`), task (`app.js:878-879`),
  link, subcanvas, widget, journal (`app.js:898-901`), and a generic `FILE`
  fallback (`app.js:902`). A `notes/*.md` file-node currently falls into the
  generic FILE fallback; add a note branch before it.
- `markdownToHTML` (`app.js:735-760`) already skips `<!-- orbit:… -->` marker
  lines, so a note body's inert marker never renders.

Inspector (`renderInspector`, `app.js:1146-1228`; `applyInspectorField`, `app.js:1240-1262`):

- the only text-authoring surface is the Markdown textarea pushed for a `text`
  node: `else if(item.type==="text")fields.push({key:"text",label:"Markdown",control:"textarea",value:item.text});`
  (`app.js:1171`). Remove it (text becomes read-only).
- AI title/prompt fields (`aiTitle`/`aiPrompt`) mutate `item.text` in
  `applyInspectorField` (`app.js:1255`). For a file-backed operator they must
  write the note body instead.
- field edits dispatch on `detail.scope` (`"task"`, `"canvas"`, default). Add a
  `"note"` scope that writes the `.md` via the note repository.

Boot / wiring:

- `configureLifeRuntime(vault)` — `app.js:296-327` constructs `MemoryIndex`,
  `LifeIndexer`, `LifeQuery`, the two catalogs and the four repositories, each
  with `canvasPathFromId: id => { const record=workspace.canvases[id]; return record?canvasPathFor(record,workspace.rootId):null; }`.
  Add the note catalog + note repository here.
- repository globals — `app.js:158-165` (`let lifeIndexer/lifeQuery/taskRepository/…`).
- rebuild fan-out — every `Promise.all([lifeIndexer.rebuild(), componentCardCatalog.rebuild(), widgetCatalog.rebuild()])`
  must add the note catalog: `app.js:349` (boot), `app.js:392` (`rebuildLifeIndex`),
  `app.js:411` (`loadGraphStarter`), `app.js:1399` (import), and the reconcile at
  `app.js:1532`. The boot-failure reset list is `app.js:354`.
- mutation helpers — `enqueueMutation` (`app.js:217`), `flushPendingWorkspaceEdits`
  (`app.js:250`), `reloadCanvasDocuments` (`app.js:255`, re-reads a canvas from the
  vault and re-validates with `isCanvas`), `scheduleSave` (`app.js:244`).

Starter:

- `createGraphStarterWorkspace()` — `app.js:48-99`. Builds Home + four hub canvases
  synchronously; every guide/example card is an inline `text` node
  (`home-guide`, `inbox-guide`, `inbox-trip`, `projects-guide`, `cb-note`,
  `wiki-guide`, `wiki-budget`, `wiki-subscriptions`, `archive-guide`,
  `archive-portfolio`). Tasks/journals are already files
  (`STARTER_TASK_PATH` at `app.js:46`, journal via `journalPath`).
- `seedGraphStarterEntities()` — `app.js:100-118`. Runs against the configured
  repositories on first run (`bootCanvasApp` `app.js:343-346`) and in
  `loadGraphStarter` (`app.js:409`); seeds the task file at the exact path the
  starter references, then the journal. Seed the note files the same way.
- First-run path: `loadWorkspace()` (`app.js:203-210`) returns
  `createGraphStarterWorkspace()` on a true fresh install. The legacy
  `demoCanvas` (`app.js:24-42`, reached only via a pre-existing `orbit-canvas-v1`
  localStorage key through `freshWorkspace`/`loadDocument`) is a migration
  artifact — see Scope.

### Conventions to match

- Native strict ES modules, no build step, no dependencies. Named helpers,
  explicit side effects, validation at storage/import/AI boundaries. Do not
  mass-format `app.js`; keep its dense one-line-statement style.
- Repositories throw `SchemaError`/`ConflictError`/`PathError` from
  `storage/vault-errors.js` with a `code`. Match the task/journal/component-card
  error shapes (e.g. `CANVAS_NOT_FOUND`, `CANVAS_INVALID`, `CANVAS_ID_DUPLICATE`,
  `CANVAS_GEOMETRY_INVALID`).
- Canvas writes: mutate a parsed doc, re-validate with `isCanvas()`, serialize
  with `JSON.stringify(doc, null, 2) + "\n"`, write with the expected hash, then
  `indexer.indexFile(canvasPath, content, {})`.

## Commands you will need

| Purpose | Command | Expected on success |
|---|---|---|
| Syntax-check a module | `node --check <file>` | exit 0, no output |
| Note suite only | `node --test storage/note-repository.test.js` | all pass |
| Full storage suite | see "Done criteria" (the AGENTS.md §13 command + the new suite) | all pass, count bumped |
| Whitespace check | `git diff --check` | exit 0, no output |
| Serve for browser check | `python3 -m http.server 4173` | serves the repo root |
| Browser smoke | `node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline` | smoke suite passes (browser-pending behaviors) |

## Suggested executor toolkit

- The project `browser-check` skill at `.pi/skills/browser-check/` runs the
  headless smoke suite (read `.pi/skills/browser-check/SKILL.md` for recipes).
  Use it for the browser-pending verifications in the Test plan. It is local
  agent tooling, not part of the deployed shell.

## Scope

**In scope** (the only files you should modify or create):

- `storage/note-catalog.js` (create) — the synchronous note content projection.
- `storage/note-repository.js` (create) — the thin path-keyed note repository.
- `storage/note-repository.test.js` (create) — the Node contract suite.
- `app.js` — imports, globals, `configureLifeRuntime`, rebuild fan-out, rendering
  note branch, `isAICard`/`parseAICard`/`noteKind`/`nodeTitle`/`nodeSummary`/
  `nodeAIContent` body-resolution, `addNode` templates, double-click + note-tool
  handlers, `createAINote`, `runAICard` output, `runLocalAssistant` add-note path,
  inspector (remove text textarea, add note editor + note scope, AI fields write
  the body), `deleteSelection` note-placement removal, a `deleteNoteEverywhere`
  confirmed action, and the starter rewrite (`createGraphStarterWorkspace` +
  `seedGraphStarterEntities`).
- `AGENTS.md` — §4.2 (notes are file-backed, not text nodes), the repository map
  (add `storage/note-catalog.js`, `storage/note-repository.js`, and the `notes/`
  layout), and §13 (add the new suite to the `node --test` command and bump the
  published count).
- `docs/life-data.md` — note contract, ownership list, node-typing section.
- `docs/architecture.md` — ownership model and repository list.
- `docs/generative-canvas.md` — AI operator and AI note are file-backed
  (supersede the text-node statements at lines 65, 69, 77, 81).

**Out of scope** (do NOT touch, even though they look related):

- `docs/adr/0004-file-unified-node-model.md` and `CONTEXT.md` — already written
  and reconciled on this branch. Do not edit them.
- `storage/life-indexer.js`, `storage/memory-index.js`, `storage/canvas-validate.js`
  — no changes. Note placements stay OUT of the placements index; the validator
  keeps accepting `text`. Extending `extractCanvasPlacements` to track note
  placements is forbidden (it would force an `orbit-id` onto notes).
- The task / journal / habit / component-card / widget repositories and their
  phase suites — unchanged. Their contract tests must keep passing unmodified.
- The daily canvas, the `year→month→day` hierarchy, on-demand canvas creation,
  and the draining-queue UX — project 002. This plan only provides the
  path-generic re-place (drain) primitive.
- A typed `orbit-type: note` entity or an `orbit-id` for notes — YAGNI; path
  identity suffices.
- A note-rename UI and canvas file-node path rewriting on rename — not needed
  while notes are path-identified and never renamed.
- Auto-migration of existing inline text nodes — hard cut; never mass-rewrite
  user canvases.
- The legacy `demoCanvas` (`app.js:24-42`) and its `freshWorkspace`/`loadDocument`
  path — reached only via a pre-existing `orbit-canvas-v1` localStorage key. The
  hard cut deliberately does not migrate existing data; on a true fresh install
  `createGraphStarterWorkspace()` is used instead. Its text nodes now render
  read-only (the interop path). Leave it untouched; see Maintenance notes.
- `window.orbitCanvas` — do not add a note command unless a browser/integration
  need appears; never expose raw vault/index internals (AGENTS.md §12).

## Git workflow

- The human lead created this worktree and branch (`rugged-cougar`); never create,
  remove, or switch worktrees.
- Commit per logical unit (one commit per step or per tightly-related group).
  Match the repo's imperative commit style (see `git log --oneline -10`); keep
  messages scoped. Example: `storage: add path-keyed note repository and catalog`.
- The worker commits and reports only; the lead owns push and pull-request
  decisions. Never push, never open a PR, never amend or reset.
- Do not commit generated browser profiles, screenshots, or logs.

## Steps

Work in this order. Each step ends in a green `node --check` and, where noted, a
green test run, so the tree is never left broken between steps.

### Step 1: Create the note content projection (`storage/note-catalog.js`)

Create `storage/note-catalog.js` modeled directly on `storage/widget-catalog.js`.
It is a disposable synchronous projection: it preloads `notes/*.md` bodies and
their canvas placements, and is rebuilt from the vault (it never owns canonical
content). Export:

- `isNotePath(path)` → `typeof path==="string" && path.startsWith("notes/") && path.endsWith(".md")`.
- `class NoteCatalog` with constructor `{vault}` (throw `TypeError` if no vault),
  and methods `rebuild()`, `reconcile(_paths=[])` (delegate to `rebuild()`, matching
  the other catalogs), `getByPath(path)`, `getFallbackByPath(path)`, `notes()`,
  `diagnostics()`.

`rebuild()` lists `vault.list("notes/")` and `vault.list("canvases/")` in parallel
(the widget-catalog pattern). For each note file: record a case-fold-collision
diagnostic via `caseFoldKey` (code `NOTE_PATH_CASE_COLLISION`); read the body;
freeze a record `{path, hash, title, body, kind, placements:[]}` into `_byPath`.
On read/parse failure, freeze a fallback `{path, title, body:"", diagnostic}` into
`_fallbackByPath` and record a `NOTE_MALFORMED`/`NOTE_UNREADABLE` diagnostic — never
throw out of `rebuild()`. Then scan each canvas document (parse, `isCanvas()`;
record `CANVAS_MALFORMED` and skip on failure) and for every `file` node with
`isNotePath(node.file)`: push `{canvasPath, nodeId}` onto that note's `placements`
(if the note exists), else record a `NOTE_FILE_MISSING` diagnostic. A note with no
placement is fine (notes, like tasks, may have zero placements) — do NOT record an
orphan diagnostic for notes.

Title derivation: the first `# Heading` line in the body, else the path slug
(`path.split("/").at(-1).replace(/\.md$/,"")`). Kind: `"inbox"` if the body includes
`<!-- orbit:inbox -->`, `"reference"` if it includes `<!-- orbit:reference -->`,
`"ai"` if it includes `<!-- orbit:ai-card -->`, else `null`. (Reuse the marker
strings; define them locally or import from a shared spot — do not duplicate the
AI marker string drift-prone; a local const matching `app.js:676` is acceptable.)

**Verify**: `node --check storage/note-catalog.js` → exit 0.

### Step 2: Create the note repository (`storage/note-repository.js`)

Create `storage/note-repository.js`, a thin **path-keyed** repository that combines
the task/journal write discipline with the component-card path-scan placement
resolution. Export `class FileNoteRepository` with constructor
`{vault, index, indexer, catalog, canvasPathFromId, now = () => new Date().toISOString()}`.
The `catalog` port is the `NoteCatalog` from Step 1 (the component-card/widget
repositories take a catalog the same way); it supplies synchronous body reads and
path-scan placements. `index`/`indexer` are the task/journal ports, used to reindex
after every write.

Methods (all async except where noted):

- `allocatePath(title)` (sync or async): `const base = slugify(title)`; build the
  set of existing note path case-fold keys from `this.catalog.notes()` (and guard
  with `vault.stat` if you prefer the vault as source of truth); probe
  `notes/${base}.md`, then `notes/${base}-2.md`, `notes/${base}-3.md`, … until the
  candidate's `caseFoldKey` is free; return `normalizePath(candidate)`. Collision
  handling runs only at creation (notes are never renamed in v1).
- `createNote({title, body="", kind=null, color, canvasId, geometry})`: derive the
  title from `body`'s first `# Heading` when `title` is omitted; build the file
  content as ordinary Markdown (prepend the kind marker line
  `<!-- orbit:inbox -->` / `<!-- orbit:reference -->` / `<!-- orbit:ai-card -->`
  when `kind` is set, then the body); `const path = this.allocatePath(title)`;
  `await this.vault.write(path, content, {expectedHash:null})`;
  `await this.indexer.indexFile(path, content, {})`; `await this.catalog.reconcile([path])`;
  if `canvasId`, `const placement = await this.addPlacement(path, canvasId, {...geometry, color})`;
  return `{path, note:this.catalog.getByPath(path), placement}`.
- `addPlacement(path, canvasId, geometry={})`: resolve the canvas path via
  `canvasPathFromId`; `vault.stat` it (throw `CANVAS_NOT_FOUND` if absent); parse +
  `isCanvas()` (throw `CANVAS_INVALID`); validate geometry is finite integers with
  positive width/height (throw `CANVAS_GEOMETRY_INVALID`); allocate a unique node id
  (`geometry.id || \`node-${randomToken()}\``, throw `CANVAS_ID_DUPLICATE` on clash);
  push `{id, type:"file", file:path, x, y, width, height}` plus `color` when given;
  re-validate with `isCanvas()`; write `JSON.stringify(doc,null,2)+"\n"` with
  `expectedHash: stat.hash`; `await this.indexer.indexFile(canvasPath, content, {})`;
  `await this.catalog.reconcile([canvasPath])`; return `{canvasId, nodeId, canvasPath, path}`.
  Clone the exact shape of `FileJournalRepository.addPlacement`
  (`journal-event-repository.js:64-88`).
- `removePlacement(canvasId, nodeId)`: load + validate the canvas; filter the node
  and its incident edges; if nothing was removed return `{removed:false,…}`; write
  with the expected hash; reindex the canvas; reconcile the catalog; return
  `{removed:true, canvasId, nodeId}`. Clone `FileTaskRepository.removePlacement`
  (`task-repository.js:151-167`).
- `updateNote(path, body)`: `const stat = await this.vault.stat(path)` (throw
  `NOTE_NOT_FOUND` if absent); `const content = String(body)` (a note has no
  frontmatter, so a full-body rewrite under the expected hash is correct and
  preservation-first is trivially satisfied); `await this.vault.write(path, content,
  {expectedHash: stat.hash})`; `await this.indexer.indexFile(path, content, {})`;
  `await this.catalog.reconcile([path])`; return `this.catalog.getByPath(path)`.
- `deleteNote(path)`: `const stat = await this.vault.stat(path)` (throw
  `NOTE_NOT_FOUND`); resolve placements via `this.catalog.getByPath(path)?.placements || []`
  (path-scan; the placements index does NOT track notes); for each placement,
  `await this.removePlacement(canvasIdFromPath(placement.canvasPath), placement.nodeId)`
  — resolve the canvas id with the inverse of `canvasPathFromId`, or add a small
  `canvasIdFromPath` port if cleaner; then `await this.vault.remove(path, {expectedHash: stat.hash})`;
  `await this.indexer.removeFile(path)`; reconcile the catalog; return
  `{path, removedPlacements}`.
- `replacePlacement(path, fromCanvasId, nodeId, toCanvasId, geometry={})` — the
  **drain primitive** consumed by project 002: `const removed = await this.removePlacement(fromCanvasId, nodeId)`;
  if `removed.removed`, `const added = await this.addPlacement(path, toCanvasId, geometry)`;
  return `{removed, added: added||null, path}`. The source path is preserved, so
  identity is unchanged. Keep it path-generic (notes, tasks, and journals all drain
  the same way in 002).

Use `SchemaError`/`PathError` from `storage/vault-errors.js` with codes matching the
existing repositories. Add a local `randomToken()` (copy the task-repository helper).

**Verify**: `node --check storage/note-repository.js` → exit 0.

### Step 3: Write the Node contract suite (`storage/note-repository.test.js`)

Create `storage/note-repository.test.js` following the task phase suite
(`storage/phase5.test.js`) and the named-suite precedent
(`storage/component-card.test.js`). Use `import { test } from "node:test"` and
`assert from "node:assert/strict"`. Build a `setup()` that wires `MemoryVault` +
`MemoryIndex` + `LifeIndexer` + `NoteCatalog` + `FileNoteRepository` with a
canvas-id↔path map (copy the harness from `phase5.test.js:18-37`), and a
`seedCanvases(vault)` helper. Cover, at minimum:

1. `createNote` writes a canonical `notes/<slug>.md`, indexes it as a **untyped**
   source record (`index.getSourceFile(path)` has `entityType:null, parseStatus:"ok"`),
   and the body bytes match (marker + Markdown).
2. `createNote` with a `canvasId` adds a standard `file`-node placement; the canvas
   still passes `isCanvas()`; the node's `file` equals the note path.
3. **Collision allocation**: creating two notes with the same title yields
   `notes/new-thought.md` then `notes/new-thought-2.md` (and a third `-3`); a
   case-fold collision (e.g. `New Thought` vs `new thought`) also allocates a suffix.
4. `addPlacement` rejects a duplicate node id with `CANVAS_ID_DUPLICATE` and a
   missing canvas with `CANVAS_NOT_FOUND`.
5. `removePlacement(canvasId, nodeId)` removes the node and its incident edges,
   keeps the canvas valid, and leaves the note file intact.
6. `updateNote(path, body)` rewrites the body under the expected hash; a stale
   expected hash (write behind the repository's back) rejects.
7. `deleteNote(path)` removes every placement (place the note on two canvases first),
   then removes the file; the canvases stay valid.
8. `replacePlacement(path, fromCanvas, nodeId, toCanvas, geometry)` (drain) removes
   the placement on the source canvas and adds one for the **same path** on the
   target; the file is untouched and identity (path) is unchanged.
9. **Regression pin (mandatory)**: after `createNote(..., {canvasId})`, assert
   `index.allPlacements()` and `index.placementsForEntity(<anything>)` contain **no**
   row for the note path (note placements are not tracked by the disposable
   placements index), while `deleteNote` and `replacePlacement` still resolve the
   placement by scanning canvas documents (i.e. they work without the placements
   index). This is the behavior the spec's Testing Decision #2 pins.

**Verify**: `node --test storage/note-repository.test.js` → all pass.

### Step 4: Wire the catalog + repository into the app boot (`app.js`)

- Add imports near `app.js:1-18`: `import { NoteCatalog, isNotePath } from "./storage/note-catalog.js";`
  and `import { FileNoteRepository } from "./storage/note-repository.js";`.
- Add globals beside `app.js:158-165`: `let noteCatalog=null;` and `let noteRepository=null;`.
- In `configureLifeRuntime(vault)` (`app.js:296-327`), construct
  `noteCatalog = new NoteCatalog({vault});` and
  `noteRepository = new FileNoteRepository({vault, index:lifeIndex, indexer:lifeIndexer, catalog:noteCatalog, canvasPathFromId: id => { const record=workspace.canvases[id]; return record?canvasPathFor(record,workspace.rootId):null; }});`.
- Add `noteCatalog.rebuild()` to every rebuild fan-out: `app.js:349`, `app.js:392`,
  `app.js:411`, `app.js:1399`, and the reconcile at `app.js:1532` (use
  `noteCatalog?.rebuild()` where the surrounding code uses optional chaining).
- Add `noteCatalog=null; noteRepository=null;` to the boot-failure reset list at
  `app.js:354`.

**Verify**: `node --check app.js` → exit 0.

### Step 5: Render + edit a file-backed note card (`app.js`)

- In `renderNodes` (`app.js:805-905`), add a note branch **before** the generic
  `FILE` fallback (`app.js:902`). Detect `const notePath = node.type==="file" && isNotePath(node.file);`.
  Resolve `const note = noteCatalog?.getByPath(node.file) || noteCatalog?.getFallbackByPath(node.file) || {path:node.file, title:node.file.split("/").pop(), body:"", diagnostic:\`Note file is unavailable: ${node.file}\`};`.
  If the note body carries the AI marker, render the AI-operator card UI (reuse the
  existing ai-card innerHTML at `app.js:839-840`, but read title/prompt from the
  resolved body via the refactored `parseAICard` from Step 6). Otherwise render
  `<div class="node-kicker">…</div><div class="node-body">${markdownToHTML(note.body)}</div>`
  with the kicker derived from kind (`INBOX · capture` / `REFERENCE · wiki`) or
  `textMeta(node)` (color taxonomy), exactly as text notes do today. A missing or
  damaged file renders a readable fallback with `note.diagnostic` (the
  component-card/widget repair pattern, `renderFallbackFileContent` at `app.js:790`);
  never render empty and never overwrite the file.
- In the inspector (`renderInspector`, `app.js:1146-1228`): for a selected note
  file-node that is NOT an AI card, push
  `{key:"noteBody", label:"Markdown", control:"textarea", value:note.body, scope:"note", notePath:item.file}`
  and a `{intent:"delete-note", label:"Delete note everywhere", notePath:item.file, danger:true}`
  action. Title the inspector "Note".
- In `applyInspectorField` (`app.js:1240-1262`), add a `detail.scope==="note"` branch:
  compute the new body (for `noteBody`, the value; for `aiTitle`/`aiPrompt`, rebuild
  via `buildAICardText` against the current note body), then write it through the
  repository with the same debounce/flush discipline the task scope uses
  (`scheduleTaskFieldUpdate`/`flushTaskFieldUpdate` pattern at `app.js:1244-1249` —
  add an analogous `scheduleNoteUpdate(path, body)` / `flushNoteUpdate(path)` pair
  that calls `noteRepository.updateNote`, then `reloadCanvasDocuments([currentCanvasId])`
  and `renderNodes()`). Guard on `canonicalWritable`.
- Wire the `delete-note` inspector action (in the `balaur-inspector-action` handler,
  `app.js:1271-1276`) to a new `deleteNoteEverywhere(path)` confirmed action modeled
  on `deleteTaskEverywhere` (`app.js:1300-1311`): confirm, resolve affected canvases
  from `noteCatalog.getByPath(path).placements`, `flushPendingWorkspaceEdits()`,
  `enqueueMutation(()=>noteRepository.deleteNote(path))`, `reloadCanvasDocuments(affected)`,
  clear selection, render, toast.
- In `deleteSelection` (`app.js:1313-1330`), extend the `taskId` branch so a note
  file-node removes only its placement (the file stays): when the selected node is a
  note (`isNotePath(node.file)`), `canonicalMutation=true` and
  `await noteRepository.removePlacement(currentCanvasId, selected.id)` then
  `reloadCanvasDocuments([currentCanvasId])`, mirroring the task branch.

**Verify**: `node --check app.js` → exit 0.

### Step 6: Migrate AI operator detection/parsing to the note body (`app.js`)

- Refactor `parseAICard` (`app.js:678-681`) to operate on a body string:
  `function parseAICardBody(body){…existing line logic on body…}` and
  `function parseAICard(node){ return parseAICardBody(noteCatalog?.getByPath(node.file)?.body || ""); }`.
  Keep `buildAICardText(title,prompt)` (`app.js:682`) unchanged.
- Change `isAICard` (`app.js:677`) to detect a file-backed note:
  `function isAICard(node){ return node?.type==="file" && isNotePath(node.file) && Boolean(noteCatalog?.getByPath(node.file)?.body?.includes(AI_CARD_MARKER)); }`.
- Update `noteKind` (`app.js:716-721`) to read kind from the catalog for a note
  file-node (`noteCatalog?.getByPath(node.file)?.kind`), returning `"inbox"`/
  `"reference"`/`null`.
- Update `nodeTitle` (`app.js:691`): in the `file` branch, if `isNotePath(node.file)`
  return `noteCatalog?.getByPath(node.file)?.title || node.file.split("/").pop()`
  before the entity lookup. Update `nodeSummary` (`app.js:724-731`) to derive the
  one-line summary from a note body with the same heading-first / first-non-marker-line
  convention used for text. Update `nodeAIContent` (`app.js:693-700`): if
  `isNotePath(node.file)` return `noteCatalog?.getByPath(node.file)?.body || ""`
  (synchronous; no per-card vault read).

The AI security boundary does NOT change: operators still propose allowlisted
operations; output never executes host code; context assembly already resolves file
bodies (`preloadAIFileInputs`, `app.js:697-712`). If preserving this boundary would
require a change to `ai/generated-operations.js` or the allowlist, STOP and report.

**Verify**: `node --check app.js` → exit 0.

### Step 7: Migrate authoring templates + double-click to file-backed notes (`app.js`)

- Add one shared app-level helper (avoid duplication) near `addNode`:
  `async function createNoteOnCanvas({title, body, kind=null, color, geometry, canvasId=currentCanvasId})`
  that guards on `canonicalWritable`/`noteRepository`, `await flushPendingWorkspaceEdits()`,
  `const result = await enqueueMutation(()=>noteRepository.createNote({title, body, kind, color, canvasId, geometry}))`,
  `await reloadCanvasDocuments([canvasId])`, and returns `result`
  (`{path, note, placement}`).
- Rewrite `addNode(kind, point)` (`app.js:1103-1124`) so it is `async`. Replace the
  `type:"text"` presets with body+color+size specs: each of
  `note/inbox/reference/goal/habit/project/ai` keeps its current Markdown (including
  the inert marker for inbox/reference/ai) as the **body** and its current color and
  width/height as the placement geometry, with `kind` set for inbox/reference/ai.
  For those kinds, compute the click-point geometry (reuse the existing
  `center`/width/height math) and `await createNoteOnCanvas(...)`, then select
  `result.placement.nodeId` and render. `widget`/`group` keep their existing
  synchronous behavior; `subcanvas`/`task` keep their early returns. After this, no
  branch of `addNode` authors a `text` node.
- Make the double-click handler (`app.js:1074-1081`) and the note-tool pointerdown
  (`app.js:1071`) `await addNode(...)`. Keep the `nodeAtClientPoint` geometry
  hit-test that prevents spawning over a card. The double-click now creates a
  `.md` file-node (default body `# New thought\nStart writing here…`, color `"2"`).
- Update the remaining `addNode` callers to tolerate the async return:
  `runAddKind` (`app.js:1769`) and `$("#newGroup").onclick` (`app.js:1784`).

**Verify**: `node --check app.js` → exit 0. Then `grep -n 'type:"text"' app.js` —
the only remaining matches must be the legacy `demoCanvas` (`app.js:24-42`) and the
read-only interop render/validator paths; no interactive authoring path may remain
(see STOP conditions).

### Step 8: Migrate the AI one-shot note, the AI operator output, and the local parser (`app.js`)

- `createAINote` (`app.js:1743`): replace the `documentData.nodes.push(textNode)` with
  `const result = await createNoteOnCanvas({title:"AI note", body:generated, color:"5", geometry:{…existing x/y/width/height…}})`;
  select `result.placement.nodeId`; keep the dialog/result/toast flow.
- `runAICard` output (`app.js:1759-1760`): the output becomes a file-backed note.
  Resolve the existing output via the `AI output` edge:
  `let outputEdge=(documentData.edges||[]).find(e=>e.fromNode===card.id&&e.label==="AI output"); let outputNode=outputEdge&&documentData.nodes.find(n=>n.id===outputEdge.toNode&&n.type==="file"&&isNotePath(n.file));`.
  If absent, `await createNoteOnCanvas({title:\`${config.title} — output\`, body:generated, color:"5", canvasId:currentCanvasId, geometry:{x:card.x+card.width+90, y:card.y, width:380, height:240}})`
  and connect the new node with the reserved `AI output` edge
  (`{id:uid("edge"), fromNode:card.id, fromSide:"right", toNode:<newNodeId>, toSide:"left", toEnd:"arrow", color:"5", label:"AI output"}`),
  then `scheduleSave()`. If present, `await noteRepository.updateNote(outputNode.file, generated)`
  (stable output reuse), reconcile, and re-render. **Preserve debouncing, queued
  reruns, and cycle detection unchanged** (`scheduleAICard`/`scheduleChangedAICards`/
  `aiCardHasCycle`, `app.js:1726-1737`); only the output's storage shape changes.
- `runLocalAssistant` add-note path (`app.js:1655`): replace the
  `applyCanvasOperations([{type:"node.add", textNode}])` with
  `await createNoteOnCanvas({title, body:preset[1], color:preset[0], geometry:{…center…}})`
  for the goal/habit/project/note kinds. Keep the response message.

**Verify**: `node --check app.js` → exit 0.

### Step 9: Make `text` read-only in the UI (`app.js`)

- Remove the inspector's text-editing textarea: delete the
  `else if(item.type==="text")fields.push({key:"text",label:"Markdown",control:"textarea",value:item.text});`
  branch at `app.js:1171`. Replace it with a read-only note, e.g.
  `else if(item.type==="text")notes.push({text:"Imported text node · read-only. Balaur authors file-backed notes (ADR-0004)."});`
  so a selected foreign `text` node shows its content (rendered on the canvas) but
  offers no edit field. The `text` render branch in `renderNodes`
  (`app.js:844-851`) stays so imported documents remain visible.
- Remove the now-dead `aiTitle`/`aiPrompt` mutation of `item.text` in
  `applyInspectorField` (`app.js:1255`); those fields are now handled by the
  `scope==="note"` branch from Step 5 (an AI operator is a note file-node).

**Verify**: `node --check app.js` → exit 0. `grep -n 'item.text=' app.js` → no
matches (no code path mutates a text node's content).

### Step 10: Rewrite the graph starter as a greenfield file-backed space (`app.js`)

- In `createGraphStarterWorkspace()` (`app.js:48-99`), replace every inline `text`
  node (`home-guide`, `inbox-guide`, `inbox-trip`, `projects-guide`, `cb-note`,
  `wiki-guide`, `wiki-budget`, `wiki-subscriptions`, `archive-guide`,
  `archive-portfolio`) with a standard `file` node referencing a stable
  `notes/*.md` path. Define those paths as module-level consts beside
  `STARTER_TASK_PATH` (`app.js:46`), e.g.
  `const STARTER_NOTES = { homeGuide:"notes/start-here-your-life-as-a-graph.md", inboxTrip:"notes/autumn-city-break-idea.md", … };`
  (slug each title through the same convention). Keep every node's `id`, geometry,
  and `color` (including the dormant `#6c757d` archive color), and keep every edge
  (`e-inbox-filed`, `e-cb-partof`, `e-wiki-relates`) with its label and color — only
  the node storage changes from `type:"text"` to `type:"file", file:<path>`. Tasks
  (`cb-task` → `STARTER_TASK_PATH`) and the journal (`wiki-journal` → `journalPath`)
  are unchanged.
- In `seedGraphStarterEntities()` (`app.js:100-118`), seed each starter note file at
  the exact path its `file` node references, via
  `noteRepository.createNote({path:STARTER_NOTES.x, body:"…", kind:…})` — exactly the
  way the task is seeded at `STARTER_TASK_PATH` today (wrap in try/catch so re-seeding
  is idempotent). The inbox capture carries the `<!-- orbit:inbox -->` marker, the
  reference pages carry `<!-- orbit:reference -->`. Because `createNote` allocates a
  path from the title, pass the explicit `path` (add an `input.path` override to
  `createNote` in Step 2 if not already present — the task repository supports
  `input.path` the same way, `task-repository.js:69`).
- There is no compatibility path and no migration of the old inline starter (hard cut).

**Verify**: `node --check app.js` → exit 0.

### Step 11: Update documentation in the same change

- `AGENTS.md` §4.2 (`AGENTS.md:99-115`): replace "Inbox notes and reference pages
  are standard text nodes with inert markers" and "AI operators are standard text
  nodes…" with the file-backed wording (notes/inbox/reference/AI are
  path-identified `notes/*.md` files placed by standard `file` nodes; the inert
  marker lives in the file body; `text` remains a valid, rendered, read-only interop
  type that Balaur never authors). Reference ADR-0004.
- `AGENTS.md` repository map (`AGENTS.md:60-72`): add `storage/note-catalog.js`
  (disposable synchronous note content projection) and `storage/note-repository.js`
  (FileNoteRepository: canonical note files, placements, and the drain primitive),
  and note the `notes/` layout alongside `tasks/`, `habits/`, etc.
- `AGENTS.md` §13 (`AGENTS.md:283-292`): add `storage/note-repository.test.js` to
  the `node --test` command and bump the published count. Run the full suite first
  to get the real new total, then update the "**172 tests**" sentence to the actual
  number and name the new note-repository suite as the addition.
- `docs/life-data.md`: add the note contract (path-identified `notes/*.md`, no
  mandatory frontmatter/`orbit-id`, kind is an inert body marker, indexer treats it
  as valid untyped Markdown), add notes to the ownership list, and update the
  node-typing section (notes are file-backed; `text` is read-only interop).
- `docs/architecture.md`: update the ownership model and the repository list to
  include the note catalog + note repository.
- `docs/generative-canvas.md`: supersede the text-node statements at lines 65, 69,
  77, 81 — an AI operator and its output are standard `file` nodes referencing
  `notes/*.md`; the portable compatibility marker lives in the file body; the
  security boundary is unchanged.

**Verify**: `git diff --check` → exit 0. Prose only; no source changed in this step.

### Step 12: Full verification gate

Run, in order, and confirm each:

1. `node --check` on every touched module:
   `node --check app.js storage/note-catalog.js storage/note-repository.js storage/note-repository.test.js`
   → exit 0 each.
2. The full AGENTS.md §13 suite **including the new suite** (the exact command now
   in AGENTS.md after Step 11):
   ```bash
   node --test \
     storage/phase1.test.js storage/phase2.test.js storage/phase3.test.js \
     storage/phase4.test.js storage/phase4-backup.test.js storage/phase5.test.js \
     storage/phase7.test.js storage/phase8.test.js storage/phase9.test.js \
     storage/phase10.test.js storage/phase-query.test.js \
     storage/note-repository.test.js
   ```
   → all pass; the total equals the bumped count published in AGENTS.md §13.
3. `git diff --check` → exit 0.
4. Browser smoke (browser-pending behaviors): serve with
   `python3 -m http.server 4173`, then
   `node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline`.
   Confirm: no uncaught console errors; the file-backed starter renders every
   document node; double-clicking empty background creates a file-backed note
   (a `notes/*.md` appears and a `file` node renders it); editing a note writes the
   `.md`; an imported `text` node renders read-only (no edit textarea). **Label these
   as browser-pending in your report; the Node suite does not verify them.** If the
   skill is absent, fall back to the manual baseline in AGENTS.md §13 and say so.

## Test plan

- New suite: `storage/note-repository.test.js` (the cases listed in Step 3,
  including the mandatory regression pin that note placements are absent from
  `index.allPlacements()` yet drain/delete still resolve them by canvas path scan).
- Structural pattern: `storage/phase5.test.js` (task repository harness) and
  `storage/component-card.test.js` (named-suite + catalog precedent).
- No existing suite is modified. The full phase suite must keep passing unmodified;
  in particular `phase3.test.js:43-46` (untyped `notes/*.md`), `phase4.test.js:297`
  (`file` node `subpath` rejected), and the `phase8.test.js` journal-placement tests
  are the guardrails that prove the indexer and validator are untouched.
- Browser-pending behaviors (double-click creates a file-backed note; note edit
  writes the `.md`; AI operator/output as files; file-backed starter renders;
  imported `text` nodes render read-only) are verified through the headless browser
  smoke skill and remain browser-pending until run.

## Done criteria

ALL must hold:

- [ ] `node --check` exits 0 for `app.js`, `storage/note-catalog.js`,
      `storage/note-repository.js`, `storage/note-repository.test.js`.
- [ ] The full AGENTS.md §13 `node --test` command (with `storage/note-repository.test.js`
      added) exits 0; the published count in AGENTS.md §13 matches the real total.
- [ ] `grep -n 'type:"text"' app.js` shows matches ONLY in the legacy `demoCanvas`
      (`app.js:24-42`) and the read-only interop render path — no interactive
      authoring path produces a `text` node.
- [ ] `grep -n 'item.text=' app.js` returns no matches (no text-node mutation).
- [ ] `storage/life-indexer.js`, `storage/memory-index.js`, `storage/canvas-validate.js`,
      and the task/journal/habit/component-card/widget repositories are unmodified
      (`git diff --stat` shows none of them).
- [ ] The new suite contains the regression pin: note placements are absent from
      `index.allPlacements()`, and drain/delete resolve them by canvas path scan.
- [ ] AGENTS.md §4.2 + repository map + §13, docs/life-data.md, docs/architecture.md,
      and docs/generative-canvas.md are updated.
- [ ] `git diff --check` exits 0; no files outside the in-scope list are modified.
- [ ] Browser smoke run (or documented manual fallback); browser-pending behaviors
      labeled as such in the report.

## STOP conditions

Stop and report back (do not improvise) if:

- A canvas document the note repository writes fails `isCanvas()` (the repository
  must re-validate before every canvas write; a failure means the placement shape is
  wrong — do not weaken the validator to make it pass).
- An existing repository/indexer/validator contract test regresses (any phase suite
  or `component-card.test.js` / `widget-repository.test.js` failure). The fix must
  not come from editing those suites or those modules.
- Preserving the AI security boundary (operators propose allowlisted operations;
  output never executes host code; `ai/generated-operations.js` allowlist) would
  require changing that boundary. Report instead.
- Implementing a step would force an `orbit-id` or mandatory frontmatter onto notes,
  or would require extending `extractCanvasPlacements` / the placements index to
  track note placements. That contradicts ADR-0004 and Testing Decision #2; report.
- The code at the locations in "Current state" does not match the excerpts (the
  codebase drifted since `9ab2c1d`).
- A step's verification fails twice after a reasonable fix attempt.
- A change appears to require touching an out-of-scope file (e.g. the validator,
  the indexer, another repository, ADR-0004, or `CONTEXT.md`).
- `pwd` does not match the assigned worktree path, `git branch --show-current` is
  `main`, `git worktree list` does not confirm this worktree, or `git status --short`
  shows unexpected modifications before edits begin.

## Maintenance notes

- **Project 002 consumes `FileNoteRepository.replacePlacement(path, fromCanvasId,
  nodeId, toCanvasId, geometry)`** as the drain primitive. Keep it path-generic and
  path-keyed so notes, tasks, and journals drain uniformly. If 002 wants a shared
  placement helper across repositories, extract it then; do not pre-build it here.
- **The `NoteCatalog` is a projection.** It is rebuilt at boot and reconciled after
  every repository write; it never owns canonical content. If a future change adds a
  note write path, it must reconcile the catalog and reindex, or render/AI detection
  will read stale bodies.
- **Note rename is intentionally absent.** Notes are path-identified and the app
  exposes no rename, so placements never go stale from an app-driven rename. If a
  rename UI is ever wanted, reuse the component-card `_rewritePlacementPaths` pattern
  (`component-card-repository.js:108-124`); do not add an `orbit-id` for it.
- **The legacy `demoCanvas` (`app.js:24-42`)** is the one remaining text-producing
  path. It is reached only via a pre-existing `orbit-canvas-v1` localStorage key
  (the pre-ADR migration fallback); a true fresh install uses
  `createGraphStarterWorkspace()`. The hard cut deliberately does not migrate it, and
  its text nodes now render read-only. If "no code path authors a text node" must be
  absolute, point `loadDocument`'s fallback at an empty canvas in a follow-up — that
  is a legacy-migration decision, out of scope here.
- **Reviewer focus**: the note repository's expected-hash preconditions and
  reindex-after-write discipline; the `isAICard`/`parseAICard` body-resolution (no
  residual `node.text` reads on file nodes); the inspector's `scope==="note"` write
  path (debounce/flush, never mutates the canvas document's content); the starter's
  path consts matching the seeded files; and that the placements index, indexer, and
  validator are byte-for-byte unchanged.

## Next step

- Recommended executor tier: premium (large, cross-cutting migration touching
  authoring, AI, rendering, and the inspector; judgment-heavy).
- Recommended model: confirm against `para/resources/model-registry.md` (top tier).
- Estimated complexity: L.
- After implementation: run `/review` (review-standards) on this worktree, then the
  two-axis feature review; project 002-daily-canvas-journal unblocks once this lands.
