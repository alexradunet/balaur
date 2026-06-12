# Plan 008: Make AGENTS.md's "One PocketBase seam" rule match reality

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on.
> If anything in "STOP conditions" occurs, stop and report. When done,
> update this plan's row in `plans/README.md`.
>
> **Drift check (run first)**: `git diff --stat c4fce47..HEAD -- AGENTS.md internal/store/`
> On drift, re-verify the excerpt below.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW (doc change + grep-verifiable claims)
- **Depends on**: none (coordinate with plan 007 to avoid AGENTS.md/README merge conflicts — this plan owns AGENTS.md)
- **Category**: tech-debt (documentation-as-law repair)
- **Planned at**: commit `c4fce47`, 2026-06-12
- **Issue**: https://github.com/alexradunet/balaur/issues/23

## Why this matters

AGENTS.md is injected into every coding agent's context as engineering law.
Its architecture section claims:

> **One PocketBase seam.** Everything that touches PocketBase internals
> (collections, records, tokens, rules) goes through `internal/store`. …
> when it breaks, the fix should be one package wide.

Measured at `c4fce47`, this is false: 24+ production (non-test) files across
`internal/{cli,conversation,ext,heads,knowledge,life,recap,self,tasks,tools,turn,web}`
and `migrations/` call `app.Save` / `app.FindRecord*` / `app.FindRecordsByFilter`
directly (examples an executor can spot-check: `internal/tasks/nudge.go:33`,
`internal/tasks/streak.go:46`, `internal/conversation/conversation.go:29`,
`internal/knowledge/knowledge.go` throughout). `internal/store` (522 lines:
`audit.go`, `llm_settings.go`, `owner_settings.go`, `time.go`) is a
cross-cutting-helpers package, not a data layer.

Two failure modes of leaving the false claim: (a) agents place new
PocketBase logic in `internal/store` while every existing domain package
does the opposite, producing a half-and-half architecture; (b) anyone
scoping a future PocketBase upgrade (pre-1.0, with v0.23 having rewritten
the whole API) budgets "one package" when the true blast radius is the
whole repo. The cheapest correct fix is to rewrite the rule to describe the
REAL architecture (domain packages own their own records; store owns
cross-cutting concerns), preserving the upgrade warning with honest scope.

A full refactor to actually centralize PB access was considered and
rejected: ~L effort, repo-wide churn, against KISS/YAGNI while PocketBase
v0.39.x is stable. Record that as the explicit tradeoff.

## Current state

- `AGENTS.md:84-87` (the bullet to replace; quote verified at c4fce47):

```
- **One PocketBase seam.** Everything that touches PocketBase internals
  (collections, records, tokens, rules) goes through `internal/store`.
  PocketBase is pre-1.0 with breaking-change precedent (v0.23 rewrote the
  whole API surface); when it breaks, the fix should be one package wide.
```

- `internal/store/` contents at c4fce47: `audit.go` (write-side audit
  helper), `llm_settings.go` (LLM provider/model config), `owner_settings.go`
  (key-value owner prefs + avatar maps), `time.go` (`PBTime` filter-format
  helper), plus `llm_settings_test.go`.
- Convention for this file (its own header): "keep it lean and high-signal —
  add a rule only when it changes a real decision."

## Commands you will need

| Purpose | Command | Expected |
|---|---|---|
| Count the reality | `grep -rln "app.FindRecord\|app.Save(\|app.FindRecordsByFilter" --include='*.go' internal/ migrations/ \| grep -v _test \| grep -v "^internal/store/" \| wc -l` | ≥ 20 (the number goes into the new text) |
| Gates | `go test ./...` | ok (docs-only) |

## Scope

**In scope**:
- `AGENTS.md` (the one bullet, plus — if and only if it repeats the claim —
  the corresponding line in `internal/self/knowledge.md`; check with
  `grep -n "internal/store\|one seam" internal/self/knowledge.md`)

**Out of scope** (do NOT touch):
- Any Go file. No store helpers, no refactors — the doc tells the truth;
  plan 015 adds the two store read-helpers ITS cleanup needs.
- README.md / DESIGN.md (plan 007 owns those).

## Git workflow

- Branch: `advisor/008-agents-md-seam`
- Commit style: `docs(agents): replace the false "one PocketBase seam" rule with the real boundary`. No push/PR unless instructed.

## Steps

### Step 1: Replace the bullet

Replace the AGENTS.md bullet quoted above with (adjust the file count to
your measured number):

```
- **PocketBase access, honestly scoped.** Domain packages (`conversation`,
  `tasks`, `knowledge`, `heads`, `life`, `recap`, `ext`, `self`) own their
  own PocketBase reads/writes — records are the domain model, not a layer
  to hide. `internal/store` is the seam for CROSS-CUTTING concerns only:
  audit, owner settings, LLM config, time formatting. Two consequences:
  (1) new domain logic talks to PocketBase directly in its own package —
  do not route it through store; (2) PocketBase is pre-1.0 with
  breaking-change precedent (v0.23 rewrote the whole API surface), and
  >20 files touch it directly — an upgrade is a REPO-WIDE change; budget
  and test it as such, never as "one package wide".
```

**Verify**: `grep -n "one package wide" AGENTS.md` → no matches;
`grep -n "REPO-WIDE" AGENTS.md` → 1 match.

### Step 2: Sweep the echo in knowledge.md

If `internal/self/knowledge.md` repeats the one-seam claim (check command in
Scope), align its sentence with the new rule in one line. If it does not,
skip.

**Verify**: `grep -rn "goes through .internal/store" AGENTS.md internal/self/knowledge.md` → no matches.

### Step 3: Gates

**Verify**: `gofmt -l .` empty; `go test ./...` ok.

## Test plan

Docs-only; greps are the assertions. (`internal/self` tests re-validate the
embedded knowledge.md if Step 2 touched it.)

## Done criteria

- [ ] Step 1 and Step 2 greps hold
- [ ] `go test ./...` exit 0
- [ ] Diff touches at most `AGENTS.md` + `internal/self/knowledge.md` (plus `plans/README.md`)
- [ ] `plans/README.md` status row updated

## STOP conditions

- The AGENTS.md bullet text differs from the excerpt (already edited since
  c4fce47) — reconcile intent first; if someone strengthened the claim
  instead, report the conflict rather than overwriting.

## Maintenance notes

- If the team ever DOES want a real PB seam (e.g. before a risky PocketBase
  major upgrade), that is a separate migration project: introduce per-domain
  repository functions package-by-package, newest packages first. Do not let
  this doc change drift back into aspiration.
- Reviewer: the new text changes a rule agents follow mechanically — read it
  once as "would an agent now put new PB calls in the right place?"
