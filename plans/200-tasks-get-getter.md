# Plan 200: Add `tasks.Get(app, id)` and collapse the find-task-node-and-hydrate pattern across `cli`, `web`, and `tools`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 07fb4d6..HEAD -- internal/tasks/tasks.go internal/cli/task.go internal/web/tasks.go internal/tools/tasks.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none (lands before #201/#202 in the batch, but no hard dep)
- **Category**: tech-debt
- **Planned at**: commit `07fb4d6`, 2026-06-26

## Why this matters

The storage fact "a task is a row in the `nodes` collection that must be
hydrated (`tasks.Hydrate`) before its legacy fields are readable" is re-encoded
at five call sites across three packages: `cli/task.go`, `web/tasks.go`, and
`tools/tasks.go` (which even has its own `findTask` helper that two sibling
tools still re-inline around). The `tasks` package exposes `Hydrate` but no
getter, so every caller hand-writes `FindRecordById("nodes", id)` + `Hydrate`.
A change to how a task node is loaded (e.g. a soft-delete filter) would have to
be made in five places.

This mirrors the already-blessed `nodes.Get` (`internal/nodes/nodes.go:207`).
Adding `tasks.Get` collapses the find+hydrate pairing behind its owning package.
It does NOT hide the `"nodes"` collection name (a deliberate repo-wide idiom) —
it collapses the find+Hydrate **pairing** only.

## Current state

### The owning package has `Hydrate` but no getter

`internal/tasks/tasks.go`:

```go
// line 373
// Hydrate is the exported form of hydrate for use by other packages
// (cli, web, tools) that load task nodes directly and need the legacy field aliases.
func Hydrate(rec *core.Record) { hydrate(rec) }
```

The blessed exemplar to mirror — `internal/nodes/nodes.go:206`:

```go
// Get fetches one node by id.
func Get(app core.App, id string) (*core.Record, error) {
	rec, err := app.FindRecordById("nodes", id)
	if err != nil {
		return nil, fmt.Errorf("finding node %q: %w", id, err)
	}
	return rec, nil
}
```

### Five call sites that hand-roll find+Hydrate

1. `internal/cli/task.go:42` — a private `findTask` already does exactly this,
   with a friendly error:
   ```go
   func findTask(app core.App, id string) (*core.Record, error) {
       rec, err := app.FindRecordById("nodes", strings.TrimSpace(id))
       if err != nil {
           return nil, fmt.Errorf("no task with id %q — check `task list`", id)
       }
       tasks.Hydrate(rec)
       return rec, nil
   }
   ```
   (Called from `taskDoneCmd`, `taskSnoozeCmd`, `taskDropCmd`.)

2. `internal/web/tasks.go:19` — `loadTaskNode`:
   ```go
   func (h *handlers) loadTaskNode(id string) (*core.Record, error) {
       rec, err := h.app.FindRecordById("nodes", id)
       if err != nil {
           return nil, err
       }
       tasks.Hydrate(rec)
       return rec, nil
   }
   ```
   (Called from `taskCard`, `taskTransition` ×2, `taskEdit`.)

3. `internal/tools/tasks.go:243` — inline inside the snooze tool:
   ```go
   rec, err := app.FindRecordById("nodes", strings.TrimSpace(args.ID))
   // ... error handling ...
   tasks.Hydrate(rec)
   ```

4. `internal/tools/tasks.go:414` — inline inside another tool:
   ```go
   rec, err := app.FindRecordById("nodes", args.ID)
   // ...
   tasks.Hydrate(rec)
   ```

5. `internal/tools/tasks.go:447` — a private `findTask` helper used by three
   tools (`tools/tasks.go:305`, `:372`, `:435`):
   ```go
   // findTask decodes an {id} argument and loads the task node, hydrating it.
   func findTask(app core.App, argsJSON string) (*core.Record, error) {
       // ... json.Unmarshal of {"id": ...} ...
       rec, err := app.FindRecordById("nodes", strings.TrimSpace(args.ID))
       // ...
       tasks.Hydrate(rec)
       return rec, nil
   }
   ```
   (Note: this one ALSO decodes JSON. Keep the JSON decoding in the tool layer —
   `tasks.Get` takes an already-decoded id, not raw JSON.)

### Convention

- `tasks.Get` should take an already-trimmed-or-not id; trim inside, mirroring
  `cli.findTask` and `tools.findTask` which both `strings.TrimSpace`.
- Each caller keeps its own user-facing error string by wrapping with `%w` (or,
  for the CLI, keeps its existing friendlier message).

## Commands you will need

| Purpose   | Command                                              | Expected            |
|-----------|------------------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`                       | exit 0              |
| Vet       | `go vet ./...`                                        | exit 0              |
| Test pkg  | `go test ./internal/tasks/... ./internal/cli/... ./internal/web/... ./internal/tools/...` | PASS |
| Full test | `go test ./...`                                       | all pass            |
| gofmt     | `gofmt -l internal/tasks internal/cli internal/web internal/tools` | prints nothing |

> If `go test ./...` fails the link step with "No space left on device", set
> `TMPDIR=/home/alex/.cache/go-tmp` and retry.

## Scope

**In scope**:
- `internal/tasks/tasks.go` (add `Get`)
- `internal/cli/task.go` (route `findTask` through `tasks.Get`)
- `internal/web/tasks.go` (route `loadTaskNode` through `tasks.Get`)
- `internal/tools/tasks.go` (route the two inline sites + `findTask` through `tasks.Get`)

**Out of scope** (do NOT touch):
- The `"nodes"` collection literal elsewhere — only the find+Hydrate **pairing**
  is being collapsed, not the collection name.
- `tasks.Hydrate` / `tasks.hydrate` — unchanged; `Get` calls `hydrate`.
- The JSON-decoding inside `tools.findTask` — it stays in the tool layer.
- **The list-loop hydrations that are NOT single-id finds** —
  `internal/cli/task.go` `taskListCmd` "all" scope (`tasks.Hydrate(r)` in a loop
  over `nodes.ListByTypeStatus`, ~line 111) and `internal/tools/tasks.go`
  `task_list`'s "all" branch (`tasks.Hydrate(r)` in a loop, ~line 166). These
  hydrate a LIST, not a single id; `tasks.Get` is the single-id seam, so leave
  these loops exactly as they are. (They use `ListByTypeStatus`, NOT
  `FindRecordById`, so the `FindRecordById` done-criterion still passes.)

## Git workflow

- Branch: `advisor/200-tasks-get-getter`
- Conventional-commit subject, e.g. `refactor(tasks): add tasks.Get and collapse find+Hydrate sites`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add `tasks.Get`

In `internal/tasks/tasks.go`, directly below `Hydrate` (after line 373), add:

```go
// Get loads a task node by id and hydrates its legacy field aliases. It is the
// single find-and-hydrate seam for callers (cli, web, tools) that hold a task
// id; collapses the FindRecordById("nodes", id)+Hydrate pairing behind the
// owning package. The collection name stays explicit here — Get hides the
// pairing, not the spine.
func Get(app core.App, id string) (*core.Record, error) {
	rec, err := app.FindRecordById("nodes", strings.TrimSpace(id))
	if err != nil {
		return nil, fmt.Errorf("tasks: no task %q: %w", id, err)
	}
	hydrate(rec)
	return rec, nil
}
```

Confirm `internal/tasks/tasks.go` already imports `fmt` and `strings` (it does —
used by `OpenTasks`, `matchTerms`, etc.).

**Verify**: `go build ./internal/tasks/...` → exit 0

### Step 2: Route `cli.findTask` through `tasks.Get`

In `internal/cli/task.go`, replace the body of `findTask` (lines 42–49) with:

```go
func findTask(app core.App, id string) (*core.Record, error) {
	rec, err := tasks.Get(app, id)
	if err != nil {
		return nil, fmt.Errorf("no task with id %q — check `task list`", id)
	}
	return rec, nil
}
```

(Keep the friendlier CLI message. After this, `strings` may become unused in
`cli/task.go` — check: `strings.TrimSpace` is used in `taskListCmd` line 116, so
`strings` stays. Run `go vet` to confirm.)

**Verify**: `go build ./internal/cli/...` → exit 0; `go vet ./internal/cli/...` → exit 0

### Step 3: Route `web.loadTaskNode` through `tasks.Get`

In `internal/web/tasks.go`, replace `loadTaskNode` (lines 19–26) with:

```go
// loadTaskNode fetches a task node by id from the nodes collection and hydrates it.
func (h *handlers) loadTaskNode(id string) (*core.Record, error) {
	return tasks.Get(h.app, id)
}
```

(The four callers — `taskCard`, `taskTransition`, `taskEdit` — already wrap the
error via `h.cardError(e, err)`, so the message change is benign.)

**Verify**: `go build ./internal/web/...` → exit 0

### Step 4: Route the three `tools` sites through `tasks.Get`

In `internal/tools/tasks.go`:

1. **`findTask` (line 447)** — keep the JSON decode, swap the load:
   ```go
   func findTask(app core.App, argsJSON string) (*core.Record, error) {
       var args struct {
           ID string `json:"id"`
       }
       if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
           return nil, fmt.Errorf("bad arguments: %w", err) // keep existing message
       }
       return tasks.Get(app, args.ID)
   }
   ```
   Preserve whatever the current decode/error wording is — only the
   `FindRecordById(...)+Hydrate` lines (currently 455–459) collapse to
   `return tasks.Get(app, args.ID)`.

2. **Inline site at line 243** (snooze tool) — replace the
   `FindRecordById("nodes", strings.TrimSpace(args.ID))` + error block + the
   following `tasks.Hydrate(rec)` (line 247) with:
   ```go
   rec, err := tasks.Get(app, args.ID)
   if err != nil {
       return /* keep the existing tool's error return shape */, nil
   }
   ```
   Match the surrounding tool's existing error-return convention (these tools
   return a plain-text error string + nil error so the model self-corrects —
   keep that shape; just swap the load).

3. **Inline site at line 414** — same treatment: replace the
   `FindRecordById("nodes", args.ID)` block + `tasks.Hydrate(rec)` (line 418)
   with a `tasks.Get(app, args.ID)` call, keeping the tool's existing
   error-return shape.

After this, check whether `strings` is still used in `tools/tasks.go` (it is —
`ParseDue` and others use it; confirm with `go vet`). Do not remove imports the
compiler still needs.

**Verify**:
- `grep -n "FindRecordById(\"nodes\"" internal/tools/tasks.go` → no matches
- `grep -n "tasks.Hydrate" internal/tools/tasks.go` → exactly ONE match remains —
  the list-loop hydration in the `task_list` "all" branch (~line 166); the three
  by-id sites (243, 414, 455) are gone
- `go build ./internal/tools/...` → exit 0; `go vet ./internal/tools/...` → exit 0

### Step 5: Full verification

**Verify**:
- `gofmt -l internal/tasks internal/cli internal/web internal/tools` → prints nothing
- `go vet ./...` → exit 0
- `go test ./internal/tasks/... ./internal/cli/... ./internal/web/... ./internal/tools/...` → PASS
- `go test ./...` → all pass

## Test plan

- Add a focused unit test `TestGet` in `internal/tasks/tasks_test.go` (follow the
  existing table/`tests.NewTestApp` pattern already used in that file):
  - happy path: create a task via `tasks.Create`, then `tasks.Get(app, rec.Id)`
    returns a record whose `GetString("status")` == "open" and whose `title`
    matches (proves hydration ran).
  - error path: `tasks.Get(app, "nonexistent")` returns a non-nil error.
  - If `internal/tasks/tasks_test.go` does not exist, model the new test on
    another store-backed test in the repo (e.g.
    `internal/tasks/*_test.go` if present, else `internal/nodes/nodes_test.go`)
    using `store`/`tests` helpers for a temp-dir app.
- The five collapsed call sites are already covered by existing cli/web/tools
  task tests; they should pass unchanged.
- Verification: `go test ./internal/tasks/...` → PASS including `TestGet`.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `grep -rn "func Get" internal/tasks/tasks.go` returns one match
- [ ] `grep -rn "FindRecordById(\"nodes\"" internal/cli/task.go internal/web/tasks.go internal/tools/tasks.go` returns no matches
- [ ] `grep -rn "tasks.Hydrate" internal/cli/task.go internal/web/tasks.go internal/tools/tasks.go` returns ONLY the two documented list-loop sites (cli `taskListCmd` "all" scope ~line 111; tools `task_list` "all" branch ~line 166) — `web/tasks.go` has none, and neither remaining call is paired with a `FindRecordById`
- [ ] `go test ./...` exits 0; `TestGet` exists and passes
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The inline site at `tools/tasks.go:243` or `:414` does something extra between
  the `FindRecordById` and the `Hydrate` (e.g. checks `rec.GetString("type")`)
  that `tasks.Get` would skip — preserve that check; if it doesn't fit the
  simple swap, report it.
- Any caller relied on the find returning a non-hydrated record (i.e. read a raw
  node field that hydration overwrites) — `tasks.Get` always hydrates.
- A web or tools task test starts failing after the swap — the error-handling
  shape diverged; report which test and why.

## Maintenance notes

- `tasks.Get` is now the one find-and-hydrate seam for task nodes. Any future
  change to task loading (a soft-delete filter, a per-type guard) goes here once.
- Reviewer: confirm each tool kept its own plain-text error-return convention
  (model self-corrects on tool errors — do not convert these to hard `error`
  returns) and that the CLI kept its `check \`task list\`` hint.
