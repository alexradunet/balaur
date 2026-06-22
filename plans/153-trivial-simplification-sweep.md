# Plan 153: Clear the cycle-15 trivial simplification sweep (dead code + reinvented stdlib + duplicated helpers)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 61e79d7..HEAD -- internal/web/settings.go internal/web/models.go internal/feature/taskcards/questsfocus.go internal/feature/lifecards/lifelogfocus.go internal/feature/journalcards/dayfocus.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S (≈30–45 min; mechanical, behavior-preserving)
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `61e79d7`, 2026-06-22

## Why this matters

The fifteenth improve cycle (ponytail over-engineering audit at `ab2c0a9`)
verified ~20 trivial wins but deliberately left them un-planned "to do
directly" (see `plans/readme.md` §"Findings considered and rejected", lines
~1176–1191). They were never executed. This plan packages the subset that
**survives re-vetting** into one executor pass. Every item is behavior-preserving
deletion or a swap to an already-used stdlib/shared helper — the existing test
suite is the regression net. Net effect: ~80 fewer lines, two stale/misleading
comments gone, one fewer hand-rolled int formatter.

**Scope discipline (read this):** several items from the original cycle-15 list
were re-checked and **excluded** because they do not survive vetting — do NOT
touch them:
- `turn.ClientFor` (`internal/turn/models.go:194`) — a *later* cycle explicitly
  rejected removing it: it differs from `clientForConfig` in `embedModel`
  semantics and backs a real test (`models_test.go:52,62`). Leave it.
- `heads.Builtins()` (`internal/heads/heads.go:51`) — backs `heads_test.go:38`.
  Leave it.
- `OpenAIClient.HTTP` (`internal/llm/openai.go:32`) — a deliberate read seam
  (`openai.go:36`). Leave it.
- `enumContains` (`internal/cards/cards.go:355`) — already delegates to
  `slices.Contains`; the "swap" is done. (Optional micro-inline only — see the
  optional appendix; skip by default.)

## Current state

The repo is Go 1.26.4 (`go.mod`), so the `min`/`max` builtins and the `slices`
package are available everywhere.

**A. Dead code (write-only or empty — staticcheck cannot see these):**

- `internal/web/settings.go` — the entire file is `package web` and nothing
  else (12 bytes). A leftover stub.
- `internal/web/models.go` — the soul-avatar picker plumbing on the home-page
  data is **write-only dead** post-gomponents-migration. The live picker is
  `settingscards.ProfileView` (its own `ProfileAvatarOption` type; its comment
  at `settingscards/settingsfocus.go:57` even says it "Mirrors AvatarOption in
  internal/web/models.go"). The web copy is assigned but never read:
  - field `AvatarOptions []AvatarOption` at `models.go:40`
  - type `AvatarOption struct {...}` at `models.go:55-61`
  - assignment `data.AvatarOptions = buildAvatarOptions(h.app)` at `models.go:72`
  - func `buildAvatarOptions(app core.App) []AvatarOption` at `models.go:480-499`

  `grep -rn '\.AvatarOptions' internal/web/` returns **only** the assignment at
  `models.go:72` — no reader. (staticcheck stays green because the assignment
  counts as a "use" of `buildAvatarOptions`.)

**B. Reinvented stdlib:**

- `internal/feature/taskcards/questsfocus.go:137` — `func itoa(n int) string`,
  a ~16-line hand-rolled base-10 int→string with the comment "avoids importing
  fmt/strconv". Called at `:114` and `:128`. `strconv.Itoa` is exactly this.
  The file does **not** currently import `strconv`.

**C. Duplicated helper (already has a shared home):**

- `internal/feature/lifecards/lifelogfocus.go:73` — `func clip(s, n)` and
  `internal/feature/journalcards/dayfocus.go:110` — `func clipDay(s, n)` are
  **byte-identical** to `ui.Clip` (`internal/ui/text.go:10`). Both carry a now-
  **false** comment ("a local copy of internal/web's clipText, off-limits to
  feature packages by the layering law") — `internal/ui` is explicitly designed
  for feature packages to import (`ui/text.go:1-3`), and `lifecards/lines.go:50`
  **already calls `ui.Clip`**. So `lifecards` already imports `ui`; `journalcards`
  may need the import added.

  `clip` is called at `lifelogfocus.go:60`; `clipDay` at `dayfocus.go` (grep to
  find the call site(s) — confirm with `grep -n 'clipDay(' internal/feature/journalcards/dayfocus.go`).

Convention: this repo aliases the html package as `h "maragu.dev/gomponents/html"`
and imports shared primitives from `internal/ui`. Match surrounding import
ordering (stdlib block, then third-party, then `github.com/alexradunet/balaur/...`).

## Commands you will need

| Purpose          | Command                                   | Expected on success     |
|------------------|-------------------------------------------|-------------------------|
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...`            | exit 0                  |
| Vet              | `go vet ./...`                            | exit 0                  |
| Tests            | `go test ./...`                           | all packages `ok`       |
| Dead-code gate   | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | exit 0, no output |
| Format check     | `gofmt -l internal/`                      | empty output            |
| Whitespace       | `git diff --check`                        | no output               |

(`make lint` runs fmt+vet+staticcheck+test together.)

## Scope

**In scope** (the only files you may modify):
- `internal/web/settings.go` (delete the file)
- `internal/web/models.go`
- `internal/feature/taskcards/questsfocus.go`
- `internal/feature/lifecards/lifelogfocus.go`
- `internal/feature/journalcards/dayfocus.go`

**Out of scope** (do NOT touch):
- `internal/feature/settingscards/settingsfocus.go` — holds the **live**
  `ProfileAvatarOption`/`buildAvatarOptions`; it is the surviving copy, not the
  dead one. Deleting from here breaks the profile picker.
- `internal/turn/models.go`, `internal/heads/heads.go`, `internal/llm/openai.go`,
  `internal/cards/cards.go` — see the exclusions in "Why this matters".
- Any test file — these changes are behavior-preserving; existing tests must
  pass unchanged. Do not edit a test to make it pass.

## Git workflow

- Branch off `origin/main` (executor worktree convention).
- One commit is fine; conventional-commit subject, e.g.
  `refactor: clear cycle-15 trivial simplification sweep (dead code, itoa, clip dups)`.
- Do NOT push or merge unless the operator instructs it.

## Steps

### Step 1: Delete the empty stub file

`rm internal/web/settings.go`

**Verify**: `test ! -f internal/web/settings.go && echo gone` → `gone`; then
`CGO_ENABLED=0 go build ./internal/web/` → exit 0.

### Step 2: Delete the dead web avatar-picker plumbing

In `internal/web/models.go` remove all four pieces (they form one dead unit):
1. the `AvatarOptions []AvatarOption` field (line ~40),
2. the `AvatarOption` type declaration (lines ~55–61),
3. the assignment line `data.AvatarOptions = buildAvatarOptions(h.app)` (~72),
4. the `buildAvatarOptions` function (lines ~480–499).

If removing these orphans an import (e.g. nothing else uses it), remove the
import too — but `store`/`core` are used elsewhere in the file, so likely none.

**Verify**:
- `grep -rn 'AvatarOption\|buildAvatarOptions' internal/web/` → no matches.
- `grep -rn 'AvatarOption\|buildAvatarOptions' internal/feature/settingscards/` → still present (untouched).
- `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 3: Replace hand-rolled `itoa` with `strconv.Itoa`

In `internal/feature/taskcards/questsfocus.go`:
1. delete the `func itoa(n int) string {...}` (line ~137 to its closing brace),
2. at the two call sites (`:114`, `:128`) replace `itoa(` with `strconv.Itoa(`,
3. add `"strconv"` to the stdlib import block (alongside `"time"`).

**Verify**:
- `grep -n 'func itoa' internal/feature/taskcards/questsfocus.go` → no match.
- `go build ./internal/feature/taskcards/` → exit 0.
- `go test ./internal/feature/taskcards/` → `ok`.

### Step 4: Collapse the two `clip` duplicates onto `ui.Clip`

1. `internal/feature/lifecards/lifelogfocus.go`: delete `func clip(...)` (~73)
   and its doc comment; change the call at `:60` from `clip(t, 120)` to
   `ui.Clip(t, 120)`. (`lifecards` already imports `internal/ui` — confirm with
   `grep -n 'internal/ui' internal/feature/lifecards/lifelogfocus.go`; if the
   import is in a *different* file in the package, add it here.)
2. `internal/feature/journalcards/dayfocus.go`: delete `func clipDay(...)` (~110)
   and its doc comment; change its call site(s) from `clipDay(` to `ui.Clip(`;
   add the `github.com/alexradunet/balaur/internal/ui` import if absent.

**Verify**:
- `grep -rn 'func clip\b\|func clipDay\b' internal/feature/` → no matches.
- `grep -rn '\bclip(\|clipDay(' internal/feature/` → no matches (all call sites moved to `ui.Clip`).
- `go test ./internal/feature/lifecards/ ./internal/feature/journalcards/` → both `ok`.

### Step 5: Full gate

Run the whole verification set from "Commands you will need". All must pass.

## Optional appendix — cosmetic idiom swaps (LOW value; skip by default)

These are *lateral* changes (an idiom for an equally-clear hand-written form),
explicitly the kind of polish the repo's `modernize` sweep (plan 126) already
scoped. **Do them only if you are already editing the file for another reason,
or the operator asks.** They are not part of this plan's done criteria.

- Clamp helpers → `max(lo, min(n, hi))` builtins: `clampImportance`
  (`knowledge/knowledge.go:42`), `clampLayout` (`cards/cards.go:275`), the clamp
  tail of `clampInt` (`cards/cards.go:362`). *(Out-of-scope files — would need a
  scope expansion; note it, don't do it silently.)*
- `sort.*` → `slices.Sort`/`slices.SortFunc`: `self/self.go:117`,
  `ext/ext.go:64`, `taskcards/calendar.go:75`, `store/llm_settings.go:83`,
  `knowledge/knowledge.go:254`, `journalcards/dayfocus.go:86`.
- manual min/max loops → `slices.Min`/`slices.Max`: `ui/sparkline.go:46`,
  `ui/spark.go:36-41`, `lifecards/measure.go:179-184` (note: these compute lo
  AND hi in one pass, so two `slices.Min`+`slices.Max` calls is two passes —
  only worth it for readability on the tiny sparkline data).
- inline the one-line `enumContains` wrapper (`cards/cards.go:355`) at its two
  call sites (`:219`, `:323`) — it already just calls `slices.Contains`.

## Test plan

No new tests. Every change is behavior-preserving (dead-code removal, a stdlib
swap with identical output, and collapsing onto a byte-identical shared helper).
The existing suite is the regression net:
- `go test ./...` must stay all-`ok` (notably `internal/web`,
  `internal/feature/taskcards`, `internal/feature/lifecards`,
  `internal/feature/journalcards`, and `internal/feature/settingscards`).
- If any test *fails*, it means a "dead" symbol was actually live — STOP (see
  below); do not edit the test.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `internal/web/settings.go` does not exist.
- [ ] `grep -rn 'AvatarOption\|buildAvatarOptions' internal/web/` → no matches.
- [ ] `grep -n 'func itoa' internal/feature/taskcards/questsfocus.go` → no match; file imports `strconv`.
- [ ] `grep -rn 'func clip\b\|func clipDay\b' internal/feature/` → no matches.
- [ ] `CGO_ENABLED=0 go build ./...` exits 0.
- [ ] `go vet ./...` exits 0.
- [ ] `go test ./...` — all packages `ok`.
- [ ] `staticcheck ./...` exits 0 with no output.
- [ ] `gofmt -l internal/` is empty; `git diff --check` is empty.
- [ ] No file outside the in-scope list is modified (`git status`).
- [ ] `plans/readme.md` status row for 153 updated.

## STOP conditions

Stop and report back (do not improvise) if:
- Any "Current state" excerpt does not match the live code (drift since `61e79d7`).
- `grep -rn '\.AvatarOptions' internal/web/` shows a **reader** other than the
  `models.go:72` assignment — then the avatar plumbing is NOT dead; skip Step 2
  and report.
- A test fails after a change (it means the "dead" code was live, or a swap
  changed behavior) — revert that step and report.
- Any step seems to require editing an out-of-scope file.

## Maintenance notes

- The dead web avatar plumbing existed because the home page once rendered the
  soul picker itself; the gomponents migration moved that to
  `settingscards.ProfileView`. If a future change re-adds a soul picker to the
  home/chat shell, build it from `settingscards`/`ui`, not by reviving this.
- Reviewer: confirm Step 2's deletions are the `internal/web` copies, not the
  `settingscards` originals (the two share names — that's the whole trap).
- The optional appendix is deliberately deferred as low-value idiom polish; it is
  not debt that blocks anything. Leave it unless touching those files anyway.
