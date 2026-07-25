---
phase: implement
status: done
project: 005-canvas-title-filenames
ticket: 03
date: 2026-07-25
commit: 8a6b8cd, b8beb1a
branch: wealthy-blowfish
---

# Implementation: Title-commit rename propagation + subcanvas creation dialog

## Status: DONE — feature complete, browser-verified, smoke hard-gate green

The feature is fully implemented in the two in-scope files (`app.js`, `index.html`),
committed as `8a6b8cd`, passes the full Node suite (204/204), `node --check`, and
`git diff --check`, and is verified in the browser (ticket probes 3–7 all pass).

The ticket's Done criterion "Browser: `browser-check … smoke --offline` exits 0" initially
could not be met because the browser-check driver called `createSubcanvas` synchronously as a
test fixture, which the mandated async/dialog contract breaks. This was reported as a STOP;
the orchestrator authorized a scope expansion, and the driver's two fixture calls were updated
to drive the new dialog (committed separately as `b8beb1a`). `smoke --offline` now passes all
14 checks. See "Issues encountered" for the full history and a pre-existing, unrelated
`components`-subcommand failure that was surfaced during verification.

## Steps completed

- [x] Step 1: Added `#subcanvasDialog` markup to `index.html` after `#taskDialog` — verified: `grep` shows `subcanvasDialog`/`subcanvasForm`/`subcanvasTitle`/`cancelSubcanvasDialog`/`createSubcanvasButton`/`closeSubcanvasDialog` present; `<dialog` count 3→4, balanced `</dialog>` count 4.
- [x] Step 2: Extended `app.js` imports (`uniqueCanvasPath, renameCanvasPath` from workspace-vault; `canvasSlug` from vault-path) — verified: `node --check app.js` exit 0.
- [x] Step 3: Added `promptCanvasTitle(initial)` helper (Promise; resolves trimmed title on Create, `""` on Cancel/×/Esc/backdrop; `settled` guard + listener cleanup) — verified: `node --check app.js` exit 0.
- [x] Step 4: Rewrote `createSubcanvas(point, defaultTitle)` to async, route through `promptCanvasTitle`, build path via `uniqueCanvasPath(canvasSlug(title, id), existingPaths)`; cancel/empty returns `null` and creates nothing — verified: `node --check app.js` exit 0; UID-based `canvases/${id}.canvas` path gone (`grep` no match).
- [x] Step 5: Updated the AI call site to `await createSubcanvas(undefined, named?)` with success-vs-cancel response and an editable pre-filled name; the `addNode` (line ~1226) and `#newCanvas` (line ~1943) call sites left as-is per the ticket — verified: `node --check app.js` exit 0.
- [x] Step 6: Added `commitCanvasTitle` (flush → enqueueMutation: saveCurrentCanvasState → renameCanvasPath → vaultStore.save → reconcileWarm; restore-on-error; sync input in finally) and rewired `#canvasTitle`: `oninput` unchanged, `onblur`→`commitCanvasTitle`, new `onkeydown` Enter→`commitCanvasTitle`; old `onblur` reset removed — verified: `node --check app.js` exit 0; `grep` confirms old reset gone.
- [x] Step 7: Static checks + full Node suite — verified: `node --check app.js` exit 0; `git diff --check` clean; suite `# pass 204`, `# fail 0` (matches the ticket-02 baseline; no storage behavior changed).
- [x] Step 8: Browser verification — **complete**. Feature probes 3–7 all pass (evidence below). The `smoke --offline` hard-gate (8.2) passes all 14 checks (exit 0) after the authorized driver-fixture fix (`b8beb1a`). Screenshot (8.8) not captured: `shot` boots and captures before the dialog can be opened; functional coverage of the dialog is provided by probes 4–7.

## Files changed

- `app.js` — imports extended; `promptCanvasTitle` helper added; `createSubcanvas` rewritten to async + dialog + slug path; AI command call site awaits and reports success/cancel; `commitCanvasTitle` added; `#canvasTitle` blur/Enter commit wiring (oninput unchanged).
- `index.html` — added the `#subcanvasDialog` native `<dialog>` (form with autofocus input, ghost Cancel, primary Create) beside `#taskDialog`.
- `.pi/skills/browser-check/scripts/browser-check.mjs` — (authorized scope expansion, commit `b8beb1a`) the two `createSubcanvas` fixture calls (smoke step 5, components navigation setup) now drive the `#subcanvasDialog` and `await` the async result.

## Verification results

Static + Node (all from this session):

```
$ node --check app.js            # exit 0
$ git diff --check               # clean
$ node --test storage/phase1..10 + phase-query + note-repository
  tests 204 / pass 204 / fail 0   # identical to the ticket-02 baseline
```

Browser feature probes (headless Chrome via the browser-check `eval` subcommand,
each in a fresh temp profile; picker stubbed to an OPFS handle per SKILL.md):

- **Probe 3 — graph-starter root path** (ticket-02 change live):
  `{ rootId: "canvas-root", rootPath: "canvases/home.canvas", rootTitle: "Home" }`
- **Probe 4 — Create via dialog** (name "City break"; the graph starter already ships a
  `project-city-break` canvas at `canvases/city-break.canvas`, so the new canvas correctly
  collides): canvas count `6 → 7`, `file: "canvases/city-break-2.canvas"`.
- **Probe 6 — collision suffix** (second "City break" in the same space): count `7 → 8`,
  `file: "canvases/city-break-3.canvas"` (distinct suffix; collision avoidance proven against
  a real pre-existing path).
- **Probe 5 — Cancel aborts**: `cancelNode: null`, count unchanged (`8 → 8`); dialog closes.
- **Probe 7 — title-commit rename + cross-reference rewrite**: created "Trip" →
  `canvases/trip.canvas` (clean unsuffixed slug); entered it; set `#canvasTitle` to
  "Renamed trip" and dispatched Enter. Result:
  `{ newTitle: "Renamed trip", newPath: "canvases/renamed-trip.canvas",
     oldRefs: [], newRefs: ["canvas-root"] }`
  — the active canvas path/title changed, the parent (Home) portal `file`-node was rewritten
  to the new path, and **no** file-node anywhere still references the old `trip.canvas`.

Baseline comparison + smoke gate (this session): with `app.js`/`index.html` stashed,
`smoke --offline` on the pristine ticket-02 baseline passes all 14 checks; with the feature
applied but the driver still on the synchronous fixture it failed deterministically, which
isolated the driver fixture (not the feature logic) as the cause. After the authorized driver
fix (`b8beb1a`), `smoke --offline` passes all 14 checks (exit 0), re-confirmed post-restore.

## Issues encountered

### Resolved blocker: the browser-check driver used `createSubcanvas` as a synchronous fixture

The ticket makes `createSubcanvas` asynchronous and dialog-backed (it must await the user's
name choice). Two places in the browser-check driver relied on the OLD synchronous contract
and broke when the feature landed:

1. **Smoke suite, step 5** (`.pi/skills/browser-check/scripts/browser-check.mjs:344`): read
   `portalNode.id` synchronously; with the async contract `portalNode` is a Promise, so `.id`
   was `undefined` and the next eval threw `TypeError: Cannot read properties of null (reading
   'getBoundingClientRect')` (the `smoke --offline` failure, stack `<anonymous>:3:29`).
   Confirmed by probe: `createSubcanvas({x:0,y:0})` returned `[object Promise]`,
   `isPromise: true`, and opened the modal dialog.
2. **Components subcommand** (`.pi/skills/browser-check/scripts/browser-check.mjs:824`): read
   the new child id immediately after `createSubcanvas()`, before the dialog was submitted.

This was first reported as a STOP because the driver was outside the ticket's declared scope
("the only files to modify: app.js, index.html"). The orchestrator authorized a scope
expansion; the fix was applied and committed separately as `b8beb1a`.

### Resolution applied (commit b8beb1a)

Both fixture calls now drive the new `#subcanvasDialog`, mirroring the picker-stub and
task-dialog `requestSubmit()` patterns already in the driver: call `createSubcanvas`, set
`#subcanvasTitle` to a fixture name ("Smoke portal" / "Components child"),
`#subcanvasForm.requestSubmit()`, and `await` the returned promise before reading the result
(the two evaluated IIFes became `async`). After the fix, `smoke --offline` passes all 14
checks (exit 0), re-confirmed after restoring the working tree.

### Pre-existing failure (not caused by this ticket)

The `components` subcommand still fails at its **inspector** step
(`Timed out waiting for: document.querySelector('#inspector [data-field-key="text"]')`).
Verified pre-existing: with BOTH the feature (`app.js`/`index.html`) and the driver fix
reverted to the ticket-02 baseline (sync `createSubcanvas`, sync fixture), `components` fails
at the identical inspector step. The step clicks a text node and opens the inspector; it does
not touch `createSubcanvas`, and the failure predates this ticket. The ticket's Done criteria
require only `smoke --offline`, which is green. Flagged here for separate triage if the
components suite is expected to pass.

### Notes / deviations

- Probe 3 reads the root path directly after a fresh boot instead of calling
  `window.orbitCanvas.loadGraphStarter?.()`: a fresh empty vault auto-seeds the graph starter
  on boot (`openVault(vault, {seed:true})`), and `loadGraphStarter()` calls a native
  `confirm()` that blocks/returns-false in headless Chrome. The direct read still verifies
  ticket-02's live behavior (`canvases/home.canvas`).
- The graph starter ships a `City break` project canvas, so a fresh "City break" create
  collides to `city-break-2.canvas`; the ticket explicitly allows this ("or city-break-2.canvas
  if a city-break canvas already exists"). A unique name ("Trip") produces a clean
  `trip.canvas`, also demonstrated.
- No `storage/*` file was modified and no Node test was added (the ticket adds none; it is
  browser-verified). The rename required no explicit `vault.remove`: `WorkspaceStore._save`
  orphan-removed the old path (probe 7 shows `oldRefs: []` and the old path is gone).
