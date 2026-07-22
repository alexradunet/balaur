# Offline-first application shell

Balaur is an installable, progressively enhanced static web application. Offline support uses browser standards and caches only deployable shell resources. User files remain in the vault and are never placed in the Service Worker cache.

## Runtime pieces

- `manifest.webmanifest` describes the installable app with relative scope, colors, and local icons.
- `main.js` is the ordered ES-module entry point.
- `offline/register.js` registers the Service Worker only in a supported secure context.
- `sw.js` maintains the versioned `orbit-shell-v12` cache.
- `IndexedDbVault` stores canonical user files in IndexedDB; the Service Worker does not intercept or cache those records.

Service Worker registration failure does not prevent the online application from starting. A separate Custom Element registration failure also leaves a controller-rendered native fallback for navigation, Today tasks, inspector actions, Add controls, readable component cards, and inactive non-executing widgets while canonical boot and saves continue. Service Workers are available on HTTPS and localhost.

## Cache strategy

The install handler precaches the application shell in `orbit-shell-v12`:

- `./`, `index.html`, the manifest, `main.js`, and `app.js`;
- ordered Custom Element registration and all seven element modules;
- offline registration, runtime storage, component-card/widget catalogs/codecs/repositories, generated-operation validation, and widget policy/envelope/protocol modules;
- all cascade-layer stylesheets, including `styles/elements.css`;
- self-hosted Pixel Loom font CSS and font files;
- the bundled `widgets/focus-orbit.html` sample application asset; and
- manifest icons.

This allowlist is the runtime module graph required by a fresh or retained profile. Canonical `cards/*.md`, user-created `widgets/*.html`, canvases, life files, and sidecar data remain IndexedDB records; the Service Worker never discovers or caches arbitrary vault paths.

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
Cache API                  application modules, styles, fonts, icons, and bundled sample widget
IndexedDB vault            canonical .canvas, .md, .html, and sidecar files; boot source
MemoryIndex                disposable in-memory query projection rebuilt from vault files
sessionStorage             provider key by default
localStorage               one-time first-run migration input only; not source of truth
```

Whole-space `.orbit.json` export/import is the portable backup for the sidecar and logical vault files. It contains no database snapshot or provider key. A single `.canvas` export remains valid JSON Canvas but may reference entity files that are not included. Clearing site data removes the shell cache and IndexedDB user data; it cannot be treated as a backup.

## Updates

`CACHE_NAME` is the cache-format version and is currently `orbit-shell-v12`. Increment it when cache semantics change or old entries must be invalidated. The Service Worker removes older `orbit-shell-*` caches during activation. It does not call `skipWaiting()` or reload an active editing session; a future immediate-update flow must ask the user, persist pending work, activate the waiting worker, and reload deliberately.

When a required runtime asset is added, moved, or removed, update `APP_SHELL` in the same change and run a shell-coverage check against loaded assets. Keep every URL relative so localhost and GitHub Pages subpath deployment use the same paths.

## Offline behavior and limits

Fresh- and retained-profile browser checks verify Service Worker control, a complete `orbit-shell-v12` install, controlled offline reload, local fonts/modules/styles, vault-first IndexedDB reconstruction, Today/task UI, declarative component cards, and canonical widget source resolution. A widget is still inactive after offline reload and requires explicit **Run**; remote provider requests remain unavailable.

Offline-capable local behavior includes canonical task/card/widget writes, Today queries, canvas navigation, import/export, and local typed proposals. Network-dependent behavior remains unavailable:

- remote AI generation and provider connection tests;
- external links;
- any future remote sync or calendar-provider adapters; and
- untrusted widget network access, which is rejected by source policy and denied by the widget CSP rather than supplied by the offline cache.

Upgrade behavior from a previously deployed Service Worker, IndexedDB quota/failure handling, and browser timezone boundaries remain pending. Offline shell success does not make site-data clearing safe: clearing browser storage removes both cache and canonical IndexedDB data.

## Validation checklist

1. Start from a fresh temporary browser profile over localhost or HTTPS.
2. Wait for `window.orbitVaultReady`, `window.orbitOfflineReady`, and `navigator.serviceWorker.ready`; verify the page is controlled.
3. Confirm `orbit-shell-v12` contains every `APP_SHELL` URL, no failed asset, no database Wasm, and no arbitrary vault file.
4. Exercise canonical task, component-card, and widget writes, then retain the same profile for controlled reload.
5. Verify Today/task state, standard file-node placements, and inactive widget source survive reload.
6. Export version-2 whole-space data and import it into a disposable staging IndexedDB vault; compare restored files and placements.
7. Enable browser offline emulation and reload without bypassing the Service Worker.
8. Verify wide and narrow shell/component layouts, local fonts/modules, Today, component cards, and explicit widget activation.
9. Reconnect and verify allowlisted same-origin resources refresh normally.
10. For a future cache-version change, test upgrade behavior from the previous active deployed worker.

Useful probes:

```js
await window.orbitOfflineReady
await navigator.serviceWorker.ready
await caches.keys()
(await caches.open("orbit-shell-v12")).keys()
await window.orbitVaultReady
window.orbitVaultStore
window.orbitCanvas.getDocument()
window.orbitCanvas.getWorkspace()
```

The explicit Node storage suites verify platform-neutral vault, codecs, repositories, in-memory indexes/catalogs, backup validation, generated-operation validation, and widget policy/protocol. Real-browser checks are required for IndexedDB durability, Service Worker control, UI flows, export/import restoration, offline reload, sandbox behavior, and responsive layout.
