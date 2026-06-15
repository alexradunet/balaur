# Gomponents Atomic — Display Atoms (Plan 02b-iv) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the display atoms `ListItem`, `List`, and `EmptyState` to `internal/ui`, with new tokenized CSS, registering `List` (new "Display" group) and `EmptyState` ("Feedback" group) as storybook stories.

**Architecture:** `ListItem` is a grid row (`[icon] | title+subtitle | meta`); `List` is a parchment container with an optional header that renders `ListItem`s; `EmptyState` is a centered column (crest + title + line + action) that composes the existing `ui.Button`. All pure server-rendered: rows are links when an `Href` is given. New CSS lifts the export's inline styles into tokenized `basm.css` rules (no raw hex).

**Tech Stack:** Go, gomponents, vanilla `basm.css`.

**Scope:** Plan 02b-iv (Phase 2b catalog). Toast, Dialog, SectionLabel/ScreenTitle follow in a later sub-plan.

**Conventions:** package `ui` uses qualified `g`/`h` imports (no dot-import). New CSS appends at the END of `basm.css`, tokenized (`var(--token)`, no raw hex, single-dash). Atom tests are `package ui_test` and use the shared `render(t, node)` helper. Stories: append a canvas func to `internal/feature/storybook/storybook.go` and a `Story` entry to the `stories` slice in `internal/feature/storybook/story.go`. After each task: `go test ./...`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`. If `git status` shows any file other than the task's own as modified (e.g. a stray `chatstream.go`), do NOT stage it — `git checkout --` it.

---

## File Structure

- **Create** `internal/ui/listitem.go`+`_test.go`, `internal/ui/list.go`+`_test.go`, `internal/ui/emptystate.go`+`_test.go`.
- **Modify** `internal/web/assets/static/basm.css` (append List/ListItem + EmptyState CSS).
- **Modify** `internal/feature/storybook/storybook.go` (canvas funcs) + `internal/feature/storybook/story.go` (register List + EmptyState).

---

## Task 1: `ui.ListItem`

A grid row: optional icon, a title (+ optional subtitle), and optional right-aligned
meta. A row with an `Href` renders as a link; otherwise a div. `First` drops the
top border.

**Files:** Create `internal/ui/listitem.go`, `internal/ui/listitem_test.go`; modify `internal/web/assets/static/basm.css`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/listitem_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestListItemLink(t *testing.T) {
	got := render(t, ui.ListItem(ui.ListItemProps{
		Icon: "scroll", Title: "Buy milk", Subtitle: "groceries",
		Meta: "3d", MetaTone: "warn", Href: "/t/1",
	}))
	for _, want := range []string{
		`<a class="list-item list-item-icon" href="/t/1">`,
		`<img class="list-icon" src="/static/icons/scroll.png" alt="" decoding="async">`,
		`<div class="list-main"><div class="list-title">Buy milk</div><div class="list-sub">groceries</div></div>`,
		`<div class="list-meta list-meta-warn">3d</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("list item missing %q in: %s", want, got)
		}
	}
}

func TestListItemPlainFirst(t *testing.T) {
	got := render(t, ui.ListItem(ui.ListItemProps{Title: "Read", First: true}))
	want := `<div class="list-item list-item-first"><div class="list-main"><div class="list-title">Read</div></div></div>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestListItem -v` — Expected: FAIL (`undefined: ui.ListItem`/`ui.ListItemProps`).

- [ ] **Step 3: Implement the atom**

Create `internal/ui/listitem.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ListItemProps configures a ListItem row. Icon is a /static/icons name (empty =
// no icon column). Subtitle and Meta are optional; MetaTone "warn" tints the meta
// ember. First drops the top divider (the first row under no header). Href, when
// set, makes the row a link.
type ListItemProps struct {
	Icon, Title, Subtitle, Meta, MetaTone string
	First                                 bool
	Href                                  string
}

// ListItem renders one row of a List: an optional pixel icon, a title (+ optional
// subtitle), and an optional right-aligned mono meta. A row with an Href is a link.
func ListItem(p ListItemProps) g.Node {
	cls := "list-item"
	if p.Icon != "" {
		cls += " list-item-icon"
	}
	if p.First {
		cls += " list-item-first"
	}

	root := []g.Node{h.Class(cls)}
	if p.Href != "" {
		root = append(root, h.Href(p.Href))
	}
	if p.Icon != "" {
		root = append(root, h.Img(h.Class("list-icon"), h.Src("/static/icons/"+p.Icon+".png"), h.Alt(""), g.Attr("decoding", "async")))
	}
	main := []g.Node{h.Class("list-main"), h.Div(h.Class("list-title"), g.Text(p.Title))}
	if p.Subtitle != "" {
		main = append(main, h.Div(h.Class("list-sub"), g.Text(p.Subtitle)))
	}
	root = append(root, h.Div(main...))
	if p.Meta != "" {
		mcls := "list-meta"
		if p.MetaTone == "warn" {
			mcls += " list-meta-warn"
		}
		root = append(root, h.Div(h.Class(mcls), g.Text(p.Meta)))
	}

	if p.Href != "" {
		return h.A(root...)
	}
	return h.Div(root...)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestListItem -v` — Expected: PASS (both).

- [ ] **Step 5: Append the ListItem CSS to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`:
```css

/* ── ListItem — a row: [icon] | title+subtitle | meta ───────────────────── */
.list-item {
  display: grid; grid-template-columns: 1fr auto; align-items: center; column-gap: 12px;
  padding: 11px 14px; text-decoration: none; border-top: 1px solid var(--parch-edge);
}
.list-item-icon { grid-template-columns: 28px 1fr auto; }
.list-item-first { border-top: none; }
.list-icon { width: 22px; height: 22px; image-rendering: pixelated; }
.list-main { min-width: 0; }
.list-title { font-size: 14.5px; color: var(--ink); line-height: 1.3; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.list-sub { font-family: var(--font-mono); font-size: 11px; color: var(--ink-muted); margin-top: 2px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.list-meta { font-family: var(--font-mono); font-size: 11.5px; color: var(--ink-muted); text-transform: uppercase; letter-spacing: .03em; white-space: nowrap; text-align: right; }
.list-meta-warn { color: var(--ember-deep); }
```

- [ ] **Step 6: Verify + commit**

Run:
```bash
go test ./internal/ui/ -run TestListItem && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
tail -16 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
```
Expected: PASS, full suite green, build clean, NO RAW HEX. Then:
```bash
git add internal/ui/listitem.go internal/ui/listitem_test.go internal/web/assets/static/basm.css
git commit -m "$(printf 'feat(ui): add ListItem atom\n\nui.ListItem(ListItemProps) — a grid row (icon | title+subtitle | meta); link\nwhen Href set, warn meta tone, first-row divider drop. New tokenized .list-item\nCSS.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: `ui.List` + story

A parchment container with an optional mono header that renders `ListItem`s.
Under a header, no row is "first" (all keep the divider); with no header, row 0
drops its divider.

**Files:** Create `internal/ui/list.go`, `internal/ui/list_test.go`; modify `internal/web/assets/static/basm.css`, `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/list_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestList(t *testing.T) {
	got := render(t, ui.List(ui.ListProps{
		Title: "Today",
		Items: []ui.ListItemProps{{Title: "a"}, {Title: "b"}},
	}))
	for _, want := range []string{
		`<div class="list"><div class="list-head">Today</div>`,
		`<div class="list-title">a</div>`,
		`<div class="list-title">b</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("list missing %q in: %s", want, got)
		}
	}
	// with a header, no item is "first" (header already separates the top row)
	if strings.Contains(got, "list-item-first") {
		t.Errorf("titled list should have no first-row divider drop: %s", got)
	}
}

func TestListNoTitleFirst(t *testing.T) {
	got := render(t, ui.List(ui.ListProps{Items: []ui.ListItemProps{{Title: "a"}, {Title: "b"}}}))
	if !strings.Contains(got, `<div class="list-item list-item-first">`) {
		t.Errorf("untitled list: row 0 should be first: %s", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestList -v` — Expected: FAIL (`undefined: ui.List`/`ui.ListProps`).

- [ ] **Step 3: Implement the atom**

Create `internal/ui/list.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ListProps configures a List. Title (optional) is the mono header; Items are the
// rows.
type ListProps struct {
	Title string
	Items []ListItemProps
}

// List renders the parchment list card: an optional uppercase mono header over a
// stack of ListItem rows. With no header, the first row drops its top divider.
func List(p ListProps) g.Node {
	kids := []g.Node{h.Class("list")}
	if p.Title != "" {
		kids = append(kids, h.Div(h.Class("list-head"), g.Text(p.Title)))
	}
	for i, it := range p.Items {
		it.First = i == 0 && p.Title == ""
		kids = append(kids, ListItem(it))
	}
	return h.Div(kids...)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestList -v` — Expected: PASS.

- [ ] **Step 5: Append the List CSS to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`:
```css

/* ── List — parchment card of ListItem rows ─────────────────────────────── */
.list {
  background: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px;
  border: 2px solid var(--parch-edge); box-shadow: var(--parch-bevel); overflow: hidden;
}
.list-head {
  font-family: var(--font-mono); font-size: 10px; font-weight: 700; letter-spacing: .09em;
  text-transform: uppercase; color: var(--gold-ink); padding: 11px 14px 9px; border-bottom: 2px solid var(--parch-edge);
}
```

- [ ] **Step 6: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append:
```go

func listCanvas() g.Node {
	return section("List", ui.List(ui.ListProps{
		Title: "Today",
		Items: []ui.ListItemProps{
			{Icon: "scroll", Title: "Buy milk", Subtitle: "groceries", Meta: "2pm"},
			{Icon: "flame", Title: "Workout", Meta: "due", MetaTone: "warn"},
			{Title: "Read chapter 4", Subtitle: "before bed"},
		},
	}))
}
```
In `internal/feature/storybook/story.go`, append to the `stories` slice (after the `{"pagination", ...}` line):
```go
	{"list", "Display", "List", listCanvas},
```

- [ ] **Step 7: Verify + commit**

Run:
```bash
go test ./internal/ui/ -run TestList && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
tail -10 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
```
Expected: PASS, full suite green, build clean, NO RAW HEX. Then:
```bash
git add internal/ui/list.go internal/ui/list_test.go internal/web/assets/static/basm.css internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add List atom + storybook story\n\nui.List(ListProps) — parchment card of ListItem rows with an optional mono\nheader. New tokenized .list CSS. Registered under a new Display group.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 3: `ui.EmptyState` + story

A centered column: optional crest, a display title (defaults to "Nothing on the
book."), an optional line, and an optional action — the action composes the
existing `ui.Button` (wood variant) as a link.

**Files:** Create `internal/ui/emptystate.go`, `internal/ui/emptystate_test.go`; modify `internal/web/assets/static/basm.css`, `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/emptystate_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestEmptyStateFull(t *testing.T) {
	got := render(t, ui.EmptyState(ui.EmptyProps{
		CrestSrc:    "/static/crest.png",
		Title:       "Nothing on the book.",
		Line:        "Tell Balaur in chat what to keep for you.",
		ActionLabel: "Start a thread",
		ActionHref:  "/",
	}))
	for _, want := range []string{
		`<div class="empty">`,
		`<img class="empty-crest" src="/static/crest.png" alt="" decoding="async">`,
		`<h3 class="empty-title">Nothing on the book.</h3>`,
		`<p class="empty-line">Tell Balaur in chat what to keep for you.</p>`,
		`<div class="empty-action"><a class="btn btn-wood" href="/">Start a thread</a></div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("empty state missing %q in: %s", want, got)
		}
	}
}

func TestEmptyStateDefaultTitle(t *testing.T) {
	got := render(t, ui.EmptyState(ui.EmptyProps{}))
	if !strings.Contains(got, `<h3 class="empty-title">Nothing on the book.</h3>`) {
		t.Errorf("default title missing: %s", got)
	}
	if strings.Contains(got, "empty-crest") || strings.Contains(got, "empty-line") || strings.Contains(got, "empty-action") {
		t.Errorf("bare empty state should omit crest/line/action: %s", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestEmptyState -v` — Expected: FAIL (`undefined: ui.EmptyState`/`ui.EmptyProps`).

- [ ] **Step 3: Implement the atom**

Create `internal/ui/emptystate.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// EmptyProps configures an EmptyState. All fields optional: CrestSrc shows a crest;
// Title defaults to "Nothing on the book."; Line is a supporting sentence;
// ActionLabel + ActionHref render a wood Button link.
type EmptyProps struct {
	CrestSrc, Title, Line, ActionLabel, ActionHref string
}

// EmptyState renders the centered empty placeholder: optional crest, a display
// title, an optional line, and an optional action button.
func EmptyState(p EmptyProps) g.Node {
	title := p.Title
	if title == "" {
		title = "Nothing on the book."
	}
	kids := []g.Node{h.Class("empty")}
	if p.CrestSrc != "" {
		kids = append(kids, h.Img(h.Class("empty-crest"), h.Src(p.CrestSrc), h.Alt(""), g.Attr("decoding", "async")))
	}
	kids = append(kids, h.H3(h.Class("empty-title"), g.Text(title)))
	if p.Line != "" {
		kids = append(kids, h.P(h.Class("empty-line"), g.Text(p.Line)))
	}
	if p.ActionLabel != "" {
		kids = append(kids, h.Div(h.Class("empty-action"),
			Button(ButtonProps{Variant: "wood", Href: p.ActionHref}, g.Text(p.ActionLabel))))
	}
	return h.Div(kids...)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestEmptyState -v` — Expected: PASS (both).

- [ ] **Step 5: Append the EmptyState CSS to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`:
```css

/* ── EmptyState — centered crest + title + line + action ────────────────── */
.empty { display: flex; flex-direction: column; align-items: center; text-align: center; gap: 14px; padding: 30px 20px; }
.empty-crest { width: 88px; height: 88px; image-rendering: pixelated; opacity: .92; }
.empty-title { margin: 0; font-family: var(--font-display); font-size: 22px; color: var(--fg-strong); }
.empty-line { margin: 0; max-width: 360px; color: var(--muted); font-size: 14px; line-height: 1.55; }
.empty-action { margin-top: 4px; }
```

- [ ] **Step 6: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append:
```go

func emptyStateCanvas() g.Node {
	return section("EmptyState", ui.EmptyState(ui.EmptyProps{
		CrestSrc:    "/static/crest.png",
		Line:        "Tell Balaur in chat what to keep for you.",
		ActionLabel: "Start a thread",
		ActionHref:  "#",
	}))
}
```
In `internal/feature/storybook/story.go`, append to the `stories` slice (after the `{"list", ...}` line):
```go
	{"emptystate", "Feedback", "EmptyState", emptyStateCanvas},
```

- [ ] **Step 7: Verify + commit**

Run:
```bash
go test ./internal/ui/ -run TestEmptyState && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./... && go vet ./...
tail -10 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
```
Expected: PASS, full suite green, build+vet clean, NO RAW HEX. Then:
```bash
git add internal/ui/emptystate.go internal/ui/emptystate_test.go internal/web/assets/static/basm.css internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add EmptyState atom + storybook story\n\nui.EmptyState(EmptyProps) — centered crest + display title (default \"Nothing on\nthe book.\") + line + a wood Button action. New tokenized .empty CSS. Registered\nunder Feedback.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification

- [ ] `go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check` — all green.
- [ ] No raw hex in the new CSS: `tail -34 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"`.
- [ ] Sidebar has a new **Display** group (List) and **EmptyState** under Feedback; `/storybook/list` and `/storybook/emptystate` render.

## What this delivers / what's next

**Delivered:** three display atoms (`ListItem`, `List`, `EmptyState`); `List` + `EmptyState` registered as stories.

**Next:** `Toast`, `Dialog` (native `<dialog>`), and the `SectionLabel`/`ScreenTitle` helpers — then the organisms (chat, knowledge/task cards) that start replacing real `html/template` surfaces.
