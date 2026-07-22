# AGENTS.md — Balaur repository guide

This file applies to the entire repository. Balaur is a local-first life-management application built on the open [JSON Canvas 1.0](https://jsoncanvas.org/spec/1.0/) format. The repository is a static site: native strict ES modules, vanilla JavaScript, no package install, build step, CDN dependency, or UI framework.

## 1. Shipped architecture

Canonical user data lives in a vault and has four parts:

```text
JSON Canvas files (.canvas)   canonical spatial documents
Markdown files (.md)          canonical life entities
.orbit/workspace.json         hierarchy, cameras, and application-only metadata
MemoryIndex                   disposable runtime query projection
```

Tasks, habits, habit logs, journals, and calendar events are canonical Markdown files. Canvases are independently valid JSON Canvas documents. The in-memory index is rebuilt from those files at boot and reconciled after changes; deleting or rebuilding it cannot delete user data.

SQLite is not part of canonical-files-only v1 and is not loaded by the browser application. A persistent index is a deferred optimization. In particular, OPFS-backed SQLite Wasm needs COOP/COEP headers that GitHub Pages cannot provide, so the pure-JavaScript in-memory index is the compatible default. Do not add a persistent database, marker-task system, or localStorage-backed source of truth without an accepted architecture change.

Preserve these constraints. Do not introduce a proprietary Canvas dialect, framework, build pipeline, runtime package manager, or host-page execution of generated code. Generated or user-authored code runs only in sandboxed file-node widgets.

The live site is deployed from `main` to <https://alexradunet.github.io/open-canvas-experiment/>.

## 2. Read the relevant documentation first

Before changing a subsystem, read its source and design document:

- `README.md` — user-visible behavior and local run instructions
- `docs/architecture.md` — standards-first architecture and ownership boundaries
- `docs/life-data.md` — canonical file layouts, codecs, repositories, indexing, and backups
- `docs/offline.md` — Service Worker cache strategy and browser validation
- `docs/generative-canvas.md` — AI operations, live widgets, and security boundaries
- `docs/design-system.md` — Balaur tokens, material roles, motion, and CSS organization
- `docs/adr/0001-file-canonical-life-data.md` — accepted file-canonical decision
- `plans/canonical-v1-files-only.md` — implementation plan and browser-pending work
- `vendor/pixel-loom/README.md` — design-system provenance

`plans/` contains historical and forward-looking material. Do not treat a plan as proof of shipped behavior. This guide describes the single shipped model; the browser-pending list below is the boundary for claims about browser verification.

## 3. Repository map

```text
index.html                  Static application shell, dialogs, templates, landmarks
main.js                     Ordered ES-module entry point and offline registration
server.mjs                  Dependency-free local static server
server.test.mjs             Static server behavior and traversal tests
app.js                      Vault-first canvas engine, workspace, JD, tasks/Today, AI, import/export
offline/register.js         Progressive Service Worker registration
sw.js                       Versioned same-origin application-shell cache
manifest.webmanifest        Install metadata, scope, colors, and local icons
icons/                      PWA icons, including a maskable-safe 512px asset
storage/vault-errors.js     Typed vault errors (path, parse, schema, conflict, storage)
storage/content-hash.js     Stable async SHA-256 content hashing
storage/vault-path.js       Cross-platform vault path normalization and entity paths
storage/frontmatter.js     Preservation-first frontmatter scan/parse/patch codec
storage/entity-codec.js    Canonical task/habit/journal/event Markdown codecs
storage/vault-store.js      VaultStore contract and media-type inference
storage/memory-vault.js     In-memory vault adapter for tests
storage/indexeddb-vault.js IndexedDB browser vault adapter; browser verification pending
storage/fs-vault.js         Node filesystem reference adapter and tooling
storage/life-indexer.js     Projects vault files into the runtime index
storage/life-query.js       App-facing query facade over MemoryIndex
storage/memory-index.js     Disposable in-memory index port
storage/canvas-validate.js  Shared strict JSON Canvas 1.0 validator
storage/workspace-vault.js  Sidecar plus per-canvas `.canvas` persistence
storage/workspace-backup.js Version-2 whole-space file-bundle export/import
storage/task-repository.js  FileTaskRepository: canonical task files and placements
storage/habit-repository.js FileHabitRepository: definitions and daily check-ins
storage/journal-event-repository.js  Journal and calendar-event repositories
storage/index-integrity.js  Runtime-index audit and purge/rebuild recovery
styles/                     Named cascade layers, tokens, shell, canvas, components, themes, responsive, motion
docs/                       Architecture and subsystem documentation
widgets/                    Sandboxed HTML file-node widgets
vendor/pixel-loom/          Self-hosted fonts and upstream design-system provenance
.github/workflows/pages.yml Deploys the repository root as a static GitHub Pages artifact
```

The storage foundation is Node-verified by the phase suites listed in §13. `MemoryVault`, `FsVault`, `MemoryIndex`, `LifeIndexer`, repositories, backup validation, and integrity auditing are reference/test/tooling surfaces. `IndexedDbVault` and the app's vault-first wiring require browser verification.

Do not edit vendored binaries as application source. If a vendor is intentionally updated, update its version, license/provenance, file list, and checksums together.

## 4. Non-negotiable data boundaries

### 4.1 JSON Canvas owns portable spatial content

A canvas document is exactly a JSON Canvas document with top-level `nodes` and `edges` arrays. Use only standard node types:

- `text`
- `file`
- `link`
- `group`

Do not add custom node types (`task`, `canvas`, `widget`, `ai`, `habit`) or application-only node and edge fields. Use standard IDs, geometry, colors, content fields, and edge routing fields. Life entities are placed through standard `file` nodes referencing canonical `.md` files. A canvas node ID identifies a placement, not the entity.

All imported, restored, or model-generated documents must pass `storage/canvas-validate.js` `isCanvas()` and the relevant operation/workspace validators. IDs must be globally unique within a document and edge endpoints must reference existing nodes. A single `.canvas` export remains valid JSON Canvas, but referenced entity files may be outside that export; only a whole-space bundle is a complete life-data backup.

### 4.2 Standard nodes plus inert markers

Special behavior is represented without changing JSON Canvas:

```md
<!-- orbit:jd 11.01 -->
<!-- orbit:ai-card -->
<!-- orbit:habit-entry id=... habit=... status=done value=1 at=... -->
```

- Johnny Decimal item notes are standard text nodes.
- Tasks, habits, journals, and calendar events are canonical Markdown files under `tasks/`, `habits/`, `habit-logs/`, `journal/`, and `events/`, each with the applicable Orbit frontmatter contract. Tasks are placed by standard `file` nodes; one task may have zero, one, or many placements.
- Habit check-ins are inert comments in append-only daily habit-log files. They are not recurring task records.
- AI operators are standard text nodes whose incoming edges provide context. For a file node, context assembly resolves the referenced file body, not just its path.
- Live HTML/Canvas/WebGL widgets are standard `file` nodes pointing to `.html` files.
- Nested-canvas portals are standard `file` nodes pointing under `canvases/`.

Markers must remain harmless and readable in other editors. There is no generated task-marker compatibility layer.

### 4.3 Workspace sidecar and canvas files

The canonical sidecar is `.orbit/workspace.json`. It owns canvas titles and paths, parent/portal relationships, active canvas, cameras, and Johnny Decimal metadata. It does not embed canvas documents. Each canvas document is stored separately under `canvases/*.canvas` and is validated independently.

Do not put hierarchy, cameras, active filters, selection, or other UI state into exported `.canvas` documents. Destructive hierarchy operations must remove related sidecar records and orphaned canvas files safely. Missing or invalid canvas files load as read-only repair placeholders with their raw content retained; saving must never replace them with an empty document.

### 4.4 Canonical files and the disposable index

Canonical Markdown files own the fields JSON Canvas does not define:

- task status, priority, scheduling, due dates, completion, recurrence, and estimates;
- habit definitions and immutable daily check-in events;
- journal dates; and
- calendar event times, dates, and timezones.

`LifeIndexer` parses vault files into `MemoryIndex`; `LifeQuery` supplies Today, task-status, habit, journal, and calendar queries. This index is runtime state, not a second owner. It is rebuilt at boot and may be purged and rebuilt at any time. Repositories write canonical files first, then reindex them. No meaningful state may exist only in the index.

Date conventions are mandatory:

- local dates: `YYYY-MM-DD`;
- instants: ISO 8601 strings;
- calendar timezones: IANA timezone names; and
- scheduling intent (`scheduled-on`) is separate from a deadline (`due-on`).

### 4.5 Whole-space backups

A version-2 `.orbit.json` backup contains the metadata-only sidecar and raw logical vault files (`.canvas`, `.md`, widgets, and other stored files). It never contains a database snapshot. Import validates paths, canvases, file-node references, entities, duplicate IDs, and diagnostics before writing to an empty staging vault. Version-1 bundles are rejected in canonical v1; the old migration format is intentionally not a second source of truth.

## 5. Runtime and initialization model

The module graph rooted at `main.js` is deliberate. The app's intended startup sequence is:

1. open the browser `IndexedDbVault`;
2. load `.orbit/workspace.json` and each referenced `.canvas` document through `WorkspaceStore`;
3. use the one-time first-run `localStorage` migration source only when the vault has no sidecar;
4. construct `MemoryIndex`, `LifeIndexer`, `LifeQuery`, and the task repository used by the shipped UI;
5. rebuild the in-memory index from the vault files;
6. render the workspace and expose `window.orbitCanvas`; and
7. progressively register the Service Worker.

The vault is the post-migration source of truth. `localStorage` is not a workspace fallback or persistence mirror. Vault-first boot, IndexedDB persistence, first-render timing, and graceful behavior in a real browser are browser-pending even though the wiring is present and the Node storage logic is tested. If the vault cannot be opened, the UI reports that canonical files are unavailable rather than treating a localStorage workspace as authoritative.

Rendering must use the loaded in-memory working set; it must not perform an asynchronous file read for every card during rendering. On writes, save the canonical file or canvas with an expected-content-hash precondition, then reindex and refresh projections. Warm reconciliation honors vault revisions and move old paths.

`window.orbitVaultReady` is the boot promise and `window.orbitVaultStore` is the configured workspace store when boot succeeds. The stable `window.orbitCanvas` surface exposes document/workspace getters, validation and operation application, AI-card execution, hierarchy/JD navigation, task creation, view switching, and whole-space export. Keep raw mutable internals private.

### 5.1 Offline application-shell rules

`sw.js` precaches the required local modules, styles, fonts, icons, manifest, and sample widget under cache `orbit-shell-v7`. It does not precache database Wasm. Same-origin GET requests use network-first with cache fallback; cross-origin requests, non-GET requests, range requests, provider calls, API keys, generated exports, and arbitrary external resources are not intercepted.

Keep `APP_SHELL` synchronized with required assets and increment `CACHE_NAME` when cache semantics or invalidation require it. Keep paths relative for GitHub Pages subpaths. Never call `skipWaiting()` to reload an active editing session without an explicit safe update flow. The Service Worker owns only static shell resources; IndexedDB owns user files.

## 6. Canvas/workspace mutation rules

`documentData` represents the active canvas and `workspace.canvases[currentCanvasId]` is its sidecar record. Keep both coherent.

When changing a canvas:

1. save the current document and camera through the existing helpers;
2. mutate the document or workspace through existing navigation/persistence helpers;
3. schedule the canonical vault save;
4. render only affected projections where practical; and
5. let `LifeIndexer` reconcile the changed canvas and placements.

Use `switchCanvas()`, `enterSubcanvas()`, `leaveSubcanvas()`, and `revealWorkspaceNode()` instead of hand-updating navigation state. A canvas file deletion removes placements only when the operation explicitly requests that; deleting an entity everywhere is a separate confirmed repository operation. Never mutate exported documents with selection, drag previews, filters, dialogs, runtime AI state, or app view.

## 7. File-canonical storage rules

All vault adapters implement the asynchronous `VaultStore` contract with safe paths, content hashes, optimistic `expectedHash` preconditions, revisions, and change records:

- `IndexedDbVault` is the browser adapter;
- `FsVault` is the Node filesystem reference adapter; and
- `MemoryVault` is the deterministic test adapter.

Use `frontmatter.js` and the entity codecs rather than a general YAML library. Patching must preserve unknown frontmatter keys, comments, ordering, BOM, line endings, and Markdown body bytes. Validate dates, instants, enums, weekday ranges, IANA zones, path case-fold collisions, and JSON Canvas structure at storage/import boundaries. Do not concatenate untrusted content into HTML, CSS, or commands.

The runtime index is intentionally disposable. A future persistent index may optimize large vaults, but it must remain a projection that can be rebuilt from canonical files. OPFS-backed SQLite Wasm is one possible future optimization, not a v1 dependency; GitHub Pages cannot provide its required COOP/COEP headers.

Activity or diagnostics that matter to recovery belong in canonical files or explicit sidecar metadata, not only in the disposable index.

## 8. Task and temporal feature rules

Task creation is file-first:

1. generate an immutable `orbit-id` and safe stable path;
2. write a canonical `tasks/*.md` file;
3. add a standard `file` node placement to the selected canvas;
4. save the canvas; and
5. index the changed files and refresh Canvas/Today projections.

A portable task has the following shape:

```md
---
orbit-schema: 1
orbit-type: task
orbit-id: "task-a1b2c3"
title: "A readable task title"
status: next
scheduled-on: "2026-07-22"
due-on: "2026-07-25"
---
Optional context remains ordinary Markdown.
```

Task completion and edits patch canonical frontmatter with an expected hash. Removing a canvas `file` node removes one placement while preserving the task and other placements. Deleting everywhere removes placements first, then the canonical file with its last-known hash. Do not infer a due date from a scheduled date.

Habits are definitions plus append-only daily check-in events. Journals and calendar events use the same file-canonical layer rather than feature-specific stores. Test local-date behavior with timezone boundaries; never derive a local date by slicing a UTC timestamp.

## 9. Johnny Decimal and nested-canvas rules

Johnny Decimal is a validated projection over the nested-canvas hierarchy:

- root → area (`10-19`);
- area → category (`11`); and
- category → item (`11.01`).

Preserve numeric ordering, parent-range validation, duplicate rejection, sidecar/heading synchronization, and reachability through standard file-node portals. Do not flatten nested documents during save or export.

## 10. AI and widget security boundaries

AI output never directly mutates the host DOM or executes host-page JavaScript. Accept only allowlisted structured operations, validate IDs/fields/URLs/geometry/operation counts and the resulting canvas, show a human-readable proposal, and require confirmation before applying it. Typed life changes must call the file repositories, not write index rows directly.

AI operators remain standard text nodes with inert markers and edge-derived inputs. Preserve debouncing, stable output-node reuse, queued reruns, and cycle detection. File-node inputs resolve canonical file bodies.

Live widgets run in iframes with exactly:

```html
sandbox="allow-scripts"
```

Do not add same-origin, navigation, popup, form, download, device, or filesystem permissions without a security review. Use direct WebGL2 only inside self-contained widgets with HTML/CSS/2D fallbacks, bounded DPR, resize and visibility handling, reduced-motion support, context-loss recovery, and cleanup. Escape dynamic HTML, validate URLs, pair new-tab links with `rel="noreferrer"`, and keep provider keys in `sessionStorage` unless the user explicitly enables **Remember API key**.

## 11. UI, CSS, and accessibility conventions

Use semantic landmarks, native forms/buttons/dialogs, Pointer Events, SVG edges, `inert` plus `aria-hidden` for hidden panels, useful accessible names, keyboard behavior, and reduced-motion adaptations. Do not introduce React, Vue, Svelte, utility CSS, CSS-in-JS, or a build pipeline.

CSS uses the explicit order in `styles/layers.css`:

```css
@layer foundation, shell, canvas, components, themes, responsive;
```

Put rules in the appropriate existing layer, prefer Balaur tokens, use logical properties and progressive enhancement, and inspect narrow layouts after desktop changes. JSON Canvas preset colors remain document semantics, separate from application-theme tokens.

## 12. JavaScript conventions

Match the native strict-ES-module style. Prefer cohesive modules and named helpers, explicit side effects, feature detection, stable IDs, event delegation, `AbortController` where useful, and validation at storage/import/AI boundaries. Disable asynchronous controls while running and surface errors in the relevant status region. Stop pointer propagation for controls inside draggable cards. Avoid globals when values can be derived from workspace, document data, or the runtime index.

Do not mass-format `app.js`. Extend `window.orbitCanvas` only for a stable browser/integration command; do not expose raw mutable storage or index internals.

## 13. Running and validating changes

There is no package manifest and no install prerequisite. Serve over HTTP, not `file://`:

```bash
python3 -m http.server 4173
```

For storage changes, run the Node-verified foundation and query suites:

```bash
node --test \
  storage/phase1.test.js storage/phase2.test.js storage/phase3.test.js \
  storage/phase4.test.js storage/phase4-backup.test.js storage/phase5.test.js \
  storage/phase7.test.js storage/phase8.test.js storage/phase9.test.js \
  storage/phase10.test.js storage/phase-query.test.js
```

This explicit suite currently passes **168 tests**: the prior 164-test suite plus four component-card backup-boundary tests in `phase4-backup.test.js` (the deleted phase6 suite is intentionally excluded). Also run `git diff --check`; for JavaScript changes run `node --check` on every touched module.

Then perform browser-level checks appropriate to the change. **The default way to check the application is the project `browser-check` skill** at `.pi/skills/browser-check/` — a dependency-free headless-Chrome-over-CDP driver that runs the baseline smoke suite below automatically (no WebDriver, no npm install). With the app served on `4173`:

```bash
node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline
```

It boots a fresh profile and asserts: no uncaught console errors or failed assets; every document node renders; the sidebar reports the file-index status (canonical v1 loads **no SQLite**); clicking a card selects it and shows the corner-bracket selection frame (no circles); double-clicking *inside* a card and the note tool *on* a card create nothing; double-clicking empty background still creates a note; the live document stays valid JSON Canvas 1.0; a controlled reload preserves the workspace; and `--offline` confirms the Service Worker renders the shell from cache. Use `--profile <dir>` to reuse a profile across runs (first-run vs existing-profile / migration testing), `--width`/`--height` for narrow-shell checks, and `--screenshot <dir>` to dump a PNG for visual review. `eval` and `shot` subcommands run one-off runtime probes and screenshots. Read `.pi/skills/browser-check/SKILL.md` for recipes and the headless event-retargeting caveat. The skill is local agent tooling under `.pi/skills/` (tracked in the repo but not part of the deployed application shell); if it is absent (e.g. a fresh clone without `.pi/`), use the manual baseline list below instead.

The skill automates most of the manual list below; fall back to a real browser for anything it cannot express (task completion + Today projection, whole-space import, destructive reset). When you do test manually, use a fresh temporary browser profile for first-run behavior and retain the same profile for reload/persistence tests.

### Browser-pending verification

Do not claim these are browser-verified from Node tests alone:

1. IndexedDB open/write/restore/quota behavior;
2. vault-first boot, first-render budget, and reload persistence;
3. task create, edit, complete, and Today UI projections;
4. version-2 export/import round-trip and destructive recovery paths;
5. offline reload, Service Worker control, and cache upgrade;
6. timezone/local-date behavior at browser boundaries; and
7. malformed-file repair behavior in the running UI.

When browser testing is available, use a fresh temporary profile for first-run behavior and a retained profile for reload/persistence. Check `await window.orbitVaultReady`, `window.orbitVaultStore`, `window.orbitCanvas.getDocument()`, `window.orbitCanvas.getWorkspace()`, and `window.orbitCanvas.getSummary()`.

## 14. Documentation expectations

Update documentation in the same change as behavior changes:

- user-visible behavior → `README.md`;
- ownership/module boundaries → `docs/architecture.md`;
- file contracts, repositories, index, or backups → `docs/life-data.md`;
- shell caching and offline behavior → `docs/offline.md`;
- AI, widgets, and provider security → `docs/generative-canvas.md`;
- tokens and visual rules → `docs/design-system.md`; and
- architecture decisions → `docs/adr/`.

Distinguish implementation from browser verification. Do not describe future persistent indexing, Tauri, sync, recurrence, or calendar-provider work as shipped.

## 15. Git and deployment hygiene

Keep changes scoped. Do not commit generated browser profiles, screenshots, logs, API keys, local databases, or temporary exports. Do not amend, reset, commit, or push unless explicitly asked. Every push to `main` deploys the repository root through GitHub Pages.

## 16. Definition of done

A change is complete when applicable:

- JSON Canvas remains standards-compliant and independently portable;
- canonical files, sidecar, and disposable index ownership remain clear;
- writes are canonical-file-first and imports/exports validate boundaries;
- existing data is not silently orphaned or overwritten;
- dynamic content and widgets retain their security boundaries;
- wide and narrow interfaces remain usable;
- static checks and relevant Node tests pass;
- browser-pending behavior is labeled rather than claimed; and
- documentation matches the shipped files-only v1 architecture.
