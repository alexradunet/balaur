---
phase: review
status: done
ticket: 02
date: 2026-07-25
verdict: approved
---

# Standards review: ticket 02 — boot swap

## Verdict: APPROVED

Reviewed `git diff ab4432d..15d5a5c` (commits `602a937` app boot rewire,
`595df04` shell overlay + shell.css + sw.js v13, `15d5a5c` delete
indexeddb-vault.js + reword two adapter comments). Diff is exactly the seven
in-scope files; net app.js ±189, index.html +10, shell.css +47, sw.js +8,
indexeddb-vault.js -252.

## Verification run

- `node --check app.js storage/vault-store.js storage/workspace-vault.js
  sw.js storage/directory-vault.js storage/memory-vault.js` → all exit 0.
- Full explicit suite (`node --test` phase1–10 + phase-query +
  note-repository) → `tests 197, pass 197, fail 0` (count unchanged; this
  ticket adds no Node tests — DirectoryVault is browser-only).
- `git diff --check ab4432d..15d5a5c` → clean.
- Dead-reference sweep `grep -rn "indexeddb-vault\|IndexedDbVault"
  --include=*.js --include=*.mjs --include=*.html . | grep -v vendor/|teach/|plans/`
  → no output.
- `grep -n "localStorage" app.js` → only theme (`orbit-canvas-theme`, :1667/:1890)
  and AI-settings (`AI_SETTINGS_KEY`/`AI_SECRET_KEY`, :1788/:1789/:1798/:1930).

## Critical correctness points (all verified in code)

1. **Seam 2 BINDING** — `app.js:326` gate reads
   `typeof window.showDirectoryPicker!=="function"` at call time; `app.js:343`
   pick calls `window.showDirectoryPicker({mode:"readwrite"})` at call time.
   The only other occurrences (`:340`/`:341`) are comment prose. No module-scope
   capture, no imported reference, no const alias. PASS.
2. **Boot sequence** — gate (missing `showDirectoryPicker` OR missing
   `crypto.subtle` → `#vaultLandingMessage` + `#openVaultFolder.disabled` +
   `setIndexStatus("Files unavailable", …)` + `return`, no fallback) at
   `app.js:326-331`; pick handler wired at `:336`; `AbortError` swallowed
   silently at `:344`; on handle: `new DirectoryVault(handle)` (`:337`),
   `hadSidecar`/`empty` detection (`:338-339`), `openVault(vault,{seed:empty})`
   shared core (`:340`), hide landing + reveal shell (`:342-343`). Module tail
   (`:1965-1968`) keeps `vaultReady=bootCanvasApp();
   window.orbitVaultReady=vaultReady; await vaultReady;` then the
   `window.orbitCanvas` assignment — `orbitVaultReady` resolves immediately in
   gated mode (early `return`) with `vaultStore` left null; `orbitCanvas`
   exposed in both modes. PASS.
3. **openVault core** (`app.js:362-388`) — `WorkspaceStore` (`:363`),
   `hasWorkspace` (`:364`), `!had → store.migrate(seed?
   createGraphStarterWorkspace():minimalFreshWorkspace())` additive Adopt
   (`:365-366`), `store.load()` with empty-workspace guard (`:367-368`),
   assign `workspace`/`vaultStore`/`window.orbitVaultStore` + diagnostic
   `setCanonicalWritable` + `configureLifeRuntime` + `seedBundledWidget`
   (`:369-373`), `seed → seedGraphStarterEntities()` + reload (`:375-377`),
   catalog rebuilds (`:378`) and `setIndexStatus` (`:379-380`), post-boot
   render block (`:382-387`). Throws on failure; pick-handler catch routes to
   `#vaultLandingMessage` (`:356-358`), reload catch routes to read-only
   affordances (`:1935-1938`). PASS.
4. **F1 rewire** — `createGraphStarterWorkspace` (`app.js:50`) calls
   `minimalFreshWorkspace()` (not `freshWorkspace(rootDocument)`) and `return
   result` (not `return normalizeWorkspace(result)`); the next line
   `root.title="Home";root.document=rootDocument;root.camera=null;` overrides
   the starter's `title:"Balaur"`, empty document, and default camera — net
   root-canvas state identical to the old path. `normalizeWorkspace`'s
   JD-field deletes and non-hub/project `kind` deletes are no-ops (starter
   sets no JD fields; all `kind` values are `hub`/`project`); its `!record.path`
   backfill never fires (root gets `path:null` then `continue`; every
   `hub(...)` and `project-city-break` carry an explicit `path` at `:88`).
   `freshWorkspace`/`normalizeWorkspace` verified deleted. Behavior preserved.
   PASS.
5. **Deletion completeness** — `demoCanvas`, `loadDocument`, `freshWorkspace`,
   `normalizeWorkspace`, localStorage-reading `loadWorkspace`, `WORKSPACE_KEY`,
   `firstRun`, `orbit-canvas-v1`, `orbit-title`, `orbit-workspace-v1` all absent
   from app.js (grep returns nothing for each). `indexeddb-vault.js` deleted;
   zero `IndexedDbVault`/`indexeddb-vault` references in `*.js`/`*.mjs`/`*.html`
   outside `docs/adr/`, `teach/`, `plans/`, `para/`. Theme and AI-settings
   localStorage uses remain. PASS.
6. **Staging re-point** — `importCanvas` version-2 branch (`app.js:1505`):
   `stagingVault=new MemoryVault()`; `canonicalVault=vaultStore.vault`
   (`:1518`); `snapshot → canonicalVault.restore(snapshot) → new
   WorkspaceStore(canonicalVault)` reload (`:1519-1520`). `loadGraphStarter`
   (`app.js:431`): `stagingVault=new MemoryVault()`; `canonicalVault=
   vaultStore.vault` (`:436`, unchanged per plan). `exportWorkspace`
   (`app.js:1488`) unchanged (diff shows nothing for it). PASS.
7. **index.html** — `#vaultLanding` overlay (wordmark, hint,
   `#openVaultFolder.primary`, `#vaultLandingMessage role="status"`) is the
   first child of `<body>` (`index.html:25-30`); `.app-shell` carries
   `hidden inert` in markup (`:31`); `#reloadVault` and `#openAnotherVault`
   `nav-item` buttons added to `.sidebar-bottom` after `#resetDemo` (`:72-73`);
   `#importButton`/`#fileInput`/`#exportWorkspaceButton`/`#resetDemo` kept.
   PASS.
8. **styles/shell.css** — inside the single `@layer shell`; braces balanced
   107/107; the critical `.app-shell[hidden]{display:none}` and
   `#vaultLanding[hidden]{display:none}` overrides present (the grid/flex
   displays would otherwise defeat UA `[hidden]`); `.vault-landing` fixed
   full-viewport centered column, `z-index:100`, `--balaur-surface-page`
   background; wordmark/hint/message use Balaur type/status tokens; optional
   fade guarded by `@media (prefers-reduced-motion: no-preference)`. PASS.
9. **sw.js** — `CACHE_NAME="orbit-shell-v13"` (`:1`); `APP_SHELL` drops
   `indexeddb-vault.js`, adds `directory-vault.js` + `memory-vault.js`. Cross
   check (`comm -23` of `grep -o 'from "\./[^"]*"' app.js | sort -u` minus
   `APP_SHELL`) → empty: all 20 app.js static imports are precached. The three
   additionally-added modules (`journal-event-repository.js`,
   `note-catalog.js`, `note-repository.js`) are genuinely imported by app.js
   and were absent from the precache list; adding them satisfies the §3.3
   acceptance criterion and is required for offline boot (in-scope file). PASS.
10. **Reload/Open another** — `#reloadVault` (`app.js:1934`): `if(!currentVault)
    return; openVault(currentVault,{seed:false}).catch(...)` with
    `setIndexStatus("Files unavailable",…)` + `setCanonicalWritable(false,…)`
    on failure; runtime globals NOT nulled. `#openAnotherVault` (`:1941`):
    `flushPendingWorkspaceEdits()` then `shell.setAttribute("inert","")`,
    `shell.hidden=true`, `$("#vaultLanding").hidden=false`; the pick handler
    wired in `bootCanvasApp` stays attached so the next pick re-runs the
    first-open flow (calling `resolve()` on the already-resolved boot promise
    is a no-op, but `currentVault` is updated and the shell is re-revealed).
    PASS.

## Findings

- [judgement] `app.js:336-360`, `app.js:1934-1938` — AGENTS.md §12
  ("Disable asynchronous controls while running and surface errors in the
  relevant status region"): `#openVaultFolder` is not disabled while
  `openVault` runs after a pick resolves, and `#reloadVault` is not disabled
  while its `openVault` call is in flight, so a double-click could start a
  second concurrent open that races on the runtime globals. The plan §2.4
  shows the pick handler in exactly this shape, so the worker followed the
  spec; the risk is low (the folder picker is browser-modal, reload is
  idempotent over the same handle, and `#openAnotherVault` sets `inert` on
  the whole shell so its buttons are non-interactive during the gate). The
  same file's task-creation buttons (`#createTaskButton`, `#todayQuickAdd`
  submit) do follow §12 with `button.disabled=true`/`finally{...disabled=false}`,
  so the convention is applied elsewhere in app.js. A trivial spec-compatible
  fix would mirror that pattern. Not a blocker; flagging for awareness.

## Notes (not findings, no action needed)

- **ADR numbering**: the plan, spec, and review-assignment prose reference
  "ADR-0004-directory-vault-storage.md", but the actual file is
  `docs/adr/0005-directory-vault-storage.md` (ADR-0004 is
  `file-unified-node-model`). The code comment at `app.js:325` correctly says
  `(ADR-0005)`. The prose drift is in the planning artifacts, not the code.
- **§2.7 comment-reword scope**: the plan scoped the `workspace-vault.js:9`
  reword to the `IndexedDbVault→DirectoryVault` naming. The worker also
  corrected the same comment's now-false clause ("localStorage is consulted
  only once as a legacy-profile migration source" → "the browser boots from a
  user-picked folder with no localStorage migration"). Comment-only, logic
  untouched, and avoids shipping a known-false statement. Reasonable.
- **Four-catalog rebuild preserved**: the plan §2.4 step 7 lists three
  catalogs (predates ticket 04). The live code rebuilds four
  (`lifeIndexer`, `componentCardCatalog`, `widgetCatalog`, `noteCatalog`).
  The ticket says "the existing block"; regressing to three would break
  `noteCatalog`. Correctly preserved, documented in the impl artifact.
- No mass-formatting: every app.js edit lands in a spec'd region (§2.1–2.8,
  §3 wiring). The new `openVault` uses the compact one-line style that matches
  the surrounding `loadGraphStarter`/`importCanvas` bodies; the deleted
  more-spaced `bootCanvasApp` is gone. No churn outside spec'd regions.

## Summary

One low-severity [judgement] finding (async vault controls not disabled
while running, spec-sanctioned shape, trivially fixable); all 10 critical
correctness points pass in code; 197/197 Node tests green; syntax, whitespace,
dead-reference, and localStorage sweeps clean. APPROVED.
