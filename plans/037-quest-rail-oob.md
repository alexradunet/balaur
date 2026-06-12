# Plan 037: Quest-rail live refresh — OOB swap after panel transitions

> **Executor instructions**: Follow this plan step by step; run every
> verification and confirm the expected result. STOP conditions are binding.
> Commit on branch `advisor/037-quest-rail-oob`. SKIP updating
> `plans/readme.md`. Audit every report claim against a tool result.
>
> **Drift check (run first)**: `git diff --stat 83ccb1e..HEAD -- internal/web/tasks.go web/templates/tasks.html internal/web/tasks_test.go`
> Any drift → compare excerpts; on mismatch, STOP.

## Status

- **Priority**: P3 · **Effort**: S–M · **Risk**: LOW–MED (the transition
  endpoint serves three surfaces — chat, boards, tasks page)
- **Depends on**: 035 (DONE, merged) · **Category**: direction
- **Planned at**: commit `83ccb1e`, 2026-06-12

## Why this matters

Plan 035's known gap: completing/snoozing/dropping a task in the quest-log
detail panel leaves the left rail stale until reload. Fix: when the
transition request comes from the /tasks page, the response additionally
carries an out-of-band re-render of the rail, so the row moves/strikes
immediately.

## Current state

- `internal/web/tasks.go:328` `taskTransition`: loads the record, switches on
  `to` (done/dropped/snooze via `tasks.Done/Drop/Snooze`), then re-renders
  the card (read the function tail for the exact render call; drop may
  render empty). The endpoint is used by: chat task cards, board
  today/quests card rows (`hx-swap="delete"` — response body discarded), and
  the quest-log detail panel (`hx-target="#tcard-{id}" hx-swap="outerHTML"`).
- The quest-log rail: `tasks_list` define in `web/templates/tasks.html`
  renders `.quest-log` → `<nav class="quest-rail">` (groups built by
  `buildQuestLog` in `internal/web/tasks.go`; groups: Dailies, Rituals,
  Quests, Side quests + a done-recently `<details>`). The rail nav currently
  has NO id.
- HTMX OOB: elements in a response bearing `hx-swap-oob="outerHTML"` are
  swapped by id wherever they exist in the DOM; if the id is absent the
  element is discarded. HTMX sends the `HX-Current-URL` request header with
  the page URL — the server can gate OOB content on the requesting page.
- Test harness: `internal/web/tasks_test.go` (quest-log tests),
  `handlers_test.go` ApiScenario pattern; existing `TestTaskTransition`
  asserts the card-swap behavior — must stay green.

## Commands

| Purpose | Command | Expect |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tests | `go test ./...` | ok |
| Vet/fmt | `go vet ./...` / `gofmt -l .` | clean |

## Scope

**In scope**: `internal/web/tasks.go`, `web/templates/tasks.html`,
`internal/web/tasks_test.go` (and `handlers_test.go` only if the existing
transition test needs an updated assertion).
**Out of scope**: `card-task.html`; the transition semantics in
`internal/tasks`; chat/board templates; routes; CSS (reuse existing classes).

## Git workflow

Branch `advisor/037-quest-rail-oob`; commit
`feat(tasks): quest rail refreshes out-of-band after panel transitions`.

## Steps

### Step 1: Name the rail

In `tasks_list`, extract the rail into its own define so it can render
standalone:

```html
{{define "quest_rail"}}
<nav class="quest-rail" id="quest-rail" {{if .OOB}}hx-swap-oob="outerHTML"{{end}}>
  …existing groups + done-recently markup, unchanged…
</nav>
{{end}}
```

`tasks_list` includes it where the nav was. The rail's view model gains an
`OOB bool` (wrap the existing questLogView or pass a small struct — keep it
dumb).

### Step 2: Conditional OOB in the handler

In `taskTransition`, after the state change and BEFORE/AFTER the existing
card render (order: card first, OOB after — htmx processes both): if the
request's `HX-Current-URL` header parses to a path equal to `/tasks`
(use `url.Parse`; ignore query — the list view is the default `?view=list`
or bare; calendar/timeline pages also live at /tasks with `?view=` — an OOB
rail arriving there is discarded because `#quest-rail` is absent from those
views… verify that claim by reading which views render the rail; if calendar
also contained it that would still be correct data), rebuild the quest-log
groups (`buildQuestLog`) and execute `quest_rail` with OOB=true into the
same response writer.

Keep the non-/tasks paths byte-identical (chat embeds and board rows must
not grow OOB payloads).

### Step 3: Tests

- New: transition POST with `HX-Current-URL: http://127.0.0.1:8090/tasks` →
  response contains BOTH `tcard-` (or the drop-empty behavior, match
  existing) AND `id="quest-rail"` with `hx-swap-oob`; the completed task no
  longer appears inside an open rhythm group in that OOB fragment.
- New: transition POST without the header (or with a chat URL) → response
  does NOT contain `quest-rail`.
- Existing `TestTaskTransition` and quest-log tests stay green.

## Done criteria

- [ ] Panel transition from /tasks updates the rail in one response (tests
      prove presence + content of the OOB fragment)
- [ ] Non-tasks surfaces' responses unchanged (test proves absence)
- [ ] All gates clean; only in-scope files (`git status`)

## STOP conditions

- `taskTransition`'s tail differs materially from the description (e.g. it
  already does OOB work) — re-survey before editing.
- Gating on `HX-Current-URL` proves unreliable in the test harness (header
  not passing through ApiScenario) — report; do not switch to a hidden form
  field without flagging it (that would touch the shared card template).

## Maintenance notes

- If the tasks page later gets per-group hx-get pagination, the whole-rail
  OOB stays correct but gets heavier; revisit then.
- Reviewer: confirm the OOB fragment is appended OUTSIDE the card markup
  (sibling, not nested inside the swapped card).
