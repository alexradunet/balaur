---
phase: spec
status: done
project: 006-widget-source-preview
date: 2026-07-25
---

# Spec: Atomic widget source preview in the review dialog

## Problem Statement

When a user reviews widget source — either from an AI proposal's `<details>` block or from a placed widget's "View source" — they see raw HTML text in `#widgetSourceDialog` but cannot see the widget actually run before deciding to Apply or Run. The only way to observe execution today is to commit: Apply writes the canonical file, then Run activates it. A mistake in generated source (broken layout, infinite loop, visual regression) is discovered only after the user has already approved it.

## Solution

Add a **Preview** button and a live iframe pane to the existing `#widgetSourceDialog`. Clicking Preview constructs a trusted document via the existing `buildWidgetDocument(source, {bootstrapSource})` primitive, wraps it in a Blob URL, and loads it into a minimal `<iframe sandbox="allow-scripts" referrerpolicy="no-referrer">`. The preview is atomic (one-shot render, not streamed). The dialog becomes a two-pane review surface: source text on one side, live execution on the other. The Preview button is explicit — the iframe is not built on dialog open, preserving the §10 explicit-execution posture.

The preview deliberately excludes the full `BalaurWidgetFrameElement` runtime (MessageChannel, theme-token projection, heartbeats, `MAX_ACTIVE_WIDGETS` cap, IntersectionObserver, visibility/pause lifecycle). The widget runs with fallback styling and silent runtime errors; full diagnostics and theming arrive at real Run via `balaur-widget-frame`.

## User Stories

1. As a user reviewing an AI-proposed widget, I want a Preview button inside the source `<details>` so I can see the widget run before I decide to Apply.
2. As a user viewing a placed widget's source, I want a Preview button in the review dialog so I can re-observe execution without leaving the dialog.
3. As a user, I want the preview to use the same scanner and CSP as Run, so I trust that a passing preview will behave identically when I Apply and Run.
4. As a user, I want the preview iframe to be torn down when I close the dialog, so no orphaned sandboxed execution or Blob URL leaks persist.
5. As a user on a narrow screen, I want the source and preview panes to stack vertically so both remain readable without horizontal scroll.
6. As a user, I want the Preview button to be disabled or hidden when the source fails the scanner, so I am not offered a preview that cannot be built.
7. As a user, I want the preview to not count against the six-active-widget cap, so previewing does not evict a placed widget.

## Implementation Decisions

### Dialog structure

The existing `#widgetSourceDialog` (created programmatically in `showWidgetSourceReview`, `app.js:883-888`) gains a preview pane region and a Preview button. The dialog's `<article>` layout becomes a two-region surface:

- **Source region** (existing): the `<code>` path label and `<pre>` source text, unchanged.
- **Preview region** (new): a container `<div>` that holds the preview iframe when active, plus a status line for scanner diagnostics.
- **Preview button**: placed in the dialog header next to the Close button. Label: "Preview". Disabled while no source is present or while a preview is already building.

On narrow screens (≤760px container width, matching the existing `@container balaur-content (width <= 760px)` breakpoint in `styles/responsive.css`), the two regions stack vertically: source above, preview below. Above 760px, they sit side-by-side in a grid or flex row with the preview region taking remaining inline space.

### Preview construction

The Preview button handler:

1. Reads the current source from the dialog's state (the same `source` string already displayed in `<pre>`).
2. Calls `buildWidgetDocument(source, {bootstrapSource})` — this internally calls `validateWidgetSource(source)`, which throws on scanner failure. The bootstrap source is a minimal no-op script (the diagnostic-boundary reporter with port null — no `__balaurReportDiagnostic` handler is installed, so diagnostics are silent). This is a new, preview-specific bootstrap string, not the full `BOOTSTRAP_SOURCE` from `elements/widget-frame.js` (which sets up MessageChannel handling the preview does not need).
3. On success: creates a Blob, obtains an object URL via `URL.createObjectURL`, creates an `<iframe sandbox="allow-scripts" referrerpolicy="no-referrer">`, sets `iframe.src` to the object URL, and appends it to the preview region. Stores the object URL for later revocation.
4. On scanner failure (caught `TypeError`): displays the error message in the preview region's status line; does not create an iframe.

### Bootstrap source for preview

A new constant, minimal bootstrap script for preview use only. It contains only the diagnostic-boundary error/unhandledrejection listeners (identical to the `DIAGNOSTIC_BOUNDARY_SOURCE` in `widgets/widget-envelope.js`) — no MessageChannel setup, no theme projection, no heartbeat. The `__balaurReportDiagnostic` global remains undefined, so the diagnostic reporter is a silent no-op. This keeps the preview boundary identical to the envelope's trust model without pulling in placement-coupled runtime.

### Teardown invariant

A `close` event listener on `#widgetSourceDialog` (attached once, at dialog creation time):

1. Removes any preview iframe from the preview region.
2. Revokes the stored Blob object URL via `URL.revokeObjectURL` (if one exists).
3. Clears the stored reference.

This prevents orphaned sandboxed execution and Blob URL leaks. Re-preview (clicking Preview again after a previous preview) first tears down the prior preview, then builds a fresh one from the current source text.

### Preview button in proposal `<details>`

The `operationDescription` function (`app.js:1753`) generates the `<details>` block for `widget.create` operations. When `operation.type === "widget.create"` and a source string is present, the `<details>` content gains a "Preview" button after the `<pre>` block. The button's click handler calls `showWidgetSourceReview({title, path, source})` with the operation's widget title, path, and source — opening the same unified dialog.

For the "View source" entry point (`balaur-widget-view-source` event, `app.js:892`), no change is needed — it already calls `showWidgetSourceReview`. The Preview button inside the dialog serves this path.

### State management

`showWidgetSourceReview` stores the current source string in a module-scoped variable (or on the dialog element via a data property) so the Preview handler can read it. The source is the same string already passed to the function and displayed in `<pre>`. No new data flow is introduced.

### Security invariants preserved

- The preview iframe uses exactly `sandbox="allow-scripts"` — no `allow-same-origin`, no `allow-forms`, no `allow-popups`, no `allow-downloads`, no `allow-scripts allow-same-origin` combination. This matches `elements/widget-frame.js:189`.
- The Blob URL is an opaque origin; the widget cannot access host cookies, storage, or DOM.
- CSP `default-src 'none'; script-src 'unsafe-inline'; ...` is injected via `<meta http-equiv>` by `buildWidgetDocument`, blocking all network, workers, nested frames, and forms.
- The scanner caps (128 KiB source, 500 elements, 64 KiB script, 64 KiB style) apply identically to preview and Run.
- The preview does not register in `activeWidgets` and does not count against `MAX_ACTIVE_WIDGETS = 6`.
- Apply remains unchanged: it writes the canonical `widgets/*.html` file, inactive until explicit Run.

### CSS additions

New rules in `styles/elements.css` (within the existing `.widget-source-dialog` rule block):

- `.widget-source-dialog .widget-preview-region`: min-block-size, border, background for the preview container; displays a centered placeholder message ("Click Preview to run the widget in a sandbox") when empty.
- `.widget-source-dialog .widget-preview-region iframe`: fills the container, block-size 100%, inline-size 100%, no border.
- `.widget-source-dialog .widget-preview-status`: small text for scanner error messages, styled with the existing `--balaur-content-on-dark-muted` token.
- `.widget-source-dialog .widget-preview-region` and the source `<pre>` stack vertically at `@container balaur-content (width <= 760px)` and sit side-by-side above that breakpoint.

The dialog's existing `inline-size: min(56rem, calc(100% - 24px))` provides enough room for a side-by-side layout at desktop widths. The preview region gets `flex: 1` or `grid` remaining space.

## Testing Decisions

### Single consumer seam: `showWidgetSourceReview` / `#widgetSourceDialog`

The preview adds no new security boundary — it reuses `buildWidgetDocument` and `validateWidgetSource`, which are already tested via `widgets/widget-runtime.test.js`. The test surface is the *consumer*: the dialog's Preview button and its teardown.

### Browser-check assertions

The browser-check smoke suite (`node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline`) should gain assertions for:

1. **Preview renders without console errors**: open the dialog via `showWidgetSourceReview` with the `widgets/focus-balaur.html` source as a known-good fixture; click Preview; assert the iframe loads and no uncaught errors appear in the console.
2. **Scanner-failing source shows diagnostic, no iframe**: open the dialog with source that violates a scanner cap (e.g., exceeds 128 KiB, or contains a forbidden element); click Preview; assert the preview region shows the error message and no iframe is created.
3. **Blob URL teardown on close**: open the dialog, click Preview, close the dialog; assert the preview iframe is removed from the DOM and the Blob URL has been revoked (no lingering `blob:` references in the document).
4. **Both entry points reach the dialog**: trigger a `widget.create` proposal and verify the `<details>` Preview button opens `#widgetSourceDialog`; trigger `balaur-widget-view-source` and verify the dialog opens with the correct source.

### Manual verification

- Narrow-layout inspection: resize to ≤760px and verify source/preview stack vertically without horizontal scroll (AGENTS §11).
- Preview does not evict placed widgets: activate 6 placed widgets, then preview a 7th; assert all 6 remain active.
- Re-preview: open dialog, preview, edit source in-place (if editable — today it is read-only, so re-preview after closing and re-opening with different source), verify the old iframe is replaced.

### Prior art

- `widgets/widget-runtime.test.js` tests `buildWidgetDocument` and `validateWidgetSource` — the primitives the preview reuses. Do not re-test them.
- The browser-check skill's existing smoke assertions (no uncaught errors, valid JSON Canvas, dialog render) provide the pattern for the new assertions.

## Out of Scope

- **DPU `stream*` APIs** (`streamHTMLUnsafe`, `streamAppendHTMLUnsafe`, `setHTMLUnsafe({runScripts})`): flag-gated Chrome 148, Chromium-only, no polyfill allowed by the no-build/no-CDN rule. The streaming polyfill does not actually stream. Value (incremental render) is cosmetic over the atomic preview. Revisit only when `stream*` hits Baseline and a real user pain demands incremental rendering.
- **SSE provider streaming**: the `runRemoteAssistant` flow requires whole-JSON parse (§6); streaming cannot be integrated without a protocol change. Deferred to a separate project.
- **Token-by-token streaming preview**: depended on DPU streaming; cut with it.
- **`setHTML` + Sanitizer API hardening of `markdownToHTML`**: `markdownToHTML` is already escape-by-construction. Sanitizer is defense-in-depth against a non-existent insertion path, with no Safari support. Already deferred on `para/resources/web-platform-2026-07.md`.
- **Full `BalaurWidgetFrameElement` in preview**: placement-coupled runtime (MessageChannel, theme tokens, heartbeats, active-widget cap, IntersectionObserver, visibility/pause) is irrelevant to a transient pre-Apply look.
- **Widget-source-text streaming into `<pre>`** (watch the code generate): thin payoff without DPU streaming preview; deferred.
- **Keeping widget/AI-card runtime alive across `switchCanvas()`**: tangential architecture change; today `aiCardRuntime.clear()` on switch is accepted.
- **Editable source in the preview dialog**: the dialog displays read-only source today. Making it editable would create a divergence between preview and canonical state. Not in scope.

## Further Notes

- The preview bootstrap source is intentionally distinct from `elements/widget-frame.js`'s `BOOTSTRAP_SOURCE`. The frame bootstrap sets up MessageChannel handling, theme projection, and heartbeat responses — none of which the preview needs. Sharing the bootstrap would silently couple the preview to the frame runtime's evolution.
- The dialog's `showWidgetSourceReview` function currently has no teardown on close. The preview introduces a `close` event listener for Blob URL revocation and iframe removal. This listener is attached once at dialog creation time (inside the `if(!dialog)` branch), not per-call.
- The Preview button should be visually distinct from the Close button (e.g., primary action styling) to signal that it is the dialog's main action, not a secondary affordance. Use the existing `.button` class from the design system.
- The preview iframe's `title` attribute should be set to the widget title for accessibility (matching `elements/widget-frame.js:191`).
- The `operationDescription` change for `widget.create` `<details>` is additive: the existing `<pre>` source block remains; the Preview button is appended after it. Event delegation on the `.ai-operation-list` container should handle the new button's click, following the existing delegation pattern in `assistantProposal`.
