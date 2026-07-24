---
phase: review
status: done
ticket: 02
date: 2026-07-24
verdict: approved
---

# Standards review: ticket 02 — boot wiring, note render/edit, AI body-detection

## Verdict: APPROVED

Scope verified: `git diff 003-file-unified-node-model...HEAD` touches only `app.js`
(105+/18-, three commits: Step 4 boot `52208a1`, Step 6 AI `48489a5`, Step 5 render/edit
`294636d`). `git status` clean.

Verified gates (all pass):
- AI security boundary intact: `git diff 003-file-unified-node-model...HEAD -- ai/` is
  empty; `ai/generated-operations.js` and the allowlist are byte-for-byte unchanged.
  Operators still propose allowlisted operations; output never executes host code.
- Guard files unchanged: `storage/life-indexer.js`, `memory-index.js`,
  `canvas-validate.js`, `note-catalog.js`, `note-repository.js`, and the
  task/journal/habit/component-card/widget repositories all diff empty.
- `node --check app.js` exit 0; `git diff --check` exit 0.
- `node --test storage/note-repository.test.js storage/phase5.test.js storage/phase8.test.js`
  → 59 pass / 0 fail. Full AGENTS.md §13 suite + note suite → 196 pass / 0 fail.

Confirmed seams:
- Render stays synchronous: the note branch (`app.js:951-962`) reads only
  `noteCatalog.getByPath`/`getFallbackByPath` (a preloaded projection); no `await`, no
  per-card vault read (AGENTS.md §5). `nodeAIContent` (`app.js:739`) resolves the note
  body synchronously, satisfying §4.2 "context assembly resolves the referenced file
  body, not just its path".
- No residual `node.text` reads on `file` nodes: every remaining `.text` access is a
  guarded `type==="text"` interop read (`app.js:733,737,766-768,775,889,1246,1636`), a
  non-node `.text()`/`.textContent`/provider-part call, or the deferred legacy AI
  output / legacy inspector text-node paths (`app.js:1328,1847`, Steps 8-9). AI
  operators (now `file` nodes) are read only through the catalog.
- Editing a note never mutates the canvas `file` reference: the note scope
  (`app.js:1310-1318`) returns before the generic `item[key]=detail.value` path, and
  notes are excluded from the generic file-path/subpath editors via `!note`
  (`app.js:1248`). Writes go only through `noteRepository.updateNote`.
- Boot wiring complete: `noteCatalog.rebuild()` present in all five fan-outs
  (`app.js:359,402,421,1486,1619`, optional-chaining style matched per site) and cleared
  in the boot-failure reset (`app.js:364`).

`aiCardSignature` change (`app.js:756`, signs `nodeAIContent(card)` instead of
`card.text`): safe and minimal. For a file-backed operator `nodeAIContent` returns the
catalog body, the same defining content `card.text` played for text nodes, so
change-detection still fires on input edits and on operator-body edits. Debouncing
(`scheduleAICard` timer), queued reruns (`state.pending` in `runAICard`), and cycle
detection (`aiCardHasCycle`) are untouched. The `lastSignature` idempotency guard still
holds. This change is required by the "no residual file-node `.text` read" constraint
and does not alter the security boundary.

Implementer judgment calls, assessed:
1. Excluding notes from the generic file-path/subpath inspector block (`!note`,
   `app.js:1248`) — CORRECT and necessary. Without it the scope-less `{key:"file"}`
   editor would hit the generic mutation path and rewrite the placement's `file`,
   breaking the placement and violating "editing never mutates the canvas content".
2. `updateNoteBody` flushes pending workspace edits before reload (`app.js:580`,
   mirroring `updateTask` at `app.js:561`) — CORRECT. `reloadCanvasDocuments`
   (`app.js:260`) swaps `record.document` from disk, which would discard unsaved
   in-memory canvas edits; the flush prevents that.

Fallback consistency checked: `_fallbackByPath` only ever holds `fallbackNote(...)`,
which always carries a `diagnostic` and an empty `body` (`storage/note-catalog.js:63-65,103`),
so the render branch's `if(note.diagnostic)` short-circuits first and a fallback never
reaches the AI-marker branch. Render AI detection and `isAICard` are therefore
consistent; a missing/damaged note renders the readable "NOTE · UNAVAILABLE" fallback
and is never overwritten (the catalog and render path are read-only).

## Findings

- [judgement] `app.js:578-608,1383-1392,1406-1411` — Duplicated Code (Fowler #2): the
  note update trio (`scheduleNoteUpdate`/`flushNoteUpdate`/`updateNoteBody`),
  `deleteNoteEverywhere`, and the `deleteSelection` note branch deliberately mirror the
  task discipline. Appropriate for now: two call sites with divergent internals
  (id + patch-merge vs path + whole-body replacement; `lifeIndex` placements vs catalog
  placements), so a shared extractor would be Speculative Generality (#9). Revisit if a
  third debounced file-editor appears. No change required.
- [judgement] `app.js:1310-1318` (vs legacy `app.js:1327-1331`) — behavior nuance:
  editing an AI operator's title/prompt through the note scope no longer calls
  `scheduleChangedAICards`, so it does not auto-schedule a rerun the way the legacy
  text-node generic path did. This follows the ticket's instruction to mirror the task
  discipline; manual "Run now" still works, and debouncing/queued-reruns/cycle-detection
  and the security boundary are unaffected. Informational; not a violation.
- [judgement] `app.js:1328` — Speculative Generality / dead code (Fowler #9): the generic
  `aiTitle`/`aiPrompt` → `item.text=buildAICardText(...)` path is now unreachable (AI
  fields carry `scope:"note"` and return in the note branch; `isAICard` requires a `file`
  node, so text nodes never get AI fields). Pre-existing code, explicitly deferred to
  Step 9 per the impl summary. Out of this ticket's scope; no file-node `.text` write
  occurs. Track for Step 9.
- [judgement] `app.js:1636,1847` — remaining text-only paths (graph-memory checkbox
  count; AI output text node) are untouched by this ticket and dormant until the
  authoring migration lands (Steps 7-10). Step 6's function list did not include graph
  memory or output creation. Informational; track for Steps 8-9.

## Domain flags

None. The diff uses the CONTEXT.md vocabulary correctly (note, placement, projection,
catalog, AI operator, inert marker); no contradictory or new terms introduced.

## Summary

4 findings, all [judgement] (duplication-by-design, an intentional no-auto-rerun nuance,
and two deferred-to-later-step dead/text-only paths); zero [hard]. AI security boundary,
all three confirmed seams, and every guard file verified intact; checks and 196 tests
green. Confidence high; ticket 02 UI behavior remains browser-pending per the plan.
