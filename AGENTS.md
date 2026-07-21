# AGENTS.md — Balaur repository guide

This file applies to the entire repository. It is written for coding agents and future maintainers working on Balaur.

## 1. Project intent

Balaur is a local-first life-management application built around the open [JSON Canvas 1.0](https://jsoncanvas.org/spec/1.0/) format. Its distinctive constraints are part of the product, not temporary implementation details:

- JSON Canvas files are the canonical portable spatial/document layer.
- Every nested canvas is independently valid JSON Canvas 1.0.
- Markdown files are the canonical human-readable life entities (tasks, habits, journals, events).
- SQLite is a disposable, rebuildable query/search index over those files, not the owner of visible content.
- A small workspace sidecar owns hierarchy and application-only canvas state.
- The browser application uses platform standards, strict ES modules, and vanilla JavaScript.
- The application shell is installable and offline-capable through a web app manifest and Service Worker.
- The static prototype has no package install, CDN dependency, build step, UI framework, or runtime package manager.
- Generated or user-authored code runs only in sandboxed file-node widgets, never in the host application.

Preserve these constraints unless a task explicitly changes the architecture. Do not solve a local problem by silently creating a proprietary Canvas dialect, introducing a framework, or making SQLite the owner of visible canvas content.

The ownership model is being migrated toward file-canonical storage: canonical `.md` life entities and `.canvas` documents, with SQLite demoted to a rebuildable index. This direction was explicitly adopted by [`docs/adr/0001-file-canonical-life-data.md`](docs/adr/0001-file-canonical-life-data.md). Read that ADR and `plans/markdown-canonical-sqlite-index.md` before changing storage, tasks, or persistence, and keep the current prototype working while moving toward the target.

The live site is deployed from `main` to:

- <https://alexradunet.github.io/open-canvas-experiment/>

## 2. Start by reading the relevant documentation

Before changing a subsystem, read its source and its design document:

- `README.md` — current user-visible behavior and local run instructions
- `docs/architecture.md` — standards-first architecture and ownership boundaries
- `docs/life-data.md` — SQLite schema, persistence, task projection, and backup format
- `docs/offline.md` — Service Worker cache strategy, lifecycle, limits, and validation
- `docs/generative-canvas.md` — AI operations, live widgets, and security boundaries
- `docs/design-system.md` — Balaur tokens, material roles, motion, and CSS organization
- `vendor/sqlite/README.md` — SQLite provenance and current VFS limitations
- `vendor/pixel-loom/README.md` — design-system provenance
- `docs/adr/0001-file-canonical-life-data.md` — accepted decision to move to file-canonical storage
- `plans/markdown-canonical-sqlite-index.md` — the file-canonical migration plan and phasing
- `plans/markdown-canonical-sqlite-index.review.md` — review of that plan with open issues

Some sections of `docs/architecture.md` describe a future modular source tree and command system. Do not mistake proposed architecture for code that already exists. The current application is a static prototype centered on `app.js`.

The file-canonical storage model (ADR-0001) is the adopted direction but is not yet implemented. Where `AGENTS.md` states a **Direction (ADR-0001)** that differs from **Currently**, treat the current text as what the code does today and the direction as what new work must move toward. Do not describe unshipped migration phases as shipped.

## 3. Repository map

```text
index.html                  Static application shell, dialogs, templates, landmarks
main.js                     Ordered ES-module entry point and offline registration
server.mjs                   Dependency-free local static server
server.test.mjs              Static server behavior and traversal tests
app.js                      Canvas engine, workspace, JD, tasks/Today, AI, import/export
offline/register.js         Progressive Service Worker registration
sw.js                       Versioned same-origin application-shell cache
manifest.webmanifest        Install metadata, scope, colors, and local icons
icons/                      PWA icons, including a maskable-safe 512px asset
storage/life-store.js       SQLite Wasm initialization, migrations, repositories, snapshots
storage/vault-errors.js     Typed vault errors (path, parse, schema, conflict, storage)
storage/content-hash.js     Stable async SHA-256 content hashing
storage/vault-path.js       Cross-platform vault path normalization and entity paths
storage/frontmatter.js      Preservation-first frontmatter scan/parse/patch codec
storage/entity-codec.js     Canonical task/habit/journal/event Markdown codecs
storage/vault-store.js      VaultStore adapter contract and media-type inference
storage/memory-vault.js     In-memory vault adapter (tests): conflicts, journal, snapshot
storage/indexeddb-vault.js  IndexedDB vault adapter (browser default; browser-verified)
storage/fs-vault.js         Node filesystem vault adapter (real-folder reference + Node tooling)
storage/life-indexer.js     Projects canonical vault files into the index (rebuild/reconcile)
storage/memory-index.js     In-memory index port (reference implementation + tests)
storage/canvas-validate.js  Shared JSON Canvas 1.0 structural validator (isCanvas)
storage/workspace-vault.js  Canonical sidecar + per-canvas .canvas persistence on a vault
storage/workspace-backup.js Version-2 whole-space .orbit.json export/import (plan §15)
storage/task-repository.js  File-canonical task repository (create/update/complete + placements)
storage/task-migration.js   Legacy marker-task -> canonical file + placement migration (plan §16)
storage/habit-repository.js File-canonical habit repository (definitions + daily check-in log)
storage/journal-event-repository.js  Journal (per-date) and calendar-event repositories
storage/index-integrity.js  Index integrity audit + purge/rebuild recovery (plan §10/§21)
styles/layers.css           Global named cascade-layer order
styles/tokens.css           Balaur primitive, semantic, component, and motion tokens
styles/foundation.css       Reset, focus, platform preference adaptations
styles/shell.css            Carved-oak top bar, library, application shell
styles/canvas.css           Camera, parchment cards, edges, portals, minimap
styles/components.css       Inspector, Today, Balaur, dialogs, shared components
styles/themes.css           Validated canvas theme presets
styles/responsive.css       Narrow-shell and content-container adaptations
styles/motion.css           Tokenized transitions, keyframes, View Transitions
docs/                       Architecture and subsystem documentation
widgets/                    Sandboxed HTML file-node widgets
vendor/pixel-loom/          Self-hosted fonts and upstream design-system provenance
vendor/sqlite/              Official SQLite Wasm module, binary, license, provenance
.github/workflows/pages.yml Deploys the repository root as a static GitHub Pages artifact
```

Do not edit vendored `.wasm`, `.mjs`, or font binaries as if they were application source. When intentionally updating a vendor, update its version, license/provenance, file list, and checksums together.

`plans/` holds implementation plans and reviews; `docs/adr/` holds architecture decision records. All ten phases (foundation) of the file-canonical migration (ADR-0001) are implemented as the `storage/vault-*.js`, `storage/frontmatter.js`, `storage/entity-codec.js`, `storage/content-hash.js`, `storage/canvas-validate.js`, `storage/life-indexer.js`, `storage/memory-index.js`, `storage/workspace-vault.js`, `storage/workspace-backup.js`, `storage/task-repository.js`, `storage/task-migration.js`, `storage/habit-repository.js`, `storage/journal-event-repository.js`, `storage/fs-vault.js`, and `storage/index-integrity.js` library modules, with Node test suites `storage/phase1.test.js` through `storage/phase10.test.js` (run via `node --test`, 149 tests). `storage/life-store.js` carries schema version 2 (the rebuildable index tables). Three vault adapters exist — `MemoryVault` (tests), `IndexedDbVault` (browser default), and `FsVault` (Node filesystem reference) — sharing one logical layout and optimistic-concurrency contract. The task, habit, journal, and calendar-event repositories are the file-canonical backend; `storage/task-migration.js` converts legacy marker tasks; `storage/index-integrity.js` audits the derived index against the vault and supports purge-and-rebuild recovery. The browser File-System-Access/Tauri adapters, the FTS5 search index, the cold-indexing performance worker, and — above all — wiring the app's UI/Today to these repositories and running the migration against a real profile are the remaining browser-verified steps.

Phase 4b is wired into `app.js`: the canvas application now boots **vault-first** through an asynchronous `bootCanvasApp()` (plan §14.1). On load it reads the workspace sidecar and canvas documents from the canonical IndexedDB vault (`storage/indexeddb-vault.js`), migrating the legacy localStorage workspace on first run, and every save is written back to the vault. If the vault is unavailable or unreadable the boot falls back to the synchronous localStorage load, so the app still starts (progressive enhancement). localStorage is now a fallback/migration mirror pending retirement. `storage/indexeddb-vault.js`, the life-store index port, and the vault-first boot require browser verification (including the §18 first-render budget); the Node tests exercise the same persistence logic against `MemoryVault`.

## 4. Non-negotiable data boundaries

The target ownership model (ADR-0001) is:

```text
JSON Canvas files (.canvas)   canonical spatial documents
Markdown files (.md)          canonical human-readable life entities
.orbit/workspace.json         application-only hierarchy and UI metadata
SQLite                        disposable, rebuildable query/search index
```

These boundaries remain non-negotiable; the migration changes *where* life state is canonical (Markdown files) and demotes SQLite to a rebuildable projection. Sections below mark **Direction (ADR-0001)** versus **Currently** where they differ.

### 4.1 JSON Canvas owns portable spatial content

A canvas document must remain shaped like:

```json
{
  "nodes": [],
  "edges": []
}
```

Use only standard JSON Canvas node types:

- `text`
- `file`
- `link`
- `group`

Do not add custom node types such as `task`, `canvas`, `widget`, `ai`, or `habit`. Do not add application-only fields to nodes or edges merely because Balaur can read them. Use standard IDs, geometry, colors, content fields, and edge routing fields. Life entities are placed through standard `file` nodes that reference their canonical `.md` files (Direction, ADR-0001); a canvas node ID is placement, not entity identity.

Before accepting imported, restored, or model-generated documents, route them through `isCanvas()` or the stricter operation/workspace validators already in `app.js`. Maintain globally unique node/edge IDs within a document and valid edge endpoints.

### 4.2 Portable capabilities use standard nodes plus inert markers

Balaur recognizes special behavior without changing the JSON Canvas schema.

**Direction (ADR-0001):** life entities are canonical Markdown files, each carrying an immutable `orbit-id` in constrained frontmatter, placed on canvases through standard `file` nodes:

```md
<!-- orbit:jd 11.01 -->
<!-- orbit:ai-card -->
<!-- orbit:habit-entry id=... habit=... status=done value=1 at=... -->
```

- Johnny Decimal item notes are standard text nodes.
- Tasks, habits, journals, and calendar events are canonical `.md` files under `tasks/`, `habits/`, `habit-logs/`, `journal/`, and `events/`, each with an immutable `orbit-id`. One entity may have zero, one, or many `file`-node placements; a canvas node ID is placement, not entity identity.
- AI operators are standard text nodes whose incoming edges provide context. When an input is a `file` node, AI context assembly resolves the referenced canonical file's body, not its path.
- Live HTML/Canvas/WebGL widgets are standard `file` nodes pointing to `.html` files.
- Nested-canvas portals are standard `file` nodes pointing under `canvases/`.

**Currently:** tasks are marker-bearing text nodes (`<!-- orbit:task task-id -->`) linked to SQLite by a stable marker ID. Legacy markers remain accepted during migration but are not generated for new file-canonical entities.

Markers must remain harmless and readable in editors that do not understand Balaur. Markdown rendering intentionally hides `<!-- orbit:... -->` compatibility lines.

### 4.3 The workspace sidecar owns hierarchy and canvas UI state

The browser workspace is stored under `orbit-workspace-v1`. Its canvas records own:

- canvas titles and paths
- parent/portal relationships
- active canvas
- per-canvas cameras
- Johnny Decimal index metadata
- independent JSON Canvas documents (**currently** embedded in the record; **Direction (ADR-0001):** stored as separate `.canvas` files, with the sidecar holding metadata only)

Do not put hierarchy, camera position, active filters, selection, or Johnny Decimal indexes into exported `.canvas` documents. The legacy `orbit-canvas-v1` and `orbit-title` keys are retained for migration/root compatibility; do not make them a new source of truth.

Every non-root canvas must still be reachable through a standard parent file node and remain independently exportable.

**Direction (ADR-0001):** each canvas document becomes an independently valid `.canvas` file under `canvases/`, and the sidecar (`.orbit/workspace.json`) keeps only metadata — not full documents. A single-canvas `.canvas` export remains valid JSON Canvas 1.0, but `file` nodes that reference canonical entity files may dangle outside a whole-space export; never claim a bare `.canvas` is a complete entity backup.

### 4.4 Canonical files own life state; SQLite is a rebuildable index

**Direction (ADR-0001):** canonical Markdown files own the operational fields that JSON Canvas 1.0 does not define:

- task status, priority, scheduling, due dates, completion, recurrence, estimates
- habit definitions and immutable daily check-in events
- journal date indexes
- calendar event times and timezones

SQLite is a disposable projection over those files. It continues to power Today, calendar ranges, habit streaks, search, sorting, and filtering, but deleting it must not delete meaningful user data: every portable row is rebuildable from `.md` and `.canvas` files. Projected rows link back through `source_path` + `entity_id`; canvas placement is a derived `entity_placements(canvas_id, node_id, entity_id)` table, not identity. SQLite must not become a second owner of full visible documents or node geometry, and task workflow state lives in frontmatter, never in custom canvas node fields.

**Currently:** SQLite (schema version 1) is the authoritative owner of this state and rows link back through `canvas_id + node_id`. Migration 2 introduces the source-file/index tables and demotes SQLite to a projection; migration 1 must remain byte-for-byte compatible.

Date conventions are mandatory:

- local dates: `YYYY-MM-DD`
- instants: ISO 8601 strings
- calendar timezones: IANA timezone names
- scheduling intent (`scheduled_on`) stays separate from a deadline (`due_on`)

### 4.5 Whole-space backups are normalized JSON

`.balaur.json` is the portable workspace backup. It contains the workspace/canvases plus normalized `lifeData` tables while retaining the internal `orbit-workspace` discriminator for backward compatibility. Never embed or export a raw SQLite database binary. Backup data must remain rebuildable across kvvfs, future OPFS, IndexedDB fallback, and native SQLite implementations.

**Direction (ADR-0001):** the version-2 backup contains the sidecar plus the logical vault files (`.canvas` and `.md`) rather than a SQLite snapshot; version-1 bundles remain importable and are migrated on import.

Single `.canvas` export/import remains focused on the active standards-compliant document.

## 5. Runtime and initialization model

The module graph rooted at `main.js` is deliberate.

**Currently:**

1. `app.js` evaluates first as a strict ES module, loads/normalizes the workspace synchronously from `localStorage`, renders the UI, and exposes `window.orbitCanvas`.
2. `storage/life-store.js` loads SQLite Wasm, indexes the workspace, exposes the store, and dispatches a readiness event.
3. `offline/register.js` progressively registers the same-origin Service Worker and exposes `window.orbitOfflineReady`.

**Direction (ADR-0001):** startup becomes asynchronous and vault-first:

1. initialize the `VaultStore` (IndexedDB default adapter);
2. load and normalize the workspace sidecar;
3. preload the active canvas document and metadata for its visible file nodes;
4. initialize the canvas application and render from an in-memory working set;
5. initialize SQLite LifeStore as a derived index;
6. reconcile the file index by vault revision (warm) or bounded-batch rebuild (cold);
7. progressively register the offline Service Worker.

Rendering must not perform per-card asynchronous file reads: preload visible file-node content when switching canvases and render from the in-memory cache, refreshing nodes as reads finish. This async refactor is the largest cross-cutting cost of the migration (see the plan and review); do not land it implicitly inside an unrelated feature.

Life-store initialization is asynchronous:

```js
const life = await window.orbitLifeReady;
```

The promise resolves to `null` on initialization failure rather than rejecting out of the module. Code that requires SQLite must handle an unavailable store and give the user a useful message. Synchronous rendering paths may use `window.orbitLifeStore` only after checking it exists.

Relevant integration points are:

- `window.orbitLifeReady`
- `window.orbitLifeStore`
- `orbit:life-store-ready`
- `orbit:life-store-error`
- `reconcileTaskMarkers()`
- `refreshLifeViews()`

Do not split this into independently ordered script tags, and do not reintroduce a synchronous whole-workspace `localStorage` read as the startup source of truth, without replacing this startup contract. Until the vault-first startup ships, preserve the current ordered synchronous contract above.

### 5.1 Offline application-shell rules

`sw.js` precaches every asset required for the shell, locally hosted fonts, SQLite Wasm, icons, and the sample widget. Same-origin GET requests use the network when available and fall back to the cache; cross-origin requests and non-GET requests are not intercepted.

- Keep `APP_SHELL` synchronized when adding, moving, or removing a required runtime asset.
- Increment `CACHE_NAME` when changing cache semantics or when an old cache must be invalidated immediately.
- Never cache AI-provider requests, API keys, generated exports, `blob:` URLs, or arbitrary cross-origin resources.
- Keep `start_url`, Service Worker scope, and all cached paths relative so GitHub Pages subpath deployment continues to work.
- Treat the Service Worker as a resilient shell, not the owner of workspace or life data. User data remains in the workspace store and SQLite.
- Registration must stay progressive: unsupported or insecure contexts still run the online application.
- Do not call `skipWaiting()` and reload an active editing session without an explicit, user-safe update flow.
- New storage/offline strategies require tests for first install, controlled reload, offline reload, reconnection, and cache upgrade.

## 6. Canvas/workspace mutation rules

`documentData` represents the active canvas document. `workspace.canvases[currentCanvasId]` is the corresponding record. Keep both coherent.

When changing the active canvas:

1. mutate the document or workspace through existing helpers;
2. call `scheduleSave()` for normal debounced persistence;
3. render only the affected projections when practical;
4. ensure the active SQLite canvas index is refreshed by persistence.

When changing a non-active canvas, `persistWorkspace()` only indexes the active record. Explicitly call `lifeStore.syncCanvasRecord(record)` for the non-active document, as `createTask()` does.

Before switching canvases, preserve the current document and camera with `saveCurrentCanvasState()`. Use `switchCanvas()`, `enterSubcanvas()`, `leaveSubcanvas()`, and `revealWorkspaceNode()` instead of hand-updating navigation globals.

Keep destructive hierarchy operations recursive and clean all related sidecar entries. If a task node is deleted, delete its task row. If a new import or typed AI operation can introduce markers, run marker reconciliation after it commits.

Do not mutate exported documents with transient DOM/UI state. Selection, drag previews, filters, current tool, dialogs, runtime AI status, and app view are runtime-only.

## 7. SQLite and migration rules

`storage/life-store.js` currently uses official SQLite Wasm `3.53.0` with the `:localStorage:` kvvfs backend. This is an acknowledged GitHub Pages-compatible prototype backend, not the desired final persistence architecture.

- Do not add a second ad hoc localStorage database.
- Do not store a raw SQLite binary in a normal localStorage value or backup.
- Do not deepen coupling to kvvfs-specific behavior.
- Keep repository APIs replaceable by Worker+OPFS and native SQLite adapters.
- Do not assume OPFS worker support on GitHub Pages; official OPFS requires COOP/COEP headers that Pages cannot set.

Schema changes require an actual migration:

1. increment `SCHEMA_VERSION`;
2. add a new `if (version < N)` transaction;
3. retain the released version-1 migration so existing databases upgrade correctly;
4. update indexes and constraints deliberately;
5. update `TABLES` and snapshot import/export if a table is portable life data;
6. update `docs/life-data.md` and backup examples;
7. test both a fresh database and an upgrade from a persisted older profile.

Do not simply rewrite migration 1 after release. `PRAGMA user_version` is the compatibility contract.

Use transactions for multi-row or multi-table writes. Parameterize values; never concatenate user content into SQL. Preserve the current empty-bind distinction because the SQLite Wasm APIs do not accept every empty-bind call form uniformly.

Repository methods expose application-facing camelCase for tasks while SQL snapshots use database column names. Keep that boundary explicit. New adapters should match the application-facing contract rather than leaking VFS details into components.

Activity entries should describe meaningful state transitions. Avoid logging render-time reads or repeated no-op reconciliations.

## 8. Task and temporal feature rules

**Direction (ADR-0001):** task creation is a file-first operation:

1. generate an immutable `orbit-id` and a safe, stable path;
2. write the canonical task Markdown file with an expected-content-hash precondition;
3. add a standard `file` node placement to the chosen canvas;
4. persist the affected canvas;
5. index the task and its placement into SQLite;
6. refresh Canvas/Today projections.

A portable task looks like:

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

Deleting a canvas `file` node removes only that placement; the entity and its other placements survive. Deleting the entity everywhere is a separate confirmed action that removes every placement, then the canonical file, then reindexes. A task with no placement remains available in Inbox/search.

**Currently:** task creation is a two-layer operation — append a marker-bearing text node (`<!-- orbit:task task-123 -->`) to a canvas and upsert its metadata into SQLite, then sync the canvas index, refresh projections, and persist the workspace. This remains the implemented behavior until the file-canonical task slice (plan Phase 5) and migration (Phase 6) ship.

Do not treat Markdown checklist items and marker-backed task cards as interchangeable without an explicit reconciliation design. Do not infer a due date from a scheduled date or vice versa.

Habits should remain event logs, not recurring tasks. Daily habit history should be preserved rather than overwritten into a current counter. Journals and calendars are projections over the same life layer; avoid creating disconnected feature-specific stores.

For local-date behavior, test around timezone boundaries. Do not derive a local `YYYY-MM-DD` by blindly slicing a UTC timestamp.

## 9. Johnny Decimal and nested-canvas rules

Johnny Decimal is a projection over the nested-canvas hierarchy:

- root → area (`10-19`)
- area → category (`11`)
- category → item (`11.01`)

Preserve validation in `validateJDCode()`:

- area ranges align to a decade and span ten numbers;
- categories fall inside their parent range;
- items are scoped to the parent category;
- duplicate IDs are rejected;
- ordering is numeric, not lexicographic.

Area, category, and canvas-item entries are file-node portals. Note-format items use an inert JD marker. Their sidecar entry and visible heading should remain synchronized when edited. A whole hierarchy must survive reload, single-level navigation, whole-space export/import, and deletion.

Do not flatten nested documents into one canvas during save or export.

## 10. AI and widget security boundaries

AI output must never directly mutate the host DOM or execute arbitrary host-page JavaScript.

For host canvas changes:

- accept only allowlisted structured operations;
- validate IDs, field patches, URLs, geometry, operation counts, and resulting documents;
- show a human-readable proposal;
- require confirmation before application;
- apply through `validateCanvasOperations()` and `applyCanvasOperations()` or their successors.

AI operators remain marker-bearing text nodes. Incoming edges are inputs; `AI output` edges identify generated output. Preserve debouncing, queued reruns, stable output-node reuse, and cycle detection.

Live widgets run in file-node iframes using:

```html
sandbox="allow-scripts"
```

Do not add `allow-same-origin`, top navigation, popups, forms, downloads, device access, or direct filesystem access without a documented security review. Resolve file paths with `safeFileURL()` and reject absolute paths, schemes, and traversal.

Use direct WebGL2 for GPU-rendered widgets rather than Three.js or another scene-graph dependency. GPU work must remain progressive and bounded:

- provide a useful HTML/CSS, SVG, or Canvas fallback when WebGL is unavailable;
- cap device-pixel ratio and resize through `ResizeObserver`;
- pause animation when hidden or outside the viewport;
- honor `prefers-reduced-motion`;
- handle `webglcontextlost` and recovery;
- release buffers/programs and cancel animation frames when a widget is torn down;
- prefer event-driven rendering over a permanent animation loop for static scenes;
- use WebGPU only as an optional capability with a WebGL/2D fallback, never as the sole renderer.

Do not load graphics engines or shader code from a CDN. Keep widget code self-contained so it works offline and inside the sandbox.

Escape all dynamic HTML with `escapeHTML()` or assign `textContent`. Pass external links through `safeURL()`, keep `target="_blank"` paired with `rel="noreferrer"`, and do not interpolate untrusted content into style/HTML/SQL without validation.

Provider keys belong in `sessionStorage` by default. `localStorage` is allowed only after the user explicitly enables **Remember API key**. Never commit keys, log them, include them in exports, or send them anywhere except the selected provider endpoint.

## 11. UI, CSS, and accessibility conventions

Use browser standards and native controls:

- semantic landmarks (`header`, `nav`, `main`, `aside`, `article`, `section`)
- native forms, labels, buttons, and `<dialog>`
- Pointer Events for mouse/touch canvas interaction
- SVG for edges
- `inert` plus `aria-hidden` for hidden interactive panels
- useful accessible names on icon-only buttons
- keyboard and reduced-motion behavior where relevant

Do not introduce React, Vue, Svelte, a utility-CSS framework, CSS-in-JS, or a build pipeline for an isolated UI change.

CSS uses the explicit layer order in `styles/layers.css`:

```css
@layer foundation, shell, canvas, components, themes, responsive;
```

Place rules in the file/layer matching their responsibility. Do not create an unlayered override file or reintroduce the old monolithic `styles.css`. Prefer semantic component classes and existing design tokens over literal colors or utility-like class chains.

Use modern CSS as progressive enhancement rather than rebuilding layout or interaction in JavaScript:

- container queries for component layout and media queries for application-shell viewport changes;
- logical properties for new direction-agnostic layout rules;
- fluid sizing with `clamp()` when bounded scaling is useful;
- semantic custom properties and `color-mix()` for theme relationships;
- `:focus-visible`, native control styling, forced-colors support, and `prefers-reduced-motion`;
- low-specificity `:where()` selectors when reusable components need easy overrides;
- `@supports` when a newer feature needs a meaningful fallback.

Check target-browser support before adopting experimental syntax. The baseline must remain usable when a progressive enhancement is unavailable.

The default visual language is Balaur Cartographer's Tavern:

- carved oak for application furniture
- parchment for canvas documents and Today ledgers
- candle gold for primary actions and selection bearings
- river teal for focus, links, and assistant state
- crisp borders and shallow, functional shadows
- Newsreader for identity and canvas titles, Work Sans for text, JetBrains Mono for controls and metadata
- tokenized motion for press, panel travel, selection entry, and nested-canvas navigation

JSON Canvas preset colors are document semantics and must remain distinct from application-theme colors. `styles/tokens.css` is the only runtime token source; vendored Pixel Loom color-token stylesheets are not loaded.

Whenever a desktop layout changes, inspect narrow viewport behavior in `styles/responsive.css`. Avoid controls that work only with hover; canvas interactions must remain usable with pointer/touch and keyboard where an established fallback exists.

## 12. JavaScript conventions

The current prototype intentionally has no transpilation or formatter. Application code runs as native strict ES modules through `main.js`. Match the surrounding vanilla JavaScript style and avoid mass-formatting `app.js` in an unrelated change.

- Prefer small cohesive modules and named helpers over duplicating state transitions.
- Keep module side effects explicit and preserve the ordered startup graph in `main.js`.
- Use feature detection and progressive enhancement instead of user-agent detection.
- Use `AbortController`, event delegation, and passive listeners where they simplify lifecycle or high-frequency input without changing behavior.
- Reuse `$`, `$$`, `clone`, `uid`, `escapeHTML`, and existing navigation/persistence helpers.
- Keep IDs stable and collision-resistant.
- Validate at import/AI/storage boundaries, not only in UI forms.
- Use optional chaining only where absence is expected; do not hide genuine invariant failures indiscriminately.
- Keep asynchronous UI actions disabled while running and surface errors in the relevant dialog/status region.
- Stop pointer propagation on controls embedded inside draggable cards.
- Preserve native dialog cancellation and focus behavior.
- Avoid introducing global state when a value can be derived from `workspace`, `documentData`, or `LifeStore`.

`app.js` currently exposes a small test/integration surface through `window.orbitCanvas`, including document/workspace getters, operation validation/application, AI-card execution, nested-canvas/JD navigation, task creation, view switching, and export. Extend this API only when browser integration or external embedding benefits from a stable command; do not expose raw mutable internals.

**Direction (ADR-0001):** as repositories become file-first and asynchronous, this surface gains placement commands (for example `addPlacement`, `removePlacement`) and its mutation methods become async; raw SQLite mutation must not be used by components once file-canonical mode is enabled.

## 13. Running and validating changes

There is currently no package manifest and no automated test runner. Do not add `npm install` as a prerequisite just to run the static app.

Serve the repository over HTTP; do not test via `file://` because ES modules, Wasm loading, and browser storage behavior differ:

```bash
python3 -m http.server 4173
```

Open <http://localhost:4173/>.

At minimum, run these static checks after JavaScript or repository edits:

```bash
node --check main.js
node --check app.js
node --check offline/register.js
node --check storage/life-store.js
node --check sw.js
git diff --check
```

Then perform browser-level checks appropriate to the change. Use a fresh temporary browser profile for first-run behavior and retain the same profile for reload/persistence tests. Headless Chrome with a temporary `--user-data-dir` and DevTools Protocol is acceptable; manual testing is also acceptable.

### Baseline browser smoke test

Verify:

1. the app loads over HTTP with no uncaught console errors or failed local assets;
2. the manifest, Service Worker, local fonts, SQLite module, and Wasm load without failed assets;
3. the sidebar reports `SQLite <version> · local` rather than an unavailable database;
4. the starter workspace renders and nested canvases open/return correctly;
5. pan, zoom, select, drag, resize, connect, and inspector basics still work;
6. Canvas/Today view switching works;
7. creating and completing a task updates both the canvas card and Today;
8. a controlled reload preserves workspace, camera, task state, and SQLite rows;
9. after one online load, an offline reload renders the shell, SQLite, fonts, and local widget;
10. current `.canvas` export remains accepted by `isCanvas()`;
11. whole-space export/import restores hierarchy and normalized life data;
12. reset/import destructive paths are tested only in a disposable profile.

For changes to hierarchy, storage, import/export, or AI operations, test malformed input as well as the happy path. For CSS changes, inspect at least one wide desktop viewport and one narrow viewport. For persistence changes, test both a clean profile and an existing profile created before the change.

Useful runtime probes in browser DevTools include:

```js
await window.orbitLifeReady
window.orbitLifeStore.stats()
window.orbitCanvas.getDocument()
window.orbitCanvas.getWorkspace()
window.orbitCanvas.getSummary()
```

Do not claim persistence or Wasm behavior was tested if only `node --check` was run.

## 14. Documentation expectations

Update documentation in the same change when behavior or architecture changes:

- user-visible capability/control → `README.md`
- ownership, module boundaries, or delivery direction → `docs/architecture.md`
- schema, repository API, migrations, storage, backup shape → `docs/life-data.md`
- app-shell caching, Service Worker lifecycle, manifest, offline behavior → `docs/offline.md`
- AI operations, widgets, provider/security model → `docs/generative-canvas.md`
- tokens, typography, layers, component visual rules → `docs/design-system.md`
- vendored asset/version update → the relevant `vendor/*/README.md` and licenses
- an architecture decision → a new record in `docs/adr/` (see `docs/adr/0001-file-canonical-life-data.md`)

Keep documentation explicit about what is implemented versus proposed. Do not describe future OPFS, Tauri, command-stack, recurrence, calendar, or sync work as shipped. The file-canonical migration (ADR-0001) is the adopted direction but is not yet implemented; mark its stages as Direction/Currently until each phase ships, and update `docs/architecture.md` and `docs/life-data.md` in the same change as the behavior.

## 15. Git and deployment hygiene

- Keep changes scoped; do not rewrite unrelated parts of the monolithic prototype.
- Do not commit generated browser profiles, screenshots, logs, API keys, local databases, or temporary exports.
- Do not amend, reset, or discard user work unless explicitly requested.
- Commit and push only when the user asks or the active task clearly includes deployment.
- Every push to `main` triggers `.github/workflows/pages.yml` and deploys the repository root.
- If deployment is part of the task, verify the GitHub Pages workflow and then smoke-test the live URL with a fresh profile.

## 16. Definition of done

A change is complete only when all applicable statements are true:

- JSON Canvas documents remain standards-compliant and independently portable.
- Workspace, Canvas, and SQLite ownership boundaries remain clear.
- File-canonical work follows ADR-0001: canonical files are written before the derived index, migrations are staged and reversible, and Direction/Currently documentation stays accurate.
- Existing profiles migrate or degrade safely.
- Imports, exports, reset, and reload do not orphan related state.
- Dynamic content is escaped and security boundaries are preserved.
- Life-store failures are handled without breaking the canvas.
- Wide and narrow interfaces remain usable and semantically structured.
- Static checks pass.
- Relevant browser behavior and persistence were exercised.
- Documentation matches the implemented behavior.
- No unrelated dependencies, build outputs, credentials, or vendor changes were introduced.

When requirements conflict, prefer data portability, user recoverability, and explicit validation over convenience. Balaur should remain useful when opened by another JSON Canvas editor, when the AI provider is absent, or when the optional life database is temporarily unavailable.
