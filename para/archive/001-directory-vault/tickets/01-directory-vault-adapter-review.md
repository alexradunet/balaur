---
phase: review
status: done
ticket: 01
date: 2026-07-25
verdict: approved
---

# Standards review: ticket 01 — directory-vault-adapter

## Verdict: APPROVED

Diff reviewed: `git diff ff4e68b..56be884` — exactly one commit (56be884), one
new file `storage/directory-vault.js` (291 lines). Verification re-run by this
reviewer: `node --check storage/directory-vault.js` → exit 0; the explicit
suite (`storage/phase1..10`, `phase-query`, `note-repository`) → 197/197 pass;
`git diff --check ff4e68b..56be884` → clean; `grep queryPermission|
requestPermission` → no matches. DirectoryVault is browser-only; this is a
code-reading + contract comparison against `MemoryVault`/`FsVault`/`vault-store`
and plan §1 / spec, not a runtime exercise (the OPFS contract suite lands in
ticket 03).

## Findings

- [judgement] `storage/directory-vault.js:233` and `:240` — Divergent Change /
  plan deviation: `move` journals and returns `existing.hash` / `existing.size`
  (the values from the `_stat(f)` at line 214), but plan §1 step 5 specifies
  `_bump(t, "move", hashOfReadText, f)` and the spec says "return meta for `to`
  (content hash preserved)". `existing` was read from disk at line 214; the
  bytes actually written to `to` come from the separate `this.read(f)` at line
  223. Under an external edit between those two reads, the journaled and
  returned `hash`/`size` describe the *old* source content, not the bytes now at
  `to`. The same module's `write` does it the disk-aware way (`const hash =
  await contentHash(text)` at `:192`, `size: byteLength(text)` at `:194`), so
  `move` is inconsistent with `write`. Impact is narrow: `LifeIndexer.
  reconcileWarm` (`storage/life-indexer.js:284`) consumes only `change.path`,
  `change.operation`, `change.oldPath` from the journal — never `change.hash`
  (verified by grep) — so warm reconciliation is unaffected, and a caller using
  the stale returned `hash` as a subsequent `expectedHash` would fail safe
  (ConflictError) rather than corrupt. Recommended fix (2 lines, low-risk):
  after `const text = await this.read(f);` at `:223`, compute `const hash =
  await contentHash(text);` and `const size = byteLength(text);`, then
  `_bump(t, "move", hash, f)` at `:233` and return `size, hash` at `:240`. This
  matches the plan's `hashOfReadText`, the spec's "content hash preserved", and
  `write`'s own pattern. Listed as judgement, not hard, because the plan also
  says "mirror `MemoryVault.move` ordering" and `MemoryVault` (in-memory, no
  external-edit window) uses `existing.hash` — so the choice is defensible, and
  nothing breaks without the fix.

- [judgement] `storage/directory-vault.js:186-188` and `:227-229` — Duplicated
  Code (smell #2): the `createWritable()` → `write(text)` → `close()` triple is
  duplicated verbatim in `write` and `move`. The plan forbids `move` delegating
  to `this.write` (it would journal a spurious "create" and bump twice), but a
  small private `async _streamWrite(handle, text)` containing just those three
  lines would remove the duplication without introducing a second bump. Low
  priority; the duplication is small and the plan's inline intent is clear.

- [judgement] `storage/directory-vault.js:25` — error-message convention
  (playbook "Errors: include the offending value and expected shape"): the
  constructor guard throws `StorageError("DirectoryVault requires a
  FileSystemDirectoryHandle", ...)` without naming what was passed. A foreign
  `handle.kind` (e.g. `"file"`) or `undefined` would aid debugging if the
  message read `...requires a FileSystemDirectoryHandle (got ${handle?.kind ??
  "missing"})`. Minor; the `code: "STORAGE_UNAVAILABLE"` is correct and stable.

## Summary

Three [judgement] findings, zero [hard]. The substantive one is the `move`
hash/size source (`:233`, `:240`) deviating from the plan's `hashOfReadText` and
the spec's "content hash preserved"; impact is narrow (race-only,
reconcileWarm unaffected, subsequent writes fail safe) and the fix is 2 lines.
All acceptance criteria met: full surface, each path through `assertSafePath`,
no content cache, exact `MemoryVault` precondition table, `FsVault`-style
case-fold-by-listing, complete DOMException → vault-error mapping with
`details: { name, message }`, synchronous `changesSince`, snapshot/restore
shape, no `queryPermission`/`requestPermission`. Static checks green; 197 tests
unchanged. Confidence high; the `move` recommendation is worth a follow-up
`paseo send` but does not block merge.
