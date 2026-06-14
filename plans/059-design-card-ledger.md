# Plan 059: DESIGN.md card ledger lists all 14 shipped card types (not 12)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 7b16063..HEAD -- DESIGN.md internal/cards/cards.go`
> If either file changed since this plan was written, compare the "Current
> state" excerpts against the live code before proceeding; on a mismatch,
> treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: docs
- **Planned at**: commit `7b16063`, 2026-06-14

## Why this matters

`DESIGN.md` calls its component inventory the **honesty ledger** — the canonical
"what is actually shipped vs. roadmap" record for this project. It currently
claims "**12** parameterized, server-rendered card resources" and enumerates 12
types, but the card registry in `internal/cards/cards.go` defines **14**. The
two undocumented cards (`lifelog`, `settings`) are real, completed features
(they shipped with plans 055 and 056). A drifted honesty ledger is worse than
no ledger: a reader can't trust it. This is a one-edit docs-truth fix that keeps
the ledger honest, in the spirit of the prior docs-sync plans (007, 018, 057).

## Current state

The ledger lives in one sentence in `DESIGN.md`. As of `7b16063`, lines 157–160 read:

```
dialogue choices — `offer_choices` renders 2–5 numbered reply buttons in chat
(keyboard 1–9); a choice posts as the owner's turn · typed card registry — 12
parameterized, server-rendered card resources under `/ui/cards/{type}` (the
composition unit for boards and on-the-spot UI): today, quests, calendar,
timeline, journal, day, measure, lines, memory, skills, heads, habits ·
```

The documented list is exactly 12: `today, quests, calendar, timeline, journal,
day, measure, lines, memory, skills, heads, habits`.

The authoritative registry is `internal/cards/cards.go`, `func init()` at
lines 45–194, `registry = []Spec{...}`. It defines **14** specs, in this order
(the `Type:` field of each entry):

1. `today` (line 48)
2. `quests` (line 56)
3. `calendar` (line 68)
4. `timeline` (line 78)
5. `journal` (line 88)
6. `day` (line 98)
7. `measure` (line 108)
8. `lines` (line 119)
9. `memory` (line 130)
10. `skills` (line 142)
11. `heads` (line 153)
12. `habits` (line 163)
13. `lifelog` (line 171)  ← **missing from DESIGN.md**
14. `settings` (line 179) ← **missing from DESIGN.md**

So the ledger is missing `lifelog` and `settings`, and the count "12" should be "14".

This repo treats docs as a maintained truth artifact — see prior plans 007
("Docs truth sync"), 018, and 057 ("holistic card-first IA docs"). Match that:
fix the fact, do not editorialize.

## Commands you will need

| Purpose   | Command                          | Expected on success |
|-----------|----------------------------------|---------------------|
| Drift     | `git diff --stat 7b16063..HEAD -- DESIGN.md internal/cards/cards.go` | empty (no drift) |
| Count registry | `grep -cE '^\t\t\tType:' internal/cards/cards.go` | `14` |
| Verify edit | `grep -n 'card resources' DESIGN.md` | shows "14 ... card resources" |
| Build     | `CGO_ENABLED=0 go build -o /tmp/balaur-059 .` | exit 0 |
| Tests     | `go test ./...`                  | all pass |

(No code changes — build/tests are a sanity gate only; this plan edits one Markdown file.)

## Scope

**In scope** (the only file you should modify):
- `DESIGN.md` (the single ledger sentence at lines 157–160)

**Out of scope** (do NOT touch):
- `internal/cards/cards.go` — it is the source of truth and is already correct; do not change the registry.
- `README.md`, `AGENTS.md`, `docs/` — any other doc. This plan fixes only the DESIGN.md ledger line. (If you notice the same "12" claim elsewhere, note it in your report; do not edit it here.)
- Card order or labels — keep the documented order matching the registry order.

## Git workflow

- Branch: `improve/059-design-card-ledger`
- One commit; message style is conventional commits (see `git log --oneline`):
  e.g. `docs(design): card ledger lists all 14 card types (was 12)`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Confirm the registry really has 14 types

Run:

```
grep -cE '^\t\t\tType:' internal/cards/cards.go
```

**Verify**: output is exactly `14`. If it is not 14, STOP and report (the
registry changed since this plan was written; the target number is wrong).

### Step 2: Edit the ledger sentence in DESIGN.md

Make exactly two changes inside the sentence at lines 157–160:

1. Change `typed card registry — 12` to `typed card registry — 14`.
2. In the enumerated list, change the ending
   `... skills, heads, habits ·`
   to
   `... skills, heads, habits, lifelog, settings ·`
   (append `, lifelog, settings` before the ` ·` separator, preserving registry order).

After the edit the sentence must read:

```
... typed card registry — 14
parameterized, server-rendered card resources under `/ui/cards/{type}` (the
composition unit for boards and on-the-spot UI): today, quests, calendar,
timeline, journal, day, measure, lines, memory, skills, heads, habits, lifelog,
settings ·
```

**Verify**:
```
grep -n 'card resources' DESIGN.md        # the line now says "14 ... card resources"
grep -c 'lifelog, settings' DESIGN.md     # → at least 1
grep -c 'registry — 12' DESIGN.md         # → 0 (the old count is gone)
```

### Step 3: Sanity-build and test

The code is untouched, but confirm the tree is still green:

```
CGO_ENABLED=0 go build -o /tmp/balaur-059 . && go test ./...
```

**Verify**: build exits 0; all tests pass.

## Test plan

No new tests — this is a documentation correction with no code behavior change.
The build + `go test ./...` run is a tree-health sanity gate, not coverage for
this change. (There is no automated DESIGN.md ↔ registry consistency test today;
adding one is explicitly out of scope for this plan — note it as a possible
follow-up in your report if you wish.)

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -c 'registry — 14' DESIGN.md` → `1`
- [ ] `grep -c 'registry — 12' DESIGN.md` → `0`
- [ ] `grep -c 'lifelog, settings' DESIGN.md` → ≥ `1`
- [ ] `grep -cE '^\t\t\tType:' internal/cards/cards.go` → `14` (registry unchanged)
- [ ] `git status --porcelain` shows only `DESIGN.md` modified, nothing else
- [ ] `CGO_ENABLED=0 go build -o /tmp/balaur-059 .` exits 0 and `go test ./...` passes
- [ ] `plans/readme.md` status row for 059 updated (unless your reviewer maintains it)

## STOP conditions

Stop and report back (do not improvise) if:

- The registry count in Step 1 is not 14 (the registry drifted; the documented number this plan specifies would be wrong).
- The DESIGN.md sentence at lines 157–160 does not match the "Current state" excerpt (DESIGN.md drifted).
- You find the "12 card" claim is repeated in another file — report it; do not fix it here (out of scope).

## Maintenance notes

- Whenever a card `Spec` is added to or removed from `internal/cards/cards.go`,
  this DESIGN.md ledger sentence (count + list) must be updated to match. A
  reviewer adding a card type should grep `DESIGN.md` for `card resources`.
- Deferred (not in this plan): an automated test asserting `len(cards.All())`
  equals the number documented in DESIGN.md, which would prevent this drift
  class permanently. Worth considering in a future tests-focused cycle.
