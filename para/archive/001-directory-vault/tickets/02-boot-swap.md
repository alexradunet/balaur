---
phase: ticket
status: done
project: 001-directory-vault
ticket: 02
blocked-by: [01]
worker: "a3dea94"
branch: "001-directory-vault"
shared-blast-radius: true
---

# Ticket 02: Boot swap — Vault gate, DirectoryVault boot, shell, delete IndexedDbVault

## What to build

The app boots through the Vault gate instead of IndexedDB. Launch becomes:
incompatibility gate → Vault gate → `showDirectoryPicker({ mode: "readwrite" })`
→ `DirectoryVault` opens → `WorkspaceStore` loads (create if empty, Adopt if
files-but-no-sidecar, open if sidecar) → index rebuild → render. `IndexedDbVault`
and the one-time `localStorage` first-run migration are deleted; staging for
import and Reset starter moves to `MemoryVault`; the Service Worker precache
swaps to the new modules at `balaur-shell-v13`.

This is ONE unsplittable ticket: between the markup change and the boot rewire
the app is unbootable (the gate overlay would cover a still-IndexedDB app), so
all edits land atomically. Source of truth: `plan.md` §2 (app.js regions
§2.1–2.8) and §3 (shell). AGENTS.md §12 forbids mass-formatting — targeted
edits only, match the surrounding compact style.

## Acceptance criteria

- [ ] §2.1 Import swap (app.js:2): replace the `IndexedDbVault` import with `import { DirectoryVault } from "./storage/directory-vault.js";` and `import { MemoryVault } from "./storage/memory-vault.js";`.
- [ ] §2.2 Finding F1 — starter builder rewire: add `minimalFreshWorkspace()` (version 1, single root canvas, empty document, title "Balaur", default camera; no localStorage, no demo canvas). In `createGraphStarterWorkspace()` replace `freshWorkspace(rootDocument)` with `minimalFreshWorkspace()` + document assignment, and `return normalizeWorkspace(result)` with `return result`.
  - [ ] **STOP trigger (F1):** stop and report if `createGraphStarterWorkspace` relies on `freshWorkspace`/`normalizeWorkspace` behavior beyond §2.2 (e.g. path backfill for a record without an explicit path). Do not improvise.
- [ ] §2.3 Deletions (targeted, in order): `demoCanvas` constant (app.js:24-47); `loadDocument()`, `freshWorkspace()`, `normalizeWorkspace()`, localStorage-reading `loadWorkspace()` (app.js:180-211); `WORKSPACE_KEY` from app.js:125 (keep the line as `const ROOT_CANVAS_ID="canvas-root";`); reword the stale boot comment at app.js:327-328 to name the user-picked folder.
- [ ] §2.4 Boot rewrite: add `let currentVault=null;` near `let vaultStore=null;`. Replace `bootCanvasApp()` with the gate + pick-await shape: (1) incompatibility gate on missing `showDirectoryPicker` or `crypto.subtle` (fill `#vaultLandingMessage`, disable `#openVaultFolder`, `setIndexStatus("Files unavailable", …)`, return — no fallback adapter); (2) wait for a successful pick, referencing the **global** `window.showDirectoryPicker({ mode: "readwrite" })` at call time (BINDING CONSTRAINT — never capture it in a module-scope const; the driver stubs it); swallow `AbortError`, route other picker errors to `#vaultLandingMessage`; (3) on a handle, `new DirectoryVault(handle)`, detect `hadSidecar`/`empty`, `openVault(vault, { seed: empty })`, reveal the shell, resolve.
- [ ] `openVault(vault, { seed })` implemented as the shared core (steps 1–8 in plan §2.4): `WorkspaceStore`; `hasWorkspace`; if `!had` → `store.migrate(seed ? createGraphStarterWorkspace() : minimalFreshWorkspace())` (additive-only Adopt via `expectedHash: null`); `store.load()` with diagnostics + empty-workspace guard; assign `workspace`/`vaultStore`/`window.balaurVaultStore` + `setCanonicalWritable` + diagnostic warnings + `configureLifeRuntime` + `seedBundledWidget`; if `seed` → `seedGraphStarterEntities()` + reload; rebuild the three catalogs + `setIndexStatus`; existing post-boot block. Throws on failure; callers route the error.
- [ ] Module tail (app.js:1825-1831) keeps its exact shape: `vaultReady=bootCanvasApp(); window.balaurVaultReady=vaultReady; await vaultReady;` then the existing `window.balaurCanvas` assignment. `window.balaurCanvas` is exposed in both gated and opened modes.
- [ ] §2.5 Wiring near app.js:1815: `#reloadVault` → `openVault(currentVault, { seed: false })` with reload-failure affordances (`setIndexStatus("Files unavailable", …)` + `setCanonicalWritable(false, …)`; do NOT null runtime globals). `#openAnotherVault` → `flushPendingWorkspaceEdits()`, `inert`+hide shell, unhide `#vaultLanding`.
- [ ] §2.6 Staging re-point: `importCanvas` version-2 branch staging → `new MemoryVault()` and `canonicalVault = vaultStore.vault`; `loadGraphStarter` staging → `new MemoryVault()` (verify `canonicalVault` at app.js:408 is already `vaultStore.vault`, do not change). Reword the stale "IndexedDB restore is one transaction" comment (app.js:1382-1384). `exportWorkspace()` unchanged.
- [ ] §2.7 Comment rewords (logic untouched): `storage/vault-store.js:4` and `storage/workspace-vault.js:9` name `DirectoryVault` instead of `IndexedDbVault`.
- [ ] §2.8 `storage/indexeddb-vault.js` deleted (`git rm`).
- [ ] §3.1 `index.html`: `#vaultLanding` overlay (wordmark, hint, `#openVaultFolder` primary button, `#vaultLandingMessage role="status"`) as the FIRST child of `<body>` before `.app-shell`; `.app-shell` carries `hidden inert` in markup; `#reloadVault` and `#openAnotherVault` nav-item buttons in `.sidebar-bottom`; `#importButton`/`#fileInput`/`#exportWorkspaceButton`/`#resetDemo` kept.
- [ ] §3.2 `styles/shell.css` (inside `@layer shell`, Balaur tokens): `.app-shell[hidden] { display: none; }` and `#vaultLanding[hidden] { display: none; }` (the grid `display` overrides UA `[hidden]`); `.vault-landing` fixed full-viewport centered column above the shell; `prefers-reduced-motion` respected.
- [ ] §3.3 `sw.js`: `CACHE_NAME = "balaur-shell-v13"`; `APP_SHELL` drops `"./storage/indexeddb-vault.js"`, adds `"./storage/directory-vault.js"` and `"./storage/memory-vault.js"`, keeps `"./storage/workspace-backup.js"`; every module in `grep -o 'from "\./[^"]*"' app.js | sort -u` is present in `APP_SHELL`.
- [ ] Verify: `node --check app.js storage/directory-vault.js sw.js` exits 0; 172-test suite all pass; dead-reference sweep (`grep -rn "indexeddb-vault\|IndexedDbVault" --include="*.js" --include="*.mjs" . | grep -v "^./vendor/\|^./teach/\|^./plans/"`) returns no output; `grep -n "localStorage" app.js` shows only theme + AI-settings lines; `git diff --check` clean.
- [ ] STOP conditions honored (plan): `WorkspaceStore.migrate` on a files-but-no-sidecar folder must be additive-only in practice; no module-scope capture of `showDirectoryPicker`; 172-test count unchanged. Browser behavior is verified in ticket 03; this ticket may be smoke-tested ad-hoc with `eval` + a manual stub.

## Domain flags

Glossary term is "Vault gate" but the HTML id is `#vaultLanding` and the CSS
class is `.vault-landing`; this is intentional (ids/classes are stable hooks,
the glossary governs prose). "Adopt" describes the files-but-no-sidecar open
(`store.migrate(minimalFreshWorkspace())`, additive-only) — do not label it
import/migrate/seed in user-facing prose.

## Blocked by

Ticket 01 (DirectoryVault adapter) — the boot rewire imports and opens it.
Runs sequentially (flagged `shared-blast-radius: true`).
