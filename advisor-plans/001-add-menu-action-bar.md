# Plan 001: Move "Add to canvas" out of the sidebar into an Add menu on the canvas action bar

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `advisor-plans/README.md`.
>
> **Drift check (run first)**:
> `git diff --stat 0ec34dd..HEAD -- index.html app.js styles/shell.css styles/canvas.css styles/motion.css styles/responsive.css README.md docs/design-system.md`
> If any of those files changed since commit `0ec34dd`, compare the
> "Current state" excerpts below against the live code before proceeding; on
> a mismatch, treat it as a STOP condition.

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: LOW
- **Depends on**: none
- **Category**: direction (UI restructure)
- **Planned at**: commit `0ec34dd`, 2026-07-22

## Why this matters

Today all nine node-creation actions live in a nine-button grid at the top of
the sidebar. That grid dominates the library panel, duplicates affordances
that already exist elsewhere (the note tool, the `＋` add-sub-canvas button),
and forces creation through a panel that is hidden by default on narrow
viewports. The floating `.canvas-tools` bar at the top-center of the canvas is
the app's established action surface. This plan consolidates creation there:
the four tool buttons stay inline, and a single **＋ Add** control opens an
oak popover menu listing all nine actions with their colored glyph, name, and
one-line description. The sidebar is freed for navigation/library duty, and
creation stays one click away in every viewport — including the narrow shell
where the sidebar is an overlay.

This is a pure presentation/interaction change. JSON Canvas documents, the
workspace sidecar, SQLite, `sw.js`, and every `data-add` behavior are
unchanged — the same `addNode(kind)` / `openAINoteDialog()` code paths run.

## Current state

### The sidebar section being removed — `index.html:46–60`

```html
      <section aria-label="Add to canvas">
        <p class="section-label">ADD TO CANVAS</p>
        <div class="add-grid">
          <button class="add-card" data-add="note"><span class="add-icon note-icon">✦</span><b>Note</b><small>Thought or idea</small></button>
          <button class="add-card" data-add="goal"><span class="add-icon goal-icon">◎</span><b>Goal</b><small>Desired outcome</small></button>
          <button class="add-card" data-add="habit"><span class="add-icon habit-icon">↻</span><b>Habit</b><small>Daily practice</small></button>
          <button class="add-card" data-add="project"><span class="add-icon project-icon">◇</span><b>Project</b><small>Active work</small></button>
          <button class="add-card wide" data-add="task"><span class="add-icon task-icon">✓</span><b>Task</b><small>Action with status, schedule, and due date</small></button>
          <button class="add-card wide" data-add="ai-note"><span class="add-icon ai-note-icon">✎</span><b>AI note</b><small>Ask a question, then create a note</small></button>
          <button class="add-card wide" data-add="ai"><span class="add-icon ai-icon">✦</span><b>AI operator</b><small>Connected inputs → generated note</small></button>
          <button class="add-card wide" data-add="widget"><span class="add-icon widget-icon">◉</span><b>Live widget</b><small>Sandboxed HTML, CSS, or WebGL</small></button>
          <button class="add-card wide subcanvas-add" data-add="subcanvas"><span class="add-icon subcanvas-icon">↘</span><b>Sub-canvas</b><small>Enter by opening or zooming into it</small></button>
        </div>
      </section>
```

Note the `data-add` values (`note goal habit project task ai-note ai widget
subcanvas`), the glyph + `<b>` name + `<small>` description per button, and
the per-kind icon color classes (`note-icon`, `goal-icon`, …). All of these
carry over into the menu. Note also that `✦` is used twice (Note = orange,
AI operator = green) — the labels in the menu are what disambiguate them.

### The action bar being extended — `index.html:92–98`

```html
        <div class="canvas-tools" role="toolbar" aria-label="Canvas tools">
          <button class="tool active" data-tool="select" title="Select (V)">↖</button>
          <button class="tool" data-tool="pan" title="Pan (H)">✋</button>
          <span></span>
          <button class="tool" data-tool="connect" title="Connect nodes (C), or drag a card’s side handle">⌁</button>
          <button class="tool" data-tool="note" title="Add note (N)">T</button>
        </div>
```

The `<span></span>` is a divider. `.canvas-tools` lives inside `#canvas`,
after `#world`, so it paints above the nodes; it is `position: absolute;
top: 16px; left: 50%; transform: translateX(-50%)` (`styles/canvas.css:449–461`)
— the `transform` makes it the containing block for absolutely positioned
descendants, which is what the popover anchors against.

### The wiring being replaced — `app.js:943` (first statement)

```js
$$("[data-add]").forEach(button=>button.onclick=()=>button.dataset.add==="ai-note"?openAINoteDialog():addNode(button.dataset.add));
```

`addNode(kind)` (`app.js:652`) routes `subcanvas` → `createSubcanvas()` and
`task` → `openTaskDialog()`; every other kind drops a preset node at the
viewport center. `openAINoteDialog()` is at `app.js:913`. None of this
changes — the menu items keep the same `data-add` contract.

### Surrounding behavior you must not break

- **dblclick guard** (`app.js:625`): `if (event.target.closest?.(".canvas-tools,.zoom-tools,.minimap,.edges")) return;` — the menu lives *inside* `.canvas-tools`, so clicks in it are already covered. Do not edit this line.
- **Global keydown** (`app.js:971–981`): handles `v h c n 0 + - Space Delete Ctrl+K Ctrl+S Alt+↑`. Do not add or change shortcuts in this plan.
- **`setAppView`** (`app.js:549`) hides `#canvas` when the Today view shows; the toolbar (and menu) hide with it.
- **DOM helpers** (`app.js:79–80`): `const $ = (selector, root=document) => root.querySelector(selector);` and `$$` likewise — both accept a root. DOM consts are bound at module top (`app.js:115–120`: `canvas`, `world`, `nodeLayer`, `edgeLayer`, `shell`, `narrowShell`). `setAppView` is never called during module evaluation (only from event handlers), so calling a hoisted function from it is safe.
- **Focus ring** (`styles/foundation.css:13`): `:focus-visible { outline: 2px solid var(--balaur-border-focus); outline-offset: 2px; }` — global. Do **not** suppress it on menu items; hover/focus background is added *in addition*, matching how `.tool` behaves.

### CSS that moves or dies

`styles/shell.css:220–258` owns the sidebar grid (to delete) and the icon
glyph classes (to relocate):

```css
  .add-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; }
  .add-card { … }                 /* lines 221–232 */
  .add-card:hover { … }           /* 233 */
  .add-card b, .add-card small { … } /* 234 */
  .add-card b { … }               /* 235 */
  .add-card small { … }           /* 236–240 */
  .add-card.wide { … }            /* 241–248 */
  .add-card.wide .add-icon { … }  /* 249 */
  .add-card.wide b { … }          /* 250 */
  .add-card.wide small { … }      /* 251 */
  .add-icon { font: 500 17px var(--balaur-font-mono); }        /* 252 */
  .note-icon { color: var(--balaur-canvas-orange); }           /* 253 */
  .goal-icon, .task-icon { color: var(--balaur-canvas-red); }  /* 254 */
  .habit-icon, .ai-icon { color: var(--balaur-canvas-green); } /* 255 */
  .project-icon { color: var(--balaur-canvas-purple); }        /* 256 */
  .widget-icon, .ai-note-icon { color: var(--balaur-canvas-cyan); } /* 257 */
  .subcanvas-icon { color: var(--balaur-action-primary); }     /* 258 */
```

Keep `.section-label` — the CANVASES and LIBRARY headings still use it.

`styles/canvas.css:449–473` owns the bar and tool buttons (exemplar for the
menu's material: `2px solid var(--balaur-color-outline)` border,
`var(--balaur-surface-oak-raised)` background, `inset 1px 1px
var(--balaur-border-default)` sheen, `var(--balaur-radius-control)` radius).

**Trap #1** — `styles/canvas.css:462` is
`.canvas-tools span { width: 1px; margin: 5px 3px; background: var(--balaur-border-default); }`.
That is a *descendant* selector. The menu markup contains many `<span>`s
inside `.canvas-tools`; without a fix this rule would render every icon and
label as a 1px divider. It must become `.canvas-tools > span`.

**Trap #2** — `styles/foundation.css` has no `[hidden]` reinforcement, so an
author `display: grid` on the panel would defeat the `hidden` attribute (UA
`[hidden] { display: none }` loses to author origin). The plan includes an
explicit `.add-menu-panel[hidden] { display: none; }`.

`styles/motion.css:2,10` references `.add-card` in two press-feedback
`:where()` lists; those references swap to `.add-menu-item`. All transitions
live in `motion.css` (layer files like `canvas.css` hold static styles only —
follow that split). The existing entry-animation pattern is
`@starting-style` (`motion.css` block containing `.ai-panel[aria-hidden="false"], dialog[open]`).

`styles/responsive.css:76–77` (inside `@media (max-width: 620px)`):
`.canvas-tools { top: auto; bottom: 16px; }` and `.zoom-tools { display: none; }`
— at narrow widths the bar sits at the *bottom*, so the menu must open
upward there.

### Tokens to use (from `styles/tokens.css`)

`--balaur-surface-oak-raised`, `--balaur-surface-oak`, `--balaur-color-outline`,
`--balaur-border-default`, `--balaur-content-on-dark`,
`--balaur-content-on-dark-muted`, `--balaur-action-primary`,
`--balaur-radius-control` (2px), `--balaur-target-min` (44px),
`--balaur-shadow-overlay` (`0 12px 0 rgb(5 3 2 / 82%)` — the hard offset
shadow used for overlays), `--balaur-canvas-*` (the six JSON Canvas glyph
colors), motion tokens `--balaur-duration-panel` (220ms),
`--balaur-ease-standard`, `--balaur-ease-enter`, `--balaur-distance-panel` (16px).

### Repo conventions this work must match

- CSS: every rule belongs to a named cascade layer; `canvas`-layer rules go in `styles/canvas.css`, transitions/keyframes in `styles/motion.css`, viewport adaptations in `styles/responsive.css` (see `docs/design-system.md` "CSS organization"). Semantic `--balaur-*` tokens only; no literal colors.
- JS: vanilla strict ES modules, dense one-liner wiring in the block near the bottom of `app.js`, named functions above; reuse `$`/`$$`. No framework, no build step.
- A11y (AGENTS.md §11): native buttons, `role="toolbar"` already present, useful accessible names, keyboard behavior, `prefers-reduced-motion` already handled globally in `motion.css` (durations zero out — no extra work needed).
- Docs (AGENTS.md §14): user-visible control changes → `README.md`; component visual rules → `docs/design-system.md`. Both are in scope.
- No new files are created, so `sw.js`'s `APP_SHELL` list needs **no** change.

## Commands you will need

| Purpose | Command | Expected on success |
|---|---|---|
| JS syntax | `node --check app.js` | exit 0 |
| Whitespace | `git diff --check` | no output |
| Storage regression | `node --test storage/` | all 149 tests pass |
| Serve (manual/browser-check) | `python3 -m http.server 4173` | serves repo root at :4173 |
| Browser smoke (if skill present) | `node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline` | exit 0 |

The `browser-check` skill lives under the gitignored `.pi/` directory and may
be absent in your checkout (fresh clone / isolated worktree). Test for it
with `test -f .pi/skills/browser-check/scripts/browser-check.mjs`. If absent,
use the manual checklist in the Test plan — it is an equivalent gate, not a
fallback of lesser rigor.

## Scope

**In scope** (the only files you should modify):

- `index.html`
- `app.js`
- `styles/shell.css`
- `styles/canvas.css`
- `styles/motion.css`
- `styles/responsive.css`
- `README.md`
- `docs/design-system.md`
- `advisor-plans/README.md` (status row only)

**Out of scope** (do NOT touch, even though they look related):

- `sw.js`, `offline/register.js`, `manifest.webmanifest` — no new assets are introduced; the precache list must stay byte-identical.
- `storage/**`, `main.js`, `server.mjs`, `widgets/**`, `vendor/**` — persistence, boot order, and vendored code are untouched by a UI move.
- The sidebar's CANVASES and LIBRARY sections and the `sidebar-bottom` — they stay exactly as they are (including the `＋` add-sub-canvas button beside CANVASES; the duplication with the menu's Sub-canvas entry is pre-existing and intentional for now).
- The Today view's `＋ Add task` button and quick-add form.
- Keyboard shortcuts — no new shortcuts; existing ones unchanged.
- `addNode` presets, `createSubcanvas`, `openTaskDialog`, `openAINoteDialog` — behavior is unchanged; only the buttons that call them move.
- `plans/` — that directory belongs to the file-canonical migration; this plan lives in `advisor-plans/`.

## Git workflow

- Branch: `advisor/001-add-menu-action-bar`
- The repo mixes plain imperative subjects with `fix:`/`style:` prefixes; use a plain imperative subject, e.g. `Move add-to-canvas actions into an action-bar Add menu`.
- One commit per step is fine, or a single commit at the end — but never leave the tree broken between steps (step order below keeps CSS consumers covered before deletions).
- Do NOT push, open a PR, or deploy. Every push to `main` deploys to GitHub Pages; that is the user's call.

## Steps

### Step 1: Replace the sidebar section with the action-bar Add menu in `index.html`

1. Delete the entire block shown in "The sidebar section being removed"
   (`index.html:46–60`, from `<section aria-label="Add to canvas">` through
   its matching `</section>`).
2. On the `<aside class="sidebar" …>` tag, change
   `aria-label="Canvas tools and library"` to `aria-label="Canvas library"`
   (the "tools" half no longer lives there).
3. Replace the `.canvas-tools` block (`index.html:92–98`) with:

```html
        <div class="canvas-tools" role="toolbar" aria-label="Canvas tools">
          <button class="tool active" data-tool="select" title="Select (V)">↖</button>
          <button class="tool" data-tool="pan" title="Pan (H)">✋</button>
          <span></span>
          <button class="tool" data-tool="connect" title="Connect nodes (C), or drag a card’s side handle">⌁</button>
          <button class="tool" data-tool="note" title="Add note (N)">T</button>
          <span></span>
          <div class="add-menu">
            <button class="tool add-menu-toggle" id="addMenuToggle" aria-haspopup="menu" aria-expanded="false" aria-controls="addMenu" title="Add to canvas">＋</button>
            <div class="add-menu-panel" id="addMenu" role="menu" aria-label="Add to canvas" hidden>
              <button class="add-menu-item" role="menuitem" data-add="note"><span class="add-icon note-icon" aria-hidden="true">✦</span><span class="add-menu-text"><b>Note</b><small>Thought or idea</small></span></button>
              <button class="add-menu-item" role="menuitem" data-add="goal"><span class="add-icon goal-icon" aria-hidden="true">◎</span><span class="add-menu-text"><b>Goal</b><small>Desired outcome</small></span></button>
              <button class="add-menu-item" role="menuitem" data-add="habit"><span class="add-icon habit-icon" aria-hidden="true">↻</span><span class="add-menu-text"><b>Habit</b><small>Daily practice</small></span></button>
              <button class="add-menu-item" role="menuitem" data-add="project"><span class="add-icon project-icon" aria-hidden="true">◇</span><span class="add-menu-text"><b>Project</b><small>Active work</small></span></button>
              <button class="add-menu-item" role="menuitem" data-add="task"><span class="add-icon task-icon" aria-hidden="true">✓</span><span class="add-menu-text"><b>Task</b><small>Action with status, schedule, and due date</small></span></button>
              <button class="add-menu-item" role="menuitem" data-add="ai-note"><span class="add-icon ai-note-icon" aria-hidden="true">✎</span><span class="add-menu-text"><b>AI note</b><small>Ask a question, then create a note</small></span></button>
              <button class="add-menu-item" role="menuitem" data-add="ai"><span class="add-icon ai-icon" aria-hidden="true">✦</span><span class="add-menu-text"><b>AI operator</b><small>Connected inputs → generated note</small></span></button>
              <button class="add-menu-item" role="menuitem" data-add="widget"><span class="add-icon widget-icon" aria-hidden="true">◉</span><span class="add-menu-text"><b>Live widget</b><small>Sandboxed HTML, CSS, or WebGL</small></span></button>
              <button class="add-menu-item" role="menuitem" data-add="subcanvas"><span class="add-icon subcanvas-icon" aria-hidden="true">↘</span><span class="add-menu-text"><b>Sub-canvas</b><small>Enter by opening or zooming into it</small></span></button>
            </div>
          </div>
        </div>
```

Every glyph, label, description, icon-color class, and `data-add` value is
copied verbatim from the deleted sidebar buttons — verify against the
"Current state" excerpt, do not retype from memory.

**Verify**:
- `grep -c 'class="add-menu-item"' index.html` → `9`
- `grep -n 'add-grid\|add-card\|ADD TO CANVAS' index.html` → no output
- `grep -c 'data-add=' index.html` → `9`

### Step 2: Menu behavior in `app.js`

1. Insert after the `setTool` function (`app.js:646–650`, i.e. between
   `setTool` and `addNode`):

```js
const addMenuToggle=$("#addMenuToggle"),addMenuPanel=$("#addMenu");
function addMenuItems(){return $$(".add-menu-item",addMenuPanel);}
function openAddMenu(){addMenuPanel.hidden=false;addMenuToggle.setAttribute("aria-expanded","true");addMenuItems()[0]?.focus();}
function closeAddMenu(refocus=false){if(addMenuPanel.hidden)return;addMenuPanel.hidden=true;addMenuToggle.setAttribute("aria-expanded","false");if(refocus)addMenuToggle.focus();}
function toggleAddMenu(){addMenuPanel.hidden?openAddMenu():closeAddMenu();}
```

2. In `setAppView` (`app.js:549`), add `closeAddMenu();` as the first
   statement of the function body, so switching to Today never leaves the
   menu logically open while hidden. (Function declarations are hoisted and
   `setAppView` only runs from event handlers, so ordering is safe.)

3. Replace the first statement of the wiring line at `app.js:943` —
   `$$("[data-add]").forEach(button=>button.onclick=()=>button.dataset.add==="ai-note"?openAINoteDialog():addNode(button.dataset.add));` —
   with:

```js
$$("[data-add]").forEach(button=>button.onclick=()=>{closeAddMenu();button.dataset.add==="ai-note"?openAINoteDialog():addNode(button.dataset.add);});
addMenuToggle.onclick=toggleAddMenu;
document.addEventListener("pointerdown",event=>{if(!addMenuPanel.hidden&&!event.target.closest?.(".add-menu"))closeAddMenu();});
addMenuPanel.addEventListener("keydown",event=>{const items=addMenuItems(),index=items.indexOf(document.activeElement);
  if(event.key==="Escape"){event.preventDefault();closeAddMenu(true);return;}
  if(event.key==="ArrowDown"){event.preventDefault();items[(index+1)%items.length].focus();}
  if(event.key==="ArrowUp"){event.preventDefault();items[(index-1+items.length)%items.length].focus();}
  if(event.key==="Home"){event.preventDefault();items[0].focus();}
  if(event.key==="End"){event.preventDefault();items[items.length-1].focus();}});
addMenuToggle.addEventListener("keydown",event=>{if(event.key==="Escape")closeAddMenu(true);if(event.key==="ArrowDown"&&addMenuPanel.hidden){event.preventDefault();openAddMenu();}});
```

Leave the rest of line 943 (`$$('[data-app-view]')…`) untouched.

Behavior notes: outside-click closes without stealing focus; Escape closes
and returns focus to the toggle; activating any item closes first, then runs
the unchanged `data-add` code path; `Enter`/`Space` on items needs no code
(they are native buttons).

**Verify**:
- `node --check app.js` → exit 0
- `grep -n 'closeAddMenu' app.js` → matches in the function block, `setAppView`, the `[data-add]` wiring, and the two keydown listeners
- `node --test storage/` → all tests pass (regression gate — this change should not affect them; if any fail, they failed before your change; verify with `git stash` and report rather than "fixing" storage code)

### Step 3: Menu styles in `styles/canvas.css` (add before deleting anything)

1. Change the divider rule at `styles/canvas.css:462` from
   `.canvas-tools span {` to `.canvas-tools > span {` (Trap #1 — without
   this, every span in the menu renders as a 1px divider).

2. Insert after the `.tool:hover, .tool.active { … }` rule (`canvas.css:473`)
   and before `.zoom-tools`:

```css
  .add-menu { position: relative; }
  .add-menu-toggle[aria-expanded="true"] { background: var(--balaur-surface-oak); color: var(--balaur-action-primary); }
  .add-menu-panel {
    position: absolute;
    top: calc(100% + 9px);
    right: 0;
    z-index: 30;
    display: grid;
    gap: 2px;
    min-width: 250px;
    border: 2px solid var(--balaur-color-outline);
    border-radius: var(--balaur-radius-control);
    padding: 5px;
    background: var(--balaur-surface-oak-raised);
    box-shadow: inset 1px 1px var(--balaur-border-default), var(--balaur-shadow-overlay);
  }
  .add-menu-panel[hidden] { display: none; }
  .add-menu-item {
    display: grid;
    grid-template-columns: 26px 1fr;
    gap: 10px;
    align-items: center;
    border: 0;
    border-radius: 1px;
    padding: 7px 9px;
    background: transparent;
    color: var(--balaur-content-on-dark);
    text-align: start;
    cursor: pointer;
  }
  .add-menu-item:hover, .add-menu-item:focus-visible { background: var(--balaur-surface-oak); }
  .add-menu-item b { display: block; font-size: 11px; }
  .add-menu-item small { display: block; margin-block-start: 1px; color: var(--balaur-content-on-dark-muted); font-size: 9px; line-height: 1.25; }
  .add-icon { font: 500 17px var(--balaur-font-mono); }
  .note-icon { color: var(--balaur-canvas-orange); }
  .goal-icon, .task-icon { color: var(--balaur-canvas-red); }
  .habit-icon, .ai-icon { color: var(--balaur-canvas-green); }
  .project-icon { color: var(--balaur-canvas-purple); }
  .widget-icon, .ai-note-icon { color: var(--balaur-canvas-cyan); }
  .subcanvas-icon { color: var(--balaur-action-primary); }
```

The last eight rules are the icon classes *relocated* from `shell.css` (they
now belong with their only consumer). Do not add `outline: none` anywhere —
the global `:focus-visible` ring must survive.

**Verify**: `grep -n 'canvas-tools > span' styles/canvas.css` → 1 match;
`grep -n '\.canvas-tools span' styles/canvas.css` → no match.

### Step 4: Delete the dead sidebar CSS from `styles/shell.css`

Delete `styles/shell.css:220–258` — the contiguous block from
`.add-grid { display: grid; …` through `.subcanvas-icon { color: var(--balaur-action-primary); }`
(see the "CSS that moves or dies" excerpt). The block before it is
`.section-label { … }` (keep it — the CANVASES/LIBRARY headings use it); the
block after it starts with `.section-heading { … }` (keep it too).

**Verify**:
- `grep -rn 'add-card\|add-grid' styles/ index.html app.js README.md` → no output
- `grep -n 'add-icon\|note-icon' styles/shell.css` → no output (they live in `canvas.css` now)
- `grep -n 'section-label\|section-heading' styles/shell.css` → still present

### Step 5: Motion in `styles/motion.css`

1. In the first `:where()` list (line 2), replace `.add-card` with
   `.add-menu-item`:
   `:where(.button, .add-menu-item, .nav-item, .tool, .zoom-tools button, .tiny-btn) {`
2. In the `:active` list (line 10), replace `.add-card` with
   `.add-menu-item`:
   `:where(.button, .add-menu-item, .tool):active {`
3. Add a transition rule (next to the other surface transitions, e.g. after
   the `.ai-panel` block):

```css
  .add-menu-panel {
    transition:
      opacity var(--balaur-duration-panel) var(--balaur-ease-standard),
      transform var(--balaur-duration-panel) var(--balaur-ease-enter);
  }
```

4. Extend the existing `@starting-style` block so the menu drops out of the
   bar (upward-opening narrow-viewport variant is handled in step 6):

```css
  @starting-style {
    .ai-panel[aria-hidden="false"],
    dialog[open] {
      opacity: 0;
      transform: translateY(var(--balaur-distance-panel));
    }
    .add-menu-panel:not([hidden]) {
      opacity: 0;
      transform: translateY(calc(-1 * var(--balaur-distance-panel)));
    }
  }
```

`prefers-reduced-motion` already zeroes all durations globally in this file —
no additional reduced-motion work.

**Verify**: `grep -n 'add-card' styles/motion.css` → no output;
`grep -c 'add-menu' styles/motion.css` → `4` (two `:where` entries, transition, starting-style).

### Step 6: Narrow-viewport flip in `styles/responsive.css`

Inside the existing `@media (max-width: 620px)` block, directly after
`.canvas-tools { top: auto; bottom: 16px; }` (line 76), add:

```css
    .add-menu-panel { top: auto; bottom: calc(100% + 9px); }
    @starting-style { .add-menu-panel:not([hidden]) { transform: translateY(var(--balaur-distance-panel)); } }
```

(The bar sits at the bottom there, so the menu opens upward and enters from
below.)

**Verify**: `grep -n 'add-menu-panel' styles/responsive.css` → 2 matches inside the 620px block.

### Step 7: Documentation

1. `README.md` — three references to the sidebar:
   - Line 56: `| Add task | Select **Task** in the sidebar or **Add task** in Today |` → `| Add task | Select **＋ Add → Task** on the canvas action bar, or **Add task** in Today |`
   - Line 114: `Select **AI note** in the sidebar.` → `Select **＋ Add → AI note** on the canvas action bar.`
   - Line 118: `1. Add an **AI operator** from the left sidebar.` → `1. Add an **AI operator** via **＋ Add** on the canvas action bar.`
2. `docs/design-system.md`:
   - Append to the **Material roles** section (after the carved-oak
     paragraph):
     `The canvas action bar is carved oak afloat on the map: a centered rail of tool buttons ending in a single ＋ Add control whose menu is an oak-raised panel under the hard overlay shadow. The menu keeps the entity glyph palette — orange note, red goal/task, green habit/AI operator, purple project, cyan widget/AI note, gold sub-canvas — so node kinds stay recognizable at a glance.`
   - In the **Motion contract** bullet list, after the "panel travel" item:
     `- the Add menu drops from the action bar with short tokenized travel (flipped upward in the narrow shell);`

**Verify**: `grep -n 'in the sidebar' README.md` → only the two remaining
legitimate uses (`Export whole space` row and the SQLite status sentence) —
no "Add"/"AI note"/"AI operator" references to the sidebar remain.

### Step 8: Browser-level verification

Run the full gate appropriate to your environment (see "Commands you will
need" for the skill-presence test).

**With the browser-check skill** (serve first: `python3 -m http.server 4173`
in the repo root, background it):

```bash
node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline
node .pi/skills/browser-check/scripts/browser-check.mjs smoke --width 380
node .pi/skills/browser-check/scripts/browser-check.mjs eval 'document.querySelectorAll(".add-menu-item").length'
node .pi/skills/browser-check/scripts/browser-check.mjs eval '(() => { document.getElementById("addMenuToggle").click(); return { hidden: document.getElementById("addMenu").hidden, expanded: document.getElementById("addMenuToggle").getAttribute("aria-expanded") }; })()'
node .pi/skills/browser-check/scripts/browser-check.mjs eval '(() => { const before = window.balaurCanvas.getDocument().nodes.length; document.getElementById("addMenuToggle").click(); document.querySelector(".add-menu-item[data-add=note]").click(); return { before, after: window.balaurCanvas.getDocument().nodes.length, menuHidden: document.getElementById("addMenu").hidden }; })()'
node .pi/skills/browser-check/scripts/browser-check.mjs eval '(() => { document.getElementById("addMenuToggle").click(); document.activeElement.dispatchEvent(new KeyboardEvent("keydown", { key: "Escape", bubbles: true })); return document.getElementById("addMenu").hidden; })()'
node .pi/skills/browser-check/scripts/browser-check.mjs shot /tmp/add-menu-open.png
```

Expected: both smoke runs exit 0; `9`; `{ hidden: false, expanded: 'true' }`;
`after === before + 1` with `menuHidden: true`; `true`; and the screenshot
shows the oak menu anchored under the bar's right end. Open the screenshot
and visually confirm: colored glyphs, two-line rows, hard offset shadow, and
(at 380px, take a second shot) the menu opening upward above the bottom bar
fully inside the viewport.

**Without the skill** — manual checklist in a real browser (fresh temporary
profile) at `http://localhost:4173`:

1. App boots with no console errors; sidebar has no ADD TO CANVAS section; CANVASES/LIBRARY/`JD`/`＋` still work.
2. The bar shows `↖ ✋ | ⌁ T | ＋`; clicking `＋` opens the menu with 9 labeled rows and colored glyphs; `＋` stays highlighted while open.
3. Keyboard: `ArrowDown`/`ArrowUp` cycle rows (wrapping), `Home`/`End` jump, `Escape` closes and refocuses `＋`, `ArrowDown` on the closed toggle opens it.
4. Click outside the menu closes it without creating anything.
5. Each action still behaves: note/goal/habit/project/ai/widget drop their preset card at viewport center and select it; task opens the task dialog; AI note opens the AI-note dialog; sub-canvas creates a portal and toasts.
6. Double-click on empty canvas still creates a note; double-click inside a card still does nothing (regression guard).
7. Switch to Today and back — no stale open menu; narrow the window below 620px — bar moves to the bottom, menu opens upward, fully visible.
8. Reload — workspace, camera, and SQLite status (`SQLite <version> · local`) preserved.

**Verify**: every item above passes; `git status --short` lists only the
in-scope files.

## Test plan

This repo has no UI unit-test harness — the `node --test` suites under
`storage/` cover persistence only and serve here as a regression gate
(step 2). Browser-level behavior is tested per AGENTS.md §13 via the
browser-check skill or the manual checklist (step 8). Cases covered:

- happy path: menu opens/closes; every one of the 9 `data-add` actions still runs its original code path;
- regression: dblclick note creation on background and the on-card guard; toolbar tools unaffected; divider rendering (Trap #1); `hidden` actually hides (Trap #2);
- keyboard: open, cycle, jump, Escape-with-refocus;
- narrow shell: upward opening, no viewport overflow;
- persistence: reload preserves state (smoke suite / manual item 8).

## Done criteria

ALL must hold:

- [ ] `node --check app.js` exits 0; `git diff --check` clean
- [ ] `node --test storage/` passes (149 tests)
- [ ] `grep -rn 'add-card\|add-grid\|ADD TO CANVAS' index.html app.js styles/ README.md` → no matches
- [ ] `grep -c 'class="add-menu-item"' index.html` → `9`; `grep -c 'data-add=' index.html` → `9`
- [ ] `grep -n '\.canvas-tools span' styles/canvas.css` → no match; `canvas-tools > span` present
- [ ] Browser gate passed: skill smoke (`--offline` and `--width 380`) exit 0 with the eval probes returning expected values, **or** the full manual checklist passed
- [ ] `git status --short` shows only the in-scope files modified
- [ ] `advisor-plans/README.md` status row for plan 001 updated to DONE

## STOP conditions

Stop and report back (do not improvise) if:

- The code at the locations quoted in "Current state" doesn't match (drift since `0ec34dd`).
- The grep gate at the end of step 4 finds `.add-card`/`.add-grid`/`.add-icon` consumers outside the files listed in Scope — something else depends on the sidebar grid and the removal is no longer safe.
- `.canvas-tools` is no longer a transformed, absolutely positioned bar (the popover's anchoring model breaks).
- The browser smoke suite fails an assertion that also fails on the pristine tree (`git stash` and re-run to check) — pre-existing failure; report, don't fix.
- At 380px the upward menu cannot fit inside the viewport with the positioning in step 6 — do not invent a fixed-position/portal scheme; report so the design can be revisited.
- You feel the need to touch `sw.js`, `storage/`, or `main.js` — you don't; re-read Scope.

## Maintenance notes

- The `data-add` value set (`note goal habit project task ai-note ai widget subcanvas`) is now a contract shared by `index.html` (menu rows) and `addNode`/the wiring in `app.js`. Adding a node kind means: a preset in `addNode`, a menu row with matching `data-add`, an icon-color class in `canvas.css`, and a README mention.
- The menu is the app's first popover. If another popover ever appears (e.g. AI suggestion overflow), extract a shared popover pattern rather than forking `.add-menu`.
- `@starting-style` is progressive enhancement: browsers without it (check caniuse before assuming) simply show the menu without the drop — the `hidden` toggle works everywhere.
- Reviewer scrutiny points: the `> span` divider-selector change (any other direct-child `<span>` in `.canvas-tools` would still render as a divider — there are exactly two, both intentional); that no `outline: none` or `:focus-visible` suppression crept in; and that `role="menu"`/`role="menuitem"` semantics stay paired with the arrow-key/Escape handling in `app.js`.
- Deliberately deferred: per-item keyboard shortcuts, typeahead in the menu, tearing frequent actions (e.g. Task) out onto the bar inline, and de-duplicating the sidebar's `＋` add-sub-canvas button.
