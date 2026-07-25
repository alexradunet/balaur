---
phase: implement
status: done
project: 001-directory-vault
ticket: 03
date: 2026-07-25
commit: 42d0f84
branch: 001-directory-vault
---

# Implementation: Ticket 03 — browser-check driver rework (OPFS contract suite + landing-aware boot)

## Pre-flight

- Worktree: `/home/balaur/.paseo/worktrees/10x2zhef/001-directory-vault` (non-main).
- Branch: `001-directory-vault` (not main).
- `git status --porcelain`: clean before edits.
- Blocker [02]: `status: done`, review `verdict: approved` (commit `15d5a5c`). The Vault gate, `DirectoryVault` boot, shell, and `IndexedDbVault` deletion are all landed.
- Binding app-side constraint verified before building the stub: `grep -n showDirectoryPicker app.js` → only `app.js:326` (gate) and `app.js:346` (`handle=await window.showDirectoryPicker({mode:"readwrite"})` inside the `onclick` handler, with the explicit "reference the global at call time" comment). No module-scope capture exists (`grep "showDirectoryPicker =" app.js | grep -v window.` → none). The stub is therefore honored by the real click.
- Drift since plan `e3103b2`: the only driver change was the note-creation assertion (now `last.type === "file" && last.file.startsWith("notes/")`), reflecting that notes became `notes/*.md` file nodes (ADR-0004). Consistent with the plan; worked from live code.

## Steps completed

- [x] **§4.1 `bootPastLanding(session, subdirectory = "vault-smoke")`** added above the `smoke` section. It waits for `#openVaultFolder` present + enabled **and** `window.balaurVaultReady !== undefined` (see deviation D1 below), captures the pre-stub gate state (`messageEmpty`, `buttonEnabled`), installs `window.showDirectoryPicker = async () => (await navigator.storage.getDirectory()).getDirectoryHandle(<subdirectory>, { create: true })`, performs a real CDP click on the button center (computed via `getBoundingClientRect`, not `el.click()`), then waits for `window.balaurCanvas && document.querySelectorAll('.canvas-node').length > 0`. Returns the gate state. Verified: present at `browser-check.mjs:251`; smoke gate record reads `message:empty button:enabled`.
- [x] **Helper applied at every balaurCanvas-wait site.** Definition + 8 call sites confirmed by grep: smoke prelude (`:280`), smoke reload (`:413`), smoke offline (`:423`), components prelude (`:457`), components reload (`:655`), failureSession boot (`:1233`), failureSession reload (`:1370`), shot (`:1917`). The only remaining `waitFor("window.balaurCanvas...")` is inside the helper itself (`:260`). The `eval` subcommand stays raw (no landing handling). Verified: each site's boot path exercised (smoke all-pass; failureSession boot+reload 3/3 via a targeted probe that blocks `/elements/register.js`; shot produced a full-canvas PNG past the gate).
- [x] **§4.2 `contract` subcommand.** `CONTRACT_EVAL` (`:1725`) dynamically imports `/storage/directory-vault.js`, makes a fresh per-run OPFS subdirectory (`contract-main-<ts>`, plus dedicated `contract-restore-<ts>` and `contract-journal-<ts>`), and asserts the full surface; returns `{ ok, failures, checks }`. The `contract(url, flags)` runner (`:1850`) splits `checks` into PASS/FAIL records, adds a console-error record if any, and exits nonzero on any failure. Wired into CLI dispatch (`:1894` `else if (command === "contract")`); usage comment and the "Unknown command" line updated. Verified: `contract` → `All 16 checks passed.`, exit 0 (full output below). All required checks present: create/read round-trip; stat meta shape; exists true/false; list ordering + prefix filter; the three expectedHash outcomes (null create-conflict, mismatch with `details.expected`/`details.actual`, correct succeeds); NOT_FOUND on read; case-fold collision on create; move success + destination-exists conflict; remove success + remove precondition; snapshot/restore round-trip into a second fresh subdirectory (8 files, contents equal); changesSince journal ordering (4 entries, strictly increasing revisions, ops create/modify/move/remove); TypeMismatchError → PATH_COMPONENT.
- [x] **§4.3 smoke rework.** Prelude is now `navigate` → `bootPastLanding(session)` → record `"gate: Vault gate passed, picker button enabled"` (asserts `#vaultLandingMessage` empty and `#openVaultFolder` enabled before the stub). Assertions 1–8 run unchanged after the prelude. Step 9 reload calls `bootPastLanding(session)` again (re-stub + re-pick same subdirectory) before the title/node-count assertions. Step 10 offline: `setOffline(true)` → `reload` → `bootPastLanding(session)` → assert `!!document.querySelector('.canvas') && !!window.balaurCanvas` → `setOffline(false)`. Verified: smoke `--offline` → `All 14 checks passed.`, exit 0; gate record present; reload persistence `7 -> 7`; offline shell from cache PASS.
- [x] **§4.4 SKILL.md.** Added the `contract` command to the Commands block; added gate item 0 and reworded items 9/10 (re-stub + re-pick) in "What the smoke suite checks"; added a "Vault gate and the picker stub (headless)" section (mechanism, the `balaurVaultReady` wait, real CDP click, re-pick-after-reload, `--profile` persistence via the same OPFS subdirectory, `eval` stays raw, manual-smoke boundaries); added a "`contract` subcommand (adapter suite via OPFS)" section. Verified: sections present at SKILL.md:93 and :123.
- [x] **Verification.** `node --check` exit 0; `contract` all pass exit 0; `smoke --offline` all pass exit 0; `git diff --check` clean. `widgets` also run: `All 26 checks passed.`, exit 0 (fixture-based, no app boot). See "Verification results" below.

## Files changed

- `.pi/skills/browser-check/scripts/browser-check.mjs` — added `bootPastLanding` helper; replaced all 8 balaurCanvas-wait sites with it (eval left raw); added the `contract` subcommand + `CONTRACT_EVAL` + CLI dispatch + usage/Unknown-command updates; reworked the smoke prelude (gate record), reload (re-pick), and offline (re-pick) steps; replaced the fixed 500ms post-dblclick sleep with a poll for the new node (deviation D2).
- `.pi/skills/browser-check/SKILL.md` — documented `contract`, the picker-stub mechanism, the re-pick-after-reload behavior, `--profile` persistence, and the manual-smoke boundaries.

## Verification results

`node --check .pi/skills/browser-check/scripts/browser-check.mjs` → exit 0.

`node .pi/skills/browser-check/scripts/browser-check.mjs contract` → exit 0:
```
PASS  create/read round-trip
PASS  stat meta shape  [mediaType=text/markdown size=5 revision=1]
PASS  exists true/false
PASS  list ordering + prefix filter  [all=["a.md","notes/a.md","notes/b.md","tasks/c.md"]]
PASS  expectedHash null create-conflict  [code=WRITE_CONFLICT]
PASS  expectedHash mismatch details  [code=WRITE_CONFLICT]
PASS  expectedHash correct succeeds
PASS  NOT_FOUND on read missing  [code=NOT_FOUND]
PASS  case-fold collision on create  [code=PATH_CASE_COLLISION]
PASS  move success
PASS  move destination-exists conflict  [code=WRITE_CONFLICT]
PASS  remove success
PASS  remove precondition mismatch  [code=WRITE_CONFLICT]
PASS  snapshot/restore round-trip  [files=8 restored=8]
PASS  changesSince journal ordering  [ops=create,modify,move,remove]
PASS  TypeMismatchError -> PATH_COMPONENT  [code=PATH_COMPONENT name=PathError]
All 16 checks passed.
```

`node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline` → exit 0 (stable across 4 runs):
```
PASS  gate: Vault gate passed, picker button enabled  [message:empty button:enabled]
PASS  boot: no uncaught console errors
PASS  boot: no failed asset requests
PASS  render: DOM cards match document nodes  [5/5]
PASS  index: canonical files indexed  [Files · 19 indexed]
PASS  select: card selected + inspector open
PASS  select: corner-bracket frame, no circles  [frame:true brackets:true border:true handlesHidden:true]
PASS  create: portal dblclick navigates without creating nodes  [19 -> 19]
PASS  create: portal probe restores original canvas
PASS  create: note tool on card creates nothing  [6 -> 6]
PASS  create: dblclick on background creates a note  [6 -> 7]
PASS  export: document is valid JSON Canvas
PASS  persist: reload keeps title and node count  [7 -> 7]
PASS  offline: shell renders from cache
All 14 checks passed.
```

`node .pi/skills/browser-check/scripts/browser-check.mjs widgets` → exit 0, `All 26 checks passed.`

`git diff --check .pi/skills/browser-check/` → clean.

## Issues encountered / deviations

**D1 — `bootPastLanding` also waits on `window.balaurVaultReady` (driver robustness, in scope).** The plan's literal helper waited only for the button to be enabled. The button exists *enabled in the markup* before `app.js` attaches its `onclick` handler (the handler is wired inside `bootCanvasApp`, and `window.balaurVaultReady` is assigned immediately after). With the `Fetch`-intercepted failure session (`/elements/register.js` blocked), the extra interception latency widens this window so the helper can CDP-click *before* the pick handler exists; the click is then a no-op and boot times out. Diagnosed with a probe: immediately after reload `buttonEnabled:true` but `balaurVaultReady` undefined; clicking only after `balaurVaultReady` is defined boots cleanly. Fix: the helper's first wait requires `window.balaurVaultReady !== undefined` in addition to the enabled button. This is a driver-only robustness change within the helper the ticket asked me to add; it does not touch app code and does not weaken the gate assertion (the gate record still asserts the button enabled + empty message). Verified: failureSession boot + reload re-pick now pass 3/3 with no console errors; smoke/contract unaffected.

**D2 — background-note assertion polls instead of a fixed 500ms sleep (driver robustness, in scope).** The plan's literal step 7 used `setTimeout(500)` after the background dblclick. Notes are now async file-backed (`createNoteOnCanvas` → `noteRepository.createNote` + `reloadCanvasDocuments`, ADR-0004, which landed after the plan). A probe measured note-creation latency: ~324ms with no card selected, but ~737ms when a card is selected (the smoke scenario, since step 4 leaves a card selected and the first click of the dblclick deselects + re-renders). The fixed 500ms therefore read the document before the async note landed (`6 -> 6`), and the deferred note then raced into the reload step (`6 -> 7`), failing both step 7 and step 9. Fix: after the dblclick, poll `window.balaurCanvas.getDocument().nodes.length > <before>` (5s ceiling, swallowed on timeout so a genuine "no note" still FAILs gracefully). The assertion logic (`count === before + 1 && isNoteFile`) is unchanged. This is a driver-only timing-robustness change to the smoke step the ticket reworks; it does not touch app code. Verified: step 7 now reads `6 -> 7` and step 9 reads `7 -> 7`, stable across 4 runs.

**F1 — `components` suite aborts at a pre-existing stale inspector probe, NOT at a boot site (out of scope, not fixed).** My `bootPastLanding` change un-blocks the `components` boot (it now reaches the suite body; previously the gate blocked `balaurCanvas` at the prelude, so the suite never ran on the new app). It then times out at `waitFor('#inspector [data-field-key="text"]')` (~line 854), which is the suite's *own* inspector probe, not a site I changed. A probe confirmed the boot/selection is healthy (`inspectorOpen:true`, `selectedNode:true`) but a text node's inspector renders only geometry fields `[x, y, width, height]` — text is edited inline on the card, so there is no `data-field-key="text"` control. That probe predates and is independent of ticket 03 (the inspector element's text-edit model changed at some point and the probe was never updated; it was masked because the suite couldn't boot past the gate). Fixing it requires redesigning the probe's edit interaction against the current inspector and is not specified by this ticket's plan, so per the implement protocol ("if the plan is ambiguous, STOP and report; do not guess"; touch only specified work) I did **not** modify it. The two `components` boot sites I own (prelude `:457`, reload `:655`) are correct and exercised; the failure is downstream of them. Recommend a small follow-up ticket to update the inspector probe (and confirm the same for any other post-gate probe drift) — it is driver-maintenance, file-disjoint from the vault work. `widgets` (the other "where time allows" suite) passes fully (26/26).

No STOP conditions triggered: headless Chrome passes the gate (`showDirectoryPicker` + `crypto.subtle` present on localhost), OPFS (`navigator.storage.getDirectory()`) is available in the headless profile, the picker stub takes effect (the app honors the global at call time), and no out-of-scope file was modified.
