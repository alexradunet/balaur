---
phase: implement
status: done
project: 001-directory-vault
ticket: 01
date: 2026-07-25
plan: plan.md
commit: 56be884
branch: 001-directory-vault
---

# Implementation: DirectoryVault adapter (Plan §1, Stage 1)

## Steps completed

- [x] Pre-flight: non-main worktree `/home/balaur/.paseo/worktrees/10x2zhef/001-directory-vault`, branch `001-directory-vault`, clean `git status`. Ticket `status: ready`, `blocked-by: []`. Drift check (`git diff --stat e3103b2..HEAD`) shows changes only in files outside this ticket's scope (app.js, note-repository work, docs = Stages 2/3/4 and a separate landed project); all dependency files this module mirrors (`vault-store.js`, `memory-vault.js`, `fs-vault.js`, `vault-errors.js`, `vault-path.js`, `content-hash.js`) are unchanged. Baseline 172-test suite confirmed green before edits.
- [x] Read plan §1 in full plus spec, ADR-0004, and the reference adapters (MemoryVault structure, FsVault disk semantics, IndexedDbVault browser-only precedent).
- [x] Created `storage/directory-vault.js` per plan §1, literally:
  - Constructor stores the handle, initializes `_revision = 0` and `_journal = []`, throws `StorageError` (`STORAGE_UNAVAILABLE`) on a missing handle or `handle.kind !== "directory"`. No `queryPermission`/`requestPermission` (verified by grep, zero matches).
  - No content cache: `read`/`stat`/`exists`/`list` all resolve the file from disk via `_file`/`_walk` → `getFile()`.
  - Private helpers: `_bump` (identical to MemoryVault), `_checkPrecondition` (exact MemoryVault table, including `details: { expected, actual }` on hash mismatch), `_checkFoldCollision` (FsVault style, case-fold by `list("")`), `_wrap` (DOMException → vault-error mapping, rethrows `VaultError` unchanged), `_dirFor`/`_fileHandle`/`_file` (low-level handle resolvers; raw DOMExceptions propagate so callers choose the NotFoundError interpretation), `_stat` (internal meta builder; NotFoundError → null), `_walk` (recursive `for await ... dirHandle.values()`).
  - Public surface, each path through `assertSafePath` first: `get revision()`, `exists`, `stat`, `read`, `list(prefix="")`, `write`, `remove`, `move`, `snapshot`, `restore`, `changesSince`.
  - Meta shape `{ path, mediaType, size, hash, modifiedAt, revision }`; `modifiedAt` is real disk mtime (`new Date(file.lastModified).toISOString()`) in `_stat`, `new Date().toISOString()` on write/move returns; `size` is `byteLength(text)`; `hash` is `contentHash(text)` read from disk.
  - `write`: stat → precondition table → fold check on create → `_fileHandle(create)` → `createWritable` → write → close → single `_bump("create"|"modify")`.
  - `move`: MemoryVault ordering (read source, destination-exists `ConflictError`, precondition on source, fold check on destination, INLINE destination write — never delegates to `this.write`, single `_bump(to, "move", hash, from)`, then remove source), so a failed destination write never deletes the source.
  - `remove`: stat first (missing → `NOT_FOUND`), precondition, remove, journal `"remove"`, return `true`.
  - `snapshot`: `{ format: "balaur-vault-snapshot", revision, files: [{ path, mediaType, text }] }` sorted by path. `restore`: validate every path + reject in-snapshot case-fold collisions up front, remove the whole current file tree, reset `_journal`/`_revision`, write each file via `this.write(..., { expectedHash: null })`, return `{ revision, count }`.
  - `changesSince(revision)`: synchronous journal filter (required by `LifeIndexer.reconcileWarm`, `storage/life-indexer.js:284`).
  - DOMException mapping at every boundary, always `details: { name, message }`: `NotFoundError` on read → `VaultError` `NOT_FOUND` (the code `workspace-vault.js:135,150` keys on for repair placeholders — verified); `NotFoundError` on stat/exists → `null`/`false` (handled in `_stat` before `_wrap`); `NotAllowedError`/`SecurityError`/`InvalidStateError`/other → `StorageError` `STORAGE_UNAVAILABLE`; `TypeMismatchError` → `PathError` `PATH_COMPONENT`; `VaultError` rethrown unchanged.
- [x] Verified (see below). Committed only `storage/directory-vault.js`.

## Files changed

- `storage/directory-vault.js` — created (291 lines). The browser-only `DirectoryVault` adapter over a `FileSystemDirectoryHandle`, full `VaultStore` surface.

## Verification results

- `node --check storage/directory-vault.js` → exit 0.
- `node --test storage/phase1.test.js storage/phase2.test.js storage/phase3.test.js storage/phase4.test.js storage/phase4-backup.test.js storage/phase5.test.js storage/phase7.test.js storage/phase8.test.js storage/phase9.test.js storage/phase10.test.js storage/phase-query.test.js` → `tests 172`, `pass 172`, `fail 0` (unchanged; no test imports this module).
- `git diff --check` → clean (exit 0, no output).
- `grep -n "queryPermission\|requestPermission" storage/directory-vault.js` → no matches.
- Commit `56be884` "storage: add DirectoryVault adapter over the File System Access API"; `git show --stat HEAD` confirms a single file changed (`storage/directory-vault.js`, 291 insertions). Staged only `git add storage/directory-vault.js` (never `-A`/`-a`).

## Issues encountered

- **No Node-test red-green loop in this ticket, by design.** The spec's Testing Decisions place this adapter's behavioral seam in Stage 3 (ticket 03's headless OPFS `contract` subcommand), and the ticket states "No Node tests added (DirectoryVault is browser-only)." The tdd skill says to test only at the spec's pre-agreed seams and not invent new ones, and the plan/STOP conditions forbid changing the 172-test count. So this ticket's verification is `node --check` + the unchanged 172 suite + `git diff --check`, exactly the boundary `IndexedDbVault` was held to. Behavioral verification of the surface arrives with ticket 03.
- **`list` and foreign files with unsafe names (observation, not a deviation).** Plan §1 specifies `list` collects `_stat` meta for each file entry, and `_stat` resolves through `_dirFor` → `assertSafePath`. A foreign file in the picked folder whose name `assertSafePath` rejects (e.g. a trailing space, a Windows device name, a non-NFC name) would make `list` throw rather than skip it. I implemented the plan literally (no special-casing), since adding unsafe-name tolerance is not in the plan and would be guessing. Flagging for the reviewer/orchestrator: if Adopt of an arbitrary user folder must tolerate such files, that is a follow-up decision, not a Stage 1 change.
- **`restore` count.** Returned `count: prepared.length` (equal to `snapshot.files.length` for valid input, and null-safe when `snapshot` is absent), matching FsVault's `restore`; the plan's `snapshot.files.length` wording is satisfied for all valid snapshots.
- **Artifact not committed.** The briefing fixes this ticket's only staged path as `storage/directory-vault.js`, so this coordination artifact is written to the working tree for the orchestrator to read but is not staged or committed.

## Domain flags

No contradictions with `CONTEXT.md`. Vocabulary used as settled: **Vault**, **DirectoryVault**, **Adopt** (the additive-only open is enabled by this adapter's `expectedHash: null` creates via `WorkspaceStore.migrate`; the boot swap itself is ticket 02). No new glossary terms warranted.
