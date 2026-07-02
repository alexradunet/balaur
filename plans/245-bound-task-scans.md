# Plan 245: Push the task-state filter into SQL so the chat hot path and minute-crons stop scanning every task ever created

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/README.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- internal/tasks/tasks.go internal/tasks/nudge.go internal/tasks/tasks_test.go internal/tasks/nudge_test.go .tours/13-companion-domain.tour`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: M
- **Risk**: MED
- **Depends on**: none
- **Category**: perf
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

Completed and dropped one-off tasks keep their node `status` column at
`"active"` forever — `tasks.Done` and `tasks.Drop` only flip the JSON prop
`props.state` to `"done"`/`"dropped"`. So the set of "active task nodes"
grows monotonically with every task the owner ever finishes. That full set
is loaded, hydrated, and filtered **in Go** on every chat turn (via
`tasks.TodayBlock`, called by the shared turn pipeline), by the nudger
**every minute**, and by the briefing cron every minute until the day's
briefing fires. At today's scale this is negligible — the point is not a
measured slowdown, but that it is the one monotonically-growing full scan
sitting on the hot path. PocketBase filter expressions can reach into JSON
columns (`props.state = 'open'`), so the state filter can move into SQL and
the scan set becomes "currently open tasks" instead of "all tasks ever".
No lifecycle semantics change: done tasks continue to sit at node
`status = "active"` with `props.state = "done"` — that is settled design,
not part of this plan.

## Current state

Relevant files (all paths repo-relative):

- `internal/tasks/tasks.go` — task domain package: Create/Update/Done/Drop/
  Snooze, plus the three loaders this plan changes (`OpenTasks`,
  `DoneBetween`) and the `hydrate` helper.
- `internal/tasks/nudge.go` — minute-cron nudger; contains `DueForNudge`,
  the third loader this plan changes.
- `internal/tasks/briefing.go` — morning briefing + `TodayBlock`; calls
  `OpenTasks` and therefore inherits the fix with **no edit** (see Scope).
- `internal/nodes/nodes.go` — generic node access; `ListByTypeStatus` is the
  unbounded scan the three loaders currently share. It stays **unchanged**
  (other callers rely on the full set).
- `internal/tasks/tasks_test.go`, `internal/tasks/nudge_test.go` — existing
  tests; new tests land here.
- `.tours/13-companion-domain.tour` — a code tour with a step anchored to
  `internal/tasks/nudge.go` line 88 (`func Nudge`); this plan's edit to
  `DueForNudge` shifts that line, so the tour anchor needs a matching fix.

### The unbounded scan

`internal/nodes/nodes.go:228-233`:

```go
// ListByTypeStatus returns nodes of one type in one status, newest first.
func ListByTypeStatus(app core.App, typ, status string) ([]*core.Record, error) {
	return app.FindRecordsByFilter("nodes",
		"type = {:t} && status = {:s}", "-created", 0, 0,
		dbx.Params{"t": typ, "s": status})
}
```

### Why the scan set never shrinks

`Done` on a one-off only flips the JSON prop, `internal/tasks/tasks.go:200-210`:

```go
	if rule.IsZero() {
		props["state"] = "done"
		props["done_at"] = store.PBTime(now.UTC())
		rec.Set("props", props)
		dehydrate(rec)
		if err := app.Save(rec); err != nil {
			return DoneResult{}, fmt.Errorf("saving task: %w", err)
		}
		hydrate(rec)
		store.Audit(app, "tasks", "task.done", rec.Id, true, nil)
		return DoneResult{}, nil
	}
```

(`dehydrate` at `tasks.go:391-393` resets the node's real `status` column to
`nodes.StatusActive` before every save — so the column stays `"active"`
forever.) `Drop` is the same shape, `internal/tasks/tasks.go:273-287`,
setting `props["state"] = "dropped"`.

### The three loaders that scan everything and filter in Go

`internal/tasks/tasks.go:289-311` (`OpenTasks` — runs on **every chat turn**
via `tasks.TodayBlock(app, now)` at `internal/turn/turn.go:98`, and in the
briefing cron via `internal/tasks/briefing.go:53` and `:240`):

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
		if !matchTerms(r, terms) {
			continue
		}
		out = append(out, r)
	}
	// Sort: someday (empty due) first, then ascending due.
	sortByDue(out)
	return out, nil
}
```

`internal/tasks/tasks.go:313-335` (`DoneBetween` — feeds life aggregation:
its only caller is `internal/life/day.go:88` inside `life.Range`, which
powers day/period views and streak-adjacent rendering; its output set must
not change):

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

`internal/tasks/nudge.go:47-80` (`DueForNudge` — runs **every minute** from
the cron registered in `main.go:210-223`; note the `nudgeBatchLimit` cap
applies AFTER the full scan):

```go
// DueForNudge returns open tasks whose nudge should fire at now: due has
// passed, never fired (or re-armed by snooze/recurrence), snooze elapsed.
// Tasks are now type=task nodes; we load all active task nodes and filter in Go.
func DueForNudge(app core.App, now time.Time) ([]*core.Record, error) {
	recs, err := nodes.ListByTypeStatus(app, "task", nodes.StatusActive)
	if err != nil {
		return nil, fmt.Errorf("tasks: loading task nodes for nudge: %w", err)
	}
	var out []*core.Record
	for _, r := range recs {
		hydrate(r)
		if r.GetString("status") != "open" {
			continue
		}
		due := r.GetDateTime("due").Time()
		if due.IsZero() || !due.Before(now) {
			continue
		}
		if r.GetString("nudged_at") != "" {
			continue
		}
		if su := r.GetString("snoozed_until"); su != "" {
			suTime := r.GetDateTime("snoozed_until").Time()
			if !suTime.IsZero() && suTime.After(now) {
				continue
			}
		}
		out = append(out, r)
		if len(out) >= nudgeBatchLimit {
			break
		}
	}
	return out, nil
}
```

Important: `hydrate` (`tasks.go:418-450`) copies `props.state` into a
read-only `status` alias via `SetRaw` — so `r.GetString("status")` in the
loaders reads the workflow state (open/done/dropped), NOT the node's real
`status` column (which is always `"active"` here). The three checks above
are all the same "props.state equals X" test in disguise.

### The key assumption to prove first

PocketBase (this repo pins `github.com/pocketbase/pocketbase v0.39.3`,
`go.mod:12`) supports JSON dot-path selectors in filter expressions on
`json` fields — i.e. `props.state = 'open'` should work in
`app.FindRecordsByFilter`. Step 1 proves this empirically before anything
else changes. Be aware of a **stale comment** elsewhere in the package that
claims the opposite for a different prop, `internal/tasks/briefing.go:154`:

```go
		// noted_at lives in props (JSON); PB filters can't reach it, so filter in Go.
```

Do NOT take that comment as truth and do NOT edit it (out of scope —
surgical changes only); the Step 1 test is the arbiter. If Step 1 fails,
STOP (see STOP conditions) — the fallback design (a real column or an
archival status) is a bigger change that must be re-scoped by the plan
author.

### Other full-set task scans — deliberately untouched

These callers of `nodes.ListByTypeStatus(app, "task", nodes.StatusActive)`
want ALL workflow states (they render done/dropped too) and are out of
scope: `internal/cli/task.go:105`, `internal/tools/tasks.go:156`,
`internal/feature/taskcards/quests.go:142`,
`internal/feature/taskcards/taskscluster.go:32`,
`internal/feature/taskcards/questsfocus.go:41`.

### Conventions that apply here

- Errors are values: wrap with `fmt.Errorf("doing x: %w", err)`, return
  early, no panics in library code. Match the existing wrapping in the
  loaders quoted above.
- Tests use the standard `testing` package; PocketBase-dependent tests use
  the temp-dir app helper `storetest.NewApp(t)` — see the top of
  `internal/tasks/tasks_test.go:15-16` for the exact pattern. No
  `time.Sleep`-based synchronization. No assertion frameworks.
- No new writes are introduced, so the audit-after-save rule is not in
  play; do not add audit calls.
- A `PostToolUse` hook runs `gofmt -w` on edited Go files; still verify
  with `gofmt -l .` at the end.
- `.tours/` is a maintained artifact: `tours_test.go` fails when a tour
  references a missing file or out-of-range line. A shifted-but-in-range
  anchor does NOT fail the test but still falsifies the tour — Step 4 fixes
  the anchor explicitly.
- `internal/self/knowledge.md` (the binary's self-description) does NOT
  need updating: this change alters no user-visible capability or
  architecture, only the SQL shape of an internal query.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Full test gate (the merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all packages ok |
| Targeted tasks tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/tasks/ -run <Name> -count=1` | ok |
| Vet | `go vet ./...` | exit 0, no output |
| Format check | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tours lint (Step 4 only) | `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` | ok |

(The host `/tmp` is a small tmpfs; the Go linker OOMs there — always prefix
test commands with `TMPDIR=$HOME/.cache/go-tmp` as shown.)

## Suggested executor toolkit

- If the `go-standards` skill is available, invoke it before writing Go —
  it carries this repo's error-wrapping, testing, and PocketBase idioms.

## Scope

**In scope** (the only files you should modify):

- `internal/tasks/tasks.go` — add the shared state-filtered loader; switch
  `OpenTasks` and `DoneBetween` to it.
- `internal/tasks/nudge.go` — switch `DueForNudge` to it; fix its doc
  comment.
- `internal/tasks/tasks_test.go` — Step 1 capability test + equivalence
  tests for `OpenTasks`/`DoneBetween`.
- `internal/tasks/nudge_test.go` — equivalence test for `DueForNudge`.
- `.tours/13-companion-domain.tour` — only the `line` field of the step
  titled "13.5 — Nudge and briefing crons" (see Step 4).

(Plus the `plans/README.md` status row on completion, per the executor
instructions.)

**Out of scope** (do NOT touch, even though they look related):

- `internal/nodes/nodes.go` — `ListByTypeStatus`'s signature and behavior
  stay as-is; the callers listed under "Other full-set task scans" rely on
  the full set.
- Task lifecycle semantics — done/dropped tasks keep node
  `status = "active"` with `props.state` carrying the workflow state. No
  archival status, no schema/migration changes.
- `internal/tasks/briefing.go` — its `OpenTasks(app, nil)` calls (lines 53
  and 240) inherit the fix with zero edits; its `loggedYesterday` measure
  scan (and the stale comment at line 154) is a small separate follow-up,
  deferred.
- `internal/turn/`, `internal/life/`, `internal/cli/`, `internal/tools/`,
  `internal/feature/` — callers of the tasks package; their behavior must
  be unchanged, which the equivalence tests guarantee.

## Git workflow

- The executor runs in an isolated git worktree branched from
  `origin/main`; branch name `advisor/245-bound-task-scans`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/
  `test`/`chore`) — e.g. recent history: `fix(turn): cross-surface
  in-flight guard — one turn at a time on the master conversation`.
- Commit per logical unit with explicit pathspecs (the main checkout is
  shared by parallel agents — stage only your own files, e.g.
  `git add internal/tasks/tasks.go internal/tasks/tasks_test.go`).
- **NEVER push**; the reviewer merges.

## Steps

### Step 1: Prove the PB JSON-path filter works (capability test)

Add `TestPropsStateJSONFilter` to `internal/tasks/tasks_test.go`. It stays
in the suite permanently as the regression guard documenting the PocketBase
JSON-filter dependency (PocketBase is pre-1.0; a future PB bump that breaks
JSON dot-path filters must fail loudly here, not silently empty the task
lists).

Shape (model setup after `TestDoneBetween` at `tasks_test.go:436`):

```go
// TestPropsStateJSONFilter guards the PocketBase capability this package's
// loaders depend on: filter expressions reaching into the props json column
// via dot paths (props.state). If a PocketBase upgrade breaks this, the
// task loaders would silently return nothing — fail here instead.
func TestPropsStateJSONFilter(t *testing.T) {
	app := storetest.NewApp(t)

	if _, err := Create(app, CreateOpts{Title: "Open one"}); err != nil {
		t.Fatalf("create open: %v", err)
	}
	recDone, err := Create(app, CreateOpts{Title: "Done one"})
	if err != nil {
		t.Fatalf("create done: %v", err)
	}
	if _, err := Done(app, recDone, time.Now()); err != nil {
		t.Fatalf("done: %v", err)
	}

	open, err := app.FindRecordsByFilter("nodes",
		"type = 'task' && status = 'active' && props.state = 'open'",
		"-created", 0, 0, nil)
	if err != nil {
		t.Fatalf("props.state=open filter: %v", err)
	}
	if len(open) != 1 || open[0].GetString("title") != "Open one" {
		t.Fatalf("props.state=open filter: got %d records, want exactly [Open one]", len(open))
	}

	done, err := app.FindRecordsByFilter("nodes",
		"type = 'task' && status = 'active' && props.state = 'done'",
		"-created", 0, 0, nil)
	if err != nil {
		t.Fatalf("props.state=done filter: %v", err)
	}
	if len(done) != 1 || done[0].GetString("title") != "Done one" {
		t.Fatalf("props.state=done filter: got %d records, want exactly [Done one]", len(done))
	}
}
```

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/tasks/ -run TestPropsStateJSONFilter -count=1 -v`
→ `--- PASS: TestPropsStateJSONFilter` and `ok`. If it FAILS (filter error,
or wrong counts), this is a STOP condition — do not proceed to Step 2.

Commit: `test(tasks): prove props.state json filter works on this PB version`

### Step 2: Pin current behavior with equivalence tests (before any loader change)

These tests encode the CURRENT behavior of the three loaders against a
mixed-state seed, so Step 3/4's loader swap is provably a no-op on output.
Write them now and run them against the **unmodified** loaders — they must
pass BEFORE the swap.

2a. Add `TestOpenTasksAndDoneBetweenAcrossStates` to
`internal/tasks/tasks_test.go` (use `storetest.NewApp(t)`; seed via the
package's own API only):

- Seed six tasks: `Create` a someday open task (no due), an open task due
  in +2h, an open task due -10m (overdue), an open task due -10m then
  `Snooze(app, rec, now.Add(time.Hour))`, a task completed via
  `Done(app, rec, doneTime)` at a fixed `doneTime`, and a task closed via
  `Drop(app, rec)`. Give each a distinct title.
- Assert `OpenTasks(app, nil)` returns exactly the 4 open titles (someday,
  future, overdue, snoozed), with the someday task FIRST (`sortByDue` puts
  zero-due first), and that each returned record is hydrated:
  `rec.GetString("status") == "open"`.
- Assert `OpenTasks(app, []string{<a term matching exactly one title>})`
  returns exactly that one task (term narrowing survives the swap).
- Assert `DoneBetween(app, start, end)` with a window containing `doneTime`
  returns exactly the completed task, hydrated
  (`GetString("status") == "done"`), and that the dropped task appears in
  neither loader's output.

2b. Add `TestDueForNudgeAcrossStates` to `internal/tasks/nudge_test.go`
(model after `TestNudgeFiresOnceAndMarks` at `nudge_test.go:74`):

- Seed: an open task due -10m (the only expected hit), an open task due
  +2h, an open someday task (no due), an open task due -10m that was then
  `Snooze`d until now+1h, a task due -10m completed via `Done`, and a task
  due -10m closed via `Drop`.
- Assert `DueForNudge(app, now)` returns exactly ONE record with the
  expected title.

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/tasks/ -run 'TestOpenTasksAndDoneBetweenAcrossStates|TestDueForNudgeAcrossStates' -count=1 -v`
→ both PASS against the current, unmodified loaders. If either fails, your
test's expectation misreads current behavior — fix the TEST (re-read the
loader code quoted in "Current state"), never the loader; if it still fails,
STOP.

Commit: `test(tasks): pin loader behavior across task states before SQL-filter swap`

### Step 3: Add the shared state-filtered loader; switch OpenTasks and DoneBetween

In `internal/tasks/tasks.go`, insert a new unexported loader immediately
above `OpenTasks` (i.e. after `Drop`, around line 288 — placing it here
keeps the tour anchors at `tasks.go:30` and `tasks.go:189` untouched):

```go
// taskRecordsByState loads active task nodes in one workflow state,
// pushing the props.state filter into SQL so the scan set is the tasks
// currently in that state — not every task ever created (done/dropped
// one-offs keep node status "active" forever; only props.state moves).
// PocketBase filter expressions reach into json columns via dot paths;
// TestPropsStateJSONFilter guards that dependency. Records come back
// hydrated, newest first (the order the old full scan returned).
func taskRecordsByState(app core.App, state string) ([]*core.Record, error) {
	recs, err := app.FindRecordsByFilter("nodes",
		"type = 'task' && status = {:a} && props.state = {:st}",
		"-created", 0, 0,
		dbx.Params{"a": nodes.StatusActive, "st": state})
	if err != nil {
		return nil, fmt.Errorf("tasks: loading %s task nodes: %w", state, err)
	}
	for _, r := range recs {
		hydrate(r)
	}
	return recs, nil
}
```

(`dbx` is already imported in `tasks.go` — see `tasks.go:8`.)

Then rewrite the two loader bodies to use it, dropping their per-record
`hydrate` + state checks (the loader now does both):

`OpenTasks` becomes:

```go
func OpenTasks(app core.App, terms []string) ([]*core.Record, error) {
	recs, err := taskRecordsByState(app, "open")
	if err != nil {
		return nil, err
	}
	var out []*core.Record
	for _, r := range recs {
		if !matchTerms(r, terms) {
			continue
		}
		out = append(out, r)
	}
	// Sort: someday (empty due) first, then ascending due.
	sortByDue(out)
	return out, nil
}
```

`DoneBetween`'s body becomes:

```go
	recs, err := taskRecordsByState(app, "done")
	if err != nil {
		return nil, err
	}
	var out []*core.Record
	for _, r := range recs {
		doneAt := r.GetDateTime("done_at").Time()
		if doneAt.IsZero() || doneAt.Before(start) || !doneAt.Before(end) {
			continue
		}
		out = append(out, r)
	}
	return out, nil
```

Keep both functions' doc comments; in `DoneBetween`'s comment the phrase
"status \"done\"" stays accurate (it refers to the hydrated alias). Do NOT
touch anything else in the file. The Go-side `matchTerms` and the
`done_at`-range filters stay in Go on purpose (KISS — they are cheap over
the now-small result set; `done_at` range could be pushed down later but is
not needed for correctness or this plan's goal).

**Verify**:
`TMPDIR=$HOME/.cache/go-tmp go test ./internal/tasks/ -count=1` → ok (all
existing tests plus Steps 1–2 tests pass unchanged).
Then `TMPDIR=$HOME/.cache/go-tmp go test ./internal/life/ ./internal/feature/... ./internal/cli/ ./internal/tools/ -count=1` → ok
(the DoneBetween/OpenTasks consumers).

Commit: `perf(tasks): push props.state filter into SQL for OpenTasks and DoneBetween`

### Step 4: Switch DueForNudge; fix its comment and the tour anchor

In `internal/tasks/nudge.go`, rewrite `DueForNudge` to use the shared
loader (same package — no import changes; `nodes` stays imported for
`nodes.Props` in `Nudge` at line 116):

```go
// DueForNudge returns open tasks whose nudge should fire at now: due has
// passed, never fired (or re-armed by snooze/recurrence), snooze elapsed.
// The open-state filter runs in SQL (taskRecordsByState); the due/fired/
// snooze checks stay in Go over the small open set.
func DueForNudge(app core.App, now time.Time) ([]*core.Record, error) {
	recs, err := taskRecordsByState(app, "open")
	if err != nil {
		return nil, fmt.Errorf("tasks: loading task nodes for nudge: %w", err)
	}
	var out []*core.Record
	for _, r := range recs {
		due := r.GetDateTime("due").Time()
		if due.IsZero() || !due.Before(now) {
			continue
		}
		if r.GetString("nudged_at") != "" {
			continue
		}
		if su := r.GetString("snoozed_until"); su != "" {
			suTime := r.GetDateTime("snoozed_until").Time()
			if !suTime.IsZero() && suTime.After(now) {
				continue
			}
		}
		out = append(out, r)
		if len(out) >= nudgeBatchLimit {
			break
		}
	}
	return out, nil
}
```

The old third comment line ("Tasks are now type=task nodes; we load all
active task nodes and filter in Go.") is now false — the replacement above
drops it.

This edit shifts lines below `DueForNudge`. The tour
`.tours/13-companion-domain.tour` has a step (title
`"13.5 — Nudge and briefing crons"`) anchored to
`internal/tasks/nudge.go` line 88, which today is `func Nudge(`. After
editing, find the new line: `grep -n "func Nudge(" internal/tasks/nudge.go`
— if it is no longer 88, edit ONLY that step's `"line"` value in
`.tours/13-companion-domain.tour` to the new number. The step's prose
("excluded by `DueForNudge` — no separate state table") remains true —
change no description text.

**Verify** (three commands):
1. `TMPDIR=$HOME/.cache/go-tmp go test ./internal/tasks/ -count=1` → ok.
2. `grep -rn "ListByTypeStatus" internal/tasks/` → NO matches (all three
   loaders now go through `taskRecordsByState`).
3. `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok.

Commit: `perf(tasks): DueForNudge queries open tasks via SQL state filter; fix tour anchor`
(stage `internal/tasks/nudge.go` and `.tours/13-companion-domain.tour`)

### Step 5: Full gate

Run, in order:

1. `gofmt -l .` → empty output
2. `go vet ./...` → exit 0
3. `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output, exit 0
4. `CGO_ENABLED=0 go build ./...` → exit 0
5. `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0, all ok
6. `git status --porcelain` → only the in-scope files (and the
   `plans/README.md` row update)

Update the plan 245 status row in `plans/README.md` (unless the dispatching
reviewer maintains the index).

Commit any remaining unstaged in-scope file; final commit history should be
3–4 small conventional commits.

## Test plan

- **New tests** (all in `internal/tasks`, std `testing`, temp-dir app via
  `storetest.NewApp(t)` — pattern: `internal/tasks/tasks_test.go:15-16`):
  - `TestPropsStateJSONFilter` (`tasks_test.go`) — the PB JSON dot-path
    filter capability guard: open + done seed, both direction filters
    return exactly one correct row each. Written in Step 1, kept forever.
  - `TestOpenTasksAndDoneBetweenAcrossStates` (`tasks_test.go`) —
    equivalence across open/someday/snoozed/done/dropped: exact result
    sets, someday-first ordering, term narrowing, hydration
    (`GetString("status")` alias), dropped excluded everywhere.
  - `TestDueForNudgeAcrossStates` (`nudge_test.go`) — exactly one hit from
    a seed of overdue/future/someday/snoozed/done/dropped.
- **Sequencing is the safety mechanism**: equivalence tests pass against
  the OLD loaders (Step 2) before the swap (Steps 3–4), proving the SQL
  filter changes the query shape, not the results.
- **Existing regression coverage that must stay green**: `TestDoneBetween`
  (`tasks_test.go:436` — window boundaries), the `OpenTasks`
  bucket/term tests (`tasks_test.go:407-433`), and
  `TestNudgeFiresOnceAndMarks` + the DueForNudge-after-nudge check
  (`nudge_test.go:223-235`).
- Verification: `TMPDIR=$HOME/.cache/go-tmp go test ./internal/tasks/ -count=1`
  → ok, including the 3 new tests; then the full gate (Step 5).
- **No benchmarks** — v1 scale makes the win unmeasurable; the change is
  justified by growth shape, not a number. Measure nothing.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0
- [ ] `go vet ./...` → exit 0; `gofmt -l .` → empty;
      `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → exit 0, no output;
      `CGO_ENABLED=0 go build ./...` → exit 0
- [ ] `grep -rn "ListByTypeStatus" internal/tasks/` → no matches
- [ ] `grep -c "func taskRecordsByState" internal/tasks/tasks.go` → 1
- [ ] `grep -c "func TestPropsStateJSONFilter" internal/tasks/tasks_test.go` → 1;
      `grep -c "func TestOpenTasksAndDoneBetweenAcrossStates" internal/tasks/tasks_test.go` → 1;
      `grep -c "func TestDueForNudgeAcrossStates" internal/tasks/nudge_test.go` → 1
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok
- [ ] `git status --porcelain` shows changes ONLY in: `internal/tasks/tasks.go`,
      `internal/tasks/nudge.go`, `internal/tasks/tasks_test.go`,
      `internal/tasks/nudge_test.go`, `.tours/13-companion-domain.tour`,
      `plans/README.md`
- [ ] `plans/README.md` status row for 245 updated (unless the reviewer
      maintains the index)

## STOP conditions

Stop and report back (do not improvise) if:

- **Step 1's capability test fails** — `props.state = '...'` filters error
  out or return wrong counts on PocketBase v0.39.3. The fallback (a real
  indexed column for task state, or an archival node status) is a
  materially bigger design change the plan author must re-scope. Do NOT
  attempt it; report the exact failure output.
- **Step 2's equivalence tests fail against the unmodified loaders** after
  one careful re-read of the loader code — the plan's description of
  current behavior is wrong, or the code drifted.
- The drift check shows any in-scope file changed since `077318a` AND the
  "Current state" excerpts no longer match the live code (in particular:
  `OpenTasks` at `tasks.go:292`, `DoneBetween` at `tasks.go:317`,
  `DueForNudge` at `nudge.go:50` still calling
  `nodes.ListByTypeStatus(app, "task", nodes.StatusActive)`).
- After the Step 3/4 swap, any pre-existing test in `internal/tasks/`,
  `internal/life/`, or `internal/feature/...` fails and one fix attempt on
  YOUR new loader (never on the tests or the consumers) does not restore
  green — a semantic difference between the SQL filter and the old Go
  filter has surfaced (e.g. tasks whose props lack a `state` key entirely;
  the old code skipped them in `OpenTasks` via
  `nodes.PropString(r, "state") != "open"`, and the SQL filter must skip
  them too — if reality disagrees, report it).
- The fix appears to require touching `internal/nodes/nodes.go`, any
  migration, or any file in the out-of-scope list.

## Maintenance notes

- **What this deliberately does not fix**: done/dropped tasks still
  accumulate forever as `status="active"` nodes, and the full-set scans in
  `internal/cli/task.go:105`, `internal/tools/tasks.go:156`, and the three
  `internal/feature/taskcards/` callers still load everything (they render
  all states, and they are owner-initiated views, not per-turn/per-minute
  paths). If scale ever actually bites, the next step is an archival node
  status or a real state column — a schema-level change, not more JSON
  filters.
- **`TestPropsStateJSONFilter` is the PB-upgrade canary.** PocketBase is
  pre-1.0 with breaking-change precedent; any PB bump that changes JSON
  filter semantics fails this test instead of silently emptying the
  owner's task lists, nudges, and briefings. Reviewers of future PB
  upgrades should treat a failure here as "the task loaders are broken",
  not "flaky test".
- **Reviewer focus**: the equivalence tests' seeds (Step 2) are the whole
  safety argument — check they cover open/someday/snoozed/done/dropped and
  that they were committed BEFORE the loader swap (commit order in the
  branch history shows this).
- **Deferred follow-ups**: (a) `internal/tasks/briefing.go:154`'s stale
  "PB filters can't reach it" comment and the `loggedYesterday` measure
  scan it excuses — small, separate change; (b) pushing `DoneBetween`'s
  `done_at` range into SQL — unnecessary while the done-set filter alone
  bounds the scan.
