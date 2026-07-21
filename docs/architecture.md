# Balaur architecture

Balaur is a standards-first, local-first web application with no UI framework, build step, package install, or runtime dependency. Native ES modules, the DOM, CSS, Pointer Events, SVG, Canvas, WebGL, IndexedDB, and Service Workers provide the platform. The static site can run on GitHub Pages and is also structured around adapters that can support Node tooling and a future desktop shell.

## Ownership model

The vault is the source of truth:

```text
IndexedDbVault (browser) / FsVault (Node) / MemoryVault (tests)
  ├─ .orbit/workspace.json      hierarchy, cameras, JD metadata
  ├─ canvases/*.canvas           independent JSON Canvas 1.0 documents
  ├─ tasks/*.md                  canonical task entities
  ├─ habits/*.md                 canonical habit definitions
  ├─ habit-logs/YYYY/*.md        append-only daily habit events
  ├─ journal/YYYY/*.md           canonical journal entries
  ├─ events/*.md                 canonical calendar events
  └─ widgets/*.html              sandboxed file-node widgets
```

JSON Canvas owns portable spatial content: nodes, geometry, edges, groups, links, and standard file references. The workspace sidecar owns hierarchy, canvas paths, cameras, active canvas, and Johnny Decimal metadata; none of that application state is added to exported Canvas documents.

Markdown frontmatter and body own life-management fields that JSON Canvas does not define. An entity's immutable `orbit-id` is its identity, a path is its locator, and a canvas node ID is a placement. One entity may have zero, one, or many standard `file`-node placements.

`LifeIndexer` projects the vault into `MemoryIndex`, and `LifeQuery` is the app-facing read facade for Today, task status, habits, journals, and event ranges. The index is a disposable runtime projection: it is rebuilt from files at boot, reconciled after vault changes, and may be discarded without data loss. Repositories always write canonical files before reindexing them.

A persistent index, including SQLite, is a deferred future optimization rather than part of canonical-files-only v1. OPFS-backed SQLite Wasm would require COOP/COEP response headers that GitHub Pages cannot provide; the pure-JavaScript in-memory projection is therefore the compatible default. No database is loaded by the browser application.

## Runtime startup

`main.js` imports the application module and progressively registers the Service Worker. The app's asynchronous vault-first boot is:

1. open `IndexedDbVault`;
2. load `.orbit/workspace.json` and each referenced `.canvas` file through `WorkspaceStore`;
3. on a genuinely empty first run, migrate the legacy localStorage workspace once into canonical vault files;
4. construct `MemoryIndex`, `LifeIndexer`, `LifeQuery`, and the task repository used by the shipped UI;
5. rebuild the in-memory projection from every vault file;
6. render the active workspace from the loaded working set; and
7. expose `window.orbitVaultReady`, `window.orbitVaultStore`, and the stable `window.orbitCanvas` integration surface.

After that one-time migration source is consumed, localStorage is not a source of truth or a persistence mirror. A vault failure is reported as unavailable canonical files; the application must not silently promote a localStorage workspace back to authority.

The wiring is present in `app.js`, but browser verification remains pending for IndexedDB durability, vault-first first render, reload behavior, and the UI's error path. The Node suites verify the platform-neutral storage logic against `MemoryVault` and `MemoryIndex`, not browser IndexedDB.

## Modules and boundaries

### Canvas and workspace

`app.js` owns the canvas interaction model, rendering, navigation, AI command flow, and UI state. `storage/canvas-validate.js` is the strict shared JSON Canvas validator. `storage/workspace-vault.js` persists a metadata-only sidecar plus one independently valid `.canvas` file per canvas. Invalid or missing canvas files become read-only repair placeholders and are never silently replaced with empty documents.

A single `.canvas` export is standards-compliant but can contain file references whose target `.md` files are not included. `storage/workspace-backup.js` provides the complete version-2 `.orbit.json` file bundle: sidecar metadata plus raw vault files. Import validates paths, canvases, references, entity parsing, duplicate IDs, and diagnostics in staging before activation.

### Life files and projections

`storage/frontmatter.js` performs constrained, preservation-first parsing and patching. It changes only Orbit-owned fields and preserves unknown keys, comments, ordering, BOM, line endings, and body content. `storage/entity-codec.js` defines the task, habit, habit-log, journal, and calendar-event contracts and validates dates, instants, enums, weekdays, and IANA timezones.

`FileTaskRepository`, `FileHabitRepository`, `FileJournalRepository`, and `FileEventRepository` are asynchronous canonical-file repositories. `storage/life-indexer.js` parses all supported vault files, projects typed rows and placements into `MemoryIndex`, detects malformed files and duplicate identities, and supports cold rebuild and warm revision reconciliation. `storage/index-integrity.js` audits the disposable projection against the files and can purge and rebuild it.

### Adapters

`VaultStore` defines asynchronous list/read/write/remove/move/stat/exists/snapshot/restore operations with hashes and revision changes. `IndexedDbVault` is the browser default; `MemoryVault` supplies deterministic tests; `FsVault` is the Node filesystem reference adapter with path, symlink, serialization, and atomic-write protections. Browser persistence and restore need real-browser verification; the adapter contracts and most behavior are Node-tested.

## JSON Canvas and nested canvases

Every canvas level is an independent JSON Canvas 1.0 document with only standard node types (`text`, `file`, `link`, `group`) and standard edge fields. A parent points to a child through a standard file node such as `canvases/11-finance.canvas`. Sidecar metadata supplies the parent, portal node, title, camera, and Johnny Decimal projection. Navigation can enter and leave nested canvases without flattening their documents.

Johnny Decimal is a validated hierarchy projection:

- root → area (`10-19`);
- area → category (`11`); and
- category → item (`11.01`).

Portals are ordinary file nodes. Simple item notes may carry harmless `orbit:jd` comments, but the comment is not the item identity; sidecar hierarchy and readable note content remain synchronized.

## AI and widgets

AI output is either a standard text-node addition or an allowlisted structured operation. The app validates IDs, URLs, geometry, operation counts, and the resulting document, presents a proposal, and requires confirmation. AI life changes must use canonical file repositories. When a connected input is a file node, context assembly resolves the canonical file body.

HTML and WebGL cards are standard `.html` file nodes rendered in iframes with `sandbox="allow-scripts"`. Widgets are self-contained, offline-friendly, bounded, reduced-motion aware, and never receive same-origin or filesystem access. Provider keys remain in sessionStorage by default and are never exported.

## Offline shell

The Service Worker caches only deployable same-origin shell resources under `orbit-shell-v7`: local modules, styles, fonts, icons, the manifest, and the sample widget. It does not cache IndexedDB records, provider calls, generated exports, or external resources. Network-first requests fall back to the shell cache when offline. See [offline.md](offline.md).

## Node-verified foundation and browser-pending work

The storage modules and the explicit phase test command pass 164 Node tests. Node verification covers codecs, path safety, vault adapters, workspace persistence, backup validation, repositories, indexing, queries, and integrity auditing. It does not prove browser IndexedDB behavior.

The following require a real browser profile: IndexedDB persistence and restore, vault-first startup and first-render timing, task create/complete/Today UI flows, export/import round-trip, offline reload and Service Worker upgrades, timezone boundaries in browser locale behavior, and malformed-file repair affordances. Documentation must label these as browser-pending until exercised.

## Future packaging, not v1 dependencies

The same adapter boundaries leave room for a Tauri shell, browser directory access, sync, workerized indexing, or a persistent index later. These are future options, not shipped runtime requirements. Any future index must remain rebuildable from canonical vault files and must not become a second owner of life state.
