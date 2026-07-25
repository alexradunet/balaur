---
phase: implement
status: done
project: 005-canvas-title-filenames
ticket: 01
date: 2026-07-25
commit: e78f0e1
branch: wealthy-blowfish
---

# Implementation: Slug, unique-path, and rename primitives (additive, Node-tested)

## Concurrent-worker race (resolved)

This ticket was double-spawned: two workers (`3b0c72f` and this worker,
`5c0e752`) began implementing it in the same `wealthy-blowfish` worktree at the
same time. `3b0c72f` detected the duplicate, wrote a `status: stopped` version
of this artifact, and ceased. Its recommended resolution was: the surviving
worker reconciles the tree to one coherent version of the four in-scope files,
finishes Steps 5–6, and commits once. This worker is that survivor.

The race explained the duplicate `canvasSlug` / `uniqueCanvasPath` /
`renameCanvasPath` blocks found mid-edit (both workers appended the same
functions); these were de-duplicated to single copies. Before committing, this
worker verified the tree was coherent and no writer was still active:

- `git diff` on all four files: pure appends plus the two import-line edits;
  **0 removed lines** in `vault-path.js` and `workspace-vault.js` (so `slugify`,
  `canvasPathFor`, `ROOT_CANVAS_PATH`, and `parseSidecar` are byte-for-byte
  unchanged), and the only removed lines in the two test files are the import
  lines re-added with the new names.
- `grep -c` confirms exactly one `export function canvasSlug`, one
  `uniqueCanvasPath`, one `renameCanvasPath`.
- `node -e "await import(...)"` links both modules (catches duplicate exports
  that `node --check` misses): LINK OK.
- A 6-second md5 stability watch on the four files: **STABLE, no concurrent
  writer**.
- `grep -n "ROOT_CANVAS_PATH\|canvasPathFor" storage/workspace-vault.js` content
  is identical to `planned-at` 71899aa (line numbers shifted down by the append;
  signatures and constants unchanged).

The final on-disk state was verified line by line and by the full suite; mixed
authorship is immaterial because the reconciled result is correct and every line
was re-verified by this worker.

## Steps completed

- [x] Pre-flight — verified: non-main worktree `wealthy-blowfish`, branch
  `wealthy-blowfish`, clean status at start; drift check
  `git diff --stat 71899aa..HEAD -- <in-scope paths>` empty; all "Current state"
  line numbers confirmed (`slugify` @57, `trimTrailing` @46, `caseFoldKey` @42,
  `unicodeCaseFold` @35; `canvasPathFor` @29, `ROOT_CANVAS_PATH` @20,
  `parseSidecar` root check @103).
- [x] Step 1: `canvasSlug(title, fallbackId)` added after `slugify` in
  `storage/vault-path.js` — verified `node --check storage/vault-path.js` exit 0;
  pure append, `slugify` unchanged.
- [x] Step 2: `uniqueCanvasPath(slug, existingPaths)` added after
  `canvasPathFor` in `storage/workspace-vault.js` — verified `node --check`
  exit 0.
- [x] Step 3: `renameCanvasPath(workspace, canvasId, newPath)` added after
  `uniqueCanvasPath` — verified `node --check storage/workspace-vault.js`
  exit 0; reads `record.path` directly (not `canvasPathFor`).
- [x] Step 4: `canvasSlug` tests in `storage/phase1.test.js` (import + one test
  block) — verified `node --test storage/phase1.test.js`: 29 pass / 0 fail.
- [x] Step 5: `uniqueCanvasPath` + `renameCanvasPath` tests and the non-root
  rename round-trip in `storage/phase4.test.js` (import + four test blocks) —
  verified `node --test storage/phase4.test.js`: 25 pass / 0 fail.
- [x] Step 6: full foundation suite + static checks — verified below.

## Spec note: truncation fixture

The ticket's illustrative title `"alpha beta gamma delta epsilon zeta eta theta"`
slugs to only 45 code points and would not trigger the 60-char cap. Per the
ticket's instruction to "build a title … whose slug exceeds 60 chars", the test
uses `"alpha beta gamma delta epsilon zeta eta theta iota kappa lambda"` (63 code
points), which truncates at the last hyphen before 60 to
`"alpha-beta-gamma-delta-epsilon-zeta-eta-theta-iota-kappa"` (56 code points, no
trailing hyphen). The hard-truncate case uses an 80-letter solid word → exactly
60 code points.

## Files changed

- `storage/vault-path.js` — added `canvasSlug` export (pure append).
- `storage/workspace-vault.js` — added `uniqueCanvasPath` and `renameCanvasPath`
  exports (pure append).
- `storage/phase1.test.js` — added `canvasSlug` to the import and one test block.
- `storage/phase4.test.js` — added `uniqueCanvasPath`/`renameCanvasPath` to the
  import and four test blocks (collision resolution, pure rename behavior, error
  cases, non-root rename round-trip proving orphan-removal).

## Verification results

```
node --check storage/vault-path.js storage/workspace-vault.js   → exit 0
node -e "await import('./storage/vault-path.js'); await import('./storage/workspace-vault.js')" → LINK OK
node --test storage/phase1.test.js   → tests 29, pass 29, fail 0
node --test storage/phase4.test.js   → tests 25, pass 25, fail 0
node --test <full 12-file foundation suite> → tests 202, pass 202, fail 0
git diff --check                     → exit 0 (no whitespace errors)
```

Full suite is 197 + 5 added = 202, 0 fail (ticket is additive; no existing
behavior changed).

## Done-criteria audit

- `canvasSlug` exported from `storage/vault-path.js`; `slugify` unchanged (0
  removed lines in that file). ✅
- `uniqueCanvasPath` and `renameCanvasPath` exported from
  `storage/workspace-vault.js`. ✅
- `node --check` on both modules exits 0. ✅
- `node --test storage/phase1.test.js storage/phase4.test.js` exits 0 with the
  new tests present. ✅
- Full suite exits 0, `# fail 0`, `# pass` = 197 + 5 = 202. ✅
- Code commit `e78f0e1` touches only the four in-scope files
  (`git show --stat e78f0e1`). The ticket frontmatter update is orchestrator-managed
  and was intentionally left unstaged. ✅
- `grep -n "ROOT_CANVAS_PATH\|canvasPathFor" storage/workspace-vault.js` content
  unchanged from 71899aa. ✅

## Issues encountered

- Duplicate concurrent worker on the same ticket (see above). Resolved by
  de-duplication, full re-verification, a stability watch confirming no active
  writer, and a single commit as the surviving worker, per `3b0c72f`'s
  recommended resolution.
