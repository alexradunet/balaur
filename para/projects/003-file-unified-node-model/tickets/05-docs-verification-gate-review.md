---
phase: review
status: done
ticket: 05
date: 2026-07-24
verdict: approved
---

# Standards review: ticket 05 — docs-verification-gate (re-review)

## Verdict: APPROVED

The single prior [hard] finding is resolved. AGENTS.md §10 (line 248) now describes
AI operators as standard `file` nodes referencing `notes/*.md` (marker in the body,
output a file-backed note on the reserved `AI output` edge, ADR-0004), consistent with
§4.2, ADR-0004, CONTEXT.md, and `app.js`. No [hard] findings remain. The stale
browser-check skill assertion is carried forward as a [judgement] follow-up, not a
blocker.

## Prior [hard] finding — RESOLVED

- Prior finding: `AGENTS.md:248` read "AI operators remain standard text nodes with
  inert markers and edge-derived inputs.", contradicting the shipped code and §4.2.
- Fix: commit `45db054` (AGENTS.md only, 1+/1-). `AGENTS.md:248` now reads: "AI
  operators are standard `file` nodes referencing `notes/*.md` files whose body carries
  the inert `<!-- balaur:ai-card -->` marker, with edge-derived inputs; their output is
  likewise a file-backed note connected by the reserved `AI output` edge (ADR-0004).
  Preserve debouncing, stable output-node reuse, queued reruns, and cycle detection.
  File-node inputs resolve canonical file bodies."
- Consistency confirmed:
  - `AGENTS.md:117` (§4.2) uses the same file-backed wording.
  - `CONTEXT.md:123-125` ("AI operator") and `CONTEXT.md:128-130` ("Text node (interop)")
    match line 248 exactly.
  - `docs/adr/0004-file-unified-node-model.md:6-7,11` reverses the text-node model and
    unifies on `file` nodes with inert body markers; line 248 now agrees.
  - `app.js:751` (`isAICard` requires `node?.type==="file" && isNotePath(node.file)` and
    a body containing `AI_CARD_MARKER`) and `app.js:1898-1899` (output node is
    `node.type==="file" && isNotePath(node.file)`, edge `label:"AI output"`) match.
  - `grep -n "text node" AGENTS.md` returns nothing (exit 1): no contradictory statement
    survives anywhere in the file.

## Scope verification (all pass)

- Only four doc files changed: `git diff --name-only 003-file-unified-node-model...HEAD`
  lists `AGENTS.md`, `docs/architecture.md`, `docs/generative-canvas.md`,
  `docs/life-data.md` and nothing else.
- No source changed: `git diff --name-only ... -- app.js storage/ ai/ index.html styles/ .pi/`
  is empty. Guard files (`storage/life-indexer.js`, `storage/memory-index.js`,
  `storage/canvas-validate.js`, and the task/journal/habit/component-card/widget
  repositories) are unmodified.
- Skill untouched: `git diff --name-only ... -- .pi/` is empty. The browser-check skill
  was not edited to force the gate green.
- The fix commit `45db054` touched only `AGENTS.md` (`git show 45db054 --name-only`), so
  the other three docs are byte-identical to what the prior review verified.
- `git status` clean; `git diff --check 003-file-unified-node-model...HEAD` exit 0.
- `node --check app.js storage/note-catalog.js storage/note-repository.js storage/note-repository.test.js`:
  all four OK, exit 0.

## Test count integrity (pass)

Ran the exact AGENTS.md §13 command including `storage/note-repository.test.js`:
`tests 196 / pass 196 / fail 0`. The published sentence at `AGENTS.md:295`
("**196 tests**: the prior 172-test suite plus twenty-four note-repository tests")
matches the real total, and the §13 command lists `storage/note-repository.test.js` at
`AGENTS.md:292`.

## Doc accuracy verified against code (pass)

- AGENTS.md §4.2 (`AGENTS.md:117`): notes/inbox/reference and AI operators are
  path-identified `notes/*.md` placed by standard `file` nodes, marker in the body,
  `text` is read-only interop, references ADR-0004. Matches `app.js:751` and the
  read-only interop note path at `app.js:766`.
- AGENTS.md §10 (`AGENTS.md:248`): now file-backed (see resolved finding above).
- docs/life-data.md (`docs/life-data.md:190,194`): note contract (path identity, no
  mandatory frontmatter/`balaur-id`, kind as inert body marker including `balaur:ai-card`,
  title derived from first `# Heading` then path slug, indexer treats as valid untyped
  record `entityType: null`/`parseStatus: "ok"`, placements resolved by canvas scan not
  the disposable index) matches `storage/note-catalog.js`, `storage/life-indexer.js`, and
  `storage/note-repository.js`.
- docs/architecture.md (`docs/architecture.md:18,27,66`): ownership tree adds
  `notes/*.md`; projection paragraph adds `NoteCatalog`; repository list adds
  `FileNoteRepository` (path-identified notes, placements, drain primitive) and
  `NoteCatalog` (ADR-0004). All match the code.
- docs/generative-canvas.md (`docs/generative-canvas.md:72,77,81`): AI operator carries
  the `<!-- balaur:ai-card -->` marker, creates a file-backed note (standard `file` node
  referencing `notes/*.md`) connected by an edge labeled `AI output`, updates that same
  note file on rerun, and interop clients see ordinary `file` nodes with the marker in
  the body. Matches `app.js:1898-1899`.
- Starter and double-click authoring remain file-backed (`app.js:143`, `app.js:1164-1170`
  → `createNoteOnCanvas`), unchanged by this ticket.

## Findings

- [judgement] `.pi/skills/browser-check/scripts/browser-check.mjs:371-373` — stale
  assertion, carried-forward follow-up (not a blocker): the smoke check reads
  `lastText: last?.text ?? ""` and asserts `info.lastText.includes("New thought")`.
  Double-click now creates a `file` node whose content lives in `notes/new-thought.md`
  with no `.text` property, so `lastText` is empty and the second conjunct fails even
  though the node count rises. Confirmed a stale assertion in local agent tooling, not a
  regression: the behavior matches ADR-0004 (`app.js:1164-1170` → `createNoteOnCanvas` →
  `noteRepository.createNote`), the implementer's direct probe shows
  `{type:"file", file:"notes/new-thought.md"}`, and the `.pi/` diff is empty (skill not
  tampered with). The skill is local agent tooling under `.pi/skills/`, not deployed
  shell, and is outside this documentation-only ticket's scope. Suggest a follow-up to
  assert `last.type === "file" && last.file.startsWith("notes/")`.

## Writing standards (pass)

The rewritten `AGENTS.md:248` uses a semicolon and parentheses, no em dash as casual
punctuation, no "Additionally/Furthermore/Moreover" opener, no filler, active voice, and
concrete named tokens (`notes/*.md`, the marker, the `AI output` edge, ADR-0004). No emoji
in the added line. The other three docs are unchanged from the prior review, which already
passed them.

## Summary

Re-review: the prior [hard] finding (AGENTS.md:248 text-node wording) is resolved by
commit 45db054 and verified consistent with §4.2, ADR-0004, CONTEXT.md, and app.js. One
carried-over [judgement] (stale browser-check smoke assertion) remains a follow-up, not a
blocker. Scope (four doc files only; source, guard files, and `.pi/` unmodified), test
count (196/196), node --check, git diff --check, and all doc-accuracy checks pass. Verdict
approved. High confidence.
