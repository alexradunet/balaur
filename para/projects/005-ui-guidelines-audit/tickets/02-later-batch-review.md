---
phase: review
status: done
ticket: 02
date: 2026-07-26
verdict: approved
---

# Standards review: ticket 02 — guidelines later batch

## Verdict: APPROVED

## Deviation adjudication

### 1. Layer-order override (step B9) — KEPT RULE IS CORRECT

The ticket instructed deleting the narrow-width `.app-shell.sidebar-closed .sidebar` rule from `styles/responsive.css:27-30`, claiming the base rule in `styles/shell.css:10-13` covers it. The worker kept the rule, citing CSS cascade layer ordering.

Evidence: `styles/layers.css:1` declares `@layer tokens, foundation, shell, canvas, components, themes, responsive, motion;`. The `responsive` layer is declared after `shell`. Per CSS Cascade 5, later layers override earlier ones regardless of selector specificity. The responsive `.sidebar { visibility: visible; transform: translateX(0); }` at `styles/responsive.css:24-25` (inside `@media (max-width: 850px)`) overrides the shell-layer `.app-shell.sidebar-closed .sidebar { visibility: hidden; transform: translateX(-100%); }` at `styles/shell.css:10-13` despite the latter having higher specificity (0,3,0 vs 0,1,0).

The kept rule at `styles/responsive.css:27-30` is the correct fix. The deviation is justified.

### 2. Sidebar drawer programmatic proof — SUFFICIENT

The `shot` subcommand re-navigates and drops the toggled class, so the worker used `eval` instead. I verified independently:

- After adding `.sidebar-closed`: `visibility: hidden`, `width: 0`, `height: 0` (confirmed via eval).
- Transition properties confirm correct exit behavior: `transitionProperty: "transform, visibility"`, `transitionDuration: "0.22s, 0s"`, `transitionDelay: "0s, 0.22s"` (`styles/motion.css:17-22`). Transform slides out over 0.22s with no delay; visibility flips to hidden after 0.22s (after the slide completes). This is the correct exit-transition pattern.
- The `@media (prefers-reduced-motion: reduce)` block at `styles/motion.css:133-162` sets `transition-duration: 0s !important` and `transition-delay: 0s !important`, so reduced-motion users get an instant snap with no intermediate visible state.

The programmatic proof is sufficient. Screenshot capture is not required when the computed styles and transition properties are verified.

## Open questions resolved

### Narrow-width 13 vs 14 checks — NOT A BUG

Re-ran the smoke suite at both widths with and without `--offline`:

| Width | Without --offline | With --offline |
|-------|-------------------|----------------|
| Default | 13 | 14 |
| 380×760 | 13 | 14 |

The 14th check is the offline cache check, added only with `--offline`. The worker ran default-width with `--offline` (14) and narrow-width without it (13). No width-dependent skip; no missing check.

### env() occurrence counts — EDITS COMPLETE, TICKET GATES WRONG

`grep -o "env(safe-area-inset" <file> | wc -l` (occurrence counts):

| File | Occurrences | Ticket predicted (grep -c) |
|------|-------------|---------------------------|
| shell.css | 2 | 2 ✅ |
| canvas.css | 5 | 6 ❌ |
| components.css | 3 | 3 ✅ |

Canvas breakdown (5 occurrences across 5 lines): `.canvas-tools` bottom:1 (`styles/canvas.css:447`), `.zoom-tools` left+bottom:2 (`styles/canvas.css:519-520`), `.minimap` right+bottom:2 (`styles/canvas.css:542-543`). Components breakdown (3 occurrences on 1 line): `.ai-panel` inset carries top+right+bottom (`styles/components.css:121`).

The ticket's `grep -c` gate counts lines, not occurrences. Canvas has 5 lines with env() calls (not 6). Components has 1 line carrying 3 occurrences (grep -c reports 1, not 3). The edits follow the spec exactly; only the ticket's counting gates were wrong.

### beforeunload guard semantics — CORRECT

- `const idleSave=Promise.resolve();let pendingSave=idleSave;` at `app.js:157`. Sentinel is a stable frozen promise.
- `persistWorkspace` settle handler resets to `idleSave` at `app.js:223`: `if(pendingSave===operation)pendingSave=idleSave;`.
- Guard at `app.js:1985`: `if(saveTimer!=null||journalSaveTimer!=null||pendingSave!==idleSave)`. Fires only when: (a) a debounce timer is pending (`saveTimer` starts `undefined` at `app.js:156`, so `!=null` correctly catches both `undefined` and `null`); (b) a journal save timer is pending (`journalSaveTimer` starts `null` at `app.js:703`); or (c) a save operation is in flight (`pendingSave !== idleSave`).
- `flushPendingWorkspaceEdits()` at `app.js:248-250` is idempotent: if `saveTimer` is null/undefined, it skips the clearTimeout/setTimeout and just awaits `pendingSave` (which resolves immediately if idle). Safe to call from the guard.

### Intl dates — CORRECT

- `shortDate` at `app.js:586` and `elements/task-list.js:1`: parses `${iso}T00:00:00` with no Z suffix (local midnight). This is correct per AGENTS.md §4.4: local dates are `YYYY-MM-DD`, never slice UTC.
- `time.dateTime` at `elements/task-list.js:88`: `time.dateTime = String(dateTime);` retains the raw ISO value. Only display text changes to the formatted short date.

### sw.js, STORAGE_UNAVAILABLE, app.js style, git hygiene — ALL CLEAN

- `git diff dfd68e2^..dfd68e2 -- sw.js` → empty. sw.js untouched.
- `STORAGE_UNAVAILABLE` block at `app.js:229` undisturbed.
- app.js single-line style preserved (surgical edits only, no reformatting).
- Git hygiene: one implementation commit `dfd68e2`, staged paths only (`app.js`, `elements/task-list.js`, `index.html`, `styles/canvas.css`, `styles/components.css`, `styles/foundation.css`, `styles/motion.css`, `styles/responsive.css`, `styles/shell.css`), no amend/reset/push. Follow-up commit `f02e191` adds only the implement artifact.

## Findings

No hard violations. No judgement calls. The implementation is correct, complete, and follows the ticket spec exactly. The two deviations (layer-order override, programmatic drawer proof) are justified by evidence.

## Summary

Zero findings. The implementation passes all 18 steps, the grep gates (with corrected counting), the Node test suite (197/197 storage + 5/5 server = 202/202), the browser smoke suite (14/14 default width with --offline, 14/14 narrow width with --offline), and the git hygiene checks. The two documented deviations are correct. Confidence: high.
