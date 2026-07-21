# Offline-first application shell

Orbit is an installable, progressively enhanced web application. Offline support is implemented with browser standards and does not change the ownership of workspace data.

## Runtime pieces

- `manifest.webmanifest` describes the installable app, relative scope, colors, and local icons.
- `main.js` is the ordered ES-module entry point.
- `offline/register.js` registers the Service Worker only in a supported secure context.
- `sw.js` maintains the versioned `orbit-shell-*` cache.

Registration failures do not prevent the normal online application from starting. Service Workers are available on HTTPS and on localhost.

## Cache strategy

The install event atomically precaches the application shell:

- HTML and JavaScript modules
- cascade-layer styles
- self-hosted fonts and design tokens
- SQLite Wasm JavaScript and binary
- the local WebGL sample widget
- manifest and icons

Same-origin GET requests are network-first. Successful basic responses refresh the cache, while network failures fall back to the matching cached response. Navigation Preload avoids serializing browser navigation behind Service Worker startup, and navigation can fall back to `index.html`. This favors current source while online and preserves a complete local shell when disconnected.

Orbit deliberately does not intercept:

- non-GET requests;
- cross-origin AI-provider calls;
- byte-range requests;
- generated `blob:` downloads;
- API keys or other browser-storage values.

The Cache API contains deployable application resources, not user data.

## User data remains separate

Offline shell caching does not replace Orbit persistence:

```text
Cache API                 application code and static assets
IndexedDB vault           canonical .canvas documents + sidecar (boot source; ADR-0001)
Workspace localStorage    fallback / migration mirror of canvases + hierarchy
SQLite kvvfs              tasks and temporal/queryable life data
sessionStorage            provider key by default
```

A whole-space `.orbit.json` export remains the portable backup. Clearing site data removes both caches and local user data.

## Updates

`CACHE_NAME` is the cache-format version. Increment it when cache semantics change or when old entries must be removed immediately. Normal source responses refresh during online use even when the cache version is unchanged.

The Service Worker does not call `skipWaiting()`. Replacing the runtime in the middle of an editing session can create mixed-version behavior, so a future immediate-update flow must ask the user, persist pending work, activate the waiting worker, and reload deliberately.

When a required runtime asset is added, moved, or removed, update `APP_SHELL` in the same change. Keep URLs relative so local hosting and GitHub Pages subpath deployment share the same code.

## Offline behavior and limits

After one successful online load, Orbit can reload its shell, SQLite Wasm, local fonts, and local widgets without a network connection. Existing workspace and life data remain available in that browser profile.

Network-dependent behavior remains explicitly unavailable offline:

- remote AI generation and provider connection tests;
- external links;
- widgets that intentionally request remote resources;
- future remote sync or calendar-provider adapters.

Local canvas editing, navigation, tasks, Today, imports, and exports do not require a provider connection.

## Validation checklist

1. Start from a clean browser profile over localhost or HTTPS.
2. Wait for `navigator.serviceWorker.ready` and verify the page is controlled.
3. Confirm the `orbit-shell-*` cache contains all `APP_SHELL` assets.
4. Create and complete a task, then reload and verify SQLite persistence.
5. Enable browser offline emulation and reload without bypassing the Service Worker.
6. Verify the app shell, fonts, SQLite, Today, and local WebGL widget still load.
7. Reconnect and verify same-origin assets refresh normally.
8. For cache-version changes, test upgrade behavior from the previous active worker.

Useful probes:

```js
await window.orbitOfflineReady
await navigator.serviceWorker.ready
await caches.keys()
(await caches.open("orbit-shell-v2")).keys()
await window.orbitVaultReady // Phase 4b canonical vault bridge
window.orbitVaultStore
```

The production OPFS SQLite adapter is a separate concern. It will require a Worker and appropriate cross-origin-isolation headers; GitHub Pages currently cannot provide those headers.
