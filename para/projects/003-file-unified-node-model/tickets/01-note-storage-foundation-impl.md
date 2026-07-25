---
phase: implement
status: done
project: 003-file-unified-node-model
ticket: 01
date: 2026-07-24
plan: plan.md
commit: 4337949
branch: 003-ticket-01
---

# Implementation: File-unified node model — note storage foundation (Steps 1-3)

Ticket 01 scope only: plan Steps 1-3, three NEW files, zero changes to `app.js`.
A Note is a path-identified canonical `notes/*.md` file with no mandatory
frontmatter and no `balaur-id`; identity is the path, kind (inbox/reference/ai)
is an inert body marker. Pre-flight passed: worktree
`/home/balaur/.paseo/worktrees/10x2zhef/003-ticket-01`, branch `003-ticket-01`
(non-main), clean status, ticket `blocked-by: []`.

## Steps completed

- [x] Step 1: `storage/note-catalog.js` — disposable synchronous note projection
      in the `widget-catalog.js` mold. `isNotePath`, `NoteCatalog` (`rebuild`,
      `reconcile`→rebuild, `getByPath`, `getFallbackByPath`, `notes`,
      `diagnostics`), `NOTE_PATH_CASE_COLLISION` via `caseFoldKey`,
      `NOTE_UNREADABLE` fallback (never throws out of rebuild), `CANVAS_MALFORMED`
      skip, `NOTE_FILE_MISSING`, title from first `# Heading` else slug, kind from
      inert markers. No orphan diagnostic (a note may have zero placements).
      Verified: `node --check storage/note-catalog.js` → exit 0.
- [x] Step 2: `storage/note-repository.js` — `FileNoteRepository`, path-keyed.
      `allocatePath` (slug + `-2/-3` by case-fold key), `createNote` (explicit
      `path` override for the starter, kind-marker prepend, index + reconcile +
      optional placement), `addPlacement` (journal shape + integer/positive
      geometry validation), `removePlacement` (task shape: node + incident edges),
      `updateNote` (full-body rewrite under expected hash), `deleteNote` (path-scan
      placements via `_removePlacementAtPath`, then remove file + reindex),
      `replacePlacement` (path-generic drain primitive for project 002). Verified:
      `node --check storage/note-repository.js` → exit 0.
- [x] Step 3: `storage/note-repository.test.js` — 24-test `node --test` contract
      suite (MemoryVault + MemoryIndex + LifeIndexer + NoteCatalog +
      FileNoteRepository). Covers create / collision-allocate / case-fold /
      place / remove / update / delete-everywhere / drain-re-place, plus the
      mandatory regression pin. Verified: `node --test storage/note-repository.test.js`
      → 24/24 pass.

Steps 4-12 (app.js wiring, rendering, AI migration, starter, docs) are out of
this ticket's scope and were not touched.

## Files changed

- `storage/note-catalog.js` (new, 161 lines) — note content projection.
- `storage/note-repository.js` (new, 183 lines) — path-keyed note repository + drain primitive.
- `storage/note-repository.test.js` (new, 318 lines) — contract suite with regression pin.

## Verification results

```
$ node --check storage/note-catalog.js storage/note-repository.js storage/note-repository.test.js
(exit 0, no output — all syntax OK)

$ node --test storage/note-repository.test.js
ℹ tests 24
ℹ pass 24
ℹ fail 0

$ node --test storage/phase1.test.js ... storage/phase-query.test.js storage/note-repository.test.js
ℹ tests 196   (172 baseline + 24 new)
ℹ pass 196
ℹ fail 0

$ node --test storage/component-card.test.js storage/widget-repository.test.js
ℹ tests 23 / pass 23 / fail 0   (named suites still green)

$ git diff --check
(exit 0, no whitespace errors)

$ git status --porcelain   (post-commit)
(empty — tree clean)

$ git diff --stat -- storage/life-indexer.js storage/memory-index.js storage/canvas-validate.js
(empty — all three byte-for-byte unchanged)

$ git diff --stat   (tracked modifications)
(empty — no tracked file modified; only the three new files were added)
```

Commit `4337949` on branch `003-ticket-01`: 3 files changed, 662 insertions.
Not pushed.

## Issues encountered

One reconciliation between two instructions, resolved by following the binding
acceptance criterion and the named precedent (not a deviation):

- The ticket/plan describe `updateNote` as writing under `expectedHash: stat.hash`
  (a fresh `vault.stat`). A fresh stat always matches the vault, so that mechanism
  alone cannot satisfy the acceptance criterion "a stale expected hash (a write
  behind the repository's back) rejects." The component-card repository — the
  ticket's primary path-scan precedent (`component-card-repository.js` `updateCard`)
  — resolves this exact case by comparing the catalog's last-known hash against the
  vault hash and throwing `WRITE_CONFLICT` on mismatch. I implemented that pattern:
  `updateNote` checks `catalog.getByPath(path).hash !== stat.hash` (reconcile +
  throw `ConflictError` `WRITE_CONFLICT` on mismatch), then writes under `stat.hash`.
  With no behind-the-back write the two hashes are equal, so the write still uses
  `stat.hash` as specified. The "updateNote rejects a write that happened behind the
  repository's back" test passes deterministically.

Minor honest simplification: notes are ordinary Markdown, so any byte sequence is
valid and there is no parse step that can fail. The catalog therefore records
`NOTE_UNREADABLE` for a read failure rather than fabricating an unreachable
`NOTE_MALFORMED` code path (no dead code). The ticket's "NOTE_MALFORMED/NOTE_UNREADABLE"
wording is satisfied by `NOTE_UNREADABLE` being the operative code.

Shared constants: `NOTE_KIND_MARKERS` and `noteTitleFromBody` are exported from
`note-catalog.js` and imported by `note-repository.js` so the inert-marker strings
and heading-title parsing have a single source of truth across the two new modules
(the ticket permits a local const; sharing avoids drift between the projection and
the writer).

## Domain flags

None. Used the reconciled `CONTEXT.md` vocabulary (Note, Placement, Drain,
Projection). No contradictions found; `CONTEXT.md` was not edited.
