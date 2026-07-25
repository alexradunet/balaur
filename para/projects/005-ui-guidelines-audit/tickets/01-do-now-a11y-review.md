---
phase: review
status: done
ticket: 01
date: 2026-07-26
verdict: approved
---

# Standards review: ticket 01 — a11y do-now batch

## Verdict: APPROVED

## Findings

- [judgement] `index.html:112` — ticket verification imprecision: the ticket says `grep -c 'aria-label'` should increase by exactly 10, but `grep -c` counts lines (31→39 = +8), not attribute occurrences (34→44 = +10). Two new aria-labels (zoomLabel, fitView) landed on a line that already contained two (zoomOut, zoomIn). The implementation correctly adds 10 aria-label attributes as intended. The impl artifact identifies and explains this discrepancy. Not a code defect.

- [judgement] `para/projects/005-ui-guidelines-audit/tickets/01-do-now-a11y-impl.md` — artifact number accuracy: the impl artifact states "attribute count increase is exactly 10 (36→46)" but the actual pre-change count is 34, not 36 (post-change 44, not 46). The delta (+10) is correct. Minor artifact inaccuracy; does not affect shipped code.

## Scope verification

All 13 steps (16 individual changes) implemented, nothing more:

| Step | Change | Status |
|------|--------|--------|
| 1 | aria-label on 4 canvas tool buttons (select, pan, connect, note) | ✅ |
| 2 | aria-label on zoomLabel and fitView | ✅ |
| 3 | aria-label on newGroup | ✅ |
| 4 | minimap div→button with type="button" and aria-label | ✅ |
| 5 | role="status" on saveState and lifeIndexStatus; aria-live="polite" on aiMessages | ✅ |
| 6 | aria-label on aiPrompt and journalBody textareas | ✅ |
| 7 | aria-hidden="true" on 3 dialog glyph spans (✓, ✎, ✦) | ✅ |
| 8 | spellcheck="false" on aiModel input | ✅ |
| 9 | curly apostrophe U+2019 in "this browser's" | ✅ |
| 10 | #canvasTitle:focus-visible rule in shell.css | ✅ |
| 11 | .canvas:focus-visible rule in canvas.css | ✅ |
| 12 | .minimap button resets (padding: 0; cursor: pointer) | ✅ |
| 13 | toggleAIKey aria-label sync in app.js | ✅ |

## Correctness checks

- **Unicode bytes verified** (python3 byte-level inspection of af8d624:index.html):
  - U+2026 (ellipsis) in "Preparing files…" — present (`\xe2\x80\xa6`)
  - U+2019 (curly apostrophe) in "card's side handle" — present (`\xe2\x80\x99`)
  - U+2019 (curly apostrophe) in "this browser's localStorage" — present (`\xe2\x80\x99`)
  - U+2212 (minus) on zoom-out button — present (`\xe2\x88\x92`)
  - U+FF0B (fullwidth plus) on zoom-in button — present (`\xef\xbc\x8b`)
  - No ASCII flattening survived into the final commit.

- **Specificity**: `#canvasTitle:focus-visible` (1,1,0) overrides `#canvasTitle` (1,0,0) within @layer shell. `.canvas:focus-visible` (0,2,0) overrides `.canvas` (0,1,0) within @layer canvas. Both correctly defeat the base `outline: 0`.

- **Minimap div→button**: CSS uses `.minimap` class selector (no `div.minimap` type selectors exist). app.js uses `$("#minimap").onclick` (ID selector, element-agnostic) and `.closest(".minimap")` (class selector). Both work on button. Button resets (`padding: 0; cursor: pointer`) added to `.minimap` rule. Global foundation.css provides `font: inherit` and `color: inherit` for all buttons.

- **Aria well-formedness**: all aria-label values non-empty; aria-hidden values are "true"; aria-live value is "polite"; role values are "status". No malformed attributes.

- **Layer placement**: shell.css rule inside @layer shell; canvas.css rules inside @layer canvas. Matches AGENTS.md §11 and styles/layers.css order.

- **Token usage**: both focus-visible rules use `var(--balaur-border-focus)`, defined in tokens.css and widely used elsewhere.

- **app.js style**: single-line handler extension preserves existing compressed style. No reformatting of surrounding code.

- **Out-of-scope boundaries**: sw.js untouched (zero diff). No CACHE_NAME bump. No storage/, elements/, or widgets/ changes.

## Git hygiene

- Two commits in range: a05fcc6 (implement) and af8d624 (Unicode fix). Linear history, clean parent chain (1bbebbd → a05fcc6 → af8d624).
- `git diff --check` clean (no whitespace errors).
- Staged paths in a05fcc6: app.js, index.html, styles/canvas.css, styles/shell.css, plus the impl artifact. Matches ticket instruction to include the artifact.
- af8d624 touches only index.html and the impl artifact — minimal fix scope.
- Process note: a05fcc6 was created after the worker amended an earlier attempt (pre-range). No amend within the reviewed range. The two-commit structure is a reasonable response to the Unicode flattening issue.

## Summary

Zero hard violations, two judgement findings (both artifact-number discrepancies, not code defects). All 16 changes implemented correctly. Unicode characters verified at byte level. Specificity, layer placement, token usage, and aria well-formedness all check out. The minimap div→button conversion is safe for CSS and JS wiring. Confidence: high.
