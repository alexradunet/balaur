---
phase: ticket
status: ready
project: 001-directory-vault
ticket: 01
blocked-by: []
worker: ""
branch: ""
shared-blast-radius: false
---

# Ticket 01: DirectoryVault adapter

## What to build

A new browser-only `VaultStore` adapter, `DirectoryVault`, over a
`FileSystemDirectoryHandle`. After this ticket, a caller can construct
`new DirectoryVault(handle)` with any directory handle (a real
`showDirectoryPicker` handle or an OPFS handle) and use the full vault
contract against the picked folder's plain files: every read goes to disk so
external edits are visible, writes are guarded by the `expectedHash`
precondition, and the session journal supports warm reconciliation. This is
the storage foundation the boot swap (ticket 02) and the headless contract
suite (ticket 03) build on.

Source of truth for the exact surface: `para/projects/001-directory-vault/plan.md`
§1 ("Architecture → 1. `storage/directory-vault.js`"). Read it in full before
editing. Mirror `storage/memory-vault.js` structure (constructor, `get
revision()`, `_bump`, `_checkPrecondition`, meta assembly) and
`storage/fs-vault.js` disk semantics (real mtime, case-fold-by-listing,
non-atomic writes guarded by `expectedHash`).

## Acceptance criteria

- [ ] `storage/directory-vault.js` created; `export class DirectoryVault extends VaultStore`.
- [ ] Constructor stores the handle, initializes `_revision = 0` and `_journal = []`, and throws `StorageError` (`STORAGE_UNAVAILABLE`) if the handle is missing or `handle.kind !== "directory"`. No `queryPermission`/`requestPermission` anywhere in the module.
- [ ] No content cache: `read`/`stat`/`list` all go to disk.
- [ ] Full public surface implemented, each path passing `assertSafePath` first: `get revision()`, `exists`, `stat`, `read`, `list(prefix="")`, `write`, `remove`, `move`, `snapshot`, `restore`, `changesSince`.
- [ ] Meta shape is the contract's `{ path, mediaType, size, hash, modifiedAt, revision }`; `modifiedAt` is real disk mtime (`new Date(file.lastModified).toISOString()`); `size` is `byteLength(text)`; `hash` is `contentHash(text)` read from disk.
- [ ] `read` of a missing path throws `VaultError` with `code: "NOT_FOUND"`; `stat`/`exists` of a missing path return `null`/`false`.
- [ ] `list` walks recursively via `for await (const entry of dirHandle.values())`, skips nothing, filters by `startsWith(prefix)`, and sorts by plain code-unit comparison (matches MemoryVault/IndexedDbVault, not FsVault's localeCompare).
- [ ] `write` runs the exact MemoryVault precondition table (`undefined` no check; `null` must-not-exist → `ConflictError` `WRITE_CONFLICT`; string must match → `ConflictError` with `details: { expected, actual }`), then case-fold collision check on create (`PathError` `PATH_CASE_COLLISION`), then parent creation + `createWritable` → write → close, then a single `_bump`.
- [ ] `move` mirrors MemoryVault ordering (a failed destination write never deletes the source): read source, destination-exists → `ConflictError`, precondition on source, fold check on destination, INLINE destination write (do NOT delegate to `this.write`), single `_bump(to, "move", hash, from)`, then remove source.
- [ ] `remove` stats first (missing → `NOT_FOUND`), runs the precondition check, removes, journals `operation: "remove"`, returns `true`.
- [ ] `snapshot` returns `{ format: "orbit-vault-snapshot", revision, files: [{ path, mediaType, text }] }` sorted by path; `restore` validates every path, rejects in-snapshot case-fold collisions up front, removes the whole current file tree, resets `_journal`/`_revision`, writes each file via `this.write(..., { expectedHash: null })`, returns `{ revision, count }`.
- [ ] `changesSince(revision)` is a synchronous journal filter (required by `LifeIndexer.reconcileWarm`, `storage/life-indexer.js:284`).
- [ ] DOMException → vault-error mapping table implemented at every boundary, always with `details: { name, message }`: `NotFoundError` on read → `VaultError` `NOT_FOUND`; `NotFoundError` on stat/exists → `null`/`false`; `NotAllowedError`/`SecurityError` → `StorageError` `STORAGE_UNAVAILABLE`; `TypeMismatchError` → `PathError` `PATH_COMPONENT`; `InvalidStateError`/other → `StorageError` `STORAGE_UNAVAILABLE`. Errors already `VaultError` rethrow unchanged.
- [ ] Verify: `node --check storage/directory-vault.js` exits 0; the 172-test suite is unchanged and all pass (no test imports this module); `git diff --check` clean.
- [ ] No Node tests added (DirectoryVault is browser-only; behavioral verification arrives with ticket 03's `contract` subcommand). Commit message matches `git log --oneline -10` style (e.g. `storage: add DirectoryVault adapter over the File System Access API`). Stage only this ticket's paths (`git add storage/directory-vault.js`).

## Blocked by

None — can start immediately. File-disjoint with ticket 04; the two form the
initial parallel frontier.
