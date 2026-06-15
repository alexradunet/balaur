# Gomponents Atomic — Nav Atoms (Plan 02b-iii) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the three navigation atoms — `Tabs`, `Breadcrumb`, `Pagination` — to `internal/ui`, each registered as a storybook Story so it appears in the sidebar under a new "Navigation" group.

**Architecture:** `Tabs` is class-only (reuses the existing `.k-tabs`/`.k-tab`/`.k-tab-active` rules). `Breadcrumb` and `Pagination` pair a typed gomponents atom with new tokenized `basm.css` rules. All three are pure server-rendered controls: items are real links (the caller supplies hrefs / `HrefFor`); active state is `aria-current="page"` + a class. Each task also adds a `*Canvas()` func + a registry entry (Group "Navigation") in `internal/feature/storybook`, so the routed storybook auto-wires its sidebar entry + `/storybook/{id}` route.

**Tech Stack:** Go 1.26 (so `min`/`max` builtins are available), gomponents, vanilla `basm.css`.

**Scope:** Plan 02b-iii (Phase 2b). The remaining catalog (List/ListItem, Toast, Dialog, EmptyState, SectionLabel/ScreenTitle) follows in a later sub-plan.

**Conventions:** package `ui` uses qualified `g`/`h` imports (no dot-import). New CSS appends at the END of `basm.css`, tokenized (`var(--token)`, no raw hex, single-dash). Atom tests are `package ui_test` and use the shared `render(t, node)` helper. Registry entries: append to the `stories` slice in `internal/feature/storybook/story.go`; canvas funcs go in `internal/feature/storybook/storybook.go`. After each task: `go test ./...`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`.

---

## File Structure

- **Create** `internal/ui/tabs.go`+`_test.go`, `internal/ui/breadcrumb.go`+`_test.go`, `internal/ui/pagination.go`+`_test.go`.
- **Modify** `internal/web/assets/static/basm.css` (append Breadcrumb + Pagination CSS).
- **Modify** `internal/feature/storybook/storybook.go` (add canvas funcs) and `internal/feature/storybook/story.go` (register stories).

---

## Task 1: `ui.Tabs` (class-only) + story

`Tabs` renders `<nav class="k-tabs">` of `<a class="k-tab">` links (active gets
`k-tab-active` + `aria-current="page"`). It reuses the existing `.k-tab` rules —
NO new CSS.

**Files:** Create `internal/ui/tabs.go`, `internal/ui/tabs_test.go`; modify `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/tabs_test.go`:
```go
package ui_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestTabs(t *testing.T) {
	got := render(t, ui.Tabs([]ui.TabItem{
		{Label: "Today", Href: "/t?f=today", Active: true},
		{Label: "Upcoming", Href: "/t?f=up"},
	}))
	want := `<nav class="k-tabs">` +
		`<a class="k-tab k-tab-active" href="/t?f=today" aria-current="page">Today</a>` +
		`<a class="k-tab" href="/t?f=up">Upcoming</a></nav>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestTabs -v` — Expected: FAIL (`undefined: ui.Tabs`/`ui.TabItem`).

- [ ] **Step 3: Implement the atom**

Create `internal/ui/tabs.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// TabItem is one tab: a label, the href it navigates to, and whether it's active.
type TabItem struct {
	Label, Href string
	Active      bool
}

// Tabs renders the Hearthwood tab strip: a <nav class="k-tabs"> of link tabs.
// The active tab carries k-tab-active + aria-current="page". Pure render — tabs
// are real links (filter routes / Datastar targets supplied by the caller);
// switching is wired above this atom.
func Tabs(items []TabItem) g.Node {
	kids := []g.Node{h.Class("k-tabs")}
	for _, it := range items {
		cls := "k-tab"
		if it.Active {
			cls += " k-tab-active"
		}
		attrs := []g.Node{h.Class(cls), h.Href(it.Href)}
		if it.Active {
			attrs = append(attrs, h.Aria("current", "page"))
		}
		attrs = append(attrs, g.Text(it.Label))
		kids = append(kids, h.A(attrs...))
	}
	return h.Nav(kids...)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestTabs -v` — Expected: PASS.

- [ ] **Step 5: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append a canvas func (after the last one):
```go

func tabsCanvas() g.Node {
	return section("Tabs", ui.Tabs([]ui.TabItem{
		{Label: "Overdue", Href: "#"},
		{Label: "Today", Href: "#", Active: true},
		{Label: "Upcoming", Href: "#"},
		{Label: "Someday", Href: "#"},
	}))
}
```

In `internal/feature/storybook/story.go`, append to the `stories` slice (after the `{"toggle", ...}` line, inside the literal):
```go
	{"tabs", "Navigation", "Tabs", tabsCanvas},
```

- [ ] **Step 6: Verify (atom test + story renders + suite)**

Run:
```bash
go test ./internal/ui/ -run TestTabs && go test ./internal/feature/storybook/ && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
```
Expected: PASS, full suite green, build clean. (The registry test `TestStoriesUniqueAndLookup` now sees 16 stories incl. `tabs`.)

- [ ] **Step 7: Commit**
```bash
git add internal/ui/tabs.go internal/ui/tabs_test.go internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add Tabs atom + storybook story\n\nui.Tabs([]TabItem) — <nav class=k-tabs> of link tabs (active = k-tab-active +\naria-current). Class-only (reuses existing .k-tab rules). Registered under a\nnew Navigation group.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: `ui.Breadcrumb` + story

A wood breadcrumb bar: link crumbs + `›` separators, the last (or href-less)
crumb is the current page (gold, not a link). New CSS.

**Files:** Create `internal/ui/breadcrumb.go`, `internal/ui/breadcrumb_test.go`; modify `internal/web/assets/static/basm.css`, `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/breadcrumb_test.go`:
```go
package ui_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestBreadcrumb(t *testing.T) {
	got := render(t, ui.Breadcrumb([]ui.Crumb{
		{Label: "Home", Href: "/"},
		{Label: "Tasks", Href: "/tasks"},
		{Label: "Today"},
	}))
	want := `<nav class="breadcrumb" aria-label="Breadcrumb">` +
		`<a class="crumb-link" href="/">Home</a>` +
		`<span class="crumb-sep" aria-hidden="true">›</span>` +
		`<a class="crumb-link" href="/tasks">Tasks</a>` +
		`<span class="crumb-sep" aria-hidden="true">›</span>` +
		`<span class="crumb-cur">Today</span></nav>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestBreadcrumb -v` — Expected: FAIL (`undefined: ui.Breadcrumb`/`ui.Crumb`).

- [ ] **Step 3: Implement the atom**

Create `internal/ui/breadcrumb.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Crumb is one breadcrumb entry. An empty Href (or the last item) renders as the
// current page (a non-link span) rather than a link.
type Crumb struct {
	Label, Href string
}

// Breadcrumb renders the Hearthwood breadcrumb bar: link crumbs separated by a
// muted › glyph, ending in the current page. The trail is a <nav aria-label>.
func Breadcrumb(items []Crumb) g.Node {
	kids := []g.Node{h.Class("breadcrumb"), h.Aria("label", "Breadcrumb")}
	for i, it := range items {
		last := i == len(items)-1
		if last || it.Href == "" {
			kids = append(kids, h.Span(h.Class("crumb-cur"), g.Text(it.Label)))
		} else {
			kids = append(kids, h.A(h.Class("crumb-link"), h.Href(it.Href), g.Text(it.Label)))
		}
		if !last {
			kids = append(kids, h.Span(h.Class("crumb-sep"), g.Attr("aria-hidden", "true"), g.Text("›")))
		}
	}
	return h.Nav(kids...)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestBreadcrumb -v` — Expected: PASS.

- [ ] **Step 5: Append the Breadcrumb CSS to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`:
```css

/* ── Breadcrumb — wood trail of link crumbs + › separators ──────────────── */
.breadcrumb {
  display: inline-flex; align-items: center; gap: 9px; flex-wrap: wrap;
  background: var(--chrome); background-image: var(--wood-planks), var(--grain-warm); background-size: auto, 4px 4px;
  border: 2px solid var(--outline-2); box-shadow: var(--bevel-up); padding: 7px 12px;
}
.breadcrumb a, .breadcrumb span {
  font-family: var(--font-mono); font-size: 11.5px; letter-spacing: .03em; text-transform: uppercase;
}
.crumb-link { color: var(--chrome-fg); text-decoration: none; }
.crumb-link:hover { color: var(--gold); }
.crumb-cur { color: var(--gold); }
.crumb-sep { color: var(--smoke); }
```

- [ ] **Step 6: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append:
```go

func breadcrumbCanvas() g.Node {
	return section("Breadcrumb", ui.Breadcrumb([]ui.Crumb{
		{Label: "Home", Href: "/"},
		{Label: "Tasks", Href: "/tasks"},
		{Label: "Today"},
	}))
}
```
In `internal/feature/storybook/story.go`, append to the `stories` slice (after `{"tabs", ...}`):
```go
	{"breadcrumb", "Navigation", "Breadcrumb", breadcrumbCanvas},
```

- [ ] **Step 7: Verify + commit**

Run:
```bash
go test ./internal/ui/ -run TestBreadcrumb && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
tail -16 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
```
Expected: PASS, full suite green, build clean, NO RAW HEX. Then:
```bash
git add internal/ui/breadcrumb.go internal/ui/breadcrumb_test.go internal/web/assets/static/basm.css internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add Breadcrumb atom + storybook story\n\nui.Breadcrumb([]Crumb) — wood trail of link crumbs + muted › separators, last\ncrumb is the current page. New tokenized .breadcrumb CSS. Registered under\nNavigation.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 3: `ui.Pagination` + story

Prev / windowed numbered slabs / next, with ellipses; the active page is a
raised gold chip, the rest inset wells; prev/next disable at the bounds. Slabs
are links (`HrefFor(page)`); disabled prev/next render as non-link spans. New CSS.

**Files:** Create `internal/ui/pagination.go`, `internal/ui/pagination_test.go`; modify `internal/web/assets/static/basm.css`, `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/pagination_test.go`:
```go
package ui_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestPagination(t *testing.T) {
	got := render(t, ui.Pagination(ui.PagerProps{
		Total: 5, Page: 3,
		HrefFor: func(n int) string { return "/p/" + strconv.Itoa(n) },
	}))
	for _, want := range []string{
		`<nav class="pager" aria-label="Pagination">`,
		`<a class="pager-slab pager-active" href="/p/3" aria-current="page">3</a>`,
		`<a class="pager-slab" href="/p/2">2</a>`,
		`<a class="pager-slab" href="/p/4">4</a>`,
		`<span class="pager-gap" aria-hidden="true">…</span>`, // window 2..4 of 5 → gaps both sides
	} {
		if !strings.Contains(got, want) {
			t.Errorf("pager missing %q in: %s", want, got)
		}
	}
}

func TestPaginationBounds(t *testing.T) {
	// page 1 of 3 → prev disabled (non-link span), next is a link
	got := render(t, ui.Pagination(ui.PagerProps{Total: 3, Page: 1, HrefFor: func(n int) string { return "#" }}))
	if !strings.Contains(got, `<span class="pager-slab pager-disabled" aria-disabled="true">‹</span>`) {
		t.Errorf("prev should be disabled at page 1: %s", got)
	}
	if !strings.Contains(got, `>›</a>`) {
		t.Errorf("next should be a link at page 1: %s", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestPagination -v` — Expected: FAIL (`undefined: ui.Pagination`/`ui.PagerProps`).

- [ ] **Step 3: Implement the atom**

Create `internal/ui/pagination.go`:
```go
package ui

import (
	"strconv"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// PagerProps configures a Pagination. Total/Page are 1-based; HrefFor maps a page
// number to its URL.
type PagerProps struct {
	Total   int
	Page    int
	HrefFor func(int) string
}

// Pagination renders prev / a window of numbered slabs / next, with ellipses
// when the window is clipped. The active page is a raised gold chip; prev/next
// are disabled (non-link spans) at the bounds. Pure render — navigation is via
// the slab links.
func Pagination(p PagerProps) g.Node {
	if p.Total < 1 {
		p.Total = 1
	}
	if p.Page < 1 {
		p.Page = 1
	}
	start := max(1, min(p.Page-1, p.Total-2))
	end := min(p.Total, start+2)

	kids := []g.Node{h.Class("pager"), h.Aria("label", "Pagination")}
	kids = append(kids, pagerSlab(p, "‹", p.Page-1, p.Page <= 1, false))
	if start > 1 {
		kids = append(kids, pagerGap())
	}
	for n := start; n <= end; n++ {
		kids = append(kids, pagerSlab(p, strconv.Itoa(n), n, false, n == p.Page))
	}
	if end < p.Total {
		kids = append(kids, pagerGap())
	}
	kids = append(kids, pagerSlab(p, "›", p.Page+1, p.Page >= p.Total, false))
	return h.Nav(kids...)
}

// pagerSlab renders one slab: a link, or a non-link span when disabled.
func pagerSlab(p PagerProps, label string, page int, disabled, active bool) g.Node {
	cls := "pager-slab"
	if active {
		cls += " pager-active"
	}
	if disabled {
		return h.Span(h.Class(cls+" pager-disabled"), g.Attr("aria-disabled", "true"), g.Text(label))
	}
	attrs := []g.Node{h.Class(cls), h.Href(p.HrefFor(page))}
	if active {
		attrs = append(attrs, h.Aria("current", "page"))
	}
	attrs = append(attrs, g.Text(label))
	return h.A(attrs...)
}

func pagerGap() g.Node {
	return h.Span(h.Class("pager-gap"), g.Attr("aria-hidden", "true"), g.Text("…"))
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestPagination -v` — Expected: PASS (both subtests).

- [ ] **Step 5: Append the Pagination CSS to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`:
```css

/* ── Pagination — prev / numbered slabs / next ──────────────────────────
   Active page = raised gold chip; the rest are inset wells. */
.pager { display: inline-flex; align-items: center; gap: 6px; }
.pager-slab {
  display: inline-flex; align-items: center; justify-content: center;
  min-width: 34px; height: 34px; padding: 0 9px;
  font-family: var(--font-mono); font-size: 12px; text-decoration: none;
  color: var(--chrome-fg); background: var(--chrome-2);
  background-image: var(--grain-warm); background-size: 4px 4px;
  border: 2px solid var(--outline-2); border-radius: var(--radius); box-shadow: var(--bevel-in);
}
.pager-slab:hover { filter: brightness(1.15); }
.pager-active {
  color: var(--gold); background: var(--chrome); border-color: var(--gold-deep);
  box-shadow: inset 0 2px 0 var(--bevel-light), inset 0 -2px 0 var(--bevel-dark), var(--drop-hard);
}
.pager-disabled { opacity: .4; }
.pager-gap { align-self: end; padding: 0 2px; color: var(--smoke); font-family: var(--font-mono); }
```

- [ ] **Step 6: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append:
```go

func paginationCanvas() g.Node {
	return section("Pagination", ui.Pagination(ui.PagerProps{
		Total: 8, Page: 3, HrefFor: func(n int) string { return "#" },
	}))
}
```
In `internal/feature/storybook/story.go`, append to the `stories` slice (after `{"breadcrumb", ...}`):
```go
	{"pagination", "Navigation", "Pagination", paginationCanvas},
```

- [ ] **Step 7: Verify + commit**

Run:
```bash
go test ./internal/ui/ -run TestPagination && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./... && go vet ./...
tail -20 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
```
Expected: PASS, full suite green, build+vet clean, NO RAW HEX. Then:
```bash
git add internal/ui/pagination.go internal/ui/pagination_test.go internal/web/assets/static/basm.css internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add Pagination atom + storybook story\n\nui.Pagination(PagerProps) — prev / windowed numbered slabs / next, ellipses,\nactive gold chip, disabled bounds. New tokenized .pager CSS. Registered under\nNavigation.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification

- [ ] `go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check` — all green.
- [ ] No raw hex in the new CSS: `tail -36 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"`.
- [ ] Sidebar now has a **Navigation** group (Tabs, Breadcrumb, Pagination); `/storybook/tabs`, `/storybook/breadcrumb`, `/storybook/pagination` each render their component.

## What this delivers / what's next

**Delivered:** three nav atoms (`Tabs` class-only, `Breadcrumb`, `Pagination`), registered as stories in a new Navigation sidebar group.

**Next:** the remaining catalog — `List`/`ListItem`, `Toast`, `Dialog` (native `<dialog>`), `EmptyState`, and the `SectionLabel`/`ScreenTitle` helpers — then the organisms (chat, knowledge/task cards) that begin replacing real `html/template` surfaces.
