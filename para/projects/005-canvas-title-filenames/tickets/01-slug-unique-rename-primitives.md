---
phase: ticket
status: done
project: 005-canvas-title-filenames
ticket: 01
blocked-by: []
worker: "3b0c72f2-9bb5-4d0e-b353-caaf40fdbbfd"
branch: "wealthy-blowfish"
shared-blast-radius: true
planned-at: 71899aa
---

# Ticket 01: Slug, unique-path, and rename primitives (additive, Node-tested)

## What to build

Add three pure, platform-neutral helper functions that the rest of this feature
builds on, plus their tests. After this ticket lands, the codebase can (a) turn a
canvas title into a human-readable filename slug, (b) resolve that slug to a
collision-free `canvases/<slug>.canvas` path, and (c) rename a canvas path
in-memory while rewriting every `file`-node reference that points at the old path.

This ticket is **strictly additive**: it adds new exports and new tests only. It
changes **no existing behavior** and removes nothing. The full existing suite
(197 tests) must stay green. Nothing in `app.js`, the sidecar schema, the root-path
constraint, or any existing test is touched here. The functions are wired into the
running app in tickets 02 and 03.

## Current state

The two modules you extend, with the exact code you must confirm before editing.

**`storage/vault-path.js`** — cross-platform safe-path utilities. You add `canvasSlug`
here, beside the existing `slugify`.

- `slugify` (lines 57–67) is the existing ASCII-only title slug used by entity paths.
  **Do not modify it.** It is the structural exemplar for `canvasSlug`:
  ```js
  export function slugify(title) {
    let slug = String(title ?? "").normalize("NFC").toLowerCase();
    slug = slug.replace(/[\/\\<>:"|?*\u0000-\u001f\u007f]/g, " ");
    slug = slug.replace(/[^a-z0-9]+/g, "-");
    slug = slug.replace(/-+/g, "-").replace(/^-+|-+$/g, "");
    slug = trimTrailing(slug);
    if (!slug) slug = "untitled";
    while (byteLength(slug) > 120) slug = slug.slice(0, slug.length - 1);
    return slug;
  }
  ```
- `trimTrailing` (lines 46–49) strips trailing spaces/periods; reuse it.
- `caseFoldKey` (lines 42–44) and `unicodeCaseFold` (lines 35–40) already exist and
  are exported; `canvasSlug` does not need them, but `uniqueCanvasPath` (in the other
  module) does.

**`storage/workspace-vault.js`** — sidecar schema, path validation, `WorkspaceStore`.
You add `uniqueCanvasPath` and `renameCanvasPath` here.

- Existing imports you will reuse (line 14–15):
  ```js
  import { assertSafePath, caseFoldKey } from "./vault-path.js";
  import { SchemaError, ParseError } from "./vault-errors.js";
  ```
  `assertSafePath`, `caseFoldKey`, and `SchemaError` are all already in scope; add no
  new imports.
- `canvasPathFor(record, rootId)` (lines 29–32) still has its two-argument signature
  and root special-case in this ticket. **Do not change it** (that is ticket 02).
  `renameCanvasPath` must therefore read the stored `record.path` directly rather than
  call `canvasPathFor`, so it does not depend on that signature.
- `WorkspaceStore._save` orphan-removal (lines 241–257): on save it writes every canvas
  document to `record.path`, writes the sidecar last, then removes any path in
  `this.ownedCanvasPaths` that the new sidecar no longer references (CAS-protected via
  `this.hashes`). Your round-trip test relies on this existing behavior to delete the
  old path after a rename; you add **no** delete logic.

**Repo conventions to match** (exemplar: `storage/vault-path.js` and
`storage/workspace-vault.js` themselves):
- Native strict ES modules, named exports, no build step, no dependencies.
- Validation throws typed errors from `storage/vault-errors.js` (`SchemaError`,
  `PathError`) with a `{ code: "..." }` and a message that includes the offending value.
- Doc comment above each exported function explaining intent (the existing functions
  all have one; match that style).
- Tests use `node:test` + `node:assert/strict`; see `storage/phase1.test.js` and
  `storage/phase4.test.js` for the house style.

**Architecture constraints (AGENTS.md) this work must honor:**
- §4.3 / §7: canvas paths are validated at storage boundaries; a corrupt sidecar must
  not escape the vault root. `renameCanvasPath` validates `newPath` with `assertSafePath`
  and the `canvases/[^/]+\.canvas` shape, mirroring `parseSidecar` line 102.
- §12: validation at storage boundaries; explicit side effects; no globals.

## Scope

**In scope** (the only files to modify):
- `storage/vault-path.js` (modify — add `canvasSlug` export)
- `storage/workspace-vault.js` (modify — add `uniqueCanvasPath` and `renameCanvasPath` exports)
- `storage/phase1.test.js` (modify — add `canvasSlug` tests beside the `slugify` test at line 108)
- `storage/phase4.test.js` (modify — add `uniqueCanvasPath`, `renameCanvasPath`, and a non-root rename round-trip test)

**Out of scope** (do NOT touch, even though it looks related):
- `storage/workspace-vault.js` `canvasPathFor`, `ROOT_CANVAS_PATH`, `parseSidecar`
  root check, or `toSidecar` — the root-path constraint lift is ticket 02.
- `app.js` — wiring the primitives into the UI is ticket 03.
- `index.html` — the subcanvas dialog is ticket 03.
- `storage/phase4-backup.test.js`, `storage/phase9.test.js`, and every other test file
  — their `canvases/root.canvas` fixtures are ticket 02's concern.
- `slugify`, `entityPath`, `componentCardPath` — unchanged; entity paths keep ASCII slugs.
- A slug-named **root** round-trip test — impossible in this ticket because `parseSidecar`
  (line 103) still rejects a root not at `canvases/root.canvas`. That test moves to
  ticket 02 (see STOP conditions). Test orphan-removal via a **non-root** rename instead.

## Steps

### Step 1: Add `canvasSlug(title, fallbackId)` to `storage/vault-path.js`

Add a new exported function immediately after `slugify` (after line 67). It differs
from `slugify` in exactly three ways, per the spec:

1. Keeps Unicode letters and digits (`\p{L}` and `\p{N}`, with the `u` flag) instead of
   ASCII `a-z0-9` only.
2. Enforces a 60-character maximum, truncated at the last hyphen boundary before the
   limit; if no hyphen exists in the first 60 chars, hard-truncate at 60. Truncate by
   **code points** (`[...slug]`) so astral characters are never split.
3. Falls back to `fallbackId` (the canvas id) instead of the string `"untitled"` when
   the slug is empty.

Target shape (load-bearing):

```js
// Human-readable canvas filename slug from a title (title-derived canvas filenames).
// Keeps Unicode letters and digits, folds everything else to hyphens, caps at 60
// characters truncated on a hyphen boundary, and falls back to fallbackId when empty.
// Distinct from slugify (ASCII-only, "untitled" fallback) which entity paths keep.
export function canvasSlug(title, fallbackId) {
  let slug = String(title ?? "").normalize("NFC").toLowerCase();
  slug = slug.replace(/[\/\\<>:"|?*\u0000-\u001f\u007f]/g, " ");
  slug = slug.replace(/[^\p{L}\p{N}]+/gu, "-");
  slug = slug.replace(/-+/g, "-").replace(/^-+|-+$/g, "");
  slug = trimTrailing(slug);
  if (!slug) return String(fallbackId);
  const chars = [...slug];
  if (chars.length > 60) {
    let cut = chars.slice(0, 60).join("");
    const boundary = cut.lastIndexOf("-");
    cut = boundary > 0 ? cut.slice(0, boundary) : cut;
    slug = cut.replace(/-+$/g, "");
  }
  return slug;
}
```

**Verify**: `node --check storage/vault-path.js` → exits 0 (no syntax errors).

### Step 2: Add `uniqueCanvasPath(slug, existingPaths)` to `storage/workspace-vault.js`

Add a new exported function (place it near `canvasPathFor`, e.g. just after it). It
returns `canvases/<slug>.canvas`, appending `-2`, `-3`, … until the path is unused.
Comparison is case-folded via the already-imported `caseFoldKey` so case-insensitive
filesystems cannot collide.

Target shape (load-bearing):

```js
// Collision-free logical .canvas path for a base slug. Returns canvases/<slug>.canvas,
// appending -2, -3, ... until the path is unused. Comparison is case-folded so
// case-insensitive filesystems cannot collide (e.g. "Trip" vs "trip").
export function uniqueCanvasPath(slug, existingPaths) {
  const taken = new Set((existingPaths || []).map((path) => caseFoldKey(path)));
  const base = `canvases/${slug}.canvas`;
  if (!taken.has(caseFoldKey(base))) return base;
  let n = 2;
  while (taken.has(caseFoldKey(`canvases/${slug}-${n}.canvas`))) n += 1;
  return `canvases/${slug}-${n}.canvas`;
}
```

**Verify**: `node --check storage/workspace-vault.js` → exits 0.

### Step 3: Add `renameCanvasPath(workspace, canvasId, newPath)` to `storage/workspace-vault.js`

Add a new exported pure function (place it after `uniqueCanvasPath`). It performs the
in-memory rename only: it validates `newPath`, sets `workspace.canvases[canvasId].path`
to it, and rewrites every `file`-node whose `file` equals the old path across **all**
canvas documents in `workspace.canvases` (a canvas can be referenced from multiple
canvases). It returns `{ oldPath, newPath, affectedCanvasIds }`. It never touches the
vault; the caller writes and saves (ticket 03).

It reads the old path from `record.path` directly (not via `canvasPathFor`, whose
signature changes in ticket 02). A rename target must have a concrete stored path;
guard the missing-path case with a typed error.

Target shape (load-bearing):

```js
// Pure in-memory canvas rename. Validates newPath, updates the record's path, and
// rewrites every file-node referencing the old path across all canvas documents (a
// canvas may be referenced from more than one canvas). Returns the change summary;
// the caller is responsible for the vault write ordering and save (which orphan-removes
// the old path via WorkspaceStore._save).
export function renameCanvasPath(workspace, canvasId, newPath) {
  const record = workspace?.canvases?.[canvasId];
  if (!record) throw new SchemaError(`No canvas record to rename: ${canvasId}`, { code: "CANVAS_RENAME_MISSING" });
  const oldPath = record.path;
  if (typeof oldPath !== "string" || !oldPath) {
    throw new SchemaError(`Canvas ${canvasId} has no concrete path to rename from`, { code: "CANVAS_RENAME_NO_PATH" });
  }
  const safeNew = assertSafePath(newPath);
  if (!/^canvases\/[^/]+\.canvas$/.test(safeNew)) {
    throw new SchemaError(`Canvas path is outside canvases/: ${safeNew}`, { code: "SIDECAR_CANVAS_PATH" });
  }
  record.path = safeNew;
  const affectedCanvasIds = [];
  for (const rec of Object.values(workspace.canvases)) {
    const nodes = rec?.document?.nodes;
    if (!Array.isArray(nodes)) continue;
    let touched = false;
    for (const node of nodes) {
      if (node && node.type === "file" && node.file === oldPath) { node.file = safeNew; touched = true; }
    }
    if (touched) affectedCanvasIds.push(rec.id);
  }
  return { oldPath, newPath: safeNew, affectedCanvasIds };
}
```

**Verify**: `node --check storage/workspace-vault.js` → exits 0.

### Step 4: Add `canvasSlug` tests to `storage/phase1.test.js`

The `slugify` test is at line 108 (`test("slugify and entityPath produce the
documented layout", ...)`). Add `canvasSlug` to the import list at line 9, then add a
new `test(...)` block immediately after the `slugify` test. Cover, per the spec:

- Unicode letters kept: `canvasSlug("Café Résumé", "canvas-x")` → `"café-résumé"`.
- Punctuation folded to hyphens and consecutive hyphens collapsed:
  `canvasSlug("Trip -- Plans!!", "canvas-x")` → `"trip-plans"`.
- Leading/trailing hyphens trimmed: `canvasSlug("  --weird--  ", "canvas-x")` → `"weird"`.
- Empty / punctuation-only title falls back to the id:
  `canvasSlug("", "canvas-a1b2c3")` → `"canvas-a1b2c3"` and
  `canvasSlug("!!!", "canvas-a1b2c3")` → `"canvas-a1b2c3"`.
- 60-character truncation at a hyphen boundary: build a title of several hyphen-separated
  words whose slug exceeds 60 chars and assert the result is `<= 60` code points, ends
  on a word boundary (no trailing hyphen, no dangling partial word), e.g.
  `canvasSlug("alpha beta gamma delta epsilon zeta eta theta", "x")` → assert
  `[...result].length <= 60`, `!result.endsWith("-")`, and that it equals the input slug
  cut at the last hyphen before 60.
- Hard-truncate at 60 when there is no hyphen: a single 80-letter word →
  `[...result].length === 60`.

Match the existing assertion style (`assert.equal`). Do not alter the existing `slugify`
assertions.

**Verify**: `node --test storage/phase1.test.js` → exits 0, all tests pass (the new
`canvasSlug` test appears in the output).

### Step 5: Add `uniqueCanvasPath` and `renameCanvasPath` tests to `storage/phase4.test.js`

Add `uniqueCanvasPath` and `renameCanvasPath` to the import block from
`./workspace-vault.js` (lines 8–12). Then add new `test(...)` blocks. Use the existing
`legacyWorkspace()` fixture (defined at line 24): its root is `canvas-root` with
`path: null` and its child `canvas-planning` has `path: "canvases/planning.canvas"`;
the root document contains a portal `file` node `portal-1` with
`file: "canvases/planning.canvas"`.

Tests to add:

1. **`uniqueCanvasPath` collision resolution** (pure, no vault):
   - No collision: `uniqueCanvasPath("trip", [])` → `"canvases/trip.canvas"`.
   - Sequential suffix: with `["canvases/trip.canvas"]` taken → `"canvases/trip-2.canvas"`;
     with `["canvases/trip.canvas", "canvases/trip-2.canvas"]` taken → `"canvases/trip-3.canvas"`.
   - Case-folded collision: with `["canvases/Trip.canvas"]` taken,
     `uniqueCanvasPath("trip", ...)` → `"canvases/trip-2.canvas"` (not `trip.canvas`).

2. **`renameCanvasPath` pure behavior** (build a small in-memory workspace object inline
   or clone `legacyWorkspace()`; give the root an explicit `path` so the rename source is
   concrete):
   - Updates the record path and returns `{ oldPath, newPath, affectedCanvasIds }`.
   - Rewrites a `file`-node in the **same** canvas and in a **cross-referencing** canvas;
     `affectedCanvasIds` lists exactly the canvases whose documents were touched.
   - Does **not** touch non-`file` nodes (e.g. a `text` node) or `file`-nodes pointing at
     a different path.
   - Throws `SchemaError` for a missing canvas id, for a record with no concrete path, and
     for a `newPath` outside `canvases/` (e.g. `"notes/x.canvas"`) or an unsafe path
     (e.g. `"../escape.canvas"` → `PathError` from `assertSafePath`).

3. **Non-root rename round-trip proves orphan-removal** (uses `MemoryVault` +
   `WorkspaceStore`, mirroring the existing tests in this file):
   ```js
   test("renameCanvasPath rewrites file-node references and save removes the orphaned old path", async () => {
     const vault = new MemoryVault();
     const store = new WorkspaceStore(vault);
     await store.migrate(legacyWorkspace());
     const ws = (await store.load()).workspace;
     const result = renameCanvasPath(ws, "canvas-planning", "canvases/planning-v2.canvas");
     assert.equal(result.oldPath, "canvases/planning.canvas");
     assert.equal(result.newPath, "canvases/planning-v2.canvas");
     assert.deepEqual(result.affectedCanvasIds, ["canvas-root"]);
     const portal = ws.canvases["canvas-root"].document.nodes.find((n) => n.id === "portal-1");
     assert.equal(portal.file, "canvases/planning-v2.canvas");
     await store.save(ws);
     assert.equal(await vault.exists("canvases/planning.canvas"), false); // old path orphan-removed
     assert.equal(await vault.exists("canvases/planning-v2.canvas"), true);
     const reloaded = (await new WorkspaceStore(vault).load()).workspace;
     assert.equal(reloaded.canvases["canvas-planning"].path, "canvases/planning-v2.canvas");
     assert.equal(reloaded.canvases["canvas-root"].document.nodes.find((n) => n.id === "portal-1").file, "canvases/planning-v2.canvas");
   });
   ```
   This renames a **non-root** canvas, so it passes under the current root-path
   constraint (the root stays at `canvases/root.canvas`). `MemoryVault` is already
   imported in this file.

**Verify**: `node --test storage/phase4.test.js` → exits 0, all tests pass including the
new ones.

### Step 6: Run the full foundation suite and the static checks

**Verify**:
```
node --test \
  storage/phase1.test.js storage/phase2.test.js storage/phase3.test.js \
  storage/phase4.test.js storage/phase4-backup.test.js storage/phase5.test.js \
  storage/phase7.test.js storage/phase8.test.js storage/phase9.test.js \
  storage/phase10.test.js storage/phase-query.test.js \
  storage/note-repository.test.js
```
→ exits 0; `# pass` is **197 + (number of tests you added)**, `# fail 0`. The prior 197
tests all still pass (this ticket is additive).

Also: `node --check storage/vault-path.js storage/workspace-vault.js` → exits 0, and
`git diff --check` → no whitespace errors.

## Done criteria

Machine-checkable. ALL must hold:
- [ ] `canvasSlug` is exported from `storage/vault-path.js`; `slugify` is byte-for-byte unchanged.
- [ ] `uniqueCanvasPath` and `renameCanvasPath` are exported from `storage/workspace-vault.js`.
- [ ] `node --check storage/vault-path.js storage/workspace-vault.js` exits 0.
- [ ] `node --test storage/phase1.test.js storage/phase4.test.js` exits 0 with the new tests present.
- [ ] The full suite command in Step 6 exits 0 with `# fail 0` and `# pass` equal to `197 + N` (N = tests added).
- [ ] `git status --short` shows only the four in-scope files modified.
- [ ] `grep -n "ROOT_CANVAS_PATH\|canvasPathFor" storage/workspace-vault.js` is unchanged from before (no signature/constant edits leaked in).

## STOP conditions

Stop and report back (do not improvise) if:
- The code at the locations in "Current state" does not match the excerpts (the codebase
  drifted since `planned-at` 71899aa). In particular, if `slugify` is not at lines 57–67
  of `storage/vault-path.js`, or `canvasPathFor`/`ROOT_CANVAS_PATH`/the `parseSidecar`
  root check are not at lines 29–32 / 20 / 103 of `storage/workspace-vault.js`.
- You feel you must change `canvasPathFor`, `ROOT_CANVAS_PATH`, `parseSidecar`, `toSidecar`,
  `app.js`, `index.html`, or any test file other than `phase1.test.js`/`phase4.test.js`
  to make this work — that belongs to ticket 02/03.
- You are tempted to write a slug-named **root** round-trip test: it cannot pass in this
  ticket because `parseSidecar` (line 103) still rejects a root not at `canvases/root.canvas`.
  Use the non-root rename round-trip in Step 5 instead; the root case is tested in ticket 02.
- A step's verification fails twice after a reasonable fix attempt.
- The full suite drops below 197 passing tests (you changed existing behavior; revert and report).

## Blocked by

None — can start immediately.

## Next step

- Recommended executor tier: mid
- Recommended model: `pi/qwen-token-plan/qwen3.7-plus` (registry tiers are user-edited/TBD; orchestrator should confirm against `para/resources/model-registry.md`)
- Estimated complexity: S
