# Storybook Completeness — Slice 2: Data-display Atoms Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add three data-display atoms — `ui.CalendarCell`, `ui.Sparkline`, `ui.DayEntry` — with new tokenized CSS, registered as storybook stories. Ports of the export's DataDisplay components (no existing reusable renderer matches their shape).

**Architecture:** Each is a new `internal/ui` func returning `g.Node` + a new tokenized CSS block appended to `basm.css`. `CalendarCell` is a month-grid day button; `Sparkline` is an SVG path+marker mini-chart (via `g.El`); `DayEntry` is a timeline row. Pure, static, fixture-driven stories.

**Tech Stack:** Go, gomponents (`g.El` for SVG, `h.*` for HTML), vanilla `basm.css`.

**Conventions:** package `ui` uses QUALIFIED `g`/`h` imports (NO dot-import). New CSS appends at the END of `basm.css`, tokenized (`var(--token)`, no raw hex, single-dash). Atom tests are `package ui_test` using the shared `render(t, node)` helper (in `internal/ui/helpers_test.go` — do NOT redefine). Stories append a canvas func to `internal/feature/storybook/storybook.go` + a `Story` entry to `story.go`. After each task: `go test ./...`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`. If `git status` shows a non-task file modified, `git checkout --` it.

Verified facts: SVG is rendered via `g.El("svg", g.Attr(...), g.El("path", ...))` (per `lifecards/measure.go`). Tokens `--ink`, `--ink-muted`, `--surface`, `--parch-edge`, `--grain-ink`, `--parch-bevel`, `--gold`, `--gold-deep`, `--gold-ink`, `--ember`, `--ember-deep`, `--teal-ink`, `--good-ink`, `--font-mono` all exist. The storybook last story entry is `{"knowledgecard", "Cards", "KnowledgeCard", knowledgeCardCanvas}`. The selected-cell ink (export `#1c0d04`) maps to `var(--ink)` to stay tokenized (imperceptible difference).

---

## File Structure

- **Create** `internal/ui/calendarcell.go`+`_test.go`, `internal/ui/sparkline.go`+`_test.go`, `internal/ui/dayentry.go`+`_test.go`.
- **Modify** `internal/web/assets/static/basm.css` (3 CSS blocks), `internal/feature/storybook/storybook.go` (3 canvases), `internal/feature/storybook/story.go` (3 stories).

---

## Task 1: `ui.CalendarCell` + story

**Files:** Create `internal/ui/calendarcell.go`, `internal/ui/calendarcell_test.go`; modify `basm.css`, `storybook.go`, `story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/calendarcell_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestCalendarCell(t *testing.T) {
	got := render(t, ui.CalendarCell(ui.CalendarCellProps{Day: 14, Pips: 2, Today: true}))
	for _, want := range []string{
		`<button class="cal-day is-today" type="button">`,
		`<span class="cal-day-num">14</span>`,
		`<span class="cal-day-pips"><i class="cal-pip cal-pip-0"></i><i class="cal-pip cal-pip-1"></i></span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("calendar cell missing %q in: %s", want, got)
		}
	}
}

func TestCalendarCellSelectedDim(t *testing.T) {
	sel := render(t, ui.CalendarCell(ui.CalendarCellProps{Day: 15, Selected: true}))
	if !strings.Contains(sel, `<button class="cal-day is-selected" type="button">`) {
		t.Errorf("selected class missing: %s", sel)
	}
	dim := render(t, ui.CalendarCell(ui.CalendarCellProps{Day: 31, Dim: true}))
	if !strings.Contains(dim, `<button class="cal-day is-dim" type="button">`) {
		t.Errorf("dim class missing: %s", dim)
	}
	if strings.Contains(dim, "cal-pip") {
		t.Errorf("0 pips should render no pip elements: %s", dim)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestCalendarCell -v` — Expected: FAIL (`undefined: ui.CalendarCell`).

- [ ] **Step 3: Implement the atom**

Create `internal/ui/calendarcell.go`:
```go
package ui

import (
	"strconv"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// CalendarCellProps configures a CalendarCell — one day in a month grid. Day is
// the date number; Pips (0-3, capped) are event dots coloured by index
// (ember/teal/gold). Today rings it, Selected fills it gold, Dim fades an
// other-month day.
type CalendarCellProps struct {
	Day      int
	Pips     int
	Today    bool
	Selected bool
	Dim      bool
}

// CalendarCell renders a square day button: the date number over up to three
// event pips. Static (catalog) — the day link/click is wired by the caller.
func CalendarCell(p CalendarCellProps) g.Node {
	cls := "cal-day"
	switch {
	case p.Selected:
		cls += " is-selected"
	case p.Today:
		cls += " is-today"
	}
	if p.Dim {
		cls += " is-dim"
	}
	n := p.Pips
	if n > 3 {
		n = 3
	}
	pips := make([]g.Node, 0, n)
	for i := 0; i < n; i++ {
		pips = append(pips, h.El("i", h.Class("cal-pip cal-pip-"+strconv.Itoa(i))))
	}
	return h.Button(h.Class(cls), h.Type("button"),
		h.Span(h.Class("cal-day-num"), g.Text(strconv.Itoa(p.Day))),
		h.Span(append([]g.Node{h.Class("cal-day-pips")}, pips...)...),
	)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestCalendarCell -v` — Expected: PASS (both). Note: when `Selected` AND `Today`, Selected wins (matches the export — selected fill replaces the today ring).

- [ ] **Step 5: Append the CalendarCell CSS**

At the END of `internal/web/assets/static/basm.css`:
```css

/* ── CalendarCell — a square day button in a month grid ─────────────────── */
.cal-day {
  position: relative; width: 100%; aspect-ratio: 1 / 1; min-width: 38px;
  display: flex; flex-direction: column; align-items: center; justify-content: center; gap: 5px;
  cursor: pointer; border-radius: 0; padding: 4px; color: var(--ink);
  background: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px;
  border: 2px solid var(--parch-edge);
}
.cal-day-num { font-family: var(--font-mono); font-size: 13px; line-height: 1; }
.cal-day.is-today { border-color: var(--gold-deep); box-shadow: inset 0 0 0 1px var(--gold-deep); }
.cal-day.is-today .cal-day-num { font-weight: 700; }
.cal-day.is-selected { background: var(--gold); background-image: none; box-shadow: var(--parch-bevel); }
.cal-day.is-selected .cal-day-num { color: var(--ink); font-weight: 700; }
.cal-day.is-dim { opacity: .4; }
.cal-day-pips { display: flex; gap: 3px; height: 5px; }
.cal-pip { width: 5px; height: 5px; display: block; }
.cal-pip-0 { background: var(--ember); }
.cal-pip-1 { background: var(--teal-ink); }
.cal-pip-2 { background: var(--gold-ink); }
.cal-day.is-selected .cal-pip { background: var(--ink); }
```

- [ ] **Step 6: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append:
```go

func calendarCellCanvas() g.Node {
	cell := func(p ui.CalendarCellProps) g.Node {
		return h.Div(h.Style("width:76px"), ui.CalendarCell(p))
	}
	return section("CalendarCell",
		cell(ui.CalendarCellProps{Day: 8, Pips: 1}),
		cell(ui.CalendarCellProps{Day: 14, Pips: 2, Today: true}),
		cell(ui.CalendarCellProps{Day: 15, Pips: 2, Selected: true}),
		cell(ui.CalendarCellProps{Day: 31, Dim: true}),
	)
}
```
In `internal/feature/storybook/story.go`, add immediately AFTER the `{"knowledgecard", "Cards", "KnowledgeCard", knowledgeCardCanvas},` line:
```go
	{"calendarcell", "Display", "CalendarCell", calendarCellCanvas},
```

- [ ] **Step 7: Verify + commit**

```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/ -run TestCalendarCell && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
tail -20 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
git status --short
```
Expected: PASS, suite green, build clean, NO RAW HEX. Stage only the five files, then:
```bash
git add internal/ui/calendarcell.go internal/ui/calendarcell_test.go internal/web/assets/static/basm.css internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add CalendarCell atom + storybook story\n\nui.CalendarCell(CalendarCellProps) — a square month-grid day button (date + 0-3\nindex-coloured event pips; today/selected/dim states). New tokenized .cal-day CSS.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: `ui.Sparkline` + story

A tiny SVG line chart: a filled area path + a line path + a square end-marker. Color
drives all three. Ports the export's `<path>`+`<rect>` shape (NOT the shipped
polyline+circle).

**Files:** Create `internal/ui/sparkline.go`, `internal/ui/sparkline_test.go`; modify `basm.css`, `storybook.go`, `story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/sparkline_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestSparkline(t *testing.T) {
	got := render(t, ui.Sparkline(ui.SparkProps{
		Data: []float64{62, 64, 61, 67, 70, 66, 72, 75, 73, 78}, Width: 200, Height: 48,
	}))
	for _, want := range []string{
		`<svg class="sparkline" width="200" height="48" viewBox="0 0 200 48"`,
		`<path d="M3.0 `,                                   // first point x = pad = 3.0
		`fill="var(--teal-ink)" opacity="0.12">`,           // area path (default colour)
		`stroke="var(--teal-ink)"`,                         // line path
		`<rect`,
		`width="5" height="5" fill="var(--teal-ink)">`,     // square end-marker
	} {
		if !strings.Contains(got, want) {
			t.Errorf("sparkline missing %q in: %s", want, got)
		}
	}
}

func TestSparklineColor(t *testing.T) {
	got := render(t, ui.Sparkline(ui.SparkProps{Data: []float64{1, 2, 3}, Color: "var(--ember-deep)"}))
	if !strings.Contains(got, `stroke="var(--ember-deep)"`) {
		t.Errorf("Color override missing: %s", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestSparkline -v` — Expected: FAIL (`undefined: ui.Sparkline`).

- [ ] **Step 3: Implement the atom**

Create `internal/ui/sparkline.go`:
```go
package ui

import (
	"math"
	"strconv"
	"strings"

	g "maragu.dev/gomponents"
)

// SparkProps configures a Sparkline. Data is the series (min/max auto-scale).
// Color (a CSS colour/var, default var(--teal-ink)) drives the line, area fill,
// and end-marker. Width/Height default 120x34.
type SparkProps struct {
	Data   []float64
	Color  string
	Width  int
	Height int
}

// ff formats a float to one decimal place (matches JS toFixed(1)).
func ff(x float64) string { return strconv.FormatFloat(x, 'f', 1, 64) }

// Sparkline renders the export's mini-chart: a filled area path + a 2px line path
// + a 5x5 square end-marker, all in Color. Empty Data renders an empty svg.
func Sparkline(p SparkProps) g.Node {
	w, ht, pad := p.Width, p.Height, 3
	if w == 0 {
		w = 120
	}
	if ht == 0 {
		ht = 34
	}
	color := p.Color
	if color == "" {
		color = "var(--teal-ink)"
	}
	svgAttrs := []g.Node{
		g.Attr("class", "sparkline"),
		g.Attr("width", strconv.Itoa(w)), g.Attr("height", strconv.Itoa(ht)),
		g.Attr("viewBox", "0 0 "+strconv.Itoa(w)+" "+strconv.Itoa(ht)),
	}
	if len(p.Data) >= 2 {
		min, max := p.Data[0], p.Data[0]
		for _, v := range p.Data {
			min, max = math.Min(min, v), math.Max(max, v)
		}
		span := max - min
		if span == 0 {
			span = 1
		}
		stepX := float64(w-pad*2) / math.Max(1, float64(len(p.Data)-1))
		xs := make([]float64, len(p.Data))
		ys := make([]float64, len(p.Data))
		var d strings.Builder
		for i, v := range p.Data {
			xs[i] = float64(pad) + float64(i)*stepX
			ys[i] = float64(pad) + float64(ht-pad*2)*(1-(v-min)/span)
			if i == 0 {
				d.WriteString("M" + ff(xs[i]) + " " + ff(ys[i]))
			} else {
				d.WriteString(" L" + ff(xs[i]) + " " + ff(ys[i]))
			}
		}
		line := d.String()
		last := len(p.Data) - 1
		area := line + " L" + ff(xs[last]) + " " + ff(float64(ht-pad)) + " L" + ff(xs[0]) + " " + ff(float64(ht-pad)) + " Z"
		svgAttrs = append(svgAttrs,
			g.El("path", g.Attr("d", area), g.Attr("fill", color), g.Attr("opacity", "0.12")),
			g.El("path", g.Attr("d", line), g.Attr("fill", "none"), g.Attr("stroke", color),
				g.Attr("stroke-width", "2"), g.Attr("stroke-linejoin", "round"), g.Attr("stroke-linecap", "round")),
			g.El("rect", g.Attr("x", ff(xs[last]-2.5)), g.Attr("y", ff(ys[last]-2.5)),
				g.Attr("width", "5"), g.Attr("height", "5"), g.Attr("fill", color)),
		)
	}
	return g.El("svg", svgAttrs...)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestSparkline -v` — Expected: PASS (both).

- [ ] **Step 5: Append the Sparkline CSS**

At the END of `internal/web/assets/static/basm.css`:
```css

/* ── Sparkline — tiny SVG trend chart (colour via attr) ─────────────────── */
.sparkline { display: block; shape-rendering: geometricPrecision; }
```

- [ ] **Step 6: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append:
```go

func sparklineCanvas() g.Node {
	data := []float64{62, 64, 61, 67, 70, 66, 72, 75, 73, 78}
	frame := func(n g.Node) g.Node {
		return h.Div(h.Class("fdn-card"), n)
	}
	return section("Sparkline",
		frame(ui.Sparkline(ui.SparkProps{Data: data, Color: "var(--teal-ink)", Width: 200, Height: 48})),
		frame(ui.Sparkline(ui.SparkProps{Data: data, Color: "var(--ember-deep)", Width: 200, Height: 48})),
	)
}
```
In `internal/feature/storybook/story.go`, add immediately AFTER the `{"calendarcell", "Display", "CalendarCell", calendarCellCanvas},` line:
```go
	{"sparkline", "Display", "Sparkline", sparklineCanvas},
```

- [ ] **Step 7: Verify + commit**

```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/ -run TestSparkline && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
git status --short
```
Expected: PASS, suite green, build clean. Stage only the five files, then:
```bash
git add internal/ui/sparkline.go internal/ui/sparkline_test.go internal/web/assets/static/basm.css internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add Sparkline atom + storybook story\n\nui.Sparkline(SparkProps) — the export mini-chart: filled area path + 2px line +\nsquare end-marker, colour-driven. New .sparkline CSS.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 3: `ui.DayEntry` + story

A day-timeline row: a mono time rail, a node dot on a vertical rail line, and the
content (title + optional detail). Tone colours the node; Last drops the trailing
rail + bottom padding.

**Files:** Create `internal/ui/dayentry.go`, `internal/ui/dayentry_test.go`; modify `basm.css`, `storybook.go`, `story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/dayentry_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestDayEntry(t *testing.T) {
	got := render(t, ui.DayEntry(ui.DayEntryProps{Time: "07:30", Title: "Fed the hens", Detail: "daily · streak 12", Tone: "teal"}))
	for _, want := range []string{
		`<div class="dayentry dayentry-teal">`,
		`<div class="dayentry-time">07:30</div>`,
		`<div class="dayentry-rail"><span class="dayentry-node"></span></div>`,
		`<div class="dayentry-content"><div class="dayentry-title">Fed the hens</div><div class="dayentry-detail">daily · streak 12</div></div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("day entry missing %q in: %s", want, got)
		}
	}
}

func TestDayEntryLastNoDetail(t *testing.T) {
	got := render(t, ui.DayEntry(ui.DayEntryProps{Time: "18:00", Title: "Watered", Last: true}))
	if !strings.Contains(got, `<div class="dayentry dayentry-gold dayentry-last">`) {
		t.Errorf("last + default gold tone missing: %s", got)
	}
	if strings.Contains(got, "dayentry-detail") {
		t.Errorf("no Detail should omit the detail div: %s", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestDayEntry -v` — Expected: FAIL (`undefined: ui.DayEntry`).

- [ ] **Step 3: Implement the atom**

Create `internal/ui/dayentry.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// DayEntryProps configures a DayEntry timeline row. Time is the left-rail label;
// Title is the entry; Detail (optional) is a sub-line. Tone ("gold" default,
// "teal", "ember") colours the node dot. Last drops the trailing rail + padding.
type DayEntryProps struct {
	Time   string
	Title  string
	Detail string
	Tone   string
	Last   bool
}

// DayEntry renders one day-timeline row: time rail | node dot | content.
func DayEntry(p DayEntryProps) g.Node {
	tone := p.Tone
	if tone == "" {
		tone = "gold"
	}
	cls := "dayentry dayentry-" + tone
	if p.Last {
		cls += " dayentry-last"
	}
	content := []g.Node{h.Class("dayentry-content"), h.Div(h.Class("dayentry-title"), g.Text(p.Title))}
	if p.Detail != "" {
		content = append(content, h.Div(h.Class("dayentry-detail"), g.Text(p.Detail)))
	}
	return h.Div(h.Class(cls),
		h.Div(h.Class("dayentry-time"), g.Text(p.Time)),
		h.Div(h.Class("dayentry-rail"), h.Span(h.Class("dayentry-node"))),
		h.Div(content...),
	)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestDayEntry -v` — Expected: PASS (both).

- [ ] **Step 5: Append the DayEntry CSS**

At the END of `internal/web/assets/static/basm.css`:
```css

/* ── DayEntry — a day-timeline row: time | node rail | content ──────────── */
.dayentry { display: grid; grid-template-columns: 54px 22px 1fr; column-gap: 12px; align-items: stretch; }
.dayentry-time { font-family: var(--font-mono); font-size: 11.5px; color: var(--ink-muted); text-transform: uppercase; letter-spacing: .03em; padding-top: 1px; text-align: right; }
.dayentry-rail { position: relative; display: flex; justify-content: center; }
.dayentry-rail::before { content: ""; position: absolute; top: 0; bottom: 0; width: 2px; background: var(--parch-edge); }
.dayentry-node { position: relative; z-index: 1; width: 12px; height: 12px; margin-top: 1px; background: var(--gold-ink); border: 2px solid var(--surface); box-shadow: 0 0 0 2px var(--gold-ink); }
.dayentry-teal .dayentry-node { background: var(--teal-ink); box-shadow: 0 0 0 2px var(--teal-ink); }
.dayentry-ember .dayentry-node { background: var(--ember-deep); box-shadow: 0 0 0 2px var(--ember-deep); }
.dayentry-last .dayentry-rail::before { bottom: auto; height: 14px; }
.dayentry-content { padding-bottom: 18px; }
.dayentry-last .dayentry-content { padding-bottom: 0; }
.dayentry-title { font-size: 14.5px; color: var(--ink); line-height: 1.3; }
.dayentry-detail { font-family: var(--font-mono); font-size: 11px; color: var(--ink-muted); margin-top: 2px; }
```

- [ ] **Step 6: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append:
```go

func dayEntryCanvas() g.Node {
	return section("DayEntry",
		h.Div(h.Class("list"), h.Div(h.Style("padding:14px"),
			ui.DayEntry(ui.DayEntryProps{Time: "07:30", Title: "Fed the hens", Detail: "daily · streak 12", Tone: "gold"}),
			ui.DayEntry(ui.DayEntryProps{Time: "13:00", Title: "Logged weight — 81.2 kg", Detail: "life log", Tone: "teal"}),
			ui.DayEntry(ui.DayEntryProps{Time: "18:00", Title: "Watered the tomatoes", Detail: "every 2 days", Tone: "ember", Last: true}),
		)),
	)
}
```
In `internal/feature/storybook/story.go`, add immediately AFTER the `{"sparkline", "Display", "Sparkline", sparklineCanvas},` line:
```go
	{"dayentry", "Display", "DayEntry", dayEntryCanvas},
```

- [ ] **Step 7: Verify + commit**

```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/ -run TestDayEntry && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./... && go vet ./...
tail -16 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
git status --short
```
Expected: PASS, suite green, build+vet clean, NO RAW HEX. Stage only the five files, then:
```bash
git add internal/ui/dayentry.go internal/ui/dayentry_test.go internal/web/assets/static/basm.css internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add DayEntry atom + storybook story\n\nui.DayEntry(DayEntryProps) — a day-timeline row (time rail | node dot | content);\nTone colours the node, Last drops the trailing rail. New tokenized .dayentry CSS.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification (controller — visual)

- [ ] `go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check` — green.
- [ ] `/storybook/calendarcell`, `/storybook/sparkline`, `/storybook/dayentry` render 200 (content-assert the component class).
- [ ] Screenshot the three (Hearthwood): the 4 calendar cell states; the teal+ember sparklines on parchment; the 3-row day timeline.

## What this delivers / what's next

**Delivered:** the data-display atoms (CalendarCell, Sparkline, DayEntry) under the Display group.

**Next (Slice 3 — completeness):** the build-from-export domain organisms — RecapCard, GuardianCard, NudgeBanner, StatCard (new CSS) + the Topbar story — completing the 1:1 catalog.
