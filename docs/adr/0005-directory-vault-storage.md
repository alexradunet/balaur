# ADR-0005: DirectoryVault — File System Access API as the sole browser vault adapter

> Renumbered from ADR-0004 because `0004-file-unified-node-model.md` landed on main during this project.

**Status:** Accepted
**Date:** 2026-07-24
**Deciders:** Repository owner
**Supersedes:** `IndexedDbVault` as the browser vault adapter; the one-time `localStorage` first-run migration in `app.js`

## Context

Balaur's browser vault was `IndexedDbVault`: opaque origin-private storage the user cannot see, sync, or edit with other tools. The real-folder vault — plain files in a user-chosen directory, syncable with Syncthing, Dropbox, or git, editable in any editor — is the primary user motivation for this project. The File System Access API (`showDirectoryPicker`) provides browser-native directory access without a build step, native shell, or new dependency. It is Chromium-only: Firefox and Safari do not implement it. A hard incompatibility gate is the correct response to a missing platform capability; a fallback adapter would multiply surface for a browser the maintainer does not use.

## Decision

`DirectoryVault` is the sole browser vault adapter. It implements the existing async `VaultStore` contract over a `FileSystemDirectoryHandle` obtained from `showDirectoryPicker({ mode: "readwrite" })`. The constructor takes the handle (the test seam; headless contract tests pass an OPFS handle via `navigator.storage.getDirectory()`). The folder is re-picked every launch with zero persisted handles. Browsers lacking `showDirectoryPicker` or `crypto.subtle` receive a full-screen incompatibility message; no fallback adapter is provided.

Existing data migrates by one-time manual whole-space `.orbit.json` export from the deployed IndexedDB version, then version-2 backup import into the picked folder (the existing `workspace-backup.js` flow re-pointed at a `MemoryVault` staging adapter). `IndexedDbVault` (`storage/indexeddb-vault.js`) and the `localStorage` first-run migration are deleted. `MemoryVault` (test adapter) and `FsVault` (Node reference/tooling) are unchanged.

File-canonical ownership (ADR-0001) is unchanged: this is an adapter swap, not an ownership change. Canonical Markdown files, JSON Canvas documents, the workspace sidecar, and the disposable `MemoryIndex` projection retain their roles and contracts.

## Consequences

- **Chromium-only support matrix.** Chrome, Edge, Brave, and Arc (all Chromium-based) can open a vault. Firefox and Safari see the incompatibility gate. This is an intentional narrowing, not an oversight.
- **Non-atomic writes guarded by `expectedHash`.** `createWritable` is not atomic rename (unlike `FsVault` temp+link). The existing expected-content-hash precondition is the conflict guard; a torn write surfaces through the existing canvas repair-placeholder and entity diagnostic paths.
- **The picked folder's file tree is the vault.** Whole-space import and restore replace the entire file tree of the picked folder. These operations are reachable only through explicitly confirmed destructive flows and validate into a staging vault first.
- **Adopt is additive-only.** Opening a folder with files but no sidecar creates the sidecar and an empty root canvas; pre-existing files are never modified or deleted, and matching files are indexed.
- **No persisted handle.** Re-pick every launch has zero invalidation edge cases. A handle locker is a known additive option deferred by YAGNI.
- **No content cache.** Every `read`, `stat`, and `list` goes to disk so external edits (editor, Syncthing) are visible on demand through a manual "Reload vault" action.

## Verification boundary

Node suite: 172 tests pass (unchanged; `DirectoryVault` is browser-only, like `IndexedDbVault` was). Browser verification: headless contract suite via OPFS handle (constructor injection seam), full-app headless smoke via picker stub (`window.showDirectoryPicker` overridden with an OPFS handle), and manual smoke on real Chromium (picker UX, empty/create, adopt, sidecar open, external-change reconciliation, conflict detection, folder deletion mid-session, migration rehearsal, offline shell reload).
