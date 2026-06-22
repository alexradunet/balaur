# Plan 140: Parse and format reminders in the owner's configured timezone

> **Executor instructions**: Follow this plan step by step. The Go compiler is a
> completeness check here — after you change the `ParseDue`/`fmtDue` signatures,
> `go build` lists every caller to fix. Run every verification command before
> moving on. If a STOP condition occurs, stop and report. When done, update the
> status row for this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat 0c06da8..HEAD -- internal/tools/ internal/turn/turn.go internal/cli/cli.go`

## Status

- **Priority**: P1
- **Effort**: M
- **Risk**: MED
- **Depends on**: none
- **Category**: bug
- **Planned at**: commit `0c06da8`, 2026-06-22

## Why this matters

The owner can pin a timezone in `owner_settings` (`timezone` key), and the
briefing (plan 132) and recap already honor it via `store.OwnerLocation(app)`.
But task/reminder parsing still uses the host clock's `time.Local`: `ParseDue`
parses "tomorrow at 10" and the date-only 09:00 default in `time.Local`, `fmtDue`
formats in `time.Local`, the turn's "present moment" line reports the host zone,
and the life/journal renderers format timestamps in `time.Local`. For an owner
whose pinned zone differs from the host OS (laptop travels, VPS in another
region), reminders land at the wrong wall-clock time and the companion reasons in
one zone while committing in another. This makes reminders consistent with the
rest of the system: one clock, the owner's.

(The CLI's `when()` helper has no `app` in scope and is a machine harness — it
keeps `time.Local`, documented, so CLI behavior is unchanged.)

## Current state

`internal/tools/tasks.go`:
- `ParseDue` (39-58) parses with `time.Local` (lines 47, 50, 54, 55).
- `fmtDue` (60-62): `return t.In(time.Local).Format("Mon, Jan 2 2006 at 15:04")`.
- `ParseDue` callers: `tasks.go:88` (task_add), `tasks.go:267` (task_snooze),
  `internal/tools/journal.go:39`, `internal/tools/life.go:60`,
  `internal/cli/cli.go:124` (the `when` helper — NO `app`).
- `fmtDue` callers (all in `tasks.go` tool closures/helpers): `107`, `214`,
  `246`, `272`, `281`.

`internal/turn/turn.go:94` — `now := time.Now()` (feeds `nowLine` at 216-217,
which reports `now.Zone()`, and `tasks.TodayBlock`).

`internal/tools/life.go` — `time.Local` formatters at `79`, `125`, `147`.
`internal/tools/journal.go` — `time.Local` formatter at `47`.

`internal/store/time.go:22` — `func OwnerLocation(app core.App) *time.Location`
(already used by recap + briefing). The `internal/tools` and `internal/turn`
packages already import `internal/store`.

## Commands you will need

| Purpose | Command                                              | Expected |
|---------|------------------------------------------------------|----------|
| Build   | `CGO_ENABLED=0 go build ./...`                       | exit 0 (drives the caller fixes) |
| Tests   | `go test ./internal/tools/ ./internal/turn/ ./internal/cli/` | all pass |
| Lint    | `make lint`                                          | exit 0   |

## Steps

### Step 1: Add a `loc *time.Location` parameter to `ParseDue` and `fmtDue`

Change the signatures and replace every internal `time.Local` with `loc`:
```go
func ParseDue(s string, loc *time.Location) (t time.Time, dateOnly bool, err error) {
	…
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.In(loc), false, nil
	}
	for _, layout := range []string{"2006-01-02T15:04:05", "2006-01-02T15:04"} {
		if t, err := time.ParseInLocation(layout, s, loc); err == nil {
			return t, false, nil
		}
	}
	if d, err := time.ParseInLocation("2006-01-02", s, loc); err == nil {
		return time.Date(d.Year(), d.Month(), d.Day(), 9, 0, 0, 0, loc), true, nil
	}
	…
}

func fmtDue(t time.Time, loc *time.Location) string {
	return t.In(loc).Format("Mon, Jan 2 2006 at 15:04")
}
```

**Verify**: `CGO_ENABLED=0 go build ./...` → now FAILS with "not enough arguments"
at every caller. That list is your Step 2 worklist.

### Step 2: Fix each caller to pass the right location

For each build error, pass the owner location (tools) or `time.Local` (CLI):
- `internal/tools/tasks.go` — in each tool's `Execute` closure (which captures
  `app`), resolve `loc := store.OwnerLocation(app)` once and pass it to every
  `ParseDue`/`fmtDue` call in that closure. If a `fmtDue` call lives in a helper
  function (e.g. the list renderer around line 214), add a `loc *time.Location`
  parameter to that helper and pass it down from the closure.
- `internal/tools/journal.go:39` and `internal/tools/life.go:60` — same:
  `loc := store.OwnerLocation(app)` in the closure, pass to `ParseDue`.
- `internal/cli/cli.go:124` (`when`) — pass `time.Local` (the CLI harness has no
  `app`; keep host-zone behavior). Add a brief comment: "CLI uses the host zone;
  the web/turn path uses the owner's configured zone."

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 3: Resolve the turn's "present moment" in the owner zone

`internal/turn/turn.go:94` — change `now := time.Now()` to
`now := time.Now().In(store.OwnerLocation(app))` so `nowLine` reports the owner's
zone and `TodayBlock` renders in it. (Confirm `store` is imported in turn.go — it
is, used elsewhere.)

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 4: Owner zone for the life/journal renderers

Replace `time.Local` with `store.OwnerLocation(app)` at
`internal/tools/life.go:79,125,147` and `internal/tools/journal.go:47` (these are
inside tool closures with `app` in scope — resolve `loc` once per closure and
reuse).

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0;
`grep -rn "time.Local" internal/tools/` → returns ONLY the CLI-comment reference
if any (ideally nothing in `internal/tools/`; `time.Local` may remain ONLY in
`internal/cli/cli.go`).

### Step 5: Full gate

**Verify**: `go test ./internal/tools/ ./internal/turn/ ./internal/cli/` → all
pass; `go test ./...` → all pass; `make lint` → exit 0.

## Test plan

- Add `TestParseDueHonorsLocation` to `internal/tools/tasks_test.go`: with a
  fixed non-UTC `*time.Location` (e.g. `time.FixedZone("test", 2*3600)` or
  `time.LoadLocation("America/New_York")`), assert (a) a date-only input parses to
  09:00 in THAT location (not host local), and (b) a `2006-01-02T15:04` input
  parses with that location's offset. Compare against the same input parsed with
  `time.UTC` to show the zone actually changes the result. `ParseDue` is pure
  given `loc`, so this is a clean unit test.
- Existing `internal/tools` tests that call `ParseDue`/`fmtDue` must be updated to
  pass a location argument (use `time.Local` in those tests to preserve their
  current expectations, unless a test specifically asserts zone behavior).
- Confirm the CLI tests still pass (CLI keeps `time.Local`).

## Done criteria

- [ ] `CGO_ENABLED=0 go build ./...` exits 0; `go vet ./...` exits 0
- [ ] `go test ./...` passes, including `TestParseDueHonorsLocation`
- [ ] `grep -rn "time.Local" internal/tools/` returns nothing (all moved to owner zone; CLI's `time.Local` lives in `internal/cli/`)
- [ ] `grep -n "OwnerLocation" internal/turn/turn.go` shows it used for `now`
- [ ] `make lint` exits 0
- [ ] Only `internal/tools/tasks.go`, `internal/tools/journal.go`,
      `internal/tools/life.go`, `internal/turn/turn.go`, `internal/cli/cli.go`,
      the touched test files, and `plans/readme.md` modified
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report if:
- A `ParseDue`/`fmtDue` caller exists that has neither `app` nor an obvious
  host-zone intent — report it rather than guessing the location.
- `store.OwnerLocation` is not resolvable from a package you must edit (import
  cycle) — report; do not duplicate the timezone-resolution logic.
- A test asserts a specific wall-clock string that changes under the owner zone in
  a way that suggests a real behavior question — report rather than force-updating.

## Scope

**In scope**: `internal/tools/tasks.go`, `internal/tools/journal.go`,
`internal/tools/life.go`, `internal/turn/turn.go`, `internal/cli/cli.go`, their
test files, `plans/readme.md` (status row).
**Out of scope**: `store.OwnerLocation` itself (unchanged); the briefing/recap
(already owner-zoned); the recap period math.

## Git workflow

- Branch off `origin/main`: `improve/140-reminder-owner-timezone`.
- One commit (or one per step); subject e.g.
  `fix(tools): parse and format reminders in the owner's configured timezone`.
- Do NOT push or open a PR.

## Maintenance notes

- The rule (now in the `go-standards` skill): wall-clock / per-day logic resolves
  `store.OwnerLocation(app)`, not bare `time.Local`/`time.Now()`. The CLI harness
  is the one intentional exception (no owner context, host zone).
- Any new time-formatting tool should resolve `loc` from `app` the same way.
