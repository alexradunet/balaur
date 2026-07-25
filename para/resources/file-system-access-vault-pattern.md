# Pattern: File System Access API as a vault adapter

From project `001-directory-vault` (Balaur, 2026-07). How a static, build-less
web app gets a real-folder vault in Chromium without a shell, a build step, or
a dependency.

## The shape

- The adapter implements the app's async store contract over a
  `FileSystemDirectoryHandle` from `showDirectoryPicker({ mode: "readwrite" })`.
  The picker grant lasts one session; re-pick every launch, persist nothing
  (zero invalidation edge cases; multi-vault is free).
- **Constructor injection is the test seam:** the adapter takes the handle as a
  parameter. Production passes the picked handle; headless tests pass an OPFS
  handle (`navigator.storage.getDirectory()` — same interface, no picker, no
  gesture). The full contract suite runs headlessly against OPFS.
- **Picker stub for full-app smoke:** the app's pick handler must reference the
  *global* `showDirectoryPicker` at call time (never a module-scope capture).
  The CDP driver then stubs `window.showDirectoryPicker` with an OPFS handle
  and clicks the real button, so the whole boot path runs headlessly. This is
  an app-code constraint, not a test hook in production.
- Browsers without the API get a hard full-screen gate, not a fallback adapter.
  One adapter, one engine; `crypto.subtle` is gated too (secure context).

## Semantics that matter on disk

- No content cache: every read/stat/list hits the folder so external edits
  (editor, Syncthing) are visible. External-change reconciliation is a manual
  "Reload vault" action — no watcher in v1.
- `createWritable` is not atomic-rename. The expected-content-hash precondition
  is the conflict guard; a torn write surfaces through existing repair paths.
- `move` must hash the content it just read (not the prior stat's hash) — on a
  disk-backed adapter the two diverge under an external-edit race.
- DOMException mapping at every boundary: `NotFoundError` → NOT_FOUND (read) or
  null/false (stat/exists); `NotAllowedError`/`SecurityError`/`InvalidStateError`
  → storage-unavailable; `TypeMismatchError` → path-component error.

## Known limitation

Chromium desktop only. `showDirectoryPicker` does not exist in Firefox, Safari,
WebKitGTK, or WKWebView — so any future native shell on Linux/macOS needs its
own fs adapter under the same store contract (see `desktop-shell-evaluation.md`).
