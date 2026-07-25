---
phase: ticket
status: done
project: 005-canvas-title-filenames
ticket: 03
blocked-by: [01, 02]
worker: "152ab791-9a9a-4ca9-8a95-dc9dc39956f7"
branch: "wealthy-blowfish"
shared-blast-radius: true
planned-at: 71899aa
---

# Ticket 03: Title-commit rename propagation + subcanvas creation dialog (browser-verified)

## What to build

Wire the ticket-01/02 primitives into the running app so canvas filenames follow their
titles. Two user-visible behaviors:

1. **Title-commit rename.** Editing the canvas title in `#canvasTitle` and committing
   (blur or Enter) renames the canvas file to the title slug, updates the sidecar `path`,
   and rewrites every `file`-node reference to the old path across all canvases. The old
   file is orphan-removed by the existing save logic. Intermediate keystrokes do not
   rename (the live `oninput` handler still syncs the title for display only).
2. **Subcanvas creation dialog.** Creating a subcanvas (toolbar button, add-menu, or the
   AI command) opens a native `<dialog>` asking for a name. Create with a non-empty name
   makes the canvas at a slug-derived, collision-free path; Cancel or an empty name aborts
   and creates nothing.

This ticket is verified in a real browser via the project `browser-check` skill, not Node
tests (the logic it composes is already Node-tested in tickets 01/02). It touches `app.js`
and `index.html` only.

## Current state

Confirm every excerpt before editing. Drift = STOP condition.

### `app.js`

- Imports to extend:
  - Line 4: `import { WorkspaceStore, hasWorkspace, canvasPathFor } from "./storage/workspace-vault.js";`
    → add `uniqueCanvasPath, renameCanvasPath`.
  - Line 10: `import { componentCardPath, slugify } from "./storage/vault-path.js";`
    → add `canvasSlug`. (Do not remove `slugify`; it is used elsewhere, e.g. line 1493.)
- Helpers already in scope you will reuse: `vaultStore` (170), `lifeIndex` (173),
  `lifeIndexer` (174), `canonicalWritable` (184), `enqueueMutation` (207),
  `flushPendingWorkspaceEdits` (240), `canvasRecord` (415), `renderWorkspaceNavigation`
  (484), `render` (1075), `uid` (138), `canvasPoint` (1090), `scheduleSave` (~234),
  `saveCurrentCanvasState` (~204), `toast`.
- `createSubcanvas(point)` today (lines 566–569) — synchronous, UID-based path, no prompt:
  ```js
  function createSubcanvas(point){
    if(!canonicalWritable){toast("Canonical files are read-only until repaired or restored");return null;}
    const center=point||canvasPoint(canvas.getBoundingClientRect().left+canvas.clientWidth/2,canvas.getBoundingClientRect().top+canvas.clientHeight/2),id=uid("canvas"),nodeId=uid("node"),siblings=Object.values(workspace.canvases).filter(record=>record.parentId===currentCanvasId).length,title=`New canvas ${siblings+1}`,node={id:nodeId,type:"file",x:Math.round(center.x-180),y:Math.round(center.y-125),width:360,height:250,color:"3",file:`canvases/${id}.canvas`};
    workspace.canvases[id]={id,title,parentId:currentCanvasId,portalNodeId:nodeId,path:node.file,document:{nodes:[],edges:[]},camera:null};documentData.nodes.push(node);selected={kind:"node",id:node.id};shell.classList.add("inspector-open");scheduleSave();render();toast("Sub-canvas created · double-click or zoom into it");return node;
  }
  ```
- The three call sites that must route through the dialog:
  - Line 1198 (inside `async function addNode`): `if(kind==="subcanvas")return createSubcanvas(point);`
  - Line 1780 (AI command, inside the async `runAssistant` flow):
    `} else if(/(?:add|create).*(?:sub.?canvas|nested canvas)/.test(lower)){createSubcanvas();response="Created a nested canvas portal. Double-click it or zoom into it to enter.";`
  - Line 1915: `$("#newCanvas").onclick=()=>createSubcanvas();`
  - Also exported at line 1981 on `window.orbitCanvas` as `createSubcanvas` (keep exporting it).
- `#canvasTitle` handlers today (line 1938):
  ```js
  $("#canvasTitle").oninput=()=>{saveCurrentCanvasState();scheduleSave();renderWorkspaceNavigation();};$("#canvasTitle").onblur=()=>{$("#canvasTitle").value=canvasRecord().title;};
  ```
- The flush-then-enqueue-then-save-then-reconcile pattern to imitate is `updateNoteBody`
  (~597–606) and `saveWorkspaceNow` (~212–218): `await flushPendingWorkspaceEdits();` then
  `await enqueueMutation(async () => { ...; await vaultStore.save(workspace); await
  lifeIndexer?.reconcileWarm(fromRevision); })`. Match that shape.

### `index.html`

- `#canvasTitle` input is at line 40; the `#newCanvas` toolbar button at line 55.
- Existing native dialogs to use as the structural exemplar (a `<dialog class="settings-dialog">`
  wrapping a `<balaur-dialog-frame>` wrapping a `<form>` with `<header>`, `.settings-body`,
  and `<footer>` with ghost Cancel + primary submit buttons): `#taskDialog` (lines 173–188)
  and `#aiNoteDialog` (lines 190–202). The dialog handlers in `app.js` for those
  (`$("#closeTaskDialog").onclick=$("#cancelTaskDialog").onclick=()=>$("#taskDialog").close();`
  and the `#taskForm`/`#aiNoteForm` submit handlers near line 1925) are the wiring exemplar.

### Conventions and constraints

- AGENTS.md §11: native `<dialog>`, semantic markup, useful accessible names, keyboard
  behavior (Enter submits, Esc cancels), `autofocus` the input. No framework, no build step.
- AGENTS.md §6: mutate through the existing persistence helpers; keep `documentData` and
  the sidecar record coherent; save the canonical file then reindex. The rename uses
  `renameCanvasPath` (in-memory) + `vaultStore.save` (which writes the doc to the new path,
  writes the sidecar last, and orphan-removes the old path via `WorkspaceStore._save`).
- AGENTS.md §10/§12: stop pointer propagation is not needed here (dialog, not a draggable
  card); disable/avoid double commits; surface failures via `toast` and the relevant status.
- The spec's ordering contract: write the document to the new path → `renameCanvasPath`
  in-memory → save the workspace. In practice `renameCanvasPath` updates `record.path`
  first and `vaultStore.save` performs the physical write to the new path plus orphan
  removal, so the commit handler is: capture state → set title → `renameCanvasPath` →
  `vaultStore.save`. No explicit `vault.remove` is needed.

## Scope

**In scope** (the only files to modify):
- `app.js` (modify — imports; async dialog-routing `createSubcanvas`; `promptCanvasTitle` helper; `commitCanvasTitle` handler + `#canvasTitle` blur/Enter wiring; AI call-site default title + conditional response)
- `index.html` (modify — add the `#subcanvasDialog` `<dialog>` beside `#taskDialog`)

**Out of scope** (do NOT touch):
- `storage/*` — the primitives and the contract are tickets 01/02; nothing changes here.
- Any `storage/*.test.js` — this ticket adds no Node tests (it is browser-verified).
- The live `#canvasTitle` `oninput` handler behavior (keep it exactly as-is).
- `switchCanvas`, `enterSubcanvas`, `leaveSubcanvas`, `activateCanvas` — unchanged.
- `window.orbitCanvas` surface beyond keeping `createSubcanvas` exported (do not add new
  globals or expose internals — AGENTS.md §12).

## Steps

### Step 1: Add the `#subcanvasDialog` markup to `index.html`

Insert a new `<dialog>` immediately after the `#taskDialog` block (after line 188), modeled
on `#taskDialog`/`#aiNoteDialog`. Single text input, Create/Cancel. Do **not** put
`required` on the input: an empty Create must abort silently (the submit handler trims and
an empty value resolves to "no creation"), per the spec.

```html
  <dialog class="settings-dialog" id="subcanvasDialog">
    <balaur-dialog-frame>
    <form id="subcanvasForm">
      <header><div><span>▦</span><div><b>New canvas</b><small>Name it so the file is easy to find</small></div></div><button type="button" id="closeSubcanvasDialog" aria-label="Close">×</button></header>
      <div class="settings-body">
        <label class="settings-field"><span>Canvas name</span><input id="subcanvasTitle" autofocus autocomplete="off" placeholder="City break"></label>
      </div>
      <footer><span></span><button type="button" class="button ghost" id="cancelSubcanvasDialog">Cancel</button><button type="submit" class="button primary" id="createSubcanvasButton">Create canvas</button></footer>
    </form>
    </balaur-dialog-frame>
  </dialog>
```

**Verify**: `grep -n "subcanvasDialog\|subcanvasTitle\|subcanvasForm" index.html` → the new
ids are present; the file still has balanced `<dialog>`/`</dialog>` tags
(`grep -c "<dialog" index.html` is one more than before).

### Step 2: Extend the `app.js` imports

- Line 4 → `import { WorkspaceStore, hasWorkspace, canvasPathFor, uniqueCanvasPath, renameCanvasPath } from "./storage/workspace-vault.js";`
- Line 10 → `import { componentCardPath, slugify, canvasSlug } from "./storage/vault-path.js";`

**Verify**: `node --check app.js` → exits 0.

### Step 3: Add a `promptCanvasTitle(initial)` helper

Add a helper (near `createSubcanvas`) that opens `#subcanvasDialog`, pre-fills and focuses
the input, and returns a Promise resolving to the trimmed title on Create, or `""` on
Cancel / × close / Esc / backdrop close. Guard against double-resolution and remove
listeners on settle so repeated calls do not leak handlers.

Target shape (load-bearing):

```js
function promptCanvasTitle(initial){
  return new Promise(resolve=>{
    const dialog=$("#subcanvasDialog"),form=$("#subcanvasForm"),input=$("#subcanvasTitle");
    const cancelBtn=$("#cancelSubcanvasDialog"),closeBtn=$("#closeSubcanvasDialog");
    input.value=initial||"";
    let settled=false;
    const finish=value=>{if(settled)return;settled=true;form.removeEventListener("submit",onSubmit);cancelBtn.removeEventListener("click",onCancel);closeBtn.removeEventListener("click",onCancel);dialog.removeEventListener("close",onClose);dialog.close();resolve(value);};
    const onSubmit=event=>{event.preventDefault();finish(input.value.trim());};
    const onCancel=()=>finish("");
    const onClose=()=>finish("");
    form.addEventListener("submit",onSubmit);
    cancelBtn.addEventListener("click",onCancel);
    closeBtn.addEventListener("click",onCancel);
    dialog.addEventListener("close",onClose);
    dialog.showModal();
    input.focus();input.select();
  });
}
```

(`finish` removes the `close` listener before calling `dialog.close()`, and the `settled`
flag makes the resulting `close` event a no-op even if ordering differs.)

**Verify**: `node --check app.js` → exits 0.

### Step 4: Rewrite `createSubcanvas` to route through the dialog with a slug path

Make `createSubcanvas(point, defaultTitle)` asynchronous: it guards writability, computes
the placement center and a default title, awaits `promptCanvasTitle`, and on a non-empty
title creates the canvas at `uniqueCanvasPath(canvasSlug(title, id), existingPaths)`. On
cancel/empty it returns `null` and creates nothing. Keep the existing placement, selection,
inspector, save, render, and toast behavior for the success path.

Target shape (load-bearing):

```js
function createSubcanvas(point, defaultTitle){
  if(!canonicalWritable){toast("Canonical files are read-only until repaired or restored");return Promise.resolve(null);}
  const center=point||canvasPoint(canvas.getBoundingClientRect().left+canvas.clientWidth/2,canvas.getBoundingClientRect().top+canvas.clientHeight/2);
  const siblings=Object.values(workspace.canvases).filter(record=>record.parentId===currentCanvasId).length;
  const initial=(defaultTitle&&defaultTitle.trim())||`New canvas ${siblings+1}`;
  return promptCanvasTitle(initial).then(title=>{
    if(!title)return null; // cancelled or empty -> abort, create nothing
    const id=uid("canvas"),nodeId=uid("node");
    const existingPaths=Object.values(workspace.canvases).map(record=>record.path).filter(Boolean);
    const path=uniqueCanvasPath(canvasSlug(title,id),existingPaths);
    const node={id:nodeId,type:"file",x:Math.round(center.x-180),y:Math.round(center.y-125),width:360,height:250,color:"3",file:path};
    workspace.canvases[id]={id:title,parentId:currentCanvasId,portalNodeId:nodeId,path,document:{nodes:[],edges:[]},camera:null};
    workspace.canvases[id].id=id;
    documentData.nodes.push(node);selected={kind:"node",id:node.id};shell.classList.add("inspector-open");scheduleSave();render();toast("Sub-canvas created · double-click or zoom into it");return node;
  });
}
```

(Note the record keeps `id` as the immutable canvas id and `title` as the user's title;
`path`/`node.file` are the slug path. Write the record cleanly as
`workspace.canvases[id]={id,title,parentId:currentCanvasId,portalNodeId:nodeId,path,document:{nodes:[],edges:[]},camera:null};` — the two-line form above is only to make the id/title split explicit.)

**Verify**: `node --check app.js` → exits 0.

### Step 5: Update the three `createSubcanvas` call sites

- Line 1198 (`addNode`): `if(kind==="subcanvas")return createSubcanvas(point);` — unchanged
  text is fine; `addNode` is async and now returns the promise. Leave as-is.
- Line 1915 (`#newCanvas`): `$("#newCanvas").onclick=()=>createSubcanvas();` — leave as-is
  (the returned promise is intentionally ignored here).
- Line 1780 (AI command): make it await the result and report success vs cancel, passing an
  editable default title. Replace the branch with:
  ```js
  } else if(/(?:add|create).*(?:sub.?canvas|nested canvas)/.test(lower)){const named=lower.match(/(?:named|called|titled)\s+(.+?)\.?$/);const node=await createSubcanvas(undefined,named?named[1].trim():undefined);response=node?"Created a nested canvas portal. Double-click it or zoom into it to enter.":"No canvas created.";
  ```
  (The captured name pre-fills the dialog and remains editable; with no name, the
  siblings-based default is used.)

**Verify**: `node --check app.js` → exits 0.

### Step 6: Add the `commitCanvasTitle` handler and rewire `#canvasTitle`

Add `commitCanvasTitle` (near the title handlers). It commits on blur/Enter: if the trimmed
title is unchanged or empty, reset the input and do nothing; otherwise claim the new title
synchronously (so a duplicate blur+Enter commit no-ops), compute a collision-free slug path
excluding the canvas itself, then flush pending edits and run the rename inside the mutation
queue: capture state → `renameCanvasPath` → `vaultStore.save` (writes doc to new path,
sidecar, orphan-removes old path) → reconcile the index. On error, restore the old title and
toast. Finally sync the input to the authoritative title.

Target shape (load-bearing):

```js
async function commitCanvasTitle(){
  const input=$("#canvasTitle"),record=canvasRecord();
  if(!record)return;
  const title=input.value.trim();
  if(!title||title===record.title){input.value=record.title;return;}
  if(!canonicalWritable||!vaultStore){input.value=record.title;return;}
  const oldTitle=record.title,id=record.id;
  record.title=title; // claim synchronously so a follow-up blur/Enter commit sees no change
  const existingPaths=Object.values(workspace.canvases).filter(r=>r.id!==id).map(r=>r.path).filter(Boolean);
  const newPath=uniqueCanvasPath(canvasSlug(title,id),existingPaths);
  try{
    await flushPendingWorkspaceEdits();
    await enqueueMutation(async()=>{
      saveCurrentCanvasState();
      renameCanvasPath(workspace,id,newPath);
      workspace.activeId=currentCanvasId;
      await vaultStore.save(workspace);
      const fromRevision=Number(lifeIndex?.getIndexState("indexedRevision")||0);
      await lifeIndexer?.reconcileWarm(fromRevision);
    });
    renderWorkspaceNavigation();render();
  }catch(error){
    record.title=oldTitle;
    console.warn("Could not rename canvas file",error);
    toast("Could not rename the canvas file");
  }finally{
    input.value=canvasRecord().title;
  }
}
```

Then replace the `#canvasTitle` wiring at line 1938. Keep `oninput` exactly as-is; replace
the `onblur` reset and add an Enter keydown:

```js
$("#canvasTitle").oninput=()=>{saveCurrentCanvasState();scheduleSave();renderWorkspaceNavigation();};
$("#canvasTitle").onblur=()=>commitCanvasTitle();
$("#canvasTitle").onkeydown=event=>{if(event.key==="Enter"){event.preventDefault();commitCanvasTitle();}};
```

**Verify**: `node --check app.js` → exits 0.

### Step 7: Static checks and the full Node suite (no regressions)

This ticket adds no Node tests, but it must not break any. Run:

**Verify**:
```
node --check app.js
node --test \
  storage/phase1.test.js storage/phase2.test.js storage/phase3.test.js \
  storage/phase4.test.js storage/phase4-backup.test.js storage/phase5.test.js \
  storage/phase7.test.js storage/phase8.test.js storage/phase9.test.js \
  storage/phase10.test.js storage/phase-query.test.js \
  storage/note-repository.test.js
```
→ `node --check` exits 0; the suite exits 0 with `# fail 0` and the same pass count as
after ticket 02 (no storage behavior changed here). Also `git diff --check` → clean.

### Step 8: Browser verification via the `browser-check` skill

Serve the app over HTTP (never `file://`), then drive the headless-Chrome skill from the
repo root. Read `.pi/skills/browser-check/SKILL.md` for the headless event-retargeting
caveat before driving dialog events.

1. Start the server in the background: `python3 -m http.server 4173 &` (from repo root).
2. Baseline smoke (hard gate):
   `node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline`
   → exit 0 (no uncaught console errors, all nodes render, file-index status present with
   no SQLite, selection frame works, live document is valid JSON Canvas, reload preserves
   the workspace, offline shell renders).
3. Slug root path in the browser (verifies ticket 02's app.js change is live). Load the
   graph starter then probe:
   `node .pi/skills/browser-check/scripts/browser-check.mjs eval "await window.orbitVaultReady; await window.orbitCanvas.loadGraphStarter?.(); (()=>{const ws=window.orbitCanvas.getWorkspace(); return ws.canvases[ws.rootId].path;})()"`
   → expect `"canvases/home.canvas"`.
4. Subcanvas Create via dialog produces a slug path. Probe (fills the input and submits the
   form, then awaits the promise returned by the exported `createSubcanvas`):
   ```
   node .pi/skills/browser-check/scripts/browser-check.mjs eval "await window.orbitVaultReady; (async()=>{const before=Object.keys(window.orbitCanvas.getWorkspace().canvases).length; const p=window.orbitCanvas.createSubcanvas(); document.querySelector('#subcanvasTitle').value='City break'; document.querySelector('#subcanvasForm').requestSubmit(); const node=await p; const ws=window.orbitCanvas.getWorkspace(); return {before, after:Object.keys(ws.canvases).length, file:node&&node.file};})()"
   ```
   → expect `after === before+1` and `file === "canvases/city-break.canvas"` (or
   `city-break-2.canvas` if a `city-break` canvas already exists in the loaded space).
5. Subcanvas Cancel aborts and creates nothing:
   ```
   node .pi/skills/browser-check/scripts/browser-check.mjs eval "await window.orbitVaultReady; (async()=>{const before=Object.keys(window.orbitCanvas.getWorkspace().canvases).length; const p=window.orbitCanvas.createSubcanvas(); document.querySelector('#cancelSubcanvasDialog').click(); const node=await p; return {node, before, after:Object.keys(window.orbitCanvas.getWorkspace().canvases).length};})()"
   ```
   → expect `node === null` and `after === before`.
6. Collision suffix: run the Create probe twice with the same name in the same space; the
   second result's `file` ends in `-2.canvas`.
7. Title-commit rename: set `#canvasTitle` to a new name, dispatch Enter (or blur), wait for
   the mutation to settle, then confirm the current canvas's `path` changed to the new slug
   and that a portal `file`-node in the parent canvas was rewritten to the new path. Example
   probe (adapt the wait to the skill's facilities):
   ```
   node .pi/skills/browser-check/scripts/browser-check.mjs eval "await window.orbitVaultReady; (async()=>{const input=document.querySelector('#canvasTitle'); const ws0=window.orbitCanvas.getWorkspace(); const id=ws0.activeId; input.value='Renamed canvas'; input.dispatchEvent(new KeyboardEvent('keydown',{key:'Enter',bubbles:true})); await new Promise(r=>setTimeout(r,600)); const ws=window.orbitCanvas.getWorkspace(); return {id, path:ws.canvases[id].path, title:ws.canvases[id].title};})()"
   ```
   → expect `title === "Renamed canvas"` and `path === "canvases/renamed-canvas.canvas"`.
   Then verify the old path no longer exists and cross-references were rewritten by
   inspecting `window.orbitCanvas.getWorkspace()` documents (no `file`-node still points at
   the old path).
8. Take a screenshot for visual review of the dialog:
   `node .pi/skills/browser-check/scripts/browser-check.mjs smoke --screenshot /tmp/balaur-005` (or the `shot` subcommand).

If a probe cannot be expressed through the skill's `eval` because of the headless
event-retargeting caveat, fall back to driving the same steps through a real browser profile
and record the observed result. Paste the probe outputs as evidence in your report.

**Verify**: smoke suite exit 0; probes 3–7 return the expected values; no uncaught console
errors during the interactions.

## Done criteria

Machine-checkable. ALL must hold:
- [ ] `node --check app.js` exits 0; `git diff --check` clean.
- [ ] The full Node suite (Step 7) exits 0 with `# fail 0` and no regression from the ticket-02 baseline.
- [ ] `index.html` contains `#subcanvasDialog` with `#subcanvasForm`, `#subcanvasTitle`, `#cancelSubcanvasDialog`, `#createSubcanvasButton`, `#closeSubcanvasDialog`.
- [ ] `createSubcanvas` is async, routes through `promptCanvasTitle`, and builds the path via `uniqueCanvasPath(canvasSlug(...))`; it is still exported on `window.orbitCanvas`.
- [ ] `#canvasTitle` keeps its `oninput` handler and gains a blur + Enter commit that calls `commitCanvasTitle`; the old `onblur` reset is gone.
- [ ] `grep -n "canvases/\${id}.canvas\|New canvas \${siblings" app.js` → the UID-based path is gone from `createSubcanvas`.
- [ ] Browser: `browser-check ... smoke --offline` exits 0.
- [ ] Browser probes: graph-starter root path is `canvases/home.canvas`; Create makes `canvases/city-break.canvas` (count +1); Cancel creates nothing (count unchanged, returns null); a duplicate name yields `-2.canvas`; a title commit renames the active canvas path to the new slug and rewrites cross-references.
- [ ] `git status --short` shows only `app.js` and `index.html` modified.

## STOP conditions

Stop and report back (do not improvise) if:
- The code at the locations in "Current state" does not match the excerpts (drift since
  `planned-at` 71899aa) — especially `createSubcanvas` (566–569), the three call sites
  (1198, 1780, 1915), the `#canvasTitle` handlers (1938), or the imports (4, 10).
- Tickets 01/02 are not already merged into this branch (their exports `canvasSlug`,
  `uniqueCanvasPath`, `renameCanvasPath` missing, or `canvasPathFor` still two-argument).
  Run `grep -n "export function canvasSlug" storage/vault-path.js` and
  `grep -n "export function uniqueCanvasPath\|export function renameCanvasPath" storage/workspace-vault.js`
  and `grep -n "canvasPathFor(record)" storage/workspace-vault.js` first; if any is missing, stop.
- The rename appears to require an explicit `vault.remove`/delete call to clear the old file
  (the existing `WorkspaceStore._save` orphan-removal should handle it). If the old file
  persists after a successful save, stop and report — do not add an ad-hoc delete.
- The smoke suite fails for a reason unrelated to your change (report the baseline failure).
- A browser probe fails twice after a reasonable fix attempt, or the skill cannot drive a
  dialog event even after consulting SKILL.md's caveat (report which interaction and why).
- You find you must modify a `storage/*` file or add a Node test to make the behavior work.

## Blocked by

- Ticket 01 (primitives) and Ticket 02 (root-path contract). This ticket composes both and
  shares `app.js` with ticket 02, so it runs after 02 lands.

## Next step

- Recommended executor tier: mid
- Recommended model: `pi/qwen-token-plan/qwen3.7-plus` (registry tiers are user-edited/TBD; orchestrator should confirm against `para/resources/model-registry.md`)
- Estimated complexity: M
