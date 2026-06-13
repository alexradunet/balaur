# Plan 046: Batch streak computation — one completions query per render, not one per task

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md` — unless a reviewer dispatched you and told you they
> maintain the index.
>
> **Drift check (run first)**: `git diff --stat dd9e60b..HEAD -- internal/tasks/streak.go internal/tasks/briefing.go internal/web/life.go internal/tasks/streak_test.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED — streak boundary semantics must not move; the existing
  `Streak()` unit tests and a new parity test are the net
- **Depends on**: none
- **Category**: perf
- **Planned at**: commit `dd9e60b`, 2026-06-12

## Why this matters

`tasks.TodayBlock` runs on **every chat turn** (`internal/turn/turn.go:95`)
to ground the model in the owner's day. For each overdue/today task with a
recurrence rule (up to 5+8 after caps), it calls `StreakFor`, which issues
one `entries` query fetching **all** completion rows for that task
(limit 0 = unbounded). A year-old daily habit is ~365 rows; several habits
mean thousands of rows re-fetched per turn, forever, on the hot path. The
same pattern repeats in the morning briefing and (unbounded by caps) on the
`/life` page. One batched query per render keeps the exact streak
semantics while removing the N+1.

## Current state

### The per-task query — `internal/tasks/streak.go:53-68, 90-104`

```go
// CompletionDays returns the distinct local days with a completion for the
// task, ascending, noon-anchored.
func CompletionDays(app core.App, taskID string, loc *time.Location) ([]time.Time, error) {
	recs, err := app.FindRecordsByFilter("entries",
		"kind = 'completion' && task = {:t}", "noted_at", 0, 0, dbx.Params{"t": taskID})
	if err != nil {
		return nil, err
	}
	var days []time.Time
	for _, r := range recs {
		d := noonOf(r.GetDateTime("noted_at").Time().In(loc))
		if len(days) == 0 || !days[len(days)-1].Equal(d) {
			days = append(days, d)
		}
	}
	return days, nil
}
```

```go
// StreakFor loads completions and computes the live streak for one task.
// Errors read as 0 — a missing streak must never block a briefing.
func StreakFor(app core.App, rec *core.Record, now time.Time) int {
	rule, err := Parse(rec.GetString("recur"))
	if err != nil || rule.IsZero() {
		return 0
	}
	days, err := CompletionDays(app, rec.Id, now.Location())
	if err != nil {
		return 0
	}
	return Streak(rule, days, now)
}
```

`Streak(rule, days, now)` is the pure function (same file) — it does not
change in this plan.

### Call site 1 — `internal/tasks/briefing.go:85-122` (hot path)

`dayLines(app, bk, now)` iterates `bk.Overdue` (cap `briefingOverdueCap`=5)
then `bk.Today` (total cap 5+8), calling `dayLine(app, r, now, overdue)`
per task; `dayLine` uses `app` for exactly one thing (line 113):

```go
	if rule, err := Parse(r.GetString("recur")); err == nil && !rule.IsZero() {
		if streak := StreakFor(app, r, now); streak > 1 {
			fmt.Fprintf(&b, " — habit, streak %d", streak)
		} else {
			b.WriteString(" — habit")
		}
	}
```

`dayLines` is shared by the briefing (`briefing.go:61`) and `TodayBlock`
(`briefing.go:206-212`, called from `internal/turn/turn.go:95` every turn).

### Call site 2 — `internal/web/life.go:76-88` (unbounded)

```go
	var habits []lifeHabitView
	if recs, err := tasks.OpenTasks(h.app, nil); err == nil {
		for _, r := range recs {
			rule, err := tasks.Parse(r.GetString("recur"))
			if err != nil || rule.IsZero() {
				continue
			}
			habits = append(habits, lifeHabitView{
				Title:     r.GetString("title"),
				Streak:    tasks.StreakFor(h.app, r, now),
				RecurLine: tasks.Describe(rule),
			})
		}
	}
```

Repo conventions: PocketBase filters use `{:param}` placeholders with
`dbx.Params` — **never** string-concatenated values; small pure functions;
table-driven tests in `internal/tasks/streak_test.go` (read it; your parity
test joins it). The repo rule says optimize hot paths with measurement in
hand — this plan's justification is the unbounded per-task row fetch on
the per-turn path, plus that the fix collapses to one query without
touching `Streak` semantics.

## Commands you will need

| Purpose   | Command                                   | Expected on success |
|-----------|-------------------------------------------|---------------------|
| Focused   | `go test ./internal/tasks/ ./internal/web/` | ok                |
| All tests | `go test ./...`                           | ok                  |
| Vet/fmt   | `go vet ./...` / `gofmt -l .`             | silent / empty      |
| Build     | `CGO_ENABLED=0 go build ./...`            | exit 0              |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need
the GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope** (the only files you should modify):
- `internal/tasks/streak.go`
- `internal/tasks/briefing.go`
- `internal/web/life.go`
- `internal/tasks/streak_test.go` (and `briefing_test.go` only if a
  signature change forces it)

**Out of scope** (do NOT touch):
- `Streak()`, `allowedGapDays`, `periodDays`, `noonOf`, `daysBetween` —
  the semantics core stays byte-identical.
- The caps (`briefingOverdueCap`/`briefingTodayCap`) and line wording of
  briefings (`dayLine` output strings must not change).
- Migrations/indexes — `idx_entries_*` already serve this query.
- `internal/turn/turn.go`.

## Git workflow

- Branch: `advisor/046-streak-batch`
- Conventional commit, e.g. `perf(tasks): batch streak completions into one query per render`
- Do NOT push or open a PR unless the operator instructed it.

## Steps

### Step 1: Add the batch function

In `internal/tasks/streak.go`:

```go
// StreaksFor computes live streaks for many tasks with ONE completions
// query (TodayBlock runs every turn — per-task queries were an N+1).
// Keyed by task id; tasks without a recurrence rule are absent. Errors
// read as an empty map — a missing streak must never block a briefing.
func StreaksFor(app core.App, recs []*core.Record, now time.Time) map[string]int {
	rules := make(map[string]Rule, len(recs))
	var ids []string
	for _, r := range recs {
		rule, err := Parse(r.GetString("recur"))
		if err != nil || rule.IsZero() {
			continue
		}
		rules[r.Id] = rule
		ids = append(ids, r.Id)
	}
	if len(ids) == 0 {
		return map[string]int{}
	}

	// kind = 'completion' && (task = {:t0} || task = {:t1} || …)
	params := dbx.Params{}
	conds := make([]string, len(ids))
	for i, id := range ids {
		key := fmt.Sprintf("t%d", i)
		conds[i] = fmt.Sprintf("task = {:%s}", key)
		params[key] = id
	}
	rows, err := app.FindRecordsByFilter("entries",
		"kind = 'completion' && ("+strings.Join(conds, " || ")+")",
		"noted_at", 0, 0, params)
	if err != nil {
		return map[string]int{}
	}

	// Same day-folding as CompletionDays, grouped per task: rows arrive
	// sorted by noted_at, so per-task subsequences stay ascending.
	days := make(map[string][]time.Time, len(ids))
	for _, r := range rows {
		id := r.GetString("task")
		d := noonOf(r.GetDateTime("noted_at").Time().In(now.Location()))
		ds := days[id]
		if len(ds) == 0 || !ds[len(ds)-1].Equal(d) {
			days[id] = append(ds, d)
		}
	}

	out := make(map[string]int, len(ids))
	for id, rule := range rules {
		out[id] = Streak(rule, days[id], now)
	}
	return out
}
```

Add `"fmt"` and `"strings"` imports as needed. Note: only the parameter
**names** are interpolated into the filter string; every **value** flows
through `dbx.Params` — keep it that way.

**Verify**: `go build ./internal/tasks/` → exit 0.

### Step 2: StreakFor becomes the one-task wrapper (one source of truth)

Replace `StreakFor`'s body with:

```go
func StreakFor(app core.App, rec *core.Record, now time.Time) int {
	return StreaksFor(app, []*core.Record{rec}, now)[rec.Id]
}
```

Keep its doc comment. If `CompletionDays` now has no non-test callers
(`grep -rn 'CompletionDays' internal --include='*.go' | grep -v _test`),
KEEP it anyway — it is exported, documented behavior with direct tests;
note it in your report instead of deleting.

**Verify**: `go test ./internal/tasks/` → existing streak/briefing tests pass.

### Step 3: dayLines batches; dayLine takes the streak

In `internal/tasks/briefing.go`:

- `dayLines`: before the loops, build the capped slice it will render
  (first ≤`briefingOverdueCap` of `bk.Overdue`, then up to the combined cap
  from `bk.Today` — mirror the existing loop bounds exactly), call
  `streaks := StreaksFor(app, capped, now)` once, and pass values down.
- `dayLine`: change signature to
  `dayLine(r *core.Record, now time.Time, overdue bool, streak int) string`
  (drop `app`), and replace the `StreakFor` call with the `streak` arg.
  The rendering logic (`streak > 1` → `" — habit, streak %d"`, else
  `" — habit"`) and the `Parse`-guard stay as they are.

**Verify**: `go test ./internal/tasks/` → ok; briefing wording unchanged
(existing briefing tests assert the lines).

### Step 4: /life page batches

In `internal/web/life.go:76-88`: first collect the recurring records
(`rule` parse-guard as today), then one
`streaks := tasks.StreaksFor(h.app, recurring, now)` call, then build the
views with `Streak: streaks[r.Id]`.

**Verify**: `go test ./internal/web/` → ok.

### Step 5: Parity test

In `internal/tasks/streak_test.go` add `TestStreaksForMatchesStreakFor`:
seed (via the package's existing test-app helper — read how
`streak_test.go`/`briefing_test.go` seed tasks + completion entries) three
tasks: a daily habit with a 3-day run ending today, a weekly habit with a
lapse (expected 0), and a task with no recurrence. Assert:

- `StreaksFor` over all three returns exactly the per-task values that
  three individual `StreakFor` calls return,
- the no-recurrence task is absent from the map (and `StreakFor` reads 0
  via the zero-value lookup),
- an empty input slice → empty map.

**Verify**: `go test ./internal/tasks/ -run TestStreaks` → ok.

### Step 6: Full gate

**Verify**: `gofmt -l .` → empty; `go vet ./...` → silent;
`go test ./...` → ok; `CGO_ENABLED=0 go build ./...` → exit 0;
`git diff --check` → empty.

## Test plan

- New: `TestStreaksForMatchesStreakFor` (parity + empty cases).
- Existing nets that must stay green: `Streak()` unit tables,
  briefing-line tests, life-page render tests.
- Verification: `go test ./...` → all pass.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `grep -c 'StreakFor(app' internal/tasks/briefing.go` → 0 (dayLine no longer queries)
- [ ] `grep -c 'tasks.StreakFor' internal/web/life.go` → 0
- [ ] `grep -c 'func StreaksFor' internal/tasks/streak.go` → 1
- [ ] `go test ./...` exits 0 (briefing wording tests unchanged and green)
- [ ] `gofmt -l .` empty, `go vet ./...` silent, `CGO_ENABLED=0 go build ./...` exits 0
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report back (do not improvise) if:

- The excerpts don't match the live code (drift).
- The parity test reveals ANY divergence between `StreaksFor` and the old
  per-task path — do not "fix" the divergence by adjusting expectations;
  report it (streak semantics are owner-visible).
- PocketBase rejects the OR-chain filter at a realistic size (test with
  ~50 conditions in the parity test if in doubt) — report; do not switch
  to string-concatenated values.
- A briefing test asserts on `dayLine`'s old signature in a way that
  changes wording.

## Maintenance notes

- The OR-chain grows with the task count on `/life` (briefing paths are
  capped at 13). If an owner someday has hundreds of recurring tasks,
  switch the filter to a two-clause form (`kind='completion'` + in-memory
  task filter) — measured, not preemptively.
- If recurrence kinds gain new rules, only `Streak`/`allowedGapDays`
  change; the batching is rule-agnostic.
- Reviewer should scrutinize: param values never enter the filter string
  (only `t%d` names do), and the capped-slice construction in `dayLines`
  mirrors the original loop bounds exactly.
