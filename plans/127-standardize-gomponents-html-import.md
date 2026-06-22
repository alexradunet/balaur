# Plan 127: Standardize the `gomponents/html` import on the `h` alias across feature cards

> **Executor instructions**: Follow this plan file by file. The Go compiler is
> your verification engine here — after each file, `go build` must pass before
> moving to the next. If anything in "STOP conditions" occurs, stop and report.
> When done, update the status row for this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat b61e060..HEAD -- internal/feature/`
> Also run `go run honnef.co/go/tools/cmd/staticcheck@latest ./... 2>/dev/null | grep ST1001`
> to see the current dot-import sites; reconcile against the file list below.

## Status

- **Priority**: P3
- **Effort**: S–M (mechanical but spans 19 files)
- **Risk**: LOW (pure rename; rendered HTML is byte-identical — existing tests pass unchanged)
- **Depends on**: none (independent of plan 125; see "Why")
- **Category**: tech-debt / idiom
- **Planned at**: commit `b61e060`, 2026-06-21

## Why this matters

The design-system layer is split on one convention. `internal/ui` (39 files),
`internal/feature/storybook`, `internal/web`, and the *newest* feature cards
(`modelcards/*`, `knowledgecards/memory.go`) alias the html package as
`h "maragu.dev/gomponents/html"` and write `h.Div(...)`, `h.Class(...)`. But 19
older `internal/feature/*cards` files dot-import it
(`. "maragu.dev/gomponents/html"`) and write bare `Div(...)`, `Class(...)`. The
split runs *through the same package* — `taskcards/today.go` dot-imports while
sibling `taskcards/taskcard.go`… also dot-imports, but `modelcards` (same layer)
aliases. Dot-import dumps ~150 element/attribute identifiers into package scope,
so a reader cannot tell `Header` (the html element) from a local symbol, and it
is the `staticcheck` ST1001 noise floor.

Standardizing on the `h` alias (the newest + majority convention) removes the
inconsistency and lets `staticcheck` run ST1001-clean. (Plan 125 suppresses
ST1001 via config regardless; after THIS plan lands, that suppression becomes
optional — you may remove the `-ST1001` line from `staticcheck.conf` so a future
*accidental* dot-import is caught. That is a one-line follow-up, noted below, not
required by this plan.)

This is a behavior-preserving rename: the generated HTML is identical, so the
full existing test suite is the regression net.

## Current state — the 19 dot-import files (verified at `b61e060`)

Each imports `. "maragu.dev/gomponents/html"` and calls html elements/attrs bare:

- `internal/feature/headscards/heads.go`
- `internal/feature/journalcards/day.go`, `internal/feature/journalcards/dayfocus.go`
- `internal/feature/knowledgecards/knowledgefocus.go`, `internal/feature/knowledgecards/skills.go`
- `internal/feature/lifecards/lifelog.go`, `internal/feature/lifecards/lifelogfocus.go`, `internal/feature/lifecards/lines.go`, `internal/feature/lifecards/measure.go`
- `internal/feature/settingscards/settings.go`, `internal/feature/settingscards/settingsfocus.go`
- `internal/feature/taskcards/calendar.go`, `internal/feature/taskcards/habits.go`, `internal/feature/taskcards/quests.go`, `internal/feature/taskcards/questsfocus.go`, `internal/feature/taskcards/taskcard.go`, `internal/feature/taskcards/taskscluster.go`, `internal/feature/taskcards/timeline.go`, `internal/feature/taskcards/today.go`

Exemplar of the target convention — `internal/feature/taskcards/today.go`
already aliases the OTHER two helper packages, so only the html import changes:
```go
import (
    "time"

    "github.com/pocketbase/pocketbase/core"
    g "maragu.dev/gomponents"
    data "maragu.dev/gomponents-datastar"
    . "maragu.dev/gomponents/html"   // <-- change to: h "maragu.dev/gomponents/html"

    "github.com/alexradunet/balaur/internal/tasks"
    "github.com/alexradunet/balaur/internal/ui"
)
```
Only the `gomponents/html` identifiers are dot-imported; gomponents core (`g.`)
and datastar (`data.`) are already aliased, so after the change ONLY html
identifiers (Div, Span, P, A, Class, ID, Header, Section, Button, Img, Style,
etc.) become undefined — the compiler will list every one to prefix with `h.`.

## Commands you will need

| Purpose     | Command                                              | Expected |
|-------------|------------------------------------------------------|----------|
| Build (pkg) | `CGO_ENABLED=0 go build ./internal/feature/taskcards/` | exit 0 (after fixing that pkg) |
| Build (all) | `CGO_ENABLED=0 go build ./...`                       | exit 0   |
| Tests       | `go test ./internal/feature/...`                     | all pass |
| Format      | `gofmt -l internal/feature/`                         | empty    |
| Dot-import scan | `grep -rn '\. "maragu.dev/gomponents/html"' internal/feature/` | empty when done |

## Steps

### Procedure (apply to each of the 19 files, one package at a time)

For each file:
1. Change the import line `. "maragu.dev/gomponents/html"` →
   `h "maragu.dev/gomponents/html"`.
2. Build the file's package (`CGO_ENABLED=0 go build ./internal/feature/<pkg>/`).
   The compiler prints every now-undefined identifier (`undefined: Div`,
   `undefined: Class`, …).
3. Prefix each reported identifier with `h.` (`Div(` → `h.Div(`, `Class(` →
   `h.Class(`, etc.). Re-build; repeat until the package compiles.
4. `gofmt -w` the file.

Work package by package so the build stays green between packages:
`taskcards` (8 files) → `lifecards` (4) → `settingscards` (2) →
`journalcards` (2) → `knowledgecards` (2) → `headscards` (1).

**Verify after each package**: `CGO_ENABLED=0 go build ./internal/feature/<pkg>/`
→ exit 0, then `go test ./internal/feature/<pkg>/` → all pass.

### Final verification

**Verify**:
- `grep -rn '\. "maragu.dev/gomponents/html"' internal/feature/` → empty
- `gofmt -l internal/feature/` → empty
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go test ./...` → all pass (HTML output is byte-identical, so storybook/card
  tests pass unchanged — if any FAIL, a prefix was missed or a non-html
  identifier was wrongly prefixed → STOP)
- `go run honnef.co/go/tools/cmd/staticcheck@latest ./... 2>/dev/null | grep ST1001`
  → empty

## Test plan

- No new tests. The change is a behavior-preserving rename; the existing
  `internal/feature/*` and `internal/feature/storybook` tests (which assert
  rendered markup) are the regression net. A passing suite proves the HTML is
  unchanged.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -rn '\. "maragu.dev/gomponents/html"' internal/feature/` returns nothing
- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go test ./...` passes
- [ ] `gofmt -l internal/feature/` empty; `git diff --check` clean
- [ ] `staticcheck ./...` reports no ST1001 in `internal/feature/`
- [ ] Only the 19 listed files (+ `plans/readme.md`) are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:
- A test that asserts rendered HTML FAILS after a file's conversion (a missed or
  wrong prefix — the output drifted).
- A file uses an identifier that is ambiguous between `gomponents/html` and
  another dot-import (there should be none — only html is dot-imported — but if
  one appears, report it).
- The conversion would require touching a file outside the 19 listed.

## Scope

**In scope**: the 19 `internal/feature/*cards` files listed above, and
`plans/readme.md` (status row).

**Out of scope**: `internal/ui`, `internal/web`, `internal/feature/storybook`
(already on `h`); any logic change; removing the `-ST1001` line from
`staticcheck.conf` (optional follow-up — see maintenance notes).

## Git workflow

- Branch off `origin/main`: `improve/127-standardize-gomponents-html-import`.
- Commit per package (e.g. `style(taskcards): alias gomponents/html as h`) so a
  reviewer can scan one package at a time; or one commit if you prefer.
- Do NOT push or open a PR unless the operator instructs it.

## Maintenance notes

- After this lands, you MAY remove the `-ST1001` exclusion from
  `staticcheck.conf` (plan 125) so any future accidental `.`-import of
  `gomponents/html` is flagged by CI. Removing it is safe only once `grep`
  confirms zero dot-imports remain repo-wide (check `internal/ui` and
  `internal/web` too — they already alias, so this should hold).
- New cards should follow the `h` convention; the storybook exemplars and
  `modelcards/*` are the models to copy.
