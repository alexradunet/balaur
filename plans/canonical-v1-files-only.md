# Plan — Canonical files-only v1 (remove SQLite, fix kept-code bugs, in-memory index)

Status: Accepted for implementation.
Supersedes the SQLite-backed parts of `plans/markdown-canonical-sqlite-index.md`.
See `docs/adr/0001-file-canonical-life-data.md` for the file-canonical decision.

## 1. Goal

Make the file-canonical system the **initial canonical version** of Balaur:

- Canonical `.md` life entities + `.canvas` documents + `.balaur/workspace.json` sidecar, stored in the vault, are the **only** source of truth.
- **No SQLite.** Remove the SQLite store, its schema migrations, the SQL index port, and the legacy marker-task/migration scaffolding. SQLite (or another persistent index) is a **deferred future optimization**, not part of this version.
- Queries (Today, calendar, habits, search) are served by an **in-memory index derived from the files at boot** (`LifeIndexer` + `MemoryIndex`), rebuilt from the vault and reconciled on change. This is pure JS over the vault, so it works identically in the browser (GitHub Pages, no special headers), Node, and Tauri.
- Remove the "old way" entirely: marker tasks, SQLite-owned tables, `localStorage` as source of truth, and the "Direction vs Currently" dual framing in docs.
- Keep it simple. Do not add a framework, build step, CDN dependency, or a persistent database.

## 2. Guiding principles (non-negotiable)

- JSON Canvas documents stay standard (`{nodes, edges}`, node types text/file/link/group). No custom node types or app-only fields.
- Markers, where any remain, are inert `<!-- balaur:... -->` HTML comments. Tasks are canonical `.md` files placed by standard `file` nodes; a canvas node id is placement, not identity.
- The vault (files) is canonical; the in-memory index is a disposable projection rebuilt from files. Deleting the index loses nothing.
- Preservation-first frontmatter: patch only Balaur-owned fields; never reflow, reorder, or drop unknown keys, comments, BOM, or line endings.
- Validate at boundaries (import, AI, storage, parse). Escape dynamic HTML. Keep security boundaries (sandboxed widgets, provider keys in sessionStorage).
- Keep the static app bootable at every committed step. No `npm install`, no build step.

## 3. Target architecture

```
Vault (IndexedDbVault browser / FsVault node / MemoryVault tests)
  ├─ .balaur/workspace.json        sidecar: hierarchy + camera + JD metadata (no documents)
  ├─ canvases/*.canvas            canonical JSON Canvas documents
  ├─ tasks/*.md  habits/*.md  habit-logs/*.md  journal/*.md  events/*.md   canonical entities
  └─ (widgets/*.html)             sandboxed file-node widgets

Boot (vault-first, async):
  1. open vault  2. load sidecar + canvas docs  3. build in-memory index (LifeIndexer.rebuild)
  4. render from the in-memory working set  5. reconcile warm on vault changes

Runtime query layer:
  repositories (FileTaskRepository, HabitRepository, JournalEventRepository) read/write entity files;
  LifeIndexer projects files into MemoryIndex; the app queries MemoryIndex for Today/calendar/habits.
```

No `storage/life-store.js`, no `vendor/sqlite` usage at runtime, no `window.balaurLifeStore`/`balaurLifeReady`, no task markers, no `localStorage` workspace as source of truth.

## 4. Task F — Foundation (additive, Node-verified; app keeps booting on the old path)

Scope: `storage/**` and storage tests only. Do **not** edit `app.js`, `main.js`, `sw.js`, or `offline/**` in this task. Do **not** delete `storage/life-store.js` yet (the app still boots on it; Task S removes it). You **may** delete `storage/sqlite-index.js`, `storage/task-migration.js`, `storage/phase6.test.js`, and `storage/phase-sqlite.test.js` (none are used by the running app).

Address these reviewer findings in the kept code. For each, update or add Node tests (`node --test`) and keep the whole suite green.

### F1. Strict JSON Canvas validator (`storage/canvas-validate.js`)
- `isCanvas({})` must return **false**: require both `nodes` and `edges` to be arrays.
- Validate every standard optional field and node type (`text`/`file`/`link`/`group`), non-empty string ids, finite geometry numbers, and edge endpoints that reference existing node ids.
- Make this the **single** validator. Replace the weaker `isCanvasDoc` in `storage/life-indexer.js` with it. (Task S swaps `app.js` to it.)

### F2. Preservation-first body edits (`storage/habit-repository.js`, `storage/journal-event-repository.js`)
- Body updates currently parse + fully reserialize, dropping unknown frontmatter keys/comments/ordering/BOM/line-endings. Fix to the task-repository pattern: patch `updated-at` surgically with `patchFields`, then replace only the bytes after the closing frontmatter delimiter (see `replaceBody` in `storage/task-repository.js`). Preserve the detected line terminator.

### F3. Workspace path safety (`storage/workspace-vault.js`)
- Require **unique case-folded** `canvases/*.canvas` paths; reject a sidecar that points a canvas at an entity path (e.g. `tasks/x.md`), at `.balaur/workspace.json`, or at the same path as another canvas.
- Validate key/record id consistency and parent/portal hierarchy references.
- A missing or invalid canvas file must load as a **read-only** placeholder that preserves the raw malformed content and is **never overwritten** with an empty document on save. Add a repair affordance flag rather than silent empty-substitution. Update `storage/phase4.test.js` accordingly (it currently enshrines empty substitution).

### F4. Task placement/deletion safety (`storage/task-repository.js`)
- `addPlacement`: **fail** if the canvas is missing (do not recreate it empty); validate the resulting document with the strict `isCanvas` before writing; validate geometry (finite, non-negative width/height), node-id uniqueness, and color; pass an `expectedHash` precondition.
- `deleteTask` ("delete everywhere"): do not swallow placement-removal errors silently — abort if placements cannot be resolved; remove the canonical file with its last-known `expectedHash` (not an unconditional delete).

### F5. Duplicate ids: no winner (`storage/life-indexer.js`)
- Detect duplicate `balaur-id`s **before** applying typed projections; suppress **every** conflicting typed projection and placement (do not let `Map.set`/last-write pick a winner). Record `DUPLICATE_ID` diagnostics for all conflicting files.
- Incremental `indexFile()` must re-evaluate identity conflicts (a new file can create a duplicate).
- Warm reconciliation must process a `move` as transactional old-path removal + new-path indexing (honor `oldPath`; do not leave stale source rows).

### F6. Real domain/temporal validation (`storage/frontmatter.js`, `storage/entity-codec.js`, repositories)
- Reject impossible dates/instants (e.g. `"2026-13-99"`, `"2026-99-99T88:77:66Z"`); validate task status / habit frequency / habit-entry status enums, weekday ranges, and IANA timezones.
- **Do not derive a local `YYYY-MM-DD` by slicing a UTC instant.** Derive local dates using the event's intended timezone. Add timezone-boundary tests.

### F7. Strict habit-entry marker validation (`storage/entity-codec.js`)
- Validate every habit-entry marker: required id + habit id, no duplicate attributes, valid status, finite value, valid instant. A malformed marker classifies the **source file** as invalid (a diagnostic), never a malformed projection row.

### F8. Strict path normalization (`storage/vault-path.js`)
- `assertSafePath()` must **reject** empty/trailing/duplicate components (`a//b`, `a/b/`) rather than silently normalizing them. Define and test the exact portable case-fold collision key (proper Unicode case folding, not just `toLowerCase`).

### F9. In-memory index correctness (`storage/memory-index.js`)
- Include `_diagId` in the transaction snapshot so rollback leaves no observable id gaps.

### F10. Line-ending fidelity (`storage/entity-codec.js`, `storage/task-repository.js`)
- Preserve the detected line terminator/separator. Do not force LF during body replacement; parse CRLF files without a leading blank line.

### F11. IndexedDB restore durability (`storage/indexeddb-vault.js`) — logic fix (browser-verified later)
- Precompute **all** content hashes before opening a single write transaction in `restore()` (avoid `TransactionInactiveError` from awaiting WebCrypto mid-transaction). Detect case-fold collisions on restore.

### F12. FsVault safety (`storage/fs-vault.js`)
- Reject symlinked components / realpaths outside the root; serialize writes; use a temporary sibling + atomic rename (no-replace) for writes/moves; enforce folded-path uniqueness; stage `restore()` and swap only after validation. Keep `storage/phase9.test.js` green and add safety tests.

### F13. Import validation (`storage/workspace-backup.js`)
- Validate **all** file-node references in imported canvases; require an empty staging vault and write files with `expectedHash: null`; rebuild + audit the index in staging; activate only on success.
- Decision (clean break): version-1 bundles are **not** supported in canonical v1 — reject them with a clear error (document this; the old way is intentionally gone).

### F14. Integrity audit completeness (`storage/index-integrity.js`)
- Beyond source existence/hash/duplicate/dangling checks: compare typed-row fields to re-parsed files, detect missing/extra typed rows, verify placement completeness against canvas files, and treat outstanding parse diagnostics as unhealthy.

### F15. In-memory runtime query layer
- Ensure `LifeIndexer` + `MemoryIndex` + the repositories work standalone (no SQLite) as the runtime layer. Provide the app-facing queries the UI needs over the in-memory index with **consistent camelCase shapes**: today/open tasks (filter + stable sort), tasks by status, habits with latest daily state + streak, journal for a date, events in a date range. Add a small query module (e.g. `storage/life-query.js`) if that keeps `app.js` clean; otherwise document the exact index methods Task S should call. Add Node tests for these queries.

### F-verification
```
node --check storage/*.js
node --test storage/phase1.test.js storage/phase2.test.js storage/phase3.test.js \
  storage/phase4.test.js storage/phase4-backup.test.js storage/phase5.test.js \
  storage/phase7.test.js storage/phase8.test.js storage/phase9.test.js storage/phase10.test.js
```
(phase6 and phase-sqlite are deleted.) All must pass. **Do not commit.** Report the final test count and a file-by-file change summary.

## 5. Task S — The atomic swap (depends on Task F being complete and green)

Scope: `app.js`, `main.js`, `sw.js`, `offline/register.js`; delete `storage/life-store.js`. Rewire the app from SQLite/markers to the file-only runtime layer built in Task F. This is browser-bound: make the changes carefully, run `node --check` on every touched file, keep the app bootable, and document precisely what needs browser verification.

- **Boot vault-first (async):** open the vault (`IndexedDbVault`), load the sidecar + canvas documents (`WorkspaceStore`), build the in-memory index (`LifeIndexer.rebuild` over `MemoryIndex`), then render. One-time import of a legacy `localStorage` workspace on first run is acceptable; after that `localStorage` is not a source of truth. Keep a graceful message if the vault is unavailable.
- **Tasks:** replace marker-task `createTask` (marker node + `store.upsertTask`) with `FileTaskRepository.createTask` (canonical `.md` file + standard `file`-node placement). Completion/edit via `updateTask`/`completeTask`. Remove `TASK_MARKER_RE`, `buildTaskText`, `reconcileTaskMarkers`, and the `balaur:life-store-ready` listener.
- **Today / views:** query the in-memory index (Task F15) instead of `lifeStore`. Preserve current rendering/interaction behavior.
- **Export/import:** use the version-2 file bundle (`storage/workspace-backup.js`) instead of `store.exportSnapshot/importSnapshot`. Remove `resetLifeDatabase` (or repurpose to "rebuild index from files").
- **Stats/status:** replace the "SQLite <version> · local" sidebar state with vault/index status (e.g. "Files · N indexed"). Remove `window.balaurLifeStore`/`balaurLifeReady` usage.
- **main.js:** remove `import "./storage/life-store.js";`; order startup so the vault boot completes before the canvas app is exposed.
- **sw.js:** remove `./storage/life-store.js`, `./vendor/sqlite/sqlite3.mjs`, `./vendor/sqlite/sqlite3.wasm` from `APP_SHELL`; add any new runtime storage modules (e.g. `storage/life-query.js`); bump `CACHE_NAME`. Keep all paths relative; no `skipWaiting()` auto-reload.
- Use the strict `isCanvas` (Task F1) for import/AI/storage validation in `app.js`.

### S-verification
```
node --check app.js main.js sw.js offline/register.js storage/*.js
```
Confirm no remaining references to `balaurLifeStore`, `balaurLifeReady`, `life-store`, `reconcileTaskMarkers`, or task markers. **Do not commit.** List every behavior that requires browser verification (boot, task create/complete, Today, export/import, offline reload).

## 6. Task D — Documentation re-frame (depends on Task S)

Scope: `AGENTS.md`, `docs/architecture.md`, `docs/life-data.md`, `docs/offline.md`, `docs/generative-canvas.md`, `docs/adr/0001-file-canonical-life-data.md`, `README.md` as needed.

- Re-frame as the **canonical files-only v1**: files are the source of truth; the in-memory index is a disposable runtime projection; **SQLite is a deferred future optimization** (note OPFS needs COOP/COEP GitHub Pages can't set).
- Remove the "Direction (ADR-0001) vs Currently" dual framing and any contradiction (the reviewer flagged AGENTS.md as internally inconsistent). Describe one shipped reality.
- Update the repository map (remove `life-store.js`, `sqlite-index.js`, `task-migration.js`; add any new module like `life-query.js`), the startup model (vault-first async), the data-boundaries section, and the test command + test count.
- `docs/life-data.md`: replace the SQLite schema/migration content with the canonical file layouts + frontmatter contracts + the in-memory index description. Keep the date conventions (local `YYYY-MM-DD`, ISO instants, IANA zones; scheduling intent separate from deadline).
- `docs/offline.md`: update the precache list (no SQLite Wasm), cache name, and probes.
- Do not describe unshipped browser work as shipped; mark browser-pending items explicitly.

## 7. Global constraints for all implementators

- Work in `/home/alex/jsoncanvas`. Native ES modules, vanilla JS, no build step, no new runtime dependency.
- **Do not run `git commit`/`git push`.** Make changes, run the verification commands, and report. The orchestrator verifies and commits.
- Keep each step bootable; do not leave the app importing a deleted module.
- Preserve AGENTS.md non-negotiable boundaries (standard JSON Canvas, inert markers, escaped HTML, sandboxed widgets, sessionStorage keys).
- A concurrent Balaur rebrand exists on `main`; do not rename functional identifiers (`balaur-*` keys, `window.balaurCanvas`) gratuitously.

## 8. Browser-pending (cannot be verified in Node — flag, don't claim)

IndexedDB vault persistence/restore/quota; vault-first boot + first-render budget; task create/complete/Today in the UI; export/import round-trip; offline reload + Service Worker upgrade; timezone/local-date boundaries in a real browser.
