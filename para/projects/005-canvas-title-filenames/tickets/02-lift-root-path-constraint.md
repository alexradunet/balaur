---
phase: ticket
status: done
project: 005-canvas-title-filenames
ticket: 02
blocked-by: [01]
worker: "cb57e88f-2925-4b2c-ab9d-e3ad998dd342"
branch: "wealthy-blowfish"
shared-blast-radius: true
planned-at: 71899aa
---

# Ticket 02: Lift the root-path constraint (contract change across storage + app.js + tests)

## What to build

Remove the special case that pins the root canvas to the hardcoded path
`canvases/root.canvas`. After this ticket, the root canvas is located by its sidecar
`path` like every other canvas (falling back to `canvases/<id>.canvas` when no path is
stored); root identity is the sidecar `rootId` field, not the filename. Concretely:
`canvasPathFor` loses its `rootId` parameter and root branch, the `ROOT_CANVAS_PATH`
constant and the `parseSidecar` "root must be canvases/root.canvas" check are deleted,
the two workspace builders in `app.js` set explicit slug-derived root paths, and every
test that asserted the old root path is updated to the id-fallback path
`canvases/canvas-root.canvas` (the fixtures keep `path: null`, so this is a minimal
assertion-only diff).

This is the contract step that unblocks ticket 03 (title-commit rename + subcanvas
dialog). It changes behavior deliberately; the suite stays green because the test
updates in this ticket track the contract change exactly.

## Current state

Confirm every excerpt below before editing. If any line has drifted, treat it as a STOP
condition.

### `storage/workspace-vault.js`

- Line 20 — the constant to delete:
  ```js
  export const ROOT_CANVAS_PATH = "canvases/root.canvas";
  ```
- Lines 25–32 — `canvasPathFor` with the root special-case to remove (note the
  two-argument signature):
  ```js
  // Logical .canvas path for a canvas record. The root is always
  // canvases/root.canvas; other canvases use their stored path or a derived
  // canvases/<id>.canvas. The result is strictly validated so a corrupt sidecar
  // cannot escape the vault root (plan §7.3, Phase 4 portal path validation).
  export function canvasPathFor(record, rootId) {
    if (record.id === rootId) return assertSafePath(ROOT_CANVAS_PATH);
    return assertSafePath(record.path || `canvases/${record.id}.canvas`);
  }
  ```
- Line 45 — the in-module caller inside `toSidecar` (the `rootId` local on line 39 is
  still needed for `activeId` on line 58, so keep that local; only drop the second
  argument here):
  ```js
      path: canvasPathFor(record, rootId),
  ```
- Lines 102–103 inside `parseSidecar` — keep line 102 (the `canvases/` shape guard),
  delete line 103 (the root special-case):
  ```js
      if (!/^canvases\/[^/]+\.canvas$/.test(path)) throw new SchemaError(`Canvas path is outside canvases/: ${path}`, { code: "SIDECAR_CANVAS_PATH" });
      if (record.id === data.rootId && path !== ROOT_CANVAS_PATH) throw new SchemaError("Root canvas must use canvases/root.canvas", { code: "SIDECAR_CANVAS_PATH" });
  ```

### `app.js`

- Line 4 — import (unchanged; `canvasPathFor` is already imported):
  ```js
  import { WorkspaceStore, hasWorkspace, canvasPathFor } from "./storage/workspace-vault.js";
  ```
- Line 140 — `const ROOT_CANVAS_ID="canvas-root";` (context only; do not change).
- Line 145 — the transient pre-boot `workspace` placeholder (`title:"Loading…"`,
  `path:null`). **Out of scope**: it is never persisted (boot replaces it via
  `migrate`/`load`; `saveWorkspaceNow` guards on `vaultStore`). Leave it untouched.
- Lines 199–201 — `minimalFreshWorkspace`, root record has `title:"Balaur"`, `path:null`:
  ```js
  function minimalFreshWorkspace(){
    return {version:1,rootId:ROOT_CANVAS_ID,activeId:ROOT_CANVAS_ID,canvases:{[ROOT_CANVAS_ID]:{id:ROOT_CANVAS_ID,title:"Balaur",parentId:null,portalNodeId:null,path:null,document:{nodes:[],edges:[]},camera:{x:80,y:55,zoom:.78}}}};
  }
  ```
- Lines 51–52 inside `createGraphStarterWorkspace` — sets the root title to "Home"
  (it inherits `path:null` from `minimalFreshWorkspace`):
  ```js
    const result=minimalFreshWorkspace(),root=result.canvases[result.rootId];
    root.title="Home";root.document=rootDocument;root.camera=null;
  ```
- The seven `canvasPathFor` callers, all currently passing `workspace.rootId` as the
  second argument:
  - Line 249: `const path=canvasPathFor(record,workspace.rootId);`
  - Line 275: `... .find(item => canvasPathFor(item, workspace.rootId) === path);`
  - Line 296: `return record ? canvasPathFor(record, workspace.rootId) : null;`
  - Line 305: `return record ? canvasPathFor(record, workspace.rootId) : null;`
  - Line 310: `canvasPathFromId: id => { const record = workspace.canvases[id]; return record ? canvasPathFor(record, workspace.rootId) : null; }`
  - Line 314: `canvasPathFromId: id => { const record = workspace.canvases[id]; return record ? canvasPathFor(record, workspace.rootId) : null; }`
  - Line 319: `canvasPathFromId: id => { const record = workspace.canvases[id]; return record ? canvasPathFor(record, workspace.rootId) : null; }`

### `storage/phase4.test.js`

- Line 9 — `ROOT_CANVAS_PATH` is in the import list from `./workspace-vault.js`.
- Line 31 — `legacyWorkspace()` fixture root record: `... path: null, ...` (keep `null`).
- Lines 48–53 — the `canvasPathFor` test (old two-arg signature, uses `ROOT_CANVAS_PATH`):
  ```js
  test("canvasPathFor maps root, stored, and derived paths; rejects unsafe paths", () => {
    assert.equal(canvasPathFor({ id: "canvas-root" }, "canvas-root"), ROOT_CANVAS_PATH);
    assert.equal(canvasPathFor({ id: "c2", path: "canvases/planning.canvas" }, "root"), "canvases/planning.canvas");
    assert.equal(canvasPathFor({ id: "c3" }, "root"), "canvases/c3.canvas");
    assert.throws(() => canvasPathFor({ id: "c4", path: "../escape.canvas" }, "root"), PathError);
    assert.throws(() => canvasPathFor({ id: "c5", path: "/abs.canvas" }, "root"), PathError);
  });
  ```
- Line 61 — `assert.equal(sidecar.canvases["canvas-root"].path, ROOT_CANVAS_PATH);`
- Lines 175, 179 — external-edit conflict test writes/stale-hashes the root file via
  `ROOT_CANVAS_PATH`.
- Line 243 — orphan test asserts `await vault.exists(ROOT_CANVAS_PATH)` is `true`.

Because the fixture root keeps `path: null`, removing the special-case makes the root
resolve to the id fallback `canvases/canvas-root.canvas`. Every `ROOT_CANVAS_PATH`
literal in this file therefore becomes the string `"canvases/canvas-root.canvas"`.

### `storage/phase4-backup.test.js`

- Line 41 — fixture root record `path: null` (keep `null`). `populatedVault()` migrates
  this fixture through `WorkspaceStore`, so the root file lands at the id-fallback path.
- Four assertions reference the old root path and must become `"canvases/canvas-root.canvas"`:
  - Line 72: `assert.ok(paths.includes("canvases/root.canvas"));`
  - Line 150: `badCanvas.files.find((f) => f.path === "canvases/root.canvas").text = ...;`
  - Line 220: `const root = missing.files.find((file) => file.path === "canvases/root.canvas");`
  - Line 277: `assert.equal(await staging.exists("canvases/root.canvas"), true);`
- No full path-list `deepEqual` hardcodes the root path (line 70 checks sort order on the
  dynamic list; line 255 compares two dynamic listings), so these four literals are the
  complete set of changes here. The backup module (`storage/workspace-backup.js`) has no
  root special-case and needs **no** change.

### `storage/phase9.test.js`  (recon correction — this file also depends on the special-case)

- Line 138 — the "integrates with WorkspaceStore" test builds a workspace whose root
  record has `path: null`, migrates it through `WorkspaceStore`, then asserts the root
  file/path. Two assertions reference the old root path and must become
  `"canvases/canvas-root.canvas"`:
  - Line 149: `assert.equal(await vault.exists("canvases/root.canvas"), true);`
  - Line 151: `assert.equal(sidecarOnDisk.canvases["canvas-root"].path, "canvases/root.canvas");`
- Lines 46–50 are a **different** test that writes the literal string
  `canvases/root.canvas` directly via `vault.write` (no `WorkspaceStore`, no
  `canvasPathFor`). It is an independent direct-write fixture and is **not** affected by
  this change — leave lines 46–50 untouched.

### Test files that must NOT change (verified independent fixtures)

`storage/phase2.test.js`, `phase3.test.js`, `phase5.test.js`, `phase8.test.js`,
`component-card.test.js`, `note-repository.test.js`, `widget-repository.test.js` all use
`canvases/root.canvas` only as an arbitrary direct-write fixture string (manual
id→path maps or literal `vault.write`/`vault.read`), never routed through
`canvasPathFor`/`toSidecar`/`parseSidecar`/`WorkspaceStore` root logic. They stay green
unchanged. Do not edit them.

### Conventions and constraints

- Match the native strict-ES-module style; keep doc comments accurate (rewrite the
  `canvasPathFor` doc comment to describe the new behavior).
- AGENTS.md §4.3: the sidecar `path` already holds an arbitrary `canvases/<name>.canvas`
  string; the `canvases/[^/]+\.canvas` regex check (line 102) remains. Only the root
  exception is removed. §9: root identity is `rootId`, consistent with the graph model.

## Scope

**In scope** (the only files to modify):
- `storage/workspace-vault.js` (modify — drop `ROOT_CANVAS_PATH`, simplify `canvasPathFor`, fix the `toSidecar` call, delete the `parseSidecar` root check)
- `app.js` (modify — explicit root paths in the two builders; seven single-arg `canvasPathFor` callers)
- `storage/phase4.test.js` (modify — rewrite `canvasPathFor` test, replace `ROOT_CANVAS_PATH` literals, add root-at-slug `parseSidecar` test + slug-named-root round-trip)
- `storage/phase4-backup.test.js` (modify — four root-path assertion literals)
- `storage/phase9.test.js` (modify — two root-path assertion literals in the WorkspaceStore test only)

**Out of scope** (do NOT touch):
- `storage/vault-path.js` — `canvasSlug` was added in ticket 01; nothing changes here.
- `storage/workspace-backup.js` — no root special-case; unaffected.
- The seven test files listed under "must NOT change" above.
- `app.js` line 145 (transient `Loading…` placeholder), `createSubcanvas`, `#canvasTitle`
  handlers, the subcanvas dialog — that is ticket 03.
- `index.html` — ticket 03.
- `uniqueCanvasPath` / `renameCanvasPath` from ticket 01 — unchanged here.

## Steps

### Step 1: Simplify `canvasPathFor` and remove `ROOT_CANVAS_PATH` in `storage/workspace-vault.js`

- Delete line 20 (`export const ROOT_CANVAS_PATH = ...`).
- Replace the `canvasPathFor` function (lines 25–32) with the single-argument form and an
  updated doc comment:
  ```js
  // Logical .canvas path for a canvas record. Every canvas (root included) uses its
  // stored path or a derived canvases/<id>.canvas; the root is no longer a special case
  // because its identity is the sidecar rootId, not its filename. The result is strictly
  // validated so a corrupt sidecar cannot escape the vault root.
  export function canvasPathFor(record) {
    return assertSafePath(record.path || `canvases/${record.id}.canvas`);
  }
  ```
- In `toSidecar`, change line 45 from `path: canvasPathFor(record, rootId),` to
  `path: canvasPathFor(record),`. Keep the `const rootId = workspace.rootId;` local
  (line 39) — it is still used by `activeId` on line 58.
- In `parseSidecar`, delete line 103 (the `Root canvas must use canvases/root.canvas`
  check). Keep line 102 (the `canvases/[^/]+\.canvas` guard) and the case-fold collision
  check that follows.

**Verify**: `node --check storage/workspace-vault.js` → exits 0.
`grep -n "ROOT_CANVAS_PATH" storage/workspace-vault.js` → no matches.

### Step 2: Set explicit slug-derived root paths in `app.js`

- In `minimalFreshWorkspace` (line 200), change the root record's `path:null` to
  `path:"canvases/balaur.canvas"` (the title is "Balaur", so the slug is `balaur`).
- In `createGraphStarterWorkspace` (line 52), the root title is overridden to "Home";
  also set the path. Change
  `root.title="Home";root.document=rootDocument;root.camera=null;`
  to
  `root.title="Home";root.path="canvases/home.canvas";root.document=rootDocument;root.camera=null;`
  (This overrides the `canvases/balaur.canvas` inherited from `minimalFreshWorkspace`.)
- The root document's `file`-nodes already point at the hub canvases
  (`canvases/inbox.canvas`, etc.) and the hubs/project set explicit paths via the `hub()`
  helper and the `project-city-break` record; those are correct and unchanged.

**Verify**: `node --check app.js` → exits 0.

### Step 3: Update the seven single-arg `canvasPathFor` callers in `app.js`

Drop the second argument at lines 249, 275, 296, 305, 310, 314, 319:
- 249 → `const path=canvasPathFor(record);`
- 275 → `... .find(item => canvasPathFor(item) === path);`
- 296, 305 → `return record ? canvasPathFor(record) : null;`
- 310, 314, 319 → `canvasPathFromId: id => { const record = workspace.canvases[id]; return record ? canvasPathFor(record) : null; }`

**Verify**: `node --check app.js` → exits 0.
`grep -n "canvasPathFor(.*,.*rootId)" app.js` → no matches (no two-arg calls remain).

### Step 4: Rewrite the `canvasPathFor` test and replace `ROOT_CANVAS_PATH` literals in `storage/phase4.test.js`

- Remove `ROOT_CANVAS_PATH` from the import list (line 9).
- Replace the test at lines 48–53 with the new-signature version, covering: stored path
  used as-is, id fallback when `path` is absent, **root uses `record.path` with no
  special-case**, root id fallback when its path is absent, and unsafe paths still throw:
  ```js
  test("canvasPathFor maps stored and derived paths; rejects unsafe paths", () => {
    assert.equal(canvasPathFor({ id: "c2", path: "canvases/planning.canvas" }), "canvases/planning.canvas");
    assert.equal(canvasPathFor({ id: "c3" }), "canvases/c3.canvas");
    assert.equal(canvasPathFor({ id: "canvas-root", path: "canvases/home.canvas" }), "canvases/home.canvas");
    assert.equal(canvasPathFor({ id: "canvas-root" }), "canvases/canvas-root.canvas");
    assert.throws(() => canvasPathFor({ id: "c4", path: "../escape.canvas" }), PathError);
    assert.throws(() => canvasPathFor({ id: "c5", path: "/abs.canvas" }), PathError);
  });
  ```
- Line 61 → `assert.equal(sidecar.canvases["canvas-root"].path, "canvases/canvas-root.canvas");`
- Line 175 → `await vault.write("canvases/canvas-root.canvas", canvasToJSON(doc(textNode("ext", "# External edit"))));`
- Line 179 → `store.hashes.set("canvases/canvas-root.canvas", "stale-hash");`
- Line 243 → `assert.equal(await vault.exists("canvases/canvas-root.canvas"), true);`

**Verify**: `grep -n "ROOT_CANVAS_PATH" storage/phase4.test.js` → no matches.

### Step 5: Add the root-at-slug `parseSidecar` test and the slug-named-root round-trip to `storage/phase4.test.js`

These prove the constraint is genuinely lifted (moved here from ticket 01 because they
require the special-case to be gone). Add two `test(...)` blocks:

```js
test("parseSidecar accepts a root canvas at any valid canvases/ path and still rejects paths outside canvases/", () => {
  const sidecar = toSidecar(legacyWorkspace());
  sidecar.canvases["canvas-root"].path = "canvases/home.canvas";
  const parsed = parseSidecar(JSON.stringify(sidecar));
  assert.equal(parsed.canvases["canvas-root"].path, "canvases/home.canvas");
  sidecar.canvases["canvas-root"].path = "notes/root.canvas";
  assert.throws(() => parseSidecar(JSON.stringify(sidecar)), SchemaError);
});

test("a slug-named root canvas saves, reloads intact, and is located by its path", async () => {
  const vault = new MemoryVault();
  const store = new WorkspaceStore(vault);
  const ws = legacyWorkspace();
  ws.canvases["canvas-root"].path = "canvases/life-os.canvas";
  await store.migrate(ws);
  assert.equal(await vault.exists("canvases/life-os.canvas"), true);
  const reloaded = (await new WorkspaceStore(vault).load()).workspace;
  assert.equal(reloaded.canvases["canvas-root"].path, "canvases/life-os.canvas");
  assert.ok(isCanvas(reloaded.canvases["canvas-root"].document));
  assert.ok(reloaded.canvases["canvas-root"].document.nodes.some((n) => n.id === "n1"));
});
```

`SchemaError`, `MemoryVault`, `isCanvas`, `toSidecar`, `parseSidecar`, and
`legacyWorkspace` are all already in scope in this file.

**Verify**: `node --test storage/phase4.test.js` → exits 0, all pass including the two new tests.

### Step 6: Update the four root-path assertions in `storage/phase4-backup.test.js`

Change the literal `"canvases/root.canvas"` to `"canvases/canvas-root.canvas"` at lines
72, 150, 220, 277 only. Do not change the fixture (line 41 keeps `path: null`).

**Verify**: `grep -n "canvases/root.canvas" storage/phase4-backup.test.js` → no matches.
`node --test storage/phase4-backup.test.js` → exits 0.

### Step 7: Update the two root-path assertions in `storage/phase9.test.js`

In the "integrates with WorkspaceStore" test only, change `"canvases/root.canvas"` to
`"canvases/canvas-root.canvas"` at lines 149 and 151. Do **not** touch the direct-write
fixture test at lines 46–50 (it writes the literal string independently and is unaffected).

**Verify**: `node --test storage/phase9.test.js` → exits 0.
`grep -n "canvases/root.canvas" storage/phase9.test.js` → matches only remain at lines
46–50 (the independent direct-write fixture), i.e. exactly two matches on lines 46 and 49/50.

### Step 8: Run the full foundation suite and static checks

**Verify**:
```
node --test \
  storage/phase1.test.js storage/phase2.test.js storage/phase3.test.js \
  storage/phase4.test.js storage/phase4-backup.test.js storage/phase5.test.js \
  storage/phase7.test.js storage/phase8.test.js storage/phase9.test.js \
  storage/phase10.test.js storage/phase-query.test.js \
  storage/note-repository.test.js
```
→ exits 0; `# fail 0`. `# pass` equals `(197 + tests added in ticket 01) + 2` (the two
new tests from Step 5); no previously-passing test regressed.

Also: `node --check storage/workspace-vault.js app.js` → exits 0, and `git diff --check`
→ no whitespace errors.

## Done criteria

Machine-checkable. ALL must hold:
- [ ] `grep -rn "ROOT_CANVAS_PATH" storage/ app.js` → no matches.
- [ ] `canvasPathFor` in `storage/workspace-vault.js` has a single parameter and no root branch.
- [ ] `grep -n "canvasPathFor(.*,.*rootId)\|canvasPathFor(.*,.*)" app.js` → no two-arg calls remain.
- [ ] `minimalFreshWorkspace` root path is `canvases/balaur.canvas`; `createGraphStarterWorkspace` sets root path `canvases/home.canvas`.
- [ ] `grep -n "canvases/root.canvas" storage/phase4.test.js storage/phase4-backup.test.js` → no matches.
- [ ] `grep -n "canvases/root.canvas" storage/phase9.test.js` → matches only in the lines 46–50 direct-write fixture.
- [ ] `node --check storage/workspace-vault.js app.js` exits 0.
- [ ] The full suite command in Step 8 exits 0 with `# fail 0` and no regression from the ticket-01 baseline.
- [ ] `git status --short` shows only the five in-scope files modified.

## STOP conditions

Stop and report back (do not improvise) if:
- The code at the locations in "Current state" does not match the excerpts (drift since
  `planned-at` 71899aa) — especially `canvasPathFor` (workspace-vault.js:29–32),
  `ROOT_CANVAS_PATH` (line 20), the `parseSidecar` root check (line 103), the seven
  app.js callers, or the test literals.
- A test file **other than** `phase4.test.js`, `phase4-backup.test.js`, `phase9.test.js`
  fails because of the root-path change. That would mean a fixture routes its root through
  the workspace machinery unexpectedly; stop and report which file/test, do not mass-edit.
- The full suite drops a previously-passing test that you cannot attribute to an intended
  assertion update in this ticket.
- You find you must change `storage/workspace-backup.js` or `storage/vault-path.js`.
- A step's verification fails twice after a reasonable fix attempt.

## Blocked by

- Ticket 01 (slug, unique-path, and rename primitives) — this ticket builds on the
  additive primitives and shares `storage/workspace-vault.js` and `storage/phase4.test.js`
  with it, so it runs after 01 lands.

## Next step

- Recommended executor tier: mid
- Recommended model: `pi/qwen-token-plan/qwen3.7-plus` (registry tiers are user-edited/TBD; orchestrator should confirm against `para/resources/model-registry.md`)
- Estimated complexity: M
