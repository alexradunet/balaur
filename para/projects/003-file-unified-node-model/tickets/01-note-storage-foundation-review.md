---
phase: review
status: done
ticket: 01
date: 2026-07-24
verdict: approved
---

# Standards review: ticket 01 — note storage foundation

## Verdict: APPROVED

Scope verified: `git diff 003-file-unified-node-model...HEAD --name-status` shows
exactly three added files (`storage/note-catalog.js`, `storage/note-repository.js`,
`storage/note-repository.test.js`) and zero modifications.

Verification run by this review (not taken on trust):

- `node --check` on all three modules: exit 0.
- `node --test storage/note-repository.test.js`: 24/24 pass.
- Full baseline + new suite (phase1–10, phase-query, note-repository): 196/196 pass, no regression.
- `git diff --check 003-file-unified-node-model...HEAD`: clean (no whitespace errors).
- Seam pin: `git diff 003-file-unified-node-model...HEAD -- storage/life-indexer.js storage/memory-index.js storage/canvas-validate.js` is empty (all three byte-for-byte unchanged).

## Confirmed seams (all hold)

1. **No orbit-id, no mandatory frontmatter.** `createNote` writes only marker + body
   (`note-repository.js:57-61` `_noteContent`); the test asserts the placed node's keys
   are exactly `["file","height","id","type","width","x","y"]`
   (`note-repository.test.js:84-92`) and that the index record is untyped
   (`entityType:null, parseStatus:"ok"`, `note-repository.test.js:48-55`).
2. **Placements stay out of the disposable index.** The three protected files are
   unchanged. The pin is genuine, not tautological: `notes/` is absent from
   `ENTITY_DIR_TO_TYPE` (`life-indexer.js:18-24`), so `entityTypeFromPath("notes/*.md")`
   is null and `extractCanvasPlacements` skips note file nodes (`life-indexer.js:163`).
   The regression-pin test asserts `index.allPlacements().length === 0` and
   `placementsForEntity(path).length === 0` while drain + delete still work
   (`note-repository.test.js:236-258`).
3. **Drain/delete resolve placements by canvas scan.** `deleteNote` reads
   `catalog.getByPath(path)?.placements` (the catalog scans canvases,
   `note-catalog.js:116-126`); `replacePlacement` routes through
   `removePlacement → _removePlacementAtPath`, which scans the canvas document
   (`note-repository.js:108-122`). Neither consults `index.placementsForEntity`.

`replacePlacement(path, fromCanvasId, nodeId, toCanvasId, geometry)`
(`note-repository.js:175-180`) is path-generic and path-keyed: it takes an explicit
path, never derives an orbit-id, and re-places the SAME path on the target — the
primitive project 002 consumes.

## Reconciliations flagged by the implementer (judged correct, not findings)

- **`updateNote` WRITE_CONFLICT pattern** (`note-repository.js:135-148`): the ticket's
  prose says "under `expectedHash: stat.hash`" while its acceptance criterion requires
  "a stale expected hash (a write behind the repository's back) rejects." A bare
  freshly-fetched `stat.hash` can never reject a behind-the-back write, so the two
  instructions conflict. The implementer followed the binding acceptance criterion and
  the named precedent (`component-card-repository.js` `updateCard`,
  `component-card-repository.js:127-131`): compare the catalog's last-known hash against
  the vault hash and throw `ConflictError WRITE_CONFLICT` on mismatch, then still write
  under `stat.hash`. Correctly implemented and deterministically tested
  (`note-repository.test.js:178-185`).
- **Dropping the unreachable `NOTE_MALFORMED` code path** (`note-catalog.js:96-103`):
  notes are arbitrary Markdown with no parse/validation step, so only a read failure is
  reachable; `NOTE_UNREADABLE` is the operative code. This matches the "no dead code"
  standard rather than deviating from the ticket.
- **Explicit `path` override is not pre-validated** (`note-repository.js:71`): path
  safety is enforced at the vault boundary — `MemoryVault.write`/`remove` call
  `assertSafePath` (`memory-vault.js:102,124`), which throws `PathError` on traversal,
  absolute, scheme, and forbidden-char paths. This matches the task-repository precedent
  (`task-repository.js:67` passes `input.path` straight to `vault.write`) and satisfies
  AGENTS.md §7 "validation at storage boundaries."

## Findings

All are `[judgement]` (smells / possible improvements). No `[hard]` findings.

- [judgement] `storage/note-catalog.js:73-138` — Long function: `rebuild()` is ~66 lines,
  over the 4–20 line standard. It mirrors `widget-catalog.js:44-110` verbatim, the
  precedent the ticket mandated cloning; an extraction candidate shared with the widget
  catalog, not a defect introduced here.
- [judgement] `storage/note-repository.js:84-105,108-133` — Duplicated Code:
  `addPlacement` / `_removePlacementAtPath` / `removePlacement` are near-clones of the
  task, journal, and component-card repositories. Sanctioned by the ticket ("cloned
  exactly") and the established per-repository pattern; a shared placement mixin would be
  a cross-cutting refactor touching all four repositories, out of this ticket's scope.
- [judgement] `storage/note-repository.js:84` — `addPlacement` does not stat the note
  path or throw `NOTE_NOT_FOUND` before placing, diverging from the journal precedent's
  first step (`journal-event-repository.js:65-66` stats the placed entity). It matches
  the ticket's detailed `addPlacement` step list, which omits a note-stat, and the public
  method can therefore create a dangling file node; the catalog surfaces that as
  `NOTE_FILE_MISSING` (`note-catalog.js:124-125`). Defensible, worth a follow-up decision.
- [judgement] `storage/note-repository.js:91` — the geometry error ("Placement geometry
  must use integers with positive dimensions") states the expected shape but not the
  offending field/value; the code standard asks errors to include the offending value.
  Copied verbatim from `component-card-repository.js:184`.
- [judgement] `storage/note-repository.js:140` — `updateNote`'s `known &&` guard skips
  the conflict check when the catalog holds no baseline entry for the path, so a
  behind-the-back write to a note the catalog never saw would go undetected. In normal
  flow the repository reconciles the catalog after every create/update/delete, so the
  baseline exists; component-card's `_sourceFor` always establishes one. Minor robustness
  note, not reachable through the repository's own API.
- [judgement] `storage/note-repository.test.js` — no test forces a read failure to
  exercise the `NOTE_UNREADABLE` fallback (`note-catalog.js:96-103`); hard to trigger on
  `MemoryVault` without mocking. Not required by the acceptance criteria.

## Tests are real

The suite asserts external behavior at the repository seam — vault bytes
(`vault.read`/`vault.exists`), index state (`getSourceFile`, `allPlacements`,
`placementsForEntity`), and parsed canvas documents — never private fields. No
tautologies and no implementation coupling. The regression pin exercises real indexer
behavior (see seam 2 above), and the drain/delete assertions run against an empty
placements index.

## Summary

6 findings, all `[judgement]` (long `rebuild()`, cross-repo duplication, missing
note-existence check in `addPlacement`, geometry error omits offending value,
`updateNote` baseline guard, untested unreadable fallback); worst is the
`addPlacement` dangling-node gap, which the catalog already diagnoses. Zero `[hard]`
findings: all three confirmed seams hold, the two implementer reconciliations are
correct, tests are genuine and green (24/24 new, 196/196 total), and only the three
new files exist. High confidence. Verdict: APPROVED.
