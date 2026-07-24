---
phase: plan
status: done
project: 001-directory-vault
date: 2026-07-24
---

# Plan 001: DirectoryVault becomes the sole browser vault

> **Executor instructions**: Follow this plan stage by stage. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report; do not improvise. This plan is the architecture and sequencing
> source; the tickets phase splits it into per-ticket work. Every ticket
> worker reads this file in full before editing.
>
> **Pre-edit identity check (mandatory before any file edit)**:
> 1. `pwd` matches the assigned worktree path.
> 2. `git branch --show-current` is `001-directory-vault` (non-main).
> 3. `git worktree list` confirms this worktree's identity.
> 4. `git status --short` shows only the expected pre-staged files (see
>    "Pre-staged files" below) and nothing else unexpected.
> Stop and report if any check fails.
>
> **Drift check (run first)**:
> `git diff --stat e3103b2..HEAD -- app.js index.html styles/shell.css sw.js storage/ .pi/skills/browser-check/ docs/ README.md AGENTS.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: L
- **Risk**: MED
- **Depends on**: none (spec `para/projects/001-directory-vault/spec.md` and ADR `docs/adr/0004-directory-vault-storage.md` are settled inputs)
- **Category**: migration
- **Planned at**: commit `e3103b2`, 2026-07-24

## Why this matters

The browser vault is `IndexedDbVault`: opaque origin-private storage the user
cannot see, sync, or edit with other tools. This change makes the vault a real
folder the user picks at launch, via the File System Access API, so its plain
files sync with Syncthing/Dropbox/git and open in any editor. A new
`DirectoryVault` adapter implements the existing `VaultStore` contract over a
`FileSystemDirectoryHandle`; boot becomes an incompatibility gate, a landing
screen (the "Vault gate"), a folder pick, and workspace load. `IndexedDbVault`
and the one-time `localStorage` first-run migration are deleted. Browsers
without `showDirectoryPicker` or `crypto.subtle` get a full-screen gate
message with no fallback. Existing data migrates by one-time whole-space
`.orbit.json` export from the deployed IndexedDB version, then version-2
import into an empty picked folder.

## Current state

Facts the executor needs, inlined. All line numbers are against `e3103b2`.

### Files and their roles

- `storage/vault-store.js` (38 lines) — the `VaultStore` base class and
  `mediaTypeFor`. Abstract async surface: `list/read/write/remove/move/stat/
  exists/snapshot/restore`, plus `get revision()`. Meta shape contract:
  `{ path, mediaType, size, hash, modifiedAt, revision }`.
- `storage/memory-vault.js` (187 lines) — the deterministic reference for the
  precondition table, journal, snapshot/restore. DirectoryVault mirrors its
  external behavior exactly.
- `storage/fs-vault.js` (161 lines) — the disk-backed reference: real mtime,
  case-fold-by-listing (`_checkFoldCollision` walks `list("")`), non-atomic
  semantics guarded by `expectedHash`.
- `storage/indexeddb-vault.js` (252 lines) — deleted by this plan. Browser-only
  precedent: verified by `node --check` and browser tests, never the Node runner.
- `storage/vault-errors.js` — `VaultError`, `PathError`, `ParseError`,
  `SchemaError`, `ConflictError`, `StorageError`; every error carries `code`
  and optional `details`.
- `storage/vault-path.js` — exports `byteLength`, `assertSafePath`,
  `caseFoldKey` (plus others the adapter does not need).
- `storage/workspace-vault.js` (260 lines) — `WorkspaceStore`, `hasWorkspace`,
  `canvasPathFor`, `parseSidecar`. `hasWorkspace(vault)` is
  `vault.exists(".orbit/workspace.json")`. `migrate(workspace)` writes every
  file with `expectedHash: null` (additive-only create; this is the Adopt
  mechanism). `load()` returns `{ workspace, diagnostics }` with read-only
  repair placeholders for missing/invalid canvases. JD stripping already lives
  in `parseSidecar`.
- `storage/life-indexer.js:284` — `reconcileWarm` calls
  `this.vault.changesSince(fromRevision)`. DirectoryVault must implement
  `changesSince` or warm reconciliation breaks on every save.
- `app.js` (1831 lines) — touched in ~7 regions (below). AGENTS.md §12 forbids
  mass-formatting: targeted edits only, match the surrounding minified-ish style.
- `index.html` — `<body>` starts at line 24 with `<div class="app-shell">`;
  `.sidebar-bottom` at lines 62–68 holds Export whole space / Reset starter /
  `#lifeIndexStatus`.
- `styles/shell.css` — `@layer shell` (the layer order is declared in
  `styles/layers.css`: `tokens, foundation, shell, canvas, components, themes,
  responsive, motion`). `.app-shell { display: grid; ... }` is the first rule.
- `sw.js` (114 lines) — `CACHE_NAME = "orbit-shell-v12"` at line 1;
  `"./storage/indexeddb-vault.js"` at line 18 of `APP_SHELL`.
- `.pi/skills/browser-check/scripts/browser-check.mjs` (1735 lines) — CDP
  driver. Subcommands: `smoke`, `components`, `widgets`, `eval`, `shot`.
  `DEFAULT_URL = "http://localhost:4173/"`. Every subcommand except `eval`
  waits on `window.orbitCanvas` after navigate (lines 254, 386 reload, 430,
  628, 1206, 1343, 1723).

### app.js regions touched

```js
// app.js:2 — import to replace
import { IndexedDbVault } from "./storage/indexeddb-vault.js";

// app.js:24-47 — demoCanvas constant (only consumer is loadDocument)
const demoCanvas = { nodes: [ ... ], edges: [ ... ] };

// app.js:49 — createGraphStarterWorkspace (STAYS; see finding F1 below)
function createGraphStarterWorkspace(){
  const rootDocument={nodes:[ ... ],edges:[]};
  const result=freshWorkspace(rootDocument),root=result.canvases[result.rootId];
  root.title="Home";root.document=rootDocument;root.camera=null;
  // ... hub(...) calls, all with explicit paths ...
  return normalizeWorkspace(result);
}

// app.js:125 — WORKSPACE_KEY dies, ROOT_CANVAS_ID stays
const WORKSPACE_KEY="orbit-workspace-v1",ROOT_CANVAS_ID="canvas-root";

// app.js:130 region — module-level placeholder workspace + `let vaultStore=null;`
// at ~160. STAYS exactly as-is (event wiring attaches before boot completes).

// app.js:180-211 — the four deletions
function loadDocument() { /* reads localStorage "orbit-canvas-v1", falls back to clone(demoCanvas) */ }
function freshWorkspace(document=loadDocument()){ /* reads localStorage "orbit-title" */ }
function normalizeWorkspace(parsed){ /* JD stripping; already duplicated in parseSidecar */ }
function loadWorkspace(){ /* reads localStorage WORKSPACE_KEY */ }

// app.js:329-365 — bootCanvasApp (rewritten; see Architecture §2)
async function bootCanvasApp(){
  try {
    const vault = new IndexedDbVault("orbit-vault");
    const store = new WorkspaceStore(vault);
    const hadWorkspace = await hasWorkspace(vault);
    const firstRun = !hadWorkspace && !localStorage.getItem(WORKSPACE_KEY) && !localStorage.getItem("orbit-canvas-v1");
    if (!hadWorkspace) await store.migrate(loadWorkspace());
    // ... load, diagnostics, configureLifeRuntime, seedBundledWidget,
    //     firstRun -> seedGraphStarterEntities, rebuilds, setIndexStatus ...
  } catch (error) { /* null all runtime globals; setIndexStatus("Files unavailable", ...) */ }
  // post-boot: currentCanvasId/documentData/camera/title, renderWorkspaceNavigation(); render(); setTimeout(fitView, 50);
}

// app.js:403 — loadGraphStarter staging (re-point)
const stagingVault=new IndexedDbVault(`orbit-vault-${uid("reset")}`), stagingStore=new WorkspaceStore(stagingVault);
// app.js:408-409
const snapshot=await stagingVault.snapshot(), canonicalVault=vaultStore.vault;

// app.js:1380 — importCanvas version-2 branch staging (re-point)
const stagingVault=new IndexedDbVault(`orbit-vault-${uid("import")}`);
// app.js:1393
const canonicalVault=new IndexedDbVault("orbit-vault");

// app.js:1815 region — button wiring; add the two new handlers near:
$("#resetDemo").onclick=loadGraphStarter;

// app.js:1825-1831 — module tail (shape STAYS)
vaultReady=bootCanvasApp();
window.orbitVaultReady=vaultReady;
await vaultReady;
window.orbitCanvas={ ... };
```

localStorage uses that STAY (not vault data): theme at app.js:1544/1766
(`orbit-canvas-theme`), AI settings at app.js:1665-1675/1806
(`AI_SETTINGS_KEY`/`AI_SECRET_KEY`). Uses that vanish with the deletions:
app.js:182, 191, 205, 210, 334.

### Conventions to match

- Adapter style: follow `storage/memory-vault.js` structure (constructor,
  `get revision()`, private `_bump`, `_checkPrecondition`, meta assembly) and
  `storage/fs-vault.js` for disk semantics. Named helpers, explicit errors at
  boundaries, no globals.
- app.js style: compact one-line function bodies are normal there; do not
  reformat untouched code (AGENTS.md §12). New app.js functions may use the
  same compact style.
- CSS: rules go inside `@layer shell` in `styles/shell.css`, using Balaur
  tokens (`var(--balaur-surface-page)`, `var(--balaur-color-outline)`, etc.);
  respect `prefers-reduced-motion` (AGENTS.md §11).
- Errors: always a typed vault error with a stable `code`; include
  `details: { name, message }` when wrapping a DOMException.

### Vocabulary (CONTEXT.md, settled)

Use these terms in code comments, docs, and commit messages: **Vault**,
**DirectoryVault** (avoid "folder vault"), **Vault gate** (avoid "landing
page"/"splash screen"; the HTML id stays `#vaultLanding`), **Adopt** (avoid
"import"/"migrate"/"seed" for the files-but-no-sidecar open). `CONTEXT.md` in
this worktree already contains the updated entries (see "Pre-staged files").

### ADR constraint

`docs/adr/0004-directory-vault-storage.md` (accepted) is the decision record:
DirectoryVault is the sole browser adapter over
`showDirectoryPicker({ mode: "readwrite" })`; hard gate on missing
`showDirectoryPicker` or `crypto.subtle`; re-pick per launch, zero persisted
handles; one-time manual `.orbit.json` migration; non-atomic `createWritable`
guarded by `expectedHash`; Adopt is additive-only; file-canonical ownership
(ADR-0001) unchanged. Every implementation decision below stays consistent
with it.

### Pre-staged files (already in this worktree, do not rewrite)

`git status` at plan time shows:
- `M CONTEXT.md` — updated glossary (Vault, DirectoryVault, Vault gate, Adopt).
- `?? docs/adr/0004-directory-vault-storage.md` — accepted ADR, final content.
- `?? para/` — grill, spec, this plan.

The docs stage verifies their content matches the spec and commits them as-is.

## Commands you will need

| Purpose | Command | Expected on success |
|---|---|---|
| Syntax check a touched module | `node --check app.js` (and each touched `storage/*.js`) | exit 0 |
| Whitespace hygiene | `git diff --check` | no output |
| Node suite (must stay green at every stage) | `node --test storage/phase1.test.js storage/phase2.test.js storage/phase3.test.js storage/phase4.test.js storage/phase4-backup.test.js storage/phase5.test.js storage/phase7.test.js storage/phase8.test.js storage/phase9.test.js storage/phase10.test.js storage/phase-query.test.js` | `tests 172`, `pass 172`, `fail 0` |
| Serve the app (separate terminal, repo root) | `python3 -m http.server 4173` | serves on localhost:4173 |
| Adapter contract suite (Stage 4) | `node .pi/skills/browser-check/scripts/browser-check.mjs contract` | `All N checks passed.`, exit 0 |
| Full smoke + offline (Stage 4) | `node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline` | `All N checks passed.`, exit 0 |
| Dead-reference sweep | `grep -rn "indexeddb-vault\|IndexedDbVault" --include="*.js" --include="*.mjs" . \| grep -v "^./vendor/\|^./teach/\|^./plans/"` | no output |
| localStorage sweep | `grep -n "localStorage" app.js` | only theme (1544/1766 region) and AI-settings (1665-1675/1806 region) lines remain |

## Suggested executor toolkit

- Read `.pi/skills/browser-check/SKILL.md` fully before Stage 4, including the
  headless event-retargeting caveat. The driver is the default browser
  verification path (AGENTS.md §13).
- `browser-check.mjs eval "<expr>"` runs one-off runtime probes; useful while
  developing the adapter before the `contract` subcommand exists:
  `node .pi/skills/browser-check/scripts/browser-check.mjs eval "(async () => { const { DirectoryVault } = await import('/storage/directory-vault.js'); ... })()"`.
- Manual checks that CDP cannot express (real picker UX, external editor
  edit + Reload vault, folder deletion mid-session, Firefox/Safari gate,
  migration rehearsal) are listed in the Testing strategy; they are
  browser-pending per AGENTS.md §13 and must be labeled, not claimed.

## Scope

Staged file sets (the tickets phase turns these into tickets; see Sequencing):

**Stage 1 (adapter)** — in scope:
- `storage/directory-vault.js` (create)

**Stage 2 (boot swap)** — in scope:
- `app.js` (targeted edits in the ~7 regions above)
- `index.html` (landing overlay, `hidden inert` on `.app-shell`, two sidebar buttons)
- `styles/shell.css` (`.vault-landing` rules, `[hidden]` fixes)
- `sw.js` (cache bump + precache list swap)

**Stage 3 (driver)** — in scope:
- `.pi/skills/browser-check/scripts/browser-check.mjs` (`contract` subcommand; landing-aware boot helper used by `smoke`, `components`, `widgets`, `shot`; smoke prelude/reload rework)
- `.pi/skills/browser-check/SKILL.md` (document `contract`, the picker stub, and the re-pick-after-reload behavior)

**Stage 4 (docs)** — in scope:
- `docs/adr/0004-directory-vault-storage.md` (commit pre-staged, verify only)
- `CONTEXT.md` (commit pre-staged, verify only)
- `docs/architecture.md`, `docs/life-data.md`, `docs/offline.md`, `README.md`, `AGENTS.md`

**Stage 5 (deploy gate)** — no file changes; a manual rehearsal (see Testing).

**Out of scope (do NOT touch, even though they look related)**:
- `storage/memory-vault.js`, `storage/fs-vault.js`, `storage/vault-store.js`
  logic, `storage/workspace-vault.js` logic, `storage/workspace-backup.js`,
  the Node test files, `storage/life-indexer.js`: the contract is settled and
  Node-verified; only two stale comments are reworded (see Stage 2 §2.6).
- `teach/*.html`: reference `indexeddb-vault.js` as historical lesson content,
  not app modules; left as-is.
- `plans/*.md`: historical artifacts; left as-is.
- `main.js`, `offline/register.js`: unchanged (boot shape survives).
- Any persisted directory handle, `queryPermission`/`requestPermission` calls,
  fallback adapter, filesystem watcher, atomic-rename emulation: ruled out by
  the spec and ADR-0004.
- The `window.orbitCanvas` surface shape: unchanged.

## Git workflow

- The orchestrator created the feature worktree `001-directory-vault` branched
  off `main`; every stage commits to this one branch. Topology is
  `main → 001-directory-vault`, one layer.
- Parallel tickets (only Stage 1 and Stage 4 qualify as file-disjoint; see
  Sequencing) stage only their own paths: `git add <paths>`, never
  `git add -A` or `git commit -a`.
- Commit per stage (or per logical unit inside a stage). Message style matches
  `git log --oneline -10` (imperative, short). Example:
  `storage: add DirectoryVault adapter over the File System Access API`.
- Workers commit and report only; the orchestrator owns push, PR, and the
  Stage 5 deploy gate. Never push or merge from a worker.

## Architecture

### 1. `storage/directory-vault.js` (Stage 1)

New module, browser-only (File System Access has no Node implementation),
verified by `node --check` and the browser `contract` suite, exactly as
`IndexedDbVault` was. No Node tests are added; the 172-test suite is unchanged.

```js
import { VaultStore, mediaTypeFor } from "./vault-store.js";
import { contentHash } from "./content-hash.js";
import { byteLength, assertSafePath, caseFoldKey } from "./vault-path.js";
import { ConflictError, PathError, StorageError, VaultError } from "./vault-errors.js";

export class DirectoryVault extends VaultStore {
  constructor(handle) { /* see below */ }
  // full surface below
}
```

**Constructor**: `constructor(handle)` stores the `FileSystemDirectoryHandle`
in `this._handle`, initializes `this._revision = 0` and `this._journal = []`.
Throws `StorageError` (`STORAGE_UNAVAILABLE`) if the handle is missing or
`handle.kind !== "directory"`. No permission management: the picker grants
readwrite for the session; OPFS handles (the test double) are already granted.
No `queryPermission`/`requestPermission` anywhere in the module.

**No content cache**: every `read`/`stat`/`list` goes to disk so external
edits are visible (ADR-0004). This is the whole point of the adapter.

**Private helpers** (keep small, one job each):

- `_bump(path, operation, hash, oldPath)` — identical to MemoryVault's:
  increments `_revision`, pushes `{ revision, path, operation, hash }` (plus
  `oldPath` when given) to `_journal`, returns the revision.
- `async _dirFor(path, { create })` — splits `assertSafePath(path)` into
  segments; walks from `this._handle` with
  `getDirectoryHandle(segment, { create })` for every segment except the last;
  returns `{ dir, name }`. DOMExceptions are mapped (table below).
- `async _fileHandle(path, { create })` — `_dirFor` then
  `dir.getFileHandle(name, { create })`.
- `async _stat(path)` — internal meta builder shared by `stat`/`write`/`move`:
  `getFileHandle(name)` (no create); `NotFoundError` → `null`;
  `handle.getFile()` → text via `file.text()`; returns
  `{ path, mediaType: mediaTypeFor(path), size: byteLength(text), hash:
  await contentHash(text), modifiedAt: new Date(file.lastModified).toISOString(),
  revision: this._revision }`. `modifiedAt` is real disk mtime (FsVault
  semantics); `revision` is the session counter (FsVault `_record` precedent).
- `_checkPrecondition(path, existing, expectedHash)` — the exact MemoryVault
  table (memory-vault.js:52-62): `undefined` → no check; `null` → existing
  must be falsy (`ConflictError` `WRITE_CONFLICT` "Expected \"p\" to not
  exist"); string → existing must exist with matching hash (`ConflictError`
  with `details: { expected, actual }` on mismatch).
- `async _checkFoldCollision(path)` — FsVault style (fs-vault.js:69-71):
  `const fold = caseFoldKey(path); for (const meta of await this.list(""))
  if (caseFoldKey(meta.path) === fold && meta.path !== path) throw new
  PathError(..., { code: "PATH_CASE_COLLISION" });`
- `_wrap(error, message)` — DOMException → vault error mapping (table below);
  rethrows errors that are already `VaultError` instances unchanged.

**DOMException → vault-error mapping** (wrap at every adapter boundary; always
include `details: { name: error.name, message: error.message }`):

| DOMException name | Mapping |
|---|---|
| `NotFoundError` on `read` | `VaultError` `NOT_FOUND` (the code `WorkspaceStore.loadWorkspace` keys on for repair placeholders) |
| `NotFoundError` on `stat`/`exists`/internal `_stat` | `null` / `false` (not an error) |
| `NotAllowedError`, `SecurityError` (permission denied or revoked mid-session) | `StorageError` `STORAGE_UNAVAILABLE` |
| `TypeMismatchError` (a directory occupies a file path or vice versa) | `PathError` `PATH_COMPONENT` |
| `InvalidStateError` and any other unexpected DOMException | `StorageError` `STORAGE_UNAVAILABLE` |
| `AbortError` | only occurs at the picker (landing layer, app.js); never inside this adapter |

**Public surface** (every path argument passes `assertSafePath` first):

- `get revision()` — returns `this._revision`.
- `async exists(path)` — `_stat` based; `NotFoundError` → `false`.
- `async stat(path)` — `_stat` result or `null`.
- `async read(path)` — `_stat`-style read; missing file throws `VaultError`
  `NOT_FOUND`; otherwise returns the text.
- `async list(prefix = "")` — recursive walk from `this._handle` via
  `for await (const entry of dirHandle.values())`; recurse
  `kind === "directory"` (paths joined with `/`), collect `_stat` meta for
  `kind === "file"`; skip nothing (`.orbit`, `canvases`, entities, widgets,
  and foreign files are all listed); filter `meta.path.startsWith(prefix)`;
  sort by plain code-unit comparison:
  `out.sort((a, b) => (a.path < b.path ? -1 : a.path > b.path ? 1 : 0))`
  (matches MemoryVault/IndexedDbVault, not FsVault's localeCompare).
- `async write(path, content, { expectedHash, mediaType } = {})` — in order:
  1. `const existing = await this._stat(p)`;
  2. `this._checkPrecondition(p, existing, expectedHash)`;
  3. if `!existing`, `await this._checkFoldCollision(p)`;
  4. create parents + file: `_fileHandle(p, { create: true })`;
  5. `const writable = await handle.createWritable(); await writable.write(text); await writable.close();`
  6. `const hash = await contentHash(text); const revision = this._bump(p, existing ? "modify" : "create", hash);`
  7. return `{ path: p, mediaType: mediaType || mediaTypeFor(p), size:
     byteLength(text), hash, modifiedAt: new Date().toISOString(), revision }`.
  `createWritable` is not atomic-rename; the `expectedHash` precondition is
  the conflict guard (ADR-0004). A torn write surfaces through the existing
  canvas repair-placeholder and entity diagnostic paths.
- `async remove(path, { expectedHash } = {})` — `_stat` first; missing →
  `VaultError` `NOT_FOUND`; `_checkPrecondition`; `fileHandle.remove()`;
  `_bump(p, "remove", existing.hash)`; return `true`.
- `async move(from, to, { expectedHash } = {})` — mirrors MemoryVault.move
  ordering so a failed destination write never deletes the source:
  1. read `from` text (`_stat` + `read` semantics; missing → `NOT_FOUND`);
  2. destination `_stat(to)` truthy → `ConflictError` "Destination exists";
  3. `this._checkPrecondition(f, existing, expectedHash)`;
  4. `await this._checkFoldCollision(t)`;
  5. write the destination INLINE (parents + `getFileHandle(create)` +
     `createWritable` + write + close). Do NOT delegate to `this.write`:
     delegating would journal a spurious "create" and bump the revision twice.
     One `_bump(t, "move", hashOfReadText, f)` only;
  6. `remove` the source file handle;
  7. return meta for `to` with the preserved content hash and
     `modifiedAt: new Date().toISOString()`.
- `async snapshot()` — `const metas = await this.list("")`, read each;
  returns `{ format: "orbit-vault-snapshot", revision: this._revision, files:
  [{ path, mediaType, text }] }` sorted by path. Identical shape to
  MemoryVault/FsVault snapshots.
- `async restore(snapshot)` — documented semantics: replaces the entire file
  tree of the picked folder (empty directories may remain; harmless, YAGNI).
  Only reachable through explicitly confirmed destructive flows (whole-space
  import, Reset starter), which validate into a staging vault first. Steps:
  1. validate every `file.path` with `assertSafePath` and reject case-fold
     collisions within the snapshot up front (a `Map` of `caseFoldKey → path`;
     duplicate → `PathError` `PATH_CASE_COLLISION`), same as FsVault `_restore`;
  2. remove every file currently returned by `this.list("")` (resolve each
     handle, `remove()`; removal failures propagate as mapped vault errors);
  3. reset `this._journal = []` and `this._revision = 0`;
  4. write each snapshot file by delegating to
     `this.write(file.path, file.text, { expectedHash: null, mediaType: file.mediaType })`
     (the tree is empty, so the per-write fold check is redundant but
     harmless; the O(n²) listing cost is acceptable for v1 vault sizes and
     restore is rare);
  5. return `{ revision: this._revision, count: snapshot.files.length }`.
- `changesSince(revision)` — synchronous
  `return this._journal.filter((e) => e.revision > revision);` identical to
  MemoryVault/FsVault. Required by `LifeIndexer.reconcileWarm`
  (storage/life-indexer.js:284).

**Verify (Stage 1)**:
- `node --check storage/directory-vault.js` → exit 0.
- 172-test suite → unchanged, all pass (no test imports this module).
- `git diff --check` → clean.
- Behavioral verification arrives with the Stage 3 `contract` subcommand;
  until then, an ad-hoc `eval` probe (toolkit section) may exercise the
  surface against an OPFS handle.

### 2. app.js boot rewire (Stage 2; targeted edits only)

AGENTS.md §12: no mass-formatting. Each edit below is a targeted replacement.

#### 2.1 Import swap (app.js:2)

Replace:
```js
import { IndexedDbVault } from "./storage/indexeddb-vault.js";
```
with:
```js
import { DirectoryVault } from "./storage/directory-vault.js";
import { MemoryVault } from "./storage/memory-vault.js";
```
`MemoryVault` is the browser staging adapter for import and Reset starter.
`hasWorkspace` is already imported at app.js:3.

#### 2.2 Starter builder rewire (finding F1 — spec gap resolved here)

The spec's deletion list removes `freshWorkspace()` and `normalizeWorkspace()`,
but `createGraphStarterWorkspace()` (app.js:49, which STAYS as the empty-folder
create path) calls both: `const result=freshWorkspace(rootDocument)` at its top
and `return normalizeWorkspace(result)` at its end. Dropping the call sites is
mandatory and safe:

- `normalizeWorkspace` on a freshly built starter is a no-op: the starter sets
  every canvas `path` explicitly, uses only valid `kind` values, carries no JD
  fields, and the root already has `path: null`. JD stripping for loaded
  sidecars lives in `parseSidecar` (workspace-vault.js), so nothing is lost.
- The `freshWorkspace(rootDocument)` call becomes the new
  `minimalFreshWorkspace()` plus a document assignment.

Add (placing it where the deleted `freshWorkspace`/`loadWorkspace` block sat,
app.js:180-211 region):
```js
function minimalFreshWorkspace(){
  return {version:1,rootId:ROOT_CANVAS_ID,activeId:ROOT_CANVAS_ID,canvases:{[ROOT_CANVAS_ID]:{id:ROOT_CANVAS_ID,title:"Balaur",parentId:null,portalNodeId:null,path:null,document:{nodes:[],edges:[]},camera:{x:80,y:55,zoom:.78}}}};
}
```
This is the Adopt path's workspace: version 1, single root canvas, empty
document, default title "Balaur", default camera. No localStorage, no demo
canvas.

In `createGraphStarterWorkspace()` change exactly two things:
```js
// before
const result=freshWorkspace(rootDocument),root=result.canvases[result.rootId];
// after
const result=minimalFreshWorkspace(),root=result.canvases[result.rootId];
root.document=rootDocument;
```
(the next line `root.title="Home";root.document=rootDocument;root.camera=null;`
already sets the document; fold the assignment into that line and keep
`root.title="Home";root.camera=null;`), and:
```js
// before
return normalizeWorkspace(result);
// after
return result;
```

#### 2.3 Deletions (app.js, targeted)

Delete, in this order (each is dead after §2.2 and §2.4 land):
- the `demoCanvas` constant (app.js:24-47);
- `loadDocument()`, `freshWorkspace()`, `normalizeWorkspace()`, and the
  localStorage-reading `loadWorkspace()` (app.js:180-211; replaced by
  `minimalFreshWorkspace` per §2.2);
- `WORKSPACE_KEY` from app.js:125, keeping the line as
  `const ROOT_CANVAS_ID="canvas-root";`
- the stale boot comment at app.js:327-328 ("The only post-migration source of
  truth is the IndexedDB vault") — reword to name the user-picked folder.

#### 2.4 Boot rewrite: gate, pick, `openVault` (app.js:329-365)

Add `let currentVault=null;` next to `let vaultStore=null;` (~app.js:160).

Replace `bootCanvasApp()` with the gate + pick-await shape. The module tail
(app.js:1825-1831) keeps its exact shape: `vaultReady=bootCanvasApp();
window.orbitVaultReady=vaultReady; await vaultReady;` then the existing
`window.orbitCanvas` assignment. `window.orbitVaultReady` resolves when a vault
has opened, or immediately when the gate fires (with `vaultStore` left null);
`window.orbitCanvas` is exposed in both cases (gated mode runs over the
placeholder workspace behind the full-screen overlay).

```js
async function bootCanvasApp(){
  // 1. Incompatibility gate: no fallback adapter (ADR-0004).
  if(typeof window.showDirectoryPicker!=="function"||!globalThis.crypto?.subtle){
    $("#vaultLandingMessage").textContent="Balaur needs a Chromium-based desktop browser (Chrome, Edge, Brave, or Arc) to open a vault folder.";
    $("#openVaultFolder").disabled=true;
    setIndexStatus("Files unavailable","This browser cannot open vault folders (File System Access API unavailable).");
    return;
  }
  // 2. Wait for a successful folder pick. AbortError (cancelled picker) is
  //    swallowed: stay on the gate, no message. Other picker errors surface in
  //    the gate status region.
  await new Promise(resolve=>{
    $("#openVaultFolder").onclick=async()=>{
      let handle;
      try{
        // BINDING CONSTRAINT (spec seam 2): reference the global at call time.
        // Never capture showDirectoryPicker in a module-scope const; the
        // browser-check driver stubs window.showDirectoryPicker with an OPFS
        // handle to run the full smoke suite headlessly.
        handle=await window.showDirectoryPicker({mode:"readwrite"});
      }catch(error){
        if(error?.name==="AbortError")return;
        $("#vaultLandingMessage").textContent=error.message;
        return;
      }
      try{
        const vault=new DirectoryVault(handle);
        const hadSidecar=await hasWorkspace(vault);
        const empty=!hadSidecar&&(await vault.list("")).length===0;
        await openVault(vault,{seed:empty});
        currentVault=vault;
        $("#vaultLanding").hidden=true;
        shell.removeAttribute("hidden");shell.removeAttribute("inert");
        resolve();
      }catch(error){
        // First-open failure: stay on the gate with the message.
        $("#vaultLandingMessage").textContent=error.message;
      }
    };
  });
}
```

`openVault(vault, { seed })` is the shared core used by first open, Reload
vault, and (via the next pick) Open another vault. It throws on failure;
callers route the error. Body, in order:

1. `const store=new WorkspaceStore(vault);`
2. `const had=await hasWorkspace(vault);`
3. if `!had`: `await store.migrate(seed?createGraphStarterWorkspace():minimalFreshWorkspace());`
   `migrate` writes with `expectedHash: null`, so this is additive-only: it
   creates `.orbit/workspace.json` and `canvases/root.canvas` and never
   touches pre-existing files (the Adopt behavior).
4. `const result=await store.load();` (existing diagnostics and read-only
   repair placeholders are retained); keep the empty-workspace guard:
   `if(!result?.workspace?.canvases||!Object.keys(result.workspace.canvases).length)throw new Error("The vault workspace is empty");`
5. assign `workspace=result.workspace; vaultStore=store;
   window.orbitVaultStore=store;` then the existing
   `setCanonicalWritable(!result.diagnostics.some(...), ...)` diagnostic logic
   (app.js:341) and the `for (const diagnostic of result.diagnostics)
   console.warn(...)` loop; `configureLifeRuntime(vault);`
   `await seedBundledWidget(vault);`
6. if `seed`: `await seedGraphStarterEntities(); workspace=(await store.load()).workspace;`
   (the existing first-run pattern; `seed` replaces the old localStorage-based
   `firstRun` computation);
7. `await Promise.all([lifeIndexer.rebuild(), componentCardCatalog.rebuild(), widgetCatalog.rebuild()]);`
   then the existing `setIndexStatus(canonicalWritable ? ... : ...)` (app.js:351);
8. the existing post-boot block (app.js:358-363): set
   `currentCanvasId`/`documentData`/`camera`/`$("#canvasTitle").value`,
   `renderWorkspaceNavigation(); render(); setTimeout(fitView, 50);`

Error routing:
- First open: the pick handler catch (§2.4 above) puts `error.message` in
  `#vaultLandingMessage` and stays on the gate.
- Reload vault: the handler catch uses the existing failure affordances:
  `setIndexStatus("Files unavailable", error.message); setCanonicalWritable(false,
  "Canonical files are unavailable; export or repair the vault before editing.");`
  Do NOT null the runtime globals on reload failure (export must keep working
  against the previously opened vault).

#### 2.5 Reload vault / Open another vault wiring (near app.js:1815)

Add next to `$("#resetDemo").onclick=loadGraphStarter;`:
```js
$("#reloadVault").onclick=()=>{
  if(!currentVault)return;
  openVault(currentVault,{seed:false}).catch(error=>{
    setIndexStatus("Files unavailable",error.message);
    setCanonicalWritable(false,"Canonical files are unavailable; export or repair the vault before editing.");
  });
};
$("#openAnotherVault").onclick=async()=>{
  await flushPendingWorkspaceEdits();
  shell.setAttribute("inert","");
  shell.hidden=true;
  $("#vaultLanding").hidden=false;
};
```
Reload vault re-reads the same handle (in-session reads need no gesture) and
rebuilds the index: the external-change reconciliation path. Open another
vault returns to the gate; the pick handler wired in `bootCanvasApp` is still
attached, so the next pick runs the first-open flow with the new handle.

#### 2.6 Staging re-point (import migration + Reset starter)

`importCanvas(file)` version-2 branch (app.js:1380 region):
```js
// before
const stagingVault=new IndexedDbVault(`orbit-vault-${uid("import")}`);
// after
const stagingVault=new MemoryVault();
```
and:
```js
// before
const canonicalVault=new IndexedDbVault("orbit-vault");
// after
const canonicalVault=vaultStore.vault;
```
Everything between and after stays: `importBundle(stagingVault, ...)`, the
staging index rebuild, the `auditIndex` pass, `snapshot()`,
`canonicalVault.restore(snapshot)`, `new WorkspaceStore(canonicalVault)`
reload, global switch, rebuilds, render. The existing confirm dialog already
warns the canonical vault is replaced. `#importButton`/`#fileInput` stay: this
is the migration path. Update the stale comment at app.js:1382-1384
("IndexedDB restore is one transaction...") to state that restore replaces the
picked folder's file tree and staging now runs in memory.

`loadGraphStarter()` (app.js:403 region):
```js
// before
const stagingVault=new IndexedDbVault(`orbit-vault-${uid("reset")}`), stagingStore=new WorkspaceStore(stagingVault);
// after
const stagingVault=new MemoryVault(), stagingStore=new WorkspaceStore(stagingVault);
```
`canonicalVault` at app.js:408 is already `vaultStore.vault` — verify, do not
change. The rest of the staging → seed → snapshot → restore → reload
choreography is unchanged.

`exportWorkspace()` (app.js:1365) stays unchanged: it operates on
`vaultStore.vault`; whole-space backups remain a feature (AGENTS.md §4.5).

#### 2.7 Comment rewords in storage modules (two lines)

- `storage/vault-store.js:4` — "IndexedDbVault (browser default), and later
  browser-directory / Tauri" → name `DirectoryVault` (browser, over a
  user-picked folder) and `FsVault`/`MemoryVault`.
- `storage/workspace-vault.js:9` — "runs against MemoryVault (tests) and
  IndexedDbVault (browser)" → "runs against MemoryVault (tests) and
  DirectoryVault (browser)".

No logic changes in either file.

#### 2.8 Delete `storage/indexeddb-vault.js`

`git rm storage/indexeddb-vault.js` (or plain delete + staged removal).

**Verify (Stage 2, code)**:
- `node --check app.js storage/directory-vault.js` → exit 0.
- 172-test suite → all pass.
- Dead-reference sweep (Commands table) → no output.
- localStorage sweep → only theme + AI-settings lines remain.
- `git diff --check` → clean.
- Browser behavior is verified in Stage 3 (the driver rework) and by the
  manual matrix; Stage 2 alone may be smoke-tested ad-hoc with the existing
  `eval` subcommand plus a manual stub.

### 3. Shell: `index.html`, `styles/shell.css`, `sw.js` (Stage 2)

Landed together with §2: the app is unbootable between markup and boot rewire
(the gate overlay would cover a still-IndexedDB app), so these edits are one
atomic stage.

#### 3.1 `index.html`

- Add the Vault gate as the FIRST child of `<body>` (before
  `<div class="app-shell">` at line 25):
```html
<div class="vault-landing" id="vaultLanding">
  <h1 class="vault-landing-wordmark">Balaur</h1>
  <p class="vault-landing-hint">Open a folder as your vault. Its files are the source of truth.</p>
  <button id="openVaultFolder" class="button primary">Open vault folder</button>
  <p id="vaultLandingMessage" role="status"></p>
</div>
```
- Add `hidden inert` to the app shell in markup:
  `<div class="app-shell" hidden inert>` (line 25). The shell is revealed by
  `bootCanvasApp` on a successful pick.
- Add two `nav-item` buttons to `.sidebar-bottom` (index.html:62-68), next to
  Export whole space / Reset starter:
```html
<button class="nav-item" id="reloadVault"><span>⟳</span>Reload vault</button>
<button class="nav-item" id="openAnotherVault"><span>⇄</span>Open another vault</button>
```
- Keep `#importButton`, `#fileInput`, `#exportWorkspaceButton`, `#resetDemo`.

#### 3.2 `styles/shell.css` (inside `@layer shell`, Balaur tokens)

- Critical `[hidden]` fix: `.app-shell { display: grid; }` overrides the UA
  `[hidden] { display: none }`, so the `hidden` attribute alone would NOT hide
  the shell. Add:
```css
.app-shell[hidden] { display: none; }
#vaultLanding[hidden] { display: none; }
```
- `.vault-landing`: `position: fixed; inset: 0;` full-viewport centered column
  (`display: flex; flex-direction: column; align-items: center; justify-content:
  center; gap: ...`), `background: var(--balaur-surface-page)`, a high
  `z-index` above the app shell. Wordmark and hint use existing type tokens;
  `#vaultLandingMessage` is an error/status region (reuse the existing error
  color token). Minimal: no animation beyond an optional fade guarded by
  `@media (prefers-reduced-motion: no-preference)`.

#### 3.3 `sw.js`

- Line 1: `const CACHE_NAME = "orbit-shell-v13";`
- In `APP_SHELL`: remove `"./storage/indexeddb-vault.js"` (line 18); add
  `"./storage/directory-vault.js"` and `"./storage/memory-vault.js"` (both now
  imported by app.js, so required for offline boot). Keep
  `"./storage/workspace-backup.js"`.
- Verify every module app.js imports is listed: after the edit, cross-check
  `grep -o 'from "\./[^"]*"' app.js | sort -u` against `APP_SHELL` (prefix
  `./`). All must be present.

### 4. browser-check driver rework (Stage 3)

Read `.pi/skills/browser-check/SKILL.md` first, including the headless
event-retargeting caveat. No new dependencies; the driver uses Node's built-in
`WebSocket`/`fetch`.

#### 4.1 Shared landing-aware boot helper

Every subcommand that waits on `window.orbitCanvas`/`window.orbitVaultStore`
(smoke:254, smoke reload:386, components:430, widgets:628/1206/1343,
shot:1723) must first get past the Vault gate, because `window.orbitCanvas`
is exposed only after a vault opens (module tail awaits the pick). Add one
helper near the top of the driver and call it at each of those sites:

```js
// Stub the picker with an OPFS handle (spec seam 2) and CDP-click the real
// gate button. Reused at first boot and after every reload: no handle is
// persisted, so the gate shows on every load and the folder is re-picked.
async function bootPastLanding(session, subdirectory = "vault-smoke") {
  await session.waitFor(`(() => { const b = document.getElementById("openVaultFolder"); return b && !b.disabled; })()`, 15000);
  await session.evaluate(`window.showDirectoryPicker = async () => (await navigator.storage.getDirectory()).getDirectoryHandle(${JSON.stringify(subdirectory)}, { create: true })`);
  // Real CDP click (user-gesture semantics), not el.click():
  const point = await session.evaluate(`(() => { const r = document.getElementById("openVaultFolder").getBoundingClientRect(); return { x: r.left + r.width / 2, y: r.top + r.height / 2 }; })()`);
  await session.click(point.x, point.y);
  await session.waitFor("window.orbitCanvas && document.querySelectorAll('.canvas-node').length > 0", 15000);
}
```

Notes for the executor:
- Same profile → same OPFS origin → the same `vault-smoke` subdirectory is
  re-picked after reload, which is what makes the persistence assertion
  meaningful.
- The stub MUST be installed after every `session.reload()` (page JS resets).
- Offline reload works: `openVault` reads only the OPFS-backed vault;
  `seedBundledWidget` fetches only when the widget file is absent (it exists
  after the first open).
- The `eval` subcommand stays raw (no landing handling); it does not wait on
  `window.orbitCanvas` unless `--wait` is given.
- Headless Chrome is Chromium, so the gate passes (`showDirectoryPicker`
  exists, `crypto.subtle` exists on localhost); the real picker is never
  called because the stub replaces it before the click.

#### 4.2 `contract` subcommand (adapter suite via OPFS)

New subcommand `contract`, structured like `smoke` (results array, PASS/FAIL
lines, exit code). It navigates (no `bootPastLanding` needed: the adapter is
imported directly), then runs ONE async `evaluate` expression that:

1. `const { DirectoryVault } = await import("/storage/directory-vault.js");`
2. creates a fresh OPFS subdirectory per run for independence:
   `(await navigator.storage.getDirectory()).getDirectoryHandle("contract-" + Date.now(), { create: true })`;
3. asserts, recording each as a named check (the driver splits the returned
   `{ ok, failures }` into PASS/FAIL records):
   - create/read round-trip; `stat` meta shape (`path`, `mediaType`, `size`,
     `hash`, ISO `modifiedAt`, numeric `revision`); `exists` true/false;
   - `list` ordering (plain code-unit) and prefix filter;
   - the three `expectedHash` outcomes: `null` create-conflict
     (`ConflictError`), hash mismatch with `details.expected`/`details.actual`,
     correct hash succeeds;
   - `NOT_FOUND` code on `read` of a missing path;
   - case-fold collision on create (`PATH_CASE_COLLISION`);
   - `move` success plus destination-exists `ConflictError`;
   - `remove` success plus remove precondition (`expectedHash` mismatch);
   - `snapshot`/`restore` round-trip (restore into a second fresh
     subdirectory; file set and contents equal);
   - `changesSince` journal ordering (revisions strictly increasing;
     `operation` values include create/modify/move/remove);
   - `TypeMismatchError` → `PATH_COMPONENT`: write onto a path whose segment
     is an existing directory name.
4. returns `{ ok, failures }`.

Wire it into the CLI dispatch (`else if (command === "contract")`) and update
the usage comment at the top of the driver and the "Unknown command" line.

#### 4.3 `smoke` rework

- Replace the current prelude (`navigate` + wait on orbitCanvas at line 254)
  with: `navigate` → `bootPastLanding(session)` → a new record
  `"gate: Vault gate passed, picker button enabled"` (asserted inside the
  helper's first wait; record `#vaultLandingMessage` empty and
  `#openVaultFolder` enabled before the stub).
- Assertions 1–8 (console errors, render count, `Files · N indexed` status,
  selection frame, portal probe, note-tool guard, background note creation,
  JSON Canvas validity) run unchanged after the prelude.
- Step 9 (reload persistence): after `session.reload()`, call
  `bootPastLanding(session)` again (re-stub + re-pick the same subdirectory),
  then the existing title/node-count assertions.
- Step 10 (`--offline`): `setOffline(true)` → `reload` → `bootPastLanding`
  (the gate, stub, and vault reads all work from cache/OPFS) → assert
  `!!document.querySelector('.canvas') && !!window.orbitCanvas` →
  `setOffline(false)`.
- `components`, `widgets` (including its extra `failureSession` instances at
  1206/1343), and `shot`: replace each `waitFor("window.orbitCanvas...")`
  that immediately follows a `navigate()` with `bootPastLanding(session)`.

#### 4.4 `SKILL.md` update

Document: the `contract` subcommand; the picker-stub mechanism
(`window.showDirectoryPicker` overridden with an OPFS handle, real CDP click
on `#openVaultFolder`); the re-pick-after-reload behavior (the Vault gate
shows on every load; the smoke suite re-stubs automatically); that
`--profile` still drives persistence testing via the same OPFS subdirectory;
that the real-picker flow, external-change reconciliation, and the gate on
Firefox/Safari are manual smoke.

**Verify (Stage 3)**:
- `node --check .pi/skills/browser-check/scripts/browser-check.mjs` → exit 0
  (it is an `.mjs` module; `node --check` works on it).
- With the app served on 4173: `contract` → all pass; `smoke --offline` → all
  pass; `components` and `widgets` → all pass (run at least `smoke --offline`;
  the others where time allows).
- `git diff --check` → clean.

### 5. Documentation (Stage 4; AGENTS.md §14: same change as behavior)

- `docs/adr/0004-directory-vault-storage.md`: pre-staged in this worktree with
  final content. Verify it matches the spec's ADR description (context,
  decision, consequences, verification boundary) and commit unchanged.
- `CONTEXT.md`: pre-staged (Vault reword + DirectoryVault / Vault gate / Adopt
  entries). Verify and commit unchanged. Do not edit further.
- `docs/architecture.md`:
  - line ~10 adapter trio → `DirectoryVault (browser) / FsVault (Node) / MemoryVault (tests)`;
  - line ~35 boot list → (1) incompatibility gate (`showDirectoryPicker` +
    `crypto.subtle`), (2) Vault gate → `showDirectoryPicker({ mode: "readwrite" })`,
    (3) DirectoryVault opens; WorkspaceStore loads/migrates (create if empty,
    Adopt if files-but-no-sidecar, open if sidecar), (4) index rebuild,
    (5) render + `window.orbitCanvas`, (6) progressive SW registration; delete
    the localStorage first-run migration step (line ~37) and state no handle
    is persisted;
  - line ~43 paragraph: drop the one-time-migration sentences; verification
    sentence → headless contract suite via OPFS, picker-stub smoke, manual
    Chromium smoke;
  - line ~75 VaultStore paragraph → DirectoryVault is the browser default,
    folder-backed, no content cache, non-atomic `createWritable` guarded by
    `expectedHash`;
  - line ~97: `orbit-shell-v12` → `orbit-shell-v13`; "It does not cache
    IndexedDB records" → "It does not cache vault files (the user-picked
    folder owns them)";
  - "Future packaging" paragraph (~105): browser directory access is now
    shipped; a future desktop shell would add its own fs-kind adapter under
    the same contract.
- `docs/life-data.md`:
  - line ~22 adapter list → DirectoryVault replaces IndexedDbVault
    (folder-backed, disk mtime, non-atomic write + expectedHash guard);
  - line ~26: drop the "Upgrading a legacy localStorage profile is a clean
    break" sentence (the migration no longer exists);
  - line ~253 vault-writes paragraph → DirectoryVault keeps nothing in
    IndexedDB; the picked folder's plain files are the vault; the Service
    Worker never caches them.
- `docs/offline.md`:
  - line ~10 and every `orbit-shell-v12` mention (6 total) → `orbit-shell-v13`;
  - line ~11 runtime-pieces bullet → user files live in the user-picked
    folder, never in the SW cache;
  - precache list description gains `directory-vault.js`/`memory-vault.js`,
    loses `indexeddb-vault.js`;
  - line ~50 localStorage row → limited to theme and AI settings (no
    migration input);
  - line ~63 verification paragraph → vault-first folder reconstruction via
    the picker stub.
- `README.md`:
  - feature bullet (~27) and storage section (~131-135) → folder vault,
    Chromium requirement, re-pick per launch, Reload vault / Open another
    vault, localStorage limited to theme/AI settings; drop the
    localStorage-clean-break sentence;
  - browser-requirements line (~185) → File System Access API
    (`showDirectoryPicker`); drop IndexedDB/OPFS;
  - new short "Migrating from the IndexedDB version" subsection: (1) export
    whole space from the currently deployed version BEFORE the switch,
    (2) open the new version and pick an EMPTY folder, (3) Import the
    `.orbit.json` (restore replaces the picked folder's file tree).
- `AGENTS.md`:
  - §3 repo map: the `storage/indexeddb-vault.js` line (line ~60) →
    `storage/directory-vault.js  File System Access browser vault adapter over a user-picked folder`;
    note `storage/memory-vault.js` doubles as the browser staging adapter;
    the paragraph after the map (line ~81) → DirectoryVault and the app's
    vault-first wiring require browser verification;
  - §5 boot model (lines ~153-161): renumber per the architecture.md list
    above; delete the localStorage first-run migration step; state that no
    handle is persisted and the folder is re-picked every launch;
  - §5.1 (line ~169): cache name `orbit-shell-v7` → `orbit-shell-v13` (the
    doc is stale; sw.js is at v12 pre-change, v13 post-change); "IndexedDB
    owns user files" → "the user-picked folder owns user files";
  - §7 adapter list (line ~191): DirectoryVault is the browser adapter;
    IndexedDbVault gone;
  - §13 browser-pending list: reword to the spec's eight items
    (DirectoryVault open/write/restore + permission loss + deleted
    directories; vault-gate boot + reload/re-pick persistence + first-render
    budget; external-change reconciliation via Reload vault; task/Today
    projections over the folder vault; version-2 export/import round-trip;
    offline reload + cache upgrade from v12; timezone/local-date boundaries;
    malformed-file repair); Node count unchanged at 172; note the
    browser-check smoke suite now asserts the gate, runs the OPFS contract
    suite, and stubs the picker for full-app smoke.

**Verify (Stage 4)**: `git diff --check` clean; no `IndexedDbVault` claims
remain in `docs/`, `README.md`, `AGENTS.md` except the ADR-0004 "supersedes"
line and explicitly historical wording:
`grep -rn "IndexedDbVault\|indexeddb-vault" docs/ README.md AGENTS.md | grep -v "adr/0004"`.
All `orbit-shell-v` mentions in docs read `v13`.

## Sequencing (input to the tickets phase)

Five stages on the one feature branch `001-directory-vault`:

| Stage | Files | Depends on | File-disjoint with | shared-blast-radius |
|---|---|---|---|---|
| 1 adapter | `storage/directory-vault.js` | none | Stage 4 | no |
| 2 boot swap | `app.js`, `index.html`, `styles/shell.css`, `sw.js`, `storage/indexeddb-vault.js` (delete), two comment rewords in `storage/vault-store.js`/`storage/workspace-vault.js` | 1 | none | YES (app.js ~7 regions + 3 files; one ticket, sequential edits) |
| 3 driver | `.pi/skills/browser-check/scripts/browser-check.mjs`, `.pi/skills/browser-check/SKILL.md` | 2 | 4 | no |
| 4 docs | `docs/adr/0004-*` (commit), `CONTEXT.md` (commit), `docs/architecture.md`, `docs/life-data.md`, `docs/offline.md`, `README.md`, `AGENTS.md` | 3 for AGENTS.md §13 wording accuracy | 1 (fully); 3 (fully) | no |
| 5 deploy gate | none (manual rehearsal) | 1–4 merged-ready | n/a | n/a |

Ordering rules for the tickets phase:
- Stage 2 is NOT splittable across parallel tickets: between the markup and
  the boot rewire the app is unbootable. It is one ticket with ordered
  internal steps (§2.1–2.8, §3).
- Stages 1 and 4 are file-disjoint and MAY run in parallel (Stage 4's
  AGENTS.md §13 wording is written to the spec'd behavior, which is settled).
  If in doubt, run sequentially 1 → 2 → 3 → 4.
- Stage 3 must follow Stage 2 (the smoke suite drives the new gate).
- The Node suite stays green after EVERY stage commit; DirectoryVault adds no
  Node tests (browser-only, like IndexedDbVault was).
- Stage 5 is a merge gate, not a code ticket (Testing strategy below).

## Testing strategy

From the spec's Testing Decisions (the binding source):

1. **Node suite**: unchanged at 172 tests; green at every stage. No Node tests
   added or removed (File System Access has no Node implementation).
2. **Headless contract suite via OPFS** (Stage 3 `contract` subcommand):
   constructor injection seam; `navigator.storage.getDirectory()` provides the
   same `FileSystemDirectoryHandle` interface without a picker or gesture.
   Covers the full contract surface (listed in §4.2).
3. **Full-app headless smoke via picker stub** (Stage 3 rework): the driver
   stubs `window.showDirectoryPicker` with an OPFS handle and CDP-clicks the
   real gate button; assertions 1–10 run unchanged, with re-stub + re-pick
   after every reload. Binding app-code constraint: the gate handler
   references the global at call time (§2.4).
4. **Manual smoke (real Chromium; browser-pending per AGENTS.md §13, label
   don't claim)**: real picker UX; empty folder → create + starter seeding;
   files-but-no-sidecar → Adopt with foreign files untouched and matching
   files indexed; sidecar folder → open; external editor edit → Reload vault
   shows the change; conflicting app save → conflict toast, no overwrite;
   delete/rename the vault folder mid-session → read-only "files unavailable"
   affordance; Open another vault switches folders; import `.orbit.json` into
   an empty picked folder (migration rehearsal); export whole space from the
   folder vault; Firefox/Safari shows the gate with a disabled button; offline
   reload serves the shell from `orbit-shell-v13`.
5. **Deploy gate (Stage 5, pre-merge, mandatory)**: shipping orphans any data
   left in the old IndexedDB vault. Before merging to `main`: (a) open the
   LIVE deployed site (https://alexradunet.github.io/open-canvas-experiment/)
   in a Chromium browser with data present and Export whole space to a
   `.orbit.json`; (b) serve the feature branch locally, pick an EMPTY folder,
   Import the `.orbit.json`; (c) confirm the workspace renders complete
   (canvases, tasks, habits, journals). The owner accepted this one-time
   manual migration in the grill. Report the rehearsal result before merge.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `node --check app.js storage/directory-vault.js sw.js` exits 0 (and `node --check .pi/skills/browser-check/scripts/browser-check.mjs`)
- [ ] 172-test suite: `tests 172`, `pass 172`, `fail 0`
- [ ] `git diff --check` clean
- [ ] `storage/indexeddb-vault.js` deleted; dead-reference sweep returns no output (excluding `vendor/`, `teach/`, `plans/`)
- [ ] `grep -n "localStorage" app.js` shows only theme + AI-settings lines
- [ ] `grep -o 'from "\./[^"]*"' app.js | sort -u` modules all present in `sw.js` `APP_SHELL`; `CACHE_NAME` is `orbit-shell-v13`
- [ ] `browser-check.mjs contract` → all pass, exit 0
- [ ] `browser-check.mjs smoke --offline` → all pass, exit 0 (includes the gate record and re-pick-after-reload)
- [ ] `index.html` has `#vaultLanding` before `.app-shell`, `.app-shell` carries `hidden inert` in markup, `#reloadVault` and `#openAnotherVault` exist
- [ ] Docs: no non-historical `IndexedDbVault` claims outside `docs/adr/0004-*`; all doc cache mentions read `orbit-shell-v13`; README has the migration subsection
- [ ] ADR-0004 and `CONTEXT.md` committed unchanged from their pre-staged content
- [ ] Stage 5 migration rehearsal performed and reported (pre-merge gate)
- [ ] No files outside the staged scope lists are modified (`git status`)

## STOP conditions

Stop and report (do not improvise) if:

- The code at the locations in "Current state" doesn't match the excerpts
  (drift since `e3103b2`); run the drift check first.
- Finding F1 turns out wrong: `createGraphStarterWorkspace` uses
  `freshWorkspace`/`normalizeWorkspace` behavior beyond what §2.2 describes
  (e.g. relies on path backfill for a record without an explicit path).
- `WorkspaceStore.migrate` on a files-but-no-sidecar folder is NOT
  additive-only in practice (a pre-existing file would be overwritten or
  removed): Adopt depends on `expectedHash: null` creates only.
- Headless Chrome fails the gate unexpectedly (`showDirectoryPicker` missing
  or `crypto.subtle` absent on localhost), or OPFS (`navigator.storage
  .getDirectory()`) is unavailable in the headless profile: the contract and
  smoke strategy depends on both.
- The picker stub does not take effect because some app code captured
  `showDirectoryPicker` at module scope (violating the §2.4 binding
  constraint): fix the app code, not the stub.
- A torn-write or non-atomic `createWritable` issue produces data loss in the
  smoke suite (it should at worst surface as a repair placeholder).
- The 172-test count changes after any stage (no Node tests may be added or
  removed by this project).
- A step's verification fails twice after a reasonable fix attempt.
- The fix appears to require touching an out-of-scope file (e.g. the
  `VaultStore` contract, `workspace-backup.js`, Node test files).
- `pwd`/branch/worktree identity checks fail, or `git status --short` shows
  unexpected modifications before edits begin.

## Maintenance notes

- If a persisted directory handle (handle locker) is ever added, the gate's
  re-pick promise loop in §2.4 and the driver's re-stub-after-reload model
  both change; revisit the smoke suite's persistence step then. YAGNI for v1.
- If vault sizes grow large, `restore`'s delegated per-write fold check is
  O(n²) directory walks; the first optimization is an inline write loop with
  the up-front fold validation only.
- `DirectoryVault` writes are non-atomic (`createWritable`); if torn-write
  corruption ever appears in lived use, revisit atomic-rename emulation. The
  `expectedHash` precondition is the v1 guard (ADR-0004).
- Reviewers should scrutinize: the DOMException mapping table coverage; the
  `move` single-journal-bump invariant; `restore`'s whole-tree removal scope;
  the gate's promise resolution paths (gate fire vs pick success vs cancelled
  picker); and that no module-scope capture of `showDirectoryPicker` exists.
- Deferred explicitly: filesystem watcher/auto-refresh (one manual Reload
  vault is the v1 answer), Firefox/Safari fallback, persistent index,
  multi-device sync conflict design (validated by manual smoke, not designed).

## Domain flags

No new contradictions. `CONTEXT.md` in this worktree is already reconciled
(modified, pre-staged): `Vault` reworded; `DirectoryVault`, `Vault gate`, and
`Adopt` added. The domain-model-close step for this project reduces to
committing that file unchanged (Stage 4). One naming note for the executor:
the glossary term is "Vault gate" but the HTML id is `#vaultLanding` and the
CSS class is `.vault-landing`; this is intentional (ids/classes are stable
hooks; the glossary governs prose).

## Next step

- Recommended executor tier: mid (standard implementation; the architecture is
  fully specified here; premium not required)
- Recommended model: confirm against `para/resources/model-registry.md` at
  spawn time (Stage 2 is the highest-judgment ticket: app.js boot rewire)
- Estimated complexity: L overall (Stage 1 M, Stage 2 L, Stage 3 M, Stage 4 S)
- After implementation: run /review on this worktree, then the Stage 5 deploy
  gate before merge
