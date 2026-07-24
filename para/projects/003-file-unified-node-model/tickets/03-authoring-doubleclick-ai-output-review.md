---
phase: review
status: done
ticket: 03
date: 2026-07-24
verdict: approved
---

# Standards review: ticket 03 — authoring templates, double-click, AI output, local parser

## Verdict: APPROVED

Scope verified: `git diff 003-file-unified-node-model...HEAD --stat` shows ONLY
`app.js` (+43/−22, one commit `0071e50` on top of ticket 02). `git status` clean.

## Verification performed

- `node --check app.js` → exit 0. `git diff --check` → exit 0.
- Full AGENTS.md §13 suite + note suite (`phase1…phase10`, `phase-query`,
  `note-repository`) → **196 pass, 0 fail**. Seam suite (`phase1/5/8/query/note`)
  → 91 pass, 0 fail.
- **AI boundary intact [highest priority]:** `git diff …HEAD -- ai/` is empty;
  `ai/generated-operations.js` and the `assistantSystemPrompt` allowlist
  (`app.js:1798-1822`) are byte-for-byte unchanged. Operators still propose
  allowlisted operations through the unchanged `applyCanvasOperations` path
  (`app.js:1563,1603`). The AI output is now a note file body rendered through
  `markdownToHTML(note.body)` (`app.js:961`), which escapes via `escapeHTML`
  (`app.js:787`) and skips `<!-- orbit:` marker lines — never executed as host
  code.
- **Temporal guarantees unchanged:** `scheduleAICard`, `scheduleChangedAICards`,
  `aiCardHasCycle` (`app.js:1737-1745`) and the `runAICard` `finally` queued-rerun
  logic are not in the diff. Only the output's storage shape changed. The new
  `AI output` edge points FROM the card and is excluded from
  `inputNodesForAICard` (`app.js:735`, `toNode===cardId` + `label!=="AI output"`),
  and the output note carries no `AI_CARD_MARKER`, so no feedback loop is
  introduced.
- **Stable output reuse (supported path):** first run finds no file-backed output
  → `createNoteOnCanvas` + push reserved `AI output` edge + `scheduleSave()`
  (`app.js:1867`); subsequent runs match `type==="file"&&isNotePath(...)` and
  `updateNote` in place (`app.js:1868`). `reloadCanvasDocuments` reassigns
  `documentData` (`app.js:271`) before the edge push, so the edge lands on the
  reloaded document and is persisted coherently.
- **grep criterion (judged):** `grep -n 'type:"text"' app.js` matches ONLY the
  legacy `demoCanvas` (`app.js:30-35`) and the graph starter (`app.js:53-91`).
  Every INTERACTIVE authoring path this ticket owns no longer authors text:
  `addNode` presets are body+color+geometry specs (`app.js:1176-1184`), the
  `dblclick` (`app.js:1131-1138`) and note-tool (`app.js:1128`) handlers route
  through `addNode`, and `createAINote` (`app.js:1850`), `runAICard` output
  (`app.js:1867`), and `runLocalAssistant` add-note (`app.js:1762`) all call
  `createNoteOnCanvas`. The remaining `nodes.push` sites are non-text: subcanvas
  portal (`app.js:546`, `type:"file"`), group/widget branch (`app.js:1191`),
  component-card placement (`app.js:1557`), and the AI-proposed `node.add`
  apply path (`app.js:1564,1603`, user-confirmed allowlisted operations, out of
  this ticket's scope). The implementer's reading is CORRECT: the starter rewrite
  is plan Step 10 = ticket 04 (`blocked-by:[03]`), so the criterion is absolute
  only after ticket 04; for ticket 03's scope it is satisfied.
- **Shared helper, no duplication:** `createNoteOnCanvas` (`app.js:1165-1171`) is
  the single create-and-place path; all four migrated callers use it (no
  duplicated logic). Error contract is sound: it throws on read-only/unwritable
  so the AI callers' existing try/catch surface the error, while `addNode` wraps
  its call in try/catch + toast + `return null` (`app.js:1197-1198`) to stay
  rejection-safe for the UI handlers. Marker handling is correct: inbox/reference/ai
  presets pass the body WITHOUT the leading marker and set `kind`, so
  `_noteContent(kind, body)` (`storage/note-repository.js:54-58`) prepends the
  marker exactly once — no doubled markers (verified against
  `NOTE_KIND_MARKERS`, `storage/note-catalog.js:21-25`; `ai` marker matches
  `AI_CARD_MARKER`, `app.js:717`).
- **Async handlers rejection-safe:** `dblclick` and note-tool only invoke
  `addNode("note",…)`, whose note branch is try/catch-wrapped; `addNode`'s top
  guard returns null and the group/widget branch is synchronous, so no unhandled
  rejections. The `nodeAtClientPoint` geometry hit-test is preserved in both
  handlers (`app.js:1126,1135`).
- **Guard files unchanged:** `git diff …HEAD -- storage/life-indexer.js
  storage/memory-index.js storage/canvas-validate.js storage/note-catalog.js
  storage/note-repository.js storage/task-repository.js
  storage/habit-repository.js storage/journal-event-repository.js` is empty.
- **Implementer judgment calls assessed:** (2) `runAddKind` (`app.js:1877`) and
  `$("#newGroup").onclick` (`app.js:1892`) ignore `addNode`'s return and `addNode`
  is rejection-safe, so no code change was needed — sound. (3) `created` (not
  `result`) in `createAINote` (`app.js:1850`) avoids colliding with the existing
  `result=$("#aiNoteResult")` in that scope — sound.

## Findings

- [judgement] `app.js:1866` — hard-cut edge case (not a rule/seam/boundary break):
  the output lookup uses `.find(edge=>…label==="AI output")`, which returns the
  FIRST matching edge. In a pre-existing dev vault that still has a legacy
  text-node `AI output` (pre-ADR-0004), that stale edge's target fails the
  `type==="file"&&isNotePath` test, so `outputNode` is undefined and the create
  branch runs again — and because `.find` keeps returning the stale edge first,
  the new file-backed output is never matched on later runs either. The result is
  not a one-time duplicate (as the impl summary frames it) but unbounded
  accumulation: each run adds another file-backed output + another `AI output`
  edge for that card. This affects ONLY legacy dev vaults (the plan hard-cuts
  them with no migration; a fresh/regenerated space seeds no AI outputs, and the
  supported path reuses stably), so it does not block approval. If legacy dev
  vaults matter, prefer the file-backed edge (e.g. `findLast`, or filter the edge
  search to targets satisfying `type==="file"&&isNotePath`).
- [judgement] `app.js:1172-1205` — Code standards (function length): `addNode` is
  now ~33 lines, over the 4-20 guideline. It remains cohesive (a preset data
  table plus two branches) and is consistent with app.js's established dense
  style; non-blocking.
- [judgement] `app.js:1867` — Readability: the create branch is a dense one-liner
  (create + edge push + `scheduleSave`). It matches the pre-existing code it
  replaced and the module's pervasive one-line style; non-blocking.

## Summary

3 findings, all [judgement], zero [hard]. AI security boundary and temporal
guarantees (debounce/queued-rerun/cycle-detection) are byte-for-byte intact and
the output is rendered escaped; every interactive authoring path this ticket owns
now produces a file-backed note through one shared, rejection-safe
`createNoteOnCanvas` helper with correct single-marker handling; guard files
unchanged; `node --check`, `git diff --check`, and 196 tests green. Worst issue is
the legacy-dev-vault output-edge accumulation (`app.js:1866`), an accepted
hard-cut consequence that does not affect the supported fresh-vault path. High
confidence; ticket 03 behaviors remain browser-pending per AGENTS.md §13.
