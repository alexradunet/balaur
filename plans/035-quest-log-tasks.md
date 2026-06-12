# Plan 035: Quest-log /tasks — two-pane rail + sticky detail panel

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report — do not improvise. Commit on branch
> `advisor/035-quest-log-tasks`. SKIP updating `plans/readme.md` (reviewer
> maintains it). Audit every report claim against a tool result.
>
> **Drift check (run first)**: `git diff --stat 90bc397..HEAD -- web/templates/tasks.html internal/web internal/tasks web/static/basm.css`
> Plan 032 may be landing concurrently (boards files, basm.css Boards
> section) — expected, unrelated. Drift in tasks templates/handlers →
> compare excerpts; on mismatch, STOP.

## Status

- **Priority**: P2 · **Effort**: M–L · **Risk**: MED (restructures the main
  list view of a shipped page)
- **Depends on**: 025 (DONE) · **Category**: direction
- **Planned at**: commit `90bc397`, 2026-06-12

## Why this matters

The mockup's tasks page is a quest log: a left rail of quests grouped by
rhythm, and a sticky right panel showing the selected quest's detail
(due, recurrence, notes, actions). Owner decisions (2026-06-12): groups are
**derived from recurrence** — Dailies (rule `daily` or `every:1d`),
Rituals (any other recurrence: `every:Nd`, `weekly:*`, `monthly:*`),
Quests (one-offs with a due date), Side quests (one-offs without).
"Campaigns" is omitted — no multi-step concept exists (honesty-ledger rule:
never imply unshipped structure). Overdue stays flagged red within groups.
Calendar and timeline views are untouched.

## Current state

- `web/templates/tasks.html`: `tasks_list` define (line 26) renders 5
  `.k-section` blocks (the current Overdue/Today/Upcoming/Someday/Done
  buckets) of full `card-task.html` cards; `tasks_calendar` (66) and
  `tasks_timeline` (96) defines — DO NOT TOUCH those two. The page nav is
  `.k-tabs.t-views` (list/calendar/timeline links via `?view=`).
- `internal/web` builds the list data — find the view-model builder
  (grep `bucketsView\|tasks_list` in `internal/web/*.go`) and read it before
  changing anything; the handler is `h.tasksPage` (web.go route
  `GET /tasks`).
- Recurrence: `internal/tasks/recur.go` — `Parse(s) (Rule, error)`;
  `Rule{Kind: ""|"daily"|"every"|"weekly"|"monthly", N, Weekdays, MonthDay}`;
  `IsZero()` means one-off. Task records: collection `tasks`, fields include
  `title`, `notes`, `status` (open/done/snoozed/dropped), `due`, `recur`
  (read the existing builder for exact names).
- Card actions: `card-task.html` posts to `/ui/tasks/{id}/transition`
  (`to=done|snooze|dropped`), swaps `#tcard-{id}` outerHTML — the detail
  panel will reuse this exact template so actions keep working.
- Existing fragment route `GET /ui/tasks/{id}/card` → `h.taskCard` renders
  `card-task.html` — the detail panel can lean on it.
- basm.css has `.k-tabs`, `.kcard`, `.tcard-*`, `.stitch`; no two-pane rules
  yet.

## Commands

| Purpose | Command | Expect |
|---|---|---|
| Build | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Tests | `go test ./...` | ok |
| Vet/fmt | `go vet ./...` / `gofmt -l .` | clean |

## Scope

**In scope**: `web/templates/tasks.html` (the `tasks_list` define + page
shell only), the tasks view-model builder file in `internal/web/` (extend —
pure additions preferred), `internal/web/web.go` (one new fragment route if
needed), `internal/web/*_test.go`, `web/static/basm.css` (a
`/* ── Quest log ── */` appendix at END of file, ~40 lines),
`internal/self/knowledge.md` + `DESIGN.md` ledger (the /tasks description
changes: quest-log list view).
**Out of scope**: `tasks_calendar`/`tasks_timeline` defines; `card-task.html`
markup; `/ui/tasks/{id}/transition` semantics; `internal/tasks` domain
package (read-only — categorization is a WEB view concern; put the
grouping func in `internal/web`); boards/chat files.

## Git workflow

Branch `advisor/035-quest-log-tasks`; commit
`feat(tasks): quest-log list view — grouped rail + sticky detail panel`.

## Steps

### Step 1: Grouping (Go, in `internal/web`)

A pure function (unit-testable) next to the existing tasks view code:

```go
// questGroup buckets an open task by rhythm: Dailies (daily / every:1d),
// Rituals (any other recurrence), Quests (one-off with due),
// Side quests (one-off without due).
func questGroup(recur string, hasDue bool) string
```

(uses `tasks.Parse`; a recur string that fails to parse counts as one-off —
forgiving, consistent with how the domain treats bad rules elsewhere — verify
that claim by reading how the builder handles Parse errors today and match
it). Extend the list view model: groups in fixed order Dailies, Rituals,
Quests, Side quests, each with its open tasks (keep the existing per-task
fields: ID, Title, DueLine, Overdue, RecurLine, Status); done-recently keeps
its own trailing section (collapsed `<details>`), and overdue tasks stay in
their rhythm group with the red flag (no separate Overdue section in the
quest-log view).

**Verify**: table-driven unit test for `questGroup` (daily, every:1d,
every:3d, weekly:mon, monthly:1, bad-rule+due, bad-rule no due, empty+due,
empty no due) → `go test ./internal/web/...` ok.

### Step 2: Two-pane template

Rewrite ONLY the `tasks_list` define:

```html
<div class="quest-log">
  <nav class="quest-rail">
    {{range .Groups}}
    <section class="quest-group">
      <h3 class="quest-group-title">{{.Name}} <span class="k-count">{{len .Tasks}}</span></h3>
      <ul>
        {{range .Tasks}}
        <li><button class="quest-row{{if .Overdue}} quest-overdue{{end}}"
              hx-get="/ui/tasks/{{.ID}}/card" hx-target="#quest-detail" hx-swap="innerHTML">
            {{.Title}}{{with .DueLine}}<span class="quest-due">{{.}}</span>{{end}}
        </button></li>
        {{end}}
      </ul>
    </section>
    {{end}}
    <details class="quest-done">…done recently rows…</details>
  </nav>
  <aside class="quest-detail" id="quest-detail">
    {{if .First}}…server-render the first task's card here (same data path as
    /ui/tasks/{id}/card, so the panel is never empty when tasks exist)…{{else}}
    <p class="k-empty">No quests yet. Speak one in the chat.</p>{{end}}
  </aside>
</div>
```

Rail rows load the EXISTING task card into the panel via the EXISTING
`/ui/tasks/{id}/card` route — no new route needed. The card's transition
forms keep their `hx-target="#tcard-{id}"` self-swap and continue to work
inside the panel. (Known cosmetic gap, accept and note it: after Done in the
panel, the rail row doesn't update until reload — fixing it needs OOB swaps
in the transition handler, which is out of scope; record in NOTES and
maintenance.)

The page keeps the `t-views` tabs and an `?view=` contract unchanged.

**Verify**: `go test ./internal/web/...` → ok; update list-view assertions
(tests grepping for the old bucket headings).

### Step 3: CSS appendix

`/* ── Quest log ── */` at END of basm.css: `.quest-log { display:grid;
grid-template-columns: minmax(0,5fr) minmax(0,7fr); gap:20px;
align-items:start }`, `.quest-detail { position:sticky; top:80px }`,
`.quest-row` (full-width text button, parchment hover, mono due line,
`.quest-overdue` ember-red), `.quest-group-title` (mono, muted, stitch
underline), single-column collapse under 860px (panel above rail). Tokens
only, no hexes.

### Step 4: Docs

DESIGN.md ledger: update the `/tasks page:` clause — "quest-log list (rhythm
groups: Dailies/Rituals/Quests/Side quests, rail + sticky detail), month
calendar, 14-day timeline". knowledge.md: same one-line truth.

**Verify**: `grep -n "quest" DESIGN.md internal/self/knowledge.md | head` → ≥1 each.

## Test plan

- `questGroup` table (Step 1).
- Handler test: `/tasks` (list view) with seeded tasks of each rhythm → 200,
  contains all four group titles in order, the daily task under Dailies,
  `quest-detail` containing the first task's `tcard-` id; with zero tasks →
  the empty line.
- Existing calendar/timeline tests untouched and green.

## Done criteria

- [ ] Four rhythm groups render in fixed order; mapping proven by unit test
- [ ] Detail panel server-renders the first task and swaps via the existing
      card route (no new transition logic)
- [ ] Calendar/timeline defines byte-identical (`git diff` shows no hunks
      in them)
- [ ] All gates clean; no out-of-scope files (`git status`)

## STOP conditions

- The tasks view-model builder is entangled such that adding groups means
  rewriting calendar/timeline data paths.
- You find yourself adding fields to `internal/tasks` or new transition
  endpoints — out of scope; report.

## Maintenance notes

- Known gap: rail row staleness after a panel-side transition (needs OOB
  swap in `taskTransition` — small follow-up; spec on request).
- "Campaigns" returns only when a real multi-step task concept ships.
- Reviewer: manual pass — group correctness against real data, sticky panel
  behavior, mobile collapse, done-recently fold.
