---
phase: ticket
status: done
project: 003-file-unified-node-model
ticket: 03
blocked-by: [02]
worker: "6f95acfd-1a2f-4b40-abf1-4bcca165f1c5"
branch: "003-ticket-03"
shared-blast-radius: true
---

# Ticket 03: Migrate — authoring templates, double-click, AI output, and local parser

## What to build

Every interactive authoring path produces a file-backed note; after this ticket no
interactive code path authors a `text` node. This is plan Steps 7-8 in one ticket:
both are the "authoring produces files" milestone, both use one shared
`createNoteOnCanvas` helper, and both live in the same authoring/AI region of
`app.js`.

Add one shared app-level helper near `addNode` (avoid duplication): `async function
createNoteOnCanvas({title, body, kind=null, color, geometry, canvasId=currentCanvasId})`
that guards on `canonicalWritable`/`noteRepository`, `await
flushPendingWorkspaceEdits()`, `const result = await
enqueueMutation(()=>noteRepository.createNote({title, body, kind, color, canvasId,
geometry}))`, `await reloadCanvasDocuments([canvasId])`, and returns `result`
(`{path, note, placement}`).

Migrate authoring templates + double-click (Step 7, `app.js`): rewrite `addNode(kind,
point)` (`app.js:1103-1124`) so it is `async`. Replace the `type:"text"` presets with
body+color+size specs: each of `note/inbox/reference/goal/habit/project/ai` keeps its
current Markdown (including the inert marker for inbox/reference/ai) as the BODY and
its current color and width/height as the placement geometry, with `kind` set for
inbox/reference/ai. For those kinds, compute the click-point geometry (reuse the
existing center/width/height math) and `await createNoteOnCanvas(...)`, then select
`result.placement.nodeId` and render. `widget`/`group` keep their existing synchronous
behavior; `subcanvas`/`task` keep their early returns. After this, NO branch of
`addNode` authors a `text` node. Make the double-click handler (`app.js:1074-1081`) and
the note-tool pointerdown (`app.js:1071`) `await addNode(...)`, keeping the
`nodeAtClientPoint` geometry hit-test that prevents spawning over a card; double-click
now creates a `.md` file-node (default body `# New thought\nStart writing here…`,
color `"2"`). Update the remaining `addNode` callers to tolerate the async return:
`runAddKind` (`app.js:1769`) and `$("#newGroup").onclick` (`app.js:1784`).

Migrate the AI one-shot note, the AI operator output, and the local parser (Step 8,
`app.js`): `createAINote` (`app.js:1743`) replaces the `documentData.nodes.push(textNode)`
with `const result = await createNoteOnCanvas({title:"AI note", body:generated,
color:"5", geometry:{…existing x/y/width/height…}})`, selects `result.placement.nodeId`,
and keeps the dialog/result/toast flow. `runAICard` output (`app.js:1759-1760`) becomes
a file-backed note: resolve the existing output via the `AI output` edge
(`let outputEdge=(documentData.edges||[]).find(e=>e.fromNode===card.id&&e.label==="AI output");
let outputNode=outputEdge&&documentData.nodes.find(n=>n.id===outputEdge.toNode&&n.type==="file"&&isNotePath(n.file));`);
if absent, `await createNoteOnCanvas({title:\`${config.title} — output\`, body:generated,
color:"5", canvasId:currentCanvasId, geometry:{x:card.x+card.width+90, y:card.y,
width:380, height:240}})` and connect the new node with the reserved `AI output` edge
(`{id:uid("edge"), fromNode:card.id, fromSide:"right", toNode:<newNodeId>, toSide:"left",
toEnd:"arrow", color:"5", label:"AI output"}`), then `scheduleSave()`; if present,
`await noteRepository.updateNote(outputNode.file, generated)` (stable output reuse),
reconcile, and re-render. Preserve debouncing, queued reruns, and cycle detection
UNCHANGED (`scheduleAICard`/`scheduleChangedAICards`/`aiCardHasCycle`,
`app.js:1726-1737`); only the output's storage shape changes. `runLocalAssistant`
add-note path (`app.js:1655`) replaces the `applyCanvasOperations([{type:"node.add",
textNode}])` with `await createNoteOnCanvas({title, body:preset[1], color:preset[0],
geometry:{…center…}})` for the goal/habit/project/note kinds, keeping the response
message.

## Acceptance criteria

- [ ] One shared `createNoteOnCanvas` helper exists and is used by every migrated authoring path (no duplicated create-and-place logic).
- [ ] `addNode` is `async`; the `note/inbox/reference/goal/habit/project/ai` presets are body+color+geometry specs that create file-backed notes (kind set for inbox/reference/ai); `widget`/`group` stay synchronous; `subcanvas`/`task` keep their early returns; no branch of `addNode` authors a `text` node.
- [ ] The double-click-on-empty-background handler and the note-tool pointerdown `await addNode(...)`, keep the `nodeAtClientPoint` hit-test, and create a `.md` file-node (default body `# New thought\nStart writing here…`, color `"2"`).
- [ ] `runAddKind` (`app.js:1769`) and `$("#newGroup").onclick` (`app.js:1784`) tolerate the async `addNode` return.
- [ ] `createAINote` produces a file-backed note via `createNoteOnCanvas` and keeps the dialog/result/toast flow.
- [ ] `runAICard` output is a file-backed note: first run creates and connects it with the reserved `AI output` edge; subsequent runs update the same file in place (stable output reuse); debouncing, queued reruns, and cycle detection are unchanged.
- [ ] `runLocalAssistant` add-note path uses `createNoteOnCanvas` for goal/habit/project/note and keeps the response message.
- [ ] `grep -n 'type:"text"' app.js` shows matches ONLY in the legacy `demoCanvas` (`app.js:24-42`) and the read-only interop render/validator paths — no interactive authoring path remains.
- [ ] `node --check app.js` exits 0.
- [ ] The AI security boundary is unchanged; if preserving it would require changing `ai/generated-operations.js` or the allowlist, STOP and report.
- [ ] `storage/life-indexer.js`, `storage/memory-index.js`, `storage/canvas-validate.js`, and the other repositories remain unchanged.
- [ ] `git diff --check` exits 0; only `app.js` is modified in this ticket.

## Blocked by

Ticket 02 (Migrate — boot wiring, note render/edit, and AI body-detection).
