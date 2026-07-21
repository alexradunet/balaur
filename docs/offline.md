# Offline-first application shell

Balaur is an installable, progressively enhanced static web application. Offline support uses browser standards and caches only deployable shell resources. User files remain in the vault and are never placed in the Service Worker cache.

## Runtime pieces

- `manifest.webmanifest` describes the installable app with relative scope, colors, and local icons.
- `main.js` is the ordered ES-module entry point.
- `offline/register.js` registers the Service Worker only in a supported secure context.
- `sw.js` maintains the versioned `orbit-shell-v7` cache.
- `IndexedDbVault` stores canonical user files in IndexedDB; the Service Worker does not intercept or cache those records.

Registration failure does not prevent the online application from starting. Service Workers are available on HTTPS and localhost.

## Cache strategy

The install handler precaches the application shell in `orbit-shell-v7`:

- `./`, `./index.html`, `./manifest.webmanifest`, `./main.js`, `./app.js`;
- `./offline/register.js`;
- the runtime storage modules: `indexeddb-vault.js`, `life-indexer.js`, `life-query.js`, `memory-index.js`, `task-repository.js`, `workspace-backup.js`, `workspace-vault.js`, `canvas-validate.js`, `vault-store.js`, `vault-path.js`, `vault-errors.js`, `content-hash.js`, `frontmatter.js`, and `entity-codec.js`;
- all cascade-layer stylesheets under `styles/`;
- self-hosted Pixel Loom fonts;
- `widgets/focus-orbit.html`; and
- the manifest icons.

There is no SQLite Wasm in the application shell. The browser runtime uses the pure-JavaScript in-memory index over canonical files. A persistent index is a deferred optimization; OPFS-backed SQLite Wasm would require COOP/COEP headers unavailable on GitHub Pages.

Allowlisted shell GET requests are network-first. Successful basic responses refresh the current cache; network failures fall back to the matching cached response, and navigations can fall back to `./index.html`. Arbitrary same-origin paths and requests carrying `Authorization` are not intercepted. Navigation Preload is enabled when available.

Balaur deliberately does not intercept:

- non-GET requests or byte-range requests;
- AI-provider calls, including same-origin requests carrying `Authorization`;
- generated `blob:` downloads;
- API keys or other browser-storage values; or
- arbitrary external resources.

The Cache API contains application resources, not user data.

## User data remains separate

```text
Cache API                  application code, styles, fonts, icons, and widgets
IndexedDB vault            canonical .canvas, .md, and sidecar files; boot source
MemoryIndex                disposable in-memory query projection rebuilt from vault files
sessionStorage             provider key by default
localStorage               one-time first-run migration input only; not source of truth
```

Whole-space `.orbit.json` export/import is the portable backup for the sidecar and logical vault files. It contains no database snapshot or provider key. A single `.canvas` export remains valid JSON Canvas but may reference entity files that are not included. Clearing site data removes the shell cache and IndexedDB user data; it cannot be treated as a backup.

## Updates

`CACHE_NAME` is the cache-format version and is currently `orbit-shell-v7`. Increment it when cache semantics change or old entries must be invalidated immediately. The Service Worker removes older `orbit-shell-*` caches during activation. It does not call `skipWaiting()` or reload an active editing session; a future immediate-update flow must ask the user, persist pending work, activate the waiting worker, and reload deliberately.

When a required runtime asset is added, moved, or removed, update `APP_SHELL` in the same change. Keep every URL relative so localhost and GitHub Pages subpath deployment use the same paths.

## Offline behavior and limits

The Service Worker wiring and shell list are present, but a real-browser offline reload remains browser-pending. After one successful online install and control, the intended behavior is that the shell, local fonts, local widget, storage modules, and UI load without a network connection. Canonical files remain in IndexedDB and the in-memory query projection is rebuilt at boot.

Network-dependent behavior remains unavailable offline:

- remote AI generation and provider connection tests;
- external links;
- widgets that intentionally request remote resources; and
- future remote sync or calendar-provider adapters.

Task editing, Today, import, and export are designed to use canonical local files, but their complete browser behavior must be verified in a real profile rather than inferred from Node tests.

## Validation checklist

1. Start from a clean browser profile over localhost or HTTPS.
2. Wait for `navigator.serviceWorker.ready` and verify the page is controlled.
3. Confirm the `orbit-shell-v7` cache contains every `APP_SHELL` asset and no database Wasm.
4. Verify IndexedDB vault creation, canonical file writes, and a controlled reload.
5. Create and complete a task, then verify the Canvas and Today projections after reload.
6. Export and import a version-2 whole-space file bundle in a disposable profile.
7. Enable browser offline emulation and reload without bypassing the Service Worker.
8. Verify the shell, fonts, storage modules, Today, and local widget load offline.
9. Reconnect and verify same-origin resources refresh normally.
10. For cache-version changes, test upgrade behavior from the previous active worker.

Useful probes:

```js
await window.orbitOfflineReady
await navigator.serviceWorker.ready
await caches.keys()
(await caches.open("orbit-shell-v7")).keys()
await window.orbitVaultReady
window.orbitVaultStore
window.orbitCanvas.getDocument()
window.orbitCanvas.getWorkspace()
```

Node tests verify the platform-neutral vault, codecs, repositories, in-memory index, backup validation, and query layer. They do not verify IndexedDB durability, Service Worker control/cache upgrades, UI task flows, export/import round-trips, offline reload, or browser timezone boundaries.
