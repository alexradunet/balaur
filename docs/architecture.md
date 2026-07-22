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
  ├─ cards/*.md                  canonical declarative component cards
  └─ widgets/*.html              canonical sandboxed widget source
```

JSON Canvas owns portable spatial content: nodes, geometry, edges, groups, links, and standard file references. The workspace sidecar owns hierarchy, canvas paths, cameras, active canvas, and Johnny Decimal metadata; none of that application state is added to exported Canvas documents.

Markdown frontmatter and body own life-management fields and declarative component-card fields that JSON Canvas does not define. An entity or component card's immutable `orbit-id` is its identity, a path is its locator, and a canvas node ID is a placement. One canonical file may have zero, one, or many standard `file`-node placements. Widget identity is its safe canonical `widgets/*.html` path.

`LifeIndexer` projects life entities into `MemoryIndex`, and `LifeQuery` is the app-facing read facade for Today, task status, habits, journals, and event ranges. ComponentCardCatalog and `WidgetCatalog` preload card/widget files and placement diagnostics for synchronous rendering; they do not become sources of truth. Every catalog and index is rebuilt from canonical vault files. Repositories write canonical files before reconciling projections.

A persistent index, including SQLite, is a deferred future optimization rather than part of canonical-files-only v1. OPFS-backed SQLite Wasm would require COOP/COEP response headers that GitHub Pages cannot provide; the pure-JavaScript in-memory projection is therefore the compatible default. No database is loaded by the browser application.

## Runtime startup

`main.js` starts guarded Custom Element registration, waits for that boundary to settle, and then imports the application module; Service Worker registration remains progressive. A component-definition fetch or evaluation failure is reported as a warning but does not block `app.js`, vault boot, canvas render, or canonical saves. In that failure mode, `app.js` feature-detects the missing definitions and supplies minimal native fallbacks: canvas and breadcrumb navigation, Today task content/actions, inspector fields/actions, delegated Add buttons, readable static component-card content, and explicitly inactive widget content with no iframe or source execution. The normal registered-component path is unchanged. The app's asynchronous vault-first boot is:

1. register the `balaur-*` presentation/runtime hosts, or continue after the guarded failure boundary;
2. open `IndexedDbVault`;
3. load `.orbit/workspace.json` and each referenced `.canvas` file through `WorkspaceStore`;
4. on a genuinely empty first run, migrate the legacy localStorage workspace once into canonical vault files;
5. construct `MemoryIndex`, `LifeIndexer`, `LifeQuery`, canonical repositories, and the component-card/widget catalogs;
6. rebuild the in-memory query and rendering projections from vault files;
7. render the active workspace from the loaded working set; and
8. expose `window.orbitVaultReady`, `window.orbitVaultStore`, and the stable `window.orbitCanvas` integration surface.

After that one-time migration source is consumed, localStorage is not a source of truth or a persistence mirror. A vault failure is reported as unavailable canonical files; the application must not silently promote a localStorage workspace back to authority. Fresh- and retained-profile browser checks exercise vault-first render, IndexedDB writes, controlled reload, canonical card/widget persistence, and whole-space restore. Quota/failure behavior and malformed-file repair affordances remain browser-pending.

## Modules and boundaries

### Canvas and workspace

`app.js` owns the canvas interaction model, rendering, navigation, AI command flow, and UI state. `storage/canvas-validate.js` is the strict shared JSON Canvas validator. `storage/workspace-vault.js` persists a metadata-only sidecar plus one independently valid `.canvas` file per canvas. Invalid or missing canvas files become read-only repair placeholders and are never silently replaced with empty documents.

A single `.canvas` export is standards-compliant but can contain file references whose target `.md` files are not included. `storage/workspace-backup.js` provides the complete version-2 `.orbit.json` file bundle: sidecar metadata plus raw vault files. Import validates paths, canvases, references, entity parsing, duplicate IDs, and diagnostics in staging before activation.

The main canvas renderer intentionally remains imperative. `app.js` owns geometry, camera transforms, selection, drag/resize/connect state, SVG edges, pointer shielding over live iframes, and the synchronous hot path. A measured `<balaur-canvas-node>` prototype was rejected because 100-node rerender, pan, and zoom p95 gates regressed. Extraction stops at bounded contents such as component cards and widget frames; there is no canvas-node Custom Element.

### Components and controllers

`elements/register.js` idempotently registers seven bounded hosts: Add menu, component card, widget frame, Today task list, workspace navigation, dialog frame, and inspector. Components receive immutable or replaceable view-model properties, reconcile keyed/stable DOM, retain native controls and landmarks, and emit bubbling composed `balaur-*` intents. Canvas-list navigation marks exactly the active button with `aria-current="page"`. Components do not own workspace mutation, repository writes, task/dialog submission state, selection, persistence debouncing, or navigation. `app.js` remains the controller: it validates intent details, performs canonical mutations, rebuilds/reloads projections, and supplies the next model.

The exception is the security runtime boundary: `<balaur-widget-frame>` owns only one reviewed widget execution instance—iframe construction, its private channel, lifecycle and resource limits, and cleanup. It still cannot mutate the host or vault; source loading and placement remain controller/repository responsibilities.

### Life files and projections

`storage/frontmatter.js` performs constrained, preservation-first parsing and patching. It changes only Orbit-owned fields and preserves unknown keys, comments, ordering, BOM, line endings, and body content. `storage/entity-codec.js` defines the task, habit, habit-log, journal, and calendar-event contracts and validates dates, instants, enums, weekdays, and IANA timezones.

`FileTaskRepository`, `FileHabitRepository`, `FileJournalRepository`, and `FileEventRepository` are asynchronous canonical-file repositories. `storage/life-indexer.js` parses all supported vault files, projects typed rows and placements into `MemoryIndex`, detects malformed files and duplicate identities, and supports cold rebuild and warm revision reconciliation. `storage/index-integrity.js` audits the disposable projection against the files and can purge and rebuild it.

### Component cards and widgets

`storage/component-card-codec.js`, `ComponentCardCatalog`, and `FileComponentCardRepository` own canonical `cards/*.md` files. The allowlisted recipes are metric, progress, callout, list, and timeline. Codec limits, immutable IDs, recipe-specific fields, path moves, optimistic hash checks, multiple placements, readable malformed-file fallbacks, and backup duplicate-ID validation are enforced outside the element. `<balaur-component-card>` is a declarative renderer and emits only an open intent.

`WidgetCatalog` and `FileWidgetRepository` own reviewed raw `widgets/*.html` files and standard file-node placements. Source validation is conservative pre-execution policy, not the sandbox itself. A create writes the canonical file first; if Canvas placement fails, the file and exact unfinished placement are retained for explicit recovery rather than recreated.

### Adapters

`VaultStore` defines asynchronous list/read/write/remove/move/stat/exists/snapshot/restore operations with hashes and revision changes. `IndexedDbVault` is the browser default; `MemoryVault` supplies deterministic tests; `FsVault` is the Node filesystem reference adapter with path, symlink, serialization, and atomic-write protections. Retained-profile and staging-vault browser checks exercise IndexedDB persistence and whole-space restore; quota/failure behavior remains browser-pending.

## JSON Canvas and nested canvases

Every canvas level is an independent JSON Canvas 1.0 document with only standard node types (`text`, `file`, `link`, `group`) and standard edge fields. A parent points to a child through a standard file node such as `canvases/11-finance.canvas`. Sidecar metadata supplies the parent, portal node, title, camera, and Johnny Decimal projection. Navigation can enter and leave nested canvases without flattening their documents.

Johnny Decimal is a validated hierarchy projection:

- root → area (`10-19`);
- area → category (`11`); and
- category → item (`11.01`).

Portals are ordinary file nodes. Simple item notes may carry harmless `orbit:jd` comments, but the comment is not the item identity; sidecar hierarchy and readable note content remain synchronized.

## AI and widgets

AI output is either a standard text-node addition or an allowlisted structured operation. The app validates plain-data shape, IDs, paths, URLs, recipe fields, geometry, operation counts, evolving canvas IDs, and the resulting document; it presents a deterministic proposal and requires confirmation. Shipped generated file operations are `component-card.create`, `component-card.update`, `widget.create`, and `widget.place`. They apply through canonical repositories. A successful file write whose Canvas placement fails is reported as recoverable durable state, and retry omits the already-completed write.

Generated widget source is shown in full before approval and remains inactive after it is saved. On explicit Run, `<balaur-widget-frame>` validates the source, builds a trusted document with CSP before generated content, and loads an opaque Blob URL into an iframe with exactly `sandbox="allow-scripts"`, `referrerpolicy="no-referrer"`, `loading="lazy"`, and an empty `allow` policy. A transferred `MessageChannel` uses closed version-1 schemas and cloned plain data; there is no host-global message command surface.

The shipped limits are 128 KiB source, 500 static elements, 64 KiB aggregate script, 64 KiB aggregate style, 64 KiB serialized message, six active widgets, 30 messages/second sustained with burst 60, and three missed five-second heartbeats. Pause, reload, source identity changes, disconnect, policy/schema/rate failures, and unexpected navigation destroy the channel/frame and revoke the Blob URL. Theme tokens, accessibility preferences, and visibility are the only host projections. No canonical content, provider secret, repository, host object, callback, mutation command, DOM access, same-origin capability, or filesystem capability is sent.

These controls are capability reduction and lifecycle containment, not hard CPU isolation. CSP and source validation block the supported resource channels and hostile probes; the self-navigation response is detection and teardown, not a claim that every request was suppressed. Provider keys remain in sessionStorage by default and are never exported.

## Offline shell

The Service Worker caches only deployable same-origin shell resources under `orbit-shell-v12`: local element/storage/AI/widget modules, styles, fonts, icons, the manifest, and the sample widget. It does not cache IndexedDB records, provider calls, generated exports, or external resources. Network-first requests fall back to the shell cache when offline. See [offline.md](offline.md).

## Verification boundary

The explicit storage foundation/query command passes 168 Node tests: the prior 164-test suite plus four component-card backup-boundary regressions. Focused suites cover component-card storage, generated operations, widget policy/protocol, and widget repositories. Real-browser checks cover fresh and retained vault-first profiles, task/Today and component boundaries, canonical card/widget reload persistence, whole-space staging restore, offline shell reload, wide/narrow layouts, hostile widget probes, and the retained imperative canvas invariants. IndexedDB quota/failure behavior, timezone boundaries in browser locale behavior, malformed-file repair affordances, and upgrade from a previously deployed Service Worker remain browser-pending.

## Future packaging, not v1 dependencies

The same adapter boundaries leave room for a Tauri shell, browser directory access, sync, workerized indexing, or a persistent index later. These are future options, not shipped runtime requirements. Any future index must remain rebuildable from canonical vault files and must not become a second owner of life state.
