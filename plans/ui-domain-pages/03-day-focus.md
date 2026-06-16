# 03 — Port the Day focus body to a gomponents component

> **Read `plans/ui-domain-pages/README.md` first** (shared recipe, conventions,
> conflict map, verification). **Exemplar**:
> `internal/feature/lifecards/lifelogfocus.go` + `lifelog.go` +
> `internal/web/{cards.go,focus.go}`. Stamped against `884d692`.
>
> **This is the richest body — do it after at least one simpler port (01 quests
> or 02 journal) so the seam is already familiar.** It has: a path-param form AND
> a query-param form (asymmetric), an outer-patch SSE re-render, a bespoke
> JS-driven recap transcript-expander, dual-mode prev/next nav, and three
> empty-state branches. Every one is a silent-break risk if the contract drifts.

## Context / why

`/focus/day` (a day-of-life: journal + recap/transcript + done + logs, prev/next
navigable) still renders from `web/templates/day-focus.html` via
`(*handlers).dayFocusHTML`. Port it to a gomponents component in
`internal/feature/journalcards` (the package owns the `day` card), filling the
`CardSize.Focus` seam (`884d692` exemplar). The day card already registers a
**tile** (`DayCard`) — you're adding the `ui.Focus` branch.

## Current state (read these)

`web/templates/day-focus.html` — `day_focus` define:

- `<div class="day-focus">` → `<div class="day-nav">`:
  - prev: `<a class="btn btn-ghost btn-sm" href="/focus/day?date={Prev}" data-on:click__prevent="@get('/focus/day?date={Prev}')">◂ prev</a>`
  - `<h2 class="day-title">{Label}{if IsToday} <span class="tag">today</span>{end}</h2>`
  - next: `{if .Next}<a … href="/focus/day?date={Next}" data-on:click__prevent="@get('/focus/day?date={Next}')">next ▸</a>{else}<span class="day-nav-spacer"></span>{end}`
- `{day_journal}` (see below)
- `<div class="stitch"></div>` + recap `<section class="k-section"><h2 class="k-heading">The day in summary</h2>` →
  `{if .Recap}<p class="recap-body">{Recap}</p>{else if .IsToday}<p class="k-sub">Today is still being written.</p>{else}<p class="k-sub">No summary kept for this day.</p>{end}`
  then **only `{if not .IsToday}`**: `<article class="recap-card recap-day"><header class="recap-head"><span class="recap-label">The conversation, preserved</span><button class="recap-expand" type="button" data-on:click="el.closest('.recap-card').classList.add('recap-open'); @get('/ui/recap/expand?type=day&start={RecapStart}')">transcript</button></header><div class="recap-children" id="recap-children-day-{RecapStart}"></div></article>`
- `<div class="stitch"></div>` + `<section class="k-section"><h2 class="k-heading">What got done</h2>{if .Done}<ul class="tl-items"><li class="tl-item"><span class="tl-time">{Time}</span> {Text}</li>…</ul>{else}<p class="k-sub">Nothing marked done this day.</p>{end}</section>`
- `<div class="stitch"></div>` + the Logs section, identical shape, heading `The day's log`, empty `Nothing logged this day.`

`day_journal` define (`#day-journal`): `<section class="k-section" id="day-journal"><h2 class="k-heading">Your thoughts</h2>{if .Journal}<div class="journal-list"><article class="journal-entry"><div class="journal-meta"><span class="tl-time">{Time}</span><form data-on:submit__prevent="@post('/ui/day/journal/{ID}/drop?date={$.Date}')"><button class="btn btn-ghost btn-sm" type="submit">remove</button></form></div><p class="journal-text">{Text}</p></article>…</div>{end}<form class="journal-form" data-on:submit__prevent="@post('/ui/day/{Date}/journal', {contentType:'form'})"><textarea name="text" rows="3" placeholder="What stays with you from this day?" required></textarea><button class="btn btn-primary btn-sm" type="submit">Keep it</button></form></section>`

Handlers/builders (`internal/web/day.go`, read for current line numbers):
`dayFocusHTML(params)` (parses `params["date"]`, default today; renders
`day_focus`), `buildDay(d, now)` → `dayData`, `dayJournalWrite` (POST
`/ui/day/{date}/journal`, **date in PATH**, field `text`), `dayJournalDrop` (POST
`/ui/day/journal/{id}/drop`, **id in PATH, date in QUERY** `?date=`), `renderDayJournal`
(renders `day_journal`, patches `#day-journal` **outer**). Recap expander handled
by `recapExpand` in `internal/web/recap.go` (type=day → `h.renderMessages` of the
transcript, patches `recap-children-day-{unix}` **inner**). View-model:
`dayData{Title, Date, Label string; IsToday bool; Prev, Next string; Journal
[]dayJournalView; Recap, RecapStart string; Done, Logs []dayLineView}`,
`dayJournalView{ID, Time, Text string}`, `dayLineView{Time, Text string}`.
`RecapStart` is `d.Unix()` as a string.

## Action contract — preserve byte-for-byte (highest-risk body)

| Trigger | Endpoint | Param location | SSE target & mode |
|---|---|---|---|
| prev / next | `@get('/focus/day?date={Prev/Next}')` (+`href` fallback, `__prevent`) | `date` query | `#main` **inner** (focusPage dual-mode re-renders the whole body) |
| "Keep it" (write) | `@post('/ui/day/{Date}/journal', {contentType:'form'})` | **`date` in PATH**, `text` form field | `#day-journal` **outer** (`renderDayJournal`) |
| "remove" (drop) | `@post('/ui/day/journal/{ID}/drop?date={Date}')` | **`id` in PATH, `date` in QUERY** | `#day-journal` **outer** |
| "transcript" | `el.closest('.recap-card').classList.add('recap-open'); @get('/ui/recap/expand?type=day&start={RecapStart}')` | `type`,`start` query | `#recap-children-day-{RecapStart}` **inner** |

Load-bearing ids/values: `#day-journal`, `#recap-children-day-{RecapStart}`
(must be `recap-children-day-` + the exact unix `RecapStart`), the `.recap-card`
class (the expander's JS does `el.closest('.recap-card')`), `tl-items`/`tl-item`/
`tl-time`, `day-nav`/`day-title`/`day-nav-spacer`. **Reproduce the recap-expand
`data-on:click` string verbatim** (the `classList.add('recap-open')` half is what
reveals the panel; the `@get` half loads it). **Next is omitted (render
`day-nav-spacer`) when `IsToday`.** Write puts date in the PATH; drop puts it in
the QUERY — do not swap them.

## Scope

**In scope:** `internal/feature/journalcards/dayfocus.go` (new) + the `registerDay`
size dispatch (in the existing `day.go` of that package), `internal/web/day.go`
(point `renderDayJournal` at the shared component; drop `dayFocusHTML` if dead),
`internal/web/focus.go` (drop `case "day"`), `web/templates/day-focus.html` (retire
defines once unused), storybook.

**Out of scope (do NOT touch):** `recapExpand`/`renderMessages` (the transcript
renderer — your component only emits the empty `#recap-children-day-{unix}`
container + the expander button), the `/ui/recap/*` handlers, the README conflict
files. **Do NOT reuse `ui.RecapCard` or `ui.DayEntry`** — they emit different
class trees (`.recapcard*`, `.dayentry*`) and `ui.RecapCard` has no transcript
expander / `.recap-card`/`.recap-open` hooks. Hand-emit the recap card + the
`tl-item` rows to keep `.recap-card`/`.tl-item` and the expander JS working.

## Steps

1. `internal/feature/journalcards/dayfocus.go`: view-models `DayFocusView{Date,
   Label string; IsToday bool; Prev, Next string; Journal []DayJournalEntry;
   Recap, RecapStart string; Done, Logs []DayLine}`, `DayJournalEntry{ID, Time,
   Text string}`, `DayLine{Time, Text string}`. `buildDayFocus(app, params)`
   mirroring `web.buildDay` (parse `params["date"]`, default today; reuse the
   `internal/life` day read + `internal/conversation` master for `RecapStart`;
   copy any tiny web-only formatting helper locally — no `internal/web` import).
2. **`DayJournal(v DayFocusView) g.Node`** — port of `day_journal` (`<section
   class="k-section" id="day-journal">`, the entry list with the per-entry drop
   form `@post('/ui/day/journal/'+e.ID+'/drop?date='+v.Date)`, the write form
   `@post('/ui/day/'+v.Date+'/journal', {contentType:'form'})` with `textarea
   name="text" required`).
3. **`DayFocus(v DayFocusView) g.Node`** — port of `day_focus`: the `day-nav`
   (prev always; next or `day-nav-spacer`), `DayJournal(v)`, the recap section
   (the three text branches + the `{if not IsToday}` recap-card expander with the
   verbatim click JS + the `recap-children-day-{RecapStart}` container), and the
   Done/Logs sections (`tl-items`). `stitch` dividers between sections.
4. `registerDay` (journalcards): `ui.Focus` → `DayFocus(buildDayFocus(app,
   params))`, else the existing `DayCard` tile. **Pass `params` through** (date).
5. `internal/web/focus.go`: delete `case "day": return h.dayFocusHTML(params)`.
   (The seam `cardFocusHTML(typ, params)` already forwards `params`.)
6. `internal/web/day.go` `renderDayJournal`: replace `ExecuteTemplate(&b,
   "day_journal", …)` with `journalcards.DayJournal(journalcards.BuildDayFocus(
   h.app, map[string]string{"date": <date>}))` → keep `PatchElements(…,
   WithSelectorID("day-journal"), WithModeOuter())`. (Export as needed.)
7. Retire `dayFocusHTML` + the `day_focus`/`day_journal` defines after step 6;
   grep `*_test.go` for `day_focus`/`day_journal`/`dayFocusHTML` first
   (`TestDayPageRenders` / `TestFocusDayShowsSections` exist — they assert the
   *route* `id="day-journal"`, `What got done`, `@post('/ui/day/`, which the port
   keeps green; leave any direct `ExecuteTemplate` test as dead code + TODO).
8. Storybook `dayfocusStory()` (variants: today writable / past with recap +
   expander / empty) + register mid-Cards-cluster; + `dayfocus_test.go` asserting
   `id="day-journal"`, the write `@post('/ui/day/.../journal'`, the drop
   `@post('/ui/day/journal/.../drop?date='`, the recap-expand JS + `@get('/ui/recap/expand?type=day&start='`, `recap-children-day-`, `tl-items`, and that `next ▸` is absent when `IsToday`.

## Done criteria

- `CGO_ENABLED=0 go build ./...` → 0; targeted + storybook tests ok; `gofmt -l`
  empty; `git diff --check` clean.
- `internal/web/focus_test.go::TestFocusDayShowsSections` (and `TestDayPageRenders`
  if present) still pass.
- Live: `curl -s 127.0.0.1:PORT/focus/day` contains `day-focus`, `id="day-journal"`,
  `What got done`, `The day's log`, `@post('/ui/day/`, the topbar + `id="dock"`.
  `curl '…/focus/day?date=<a past YYYY-MM-DD>'` contains the recap-card +
  `recap-children-day-` + `@get('/ui/recap/expand?type=day&start=`. Manually: write
  + remove a journal entry (→ `#day-journal` updates), prev/next nav, and expand a
  past-day transcript.

## Test plan

`internal/feature/journalcards/dayfocus_test.go` (mirror `lifelogfocus_test.go`):
assert the full contract above on a populated past-day `DayFocusView`, a today
view (no `next`, recap "still being written", no expander), and an empty day.
Re-run `TestFocusDayShowsSections`, `TestDayPageRenders`, and any `dayJournal*`
handler tests.

## Maintenance note

`#day-journal` has one renderer (`DayJournal`) shared by `DayFocus` and
`renderDayJournal`. The transcript is still rendered by `recapExpand`
(`renderMessages`) — your component only provides the button + the empty
container. Review must check the path-vs-query form asymmetry and the verbatim
recap-expand JS.

## Escape hatches

- If `renderDayJournal` reads the date differently than `buildDayFocus` expects,
  reconcile to one builder signature — don't fork `#day-journal` markup.
- If you're tempted to "simplify" the recap-expand into `ui.RecapCard` — don't;
  it breaks the expander. If a clean reuse isn't obvious, hand-emit (that's
  correct here).
- Anything pulling in the README conflict files → STOP.
</content>
