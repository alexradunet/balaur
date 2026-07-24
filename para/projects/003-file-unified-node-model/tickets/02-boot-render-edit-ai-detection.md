---
phase: ticket
status: done
project: 003-file-unified-node-model
ticket: 02
blocked-by: [01]
worker: "bb8ec5ea-61d4-4767-b2aa-a053276e32a8"
branch: "003-ticket-02"
shared-blast-radius: true
---

# Ticket 02: Migrate — boot wiring, note render/edit, and AI body-detection

## What to build

Turn the note foundation on in the app and make file-backed notes first-class in the
UI: they render, they edit (writing the `.md`), and AI-operator detection reads the
resolved note body instead of a text node's `text` field. This is plan Steps 4-6 in
one ticket because they are mutually coupled: the render branch's AI-card UI (Step 5)
calls the refactored `parseAICardBody` (Step 6), and both read the `noteCatalog` /
`noteRepository` globals that boot wiring (Step 4) creates.

Boot wiring (Step 4, `app.js`): add `import { NoteCatalog, isNotePath } from
"./storage/note-catalog.js";` and `import { FileNoteRepository } from
"./storage/note-repository.js";` near the top imports; add `let noteCatalog=null;` and
`let noteRepository=null;` beside the repository globals (`app.js:158-165`); in
`configureLifeRuntime(vault)` (`app.js:296-327`) construct `noteCatalog = new
NoteCatalog({vault});` and `noteRepository = new FileNoteRepository({vault,
index:lifeIndex, indexer:lifeIndexer, catalog:noteCatalog, canvasPathFromId: id => {
const record=workspace.canvases[id]; return record?canvasPathFor(record,
workspace.rootId):null; }});`; add `noteCatalog.rebuild()` to every rebuild fan-out
(`app.js:349`, `392`, `411`, `1399`, and the reconcile at `1532`, using
`noteCatalog?.rebuild()` where the surrounding code uses optional chaining); add
`noteCatalog=null; noteRepository=null;` to the boot-failure reset list (`app.js:354`).

Render + edit a file-backed note card (Step 5, `app.js`): in `renderNodes`
(`app.js:805-905`) add a note branch BEFORE the generic `FILE` fallback (`app.js:902`).
Detect `isNotePath(node.file)`; resolve `noteCatalog?.getByPath(node.file) ||
noteCatalog?.getFallbackByPath(node.file) || {path, title, body:"", diagnostic}`. If
the body carries the AI marker, render the AI-operator card UI (reuse the existing
ai-card innerHTML at `app.js:839-840`, reading title/prompt from the resolved body via
the refactored `parseAICard` from Step 6). Otherwise render the kicker + Markdown body
(kicker from kind — `INBOX · capture` / `REFERENCE · wiki` — or `textMeta(node)`),
exactly as text notes do today. A missing/damaged file renders a readable fallback with
`note.diagnostic` (the `renderFallbackFileContent` repair pattern at `app.js:790`);
never render empty and never overwrite the file. In the inspector (`renderInspector`,
`app.js:1146-1228`): for a selected note file-node that is NOT an AI card, push a
Markdown textarea `{key:"noteBody", label:"Markdown", control:"textarea", value:note.body,
scope:"note", notePath:item.file}` and a `{intent:"delete-note", label:"Delete note
everywhere", notePath:item.file, danger:true}` action; title the inspector "Note". In
`applyInspectorField` (`app.js:1240-1262`) add a `detail.scope==="note"` branch that
computes the new body (for `noteBody`, the value; for `aiTitle`/`aiPrompt`, rebuild via
`buildAICardText` against the current note body) and writes it through the repository
with the same debounce/flush discipline the task scope uses — add an analogous
`scheduleNoteUpdate(path, body)` / `flushNoteUpdate(path)` pair that calls
`noteRepository.updateNote`, then `reloadCanvasDocuments([currentCanvasId])` and
`renderNodes()`; guard on `canonicalWritable`. Wire the `delete-note` inspector action
(in the `balaur-inspector-action` handler, `app.js:1271-1276`) to a new
`deleteNoteEverywhere(path)` confirmed action modeled on `deleteTaskEverywhere`
(`app.js:1300-1311`): confirm, resolve affected canvases from
`noteCatalog.getByPath(path).placements`, `flushPendingWorkspaceEdits()`,
`enqueueMutation(()=>noteRepository.deleteNote(path))`, `reloadCanvasDocuments(affected)`,
clear selection, render, toast. In `deleteSelection` (`app.js:1313-1330`), when the
selected node is a note (`isNotePath(node.file)`), set `canonicalMutation=true` and
`await noteRepository.removePlacement(currentCanvasId, selected.id)` then
`reloadCanvasDocuments([currentCanvasId])` — removing only the placement, the file
stays — mirroring the task branch.

Migrate AI detection/parsing to the note body (Step 6, `app.js`): refactor
`parseAICard` (`app.js:678-681`) into `parseAICardBody(body)` (the existing line logic
on a body string) plus `parseAICard(node){ return parseAICardBody(noteCatalog?.getByPath(node.file)?.body || ""); }`; keep `buildAICardText(title,prompt)` unchanged.
Change `isAICard` (`app.js:677`) to detect a file-backed note: `node?.type==="file" &&
isNotePath(node.file) && Boolean(noteCatalog?.getByPath(node.file)?.body?.includes(AI_CARD_MARKER))`.
Update `noteKind` (`app.js:716-721`) to read kind from the catalog for a note file-node;
update `nodeTitle` (`app.js:691`) so the `file` branch returns the catalog title (else
the path slug) for a note before the entity lookup; update `nodeSummary`
(`app.js:724-731`) to derive the one-line summary from a note body with the same
heading-first / first-non-marker-line convention; update `nodeAIContent`
(`app.js:693-700`) to return `noteCatalog?.getByPath(node.file)?.body || ""` for a note
(synchronous; no per-card vault read). The AI security boundary does NOT change.

## Acceptance criteria

- [ ] `noteCatalog` and `noteRepository` are constructed in `configureLifeRuntime`, added to all five rebuild fan-outs (`app.js:349`, `392`, `411`, `1399`, `1532`), and cleared in the boot-failure reset (`app.js:354`).
- [ ] A file-backed note renders its Markdown (kicker from kind or color taxonomy) via the synchronous `NoteCatalog` projection; no async vault read happens per card during render (AGENTS.md §5).
- [ ] A note whose body carries the AI marker renders the AI-operator card UI, reading title/prompt from the resolved body.
- [ ] A missing/damaged note file renders a readable fallback with its diagnostic and is never overwritten.
- [ ] The inspector offers a Markdown editor (`scope==="note"`) for a selected non-AI note file-node, plus a "Delete note everywhere" danger action; editing writes the `.md` through `noteRepository.updateNote` with debounce/flush, then reindex + reconcile + re-render; editing never mutates the canvas document's content.
- [ ] `deleteNoteEverywhere(path)` confirms, resolves affected canvases by path scan, deletes the file and all placements, reloads, and toasts.
- [ ] `deleteSelection` removes only a note's placement (the file stays), mirroring the task branch.
- [ ] `isAICard`, `parseAICard`/`parseAICardBody`, `noteKind`, `nodeTitle`, `nodeSummary`, and `nodeAIContent` all resolve note bodies from the catalog; there are no residual `node.text` reads on `file` nodes.
- [ ] The AI security boundary is unchanged (operators propose allowlisted operations; output never executes host code; `ai/generated-operations.js` untouched). If preserving it would require changing that boundary, STOP and report.
- [ ] `node --check app.js` exits 0.
- [ ] `storage/life-indexer.js`, `storage/memory-index.js`, `storage/canvas-validate.js`, and the task/journal/habit/component-card/widget repositories remain unchanged.
- [ ] `git diff --check` exits 0; only `app.js` is modified in this ticket.

## Blocked by

Ticket 01 (Expand — note catalog, note repository, and contract suite).
