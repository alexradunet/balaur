---
project: 001-directory-vault
closed: 2026-07-25
status: merged
---

# Closure: DirectoryVault — the File System Access API as the sole browser vault

## What was built

Balaur's vault is plain files in a user-picked folder: `DirectoryVault`
implements the `VaultStore` contract over `showDirectoryPicker({mode:"readwrite"})`,
with a Vault-gate boot, create/Adopt/open folder states, Reload and Open-another
actions, MemoryVault-staged import, a hard incompatibility gate for non-Chromium
browsers, and a reworked browser-check driver (OPFS contract suite 16/16,
picker-stub smoke 14/14). `IndexedDbVault` and the localStorage first-run
migration were deleted. Shipped to `main` (fast-forward `fce0f30`) and deployed
2026-07-25; ADR-0005 records the decision.

## What was distilled

- `para/resources/file-system-access-vault-pattern.md` — the adapter shape: constructor-injected handle, OPFS as headless test double, call-time-global picker stub, disk semantics (no cache, non-atomic writes, move hashes read content).
- `para/resources/desktop-shell-evaluation.md` — Tauri/Electron/Neutralino/Bun ranking and why no shell shipped, for when the desktop-app question returns.
- `para/resources/lessons/adr-number-collision.md` — allocate ADR numbers at commit time against current main.
- `para/resources/lessons/paseo-spawn-quirks.md` — Paseo output parsing, early-return recovery, and closed-agent resume.

## What was left behind

- The deferred hardening from the feature review (flip read-only in place on `STORAGE_UNAVAILABLE` save failures) — picked up immediately after closure as a trivial fix, not carried as project debt.
- Planning artifacts' "ADR-0004" references — kept as project history; the ADR itself carries the renumber provenance.
- Mobile app and desktop shell — separate future projects by owner decision (grill 2026-07-24); the shell evaluation is distilled above.
- Typed toolchain (JSDoc + tsc) — tracked separately as `para/projects/004-jsdoc-types`.
