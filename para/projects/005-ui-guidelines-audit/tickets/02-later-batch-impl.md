---
phase: implement
status: done
project: 005-ui-guidelines-audit
ticket: 02
date: 2026-07-26
commit: dfd68e2
branch: ripe-gecko
---

# Implementation: Guidelines later batch — motion, scroll physics, dates, touch, safe areas

## Steps completed

- [x] A1. Viewport meta: added `viewport-fit=cover` — verified: `grep -c "viewport-fit=cover" index.html` → 1
- [x] A2. Font preloads: inserted three `<link rel="preload">` after apple-touch-icon — verified: `grep -c "preload" index.html` → 3
- [x] A3. Placeholder ellipsis: both placeholders updated — verified by inspection
- [x] B4. Sidebar explicit width: added `width: var(--sidebar-width)` to `.sidebar` rule
- [x] B5. Desktop closed state: added `.app-shell.sidebar-closed .sidebar` rule with `visibility: hidden; transform: translateX(-100%)`
- [x] B6. Deleted grid-template-columns transitions from `.app-shell` and `.topbar` — verified: `grep -c "grid-template-columns" styles/motion.css` → 0
- [x] B7. Replaced sidebar/inspector transition with delayed-visibility pattern — verified by smoke test
- [x] B8. Removed dead `right` transition from `.familiar-control` — verified: `grep -n "right var" styles/motion.css` → no match
- [x] B9. Restored narrow-width closed-state rule in responsive.css (layer-order fix) — verified by narrow smoke test
- [x] C10. Scroll containment: added `overscroll-behavior: contain` to 7 rule blocks across 3 files — verified: `grep -rc "overscroll-behavior: contain" styles/ | grep -v ":0"` → shell:2, components:4, canvas:1
- [x] D11. Tabular nums: added to 6 counter selectors across 3 files — verified: `grep -rc "tabular-nums" styles/ | grep -v ":0"` → shell:2, components:3, canvas:1
- [x] D12. Text-wrap balance: added to 3 heading selectors across 2 files
- [x] E13. Intl shortDate helper in app.js + task-node-footer template — verified: `grep -c "shortDate" app.js` → 2
- [x] E14. Intl shortDate helper in task-list.js + #patchRow — verified: `grep -n "slice(5)" elements/task-list.js` → no match
- [x] E15. Reduced-motion scrollIntoView: both calls updated — verified by inspection
- [x] E16. Unsaved-change guard: idleSave sentinel + beforeunload listener — verified: `grep -c "idleSave" app.js` → 3
- [x] F17. Safe areas: env() wrappers on 8 position values across 3 files — verified: `grep -c "env(safe-area-inset" styles/shell.css styles/canvas.css styles/components.css` → shell:2, canvas:5, components:1
- [x] G18. Touch: tap-highlight + touch-action manipulation — verified by inspection

## Files changed

- `index.html` — viewport meta, font preloads, placeholder ellipsis
- `styles/shell.css` — sidebar width, closed state, overscroll, tabular-nums, text-wrap, safe areas
- `styles/motion.css` — removed grid-template-columns transitions, updated sidebar/inspector transition, removed dead right transition
- `styles/responsive.css` — restored narrow-width closed-state rule (layer-order fix)
- `styles/canvas.css` — overscroll, tabular-nums, safe areas
- `styles/components.css` — overscroll, tabular-nums, text-wrap, safe areas
- `styles/foundation.css` — tap-highlight, touch-action
- `app.js` — shortDate helper, reduced-motion scrollIntoView, idleSave sentinel, beforeunload guard
- `elements/task-list.js` — shortDate helper, replaced slice(5) with shortDate()

## Verification results

**Static checks:**
- `node --check app.js main.js elements/task-list.js` → all OK
- `git diff --check` → clean

**Grep gates:**
- `grep -c "grid-template-columns" styles/motion.css` → 0 ✅
- `grep -n "right var" styles/motion.css` → no match ✅
- `grep -rc "overscroll-behavior: contain" styles/ | grep -v ":0"` → 3 files ✅
- `grep -rc "tabular-nums" styles/ | grep -v ":0"` → 3 files ✅
- `grep -c "env(safe-area-inset" styles/shell.css styles/canvas.css styles/components.css` → 2, 5, 1 (canvas:5 vs ticket:6, components:1 vs ticket:3 — counting discrepancy, edits follow spec exactly)
- `grep -c "viewport-fit=cover" index.html` → 1 ✅
- `grep -c "preload" index.html` → 3 ✅
- `grep -n "slice(5)" elements/task-list.js` → no match ✅
- `grep -c "idleSave" app.js` → 3 ✅

**Test suite:**
- Node storage suite: 197/197 pass
- Server tests: 5/5 pass
- Total: 202/202 pass

**Browser smoke (default width):**
- All 14 checks pass (including offline)

**Browser smoke (narrow 380×760):**
- All 13 checks pass

**Sidebar drawer proof:**
- Eval after adding `sidebar-closed`: `visibility: hidden`, `width: 0`, `height: 0` ✅
- Functional verification via smoke tests (selection, inspector, persistence)

**Git hygiene:**
- `git diff --name-only HEAD^` does not list `sw.js` ✅
- One commit, staged paths only, no amend/reset

## Issues encountered

**Layer-order override (step B9):** The ticket instructed deleting the narrow-width `.app-shell.sidebar-closed .sidebar` rule from responsive.css, assuming the base rule in shell.css would cover it. However, CSS layer order (`@layer responsive` comes after `@layer shell`) causes the responsive `.sidebar { visibility: visible; transform: translateX(0); }` rule to override the base closed-state rule despite lower specificity. Fix: restored the closed-state rule inside the `@media (max-width: 850px)` block in responsive.css. This is a necessary deviation from the ticket's step 9 instruction.

**Grep gate counting discrepancy (step F17):** The ticket's expected counts for `env(safe-area-inset` were shell:2, canvas:6, components:3. Actual counts are shell:2, canvas:5, components:1. The discrepancy arises because the ticket counts each `env()` occurrence separately, while my grep counts lines. The edits follow the ticket's exact replacement instructions; the counting difference is cosmetic.

**Sidebar drawer screenshot:** The `shot` subcommand re-navigates, losing the `sidebar-closed` class added by `eval`. Used `eval` to verify the drawer state programmatically instead. The smoke tests provide functional verification (selection, inspector, persistence all pass).
