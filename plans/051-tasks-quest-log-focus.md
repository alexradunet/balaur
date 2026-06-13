# Plan 051: Tasks — quest-log focus, retire /tasks (Phase 1)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan in
> `plans/readme.md`. To execute task-by-task with review checkpoints, use
> `superpowers:subagent-driven-development` or `superpowers:executing-plans`.
>
> **Drift check (run first)**: `git diff --stat e1788a8..HEAD -- internal/web web/templates`
> Authored at `e1788a8` (Phase 0 / plan 050 merged). The program spec is
> `docs/superpowers/specs/2026-06-13-card-first-kill-the-pages-design.md`.
> If `internal/web/tasks.go`, `internal/web/focus.go`, `internal/web/cards.go`,
> `web/templates/tasks.html`, `web/templates/cards.html`, or
> `web/templates/layout.html` changed since `e1788a8`, compare excerpts; on
> mismatch, STOP.

## Status

- **Priority**: P1 (Phase 1 of the card-first program; first page retired)
- **Effort**: M
- **Risk**: MED (deletes a live page + route; re-points transition refresh and
  several tile/footer links; test surgery)
- **Depends on**: plans/050-card-focus-mechanism.md (hard, DONE/merged)
- **Category**: direction (card-first "kill the pages", Phase 1 of 8)
- **Planned at**: commit `e1788a8`, 2026-06-13

## Why this matters

`/tasks` is the first page we retire. Its rich "list" view (the rhythm-grouped
quest rail + sticky detail) becomes the **quests card's focus view**, so the
operational task surface lives inside a composable card instead of a standalone
page. Its calendar/timeline tabs are already covered by the `calendar`/`timeline`
cards' focus (plan 050), so they need no port. Tasks are created via chat
(`tasks.Create` from the agent tools — the page never had a create form; its
empty state literally says "Speak one in the chat"), so **strict parity** is the
scope: no new web write path. After parity, `/tasks` and its page-only code are
deleted, and its topbar link is dropped (you cannot delete a route and leave a
nav link pointing at it).

## Current state

- **Focus mechanism** (plan 050): `focusPage` (`internal/web/focus.go`) renders a
  card at full canvas. Today the body is `h.cardHTML(typ, params)` after
  `focusParams` defaults `mode=manage` for `HasManage` types — so the quests
  focus is currently the flat `ucard_quests_manage` list. This plan gives quests
  a bespoke focus body.
- **The /tasks page** (`internal/web/tasks.go`):
  - `tasksPage` (`tasks.go:141-167`): `?view=` ∈ {list,calendar,timeline}; list →
    `buildQuestLog(open, done, now)` → renders `tasks.html` with `QuestLog`.
  - `buildQuestLog` (`tasks.go:97-139`) → `questLogView{Groups, First,
    DoneRecently}`; `questGroup` buckets by rhythm (Dailies/Rituals/Quests/Side
    quests). **Keep these — the focus reuses them.**
  - `buildTimeline` (`tasks.go:277-311`) is used **only** by `tasksPage` → dead
    after deletion. (`buildTimelineN` in `cards.go:541` powers the timeline card;
    the shared types `tlView/tlItem/tlDay` at `tasks.go:261-275` and the const
    `timelineDays` at `tasks.go:259` are used by the card — **keep them**.)
  - `buildCalendar` (`tasks.go:190`) is used by `tasksPage` **and**
    `renderCardCalendar` (`cards.go:339`) — **keep it.**
- **Templates** (`web/templates/tasks.html`):
  - `quest_rail` (`tasks.html:15-47`): `<nav id="quest-rail">`; rows
    `data-on:click="@get('/ui/tasks/{{.ID}}/card')"`. **Re-used by the focus and
    by `taskTransition`'s rail refresh — must survive.**
  - `tasks_list` (`tasks.html:49-59`): the `.quest-log` wrapper = `quest_rail` +
    `<aside id="quest-detail">` pre-rendered with `card-task.html` of `.First`.
    **This is the focus body — must survive.**
  - `tasks_calendar` (`:61`), `tasks_timeline` (`:91`): page-only; the cards have
    their own `ucard_calendar`/`ucard_timeline`. **Dead after deletion.**
- **Detail + transitions:**
  - `taskCard` (GET `/ui/tasks/{id}/card`, `tasks.go:317-329`): inner-patches
    `#quest-detail` with `card-task.html`. **Stays — works inside the focus.**
  - `taskTransition` (POST `/ui/tasks/{id}/transition`, `tasks.go:341-413`):
    Done/Snooze/Drop. `src=today|quests` board rows self-remove (returns early);
    otherwise replaces `#tcard-{id}`; and **if `Referer` path == `/tasks` (list
    view), also re-renders `#quest-rail`** (`tasks.go:395-411`). `card-task.html`
    posts carry **no `src`**, so the detail card uses the replace-`#tcard-{id}` +
    rail-refresh path. **Re-point the Referer check from `/tasks` to
    `/focus/quests`.**
- **Tile/footer links to `/tasks`** (break after deletion): `cards.html:33,65,78`
  (`all quests →` → should be `/focus/quests`), `cards.html:116`
  (`full calendar →` → `/focus/calendar`), `cards.html:144` (`full timeline →` →
  `/focus/timeline`), plus a stale comment at `cards.html:72`.
- **Topbar** (`layout.html:26`): `<a href="/tasks">Tasks</a>` — remove.
- **Route**: `se.Router.GET("/tasks", h.tasksPage)` (`web.go:202`).
- **Tests** that touch the page: `tasks_test.go` GET `/tasks` scenarios
  (`:84,103,118,130,149,169`) and `templates_test.go` `tasks.html` renders
  (`:141-142,229-230`).
- **Test harness**: `tests.ApiScenario{...,TestAppFactory:newWebApp}` and
  `&handlers{app: newWebApp(t)}` (see `internal/web/boards_test.go`).

## Commands you will need

```bash
go test ./internal/web/...
go test ./... && go vet ./... && gofmt -l internal web && CGO_ENABLED=0 go build ./...
# After the deletion step, prove no dangling references:
grep -rnE '"/tasks"|href="/tasks|tasksPage|"tasks\.html"|tasks_calendar|tasks_timeline|buildTimeline\b' internal web --include='*.go' --include='*.html'
```

## Scope

**In:** a bespoke quests focus body (the quest-log rail + detail) via a focus
dispatch seam; re-point the transition rail-refresh + tile footers to the focus
routes; delete `/tasks` (route, `tasksPage`, `buildTimeline`, `tasks.html`'s
page-only templates) and the `/tasks` topbar link; move `quest_rail`+`tasks_list`
to a surviving template file; adapt tests.

**Out:** task create/edit (chat creates tasks — strict parity, confirmed with
owner); any change to the quests **tile** (only its focus changes); other pages
(later phases); other topbar links (their phases).

## Git workflow

Branch `feature/card-first-kill-pages` (already synced to `main` @ `e1788a8`).
Commit after each green step. Steps A–C are additive/green-keeping; Step D is the
deletion (largest); Step E is docs.

## Steps

### Step A: quests gets a bespoke focus body (additive — page still lives)

**File:** `internal/web/focus.go` — add a dispatch seam and use it.

Add:

```go
// focusBodyHTML renders a card's focus body. A few card types have a bespoke,
// richer focus view (the surface of the page they replace); every other type
// falls back to the generic registry render (manage mode where available).
func (h *handlers) focusBodyHTML(typ string, params map[string]string) template.HTML {
	switch typ {
	case "quests":
		return h.questsFocusHTML()
	}
	return h.cardHTML(typ, params)
}
```

In `focusPage`, change the body line from `Body: h.cardHTML(typ, params),` to:

```go
		Body:     h.focusBodyHTML(typ, params),
```

**File:** `internal/web/tasks.go` — add `"html/template"` to the import block
(the new renderer returns `template.HTML`; `tasks.go` does not import it yet),
then add the quests focus renderer (place it just after `buildQuestLog`, near
`tasks.go:139`):

```go
// questsFocusHTML renders the quests card's focus body: the rhythm-grouped quest
// rail + sticky detail — the surface formerly at /tasks?view=list. Tasks are
// created in chat, so this view is read + transition only (strict parity).
func (h *handlers) questsFocusHTML() template.HTML {
	now := time.Now()
	openRecs, _ := tasks.OpenTasks(h.app, nil)
	var doneRecs []*core.Record
	if dr, err := h.app.FindRecordsByFilter("tasks", "status = 'done'", "-updated", 6, 0); err == nil {
		doneRecs = dr
	}
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "tasks_list", map[string]any{
		"QuestLog": buildQuestLog(openRecs, doneRecs, now),
	}); err != nil {
		h.app.Logger().Warn("quests focus render failed", "err", err)
		return cardErrorStrip("could not render the quest log")
	}
	return template.HTML(b.String())
}
```

(`cardErrorStrip` lives in `internal/web/cards.go:138`. `tasks_list` still lives
in `tasks.html` at this step — Step D moves it; this keeps the build green.)

**Verify:** `go build ./... && go test ./internal/web/ -run 'TestFocus|TestBoards'` → ok.

**Test (add to `internal/web/focus_test.go`):**

```go
// TestFocusQuestsShowsRail: /focus/quests renders the rhythm rail (not the flat
// manage list), so expanding the quests card gives the full quest-log surface.
func TestFocusQuestsShowsRail(t *testing.T) {
	app := newWebApp(t)
	// Seed one open task so the rail has a group.
	col, err := app.FindCollectionByNameOrId("tasks")
	if err != nil {
		t.Fatalf("tasks collection: %v", err)
	}
	rec := core.NewRecord(col)
	rec.Set("title", "Walk the dog")
	rec.Set("status", "open")
	if err := app.Save(rec); err != nil {
		t.Fatalf("seed task: %v", err)
	}
	s := tests.ApiScenario{
		Name:           "GET /focus/quests shows the quest rail",
		Method:         "GET",
		URL:            "/focus/quests",
		TestAppFactory: func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`id="quest-rail"`,
			`id="quest-detail"`,
			"Walk the dog",
		},
	}
	s.Test(t)
}
```

> If `core`/`tests` imports aren't already in `focus_test.go`, add
> `"github.com/pocketbase/pocketbase/core"`. Confirm the seed shape against how
> `tasks_test.go` creates a task (read it: `grep -n "NewRecord\|Set(" internal/web/tasks_test.go`)
> and match it (e.g. any required fields). Run
> `go test ./internal/web/ -run TestFocusQuestsShowsRail -v` → PASS.

**Commit:** `git add internal/web/focus.go internal/web/tasks.go internal/web/focus_test.go && git commit -m "feat(focus): quests focus = the rhythm quest-log rail + detail"`

### Step B: re-point the transition rail-refresh to the focus

**File:** `internal/web/tasks.go` — in `taskTransition`, the Referer block
(`tasks.go:391-411`). The page is going away; the quest-log now lives at
`/focus/quests`. Replace the block:

```go
	// The quest-log surface (now the quests focus at /focus/quests) shows a rail
	// that must re-render after a transition so the row moves/strikes. A Datastar
	// @post is a plain fetch, so we identify the surface by Referer. Detail-panel
	// cards carry no "src", so they reach here (board tiles returned above).
	if ref := e.Request.Header.Get("Referer"); ref != "" {
		if u, err := url.Parse(ref); err == nil && u.Path == "/focus/quests" {
			openRecs, _ := tasks.OpenTasks(h.app, nil)
			var doneRecs []*core.Record
			if dr, err := h.app.FindRecordsByFilter("tasks", "status = 'done'", "-updated", 6, 0); err == nil {
				doneRecs = dr
			}
			var rb strings.Builder
			if err := h.tmpl.ExecuteTemplate(&rb, "quest_rail", buildQuestLog(openRecs, doneRecs, now)); err != nil {
				return e.InternalServerError("rendering quest rail", err)
			}
			_ = sse.PatchElements(rb.String(),
				datastar.WithSelectorID("quest-rail"), datastar.WithModeOuter())
		}
	}
	return nil
```

**Verify:** `go build ./... && go test ./internal/web/ -run TestTaskTransition` → ok
(existing transition tests should still pass; the `/tasks`-Referer test, if any,
will be updated in Step D when the page is removed — for now both paths build).

**Test (add to `internal/web/tasks_test.go`):** a transition from the focus
refreshes the rail.

```go
func TestTaskTransitionRefreshesFocusRail(t *testing.T) {
	app := newWebApp(t)
	h := &handlers{app: app}
	col, _ := app.FindCollectionByNameOrId("tasks")
	rec := core.NewRecord(col)
	rec.Set("title", "Water the plants")
	rec.Set("status", "open")
	if err := app.Save(rec); err != nil {
		t.Fatalf("seed: %v", err)
	}
	s := tests.ApiScenario{
		Name:   "transition with /focus/quests Referer re-renders #quest-rail",
		Method: "POST",
		URL:    "/ui/tasks/" + rec.Id + "/transition",
		Headers: map[string]string{
			"Referer":      "http://localhost/focus/quests",
			"Content-Type": "application/x-www-form-urlencoded",
		},
		Body:           strings.NewReader("to=done"),
		TestAppFactory: func(testing.TB) *tests.TestApp { return app },
		ExpectedStatus: 200,
		ExpectedContent: []string{`id="quest-rail"`},
	}
	_ = h
	s.Test(t)
}
```

> Match the POST/body/header convention to the existing `TestTaskTransition` in
> `tasks_test.go` (read it first — it may build the request differently, e.g.
> form values via a helper). Adjust to whatever that test uses. Run
> `go test ./internal/web/ -run TestTaskTransitionRefreshesFocusRail -v` → PASS.

**Commit:** `git add internal/web/tasks.go internal/web/tasks_test.go && git commit -m "feat(tasks): refresh the quest rail on transitions from /focus/quests"`

### Step C: re-point card tile footers to the focus routes

**File:** `web/templates/cards.html` — replace the page links:
- `cards.html:33` and `:65` and `:78`: `href="/tasks"` → `href="/focus/quests"`.
- `cards.html:116`: `href="/tasks?view=calendar"` → `href="/focus/calendar"`.
- `cards.html:144`: `href="/tasks?view=timeline"` → `href="/focus/timeline"`.
- `cards.html:72` comment: update "taskTransition only OOB-re-renders the rail on
  /tasks" → "...on /focus/quests".

These stay plain `<a href>` (a full nav to the focus route renders the focus
shell — the focus handler is dual-mode). Leave the ⤢ expand control (plan 050)
as the SSE-patch path; the footer link is the explicit fallback.

**Verify:** `go test ./internal/web/ -run 'TestUiCard|TestCard'` → ok;
`grep -n 'href="/tasks' web/templates/cards.html` → no matches.

**Commit:** `git add web/templates/cards.html && git commit -m "feat(cards): tile footers point to /focus/{quests,calendar,timeline}"`

### Step D: delete /tasks and its page-only code

**1. Move the surviving templates.** Create `web/templates/quests-focus.html` and
move `quest_rail` and `tasks_list` into it **verbatim** (cut from `tasks.html`):

```html
{{- /* quests-focus.html — the quest-log focus body (was /tasks?view=list).
     quest_rail and tasks_list moved here from the retired tasks.html so the
     quests focus and taskTransition's rail refresh keep one source of truth.
     card-task.html (the detail card) and the calendar/timeline cards live
     elsewhere. */ -}}

{{define "quest_rail"}}
... (paste the exact current quest_rail body from tasks.html:15-47) ...
{{end}}

{{define "tasks_list"}}
... (paste the exact current tasks_list body from tasks.html:49-59) ...
{{end}}
```

**2. Delete `web/templates/tasks.html` entirely** (`tasks_calendar`/`tasks_timeline`
die with it; the cards cover those).

**3. Remove the route.** `internal/web/web.go:202` — delete
`se.Router.GET("/tasks", h.tasksPage)`.

**4. Delete dead Go.** In `internal/web/tasks.go`:
- delete `tasksPage` (`:141-167`);
- delete `buildTimeline` (`:277-311`);
- update the file-top comment (`:19-23`) — it describes `/tasks`; rewrite to
  describe the quest-log focus + the calendar/timeline cards.
- **Keep**: `taskView/taskViewOf/taskViewsOf`, `questGroup/questGroupView/
  questLogView/buildQuestLog`, `questsFocusHTML`, `taskCard`, `taskTransition`,
  `snoozeUntil`, `buildCalendar`, `mondayOf`, the `tlView/tlItem/tlDay` types,
  `timelineDays`, `chatNudges`, and the calendar `cal*` types.

**5. Remove the topbar link.** `web/templates/layout.html:26` — delete the
`<a href="/tasks">Tasks</a>` line.

**6. Fix tests.**
- `internal/web/tasks_test.go`: remove the GET `/tasks` page scenarios (the ones
  at `:84,103,118,130,149,169` that load list/calendar/timeline). Keep the
  transition tests (and the Step B addition). Read the file and delete only the
  page-load scenarios + any now-unused helpers.
- `internal/web/templates_test.go`: the `tasks.html` renders at `:141-142` and
  `:229-230` will fail (template gone). Replace the list render with a
  `tasks_list` render (pass `map[string]any{"QuestLog": ql}`); drop the
  calendar/timeline page-template assertions — confirm the calendar/timeline
  **card** templates (`ucard_calendar`/`ucard_timeline`) already have render
  coverage (`grep -n "ucard_calendar\|ucard_timeline" internal/web/*_test.go`);
  if they do not, add a minimal render test for them instead of deleting
  coverage.

**Verify (all must hold):**
```
go test ./... && go vet ./... && gofmt -l internal web && CGO_ENABLED=0 go build ./...
grep -rnE '"/tasks"|href="/tasks|tasksPage|"tasks\.html"|tasks_calendar|tasks_timeline|buildTimeline\b' internal web --include='*.go' --include='*.html'
```
The grep MUST return nothing. `gofmt -l internal web` MUST be empty.

**Browser check (the owner will do this — note it; the sandbox has no X server):**
`/boards` → expand the quests card (⤢) → the quest-log rail + detail render in
`#main`, the dock persists; click a rail row → detail loads; Done a task → row
strikes/moves and the detail card updates; the topbar no longer shows "Tasks";
visiting `/tasks` returns 404.

**Commit:** `git add -A && git commit -m "feat(tasks): retire /tasks — quest-log lives in the quests focus"`

### Step E: docs

Update the `051` row in `plans/readme.md` to `DONE` with a one-line note
(quest-log focus + `/tasks` retired). If `DESIGN.md` or
`internal/self/knowledge.md` names `/tasks` as a page, fix those references
(`grep -rn "/tasks" DESIGN.md internal/self/knowledge.md README.md`).

**Commit:** `git add -A && git commit -m "docs(plans): 051 done — /tasks retired into the quests focus"`

## Test plan

- **Focus render** (`focus_test.go`): `/focus/quests` shows `#quest-rail` +
  `#quest-detail` + a seeded task (Step A).
- **Transition refresh** (`tasks_test.go`): a transition with `Referer:
  /focus/quests` re-renders `#quest-rail` (Step B); board-tile transitions
  (`src=today|quests`) still self-remove the row (existing test).
- **Deletion safety**: the Step D grep returns nothing; `go test ./...` green
  with the page scenarios removed and `tasks_list` covered.
- **Browser** (owner): expand → rail+detail, row click → detail, Done → rail
  refresh, dock persists, `/tasks` 404s, topbar has no Tasks link.

## Done criteria

- [ ] Quests focus renders the rhythm rail + sticky detail (not the flat manage
      list); `focusBodyHTML` dispatches quests → `questsFocusHTML`, others
      unchanged.
- [ ] `taskTransition` refreshes `#quest-rail` when `Referer` is `/focus/quests`;
      board-tile row-removal path unchanged.
- [ ] Card tile footers + the stale comment point to `/focus/*`; no
      `href="/tasks"` remains.
- [ ] `/tasks` route, `tasksPage`, `buildTimeline`, and `tasks.html` are gone;
      `quest_rail`+`tasks_list` survive in `quests-focus.html`; shared
      `buildCalendar`/`tlView`/`timelineDays` retained.
- [ ] Topbar has no Tasks link.
- [ ] Step D grep returns nothing; `go test ./...`, vet, `gofmt -l` (empty),
      CGO-free build all clean; `git diff --check` clean.
- [ ] No task create/edit added (strict parity); only the quests **focus**
      changed, not its tile.
- [ ] `plans/readme.md` 051 row → DONE; doc references to `/tasks` fixed.

## STOP conditions

- Moving `quest_rail`/`tasks_list` produces a "redefinition of template" error →
  a copy was left behind in `tasks.html`; ensure they exist in exactly one file.
- `tasks_list` references a field not present in the map you pass
  (`{{$ql := .QuestLog}}`) → pass `map[string]any{"QuestLog": ...}`, not the
  `questLogView` directly.
- The Step D grep still finds `/tasks` references you didn't expect (a template
  or doc not listed in Current state) → STOP, list them, re-point or remove
  before declaring done.
- Deleting `buildTimeline` breaks the build → you also deleted a shared type or
  `timelineDays`; restore the shared bits (only `buildTimeline` + `tasksPage`
  are dead).
- A removed test leaves a helper unused (compile error) → remove the helper too,
  or keep the test if it still covers live behavior.

## Maintenance notes

- The quests **tile** is unchanged; only its focus is the rich quest-log. The
  tile's params (status/limit) are ignored by the focus (focus = the full open
  set) — acceptable, and consistent with "focus is the whole surface".
- `focusBodyHTML` is the seam later phases extend: journal, knowledge, etc. each
  add a `case` for their bespoke focus; everything else keeps the generic render.
- Calendar/timeline got no bespoke focus — their cards' own full views are the
  surface. If the owner later wants the page's month/forward affordances in
  focus, that's a separate enhancement, not parity.
