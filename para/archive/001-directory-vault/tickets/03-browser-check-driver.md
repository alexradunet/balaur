---
phase: ticket
status: done
project: 001-directory-vault
ticket: 03
blocked-by: [02]
worker: "f513896"
branch: "001-directory-vault"
shared-blast-radius: false
---

# Ticket 03: browser-check driver rework — OPFS contract suite + landing-aware boot

## What to build

The headless browser-check driver learns to get past the Vault gate and to
verify the DirectoryVault contract. A shared `bootPastLanding` helper stubs
`window.showDirectoryPicker` with an OPFS handle and CDP-clicks the real gate
button, applied at every site that waits on `window.balaurCanvas`; a new
`contract` subcommand exercises the full adapter surface against a fresh OPFS
subdirectory; the smoke suite re-stubs and re-picks after every reload
(including offline). This ticket owns ALL driver-file changes.

Read `.pi/skills/browser-check/SKILL.md` fully first, including the headless
event-retargeting caveat. No new dependencies (Node built-in `WebSocket`/`fetch`).
Source of truth: `plan.md` §4 (§4.1 helper, §4.2 contract, §4.3 smoke rework,
§4.4 SKILL.md).

## Acceptance criteria

- [ ] §4.1 `bootPastLanding(session, subdirectory = "vault-smoke")` helper added near the top: wait for `#openVaultFolder` present + enabled; install `window.showDirectoryPicker = async () => (await navigator.storage.getDirectory()).getDirectoryHandle(<subdirectory>, { create: true })`; perform a REAL CDP click on the button center (user-gesture semantics, not `el.click()`); wait for `window.balaurCanvas && document.querySelectorAll('.canvas-node').length > 0`.
- [ ] The helper is called at every balaurCanvas-waiting site: smoke (254), smoke reload (386), components (430), widgets (628/1206/1343), shot (1723). The `eval` subcommand stays raw (no landing handling).
- [ ] §4.2 `contract` subcommand added (structured like `smoke`: results array, PASS/FAIL lines, exit code): navigate (no `bootPastLanding`), run one async `evaluate` that dynamically imports `/storage/directory-vault.js`, creates a fresh per-run OPFS subdirectory (`"contract-" + Date.now()`), asserts the full surface, and returns `{ ok, failures }`. Checks: create/read round-trip; `stat` meta shape; `exists` true/false; `list` ordering + prefix filter; the three `expectedHash` outcomes (null create-conflict, hash mismatch with `details.expected`/`details.actual`, correct hash succeeds); `NOT_FOUND` on missing read; case-fold collision on create (`PATH_CASE_COLLISION`); `move` success + destination-exists `ConflictError`; `remove` success + remove precondition mismatch; `snapshot`/`restore` round-trip into a second fresh subdirectory; `changesSince` journal ordering (strictly increasing revisions; create/modify/move/remove present); `TypeMismatchError` → `PATH_COMPONENT`.
- [ ] `contract` wired into CLI dispatch (`else if (command === "contract")`); usage comment at the top and the "Unknown command" line updated.
- [ ] §4.3 `smoke` rework: prelude becomes navigate → `bootPastLanding(session)` → a `"gate: Vault gate passed, picker button enabled"` record (assert `#vaultLandingMessage` empty and `#openVaultFolder` enabled before the stub). Assertions 1–8 unchanged after the prelude. Step 9 reload: after `session.reload()`, call `bootPastLanding(session)` again (re-stub + re-pick the same subdirectory), then the existing title/node-count assertions. Step 10 `--offline`: `setOffline(true)` → reload → `bootPastLanding` → assert `!!document.querySelector('.canvas') && !!window.balaurCanvas` → `setOffline(false)`. `components`, `widgets` (incl. failureSession instances at 1206/1343), and `shot`: replace each post-`navigate()` `waitFor("window.balaurCanvas...")` with `bootPastLanding(session)`.
- [ ] §4.4 `SKILL.md` updated: document the `contract` subcommand; the picker-stub mechanism (`window.showDirectoryPicker` overridden with an OPFS handle, real CDP click on `#openVaultFolder`); re-pick-after-reload behavior (gate shows every load; smoke re-stubs automatically); `--profile` still drives persistence via the same OPFS subdirectory; real-picker flow, external-change reconciliation, and the Firefox/Safari gate are manual smoke.
- [ ] Verify: `node --check .pi/skills/browser-check/scripts/browser-check.mjs` exits 0; with the app served on 4173, `contract` → all pass exit 0, `smoke --offline` → all pass exit 0 (gate record + re-pick-after-reload present), and `components` + `widgets` pass where time allows; `git diff --check` clean.
- [ ] STOP conditions honored (plan): headless Chrome must pass the gate (`showDirectoryPicker` + `crypto.subtle` present on localhost) and OPFS (`navigator.storage.getDirectory()`) must be available in the headless profile; if the picker stub does not take effect, the fault is app code capturing `showDirectoryPicker` at module scope — fix the app code (ticket 02's binding constraint), not the stub.

## Blocked by

Ticket 02 (boot swap) — the smoke suite drives the new Vault gate and picker
stub. Owns all driver-file changes; file-disjoint with tickets 01 and 04.
