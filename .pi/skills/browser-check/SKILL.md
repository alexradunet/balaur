---
name: browser-check
description: Verify the Balaur canvas app in headless Chrome over CDP — the default way to check this application after any change. Runs the AGENTS.md §13 baseline smoke suite (boot, render, file index, selection, card creation, persistence, offline), plus ad-hoc runtime probes and screenshots. Use whenever a change to app.js, styles/, storage/, sw.js, or index.html needs browser-level verification.
---

# Browser check (headless Chrome + CDP)

This is the **default browser verification path** for this repository (AGENTS.md §13).
`node --check` alone never proves persistence, IndexedDB, or canvas behavior — run this skill
whenever JavaScript, CSS, storage, or shell assets change.

## Prerequisites

- `google-chrome` (or `chromium`) on PATH — no npm install, no WebDriver.
- The app served over HTTP (never `file://`):

```bash
python3 -m http.server 4173   # from the repository root
```

The driver script is `scripts/browser-check.mjs` (relative to this skill directory),
run from the repository root. It uses Node's built-in `WebSocket`/`fetch`; nothing to install.

## Commands

```bash
# Full baseline smoke suite (fresh profile). Exit code 0 = all pass.
node .pi/skills/browser-check/scripts/browser-check.mjs smoke

# Smoke with extras:
#   --offline          also test Service Worker offline reload
#   --profile <dir>    reuse a profile across runs (persistence/migration testing)
#   --width/--height   viewport size (e.g. --width 380 for narrow-shell checks)
#   --screenshot <dir> save selected-card.png for visual review
node .pi/skills/browser-check/scripts/browser-check.mjs smoke --offline --screenshot /tmp/shots

# One-off runtime probe (prints JSON). The page is loaded and app boot is awaited.
node .pi/skills/browser-check/scripts/browser-check.mjs eval "window.orbitCanvas.getSummary()"
node .pi/skills/browser-check/scripts/browser-check.mjs eval "await window.orbitVaultReady && window.orbitCanvas.getSummary()"
node .pi/skills/browser-check/scripts/browser-check.mjs eval "document.title" --wait "window.orbitCanvas"

# Screenshot (full page, or one element with --selector).
node .pi/skills/browser-check/scripts/browser-check.mjs shot /tmp/canvas.png
node .pi/skills/browser-check/scripts/browser-check.mjs shot /tmp/card.png --selector ".canvas-node.selected"

# DirectoryVault adapter contract suite over an OPFS handle (no app boot).
# Dynamically imports /storage/directory-vault.js, creates a fresh OPFS
# subdirectory per run, and asserts the full VaultStore surface. Exit 0 = all pass.
node .pi/skills/browser-check/scripts/browser-check.mjs contract
```

## What the smoke suite checks

0. The Vault gate is passed: the picker button is enabled with no gate error
   message, the picker stub is installed, and a real CDP click opens a vault
   (see "Vault gate and the picker stub" below).
1. Boot with no uncaught console errors and no failed asset requests.
2. Every document node renders as a card (DOM count == document count).
3. Sidebar reports `Files · N indexed` (canonical file index came up over the vault; no SQLite in canonical v1).
4. Clicking a card selects it, opens the inspector, and shows the selection
   frame (visible, solid border).
5. Double-clicking **inside** a card creates nothing (regression guard).
6. The note tool clicking **on** a card creates nothing.
7. Double-clicking empty background still creates a note.
8. The live document remains valid JSON Canvas 1.0.
9. Controlled reload (same profile) preserves title and node count — the suite
   re-stubs the picker and re-picks the same OPFS subdirectory after the reload.
10. With `--offline`: offline reload re-stubs the picker and renders the shell
    from the SW cache (`orbit-shell-v13`).

## Recipes

**Persistence / migration testing** — run twice against the same profile so the
second run exercises an existing (pre-change) profile:

```bash
P=$(mktemp -d)/profile
node .pi/skills/browser-check/scripts/browser-check.mjs smoke --profile "$P"
# ...change code...
node .pi/skills/browser-check/scripts/browser-check.mjs smoke --profile "$P"
```

**Narrow viewport** — pair with a manual look at `styles/responsive.css`:

```bash
node .pi/skills/browser-check/scripts/browser-check.mjs smoke --width 380 --height 800 --screenshot /tmp/narrow
```

**Deep probes** — useful runtime surfaces: `window.orbitCanvas.getDocument()`,
`.getWorkspace()`, `.getSummary()`, `await window.orbitVaultReady`,
`window.orbitVaultStore`.

## Vault gate and the picker stub (headless)

The app boots behind a full-screen Vault gate and exposes `window.orbitCanvas`
only after a folder is picked, so every subcommand that waits on `orbitCanvas`
(`smoke`, `components`, `widgets`' failure sessions, `shot`) first runs
`bootPastLanding(session)`. It:

1. waits for `#openVaultFolder` to be present, enabled, **and** for
   `window.orbitVaultReady` to be defined (the app wires the pick handler right
   before assigning that promise; waiting on it avoids clicking before the
   handler exists — a race the `Fetch`-intercepted failure session reliably hits);
2. installs `window.showDirectoryPicker = async () => (await navigator.storage.getDirectory()).getDirectoryHandle("vault-smoke", { create: true })` — an OPFS handle with the same `FileSystemDirectoryHandle` interface, no picker, no gesture;
3. performs a **real CDP click** on the button center (user-gesture semantics, not `el.click()`); the app references the global `showDirectoryPicker` at call time, so the stub is honored;
4. waits for `window.orbitCanvas` plus at least one rendered card.

The app persists no handle, so the gate shows on every load and the folder is
re-picked after each `reload()`: `bootPastLanding` is called after every
`navigate()` and `reload()`. Same profile → same OPFS origin → the same
`vault-smoke` subdirectory is re-picked, which is what makes the persistence
assertions meaningful; `--profile` therefore still drives persistence testing.
The `eval` subcommand stays raw (no landing handling) — pass `--wait` yourself
if you need the app booted, and stub the picker manually when probing a booted
app. Offline reload works because `openVault` reads only the OPFS-backed vault
and the bundled widget is already cached after the first open.

The real-picker UX, external-change reconciliation (Reload vault), and the
incompatibility gate on Firefox/Safari are **manual smoke** (CDP cannot drive a
native picker or a non-Chromium engine); they are browser-pending per AGENTS.md
§13 and must be labeled, not claimed.

## `contract` subcommand (adapter suite via OPFS)

`contract` verifies `DirectoryVault` against the `VaultStore` contract without
booting the app: it navigates, then runs one async eval that dynamically imports
`/storage/directory-vault.js` from the served origin, creates a fresh OPFS
subdirectory per run (`contract-main-<ts>`, etc.), and asserts the full surface —
create/read round-trip; `stat` meta shape; `exists` true/false; `list` ordering
and prefix filter; the three `expectedHash` outcomes (null create-conflict, hash
mismatch with `details.expected`/`details.actual`, correct hash); `NOT_FOUND` on
read; case-fold collision on create; `move` success plus destination-exists
conflict; `remove` success plus remove precondition; `snapshot`/`restore`
round-trip into a second fresh subdirectory; `changesSince` journal ordering
(strictly increasing revisions; create/modify/move/remove present); and
`TypeMismatchError` → `PATH_COMPONENT`. It returns `{ ok, failures, checks }`;
the driver prints PASS/FAIL per check and exits nonzero on any failure. Use it
whenever `storage/directory-vault.js` or the contract semantics change.

## Caveats

- Headless Chrome suppresses `click`/`dblclick` when the pointerdown target is
  removed mid-click (this app re-renders on selection); other browsers retarget
  those events instead. Do not conclude "event never fires" from headless runs
  alone when reasoning about cross-browser behavior — guard with hit tests
  (`document.elementFromPoint`) rather than relying on `event.target`.
- Service Worker tests need a reused profile (`--profile`) plus `--offline`;
  first install happens on the first online load.
- Destructive paths (reset, whole-space import) should only be exercised in a
  disposable profile — the default temp profile is disposable by design.
- Screenshots are PNG dumps: review them for visual changes (selection frame,
  themes, layout), since the smoke suite checks structure, not pixels.
- If a flow cannot be expressed through CDP probes, fall back to manual testing
  in a real browser; this skill covers the baseline, not every interaction.
