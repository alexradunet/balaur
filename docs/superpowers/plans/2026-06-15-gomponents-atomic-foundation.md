# Gomponents Atomic Storybook — Foundation Slice (Plan 01) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Land the foundation of the gomponents atomic migration — clean the Hearthwood CSS, relocate static assets into `internal/web/assets`, establish the typed `ui` atom layer with `ui.Button`, port the page shell to `internal/ui/shell`, and serve a live `/storybook` page rendering a real Button in the real shell.

**Architecture:** Extend the existing no-web-import `internal/ui` leaf with element-named atom constructors (requires dropping the `gomponents/html` dot-import in package `ui` — empirically required, see Task 3). Add `internal/ui/shell` (the page shell, ports `layout.html`) and `internal/feature/storybook` (the gallery body). A plain `internal/web` handler composes `shell.Page` at `/storybook` — the storybook is **not** a registered card. Static assets move to `internal/web/assets` with their own `embed.FS`; URLs stay `/static/...`.

**Tech Stack:** Go, gomponents (`maragu.dev/gomponents` + `.../html`), PocketBase router, `//go:embed`, vanilla CSS (`basm.css`, no build step).

**Scope:** This is Plan 01 of the 6-phase migration in `docs/superpowers/specs/2026-06-15-gomponents-atomic-storybook-design.md`. It covers spec **Phase 0** (CSS fixes + asset relocation) and **Phase 1** (shell + first atom, live). Phases 2–6 get their own plans.

**Deviations from the spec (deliberate):**
1. **Per-atom CSS rules are authored with each atom (Phase 2+), not front-loaded here.** Phase 0 CSS work is limited to bug-fixes; adding rules for atoms not yet built would be write-only output (AGENTS.md YAGNI). `ui.Button` is class-only against the existing `.btn` rules, so it needs no new CSS.
2. **Package `ui` adopts a qualified html import** (`h "maragu.dev/gomponents/html"`, never dot-import). Verified necessary: a package-level `func Button` collides with the dot-imported `html.Button` (`Button already declared through dot-import`). A qualified import in the atom file does **not** fix it — the collision is package-scope, so no file in package `ui` may dot-import html.
3. **The storybook is served at `/storybook` here** (boards stays at `/` until its Phase-3 cutover), per the spec's de-risk note.
4. **`shell.Canvas` is deferred** to the phase that first needs a width-constrained body (YAGNI now).

---

## File Structure

**Created:**
- `internal/web/assets/embed.go` — `//go:embed static`; the static-asset FS.
- `internal/web/assets/embed_test.go` — asset-presence test (moved from `web/`).
- `internal/web/assets/css_tokens_test.go` — guards the basm.css token fixes.
- `internal/ui/button.go` — the `ui.Button` atom.
- `internal/ui/button_test.go` — Button render tests.
- `internal/ui/shell/shell.go` — `shell.Page`, `Topbar`, `PageProps` (ports `layout.html`).
- `internal/ui/shell/shell_test.go` — shell render tests.
- `internal/feature/storybook/storybook.go` — `storybook.Body()` gallery node.
- `internal/feature/storybook/storybook_test.go` — Body render test (empty-DB).
- `internal/web/storybook.go` — the `storybookHome` handler.

**Modified:**
- `web/static/` → moved to `internal/web/assets/static/` (git mv).
- `web/embed.go` — embed `templates` only (static moved).
- `internal/web/web.go` — static FS now from `assets`, register `GET /storybook`.
- `internal/ui/components.go` — convert dot-import → qualified `h.`.
- `internal/web/assets/static/basm.css` — define `--indigo-deep`; replace stale Forest-at-Dusk tokens.

**Deleted:**
- `web/embed_assets_test.go` — moved into `internal/web/assets/`.

---

## Task 1: Relocate static assets to `internal/web/assets`

Moves `web/static/` beside the rendering code with its own embed package. No
behavior change — URLs stay `/static/...`. Intermediate steps will not compile;
build + commit only at the end.

**Files:**
- Create: `internal/web/assets/embed.go`
- Move: `web/static/` → `internal/web/assets/static/`
- Modify: `web/embed.go`
- Modify: `internal/web/web.go:161-166` (the `staticFS` setup) and its imports
- Move: `web/embed_assets_test.go` → `internal/web/assets/embed_test.go`

- [ ] **Step 1: Move the static directory**

Run:
```bash
git mv web/static internal/web/assets/static
```

- [ ] **Step 2: Create the assets embed package**

Create `internal/web/assets/embed.go`:
```go
// Package assets embeds Balaur's static web assets — the Hearthwood basm.css,
// self-hosted fonts, pixel icons, avatars, crest, and logo — so the single
// binary serves them. Served at /static/... via apis.Static in internal/web.
package assets

import "embed"

//go:embed static
var FS embed.FS
```

- [ ] **Step 3: Shrink the top-level web embed to templates only**

Replace the entire contents of `web/embed.go` with:
```go
// Package web holds the embedded HTML templates — the legacy html/template
// surface being migrated to gomponents. Static assets moved to
// internal/web/assets. This package and its templates are removed once the
// migration is complete.
package web

import "embed"

//go:embed templates
var FS embed.FS
```

- [ ] **Step 4: Point web.go's static FS at the assets package**

In `internal/web/web.go`, add the import (next to the existing
`webassets "github.com/alexradunet/balaur/web"`):
```go
	webstatic "github.com/alexradunet/balaur/internal/web/assets"
```

Then change the `staticFS` source (currently `internal/web/web.go:163`):
```go
	staticFS, err := fs.Sub(webassets.FS, "static")
```
to:
```go
	staticFS, err := fs.Sub(webstatic.FS, "static")
```
Leave the template parse (`template.Must(... ParseFS(webassets.FS, "templates/*.html"))`) unchanged — templates still live in the top-level `web` package.

- [ ] **Step 5: Move and repackage the embed-presence test**

Run:
```bash
git mv web/embed_assets_test.go internal/web/assets/embed_test.go
```
Then replace the contents of `internal/web/assets/embed_test.go` with (package
rename + assert the stable Hearthwood assets; `board.js` is intentionally not
pinned — it is removed when boards is cut):
```go
package assets

import "testing"

// TestEmbedAssetPresence verifies the Hearthwood static assets are carried in
// the embedded FS so the single-binary build serves them.
func TestEmbedAssetPresence(t *testing.T) {
	paths := []string{
		"static/basm.css",
		"static/icons/scroll.png",
		"static/icons/tome.png",
		"static/fonts/piazzolla.ttf",
		"static/fonts/jersey-15.ttf",
	}
	for _, p := range paths {
		f, err := FS.Open(p)
		if err != nil {
			t.Errorf("asset missing from embed FS: %s: %v", p, err)
			continue
		}
		f.Close()
	}
}
```

- [ ] **Step 6: Build, test, and verify the app still serves assets**

Run:
```bash
CGO_ENABLED=0 go build ./... && go test ./internal/web/... && go vet ./...
```
Expected: build + tests PASS.

Then run the app and confirm the stylesheet still serves:
```bash
go run . &  sleep 3
curl -sS -o /dev/null -w "%{http_code} %{content_type}\n" http://127.0.0.1:8090/static/basm.css
kill %1
```
Expected: `200 text/css; charset=utf-8` (the page renders unchanged in a browser).

- [ ] **Step 7: Commit**

```bash
git add internal/web/assets web/embed.go internal/web/web.go
git commit -m "$(printf 'refactor(web): relocate static assets to internal/web/assets\n\nMove web/static into its own embed package beside the rendering code.\nURLs stay /static/...; the top-level web package now embeds only templates.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: Fix the basm.css token bugs (guarded by a test)

`--indigo-deep` is referenced (owner-portrait keyline, `basm.css` ~515/695) but
never defined, and four stale "Forest at Dusk" tokens (`--line`, `--accent`,
`--border`, `--parchment`) survive — `--border`/`--parchment` have no fallback
and render wrong. Fix them, guarded by a Go test.

**Files:**
- Create: `internal/web/assets/css_tokens_test.go`
- Modify: `internal/web/assets/static/basm.css`

- [ ] **Step 1: (One-time) confirm basm.css matches the export tokens**

This plan assumes `basm.css` is already the export's Hearthwood. Spot-check the
token block against the export before editing:
```bash
diff <(grep -oE -- '--[a-z0-9-]+: *light-dark\([^;]*\)' internal/web/assets/static/basm.css | sort) \
     <(grep -oE -- '--[a-z0-9-]+: *light-dark\([^;]*\)' "Balaur_Design/_ds/balaur-basm-design-system-0c1b20fd-0bf4-4b2c-bbd1-bfa417af0a6b/tokens/colors.css" | sort)
```
Expected: the only difference is `--indigo-deep` (which we are about to add) and
any whitespace. If a substantive re-skin surfaces here, stop and reconcile
toward the export — that is added scope for this task, per the spec's Phase-0
note.

- [ ] **Step 2: Write the failing test**

Create `internal/web/assets/css_tokens_test.go`:
```go
package assets

import (
	"strings"
	"testing"
)

// TestNoUndefinedHearthwoodTokens guards the Phase-0 CSS fixes: --indigo-deep
// must be defined (the owner-portrait keyline references it with no fallback),
// and no stale Forest-at-Dusk token (--line/--accent/--border/--parchment) may
// remain referenced.
func TestNoUndefinedHearthwoodTokens(t *testing.T) {
	b, err := FS.ReadFile("static/basm.css")
	if err != nil {
		t.Fatalf("read basm.css: %v", err)
	}
	css := string(b)

	if !strings.Contains(css, "--indigo-deep:") {
		t.Error("--indigo-deep is referenced (owner-portrait keyline) but never defined")
	}
	for _, stale := range []string{"var(--border)", "var(--parchment)", "var(--line", "var(--accent"} {
		if strings.Contains(css, stale) {
			t.Errorf("stale Forest-at-Dusk token still referenced: %s", stale)
		}
	}
}
```

- [ ] **Step 3: Run the test to verify it fails**

Run:
```bash
go test ./internal/web/assets/ -run TestNoUndefinedHearthwoodTokens -v
```
Expected: FAIL — `--indigo-deep` missing and the four stale tokens present.

- [ ] **Step 4: Define `--indigo-deep` in the token block**

In `internal/web/assets/static/basm.css`, find the line:
```css
  --indigo-ink: #3d54a0;                       /* user hue on parchment */
```
and insert immediately after it:
```css
  --indigo-deep: light-dark(#2c3a72, #6f86c8); /* deep indigo — owner-portrait keyline */
```

- [ ] **Step 5: Replace the stale Forest-at-Dusk token references**

Apply these exact replacements in `internal/web/assets/static/basm.css` (replace
every occurrence):
- `var(--line,#0003)` → `var(--parch-edge)`
- `var(--line,#0002)` → `var(--parch-edge)`
- `var(--accent,#7a5)` → `var(--gold)`
- `var(--border)` → `var(--parch-edge)`
- `var(--parchment)` → `var(--surface-2)`

(These appear in `.head-switcher-*`, `.head-row*`, `.head-group-pip`, a
`border-bottom`, and `.quest-row:hover` — all current UI; the replacements map
each to its Hearthwood equivalent per the spec.)

- [ ] **Step 6: Run the test to verify it passes**

Run:
```bash
go test ./internal/web/assets/ -run TestNoUndefinedHearthwoodTokens -v
```
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/web/assets/static/basm.css internal/web/assets/css_tokens_test.go
git commit -m "$(printf 'fix(css): define --indigo-deep and drop stale Forest-at-Dusk tokens\n\nThe owner-portrait keyline referenced an undefined --indigo-deep; four\nForest-at-Dusk leftovers (--line/--accent/--border/--parchment) are remapped\nto their Hearthwood equivalents. Guarded by a token-presence test.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 3: Convert package `ui` to a qualified html import

Element-named atoms (`Button`, later `Select`/`Dialog`) cannot coexist with a
`gomponents/html` dot-import anywhere in package `ui`. `components.go` is the
only package-`ui` file that dot-imports html (`cardhead_test.go` is
`package ui_test`, external — leave it). Convert it. No behavior change.

**Files:**
- Modify: `internal/ui/components.go`

- [ ] **Step 1: Rewrite components.go with a qualified import**

Replace the entire contents of `internal/ui/components.go` with:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ErrorStrip is the inline card-error fragment, the gomponents equivalent of
// the legacy cardErrorStrip. g.Text auto-escapes msg, so a model- or
// user-derived string can never inject markup — the no-raw-HTML firewall.
// Never replace g.Text here with g.Raw.
func ErrorStrip(msg string) g.Node {
	return h.Div(h.Class("card-note card-note-error"), g.Text(msg))
}

// CardHead renders the shared kcard header: a kcard-kind span with the
// tool-icon image and the card title, plus an optional trailing node (a
// kcard-meta param line, a "manage all →" link, a tag, …). It exists so the
// card frame lives once instead of being hand-copied across every feature card.
// Attribute order (class, src, alt on the img) is load-bearing: the rendered
// HTML must stay byte-identical to the hand-rolled headers it replaces.
func CardHead(iconSrc, title string, trailing ...g.Node) g.Node {
	children := []g.Node{
		h.Span(h.Class("kcard-kind"),
			h.Img(h.Class("tool-icon"), h.Src(iconSrc), h.Alt("")),
			g.Text(title),
		),
	}
	children = append(children, trailing...)
	return h.Header(h.Class("kcard-head"), g.Group(children))
}
```

- [ ] **Step 2: Verify the package builds and output is byte-identical**

Run:
```bash
go test ./internal/ui/ -run TestCardHead -v && go build ./internal/ui/...
```
Expected: PASS — `cardhead_test.go`'s golden strings are unchanged, proving the
conversion is byte-for-byte identical.

- [ ] **Step 3: Commit**

```bash
git add internal/ui/components.go
git commit -m "$(printf 'refactor(ui): qualified html import so package ui can define element-named atoms\n\nA package-level func Button collides with a gomponents/html dot-import\n(Button already declared through dot-import). Switch components.go to a\nqualified h. import; output is byte-identical (golden tests unchanged).\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 4: The `ui.Button` atom

Port the export's `Button({variant, size, href, children})` (class-only against
the existing `.btn`/`.btn-primary`/`.btn-ghost`/`.btn-wood`/`.btn-sm` rules).

**Files:**
- Create: `internal/ui/button_test.go`
- Create: `internal/ui/button.go`

- [ ] **Step 1: Write the failing test**

Create `internal/ui/button_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func render(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}

func TestButtonVariants(t *testing.T) {
	cases := []struct {
		name  string
		props ui.ButtonProps
		want  string
	}{
		{"primary default", ui.ButtonProps{}, `<button class="btn btn-primary">Go</button>`},
		{"ghost", ui.ButtonProps{Variant: "ghost"}, `<button class="btn btn-ghost">Go</button>`},
		{"wood", ui.ButtonProps{Variant: "wood"}, `<button class="btn btn-wood">Go</button>`},
		{"small primary", ui.ButtonProps{Size: "sm"}, `<button class="btn btn-primary btn-sm">Go</button>`},
		{"link", ui.ButtonProps{Href: "/focus/settings"}, `<a class="btn btn-primary" href="/focus/settings">Go</a>`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := render(t, ui.Button(c.props, g.Text("Go")))
			if got != c.want {
				t.Fatalf("\n got: %s\nwant: %s", got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```bash
go test ./internal/ui/ -run TestButtonVariants -v
```
Expected: FAIL — `undefined: ui.Button` / `ui.ButtonProps`.

- [ ] **Step 3: Implement the Button atom**

Create `internal/ui/button.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ButtonProps configures a Button atom. Variant defaults to "primary".
type ButtonProps struct {
	Variant string // "primary" (default), "ghost", or "wood"
	Size    string // "" (default) or "sm"
	Href    string // when set, renders an <a> instead of a <button>
}

// buttonClass composes the Hearthwood button classes in the export's order:
// "btn", then the variant, then the optional size.
func buttonClass(p ButtonProps) string {
	variant := "btn-primary"
	switch p.Variant {
	case "ghost":
		variant = "btn-ghost"
	case "wood":
		variant = "btn-wood"
	}
	cls := "btn " + variant
	if p.Size == "sm" {
		cls += " btn-sm"
	}
	return cls
}

// Button renders the Hearthwood button atom. Extra attributes/children — a label
// (g.Text), Type("submit"), or a Datastar attribute — are passed through the
// variadic children. Datastar actions go on the enclosing <form> via
// data.On("submit", url, data.ModifierPrevent), not on the button itself.
func Button(p ButtonProps, children ...g.Node) g.Node {
	if p.Href != "" {
		return h.A(append([]g.Node{h.Class(buttonClass(p)), h.Href(p.Href)}, children...)...)
	}
	return h.Button(append([]g.Node{h.Class(buttonClass(p))}, children...)...)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run:
```bash
go test ./internal/ui/ -run TestButtonVariants -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/button.go internal/ui/button_test.go
git commit -m "$(printf 'feat(ui): add the Button atom\n\nFirst typed Hearthwood atom, ported from the Claude Design export. Class-only\nagainst the existing .btn rules; renders <a> when Href is set, else <button>.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 5: The page shell — `internal/ui/shell`

Port `layout.html` (page_head + topbar + shell_open/close) to a typed
`shell.Page`. Drop the "Boards" nav link (boards is being cut); nav is
Storybook (`/`), Settings (`/focus/settings`), Engine room (`/_/`). The
companion dock is a passed-in node (a placeholder until Phase 4).

**Files:**
- Create: `internal/ui/shell/shell_test.go`
- Create: `internal/ui/shell/shell.go`

- [ ] **Step 1: Write the failing test**

Create `internal/ui/shell/shell_test.go`:
```go
package shell_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui/shell"
)

func TestPage(t *testing.T) {
	var b strings.Builder
	page := shell.Page(shell.PageProps{
		Title:  "Storybook",
		Active: "storybook",
		Body:   g.Text("BODY"),
		Dock:   g.Text("DOCK"),
	})
	if err := page.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()

	for _, want := range []string{
		"<!doctype html>",
		`<html lang="en">`,
		`<title>Storybook · Balaur</title>`,
		`<link rel="stylesheet" href="/static/basm.css">`,
		`<script type="module" src="/static/datastar.js"></script>`,
		`<main id="main">BODY</main>`,
		`<aside id="dock">DOCK</aside>`,
		`localStorage.getItem('basm-theme')`, // the no-flash script survived
		`<a href="/" aria-current="page">`,    // active nav link
	} {
		if !strings.Contains(got, want) {
			t.Errorf("shell missing %q\nfull:\n%s", want, got)
		}
	}
	if strings.Contains(got, ">Boards<") {
		t.Error("Boards nav link should be gone (boards is cut)")
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```bash
go test ./internal/ui/shell/ -v
```
Expected: FAIL — package `shell` does not exist.

- [ ] **Step 3: Implement the shell**

Create `internal/ui/shell/shell.go`:
```go
// Package shell renders the Balaur page shell — the single place that emits a
// full <html> document. It ports the legacy layout.html (page_head, topbar,
// the card-first shell). Pages provide a Body and a Dock node; everything else
// patches into #main / #dock.
package shell

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// noFlashScript applies the saved theme + dock state before first paint, so the
// page never flashes the wrong colour scheme. Ported verbatim from layout.html.
const noFlashScript = `(function(){var d=document.documentElement,t=localStorage.getItem('basm-theme');if(t)d.classList.add(t);if(localStorage.getItem('basm-dock-full')==='1')d.classList.add('dock-full');var w=parseInt(localStorage.getItem('basm-dock-w'),10);if(w>=280&&w<=720)d.style.setProperty('--sidebar-w',w+'px');}());`

// PageProps configures a full page. Active is the nav key for aria-current
// ("storybook", "settings"); Body fills #main; Dock fills the companion #dock.
type PageProps struct {
	Title  string
	Active string
	Body   g.Node
	Dock   g.Node
}

// Page renders the full <html> document for one Balaur page.
func Page(p PageProps) g.Node {
	return g.Group([]g.Node{
		g.Raw("<!doctype html>"),
		h.HTML(h.Lang("en"),
			h.Head(
				pageHead(),
				h.TitleEl(g.Text(p.Title+" · Balaur")),
			),
			h.Body(
				topbar(p.Active),
				h.Div(h.Class("with-sidebar"),
					h.Main(h.ID("main"), p.Body),
				),
				h.Aside(h.ID("dock"), p.Dock),
			),
		),
	})
}

// pageHead is the shared <head> contents (minus <title>): meta, stylesheet, the
// no-flash theme script, favicon, and the Datastar + basm.js scripts.
func pageHead() g.Node {
	return g.Group([]g.Node{
		h.Meta(h.Charset("utf-8")),
		h.Meta(h.Name("viewport"), h.Content("width=device-width, initial-scale=1")),
		h.Link(h.Rel("stylesheet"), h.Href("/static/basm.css")),
		h.Script(g.Raw(noFlashScript)),
		h.Link(h.Rel("icon"), h.Href("/static/logo.png"), h.Type("image/png")),
		h.Link(h.Rel("apple-touch-icon"), h.Href("/static/logo.png")),
		h.Script(h.Type("module"), h.Src("/static/datastar.js")),
		h.Script(h.Src("/static/basm.js"), h.Defer()),
	})
}

// topbar is the wood-chrome header: crest brand, mono nav, theme toggle. The
// active link carries aria-current="page".
func topbar(active string) g.Node {
	return h.Header(h.Class("topbar"),
		h.A(h.Class("brand"), h.Href("/"),
			h.Img(h.Class("crest"), h.Src("/static/crest.png"), h.Alt(""), g.Attr("decoding", "async")),
			g.Text("Balaur"),
		),
		h.Nav(
			navLink("/", "Storybook", "storybook", active),
			navLink("/focus/settings", "Settings", "settings", active),
			h.A(h.Href("/_/"), h.Target("_blank"), g.Attr("rel", "noopener noreferrer"), g.Text("Engine room")),
		),
		h.Button(h.Class("theme-toggle"), h.Type("button"),
			g.Attr("onclick", "basmToggleTheme()"),
			h.Title("Toggle light/dark mode"),
			h.Aria("label", "Toggle light/dark mode"),
			g.Text("◑"),
		),
	)
}

func navLink(href, label, key, active string) g.Node {
	attrs := []g.Node{h.Href(href)}
	if key == active {
		attrs = append(attrs, h.Aria("current", "page"))
	}
	attrs = append(attrs, g.Text(label))
	return h.A(attrs...)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run:
```bash
go test ./internal/ui/shell/ -v
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/shell
git commit -m "$(printf 'feat(ui/shell): port the page shell to gomponents\n\nshell.Page renders the full <html> document (page_head, topbar, #main, #dock),\nporting layout.html. Drops the Boards nav link; carries the no-flash theme\nscript verbatim. The companion dock is a passed-in node (wired in Phase 4).\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 6: Serve the storybook at `/storybook` (thin end-to-end slice)

The payoff: a live page composing the real shell + a real atom, rendering on an
empty DB. `storybook.Body()` returns the gallery node; a plain `web` handler
composes `shell.Page`. The storybook is **not** a registered card and is **not**
added to `internal/feature/all`.

**Files:**
- Create: `internal/feature/storybook/storybook_test.go`
- Create: `internal/feature/storybook/storybook.go`
- Create: `internal/web/storybook.go`
- Modify: `internal/web/web.go` (register the route)

- [ ] **Step 1: Write the failing test for the gallery body**

Create `internal/feature/storybook/storybook_test.go`:
```go
package storybook_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/feature/storybook"
)

func TestBodyRendersAtoms(t *testing.T) {
	var b strings.Builder
	if err := storybook.Body().Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	for _, want := range []string{
		`<h1`,
		`class="btn btn-primary"`,
		`class="btn btn-ghost"`,
		`class="btn btn-wood"`,
		`class="btn btn-primary btn-sm"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("storybook body missing %q", want)
		}
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run:
```bash
go test ./internal/feature/storybook/ -v
```
Expected: FAIL — package `storybook` does not exist.

- [ ] **Step 3: Implement the gallery body**

Create `internal/feature/storybook/storybook.go`:
```go
// Package storybook builds the Hearthwood component gallery — the product
// surface at /. Body() returns the gallery node; the web gateway composes it
// into shell.Page. It is NOT a registered card and renders from in-package
// fixtures only (never PocketBase), so it works on an empty database.
package storybook

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// Body is the full storybook gallery. New component sections are appended here
// as atoms/organisms land in later phases.
func Body() g.Node {
	return h.Div(h.Class("sb"),
		h.H1(g.Text("Balaur — Hearthwood storybook")),
		section("Buttons",
			ui.Button(ui.ButtonProps{}, g.Text("Primary")),
			ui.Button(ui.ButtonProps{Variant: "ghost"}, g.Text("Ghost")),
			ui.Button(ui.ButtonProps{Variant: "wood"}, g.Text("Wood")),
			ui.Button(ui.ButtonProps{Size: "sm"}, g.Text("Small")),
		),
	)
}

// section wraps a labelled group of component variants.
func section(label string, items ...g.Node) g.Node {
	return h.Section(h.Class("sb-section"),
		h.H2(g.Text(label)),
		h.Div(h.Class("sb-row"), g.Group(items)),
	)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run:
```bash
go test ./internal/feature/storybook/ -v
```
Expected: PASS.

- [ ] **Step 5: Add the web handler**

Create `internal/web/storybook.go`:
```go
package web

import (
	g "maragu.dev/gomponents"

	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/feature/storybook"
	"github.com/alexradunet/balaur/internal/ui/shell"
)

// storybookHome serves the Hearthwood component gallery. It composes the real
// shell so the storybook can never drift from production styling. The companion
// dock is empty until the chat organisms land (Phase 4).
func (h *handlers) storybookHome(e *core.RequestEvent) error {
	e.Response.Header().Set("Content-Type", "text/html; charset=utf-8")
	page := shell.Page(shell.PageProps{
		Title:  "Storybook",
		Active: "storybook",
		Body:   storybook.Body(),
		Dock:   g.Text(""),
	})
	return page.Render(e.Response)
}
```

- [ ] **Step 6: Register the route**

In `internal/web/web.go`, just after the `se.Router.GET("/", h.boardHome)` line
(currently `internal/web/web.go:196`), add:
```go
	se.Router.GET("/storybook", h.storybookHome)
```

- [ ] **Step 7: Build, test, and view the page**

Run:
```bash
CGO_ENABLED=0 go build ./... && go test ./... && go vet ./...
```
Expected: build + all tests PASS.

Then run the app and confirm the page renders:
```bash
go run . &  sleep 3
curl -sS http://127.0.0.1:8090/storybook | grep -o 'class="btn btn-wood"'
kill %1
```
Expected: prints `class="btn btn-wood"` (open `http://127.0.0.1:8090/storybook`
in a browser to see the Hearthwood shell + buttons live).

- [ ] **Step 8: Commit**

```bash
git add internal/feature/storybook internal/web/storybook.go internal/web/web.go
git commit -m "$(printf 'feat(web): serve the Hearthwood storybook at /storybook\n\nThe foundation thin slice end-to-end: a plain web handler composes shell.Page\nwith storybook.Body() (the Button atom). Renders on an empty DB; not a\nregistered card. Boards stays at / until its Phase-3 cutover.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification

- [ ] Run the full suite and build:
```bash
go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check
```
Expected: all PASS, no whitespace errors.

- [ ] Manually confirm in a browser: `http://127.0.0.1:8090/storybook` shows the
  wood topbar (no "Boards" link), the crest brand, the four Hearthwood buttons,
  and the theme toggle (◑) flips light/dark without a flash. The existing app at
  `/` (boards) and `/static/basm.css` still work unchanged.

## What this slice delivers / what's next

**Delivered:** clean Hearthwood CSS, relocated assets, the `ui` atom layer
convention (qualified html import) + `ui.Button`, the `shell.Page` keystone, and
a live `/storybook`.

**Next plan (Phase 2):** the full atom + molecule library (Tag, Pip, Card,
Badge, Toggle, Avatar, Icon, Select, Tabs, …) with per-atom CSS rules, plus the
first dedupe (collapse `memoryPips`/`recordPips` into `ui.Pips`).
