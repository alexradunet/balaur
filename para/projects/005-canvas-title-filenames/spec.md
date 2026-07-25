---
phase: spec
status: done
project: 005-canvas-title-filenames
date: 2026-07-25
---

# Spec: Title-derived canvas filenames

## Problem Statement

Canvas files are named with opaque UIDs (`canvases/canvas-a1b2c3.canvas`). The filename carries no information; a user opening the vault folder in a file manager cannot tell which file is which canvas. The root canvas is additionally pinned to a hardcoded `canvases/root.canvas` path by a special-case check in the sidecar parser, an exception that earns nothing because root identity already lives in the sidecar `rootId` field.

## Solution

Canvas filenames derive from the title slug: `canvases/city-break.canvas` instead of `canvases/canvas-a1b2c3.canvas`. The sidecar `id` remains the identity; the filename is a human-readable locator. Renaming a canvas title renames the file, updates the sidecar `path`, and rewrites every `file`-node reference across all canvas documents in the workspace. Collisions resolve with numeric suffixes (`trip`, `trip-2`, `trip-3`). The root canvas follows the same rule; the hardcoded root-path constraint is lifted. Subcanvas creation prompts for a title via a dialog; cancel aborts. No migration of existing vaults; the graph starter and all new canvases use slugs from the start.

## User Stories

1. As a user, I want canvas files named after their title, so that I can identify them in a file manager without opening the app.
2. As a user, I want the root canvas named after its title, so that it is not a special case I have to remember.
3. As a user, I want renaming a canvas title to rename the file and update all references, so that no cross-reference silently breaks.
4. As a user, I want rename to trigger on title commit (blur or Enter), so that intermediate keystrokes do not cause file churn.
5. As a user, I want a numeric suffix when two canvases slug to the same name, so that no file is silently overwritten.
6. As a user, I want a title prompt when I create a subcanvas, so that the file is named meaningfully from the start.
7. As a user, I want to cancel the subcanvas title prompt, so that no throwaway canvas is created.
8. As a user, I want the graph starter to use slug filenames, so that a fresh vault is human-readable from day one.
9. As a user, I want a title that produces an empty slug to fall back to the canvas id, so that the file always has a valid name.
10. As a user, I want long titles truncated at a hyphen boundary, so that filenames stay within platform limits without a dangling partial word.

## Implementation Decisions

**Slug function.** A new `canvasSlug(title, fallbackId)` function lives in the vault-path module alongside the existing `slugify`. It differs from `slugify` in three ways: it keeps Unicode letters and digits (using Unicode property escapes `\p{L}` and `\p{N}`) rather than only ASCII `a-z0-9`; it enforces a 60-character maximum truncated at the last hyphen boundary before the limit (if no hyphen exists, hard-truncate at 60); and it falls back to `fallbackId` (the canvas id) rather than the string `"untitled"` when the slug is empty. The existing `slugify` is unchanged; entity paths continue to use it.

**Collision resolution.** A `uniqueCanvasPath(slug, existingPaths)` function in the workspace-vault module takes a base slug and a set of already-used canvas paths, and returns `canvases/<slug>.canvas`, appending `-2`, `-3`, etc. until the path is unused. Comparison is case-folded (using the existing `caseFoldKey`) to prevent collisions on case-insensitive filesystems.

**Lift the root-path constraint.** The `canvasPathFor` function drops its rootId special-case: the root canvas uses `record.path` like every other canvas. The `rootId` parameter is removed from the signature; callers pass only the record. The sidecar parser drops the check that forces the root to `canvases/root.canvas`. The `ROOT_CANVAS_PATH` constant is removed. The root canvas in `minimalFreshWorkspace` and the graph starter sets an explicit slug-derived path (`canvases/balaur.canvas` for the default title "Balaur", `canvases/home.canvas` for the graph starter title "Home").

**Rename propagation.** A pure `renameCanvasPath(workspace, canvasId, newPath)` function in the workspace-vault module performs the in-memory rename: it updates `workspace.canvases[canvasId].path` and rewrites every `file`-node whose `file` field equals the old path across all canvas documents in `workspace.canvases`. It returns `{ oldPath, newPath, affectedCanvasIds }`. The caller (app.js) is responsible for the vault write ordering: write the canvas document to the new path, call `renameCanvasPath` on the in-memory workspace, save the workspace (which writes all updated documents and the sidecar, then removes the orphaned old path via the existing orphan-cleanup in `WorkspaceStore._save`). The existing `ownedCanvasPaths` orphan-removal logic in `WorkspaceStore._save` handles deleting the old file once the sidecar no longer references it; no explicit delete call is needed.

**Title-commit trigger.** The `#canvasTitle` input gains a commit handler on `blur` and `Enter` (keydown). On commit, if the trimmed value differs from the current record title, the rename propagation runs. The existing `oninput` handler continues to call `saveCurrentCanvasState` and `scheduleSave` for live title sync; the commit handler adds the file-rename step on top. The existing `onblur` handler that resets the input to the record title is replaced by the commit handler (which updates the record title and then syncs the input).

**Subcanvas creation dialog.** `createSubcanvas` shows a native `<dialog>` with a single text input and Create/Cancel buttons. On Create (non-empty title), it creates the canvas with a slug-derived path via `canvasSlug` + `uniqueCanvasPath`. On Cancel or empty title, it aborts without creating anything. The dialog markup is added to `index.html` alongside the existing dialogs. The `#newCanvas` button and the AI `createSubcanvas` call both route through the dialog; the AI path passes a default title that the user can edit.

**Graph starter.** The graph starter already uses slug-like paths for hub canvases (`canvases/inbox.canvas`, etc.). The root canvas path changes from the implicit `canvases/root.canvas` to an explicit `canvases/home.canvas` set in `createGraphStarterWorkspace`. The `file`-node references in the root document that point to hub canvases are already correct and unchanged.

**Sidecar schema.** No schema change. The `path` field already holds an arbitrary `canvases/<name>.canvas` string; the only change is that the parser no longer rejects a root canvas whose path is not `canvases/root.canvas`. The `canvases/[^/]+\.canvas` regex check remains.

## Testing Decisions

The highest seam is the workspace-vault module: it is platform-neutral, asynchronous, and already tested by `storage/phase4.test.js` against `MemoryVault`. All new pure functions (`canvasSlug`, `uniqueCanvasPath`, `renameCanvasPath`) and the lifted root-path constraint are tested there. The existing `phase4.test.js` suite is the prior art: it builds a workspace object, calls `toSidecar`/`parseSidecar`/`WorkspaceStore`, and asserts on vault state. New tests follow the same pattern.

Specific test cases:

- `canvasSlug`: Unicode letters kept (`"Café Résumé"` → `"café-résumé"`), punctuation folded, consecutive hyphens collapsed, leading/trailing trimmed, 60-char truncation at hyphen boundary, empty title falls back to the canvas id.
- `uniqueCanvasPath`: no collision returns base slug; collision appends `-2`, `-3`; case-folded collision detected (`"Trip"` vs `"trip"`).
- `canvasPathFor`: root canvas uses `record.path` (no special-case); missing path derives from id fallback.
- `parseSidecar`: accepts a root canvas at any valid `canvases/*.canvas` path; still rejects paths outside `canvases/`.
- `renameCanvasPath`: updates the record path; rewrites `file`-node references in the same canvas and in cross-referencing canvases; returns correct `affectedCanvasIds`; does not touch non-file nodes or nodes pointing to other paths.
- `WorkspaceStore` round-trip: save a workspace with a slug-named root canvas, reload, verify the root document is intact and the old path is removed.

The app.js changes (dialog, title-commit handler, `createSubcanvas` prompt) are browser-pending and verified via the browser-check skill, not Node tests.

## Out of Scope

- Migration of existing UID-named vault files: no existing vaults to migrate; fresh build.
- Hierarchical folder structure: ruled out in the grill; the sidecar already encodes the hierarchy.
- Entities nested inside canvas folders: breaks multi-placement and identity-location separation.
- Rename-only-the-portal-node shortcut: silently breaks cross-references; ruled out in the grill.
- Debounced rename on every keystroke: ruled out; rename fires on commit only.
- Persistent index changes: the disposable in-memory index is rebuilt at boot; no schema change needed.

## Further Notes

The existing `slugify` in vault-path.js is used by entity paths (`entityPath`, `componentCardPath`) and must not change. `canvasSlug` is a separate function with different rules (Unicode letters, 60-char limit, id fallback). The two functions share the same hyphen-collapsing and trimming logic; a shared internal helper is acceptable but not required.

The `WorkspaceStore._save` orphan-removal logic (`ownedCanvasPaths`) already handles deleting canvas files that are no longer referenced by the sidecar. After `renameCanvasPath` updates the in-memory workspace and `save` is called, the old path is automatically removed as an orphan. No explicit `vault.remove` call is needed in the rename path, provided the save succeeds. If the save fails (e.g., a hash conflict on an unrelated file), the old file remains and the workspace is in a consistent state (the sidecar still references the old path until the save succeeds).

The `canvasIdFromPath` function in app.js currently falls back to stripping the `.canvas` extension from the path when no record matches. With slug-derived paths this fallback still works (the slug is not the id, but the fallback is only used for indexing diagnostics, not identity). No change needed there.

The `phase4-backup.test.js` suite references `canvases/root.canvas` in several assertions. Those assertions must be updated to use the new root path once the graph starter and `minimalFreshWorkspace` are updated. The backup module itself (`workspace-backup.js`) has no root-path special-case and needs no changes.
