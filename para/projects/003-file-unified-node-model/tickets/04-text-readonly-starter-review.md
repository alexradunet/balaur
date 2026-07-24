---
phase: review
status: done
ticket: 04
date: 2026-07-24
verdict: approved
---

# Standards review: ticket 04 — text-readonly-starter

## Verdict: APPROVED

Reviewed `git diff 003-file-unified-node-model...HEAD` (one commit `a88ee7b`, `app.js`
only, 45+/13-). Verified against AGENTS.md §4.1, §4.2, §9, §12, CONTEXT.md, the ticket,
and the 12-smell baseline. No [hard] findings.

## Verification (review-focus checklist)

- TEXT READ-ONLY (Step 9):
  - Inspector text textarea removed; replaced by a read-only note at `app.js:1299`
    (`else if(item.type==="text")notes.push({text:"Imported text node · read-only. …"})`).
    `notes` is in scope (`app.js:1276`). No editable field is offered for a `text` node.
  - The `text` render branch STAYS: `app.js:922` (`markdownToHTML(node.text)`), so imported
    documents remain visible. Title/summary reads at `app.js:766`/`app.js:770` are read-only.
  - `grep -n 'item.text=' app.js` → NO matches (exit 1). No code path mutates a text node.
  - Dead `aiTitle`/`aiPrompt` `item.text=` mutation removed from `applyInspectorField`
    (diff at `app.js:1378`); the dangling `else if` became `if`. Those fields now carry
    `scope:"note",notePath:item.file` (`app.js:1284`) and are handled by the note branch
    (`app.js:636-638`). `buildAICardText` is still referenced (`app.js:638`, defined
    `app.js:757`), so its removal did not orphan the helper.
- STARTER REWRITE (Step 10):
  - `STARTER_NOTES` defined beside `STARTER_TASK_PATH` (`app.js:51-62`): ten slugged
    `notes/*.md` paths matching the `slugify` convention (`storage/vault-path.js:57`),
    e.g. `homeGuide:"notes/start-here-your-life-as-a-graph.md"`,
    `archivePortfolio:"notes/portfolio-refresh-completed.md"`.
  - All ten inline text nodes converted to standard `file` nodes (`grep -c 'file:STARTER_NOTES\.'` = 10).
    Every node `id`, geometry (`x/y/width/height`), and `color` preserved line-for-line in the diff.
  - Dormant `#6c757d` (`DORMANT_NODE_COLOR`, `app.js:48`) preserved on `archive-portfolio` (`app.js:103`).
  - All three labelled edges preserved with label + color: `e-inbox-filed` (`filed-to`, color 6,
    `app.js:81`), `e-cb-partof` (`part-of`, color 6, `app.js:91`), `e-wiki-relates` (`relates-to`,
    color 5, `app.js:99`).
  - Task node `cb-task` → `STARTER_TASK_PATH` (`app.js:89`) and journal node `wiki-journal` →
    `journalFile` (`app.js:97`) unchanged.
  - Seeded note bodies match the previous inline text exactly (marker stripped from the body,
    re-added by the repository via `kind`); no content drift.
- SEEDING IDEMPOTENCY:
  - Each starter note seeded via `noteRepository.createNote({path, body, kind})` in a per-seed
    try/catch (`app.js:131-145`), mirroring the task seed (`app.js:116-124`).
  - `createNote` writes with `expectedHash:null`, which throws `WRITE_CONFLICT` when the file
    already exists (`storage/note-repository.js:69`, `storage/memory-vault.js:51-52`). The
    try/catch swallows it, so re-seeding is idempotent. Pattern is correct for idempotency.
  - No doubled markers: the seeded `body` excludes the marker; `_noteContent` prepends it once
    (`storage/note-repository.js:53-57`). `inboxTrip` carries `kind:"inbox"`; `wikiBudget`,
    `wikiSubscriptions`, `archivePortfolio` carry `kind:"reference"`; guides/`cbNote` carry none.
- DEMOCANVAS UNTOUCHED: `git diff … -- app.js` touches none of `n-focus`/`n-project`/`n-habit`/
  `n-idea`/`n-goal`/`n-reading` or the `demoCanvas` binding (legacy text nodes at `app.js:30-35`).
- GUARD FILES UNCHANGED: `git diff --stat` is empty for `storage/life-indexer.js`,
  `storage/memory-index.js`, `storage/canvas-validate.js`, `storage/note-catalog.js`,
  `storage/note-repository.js`, and the task/journal/habit/component-card/widget repositories.
- FINAL HARD-CUT STATE: `grep -nE 'type:\s*"text"' app.js` matches ONLY the legacy `demoCanvas`
  (`app.js:30-35`). The read-only interop path uses `node.type==="text"` (`app.js:766/770/922`),
  not a `type:"text"` literal. Every interactive authoring path (double-click `createNoteOnCanvas`
  `app.js:1198-1201`, note-tool presets `app.js:1230`, AI note `app.js:1882`, AI output
  `app.js:1899`) routes through `noteRepository.createNote`, which emits a `type:"file"` node.
  No interactive path produces a text node.
- STATIC + TESTS: `node --check app.js` exit 0; `git diff --check` exit 0; full foundation + note
  suite (`phase1..10`, `phase-query`, `note-repository`) = 196 pass / 0 fail; `git status` clean,
  only `app.js` modified.

## Findings

- [judgement] `app.js:113-153` — Divergent Change / function length: `seedGraphStarterEntities`
  is ~40 lines and `createGraphStarterWorkspace` (`app.js:63-112`) ~50, past the 4-20 line
  guideline. Acceptable here: `app.js` is an explicitly large pre-existing module, each function
  is cohesive (one builds the starter space, one seeds it), and the `seeds` data-table + loop
  (`app.js:131-145`) actively avoids ten duplicated try/catch blocks. Splitting would be
  speculative (YAGNI). No change requested.
- [judgement] `app.js:132-141` — the seeded note bodies are long inline string literals, but this
  is consistent with the pre-existing task seed (`app.js:119-122`) and journal seed (`app.js:150`)
  in the same function. Consistent with module style; no change requested.

## Summary

Two judgement notes (function length / inline seed bodies, both consistent with the existing
module style), zero hard findings. Every starter id/geometry/color/edge preserved, demoCanvas and
all guard files untouched, `item.text=` fully eliminated, seeding idempotent via the
WRITE_CONFLICT-on-existing pattern, hard-cut state confirmed; static checks and 196 tests green.
Verdict: approved. High confidence.
