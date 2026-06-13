# Plan 052: Journal focus + Day card — retire /journal and /day (Phase 2)

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving on. If a
> STOP condition occurs, stop and report — do not improvise. When done, update
> the `052` row in `plans/readme.md`. Execute task-by-task with
> `superpowers:subagent-driven-development` or `superpowers:executing-plans`.
>
> **Drift check (run first)**: `git diff --stat 7c4e2fa..HEAD -- internal/web web/templates internal/cards`
> Authored at `7c4e2fa` (Phase 1 / plan 051 merged). Program spec:
> `docs/superpowers/specs/2026-06-13-card-first-kill-the-pages-design.md`.
> If `internal/web/journal.go`, `internal/web/day.go`, `internal/web/cards.go`,
> `internal/cards/cards.go`, `web/templates/journal.html`,
> `web/templates/day.html`, `web/templates/cards.html`,
> `web/templates/recap-cards.html`, or `web/templates/layout.html` changed since
> `7c4e2fa`, compare excerpts; on mismatch, STOP.

## Status

- **Priority**: P1 (Phase 2 of the card-first program)
- **Effort**: L (two page retirements; one new card type; two bespoke focuses;
  several inbound-link re-points; test surgery)
- **Risk**: MED–HIGH (deletes two live pages with write paths; the `day` card is
  net-new; calendar/recap links re-pointed)
- **Depends on**: plans/050 (focus mechanism + `focusBodyHTML` seam), plans/051
  (the seam precedent) — both DONE/merged
- **Category**: direction (card-first "kill the pages", Phase 2 of 8)
- **Planned at**: commit `7c4e2fa`, 2026-06-13

## Why this matters

`/journal` (the "candle" — free/guided writing) becomes the **journal card's
focus**. `/day/{date}` is a richer surface — a **day-of-life aggregation**
(journal + that day's recap & transcript + what got done + what was logged, with
prev/next nav) — so it becomes a **new `day` card** (a tile summary + a full
focus). Both pages are then deleted and every inbound link (calendar cells,
recap "visit", the journal card footer) re-points to `/focus/day`. The write
paths already exist (`/ui/journal`, `/ui/journal/prompt`, `/ui/day/{date}/journal`,
`/ui/day/journal/{id}/drop`) and are reused unchanged — this plan moves their
*surfaces* into focus, it does not add new writes.

## Current state

### Focus seam (plan 050/051)
`focusPage` (`internal/web/focus.go`) → `focusBodyHTML(typ, params)` switch:
`quests` → `questsFocusHTML`, default → `cardHTML`. This plan adds `journal` and
`day` cases. `focusCanonicalQuery` keeps every param except `from` in the
reflected URL — so `/focus/day?date=…` round-trips.

### Journal — the candle (`internal/web/journal.go`, `web/templates/journal.html`)
- `journalPage` (`journal.go:37-44`) renders `journal.html` (via `shell_open`):
  free-hand/guided tabs, `#candle-prompt`, then `{{template
  "journal_candle_body" .}}`.
- `journal_candle_body` (`journal.html:22-44`, id `#journal-candle-body`): a write
  form `@post('/ui/journal')` + today's entries; each entry has a
  `href="/day/{{.Date}}"` "→ this day" link.
- `journalWrite` (POST `/ui/journal`, `journal.go:48-59`) writes via
  `life.JournalWrite`, re-renders `#journal-candle-body` (`renderCandleBody`,
  `journal.go:149-163`).
- `journalPrompt` (GET `/ui/journal/prompt`, `journal.go:64-80`) patches
  `#candle-prompt` (inner) with one guided line.
- `buildCandleData(now)` (`journal.go:118-147`) → `candleData{Title, MainClass,
  Dock, Today, Journal[]}`. **Reused by the focus.**

### Day — the day-of-life page (`internal/web/day.go`, `web/templates/day.html`)
- `dayPage` (`day.go:45-60`) renders `day.html` (a standalone `<!DOCTYPE>` doc):
  `day-nav` (prev/next `href="/day/{Prev|Next}"`), `{{template "day_journal" .}}`,
  a recap section (+ transcript expander `@get('/ui/recap/expand?type=day&start=…')`),
  "What got done" (`.Done`), "The day's log" (`.Logs`).
- `day_journal` (`day.html:70-94`, id `#day-journal`): entries with per-entry drop
  `@post('/ui/day/journal/{id}/drop?date=…')` + write `@post('/ui/day/{date}/journal')`.
- `buildDay(d, now)` (`day.go:62-131`) → `dayData{Title,Date,Label,IsToday,Prev,
  Next,Journal[],Recap,RecapStart,Done[],Logs[]}` from `life.Day(app, convID, d)`.
  **Reused by the tile and the focus.**
- `dayJournalWrite` / `dayJournalDrop` (`day.go:134-162`) reuse `renderDayJournal`
  (`day.go:164-193`) which patches `#day-journal` (outer) with `day_journal`.
  **Kept — work unchanged inside the focus.**
- `dayStartOf`, `dayLayout` (`day.go:195-197,22`): kept.

### Journal card (read-only, exists)
`renderCardJournal` (`cards.go:354-373`) → `ucard_journal` (recent entries). Its
**tile is unchanged**; only its focus is added.

### Inbound links to retire-targets
- topbar `<a href="/journal">Journal</a>` (`layout.html:27`).
- calendar card cell `href="/day/{{.Date}}"` (`cards.html:107`).
- journal card footer `href="/day/{{.TodayDate}}"` "today's page →" (`cards.html:169`).
- recap card `href="/day/{{.Date}}"` "visit" (`recap-cards.html:12`).
- candle entry `href="/day/{{.Date}}"` "→ this day" (`journal.html:36`, inside
  `journal_candle_body`).

### Routes (`web.go:203-208`)
`GET /journal`, `POST /ui/journal`, `GET /ui/journal/prompt`,
`GET /day/{date}`, `POST /ui/day/{date}/journal`, `POST /ui/day/journal/{id}/drop`.
**Delete only the two GET page routes; keep the four `/ui/*` write routes.**

### Tests
`journal_test.go:43` (GET `/journal`), `:166` (GET `/day/…`);
`handlers_test.go:905` (GET `/day/2026-01-15`); `templates_test.go:200,216`
(render `day.html`).

### Card registry (`internal/cards/cards.go`)
`journal` spec exists. **Add a `day` spec.** `cardErrorStrip` is in
`cards.go:138`; `intParam`/`queryToMap` in `cards.go`.

## Commands you will need

```bash
go test ./internal/web/... ./internal/cards/...
go test ./... && go vet ./... && gofmt -l internal web && CGO_ENABLED=0 go build ./...
grep -rnE '"/journal"|href="/journal|href="/day/|"/day/|journalPage|dayPage|"journal\.html"|"day\.html"' internal web --include='*.go' --include='*.html'
```

## Scope

**In:** journal card focus (candle: tabs + guided prompt + write + history); a new
`day` card (tile summary + full day-of-life focus with prev/next); re-point all
five inbound links to `/focus/day`; delete `/journal` + `/day` GET routes, their
page handlers and templates, and the topbar `/journal` link; move the reused
templates to surviving files; adapt tests.

**Out:** any new write endpoint (the four `/ui/*` writes are reused as-is); the
journal/day card **tiles' existing** behavior beyond what's specified; other
pages and topbar links (their phases).

## Git workflow

Branch `feature/card-first-kill-pages` (synced to `main` @ `7c4e2fa`). Commit
after each green step. Steps A–C are additive/green; Step D is the deletion;
Step E is tests/docs.

## Steps

### Step A: journal card focus = the candle (additive)

**File:** `web/templates/journal-focus.html` (new) — the focus candle controls.
It references `journal_candle_body`, which still lives in `journal.html` at this
step (single definition — moved in Step D):

```html
{{- /* journal-focus.html — the "candle" as the journal card's focus body
     (free/guided write + history). Was /journal. The focus header (Back +
     "Journal") is supplied by focus_main; this is the inner body. */ -}}
{{define "journal_focus"}}
<div class="candle-focus">
  <div class="k-tabs" role="tablist">
    <button class="k-tab k-tab-active" type="button"
            data-on:click="document.getElementById('candle-prompt').innerHTML='';el.parentElement.querySelectorAll('.k-tab').forEach(b=>b.classList.remove('k-tab-active'));el.classList.add('k-tab-active')">
      free hand
    </button>
    <button class="k-tab" type="button"
            data-on:click="el.parentElement.querySelectorAll('.k-tab').forEach(b=>b.classList.remove('k-tab-active'));el.classList.add('k-tab-active');@get('/ui/journal/prompt')">
      guided
    </button>
  </div>
  <div id="candle-prompt"></div>
  {{template "journal_candle_body" .}}
</div>
{{end}}
```

**File:** `internal/web/journal.go` — add the focus renderer (after
`renderCandleBody`, near `journal.go:163`):

```go
// journalFocusHTML renders the journal card's focus body: the candle
// (free/guided write + guided prompt + today's history). Was the /journal page.
func (h *handlers) journalFocusHTML() template.HTML {
	data, err := h.buildCandleData(time.Now())
	if err != nil {
		h.app.Logger().Warn("journal focus render failed", "err", err)
		return cardErrorStrip("could not open the journal")
	}
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "journal_focus", data); err != nil {
		h.app.Logger().Warn("journal focus template failed", "err", err)
		return cardErrorStrip("could not render the journal")
	}
	return template.HTML(b.String())
}
```

> `journal.go` already imports `strings` and `time`; add `"html/template"` to its
> imports (the function returns `template.HTML`). `cardErrorStrip` is in
> `cards.go` (same package).

**File:** `internal/web/focus.go` — add the dispatch case in `focusBodyHTML`:

```go
	case "journal":
		return h.journalFocusHTML()
```

**Verify:** `go build ./... && go test ./internal/web/ -run 'TestFocus|TestJournal'` → ok.

**Test (add to `internal/web/focus_test.go`):**

```go
// TestFocusJournalShowsCandle: /focus/journal renders the candle (write form +
// guided tab), so expanding the journal card gives the full writing surface.
func TestFocusJournalShowsCandle(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /focus/journal shows the candle",
		Method:         "GET",
		URL:            "/focus/journal",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`@post('/ui/journal'`,   // write form
			`/ui/journal/prompt`,    // guided tab
			`id="journal-candle-body"`,
		},
	}
	s.Test(t)
}
```

**Verify:** `go test ./internal/web/ -run TestFocusJournalShowsCandle -v` → PASS.
**Commit:** `git add internal/web/journal.go internal/web/focus.go web/templates/journal-focus.html internal/web/focus_test.go && git commit -m "feat(focus): journal card focus = the candle (write/guided/history)"`

### Step B: the new `day` card (registry + tile + focus) (additive)

**File:** `internal/cards/cards.go` — add a spec to the `registry` slice (after
the `journal` entry):

```go
		{
			Type:  "day",
			Label: "Day",
			Icon:  "scroll",
			W:     4,
			H:     22,
			Params: []ParamSpec{
				{Name: "date", Doc: "YYYY-MM-DD to show (default: today)"},
			},
		},
```

**File:** `internal/cards/cards_test.go` — extend the registry coverage if a test
asserts a card count or iterates all types (read it: `grep -n "All()\|len(" internal/cards/cards_test.go`); update any hard count.

**File:** `web/templates/cards.html` — add the day tile template near the other
`ucard_*` defines (place a `{{define "ucard_day"}}`):

```html
{{define "ucard_day"}}
<article class="kcard ucard ucard-day">
  <header class="kcard-head"><span class="kcard-kind">▪ day</span>
    <span class="tag">{{.Label}}</span></header>
  <ul class="ucard-stats">
    <li>{{.JournalN}} journal</li>
    <li>{{.DoneN}} done</li>
    <li>{{.LogN}} logged</li>
    <li>{{if .HasRecap}}recap kept{{else if .IsToday}}still being written{{else}}no recap{{end}}</li>
  </ul>
  <footer class="kcard-actions"><a href="/focus/day?date={{.Date}}">open the day →</a></footer>
</article>
{{end}}
```

**File:** `internal/web/cards.go` — register the renderer in the `cardInto`
switch (near `cards.go:99`) and add the tile view + renderer:

```go
	case "day":
		return h.renderCardDay(w, params)
```

```go
type cardDayView struct {
	Date, Label              string
	IsToday                  bool
	JournalN, DoneN, LogN    int
	HasRecap                 bool
}

func (h *handlers) renderCardDay(w io.Writer, params map[string]string) error {
	now := time.Now()
	d := dayStartOf(now)
	if s := params["date"]; s != "" {
		if t, err := time.ParseInLocation(dayLayout, s, now.Location()); err == nil {
			d = dayStartOf(t)
		}
	}
	dd, err := h.buildDay(d, now)
	if err != nil {
		return err
	}
	return h.tmpl.ExecuteTemplate(w, "ucard_day", cardDayView{
		Date:     dd.Date,
		Label:    dd.Label,
		IsToday:  dd.IsToday,
		JournalN: len(dd.Journal),
		DoneN:    len(dd.Done),
		LogN:     len(dd.Logs),
		HasRecap: dd.Recap != "",
	})
}
```

**File:** `web/templates/day-focus.html` (new) — the full day view as a focus body.
Move the `<main>` contents of `day.html` (`day.html:11-65`) here as `day_focus`,
re-pointing the nav to the focus route, and reference `day_journal` (still in
`day.html` at this step — moved in Step D):

```html
{{- /* day-focus.html — the day-of-life view as the day card's focus body
     (journal + recap/transcript + done + logs). Was /day/{date}. Prev/next
     navigate the focus in place. */ -}}
{{define "day_focus"}}
<div class="day-focus">
  <div class="day-nav">
    <a class="btn btn-ghost btn-sm" href="/focus/day?date={{.Prev}}"
       data-on:click__prevent="@get('/focus/day?date={{.Prev}}')">◂ prev</a>
    <h2 class="day-title">{{.Label}}{{if .IsToday}} <span class="tag">today</span>{{end}}</h2>
    {{if .Next}}<a class="btn btn-ghost btn-sm" href="/focus/day?date={{.Next}}"
       data-on:click__prevent="@get('/focus/day?date={{.Next}}')">next ▸</a>{{else}}<span class="day-nav-spacer"></span>{{end}}
  </div>

  {{template "day_journal" .}}

  <div class="stitch"></div>
  <section class="k-section">
    <h2 class="k-heading">The day in summary</h2>
    {{if .Recap}}<p class="recap-body">{{.Recap}}</p>
    {{else if .IsToday}}<p class="k-sub">Today is still being written.</p>
    {{else}}<p class="k-sub">No summary kept for this day.</p>{{end}}
    {{if not .IsToday}}
    <article class="recap-card recap-day">
      <header class="recap-head">
        <span class="recap-label">The conversation, preserved</span>
        <button class="recap-expand" type="button"
                data-on:click="el.closest('.recap-card').classList.add('recap-open'); @get('/ui/recap/expand?type=day&start={{.RecapStart}}')">
          transcript
        </button>
      </header>
      <div class="recap-children" id="recap-children-day-{{.RecapStart}}"></div>
    </article>
    {{end}}
  </section>

  <div class="stitch"></div>
  <section class="k-section">
    <h2 class="k-heading">What got done</h2>
    {{if .Done}}<ul class="tl-items">{{range .Done}}<li class="tl-item"><span class="tl-time">{{.Time}}</span> {{.Text}}</li>{{end}}</ul>
    {{else}}<p class="k-sub">Nothing marked done this day.</p>{{end}}
  </section>

  <div class="stitch"></div>
  <section class="k-section">
    <h2 class="k-heading">The day's log</h2>
    {{if .Logs}}<ul class="tl-items">{{range .Logs}}<li class="tl-item"><span class="tl-time">{{.Time}}</span> {{.Text}}</li>{{end}}</ul>
    {{else}}<p class="k-sub">Nothing logged this day.</p>{{end}}
  </section>
</div>
{{end}}
```

**File:** `internal/web/day.go` — add the focus renderer (after `buildDay`):

```go
// dayFocusHTML renders the day card's focus body: the full day-of-life view for
// the date param (default today), with prev/next navigating the focus. Was the
// /day/{date} page.
func (h *handlers) dayFocusHTML(params map[string]string) template.HTML {
	now := time.Now()
	d := dayStartOf(now)
	if s := params["date"]; s != "" {
		if t, err := time.ParseInLocation(dayLayout, s, now.Location()); err == nil {
			d = dayStartOf(t)
		}
	}
	data, err := h.buildDay(d, now)
	if err != nil {
		h.app.Logger().Warn("day focus render failed", "err", err)
		return cardErrorStrip("could not open the day")
	}
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "day_focus", data); err != nil {
		h.app.Logger().Warn("day focus template failed", "err", err)
		return cardErrorStrip("could not render the day")
	}
	return template.HTML(b.String())
}
```

> Add `"html/template"` to `day.go`'s imports (returns `template.HTML`).

**File:** `internal/web/focus.go` — add the dispatch case:

```go
	case "day":
		return h.dayFocusHTML(params)
```

**Verify:** `go build ./... && go test ./internal/web/ ./internal/cards/` → ok.

**Tests (add to `internal/web/focus_test.go`):**

```go
// TestFocusDayShowsSections: /focus/day renders the day-of-life sections.
func TestFocusDayShowsSections(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /focus/day shows the day view",
		Method:         "GET",
		URL:            "/focus/day",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{
			`id="day-journal"`,
			"What got done",
			"The day's log",
			`@post('/ui/day/`, // the day journal write form
		},
	}
	s.Test(t)
}

// TestUiCardDayTile: the day tile renders the day-of-life summary.
func TestUiCardDayTile(t *testing.T) {
	s := tests.ApiScenario{
		Name:           "GET /ui/cards/day renders the tile",
		Method:         "GET",
		URL:            "/ui/cards/day",
		TestAppFactory: newWebApp,
		ExpectedStatus: 200,
		ExpectedContent: []string{"ucard-day", "journal", `/focus/day?date=`},
	}
	s.Test(t)
}
```

**Verify:** `go test ./internal/web/ -run 'TestFocusDay|TestUiCardDay' -v` → PASS.
**Commit:** `git add internal/cards web/templates/cards.html web/templates/day-focus.html internal/web/cards.go internal/web/day.go internal/web/focus.go internal/web/focus_test.go && git commit -m "feat(cards): new day card — day-of-life tile + focus"`

### Step C: re-point inbound /day and /journal links

Edit each link to target the focus (plain `<a href>`; a full nav renders the
focus shell):
- `web/templates/cards.html:107` calendar cell: `href="/day/{{.Date}}"` →
  `href="/focus/day?date={{.Date}}"`.
- `web/templates/cards.html:169` journal-card footer: `href="/day/{{.TodayDate}}"`
  → `href="/focus/day?date={{.TodayDate}}"`.
- `web/templates/recap-cards.html:12` "visit": `href="/day/{{.Date}}"` →
  `href="/focus/day?date={{.Date}}"`.
- The candle entry "→ this day" lives in `journal_candle_body`
  (`journal.html:36`): `href="/day/{{.Date}}"` → `href="/focus/day?date={{.Date}}"`.
  (This template moves in Step D; re-point it now in place.)

**Verify:** `go test ./internal/web/ -run 'TestUiCard|TestRecap|TestCard'` → ok;
`grep -rn 'href="/day/' web/templates` → no matches.
**Commit:** `git add web/templates && git commit -m "feat(cards): calendar/recap/journal links point to /focus/day"`

### Step D: delete /journal and /day pages

1. **Move surviving templates.**
   - Cut `{{define "journal_candle_body"}}…{{end}}` from `journal.html` into
     `web/templates/journal-focus.html` (so it is defined exactly once).
   - Cut `{{define "day_journal"}}…{{end}}` from `day.html` into
     `web/templates/day-focus.html` (defined exactly once).
2. **Delete** `web/templates/journal.html` and `web/templates/day.html` entirely.
3. **Remove routes** `GET /journal` and `GET /day/{date}` (`web.go:203,206`).
   Keep the four `/ui/*` write routes.
4. **Delete page handlers**: `journalPage` (`journal.go:37-44`) and `dayPage`
   (`day.go:45-60`). **Keep** `journalWrite`, `journalPrompt`, `composeJournalPrompt`,
   `buildCandleData`, `renderCandleBody`, `journalFocusHTML`, `journalPageDayEntries`;
   and `buildDay`, `dayJournalWrite`, `dayJournalDrop`, `renderDayJournal`,
   `dayFocusHTML`, `dayStartOf`, `dayLayout`. Update the file-top comments in
   `journal.go`/`day.go` (they describe the pages).
5. **Remove the topbar link** `<a href="/journal">Journal</a>` (`layout.html:27`).
6. **Tests** (read the files first; remove/adapt minimally, keep coverage):
   - `journal_test.go:43` (GET `/journal`): re-point to `GET /focus/journal` (the
     candle now lives there) asserting the write form; keep the `journalWrite`/
     prompt tests (endpoints unchanged).
   - `journal_test.go:166` and `handlers_test.go:905` (GET `/day/…`): re-point to
     `GET /focus/day?date=…` asserting the day sections; keep `dayJournalWrite`/
     `dayJournalDrop` tests.
   - `templates_test.go:200,216` (render `day.html`): render `day_focus` instead
     (pass the same `dayData`/`today` value); the standalone-doc assertions
     (DOCTYPE/title) drop — `day_focus` is a body fragment.

**Verify (all must hold):**
```
go test ./... && go vet ./... && gofmt -l internal web && CGO_ENABLED=0 go build ./...
grep -rnE '"/journal"|href="/journal|href="/day/|journalPage|dayPage|"journal\.html"|"day\.html"' internal web --include='*.go' --include='*.html'
```
The grep MUST return nothing except, possibly, a test that asserts a retired
route 302s (acceptable — like `TestTasksRouteRetired`). `gofmt -l` MUST be empty.

**Browser check (owner — sandbox has no display):** `/boards` → expand the
journal card (⤢) → candle write + guided prompt + history; put a `day` card on a
board → tile shows counts → expand → full day view, prev/next navigates, write +
remove an entry works, recap/done/logs show; a calendar cell click opens
`/focus/day?date=…`; topbar has no Journal link; `/journal` and `/day/X` 302 to
`/boards`.

**Commit:** `git add -A && git commit -m "feat(journal,day): retire /journal and /day into the journal + day cards"`

### Step E: docs

Update the `052` row in `plans/readme.md` to DONE (journal focus + day card; both
pages retired). Fix `/journal`/`/day` references in `DESIGN.md`, `README.md`,
`internal/self/knowledge.md` (`grep -rn '/journal\|/day/' DESIGN.md README.md internal/self/knowledge.md`).

**Commit:** `git add -A && git commit -m "docs: journal/day are cards now; 052 done"`

## Test plan

- **Journal focus** (`focus_test.go`): `/focus/journal` shows the write form +
  guided tab + `#journal-candle-body`.
- **Day focus + tile** (`focus_test.go`): `/focus/day` shows journal/done/logs +
  the write form; `/ui/cards/day` shows the tile summary + `/focus/day` link.
- **Writes still work**: `journalWrite`, `dayJournalWrite`, `dayJournalDrop`,
  `journalPrompt` tests pass (endpoints unchanged; re-pointed Referer where the
  test asserted a page).
- **Deletion safety**: the Step D grep is clean; `go test ./...` green; the
  re-pointed `day.html`→`day_focus` template test renders.
- **Browser** (owner): the Step D checklist.

## Done criteria

- [ ] `focusBodyHTML` dispatches `journal` → candle and `day` → day-of-life; other
      types unchanged.
- [ ] New `day` card: registry spec + `renderCardDay` tile + `dayFocusHTML` focus
      with a working `date` param and prev/next.
- [ ] `/journal` + `/day/{date}` GET routes, `journalPage`, `dayPage`,
      `journal.html`, `day.html` deleted; the four `/ui/*` write routes and their
      handlers retained and reused.
- [ ] `journal_candle_body` + `day_journal` survive in exactly one file each.
- [ ] All five inbound links point to `/focus/day`; no `href="/day/"` or
      `href="/journal"` remains; topbar Journal link gone.
- [ ] Step D grep clean; `go test ./...`, vet, `gofmt -l` (empty), CGO-free build
      clean; `git diff --check` clean.
- [ ] No new write endpoint added.
- [ ] `plans/readme.md` 052 row → DONE; doc refs fixed.

## STOP conditions

- "redefinition of template" for `journal_candle_body` or `day_journal` → a copy
  was left in the deleted file; each must be defined exactly once.
- `day_focus` references a field absent from `dayData` → re-check against
  `day.go:32-43`; the template was moved verbatim, so field names must match.
- The day card's `buildDay` returns an error for a far-past/empty date → it
  should render an empty day, not error; if `life.Day` errors on no data, STOP
  and report (the page tolerated it — match that).
- A removed test orphans a helper (compile error) → remove the helper too, or
  keep the test re-pointed.
- The Step D grep finds a `/day/`/`/journal` reference not listed in Current
  state → STOP, list it, re-point or remove before declaring done.

## Maintenance notes

- The journal/day **card tiles** stay read-only summaries; the rich writing/day
  surfaces are the **focus**. The day tile defaults to today; the focus honors
  the `date` param (and calendar/recap deep-link to a specific day via it).
- Prev/next in the day focus `@get('/focus/day?date=…')` without a `from`, so
  after navigating days the Back control falls through to `/boards` — acceptable;
  thread `from` later if deep day-navigation back-to-board matters.
- `focusBodyHTML` now has three bespoke cases (quests, journal, day). Knowledge
  (053) and Heads (054) add theirs the same way.
