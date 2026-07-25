---
phase: ticket
status: in-progress
project: 005-ui-guidelines-audit
ticket: 02
blocked-by: []
worker: "a85a55e"
branch: "ripe-gecko"
shared-blast-radius: false
planned-at: 1016146
---

# Ticket 02: Guidelines later batch — motion, scroll physics, dates, touch, safe areas

## What to build

The remaining mechanical fixes from the 2026-07-26 UI audit
(`para/projects/005-ui-guidelines-audit/audit.md`, "Later batch" section).
UI-only: `index.html`, six stylesheets, `app.js`, `elements/task-list.js`.
No storage layer, no Service Worker change (all fonts are already in
`APP_SHELL`; the file list is unchanged, so do NOT touch `sw.js` and do NOT
bump `CACHE_NAME`).

Follow AGENTS.md §11 and §12: rules stay in their existing named cascade
layers, use Balaur tokens (`--balaur-duration-*`, `--balaur-ease-*`), do not
mass-format `app.js` (single-line style, surgical edits only).

The branch is synced to the main tip (1016146), which includes the
STORAGE_UNAVAILABLE save-readonly fix in `markSaveResult` — do not disturb it.

## Steps (in order)

### A. Shell enablers (index.html)

1. Viewport meta (line 5): `content="width=device-width, initial-scale=1.0"` →
   `content="width=device-width, initial-scale=1.0, viewport-fit=cover"`
   (enables the `env(safe-area-inset-*)` values in step F).

2. Font preloads — insert immediately after the `<link rel="apple-touch-icon" ...>`
   line, before the stylesheets (all three files exist and are precached):

   ```html
   <link rel="preload" as="font" type="font/woff2" crossorigin href="vendor/pixel-loom/fonts/worksans-400-latin.woff2" />
   <link rel="preload" as="font" type="font/woff2" crossorigin href="vendor/pixel-loom/fonts/newsreader-600-latin.woff2" />
   <link rel="preload" as="font" type="font/woff2" crossorigin href="vendor/pixel-loom/fonts/jetbrainsmono-400-latin.woff2" />
   ```

3. Example placeholders end with `…`:
   - `placeholder="Review monthly budget"` → `placeholder="Review monthly budget…"`
   - `placeholder="Paste your API key"` → `placeholder="Paste your API key…"`

### B. Sidebar collapse becomes a transform drawer (shell.css, motion.css, responsive.css)

The desktop collapse currently animates `grid-template-columns` (a layout
animation). Unify it with the narrow-width drawer pattern: the grid track
snaps, the sidebar box slides with `transform`, and `visibility` flips after
the exit slide finishes.

4. `styles/shell.css` — in the `.sidebar` rule block, add an explicit width so
   the box no longer depends on the grid track: `width: var(--sidebar-width);`
   (add it next to the existing `grid-row`/`grid-column` declarations).

5. `styles/shell.css` — immediately after the `.app-shell.sidebar-closed { --sidebar: 0px; }`
   rule, add the desktop closed state:

   ```css
   .app-shell.sidebar-closed .sidebar {
     visibility: hidden;
     transform: translateX(-100%);
   }
   ```

6. `styles/motion.css` — delete these two rules entirely (the track now snaps):

   ```css
   .app-shell {
     transition: grid-template-columns var(--balaur-duration-panel) var(--balaur-ease-standard);
   }

   .topbar {
     transition: grid-template-columns var(--balaur-duration-panel) var(--balaur-ease-standard);
   }
   ```

7. `styles/motion.css` — replace the `.sidebar,\n  .inspector { transition: ... }`
   rule with a version that delays `visibility` on exit so the slide-out is
   visible, plus an open-state override that removes the delay on entry:

   ```css
   .sidebar,
   .inspector {
     transition:
       transform var(--balaur-duration-panel) var(--balaur-ease-enter),
       visibility 0s linear var(--balaur-duration-panel);
   }
   .app-shell:not(.sidebar-closed) .sidebar,
   .app-shell:not(.inspector-open) .inspector {
     transition-delay: 0s;
   }
   ```

8. `styles/motion.css` — the `.familiar-control` transition lists
   `right var(--balaur-duration-panel) var(--balaur-ease-standard),` but no
   rule anywhere changes `right` (verified: only position: static at narrow
   widths, which is not animatable). Delete the `right` entry; keep color,
   border-color, and transform.

9. `styles/responsive.css` — inside `@media (max-width: 850px)`, the
   `.app-shell.sidebar-closed .sidebar { visibility: hidden; transform: translateX(-100%); }`
   rule is now redundant with step 5. Delete that rule block only; keep the
   `.sidebar { position: fixed; ... }` and `.app-shell.sidebar-closed .sidebar`
   is fully removed (the base rule covers it). Keep the inspector drawer rules
   as they are.

### C. Scroll containment (three layers)

10. Add `overscroll-behavior: contain;` to these existing rule blocks:
    - `styles/shell.css`: `.sidebar`, `.canvas-icon-panel`
    - `styles/components.css`: `.inspector`, `.ai-messages`, `.settings-dialog`, `.today-view`
    - `styles/canvas.css`: `.add-menu-panel`

### D. Typography (shell.css, canvas.css, components.css)

11. `font-variant-numeric: tabular-nums;` on the counters:
    - `styles/shell.css`: `.nav-item em`, `.save-state`
    - `styles/canvas.css`: `.zoom-tools button`
    - `styles/components.css`: `.today-stats b`, `.task-dates time`, `.journal-date`

12. `text-wrap: balance;` (progressive enhancement) on:
    - `styles/shell.css`: `.vault-landing-wordmark`
    - `styles/components.css`: `.today-head h1`, `.today-section h2`

### E. Dates and motion in JavaScript (app.js, elements/task-list.js)

13. `app.js` — add a display formatter next to the existing `localDateISO`
    function (it is the display inverse; storage keeps `YYYY-MM-DD`):

    ```js
    const shortDate=iso=>new Intl.DateTimeFormat(undefined,{month:"short",day:"numeric"}).format(new Date(`${iso}T00:00:00`));
    ```

    Then in the task-node-footer template (the line containing
    `task.scheduledOn?\`Plan ${escapeHTML(task.scheduledOn)}\``), replace
    `escapeHTML(task.scheduledOn)` with `escapeHTML(shortDate(task.scheduledOn))`
    and `escapeHTML(task.dueOn)` with `escapeHTML(shortDate(task.dueOn))`.

14. `elements/task-list.js` — add the same one-line `shortDate` helper at
    module scope (this element shares no module with app.js; a one-line local
    helper is deliberate, not accidental duplication). In `#patchRow`, replace
    `` `Plan ${String(item.scheduledOn).slice(5)}` `` with
    `` `Plan ${shortDate(item.scheduledOn)}` `` and
    `` `Due ${String(item.dueOn).slice(5)}` `` with
    `` `Due ${shortDate(item.dueOn)}` ``.
    The `time.dateTime` attribute keeps the raw ISO value — only display
    text changes.

15. `app.js` — both `scrollIntoView({behavior:"smooth",block:"end"})` calls
    (in `assistantMessage` and `assistantProposal`) become
    `scrollIntoView({behavior:reducedMotion.matches?"auto":"smooth",block:"end"})`.
    The `reducedMotion` matchMedia const already exists near the top of the file.

16. `app.js` — unsaved-change guard. Three surgical edits:
    - At module scope near `let pendingSave=Promise.resolve();`, introduce a
      stable sentinel and use it: `const idleSave=Promise.resolve();let pendingSave=idleSave;`
      (replace the existing declaration; keep `let mutationQueue=Promise.resolve();` untouched).
    - In `persistWorkspace`, the settle handler `if(pendingSave===operation)pendingSave=Promise.resolve();`
      becomes `if(pendingSave===operation)pendingSave=idleSave;`.
    - Immediately above the existing `// Async vault writes cannot be awaited
      from beforeunload.` comment near the bottom of the file, add:

      ```js
      window.addEventListener("beforeunload",event=>{
        if(saveTimer!=null||journalSaveTimer!=null||pendingSave!==idleSave){
          flushPendingWorkspaceEdits();
          event.preventDefault();
          event.returnValue="";
        }
      });
      ```

      Keep the existing comment (it documents why the guard warns instead of
      awaiting). Note `!=null` is intentional: `saveTimer` starts `undefined`.

### F. Safe areas (shell.css, canvas.css, components.css)

17. Use the two-argument form so unsupported environments fall back to 0:
    - `styles/shell.css` `.familiar-control`: `right: 18px;` →
      `right: calc(18px + env(safe-area-inset-right, 0px));` and `bottom: 112px;` →
      `bottom: calc(112px + env(safe-area-inset-bottom, 0px));`
    - `styles/canvas.css` `.canvas-tools`: `bottom: 16px;` →
      `bottom: calc(16px + env(safe-area-inset-bottom, 0px));`
    - `styles/canvas.css` `.zoom-tools`: `left: 17px;` →
      `left: calc(17px + env(safe-area-inset-left, 0px));` and `bottom: 17px;` →
      `bottom: calc(17px + env(safe-area-inset-bottom, 0px));`
    - `styles/canvas.css` `.minimap`: `right: 17px;` →
      `right: calc(17px + env(safe-area-inset-right, 0px));` and `bottom: 17px;` →
      `bottom: calc(17px + env(safe-area-inset-bottom, 0px));`
    - `styles/components.css` `.ai-panel`: `inset: 76px 16px 16px auto;` →
      `inset: calc(76px + env(safe-area-inset-top, 0px)) calc(16px + env(safe-area-inset-right, 0px)) calc(16px + env(safe-area-inset-bottom, 0px)) auto;`

### G. Touch (foundation.css)

18. `styles/foundation.css` — add to the `body` rule:
    `-webkit-tap-highlight-color: transparent;`
    and add a new rule after the `button, input, textarea, select` font rule:

    ```css
    a, button, input, select, textarea { touch-action: manipulation; }
    ```

    Do not apply `touch-action` to `.canvas` (it owns its own pan/zoom gestures).

## Out of scope

- `sw.js` (fonts already precached; no file-list or cache-semantics change).
- `storage/**`, `widgets/**`, `elements/` other than task-list.js.
- The audit's three design-decision items: resize-handle keyboard path,
  URL/deep-link state sync, Title Case copy pass.
- Do not disturb the STORAGE_UNAVAILABLE block in `markSaveResult`.

## Verification (all required)

- `node --check app.js main.js` and `node --check elements/*.js`
- `git diff --check` → clean
- Grep gates:
  - `grep -c "grid-template-columns" styles/motion.css` → 0
  - `grep -n "right var" styles/motion.css` → no match (dead transition gone)
  - `grep -rc "overscroll-behavior: contain" styles/ | grep -v ":0"` → 3 files (shell, components, canvas)
  - `grep -rc "tabular-nums" styles/ | grep -v ":0"` → 3 files
  - `grep -c "env(safe-area-inset" styles/shell.css styles/canvas.css styles/components.css` → 2, 6, 3
  - `grep -c "viewport-fit=cover" index.html` → 1
  - `grep -c "preload" index.html` → 3
  - `grep -n "slice(5)" elements/task-list.js` → no match
  - `grep -c "idleSave" app.js` → 3
- Full Node suite: the AGENTS.md §13 command plus `server.test.mjs` (202 tests) → all pass
- Browser smoke (AGENTS.md §13): serve on 4173, run
  `node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline`,
  then a second smoke with `--width 380 --height 760`. Kill the server after.
- Sidebar drawer proof: with the server up, use the skill's `shot`/`eval`
  subcommands at default width to capture the shell, toggle
  `document.querySelector(".app-shell").classList.add("sidebar-closed")`,
  capture again, and confirm the closed shot shows the full-width canvas with
  no sidebar remnant. Include both PNG names in the artifact.
- Confirm `git diff --name-only HEAD^` never lists `sw.js`.

## Commit

One commit, staged paths only — never `git add -A`, never amend or reset
(AGENTS.md §16; a follow-up commit is always fine):

```
git add index.html styles/foundation.css styles/motion.css styles/shell.css \
  styles/canvas.css styles/components.css styles/responsive.css \
  app.js elements/task-list.js
```

Message: `005: guidelines later batch — transform drawer, scroll physics, Intl dates, touch, safe areas`.
Never push.

## Artifact

Write the implement artifact to
`para/projects/005-ui-guidelines-audit/tickets/02-later-batch-impl.md`
(steps completed with per-step verification, files changed, verification
results including the screenshot filenames, issues encountered). Commit the
artifact as a separate follow-up commit (`005: implement artifact for ticket 02`)
so the artifact can reference the real implementation commit SHA.
