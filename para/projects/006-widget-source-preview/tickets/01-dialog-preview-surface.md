---
phase: ticket
status: review
project: 006-widget-source-preview
ticket: 01
title: Dialog preview surface
blocked-by: []
worker: 8ec5b5ba-dcbf-424e-9317-748a2b7ac7f0
branch: 006-widget-source-preview
shared-blast-radius: true
planned-at: b9f94d7
date: 2026-07-25
---

# Ticket 01: Dialog preview surface

## What to build

Add a live, atomic widget-source preview inside the existing `#widgetSourceDialog`. When the user clicks a new "Preview" button in the dialog header, the current source text is passed through the existing `buildWidgetDocument` primitive, wrapped in a Blob URL, and loaded into a sandboxed `<iframe sandbox="allow-scripts">`. On dialog close, the iframe is removed and the Blob URL is revoked. Scanner-failing source shows a diagnostic message instead of an iframe. The dialog becomes a two-pane review surface: source text on one side, live execution on the other. Narrow screens (≤760px container width) stack the panes vertically.

## Current state

### `showWidgetSourceReview` — `app.js:883-891`

This function creates (once) and populates the `#widgetSourceDialog`. Current dialog innerHTML:

```js
// app.js:886-887
dialog.innerHTML='<article><header><div><small>REVIEWED CANONICAL SOURCE</small><h2></h2></div><button type="button" data-close-widget-source aria-label="Close source review">Close</button></header><p class="widget-capability-summary">Sandboxed scripts and inline styles only. No host data or mutation, storage, network, forms, popups, workers, or nested frames. Self-navigation pauses the widget; hard request suppression is not claimed.</p><code></code><pre></pre></article>';
```

The function receives `{title, path, source}` and writes them into the dialog. No teardown on close exists today.

### `buildWidgetDocument` — `widgets/widget-envelope.js:19-33`

The trusted-document builder. Takes `source` + `{bootstrapSource}`, calls `validateWidgetSource(source)` (throws `TypeError` on scanner failure), injects CSP via `<meta http-equiv>`, and returns a complete HTML string. Already tested in `widgets/widget-runtime.test.js`.

### `DIAGNOSTIC_BOUNDARY_SOURCE` — `widgets/widget-envelope.js:7-14`

Minimal error/unhandledrejection reporter. References `globalThis.__balaurReportDiagnostic?.(...)` which is undefined in the preview context, making diagnostics a silent no-op. This is the pattern to copy for the preview bootstrap — NOT the full `BOOTSTRAP_SOURCE` from `elements/widget-frame.js` (which sets up MessageChannel, theme projection, heartbeats — none needed for preview).

### `BalaurWidgetFrameElement.activate()` — `elements/widget-frame.js:166-193`

The construction to mirror for iframe creation: `buildWidgetDocument(source, {bootstrapSource})` → `new Blob([doc])` → `URL.createObjectURL` → `<iframe sandbox="allow-scripts" referrerpolicy="no-referrer">` → set `iframe.src` → append. The preview follows this exact pattern but omits `loading="lazy"`, `allow=""`, the `load` event handler, and the `activeWidgets` tracking.

### Existing dialog CSS — `styles/elements.css:285-309`

```css
.widget-source-dialog {
  inline-size: min(56rem, calc(100% - 24px));
  max-block-size: calc(100dvh - 24px);
  border: 1px solid var(--balaur-border-default);
  padding: 0;
  background: var(--balaur-surface-oak);
  color: var(--balaur-content-on-dark);
}
.widget-source-dialog article { padding: 18px; }
.widget-source-dialog header { display: flex; align-items: start; justify-content: space-between; gap: 16px; }
```

### Responsive breakpoint — `styles/responsive.css:2`

```css
@container balaur-content (width <= 760px) {
```

This is the existing narrow-screen breakpoint. New stacking rules go inside this block.

### Import block — `app.js:1-21`

`buildWidgetDocument` is NOT currently imported in `app.js`. The executor must add the import.

### Repo conventions

- ES-module imports at the top of `app.js`, one `import` per line, alphabetical within a group. See `app.js:1-21`.
- Dialog construction uses `document.createElement("dialog")` + `innerHTML` + `$()` query helpers. See `app.js:885-888`.
- CSS uses Balaur design tokens (`--balaur-content-on-dark-muted`, `--balaur-border-default`, etc.). See `styles/elements.css:297-309`.
- Logical properties (`inline-size`, `block-size`) over physical (`width`, `height`). See `styles/elements.css:298-299`.
- Container queries use `@container balaur-content`, not `@media`. See `styles/responsive.css:2`.

### Security invariants (AGENTS.md §10)

These MUST hold after this ticket:
1. Preview iframe uses exactly `sandbox="allow-scripts"` — no `allow-same-origin`, no `allow-forms`, no `allow-popups`, no `allow-downloads`.
2. Blob URL is an opaque origin; widget cannot access host cookies, storage, or DOM.
3. CSP `default-src 'none'; script-src 'unsafe-inline'; ...` is injected via `<meta http-equiv>` by `buildWidgetDocument` — blocks all network, workers, nested frames, forms.
4. Scanner caps (128 KiB source, 500 elements, 64 KiB script, 64 KiB style) from `widgets/widget-policy.js` apply identically.
5. Preview does NOT register in `activeWidgets` and does NOT count against `MAX_ACTIVE_WIDGETS = 6`.
6. Apply remains unchanged — it writes the canonical `widgets/*.html` file, inactive until explicit Run.
7. Preview bootstrap is distinct from `elements/widget-frame.js` `BOOTSTRAP_SOURCE` — no MessageChannel, no theme tokens, no heartbeats.

## Scope

**In scope** (the only files to modify):
- `app.js` (modify — add import, modify `showWidgetSourceReview`, add preview bootstrap constant, add preview handler, add close-event teardown)
- `styles/elements.css` (modify — add preview region, iframe, status line, and two-pane layout CSS within the `.widget-source-dialog` rule block)
- `styles/responsive.css` (modify — add stacking rule inside the existing `@container balaur-content (width <= 760px)` block)

**Out of scope** (do NOT touch):
- `elements/widget-frame.js` — the full-frame runtime; preview deliberately excludes it
- `widgets/widget-envelope.js` — `buildWidgetDocument` and `DIAGNOSTIC_BOUNDARY_SOURCE` are reused as-is; do not modify
- `widgets/widget-policy.js` — scanner caps are reused as-is
- `index.html` — no new top-level DOM; the dialog is created programmatically
- Any `*.test.js` file — no new unit tests (spec defers to browser-check + manual)
- `operationDescription` / `assistantProposal` — ticket 02 handles those

## Steps

### Step 1: Add the `buildWidgetDocument` import to `app.js`

Add this import line after the existing imports (after `app.js:21`):

```js
import { buildWidgetDocument } from "./widgets/widget-envelope.js";
```

**Verify**: `node --check app.js` exits 0.

### Step 2: Define the preview bootstrap constant

Add a module-scoped constant near the top of `app.js` (after the existing constants like `COLORS`, `NOTE_MARKERS`, etc. — around `app.js:22-30`). This is a NEW constant, distinct from `elements/widget-frame.js` `BOOTSTRAP_SOURCE` and from `widgets/widget-envelope.js` `DIAGNOSTIC_BOUNDARY_SOURCE` (which is not exported). It contains only the diagnostic-boundary error/unhandledrejection listeners:

```js
const PREVIEW_BOOTSTRAP_SOURCE = `
(() => {
  const report = (level, value) => {
    const message = value instanceof Error ? value.message : String(value ?? "Unknown widget error");
    globalThis.__balaurReportDiagnostic?.({ level, message: message.slice(0, 4096) });
  };
  addEventListener("error", (event) => report("error", event.error || event.message));
  addEventListener("unhandledrejection", (event) => report("error", event.reason));
})();`;
```

This mirrors `widgets/widget-envelope.js:7-14` exactly. `__balaurReportDiagnostic` is undefined in the preview context, so diagnostics are a silent no-op.

**Verify**: `node --check app.js` exits 0.

### Step 3: Modify the dialog innerHTML to add the preview region and Preview button

In `showWidgetSourceReview` (`app.js:883`), modify the `dialog.innerHTML` assignment (currently `app.js:886-887`). The new innerHTML adds:
1. A Preview button in the `<header>` next to the Close button.
2. A preview region `<div>` after the `<pre>` element.

The new innerHTML:

```js
dialog.innerHTML='<article><header><div><small>REVIEWED CANONICAL SOURCE</small><h2></h2></div><div class="widget-source-dialog-actions"><button type="button" class="widget-preview-button" data-preview-widget>Preview</button><button type="button" data-close-widget-source aria-label="Close source review">Close</button></div></header><p class="widget-capability-summary">Sandboxed scripts and inline styles only. No host data or mutation, storage, network, forms, popups, workers, or nested frames. Self-navigation pauses the widget; hard request suppression is not claimed.</p><div class="widget-source-dialog-body"><div class="widget-source-region"><code></code><pre></pre></div><div class="widget-preview-region" aria-live="polite"><p class="widget-preview-placeholder">Click Preview to run the widget in a sandbox.</p></div></div></article>';
```

Key structural changes:
- The header's direct `<button>` Close is now wrapped in a `<div class="widget-source-dialog-actions">` alongside the new Preview button.
- The `<code>` and `<pre>` are wrapped in `<div class="widget-source-region">`.
- A new `<div class="widget-preview-region">` sits beside the source region, with a placeholder `<p>`.
- The preview region has `aria-live="polite"` for screen-reader announcements of status changes.

Also update the Close button wiring: the existing `$("[data-close-widget-source]",dialog).onclick=()=>dialog.close();` at `app.js:888` remains unchanged.

**Verify**: `node --check app.js` exits 0.

### Step 4: Store the current source and attach the close-event teardown (inside the `if(!dialog)` branch)

Still inside the `if(!dialog)` branch (after the `innerHTML` assignment, before the closing `}` of the `if` block), add:

1. A `close` event listener on the dialog for teardown:

```js
let previewObjectUrl = null;
dialog.addEventListener("close", () => {
  const region = $(".widget-preview-region", dialog);
  if (region) {
    const iframe = $("iframe", region);
    if (iframe) iframe.remove();
    region.innerHTML = '<p class="widget-preview-placeholder">Click Preview to run the widget in a sandbox.</p>';
  }
  if (previewObjectUrl) {
    URL.revokeObjectURL(previewObjectUrl);
    previewObjectUrl = null;
  }
});
```

The `previewObjectUrl` variable is module-scoped to `showWidgetSourceReview`'s closure — but since `showWidgetSourceReview` is called multiple times and the dialog is created once, the variable must persist across calls. Use a variable declared in the `if(!dialog)` block scope that the Preview handler (Step 5) closes over. Alternatively, store it on the dialog element: `dialog._previewObjectUrl = null;` and read/write it from both the close handler and the Preview handler. The element-property approach is simpler and avoids closure issues.

**Recommended approach** — store on the dialog element:

```js
dialog._previewObjectUrl = null;
dialog.addEventListener("close", () => {
  const region = $(".widget-preview-region", dialog);
  if (region) {
    const iframe = $("iframe", region);
    if (iframe) iframe.remove();
    region.innerHTML = '<p class="widget-preview-placeholder">Click Preview to run the widget in a sandbox.</p>';
  }
  if (dialog._previewObjectUrl) {
    URL.revokeObjectURL(dialog._previewObjectUrl);
    dialog._previewObjectUrl = null;
  }
});
```

2. The Preview button click handler:

```js
$("[data-preview-widget]", dialog).onclick = () => {
  const source = $("pre", dialog).textContent;
  if (!source) return;
  // Tear down prior preview before building a new one
  const region = $(".widget-preview-region", dialog);
  const priorIframe = $("iframe", region);
  if (priorIframe) priorIframe.remove();
  if (dialog._previewObjectUrl) {
    URL.revokeObjectURL(dialog._previewObjectUrl);
    dialog._previewObjectUrl = null;
  }
  const statusLine = $(".widget-preview-status", region);
  if (statusLine) statusLine.remove();
  const placeholder = $(".widget-preview-placeholder", region);
  if (placeholder) placeholder.remove();
  try {
    const documentSource = buildWidgetDocument(source, { bootstrapSource: PREVIEW_BOOTSTRAP_SOURCE });
    dialog._previewObjectUrl = URL.createObjectURL(new Blob([documentSource], { type: "text/html" }));
    const iframe = document.createElement("iframe");
    iframe.setAttribute("sandbox", "allow-scripts");
    iframe.setAttribute("referrerpolicy", "no-referrer");
    iframe.title = $("h2", dialog).textContent || "Widget preview";
    iframe.src = dialog._previewObjectUrl;
    region.append(iframe);
  } catch (error) {
    const status = document.createElement("p");
    status.className = "widget-preview-status";
    status.textContent = error.message || "Preview unavailable";
    region.append(status);
  }
};
```

Key points:
- Reads source from `$("pre", dialog).textContent` — the same source already displayed.
- Tears down any prior preview (iframe + Blob URL + status) before building a new one.
- Uses `buildWidgetDocument(source, { bootstrapSource: PREVIEW_BOOTSTRAP_SOURCE })` — this internally calls `validateWidgetSource(source)` which throws `TypeError` on scanner failure.
- On success: creates Blob, gets object URL, creates iframe with `sandbox="allow-scripts"` and `referrerpolicy="no-referrer"`, sets `iframe.title` for accessibility, appends to region.
- On failure: shows error message in a `<p class="widget-preview-status">` inside the preview region; no iframe created.
- Does NOT touch `activeWidgets`, does NOT register the preview, does NOT count against `MAX_ACTIVE_WIDGETS`.

**Verify**: `node --check app.js` exits 0.

### Step 5: Store the current source on each call (outside the `if(!dialog)` branch)

After the `if(!dialog)` block, the existing code sets `$("h2", dialog).textContent=title; $("code", dialog).textContent=path; $("pre", dialog).textContent=source;`. No change needed here — the Preview handler reads from `$("pre", dialog).textContent` which is already set.

However, the Preview button should be disabled when there is no source. Add after the text assignments:

```js
$("[data-preview-widget]", dialog).disabled = !source;
```

**Verify**: `node --check app.js` exits 0.

### Step 6: Add CSS for the two-pane layout and preview region

In `styles/elements.css`, after the existing `.widget-source-dialog` rules (after line 309), add:

```css
.widget-source-dialog-actions { display: flex; gap: 8px; align-items: center; }
.widget-source-dialog-body { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; margin-block-start: 12px; }
.widget-source-region { min-inline-size: 0; }
.widget-preview-region {
  min-inline-size: 0;
  min-block-size: 12rem;
  border: 1px solid var(--balaur-border-default);
  background: var(--balaur-surface-page);
  display: flex;
  align-items: center;
  justify-content: center;
}
.widget-preview-region iframe { inline-size: 100%; block-size: 100%; border: 0; }
.widget-preview-placeholder { color: var(--balaur-content-on-dark-muted); font-size: 0.85rem; }
.widget-preview-status { color: var(--balaur-status-danger); font-size: 0.85rem; padding: 12px; }
```

Key points:
- `.widget-source-dialog-body` uses `grid-template-columns: 1fr 1fr` for side-by-side at desktop.
- `.widget-preview-region` has a min-block-size so the placeholder is visible even before an iframe loads.
- The iframe fills the region with `inline-size: 100%; block-size: 100%; border: 0;`.
- Placeholder and status text use existing tokens.
- The Preview button uses the existing `.button` class or default button styling — it is in the header actions area, visually distinct from Close via ordering (primary action first).

**Verify**: `node --check styles/elements.css` is not applicable (CSS); instead visually verify in Step 8.

### Step 7: Add the narrow-screen stacking rule

In `styles/responsive.css`, inside the existing `@container balaur-content (width <= 760px)` block (line 2-7), add:

```css
.widget-source-dialog-body { grid-template-columns: 1fr; }
```

This makes the source and preview panes stack vertically on narrow screens.

**Verify**: Visually verify in Step 8.

### Step 8: Manual verification

Serve the app and open in a browser:

```bash
python3 -m http.server 4173
```

1. Open the app, pick a vault, navigate to a canvas with a widget.
2. Right-click the widget → "View source" (or trigger `balaur-widget-view-source` via the browser console: `document.dispatchEvent(new CustomEvent("balaur-widget-view-source", {detail: {title: "Test", path: "widgets/test.html", source: "<!doctype html><title>Test</title><p>Hello</p>"}}))`).
3. Verify the dialog opens with the source text on the left and a "Click Preview to run the widget in a sandbox." placeholder on the right.
4. Click "Preview" — verify an iframe appears on the right showing the widget rendering.
5. Close the dialog — verify no console errors.
6. Re-open the dialog — verify the preview region is reset to the placeholder.
7. Resize to ≤760px container width — verify source and preview stack vertically.
8. Test with scanner-failing source (e.g., source exceeding 128 KiB) — verify the preview region shows an error message and no iframe is created.

**Verify**: No uncaught console errors. Preview renders. Teardown on close works. Narrow layout stacks.

### Step 9: Run the browser-check smoke suite

```bash
node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline
```

**Verify**: All existing smoke assertions pass with no new failures. The preview changes must not break any existing dialog, canvas, or offline behavior.

### Step 10: Run the Node storage test suite

```bash
node --test \
  storage/phase1.test.js storage/phase2.test.js storage/phase3.test.js \
  storage/phase4.test.js storage/phase4-backup.test.js storage/phase5.test.js \
  storage/phase7.test.js storage/phase8.test.js storage/phase9.test.js \
  storage/phase10.test.js storage/phase-query.test.js \
  storage/note-repository.test.js
```

**Verify**: All 197 tests pass. This ticket does not touch storage code, so no regressions are expected.

## Done criteria

Machine-checkable. ALL must hold:
- [ ] `node --check app.js` exits 0
- [ ] `node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline` exits 0 with no new failures
- [ ] `node --test storage/phase*.test.js storage/phase4-backup.test.js storage/phase-query.test.js storage/note-repository.test.js` → 197 tests pass
- [ ] Dialog opens with source region + preview region side-by-side at desktop widths
- [ ] Preview button creates an iframe via `buildWidgetDocument` + Blob URL with `sandbox="allow-scripts"` only
- [ ] Close event removes the iframe and revokes the Blob URL
- [ ] Scanner-failing source shows error message in preview region, no iframe created
- [ ] Narrow layout (≤760px container width) stacks source and preview vertically
- [ ] Preview does NOT register in `activeWidgets` or count against `MAX_ACTIVE_WIDGETS`
- [ ] No files outside the in-scope list are modified (`git status`)

## STOP conditions

Stop and report back (do not improvise) if:
- The code at `app.js:883-891` does not match the excerpts above (the codebase drifted since `planned-at: b9f94d7`).
- `buildWidgetDocument` is not exported from `widgets/widget-envelope.js` or its signature changed.
- The `@container balaur-content (width <= 760px)` breakpoint does not exist in `styles/responsive.css`.
- The browser-check smoke suite fails on an assertion unrelated to the preview changes (pre-existing failure).
- The fix appears to require touching an out-of-scope file.

## Blocked by

None — can start immediately.

## Next step

- Recommended executor tier: mid
- Recommended model: see `para/resources/model-registry.md`
- Estimated complexity: M
