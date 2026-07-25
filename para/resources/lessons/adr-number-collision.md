# Lesson: ADR number collisions across parallel work

From project `001-directory-vault` (Balaur, 2026-07).

## What happened

The project chose "ADR-0004" at grill time, when `main` had ADRs 0001–0003.
While the project ran, another project merged `0004-file-unified-node-model.md`
into `main`. The feature branch carried a second ADR-0004 until review caught
it; the fix was a rename to `0005-directory-vault-storage.md` plus a provenance
note, and a grep to separate references to the two ADRs (some docs legitimately
cited the *other* 0004).

## The rule

ADR numbers are allocated at **commit time against current `main`**, not at
decision time. When a project's ADR is about to land (domain-model or docs
ticket), re-check `ls docs/adr/` on the merged base and renumber if the slot
was taken. Planning artifacts (grill/spec/plan) may keep the old number as
project history; the ADR file itself carries the provenance note
("renumbered from N because …").
