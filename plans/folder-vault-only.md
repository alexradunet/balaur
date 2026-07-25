# Plan — Folder-vault-only storage (Obsidian-style)

**Status:** Accepted design, pending implementation.
**Supersedes:** the IndexedDB-default boot wired by `plans/canonical-v1-files-only.md` (the file-canonical ownership model itself is unchanged — ADR-0001 stays).
**Companion:** none. This is a self-contained plan.

## 1. Goal

Make the vault exactly Obsidian's model: **a folder of canonical files on disk is
the vault**, opened by pointing at a folder. Eliminate IndexedDB entirely. Files
stay canonical (ADR-0001 intact); SQLite remains a *future disposable index only*
and is out of scope here.

## 2. Decisions log (settled with the owner — do not relitigate)

1. **Files canonical.** ADR-0001 unchanged. No SQLite as source of truth.
2. **Folder-vault-only.** One live adapter, `DirectoryVault`, over the File System
   Access API (`showDirectoryPicker` → `FileSystemDirectoryHandle`). **Zero
   IndexedDB.** `IndexedDbVault` is deleted from the app.
3. **Chromium-only, hard gate.** `showDirectoryPicker` is undefined on
   Firefox/Safari → feature-detect and show a "needs a Chromium-based browser
   (Chrome/Edge/Brave/Arc)" message instead of a dead button.
4. **No persistent vault memory.** No registry, no recents, no auto-open. Every
   launch and every vault switch starts at a landing screen → OS folder picker
   (a user gesture). localStorage survives only for theme, AI settings, and
   Remember-API-key (none of it vault data or a handle).
5. **Landing screen:** Balaur wordmark + one unified **"Open vault folder"**
   button + non-Chromium fallback + an error/status region.
6. **Folder open/create/adopt (additive-only):**
   - has `.balaur/workspace.json` → **open**;
   - **empty** folder → **create** + seed starter tasks (reuse existing first-run seeding);
   - files but **no sidecar** → **adopt**: add `.balaur/` + a minimal root canvas,
     index what matches our codecs, leave everything else as plain files;
   - **never modify or delete pre-existing files**; seeding happens only for an
     empty folder.
7. **External changes:** one manual **"Reload vault"** action (re-read folder,
   rebuild index). `expectedHash` preconditions already prevent silent overwrite
   of an external edit. No auto-refresh / watcher.
8. **Migration: hard break, none.** Shipping this orphans any existing IndexedDB
   data irreversibly (accepted). **Before deploy, export anything worth keeping
   via the current "Export whole space"/"Export .canvas".**
9. **Export/import surface:** keep single-`.canvas` export; **remove** whole-space
   `.balaur.json` bundle export + import from the UI; no zip / copy-vault in v1.
   (The `storage/workspace-backup.js` module and its Node tests stay — only the
   app wiring/buttons are removed — so the 168-test suite is not broken.)

## 3. Non-negotiables carried from AGENTS.md

- JSON Canvas stays standards-compliant; `storage/canvas-validate.js` validates at
  every boundary.
- Writes are canonical-file-first with `expectedHash` preconditions.
- No framework/build step/CDN dependency. Native strict ES modules.
- CSS in the existing named layers with Balaur tokens.
- Update docs in the same change as behavior (§14).
- Distinguish implementation from browser verification (§13/§16).

## 4. New file: `storage/directory-vault.js`

`export class DirectoryVault extends VaultStore` — a live folder adapter. Model it
on `storage/memory-vault.js` (journal/revision/`changesSince`/`snapshot`/`restore`)
and `storage/indexeddb-vault.js` (`_checkPrecondition`, meta shape), but backed by a
`FileSystemDirectoryHandle`. **No in-memory content cache** — the folder is the
source of truth and external edits must be visible on re-read; every `read`/`list`
goes to disk.

Constructor: `constructor(handle)` storing the directory handle, `_revision = 0`,
`_journal = []`.

Reuse helpers: `mediaTypeFor` (vault-store.js), `contentHash` (content-hash.js),
`byteLength`/`assertSafePath`/`caseFoldKey` (vault-path.js), and the error types
(vault-errors.js: `ConflictError`, `PathError`, `VaultError`).

Methods (all async; call `assertSafePath(path)` on every path first):

- `get revision()` → `this._revision`.
- `_dirHandle(path, {create})` — walk path segments with
  `getDirectoryHandle(seg, { create })`; throw `PathError` on failure. Leaf file
  via `getDirectoryHandle(parent).getFileHandle(name, { create })`.
- `_meta(fileHandle, path)` — `getFile()` → text → `{ path, mediaType:
  mediaTypeFor(path), size: byteLength(content), hash: await contentHash(content),
  modifiedAt: new Date(file.lastModified).toISOString(), revision: this._revision }`.
- `list(prefix = "")` — recursively walk from the prefix directory using
  `for await (const entry of dirHandle.values())`; recurse `kind === "directory"`,
  collect `_meta` for `kind === "file"`. Return the array. (Skip nothing — `.balaur`,
  `canvases`, entities, widgets are all real files.)
- `read(path)` — file handle → text; if missing throw
  `VaultError("Not found: …", { code: "NOT_FOUND" })`.
- `stat(path)` — `_meta` or `null` if missing.
- `exists(path)` — boolean.
- `write(path, content, { expectedHash, mediaType } = {})`:
  1. read existing meta (or `null`); run the same precondition check as
     `IndexedDbVault._checkPrecondition` (`undefined`→skip; `null`→must not exist;
     else hash must match) → `ConflictError` on mismatch;
  2. ensure parent dirs (`_dirHandle(parent, { create: true })`);
  3. `const w = await fileHandle.createWritable(); await w.write(content); await w.close();`
     <!-- ponytail: createWritable→close is not atomic-rename like FsVault's temp+rename;
          expectedHash preconditions guard conflicts. Revisit if partial-write corruption appears. -->
  4. bump revision + push journal `{ revision, path, operation: "write", hash }`;
  5. return the new meta.
- `remove(path, { expectedHash } = {})` — precondition check; `fileHandle.remove()`;
  journal `operation: "remove"`.
- `move(from, to, { expectedHash } = {})` — read `from`, `write(to, …)`, remove
  `from`; journal `operation: "move"` with `oldPath: from`. Mirror MemoryVault move.
- `snapshot()` — `list("")` then `read` each → `{ files: [{ path, content, … }] }`.
- `restore(snapshot)` — `write` each file. Mirror MemoryVault.
- **`changesSince(revision)`** — `this._journal.filter(e => e.revision > revision)`.
  **Required:** `LifeIndexer.reconcileWarm(fromRevision)` calls it (see
  `storage/life-indexer.js`). Without it, warm reconciliation breaks.

Permission note: `DirectoryVault` does **not** manage permissions. Boot obtains the
handle via `showDirectoryPicker()` (session-granted). Reads/writes within the
session need no further gesture. (A future registry would add
`queryPermission`/`requestPermission`; not in v1.)

Browser-only: File System Access is unavailable in Node, so this module is verified
by `node --check` and manual browser testing only — exactly like `IndexedDbVault`.
No Node unit test (cannot run headless without a real picker/gesture).

## 5. Rewire `app.js`

Grep for the symbols below; line numbers drift, so locate by name.

**Imports (top of file):**
- Remove `import { IndexedDbVault } from "./storage/indexeddb-vault.js";`
- Remove `import { exportBundle, importBundle, serializeBundle, assertCompleteExport } from "./storage/workspace-backup.js";`
- Add `import { DirectoryVault } from "./storage/directory-vault.js";`
- Add `import { MemoryVault } from "./storage/memory-vault.js";` (only if a staging
  vault is still needed by reset — see §5.5; otherwise omit).

**Delete the legacy localStorage workspace path:**
- `loadDocument()`, `freshWorkspace()`, `normalizeWorkspace()`, and the
  localStorage-reading `loadWorkspace()` (the `WORKSPACE_KEY` /
  `balaur-canvas-v1` / `balaur-title` readers). Keep `createJohnnyDecimalStarterWorkspace()`
  (the seeded starter builder) — it is the source for the empty-folder create path.
- Add a small `minimalFreshWorkspace()` that returns a `version:1` workspace with a
  single root canvas whose document is `{ nodes: [], edges: [] }` and a default title
  (no localStorage, no demo canvas). This is the **adopt** path's workspace.

**Rewrite `bootCanvasApp()` into two pieces:**
- `openVault(vault, { seed })` — the shared core used by first-open, "open another",
  and "reload":
  1. `const store = new WorkspaceStore(vault);`
  2. `const had = await hasWorkspace(vault);`
  3. if `!had`: `const ws = seed ? createJohnnyDecimalStarterWorkspace() : minimalFreshWorkspace(); await store.migrate(ws);`
     (`migrate` writes with `expectedHash: null`, i.e. additive — it only creates
     `.balaur/workspace.json` + `canvases/root.canvas`, never touching other files.)
  4. `const result = await store.load();` (existing diagnostics/repair-placeholder
     behavior is retained.)
  5. set `workspace`, `vaultStore`, `window.balaurVaultStore`; `setCanonicalWritable(…)`;
     `configureLifeRuntime(vault)`; `seedBundledWidget(vault)`;
  6. if `seed` (empty folder): `await seedStarterTasks(); workspace = (await store.load()).workspace;`
  7. `await Promise.all([lifeIndexer.rebuild(), componentCardCatalog.rebuild(), widgetCatalog.rebuild()]);`
     update `setIndexStatus(…)`;
  8. set `currentCanvasId`/`documentData` and render (reuse the existing post-boot
     render code that follows today's `bootCanvasApp`).
- `bootCanvasApp()` now just shows the landing screen and returns a promise that
  resolves once a vault is opened. Keep `vaultReady = bootCanvasApp();
  window.balaurVaultReady = vaultReady; await vaultReady;` at the module tail.

**Landing / open / reload / switch wiring (set up early, before `await vaultReady`):**
- `#openVaultFolder` click → guard `if (!("showDirectoryPicker" in window))` (show
  fallback); else `const handle = await showDirectoryPicker({ mode: "readwrite" });`
  → `const vault = new DirectoryVault(handle);` → decide `seed`:
  `const empty = !(await hasWorkspace(vault)) && (await vault.list("")).length === 0;`
  → `await openVault(vault, { seed: empty });` → hide landing, reveal `.app-shell`.
  Wrap in try/catch → surface `error.message` in the landing error region;
  `AbortError` (user cancelled the picker) is silent.
- `#reloadVault` click → re-run `openVault(currentVault, { seed: false })` on the
  **existing** handle (no picker, no gesture needed for in-session reads). Re-render.
- `#openAnotherVault` click → show the landing screen again (re-pick on button click).
- Keep a module-level `currentVault` reference for reload.

**Remove bundle export/import wiring:**
- Delete `exportWorkspace()` (the `exportBundle`/`serializeBundle` handler) and
  `importCanvas(file)` (the `importBundle` staging handler).
- Keep `exportCanvas()` (single `.canvas` export) and the Ctrl/Cmd+S → `exportCanvas`
  binding.
- Remove the button bindings for `#exportWorkspaceButton`, `#importButton`,
  `#fileInput`, `#exportJDWorkspace`. Keep `#exportButton` → `exportCanvas`.

**`resetDemo` (≈ line 420):** rewire off IndexedDB. New behavior: confirm → build
`createJohnnyDecimalStarterWorkspace()` → save into the **current** `vaultStore`
(a normal `save()`, which CAS-overwrites owned canvases and removes orphans) →
`seedStarterTasks()` → rebuild indexes → re-render. No staging vault needed; drop the
`IndexedDbVault("balaur-vault-…")` staging dance (and the `MemoryVault` import if it
was only for this).

## 6. `index.html`

- Add a **landing overlay** as the first child of `<body>` (sibling before
  `<div class="app-shell">`), visible by default; `.app-shell` starts hidden/inert
  until a vault opens:
  ```html
  <div id="vaultLanding" class="vault-landing">
    <h1 class="vault-landing__wordmark">Balaur</h1>
    <p class="vault-landing__hint">Open a folder as your vault. Its files are the source of truth.</p>
    <button id="openVaultFolder" class="button primary">Open vault folder</button>
    <p id="vaultLandingMessage" class="vault-landing__message" role="status"></p>
  </div>
  ```
  On non-Chromium, JS fills `#vaultLandingMessage` with the "needs a Chromium-based
  browser (Chrome/Edge/Brave/Arc)" notice and disables `#openVaultFolder`.
- **Remove:** `#importButton` (line ~41), `#fileInput` (~43),
  `#exportWorkspaceButton` (~64), `#exportJDWorkspace` (~184).
- **Keep:** `#exportButton` "Export .canvas" (~42), `#resetDemo` (~65).
- **Add** to `.sidebar-bottom` (near ~62–65): two `nav-item` buttons
  `#reloadVault` ("Reload vault") and `#openAnotherVault` ("Open another vault").

## 7. Styles

Add landing-overlay rules in the appropriate existing layer (`@layer shell` per
`styles/layers.css`), using Balaur tokens; center the wordmark/button; respect
`prefers-reduced-motion`. Reveal `.app-shell` by toggling a class/`hidden` when a
vault opens. Keep it minimal.

## 8. `sw.js`

- `CACHE_NAME` is currently `"balaur-shell-v12"` → bump to `"balaur-shell-v13"`.
- `APP_SHELL`: **add** `"./storage/directory-vault.js"`. **Remove**
  `"./storage/indexeddb-vault.js"` (no longer imported). Remove
  `"./storage/workspace-backup.js"` only after confirming no remaining importer
  (`grep -rn "workspace-backup" app.js`); the module file itself stays for its tests.
- Keep all paths relative (GitHub Pages subpath).

## 9. Delete

- `storage/indexeddb-vault.js` — no remaining importer after the rewire (verify with
  `grep -rn "indexeddb-vault\|IndexedDbVault" --include="*.js" .` excluding
  `node_modules`/`vendor`). It is browser-only and untested in Node, so deletion
  breaks no test suite.

Keep `storage/workspace-backup.js` and its tests (`phase4.test.js`,
`phase4-backup.test.js`) — only the app UI wiring is removed.

## 10. Documentation updates (same change, per §14)

- **AGENTS.md:** §1 (remove "IndexedDB/OPFS … File System Access API" as the storage
  trio → vault is a user folder via File System Access; note Chromium-only), §3 repo
  map (replace `indexeddb-vault.js` with `directory-vault.js`), §5 runtime/init model
  (landing-screen → folder-vault boot; no localStorage migration; no IndexedDB),
  §5.1 (SW still shell-only; user data is in the folder), §7 (DirectoryVault is the
  browser adapter; IndexedDbVault gone), §13 browser-pending list (folder-vault boot,
  permission, reload), §16 definition of done. Note the cache is now `balaur-shell-v13`.
- **docs/architecture.md:** adapter list + "future desktop shell" note (web is now
  folder-vault-only; a Tauri shell adds an `fs`-kind adapter + config-file registry).
- **docs/life-data.md:** vault adapters section — `DirectoryVault` replaces
  `IndexedDbVault` as the browser adapter.
- **docs/offline.md:** user data lives in the user's folder (not IndexedDB); shell
  cache unchanged; cache version bump.
- **README.md:** user-visible behavior — opening/creating/adopting a folder vault,
  Chromium requirement, "Reload vault", removed bundle export/import, the hard-break
  data note.
- **docs/adr/0001-file-canonical-life-data.md:** add a short addendum that the browser
  adapter is now `DirectoryVault` (folder) and IndexedDB is removed; the file-canonical
  decision itself is unchanged. (No new ADR required — this is an adapter swap, not an
  ownership change.)

## 11. `browser-check` skill / smoke suite

`.pi/skills/browser-check/` currently asserts the IndexedDB/file-index boot. Rework
the boot-related assertions for the folder-vault model:
- The landing screen renders; on Chromium the "Open vault folder" button is enabled.
- Boot now requires a user gesture + a real OS picker, which **cannot be driven
  headlessly** — so the folder-vault boot, permission grant, adopt/create, and reload
  are **manual-browser-verified / browser-pending** (per §13). Document this in the
  skill and AGENTS.md rather than faking it.
- Keep the assertions that still apply (no uncaught console errors, valid JSON Canvas
  document, offline shell from cache). Read `.pi/skills/browser-check/SKILL.md` for
  the headless event-retargeting caveat before editing the driver.

## 12. Verification

Static (must pass):
```bash
node --check storage/directory-vault.js app.js
git diff --check
node --test \
  storage/phase1.test.js storage/phase2.test.js storage/phase3.test.js \
  storage/phase4.test.js storage/phase4-backup.test.js storage/phase5.test.js \
  storage/phase7.test.js storage/phase8.test.js storage/phase9.test.js \
  storage/phase10.test.js storage/phase-query.test.js
```
(The 168-test suite must still pass — we only *add* a browser-only module and remove
app wiring; `workspace-backup.js` and all tested modules stay.)

Browser (manual, Chromium — browser-pending per §13):
1. Launch → landing screen; open an **empty** folder → creates + seeds starter tasks.
2. Open a folder with **files but no `.balaur`** → adopts (adds `.balaur/`, indexes
   matching files, pre-existing files untouched).
3. Open a folder that **has `.balaur/workspace.json`** → opens it.
4. Edit a vault file externally → click **Reload vault** → change appears; a
   conflicting save raises a ConflictError rather than overwriting.
5. **Open another vault** → back to landing → pick a different folder → switches.
6. Single `.canvas` export still works; bundle export/import buttons are gone.
7. Offline reload serves the shell from `balaur-shell-v13`.
8. Firefox/Safari → landing shows the Chromium-required notice, no dead button.

## 13. Data-loss warning (deploy gate)

Shipping this **irreversibly orphans any data in the old IndexedDB vault.** Before
deploying to `main`, export anything worth keeping from the live site via the current
"Export whole space" / "Export .canvas". The owner has accepted this hard break.

## 14. Definition of done

- Folder is the sole vault; zero IndexedDB in the running app.
- Open/create/adopt are additive-only; pre-existing files never modified/deleted.
- `DirectoryVault` implements the full `VaultStore` contract **plus `changesSince`**;
  warm reconciliation works.
- Landing screen + Chromium gate + Reload + Open-another wired.
- Bundle export/import removed from UI; single-`.canvas` export retained; 168-test
  suite green; `node --check` clean.
- SW cache bumped to `balaur-shell-v13` with `directory-vault.js` added.
- Docs (§10) updated; browser-pending behavior labeled, not claimed.
