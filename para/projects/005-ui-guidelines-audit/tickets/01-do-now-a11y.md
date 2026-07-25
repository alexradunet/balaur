---
phase: ticket
status: ready
project: 005-ui-guidelines-audit
ticket: 01
blocked-by: []
branch: "ripe-gecko"
shared-blast-radius: false
planned-at: 71899aa
---

# Ticket 01: Accessibility do-now batch (Vercel Web Interface Guidelines)

## What to build

Fix the sixteen mechanical accessibility and copy gaps found in the
2026-07-26 UI audit (`para/projects/005-ui-guidelines-audit/audit.md`,
"Do-now batch" section). UI-only: `index.html`, two stylesheet rules, one
app.js handler. No storage layer, no behavior change, no Service Worker
cache-semantics change (the file list is unchanged and the runtime strategy
is network-first, so `CACHE_NAME` must NOT be bumped; do not touch `sw.js`).

Follow AGENTS.md §11 and §12: rules go in the existing named cascade layers,
use Balaur tokens, do not mass-format `app.js`.

## Steps (in order)

1. `index.html` canvas tools — add aria-labels, keep the title tooltips:
   - `<button class="tool active" data-tool="select" title="Select (V)">` → add `aria-label="Select"`
   - `<button class="tool" data-tool="pan" title="Pan (H)">` → add `aria-label="Pan"`
   - `<button class="tool" data-tool="connect" title="Connect nodes (C), or drag a card’s side handle">` → add `aria-label="Connect"`
   - `<button class="tool" data-tool="note" title="Add note (N)">` → add `aria-label="Add note"`

2. `index.html` zoom tools (the line with `id="zoomLabel"`):
   - `<button id="zoomLabel" title="Reset zoom">100%</button>` → add `aria-label="Reset zoom to 100%"`
   - `<button id="fitView" title="Fit canvas">⌗</button>` → add `aria-label="Fit canvas to view"`

3. `index.html` sidebar: `<button class="tiny-btn" id="newGroup" title="Add group">＋</button>` → add `aria-label="Add group"`.

4. `index.html` minimap — replace the clickable div with a button:
   `<div class="minimap" id="minimap">` → `<button type="button" class="minimap" id="minimap" aria-label="Fit canvas to view">`
   (closing tag `</div>` → `</button>`; inner `mini-world`/`mini-viewport` divs unchanged).
   `app.js` already wires `$("#minimap").onclick=fitView;` — a button makes it
   keyboard-reachable; do not change app.js for this.

5. `index.html` live regions:
   - `<span class="save-state" id="saveState">` → add `role="status"`
   - `<span id="lifeIndexStatus">` → add `role="status"`
   - `<div class="ai-messages" id="aiMessages">` → add `aria-live="polite"`

6. `index.html` missing labels:
   - `<textarea id="aiPrompt" rows="2" placeholder="Ask about or change this canvas…">` → add `aria-label="Ask Balaur about this canvas"`
   - `<textarea id="journalBody" rows="6" placeholder="Write a few lines about today…">` → add `aria-label="Daily note"`

7. `index.html` decorative dialog glyphs — add `aria-hidden="true"` to the three
   dialog header glyph spans: `<span>✓</span>` (taskDialog), `<span>✎</span>`
   (aiNoteDialog), `<span>✦</span>` (aiSettingsDialog). Do not touch the
   `ai-spark` span or any span that already has aria-hidden.

8. `index.html` AI settings: `<input id="aiModel" required placeholder="mistral-small-latest">` → add `spellcheck="false"`.

9. `index.html` copy: in the rememberAIKey small text, replace the straight
   apostrophe in `this browser's` with a curly one: `this browser’s`.

10. `styles/shell.css` — the `#canvasTitle` rule sets `outline: 0` with no
    replacement. Immediately after the `#canvasTitle` rule block, add:

    ```css
    #canvasTitle:focus-visible {
      outline: 2px solid var(--balaur-border-focus);
      outline-offset: 4px;
    }
    ```

11. `styles/canvas.css` — `.canvas` sets `outline: 0` but is `tabindex="0"`.
    Immediately after the `.canvas` rule block (before `.canvas.panning`), add:

    ```css
    .canvas:focus-visible {
      outline: 2px solid var(--balaur-border-focus);
      outline-offset: -2px;
    }
    ```

    (Negative offset keeps the ring inside the overflow-hidden viewport.)

12. `styles/canvas.css` — the `.minimap` rule gains button resets: add
    `padding: 0;` and `cursor: pointer;` to the existing `.minimap` rule block.
    Its border, background, and font inheritance are already correct.

13. `app.js` — the `$("#toggleAIKey").onclick` handler updates the button text
    ("Show"/"Hide") but leaves `aria-label="Show API key"` stale. Extend the
    handler so the aria-label tracks the new text:
    `$("#toggleAIKey").setAttribute("aria-label", show ? "Hide API key" : "Show API key");`
    appended after the `textContent` assignment inside the same arrow body.
    Do not reformat anything else in app.js.

## Out of scope

- Everything in the audit's "Later batch" section (scroll containment,
  transform-based slides, Intl task dates, touch-action, safe areas,
  tabular-nums, font preload, beforeunload, resize-handle keyboard path,
  URL state, Title Case decision).
- `sw.js`, `storage/**`, `elements/task-list.js`, `widgets/**`.

## Verification (all required)

- `node --check app.js main.js` and `node --check elements/*.js`
- `git diff --check` → clean
- Grep gates:
  - `grep -c 'aria-label' index.html` increased by exactly 10 (7 buttons + 2 textareas + minimap button)
  - `grep -o 'role="status"' index.html | wc -l` increases by exactly 2 over the pre-change count (saveState, lifeIndexStatus)
  - `grep -n 'toggleAIKey' app.js` shows the aria-label sync
- Browser smoke (AGENTS.md §13): serve with `python3 -m http.server 4173`
  (background, kill afterwards), then
  `node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline`
  must pass. If the skill errors for environmental reasons, capture the
  output in the artifact and note it; do not skip silently.
- Optional if time allows: a second smoke pass with `--width 380 --height 760`
  to confirm the narrow shell.

## Commit

One commit, staged paths only (`git add index.html styles/shell.css styles/canvas.css app.js`),
never `git add -A`. Message: `005: a11y do-now batch — labels, live regions, focus rings, minimap button`.
Never push.

## Artifact

Write the implement artifact to
`para/projects/005-ui-guidelines-audit/tickets/01-do-now-a11y-impl.md`
(steps completed, files changed, verification results, issues encountered)
and include it in the commit.
