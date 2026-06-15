# Storybook Shell + Reusable Sidebar Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the flat `/storybook` gallery with a routed storybook shell — a reusable `shell.Sidebar` beside a canvas — driven by a story registry, with `/storybook` (Overview) and `/storybook/{id}` (one component per page).

**Architecture:** A generic `Sidebar` (sections→items+active) and a `SidebarPage` doc layout live in `internal/ui/shell`. A pure-data story registry in `internal/feature/storybook` is the single source for both the sidebar nav and the routes; the `internal/web` gateway groups stories into the sidebar and renders per-route pages. New tokenized storybook-chrome CSS is appended to `basm.css`.

**Tech Stack:** Go, gomponents (`maragu.dev/gomponents` + `.../html`), PocketBase router, vanilla `basm.css`.

**Spec:** `docs/superpowers/specs/2026-06-15-storybook-shell-sidebar-design.md`.

**Conventions:** package `ui`/`shell` use qualified `g`/`h` imports (no dot-import). New CSS appends at the END of `internal/web/assets/static/basm.css`, tokenized (`var(--token)`, no raw hex, single-dash). After each task: `go test ./...`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`. Shell tests are `package shell_test` and render inline (`var b strings.Builder; n.Render(&b)`), matching the existing `shell_test.go`.

---

## File Structure

- **Create** `internal/ui/shell/sidebar.go` — `Sidebar`, `SidebarPage`, their prop types.
- **Create** `internal/ui/shell/sidebar_test.go` — golden tests.
- **Create** `internal/feature/storybook/story.go` — `Story`, `Stories()`, `Lookup()`.
- **Create** `internal/feature/storybook/overview.go` — `Overview()`.
- **Create** `internal/feature/storybook/story_test.go` — registry + overview tests.
- **Modify** `internal/feature/storybook/storybook.go` — split `Body()` into per-component canvas funcs.
- **Modify** `internal/web/assets/static/basm.css` — append storybook-chrome CSS.
- **Modify** `internal/web/storybook.go` — replace the single handler with Overview + per-id; add `sidebarFor`.
- **Modify** `internal/web/web.go` — register `/storybook` and `/storybook/{id}`.

---

## Task 1: `shell.Sidebar` component

**Files:** Create `internal/ui/shell/sidebar.go`, `internal/ui/shell/sidebar_test.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/shell/sidebar_test.go`:
```go
package shell_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui/shell"
)

func TestSidebar(t *testing.T) {
	var b strings.Builder
	n := shell.Sidebar(shell.SidebarProps{
		Brand: g.Text("BALAUR"),
		Sections: []shell.SidebarSection{{
			Label: "Atoms",
			Items: []shell.SidebarItem{
				{Label: "Button", Href: "/storybook/button", Active: true},
				{Label: "Tag", Href: "/storybook/tag"},
			},
		}},
		Footer: g.Text("FOOT"),
	})
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	for _, want := range []string{
		`<aside class="sb-side">`,
		`<header class="sb-brand">BALAUR</header>`,
		`<div class="sb-nav-label">Atoms</div>`,
		`<a class="sb-nav-item sb-nav-item-active" href="/storybook/button" aria-current="page">Button</a>`,
		`<a class="sb-nav-item" href="/storybook/tag">Tag</a>`,
		`<footer class="sb-foot">FOOT</footer>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("sidebar missing %q in: %s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/shell/ -run TestSidebar -v` — Expected: FAIL (`undefined: shell.Sidebar`).

- [ ] **Step 3: Implement the component**

Create `internal/ui/shell/sidebar.go`:
```go
package shell

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// SidebarItem is one nav link. Active marks the current page.
type SidebarItem struct {
	Label, Href string
	Active      bool
}

// SidebarSection is a labelled group of nav items.
type SidebarSection struct {
	Label string
	Items []SidebarItem
}

// SidebarProps configures a Sidebar. Brand (optional) is the header content;
// Footer (optional) is the pinned bottom slot (e.g. a theme toggle).
type SidebarProps struct {
	Brand    g.Node
	Sections []SidebarSection
	Footer   g.Node
}

// Sidebar renders the reusable wood nav rail: an optional brand header, grouped
// nav links, and an optional pinned footer. Generic — it has no knowledge of the
// storybook; callers supply the sections. The active item carries
// aria-current="page" and the sb-nav-item-active class.
func Sidebar(p SidebarProps) g.Node {
	groups := make([]g.Node, 0, len(p.Sections))
	for _, sec := range p.Sections {
		items := []g.Node{h.Div(h.Class("sb-nav-label"), g.Text(sec.Label))}
		for _, it := range sec.Items {
			items = append(items, sidebarItem(it))
		}
		groups = append(groups, h.Div(h.Class("sb-nav-group"), g.Group(items)))
	}
	kids := []g.Node{h.Class("sb-side")}
	if p.Brand != nil {
		kids = append(kids, h.Header(h.Class("sb-brand"), p.Brand))
	}
	kids = append(kids, h.Nav(h.Class("sb-nav"), g.Group(groups)))
	if p.Footer != nil {
		kids = append(kids, h.Footer(h.Class("sb-foot"), p.Footer))
	}
	return h.Aside(kids...)
}

func sidebarItem(it SidebarItem) g.Node {
	cls := "sb-nav-item"
	if it.Active {
		cls += " sb-nav-item-active"
	}
	attrs := []g.Node{h.Class(cls), h.Href(it.Href)}
	if it.Active {
		attrs = append(attrs, h.Aria("current", "page"))
	}
	attrs = append(attrs, g.Text(it.Label))
	return h.A(attrs...)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/shell/ -run TestSidebar -v` — Expected: PASS.

- [ ] **Step 5: Commit**
```bash
git add internal/ui/shell/sidebar.go internal/ui/shell/sidebar_test.go
git commit -m "$(printf 'feat(ui/shell): add reusable Sidebar component\n\nGeneric wood nav rail (sections -> items + active, optional brand/footer).\nActive item carries aria-current. No storybook knowledge; callers feed data.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: `shell.SidebarPage` layout

The full `<html>` doc for a sidebar surface: head (reusing `pageHead()`) + a
`.sb-root` grid of the sidebar + a `.sb-canvas` main with a breadcrumb.

**Files:** Modify `internal/ui/shell/sidebar.go`; add a test to `internal/ui/shell/sidebar_test.go`.

- [ ] **Step 1: Write the failing test**

Append to `internal/ui/shell/sidebar_test.go`:
```go
func TestSidebarPage(t *testing.T) {
	var b strings.Builder
	n := shell.SidebarPage(shell.SidebarPageProps{
		Title:   "Button",
		Sidebar: g.El("aside", g.Text("SIDE")),
		Crumb:   "Button",
		Body:    g.Text("CANVAS"),
	})
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	for _, want := range []string{
		"<!doctype html>",
		`<title>Button · Balaur</title>`,
		`<link rel="stylesheet" href="/static/basm.css">`,
		`<div class="sb-root">`,
		`<aside>SIDE</aside>`,
		`<main class="sb-canvas">`,
		`<header class="sb-crumb">Storybook / Button</header>`,
		`CANVAS`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("sidebar page missing %q in: %s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/shell/ -run TestSidebarPage -v` — Expected: FAIL (`undefined: shell.SidebarPage`).

- [ ] **Step 3: Implement the layout**

Append to `internal/ui/shell/sidebar.go`:
```go

// SidebarPageProps configures a SidebarPage. Crumb is the breadcrumb tail
// (empty -> just "Storybook"); Body fills the canvas; Sidebar is the rail node.
type SidebarPageProps struct {
	Title   string
	Sidebar g.Node
	Crumb   string
	Body    g.Node
}

// SidebarPage renders a full <html> document for a sidebar surface: the shared
// page head, then a .sb-root grid of the sidebar and a scrollable .sb-canvas
// main with a breadcrumb header. No app #dock — this is its own surface.
func SidebarPage(p SidebarPageProps) g.Node {
	crumb := "Storybook"
	if p.Crumb != "" {
		crumb = "Storybook / " + p.Crumb
	}
	return g.Group([]g.Node{
		g.Raw("<!doctype html>"),
		h.HTML(h.Lang("en"),
			h.Head(pageHead(), h.TitleEl(g.Text(p.Title+" · Balaur"))),
			h.Body(
				h.Div(h.Class("sb-root"),
					p.Sidebar,
					h.Main(h.Class("sb-canvas"),
						h.Header(h.Class("sb-crumb"), g.Text(crumb)),
						p.Body,
					),
				),
			),
		),
	})
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/shell/ -v` — Expected: PASS (TestSidebar, TestSidebarPage, TestPage).

- [ ] **Step 5: Commit**
```bash
git add internal/ui/shell/sidebar.go internal/ui/shell/sidebar_test.go
git commit -m "$(printf 'feat(ui/shell): add SidebarPage layout\n\nFull <html> doc: shared head + .sb-root grid (sidebar + .sb-canvas with a\nbreadcrumb). Reuses pageHead; no app #dock. The storybook surface frame.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 3: Storybook-chrome CSS

Append the layout CSS for the sidebar + canvas + overview tiles. Tokenized.

**Files:** Modify `internal/web/assets/static/basm.css`.

- [ ] **Step 1: Append the CSS block to the END of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`:
```css

/* ── Storybook shell — sidebar rail + canvas ─────────────────────────── */
.sb-root { display: grid; grid-template-columns: 232px 1fr; min-height: 100vh; }
.sb-side {
  display: flex; flex-direction: column; height: 100vh;
  position: sticky; top: 0; overflow-y: auto;
  background: var(--chrome); background-image: var(--wood-planks);
  border-right: 2px solid var(--outline-2); box-shadow: var(--bevel-up);
}
.sb-brand {
  display: flex; align-items: center; gap: 8px; padding: 14px 16px;
  font-family: var(--font-pixel); font-size: 13px; letter-spacing: .06em;
  color: var(--gold); border-bottom: 2px solid var(--outline-2);
}
.sb-brand img { width: 22px; height: 22px; image-rendering: pixelated; }
.sb-nav { flex: 1; padding: 8px 0; }
.sb-nav-group { padding: 6px 0; }
.sb-nav-label {
  padding: 6px 16px 4px; font-family: var(--font-mono); font-size: 9.5px;
  font-weight: 700; letter-spacing: .1em; text-transform: uppercase; color: var(--muted);
}
.sb-nav-item {
  display: block; padding: 6px 16px; text-decoration: none;
  font-family: var(--font-mono); font-size: 12px; letter-spacing: .02em;
  color: var(--chrome-fg); border-left: 3px solid transparent;
}
.sb-nav-item:hover { filter: brightness(1.18); }
.sb-nav-item-active { color: var(--gold); background: var(--chrome-2); border-left-color: var(--ember); }
.sb-foot { padding: 12px 16px; border-top: 2px solid var(--outline-2); }
.sb-canvas { height: 100vh; overflow-y: auto; padding: 0 5vw 96px; background: var(--bg); }
.sb-crumb {
  position: sticky; top: 0; z-index: 5; margin: 0 0 8px; padding: 14px 0;
  font-family: var(--font-mono); font-size: 10.5px; letter-spacing: .08em;
  text-transform: uppercase; color: var(--muted);
  background: var(--bg); border-bottom: 2px dashed var(--hair);
}
/* Overview stat tiles + tiers */
.sb-lede { margin: 18px 0 6px; font-family: var(--font-display); font-size: 30px; color: var(--fg-strong); }
.sb-stats { display: flex; flex-wrap: wrap; gap: 14px; margin: 14px 0 28px; }
.sb-stat {
  padding: 12px 18px; min-width: 92px; color: var(--ink);
  background: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px;
  border: 2px solid var(--parch-edge); box-shadow: var(--parch-bevel);
}
.sb-stat b { display: block; font-family: var(--font-display); font-size: 24px; color: var(--ink); }
.sb-stat span { font-family: var(--font-mono); font-size: 9.5px; text-transform: uppercase; letter-spacing: .06em; color: var(--ink-muted); }
```

- [ ] **Step 2: Verify build + suite + token test**

Run:
```bash
CGO_ENABLED=0 go build ./... && go test ./internal/web/assets/ && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
```
Expected: build clean, assets token test passes (no undefined tokens / raw hex introduced), full suite green. Confirm no raw hex in the appended block:
```bash
tail -45 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
```

- [ ] **Step 3: Commit**
```bash
git add internal/web/assets/static/basm.css
git commit -m "$(printf 'feat(storybook): add sidebar-shell chrome CSS\n\nTokenized .sb-root grid, .sb-side wood rail (+ brand/nav/active/foot),\n.sb-canvas + .sb-crumb, and the Overview stat tiles. No raw hex.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 4: Story registry + per-component canvases + Overview

Split the current `Body()` sections into per-component canvas funcs, add the
pure-data registry, and the Overview. `Body()` stays (composing the canvases) so
the build + existing `TestBodyRendersAtoms` remain green; it is removed in Task 5.

**Files:** Modify `internal/feature/storybook/storybook.go`; create `internal/feature/storybook/story.go`, `internal/feature/storybook/overview.go`, `internal/feature/storybook/story_test.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/feature/storybook/story_test.go`:
```go
package storybook_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/storybook"
)

func TestStoriesUniqueAndLookup(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range storybook.Stories() {
		if s.ID == "" || s.Group == "" || s.Title == "" || s.Canvas == nil {
			t.Fatalf("incomplete story: %+v", s)
		}
		if seen[s.ID] {
			t.Fatalf("duplicate story id %q", s.ID)
		}
		seen[s.ID] = true
	}
	if len(seen) < 15 {
		t.Errorf("expected >=15 stories, got %d", len(seen))
	}
	if _, ok := storybook.Lookup("button"); !ok {
		t.Error(`Lookup("button") not found`)
	}
	if _, ok := storybook.Lookup("nope"); ok {
		t.Error(`Lookup("nope") should be false`)
	}
}

func TestButtonCanvasRenders(t *testing.T) {
	s, _ := storybook.Lookup("button")
	var b strings.Builder
	if err := s.Canvas().Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := b.String(); !strings.Contains(got, `class="btn btn-primary"`) {
		t.Errorf("button canvas missing button: %s", got)
	}
}

func TestOverviewRenders(t *testing.T) {
	var b strings.Builder
	if err := storybook.Overview().Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	for _, want := range []string{"Woven, not rendered.", `class="sb-stats"`, `href="/storybook/button"`} {
		if !strings.Contains(got, want) {
			t.Errorf("overview missing %q in: %s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/feature/storybook/ -run 'TestStories|TestButtonCanvas|TestOverview' -v` — Expected: FAIL (`undefined: storybook.Stories`/`Lookup`/`Overview`/`Story`).

- [ ] **Step 3: Replace storybook.go with per-component canvases**

Replace the ENTIRE contents of `internal/feature/storybook/storybook.go` with (each
former section becomes a `*Canvas()` func; `Body()` composes them so existing
callers/tests keep working; the `section` helper is unchanged):
```go
// Package storybook builds the Hearthwood component gallery — the storybook
// surface. Each component is a Story with a Canvas() of its variants; the
// registry (story.go) is the single source for the sidebar nav and the routes.
// Renders from in-package fixtures only (never PocketBase), so it works on an
// empty database.
package storybook

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// Body is the legacy all-in-one gallery (every canvas stacked). Retained so the
// build stays green during the routed-shell migration; removed once the routed
// handler lands.
func Body() g.Node {
	canvases := make([]g.Node, 0, len(Stories())+1)
	canvases = append(canvases, h.H1(g.Text("Balaur — Hearthwood storybook")))
	for _, s := range Stories() {
		canvases = append(canvases, s.Canvas())
	}
	return h.Div(h.Class("sb"), g.Group(canvases))
}

// section wraps a labelled group of component variants.
func section(label string, items ...g.Node) g.Node {
	return h.Section(h.Class("sb-section"),
		h.H2(g.Text(label)),
		h.Div(h.Class("sb-row"), g.Group(items)),
	)
}

func buttonCanvas() g.Node {
	return section("Button",
		ui.Button(ui.ButtonProps{}, g.Text("Primary")),
		ui.Button(ui.ButtonProps{Variant: "ghost"}, g.Text("Ghost")),
		ui.Button(ui.ButtonProps{Variant: "wood"}, g.Text("Wood")),
		ui.Button(ui.ButtonProps{Size: "sm"}, g.Text("Small")),
	)
}

func tagCanvas() g.Node {
	return section("Tag", ui.Tag(g.Text("daily")), ui.Tag(g.Text("⟳ weekly")))
}

func pipsCanvas() g.Node {
	return section("Pips", ui.Pips(1, 5, ""), ui.Pips(3, 5, ""), ui.Pips(5, 5, ""))
}

func cardCanvas() g.Node {
	return section("Card", ui.Card(h.H3(g.Text("A parchment card")), h.P(g.Text("Body text on parchment."))))
}

func stitchCanvas() g.Node  { return section("Stitch", ui.Stitch()) }
func folkbandCanvas() g.Node { return section("FolkBand", ui.FolkBand()) }

func avatarCanvas() g.Node {
	return section("Avatar",
		ui.Avatar(ui.AvatarProps{Src: "/static/avatars/balaur-01.png", Kind: "balaur", Alt: "Wise"}),
		ui.Avatar(ui.AvatarProps{Src: "/static/avatars/balaur-01.png", State: "thinking"}),
		ui.Avatar(ui.AvatarProps{Src: "/static/avatars/soul-01.png", Kind: "soul", Alt: "Owner"}),
	)
}

func iconCanvas() g.Node {
	return section("Icon", ui.Icon("scroll"), ui.Icon("tome"), ui.Icon("quill"), ui.Icon("lens"), ui.Icon("flame"))
}

func badgeCanvas() g.Node {
	return section("Badge",
		ui.Badge(ui.BadgeProps{}, g.Text("3")),
		ui.Badge(ui.BadgeProps{Tone: ui.BadgeEmber}, g.Text("9")),
		ui.Badge(ui.BadgeProps{Tone: ui.BadgeTeal}, g.Text("new")),
		ui.Badge(ui.BadgeProps{Tone: ui.BadgeWood}, g.Text("draft")),
		ui.Badge(ui.BadgeProps{Tone: ui.BadgeGold, Dot: true}),
		ui.Badge(ui.BadgeProps{Tone: ui.BadgeEmber, Dot: true}),
	)
}

func alertCanvas() g.Node {
	return section("Alert",
		ui.Alert(ui.AlertProps{Tone: "info", Title: "Heads up"}, g.Text("Your data stays on the box unless you switch models yourself.")),
		ui.Alert(ui.AlertProps{Tone: "warn", Title: "Caution"}, g.Text("This action enables OS access for the session.")),
		ui.Alert(ui.AlertProps{Tone: "danger", Title: "Stop"}, g.Text("This will permanently delete the record.")),
	)
}

func tooltipCanvas() g.Node {
	return section("Tooltip", ui.Tooltip(ui.TooltipProps{Label: "Keep it"}, ui.Button(ui.ButtonProps{Variant: "ghost"}, g.Text("hover me"))))
}

func skeletonCanvas() g.Node {
	return section("Skeleton",
		ui.SkeletonLine("100%"), ui.SkeletonLine("60%"),
		ui.Skeleton(ui.SkeletonProps{Variant: "block"}),
		ui.Skeleton(ui.SkeletonProps{Variant: "avatar"}),
	)
}

func textfieldCanvas() g.Node {
	return section("TextField",
		ui.TextField(ui.FieldProps{Label: "Name", Placeholder: "Your name", Name: "name"}),
		ui.TextField(ui.FieldProps{Label: "Email", Type: "email", Value: "you@yourbox", Name: "email", Hint: "Used only on your box."}),
		ui.TextField(ui.FieldProps{Label: "Token", ID: "tok", Name: "token", Error: "Required."}),
	)
}

func selectCanvas() g.Node {
	return section("Select", ui.Select(ui.SelectProps{Label: "Model", Options: []string{"local", "openai", "anthropic"}, Value: "local", Name: "model"}))
}

func toggleCanvas() g.Node {
	return section("Toggle",
		ui.Toggle(ui.ToggleProps{Label: "Notifications", ID: "notif", Checked: true}),
		ui.Toggle(ui.ToggleProps{Label: "OS access", ID: "os"}),
		ui.Toggle(ui.ToggleProps{Label: "Disabled", ID: "dis", Disabled: true}),
	)
}
```

- [ ] **Step 4: Create the registry**

Create `internal/feature/storybook/story.go`:
```go
package storybook

import g "maragu.dev/gomponents"

// Story is one component's storybook entry: its url/anchor ID, sidebar Group and
// Title, and the Canvas that renders its variants. The ordered registry below is
// the single source for both the sidebar nav and the /storybook/{id} routes.
type Story struct {
	ID     string
	Group  string
	Title  string
	Canvas func() g.Node
}

var stories = []Story{
	{"button", "Atoms", "Button", buttonCanvas},
	{"tag", "Atoms", "Tag", tagCanvas},
	{"pips", "Atoms", "Pips", pipsCanvas},
	{"card", "Atoms", "Card", cardCanvas},
	{"stitch", "Atoms", "Stitch", stitchCanvas},
	{"folkband", "Atoms", "FolkBand", folkbandCanvas},
	{"avatar", "Atoms", "Avatar", avatarCanvas},
	{"icon", "Atoms", "Icon", iconCanvas},
	{"badge", "Feedback", "Badge", badgeCanvas},
	{"alert", "Feedback", "Alert", alertCanvas},
	{"tooltip", "Feedback", "Tooltip", tooltipCanvas},
	{"skeleton", "Feedback", "Skeleton", skeletonCanvas},
	{"textfield", "Forms", "TextField", textfieldCanvas},
	{"select", "Forms", "Select", selectCanvas},
	{"toggle", "Forms", "Toggle", toggleCanvas},
}

// Stories returns the ordered registry.
func Stories() []Story { return stories }

// Lookup returns the story with the given ID.
func Lookup(id string) (Story, bool) {
	for _, s := range stories {
		if s.ID == id {
			return s, true
		}
	}
	return Story{}, false
}
```

- [ ] **Step 5: Create the Overview**

Create `internal/feature/storybook/overview.go`:
```go
package storybook

import (
	"strconv"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Overview is the storybook landing canvas: the lede, stat tiles derived from the
// registry, and the component tiers as links into each story.
func Overview() g.Node {
	tiers := map[string][]g.Node{}
	var order []string
	for _, s := range Stories() {
		if _, ok := tiers[s.Group]; !ok {
			order = append(order, s.Group)
		}
		tiers[s.Group] = append(tiers[s.Group],
			h.A(h.Class("tag"), h.Href("/storybook/"+s.ID), g.Text(s.Title)))
	}
	sections := make([]g.Node, 0, len(order))
	for _, grp := range order {
		sections = append(sections, section(grp, tiers[grp]...))
	}

	return h.Div(h.Class("sb"),
		h.P(h.Class("sb-lede"), g.Text("Woven, not rendered.")),
		h.P(g.Text("A typed gomponents component library — every story renders server-side from fixtures, on an empty database.")),
		h.Div(h.Class("sb-stats"),
			stat(strconv.Itoa(len(Stories())), "components"),
			stat("0", "radius"),
			stat("2px", "outlines"),
			stat("8px", "row unit"),
		),
		g.Group(sections),
	)
}

func stat(value, label string) g.Node {
	return h.Div(h.Class("sb-stat"), h.B(g.Text(value)), h.Span(g.Text(label)))
}
```

- [ ] **Step 6: Run to verify it passes (+ existing gallery test still green)**

Run:
```bash
go test ./internal/feature/storybook/ -v 2>&1 | tail -15
go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
```
Expected: the new tests pass, `TestBodyRendersAtoms` still passes (Body composes the canvases), full suite green, build clean.

- [ ] **Step 7: Commit**
```bash
git add internal/feature/storybook/storybook.go internal/feature/storybook/story.go internal/feature/storybook/overview.go internal/feature/storybook/story_test.go
git commit -m "$(printf 'feat(storybook): story registry + per-component canvases + Overview\n\nSplit the flat gallery into per-component Canvas funcs, add the pure-data\nStory registry (single source for nav + routes), and the Overview landing\n(stats derived from the registry). Body() retained until the routed handler.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 5: Routed handler + sidebar assembly

Replace the single `storybookHome` with Overview + per-id handlers, group the
registry into the sidebar, register the routes, and remove the now-unused
`Body()` + its test.

**Files:** Modify `internal/web/storybook.go`, `internal/web/web.go`, `internal/feature/storybook/storybook.go` (remove `Body`), `internal/feature/storybook/storybook_test.go` (remove the superseded test).

- [ ] **Step 1: Replace the handler file**

Replace the ENTIRE contents of `internal/web/storybook.go` with:
```go
package web

import (
	g "maragu.dev/gomponents"
	hh "maragu.dev/gomponents/html" // aliased: the handler receiver is named h

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/feature/storybook"
	"github.com/alexradunet/balaur/internal/ui/shell"
)

// storybookHome serves the storybook Overview at /storybook.
func (h *handlers) storybookHome(e *core.RequestEvent) error {
	return renderStorybook(e, "", "", storybook.Overview())
}

// storybookStory serves one component's page at /storybook/{id}. An unknown id
// falls back to the Overview.
func (h *handlers) storybookStory(e *core.RequestEvent) error {
	id := e.Request.PathValue("id")
	if s, ok := storybook.Lookup(id); ok {
		return renderStorybook(e, s.ID, s.Title, s.Canvas())
	}
	return renderStorybook(e, "", "", storybook.Overview())
}

// renderStorybook composes the sidebar + canvas page for the given active story.
func renderStorybook(e *core.RequestEvent, active, crumb string, body g.Node) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	title := "Storybook"
	if crumb != "" {
		title = crumb
	}
	page := shell.SidebarPage(shell.SidebarPageProps{
		Title:   title,
		Sidebar: shell.Sidebar(sidebarFor(active)),
		Crumb:   crumb,
		Body:    body,
	})
	return page.Render(e.Response)
}

// sidebarFor builds the storybook sidebar from the registry, marking the active
// item. Grouping lives here so the registry stays free of the shell package.
func sidebarFor(active string) shell.SidebarProps {
	sections := []shell.SidebarSection{{
		Label: "Start",
		Items: []shell.SidebarItem{{Label: "Overview", Href: "/storybook", Active: active == ""}},
	}}
	idx := map[string]int{}
	for _, s := range storybook.Stories() {
		i, ok := idx[s.Group]
		if !ok {
			sections = append(sections, shell.SidebarSection{Label: s.Group})
			i = len(sections) - 1
			idx[s.Group] = i
		}
		sections[i].Items = append(sections[i].Items, shell.SidebarItem{
			Label: s.Title, Href: "/storybook/" + s.ID, Active: s.ID == active,
		})
	}
	return shell.SidebarProps{
		Brand: g.Group([]g.Node{
			hh.Img(hh.Class("crest"), hh.Src("/static/crest.png"), hh.Alt(""), g.Attr("decoding", "async")),
			g.Text("Balaur"),
		}),
		Sections: sections,
		Footer: hh.Button(hh.Class("theme-toggle"), hh.Type("button"),
			g.Attr("onclick", "basmToggleTheme()"),
			hh.Title("Toggle light/dark mode"), hh.Aria("label", "Toggle light/dark mode"),
			g.Text("◑")),
	}
}
```

- [ ] **Step 2: Register the routes**

In `internal/web/web.go`, find the line:
```go
	se.Router.GET("/storybook", h.storybookHome)
```
and replace it with:
```go
	se.Router.GET("/storybook", h.storybookHome)
	se.Router.GET("/storybook/{id}", h.storybookStory)
```

- [ ] **Step 3: Remove the now-unused `Body()` + its test**

In `internal/feature/storybook/storybook.go`, delete the `Body` function (the
doc comment + func). Leave `section` and all the `*Canvas` funcs.

In `internal/feature/storybook/storybook_test.go`, delete `TestBodyRendersAtoms`
(superseded by the per-canvas tests in `story_test.go`). If that leaves the file
empty of tests, delete the file: `git rm internal/feature/storybook/storybook_test.go`.

- [ ] **Step 4: Verify build + suite + view**

Run:
```bash
CGO_ENABLED=0 go build ./... && go vet ./... && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
```
Expected: build+vet clean, full suite green. Then smoke the routes:
```bash
go run . serve >/tmp/sb.log 2>&1 &  SRV=$!
curl --retry-connrefused --retry 40 --retry-delay 1 --max-time 90 -sS http://127.0.0.1:8090/storybook | grep -o 'class="sb-side"\|Woven, not rendered.' | sort -u
curl -sS http://127.0.0.1:8090/storybook/button | grep -o 'aria-current="page">Button\|class="btn btn-primary"' | sort -u
kill $SRV
```
Expected: Overview shows `class="sb-side"` + `Woven, not rendered.`; the button page shows the active nav item + the button. (If running the server is impractical, rely on build + tests and note it.)

- [ ] **Step 5: Commit**
```bash
git add internal/web/storybook.go internal/web/web.go internal/feature/storybook/storybook.go internal/feature/storybook/storybook_test.go
git commit -m "$(printf 'feat(web): routed storybook shell (Overview + per-component pages)\n\nGET /storybook -> Overview; GET /storybook/{id} -> one component in the canvas\nwith its sidebar item active. sidebarFor groups the registry into shell.Sidebar.\nRemove the obsolete flat Body() gallery.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification

- [ ] `go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check` — all green.
- [ ] No raw hex in the appended CSS: `tail -45 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"`.
- [ ] Visually confirm (chromium screenshot or browser at `127.0.0.1:8090/storybook`): sidebar with Start/Atoms/Feedback/Forms groups + theme toggle, Overview canvas with stats; clicking a component shows its page with the nav item highlighted.

## What this delivers / what's next

**Delivered:** the routed storybook shell — reusable `shell.Sidebar`, `SidebarPage`, the story registry, Overview, and the 15 atoms as per-component pages.

**Next:** register remaining components as stories as they're ported (List, Toast, Dialog, Tabs, chat/knowledge organisms); add the Brand/Colors/Type/Spacing guideline pages; optional Datastar canvas-swap (patch only `.sb-canvas` on nav click).
