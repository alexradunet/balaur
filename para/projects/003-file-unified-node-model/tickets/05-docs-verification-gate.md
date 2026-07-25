---
phase: ticket
status: done
project: 003-file-unified-node-model
ticket: 05
blocked-by: [04]
worker: "9cd5a5a6-0eb9-4a08-bae4-1054e4382629"
branch: "003-ticket-05"
shared-blast-radius: true
---

# Ticket 05: Docs + verification gate

## What to build

Update the documentation to match the shipped file-unified model, then run the full
verification gate. This is plan Steps 11-12 in one ticket because the docs must publish
the real test count that the gate confirms. Prose and verification only; no source
changes here.

Update documentation in the same change (Step 11):
- `AGENTS.md` §4.2 (`AGENTS.md:99-115`): replace "Inbox notes and reference pages are
  standard text nodes with inert markers" and "AI operators are standard text nodes…"
  with the file-backed wording (notes/inbox/reference/AI are path-identified
  `notes/*.md` files placed by standard `file` nodes; the inert marker lives in the
  file body; `text` remains a valid, rendered, read-only interop type that Balaur never
  authors). Reference ADR-0004.
- `AGENTS.md` repository map (`AGENTS.md:60-72`): add `storage/note-catalog.js`
  (disposable synchronous note content projection) and `storage/note-repository.js`
  (FileNoteRepository: canonical note files, placements, and the drain primitive), and
  note the `notes/` layout alongside `tasks/`, `habits/`, etc.
- `AGENTS.md` §13 (`AGENTS.md:283-292`): add `storage/note-repository.test.js` to the
  `node --test` command and bump the published count. Run the full suite first to get
  the real new total, then update the "**172 tests**" sentence to the actual number and
  name the new note-repository suite as the addition.
- `docs/life-data.md`: add the note contract (path-identified `notes/*.md`, no
  mandatory frontmatter/`balaur-id`, kind is an inert body marker, indexer treats it as
  valid untyped Markdown), add notes to the ownership list, and update the node-typing
  section (notes are file-backed; `text` is read-only interop).
- `docs/architecture.md`: update the ownership model and the repository list to include
  the note catalog + note repository.
- `docs/generative-canvas.md`: supersede the text-node statements at lines 65, 69, 77,
  81 — an AI operator and its output are standard `file` nodes referencing `notes/*.md`;
  the portable compatibility marker lives in the file body; the security boundary is
  unchanged.

Run the full verification gate (Step 12), in order:
1. `node --check app.js storage/note-catalog.js storage/note-repository.js
   storage/note-repository.test.js` → exit 0 each.
2. The full AGENTS.md §13 suite INCLUDING the new suite (the exact command now in
   AGENTS.md after the docs update):
   `node --test storage/phase1.test.js storage/phase2.test.js storage/phase3.test.js
   storage/phase4.test.js storage/phase4-backup.test.js storage/phase5.test.js
   storage/phase7.test.js storage/phase8.test.js storage/phase9.test.js
   storage/phase10.test.js storage/phase-query.test.js storage/note-repository.test.js`
   → all pass; the total equals the bumped count published in AGENTS.md §13.
3. `git diff --check` → exit 0.
4. Browser smoke (browser-pending behaviors): serve with `python3 -m http.server 4173`,
   then `node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline`.
   Confirm: no uncaught console errors; the file-backed starter renders every document
   node; double-clicking empty background creates a file-backed note (a `notes/*.md`
   appears and a `file` node renders it); editing a note writes the `.md`; an imported
   `text` node renders read-only (no edit textarea). LABEL these as browser-pending in
   the report; the Node suite does not verify them. If the skill is absent, fall back to
   the manual baseline in AGENTS.md §13 and say so.

## Acceptance criteria

- [ ] AGENTS.md §4.2 states notes/inbox/reference/AI are file-backed `notes/*.md` placed by standard `file` nodes (marker in the body; `text` is read-only interop) and references ADR-0004.
- [ ] AGENTS.md repository map lists `storage/note-catalog.js` and `storage/note-repository.js` and the `notes/` layout.
- [ ] AGENTS.md §13 `node --test` command includes `storage/note-repository.test.js`, and the published test count matches the real total from running the suite.
- [ ] `docs/life-data.md` documents the note contract, ownership, and node-typing (notes file-backed; `text` read-only interop).
- [ ] `docs/architecture.md` updates the ownership model and repository list (note catalog + note repository).
- [ ] `docs/generative-canvas.md` supersedes the text-node statements at lines 65, 69, 77, 81 (AI operator/output are file-backed; marker in the body; security boundary unchanged).
- [ ] `node --check` exits 0 for `app.js`, `storage/note-catalog.js`, `storage/note-repository.js`, `storage/note-repository.test.js`.
- [ ] The full AGENTS.md §13 `node --test` command (with the note suite) exits 0; the total equals the bumped published count.
- [ ] `grep -n 'type:"text"' app.js` shows matches only in the legacy `demoCanvas` and the read-only interop render path; `grep -n 'item.text=' app.js` returns no matches.
- [ ] `storage/life-indexer.js`, `storage/memory-index.js`, `storage/canvas-validate.js`, and the task/journal/habit/component-card/widget repositories are unmodified (`git diff --stat` shows none of them).
- [ ] The new suite contains the regression pin: note placements are absent from `index.allPlacements()`, and drain/delete resolve them by canvas path scan.
- [ ] `git diff --check` exits 0; no files outside the in-scope list are modified.
- [ ] Browser smoke run (or documented manual fallback); browser-pending behaviors are labeled as such in the report.

## Blocked by

Ticket 04 (Contract — text read-only + greenfield file-backed starter).
