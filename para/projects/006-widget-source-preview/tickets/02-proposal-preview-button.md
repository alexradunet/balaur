---
phase: ticket
status: ready
project: 006-widget-source-preview
ticket: 02
title: Proposal details Preview button
blocked-by: [01]
worker: ""
branch: ""
shared-blast-radius: true
planned-at: b9f94d7
date: 2026-07-25
---

# Ticket 02: Proposal details Preview button

## What to build

Add a "Preview" button inside the `<details>` block that `operationDescription` generates for `widget.create` operations in AI proposals. Clicking the button opens the same `#widgetSourceDialog` (from ticket 01) via `showWidgetSourceReview`, passing the operation's widget title, path, and source. This gives users a one-click path from an AI proposal to the live preview without manually copying source.

## Current state

### `operationDescription` — `app.js:1753-1757`

Returns an HTML string for each operation in an AI proposal. For `widget.create`:

```js
// app.js:1753-1757
function operationDescription(operation) {
  if(operation.type==="component-card.create"||operation.type==="component-card.update"||operation.type==="widget.create"){
    const description=describeGeneratedOperation(operation),source=description.source;
    return `<div class="${source?"widget-operation-review":""}"><b>${escapeHTML(description.title)}</b> · ${escapeHTML(description.summary)}${description.details.length?`<small>${escapeHTML(description.details.join(" · "))}</small>`:""}${source?`<details><summary>Review complete source (${source.length} characters)</summary><pre>${escapeHTML(source)}</pre></details>`:""}</div>`;
  }
  // ...
}
```

The `<details>` contains a `<pre>` with the escaped source. The `source` variable is the raw widget source string from `operation.widget.source` (via `describeGeneratedOperation` at `ai/generated-operations.js:297-326`).

### `assistantProposal` — `app.js:1763-1790`

Creates the `.ai-operation-list` container and populates it via `operationDescription`. The list is re-rendered by `renderPending()` after partial applies. There is no existing click delegation on `.ai-operation-list` — Apply/Discard are separate buttons outside the list.

### `showWidgetSourceReview` — `app.js:883`

Accepts `{title, path, source}` and opens the `#widgetSourceDialog`. After ticket 01, this dialog has a Preview button that builds and shows the live iframe.

### `escapeHTML` — used throughout `app.js` for safe HTML insertion.

### Repo conventions

- `operationDescription` returns an HTML string; it does not create DOM elements. The caller (`assistantProposal`) sets `innerHTML`.
- Event delegation pattern: use `event.target.closest?.(selector)` to detect clicks on dynamically-created elements. See `app.js:483`, `app.js:702`, `app.js:1028-1032` for examples.
- Data attributes carry operation-specific values on buttons: `data-preview-widget-source` marks the button; `data-widget-title`, `data-widget-path` carry metadata. The source is read from the sibling `<pre>` textContent (same pattern as ticket 01's dialog Preview handler).

### Security invariants (AGENTS.md §10)

This ticket does NOT introduce new execution. It opens the existing dialog (ticket 01) which handles all sandboxing. The Preview button in the `<details>` is a convenience entry point — it calls `showWidgetSourceReview` which already gates execution behind an explicit Preview click in the dialog.

## Scope

**In scope** (the only files to modify):
- `app.js` (modify — change `operationDescription` to add the Preview button HTML; add event delegation in `assistantProposal` for the button click)

**Out of scope** (do NOT touch):
- `styles/elements.css` — the existing `.widget-operation-review details` and `.widget-operation-review pre` rules (lines 281-295) already style the `<details>` block. The Preview button inside it uses default button styling. If styling adjustments are needed, they are minor and can be added here, but the ticket does not require new CSS.
- `styles/responsive.css` — no responsive changes needed for the proposal `<details>`.
- `widgets/widget-envelope.js`, `elements/widget-frame.js`, `widgets/widget-policy.js` — primitives are reused as-is.
- `operationDescription` for `component-card.create` / `component-card.update` — no Preview button for component cards (they are declarative data, not executed source).
- `showWidgetSourceReview` — ticket 01 owns that function.

## Steps

### Step 1: Modify `operationDescription` to add a Preview button for `widget.create`

In `operationDescription` (`app.js:1753-1757`), the `widget.create` branch currently generates:

```js
${source?`<details><summary>Review complete source (${source.length} characters)</summary><pre>${escapeHTML(source)}</pre></details>`:""}
```

Change it to add a Preview button after the `<pre>` inside the `<details>`, only for `widget.create` operations:

```js
${source?`<details><summary>Review complete source (${source.length} characters)</summary><pre>${escapeHTML(source)}</pre>${operation.type==="widget.create"?`<button type="button" class="widget-preview-button" data-preview-widget-source data-widget-title="${escapeHTML(operation.widget.title)}" data-widget-path="${escapeHTML(operation.widget.path)}">Preview</button>`:""}</details>`:""}
```

Key points:
- The button is inside the `<details>`, after the `<pre>`, so it is visible when the user expands the source review.
- `data-preview-widget-source` marks the button for event delegation.
- `data-widget-title` and `data-widget-path` carry the operation's widget metadata.
- The source is NOT stored as a data attribute (it could be large); instead, the click handler reads it from the sibling `<pre>` textContent.
- The button uses `class="widget-preview-button"` for consistent styling with the dialog's Preview button (ticket 01).
- Component-card operations do NOT get a Preview button (they are declarative, not executed).

**Verify**: `node --check app.js` exits 0.

### Step 2: Add event delegation in `assistantProposal` for the Preview button

In `assistantProposal` (`app.js:1763`), after the `list` element is created and before the `apply.onclick` handler, add a click delegation handler on `list`:

```js
list.addEventListener("click", event => {
  const button = event.target.closest?.("[data-preview-widget-source]");
  if (!button || !list.contains(button)) return;
  event.stopPropagation();
  const pre = button.closest("details")?.querySelector("pre");
  const source = pre?.textContent || "";
  showWidgetSourceReview({
    title: button.dataset.widgetTitle || "Live widget",
    path: button.dataset.widgetPath || "",
    source,
  });
});
```

Key points:
- Uses `event.target.closest?.("[data-preview-widget-source]")` — the standard delegation pattern in this codebase (see `app.js:483`, `app.js:702`).
- `list.contains(button)` guards against events bubbling from outside the list.
- `event.stopPropagation()` prevents the click from reaching parent handlers.
- Reads source from the sibling `<pre>` textContent inside the same `<details>` — avoids storing large source strings as data attributes.
- Reads title and path from `button.dataset.widgetTitle` and `button.dataset.widgetPath`.
- Calls `showWidgetSourceReview` which opens the dialog (ticket 01) with the Preview button already wired.

Note: the `renderPending` function re-renders `list.innerHTML`, which destroys and recreates all child elements. The delegation handler is attached to `list` itself (not its children), so it survives re-renders. This is the standard advantage of delegation.

**Verify**: `node --check app.js` exits 0.

### Step 3: Add minimal CSS for the Preview button inside `<details>`

In `styles/elements.css`, after the existing `.widget-operation-review` rules (around line 295), add:

```css
.widget-operation-review .widget-preview-button {
  margin-block-start: 8px;
}
```

This gives the button some spacing from the `<pre>` block. The button uses default styling; no need for a custom background or border.

**Verify**: Visual check in Step 4.

### Step 4: Manual verification

Serve the app:

```bash
python3 -m http.server 4173
```

1. Open the app, pick a vault.
2. Trigger a local widget proposal: click the "Propose widget" button (or call `proposeLocalWidget()` from the console if a UI button does not exist).
3. In the AI proposal, expand the `<details>` block for the `widget.create` operation.
4. Verify a "Preview" button appears after the `<pre>` source block.
5. Click the Preview button — verify the `#widgetSourceDialog` opens with the correct title, path, and source.
6. Click Preview in the dialog — verify the iframe renders the widget.
7. Close the dialog — verify teardown (no orphaned iframe or Blob URL).
8. Apply the proposal — verify the widget is created normally (Apply is unchanged).

**Verify**: Preview button in `<details>` opens dialog. Dialog Preview renders iframe. Apply still works.

### Step 5: Run the browser-check smoke suite

```bash
node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline
```

**Verify**: All existing smoke assertions pass with no new failures.

### Step 6: Run the Node storage test suite

```bash
node --test \
  storage/phase1.test.js storage/phase2.test.js storage/phase3.test.js \
  storage/phase4.test.js storage/phase4-backup.test.js storage/phase5.test.js \
  storage/phase7.test.js storage/phase8.test.js storage/phase9.test.js \
  storage/phase10.test.js storage/phase-query.test.js \
  storage/note-repository.test.js
```

**Verify**: All 197 tests pass. This ticket does not touch storage code.

## Done criteria

Machine-checkable. ALL must hold:
- [ ] `node --check app.js` exits 0
- [ ] `node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline` exits 0 with no new failures
- [ ] `node --test storage/phase*.test.js storage/phase4-backup.test.js storage/phase-query.test.js storage/note-repository.test.js` → 197 tests pass
- [ ] `widget.create` `<details>` in AI proposals shows a Preview button after the `<pre>` block
- [ ] `component-card.create` / `component-card.update` `<details>` do NOT show a Preview button
- [ ] Clicking the Preview button opens `#widgetSourceDialog` with the correct title, path, and source
- [ ] The dialog's Preview button (ticket 01) renders the iframe from this entry point
- [ ] Apply remains unchanged — widget creation works normally after previewing
- [ ] No files outside the in-scope list are modified (`git status`)

## STOP conditions

Stop and report back (do not improvise) if:
- The code at `app.js:1753-1757` does not match the excerpts above (the codebase drifted since `planned-at: b9f94d7`).
- `operationDescription` no longer returns an HTML string or the `<details>` structure changed.
- `showWidgetSourceReview` is not available as a module-scoped function (e.g., it was moved or renamed by ticket 01).
- The browser-check smoke suite fails on an assertion unrelated to the proposal changes (pre-existing failure).
- The fix appears to require touching an out-of-scope file.

## Blocked by

Ticket 01 (Dialog preview surface) — the dialog must exist with the Preview button and iframe construction before this entry point can open it.

## Next step

- Recommended executor tier: mid
- Recommended model: see `para/resources/model-registry.md`
- Estimated complexity: S
