---
phase: implement
status: done
project: 001-directory-vault
ticket: 02
date: 2026-07-25
commit: 15d5a5c
branch: 001-directory-vault
---

# Implementation: Boot swap — Vault gate, DirectoryVault boot, shell, delete IndexedDbVault

Landed as three logical commits on `001-directory-vault` (starting HEAD `ab4432d`):

- `602a937` app: boot through the Vault gate and DirectoryVault
- `595df04` shell: add the Vault gate overlay and bump precache to balaur-shell-v13
- `15d5a5c` storage: delete IndexedDbVault and reword stale adapter comments

## Pre-flight and STOP-condition checks

- Worktree `/home/balaur/.paseo/worktrees/10x2zhef/001-directory-vault`, branch
  `001-directory-vault` (non-main), clean status at start. Ticket status `ready`,
  blocker `[01]` done (DirectoryVault shipped at `storage/directory-vault.js`).
- **Drift**: app.js drifted +140 lines since the plan's `e3103b2` baseline because
  done sibling tickets 01 (adapter) and 04 (note repository, ADR-0004) landed.
  Every edit region was re-verified against the live code before editing; the
  code shapes matched the plan's "Current state" excerpts (only line numbers
  shifted). Not a contradiction — proceeded.
- **Finding F1 (verified SAFE, no STOP)**: `normalizeWorkspace`'s path backfill
  (`if(!record.path)`) never fires on the starter — the root gets `path=null`
  then `continue`, and every `hub()` canvas plus `project-city-break` carry an
  explicit `path`. Its JD-field and non-hub/project `kind` deletes are no-ops
  (the starter sets no JD fields; all kinds are `hub`/`project`).
  `freshWorkspace`'s title/camera are immediately overridden by
  `root.title="Home"`/`root.camera=null`. Dropping both calls is
  behavior-preserving.
- **Adopt additive-only (verified, no STOP)**: `WorkspaceStore` constructor sets
  `ownedCanvasPaths = new Set()` (workspace-vault.js:175), populated only by
  `load()` (:192). `openVault` calls `store.migrate()` before `store.load()`, so
  `_save`'s orphan-removal loop iterates an empty set. `migrate` (`fresh=true`)
  writes only `canvases/root.canvas` and `.balaur/workspace.json` with
  `expectedHash: null` (additive creates). Pre-existing foreign files are never
  modified or removed.
- **Binding constraint (Seam 2)**: the pick handler references the global
  `window.showDirectoryPicker({ mode: "readwrite" })` at call time (app.js:343);
  the gate check reads `window.showDirectoryPicker` at call time too (:326). No
  module-scope capture exists (grep confirms only those two call-time references
  plus comments).

## Steps completed

- [x] §2.1 Import swap (app.js:2): `IndexedDbVault` → `import { DirectoryVault }`
  + `import { MemoryVault }`. Verified: `node --check app.js` OK; no
  `IndexedDbVault` references remain in app.js.
- [x] §2.2 Starter rewire (F1): added `minimalFreshWorkspace()` (version 1, single
  root canvas, empty document, title "Balaur", default camera; no localStorage,
  no demo canvas) at app.js:199. `createGraphStarterWorkspace` now calls
  `minimalFreshWorkspace()` (the following line already assigns
  `root.document=rootDocument`) and `return result` instead of
  `return normalizeWorkspace(result)`. Verified: F1 safe (above); `node --check`.
- [x] §2.3 Deletions: `demoCanvas` constant removed; `loadDocument`,
  `freshWorkspace`, `normalizeWorkspace`, and localStorage-reading `loadWorkspace`
  removed; `WORKSPACE_KEY` removed (line kept as `const ROOT_CANVAS_ID="canvas-root";`);
  stale boot comment reworded to name the user-picked folder. Verified: grep for
  all deleted symbols returns nothing; `balaur-canvas-v1`/`balaur-title` gone.
- [x] §2.4 Boot rewrite: added `let currentVault=null;` (app.js:171).
  `bootCanvasApp()` is now (1) incompatibility gate on missing
  `showDirectoryPicker`/`crypto.subtle` (fills `#vaultLandingMessage`, disables
  `#openVaultFolder`, `setIndexStatus("Files unavailable", …)`, returns — no
  fallback) and (2) a pick-await promise referencing the global picker at call
  time, swallowing `AbortError`, routing other picker errors to
  `#vaultLandingMessage`, and on a handle building `new DirectoryVault(handle)`,
  detecting `hadSidecar`/`empty`, calling `openVault(vault,{seed:empty})`,
  setting `currentVault`, hiding the gate, revealing the shell, resolving.
  `openVault(vault,{seed})` is the shared core (steps 1–8): WorkspaceStore;
  `hasWorkspace`; `!had → store.migrate(seed?createGraphStarterWorkspace():minimalFreshWorkspace())`
  (additive Adopt); `store.load()` with the empty-workspace guard; assign
  `workspace`/`vaultStore`/`window.balaurVaultStore` + `setCanonicalWritable`
  diagnostics + `configureLifeRuntime` + `seedBundledWidget`; `seed →
  seedGraphStarterEntities()` + reload; rebuild the catalogs + `setIndexStatus`;
  the post-boot block. Throws on failure; callers route the error. Verified:
  `node --check app.js` OK.
  - **Note (live drift preserved)**: the catalog rebuild and boot-failure nulling
    in the live code include `noteCatalog`/`noteRepository` (added by ticket 04).
    The plan §2.4 lists three catalogs because it predates ticket 04; the ticket
    says "the existing block", so the live four-catalog rebuild
    (`lifeIndexer`, `componentCardCatalog`, `widgetCatalog`, `noteCatalog`) is
    preserved, not regressed to three.
- [x] Module tail keeps its exact shape (app.js:1965-1968):
  `vaultReady=bootCanvasApp(); window.balaurVaultReady=vaultReady; await vaultReady;`
  then the existing `window.balaurCanvas` assignment. In incompatibility-gated mode
  `bootCanvasApp` returns immediately so `balaurCanvas` is exposed over the
  placeholder workspace; in the normal landing flow it resolves after a pick.
- [x] §2.5 Wiring (app.js:1934-1946): `#reloadVault` →
  `openVault(currentVault,{seed:false})` guarded by `if(!currentVault)return;`,
  with reload-failure affordances (`setIndexStatus("Files unavailable", …)` +
  `setCanonicalWritable(false, …)`; runtime globals NOT nulled). `#openAnotherVault`
  → `flushPendingWorkspaceEdits()`, `inert`+hide shell, unhide `#vaultLanding`.
- [x] §2.6 Staging re-point: `importCanvas` version-2 staging → `new MemoryVault()`
  and `canonicalVault = vaultStore.vault`; stale "IndexedDB restore is one
  transaction" comment reworded (staging runs in memory; restore replaces the
  picked folder's file tree). `loadGraphStarter` staging → `new MemoryVault()`;
  its `canonicalVault` was already `vaultStore.vault` (verified, unchanged).
  `exportWorkspace()` unchanged. Verified: `node --check`.
- [x] §2.7 Comment rewords: `storage/vault-store.js` adapter comment now names
  DirectoryVault (browser, user-picked folder) / FsVault (Node) / MemoryVault
  (tests); `storage/workspace-vault.js` comment now names DirectoryVault (browser).
  Logic untouched. Verified: `node --check` both modules.
- [x] §2.8 `storage/indexeddb-vault.js` deleted via `git rm`. Verified: file gone;
  dead-reference sweep clean.
- [x] §3.1 `index.html`: `#vaultLanding` overlay (wordmark, hint, `#openVaultFolder`
  primary button, `#vaultLandingMessage role="status"`) added as the FIRST child of
  `<body>` before `.app-shell`; `.app-shell` carries `hidden inert` in markup;
  `#reloadVault` and `#openAnotherVault` nav-item buttons added to `.sidebar-bottom`
  after Reset starter; `#importButton`/`#fileInput`/`#exportWorkspaceButton`/`#resetDemo`
  kept.
- [x] §3.2 `styles/shell.css` (inside `@layer shell`, Balaur tokens): the critical
  `.app-shell[hidden]{display:none}` and `#vaultLanding[hidden]{display:none}`
  overrides (the grid/flex displays override the UA `[hidden]`); `.vault-landing`
  fixed full-viewport centered column above the shell (`z-index:100`,
  `--balaur-surface-page` background); wordmark/hint/message use existing type and
  status tokens; optional fade guarded by `@media (prefers-reduced-motion: no-preference)`.
  Verified: braces balanced (107/107), single `@layer shell`.
- [x] §3.3 `sw.js`: `CACHE_NAME = "balaur-shell-v13"`; `APP_SHELL` drops
  `indexeddb-vault.js`, adds `directory-vault.js` + `memory-vault.js`, keeps
  `workspace-backup.js`. Verified: `node --check sw.js` OK; every module in
  `grep -o 'from "\./[^"]*"' app.js | sort -u` is present in `APP_SHELL` (none missing).

## Files changed

- `app.js` — import swap; `minimalFreshWorkspace`; `createGraphStarterWorkspace`
  rewire; deletions (demoCanvas, loadDocument/freshWorkspace/normalizeWorkspace/
  loadWorkspace, WORKSPACE_KEY); `bootCanvasApp` gate+pick rewrite; new `openVault`
  core; `currentVault`; Reload/Open-another wiring; import + Reset staging re-pointed
  to MemoryVault; comment rewords.
- `index.html` — `#vaultLanding` overlay; `hidden inert` on `.app-shell`; two sidebar buttons.
- `styles/shell.css` — `[hidden]` overrides; `.vault-landing` rules; reduced-motion fade.
- `sw.js` — cache v13; precache swap + the three app.js-imported modules that were
  missing from the precache list (see Issues).
- `storage/indexeddb-vault.js` — deleted.
- `storage/vault-store.js` — adapter comment reword (comment only).
- `storage/workspace-vault.js` — adapter comment reword (comment only).

## Verification results

- `node --check app.js storage/directory-vault.js storage/memory-vault.js storage/vault-store.js storage/workspace-vault.js sw.js` → all exit 0.
- Full explicit suite (`node --test` phase1–10 + phase-query + note-repository) →
  `tests 197`, `pass 197`, `fail 0` (count unchanged; this ticket adds/removes no
  Node tests — DirectoryVault is browser-only, verified by ticket 03's driver).
- `git diff --check HEAD` → clean.
- Dead-reference sweep `grep -rn "indexeddb-vault\|IndexedDbVault" --include="*.js" --include="*.mjs" . | grep -v "^./vendor/\|^./teach/\|^./plans/"` → no output. Remaining references are only in historical ADR text (`docs/adr/`), `teach/` lessons, `plans/`, and `para/` project artifacts.
- `grep -n "localStorage" app.js` → only theme (`balaur-canvas-theme`, lines 1667/1890) and AI-settings (`AI_SETTINGS_KEY`/`AI_SECRET_KEY`, lines 1788/1789/1798/1930).
- `grep -o 'from "\./[^"]*"' app.js | sort -u` → all 20 modules present in `APP_SHELL`; `CACHE_NAME` is `balaur-shell-v13`.
- `showDirectoryPicker` referenced only as the global at call time (app.js:326 gate, :343 pick); no module-scope capture.
- `git diff --name-status ab4432d..HEAD` → exactly the seven in-scope files; nothing out of scope.

Browser behavior (gate UX, picker stub, reload persistence, offline) is ticket 03's
driver + manual smoke per the plan; this ticket's verification is static + the Node suite.

## Issues encountered

- **Pre-existing precache gap caught by the §3.3 cross-check**: three modules
  imported by app.js were absent from `APP_SHELL` — `journal-event-repository.js`
  (pre-existing) and `note-catalog.js` + `note-repository.js` (added by ticket 04).
  The §3.3 acceptance criterion ("every module app.js imports is present in
  APP_SHELL") and the v13 re-precache require them for offline boot, so they were
  added to the storage group in `sw.js` (in-scope file). Without this, ticket 03's
  `smoke --offline` would fail to load those modules from cache.
- **Minor documented deviation (§2.7)**: the plan scoped the `workspace-vault.js:9`
  reword to the IndexedDbVault→DirectoryVault naming. The same comment line also
  claimed "localStorage is consulted only once as a legacy-profile migration
  source" — a statement this ticket falsifies by deleting that migration (the
  module itself never consulted localStorage; the migration lived in app.js's
  deleted `loadWorkspace`). The clause was corrected to "the browser boots from a
  user-picked folder with no localStorage migration" to avoid shipping a known-false
  comment. Comment-only; logic untouched.
- **Live four-catalog rebuild preserved** (see §2.4 note): the plan's three-catalog
  list predates ticket 04; the live `noteCatalog.rebuild()` was kept rather than
  regressed, consistent with the ticket's "the existing block" language.
- The ticket file's frontmatter (`status: in-progress`, `worker`, `branch`) was set
  by the orchestrator before spawn; it is left unstaged for the orchestrator to
  finalize and is not part of this worker's commits.
