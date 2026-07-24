---
phase: ticket
status: ready
project: 001-directory-vault
ticket: 04
blocked-by: []
worker: ""
branch: ""
shared-blast-radius: false
---

# Ticket 04: Documentation pass + commit pre-staged ADR-0004 and CONTEXT.md

## What to build

The documentation matches the shipped DirectoryVault behavior (AGENTS.md §14:
docs change in the same change as behavior). The pre-staged ADR-0004 and
`CONTEXT.md` are verified against the spec and committed unchanged; the
architecture, life-data, offline, README, and AGENTS docs are reworded from
IndexedDB to the folder-backed DirectoryVault, the Vault gate boot, and the
`orbit-shell-v13` cache.

Source of truth: `plan.md` §5 (per-file edit list) and the spec's Documentation
section. Use `CONTEXT.md` vocabulary (Vault, DirectoryVault, Vault gate, Adopt).

## Acceptance criteria

- [ ] `docs/adr/0004-directory-vault-storage.md` (pre-staged) verified against the spec's ADR description (context, decision, consequences, verification boundary; states ADR-0001 file-canonical ownership is unchanged) and committed unchanged.
- [ ] `CONTEXT.md` (pre-staged: Vault reword + DirectoryVault / Vault gate / Adopt entries) verified and committed unchanged; not edited further.
- [ ] `docs/architecture.md`: adapter trio → `DirectoryVault (browser) / FsVault (Node) / MemoryVault (tests)`; boot list renumbered to (1) incompatibility gate, (2) Vault gate → `showDirectoryPicker({ mode: "readwrite" })`, (3) DirectoryVault opens + WorkspaceStore loads/migrates (create/Adopt/open), (4) index rebuild, (5) render + `window.orbitCanvas`, (6) progressive SW registration; localStorage first-run migration step deleted; "no handle persisted" stated; verification sentence → headless OPFS contract suite + picker-stub smoke + manual Chromium smoke; VaultStore paragraph → DirectoryVault browser default, folder-backed, no content cache, non-atomic `createWritable` guarded by `expectedHash`; `orbit-shell-v12` → `v13`; "does not cache IndexedDB records" → "does not cache vault files (the user-picked folder owns them)"; future-packaging paragraph → browser directory access shipped, a future desktop shell adds its own fs-kind adapter under the same contract.
- [ ] `docs/life-data.md`: adapter list → DirectoryVault replaces IndexedDbVault (folder-backed, disk mtime, non-atomic write + expectedHash guard); drop the "Upgrading a legacy localStorage profile is a clean break" sentence; vault-writes paragraph → DirectoryVault keeps nothing in IndexedDB, the picked folder's plain files are the vault, the SW never caches them.
- [ ] `docs/offline.md`: every `orbit-shell-v12` mention (6 total) → `v13`; runtime-pieces bullet → user files live in the user-picked folder, never in the SW cache; precache description gains `directory-vault.js`/`memory-vault.js`, loses `indexeddb-vault.js`; localStorage row → limited to theme and AI settings; verification paragraph → vault-first folder reconstruction via the picker stub.
- [ ] `README.md`: feature bullet + storage section → folder vault, Chromium requirement, re-pick per launch, Reload vault / Open another vault, localStorage limited to theme/AI settings; drop the localStorage-clean-break sentence; browser-requirements line → File System Access API (`showDirectoryPicker`), drop IndexedDB/OPFS; new short "Migrating from the IndexedDB version" subsection (export whole space from the deployed version BEFORE the switch; pick an EMPTY folder; Import the `.orbit.json`).
- [ ] `AGENTS.md`: §3 repo map `indexeddb-vault.js` line → `directory-vault.js` (+ note `memory-vault.js` doubles as the browser staging adapter) and the paragraph after the map; §5 boot model renumbered per the architecture.md list, localStorage migration step deleted, no-handle-persisted stated; §5.1 cache name corrected to `orbit-shell-v13` and "IndexedDB owns user files" → "the user-picked folder owns user files"; §7 adapter list → DirectoryVault is the browser adapter, IndexedDbVault gone; §13 browser-pending list reworded to the spec's eight items, Node count unchanged at 172, note the smoke suite now asserts the gate, runs the OPFS contract suite, and stubs the picker. AGENTS.md §13 wording is written to the settled spec (not to ticket 03's code), which is why this ticket is parallel-safe with ticket 01.
- [ ] Verify: `git diff --check` clean; `grep -rn "IndexedDbVault\|indexeddb-vault" docs/ README.md AGENTS.md | grep -v "adr/0004"` returns no output (only the ADR-0004 "supersedes" line and explicitly historical wording remain); all `orbit-shell-v` mentions in docs read `v13`.
- [ ] Commit the pre-staged ADR + CONTEXT.md unchanged; stage only this ticket's doc paths (`git add <paths>`, never `git add -A`).

## Blocked by

None — can start immediately. File-disjoint with ticket 01 (and with ticket
03); forms the initial parallel frontier with ticket 01. The plan table's soft
"depends on 3 for AGENTS.md §13 wording" is non-blocking: that wording is
written to the settled spec, and the owner confirmed `blocked-by: []`.
