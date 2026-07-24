---
phase: spec
status: done
project: 001-directory-vault
date: 2026-07-24
---

# Spec: DirectoryVault — the File System Access API as the sole vault

## Problem Statement

The browser vault is `IndexedDbVault`: opaque origin-private storage the user
cannot see, sync, or edit with other tools. It blocks the real-folder vault
(grill 2026-07-24): files on disk in a user-chosen directory, syncable with
Syncthing/Dropbox/git, editable in any editor. The one-time `localStorage`
first-run migration in `app.js` is the last remnant of the pre-vault era and
dies with it. Browsers without the File System Access API (Firefox, Safari)
must fail with a clear gate, not a dead button or a broken boot.

## Solution

Add `storage/directory-vault.js`: `DirectoryVault`, a `VaultStore` adapter over
a `FileSystemDirectoryHandle`. The constructor takes the handle (the test
seam). Boot becomes: incompatibility gate → landing screen → user picks a
folder via `showDirectoryPicker({ mode: "readwrite" })` → `DirectoryVault`
opens → `WorkspaceStore` loads (or creates/adopts) → workspace renders. The
folder is re-picked every launch; no handle is persisted anywhere. Delete
`storage/indexeddb-vault.js` and the `localStorage` workspace migration. Gate
browsers lacking `showDirectoryPicker` or `crypto.subtle` with a full-screen
"requires a Chromium-based desktop browser" message and no fallback adapter.
Existing data migrates by one-time whole-space `.orbit.json` export from the
currently deployed IndexedDB version, then version-2 backup import into the
picked folder (the existing `storage/workspace-backup.js` flow, re-pointed at
a `MemoryVault` staging adapter and the live `DirectoryVault`).

## User Stories

1. As a desktop user on Chrome/Edge/Brave/Arc, I want to pick a folder at launch, so that its plain files are my vault and I can sync or edit them with any tool.
2. As a user, I want an empty picked folder to be seeded with the graph starter workspace and starter entities, so that I start with a working space.
3. As a user, I want a picked folder that already has files but no `.orbit/workspace.json` to be adopted (sidecar + empty root canvas added, existing files untouched and indexed where they match the codecs), so that I can vault a folder of existing notes.
4. As a user, I want a picked folder with a valid sidecar to open as-is, so that my space persists across launches.
5. As a user, I want to re-pick the folder on every launch with zero stored state, so that nothing about my vault lives outside the folder.
6. As a user on Firefox or Safari (or a non-secure context), I want a full-screen message that Balaur requires a Chromium-based desktop browser, so that I understand the failure instead of seeing a dead UI.
7. As a user who cancels the folder picker, I want to stay on the landing screen with no error noise, so that I can pick when ready.
8. As a user, I want a "Reload vault" action that re-reads the same folder handle and rebuilds the index, so that changes made externally (editor, Syncthing) become visible on demand.
9. As a user, I want an "Open another vault" action that returns to the landing screen, so that I can switch folders mid-session.
10. As a user editing a file externally, I want a conflicting app save to fail with a conflict message instead of overwriting my edit, so that external work is never silently lost.
11. As a user whose vault folder is deleted or unlinked mid-session, I want saves to fail with a clear "files unavailable" status and read-only UI, so that I know to re-open the vault.
12. As an existing user of the deployed IndexedDB version, I want to export a whole-space `.orbit.json` before the switch and import it into my picked folder on the new version, so that I keep my data through a documented one-time migration.
13. As a user, I want whole-space export to keep working against the folder vault, so that backups remain possible (AGENTS.md §4.5).
14. As a user, I want theme and AI-provider settings to keep using localStorage, so that those preferences survive without being vault data.
15. As a tester or agent, I want `DirectoryVault` to accept any `FileSystemDirectoryHandle` in its constructor, so that headless contract tests run against an OPFS handle without driving the native picker.
16. As a tester, I want the landing handler to call the global `showDirectoryPicker` at call time, so that the browser-check driver can stub it with an OPFS handle and run the full smoke suite headlessly.
17. As a tester, I want the folder picker to require `mode: "readwrite"`, so that read-only grants fail fast at open instead of at first save.

## Implementation Decisions

### New module: `storage/directory-vault.js`

Exports `class DirectoryVault extends VaultStore` (base from
`storage/vault-store.js`). Browser-only, like `IndexedDbVault` was: verified by
`node --check` and browser tests, never by the Node runner (File System Access
does not exist in Node). Reuses existing helpers only: `mediaTypeFor`
(vault-store.js), `contentHash` (content-hash.js), `byteLength` /
`assertSafePath` / `caseFoldKey` (vault-path.js), and the error types
(vault-errors.js). No in-memory content cache: the folder is the source of
truth; every `read`/`stat`/`list` goes to disk so external edits are visible.

Constructor: `constructor(handle)`. Stores the `FileSystemDirectoryHandle`,
initializes `_revision = 0` and `_journal = []`. Throws `StorageError`
(`STORAGE_UNAVAILABLE`) if the handle is missing or `handle.kind !==
"directory"`. Does not manage permissions: the picker grants readwrite for the
session; no `queryPermission`/`requestPermission` in v1.

Full surface (every path argument passes `assertSafePath` first; meta shape is
the contract's `{ path, mediaType, size, hash, modifiedAt, revision }`):

- `get revision()` — session-local monotonic counter (same semantics as `MemoryVault`/`FsVault`).
- `async exists(path)` — boolean; a `NotFoundError` DOMException means `false`.
- `async stat(path)` — meta or `null`. `modifiedAt` is `new Date(file.lastModified).toISOString()` (real disk mtime); `size` is `byteLength(text)`; `hash` is `contentHash(text)` read from disk.
- `async read(path)` — file text; missing file throws `VaultError` with `code: "NOT_FOUND"` (the code `WorkspaceStore.loadWorkspace` keys on for repair placeholders).
- `async list(prefix = "")` — recursive walk from the root via `for await (const entry of dirHandle.values())`; recurse `kind === "directory"`, collect meta for `kind === "file"`; skip nothing (`.orbit`, `canvases`, entities, widgets, and foreign files are all listed); filter by `startsWith(prefix)`; sort by plain code-unit path comparison (matches `MemoryVault`/`IndexedDbVault`).
- `async write(path, content, { expectedHash, mediaType } = {})` — (1) stat the existing file; (2) run the exact precondition table from `MemoryVault._checkPrecondition`: `undefined` → no check, `null` → must not exist (`ConflictError` otherwise), string → existing hash must match (`ConflictError` with `details: { expected, actual }` otherwise); (3) on create, case-fold collision check by comparing `caseFoldKey(path)` against `list("")` results (`PathError` `PATH_CASE_COLLISION`, same as `FsVault._checkFoldCollision`); (4) create parent directories with `getDirectoryHandle(segment, { create: true })` per segment; (5) `getFileHandle(name, { create: true })` → `createWritable()` → `write(text)` → `close()`; (6) bump revision, push `{ revision, path, operation: "create"|"modify", hash }` to the journal; (7) return meta with `modifiedAt: new Date().toISOString()`. `createWritable` is not atomic-rename (unlike `FsVault` temp+link); the `expectedHash` precondition is the conflict guard, and a torn write surfaces through the existing canvas repair-placeholder and entity diagnostic paths.
- `async remove(path, { expectedHash } = {})` — stat first; missing → `VaultError` `NOT_FOUND`; precondition check; `fileHandle.remove()`; journal `operation: "remove"`; return `true`.
- `async move(from, to, { expectedHash } = {})` — read `from` (missing → `NOT_FOUND`); destination exists → `ConflictError`; case-fold check on `to`; precondition check on `from`; write `to` with `expectedHash: null`; remove `from`; journal `operation: "move"` with `oldPath: from`; return meta for `to` (content hash preserved). Mirrors `MemoryVault.move` ordering so a failed destination write never deletes the source.
- `async snapshot()` — `list("")` then `read` each; returns `{ format: "orbit-vault-snapshot", revision, files: [{ path, mediaType, text }] }` sorted by path. Identical shape to `MemoryVault.snapshot`.
- `async restore(snapshot)` — validate every path with `assertSafePath` and reject case-fold collisions within the snapshot up front (`PathError` `PATH_CASE_COLLISION`, same as the other adapters); then remove every file currently returned by `list("")` (the whole picked tree's files; empty directories may remain, harmless), reset `_journal`/`_revision`, and write each snapshot file. Returns `{ revision, count }`. Documented semantics: restore replaces the entire file tree of the picked folder. It is only reachable through explicitly confirmed destructive flows (whole-space import, Reset starter), and import validates into a staging vault first.
- `changesSince(revision)` — synchronous journal filter, identical to `MemoryVault`/`FsVault`. Required: `LifeIndexer.reconcileWarm` calls `this.vault.changesSince(fromRevision)` (`storage/life-indexer.js:284`); without it warm reconciliation breaks.

DOMException → vault-error mapping (wrap at each adapter boundary; include
`details: { name, message }` from the DOMException):

| DOMException | Mapping |
|---|---|
| `NotFoundError` on read | `VaultError` `NOT_FOUND` |
| `NotFoundError` on stat/exists | `null` / `false` |
| `NotAllowedError`, `SecurityError` (permission denied or revoked mid-session) | `StorageError` `STORAGE_UNAVAILABLE` |
| `TypeMismatchError` (a directory occupies a file path or vice versa) | `PathError` `PATH_COMPONENT` |
| `InvalidStateError` and other unexpected DOMExceptions | `StorageError` `STORAGE_UNAVAILABLE` |
| `AbortError` | only occurs at the picker (landing layer), never inside the adapter |

### Boot flow: `app.js` (targeted edits only; AGENTS.md §12 forbids mass-formatting)

Module-level placeholder workspace and event wiring stay exactly as they are
(app.js:130 region). The landing overlay is static HTML visible by default;
`.app-shell` starts `hidden` and `inert` and is revealed when a vault opens.
Module tail keeps its shape: `vaultReady = bootCanvasApp();
window.orbitVaultReady = vaultReady; await vaultReady;` then the existing
`window.orbitCanvas` assignment. `window.orbitVaultReady` resolves when a vault
has opened, or immediately when the incompatibility gate fires (with
`vaultStore = null`); `window.orbitCanvas` is exposed in both cases (gated mode
runs over the placeholder workspace behind the full-screen overlay).

Exact boot sequence:

1. `bootCanvasApp()` runs the incompatibility gate first: if `typeof window.showDirectoryPicker !== "function"` or `!globalThis.crypto?.subtle`, fill `#vaultLandingMessage` with "Balaur needs a Chromium-based desktop browser (Chrome, Edge, Brave, or Arc) to open a vault folder.", disable `#openVaultFolder`, call `setIndexStatus("Files unavailable", …)`, and return. No fallback adapter.
2. Otherwise wire `#openVaultFolder.onclick` to the pick handler. The handler calls the global `showDirectoryPicker({ mode: "readwrite" })` at call time (no module-scope capture of the function reference, so the browser-check driver can stub it). `AbortError` (picker cancelled) is swallowed: stay on the landing, no message. Any other error goes to `#vaultLandingMessage`.
3. On a handle: `const vault = new DirectoryVault(handle)`; detect folder state: `hadSidecar = await hasWorkspace(vault)`, `empty = !hadSidecar && (await vault.list("")).length === 0`; call `openVault(vault, { seed: empty })`; on success hide the landing and remove `hidden`/`inert` from `.app-shell`.
4. `openVault(vault, { seed })` is the shared core used by first open, Open another, and Reload vault:
   1. `const store = new WorkspaceStore(vault);`
   2. `const had = await hasWorkspace(vault);`
   3. if `!had`: `await store.migrate(seed ? createGraphStarterWorkspace() : minimalFreshWorkspace());` — `migrate` writes with `expectedHash: null`, so this is additive-only: it creates `.orbit/workspace.json` and `canvases/root.canvas` and never touches pre-existing files (the adopt behavior);
   4. `const result = await store.load();` (existing diagnostics and read-only repair placeholders are retained); empty-workspace guard stays;
   5. assign `workspace`, `vaultStore`, module-level `currentVault = vault`, `window.orbitVaultStore = store`; run the existing `setCanonicalWritable(…)` diagnostic logic; `configureLifeRuntime(vault)`; `await seedBundledWidget(vault)`;
   6. if `seed`: `await seedGraphStarterEntities(); workspace = (await store.load()).workspace;` (the existing first-run pattern; `seed` replaces the old localStorage-based `firstRun` computation);
   7. `await Promise.all([lifeIndexer.rebuild(), componentCardCatalog.rebuild(), widgetCatalog.rebuild()]);` and the existing `setIndexStatus(…)`;
   8. the existing post-boot block: set `currentCanvasId`/`documentData`/`camera`/title, `renderWorkspaceNavigation(); render(); setTimeout(fitView, 50);`
   9. on error before any vault has opened: surface `error.message` in `#vaultLandingMessage` and stay on the landing. On error during Reload: use the existing save-state/index-status failure affordances plus `setCanonicalWritable(false, …)`.
5. `#reloadVault.onclick` → `openVault(currentVault, { seed: false })` on the existing handle (in-session reads need no gesture), then re-render. This is the external-change reconciliation path.
6. `#openAnotherVault.onclick` → `await flushPendingWorkspaceEdits()`, then re-show the landing (hide + `inert` `.app-shell`, unhide landing). The next pick runs step 3 with the new handle.

New tiny helper `minimalFreshWorkspace()`: a `version: 1` workspace with the
single root canvas, empty `{ nodes: [], edges: [] }` document, default title
("Balaur"), default camera. No localStorage, no demo canvas. This is the adopt
path's workspace; `createGraphStarterWorkspace()` (existing, app.js:49) is the
empty-folder create path.

### Deletions and dead-reference cleanup (app.js, targeted)

- `import { IndexedDbVault } from "./storage/indexeddb-vault.js";` (app.js:2) — replaced by `import { DirectoryVault } from "./storage/directory-vault.js";` and `import { MemoryVault } from "./storage/memory-vault.js";` (staging).
- The `demoCanvas` constant (app.js:24 region) — its only consumer is `loadDocument()`.
- `loadDocument()`, `freshWorkspace()`, `normalizeWorkspace()`, and the localStorage-reading `loadWorkspace()` (app.js:180–211 region). `normalizeWorkspace`'s JD stripping already lives in `parseSidecar` (workspace-vault.js), so nothing is lost.
- The `WORKSPACE_KEY` constant (app.js:125); keep `ROOT_CANVAS_ID` on that line.
- The localStorage clauses of the old `firstRun` computation (app.js:334) — subsumed by the `seed` parameter.
- `storage/indexeddb-vault.js` — the file. Verify with `grep -rn "indexeddb-vault\|IndexedDbVault" --include="*.js" --include="*.mjs" .` (excluding `vendor/`, `node_modules/`, `teach/`) that no importer remains.
- Stale comments naming `IndexedDbVault` in `storage/vault-store.js:4` and `storage/workspace-vault.js:9` — reword to name `DirectoryVault`.
- Kept localStorage uses (not vault data, not the migration): theme (`orbit-canvas-theme`), AI settings and Remember-API-key (`AI_SETTINGS_KEY`/`AI_SECRET_KEY`).

### Staging re-point (import migration + Reset starter)

- `importCanvas(file)` version-2 branch (app.js:1380 region): staging becomes `new MemoryVault()` (pure JS, browser-importable); after the existing staging index rebuild and `auditIndex` pass, `const snapshot = await stagingVault.snapshot(); const canonicalVault = vaultStore.vault; await canonicalVault.restore(snapshot);` then the existing `new WorkspaceStore(canonicalVault)` reload choreography unchanged. `#importButton`/`#fileInput` stay — this is the migration path. The existing confirm dialog already warns the canonical vault is replaced; README migration steps direct users to pick an empty folder first (restore replaces the picked folder's file tree).
- `exportWorkspace()` stays unchanged (operates on `vaultStore.vault`; AGENTS.md §4.5 keeps whole-space backups a feature). `#exportWorkspaceButton`, `#exportButton`, Ctrl/Cmd+S → `exportCanvas` all stay.
- `loadGraphStarter()` (app.js:403 region): staging becomes `new MemoryVault()`; `canonicalVault` becomes `vaultStore.vault` instead of `new IndexedDbVault("orbit-vault")`; the rest of the staging → seed → snapshot → restore → reload choreography is unchanged.

### Shell: `index.html`, `styles/shell.css`, `sw.js`

- `index.html`: add the landing overlay as the first child of `<body>`, before `.app-shell`: a `#vaultLanding` container holding an `h1` wordmark ("Balaur"), a one-line hint ("Open a folder as your vault. Its files are the source of truth."), `<button id="openVaultFolder" class="button primary">Open vault folder</button>`, and `<p id="vaultLandingMessage" role="status"></p>`. Add `hidden inert` attributes to `.app-shell` in markup. Add two `nav-item` buttons to `.sidebar-bottom` near Export whole space / Reset starter: `#reloadVault` ("Reload vault") and `#openAnotherVault` ("Open another vault"). Keep `#importButton`, `#fileInput`, `#exportWorkspaceButton`, `#resetDemo`.
- `styles/shell.css` (`@layer shell`, Balaur tokens): `.vault-landing` fixed full-viewport, centered column above the app shell; respect `prefers-reduced-motion`. Minimal.
- `sw.js`: bump `CACHE_NAME` from `orbit-shell-v12` to `orbit-shell-v13`; in `APP_SHELL` remove `"./storage/indexeddb-vault.js"` and add `"./storage/directory-vault.js"` and `"./storage/memory-vault.js"` (now imported by app.js, so required for offline boot). Keep `"./storage/workspace-backup.js"`. Verify every module app.js imports is listed.

### Documentation (same change, AGENTS.md §14)

- New `docs/adr/0004-directory-vault-storage.md` (accepted): context (real-folder vault motivation; IndexedDB is opaque origin storage; Chromium-only support reality); decision (DirectoryVault is the sole browser adapter over `showDirectoryPicker({ mode: "readwrite" })`; hard incompatibility gate on missing `showDirectoryPicker` or `crypto.subtle`; re-pick per launch with zero persisted handles; one-time manual `.orbit.json` export/import migration; IndexedDB adapter and localStorage first-run migration deleted); consequences (Chromium-only support matrix; non-atomic `createWritable` guarded by `expectedHash`; the picked folder's file tree is the vault, so import/restore replace it; adopt is additive-only). States explicitly that file-canonical ownership (ADR-0001) is unchanged: this is an adapter swap, not an ownership change.
- `docs/architecture.md`: adapter trio line (~10) and boot list (~35) → DirectoryVault landing-picker boot; VaultStore paragraph (~75) → DirectoryVault is the browser default, folder-backed, no content cache; "Future packaging" paragraph (~105) → browser directory access is now shipped, a future desktop shell would add its own fs-kind adapter under the same contract; verification-boundary paragraph → Node suite still 172 tests, browser checks reworded.
- `docs/life-data.md`: adapter list (~22) and vault-writes paragraph (~253) → DirectoryVault replaces IndexedDbVault (folder-backed, disk mtime, non-atomic write + expectedHash guard); drop the "Upgrading a legacy localStorage profile is a clean break" sentence (the migration no longer exists).
- `docs/offline.md`: runtime-pieces bullet (~11) → user files live in the user-picked folder, never in the SW cache; cache version v12 → v13 in both mentions; precache list gains `directory-vault.js`/`memory-vault.js`, loses `indexeddb-vault.js`.
- `README.md`: feature bullet (~27) and storage section (~131–135) → folder vault, Chromium requirement, re-pick per launch, Reload/Open another, localStorage limited to theme/AI settings; browser-requirements line (~185) → File System Access API, drop IndexedDB/OPFS; new short "Migrating from the IndexedDB version" subsection: export whole space from the deployed version before the switch, pick an empty folder in the new version, Import the `.orbit.json`.
- `AGENTS.md`: §3 repo map (`indexeddb-vault.js` line → `directory-vault.js`; note `memory-vault.js` doubles as the browser staging adapter) and the paragraph after the map; §5 boot model renumbered: (1) incompatibility gate, (2) landing → `showDirectoryPicker({ mode: "readwrite" })`, (3) DirectoryVault opens and WorkspaceStore loads/migrates (create if empty, adopt if files-but-no-sidecar, open if sidecar), (4) index rebuild, (5) render + `window.orbitCanvas`, (6) progressive SW registration — delete the localStorage first-run migration step and state that no handle is persisted; §5.1 cache name (currently stale at `orbit-shell-v7` in the doc; correct to `orbit-shell-v13`) and "IndexedDB owns user files" → "the user-picked folder owns user files"; §7 adapter list (DirectoryVault is the browser adapter; IndexedDbVault gone); §13 browser-pending list reworded (see Testing Decisions) with the Node count unchanged at 172 and a note that the browser-check smoke suite now asserts the landing/gate, runs the OPFS contract suite, and stubs the picker for full-app smoke.
- `CONTEXT.md` is not edited by this change; see Domain flags.

## Testing Decisions

Good tests here exercise the adapter through the `VaultStore` contract (external
behavior: meta shapes, precondition outcomes, error codes), not DirectoryVault
internals. The contract is already adapter-neutral and Node-verified: the 172
test suite (`storage/phase*.test.js`) runs against `MemoryVault` and proves the
contract; `FsVault` proves the disk-backed semantics DirectoryVault mirrors
(real mtime, case-fold-by-listing, non-transactional writes). DirectoryVault is
browser-only (File System Access has no Node implementation), exactly as
`IndexedDbVault` was: `node --check` plus browser verification. The Node suite
count stays 172; no Node tests are added or removed.

Seams, highest first:

1. **Headless contract suite via OPFS (settled in the grill).** `DirectoryVault` takes a `FileSystemDirectoryHandle` in its constructor; `navigator.storage.getDirectory()` returns the same interface (origin-private, no picker, no gesture). The browser-check driver runs one async `eval` expression that dynamically imports `/storage/directory-vault.js` from the served origin, creates a fresh OPFS subdirectory, and asserts the contract: create/read/stat/exists/list ordering and prefix filter; the three `expectedHash` outcomes (null-create conflict, hash mismatch with `details`, correct hash); `NOT_FOUND` on read; case-fold collision on create; move plus destination-exists conflict; remove plus remove precondition; snapshot/restore round-trip; `changesSince` journal ordering; `TypeMismatchError` → `PATH_COMPONENT` (write onto a directory name). Returns `{ ok, failures }`. Lives in the browser-check skill (new `contract` subcommand or a documented `eval` recipe), no new dependencies.
2. **Full-app headless smoke via picker stub (proposed; pending confirmation — see Seams pending confirmation).** The driver `eval`s `window.showDirectoryPicker = async () => (await navigator.storage.getDirectory()).getDirectoryHandle("vault-smoke", { create: true })`, then performs a real CDP click on `#openVaultFolder`. The existing smoke assertions 1–10 then run unchanged: no console errors, node render count, `Files · N indexed` status, selection frame, card-creation guards, background note creation, live JSON Canvas validity, reload persistence (same profile → same OPFS origin → re-stub and re-pick the same subdirectory), and `--offline` shell reload from `orbit-shell-v13`. This requires exactly one thing from app code: the landing handler references the global `showDirectoryPicker` at call time (already specified above). The alternative is leaving full-app boot as manual-only and limiting headless coverage to the landing, the gate, and the contract suite.
3. **Manual smoke (real Chromium, browser-pending per AGENTS.md §13):** real picker UX; empty folder → create + starter seeding; files-but-no-sidecar folder → adopt with foreign files untouched and matching files indexed; sidecar folder → open; edit a vault file in an external editor → Reload vault shows the change; a conflicting app save raises the conflict toast instead of overwriting; delete or rename the vault folder externally mid-session → saves fail with the read-only "files unavailable" affordance; Open another vault switches folders; import a `.orbit.json` into an empty picked folder (the migration rehearsal); export whole space from the folder vault; Firefox or Safari shows the gate message with a disabled button; offline reload serves the shell from `orbit-shell-v13`.

The smoke suite's current IndexedDB-boot assertions (`.pi/skills/browser-check/`)
are reworked to the landing/gate + stub flow above; read
`.pi/skills/browser-check/SKILL.md` (including the headless event-retargeting
caveat) before editing the driver.

AGENTS.md §13 browser-pending list becomes: (1) DirectoryVault open/write/restore
behavior, permission loss, and externally deleted directories; (2) vault-gate
boot, reload + re-pick persistence, and first-render budget; (3) external-change
reconciliation via Reload vault; (4) task create/edit/complete and Today
projections over the folder vault; (5) version-2 export/import round-trip against
the folder vault; (6) offline reload and cache upgrade from `orbit-shell-v12`;
(7) timezone/local-date boundaries; (8) malformed-file repair in the running UI.

## Seams confirmation record

The grill settled seam 1 (constructor injection + OPFS contract tests via
browser-check; real picker is manual smoke).

- **Seam 2 (picker stub for full-app headless smoke): confirmed yes** (owner, 2026-07-24). The driver stubs `window.showDirectoryPicker` with an OPFS handle and CDP-clicks the real landing button, keeping the full smoke suite headless. The implementation constraint is binding: the landing handler references the global `showDirectoryPicker` at call time (no module-scope capture of the function reference).

## Out of Scope

- Persisted directory handle / handle locker: YAGNI for v1; re-pick every launch has zero invalidation edge cases (grill).
- Firefox/Safari fallback adapter: hard gate instead; one adapter, one engine (grill).
- Mobile app, Tauri/Electron/Neutralino shells, Bun runtime: separate later projects (grill).
- Persistent index / OPFS SQLite: doubly deferred (grill); the disposable `MemoryIndex` is unchanged.
- Auto-refresh or filesystem watcher for external changes: one manual Reload vault action is the v1 answer (accepted plan `plans/folder-vault-only.md` §2.7).
- Atomic-rename writes: `createWritable` + `expectedHash` preconditions is the v1 guard; revisit only if torn-write corruption appears in lived use.
- Removing whole-space bundle export/import from the UI: `plans/folder-vault-only.md` §2.9 proposed this, but the grill's migration path requires import and AGENTS.md §4.5 keeps backups a feature; both stay.
- `queryPermission`/`requestPermission` flows: session grant from the picker suffices; no persisted handle means no re-grant path to manage.
- Multi-device sync conflict design: validated during implementation via manual smoke, not designed up front (grill).
- TypeScript/typed toolchain: separate project `002-typed-toolchain` if ever (grill).
- Cleaning up empty directories left after `restore`: harmless; YAGNI.

## Further Notes

- **Divergences from `plans/folder-vault-only.md` (accepted design), resolved in favor of the later grill artifact and the orchestrator's task context:** (1) migration is the one-time `.orbit.json` export/import, not the plan's hard break with bundle-UI removal; (2) the decision record is a new ADR-0004, not the plan's ADR-0001 addendum; (3) the starter builder is `createGraphStarterWorkspace()` — the plan's `createJohnnyDecimalStarterWorkspace` name predates ADR-0003 and no longer exists in the codebase. The plan's open/create/adopt folder states (§2.6) and Reload vault action (§2.7) are adopted unchanged; they are not contradicted by the grill and are required for a coherent folder UX.
- **Deploy gate (data loss):** shipping this orphans any data left in the old IndexedDB vault. Before merging to `main`, export the whole space from the live deployed site and import it on the new version (README migration subsection). The owner accepted this in the grill.
- **AGENTS.md §5.1 is stale:** it names cache `orbit-shell-v7`; `sw.js` is at `orbit-shell-v12`. The doc pass corrects it to `orbit-shell-v13`.
- **`teach/*.html`** reference `indexeddb-vault.js` as historical lesson content, not app modules; they are not in the app shell and are left as-is.
- **`window.orbitVaultReady` / `window.orbitVaultStore` / `window.orbitCanvas`** keep their existing meanings; `orbitVaultStore` is `null` until a vault opens and in gated mode; integration probes use `--wait "window.orbitCanvas"` as today.
- The `beforeunload` note about async writes (app.js tail region) remains true: durability comes from the serialized mutation queue; `createWritable().close()` is awaited inside it.

## Domain flags

Terms for `CONTEXT.md` reconciliation (not edited here):

- **Vault** — the current entry says "Browser runtime uses `IndexedDbVault`". After this change the browser runtime uses `DirectoryVault`; propose: "Browser runtime uses `DirectoryVault` over a user-picked folder (File System Access API); Node tooling and tests use `FsVault` or `MemoryVault`."
- **DirectoryVault** (proposed entry) — the browser `VaultStore` adapter over a `FileSystemDirectoryHandle` from `showDirectoryPicker({ mode: "readwrite" })`. The picked folder's plain files are the vault; no content cache, no persisted handle; the folder is re-picked every launch.
- **Vault gate / landing** (proposed entry) — the full-screen landing screen shown at every launch: wordmark, "Open vault folder" button, and status region. On browsers without `showDirectoryPicker` or `crypto.subtle` it becomes the incompatibility gate and never dismisses.
- **Adopt** (proposed entry) — opening a folder that has files but no `.orbit/workspace.json`: the sidecar and an empty root canvas are added (additive-only), matching files are indexed, and pre-existing foreign files are never modified or deleted.
