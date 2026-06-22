# Plan 128: Extract a single `tasks.DueLine` helper (dedup three copies)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If
> anything in "STOP conditions" occurs, stop and report. When done, update the
> status row for this plan in `plans/readme.md`.
>
> **Drift check (run first)**: `git diff --stat b61e060..HEAD -- internal/tasks/ internal/web/tasks.go internal/feature/taskcards/quests.go internal/feature/taskcards/today.go`
> Compare the three "Current state" excerpts against the live code; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P3
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: idiom / tech-debt
- **Planned at**: commit `b61e060`, 2026-06-21

## Why this matters

The user-facing "due line" for a task ("3 days late — was Mon, Jan 2 at 14:00" /
"due Mon, Jan 2 at 14:00") is copy-pasted in three places, and the date layout
literal `"Mon, Jan 2 at 15:04"` appears six times. A wording or format change
must touch all three in lockstep, and they can drift. One exported helper in the
`internal/tasks` domain package is the single source of truth; the card builders
call it. (One of the three copies, in `internal/web/tasks.go`, is reached only
through the legacy `card-task.html` path that plans 111–117 will retire, but it
is live today and must stay correct until then.)

## Current state — the three identical copies

`internal/tasks/nudge.go` already exports `Lateness(due, now time.Time) string`
(line 78). The repeated logic wraps it:

`internal/web/tasks.go:43-48` (inside `taskViewOf`):
```go
if local.Before(now) && v.Status == "open" {
    v.Overdue = true
    v.DueLine = tasks.Lateness(due, now) + " — was " + local.Format("Mon, Jan 2 at 15:04")
} else {
    v.DueLine = "due " + local.Format("Mon, Jan 2 at 15:04")
}
```
(`local := due.In(now.Location())` is computed just above; `due` is the task's
due `time.Time`.)

`internal/feature/taskcards/quests.go:96-103`:
```go
local := d.In(now.Location())
if local.Before(now) && v.Status == "open" {
    v.Overdue = true
    v.DueLine = tasks.Lateness(d, now) + " — was " + local.Format("Mon, Jan 2 at 15:04")
} else {
    v.DueLine = "due " + local.Format("Mon, Jan 2 at 15:04")
}
```

`internal/feature/taskcards/today.go:53-59`:
```go
if d := rec.GetDateTime("due").Time(); !d.IsZero() {
    local := d.In(now.Location())
    if local.Before(now) && row.Status == "open" {
        row.DueLine = tasks.Lateness(d, now) + " — was " + local.Format("Mon, Jan 2 at 15:04")
    } else {
        row.DueLine = "due " + local.Format("Mon, Jan 2 at 15:04")
    }
}
```

Note the `Overdue` boolean is set in two of the three sites; the helper returns
only the *line string*, and each caller keeps its own `Overdue`/`Status`
bookkeeping (the helper takes `status` so it can decide "was" vs "due").

Convention: exported funcs in `internal/tasks` carry a godoc comment starting
with the name (see `Lateness` in `nudge.go`).

## Commands you will need

| Purpose | Command                                        | Expected |
|---------|------------------------------------------------|----------|
| Build   | `CGO_ENABLED=0 go build ./...`                 | exit 0   |
| Tests   | `go test ./internal/tasks/ ./internal/feature/taskcards/ ./internal/web/` | all pass |
| Format  | `gofmt -l internal/`                           | empty    |

## Steps

### Step 1: Add `tasks.DueLine` to `internal/tasks/nudge.go`

Add, next to `Lateness`:
```go
// DueLine renders a task's due time as one owner-facing line: an overdue
// open task reads "<lateness> — was <when>", everything else "due <when>".
// status is the task's status (only "open" tasks read as overdue).
func DueLine(due, now time.Time, status string) string {
    local := due.In(now.Location())
    when := local.Format("Mon, Jan 2 at 15:04")
    if local.Before(now) && status == "open" {
        return Lateness(due, now) + " — was " + when
    }
    return "due " + when
}
```

**Verify**: `go test ./internal/tasks/` → all pass.

### Step 2: Call it from the three sites

- `internal/web/tasks.go`: replace the if/else DueLine block with
  `v.DueLine = tasks.DueLine(due, now, v.Status)` and set the Overdue flag
  separately: `v.Overdue = due.In(now.Location()).Before(now) && v.Status == "open"`.
  (Keep `local` only if still used elsewhere in the function; if it becomes
  unused after this, remove it.)
- `internal/feature/taskcards/quests.go`: same shape —
  `v.DueLine = tasks.DueLine(d, now, v.Status)` and
  `v.Overdue = d.In(now.Location()).Before(now) && v.Status == "open"`.
- `internal/feature/taskcards/today.go`: inside the `!d.IsZero()` block,
  `row.DueLine = tasks.DueLine(d, now, row.Status)` (no Overdue flag here).

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 3: Full gate

**Verify**:
- `go test ./internal/tasks/ ./internal/feature/taskcards/ ./internal/web/` → all pass
- `gofmt -l internal/` → empty; `git diff --check` → clean
- `grep -rn '"Mon, Jan 2 at 15:04"' internal/` → only inside `tasks.DueLine`

## Test plan

- Add one table test `TestDueLine` in `internal/tasks/nudge_test.go` (create if
  absent, else append) covering: an overdue open task (`status="open"`, due in
  the past) → starts with the `Lateness` text + " — was "; a future task → starts
  with "due "; a past task that is NOT open (`status="done"`) → "due " (not
  overdue). Model the table style on the existing `internal/tasks` tests.
- Existing card tests are the parity net: the rendered `DueLine` strings must be
  byte-identical to before, so `internal/feature/taskcards` and `internal/web`
  tests pass unchanged.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go test ./...` passes; `TestDueLine` exists and passes
- [ ] `grep -rn '"Mon, Jan 2 at 15:04"' internal/` returns matches ONLY inside `internal/tasks/nudge.go`
- [ ] `gofmt -l internal/` empty; `git diff --check` clean
- [ ] Only in-scope files modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:
- The "Current state" excerpts don't match the live code (drift).
- A card test FAILS after the swap (the rendered line drifted — the helper's
  output is not byte-identical to a call site's prior output).
- Removing `local` orphans something you can't cleanly resolve in-scope.

## Scope

**In scope**: `internal/tasks/nudge.go`, `internal/tasks/nudge_test.go`,
`internal/web/tasks.go`, `internal/feature/taskcards/quests.go`,
`internal/feature/taskcards/today.go`, `plans/readme.md` (status row).

**Out of scope**: `tasks.Lateness` itself (unchanged); the `dayLine`
formatting in `internal/tasks/briefing.go` (a different, intentionally distinct
format — do not fold it in); the legacy `card-task.html` template.

## Git workflow

- Branch off `origin/main`: `improve/128-extract-tasks-dueline`.
- One commit; conventional subject, e.g.
  `refactor(tasks): extract DueLine helper (dedup 3 sites)`.
- Do NOT push or open a PR unless the operator instructs it.

## Maintenance notes

- When the legacy `card-task.html` path is removed (plans 111–117), the
  `internal/web/tasks.go` `taskViewOf` caller may go with it; the helper stays,
  used by the live `taskcards` components.
- Any future change to the due-line wording/format now happens in exactly one
  place — that is the point.
