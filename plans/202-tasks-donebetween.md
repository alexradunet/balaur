# Plan 202: Add `tasks.DoneBetween` and move the done-task derivation out of `life.Range`

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 07fb4d6..HEAD -- internal/tasks/tasks.go internal/life/day.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none (independent of #200, though both touch `tasks`)
- **Category**: tech-debt
- **Planned at**: commit `07fb4d6`, 2026-06-26

## Why this matters

`internal/life/day.go`'s `Range` aggregator re-implements `tasks`' done-task
semantics by hand: it loads raw task nodes via `nodes.ListByTypeStatus`,
hydrates each with `tasks.Hydrate`, then re-derives "this task is done in range"
(`status == "done"` and `done_at` within `[start, end)`). That knowledge —
how task completion is recorded and what "done" means — belongs to the `tasks`
package. If completion recording changes (e.g. a `completed_at` rename, or a
new terminal status), `life` breaks silently because it owns a copy of the rule.

This plan adds `tasks.DoneBetween(app, start, end)` — the symmetric sibling of
the existing `tasks.OpenTasks` — and has `life.Range` call it for the task half.
The completion-**entries** half stays in `life` (those are `entries`-collection
rows, not task nodes, and `life` rightly owns assembling the cross-source
`RangeData`).

## Current state

`internal/life/day.go`, the `Range` function (lines 77–113). The task-half block
to move is lines 87–100:

```go
func Range(app core.App, start, end time.Time) (RangeData, error) {
	data := RangeData{}

	// Logged measures: type=measure nodes whose noted_at falls in [start, end).
	logged, err := listMeasuresInRange(app, start, end)
	if err != nil {
		return data, fmt.Errorf("range logged query: %w", err)
	}
	data.Logged = logged

	// Done tasks: active type=task nodes with status=done and done_at in range.
	if all, err2 := nodes.ListByTypeStatus(app, "task", nodes.StatusActive); err2 == nil {
		for _, r := range all {
			tasks.Hydrate(r)
			if r.GetString("status") != "done" {
				continue
			}
			doneAt := r.GetDateTime("done_at").Time()
			if doneAt.IsZero() || doneAt.Before(start) || !doneAt.Before(end) {
				continue
			}
			data.Done = append(data.Done, r)
		}
	}

	// Completion entries in range. Limit 0 (unlimited): a month/quarter/year can
	// hold far more than a single day's worth.
	recs, err := app.FindRecordsByFilter("entries",
		"kind = 'completion' && noted_at >= {:s} && noted_at < {:e}", "noted_at", 0, 0,
		dbx.Params{"s": store.PBTime(start), "e": store.PBTime(end)})
	if err != nil {
		return data, fmt.Errorf("range completions query: %w", err)
	}
	data.Done = append(data.Done, recs...)

	return data, nil
}
```

`internal/life/day.go` imports (lines 3–14) include both `nodes` and `tasks`.
**`nodes` is used in this file ONLY at line 88** (`nodes.ListByTypeStatus` /
`nodes.StatusActive`) — confirm with `grep -n "nodes\." internal/life/day.go`.
Removing the task-half block orphans the `nodes` import in `day.go`. `tasks`
stays (the file will now call `tasks.DoneBetween`).

The existing sibling to mirror — `internal/tasks/tasks.go:289`:

```go
// OpenTasks returns open tasks, optionally narrowed by LIKE terms over
// title and notes (ANDed — each term must match), due-ascending with
// someday items (empty due) first.
func OpenTasks(app core.App, terms []string) ([]*core.Record, error) {
	recs, err := nodes.ListByTypeStatus(app, "task", nodes.StatusActive)
	if err != nil {
		return nil, fmt.Errorf("tasks: loading task nodes: %w", err)
	}
	var out []*core.Record
	for _, r := range recs {
		hydrate(r)
		if nodes.PropString(r, "state") != "open" {
			continue
		}
		// ...
		out = append(out, r)
	}
	// ...
	return out, nil
}
```

Note: the `tasks` package uses the unexported `hydrate` internally (the exported
`Hydrate` just calls it). `tasks` already imports `nodes`, `fmt`, `time`,
`core`.

> **Behavior note**: the original task-half silently ignores a load error
> (`if all, err2 := ...; err2 == nil`), while the completions query directly
> below it propagates its error. This plan makes the task half **propagate** its
> error too (consistent with the completions half and with `OpenTasks`). This is
> a deliberate, minor behavior change: on a task-node load failure `Range` now
> returns an error instead of partial data. Call it out in the commit message.

## Commands you will need

| Purpose   | Command                                         | Expected            |
|-----------|-------------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`                  | exit 0              |
| Vet       | `go vet ./...`                                   | exit 0              |
| Test pkg  | `go test ./internal/tasks/... ./internal/life/...` | PASS             |
| Full test | `go test ./...`                                  | all pass            |
| gofmt     | `gofmt -l internal/tasks internal/life`          | prints nothing      |

> If `go test ./...` fails the link step with "No space left on device", set
> `TMPDIR=/home/alex/.cache/go-tmp` and retry.

## Scope

**In scope**:
- `internal/tasks/tasks.go` (add `DoneBetween`)
- `internal/life/day.go` (call `tasks.DoneBetween`; drop the orphaned `nodes` import)

**Out of scope** (do NOT touch):
- The completion-**entries** query in `Range` (lines 102–110) — stays in `life`;
  those are `entries`-collection rows, not task nodes.
- `life.Day` (lines 30–65) and `listMeasuresInRange` — unchanged.
- `tasks.OpenTasks` — unchanged; `DoneBetween` is a new sibling, not a refactor
  of it.

## Git workflow

- Branch: `advisor/202-tasks-donebetween`
- Conventional-commit subject, e.g.
  `refactor(tasks,life): move done-task derivation behind tasks.DoneBetween`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add `tasks.DoneBetween`

In `internal/tasks/tasks.go`, add directly below `OpenTasks` (after its closing
brace at line 311):

```go
// DoneBetween returns active task nodes completed within [start, end): status
// "done" with done_at in range. The symmetric sibling of OpenTasks — it owns the
// "what counts as a done task" rule so cross-domain aggregators (life) don't
// re-derive it. Records are returned hydrated (legacy field aliases set).
func DoneBetween(app core.App, start, end time.Time) ([]*core.Record, error) {
	recs, err := nodes.ListByTypeStatus(app, "task", nodes.StatusActive)
	if err != nil {
		return nil, fmt.Errorf("tasks: loading task nodes: %w", err)
	}
	var out []*core.Record
	for _, r := range recs {
		hydrate(r)
		if r.GetString("status") != "done" {
			continue
		}
		doneAt := r.GetDateTime("done_at").Time()
		if doneAt.IsZero() || doneAt.Before(start) || !doneAt.Before(end) {
			continue
		}
		out = append(out, r)
	}
	return out, nil
}
```

**Verify**: `go build ./internal/tasks/...` → exit 0

### Step 2: Call it from `life.Range`; drop the `nodes` import

In `internal/life/day.go`, replace the task-half block (lines 87–100) with:

```go
	// Done tasks: completed task nodes with done_at in range. tasks owns the
	// done-task rule (see tasks.DoneBetween) so this aggregator doesn't re-derive it.
	done, err := tasks.DoneBetween(app, start, end)
	if err != nil {
		return data, fmt.Errorf("range done-tasks query: %w", err)
	}
	data.Done = done
```

Then remove `"github.com/alexradunet/balaur/internal/nodes"` from the import
block (now orphaned — verify with `grep -n "nodes\." internal/life/day.go`
returning nothing). Keep `tasks`, `recap`, `store`, `dbx`, `fmt`, `time`, `core`
(all still used in `day.go`). Run `gofmt`.

**Verify**:
- `grep -n "nodes\." internal/life/day.go` → no matches
- `grep -n "internal/nodes" internal/life/day.go` → no matches
- `go build ./internal/life/...` → exit 0
- `go vet ./internal/life/...` → exit 0

### Step 3: Full verification

**Verify**:
- `gofmt -l internal/tasks internal/life` → prints nothing
- `go vet ./...` → exit 0
- `go test ./internal/tasks/... ./internal/life/...` → PASS
- `go test ./...` → all pass

## Test plan

- Add `TestDoneBetween` in `internal/tasks/tasks_test.go` (or the nearest
  store-backed task test file), modeled on the existing `OpenTasks`/`Bucket`
  tests if present:
  - Create three tasks; mark one done "yesterday", one done "inside the window",
    leave one open. Assert `DoneBetween(app, windowStart, windowEnd)` returns
    exactly the one completed inside the window (and that it is hydrated:
    `GetString("status") == "done"`).
  - Boundary: a task with `done_at == end` is **excluded** (`!doneAt.Before(end)`),
    one with `done_at == start` is **included**.
- The `life` package's existing `Range`/`Day` tests should pass unchanged
  (same records selected). If `internal/life` has a `Range` test that asserts
  partial data on a forced error, update it for the new propagate-error
  behavior; otherwise leave it.
- Verification: `go test ./internal/tasks/... ./internal/life/...` → PASS.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `grep -n "func DoneBetween" internal/tasks/tasks.go` returns one match
- [ ] `grep -n "internal/nodes" internal/life/day.go` returns no matches
- [ ] `go test ./...` exits 0; `TestDoneBetween` exists and passes
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/README.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- `internal/life/day.go` uses `nodes` somewhere OTHER than the task-half block
  (the `grep -n "nodes\."` after removal is non-empty) — do not delete the
  import; report it.
- Removing the task-half block reveals `tasks` or `store` or `dbx` is no longer
  used in `day.go` — only `nodes` should become orphaned; if another import
  orphans too, recheck the excerpt match.
- A `life` test asserts the old partial-data-on-error behavior and you cannot
  tell whether to update it — report rather than guess.

## Maintenance notes

- `tasks.DoneBetween` is now the one place that knows "a done task is status=done
  with done_at in range." A future completion-recording change updates it once;
  `life` and any other aggregator inherit the fix.
- Reviewer: confirm the half-open interval is preserved exactly (`done_at >=
  start && done_at < end`), and that the deliberate error-propagation change in
  `Range` is acceptable (it matches the completions query directly below it).
- The completion-**entries** half intentionally stays in `life.Range` — it reads
  the `entries` collection, which is not a task-node concern.
