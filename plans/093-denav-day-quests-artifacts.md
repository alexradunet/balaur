# Plan 093: Day & Quests artifacts are flat and nav-free (no date stepper, no master/detail rail)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. When done, update the status row for this plan
> in `plans/readme.md`.
>
> **Drift check (run first)**:
> `git diff --stat 766b7aa..HEAD -- internal/feature/journalcards/dayfocus.go internal/feature/taskcards/questsfocus.go internal/web/tasks.go internal/web/handlers_test.go internal/web/tasks_test.go internal/web/journal_test.go internal/web/show_test.go internal/web/templates_test.go internal/web/assets/static/basm.css internal/feature/storybook/stories_cards.go`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.
>
> **Sandbox note**: in a TLS-intercepting sandbox (Hyperagent), Go commands
> need the GOPROXY shim — see `docs/hyperagent-sandbox.md`. GOSUMDB stays on.

## Status

- **Priority**: P1
- **Effort**: M–L
- **Risk**: MED (notable test churn; one behavior change on task completion)
- **Depends on**: none (independent of 092 and 094; any order)
- **Category**: direction (UX) / tech-debt
- **Planned at**: commit `766b7aa`, 2026-06-17

## Why this matters

Two in-chat artifacts carry their own navigation, which the owner has flagged
as confusing (an artifact lives *inside* the chat; the chat is the navigation
surface — the artifact should not navigate too):

- **day** (`day-focus`) has a `day-nav` prev/next **date stepper** plus a
  recap **transcript expander** (a drill-down).
- **quests** (`quest-log`) is a **master/detail** layout: a `quest-rail` whose
  rows load a task's detail card into a side aside.

The decision (confirmed 2026-06-17): artifacts are flat, nav-free units. The
**day** artifact shows the requested day with no stepper; the **quests**
artifact shows a flat, rhythm-grouped **stack of task cards** with no rail and
no detail pane. To see another day, the owner asks the agent / re-summons.

Enabling fact: **after plan 089 retired the `/focus/{type}` pages, `ui.Focus`
is rendered only by the in-chat artifact path** (`cardFocusHTML`/`uicardBody`
in `internal/web/cards.go`) — so changing these Focus renderers affects nothing
else. And task transitions **already self-replace each card in place** via
`#tcard-{id}` (see `tasks.go:257-262`), so dropping the rail's out-of-band
refresh is clean.

## Current state

Files and their roles:

- `internal/feature/journalcards/dayfocus.go` — `DayFocusView` (view-model,
  lines 26–37), `BuildDayFocus` (42–113), `DayJournal` (write-form section,
  129–171), `DayFocus` (the renderer with the nav, 176–291).
- `internal/feature/taskcards/questsfocus.go` — `QuestsFocusView` (22–28),
  `BuildQuestsFocus` (35–45), `QuestRail` (the nav rail, 111–169), `QuestsFocus`
  (rail + detail, 173–184). `TaskCard` lives in `taskcards/taskcard.go`.
- `internal/web/tasks.go` — the transition handler; lines 250–279 hold the
  `src=` row-remove path, the `#tcard-{id}` in-place replace, and the
  `#quest-rail` out-of-band refresh keyed on `Referer == /ui/show/quests`.
- `internal/web/day.go` — `renderDayJournal` calls `BuildDayFocus` + `DayJournal`
  only (reads `.Date`/`.Journal`); it does NOT read `Prev`/`Next`/`RecapStart`.
- Tests: `handlers_test.go:501` (`day-focus`,`day-nav`,`January`),
  `tasks_test.go` (`TestQuestsArtifactEndpoint` → `quest-log`;
  `TestTaskTransitionRailRefresh` → `#quest-rail`), `journal_test.go`
  (`day-focus` in the integration test), `show_test.go` (`quest-log`),
  `templates_test.go` (quests → `quest-log`/`quest-rail`; day → `day-focus`).
- CSS: `.day-nav` (1446), `.day-nav-spacer` (1448); `.quest-log` (2302),
  `.quest-rail` (2313), `.quest-group` (~2325), `.quest-group-title` (~2328),
  `.quest-row` (~2339), `.quest-detail` (2367), `.quest-done` (~2372).
  `.tasks-stack` (the bare task-card stack from plan 090) and `.k-section` /
  `.k-heading` already exist and are reused below.

### `DayFocus` today — the nav to remove (`dayfocus.go:176-291`, abridged)

```go
func DayFocus(v DayFocusView) g.Node {
	var nextNode g.Node
	if v.Next != "" { nextNode = A(... Href("/ui/show/day?date="+v.Next) ... "next ▸") } else { nextNode = Span(Class("day-nav-spacer")) }
	titleKids := []g.Node{Class("day-title"), g.Text(v.Label)}
	if v.IsToday { titleKids = append(titleKids, g.Text(" "), Span(Class("tag"), g.Text("today"))) }
	// ... recap section (with a transcript expander button when !IsToday) ...
	// ... done section, logs section ...
	return Div(Class("day-focus"),
		Div(Class("day-nav"),
			A(... Href("/ui/show/day?date="+v.Prev) ... "◂ prev"),
			H2(titleKids...),
			nextNode,
		),
		DayJournal(v),
		Div(Class("stitch")),
		Section(recapSectionKids...),   // <- contains the transcript expander when !IsToday
		Div(Class("stitch")),
		Section(... "What got done" ... doneContent),
		Div(Class("stitch")),
		Section(... "The day's log" ... logsContent),
	)
}
```

The transcript expander inside the recap section (`dayfocus.go:211-229`, the
`Article(Class("recap-card recap-day") ... Button(Class("recap-expand") ...
@get('/ui/recap/expand?type=day&start=…') ... Div(Class("recap-children") …))`)
is the drill-down to remove; keep the recap **summary text** (`recapText`).

`DayFocusView` fields `Prev`, `Next`, `RecapStart` exist ONLY to feed the
stepper + expander and become dead after this change.

### `QuestsFocus` today — master/detail to flatten (`questsfocus.go:173-184`)

```go
func QuestsFocus(v QuestsFocusView) g.Node {
	var detail g.Node
	if v.First != nil { detail = TaskCard(*v.First) } else { detail = ui.EmptyState(...) }
	return Div(Class("quest-log"),
		QuestRail(v),                                   // <nav class="quest-rail"> rows @get('/ui/tasks/{id}/card')
		g.El("aside", Class("quest-detail"), ID("quest-detail"), detail),
	)
}
```

`QuestRail` (111–169) renders rhythm groups (`v.Groups`: Dailies / Rituals /
Quests / Side quests) as buttons that load a detail card. `BuildQuestsFocus`
already groups the open tasks (and gathers `DoneRecently`).

### Transition handler today (`tasks.go:250-279`)

```go
	if src := e.Request.FormValue("src"); src == "today" || src == "quests" {
		_ = sse.PatchElements("", datastar.WithSelectorID("urow-"+src+"-"+rec.Id), datastar.WithModeRemove())
		return nil
	}
	html, err := h.taskCardHTML(rec)               // #tcard-{id} in-place replace
	...
	_ = sse.PatchElements(html, datastar.WithSelectorID("tcard-"+rec.Id), datastar.WithModeOuter())

	// The quests artifact (/ui/show/quests) shows a rail that must re-render ...
	if ref := e.Request.Header.Get("Referer"); ref != "" {
		if u, err := url.Parse(ref); err == nil && u.Path == "/ui/show/quests" {
			// ... QuestRail(BuildQuestsFocus(h.app)) outer-patched into #quest-rail ...
		}
	}
	return nil
```

The `src=` row-remove path (board/summary tiles) is **out of scope** — leave
it. Only the `#quest-rail` block (the `Referer == /ui/show/quests` branch) is
removed: the flat stack uses full `TaskCard`s whose transition self-replaces via
`#tcard-{id}` (the line above it), so no rail refresh is needed.

### Repo conventions to match

- Match the gomponents import style already in each file
  (`g "maragu.dev/gomponents"`, dot-import `. "maragu.dev/gomponents/html"`).
- `ui.EmptyState(ui.EmptyProps{Compact: true, Line: "…"})` is the shared empty
  state (already used in `questsfocus.go`).
- Tests use `tests.ApiScenario` substring checks; `NotExpectedContent` locks
  removals.

## Commands you will need

| Purpose   | Command                                            | Expected on success |
|-----------|----------------------------------------------------|---------------------|
| Build     | `CGO_ENABLED=0 go build ./...`                     | exit 0              |
| Vet       | `go vet ./...`                                     | exit 0              |
| Test (pkg)| `go test ./internal/web/... ./internal/feature/...`| all pass            |
| Test (all)| `go test ./...`                                    | all pass            |
| Format    | `gofmt -l internal/`                               | no output           |
| Diff check| `git diff --check`                                 | no output           |

## Scope

**In scope** (the only files you should modify):
- `internal/feature/journalcards/dayfocus.go` — denav `DayFocus`; drop dead view-model fields.
- `internal/feature/journalcards/dayfocus_test.go` / `journalfocus_test.go` (whichever asserts `day-nav`/`Prev`/`Next`) — update.
- `internal/feature/taskcards/questsfocus.go` — flatten `QuestsFocus`; remove/repurpose `QuestRail`.
- `internal/feature/taskcards/questsfocus_test.go` — update to the flat stack.
- `internal/web/tasks.go` — remove the `#quest-rail` OOB-refresh block (and the now-unused `taskcards` import / `url` import IF they become unused — check).
- `internal/web/handlers_test.go`, `internal/web/tasks_test.go`, `internal/web/journal_test.go`, `internal/web/show_test.go`, `internal/web/templates_test.go` — update assertions.
- `internal/web/assets/static/basm.css` — remove orphaned `.day-nav*` / `.quest-rail` / `.quest-detail` / `.quest-log` / `.quest-row` / `.quest-done` rules.
- `internal/feature/storybook/stories_cards.go` — update the day + quests stories.

**Out of scope** (do NOT touch):
- `internal/web/cards.go`, `internal/web/show.go` — the artifact dispatch/door are correct.
- The `src=today|quests` row-remove path in `tasks.go` (board/summary tiles).
- `BuildDayFocus` / `BuildQuestsFocus` data assembly (except deleting the now-dead `Prev`/`Next`/`RecapStart` assignments in `BuildDayFocus`).
- `internal/web/day.go` — `renderDayJournal` reuses `DayJournal`/`.Date`/`.Journal` only; do not change it.
- Plans 092 (settings) and 094 (artifact cap) surfaces.

## Git workflow

- Branch: `improve/093-denav-day-quests-artifacts`.
- Commit per logical unit; conventional-commit style, e.g.
  `refactor(web): quests artifact is a flat task-card stack, no rail`.
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Denav `DayFocus`

In `dayfocus.go`:

1. In `DayFocus`, remove the `Div(Class("day-nav"), …)` block and the
   `nextNode` logic. Render the day title as a plain heading inside `.day-focus`
   (keep the `today` tag). Keep `DayJournal(v)`, the recap **summary text**
   section, the Done section, and the Logs section, separated by the existing
   `Div(Class("stitch"))`.
2. In the recap section, remove the transcript expander
   (`Article(Class("recap-card recap-day") … recap-expand … recap-children …)`,
   `dayfocus.go:211-229`) — keep only `recapText`. The `recapSectionKids`
   `if !v.IsToday { … }` append goes away.
3. Delete the now-dead fields `Prev`, `Next`, `RecapStart` from `DayFocusView`
   and their assignments in `BuildDayFocus` (lines ~57–62, 58 `RecapStart`).
   Keep `Date`, `Label`, `IsToday`, `Journal`, `Recap`, `Done`, `Logs`.

Keep the root container class `.day-focus` (tests + CSS depend on it; only the
inner `.day-nav` is removed).

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0. Then
`grep -n "day-nav\|RecapStart\|\.Prev\|\.Next" internal/feature/journalcards/dayfocus.go`
→ nothing.

### Step 2: Flatten `QuestsFocus` into a grouped task-card stack

In `questsfocus.go`, replace the master/detail `QuestsFocus` with a flat,
rhythm-grouped stack of full `TaskCard`s — no rail nav, no detail aside. Target
shape:

```go
// QuestsFocus renders the quests artifact: rhythm-grouped sections, each a
// flat stack of TaskCards. No rail, no detail pane (plan 093) — navigation
// lives in the chat/sidebar, not inside the artifact.
func QuestsFocus(v QuestsFocusView) g.Node {
	if len(v.Groups) == 0 {
		return Div(Class("quest-stack"),
			ui.EmptyState(ui.EmptyProps{Compact: true, Line: "No quests yet. Speak one in the chat."}))
	}
	sections := make([]g.Node, 0, len(v.Groups)+1)
	for _, grp := range v.Groups {
		cards := make([]g.Node, 0, len(grp.Tasks))
		for _, t := range grp.Tasks {
			cards = append(cards, TaskCard(t))
		}
		sections = append(sections,
			Section(Class("k-section"),
				H2(Class("k-heading"),
					g.Text(grp.Name+" "),
					Span(Class("k-count"), g.Text(itoa(len(grp.Tasks)))),
				),
				Div(Class("tasks-stack"), g.Group(cards)),
			),
		)
	}
	// "Done recently": a plain headed stack (no <details> disclosure).
	if len(v.DoneRecently) > 0 {
		cards := make([]g.Node, 0, len(v.DoneRecently))
		for _, t := range v.DoneRecently {
			cards = append(cards, TaskCard(t))
		}
		sections = append(sections,
			Section(Class("k-section"),
				H2(Class("k-heading"), g.Text("Done recently "),
					Span(Class("k-count"), g.Text(itoa(len(v.DoneRecently))))),
				Div(Class("tasks-stack"), g.Group(cards)),
			),
		)
	}
	return Div(Class("quest-stack"), g.Group(sections))
}
```

Then **delete `QuestRail`** (111–169) — it is now unused. After removing it,
check whether the `data` (gomponents-datastar) and `ui` imports are still used
in `questsfocus.go`: `ui.EmptyState` keeps `ui`; `QuestRail` was the only
`data.On` user, so **remove the now-unused `data "maragu.dev/gomponents-datastar"`
import** if the build flags it. `BuildQuestsFocus` still computes `First` (the
detail pre-render) which is now unused — remove `First` from `QuestsFocusView`
and stop computing it in `buildQuestsFocusFrom` (lines ~72–79), or leave it and
accept a dead field. **Prefer removing it** (no dead code), updating
`questsfocus_test.go` accordingly.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.
`grep -n "quest-rail\|quest-detail\|QuestRail" internal/feature/taskcards/questsfocus.go` → nothing.

### Step 3: Remove the `#quest-rail` OOB refresh from the transition handler

In `tasks.go`, delete the `if ref := e.Request.Header.Get("Referer"); …
u.Path == "/ui/show/quests" { … #quest-rail … }` block (lines ~264–277). The
`#tcard-{id}` outer-patch above it already updates the card in place in the flat
stack. Then check imports: if `taskcards` and/or `net/url` become unused in
`tasks.go`, remove them (grep the file — `taskcards` / `url.` may be used
elsewhere; only remove if truly unused).

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `go vet ./...` → exit 0.

### Step 4: Remove orphaned CSS

In `basm.css`, delete the rules that no longer have markup: `.day-nav`,
`.day-nav-spacer`, `.quest-log` (and its `@media` block ~2380), `.quest-rail`,
`.quest-group`, `.quest-group-title`, `.quest-row`, `.quest-row.quest-overdue`
(if present), `.quest-detail`, `.quest-done`. Keep `.tasks-stack`, `.k-section`,
`.k-heading`, `.k-count`, `.day-focus` (still used). If `.quest-stack` needs
spacing, add one minimal rule
`.quest-stack { display:flex; flex-direction:column; gap: var(--space-4); }`.

**Verify**: `grep -rn "quest-rail\|quest-detail\|quest-log\|day-nav" internal/`
→ returns nothing (no Go, no CSS, no test).

### Step 5: Update the storybook day + quests stories

In `stories_cards.go` (grep `DayFocus` / `QuestsFocus`), update the two stories
to the flat renders and drop any nav/rail/stepper do-or-don't copy.

**Verify**: `go test ./internal/feature/storybook/...` → all pass.

### Step 6: Update the web tests

1. `handlers_test.go:501` — change `ExpectedContent: []string{"day-focus",
   "day-nav", "January"}` to `[]string{"day-focus", "January"}` and add
   `NotExpectedContent: []string{"day-nav"}`.
2. `journal_test.go` `TestJournalCandleIntegration` — keeps asserting
   `day-focus` + entry text (still valid). Confirm it passes unchanged.
3. `tasks_test.go`:
   - `TestQuestsArtifactEndpoint`: change `ExpectedContent` from `"quest-log"`
     to `"quest-stack"` (keep `"Morning stretch"`); the empty-state sub-test
     keeps `"No quests yet. Speak one in the chat."`.
   - `TestTaskTransitionRailRefresh`: the artifact no longer has a rail. For the
     `Referer=/ui/show/quests` sub-test, drop the `id="quest-rail"` expectation;
     assert the in-place card replace instead (`ExpectedContent:
     []string{"datastar-patch-elements", "tcard-"}`, `NotExpectedContent:
     []string{`id="quest-rail"`}`). Keep the `src=today` / `src=quests`
     row-remove sub-tests unchanged (that path is out of scope and intact).
4. `show_test.go`: change the `"quest-log"` assertion to `"quest-stack"`.
5. `templates_test.go`: update quests assertions (`quest-log`/`quest-rail` →
   `quest-stack` + a group heading or task title) and confirm the day
   assertions (`day-focus`) still pass; remove any `day-nav` assertion.

**Verify**: `go test ./internal/web/...` → all pass.

### Step 7: Full gates

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass
- `gofmt -l internal/` → no output
- `git diff --check` → no output

## Test plan

- `tasks_test.go`: quests artifact renders `quest-stack` + grouped task cards
  (happy path), empty state line (edge), transition self-replaces `#tcard-{id}`
  with **no** `#quest-rail` (the behavior change this plan introduces).
- `handlers_test.go` / `templates_test.go`: day artifact has `day-focus` but no
  `day-nav`.
- Pattern to follow: the existing `tests.ApiScenario` blocks in `tasks_test.go`
  (note the `Referer` header and `NotExpectedContent`).
- Verification: `go test ./...` → all pass.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0
- [ ] `gofmt -l internal/` prints nothing
- [ ] `grep -rn "quest-rail\|quest-detail\|quest-log\|day-nav\|QuestRail" internal/` returns nothing
- [ ] `grep -rn "RecapStart" internal/feature/journalcards/` returns nothing
- [ ] `git status` shows only in-scope files modified
- [ ] `plans/readme.md` status row for 093 updated

## STOP conditions

Stop and report back (do not improvise) if:

- The `DayFocus` / `QuestsFocus` / transition-handler code does not match the
  excerpts (drift since `766b7aa`).
- `ui.Focus` is rendered somewhere other than `cardFocusHTML`/`uicardBody`
  (grep `ui.Focus` across `internal/`; a surviving page/board consumer means
  flattening these would break it — STOP).
- `TaskCard`'s transition control posts with a `src=` value when shown in the
  quests artifact (it should NOT — the artifact uses the detail-card path that
  hits `#tcard-{id}`). If quests-artifact cards send `src=quests`, the
  row-remove path fires instead and the stack won't update — STOP and report.
- Removing `QuestRail` leaves `BuildQuestsFocus`/`First` in a state where other
  callers (grep `BuildQuestsFocus`, `QuestRail`, `.First`) break — STOP.
- A verification fails twice after a reasonable fix attempt.

## Maintenance notes

- **Behavior change**: completing a task in the quests artifact now updates the
  card in place (struck/done state) instead of moving it into a separate rail
  section. This is intended (flat, nav-free). If "completed quests should
  disappear from the open stack" is later wanted, do it by removing
  `#tcard-{id}` on completion from the quests surface specifically — not by
  reintroducing the rail.
- The recap **transcript expander** was removed from the day artifact (a
  drill-down). The day's raw conversation is still reachable via the recap
  telescope on the chat itself (`/ui/recap/expand` is still wired there);
  only the in-artifact button is gone.
- To view another day, the agent summons `card_show {type:"day",
  params:{date:"YYYY-MM-DD"}}` (or the owner asks). The `date` param still
  works end to end.
- Reviewer should scrutinize: the `tasks.go` import list after removing the
  rail block (no unused `taskcards`/`url`), and that the `src=` row-remove path
  is untouched.
- Sibling plans 092 (settings per-section) and 094 (cap active artifacts) are
  independent.
