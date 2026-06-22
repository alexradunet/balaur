# Plan 155: Remove the redundant `skills.enabled` boolean (derive enablement from `status`)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 4a8c8c9..HEAD -- internal/knowledge/knowledge.go internal/feature/knowledgecards/skills.go internal/feature/knowledgecards/knowledgefocus.go internal/cli/knowledge.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S–M (≈45–60 min)
- **Risk**: MED (touches the skills lifecycle queries and the skill cards UI)
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `4a8c8c9`, 2026-06-22

## Why this matters

The `skills` collection has both a `status` select (`proposed` | `active` |
`archived` | `rejected`) and an `enabled` bool. `enabled` is fully derived:
`ProposeSkill` writes `enabled=false`, and `Transition` writes
`enabled = (to == "active")` — so `enabled == (status == "active")` always.
The two hot queries (`ActiveSkills`, `LoadSkill`) filter on
`status = 'active' && enabled = true`, where the second term is redundant. This
dual-field pattern is a consistency footgun (any write that flips one without
the other corrupts the invariant) for zero added information. Deriving
enablement from `status` removes the bool and the synchronized write. This is a
prerequisite for the consolidated schema baseline (plan 156) to drop the
`enabled` column.

> **Scope discipline — the `enabled` trap:** `llm_models` and `llm_providers`
> ALSO have an `enabled` bool (in `internal/store/llm_settings.go`), and it is a
> genuine, independent field — **do not touch it.** This plan only removes
> `skills.enabled`. Every change here is inside `internal/knowledge` and the
> knowledge cards/CLI.

## Current state

`skills.enabled` is declared in `migrations/1749600000_init.go:104`
(`&core.BoolField{Name: "enabled"}`). **Do not touch the migration in this
plan** — plan 156 owns the schema rebuild; here we only stop the code from
using the field.

`internal/knowledge/knowledge.go` writes/queries it at four sites:

`ProposeSkill` (line 110):
```go
	rec.Set("status", StatusProposed)
	rec.Set("enabled", false) // enabled flips on approval
```

`Transition` (lines 143–146):
```go
	rec.Set("status", to)
	if kind == Skill {
		rec.Set("enabled", to == StatusActive)
	}
```

`ActiveSkills` (lines 300–303):
```go
func ActiveSkills(app core.App) ([]*core.Record, error) {
	return app.FindRecordsByFilter(string(Skill),
		"status = 'active' && enabled = true", "name", 0, 0, nil)
}
```

`LoadSkill` (lines 306–309):
```go
	rec, err := app.FindFirstRecordByFilter(string(Skill),
		"status = 'active' && enabled = true && name = {:name}",
		dbx.Params{"name": name})
```

The `StatusActive = "active"` constant lives at `knowledge.go:37`.

`internal/feature/knowledgecards/skills.go` reads `enabled` into view-models at
three sites (the package imports `internal/knowledge` at line 16):
- `buildSkillsSummary` (line 62): `Enabled: r.GetBool("enabled"),`
- `SkillRecordOf` (line 90): `Enabled: r.GetBool("enabled"),`

The view-model fields `SkillRow.Enabled` (line 28) and `SkillRecord.Enabled`
(line 39) drive two "enabled" pills (`skills.go:134`, `skills.go:149`). Note
both `buildSkillsSummary` and `buildSkillsManage`'s active list come from
`knowledge.FilterActive(... Skill ...)` which already returns only
`status='active'` records, so those rows are always enabled today.

`internal/feature/knowledgecards/knowledgefocus.go` (line 258) also reads it:
```go
		out = append(out, SkillRecordCard(SkillRecord{
			...
			Enabled:     r.GetBool("enabled"),
			...
		}))
```

`internal/cli/knowledge.go` (line 36) exports it:
```go
		"enabled":     r.GetBool("enabled"),
```

**Approach:** keep the `Enabled bool` view-model fields and the CLI key (so the
UI pills and CLI output are byte-unchanged), but change every **data source**
from `r.GetBool("enabled")` to `r.GetString("status") == knowledge.StatusActive`.
Then remove the two `rec.Set("enabled", …)` writes and the `&& enabled = true`
query terms. Result: identical behavior, one fewer column.

Convention: html aliased `h`, gomponents, `internal/knowledge` is the domain
package the cards import.

## Commands you will need

| Purpose          | Command                                   | Expected on success     |
|------------------|-------------------------------------------|-------------------------|
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...`            | exit 0                  |
| Vet              | `go vet ./...`                            | exit 0                  |
| Tests            | `go test ./...`                           | all packages `ok`       |
| Dead-code gate   | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | exit 0, no output |
| Format check     | `gofmt -l internal/`                      | empty output            |
| Whitespace       | `git diff --check`                        | no output               |

(In a TLS-intercepting sandbox the Go commands need the GOPROXY shim — see
`docs/hyperagent-sandbox.md`.)

## Scope

**In scope** (the only files you may modify):
- `internal/knowledge/knowledge.go`
- `internal/feature/knowledgecards/skills.go`
- `internal/feature/knowledgecards/knowledgefocus.go`
- `internal/cli/knowledge.go`
- Test files in `internal/knowledge/` and `internal/feature/knowledgecards/`
  ONLY if a test references `skills.enabled` and breaks (update it to assert
  `status` instead). Do NOT weaken a test to pass.

**Out of scope** (do NOT touch):
- `internal/store/llm_settings.go` and anything about `llm_models.enabled` /
  `llm_providers.enabled` — a different, real field.
- `migrations/*` — the `enabled` column stays until plan 156.
- The memory lifecycle (memories never had `enabled`).

## Git workflow

- Branch off `origin/main` (executor worktree convention).
- One commit; conventional-commit subject, e.g.
  `refactor: drop redundant skills.enabled, derive from status`.
- Do NOT push or merge unless the operator instructs it.

## Steps

### Step 1: Remove the `enabled` writes in the lifecycle

In `internal/knowledge/knowledge.go`:
1. In `ProposeSkill` delete the `rec.Set("enabled", false)` line (line ~110).
2. In `Transition` delete the whole skill block (lines ~144–146):
   ```go
	if kind == Skill {
		rec.Set("enabled", to == StatusActive)
	}
   ```

**Verify**: `grep -n 'Set("enabled"' internal/knowledge/knowledge.go` → no match.

### Step 2: Drop the `enabled` term from the queries

In `internal/knowledge/knowledge.go`:
1. `ActiveSkills` filter → `"status = 'active'"`.
2. `LoadSkill` filter → `"status = 'active' && name = {:name}"`.

**Verify**: `grep -n 'enabled = true' internal/knowledge/knowledge.go` → no match; `go build ./internal/knowledge/` → exit 0.

### Step 3: Derive `Enabled` from `status` in the skill cards

In `internal/feature/knowledgecards/skills.go`, change both reads:
- `buildSkillsSummary` (line ~62): `Enabled: r.GetString("status") == knowledge.StatusActive,`
- `SkillRecordOf` (line ~90): `Enabled: r.GetString("status") == knowledge.StatusActive,`

In `internal/feature/knowledgecards/knowledgefocus.go` (line ~258):
- `Enabled: r.GetString("status") == knowledge.StatusActive,`
  (If `knowledgefocus.go` does not already import `internal/knowledge`, add it:
  `"github.com/alexradunet/balaur/internal/knowledge"`. Verify with
  `grep -n 'internal/knowledge' internal/feature/knowledgecards/knowledgefocus.go`.)

Leave the `SkillRow.Enabled` / `SkillRecord.Enabled` struct fields and the
"enabled" pill render code unchanged.

**Verify**: `grep -rn 'GetBool("enabled")' internal/feature/knowledgecards/` → no matches; `go build ./internal/feature/knowledgecards/` → exit 0.

### Step 4: Derive the CLI export from `status`

In `internal/cli/knowledge.go` (line ~36):
```go
		"enabled":     r.GetString("status") == "active",
```
(CLI uses the literal `"active"` to avoid importing the knowledge package if it
doesn't already; check with `grep -n 'internal/knowledge' internal/cli/knowledge.go`
and use `knowledge.StatusActive` if the import is already present.)

**Verify**: `grep -rn 'GetBool("enabled")' internal/cli/` → no matches; `go build ./internal/cli/` → exit 0.

### Step 5: Reconcile tests

Run `go test ./internal/knowledge/... ./internal/feature/knowledgecards/...`.
If a test fails because it set or asserted `skills.enabled`, update it to drive
the lifecycle through `Transition`/`status` instead (the behavior is unchanged;
only the storage field is gone). If no test references it, nothing to do.

**Verify**: those package tests → `ok`.

### Step 6: Full gate

Run the whole verification set from "Commands you will need". All must pass.

## Test plan

No new tests required (behavior-preserving). The existing knowledge lifecycle
tests are the contract: an approved skill must still appear in `ActiveSkills`
and be loadable via `LoadSkill`; a proposed/archived skill must not. If those
assertions don't already exist and a reviewer wants belt-and-suspenders, add a
small table test in `internal/knowledge/knowledge_test.go` asserting that after
`Transition(..., StatusActive)` the skill is returned by `ActiveSkills`, and
after `Transition(..., StatusArchived)` it is not — but this is optional.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -rn 'enabled = true' internal/knowledge/` → no matches.
- [ ] `grep -rn 'Set("enabled"' internal/knowledge/` → no matches.
- [ ] `grep -rn 'GetBool("enabled")' internal/knowledge/ internal/feature/knowledgecards/ internal/cli/` → no matches.
- [ ] `grep -rn 'GetBool("enabled")' internal/store/` → STILL PRESENT (llm fields untouched — sanity check that scope held).
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go vet ./...` exits 0.
- [ ] `go test ./...` — all packages `ok`.
- [ ] `staticcheck ./...` exits 0 with no output.
- [ ] `gofmt -l internal/` empty; `git diff --check` empty.
- [ ] No file outside the in-scope list is modified (`git status`).
- [ ] `plans/readme.md` status row for 155 updated.

## STOP conditions

Stop and report back (do not improvise) if:
- Any "Current state" excerpt does not match the live code (drift since `4a8c8c9`).
- A non-card consumer reads `skills.enabled` outside the in-scope files (e.g. a
  search-index builder) — handle it or report; it may change scope.
- A test asserts `skills.enabled` in a way that implies it is meant to diverge
  from `status` — report rather than forcing it.
- You find yourself about to edit `internal/store/llm_settings.go` — that's the
  wrong `enabled`; stop.

## Maintenance notes

- After this lands, plan 156's baseline does not declare the `skills.enabled`
  column, and the proposed `(status, enabled)` compound index becomes
  unnecessary — `idx_skills_status` alone covers the `status='active'` queries.
- Reviewer: confirm the two "enabled" pills still render for active skills (they
  derive from `status` now) and that `LoadSkill`/`ActiveSkills` still exclude
  proposed/archived skills.
