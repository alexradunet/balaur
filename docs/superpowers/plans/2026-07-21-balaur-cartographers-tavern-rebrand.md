# Balaur Cartographer’s Tavern Rebrand Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebrand the existing standards-first Orbit JSON Canvas application as Balaur, establish an app-local CSS design system and motion foundation, and apply the approved Cartographer’s Tavern identity without changing portable data or workflow behavior.

**Architecture:** Keep the current static HTML, CSS, JavaScript, JSON Canvas, SQLite Wasm, and PWA architecture. Add one CSS token source (`styles/tokens.css`) and one motion recipe layer (`styles/motion.css`), migrate existing component styles to semantic custom properties, and keep native HTML controls. Progressive features must preserve immediate functional fallbacks.

**Tech Stack:** Semantic HTML, CSS custom properties, cascade layers, size container queries, native `<dialog>`, `inert`, View Transition API with fallback, vanilla ES modules, SVG, Service Worker/Cache API, SQLite Wasm.

## Global Constraints

- Read `docs/superpowers/specs/2026-07-21-balaur-cartographers-tavern-design.md` before editing.
- No framework, OCTANT package, component library, package manager, build step, CDN, or runtime dependency.
- Preserve `orbit:` Markdown markers, `orbit-*` browser storage keys, `window.orbitCanvas`, `window.orbitLifeStore`, workspace bundle `format: "orbit-workspace"`, SQLite schema, and JSON Canvas data.
- Visible product and assistant identity becomes Balaur; whole-space download filename becomes `.balaur.json`.
- Runtime token source is `styles/tokens.css`; do not add a hand-maintained duplicate JSON token file.
- Every author CSS rule belongs to `tokens`, `foundation`, `shell`, `canvas`, `components`, `themes`, `responsive`, or `motion`.
- Keep Newsreader, Work Sans, and JetBrains Mono; do not add fonts.
- Motion final state belongs to DOM state. Reduced motion skips View Transitions and makes state changes immediate.
- Native controls and dialogs keep their semantics, focus behavior, labels, and keyboard operation.
- Target 44 × 44 CSS pixel touch controls and a visible 2 px focus perimeter.
- UI verification is browser-driven. The repository has no UI test runner; do not add source-text or screenshot tests merely to test CSS plumbing.
- Each task must leave the application runnable from `python3 -m http.server 4173`.

---

### Task 1: Establish token and motion layers

**Files:**
- Create: `styles/tokens.css`
- Create: `styles/motion.css`
- Modify: `styles/layers.css:1`
- Modify: `styles/foundation.css:1-28`
- Modify: `index.html:12-20`
- Modify: `sw.js:1-18`

**Interfaces:**
- Consumes: self-hosted `@font-face` families from `vendor/pixel-loom/fonts.css`.
- Produces: `--balaur-*` primitive/semantic tokens and a `motion` cascade layer consumed by every later task.

- [ ] **Step 1: Declare the complete cascade order**

Replace `styles/layers.css` with:

```css
@layer tokens, foundation, shell, canvas, components, themes, responsive, motion;
```

- [ ] **Step 2: Create the CSS token source**

Create `styles/tokens.css` with this vocabulary. The legacy aliases at the bottom are explicitly temporary so Tasks 2–6 can migrate one file at a time without a broken intermediate commit.

```css
@layer tokens {
  :root {
    color-scheme: dark;

    --balaur-color-soot-950: #100d0b;
    --balaur-color-soot-900: #17100b;
    --balaur-color-oak-900: #24150c;
    --balaur-color-oak-800: #2e1a0e;
    --balaur-color-oak-700: #3a2112;
    --balaur-color-parchment-300: #d7c48f;
    --balaur-color-parchment-200: #eee0ba;
    --balaur-color-ink-900: #2a2015;
    --balaur-color-ink-700: #66583c;
    --balaur-color-bone-100: #f1e7d4;
    --balaur-color-bone-300: #cfc1aa;
    --balaur-color-gold-500: #f2c14e;
    --balaur-color-teal-400: #5ed0bd;
    --balaur-color-leather-600: #806438;
    --balaur-color-leather-800: #4d3322;
    --balaur-color-danger-500: #a65745;
    --balaur-color-success-500: #6f9f4b;
    --balaur-color-outline: #090503;

    --balaur-surface-page: var(--balaur-color-soot-950);
    --balaur-surface-map: var(--balaur-color-soot-900);
    --balaur-surface-oak: var(--balaur-color-oak-900);
    --balaur-surface-oak-raised: var(--balaur-color-oak-800);
    --balaur-surface-parchment: var(--balaur-color-parchment-300);
    --balaur-surface-parchment-raised: var(--balaur-color-parchment-200);
    --balaur-content-on-dark: var(--balaur-color-bone-100);
    --balaur-content-on-dark-muted: var(--balaur-color-bone-300);
    --balaur-content-on-paper: var(--balaur-color-ink-900);
    --balaur-content-on-paper-muted: var(--balaur-color-ink-700);
    --balaur-border-default: var(--balaur-color-leather-600);
    --balaur-border-subtle: var(--balaur-color-leather-800);
    --balaur-border-focus: var(--balaur-color-teal-400);
    --balaur-action-primary: var(--balaur-color-gold-500);
    --balaur-action-assistant: var(--balaur-color-teal-400);
    --balaur-status-danger: var(--balaur-color-danger-500);
    --balaur-status-success: var(--balaur-color-success-500);

    --balaur-font-display: "Newsreader", Georgia, serif;
    --balaur-font-body: "Work Sans", system-ui, sans-serif;
    --balaur-font-mono: "JetBrains Mono", ui-monospace, monospace;
    --balaur-text-xs: 0.75rem;
    --balaur-text-sm: 0.8125rem;
    --balaur-text-md: 1rem;
    --balaur-text-lg: 1.25rem;
    --balaur-text-xl: 2.125rem;

    --balaur-space-1: 4px;
    --balaur-space-2: 8px;
    --balaur-space-3: 12px;
    --balaur-space-4: 16px;
    --balaur-space-5: 24px;
    --balaur-space-6: 32px;
    --balaur-target-min: 44px;
    --balaur-radius-control: 2px;
    --balaur-radius-panel: 4px;
    --balaur-outline-width: 2px;
    --balaur-shadow-paper: 4px 5px 0 var(--balaur-color-outline);
    --balaur-shadow-overlay: 0 12px 0 rgb(5 3 2 / 82%);
    --balaur-texture-oak:
      repeating-linear-gradient(90deg, rgb(255 255 255 / 2%) 0 36px, rgb(0 0 0 / 14%) 37px 40px),
      linear-gradient(90deg, var(--balaur-color-oak-700), var(--balaur-color-oak-900) 48%, var(--balaur-color-oak-800));
    --balaur-texture-map:
      linear-gradient(rgb(200 169 107 / 4%) 1px, transparent 1px),
      linear-gradient(90deg, rgb(200 169 107 / 4%) 1px, transparent 1px),
      radial-gradient(circle at 68% 38%, rgb(104 69 33 / 13%), transparent 42%);

    --balaur-duration-instant: 0ms;
    --balaur-duration-press: 80ms;
    --balaur-duration-focus: 120ms;
    --balaur-duration-selection: 160ms;
    --balaur-duration-panel: 220ms;
    --balaur-duration-travel: 280ms;
    --balaur-ease-standard: cubic-bezier(.2, .8, .2, 1);
    --balaur-ease-enter: cubic-bezier(.16, 1, .3, 1);
    --balaur-ease-exit: cubic-bezier(.4, 0, 1, 1);
    --balaur-distance-press: 2px;
    --balaur-distance-panel: 16px;
    --balaur-scale-travel: .06;

    /* JSON Canvas 1.0 data-color compatibility palette. */
    --balaur-canvas-red: #ff7b78;
    --balaur-canvas-orange: #efa66a;
    --balaur-canvas-yellow: #e9d56b;
    --balaur-canvas-green: #7ee0a1;
    --balaur-canvas-cyan: #64cbd0;
    --balaur-canvas-purple: #a78bfa;

    /* Transitional aliases; Task 6 removes them after all consumers migrate. */
    --bg: var(--balaur-surface-page);
    --panel: var(--balaur-surface-oak);
    --panel-2: var(--balaur-surface-oak-raised);
    --line: var(--balaur-border-subtle);
    --muted: var(--balaur-content-on-dark-muted);
    --text: var(--balaur-content-on-dark);
    --accent: var(--balaur-action-primary);
    --sidebar: 250px;
    --inspector: 0px;
  }
}
```

- [ ] **Step 3: Create the motion recipe layer**

Create `styles/motion.css`. This file owns motion recipes; individual component files must eventually contain no duration/easing literals.

```css
@layer motion {
  :where(.button, .add-card, .nav-item, .tool, .zoom-tools button, .tiny-btn) {
    transition:
      color var(--balaur-duration-focus) var(--balaur-ease-standard),
      background-color var(--balaur-duration-focus) var(--balaur-ease-standard),
      border-color var(--balaur-duration-focus) var(--balaur-ease-standard),
      transform var(--balaur-duration-press) var(--balaur-ease-standard);
  }

  :where(.button, .add-card, .tool):active {
    transform: translateY(var(--balaur-distance-press));
  }

  .app-shell {
    transition: grid-template-columns var(--balaur-duration-panel) var(--balaur-ease-standard);
  }

  .ai-panel {
    transition:
      opacity var(--balaur-duration-panel) var(--balaur-ease-standard),
      transform var(--balaur-duration-panel) var(--balaur-ease-enter);
  }

  @media (prefers-reduced-motion: reduce) {
    :root {
      --balaur-duration-press: 0ms;
      --balaur-duration-focus: 0ms;
      --balaur-duration-selection: 0ms;
      --balaur-duration-panel: 0ms;
      --balaur-duration-travel: 0ms;
      --balaur-distance-press: 0px;
      --balaur-distance-panel: 0px;
      --balaur-scale-travel: 0;
    }

    *, *::before, *::after {
      scroll-behavior: auto !important;
      animation-duration: 0s !important;
      animation-delay: 0s !important;
      transition-duration: 0s !important;
      transition-delay: 0s !important;
    }

    ::view-transition-group(*),
    ::view-transition-old(*),
    ::view-transition-new(*) {
      animation-duration: 0s !important;
    }

    :where(.connection-preview path, .ai-card.running .node-accent, .ai-message.loading p::after) {
      animation: none;
    }
  }
}
```

- [ ] **Step 4: Narrow `foundation.css` to reset and platform preferences**

Replace its token block with body/focus/accessibility rules that consume semantic tokens. Preserve the reset.

```css
@layer foundation {
  *, *::before, *::after { box-sizing: border-box; }
  html { color-scheme: dark; scrollbar-color: var(--balaur-border-default) var(--balaur-surface-page); }
  html, body { width: 100%; height: 100%; margin: 0; overflow: hidden; }
  body {
    background: var(--balaur-surface-page);
    color: var(--balaur-content-on-dark);
    font: var(--balaur-text-sm)/1.45 var(--balaur-font-body);
    -webkit-font-smoothing: antialiased;
  }
  button, input, textarea, select { font: inherit; accent-color: var(--balaur-action-primary); }
  button { color: inherit; }
  :focus-visible { outline: 2px solid var(--balaur-border-focus); outline-offset: 2px; }
  ::selection { background: var(--balaur-action-primary); color: var(--balaur-content-on-paper); }

  @media (prefers-reduced-transparency: reduce) {
    :where(.topbar, .ai-panel) { backdrop-filter: none; }
  }
  @media (prefers-contrast: more) {
    :where(button, input, textarea, select, .canvas-node) { border-width: 2px; }
  }
  @media (forced-colors: active) {
    :focus-visible { outline-color: Highlight; }
    :where(.canvas-node, .settings-dialog, .ai-panel) {
      border-color: CanvasText;
      background: Canvas;
      color: CanvasText;
    }
  }
}
```

- [ ] **Step 5: Load the new files without removing Linen yet**

In `index.html`, add `styles/tokens.css` after `styles/layers.css` and `styles/motion.css` after `styles/responsive.css`. Keep `vendor/pixel-loom/tokens/linen.css` until Task 6 because untouched component rules still consume its names.

- [ ] **Step 6: Add the new styles to the offline shell and bump the cache**

Change `CACHE_NAME` to `orbit-shell-v2` and add `./styles/tokens.css` and `./styles/motion.css` in stylesheet load order. Keep the internal `orbit-shell-` cache prefix so the existing activation cleanup deletes version 1.

- [ ] **Step 7: Verify the foundation in the browser**

Run: `python3 -m http.server 4173`

Open `http://127.0.0.1:4173`. Expected:

- application loads with no console errors;
- existing seeded workspace and SQLite status appear;
- computed `--balaur-surface-map` is `#17100b`;
- focus outline is teal;
- reload succeeds after the Service Worker update.

- [ ] **Step 8: Commit**

```bash
git add index.html sw.js styles/layers.css styles/tokens.css styles/foundation.css styles/motion.css
git commit -m "Add Balaur token and motion foundations"
```

### Task 2: Rebrand identity, assistant, exports, and PWA assets

**Files:**
- Create: `icons/balaur.svg`
- Create: `icons/balaur-192.png`
- Create: `icons/balaur-512.png`
- Delete: `icons/orbit.svg`
- Delete: `icons/orbit-192.png`
- Delete: `icons/orbit-512.png`
- Modify: `index.html:6-11,25-28,36-41,126-145,199-210`
- Modify: `app.js:1,655-667,715-720,781-792`
- Modify: `manifest.webmanifest:3-15`
- Modify: `sw.js:33-36`
- Modify: `offline/register.js:13`
- Modify: `storage/life-store.js:32,208`

**Interfaces:**
- Consumes: Balaur identity palette from Task 1.
- Produces: one visible Balaur identity, one `Ask Balaur` control, Balaur PWA assets, and `.balaur.json` download naming while internal data namespaces remain unchanged.

- [ ] **Step 1: Replace visible product copy and mark up the familiar control**

Use this identity structure in `index.html`:

```html
<div class="brand-mark" aria-hidden="true"><span class="brand-coil"></span><span class="brand-eye"></span></div>
<div><strong>Balaur</strong><small>life atlas</small></div>
```

Use this single assistant control; do not add a second trigger:

```html
<button class="button assistant-button familiar-control" id="assistantButton" aria-label="Ask Balaur">
  <span class="familiar-glyph" aria-hidden="true">◒</span>
  <span class="familiar-label">Ask Balaur</span>
</button>
```

Change:

- document title to `Balaur — life on a canvas`;
- assistant aside label and heading to `Balaur`;
- provider notice to “It is never sent to Balaur or GitHub Pages.”;
- keep `.orbit` in the import `accept` list for backward compatibility with old backups.

- [ ] **Step 2: Change only user-facing JavaScript strings and backup filename**

In `app.js`:

```js
async function exportWorkspace() {
  persistWorkspace();
  const store = await window.orbitLifeReady;
  const lifeData = store?.exportSnapshot?.() || null;
  downloadJSON(
    { format: "orbit-workspace", version: 1, exportedAt: new Date().toISOString(), workspace, lifeData },
    `${slug(workspace.canvases[workspace.rootId].title)}.balaur.json`
  );
  toast(`${Object.keys(workspace.canvases).length} canvases and life data exported`);
}
```

The object’s `format` must remain `orbit-workspace`. Change import confirmation to “Import this Balaur space…”, and change the remote model prompt opening to:

```js
return `You are Balaur, an assistant operating a JSON Canvas 1.0 life-management canvas. Respond with exactly one JSON object and no markdown fences: {"message":"Brief response to the user","operations":[]}.
```

Also change the opening code comment to Balaur, `offline/register.js`’s warning to “Balaur could not enable offline mode”, and the two user-visible life-store errors to “this Balaur build” / “Could not initialize the Balaur life database”. Do not rename `WORKSPACE_KEY`, AI storage keys, `window.orbitCanvas`, life-store globals, events, marker regexes, or cache prefixes.

- [ ] **Step 3: Create the cartographer icon**

Create `icons/balaur.svg`:

```svg
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" role="img" aria-label="Balaur">
  <rect width="512" height="512" rx="64" fill="#100d0b"/>
  <circle cx="256" cy="256" r="172" fill="#17100b" stroke="#806438" stroke-width="12"/>
  <path d="M132 319c37 66 151 79 215 17 55-54 37-142-25-164-50-18-104 10-106 54-2 36 40 57 70 36 19-13 20-39 5-53" fill="none" stroke="#f2c14e" stroke-width="24" stroke-linecap="square"/>
  <path d="M284 168l48-49 3 49 46-15-24 43 36 27-56 7-27-31z" fill="#5ed0bd" stroke="#090503" stroke-width="10" stroke-linejoin="miter"/>
  <circle cx="337" cy="188" r="8" fill="#100d0b"/>
  <path d="M117 351l40-9m194 17 38 14M105 249l41 4" stroke="#806438" stroke-width="10"/>
</svg>
```

Render exact 192 × 192 and 512 × 512 PNGs from the SVG using the browser screenshot tool. Confirm each PNG’s dimensions before deleting the Orbit icons.

- [ ] **Step 4: Update manifest and offline cache paths**

Set manifest name/short name to Balaur, theme/background to `#100d0b`, and icon paths to `icons/balaur-*`. Update `sw.js` paths to the same filenames and bump `CACHE_NAME` to `orbit-shell-v3`; keep `widgets/focus-orbit.html` because “orbit” there describes the visualization, not the product namespace.

- [ ] **Step 5: Verify identity and compatibility**

Browser checks:

- tab title, installed-app metadata, brand, assistant heading, and provider notice say Balaur;
- exactly one `Ask Balaur` control exists;
- export filename ends in `.balaur.json`;
- exported JSON still contains `"format": "orbit-workspace"`;
- re-importing that backup succeeds;
- existing local workspace loads without reset.

- [ ] **Step 6: Commit**

```bash
git add index.html app.js offline/register.js storage/life-store.js manifest.webmanifest sw.js icons/balaur.svg icons/balaur-192.png icons/balaur-512.png
git rm icons/orbit.svg icons/orbit-192.png icons/orbit-512.png
git commit -m "Rebrand Orbit identity as Balaur"
```

### Task 3: Build the carved-oak shell and responsive familiar

**Files:**
- Modify: `styles/shell.css:1-67`
- Modify: `styles/motion.css`
- Modify: `styles/responsive.css:1-11`

**Interfaces:**
- Consumes: semantic surface, content, border, target, and motion tokens from Task 1; familiar markup from Task 2.
- Produces: stable top bar/library shell, compact capture shelf, and one global assistant entry point whose position changes by layout.

- [ ] **Step 1: Migrate shell geometry and materials to semantic tokens**

Keep the current grid and state classes. Replace vendor color tokens and blurred-glass chrome with oak:

```css
@layer shell {
  .app-shell {
    --sidebar: 250px;
    --inspector: 0px;
    display: grid;
    grid-template: 64px minmax(0, 1fr) / var(--sidebar) minmax(0, 1fr) var(--inspector);
    width: 100vw;
    height: 100dvh;
    background: var(--balaur-surface-page);
  }

  .app-shell.inspector-open { --inspector: 292px; }
  .app-shell.sidebar-closed { --sidebar: 0px; }

  .topbar {
    grid-column: 1 / -1;
    display: grid;
    grid-template-columns: var(--sidebar) minmax(0, 1fr) auto;
    align-items: center;
    border-block-end: 2px solid var(--balaur-color-outline);
    background: var(--balaur-texture-oak);
    box-shadow: inset 0 1px rgb(255 255 255 / 12%);
    z-index: 20;
  }

  .sidebar {
    grid-row: 2;
    grid-column: 1;
    border-inline-end: 2px solid var(--balaur-color-outline);
    background: var(--balaur-texture-oak);
    color: var(--balaur-content-on-dark);
    padding: var(--balaur-space-4) var(--balaur-space-3);
    overflow: hidden auto;
    display: flex;
    flex-direction: column;
    gap: var(--balaur-space-5);
    z-index: 10;
  }
}
```

Use logical border/padding properties throughout the shell except for the spatially docked familiar.

- [ ] **Step 2: Convert the capture cards into a compact carved shelf**

All nine actions remain visible in the same order. Use two compact columns and uniform 58 px minimum height; wide actions still span both columns.

```css
.add-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 6px; }
.add-card {
  position: relative;
  min-height: 58px;
  padding: 9px;
  border: 1px solid var(--balaur-border-default);
  border-radius: var(--balaur-radius-control);
  background: var(--balaur-surface-oak-raised);
  box-shadow: inset 1px 1px rgb(255 255 255 / 11%);
  color: var(--balaur-content-on-dark);
  text-align: start;
  cursor: pointer;
}
.add-card.wide {
  grid-column: 1 / -1;
  min-height: 48px;
  display: grid;
  grid-template-columns: 26px 1fr;
  grid-template-rows: auto auto;
}
```

Keep literal labels and descriptions. Do not add runes, quest names, or a hidden More menu.

- [ ] **Step 3: Style brand, navigation, buttons, and status without vendor tokens**

Use Newsreader only for the brand wordmark and current canvas title. Use JetBrains Mono for breadcrumbs, status, metadata, and section labels. Make primary controls carved gold/ink rather than gradients. Set button minimum height to 44 px where the full label is shown; compact spatial tool buttons are handled in Task 4.

Use the coiled cartographer mark rather than the previous orbital ellipse:

```css
.brand-mark {
  position: relative;
  width: 30px;
  height: 30px;
  color: var(--balaur-action-primary);
}
.brand-coil {
  position: absolute;
  inset: 4px;
  border: 2px solid currentColor;
  border-inline-start-color: transparent;
  border-radius: 50%;
  transform: rotate(-28deg);
}
.brand-coil::after {
  content: "";
  position: absolute;
  right: -4px;
  top: -3px;
  width: 8px;
  height: 8px;
  border: 2px solid var(--balaur-color-outline);
  background: var(--balaur-action-assistant);
  transform: rotate(32deg);
}
.brand-eye {
  position: absolute;
  right: 7px;
  top: 7px;
  width: 3px;
  height: 3px;
  background: var(--balaur-color-outline);
}
```

- [ ] **Step 4: Dock the single familiar responsively**

The button remains in `.top-actions` in DOM order but is visually docked near the map/minimap on desktop:

```css
.familiar-control {
  position: fixed;
  right: calc(var(--inspector) + 18px);
  bottom: 112px;
  z-index: 24;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: var(--balaur-target-min);
  border: 2px solid var(--balaur-color-outline);
  background: var(--balaur-surface-oak-raised);
  box-shadow: inset 1px 1px var(--balaur-border-default), 0 0 18px rgb(242 193 78 / 20%);
  color: var(--balaur-content-on-dark);
}
.familiar-glyph { color: var(--balaur-action-primary); font-size: 1.125rem; }
```

At `max-width: 620px`, return it to the top bar with `position: static`, hide only `.familiar-label`, retain `aria-label="Ask Balaur"`, and keep a 44 px square target.

- [ ] **Step 5: Move all shell transition declarations to `motion.css`**

Remove `transition` properties from `shell.css`. Extend `motion.css` for `.topbar`, breadcrumb buttons, app-view buttons, the familiar, and sidebar state using the Task 1 tokens. No duration literal may remain in `shell.css`.

- [ ] **Step 6: Verify shell behavior at both reference sizes**

Desktop 1440 × 1000:

- library width and inspector transition remain stable;
- all capture actions are visible;
- familiar sits above the minimap and shifts when inspector opens;
- focus is unobscured.

Mobile 390 × 844:

- brand collapse and top actions do not clip;
- familiar is the only assistant trigger and remains 44 px;
- sidebar and inspector behavior matches the existing application.

- [ ] **Step 7: Commit**

```bash
git add styles/shell.css styles/motion.css styles/responsive.css
git commit -m "Build the Balaur oak application shell"
```

### Task 4: Turn the canvas into an inked map with parchment objects

**Files:**
- Modify: `index.html:214-225`
- Modify: `app.js:87-95,359-408,248-250`
- Modify: `styles/canvas.css:1-68`
- Modify: `styles/motion.css`
- Modify: `styles/themes.css:1-8`

**Interfaces:**
- Consumes: Task 1 tokens/motion recipes and existing `selected`, `renderNodes()`, `switchCanvas()`, portal preview, group, widget, and task classes.
- Produces: map material, readable parchment nodes, group/widget exceptions, one-shot `selection-entering` state, and a reduced-motion-aware View Transition boundary.

- [ ] **Step 1: Add a non-interactive bearing element to the node template**

Insert immediately inside `<article class="canvas-node">`:

```html
<div class="selection-bearing" aria-hidden="true"><i></i><i></i></div>
```

Do not add focusable or labeled descendants.

- [ ] **Step 2: Make bearing entry state explicit in `renderNodes()`**

Add module state near the other render state:

```js
let renderedSelectionKey = null;
```

Immediately before the existing `nodeLayer.innerHTML = ""` statement, add:

```js
const selectionKey = selected?.kind === "node" ? selected.id : null;
const selectionEntering = selectionKey !== null && selectionKey !== renderedSelectionKey;
```

Replace the current selected-state expression in the node loop with:

```js
const isSelected = selectionKey === node.id;
element.classList.toggle("selected", isSelected);
element.classList.toggle("selection-entering", isSelected && selectionEntering);
```

Immediately before the existing `updateCounts()` call, add:

```js
renderedSelectionKey = selectionKey;
```

Do not duplicate or omit the current node-type branches, event bindings, `updateCounts()`, or `renderMinimap()`.

- [ ] **Step 3: Introduce one reduced-motion-aware View Transition boundary**

Add:

```js
const reducedMotion = matchMedia("(prefers-reduced-motion: reduce)");

function updateWithViewTransition(update) {
  if (reducedMotion.matches || !document.startViewTransition) {
    update();
    return null;
  }
  return document.startViewTransition(update);
}
```

Replace `switchCanvas()` with the following expanded form. The update executes exactly once, and the navigation data attribute is removed in both progressive and fallback branches.

```js
function switchCanvas(id, { direction = "in", focusNodeId = null, fit = false } = {}) {
  if (!workspace.canvases[id] || id === currentCanvasId) return;
  saveCurrentCanvasState();
  document.body.dataset.canvasNavigation = direction;
  const update = () => activateCanvas(id, { focusNodeId, fit });
  const transition = updateWithViewTransition(update);
  if (transition) {
    transition.finished.finally(() => delete document.body.dataset.canvasNavigation);
  } else {
    delete document.body.dataset.canvasNavigation;
  }
  scheduleSave();
}
```

- [ ] **Step 4: Replace the canvas field with the inked-map material**

Use component-local map variables so `themes.css` can alter the map without recoloring the oak shell:

```css
.canvas {
  --map-surface: var(--balaur-surface-map);
  --map-grid: rgb(200 169 107 / 8%);
  width: 100%;
  height: 100%;
  position: relative;
  overflow: hidden;
  outline: 0;
  background-color: var(--map-surface);
  background-image:
    linear-gradient(var(--map-grid) 1px, transparent 1px),
    linear-gradient(90deg, var(--map-grid) 1px, transparent 1px),
    radial-gradient(circle at 68% 38%, rgb(104 69 33 / 13%), transparent 42%);
  background-size: 24px 24px, 24px 24px, auto;
}
```

- [ ] **Step 5: Restyle ordinary nodes as parchment and preserve semantic markers**

```css
.canvas-node {
  --node-surface: var(--balaur-surface-parchment);
  --node-content: var(--balaur-content-on-paper);
  --node-muted: var(--balaur-content-on-paper-muted);
  position: absolute;
  overflow: hidden;
  border: 1px solid var(--balaur-border-default);
  border-radius: var(--balaur-radius-control);
  background: var(--node-surface);
  box-shadow: var(--balaur-shadow-paper);
  color: var(--node-content);
  user-select: none;
  cursor: grab;
}
.canvas-node.selected:not(.group-node) { overflow: visible; }
.node-content { height: 100%; overflow: hidden; padding: 17px 18px 14px; }
.node-kicker { color: var(--node-muted); font-family: var(--balaur-font-mono); }
.node-content h2,
.node-content h3,
.ai-card-title { color: var(--node-content); font-family: var(--balaur-font-display); }
.node-content p,
.node-content ul { color: var(--node-muted); font-family: var(--balaur-font-body); }
.node-accent {
  position: absolute;
  inset: 11px 11px auto auto;
  width: 9px;
  height: 9px;
  border: 2px solid var(--balaur-color-ink-900);
  border-radius: 50%;
  background: var(--balaur-border-default);
  z-index: 3;
}
```

Retain the six JSON Canvas color mappings by changing only `.node-accent` and `.node-kicker` color. Map data colors 1–6 to `--balaur-canvas-red`, `--balaur-canvas-orange`, `--balaur-canvas-yellow`, `--balaur-canvas-green`, `--balaur-canvas-cyan`, and `--balaur-canvas-purple`. The kicker text remains the color-independent type/status cue.

- [ ] **Step 6: Implement the Balaur bearing without obstructing controls**

```css
.selection-bearing {
  position: absolute;
  inset: -15px;
  display: none;
  pointer-events: none;
  z-index: 7;
}
.canvas-node.selected:not(.group-node) > .selection-bearing { display: block; }
.selection-bearing i {
  position: absolute;
  inset: 3px -2px;
  border: 1px solid var(--balaur-action-primary);
  border-radius: 50%;
  transform: rotate(-7deg);
}
.selection-bearing i + i {
  inset: -2px 4px;
  border-color: var(--balaur-action-assistant);
  transform: rotate(9deg);
}
```

Place connection handles at `z-index: 8` and keep their pointer events unchanged.

- [ ] **Step 7: Preserve group, portal, task, AI, and widget material exceptions**

- Group nodes: transparent oak wash, leather dashed boundary, no parchment shadow, no bearing ellipse.
- Portal nodes: parchment outer card with the existing dark inset `.portal-preview`.
- Task nodes: parchment base; status remains in kicker/footer and semantic marker.
- AI operators: parchment base with teal/gold marker; running state stays text-backed.
- Widgets: oak/parchment frame; iframe content remains untouched and sandboxed.

Use component-local variables rather than copying full node rules:

```css
.group-node {
  --node-content: var(--balaur-content-on-dark);
  --node-muted: var(--balaur-content-on-dark-muted);
  background: rgb(36 21 12 / 42%);
  border: 1px dashed var(--balaur-border-default);
  box-shadow: none;
}
.subcanvas-node { --node-surface: #cfb878; }
.html-widget { --node-surface: var(--balaur-surface-oak-raised); }
```

- [ ] **Step 8: Migrate edges, tools, minimap, and themes**

Use leather-muted edges, gold selected edges, and teal connection previews. Use oak tool wells with 44 px touch targets while permitting 30–32 px visual glyphs inside the target. Rewrite `themes.css` so warm/calm/contrast set only `.canvas` local variables:

```css
body[data-canvas-theme="calm"] .canvas {
  --map-surface: #0b1316;
  --map-grid: rgb(94 208 189 / 11%);
}
body[data-canvas-theme="contrast"] .canvas {
  --map-surface: #050505;
  --map-grid: rgb(255 255 255 / 28%);
}
```

Do not let a canvas theme recolor the application shell or parchment text.

- [ ] **Step 9: Centralize canvas motion**

Move `connection-flow`, AI-running shimmer, View Transition rules, and bearing entry keyframes from `canvas.css` into `motion.css`. Use token durations. Bearing animation applies only to `.selection-entering`:

```css
.selection-entering .selection-bearing i {
  animation: balaur-bearing-in var(--balaur-duration-selection) var(--balaur-ease-enter) both;
}
@keyframes balaur-bearing-in {
  from { opacity: 0; transform: rotate(-7deg) scale(.94); }
}
```

Give the second ring its own final 9° transform so it does not inherit the first ring’s angle. Reduced motion renders both rings immediately.

- [ ] **Step 10: Verify canvas behavior, not just appearance**

At 1440 × 1000:

- pan, centered zoom, fit, minimap, filters, drag, resize, connection handles, connect mode, edge selection, and deletion still work;
- selecting a new ordinary node draws once; rerendering the same selection does not replay;
- groups have region treatment; widgets and portals remain usable;
- sub-canvas navigation works with and without `document.startViewTransition`;
- reduced motion skips travel and bearing animation.

- [ ] **Step 11: Commit**

```bash
git add index.html app.js styles/canvas.css styles/motion.css styles/themes.css
git commit -m "Turn Balaur canvas into an inked map"
```

### Task 5: Apply tavern materials to Today, inspector, dialogs, and assistant

**Files:**
- Modify: `index.html:107-146,149-212`
- Modify: `styles/components.css:1-21`
- Modify: `styles/motion.css`

**Interfaces:**
- Consumes: Task 1 semantic tokens, Task 2 Balaur assistant naming, and existing Today/dialog/assistant DOM contracts.
- Produces: coherent secondary surfaces with literal copy, semantic state, and no component-local motion magic numbers.

- [ ] **Step 1: Remove decorative sequence numbers from Today**

The four projections are not sequential steps. Remove only the `01`, `02`, `03`, and `04` spans from the section headers; keep headings, supporting labels, IDs, and DOM order.

- [ ] **Step 2: Style the inspector as an oak ledger cabinet**

Keep its generated markup and field order. Use oak shell, inset dark fields, one divider per logical group, and semantic focus tokens:

```css
.inspector {
  grid-row: 2;
  grid-column: 3;
  border-inline-start: 2px solid var(--balaur-color-outline);
  background: var(--balaur-texture-oak);
  color: var(--balaur-content-on-dark);
  overflow: hidden auto;
}
.field :where(input, textarea, select) {
  width: 100%;
  border: 1px solid var(--balaur-border-default);
  border-radius: var(--balaur-radius-control);
  background: var(--balaur-color-soot-900);
  color: var(--balaur-content-on-dark);
}
.field :where(input, textarea, select):focus-visible { border-color: var(--balaur-border-focus); }
```

Do not wrap every field in a new panel.

- [ ] **Step 3: Turn Today into a tavern notice board**

Use oak for the page background and parchment for task sections. Keep Planned today, Overdue, Inbox & next, and Completed literal.

```css
.today-view {
  width: 100%;
  height: 100%;
  overflow: auto;
  background: var(--balaur-texture-oak);
  color: var(--balaur-content-on-dark);
  padding: 38px clamp(20px, 4vw, 58px) 70px;
}
.today-section,
.today-quick-add {
  border: 1px solid var(--balaur-border-default);
  border-radius: var(--balaur-radius-control);
  background: var(--balaur-surface-parchment);
  box-shadow: var(--balaur-shadow-paper);
  color: var(--balaur-content-on-paper);
}
.today-section > header {
  border-block-end: 1px solid var(--balaur-border-default);
  background: color-mix(in srgb, var(--balaur-surface-parchment) 88%, var(--balaur-color-leather-600));
}
.today-task { border-block-end: 1px solid rgb(128 100 56 / 48%); }
.today-task :where(.task-copy, .task-copy b) { color: var(--balaur-content-on-paper); }
.today-task :where(.task-copy small, .task-dates time) { color: var(--balaur-content-on-paper-muted); }
```

Give overdue sections a danger edge plus the existing “Needs a decision” text. Do not rely on red alone.

- [ ] **Step 4: Present dialogs as parchment folios**

Retain native `<dialog>`, existing forms, validation, close buttons, and focus. Use parchment/ink for the folio, light parchment wells for inputs, and an opaque soot backdrop. Remove `backdrop-filter` from the baseline; reduced-transparency users then need no special dialog fallback.

- [ ] **Step 5: Restyle the assistant without importing RPG dialogue**

Use oak panel chrome, parchment assistant messages, dark inset provider/tool details, and the existing plain message flow. The panel remains an `aside` with `inert` toggled by `setAssistantOpen()`.

```css
.ai-panel {
  background: var(--balaur-texture-oak);
  border: 2px solid var(--balaur-color-outline);
  border-radius: var(--balaur-radius-panel);
  box-shadow: var(--balaur-shadow-overlay);
}
.ai-message.assistant > :where(div > p, p) {
  background: var(--balaur-surface-parchment);
  color: var(--balaur-content-on-paper);
  border: 1px solid var(--balaur-border-default);
}
.ai-context,
.ai-operation-list,
.ai-form {
  background: var(--balaur-color-soot-900);
  color: var(--balaur-content-on-dark-muted);
}
```

- [ ] **Step 6: Move component animations to `motion.css`**

Move the existing `.ai-panel` transition, `.toast` transition, `.ai-message.loading p::after` animation, and `@keyframes thinking` into `motion.css`. Add these opening-only progressive hooks there; `components.css` may define static transforms for closed/open states but no duration/easing literals.

```css
@starting-style {
  .ai-panel[aria-hidden="false"],
  dialog[open] {
    opacity: 0;
    transform: translateY(var(--balaur-distance-panel));
  }
}
```

Opening, closing, focus, and final visibility must work when `@starting-style` is unsupported.

- [ ] **Step 7: Verify secondary flows**

Browser exercise:

- Today quick capture, task completion, and task-to-canvas reveal;
- inspector edits for node text, geometry, color, task metadata, and edges;
- task, Johnny Decimal, AI note, and provider dialogs, including invalid states;
- assistant open/close, suggestions, local response, settings, focus transfer, and inert state;
- parchment text contrast and focus visibility.

- [ ] **Step 8: Commit**

```bash
git add index.html styles/components.css styles/motion.css
git commit -m "Style Balaur planning and assistant surfaces"
```

### Task 6: Complete responsive, accessibility, and clean token cutover

**Files:**
- Modify: `index.html:12-20`
- Modify: `styles/tokens.css`
- Modify: `styles/foundation.css`
- Modify: `styles/shell.css`
- Modify: `styles/canvas.css`
- Modify: `styles/components.css`
- Modify: `styles/themes.css`
- Modify: `styles/responsive.css:1-11`
- Modify: `styles/motion.css`
- Modify: `sw.js:10-18`

**Interfaces:**
- Consumes: every migrated style from Tasks 3–5.
- Produces: final no-bridge design system, component-owned responsive rules, preference adaptations, and no dependency on Pixel Loom’s Linen color tokens.

- [ ] **Step 1: Make component composition container-owned**

Keep `.canvas-wrap` as the named inline-size container, renamed to `balaur-content`. Use `@container balaur-content (width <= 760px)` for Today column/row composition and task metadata placement. Keep viewport media queries for top-bar, sidebar, inspector, minimap, and assistant docking only.

- [ ] **Step 2: Enforce narrow layout and target sizes**

At `max-width: 850px`, sidebar starts closed and inspector narrows. At `max-width: 620px`, inspector is hidden as today, canvas tools move to the bottom, minimap/zoom cluster hide, dialogs become one-column, and Today stacks. Set touch targets to 44 px through padding or pseudo hit area without visually inflating every glyph.

- [ ] **Step 3: Verify focus cannot be obscured**

Add `scroll-padding-block` to scroll-owning Today, inspector, sidebar, and assistant regions. Ensure the bottom tool cluster and assistant do not cover the currently focused control. Focus indication remains a 2 px teal perimeter on oak, parchment, map, and inset wells.

- [ ] **Step 4: Finish motion preference behavior**

Under reduced motion:

- `updateWithViewTransition()` performs immediate updates;
- semantic durations/distances are zero;
- no connection flow, AI shimmer, thinking animation, bearing draw, panel travel, or button movement remains;
- final status text and state remain visible.

Under forced colors, preserve system-color borders and focus. Under more contrast, strengthen decorative boundaries without changing layout. Under reduced transparency, remove any remaining blur/translucency.

- [ ] **Step 5: Remove the temporary token bridge and Linen color dependency**

Before removal, search `styles/` for these legacy/vendor consumers:

```text
--bg|--panel|--panel-2|--line|--muted|--text|--accent|--background|--surface-container|--on-surface|--primary|--secondary|--tertiary|--outline|--font-headline|--font-body|--font-mono|--green|--red|--orange|--yellow|--cyan|--purple
```

Migrate every application consumer to `--balaur-*` or component-local variables. `vendor/pixel-loom/fonts.css` contains only self-hosted `@font-face` declarations; the final family aliases are `--balaur-font-display`, `--balaur-font-body`, and `--balaur-font-mono`.

Then:

- delete the temporary alias block from `styles/tokens.css`;
- remove `vendor/pixel-loom/tokens/linen.css` from `index.html`;
- remove the Linen path from `sw.js`;
- keep `vendor/pixel-loom/fonts.css` and font files.
- bump `CACHE_NAME` to `orbit-shell-v4` so installed copies receive the final cache manifest;

Expected search after removal: no legacy/vendor color token matches in `styles/`.

- [ ] **Step 6: Verify at the two reference viewports and preference modes**

Desktop 1440 × 1000 and mobile 390 × 844:

- no horizontal document overflow;
- map retains the majority of the viewport;
- all actions remain reachable;
- Today order is unchanged;
- dialogs fit without clipping;
- assistant control is single and consistent;
- focus and state remain understandable in reduced motion and forced colors.

- [ ] **Step 7: Commit**

```bash
git add index.html sw.js styles/tokens.css styles/foundation.css styles/shell.css styles/canvas.css styles/components.css styles/themes.css styles/responsive.css styles/motion.css
git commit -m "Complete Balaur responsive design system cutover"
```

### Task 7: Update documentation and verify the complete rebrand

**Files:**
- Modify: `AGENTS.md`
- Modify: `README.md`
- Modify: `docs/architecture.md`
- Modify: `docs/design-system.md`
- Modify: `docs/generative-canvas.md`
- Modify: `docs/life-data.md`
- Modify: `docs/offline.md`

**Interfaces:**
- Consumes: working application and final token/motion contracts from Tasks 1–6.
- Produces: accurate product/design-system documentation and end-to-end evidence for every acceptance criterion.

- [ ] **Step 1: Update user-facing product documentation**

Replace user-facing Orbit product names with Balaur across the listed documents and update `AGENTS.md`’s repository identity and `.balaur.json` guidance. Preserve literal technical references where compatibility requires them:

- `<!-- orbit:task ... -->`, `<!-- orbit:jd ... -->`, and `<!-- orbit:ai-card -->` markers;
- `orbit-workspace` bundle format;
- internal storage/global names when documenting compatibility;
- `widgets/focus-orbit.html` when referring to the file path or the “Focus orbit” visualization.

Change `.orbit.json` user guidance to `.balaur.json` and explain that the internal bundle format remains stable.

- [ ] **Step 2: Rewrite the existing design-system document around the actual implementation**

`docs/design-system.md` must document:

- Cartographer’s Tavern subject and material roles;
- six-color identity palette plus explicit text/ink pairs;
- Newsreader/Work Sans/JetBrains Mono roles;
- primitive → semantic → component-local token tiers;
- cascade layer order;
- Baseline/progressive-enhancement policy;
- native component boundary and why OCTANT/Web Components/Shadow DOM are not used here;
- container-query versus media-query rule;
- motion token table, API boundary, cancellation rule, and reduced-motion substitutions;
- DTCG 2025.10 compatibility posture: CSS source now, generated JSON only when tooling exists;
- links from the specification’s Research basis section.

Do not claim full WCAG 2.2 conformance; record the existing freeform move/resize alternative as a separate functional accessibility gap.

- [ ] **Step 3: Run the full desktop smoke scenario**

At 1440 × 1000:

1. Load seeded workspace and confirm SQLite ready.
2. Pan, zoom, fit, select, drag, resize, connect, and delete.
3. Enter/leave sub-canvas by Open, double-click, zoom threshold, breadcrumb, and `Alt+ArrowUp`.
4. Create note, task, Johnny Decimal entry, AI note, AI operator, widget, group, and sub-canvas.
5. Edit node/task/edge values in inspector.
6. Quick-capture and complete a task in Today.
7. Open/close every dialog and assistant surface.
8. Export/import `.canvas` and `.balaur.json`.
9. Confirm no console error or horizontal overflow.

- [ ] **Step 4: Run mobile, reduced-motion, and forced-color checks**

At 390 × 844:

- exercise library/inspector access, canvas tools, Today, dialogs, and assistant;
- verify 44 px touch targets and no clipping;
- emulate `prefers-reduced-motion: reduce` and confirm immediate portal/selection/panel changes;
- emulate `prefers-contrast: more` and inspect boundaries;
- emulate `forced-colors: active` where browser tooling supports it and confirm text, borders, and focus remain visible.

- [ ] **Step 5: Verify offline behavior**

Load once online, wait for Service Worker activation, switch the browser offline, and reload. Expected: shell, tokens, motion styles, fonts, icons, SQLite Wasm, workspace, and widget fallback load; network AI remains unavailable with explicit provider behavior.

- [ ] **Step 6: Capture and critique final screenshots**

Capture:

- canvas at 1440 × 1000;
- Today at 1440 × 1000;
- task dialog and Balaur assistant;
- canvas and Today at 390 × 844.

Compare with the approved Cartographer’s Tavern reference. Remove any grain, border, shadow, motif, or micro-label that competes with titles, state, or primary actions. Verify the signature remains the bearing/familiar pair rather than scattered decoration.

- [ ] **Step 7: Run final source checks**

Search expectations:

- user-facing `Orbit` in application sources, README, and current product docs: no matches; historical ADRs/plans/specs and explicitly documented internal compatibility names are excluded;
- `--balaur-duration-*` and `--balaur-ease-*`: definitions in `tokens.css`, consumers in `motion.css`;
- duration/easing literals in `shell.css`, `canvas.css`, `components.css`, and `responsive.css`: no matches;
- `vendor/pixel-loom/tokens/linen.css` in `index.html` or `sw.js`: no matches;
- internal `orbit:` markers, storage keys, globals, event names, cache cleanup prefix, and `orbit-workspace`: still present.

- [ ] **Step 8: Commit**

```bash
git add AGENTS.md README.md docs/architecture.md docs/design-system.md docs/generative-canvas.md docs/life-data.md docs/offline.md
git commit -m "Document the Balaur standards-first design system"
```
