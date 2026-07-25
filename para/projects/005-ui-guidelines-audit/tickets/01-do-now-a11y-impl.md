---
phase: implement
status: done
project: 005-ui-guidelines-audit
ticket: 01
date: 2026-07-26
commit: af8d624
branch: ripe-gecko
---

# Implementation: Accessibility do-now batch

## Steps completed

- [x] Step 1: Canvas tool buttons — added `aria-label` to select, pan, connect, note tools — verified: `grep -n 'data-tool.*aria-label' index.html` shows all four
- [x] Step 2: Zoom tools — added `aria-label="Reset zoom to 100%"` to zoomLabel, `aria-label="Fit canvas to view"` to fitView — verified: line 112 grep
- [x] Step 3: Sidebar newGroup — added `aria-label="Add group"` — verified: line 60 grep
- [x] Step 4: Minimap div→button — replaced `<div class="minimap">` with `<button type="button" class="minimap" aria-label="Fit canvas to view">`, closing tag updated — verified: line 115 grep
- [x] Step 5: Live regions — added `role="status"` to saveState and lifeIndexStatus, `aria-live="polite"` to aiMessages — verified: `grep -o 'role="status"' | wc -l` = 8 (was 6, +2)
- [x] Step 6: Missing labels — added `aria-label="Ask Balaur about this canvas"` to aiPrompt, `aria-label="Daily note"` to journalBody — verified: grep
- [x] Step 7: Decorative dialog glyphs — added `aria-hidden="true"` to ✓ (taskDialog), ✎ (aiNoteDialog), ✦ (aiSettingsDialog) spans — verified: grep
- [x] Step 8: AI model input — added `spellcheck="false"` to aiModel — verified: grep
- [x] Step 9: Curly apostrophe — replaced straight `'` with curly `'` (U+2019) in "this browser's localStorage" — verified: `cat -v` shows `M-bM-^@M-^Y`
- [x] Step 10: `#canvasTitle:focus-visible` — added focus-visible rule with `outline: 2px solid var(--balaur-border-focus); outline-offset: 4px` after the `#canvasTitle` block in shell.css — verified: grep
- [x] Step 11: `.canvas:focus-visible` — added focus-visible rule with `outline: 2px solid var(--balaur-border-focus); outline-offset: -2px` before `.canvas.panning` in canvas.css — verified: grep
- [x] Step 12: `.minimap` button resets — added `padding: 0; cursor: pointer;` to existing `.minimap` rule in canvas.css — verified: grep
- [x] Step 13: toggleAIKey aria-label sync — appended `setAttribute("aria-label",...)` after textContent assignment — verified: `grep -n 'toggleAIKey' app.js` shows the sync

## Files changed

- `index.html` — 10 new aria-labels (4 tool buttons, zoomLabel, fitView, newGroup, minimap button, aiPrompt, journalBody), 2 role="status" (saveState, lifeIndexStatus), 1 aria-live="polite" (aiMessages), 3 aria-hidden="true" (dialog glyphs), 1 spellcheck="false" (aiModel), 1 curly apostrophe, minimap div→button conversion
- `styles/shell.css` — added `#canvasTitle:focus-visible` rule
- `styles/canvas.css` — added `.canvas:focus-visible` rule, added `padding: 0; cursor: pointer;` to `.minimap`
- `app.js` — extended toggleAIKey handler with aria-label sync (one line, no reformatting)

## Verification results

```
node --check app.js main.js               → OK
node --check elements/*.js                 → all OK
git diff --check                           → clean (no whitespace errors)
aria-label occurrences: 36 → 46 (+10)     → exact match
role="status" occurrences: 6 → 8 (+2)     → exact match
toggleAIKey aria-label sync               → present on line 1934
browser-check smoke (default)              → 14/14 PASS
browser-check smoke (380×760)              → 14/14 PASS
```

## Issues encountered

- Initial edit of the zoom toolbar line inadvertently replaced Unicode minus (U+2212) and fullwidth plus (U+FF0B) glyphs with ASCII hyphen-minus and plus. Detected via `cat -v` comparison against `git show HEAD:index.html`. Fixed by re-editing the line with the correct Unicode characters restored.
- The ticket's `grep -c 'aria-label'` gate counts lines, not occurrences. Line count went 31→39 (+8) because two new aria-labels landed on lines that already had other aria-labels. The actual attribute count increase is exactly 10 (36→46), matching the ticket's intent.
- **Fixes round (orchestrator review):** Two Unicode characters were flattened during the bulk edit of `index.html`. (a) Line 74: `Preparing files…` (U+2026) became `Preparing files...` (three ASCII dots). (b) Line 90: `drag a card's side handle` (U+2019 curly apostrophe) became `drag a card's side handle` (U+0027 straight apostrophe). Both restored via targeted sed with explicit UTF-8 byte sequences. The edit tool's `oldText`/`newText` matching appears to normalize curly quotes and ellipsis to ASCII during file I/O; used `sed -i` with `printf '\xe2\x80\x99'` for the curly apostrophe and direct UTF-8 literal for the ellipsis.
