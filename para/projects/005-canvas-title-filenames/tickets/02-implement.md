---
phase: implement
status: done
project: 005-canvas-title-filenames
ticket: 02
date: 2026-07-25
commit: a3a9efe
branch: wealthy-blowfish
---

# Implementation: Lift the root-path constraint (root located by sidecar path)

## Summary

Removed the special case that pinned the root canvas to the hardcoded path
`canvases/root.canvas`. The root is now located by its sidecar `path` like every
other canvas (falling back to `canvases/<id>.canvas` when no path is stored);
root identity is the sidecar `rootId` field, not the filename. This is the
contract step that unblocks ticket 03.

## TDD red-green

The two genuinely new tests (Step 5) were added FIRST and confirmed RED against
the old implementation, failing for the right reason:

- `parseSidecar accepts a root canvas at any valid canvases/ path …` → threw
  `SchemaError: Root canvas must use canvases/root.canvas` (the constraint being
  lifted).
- `a slug-named root canvas saves, reloads intact …` → the root file landed at
  `canvases/root.canvas`, so `vault.exists("canvases/life-os.canvas")` was false.

After Steps 1–4 (implementation + assertion updates) both tests went GREEN. The
rewritten `canvasPathFor` test and the updated assertions specify the new
contract. phase4.test.js: 27 pass / 0 fail (was 25; +2 new tests).

## Steps completed

- [x] Pre-flight — verified: non-main worktree `wealthy-blowfish`, branch
  `wealthy-blowfish`; the only dirty files at start were the two ticket `.md`
  coordination files (orchestrator frontmatter: status/worker/branch), not
  in-scope source. Ticket 01 `status: done`. Drift check
  `git diff --stat 71899aa..HEAD -- <in-scope paths>` showed ticket-01 appends in
  `workspace-vault.js` (+42) and `phase4.test.js` (+90); every "Current state"
  excerpt was re-read and matches the live code (line numbers shifted down by the
  appends; content byte-identical). No STOP condition.
- [x] Step 1: Simplify `canvasPathFor` + remove `ROOT_CANVAS_PATH`
  (`storage/workspace-vault.js`) — deleted the constant, replaced `canvasPathFor`
  with the single-arg form and a rewritten doc comment, changed the `toSidecar`
  call to `canvasPathFor(record)`, deleted the `parseSidecar` root-path check.
  Kept the `rootId` local in `toSidecar` (still used by `activeId`). Verified:
  `node --check storage/workspace-vault.js` OK; `grep ROOT_CANVAS_PATH` → no
  matches.
- [x] Step 2: Explicit slug-derived root paths (`app.js`) — `minimalFreshWorkspace`
  root `path:"canvases/balaur.canvas"`; `createGraphStarterWorkspace` sets
  `root.path="canvases/home.canvas"`. Verified: `node --check app.js` OK.
- [x] Step 3: Seven single-arg `canvasPathFor` callers (`app.js` lines 249, 275,
  296, 305, 310, 314, 319) — dropped the second argument. Verified:
  `node --check app.js` OK; `grep "canvasPathFor(.*,.*rootId)\|canvasPathFor(.*,.*)"`
  → no matches.
- [x] Step 4: Rewrite `canvasPathFor` test + replace `ROOT_CANVAS_PATH` literals
  (`storage/phase4.test.js`) — removed `ROOT_CANVAS_PATH` from the import,
  rewrote the test to the new signature (stored path, id fallback, root uses
  `record.path` with no special-case, root id fallback, unsafe paths throw), and
  replaced the literals at the toSidecar assertion, the external-edit write, the
  stale-hash set, and the orphan exists-check with `"canvases/canvas-root.canvas"`.
  Verified: `grep ROOT_CANVAS_PATH` → no matches.
- [x] Step 5: Add root-at-slug `parseSidecar` test + slug-named-root round-trip
  (`storage/phase4.test.js`) — the two new tests. Verified:
  `node --test storage/phase4.test.js` → 27 pass / 0 fail including both new tests.
- [x] Step 6: Four root-path assertion literals (`storage/phase4-backup.test.js`
  lines 72, 150, 220, 277) → `"canvases/canvas-root.canvas"`; fixture line 41 kept
  `path: null`. Verified: `grep canvases/root.canvas` → no matches;
  `node --test storage/phase4-backup.test.js` → 18 pass / 0 fail.
- [x] Step 7: Two root-path assertions in the WorkspaceStore test
  (`storage/phase9.test.js` lines 149, 151) → `"canvases/canvas-root.canvas"`; the
  independent direct-write fixture at lines 46/49/50 left untouched. Verified:
  `node --test storage/phase9.test.js` → 13 pass / 0 fail; `grep canvases/root.canvas`
  matches only lines 46, 49, 50.
- [x] Step 8: Full foundation suite + static checks — `node --check
  storage/workspace-vault.js app.js` OK; `git diff --check` → no whitespace errors;
  full suite → **204 pass / 0 fail** (202 baseline + 2 new tests, exactly as
  predicted; no regression).

## Files changed

- `storage/workspace-vault.js` — deleted `ROOT_CANVAS_PATH`; `canvasPathFor` is
  now single-parameter with no root branch (rewritten doc comment); `toSidecar`
  calls `canvasPathFor(record)`; removed the `parseSidecar` "root must be
  canvases/root.canvas" check (the `canvases/[^/]+\.canvas` guard and case-fold
  collision check remain).
- `app.js` — `minimalFreshWorkspace` root path `canvases/balaur.canvas`;
  `createGraphStarterWorkspace` root path `canvases/home.canvas`; seven
  `canvasPathFor` callers now single-arg.
- `storage/phase4.test.js` — import no longer pulls `ROOT_CANVAS_PATH`;
  `canvasPathFor` test rewritten to the new signature; four root-path literals
  replaced; two new tests (root-at-slug `parseSidecar`, slug-named-root
  round-trip); plus the drift reconciliation noted below.
- `storage/phase4-backup.test.js` — four root-path assertion literals replaced.
- `storage/phase9.test.js` — two root-path assertions in the WorkspaceStore test
  replaced.

## Verification results

```
$ node --check storage/workspace-vault.js app.js   # exits 0
$ git diff --check                                  # no whitespace errors
$ grep -rn "ROOT_CANVAS_PATH" storage/ app.js       # no matches
$ grep -n "canvasPathFor(.*,.*rootId)\|canvasPathFor(.*,.*)" app.js   # no matches
$ grep -n "canvases/root.canvas" storage/phase4.test.js storage/phase4-backup.test.js   # no matches
$ grep -n "canvases/root.canvas" storage/phase9.test.js   # lines 46, 49, 50 only (independent fixture)

$ node --test storage/phase1.test.js … storage/note-repository.test.js
ℹ tests 204
ℹ pass 204
ℹ fail 0
```

All done-criteria checks hold. `git status --short` shows the five in-scope
source files committed in `a3a9efe`; the two ticket `.md` files remain modified
by the orchestrator's frontmatter coordination and were intentionally left
unstaged.

## Issues encountered

**Drift reconciliation (two extra fixture literals in phase4.test.js).** The
done criterion `grep -n "canvases/root.canvas" storage/phase4.test.js → no
matches` could not be satisfied by the ticket's enumerated Step-4 edits alone.
Ticket 01 (planned after this ticket's `planned-at` 71899aa) added two
`renameCanvasPath` tests whose in-memory fixtures use the literal string
`canvases/root.canvas` as an arbitrary path for a canvas whose id is `c-root`
(not `canvas-root`), never routed through `canvasPathFor`/`toSidecar`/
`parseSidecar`/`WorkspaceStore` root logic (former lines 330 and 368). These were
not in the ticket's "Current state" excerpts (written pre-ticket-01) nor in its
step list, but they are in an in-scope file and are the only thing standing
between the post-Step-4 state and the done-criteria grep.

Resolution: changed those two fixture literals from `canvases/root.canvas` to
`canvases/c-root.canvas` (matching their record id). This is provably
behavior-preserving: no assertion in either `renameCanvasPath` test depends on
the root fixture's path string (they assert on the child's path, the affected-id
list, and error cases). Verified safe before applying, and the full suite stays
green. Flagging it here as a deliberate, minimal deviation from the enumerated
steps, made to satisfy an explicit machine-checkable done criterion; a reviewer
may confirm the two lines in the `git diff` for `storage/phase4.test.js`.

No other deviations. `storage/workspace-backup.js` and `storage/vault-path.js`
were not touched; the seven "must NOT change" test files were not touched.
