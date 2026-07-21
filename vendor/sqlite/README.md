# Vendored SQLite Wasm

Orbit vendors the official SQLite Wasm ES module so the static application can use SQL without a package install, CDN, or build step.

- Package: `@sqlite.org/sqlite-wasm`
- Version: `3.53.0-build1`
- Source: https://www.npmjs.com/package/@sqlite.org/sqlite-wasm
- Upstream: https://sqlite.org/wasm/
- License: Apache License 2.0 (`LICENSE`)

Vendored files:

- `sqlite3.mjs` — `dist/index.mjs`
- `sqlite3.wasm` — `dist/sqlite3.wasm`

SHA-256 at import:

```text
f80870f0fa03a39a3338d17ed3fbea04808d344c88e724d90d5f37b9b7b83154  sqlite3.mjs
02d7e48164395fa68f81c6ec33e9da5461be397dc57602ac0cd89b4bbba1d312  sqlite3.wasm
```

## Browser backend

The GitHub Pages prototype opens `:localStorage:` through SQLite's `kvvfs`. This is intentionally a starter backend: it is synchronous, main-thread only, and subject to the browser's small localStorage quota. It gives Orbit a real SQLite schema and migration path while remaining deployable on GitHub Pages.

The production browser adapter should move the same schema to SQLite OPFS in a worker. The official OPFS build requires COOP/COEP response headers, which GitHub Pages cannot configure. A Tauri build can use native SQLite behind the same LifeStore interface.
