# Plan 124: Delete the staticcheck-confirmed dead code, the deprecated import, and the silent test bug

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat b61e060..HEAD -- internal/web/tasks.go internal/web/models.go internal/web/chat.go internal/kronk/officialmodel.go internal/cli/cli_test.go internal/web/fakeclient_test.go internal/search/index.go internal/search/fts5_test.go internal/tools/tasks_test.go`
> If any in-scope file changed since this plan was written, re-run the
> staticcheck command in Step 0 and reconcile against its live output before
> proceeding; on a mismatch with the symbols listed below, treat it as a STOP
> condition.

## Status

- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: tech-debt
- **Planned at**: commit `b61e060`, 2026-06-21

## Why this matters

`go vet` and `gofmt` (the CI gates) pass clean, but `staticcheck` — the standard
Go static analyzer — finds a cluster of provably-dead code that has accumulated
with zero CI signal: an entire unused calendar/timeline scaffolding block in
`internal/web/tasks.go` (a duplicate of the shipped `internal/feature/taskcards`
version), four orphaned helpers in `internal/web/models.go` (the ones plan 116
was meant to delete but missed), a dead `cardURL`, a dead `kronk.Official()`, two
unused test helpers, a deprecated import, and one test that silently asserts
nothing. Deleting all of it honors the suckless rule (delete dead code, one
source of truth) and — critically — makes the tree `staticcheck`-clean so that
**plan 125** can wire `staticcheck` into CI without it going red on day one.

Every symbol below was confirmed dead by `staticcheck` (U1000 "unused") and by a
manual reference search; the `taskCard`/`taskTransition`/`chatNudges`/
`taskCardHTML` functions in the same `tasks.go` file are **live** (route-wired in
`internal/web/web.go`) and must NOT be touched.

## Current state

- `internal/web/tasks.go` — the dead calendar/timeline cluster is lines 78–182:
  `calItem` (78), `calCell` (83), `calView` (91), `buildCalendar` (97),
  `mondayOf` (155), `const timelineDays` (166), `tlItem` (168), `tlDay` (173),
  `tlView` (179). The live `taskCard` handler begins at line 188 — stop before it.
- `internal/web/models.go` — four orphans: `type modelsPageData` (62), method
  `(*handlers).renderModelsPanel` (138, returns `template.HTML`),
  `buildBalaurHeadOptions` (495), `buildBalaurHeadOptionsFor` (574). The live
  avatar roster builder is `buildAvatarOptions` (471) — a different function;
  do not touch it. `models.go` also uses `template.HTMLEscapeString` in
  `installRuntime` (~line 548), so the `html/template` import stays after the
  deletions — that is expected.
- `internal/web/chat.go:72` — `func cardURL` (unused).
- `internal/kronk/officialmodel.go:62` — `func Official()` returns
  `OfficialByKey("medium")`; zero callers (live code uses `OfficialModels()` /
  `OfficialByKey()`).
- `internal/cli/cli_test.go:90` — `func executeRaw` (unused test helper).
- `internal/web/fakeclient_test.go:26` — `func seedFailingModel` (unused test helper).
- `internal/search/index.go:15` and `internal/search/fts5_test.go:19` — both
  blank-import `_ "github.com/ncruces/go-sqlite3/embed"`, which staticcheck
  flags SA1019 as deprecated/"unnecessary": the adjacent `_ ".../driver"` import
  already pulls the embedded WASM build.
- `internal/tools/tasks_test.go:88` — `out` is assigned from a `task_add`
  `Execute(...)` then overwritten at ~line 95 by a `task_list` call before being
  read (staticcheck SA4006 "this value of out is never used"), so the
  `task_add` output is silently never asserted.

Convention to follow: delete each symbol **with its doc comment**. Match the
surrounding gofmt style. The repo uses standard `testing` (no assertion libs).

## Commands you will need

| Purpose     | Command                                              | Expected on success |
|-------------|------------------------------------------------------|---------------------|
| Build       | `CGO_ENABLED=0 go build ./...`                       | exit 0              |
| Vet         | `go vet ./...`                                        | exit 0              |
| Tests       | `go test ./...`                                       | all pass            |
| Format      | `gofmt -l internal/`                                  | empty output        |
| Whitespace  | `git diff --check`                                    | no output           |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | see steps        |

(`go run …staticcheck@latest` downloads to the module cache and analyzes
read-only; it does not modify the working tree.)

## Step 0: Baseline the staticcheck output

Run `go run honnef.co/go/tools/cmd/staticcheck@latest ./... 2>/dev/null`.
Confirm it lists the U1000/SA1019/SA4006 entries named in "Current state". The
ST1001 "dot imports" lines are expected and out of scope here (plan 127 handles
them). Keep this output to compare against in Step 9.

## Step 1: Delete the calendar/timeline cluster in `internal/web/tasks.go`

Remove `calItem`, `calCell`, `calView`, `buildCalendar`, `mondayOf`,
`timelineDays`, `tlItem`, `tlDay`, `tlView` (lines 78–182, each with its
comment). Do NOT touch anything from `taskCard` (188) onward, nor
`taskView`/`taskViewOf`/`questGroup` above (27–76) — those are live.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

## Step 2: Delete the four orphans in `internal/web/models.go`

Remove `modelsPageData`, `renderModelsPanel`, `buildBalaurHeadOptions`,
`buildBalaurHeadOptionsFor` (each with its comment).

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0. (If the build complains
that `html/template` is now unused, STOP — that means a removal was wider than
intended; `installRuntime` should still use `template.HTMLEscapeString`.)

## Step 3: Delete `cardURL` (`internal/web/chat.go:72`) and `Official` (`internal/kronk/officialmodel.go:62`)

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

## Step 4: Delete the unused test helpers

Remove `executeRaw` (`internal/cli/cli_test.go:90`) and `seedFailingModel`
(`internal/web/fakeclient_test.go:26`), each with its comment.

**Verify**: `go test ./internal/cli/ ./internal/web/` → all pass.

## Step 5: Remove the deprecated `go-sqlite3/embed` import

Delete the `_ "github.com/ncruces/go-sqlite3/embed"` line from
`internal/search/index.go:15` and `internal/search/fts5_test.go:19`. Leave the
adjacent `_ ".../driver"` import in place.

**Verify**: `go test ./internal/search/` → all pass (the FTS5 index still builds
and queries — the driver registers the WASM build without the extra import).

## Step 6: Fix the silent assertion in `internal/tools/tasks_test.go:88`

Read the test around lines 85–100. The first `out, err := …task_add….Execute(...)`
result is discarded. Add a meaningful assertion on that `task_add` output before
the `task_list` call reassigns `out` — assert what `task_add` is contracted to
return (e.g. it is non-empty and/or carries the task's refresh marker /
confirmation text, mirroring how other tests in this file check tool output). If,
after reading, the `task_add` return is genuinely not worth asserting, change the
assignment to `_, err :=` so the dead store is explicit rather than misleading.
Do not weaken any existing assertion.

**Verify**: `go test ./internal/tools/` → all pass.

## Step 7: Format and full build/test

**Verify**:
- `gofmt -l internal/` → empty
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass

## Step 8: Confirm staticcheck no longer reports the deleted symbols

Run `go run honnef.co/go/tools/cmd/staticcheck@latest ./... 2>/dev/null`.
Expected: the U1000 lines for every symbol deleted above are gone, the two
SA1019 `go-sqlite3/embed` lines are gone, and the SA4006
`internal/tools/tasks_test.go:88` line is gone. Only the ST1001 dot-import lines
(out of scope) should remain.

## Test plan

- No new test files. Step 6 strengthens one existing test
  (`internal/tools/tasks_test.go`) so it actually asserts the `task_add` output.
- Regression net: the full `go test ./...` must stay green — deletions are of
  unreferenced symbols, so behavior is unchanged.
- Model the Step 6 assertion on the other tool-output checks already in
  `internal/tools/tasks_test.go`.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` passes
- [ ] `gofmt -l internal/` is empty and `git diff --check` is clean
- [ ] `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` reports NO U1000
      for the deleted symbols, NO SA1019 for `go-sqlite3/embed`, and NO SA4006
      for `tasks_test.go:88`
- [ ] `grep -rn "buildCalendar\|modelsPageData\|renderModelsPanel\|buildBalaurHeadOptions\|func cardURL\|func Official(\|func executeRaw\|seedFailingModel" internal/` returns nothing
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- A symbol listed as dead turns out to have a live caller (build fails when you
  delete it) — the codebase drifted; report which symbol.
- Removing the `html/template` import becomes necessary in `models.go` (it
  should not — `installRuntime` still uses it).
- Removing the `go-sqlite3/embed` import breaks `go test ./internal/search/`
  (the driver no longer registers) — restore it and report.
- Any deletion forces a change to a file not in the Scope list.

## Scope

**In scope** (the only files you may modify):
- `internal/web/tasks.go`, `internal/web/models.go`, `internal/web/chat.go`
- `internal/kronk/officialmodel.go`
- `internal/cli/cli_test.go`, `internal/web/fakeclient_test.go`
- `internal/search/index.go`, `internal/search/fts5_test.go`
- `internal/tools/tasks_test.go`
- `plans/readme.md` (status row only)

**Out of scope** (do NOT touch):
- The live `taskCard`/`taskTransition`/`chatNudges`/`taskCardHTML` functions in
  `tasks.go` and `buildAvatarOptions` in `models.go`.
- The ST1001 dot imports (plan 127) and CI wiring (plan 125).
- The `internal/feature/taskcards` calendar/timeline components — those are the
  LIVE versions; this plan deletes only the dead `internal/web` duplicate.

## Git workflow

- Branch off `origin/main`: `improve/124-staticcheck-dead-code-cleanup`.
- Commit per step or per logical group; conventional-commit subjects, e.g.
  `refactor(web): delete dead calendar/timeline scaffolding (staticcheck U1000)`.
- Do NOT push or open a PR unless the operator instructs it.

## Maintenance notes

- This plan is the prerequisite for **plan 125** (staticcheck/govulncheck CI
  gate). Land 124 first so CI starts green.
- The `internal/web/tasks.go` calendar/timeline block was a duplicate of
  `internal/feature/taskcards`; if a calendar/timeline web handler is ever
  re-added, build it from the `taskcards` components, not by reviving this
  scaffolding.
- A reviewer should confirm the diff is deletion-only (plus the one test
  assertion in Step 6) and that `git status` shows no out-of-scope files.
