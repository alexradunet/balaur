# Plan 055: Life focus — a new lifelog card, retire /life (Phase 5)

> **Executor instructions**: Follow this plan step by step. Run every Verify and
> confirm before moving on. On a STOP condition, stop and report. When done,
> update the `055` row in `plans/readme.md`. Execute with
> `superpowers:subagent-driven-development` or `superpowers:executing-plans`.
>
> **Drift check (run first)**: `git diff --stat 0c1a273..HEAD -- internal/web web/templates internal/cards`
> Authored at `0c1a273` (Phase 4 / plan 054 merged). Spec:
> `docs/superpowers/specs/2026-06-13-card-first-kill-the-pages-design.md`.
> If `internal/web/life.go`, `internal/web/cards.go`, `internal/cards/cards.go`,
> `web/templates/life.html`, `web/templates/cards.html`, or
> `web/templates/layout.html` changed since `0c1a273`, compare excerpts; on
> mismatch, STOP.

## Status

- **Priority**: P2 (Phase 5 of the card-first program)
- **Effort**: M
- **Risk**: LOW–MED (one new read-only card; retire one page; re-point 3 footers)
- **Depends on**: plans/050 (focus seam), plans/052 (the day-card precedent for a
  net-new card) — DONE/merged
- **Category**: direction (card-first "kill the pages", Phase 5 of 8)
- **Planned at**: commit `0c1a273`, 2026-06-13

## Why this matters

`/life` is the owner's life-overview: a **Habits** strip (recurring tasks +
streaks) and a **Tracked** grid (each logged kind — numeric kinds chart a
sparkline, text kinds list recent lines). It is **read-only** — entries are
logged via chat (`life.Log` from the agent tools), never a web form ("Tell
Balaur what matters… and it appears here"). So, like tasks, **strict parity =
no create form** (the spec's "entry create" does not match reality). `/life`
becomes a new **`lifelog` card** (a tile summary + a full-overview focus), then
the page is retired. The per-kind `measure`/`lines` and `habits` cards already
exist and are unchanged.

## Current state

### Focus seam
`focusBodyHTML` (`internal/web/focus.go`) has cases quests/journal/day/memory/
skills; default → `cardHTML`. This plan adds a `lifelog` case.

### The /life page (`internal/web/life.go`, `web/templates/life.html`)
- `lifePage` (`life.go:41-78`) builds `{Title, Dock, Kinds []lifeKindView, Habits
  []lifeHabitView}` and renders `life.html`. The kinds loop calls `life.Kinds`,
  `life.Series`, `life.Summarize`, `sparkPoints`, `numericValues`.
- `buildHabits(now)` (`life.go:80-104`) — recurring tasks + streaks. **Shared
  with the `habits` card (`renderCardHabits` in cards.go) — KEEP.**
- `lifeKindView`/`lifeHabitView` types, `lifeWindowDays`, `sparkW/sparkH`,
  `sparkPoints`, `numericValues` (`life.go`) — KEEP (the focus + `measure` card
  use them).
- `life.html`: page wrapper (`shell_open` + `<h1>` + the body) — the **body**
  (habits section + tracked grid + empty state) becomes the focus.

### Per-kind cards (unchanged)
`renderCardMeasure`/`renderCardLines`/`renderCardHabits` (`cards.go`). Their tile
footers link to `/life`: `cards.html:213` (measure), `:235` (lines), `:388`
(habits) — re-point to `/focus/lifelog`.

### Topbar
`<a href="/life">Life</a>` (`layout.html:26`) — remove.

### Route (`web.go:201`)
`GET /life` — delete.

### Card registry (`internal/cards/cards.go`)
Add a `lifelog` spec. (`measure`/`lines`/`habits` already exist.)

### Tests
A life page test + a `life.html` template render test exist — read
`internal/web/*_test.go` (`grep -n 'life\|/life\|Life' internal/web/*_test.go`).

## Commands you will need
```bash
go test ./internal/web/... ./internal/cards/...
go test ./... && go vet ./... && gofmt -l internal web && CGO_ENABLED=0 go build ./...
grep -rnE '"/life"|href="/life|lifePage|"life\.html"' internal web --include='*.go' --include='*.html'
```

## Scope

**In:** a new `lifelog` card (registry spec + `renderCardLifelog` tile +
`lifelogFocusHTML` focus = the life overview); extract a `lifeOverview` helper
from `lifePage`; move the `life.html` body to a surviving file; delete `/life`
route + `lifePage` + the `life.html` page; re-point the 3 card footers + remove
the Life topbar link; adapt tests.

**Out:** any web log/create form (chat logs entries — strict parity); the
`measure`/`lines`/`habits` cards' existing behavior; other pages.

## Git workflow
Branch `feature/card-first-kill-pages` (synced to `main` @ `0c1a273`). Commit
after each green step. A–C additive; D deletes; E docs.

## Steps

### Step A: move the `life.html` body to `life_body`, then add `lifeOverview` + the focus (additive)

**FIRST, move the template** (so `life_body` is defined before anything renders
it): create `web/templates/lifelog-focus.html` and move the **body** of
`life.html` into it as `{{define "life_body"}}…{{end}}` — the habits section + the
tracked grid + the empty state (everything between the `<h1>` and `shell_close`).
Leave `life.html` containing only the page wrapper for now (it still references
`life_body` by name; deleted in Step D). `life_body` must be defined **exactly
once** after this (`grep -rn '{{define "life_body"}}' web/templates` → 1).

```html
{{- /* lifelog-focus.html — the life overview (habits + tracked kinds) as the
     lifelog card's focus body. Read-only; entries are logged via chat. */ -}}
{{define "life_body"}}
... (paste life.html's body verbatim: {{if .Habits}}… + {{if .Kinds}}… tracked
     grid (class="k-grid life-grid") + {{else}} empty state) ...
{{end}}
```

The body references `.Habits` and `.Kinds`; `lifelogFocusHTML` (below) passes
`map[string]any{"Kinds":…, "Habits":…}`, matching.

**THEN, in `internal/web/life.go`** — extract the kinds-loop into a helper the
focus and tile share, and add the focus renderer. Add `"html/template"` to
imports.

Add (place near `lifePage`):

```go
// lifeOverview builds the life-overview view-models (tracked kinds + habits) —
// the data behind the lifelog card tile and focus, and formerly the /life page.
func (h *handlers) lifeOverview(now time.Time) (kinds []lifeKindView, habits []lifeHabitView) {
	ks, err := life.Kinds(h.app)
	if err == nil {
		for _, k := range ks {
			recs, err := life.Series(h.app, k.Kind, now.AddDate(0, 0, -lifeWindowDays))
			if err != nil {
				continue
			}
			v := lifeKindView{Kind: k.Kind, Unit: k.Unit, Count: k.Count}
			if s := life.Summarize(recs); s.Points > 0 {
				v.Numeric = true
				v.LastVal = fmt.Sprintf("%g", s.Last)
				v.LastAt = s.LastAt.In(now.Location()).Format("Jan 2")
				if s.Points > 1 {
					v.Change = fmt.Sprintf("%+.4g over %dd", s.Last-s.First, lifeWindowDays)
					v.Points, v.SparkLastX, v.SparkLastY = sparkPoints(numericValues(recs), sparkW, sparkH)
				}
			} else {
				for i := len(recs) - 1; i >= 0 && len(v.Recent) < 5; i-- {
					line := recs[i].GetDateTime("noted_at").Time().In(now.Location()).Format("Jan 2")
					if t := recs[i].GetString("text"); t != "" {
						line += " — " + clipText(t, 120)
					}
					v.Recent = append(v.Recent, line)
				}
			}
			kinds = append(kinds, v)
		}
	}
	return kinds, h.buildHabits(now)
}

// lifelogFocusHTML renders the lifelog card's focus body: the full life overview
// (habits + every tracked kind). Was the /life page. Read-only — entries are
// logged via chat.
func (h *handlers) lifelogFocusHTML() template.HTML {
	now := time.Now()
	kinds, habits := h.lifeOverview(now)
	var b strings.Builder
	if err := h.tmpl.ExecuteTemplate(&b, "life_body", map[string]any{
		"Kinds": kinds, "Habits": habits,
	}); err != nil {
		h.app.Logger().Warn("lifelog focus render failed", "err", err)
		return cardErrorStrip("could not open the life overview")
	}
	return template.HTML(b.String())
}
```

> `cardErrorStrip` is in `cards.go` (same package). `life.go` already imports
> `fmt`, `strings`, `time`, `life`, `tasks`, `core`.

**File:** `internal/web/focus.go` — add the dispatch case in `focusBodyHTML`:

```go
	case "lifelog":
		return h.lifelogFocusHTML()
```

**Verify:** `go build ./... && go test ./internal/web/ -run 'TestFocus'` → ok
(`life_body` is now defined from the move above, so `/focus/lifelog` renders;
`lifeOverview`/`lifelogFocusHTML` compile and the focus route works).
**Commit:** `git add web/templates/lifelog-focus.html web/templates/life.html internal/web/life.go internal/web/focus.go && git commit -m "feat(focus): lifelog focus = the life overview (life_body moved out of the page)"`

### Step B: the `lifelog` registry spec + tile

**File:** `internal/cards/cards.go` — add to the `registry` slice (after `lines`
or near `habits`):

```go
		{
			Type:  "lifelog",
			Label: "Life",
			Icon:  "orb",
			W:     6,
			H:     24,
			// no params — the full tracked overview + habits
		},
```

**File:** `internal/cards/cards_test.go` — add `lifelog` to whatever enumerates
all types / asserts the count / `HasManage` false-list (read it; the day card in
plan 052 set the precedent).

**File:** `web/templates/cards.html` — add the tile define (compact: habit strip +
tracked-kind tags, no sparklines):

```html
{{define "ucard_lifelog"}}
<article class="kcard ucard ucard-lifelog" id="ucard-lifelog">
  <header class="kcard-head"><span class="kcard-kind"><img class="tool-icon" src="/static/icons/orb.png" alt="">Life</span></header>
  {{if .Habits}}
  <div class="habit-strip">
    {{range .Habits}}<span class="tag habit-tag" title="{{.RecurLine}}">{{.Title}}{{if gt .Streak 0}} · {{.Streak}}{{end}}</span>{{end}}
  </div>
  {{end}}
  {{if .Kinds}}
  <ul class="ucard-stats">
    {{range .Kinds}}<li>{{.Kind}} <span class="kcard-meta">{{.Count}}</span></li>{{end}}
  </ul>
  {{else}}
  <p class="k-sub">Nothing tracked yet — tell Balaur what matters.</p>
  {{end}}
  <footer class="kcard-actions"><a href="/focus/lifelog">open life →</a></footer>
</article>
{{end}}
```

> Match the `kcard-head`/`tool-icon` markup to a sibling `ucard_*` (read e.g.
> `ucard_journal`); the snippet above mirrors the day card from plan 052.

**File:** `internal/web/cards.go` — register the renderer in the `cardInto`
switch and add the tile renderer:

```go
	case "lifelog":
		return h.renderCardLifelog(w, params)
```

```go
type cardLifelogView struct {
	Habits []lifeHabitView
	Kinds  []lifeKindView
}

func (h *handlers) renderCardLifelog(w io.Writer, _ map[string]string) error {
	kinds, habits := h.lifeOverview(time.Now())
	return h.tmpl.ExecuteTemplate(w, "ucard_lifelog", cardLifelogView{Habits: habits, Kinds: kinds})
}
```

**Verify:** `go build ./... && go test ./internal/web/ ./internal/cards/ -run 'TestUiCard|TestFocus|TestAll|TestHasManage'` → ok.

**Tests (add to `internal/web/focus_test.go`):**

```go
// TestUiCardLifelogTile: the lifelog tile renders.
func TestUiCardLifelogTile(t *testing.T) {
	s := tests.ApiScenario{
		Name: "GET /ui/cards/lifelog renders the tile", Method: "GET",
		URL: "/ui/cards/lifelog", TestAppFactory: newWebApp, ExpectedStatus: 200,
		ExpectedContent: []string{"ucard-lifelog", `/focus/lifelog`},
	}
	s.Test(t)
}

// TestFocusLifelogShowsOverview: /focus/lifelog renders the life overview body.
func TestFocusLifelogShowsOverview(t *testing.T) {
	s := tests.ApiScenario{
		Name: "GET /focus/lifelog shows the overview", Method: "GET",
		URL: "/focus/lifelog", TestAppFactory: newWebApp, ExpectedStatus: 200,
		ExpectedContent: []string{"life-grid"},
	}
	s.Test(t)
}
```

> `life_body` must define `class="life-grid"` (it moves verbatim from `life.html`
> in Step C — its tracked grid is `<div class="k-grid life-grid">`). If you fold
> the `life_body` move into Step A/B, assert against the real moved markup. The
> empty-DB focus renders the empty-state `<p class="k-empty">` (no `life-grid`) —
> if `newWebApp` has no kinds, assert against the empty-state text instead, or
> seed a kind via `life.Log`. Read how other tests seed life entries.

**Verify:** `go test ./internal/web/ -run 'TestUiCardLifelog|TestFocusLifelog' -v` → PASS.
**Commit:** `git add internal/cards web/templates/cards.html internal/web/cards.go internal/web/life.go internal/web/focus.go internal/web/focus_test.go && git commit -m "feat(cards): new lifelog card — life-overview tile + focus"`

### Step C: (folded into Step A)

The `life_body` template move was done in Step A (so every step renders green).
Nothing to do here — proceed to Step D.

### Step D: delete /life

1. **Delete** `web/templates/life.html` (body now in `lifelog-focus.html`).
2. **Remove the route** `GET /life` (`web.go:201`).
3. **Delete** `lifePage` from `life.go`. **KEEP** `lifeOverview`, `lifelogFocusHTML`,
   `buildHabits` (habits card), `lifeKindView`/`lifeHabitView`, `lifeWindowDays`,
   `sparkW/sparkH`, `sparkPoints`, `numericValues`. Remove any import left unused
   (likely none). Update the file-top comment that describes the page.
4. **Re-point the 3 card footers** (`cards.html:213,235,388`): `href="/life"` →
   `href="/focus/lifelog"`.
5. **Remove the topbar link** `<a href="/life">Life</a>` (`layout.html:26`).
6. **Tests:** the `/life` page GET test → retired-route 302 guard; the `life.html`
   template render test → render `life_body` (pass `{Kinds,Habits}`). Keep any
   `buildHabits`/habits-card test.

**Verify (all must hold):**
```
go test ./... && go vet ./... && gofmt -l internal web && CGO_ENABLED=0 go build ./...
grep -rnE '"/life"|href="/life|lifePage|"life\.html"' internal web --include='*.go' --include='*.html'
```
The grep returns nothing except possibly a retired-route 302 guard test.
`gofmt -l` empty.

**Browser check (owner — no display here):** drop a `lifelog` card on a board →
tile shows habits + tracked kinds → expand → the full overview (sparklines, recent
lines); the measure/lines/habits card footers open `/focus/lifelog`; topbar has no
Life link; `/life` 302s to `/boards`.

**Commit:** `git add -A && git commit -m "feat(life): retire /life into the lifelog card focus"`

### Step E: docs
Update the `055` row in `plans/readme.md` → DONE. Fix `/life` refs in `DESIGN.md`,
`README.md`, `internal/self/knowledge.md` (`grep -rn '/life' DESIGN.md README.md internal/self/knowledge.md`).

**Commit:** `git add -A && git commit -m "docs: life overview is the lifelog card now; 055 done"`

## Test plan
- **Tile + focus** (`focus_test.go`): `/ui/cards/lifelog` renders; `/focus/lifelog`
  renders the overview (or empty state on a fresh app).
- **Habits card unaffected**: `buildHabits` still shared; its test passes.
- **Deletion safety**: Step D grep clean; one `life_body` define; `go test ./...`
  green.
- **Browser** (owner): the Step D checklist.

## Done criteria
- [ ] `focusBodyHTML` dispatches `lifelog` → the overview; others unchanged.
- [ ] New `lifelog` card: registry spec + `renderCardLifelog` tile +
      `lifelogFocusHTML` focus; `lifeOverview` shared by both.
- [ ] `/life` route + `lifePage` + `life.html` deleted; `buildHabits` + the
      sparkline helpers + `lifeKindView`/`lifeHabitView` retained.
- [ ] `life_body` defined in exactly one file (`lifelog-focus.html`).
- [ ] The 3 card footers → `/focus/lifelog`; no `href="/life"` remains; Life
      topbar link gone.
- [ ] Step D grep clean; `go test ./...`, vet, `gofmt -l` (empty), CGO-free build
      clean; `git diff --check` clean.
- [ ] No web log/create form added (chat logs entries).
- [ ] `plans/readme.md` 055 → DONE; doc refs fixed.

## STOP conditions
- "redefinition of template life_body" → it exists in both `life.html` and
  `lifelog-focus.html`; define it once.
- Deleting `lifePage` breaks `buildHabits`/`sparkPoints` callers → they are shared
  (keep); only `lifePage` is page-only.
- The Step D grep finds a `/life` reference not in Current state → STOP, list,
  re-point or remove.

## Maintenance notes
- The `lifelog` tile re-runs `life.Kinds` + `Series` per kind (same cost as the
  old page); fine for one card. `focusBodyHTML` cases now: quests, journal, day,
  memory, skills, lifelog. Phase 6 (Settings) is the last surface phase.
