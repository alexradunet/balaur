# Desktop shell evaluation (Tauri / Electron / Neutralino / Bun / none)

From the grill in project `001-directory-vault` (Balaur, 2026-07). Settled for
that project; reuse this if the "bundle Balaur as a desktop app" question
returns. Mobile was ruled out of scope (separate later project, limited
features), which removes Tauri v2's main forward argument.

## Decision: no shell in v1 — File System Access API in the browser

The real-folder vault (the actual motivation) shipped as a pure web adapter
(`DirectoryVault`, ADR-0005). Zero toolchain, zero bundle, zero new constraint.

## If a shell is wanted later, the ranking

1. **Neutralino** — best fit for this repo. JS-only (no Rust), ~2MB prebuilt
   C++ binary, built-in fs API + native directory dialog, single vendored JS
   client (fits the `vendor/` provenance pattern). Costs: three system webview
   engines (WebKitGTK is untested and runs on the maintainer's own NixOS box),
   small-community maintenance risk, first npm dev-tooling (`neu` CLI).
2. **Electron** — the heavyweight fallback. Reuses the Node-verified `FsVault`
   in the main process almost for free, single Chromium engine (the one the
   browser-check skill already tests), Obsidian precedent in this exact
   category. Costs: ~100MB bundle, npm packaging toolchain, IPC security
   surface (contextIsolation, no nodeIntegration, allowlisted channels).
3. **Tauri** — rejected for this repo: rewrites the Node-verified `FsVault` in
   Rust (new language, tests don't transfer), three untested webviews, and its
   mobile advantage was ruled out of scope.
4. **Bun as a single binary** — rejected: gives a server, not a window; either
   opens a browser tab (no app feel, no secure-context guarantees for
   WebCrypto) or reinvents Electron with more glue.

## Constraint any shell must respect

`showDirectoryPicker` does not exist in WebKitGTK/WKWebView, so a shell on
Linux/macOS cannot reuse `DirectoryVault`; it ships its own fs adapter under
the same `VaultStore` contract (additive, not a rewrite). The web app stays
build-less; the shell is an additive packaging directory with its own
tooling, recorded in an ADR.
