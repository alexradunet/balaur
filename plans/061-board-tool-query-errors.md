# Plan 061: agent board tools stop swallowing board-load query errors

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 7b16063..HEAD -- internal/tools/ui.go internal/tools/ui_test.go`
> If either file changed since this plan was written, compare the "Current
> state" excerpts against the live code before proceeding; on a mismatch,
> treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: MED
- **Depends on**: none
- **Category**: bug (correctness / error handling)
- **Planned at**: commit `7b16063`, 2026-06-14

## Why this matters

The two agent-facing board tools — `board_compose` and `board_add_card`, in
`internal/tools/ui.go` — load the boards collection with the query error
discarded by a blank identifier (`_`). When that query fails:

- `board_compose` (line 182) silently treats the board list as empty, so the new
  board's `sort` is computed from nothing (collides at `sort = 0`).
- `board_add_card` (line 253) silently treats the board list as empty and then
  tells the model **"no board matches X — boards: "** (an empty list) — i.e. it
  reports "you have no boards" when the real problem is a failed DB query. The
  model may then create a duplicate board or give the owner a wrong answer.

These tools are how the model manipulates the dashboard on the owner's behalf, so
a swallowed error becomes a wrong action or a misleading message with no trace.
This is exactly the class the prior "error-swallow sweep" (plan 044) targeted;
these two sites were not covered then. The repo convention for agent tools is to
**return the error text as the tool's result string** (with `nil` Go error) so
the model sees it and can react — see the many `return fmt.Sprintf("board_…: …",
err), nil` sites already in this same file.

## Current state

`internal/tools/ui.go`. Both tools return `(string, error)` from their `Execute`
func, where the **string is the message the model sees** and a non-nil Go error
is reserved for genuine infrastructure faults; throughout this file, expected
failures are surfaced as `return fmt.Sprintf("...: %s", err), nil`.

**Site 1 — `boardComposeTool`, computing the next sort value (lines ~181–190):**

```go
		// Find next sort value.
		existing, _ := app.FindRecordsByFilter("boards", "1=1", "sort", 0, 0, nil)
		maxSort := -1
		for _, r := range existing {
			s := int(r.GetFloat("sort"))
			if s > maxSort {
				maxSort = s
			}
		}
		nextSort := maxSort + 1
```

Note `err` is already in scope here (declared at the earlier
`cleaned, err := cards.ValidateCards(args.Cards)` line), so reuse it with `:=`
(left side introduces the new `existing`).

**Site 2 — `boardAddCardTool`, resolving the target board (lines ~252–262):**

```go
		// Resolve the board: load all boards and match by id, then name, then substring.
		all, _ := app.FindRecordsByFilter("boards", "1=1", "sort", 0, 0, nil)

		// Helper: build listing for error messages.
		boardNames := func() string {
			names := make([]string, 0, len(all))
			for _, r := range all {
				names = append(names, r.GetString("name"))
			}
			return strings.Join(names, ", ")
		}
```

Note `err` is already in scope here (declared at the earlier
`cleaned, err := cards.Validate(args.Type, args.Params)` line), so reuse it.

The file already imports `fmt` and `strings` (used throughout). Confirm with
`grep -n '"fmt"\|"strings"' internal/tools/ui.go` before editing.

## Commands you will need

| Purpose   | Command                  | Expected on success |
|-----------|--------------------------|---------------------|
| Drift     | `git diff --stat 7b16063..HEAD -- internal/tools/ui.go internal/tools/ui_test.go` | empty |
| Build     | `CGO_ENABLED=0 go build -o /tmp/balaur-061 .` | exit 0 |
| Vet       | `go vet ./internal/tools/` | exit 0 |
| Package tests | `go test ./internal/tools/` | all pass |
| All tests | `go test ./...`          | all pass |
| No blank board query remains | `grep -nE ', _ := app\.FindRecordsByFilter\("boards"' internal/tools/ui.go` | no matches |

## Scope

**In scope** (the only files you should modify):
- `internal/tools/ui.go` — the two `FindRecordsByFilter("boards", ...)` sites only
- `internal/tools/ui_test.go` — only if you add the happy-path assertion noted in Test plan; do not rewrite existing tests

**Out of scope** (do NOT touch):
- Any other `, _ :=` / `_, _ =` in `ui.go` or elsewhere — this plan is exactly
  the two `boards` query sites above. (If you spot other swallowed errors, list
  them in your report; do not fix them here.)
- The tool specs, argument schemas, audit calls, validation, or the resolve
  (id → name → substring) matching logic — leave behavior identical except for
  surfacing the query error.
- Any change to the `(string, error)` tool contract or to the message text on
  the *success* path.

## Git workflow

- Branch: `improve/061-board-tool-query-errors`
- One commit; conventional-commit style: e.g.
  `fix(tools): surface board-load query errors in board_compose/board_add_card`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Surface the error in `board_compose` (Site 1)

Change the blank-identifier load to check the error and return a model-facing
message before computing `nextSort`:

```go
		// Find next sort value.
		existing, err := app.FindRecordsByFilter("boards", "1=1", "sort", 0, 0, nil)
		if err != nil {
			return fmt.Sprintf("board_compose: loading boards: %s", err), nil
		}
		maxSort := -1
		for _, r := range existing {
```

(Reuse the in-scope `err`; `existing` is the new variable, so `:=` is correct.)

**Verify**: `go build ./internal/tools/` compiles (no "declared and not used" or
"no new variables on left side of :=" error).

### Step 2: Surface the error in `board_add_card` (Site 2)

Change the blank-identifier load the same way, before the `boardNames` helper:

```go
		// Resolve the board: load all boards and match by id, then name, then substring.
		all, err := app.FindRecordsByFilter("boards", "1=1", "sort", 0, 0, nil)
		if err != nil {
			return fmt.Sprintf("board_add_card: loading boards: %s", err), nil
		}
```

(Reuse the in-scope `err`; `all` is the new variable.)

**Verify**: `go build ./internal/tools/` compiles.

### Step 3: Build, vet, test the package and the whole tree

```
go vet ./internal/tools/
go test ./internal/tools/
CGO_ENABLED=0 go build -o /tmp/balaur-061 . && go test ./...
```

**Verify**: vet clean; `internal/tools` tests pass (existing
`TestBoardComposeCreatesRecord`, `TestBoardAddCardHappyPath`, etc. still green —
this proves the success path is unchanged); whole-tree build + tests pass.

### Step 4: Confirm both swallow sites are gone

```
grep -nE ', _ := app\.FindRecordsByFilter\("boards"' internal/tools/ui.go
```

**Verify**: no matches.

## Test plan

- The **success path is already covered** by existing tests in
  `internal/tools/ui_test.go` (`TestBoardComposeCreatesRecord`,
  `TestBoardComposeWritesAuditLog`, `TestBoardAddCardHappyPath`,
  `TestBoardAddCardWritesAuditLog`). Keeping them green is the primary regression
  signal that you only added an error branch and changed nothing else.
- The **failure path is not cheaply unit-testable**: forcing
  `FindRecordsByFilter("boards", ...)` to return an error requires a broken or
  missing collection, which the shared `storetest.NewApp(t)` harness does not
  provide a clean hook for. Do **not** invent a fragile failure-injection
  mechanism. It is acceptable for this plan to verify the failure branch by
  inspection (the new `if err != nil { return ... }` blocks) plus the grep in
  Step 4. If you find an existing, idiomatic way in this repo's tests to make a
  collection query fail (search `ui_test.go` and `storetest` for precedent), you
  may add one negative test asserting the returned string starts with
  `"board_compose: loading boards:"` / `"board_add_card: loading boards:"`; if
  no clean precedent exists, skip it and say so in your report.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -nE ', _ := app\.FindRecordsByFilter\("boards"' internal/tools/ui.go` → no matches
- [ ] `grep -c 'board_compose: loading boards' internal/tools/ui.go` → `1`
- [ ] `grep -c 'board_add_card: loading boards' internal/tools/ui.go` → `1`
- [ ] `go vet ./internal/tools/` exits 0
- [ ] `go test ./internal/tools/` passes (all existing board tests green)
- [ ] `CGO_ENABLED=0 go build -o /tmp/balaur-061 .` exits 0 and `go test ./...` passes
- [ ] `git status --porcelain` shows only `internal/tools/ui.go` (and `ui_test.go` only if you added a negative test) modified
- [ ] `plans/readme.md` status row for 061 updated (unless your reviewer maintains it)

## STOP conditions

Stop and report back (do not improvise) if:

- Either `FindRecordsByFilter("boards", ...)` site does not match the "Current
  state" excerpt (the file drifted).
- Adding `existing, err :=` / `all, err :=` produces a compile error about `err`
  (e.g. it is NOT already in scope where you expected) — re-read the surrounding
  function and report what scope you actually found rather than guessing.
- A previously-passing `internal/tools` test fails after your change — that means
  you altered behavior beyond surfacing the error; stop and report.

## Maintenance notes

- The convention these fixes follow — expected failures returned as the tool's
  result string with a `nil` Go error so the model sees them — is the contract
  for every tool in `internal/tools`. New board tools should load boards the same
  way (checked error → `fmt.Sprintf("<tool>: loading boards: %s", err), nil`).
- A reviewer should confirm the change is purely additive (a new error branch)
  and that the success-path message/format and the board-resolution logic are
  byte-for-byte unchanged.
- Related but intentionally NOT in this plan: the web handlers in
  `internal/web/boards.go` swallow `loadBoards()` errors at four call sites —
  that is plan 062.
