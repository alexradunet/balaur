# Plan 238: Honor the owner timezone in the web task surface and the nudge cron (matching briefing/recap)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, add the status row for this plan
> in `plans/README.md` (no 238 row exists yet — see Step 5) — unless a
> reviewer dispatched you and told you they maintain the index.
>
> **Drift check (run first)**:
> `git diff --stat 077318a..HEAD -- main.go internal/web/tasks.go internal/web/nudge.go internal/web/tasks_test.go internal/feature/taskcards internal/tasks/tasks_test.go internal/tasks/nudge_test.go .tours/18-component-system-and-storybook.tour`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `077318a`, 2026-07-01

## Why this matters

Balaur has an owner-pinned timezone: the `owner_settings` key `timezone`
(an IANA name), resolved by `store.OwnerLocation(app)`. Its whole purpose is
that the box's clock and the owner's day can differ — this checkout runs on a
VPS. The recap and briefing crons already honor it
(`time.Now().In(store.OwnerLocation(app))` in `main.go`), and
`internal/cli/cli.go:127` documents the contract: "CLI uses the host zone; the
web/turn path uses the owner's configured zone."

The web task surface and the nudge cron violate that contract. They run on bare
`time.Now()` (host zone), so when the owner's timezone differs from the box's:

- A due the owner types into the inline edit form as `15:00` is parsed as
  15:00 **host** time — hours off in the owner's day.
- Snooze quick-picks "tonight" (20:00) and "tomorrow" (09:00) target host-zone
  wall clock.
- The Today card and the Overdue/Today/Upcoming buckets split on the host's
  calendar day, so a task due tonight (owner time) can show as tomorrow or
  yesterday.
- Rendered due lines and the datetime-local edit pre-fill display host-zone
  wall times.
- The nudge cron composes reminder prose ("was Mon, Jan 2 at 15:04") in the
  host zone, so nudges and briefings state different clock times for the same
  task.

The fix is call-site-only: every helper below the gateway already takes `now`
(or `until`, or a `*time.Location`) as a parameter. No schema or stored-data
change — dues are stored as UTC instants; only the parse/render interpretation
changes. When the `timezone` setting is unset, `store.OwnerLocation` falls
back to `time.Local`, so behavior on a default install is bit-for-bit
unchanged.

## Current state

### The owner-zone primitive (do not modify — read for context)

`internal/store/time.go:22-32`:

```go
func OwnerLocation(app core.App) *time.Location {
	name := GetOwnerSetting(app, "timezone", "")
	if name == "" {
		return time.Local
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return time.Local
	}
	return loc
}
```

The pattern to replicate is already live in three places:
`main.go:198` (recap: `recap.EnsureSummaries(ctx, app, client, master.Id, time.Now().In(store.OwnerLocation(app)))`),
`main.go:239` (briefing, excerpted below), and `internal/web/web.go:298`
(`now := time.Now().In(loc)` with `loc := store.OwnerLocation(h.app)` in
`dockData`).

### Bug site 1 — the nudge cron, `main.go:210-223`

```go
func registerNudge(app core.App) {
	if os.Getenv("BALAUR_NUDGE") == "0" {
		return
	}
	scheduleJob(app, "nudge", "* * * * *", true, func(client llm.Client) {
		now := time.Now()
		if tasks.NudgeSuppressed(app, now) { // owner muted/disabled nudges (soft layer)
			return
		}
		if err := tasks.Nudge(app, client, now); err != nil {
			app.Logger().Warn("nudge: run stopped", "error", err)
		}
	})
}
```

Contrast the correct sibling, `main.go:238-239`:

```go
	scheduleJob(app, "briefing", "* * * * *", true, func(client llm.Client) {
		now := time.Now().In(store.OwnerLocation(app))
```

Nudge composition already converts the due into `now`'s location, so passing an
owner-zone `now` fixes the prose with no other change —
`internal/tasks/nudge.go:139-146`:

```go
func DueLine(due, now time.Time, status string) string {
	local := due.In(now.Location())
	when := local.Format("Mon, Jan 2 at 15:04")
	if local.Before(now) && status == "open" {
		return Lateness(due, now) + " — was " + when
	}
	return "due " + when
}
```

### Bug site 2 — the manual "Nudge now" button, `internal/web/nudge.go:59-64`

```go
func (h *handlers) nudgeNow(e *core.RequestEvent) error {
	if err := tasks.Nudge(h.app, nil, time.Now()); err != nil {
		return h.cardError(e, err)
	}
	return h.renderNudgeSection(e)
}
```

(`nudgeMute` at `internal/web/nudge.go:50` formats an *instant* as RFC3339 for
the mute-until comparison — zone-irrelevant; leave it alone.)

### Bug site 3 — the web task handlers, `internal/web/tasks.go`

`internal/web/tasks.go:51-53` (renders one card; `now` drives the due line,
overdue flag, and edit-form pre-fill):

```go
func (h *handlers) taskCardHTML(rec *core.Record) (string, error) {
	return renderNodeHTML(taskcards.TaskCard(taskcards.TaskViewOf(rec, time.Now()))), nil
}
```

`internal/web/tasks.go:60` (drives the Done timestamp/recurrence re-anchor and
the snooze target):

```go
	now := time.Now()
```

`internal/web/tasks.go:131-138` (the datetime-local due is parsed in the HOST
zone, and `tasks.Update`'s recurrence re-anchor gets a host-zone now):

```go
	if v := strings.TrimSpace(e.Request.FormValue("due")); v != "" {
		due, err := parseLocalDue(v, time.Now().Location())
		if err != nil {
			return h.cardError(e, err)
		}
		opts.Due = due
	}
	if err := tasks.Update(h.app, rec, time.Now(), opts); err != nil {
```

The helpers below are already location/now-parametrized — do NOT modify them,
only their call sites. `internal/web/tasks.go:153-160` (`parseLocalDue(s
string, loc *time.Location)`) and `internal/web/tasks.go:163-177`:

```go
func snoozeUntil(pick string, now time.Time) (time.Time, error) {
	switch pick {
	case "1h":
		return now.Add(time.Hour), nil
	case "tonight":
		t := time.Date(now.Year(), now.Month(), now.Day(), 20, 0, 0, 0, now.Location())
		if !t.After(now) {
			t = now.Add(time.Hour) // evening already: an hour of quiet instead
		}
		return t, nil
	case "tomorrow":
		return time.Date(now.Year(), now.Month(), now.Day(), 9, 0, 0, 0, now.Location()).AddDate(0, 0, 1), nil
	}
	return time.Time{}, fmt.Errorf("unknown snooze pick %q", pick)
}
```

`internal/web/tasks.go` already imports
`"github.com/alexradunet/balaur/internal/store"` (line 14) — no import change
needed there.

### Bug site 4 — the taskcards view builders (the bucket/view-model "now")

Every task card builder in `internal/feature/taskcards` resolves its own bare
`time.Now()`. Each has `app core.App` in scope (as a parameter or the
registration closure), so the fix is one line per builder plus a `store`
import. The seven sites (all verified — these are ALL non-test `time.Now()`
occurrences in the package):

`internal/feature/taskcards/questsfocus.go:35-36`:

```go
func BuildQuestsFocus(app core.App) QuestsFocusView {
	now := time.Now()
```

`internal/feature/taskcards/quests.go:122-123` (`func renderQuests(app
core.App, params map[string]string) g.Node {` then `now := time.Now()`).

`internal/feature/taskcards/today.go:32-35`:

```go
func buildToday(app core.App) TodayView {
	now := time.Now()
	recs, _ := tasks.OpenTasks(app, nil)
	bk := tasks.Bucket(recs, now)
```

`internal/feature/taskcards/habits.go:25-26` (`func buildHabits(app core.App)
[]HabitView {` then `now := time.Now()`).

`internal/feature/taskcards/taskscluster.go:20-21` (`func renderTasks(app
core.App, params map[string]string) g.Node {` then `now := time.Now()`).

`internal/feature/taskcards/calendar.go:40-42`:

```go
func buildCalendar(app core.App, monthParam string) CalView {
	now := time.Now()
	loc := now.Location()
```

`internal/feature/taskcards/timeline.go:37-43`:

```go
func buildTimeline(app core.App, days int) TLView {
	if days <= 0 {
		days = tlDefaultDays
	}
	now := time.Now()
	loc := now.Location()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
```

In `calendar.go` and `timeline.go`, `loc := now.Location()` becomes the owner
zone automatically once `now` is owner-zone — no second edit needed.

The bucketing these feed is day-boundary math in `now`'s location —
`internal/tasks/tasks.go:344-346`:

```go
func Bucket(recs []*core.Record, now time.Time) Buckets {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	dayEnd := dayStart.AddDate(0, 0, 1)
```

And the view-model rendering is zone-visible —
`internal/feature/taskcards/quests.go:100-104`:

```go
	if d := rec.GetDateTime("due").Time(); !d.IsZero() {
		v.Overdue = d.In(now.Location()).Before(now) && v.Status == "open"
		v.DueLine = tasks.DueLine(d, now, v.Status)
		// datetime-local value in the same zone the due line displays in.
		v.DueInput = d.In(now.Location()).Format("2006-01-02T15:04")
	}
```

### The contract this fix restores (do not modify)

`internal/cli/cli.go:125-128`:

```go
// when parses a CLI time flag with the same spec the model gets
// (tools.ParseDue); empty input returns the zero time.
// CLI uses the host zone; the web/turn path uses the owner's configured zone.
func when(flag, value string) (time.Time, error) {
```

The CLI's host-zone behavior is its *documented* contract — leave
`internal/cli` untouched. The turn path already complies (e.g.
`internal/tools/journal.go:39-40` uses `store.OwnerLocation(app)` for
`ParseDue`).

### Conventions that apply here

- **Errors**: `fmt.Errorf("doing x: %w", err)`, return early, no panics in
  library code. (This plan adds no new error paths.)
- **No global mutable state**; pass `core.App` explicitly. `OwnerLocation` is
  resolved per call on purpose ("no global state, and a dashboard edit takes
  effect on the next cron tick" — its own doc comment). Do NOT cache it.
- **Tests**: std `testing` package, table-driven where it helps.
  PocketBase-dependent tests use temp-dir apps — `storetest.NewApp(t)`
  (`internal/storetest/storetest.go:18`) or the web package's `newWebApp(t)`
  (`internal/web/handlers_test.go:29`). No `time.Sleep` synchronization.
- **`.tours/` are maintained artifacts**: `tours_test.go` catches missing
  files/out-of-range lines only; if your change shifts an anchored line, fix
  the tour anchor in the same commit. Anchors touching this plan's files:
  `.tours/18-component-system-and-storybook.tour` anchors
  `internal/feature/taskcards/quests.go` **line 24** (`func QuestsCard`) —
  adding the `store` import shifts it (Step 3 fixes it).
  `.tours/07-the-web-gateway.tour` anchors `internal/web/tasks.go` **line
  186** — Step 2 keeps all edits in-place (no line-count change) so this
  anchor stays correct; verify with `wc -l`.
- **`internal/self/knowledge.md`** does NOT need updating: it contains no
  timezone/zone claims (`grep -in timezone internal/self/knowledge.md` is
  empty), and this change adds no capability — it makes existing surfaces
  honor an existing setting.
- Audit calls: unchanged. `tasks.Done`/`Snooze`/`Drop`/`Nudge` already audit
  after the successful write; this plan does not touch them.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Full test gate (merge gate) | `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` | exit 0, all pass |
| Targeted tests | `TMPDIR=$HOME/.cache/go-tmp go test ./internal/<pkg>/ -run <Name> -count=1` | ok |
| Vet | `go vet ./...` | exit 0 |
| Format | `gofmt -l .` | empty output |
| Staticcheck | `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` | no output, exit 0 |
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tours lint | `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` | ok |

(The host `/tmp` is a small tmpfs; the Go linker OOMs there — always set
`TMPDIR` as shown for test commands.)

## Suggested executor toolkit

- If the `go-standards` skill is available, invoke it before writing the tests
  in Step 4 (testing idioms, owner-timezone cron math).
- If `graphify-out/graph.json` exists, `graphify query "owner timezone task
  bucket nudge"` gives a scoped subgraph for orientation. Do not run
  `graphify update` unless your environment's instructions say to after code
  changes.

## Scope

**In scope** (the only files you may modify):

- `main.go` (one line in `registerNudge`)
- `internal/web/tasks.go` (four in-place call-site edits)
- `internal/web/nudge.go` (one line in `nudgeNow`)
- `internal/feature/taskcards/questsfocus.go`
- `internal/feature/taskcards/quests.go`
- `internal/feature/taskcards/calendar.go`
- `internal/feature/taskcards/timeline.go`
- `internal/feature/taskcards/today.go`
- `internal/feature/taskcards/habits.go`
- `internal/feature/taskcards/taskscluster.go`
- `internal/web/tasks_test.go` (add tests)
- `internal/feature/taskcards/today_zone_test.go` (create — `today_test.go`
  already exists with package `taskcards_test` and stays untouched; the new
  test needs the internal `taskcards` package to call `buildToday`)
- `internal/tasks/tasks_test.go` (add one test)
- `internal/tasks/nudge_test.go` (add one test)
- `.tours/18-component-system-and-storybook.tour` (anchor line bump only)
- `plans/README.md` (status row only)

**Out of scope** (do NOT touch, even though they look related):

- `internal/cli/**` — host-zone parsing is its documented contract
  (`cli.go:127`); this plan changes no shared helper signature, so the comment
  stays true and untouched.
- `internal/tasks/*.go` non-test files — `Done`, `Update`, `Snooze`, `Bucket`,
  `Nudge`, `DueLine` already take `now`/`until` parameters; no threading is
  needed. If you believe one needs a signature change, STOP.
- `internal/store/**` — `OwnerLocation` is correct as-is; do not add an
  `OwnerNow` helper in this plan (deferred, see Maintenance notes).
- `main.go` `registerBriefing`/`registerRecap` and `internal/recap/**` —
  already owner-zone correct.
- `internal/web/day.go` and `internal/web/compact.go` — same bug class but a
  different surface (journal day page, recap compaction); deliberately
  deferred (see Maintenance notes).
- `internal/web/nudge.go:50` (`nudgeMute`) — RFC3339 instant, zone-irrelevant.
- Stored data / migrations — dues remain UTC instants; no backfill.
- `internal/self/knowledge.md` — no capability change (see Current state).

## Git workflow

- Work in an isolated git worktree branched from `origin/main`; branch name
  `advisor/238-owner-timezone-web-tasks-nudge`.
- Conventional-commit subjects (`feat`/`fix`/`docs`/`refactor`/`style`/`test`/
  `chore`), e.g. `fix(tasks): honor owner timezone in web task surface and nudge cron`.
- Commit per logical unit **with explicit pathspecs** (the main checkout is
  shared by parallel agents — stage only your own files, never `git add -A`).
- **NEVER push**; the reviewer merges.

## Steps

### Step 1: Owner-zone `now` in the nudge cron and the "Nudge now" button

1. `main.go:215` — change in place (mirrors `main.go:239` exactly):

   ```go
   		now := time.Now().In(store.OwnerLocation(app))
   ```

   `store` is already imported in `main.go`.

2. `internal/web/nudge.go:60` — change in place:

   ```go
   	if err := tasks.Nudge(h.app, nil, time.Now().In(store.OwnerLocation(h.app))); err != nil {
   ```

   `store` is already imported in `internal/web/nudge.go`.

**Verify**:
- `grep -c "time.Now().In(store.OwnerLocation(app))" main.go` → `3`
  (recap line 198, nudge line 215, briefing line 239)
- `grep -c "time.Now().In(store.OwnerLocation(h.app))" internal/web/nudge.go` → `1`
- `CGO_ENABLED=0 go build ./...` → exit 0

Commit: `fix(tasks): nudge cron and nudge-now run on owner-zone now` with
pathspecs `main.go internal/web/nudge.go`.

### Step 2: Owner-zone `now` in the web task handlers (in-place edits only)

All four edits in `internal/web/tasks.go` replace text on an existing line
without adding or removing lines — this preserves the tour anchor at line 186
(`.tours/07-the-web-gateway.tour`). Do NOT hoist a shared `loc` variable (that
would add a line); the double `OwnerLocation` lookup inside `taskEdit` is a
cheap settings-row read and is acceptable.

1. Line 52 (`taskCardHTML`):

   ```go
   	return renderNodeHTML(taskcards.TaskCard(taskcards.TaskViewOf(rec, time.Now().In(store.OwnerLocation(h.app))))), nil
   ```

2. Line 60 (`taskTransition` — feeds `tasks.Done` and `snoozeUntil`):

   ```go
   	now := time.Now().In(store.OwnerLocation(h.app))
   ```

3. Line 132 (`taskEdit` — the datetime-local parse location):

   ```go
   		due, err := parseLocalDue(v, store.OwnerLocation(h.app))
   ```

4. Line 138 (`taskEdit` — `tasks.Update`'s recurrence re-anchor now):

   ```go
   	if err := tasks.Update(h.app, rec, time.Now().In(store.OwnerLocation(h.app)), opts); err != nil {
   ```

**Verify**:
- `grep -n "time.Now()" internal/web/tasks.go | grep -v OwnerLocation` → empty
- `wc -l < internal/web/tasks.go` → `210` (line count unchanged → tour anchor intact)
- `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -count=1` → ok
  (existing tests seed dues in `time.Local`; with the `timezone` setting unset,
  `OwnerLocation` falls back to `time.Local`, so they must stay green — see
  STOP conditions)

Commit: `fix(web): task handlers parse and render in the owner timezone` with
pathspec `internal/web/tasks.go`.

### Step 3: Owner-zone `now` in the seven taskcards builders

For EACH of the seven files below, (a) add
`"github.com/alexradunet/balaur/internal/store"` to the internal import group
(alphabetical: after `nodes`, before `tasks` where both exist), and (b) change
the one `now := time.Now()` line to:

```go
	now := time.Now().In(store.OwnerLocation(app))
```

Files and the line holding `now := time.Now()` as of the planned-at commit
(numbers shift by +1 after the import is added to the same file):

- `internal/feature/taskcards/questsfocus.go:36` (in `BuildQuestsFocus`)
- `internal/feature/taskcards/quests.go:123` (in `renderQuests`)
- `internal/feature/taskcards/today.go:33` (in `buildToday`)
- `internal/feature/taskcards/habits.go:26` (in `buildHabits`)
- `internal/feature/taskcards/taskscluster.go:21` (in `renderTasks`)
- `internal/feature/taskcards/calendar.go:41` (in `buildCalendar`)
- `internal/feature/taskcards/timeline.go:41` (in `buildTimeline`)

Leave the `loc := now.Location()` lines in `calendar.go`/`timeline.go` as-is —
they now inherit the owner zone.

Then fix the shifted tour anchor: in
`.tours/18-component-system-and-storybook.tour`, the step with
`"file": "internal/feature/taskcards/quests.go"` has `"line": 24` (anchoring
`func QuestsCard`). Set its `"line"` to the new line of `func QuestsCard(`
(run `grep -n "func QuestsCard(" internal/feature/taskcards/quests.go`;
expected `25` after the one-line import addition).

**Verify**:
- `grep -rn "time.Now()" internal/feature/taskcards --include="*.go" | grep -v _test.go | grep -v OwnerLocation` → empty
- `grep -c "internal/store" internal/feature/taskcards/*.go` — each of the
  seven edited files shows `1`
- `CGO_ENABLED=0 go build ./...` → exit 0 (also proves no import cycle)
- `TMPDIR=$HOME/.cache/go-tmp go test ./internal/feature/taskcards/ -count=1` → ok
- `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok

Commit: `fix(taskcards): card builders resolve now in the owner timezone` with
pathspecs `internal/feature/taskcards/*.go .tours/18-component-system-and-storybook.tour`.

### Step 4: Regression tests

Write the six tests specified in the Test plan below (three in
`internal/web/tasks_test.go`, one new file
`internal/feature/taskcards/today_zone_test.go`, one in
`internal/tasks/tasks_test.go`, one in `internal/tasks/nudge_test.go` — the
Bucket and DueLine ones are small). Exact cases, fixtures, and expected values
are in the Test plan.

**Verify**:
- `TMPDIR=$HOME/.cache/go-tmp go test ./internal/web/ -run 'TestTaskEditParsesDueInOwnerZone|TestQuestsFocusRendersOwnerZoneDue|TestTaskSnoozeTomorrowUsesOwnerZone' -count=1 -v` → 3 tests PASS (none skipped, unless the host zone is UTC+14)
- `TMPDIR=$HOME/.cache/go-tmp go test ./internal/feature/taskcards/ -run TestBuildTodayRendersOwnerZoneDueLine -count=1 -v` → PASS
- `TMPDIR=$HOME/.cache/go-tmp go test ./internal/tasks/ -run 'TestBucketHonorsNowZone|TestDueLineRendersInNowZone' -count=1 -v` → 2 tests PASS

Commit: `test(tasks): owner-timezone regression coverage for web tasks and nudge prose`
with pathspecs `internal/web/tasks_test.go internal/feature/taskcards/today_zone_test.go internal/tasks/tasks_test.go internal/tasks/nudge_test.go`.

### Step 5: Full gate

Run, in order:

1. `gofmt -l .` → empty output
2. `go vet ./...` → exit 0
3. `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output, exit 0
4. `CGO_ENABLED=0 go build ./...` → exit 0
5. `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0
6. `git diff --check` → no output
7. `git status --porcelain` → only in-scope files listed

Add a plan 238 row to `plans/README.md` (no row for 238 exists at the
planned-at commit — the newest status table, `| Plan | Builds | Status |`,
ends at plan 234). Append to that table (or to a newer status table for this
batch if one exists by then), e.g.:

```
| 238 | Owner-timezone fix — web task surface + nudge cron honor the `timezone` setting (call-site-only) | DONE |
```

Skip this if the dispatching reviewer maintains the index. Commit the row
change separately: `chore(plans): mark 238 done` with pathspec
`plans/README.md`.

## Test plan

All tests use the standard `testing` package and existing temp-dir app
helpers; no real model, no `time.Sleep`. The chosen owner zone is
`Pacific/Kiritimati` (UTC+14, no DST): no plausible host runs at +14, so a
host-zone regression always produces a different wall time. Guard each web/
taskcards test with:

```go
if _, off := time.Now().Zone(); off == 14*3600 {
	t.Skip("host zone is UTC+14; owner-zone and host-zone results coincide")
}
```

Set the owner zone with `store.SetOwnerSetting(app, "timezone",
"Pacific/Kiritimati")` (`internal/store/owner_settings.go:47`) right after
creating the test app.

1. **`TestTaskEditParsesDueInOwnerZone`** — `internal/web/tasks_test.go`.
   Model after `TestTaskEdit` (same file, line 232) for the `tests.ApiScenario`
   POST to `/ui/tasks/{id}/edit`, and after `internal/web/show_test.go:54` for
   `AfterTestFunc` (the harness closes the app after the run, so DB assertions
   must happen inside `AfterTestFunc`). Fixture: `newWebApp(t)`, set the
   timezone setting, seed via `seedTaskWithRecur(t, app, "Tax return", "open",
   "", time.Now().Add(time.Hour))`. Body:
   `title=Tax+return&due=2027-03-01T15:00&recur=&notes=`. In `AfterTestFunc`,
   reload with `app.FindRecordById("nodes", rec.Id)`, call `itasks.Hydrate`,
   and assert the stored due instant:

   ```go
   loc, _ := time.LoadLocation("Pacific/Kiritimati")
   want := time.Date(2027, 3, 1, 15, 0, 0, 0, loc)
   if d := got.GetDateTime("due").Time(); !d.Equal(want) { ... }
   ```

   This is the regression this plan fixes: before the fix the stored instant
   is 15:00 host zone, not 15:00 owner zone. (New imports needed in the test
   file: `net/http`, `github.com/alexradunet/balaur/internal/store`.)

2. **`TestQuestsFocusRendersOwnerZoneDue`** — `internal/web/tasks_test.go`.
   Model after `TestQuestsFocusPrefillsEditForm` (same file, line 259).
   Seed one task with `due = time.Date(2030, 3, 4, 2, 0, 0, 0, time.UTC)`
   (02:00 UTC = 16:00 same day in UTC+14). GET `/ui/show/quests`,
   `ExpectedContent: []string{`value="2030-03-04T16:00"`}` — proves
   `BuildQuestsFocus` → `TaskViewOf` runs on owner-zone `now` (the `DueInput`
   pre-fill renders in `now.Location()`, quests.go:104).

3. **`TestTaskSnoozeTomorrowUsesOwnerZone`** — `internal/web/tasks_test.go`.
   ApiScenario POST to `/ui/tasks/{id}/transition` with body
   `to=snooze&until=tomorrow` (headers as in `TestTaskTransitionEmitsToast`,
   same file, line 73). Compute the expectation BEFORE running the scenario:

   ```go
   ownerNow := time.Now().In(loc)
   want := time.Date(ownerNow.Year(), ownerNow.Month(), ownerNow.Day(), 9, 0, 0, 0, loc).AddDate(0, 0, 1)
   ```

   In `AfterTestFunc`, reload + `itasks.Hydrate`, read the snooze target from
   props (`tasks.Snooze` stores `props["snoozed_until"] =
   store.PBTime(until.UTC())` — see `internal/tasks/tasks.go:255-270`; read it
   the way `TestSnoozeAndDrop` in `internal/tasks/tasks_test.go:228` does, via
   `nodes.Props(rec)` + `store.ParsePBTime`), and assert it equals `want`.
   (Known negligible edge: crossing owner-zone midnight between computing
   `want` and the request; do not add synchronization for it.)

4. **`TestBucketHonorsNowZone`** — `internal/tasks/tasks_test.go`, package
   `tasks`. Model after `TestBucket` (same file, line 393). Fully
   deterministic (fixed `now`, no live clock): create one task due
   `time.Date(2030, 6, 15, 23, 0, 0, 0, kir)` where `kir` is
   Pacific/Kiritimati. With `now := time.Date(2030, 6, 15, 1, 0, 0, 0, kir)`
   (= 2030-06-14 11:00 UTC), `Bucket` must place it in `Today`; with the same
   instant expressed as `now.UTC()`, `Bucket` must place it in `Upcoming`
   (due = 2030-06-15 09:00 UTC, past the UTC June-14 day end). This documents
   the day-boundary semantics that make the owner-zone threading in Steps 2-3
   fix the Overdue/Today split.

5. **`TestBuildTodayRendersOwnerZoneDueLine`** — new file
   `internal/feature/taskcards/today_zone_test.go`, package `taskcards`
   (internal package test — `buildToday` is unexported; the existing
   `today_test.go` is package `taskcards_test` and cannot host it, so leave
   that file alone. `filterBucket` is called the same way from
   `taskscluster_test.go`, use that file as the structural pattern; app via
   `storetest.NewApp(t)`). Set the timezone setting, create one task with the
   far-past due `time.Date(2020, 1, 2, 12, 0, 0, 0, time.UTC)` (always
   Overdue in every zone → always in `buildToday`'s rows; non-recurring past
   dues are accepted, see `internal/tasks/tasks_test.go:376`). Assert
   `len(buildToday(app).Rows) == 1` and
   `strings.Contains(v.Rows[0].DueLine, "Jan 3 at 02:00")` — 12:00 UTC is
   02:00 next day in UTC+14; a host-zone regression renders the host wall time
   instead.

6. **`TestDueLineRendersInNowZone`** — `internal/tasks/nudge_test.go`. Pure
   function, no app. `due := time.Date(2030, 1, 2, 12, 0, 0, 0, time.UTC)`,
   `now := due.Add(time.Hour).In(time.FixedZone("owner", 14*3600))`,
   `got := DueLine(due, now, "open")`; assert
   `strings.Contains(got, "Jan 3 at 02:00")` and
   `strings.HasPrefix(got, "overdue 1h")`. This pins the contract
   `registerNudge` now relies on: owner-zone `now` ⇒ owner-zone nudge prose.

(That is six tests total; items 4-6 are small unit tests.)

Verification: the Step 4 commands, then the full gate in Step 5 —
`TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0 including all six
new tests.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `gofmt -l .` → empty; `go vet ./...` → exit 0;
      `go run honnef.co/go/tools/cmd/staticcheck@latest ./...` → no output;
      `CGO_ENABLED=0 go build ./...` → exit 0
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test ./... -count=1` → exit 0
- [ ] `grep -n "time.Now()" internal/web/tasks.go | grep -v OwnerLocation` → empty
- [ ] `grep -rn "time.Now()" internal/feature/taskcards --include="*.go" | grep -v _test.go | grep -v OwnerLocation` → empty
- [ ] `sed -n '215p' main.go | grep -c "OwnerLocation"` → `1` (or the
      equivalent grep if lines drifted: `grep -A5 'scheduleJob(app, "nudge"' main.go | grep -c OwnerLocation` → `1`)
- [ ] `grep -c "OwnerLocation" internal/web/nudge.go` → `1`
- [ ] The six new tests exist and pass by name (Step 4 verify commands)
- [ ] `TMPDIR=$HOME/.cache/go-tmp go test . -run TestTours -count=1` → ok
- [ ] `wc -l < internal/web/tasks.go` → `210`
- [ ] `git status --porcelain` lists no files outside the in-scope list
- [ ] `plans/README.md` status row for 238 added (per Step 5; unless the
      reviewer maintains the index)

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows changes in in-scope files and any "Current state"
  excerpt no longer matches the live code (e.g. the `time.Now()` sites moved
  or were already fixed).
- Adding the `store` import to any `internal/feature/taskcards` file creates
  an import cycle (`go build` reports one) — `internal/store` must not import
  taskcards; if it somehow does now, report instead of restructuring.
- Any of the seven taskcards builders no longer has `app core.App` in scope at
  the `time.Now()` line (i.e. the `now` is baked more than one call level
  below `app`) — report the layering; do not refactor signatures.
- After Step 2 or Step 3, any EXISTING test fails — in particular
  `TestTaskEdit`, `TestQuestsFocusPrefillsEditForm` (both seed dues in
  `time.Local`, and with the `timezone` setting unset `OwnerLocation` falls
  back to `time.Local`, so they must stay green). A failure means a hidden
  host-zone dependency exists; report it rather than editing those tests.
- The fix appears to require modifying `internal/tasks/*.go` (non-test),
  `internal/store/**`, `internal/cli/**`, or any other out-of-scope file.
- `tests.ApiScenario` has no `AfterTestFunc` field in this PocketBase version
  (it does at pinned v0.39.3 — see `internal/web/show_test.go:54`; if the
  dependency moved, report).
- A step's verification fails twice after a reasonable fix attempt.

## Maintenance notes

- **Owner-visible reinterpretation, no backfill**: dues are stored as UTC
  instants. An owner whose `timezone` setting ALREADY differed from the box
  zone entered dues that were stored under the wrong (host-zone)
  interpretation; after this fix those existing dues render shifted by the
  zone difference. This is the stored truth finally displayed consistently —
  the owner re-edits outliers. No migration is attempted on purpose (there is
  no way to know which stored dues the owner meant in which zone).
- **Same bug class deliberately deferred**: `internal/web/day.go` (lines 23
  and 41 — the journal day page parses `YYYY-MM-DD` in the host zone) and
  `internal/web/compact.go` (lines 44 and 77 — recap draft/commit `now`) still
  use bare `time.Now()`. They interact with recap's owner-zone day math and
  deserve their own pass; do not fold them in here.
- **Possible consolidation**: `time.Now().In(store.OwnerLocation(app))` now
  appears ~14 times across main/web/taskcards. A `store.OwnerNow(app)` helper
  is a reasonable follow-up sweep (one source of truth), deferred here to keep
  this change call-site-only and reviewable.
- **CLI stays host-zone** by documented contract (`internal/cli/cli.go:127`).
  If a future change routes CLI task edits through the web/turn helpers,
  revisit that comment.
- **DST**: snooze "tonight"/"tomorrow" and the bucket day boundaries use
  `time.Date(...)` in the owner zone, so DST transitions are handled by the
  stdlib; Pacific/Kiritimati in tests avoids DST for determinism.
- **Reviewer scrutiny**: confirm every edit is a call site only (no helper
  signature changed), and that behavior with the `timezone` setting unset is
  identical (fallback `time.Local` — the whole existing test suite passing
  unmodified is the evidence).
