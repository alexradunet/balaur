---
phase: implement
status: done
project: 003-file-unified-node-model
ticket: 03
date: 2026-07-24
plan: plan.md
commit: 0071e50
branch: 003-ticket-03
---

# Implementation: Ticket 03 — authoring templates, double-click, AI output, local parser (plan Steps 7-8)

Scope is `app.js` only. Every interactive authoring path now produces a
file-backed note through one shared `createNoteOnCanvas` helper; no in-scope
authoring path authors a `text` node. Ticket 01 (`storage/note-catalog.js`,
`storage/note-repository.js`) and ticket 02 (boot wiring, render/edit, AI
body-detection) are the dependencies and were left byte-for-byte unchanged.

Pre-flight passed: worktree
`/home/balaur/.paseo/worktrees/10x2zhef/003-ticket-03`, branch `003-ticket-03`
(non-main), clean status, ticket `blocked-by: [02]` with ticket 02 `done`.

## Steps completed

- [x] **Shared helper** — added `async function createNoteOnCanvas({title, body,
      kind=null, color, geometry, canvasId=currentCanvasId})` immediately above
      `addNode`. It guards on `canonicalWritable`/`noteRepository` (throws
      `"Canonical files are unavailable or read-only."`, matching the existing
      note-scope guard at the former `app.js:579`), `await
      flushPendingWorkspaceEdits()`, `const result = await
      enqueueMutation(()=>noteRepository.createNote({title, body, kind, color,
      canvasId, geometry}))`, `await reloadCanvasDocuments([canvasId])`, and
      returns `result` (`{path, note, placement}`). No duplicated
      create-and-place logic; every migrated path calls this one helper.
- [x] **Step 7 — `addNode` async + templates** — `addNode` is now `async`. The
      `note/inbox/reference/goal/habit/project/ai` presets are now
      body+color+width+height+kind specs (no `type:"text"`, no `text:` field).
      Each keeps its current Markdown as the body and its current color and
      width/height as the placement geometry; `kind` is set for
      inbox/reference/ai. For those kinds the click-point geometry reuses the
      existing `center`/width/height math, then `await createNoteOnCanvas(...)`,
      then selects `result.placement.nodeId` and renders. `widget`/`group` keep
      their existing synchronous behavior (the `if(preset.type)` branch);
      `subcanvas`/`task` keep their early returns. No branch of `addNode`
      authors a `text` node. Verified: `node --check app.js` exit 0.
- [x] **Step 7 — double-click + note tool** — the `dblclick` handler and the
      note-tool `pointerdown` branch are now `async` and `await addNode(...)`,
      keeping the `nodeAtClientPoint` geometry hit-test that prevents spawning
      over a card. Double-click on empty background creates a `.md` file-node
      (the `note` preset: body `# New thought\nStart writing here…`, color
      `"2"`). Verified: `node --check app.js` exit 0.
- [x] **Step 7 — async-return callers** — verified `runAddKind`
      (`else addNode(kind);`) and `$("#newGroup").onclick=()=>addNode("group")`
      tolerate the async return: neither consumes the return value, and
      `addNode` is rejection-safe (top guard toasts + returns null; the note
      branch wraps `createNoteOnCanvas` in try/catch and toasts + returns null;
      the group/widget branch is synchronous). No unhandled rejections, no
      broken logic; no code change was required. See Issues #3.
- [x] **Step 8 — `createAINote`** — replaced `documentData.nodes.push(textNode)`
      with `const created = await createNoteOnCanvas({title:"AI note",
      body:generated, color:"5", geometry:{…existing x/y/width/height…}})`,
      selects `created.placement.nodeId`, and keeps the dialog/result/toast flow
      (it sits inside the existing try/catch, so a failure still surfaces
      `error.message` in the dialog result). Verified: `node --check app.js`
      exit 0.
- [x] **Step 8 — `runAICard` output** — the output is now a file-backed note.
      Resolves the existing output via the `AI output` edge with
      `node.type==="file"&&isNotePath(node.file)`. If absent: `await
      createNoteOnCanvas({title:\`${config.title} — output\`, body:generated,
      color:"5", canvasId:currentCanvasId, geometry:{x:card.x+card.width+90,
      y:card.y, width:380, height:240}})`, then connects the new node with the
      reserved `AI output` edge (`toNode:created.placement.nodeId`), then
      `scheduleSave()`. If present: `await
      noteRepository.updateNote(outputNode.file, generated)` (stable output
      reuse), then re-render. Debouncing (`scheduleAICard`), queued reruns
      (`state.pending` + the `finally` re-schedule), and cycle detection
      (`aiCardHasCycle`) are UNCHANGED; only the output's storage shape changed.
      The redundant trailing `scheduleSave()` was removed (the create branch
      calls it; the update branch persists via `updateNote` and leaves the
      canvas document unchanged). Verified: `node --check app.js` exit 0.
- [x] **Step 8 — `runLocalAssistant`** — the add-note path now uses `await
      createNoteOnCanvas({title, body:preset[1], color:preset[0],
      geometry:{…center…}})` for the goal/habit/project/note kinds, replacing
      `applyCanvasOperations([{type:"node.add", textNode}])`. Keeps the response
      message and sits inside the existing try/catch. Verified: `node --check
      app.js` exit 0.

## Files changed

- `app.js` — +43/−22. Added `createNoteOnCanvas`; made `addNode` async and
  converted its presets to file-backed specs; made the `dblclick` and note-tool
  `pointerdown` handlers async; migrated `createAINote`, `runAICard` output, and
  the `runLocalAssistant` add-note path to `createNoteOnCanvas`.

No other file was touched. `git status` shows only `app.js` modified (now
committed). `git diff --stat HEAD~1 HEAD -- ai/ storage/life-indexer.js
storage/memory-index.js storage/canvas-validate.js storage/note-catalog.js
storage/note-repository.js` is empty: the AI security boundary
(`ai/generated-operations.js` + the allowlist in `assistantSystemPrompt`), the
indexer, the validator, and the note catalog/repository are byte-for-byte
unchanged, as are the task/journal/habit/component-card/widget repositories.

## Verification results

- `node --check app.js` → exit 0.
- `node --test storage/phase1.test.js storage/phase5.test.js
  storage/phase8.test.js storage/phase-query.test.js
  storage/note-repository.test.js` → **91 pass, 0 fail** (the briefing's seam
  suite).
- Full AGENTS.md §13 suite + the note suite (`phase1…phase10`, `phase-query`,
  `note-repository`) → **196 pass, 0 fail** (no regression).
- `git diff --check` → exit 0 (no whitespace errors).
- `git status --porcelain` → clean after commit; only `app.js` was modified.
- `grep -n 'type:"text"' app.js` → matches ONLY the legacy `demoCanvas`
  (`app.js:30-35`) and the graph starter (`app.js:53-91`). All ticket-03
  interactive authoring paths are gone: filtering out those two retained regions
  leaves zero matches. See Issues #1 for why the starter still matches.
- AI security boundary intact: operators still propose allowlisted operations
  through the unchanged `applyCanvasOperations`/`ai/generated-operations.js`
  path; AI output is now a note file body rendered through `markdownToHTML`
  (HTML-escaped), never executed as host code.

## Issues encountered

1. **The `grep type:"text"` acceptance criterion is forward-looking.** After this
   ticket the grep still matches the graph starter (`createGraphStarterWorkspace`,
   `app.js:53-91`). The greenfield file-backed starter rewrite is plan **Step 10 =
   ticket 04** (`blocked-by: [03]`, not yet started); it is explicitly out of this
   ticket's scope (Steps 7-8). The criterion as worded describes the plan's final
   state (after Step 10). For ticket 03's scope it is satisfied: every interactive
   authoring path this ticket owns (addNode templates, double-click, note tool,
   `createAINote`, `runAICard` output, `runLocalAssistant`) no longer authors a
   `text` node; the only remaining matches are the intentionally retained legacy
   `demoCanvas` and the starter that ticket 04 rewrites. I did NOT touch the
   starter (that is ticket 04's work and a separate review). The orchestrator
   should sequence ticket 04 to complete the grep criterion absolutely.

2. **Legacy text-node AI outputs (hard-cut consequence).** `runAICard` now resolves
   its output by `type==="file"&&isNotePath(...)`, exactly as the ticket specifies.
   A pre-existing dev vault that still has a `text`-node output (from before
   ADR-0004) will not match, so the next run creates a new file-backed output and a
   second `AI output` edge. This is inherent to the plan's hard-cut decision (no
   auto-migration of existing inline text nodes) combined with the ticket's
   specified lookup; it affects only pre-existing dev vaults, not a fresh/regenerated
   space (the starter seeds no AI outputs). Browser-pending; not exercised by the
   Node seam. No deviation from the ticket.

3. **`runAddKind` / `newGroup.onclick` needed no code change.** The ticket listed
   them as callers to "tolerate the async return." Both ignore `addNode`'s return
   value, and `addNode` is rejection-safe, so they already tolerate it. Making a
   no-op edit would be gratuitous; I verified the behavior instead (see Steps
   completed).

4. **Two small, justified deviations from a strictly literal reading:**
   - `createNoteOnCanvas`'s guard **throws** (matching the existing note-scope
     guard) so the AI callers' try/catch surface the error; `addNode` wraps its
     `createNoteOnCanvas` call in try/catch and toasts + returns null, preserving
     `addNode`'s existing UI contract (it already toasted + returned null on
     read-only) and keeping the UI event handlers rejection-safe.
   - In `createAINote` the createNoteOnCanvas result is named `created` (not
     `result`) to avoid colliding with the existing `result=$("#aiNoteResult")`
     DOM element in that function's scope.
   - For the inbox/reference/ai presets the body is passed WITHOUT the leading
     marker and `kind` is set, so `_noteContent(kind, body)` prepends the marker
     exactly once. The resulting file body is byte-identical to the old text-node
     `text` field (e.g. `<!-- balaur:inbox -->\n# New capture\n…`); passing both a
     marker-bearing body AND `kind` would have doubled the marker.

## Browser-pending

The following are UI behaviors the Node seam does NOT cover and remain
browser-pending (verify with `.pi/skills/browser-check/` smoke when available):

- Double-clicking empty background creates a `notes/*.md` file-backed note that
  renders; the note-tool pointerdown does the same; the `nodeAtClientPoint`
  hit-test still prevents spawning over a card.
- The add-menu templates (note/inbox/reference/goal/habit/project/ai) create
  file-backed notes with the correct kind marker and placement color.
- `createAINote` produces a file-backed note and keeps the dialog/result/toast
  flow.
- `runAICard` first run creates + connects a file-backed output with the reserved
  `AI output` edge; subsequent runs update the same file in place (stable reuse);
  debouncing/queued reruns/cycle detection behave as before.
- `runLocalAssistant` add-note (goal/habit/project/note) creates a file-backed
  note and keeps the response message.

## Domain flags

None. `CONTEXT.md` was already reconciled by ticket 02 (Note, Placement, AI
operator, Text node (interop), Drain). This ticket introduces no new or
contradictory terminology; it only routes authoring through the existing
note/Placement model.
