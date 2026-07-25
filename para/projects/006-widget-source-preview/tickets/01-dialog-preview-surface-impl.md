---
phase: implement
status: done
project: 006-widget-source-preview
ticket: 01
date: 2026-07-25
commit: 568f993
branch: 006-widget-source-preview
---

# Implementation: Dialog preview surface

## Steps completed
- [x] Step 1: Add `buildWidgetDocument` import to `app.js` — verified: `node --check app.js` exits 0
- [x] Step 2: Define `PREVIEW_BOOTSTRAP_SOURCE` constant — verified: `node --check app.js` exits 0
- [x] Step 3: Modify dialog innerHTML (Preview button + two-pane body) — verified: `node --check app.js` exits 0
- [x] Step 4: Close-event teardown (iframe removal + Blob URL revocation) — verified: `node --check app.js` exits 0
- [x] Step 5: Preview button click handler (buildWidgetDocument + Blob URL + sandboxed iframe) — verified: `node --check app.js` exits 0; disabled when no source
- [x] Step 6: CSS for two-pane layout and preview region — added after existing `.widget-source-dialog` rules in `styles/elements.css`
- [x] Step 7: Narrow-screen stacking rule — added inside `@container balaur-content (width <= 760px)` in `styles/responsive.css`
- [x] Step 9: Browser-check smoke suite — 14/14 checks passed (including offline)
- [x] Step 10: Node storage test suite — 204/204 tests pass, 0 failures

## Files changed
- `app.js` — added `buildWidgetDocument` import, `PREVIEW_BOOTSTRAP_SOURCE` constant, modified `showWidgetSourceReview` (new innerHTML with Preview button + two-pane body, close-event teardown, preview click handler, disabled state)
- `styles/elements.css` — added `.widget-source-dialog-actions`, `.widget-source-dialog-body`, `.widget-source-region`, `.widget-preview-region`, `.widget-preview-placeholder`, `.widget-preview-status` rules
- `styles/responsive.css` — added `.widget-source-dialog-body { grid-template-columns: 1fr; }` inside the `@container balaur-content (width <= 760px)` block

## Verification results
- `node --check app.js` → OK
- `git diff --check` → no whitespace errors
- `git status --porcelain` → only 3 in-scope files modified (app.js, styles/elements.css, styles/responsive.css)
- `node --test storage/phase*.test.js storage/phase4-backup.test.js storage/phase-query.test.js storage/note-repository.test.js` → 204 pass, 0 fail
- `node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline` → 14/14 checks passed

## Security invariants (AGENTS.md §10)
- Preview iframe uses exactly `sandbox="allow-scripts"` — no `allow-same-origin`, `allow-forms`, `allow-popups`, `allow-downloads`
- Blob URL is opaque origin; widget cannot access host cookies, storage, or DOM
- CSP injected via `buildWidgetDocument` (`<meta http-equiv>`) blocks all network, workers, nested frames, forms
- Scanner caps (128 KiB source, 500 elements, 64 KiB script, 64 KiB style) apply identically via `validateWidgetSource` inside `buildWidgetDocument`
- Preview does NOT register in `activeWidgets` and does NOT count against `MAX_ACTIVE_WIDGETS = 6`
- Apply remains unchanged
- Preview bootstrap (`PREVIEW_BOOTSTRAP_SOURCE`) is distinct from `elements/widget-frame.js` `BOOTSTRAP_SOURCE` — no MessageChannel, no theme tokens, no heartbeats

## Issues encountered
- Drift check: `app.js` and `styles/responsive.css` had changes since planned-at SHA b9f94d7 (from the 005-ui-guidelines-audit feature merge). The changes did not affect the lines modified by this ticket (the `showWidgetSourceReview` function body is identical; the `@container` block is unchanged). Proceeded without issue.
- Test count: ticket expected 197 tests; actual is 204 (tests were added since the ticket was written). All pass.
