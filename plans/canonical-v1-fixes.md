# Plan — Canonical files-only v1: review fixes

Status: Accepted for implementation. Follows `plans/canonical-v1-files-only.md` (Tasks F/S/D are committed: `80b1563`, `aead70c`, `77f1fd0`, `e57605e`).

Two independent reviewers (gpt-5.6-sol xhigh, qwen3.8-max-preview) audited the change set. Both: static checks + all 145 Node tests pass, foundation is sound, but the **app-integration paths have blocking data-safety defects** the Node tests don't cover. This plan consolidates and prioritizes their findings. Items marked ✅ were independently verified against the code by the orchestrator.

Fix in priority order. Keep the app bootable, preserve AGENTS.md boundaries, run `node --check` + the 145-test suite after each change, add/adjust Node tests where a fix is Node-testable. Do not commit (orchestrator commits).

## P0 — Critical (block shipping)

### C1 ✅ Johnny Decimal workspaces crash render: `jdKind` not persisted in the sidecar
`storage/workspace-vault.js:38-62` (`toSidecar`) persists `jdCode`/`jdTitle` but drops `jdKind`; `app.js:508` calls `subcanvas.jdKind.toUpperCase()` whenever `jdCode` is set → throws after the boot round-trip through `WorkspaceStore`, rejecting the top-level module (no `window.balaurCanvas`, no SW registration).
**Fix:** persist `jdKind` in `toSidecar` and restore it when the sidecar is loaded back into canvas records; as a belt-and-braces guard, derive `jdKind` from `workspace.johnnyDecimal.entries[code].kind` before render and make `app.js:508` tolerant of a missing `jdKind`. Add a Node test that round-trips a JD workspace (with `jdKind`) through `WorkspaceStore` and asserts `jdKind` survives.

### C2 ✅ Whole-space import is not durable across reload (vault-name mismatch)
`app.js:789-796` imports into `new IndexedDbVault(\`balaur-vault-${uid("import")}\`)` and switches globals in memory only; boot (`app.js:223`) hardcodes `new IndexedDbVault("balaur-vault")`. On reload the imported space (and post-import edits) is stranded; the toast reports success.
**Fix:** validate + rebuild (+ audit) in the staging vault, then **atomically copy into the canonical `balaur-vault`** (e.g. `const snap = await staging.snapshot(); await canonical.restore(snap);`) and re-boot from `balaur-vault`; switch globals only after activation succeeds. Add a Node test: import a bundle into a staging `MemoryVault`, activate into a canonical `MemoryVault`, and assert the canonical vault holds the imported files.

### C3 ✅ Ordinary "Delete node" deletes the whole task entity everywhere
`app.js` `deleteSelection` (~735-744) calls `taskRepository.deleteTask(taskId)` for a task file-node, deleting the canonical task + every placement, with no task-specific confirmation — violates the placement/entity separation (AGENTS.md/README).
**Fix:** node deletion calls `taskRepository.removePlacement(currentCanvasId, node.id)` only (entity + other placements survive). Expose "Delete task everywhere" as a **separate, explicitly confirmed** action (e.g. an inspector menu item) that calls `deleteTask`.

### C4 ✅ Pending canvas edits can be silently lost (create/delete reload + beforeunload + premature "Saved")
`app.js` `createTask`/`deleteSelection` mutate canonical canvases through the repository then replace the whole working workspace via `vaultStore.load()` without flushing pending debounced edits; `beforeunload` (~1021) cannot await the async IndexedDB save; the UI shows "Saved locally" before the promise resolves.
**Fix:** route workspace saves and repository canvas mutations through one **serialized mutation queue**; flush pending edits successfully before repository canvas operations; reload/merge only the affected canvas document (not the whole workspace); update the "Saved" status only after the save resolves; keep a durable failure state. Do not rely on async `beforeunload` for durability (document the limit and/or keep a minimal synchronous unload backstop).

### C5 ✅ Save-time workspace validation is missing (can overwrite entity files / collide canvases)
`storage/workspace-vault.js` `parseSidecar` validates paths/uniqueness/hierarchy on **load**, but `_save`/`toSidecar` accept anything on **write** — a malformed record can point a canvas at `tasks/x.md` and overwrite that task with canvas JSON; duplicate paths can overwrite one canvas with another.
**Fix:** validate the resulting sidecar (record key/id consistency, folded-path uniqueness, `canvases/*.canvas` namespace, hierarchy, root path) **before the first write** in `_save`; refuse to save (throw a typed error) rather than write an unsafe sidecar.

### C6 ✅ Unconditional deletion of unreferenced `canvases/*` files
`storage/workspace-vault.js:219-223` deletes every `canvases/*` file not referenced by the sidecar, unconditionally and without `expectedHash` — a newly created or externally edited canvas absent from a stale in-memory sidecar can be erased.
**Fix:** delete only canvases **previously owned by the loaded sidecar** (track the prior path set), using their last-known `expectedHash`; preserve unknown files (leave them as recovery orphans, optionally with a diagnostic).

### C7 ✅ Repository updates validate domain rules AFTER writing
`storage/task-repository.js:88-102`, `storage/habit-repository.js:78-92`, `storage/journal-event-repository.js:121-135` write `patchFields(...)` output (which accepts syntactically valid but unsupported enums/values) and only then `parse*()` throws — e.g. task status `bogus` is written and indexed as a parse error before the call fails.
**Fix:** parse + fully validate the patched content **before** `vault.write()`; write only validated content and return the already-validated object. Add Node tests asserting an invalid enum/value patch throws and leaves the file unchanged.

### C8 ✅ Event deletion ignores optimistic concurrency and doesn't await index cleanup
`storage/journal-event-repository.js:138-141` deletes unconditionally (no `expectedHash`) and calls `removeFile()` without `await` (unhandled rejection possible).
**Fix:** remove with the `_sourceFor()` hash via `vault.remove(path, { expectedHash: hash })` and `await this.indexer.removeFile(path)`.

### C9 FsVault lacks real no-replace/CAS safety and restore has no rollback
`storage/fs-vault.js:70-91,100-109` — POSIX `rename()` can overwrite a destination created after preflight (defeating `expectedHash`/destination checks); if root→backup succeeds but staging→root fails, restore leaves the vault root missing.
**Fix:** use actual no-replace primitives (e.g. `rename` only after re-validating expected content at commit, or `O_EXCL`-style link/swap) and serialize writers; implement an explicit restore rollback that restores the backup on activation failure. Add Node tests for the conflict + restore-failure paths.

### C10 Whole-space export can silently produce an incomplete backup
`storage/workspace-backup.js:33-50` + `app.js:779-784` — `exportBundle()` omits unreadable files and returns diagnostics, but the app discards them and announces success.
**Fix:** when `exportBundle` diagnostics are non-empty, abort the UI export or require explicit acknowledgement with a prominent "incomplete backup" warning naming the skipped files.

## P1 — Significant

- **S1 ✅ "Reset/Load starter" is not a whole-space reset and doesn't seed starter tasks** (`app.js:322-326`): it replaces canvas metadata/documents but leaves existing task/habit/journal/event files and never calls `seedStarterTasks()` (only first boot seeds). Fix: perform reset through a staged whole-vault replacement and seed starter tasks before activation; first-run and reset must produce the same starter space.
- **S2 ✅ AI file-node context is the path, not the canonical body** (`app.js:436,972` `nodeAIContent`/`nodeTitle`): an AI operator on a task file-node receives `tasks/...md` instead of title/body, contradicting the architecture/generative-canvas docs. Fix: when a file node resolves to an indexed entity, feed the parsed title+body into AI context (preload file-node content asynchronously with diagnostics).
- **S3 `index-integrity.js` is dead code in the app AND wrong for JD canvas paths** (`index-integrity.js:50` derives canvas id from the path basename, not the workspace-aware `canvasIdFromPath` → false MISSING/EXTRA_PLACEMENT for JD canvases; phase10 never tests a placement). Fix: inject the same `canvasIdFromPath`, add a JD-placement test, and wire a "rebuild index" recovery action into the app (or remove it and note it as deferred).
- **S4 `sw.js` dynamically caches every same-origin GET, not just `APP_SHELL`** (`sw.js:65-84`): a same-origin AI-provider `/models` request can be cached despite the "never cache provider requests" rule. Fix: restrict runtime cache writes/fallback to an explicit shell allowlist; bypass authorization/provider requests.
- **S5 Task title editing fires one repository write per input event with no serialization** (`app.js:728`): concurrent keystrokes share a stale source hash → later characters conflict and the persisted title can stop early. Fix: debounce + serialize per-task updates, or save on change/blur.
- **S6 Read-only repair placeholders / vault-unavailable fallback are editable-but-unsavable in the UI** (`workspace-vault.js:132-151`, `app.js:247-251`): mutation controls stay enabled, the app says "Saved," but `WorkspaceStore` skips the record. Fix: disable mutation controls for repair/unavailable workspaces and show a persistent repair/export message.
- **S7 Warm move reconciliation can leave stale rows** (`life-indexer.js:276-291`): coalescing by new path loses `oldPath` when a move is followed by modify/remove; the read-failure branch calls `removeFile()` without awaiting. Fix: preserve move ancestry while coalescing; process old-path removal + new-path indexing in one transaction; await every removal.
- **S8 A malformed duplicate does not suppress the valid "winner"** (`life-indexer.js:72-98,151-169`): `entityId` is assigned only after full entity parsing, so a file with a valid duplicate id but invalid status escapes duplicate detection. Fix: extract constrained identity independently of full typed parsing; detect identity conflicts before projecting.
- **S9 Habit-marker validation can ignore a malformed marker** (`entity-codec.js:190-199`): one valid marker + a later unterminated marker passes the global "any complete marker" check. Fix: tokenize every `balaur:habit-entry` occurrence and require each to match the full grammar.
- **S10 Habit check-in append is not preservation-first** (`habit-repository.js:108-114`): `replace(/\s+$/,"")` strips user trailing whitespace and always appends LF (changes CRLF). Fix: detect the file terminator; append without trimming/rewriting existing bytes.
- **S11 `life-query.js:44` event range compares ISO strings, not instants** — different-offset timestamps can be mis-included/excluded/mis-sorted. Fix: validate bounds once and compare numeric epoch values.
- **S12 `canvas-validate.js` accepts non-JSON-Canvas-1.0 structures** — fractional geometry, `subpath` not starting with `#`, non-standard `backgroundZoom`. Fix: enforce integer geometry, fragment subpaths, and only spec-defined fields (keep the project's required top-level arrays).
- **S13 `vault-path.js` still aliases paths** — `assertSafePath("a\\b")` returns `a/b` despite the no-rewrite contract; hand-written folding misses some default-fold mappings. Fix: reject backslashes; use a complete, tested portable case-fold.
- **S14 Performance (deferrable):** every save rewrites all canvas files + reindexes everything and the IndexedDB change journal grows unboundedly (`workspace-vault.js _save`, `persistWorkspace`→`reconcileWarm`). Fix later: skip unchanged canvases, batch reconciliation, prune the change journal below the indexed revision.

## P2 — Minor

- `refreshLifeViews()` and the module-level `indexStatus` variable in `app.js` are dead code — remove.
- `#resetDemo` double-confirms (its handler + `loadJohnnyDecimalStarter` each `confirm()`).
- Sidecar `.balaur/workspace.json` is counted in `stats.sourceFiles` (sidebar noise).
- `importBundle` "requires empty staging vault" guard has no test; app import runs `rebuild()` but not `auditIndex` in staging.
- Document (README/life-data) that upgrading a legacy localStorage profile drops task workflow state (intended clean break).
- Stale references: `life-indexer.js:4-6` comment (SQLite consumer), `workspace-backup.js:64` v1 message, `docs/design-system.md:3`, `vendor/sqlite/README.md`, and `docs/generative-canvas.md`/`docs/life-data.md`/README overstatements (file-content AI context, transactional move, FsVault atomicity, durable backup/restore, placement-only deletion) — align with the fixed behavior.
- Annotate historical `docs/superpowers/**` rebrand docs as superseded.

## Verification

```
node --check app.js main.js sw.js offline/register.js storage/*.js
node --test storage/phase1.test.js storage/phase2.test.js storage/phase3.test.js \
  storage/phase4.test.js storage/phase4-backup.test.js storage/phase5.test.js \
  storage/phase7.test.js storage/phase8.test.js storage/phase9.test.js \
  storage/phase10.test.js storage/phase-query.test.js
git diff --check
```
Add Node tests for every Node-testable fix (C1, C2, C5, C6, C7, C8, C9, C10, S3, S7–S13). Browser-pending items (boot timing/fallback, IndexedDB durability, UI task flows, offline reload, timezone locales) stay flagged, not claimed.
