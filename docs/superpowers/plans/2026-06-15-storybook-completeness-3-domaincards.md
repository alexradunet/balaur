# Storybook Completeness — Slice 3: Domain Cards + Topbar Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the last five catalog components — `ui.RecapCard`, `ui.GuardianCard`, `ui.NudgeBanner`, `ui.StatCard` (new domain organisms → Cards group), and a `Topbar` story (export the existing `shell.topbar` → Navigation group) — completing the 1:1 component catalog.

**Architecture:** The four cards are new `internal/ui` funcs returning `g.Node` + new tokenized CSS lifted from the export inline styles. `GuardianCard` reuses the existing `.dlg-corner*` brackets and composes `ui.Button`; `StatCard` composes `ui.Sparkline`. `Topbar` exports the existing shell renderer.

**Tech Stack:** Go, gomponents, vanilla `basm.css`.

**Conventions:** package `ui` uses QUALIFIED `g`/`h` imports (NO dot-import). New CSS appends at the END of `basm.css`, tokenized (`var(--token)`, no raw hex; raw `rgba()` is allowed for highlights/scrims). Atom tests are `package ui_test` using the shared `render(t, node)` helper. Stories append a canvas func to `internal/feature/storybook/storybook.go` + a `Story` entry to `story.go`. After each task: `go test ./...`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`. If `git status` shows a non-task file modified, `git checkout --` it.

Verified facts: tokens `--surface`, `--surface-2`, `--grain-ink`, `--ink`, `--ink-muted`, `--parch-edge`, `--parch-bevel`, `--bevel-in`, `--gold-deep`, `--gold-ink`, `--teal-ink`, `--good-ink`, `--ember-deep`, `--font-mono`, `--font-body`, `--font-display` all exist. `.dlg-corner` / `.dlg-corner-tl|tr|bl|br` exist (reuse for GuardianCard). `ui.Button(ui.ButtonProps{Size,Variant,Href}, children...)` and `ui.Sparkline(ui.SparkProps{Data,Color,Width,Height})` exist. Icons `orb`, `shield`, `bell`, `gem` exist in `/static/icons/`. The storybook last story entry is `{"dayentry", "Display", "DayEntry", dayEntryCanvas}`.

---

## Task 1: `ui.RecapCard` + story

**Files:** Create `internal/ui/recapcard.go`, `internal/ui/recapcard_test.go`; modify `basm.css`, `storybook.go`, `story.go`.

- [ ] **Step 1: Write the failing test** — Create `internal/ui/recapcard_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestRecapCard(t *testing.T) {
	got := render(t, ui.RecapCard(ui.RecapProps{
		When: "earlier today", Summary: "We planned the orchard work.",
		Points: []string{"Watered the tomatoes", "Exported notes"},
	}))
	for _, want := range []string{
		`<article class="recapcard">`,
		`<img class="recapcard-orb" src="/static/icons/orb.png" alt="" decoding="async">`,
		`<span class="recapcard-kicker">Recap</span>`,
		`<span class="recapcard-when">earlier today</span>`,
		`<p class="recapcard-summary">We planned the orchard work.</p>`,
		`<li class="recapcard-point"><span class="recapcard-sq">▪</span><span>Watered the tomatoes</span></li>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("recap card missing %q in: %s", want, got)
		}
	}
}

func TestRecapCardEmpty(t *testing.T) {
	got := render(t, ui.RecapCard(ui.RecapProps{}))
	if strings.Contains(got, "recapcard-summary") || strings.Contains(got, "recapcard-point") {
		t.Errorf("empty summary/points should omit the <p>/<ul>: %s", got)
	}
}
```

- [ ] **Step 2: Run** `go test ./internal/ui/ -run TestRecapCard -v` — Expected: FAIL (undefined).

- [ ] **Step 3: Implement** — Create `internal/ui/recapcard.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// RecapProps configures a RecapCard. Kicker defaults "Recap"; When is the mono
// timeframe (default "earlier today"); Summary (optional) is the gist; Points
// (optional) are remembered items, each a teal-square bullet.
type RecapProps struct {
	Kicker  string
	When    string
	Summary string
	Points  []string
}

// RecapCard renders the daily-recap parchment card: orb header + kicker + when,
// an optional summary, and an optional teal-bulleted point list.
func RecapCard(p RecapProps) g.Node {
	kicker := p.Kicker
	if kicker == "" {
		kicker = "Recap"
	}
	when := p.When
	if when == "" {
		when = "earlier today"
	}
	kids := []g.Node{
		h.Class("recapcard"),
		h.Span(h.Class("recapcard-dot")),
		h.Header(h.Class("recapcard-head"),
			h.Img(h.Class("recapcard-orb"), h.Src("/static/icons/orb.png"), h.Alt(""), g.Attr("decoding", "async")),
			h.Span(h.Class("recapcard-kicker"), g.Text(kicker)),
			h.Span(h.Class("recapcard-when"), g.Text(when)),
		),
	}
	if p.Summary != "" {
		kids = append(kids, h.P(h.Class("recapcard-summary"), g.Text(p.Summary)))
	}
	if len(p.Points) > 0 {
		items := make([]g.Node, 0, len(p.Points)+1)
		items = append(items, h.Class("recapcard-points"))
		for _, pt := range p.Points {
			items = append(items, h.Li(h.Class("recapcard-point"),
				h.Span(h.Class("recapcard-sq"), g.Text("▪")),
				h.Span(g.Text(pt)),
			))
		}
		kids = append(kids, h.Ul(items...))
	}
	return h.Article(kids...)
}
```

- [ ] **Step 4: Run** `go test ./internal/ui/ -run TestRecapCard -v` — Expected: PASS (both).

- [ ] **Step 5: Append the RecapCard CSS** at the END of `basm.css`:
```css

/* ── RecapCard — the daily-recap parchment card ─────────────────────────── */
.recapcard {
  position: relative; color: var(--ink);
  background: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px;
  border: 2px solid var(--parch-edge); box-shadow: var(--parch-bevel); padding: 15px 17px 16px;
}
.recapcard-dot { position: absolute; top: 6px; right: 6px; width: 7px; height: 7px; background: var(--gold-ink); }
.recapcard-head { display: flex; align-items: center; gap: 9px; margin-bottom: 9px; }
.recapcard-orb { width: 20px; height: 20px; image-rendering: pixelated; }
.recapcard-kicker { font-family: var(--font-mono); font-size: 10px; font-weight: 700; letter-spacing: .1em; text-transform: uppercase; color: var(--teal-ink); }
.recapcard-when { margin-left: auto; font-family: var(--font-mono); font-size: 10px; text-transform: uppercase; letter-spacing: .05em; color: var(--ink-muted); }
.recapcard-summary { margin: 0 0 11px; font-size: 15px; line-height: 1.55; }
.recapcard-points { margin: 0; padding: 0; list-style: none; display: flex; flex-direction: column; gap: 6px; }
.recapcard-point { display: grid; grid-template-columns: 14px 1fr; gap: 8px; font-size: 13.5px; line-height: 1.45; color: var(--ink); }
.recapcard-sq { color: var(--teal-ink); }
```

- [ ] **Step 6: Register the story.** In `storybook.go` append:
```go

func recapCardCanvas() g.Node {
	return section("RecapCard",
		h.Div(h.Style("max-width:400px"),
			ui.RecapCard(ui.RecapProps{
				When: "earlier today", Summary: "We planned the orchard work and set the tomato watering. You asked me to keep two things.",
				Points: []string{"Garden — tomatoes & peppers, watered at dusk", "Notes exported as Markdown", "Mend the deer fence before the weekend"},
			})),
	)
}
```
In `story.go`, add immediately AFTER `{"dayentry", "Display", "DayEntry", dayEntryCanvas},`:
```go
	{"recapcard", "Cards", "RecapCard", recapCardCanvas},
```

- [ ] **Step 7: Verify + commit**
```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/ -run TestRecapCard && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
tail -16 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
git status --short
```
Expected: PASS, green, clean, NO RAW HEX. Stage only the 5 files, then commit:
```bash
git add internal/ui/recapcard.go internal/ui/recapcard_test.go internal/web/assets/static/basm.css internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add RecapCard organism + storybook story\n\nui.RecapCard(RecapProps) — the daily-recap parchment card (orb header, kicker/\nwhen, optional summary + teal-bulleted points). New tokenized .recapcard CSS.\nCards group.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: `ui.GuardianCard` + story

Reuses the existing `.dlg-corner*` brackets; composes `ui.Button` for the 3 actions.

**Files:** Create `internal/ui/guardiancard.go`, `internal/ui/guardiancard_test.go`; modify `basm.css`, `storybook.go`, `story.go`.

- [ ] **Step 1: Write the failing test** — Create `internal/ui/guardiancard_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestGuardianCard(t *testing.T) {
	got := render(t, ui.GuardianCard(ui.GuardianProps{
		Title: "Read your Documents folder?", Detail: "To find the budget spreadsheet.",
		Scope: "read · ~/Documents · this session", AllowOnceHref: "#",
	}))
	for _, want := range []string{
		`<article class="guardian">`,
		`<span class="dlg-corner dlg-corner-tl"></span>`,
		`<img class="guardian-icon" src="/static/icons/shield.png" alt="" decoding="async">`,
		`<span class="guardian-kicker">OS access</span>`,
		`<h3 class="guardian-title">Read your Documents folder?</h3>`,
		`<p class="guardian-detail">To find the budget spreadsheet.</p>`,
		`<div class="guardian-scope">read · ~/Documents · this session</div>`,
		`<a class="btn btn-primary btn-sm" href="#">Allow once</a>`,
		`<button class="btn btn-ghost btn-sm">Always</button>`,
		`<button class="btn btn-ghost btn-sm">Deny</button>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("guardian card missing %q in: %s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run** `go test ./internal/ui/ -run TestGuardianCard -v` — Expected: FAIL (undefined).

- [ ] **Step 3: Implement** — Create `internal/ui/guardiancard.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// GuardianProps configures a GuardianCard — an OS-access consent panel. Kicker
// defaults "OS access"; Title is the request; Detail (optional) is the why; Scope
// (optional) is the exact permission chip. The three Href fields wire the actions
// (Allow once = primary; Always/Deny = ghost; empty Href → plain buttons).
type GuardianProps struct {
	Kicker         string
	Title          string
	Detail         string
	Scope          string
	AllowOnceHref  string
	AllowAlwaysHref string
	DenyHref       string
}

// GuardianCard renders the gold-bracketed consent card: shield + kicker, the
// request title, optional detail + scope chip, and Allow-once / Always / Deny.
func GuardianCard(p GuardianProps) g.Node {
	kicker := p.Kicker
	if kicker == "" {
		kicker = "OS access"
	}
	kids := []g.Node{
		h.Class("guardian"),
		h.Span(h.Class("dlg-corner dlg-corner-tl")),
		h.Span(h.Class("dlg-corner dlg-corner-tr")),
		h.Span(h.Class("dlg-corner dlg-corner-bl")),
		h.Span(h.Class("dlg-corner dlg-corner-br")),
		h.Header(h.Class("guardian-head"),
			h.Img(h.Class("guardian-icon"), h.Src("/static/icons/shield.png"), h.Alt(""), g.Attr("decoding", "async")),
			h.Span(h.Class("guardian-kicker"), g.Text(kicker)),
		),
		h.H3(h.Class("guardian-title"), g.Text(p.Title)),
	}
	if p.Detail != "" {
		kids = append(kids, h.P(h.Class("guardian-detail"), g.Text(p.Detail)))
	}
	if p.Scope != "" {
		kids = append(kids, h.Div(h.Class("guardian-scope"), g.Text(p.Scope)))
	}
	kids = append(kids, h.Footer(h.Class("guardian-actions"),
		Button(ButtonProps{Size: "sm", Href: p.AllowOnceHref}, g.Text("Allow once")),
		Button(ButtonProps{Variant: "ghost", Size: "sm", Href: p.AllowAlwaysHref}, g.Text("Always")),
		Button(ButtonProps{Variant: "ghost", Size: "sm", Href: p.DenyHref}, g.Text("Deny")),
	))
	return h.Article(kids...)
}
```

- [ ] **Step 4: Run** `go test ./internal/ui/ -run TestGuardianCard -v` — Expected: PASS. (Note: the test uses empty Always/Deny Href → `ui.Button` renders `<button …>` not `<a>`; AllowOnce has Href "#" → `<a … href="#">`.)

- [ ] **Step 5: Append the GuardianCard CSS** at the END of `basm.css`:
```css

/* ── GuardianCard — OS-access consent (reuses .dlg-corner brackets) ──────── */
.guardian {
  position: relative; color: var(--ink);
  background: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px;
  border: 2px solid var(--gold-deep); box-shadow: var(--parch-bevel); padding: 16px 18px;
}
.guardian-head { display: flex; align-items: center; gap: 10px; margin-bottom: 10px; }
.guardian-icon { width: 24px; height: 24px; image-rendering: pixelated; }
.guardian-kicker { font-family: var(--font-mono); font-size: 10px; font-weight: 700; letter-spacing: .1em; text-transform: uppercase; color: var(--gold-ink); }
.guardian-title { margin: 0 0 7px; font-family: var(--font-display); font-size: 21px; color: var(--ink); line-height: 1.12; }
.guardian-detail { margin: 0 0 11px; font-size: 14.5px; line-height: 1.5; }
.guardian-scope {
  font-family: var(--font-mono); font-size: 11.5px; color: var(--ink-muted);
  background: var(--surface-2); border: 2px solid var(--parch-edge); box-shadow: var(--bevel-in);
  padding: 7px 10px; margin-bottom: 13px; overflow-wrap: anywhere;
}
.guardian-actions { display: flex; flex-wrap: wrap; gap: 9px; }
```

- [ ] **Step 6: Register the story.** In `storybook.go` append:
```go

func guardianCardCanvas() g.Node {
	return section("GuardianCard",
		h.Div(h.Style("max-width:400px"),
			ui.GuardianCard(ui.GuardianProps{
				Kicker: "OS access", Title: "Read your Documents folder?",
				Detail: "To find the budget spreadsheet you mentioned. Read-only, and only this once.",
				Scope:  "read · ~/Documents · this session",
				AllowOnceHref: "#", AllowAlwaysHref: "#", DenyHref: "#",
			})),
	)
}
```
In `story.go`, add immediately AFTER `{"recapcard", "Cards", "RecapCard", recapCardCanvas},`:
```go
	{"guardiancard", "Cards", "GuardianCard", guardianCardCanvas},
```

- [ ] **Step 7: Verify + commit** (same shape as Task 1 Step 7; tail-check the last 12 CSS lines):
```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/ -run TestGuardianCard && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
tail -12 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
git status --short
git add internal/ui/guardiancard.go internal/ui/guardiancard_test.go internal/web/assets/static/basm.css internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add GuardianCard organism + storybook story\n\nui.GuardianCard(GuardianProps) — the gold-bracketed OS-access consent card\n(shield + kicker, request title, scope chip, Allow-once/Always/Deny; reuses\n.dlg-corner + ui.Button). New tokenized .guardian CSS. Cards group.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 3: `ui.NudgeBanner` + story

**Files:** Create `internal/ui/nudgebanner.go`, `internal/ui/nudgebanner_test.go`; modify `basm.css`, `storybook.go`, `story.go`.

- [ ] **Step 1: Write the failing test** — Create `internal/ui/nudgebanner_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestNudgeBanner(t *testing.T) {
	got := render(t, ui.NudgeBanner(ui.NudgeProps{
		When: "18:00", Message: "The tomatoes thirst.",
		Replies: []ui.NudgeReply{{Label: "It is done.", Hint: "mark done"}, {Label: "At nightfall.", Hint: "snooze · 21:00"}},
	}))
	for _, want := range []string{
		`<div class="nudge">`,
		`<img class="nudge-icon" src="/static/icons/bell.png" alt="" decoding="async">`,
		`<span class="nudge-kicker">Nudge</span>`,
		`<span class="nudge-when">18:00</span>`,
		`<p class="nudge-msg">The tomatoes thirst.</p>`,
		`<button class="nudge-reply" type="button"><span>It is done.</span><span class="nudge-reply-hint">mark done</span></button>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("nudge banner missing %q in: %s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run** `go test ./internal/ui/ -run TestNudgeBanner -v` — Expected: FAIL (undefined).

- [ ] **Step 3: Implement** — Create `internal/ui/nudgebanner.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// NudgeReply is one owner reply: a Label and a mono Hint.
type NudgeReply struct {
	Label string
	Hint  string
}

// NudgeProps configures a NudgeBanner — the evening reminder. Kicker defaults
// "Nudge"; When is the mono time (default "18:00"); Message is the spoken ask;
// Replies are the owner's established answers.
type NudgeProps struct {
	Kicker  string
	When    string
	Message string
	Replies []NudgeReply
}

// NudgeBanner renders the evening nudge: a bell + kicker + time header, the spoken
// message, and the owner's reply buttons (label + hint).
func NudgeBanner(p NudgeProps) g.Node {
	kicker := p.Kicker
	if kicker == "" {
		kicker = "Nudge"
	}
	when := p.When
	if when == "" {
		when = "18:00"
	}
	replies := make([]g.Node, 0, len(p.Replies)+1)
	replies = append(replies, h.Class("nudge-replies"))
	for _, r := range p.Replies {
		replies = append(replies, h.Button(h.Class("nudge-reply"), h.Type("button"),
			h.Span(g.Text(r.Label)),
			h.Span(h.Class("nudge-reply-hint"), g.Text(r.Hint)),
		))
	}
	return h.Div(h.Class("nudge"),
		h.Div(h.Class("nudge-head"),
			h.Img(h.Class("nudge-icon"), h.Src("/static/icons/bell.png"), h.Alt(""), g.Attr("decoding", "async")),
			h.Span(h.Class("nudge-kicker"), g.Text(kicker)),
			h.Span(h.Class("nudge-when"), g.Text(when)),
		),
		h.P(h.Class("nudge-msg"), g.Text(p.Message)),
		h.Div(replies...),
	)
}
```

- [ ] **Step 4: Run** `go test ./internal/ui/ -run TestNudgeBanner -v` — Expected: PASS.

- [ ] **Step 5: Append the NudgeBanner CSS** at the END of `basm.css`:
```css

/* ── NudgeBanner — the evening reminder ─────────────────────────────────── */
.nudge {
  color: var(--ink);
  background: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px;
  border: 2px solid var(--gold-deep); box-shadow: var(--parch-bevel); padding: 13px 16px 14px;
}
.nudge-head { display: flex; align-items: center; gap: 10px; margin-bottom: 10px; }
.nudge-icon { width: 20px; height: 20px; image-rendering: pixelated; }
.nudge-kicker { font-family: var(--font-mono); font-size: 10px; font-weight: 700; letter-spacing: .1em; text-transform: uppercase; color: var(--gold-ink); }
.nudge-when { margin-left: auto; font-family: var(--font-mono); font-size: 10px; text-transform: uppercase; letter-spacing: .05em; color: var(--ink-muted); }
.nudge-msg { margin: 0 0 12px; font-size: 15.5px; line-height: 1.5; }
.nudge-replies { display: flex; flex-wrap: wrap; gap: 8px; }
.nudge-reply {
  display: inline-flex; align-items: baseline; gap: 9px; cursor: pointer; text-align: left;
  border-radius: 0; background: var(--surface-2); border: 2px solid var(--parch-edge);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, .4); padding: 7px 11px; font: 14px var(--font-body); color: var(--ink);
}
.nudge-reply-hint { font-family: var(--font-mono); font-size: 9.5px; text-transform: uppercase; letter-spacing: .04em; color: var(--ink-muted); }
```

- [ ] **Step 6: Register the story.** In `storybook.go` append:
```go

func nudgeBannerCanvas() g.Node {
	return section("NudgeBanner",
		h.Div(h.Style("max-width:440px"),
			ui.NudgeBanner(ui.NudgeProps{
				When: "18:00", Message: "The evening comes, and the tomatoes thirst. Will you tend them now?",
				Replies: []ui.NudgeReply{
					{Label: "It is done.", Hint: "mark done"},
					{Label: "At nightfall.", Hint: "snooze · 21:00"},
					{Label: "Tomorrow, I swear it.", Hint: "snooze · tomorrow"},
				},
			})),
	)
}
```
In `story.go`, add immediately AFTER `{"guardiancard", "Cards", "GuardianCard", guardianCardCanvas},`:
```go
	{"nudgebanner", "Cards", "NudgeBanner", nudgeBannerCanvas},
```

- [ ] **Step 7: Verify + commit**
```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/ -run TestNudgeBanner && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
tail -14 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
git status --short
git add internal/ui/nudgebanner.go internal/ui/nudgebanner_test.go internal/web/assets/static/basm.css internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add NudgeBanner organism + storybook story\n\nui.NudgeBanner(NudgeProps) — the evening reminder (bell + kicker + time, the\nspoken ask, owner reply buttons w/ hints). New tokenized .nudge CSS. Cards group.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 4: `ui.StatCard` + story

Composes `ui.Sparkline`. Delta tone drives the arrow, the delta colour, and the
sparkline stroke colour.

**Files:** Create `internal/ui/statcard.go`, `internal/ui/statcard_test.go`; modify `basm.css`, `storybook.go`, `story.go`.

- [ ] **Step 1: Write the failing test** — Create `internal/ui/statcard_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestStatCard(t *testing.T) {
	got := render(t, ui.StatCard(ui.StatProps{
		Icon: "gem", Label: "Weight", Value: "81.2", Unit: "kg", Delta: "0.6 this week",
		DeltaTone: "down", Data: []float64{83, 82.6, 82.1, 81.9, 81.2},
	}))
	for _, want := range []string{
		`<article class="statcard">`,
		`<img class="statcard-icon" src="/static/icons/gem.png" alt="" decoding="async">`,
		`<span class="statcard-label">Weight</span>`,
		`<span class="statcard-value">81.2</span>`,
		`<span class="statcard-unit">kg</span>`,
		`<span class="statcard-delta statcard-delta-down">▼ 0.6 this week</span>`,
		`<svg class="sparkline"`,
		`stroke="var(--ember-deep)"`, // down tone tints the sparkline
	} {
		if !strings.Contains(got, want) {
			t.Errorf("stat card missing %q in: %s", want, got)
		}
	}
}

func TestStatCardUpNoUnit(t *testing.T) {
	got := render(t, ui.StatCard(ui.StatProps{Icon: "gem", Label: "Steps", Value: "8,210", Delta: "12% vs avg", DeltaTone: "up", Data: []float64{6800, 7400, 8210}}))
	if !strings.Contains(got, `<span class="statcard-delta statcard-delta-up">▲ 12% vs avg</span>`) {
		t.Errorf("up delta missing: %s", got)
	}
	if strings.Contains(got, "statcard-unit") {
		t.Errorf("empty Unit should omit the unit span: %s", got)
	}
	if !strings.Contains(got, `stroke="var(--good-ink)"`) {
		t.Errorf("up tone should tint sparkline good-ink: %s", got)
	}
}
```

- [ ] **Step 2: Run** `go test ./internal/ui/ -run TestStatCard -v` — Expected: FAIL (undefined).

- [ ] **Step 3: Implement** — Create `internal/ui/statcard.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// StatProps configures a StatCard — a Life metric. Icon is a /static/icons name;
// Label is the metric name; Value is the figure; Unit (optional) follows it;
// Delta (optional) is the change; DeltaTone ("up"/"down"/"flat") drives the arrow,
// the delta colour, and the sparkline stroke; Data is the trend series.
type StatProps struct {
	Icon      string
	Label     string
	Value     string
	Unit      string
	Delta     string
	DeltaTone string
	Data      []float64
}

// StatCard renders the metric card: icon + label, the big value + unit + delta,
// and a tone-tinted Sparkline trend beneath.
func StatCard(p StatProps) g.Node {
	arrow, sparkColor := "", "var(--teal-ink)"
	switch p.DeltaTone {
	case "up":
		arrow, sparkColor = "▲ ", "var(--good-ink)"
	case "down":
		arrow, sparkColor = "▼ ", "var(--ember-deep)"
	}
	valueRow := []g.Node{h.Class("statcard-value-row"), h.Span(h.Class("statcard-value"), g.Text(p.Value))}
	if p.Unit != "" {
		valueRow = append(valueRow, h.Span(h.Class("statcard-unit"), g.Text(p.Unit)))
	}
	if p.Delta != "" {
		tone := p.DeltaTone
		if tone == "" {
			tone = "flat"
		}
		valueRow = append(valueRow, h.Span(h.Class("statcard-delta statcard-delta-"+tone), g.Text(arrow+p.Delta)))
	}
	return h.Article(h.Class("statcard"),
		h.Div(h.Class("statcard-head"),
			h.Img(h.Class("statcard-icon"), h.Src("/static/icons/"+p.Icon+".png"), h.Alt(""), g.Attr("decoding", "async")),
			h.Span(h.Class("statcard-label"), g.Text(p.Label)),
		),
		h.Div(valueRow...),
		Sparkline(SparkProps{Data: p.Data, Color: sparkColor, Width: 150, Height: 34}),
	)
}
```

- [ ] **Step 4: Run** `go test ./internal/ui/ -run TestStatCard -v` — Expected: PASS (both).

- [ ] **Step 5: Append the StatCard CSS** at the END of `basm.css`:
```css

/* ── StatCard — a Life metric: icon, value, delta, sparkline ────────────── */
.statcard {
  display: flex; flex-direction: column; gap: 11px; color: var(--ink);
  background: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px;
  border: 2px solid var(--parch-edge); box-shadow: var(--parch-bevel); padding: 15px 16px 14px;
}
.statcard-head { display: flex; align-items: center; gap: 8px; }
.statcard-icon { width: 18px; height: 18px; image-rendering: pixelated; }
.statcard-label { font-family: var(--font-mono); font-size: 10px; font-weight: 700; letter-spacing: .08em; text-transform: uppercase; color: var(--ink-muted); }
.statcard-value-row { display: flex; align-items: baseline; gap: 7px; flex-wrap: wrap; }
.statcard-value { font-family: var(--font-display); font-size: 32px; color: var(--ink); line-height: .9; }
.statcard-unit { font-family: var(--font-mono); font-size: 12px; color: var(--ink-muted); }
.statcard-delta { margin-left: auto; font-family: var(--font-mono); font-size: 11.5px; }
.statcard-delta-up { color: var(--good-ink); }
.statcard-delta-down { color: var(--ember-deep); }
.statcard-delta-flat { color: var(--ink-muted); }
```

- [ ] **Step 6: Register the story.** In `storybook.go` append:
```go

func statCardCanvas() g.Node {
	box := func(n g.Node) g.Node { return h.Div(h.Style("max-width:260px"), n) }
	return section("StatCard",
		box(ui.StatCard(ui.StatProps{Icon: "gem", Label: "Weight", Value: "81.2", Unit: "kg", Delta: "0.6 this week", DeltaTone: "down", Data: []float64{83, 82.6, 82.1, 82.4, 81.9, 81.6, 81.2}})),
		box(ui.StatCard(ui.StatProps{Icon: "gem", Label: "Steps", Value: "8,210", Delta: "12% vs avg", DeltaTone: "up", Data: []float64{6800, 7100, 7400, 7900, 8100, 8000, 8210}})),
	)
}
```
In `story.go`, add immediately AFTER `{"nudgebanner", "Cards", "NudgeBanner", nudgeBannerCanvas},`:
```go
	{"statcard", "Cards", "StatCard", statCardCanvas},
```

- [ ] **Step 7: Verify + commit**
```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/ -run TestStatCard && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
tail -16 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
git status --short
git add internal/ui/statcard.go internal/ui/statcard_test.go internal/web/assets/static/basm.css internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add StatCard organism + storybook story\n\nui.StatCard(StatProps) — a Life metric card (icon+label, big value+unit+delta,\ntone-tinted ui.Sparkline trend). New tokenized .statcard CSS. Cards group.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 5: `Topbar` story (export `shell.Topbar`)

**Files:** Modify `internal/ui/shell/shell.go` (export `topbar`→`Topbar`), `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`, `internal/web/assets/static/basm.css` (one storybook override).

- [ ] **Step 1: Export the shell topbar**

In `internal/ui/shell/shell.go`, rename the unexported `func topbar(active string) g.Node` to `func Topbar(active string) g.Node` and update its sole caller in the same file (the `Page`/`SidebarPage` body that calls `topbar(...)` → `Topbar(...)`). Keep the doc comment. Run `go build ./internal/ui/shell/` to confirm no other caller breaks.

- [ ] **Step 2: Add the storybook override CSS** at the END of `basm.css`:
```css

/* ── Storybook: contain the sticky .topbar inside the canvas tile ───────── */
.sb-canvas .topbar { position: static; }
```

- [ ] **Step 3: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, add the shell import to the import block (after the `chat` import):
```go
	"github.com/alexradunet/balaur/internal/ui/shell"
```
Then append:
```go

func topbarCanvas() g.Node {
	return section("Topbar", h.Div(h.Style("position:relative"), shell.Topbar("storybook")))
}
```
In `story.go`, add immediately AFTER `{"statcard", "Cards", "StatCard", statCardCanvas},`:
```go
	{"topbar", "Navigation", "Topbar", topbarCanvas},
```

- [ ] **Step 4: Verify + commit**
```bash
cd /home/alex/Projects/balaur
go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./... && go vet ./...
git status --short
```
Expected: green, build+vet clean. Note: the existing `shell_test.go` `TestPage` asserts the rendered topbar markup — it still passes (the rename is internal; the rendered HTML is unchanged). Stage only the four files, then:
```bash
git add internal/ui/shell/shell.go internal/feature/storybook/storybook.go internal/feature/storybook/story.go internal/web/assets/static/basm.css
git commit -m "$(printf 'feat(ui): add Topbar storybook story (export shell.Topbar)\n\nExport the existing shell topbar renderer and register a Navigation-group story\n(wrapped + .sb-canvas override so the sticky bar sits inside the canvas tile).\nCompletes the 1:1 component catalog.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification (controller — visual)

- [ ] `go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check` — green.
- [ ] `/storybook/recapcard|guardiancard|nudgebanner|statcard|topbar` render 200 (content-assert the class).
- [ ] Screenshot the five (Hearthwood): the recap card with points; the gold-bracketed guardian consent with 3 buttons; the nudge banner with reply buttons; the two stat cards with tone-tinted sparklines + ▲/▼ deltas; the topbar tile.
- [ ] Live chat untouched: `git diff --stat main..HEAD -- web/templates/*.html internal/web/chat*.go` empty.

## What this delivers / what's next

**Delivered:** the last five gap components — the catalog is now **1:1 complete** with the export (Chrome page intentionally dropped for flat-dither; the 3 Screens deferred). The Cards group has TaskCard/KnowledgeCard/RecapCard/GuardianCard/NudgeBanner/StatCard; Topbar joins Navigation.

**Next:** the deferred wiring slice (route the live chat gateway through the chat organisms) and the Screens, when ready.
