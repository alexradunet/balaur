# Gomponents Atomic — Typography Atoms (Plan 02b-vi) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the last two catalog text helpers — `SectionLabel` and `ScreenTitle` — to `internal/ui`, with new tokenized CSS, registering both as storybook stories under a new "Typography" group.

**Architecture:** `SectionLabel` is a mono uppercase caption followed by a flex trailing hairline rule; its accent color defaults to `--gold` and is overridable per-instance via an inline `--sl-accent` custom property (the same idiom `ui.Avatar` uses for `--avatar-size`). `ScreenTitle` is an optional mono eyebrow over a display `<h1>` headline with `clamp()` fluid sizing. Both sit on the page background, so they use the page-bg tokens `--gold`/`--fg-strong`. Pure server-rendered; new CSS lifts the export's `Screens.js` inline styles into tokenized `basm.css` rules.

**Tech Stack:** Go, gomponents (`h.Style(v)` is the style-attribute helper), vanilla `basm.css`.

**Scope:** Plan 02b-vi (Phase 2b catalog — the FINAL atom slice). Next is organisms.

**Conventions:** package `ui` uses QUALIFIED `g`/`h` imports (NO dot-import — `func Button` collides with `html.Button`). New CSS appends at the END of `basm.css`, tokenized (`var(--token)`, no raw hex, single-dash classes). Atom tests are `package ui_test` and use the shared `render(t, node)` helper (in `internal/ui/helpers_test.go` — do NOT redefine). Stories: append a canvas func to `internal/feature/storybook/storybook.go` and a `Story` entry to the `stories` slice in `internal/feature/storybook/story.go` (positional `{ID, Group, Title, Canvas}`). After each task: `go test ./...`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`. If `git status` shows any file other than the task's own as modified (e.g. a stray `chatstream.go` from a linter), do NOT stage it — `git checkout --` it.

---

## File Structure

- **Create** `internal/ui/sectionlabel.go`+`_test.go`, `internal/ui/screentitle.go`+`_test.go`.
- **Modify** `internal/web/assets/static/basm.css` (append SectionLabel CSS, then ScreenTitle CSS).
- **Modify** `internal/feature/storybook/storybook.go` (canvas funcs) + `internal/feature/storybook/story.go` (register both under a new "Typography" group).

Verified facts (do not re-derive): tokens `--gold`, `--hair`, `--fg-strong`, `--font-display`, `--font-mono`, `--smoke` all exist in `basm.css`. Classes `.section-label*` and `.screen-title*` are unused. `h.Style(v string)` sets the `style` attribute (per `internal/ui/avatar.go:40`).

---

## Task 1: `ui.SectionLabel` + story

A mono uppercase caption + a trailing dashed hairline rule that fills the row.
`Accent` (optional CSS color token ref, e.g. `var(--smoke)`) overrides the gold
caption color via an inline `--sl-accent` custom property.

**Files:** Create `internal/ui/sectionlabel.go`, `internal/ui/sectionlabel_test.go`; modify `internal/web/assets/static/basm.css`, `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/sectionlabel_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestSectionLabelDefault(t *testing.T) {
	got := render(t, ui.SectionLabel(ui.SectionLabelProps{Text: "Today"}))
	want := `<div class="section-label"><span class="section-label-text">Today</span><span class="section-label-rule"></span></div>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestSectionLabelAccent(t *testing.T) {
	got := render(t, ui.SectionLabel(ui.SectionLabelProps{Text: "This week", Accent: "var(--smoke)"}))
	if !strings.Contains(got, `<div class="section-label" style="--sl-accent:var(--smoke)">`) {
		t.Errorf("accent should set the --sl-accent custom property: %s", got)
	}
	if !strings.Contains(got, `<span class="section-label-text">This week</span>`) {
		t.Errorf("label text missing: %s", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestSectionLabel -v` — Expected: FAIL (`undefined: ui.SectionLabel`/`ui.SectionLabelProps`).

- [ ] **Step 3: Implement the atom**

Create `internal/ui/sectionlabel.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// SectionLabelProps configures a SectionLabel. Text is the caption; Accent is an
// optional CSS color reference (e.g. "var(--smoke)") that overrides the gold
// caption color via the inline --sl-accent custom property.
type SectionLabelProps struct {
	Text   string
	Accent string
}

// SectionLabel renders a mono uppercase caption followed by a trailing dashed
// hairline rule that fills the rest of the row.
func SectionLabel(p SectionLabelProps) g.Node {
	root := []g.Node{h.Class("section-label")}
	if p.Accent != "" {
		root = append(root, h.Style("--sl-accent:"+p.Accent))
	}
	root = append(root,
		h.Span(h.Class("section-label-text"), g.Text(p.Text)),
		h.Span(h.Class("section-label-rule")),
	)
	return h.Div(root...)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestSectionLabel -v` — Expected: PASS (both).

- [ ] **Step 5: Append the SectionLabel CSS to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`:
```css

/* ── SectionLabel — mono caption + trailing dashed hairline rule ─────────── */
.section-label { display: flex; align-items: center; gap: 10px; margin-bottom: 13px; }
.section-label-text {
  font-family: var(--font-mono); font-size: 11px; letter-spacing: .1em; text-transform: uppercase;
  color: var(--sl-accent, var(--gold)); white-space: nowrap;
}
.section-label-rule {
  flex: 1; height: 2px;
  background: linear-gradient(to right, var(--hair) 50%, transparent 50%) 0 0 / 8px 2px repeat-x;
}
```

- [ ] **Step 6: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append:
```go

func sectionLabelCanvas() g.Node {
	return section("SectionLabel",
		ui.SectionLabel(ui.SectionLabelProps{Text: "Today"}),
		ui.SectionLabel(ui.SectionLabelProps{Text: "This week", Accent: "var(--smoke)"}),
	)
}
```
In `internal/feature/storybook/story.go`, add to the `stories` slice immediately AFTER the `{"dialog", "Feedback", "Dialog", dialogCanvas},` line:
```go
	{"sectionlabel", "Typography", "SectionLabel", sectionLabelCanvas},
```

- [ ] **Step 7: Verify + commit**

Run:
```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/ -run TestSectionLabel && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
tail -12 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
git status --short
```
Expected: PASS, full suite green, build clean, NO RAW HEX. If `git status --short` shows any file other than `internal/ui/sectionlabel.go`, `internal/ui/sectionlabel_test.go`, `internal/web/assets/static/basm.css`, `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`, do NOT stage it — `git checkout -- <file>`. Then:
```bash
git add internal/ui/sectionlabel.go internal/ui/sectionlabel_test.go internal/web/assets/static/basm.css internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add SectionLabel atom + storybook story\n\nui.SectionLabel(SectionLabelProps) — a mono uppercase caption + trailing dashed\nhairline rule; Accent overrides the gold via an inline --sl-accent custom\nproperty. New tokenized .section-label CSS. New Typography group.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: `ui.ScreenTitle` + story

An optional mono eyebrow over a display `<h1>` headline with fluid `clamp()`
sizing — the page header used at the top of each screen.

**Files:** Create `internal/ui/screentitle.go`, `internal/ui/screentitle_test.go`; modify `internal/web/assets/static/basm.css`, `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/screentitle_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestScreenTitleFull(t *testing.T) {
	got := render(t, ui.ScreenTitle(ui.ScreenTitleProps{Eyebrow: "Tuesday", Title: "On the book."}))
	want := `<div class="screen-title"><div class="screen-title-eyebrow">Tuesday</div><h1 class="screen-title-head">On the book.</h1></div>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestScreenTitleNoEyebrow(t *testing.T) {
	got := render(t, ui.ScreenTitle(ui.ScreenTitleProps{Title: "Memory"}))
	want := `<div class="screen-title"><h1 class="screen-title-head">Memory</h1></div>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
	if strings.Contains(got, "screen-title-eyebrow") {
		t.Errorf("no-eyebrow title should omit the eyebrow div: %s", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestScreenTitle -v` — Expected: FAIL (`undefined: ui.ScreenTitle`/`ui.ScreenTitleProps`).

- [ ] **Step 3: Implement the atom**

Create `internal/ui/screentitle.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ScreenTitleProps configures a ScreenTitle. Eyebrow (optional) is the mono
// uppercase kicker; Title is the display headline.
type ScreenTitleProps struct {
	Eyebrow string
	Title   string
}

// ScreenTitle renders a page header: an optional mono eyebrow over a display
// <h1> with fluid clamp() sizing.
func ScreenTitle(p ScreenTitleProps) g.Node {
	kids := []g.Node{h.Class("screen-title")}
	if p.Eyebrow != "" {
		kids = append(kids, h.Div(h.Class("screen-title-eyebrow"), g.Text(p.Eyebrow)))
	}
	kids = append(kids, h.H1(h.Class("screen-title-head"), g.Text(p.Title)))
	return h.Div(kids...)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestScreenTitle -v` — Expected: PASS (both).

- [ ] **Step 5: Append the ScreenTitle CSS to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`:
```css

/* ── ScreenTitle — mono eyebrow over a display headline ──────────────────── */
.screen-title-eyebrow {
  font-family: var(--font-mono); font-size: 11px; letter-spacing: .12em; text-transform: uppercase;
  color: var(--gold); margin-bottom: 7px;
}
.screen-title-head {
  margin: 0; font-family: var(--font-display); font-size: clamp(28px, 5vw, 40px); color: var(--fg-strong); line-height: 1;
}
```

- [ ] **Step 6: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append:
```go

func screenTitleCanvas() g.Node {
	return section("ScreenTitle",
		ui.ScreenTitle(ui.ScreenTitleProps{Eyebrow: "Tuesday · 14 May", Title: "On the book."}),
		ui.ScreenTitle(ui.ScreenTitleProps{Title: "Memory"}),
	)
}
```
In `internal/feature/storybook/story.go`, add to the `stories` slice immediately AFTER the `{"sectionlabel", "Typography", "SectionLabel", sectionLabelCanvas},` line:
```go
	{"screentitle", "Typography", "ScreenTitle", screenTitleCanvas},
```

- [ ] **Step 7: Verify + commit**

Run:
```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/ -run TestScreenTitle && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./... && go vet ./...
tail -10 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
git status --short
```
Expected: PASS, full suite green, build+vet clean, NO RAW HEX. If `git status --short` shows any file other than the five task files, do NOT stage it — `git checkout -- <file>`. Then:
```bash
git add internal/ui/screentitle.go internal/ui/screentitle_test.go internal/web/assets/static/basm.css internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add ScreenTitle atom + storybook story\n\nui.ScreenTitle(ScreenTitleProps) — optional mono eyebrow over a display <h1>\nwith fluid clamp() sizing. New tokenized .screen-title CSS. Registered under\nTypography.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification

- [ ] `go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check` — all green.
- [ ] No raw hex in the new CSS: `tail -20 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"`.
- [ ] Sidebar has a new **Typography** group (SectionLabel, ScreenTitle); `/storybook/sectionlabel` and `/storybook/screentitle` render 200.

## What this delivers / what's next

**Delivered:** the final two catalog text helpers (`SectionLabel`, `ScreenTitle`); both registered as stories under a new Typography group. `internal/ui` reaches **22 atoms** — the atom catalog is complete.

**Next:** ORGANISMS — the chat Message/ToolRow/Composer and the knowledge/task cards — which begin *replacing* real `html/template` surfaces rather than adding to the catalog, → Phase 3 (storybook → `/`), → cut boards + delete `web/` (Phase 6).
