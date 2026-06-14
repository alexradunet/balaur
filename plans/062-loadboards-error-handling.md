# Plan 062: web board handlers stop swallowing loadBoards() errors (nil-deref guard)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat 7b16063..HEAD -- internal/web/boards.go internal/web/boards_test.go`
> If either file changed since this plan was written, compare the "Current
> state" excerpts against the live code before proceeding; on a mismatch,
> treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: MED
- **Depends on**: none
- **Category**: bug (robustness / error handling)
- **Planned at**: commit `7b16063`, 2026-06-14

## Why this matters

`internal/web/boards.go` defines `loadBoards() ([]*boardRecord, error)`. Three
handlers check its error correctly (`boardsIndex` line 277, the home loader line
301, `boardsDelete` line 445), but **four mutating handlers discard it** with
`boards, _ := h.loadBoards()`:

- `boardsCreate` (line 382)
- `boardsRename` (line 418)
- `boardsCardAdd` (line 517)
- `boardsCardRemove` (line 576)

In `boardsCardAdd` and `boardsCardRemove`, after the swallow the code finds the
"current" board by id and then calls `h.renderBoardCards(current)` and renders
the `board_grid` template with `Current: current`. If `loadBoards()` failed,
`boards` is `nil`, no match is found, `current` is `nil`, and the code proceeds
to use a nil `*boardRecord` — a nil-dereference / 500 on a live request path
*after the mutation already succeeded*, with no useful error for the owner. This
is the same swallowed-error class plan 044 swept elsewhere; these four sites were
not covered. The three sibling handlers already show the intended pattern:
`return e.InternalServerError("loading boards", err)`.

## Current state

`internal/web/boards.go`, as of `7b16063`.

`loadBoards` (lines 177–191):

```go
// loadBoards returns all boards sorted by sort field, then name.
func (h *handlers) loadBoards() ([]*boardRecord, error) {
	recs, err := h.app.FindRecordsByFilter("boards", "1=1", "sort,name", 0, 0, nil)
	if err != nil {
		return nil, err
	}
	...
	return out, nil
}
```

The **correct** sibling pattern already in the file (e.g. `boardsDelete`, line 445):

```go
	boards, err := h.loadBoards()
	if err != nil {
		return e.InternalServerError("loading boards", err)
	}
```

The **four sites to fix** (each currently swallows the error):

1. `boardsCreate` — line 382:
```go
	// Determine next sort value.
	existing, _ := h.loadBoards()
	nextSort := len(existing)
```

2. `boardsRename` — line 418 (after the record is saved, builds `board_header`):
```go
	boards, _ := h.loadBoards()
	var current *boardRecord
	for _, b := range boards {
		if b.ID == id {
			current = b
			break
		}
	}
```

3. `boardsCardAdd` — line 517 (after save, before `renderBoardCards` + `board_grid`):
```go
	boards, _ := h.loadBoards()
	var current *boardRecord
	for _, b := range boards {
		if b.ID == id {
			current = b
			break
		}
	}
	h.renderBoardCards(current)
```

4. `boardsCardRemove` — line 576 (identical shape to #3):
```go
	boards, _ := h.loadBoards()
	var current *boardRecord
	for _, b := range boards {
		if b.ID == id {
			current = b
			break
		}
	}
	h.renderBoardCards(current)
```

Each handler signature is `func (h *handlers) boardsX(e *core.RequestEvent) error`,
and `e.InternalServerError(msg string, err error) error` is the established way
these handlers report a server fault (already used at lines 378, 390, 414, 447,
459, 510, 514, 533, etc.). In every one of the four sites, `err` is already
declared earlier in the function, so `existing, err :=` / `boards, err :=` is the
correct form (the slice variable on the left is new).

## Commands you will need

| Purpose   | Command                  | Expected on success |
|-----------|--------------------------|---------------------|
| Drift     | `git diff --stat 7b16063..HEAD -- internal/web/boards.go internal/web/boards_test.go` | empty |
| Build     | `CGO_ENABLED=0 go build -o /tmp/balaur-062 .` | exit 0 |
| Vet       | `go vet ./internal/web/` | exit 0 |
| Package tests | `go test ./internal/web/` | all pass |
| All tests | `go test ./...`          | all pass |
| No blank loadBoards remains | `grep -nE ', _ := h\.loadBoards\(\)' internal/web/boards.go` | no matches |

## Scope

**In scope** (the only file you should modify):
- `internal/web/boards.go` — the four `, _ := h.loadBoards()` sites only

**Out of scope** (do NOT touch):
- `internal/web/boards_test.go` — unless you add the negative test described in
  Test plan (only if a clean precedent exists); do not rewrite existing tests.
- `loadBoards` itself, `renderBoardCards`, the `boardRecord`/`boardView` types,
  the templates, or the SSE patch calls — leave behavior identical except for
  returning a 500 instead of proceeding with a nil board on a load failure.
- The three handlers that already check the error (277, 301, 445).
- Any unrelated `, _ :=` elsewhere in the file. Exactly four sites change.
- Do NOT introduce a shared helper / refactor the duplicated find-current loop in
  this plan — the repo has repeatedly deferred `internal/web` decomposition
  (see `plans/readme.md` rejected notes). Keep the change minimal: add the error
  check at each site, nothing more.

## Git workflow

- Branch: `improve/062-loadboards-error-handling`
- One commit; conventional-commit style: e.g.
  `fix(web): board handlers surface loadBoards errors instead of nil-deref`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: `boardsCreate` (line 382)

```go
	// Determine next sort value.
	existing, err := h.loadBoards()
	if err != nil {
		return e.InternalServerError("loading boards", err)
	}
	nextSort := len(existing)
```

**Verify**: `go build ./internal/web/` compiles.

### Step 2: `boardsRename` (line 418)

```go
	boards, err := h.loadBoards()
	if err != nil {
		return e.InternalServerError("loading boards", err)
	}
	var current *boardRecord
	for _, b := range boards {
```

**Verify**: `go build ./internal/web/` compiles.

### Step 3: `boardsCardAdd` (line 517)

```go
	boards, err := h.loadBoards()
	if err != nil {
		return e.InternalServerError("loading boards", err)
	}
	var current *boardRecord
	for _, b := range boards {
```

**Verify**: `go build ./internal/web/` compiles.

### Step 4: `boardsCardRemove` (line 576)

```go
	boards, err := h.loadBoards()
	if err != nil {
		return e.InternalServerError("loading boards", err)
	}
	var current *boardRecord
	for _, b := range boards {
```

**Verify**: `go build ./internal/web/` compiles.

### Step 5: Vet, build, test

```
go vet ./internal/web/
go test ./internal/web/
CGO_ENABLED=0 go build -o /tmp/balaur-062 . && go test ./...
```

**Verify**: vet clean; `internal/web` tests pass (existing board handler tests in
`boards_test.go` / `handlers_test.go` still green — proving the success path is
unchanged); whole-tree build + tests pass.

### Step 6: Confirm all four swallow sites are gone

```
grep -nE ', _ := h\.loadBoards\(\)' internal/web/boards.go
grep -c 'return e.InternalServerError("loading boards", err)' internal/web/boards.go
```

**Verify**: first grep → no matches; second grep → `4` new sites + the 1
pre-existing site in `boardsDelete` = **5** total. (If `boardsDelete` already
used a slightly different message string, the count of the exact line may be 4 +
that; confirm by reading — the key invariant is the four new sites each have the
checked-error return.)

## Test plan

- Existing handler tests in `internal/web/boards_test.go` and
  `internal/web/handlers_test.go` cover the success paths of create/rename/
  card-add/card-remove; keeping them green is the regression signal that you only
  added an error branch.
- The failure path (loadBoards returns an error) is not cheaply unit-testable
  with the `newWebApp`/`storetest` harness — do not build a fragile injection. If
  you find an idiomatic precedent in this repo for forcing a collection query to
  fail, you may add one negative test asserting a 500 is returned; otherwise skip
  it and say so in your report. The grep in Step 6 plus inspection is sufficient
  evidence for this change.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -nE ', _ := h\.loadBoards\(\)' internal/web/boards.go` → no matches
- [ ] The four handlers `boardsCreate`, `boardsRename`, `boardsCardAdd`, `boardsCardRemove` each contain a checked-error `return e.InternalServerError("loading boards", err)` after their `loadBoards()` call
- [ ] `go vet ./internal/web/` exits 0
- [ ] `go test ./internal/web/` passes (all existing board tests green)
- [ ] `CGO_ENABLED=0 go build -o /tmp/balaur-062 .` exits 0 and `go test ./...` passes
- [ ] `git status --porcelain` shows only `internal/web/boards.go` (and `boards_test.go` only if you added a negative test) modified
- [ ] `plans/readme.md` status row for 062 updated (unless your reviewer maintains it)

## STOP conditions

Stop and report back (do not improvise) if:

- Any of the four sites does not match its "Current state" excerpt (the file drifted).
- A `boards, err :=` change produces a compile error about `err` not being in
  scope, or a "no new variables on left side of :=" error — re-read the function
  and report the actual scope rather than guessing.
- A previously-passing `internal/web` test fails after your change (you altered
  behavior beyond surfacing the error).
- You conclude the right fix requires extracting a shared helper / touching
  `renderBoardCards` — STOP; that is explicitly out of scope here.

## Maintenance notes

- All board handlers should treat `loadBoards()` like any other fallible load:
  check the error and return `e.InternalServerError("loading boards", err)`. A
  reviewer should grep `loadBoards` and confirm no `, _ :=` swallow remains.
- The duplicated "load boards → find current by id → render" block across
  `boardsRename`/`boardsCardAdd`/`boardsCardRemove` is real duplication, but
  consolidating it is deliberately deferred (the project has repeatedly declined
  `internal/web` decomposition). If that decision is ever revisited, the nil-guard
  added here moves into the shared helper.
