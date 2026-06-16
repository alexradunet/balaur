# 01 — Port the Quests focus body to a gomponents component

> **Read `plans/ui-domain-pages/README.md` first** — it has the shared port
> recipe, conventions, conflict map, and verification gates this plan assumes.
> **Read the exemplar**: `internal/feature/lifecards/lifelogfocus.go` +
> `lifelog.go` (`registerLifelog`) + `internal/web/cards.go` (`cardFocusHTML`) +
> `internal/web/focus.go` (`focusBodyHTML`). Stamped against commit `884d692`.

## Context / why

`/focus/quests` (the quest log — the prominent Quests domain page) still renders
its body from the legacy `web/templates/quests-focus.html` template via
`(*handlers).questsFocusHTML`. Every other part of the redesign now uses
gomponents components; this ports the quests body to one, filling the
`CardSize.Focus` seam (the `lifelog` body was ported the same way in `884d692`).

The quests body has one wrinkle the lifelog body didn't: **the rail is
re-rendered by a handler**. `(*handlers).taskTransition`, when the request's
`Referer` is `/focus/quests`, re-renders the whole `quest_rail` and patches
`#quest-rail`. So the rail can't just live in the new component — it must become
a **shared rail renderer** used by both the focus body and `taskTransition`.

## Current state (read these)

`web/templates/quests-focus.html` defines two templates:

- `quest_rail` (`#quest-rail`): rhythm-grouped rail. Per group:
  `<section class="quest-group"><h3 class="quest-group-title">{Name} <span class="k-count">{n}</span></h3><ul><li><button class="quest-row[ quest-overdue]" data-on:click="@get('/ui/tasks/{ID}/card')">{Title}<span class="quest-due">{DueLine}</span></button></li>…`.
  Empty: `<p class="k-empty">No quests yet. Speak one in the chat.</p>`. Then an
  optional `<details class="quest-done"><summary class="quest-group-title">Done recently <span class="k-count">{n}</span></summary>…</details>`.
- `tasks_list`: `<div class="quest-log">{quest_rail}<aside class="quest-detail" id="quest-detail">{card-task.html of First, or the k-empty}</aside></div>`.

Handlers/builders (read for exact line numbers — they may have drifted):
`internal/web/tasks.go` — `questsFocusHTML` (renders `tasks_list`),
`buildQuestLog` (groups open + done-recently into `questLogView`),
`loadQuestLogRecs`, `taskCard` (GET `/ui/tasks/{id}/card`), `taskTransition`
(POST `/ui/tasks/{id}/transition`). View-models: `questLogView{Groups
[]questGroupView, First *taskView, DoneRecently []taskView}`,
`questGroupView{Name string, Tasks []taskView}`, `taskView{ID,Title,Notes,Status,
DueLine string, Overdue bool, RecurLine string}`.

The existing gomponents task card is `internal/feature/taskcards/taskcard.go`
`TaskCard(TaskView)` — root `id="tcard-{id}"`, class `kcard tcard tcard-{status}`,
and the Done/Snooze/Drop forms `@post('/ui/tasks/{id}/transition',
{contentType:'form'})`. It mirrors `card-task.html`. `TaskView` mirrors `taskView`.

## Action contract — MUST be preserved byte-for-byte

| Trigger | Endpoint | Fields | SSE target & mode | Handler |
|---|---|---|---|---|
| rail button click | `@get('/ui/tasks/{id}/card')` | — (id in path) | `#quest-detail` **inner** | `taskCard` |
| detail card Done/Snooze/Drop | `@post('/ui/tasks/{id}/transition', {contentType:'form'})` | `to`=done\|snooze\|dropped; `until`=1h\|tonight\|tomorrow (snooze) | on `Referer:/focus/quests` → `#quest-rail` **outer**; else `#tcard-{id}` **outer** | `taskTransition` |

The detail card's transition forms come from `TaskCard` (already correct — do not
re-implement them). The rail's `quest-row` buttons must keep
`data-on:click="@get('/ui/tasks/{id}/card')"` (no `__prevent` — it matches the
template). Element ids `#quest-rail`, `#quest-detail`, and the card's `#tcard-{id}`
are load-bearing.

> Note: the `src=quests` / `#urow-quests-{id}` removal path in `taskTransition`
> belongs to the **tile** (`QuestsCard` board rows), NOT this focus rail — leave
> it alone; the focus rail refreshes whole via the `Referer` path.

## Scope

**In scope (edit):** `internal/feature/taskcards/` (new `questsfocus.go` + the
`registerQuests` dispatch), `internal/web/tasks.go` (point `taskTransition`'s
rail-refresh at the shared renderer; drop the `questsFocusHTML` body if dead),
`internal/web/focus.go` (drop the `case "quests"` arm),
`web/templates/quests-focus.html` (retire `quest_rail`/`tasks_list` once unused),
`internal/feature/storybook/{stories_cards.go,story.go}` (story).

**Out of scope (do NOT touch):** `cardInto`/`cardSizeInto`/`cardFocusHTML` in
`cards.go` (the seam already exists), the `taskCard` handler's card markup and the
transition handler's *logic* (only its rail-refresh *render call* changes), the
`QuestsCard`/`QuestsManageCard` tiles and the `src=quests` tile-row path, anything
in the README's Ollama conflict list.

## Steps

1. In `internal/feature/taskcards/questsfocus.go` add the view-models
   (`QuestsFocusView{Groups []QuestGroupView, First *TaskView, DoneRecently
   []TaskView}`, `QuestGroupView{Name string, Tasks []TaskView}` — reuse the
   existing `TaskView`) and `buildQuestsFocus(app)` mirroring `web.buildQuestLog`
   + `loadQuestLogRecs` (reuse the `internal/tasks` domain; do not import
   `internal/web`). Map records → `TaskView` the same way `taskcards` already does
   for its tiles (find the existing `taskViewOf`-equivalent in the package and
   reuse it).
2. Add **`QuestRail(v QuestsFocusView) g.Node`** — a byte-faithful port of the
   `quest_rail` template (`<nav class="quest-rail" id="quest-rail">` … the groups,
   the `quest-row`/`quest-overdue`/`quest-due`/`k-count`/`k-empty` classes, the
   `Done recently` `<details class="quest-done">`). Each row button:
   `g.El("button", g.Attr("class","quest-row"…), g.Attr("data-on:click",
   "@get('/ui/tasks/"+t.ID+"/card')"), …)`.
3. Add **`QuestsFocus(v QuestsFocusView) g.Node`** — port of `tasks_list`:
   `<div class="quest-log">` + `QuestRail(v)` + `<aside class="quest-detail"
   id="quest-detail">` containing `TaskCard(*v.First)` when set, else the `k-empty`.
4. `registerQuests` (in `taskcards`): dispatch on size — `ui.Focus` →
   `QuestsFocus(buildQuestsFocus(app))`, else the existing tile. (Mirror
   `registerLifelog`.)
5. In `internal/web/focus.go` `focusBodyHTML`, **delete `case "quests": return
   h.questsFocusHTML()`** so quests falls through to the `cardFocusHTML` seam.
6. In `internal/web/tasks.go` `taskTransition`, replace the rail-refresh
   `ExecuteTemplate(&rb, "quest_rail", …)` with a render of the shared rail
   component: `taskcards.QuestRail(taskcards.BuildQuestsFocus(h.app))` (export
   `buildQuestsFocus`/`QuestRail` as needed) → `PatchElements(rb.String(),
   WithSelectorID("quest-rail"), WithModeOuter())`. Keep the patch selector/mode
   and all transition logic identical. **If `taskTransition` builds the rail from
   already-loaded recs in a way that doesn't map cleanly to `BuildQuestsFocus`,
   STOP and report — do not duplicate the rail markup in two places.**
7. Retire `questsFocusHTML` and the `quest_rail`/`tasks_list` template defines
   **only after** step 6 (so the rail-refresh no longer needs the template).
   First `grep -rn 'quest_rail\|tasks_list\|questsFocusHTML' --include='*_test.go'
   internal/web` — if a test executes them, leave them as dead code and add a
   `// TODO(ui-redesign): retire once <test> is reconciled` note instead.
8. Storybook: `questsfocusStory()` in `stories_cards.go` (variants: populated
   rail + detail, empty), registered mid-Cards-cluster in `story.go`. + a
   `questsfocus_test.go` asserting `quest-rail`, `quest-detail`, `quest-row`,
   `quest-group-title`, `k-empty`, and a row's `@get('/ui/tasks/…/card')`.

## Done criteria (machine-checkable)

- `CGO_ENABLED=0 go build ./...` → exit 0.
- `CGO_ENABLED=0 go test ./internal/feature/taskcards/... ./internal/web/... ./internal/feature/storybook/...` → ok.
- `gofmt -l internal/feature/taskcards internal/web internal/feature/storybook` → empty.
- Live: serve, then `curl -s 127.0.0.1:PORT/focus/quests` contains
  `id="quest-rail"`, `id="quest-detail"`, `class="topbar"`,
  `href="/focus/quests" aria-current="page"`, `id="dock"`; and seed one open task
  (see `internal/web/focus_test.go::TestFocusQuestsShowsRail` for the seeding
  pattern) and assert the task title + a `quest-row` appears.
- Existing `internal/web/focus_test.go::TestFocusQuestsShowsRail` still passes
  (it asserts `#quest-rail`, `#quest-detail`, the seeded task).

## Test plan

Add `internal/feature/taskcards/questsfocus_test.go` (mirror
`internal/feature/lifecards/lifelogfocus_test.go`): assert the class/id contract
on a populated `QuestsFocusView` and the empty state. Keep / re-run
`TestFocusQuestsShowsRail` and `TestTaskTransition*` (the rail-refresh path).

## Maintenance note

The rail now has one renderer (`QuestRail`) used by the focus body and
`taskTransition` — future rail changes touch one place. Watch in review that the
`@get('/ui/tasks/{id}/card')` (no `__prevent`) and `#quest-rail`/`#quest-detail`
ids stay exact. A natural follow-up (separate plan): port the `taskCard` handler
and `taskTransition`'s `#tcard-{id}` re-render to `TaskCard` too, then retire
`card-task.html`.

## Escape hatches

- If `TaskCard`'s markup has diverged from `card-task.html` (what `taskCard`
  patches into `#quest-detail` on click), the initial detail and the clicked
  detail would differ — STOP and report; either align `taskCard` to render
  `TaskCard` in the same change, or keep the initial detail rendering via the
  existing handler path.
- If anything you need pulls in `internal/web/models.go`, `internal/kronk`, or the
  other README conflict files — STOP; you're out of scope.
</content>
