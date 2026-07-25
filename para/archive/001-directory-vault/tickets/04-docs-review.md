---
phase: review
status: done
ticket: 04
date: 2026-07-25
verdict: approved
---

# Standards review: ticket 04 — documentation pass

## Verdict: APPROVED

Diffed exactly the two assigned commits: `git diff 4bf9329..ff4e68b`
(architecture.md, life-data.md, offline.md, README.md, AGENTS.md) and
`git show 546658d` (ADR 0004→0005 rename + provenance note). The working tree
holds sibling tickets' work and was not reviewed.

## Verification run

- `git diff --check 4bf9329..ff4e68b` — clean.
- `grep -rn "IndexedDbVault\|indexeddb-vault" docs/ README.md AGENTS.md` — only
  `docs/adr/0001-...` (accepted historical record, untouched) and
  `docs/adr/0005-...` (the decision record that supersedes IndexedDbVault, in
  its Supersedes/Context/Verification fields). No stale prose references.
- `grep -rn "balaur-shell-v" docs/ README.md AGENTS.md` — all current docs read
  `v13` (offline.md ×6, architecture.md ×1, AGENTS.md ×1); only
  `docs/superpowers/plans/` history shows v2/v3/v4, untouched.
- ADR-0004 references to `0004-file-unified-node-model.md` (notes,
  FileNoteRepository, AI operators in generative-canvas.md, architecture.md,
  life-data.md, AGENTS.md, CONTEXT.md) all remain `0004` and untouched.
  `docs/adr/0005-...` is the only file referencing `ADR-0005`.
- `node --test` on the explicit §13 suite: **197 pass / 0 fail** (172 + 25
  note-repository). The 197 in architecture.md:102, life-data.md:287, and
  AGENTS.md:295 is the real, verified count.
- No doc claims browser-verified status for DirectoryVault. AGENTS.md:83
  ("require browser verification"), :163 ("browser-pending"), :309 ("Do not
  claim these are browser-verified") all frame it as pending; the
  verify-language in architecture.md:102 / life-data.md:289 / offline.md:63,63
  describes the headless OPFS contract suite and picker-stub smoke (automated),
  not manual verification — the pre-existing convention preserved by this
  reword.
- Boot sequence, cache `v13`, "user-picked folder owns user files", README
  migration subsection (export → empty folder → import `.balaur.json`), and ADR
  content all match spec §Documentation and plan §5.

## Findings

- [judgement] `AGENTS.md:311` — content omission: §13 item 1 reads
  "`DirectoryVault` open/write/restore and permission-loss behavior" but drops
  "externally deleted directories" that both spec §Testing Decisions and plan §5
  list as part of item 1 ("DirectoryVault open/write/restore + permission loss +
  deleted directories"). The scenario (folder deleted/unlinked mid-session,
  user story 11) is implicitly browser-pending under "DirectoryVault...
  behavior" and is named in ADR-0005:33's manual-smoke list, so it is not lost
  from the docs — only the explicit §13 mention is gone. Optional one-line fix:
  append "and externally deleted directories" to item 1 for parity with the
  spec/plan.
- [judgement] `docs/adr/0005-directory-vault-storage.md:33` — count mismatch:
  the ADR's verification boundary says "172 tests pass (unchanged)" while the
  living docs (architecture.md:102, life-data.md:287, AGENTS.md:295) say 197.
  The ADR matches the spec's explicit "172" ADR requirement and is a dated,
  pre-staged decision record (committed 45fcf60, not edited by this ticket); the
  "(unchanged)" qualifier is correct (the directory-vault decision added no Node
  tests). The 197 in the living docs is the real, verified count. The gap is a
  timing artifact (ADR written before the note-repository tests were counted in
  AGENTS.md §13), not a defect — but a reader cross-referencing the ADR and the
  living docs sees two numbers.
- [judgement] `AGENTS.md:311-319` — item count: §13 lists 9 browser-pending
  items; spec §Testing Decisions and plan §5 specify 8. The difference is that
  "first-render budget" was split out of the bundled "vault-gate boot, reload +
  re-pick persistence, and first-render budget" item into its own item 3.
  Content coverage is identical; the split is arguably clearer. Non-blocking.

## Summary

3 judgement findings, 0 hard violations. The docs faithfully reword from
IndexedDB to the folder-backed DirectoryVault: adapter trio, gate→picker→
DirectoryVault→WorkspaceStore→rebuild→render→progressive-SW boot, `balaur-shell-v13`
cache, "user-picked folder owns user files", the README migration subsection, and
the ADR-0005 renumber with provenance all match the spec and plan. The 197 test
count is real and verified. The findings are minor wording/count drift against the
spec's literal text; none block merge. Confidence high.
