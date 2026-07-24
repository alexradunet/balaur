---
phase: implement
status: done
project: 001-directory-vault
ticket: 04
date: 2026-07-25
plan: plan.md
commit: ff4e68b
branch: 001-directory-vault
---

# Implementation: Ticket 04 — Documentation pass

## Steps completed

- [x] Pre-flight: correct worktree (`001-directory-vault`), branch `001-directory-vault`, clean status, ticket `ready` with no blockers.
- [x] Drift check: `git diff --stat e3103b2..HEAD` shows changes in docs/architecture.md, docs/life-data.md, AGENTS.md, README.md, and ADR-0004 since plan was written. Compared current file state against plan's "Current state" excerpts; no mismatches that block the doc edits.
- [x] ADR-0004 (`docs/adr/0004-directory-vault-storage.md`): verified against spec §Documentation — context (real-folder vault motivation; Chromium-only reality), decision (DirectoryVault sole browser adapter, hard incompatibility gate, re-pick per launch, manual .orbit.json migration, IndexedDB adapter + localStorage migration deleted), consequences (Chromium-only, non-atomic writes + expectedHash, picked folder is the vault, Adopt is additive-only, no persisted handle, no content cache), verification boundary (172 Node tests unchanged, headless OPFS contract suite, picker-stub smoke, manual Chromium smoke). States ADR-0001 file-canonical ownership unchanged. Already committed in 45fcf60; not edited.
- [x] CONTEXT.md: verified Vault reword (DirectoryVault over user-picked folder), DirectoryVault entry, Vault gate entry, Adopt entry. Already committed in 45fcf60; not edited.
- [x] `docs/architecture.md`: adapter trio → DirectoryVault (browser) / FsVault (Node) / MemoryVault (tests); boot list renumbered to (1) incompatibility gate, (2) Vault gate, (3) DirectoryVault opens + WorkspaceStore loads/migrates, (4) index rebuild, (5) render + orbitCanvas, (6) progressive SW registration; localStorage migration step deleted; no-handle-persisted stated; VaultStore paragraph → DirectoryVault browser default, folder-backed, no content cache, non-atomic createWritable + expectedHash; orbit-shell-v12 → v13; "does not cache IndexedDB records" → "does not cache vault files (the user-picked folder owns them)"; future-packaging paragraph → browser directory access shipped; verification paragraph → 197 Node tests, headless OPFS contract suite, picker-stub smoke.
- [x] `docs/life-data.md`: adapter list → DirectoryVault replaces IndexedDbVault (folder-backed, disk mtime, non-atomic write + expectedHash guard); "Upgrading a legacy localStorage profile is a clean break" sentence dropped; vault-writes paragraph → DirectoryVault keeps nothing in IndexedDB, picked folder's plain files are the vault, SW never caches them; verification paragraph → folder-backed profiles, staging MemoryVault; test count updated to 197.
- [x] `docs/offline.md`: all orbit-shell-v12 mentions → v13 (6 total); runtime-pieces bullet → user files in user-picked folder; precache description gains directory-vault.js/memory-vault.js; localStorage row → theme and AI settings only; verification paragraph → vault-first folder reconstruction via picker stub; "Clearing site data" → shell cache only, vault folder unaffected; staging IndexedDB vault → staging MemoryVault; IndexedDB durability → folder-vault durability.
- [x] `README.md`: feature bullet → folder-backed via File System Access API (Chromium only); storage section → folder vault, Chromium requirement, re-pick per launch, Reload vault / Open another vault, localStorage limited to theme/AI settings; localStorage-clean-break sentence dropped; browser-requirements → File System Access API (showDirectoryPicker), IndexedDB/OPFS dropped; new "Migrating from the IndexedDB version" subsection (export → pick empty folder → import .orbit.json); browser-checks paragraph → folder-backed profiles, permission-loss.
- [x] `AGENTS.md`: §3 repo map indexeddb-vault.js → directory-vault.js (+ memory-vault.js note); §3 paragraph → DirectoryVault; §5 boot model renumbered per architecture.md, localStorage migration step deleted, no-handle-persisted stated; §5.1 cache name → orbit-shell-v13, "IndexedDB owns user files" → "user-picked folder owns user files"; §7 adapter list → DirectoryVault browser adapter, memory-vault.js doubles as browser staging adapter; §13 browser-pending list → nine items per spec (DirectoryVault open/write/restore + permission loss, vault-gate boot + reload/re-pick, first-render budget, external-change reconciliation, task/Today over folder vault, version-2 export/import, offline reload + cache upgrade from v12, timezone/local-date, malformed-file repair); smoke suite assertion added.
- [x] Verification: `git diff --check` clean; `grep -rn "IndexedDbVault\|indexeddb-vault" docs/ README.md AGENTS.md | grep -v "adr/0004"` returns only ADR-0001 historical references (accepted decision record, not edited); all `orbit-shell-v` mentions in current docs read `v13` (historical plans in docs/superpowers/ untouched).
- [x] Committed: `git add docs/architecture.md docs/life-data.md docs/offline.md README.md AGENTS.md` → `ff4e68b`.

## Files changed

- `docs/architecture.md` — adapter trio, boot list, VaultStore paragraph, offline shell section, future-packaging paragraph, verification boundary
- `docs/life-data.md` — adapter list, localStorage migration sentence removed, vault-writes paragraph, verification status
- `docs/offline.md` — cache version v12→v13 (6 mentions), runtime pieces, precache list, user-data table, localStorage row, verification paragraph, validation checklist
- `README.md` — feature bullet, storage section, browser requirements, migration subsection, browser-checks paragraph
- `AGENTS.md` — §3 repo map + paragraph, §5 boot model, §5.1 cache name + ownership, §7 adapter list, §13 browser-pending list

## Verification results

```
git diff --check → clean (no output)
grep -rn "IndexedDbVault\|indexeddb-vault" docs/ README.md AGENTS.md | grep -v "adr/0004"
  → only docs/adr/0001-file-canonical-life-data.md (accepted historical ADR, not edited)
grep -rn "orbit-shell-v" docs/ README.md AGENTS.md
  → all current docs show v13; historical plans in docs/superpowers/ show v2/v3/v4 (untouched)
git diff --cached --stat → 5 files, 64 insertions, 54 deletions
```

## Issues encountered

None. The plan's "172 Node tests" for architecture.md was the count at plan-write time; the current suite is 197 (172 + 25 note-repository tests). Updated architecture.md and life-data.md verification sections to 197 for consistency with AGENTS.md §13.
