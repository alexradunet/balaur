# Gomponents Atomic — Core Atoms + First Dedupe (Plan 02a) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the core class-only Hearthwood atoms (`Tag`, `Pips`, `Card`, `Stitch`, `FolkBand`, `Avatar`, `Icon`) to `internal/ui`, collapse the duplicated `memoryPips`/`recordPips` and the hand-rolled task tag into those atoms (byte-identically), extract a shared test helper, and showcase them all in `/storybook`.

**Architecture:** Each atom is a small `func(...) g.Node` in its own file in package `ui`, using the qualified `h "maragu.dev/gomponents/html"` import (never dot-import — element-named funcs collide otherwise). All seven reuse CSS classes that already exist in `basm.css`, so no new CSS and the dedupe refactors stay byte-identical (existing golden tests must still pass unchanged).

**Tech Stack:** Go, gomponents (`maragu.dev/gomponents` + `.../html`), existing `basm.css`.

**Scope:** Plan 02a of the migration (`docs/superpowers/specs/2026-06-15-gomponents-atomic-storybook-design.md`, Phase 2). The new-CSS atoms (Badge, Toggle, Toast, Alert, Tooltip, Skeleton, List, **Tabs**, TextField, Select, Breadcrumb, Pagination, SectionLabel, ScreenTitle) follow in Plan 02b.

**Conventions for every task:**
- Package `ui` files use `g "maragu.dev/gomponents"` + `h "maragu.dev/gomponents/html"` (qualified; no dot-import).
- After any change, run `go test ./...` (the FULL suite — a cross-cutting refactor can break a golden test or a `.tours` anchor in another package), `CGO_ENABLED=0 go build ./...`, `go vet ./...`.
- Atom tests live in `package ui_test` and use the shared `render(t, node)` helper from Task 1.

---

## File Structure

**Created:**
- `internal/ui/helpers_test.go` — shared `render(t, node) string` test helper.
- `internal/ui/tag.go` + `internal/ui/tag_test.go` — `ui.Tag`.
- `internal/ui/pip.go` + `internal/ui/pip_test.go` — `ui.Pips`.
- `internal/ui/card.go` + `internal/ui/card_test.go` — `ui.Card`, `ui.Stitch`, `ui.FolkBand`.
- `internal/ui/avatar.go` + `internal/ui/avatar_test.go` — `ui.Avatar`, `ui.Icon`.

**Modified:**
- `internal/ui/button_test.go` — drop the local `render` (moves to helpers_test.go).
- `internal/feature/taskcards/taskcard.go` — use `ui.Tag` for the recur chip.
- `internal/feature/knowledgecards/memory.go` — use `ui.Pips`; delete `memoryPips`/`recordPips`.
- `internal/feature/storybook/storybook.go` + `storybook_test.go` — showcase the new atoms.

---

## Task 1: Extract the shared `render` test helper

The `render` helper currently lives in `button_test.go`. Every new atom test in
`package ui_test` needs it, and Go forbids redeclaring it per file — so move it
to a shared file once.

**Files:**
- Create: `internal/ui/helpers_test.go`
- Modify: `internal/ui/button_test.go`

- [ ] **Step 1: Create the shared helper file**

Create `internal/ui/helpers_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"
)

// render renders a gomponents node to its HTML string, failing the test on
// error. Shared by the atom tests in this package.
func render(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}
```

- [ ] **Step 2: Remove the duplicate helper + now-unused import from button_test.go**

Replace the ENTIRE contents of `internal/ui/button_test.go` with (the `render`
func and the `strings` import are gone; everything else is unchanged):
```go
package ui_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

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

- [ ] **Step 3: Verify the ui package still tests + builds**

Run:
```bash
go test ./internal/ui/ -v 2>&1 | tail -15 && CGO_ENABLED=0 go build ./internal/ui/...
```
Expected: PASS (TestButtonVariants + the existing CardHead/ErrorStrip tests), build clean. The `render` helper now resolves from `helpers_test.go`.

- [ ] **Step 4: Commit**

```bash
git add internal/ui/helpers_test.go internal/ui/button_test.go
git commit -m "$(printf 'refactor(ui): extract shared render test helper\n\nMove render(t, node) out of button_test.go into helpers_test.go so the\nincoming atom tests can share it without redeclaration collisions.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: `ui.Tag` atom + collapse the task recur chip

Export `Tag` is `<span class="tag">{children}</span>` (the `▪` prefix comes from
`.tag::before` in CSS). `taskcard.go` hand-rolls exactly that for the recurrence
chip — replace it (byte-identical).

**Files:**
- Create: `internal/ui/tag.go`, `internal/ui/tag_test.go`
- Modify: `internal/feature/taskcards/taskcard.go`

- [ ] **Step 1: Write the failing test**

Create `internal/ui/tag_test.go`:
```go
package ui_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestTag(t *testing.T) {
	got := render(t, ui.Tag(g.Text("daily")))
	want := `<span class="tag">daily</span>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestTag -v`
Expected: FAIL — `undefined: ui.Tag`.

- [ ] **Step 3: Implement the atom**

Create `internal/ui/tag.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Tag is the small mono chip with a teal ▪ prefix (the prefix is .tag::before
// in CSS, not markup). Children are the label.
func Tag(children ...g.Node) g.Node {
	return h.Span(append([]g.Node{h.Class("tag")}, children...)...)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestTag -v`
Expected: PASS.

- [ ] **Step 5: Collapse the hand-rolled tag in taskcard.go**

In `internal/feature/taskcards/taskcard.go`, find:
```go
		g.If(v.RecurLine != "", Span(Class("tag"), g.Text(v.RecurLine))),
```
and replace with:
```go
		g.If(v.RecurLine != "", ui.Tag(g.Text(v.RecurLine))),
```
Then ensure the package imports `ui`. The import block already imports
gomponents and gomponents/html; add (if not present), in the module-path import
group:
```go
	"github.com/alexradunet/balaur/internal/ui"
```

- [ ] **Step 6: Verify taskcards still renders byte-identically + full suite**

Run:
```bash
go test ./internal/feature/taskcards/... -v 2>&1 | tail -20
go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
```
Expected: taskcards tests PASS unchanged (`ui.Tag` emits the same `<span class="tag">…</span>`), full suite green, build clean.

- [ ] **Step 7: Commit**

```bash
git add internal/ui/tag.go internal/ui/tag_test.go internal/feature/taskcards/taskcard.go
git commit -m "$(printf 'feat(ui): add Tag atom; use it for the task recur chip\n\nui.Tag(children) -> <span class=\"tag\">…</span>, replacing the hand-rolled\nSpan(Class(\"tag\")) in taskcard.go (byte-identical, golden tests unchanged).\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 3: `ui.Pips` atom + collapse `memoryPips`/`recordPips`

`memoryPips` and `recordPips` in `knowledgecards/memory.go` are byte-identical
duplicates. Replace both with one `ui.Pips`. The existing golden tests pin
`class="kcard-pips"`, `title="importance 3/5"`, `class="pip pip-on"`/`class="pip"`
— `ui.Pips` must reproduce them exactly. (`memory.go` already imports `ui`; `fmt`
is still used elsewhere, so it stays.)

**Files:**
- Create: `internal/ui/pip.go`, `internal/ui/pip_test.go`
- Modify: `internal/feature/knowledgecards/memory.go`

- [ ] **Step 1: Write the failing test**

Create `internal/ui/pip_test.go`:
```go
package ui_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestPips(t *testing.T) {
	got := render(t, ui.Pips(3, 5, ""))
	want := `<span class="kcard-pips" title="importance 3/5">` +
		`<i class="pip pip-on"></i><i class="pip pip-on"></i><i class="pip pip-on"></i>` +
		`<i class="pip"></i><i class="pip"></i></span>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestPipsExplicitTitle(t *testing.T) {
	got := render(t, ui.Pips(0, 3, "ctx"))
	want := `<span class="kcard-pips" title="ctx"><i class="pip"></i><i class="pip"></i><i class="pip"></i></span>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestPips -v`
Expected: FAIL — `undefined: ui.Pips`.

- [ ] **Step 3: Implement the atom**

Create `internal/ui/pip.go`:
```go
package ui

import (
	"fmt"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Pips renders the importance indicator: max small squares, the first `level`
// filled (pip-on). An empty title defaults to "importance {level}/{max}".
func Pips(level, max int, title string) g.Node {
	if title == "" {
		title = fmt.Sprintf("importance %d/%d", level, max)
	}
	pips := make([]g.Node, max)
	for i := 0; i < max; i++ {
		cls := "pip"
		if i < level {
			cls = "pip pip-on"
		}
		pips[i] = h.I(h.Class(cls))
	}
	return h.Span(h.Class("kcard-pips"), g.Attr("title", title), g.Group(pips))
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestPips -v`
Expected: PASS (both subtests).

- [ ] **Step 5: Collapse the duplicates in memory.go**

In `internal/feature/knowledgecards/memory.go`:

1. Replace the call `memoryPips(row.Importance),` with `ui.Pips(row.Importance, 5, ""),`
2. Replace the call `recordPips(r.Importance),` with `ui.Pips(r.Importance, 5, ""),`
3. Delete the entire `memoryPips` function:
```go
// memoryPips renders the 5-pip importance indicator for a summary row.
func memoryPips(importance int) g.Node {
	pips := make([]g.Node, 5)
	for i := 0; i < 5; i++ {
		if i < importance {
			pips[i] = I(Class("pip pip-on"))
		} else {
			pips[i] = I(Class("pip"))
		}
	}
	return Span(
		Class("kcard-pips"),
		g.Attr("title", fmt.Sprintf("importance %d/5", importance)),
		g.Group(pips),
	)
}
```
4. Delete the entire `recordPips` function:
```go
// recordPips renders the 5-pip importance indicator for a record card.
func recordPips(importance int) g.Node {
	pips := make([]g.Node, 5)
	for i := 0; i < 5; i++ {
		if i < importance {
			pips[i] = I(Class("pip pip-on"))
		} else {
			pips[i] = I(Class("pip"))
		}
	}
	return Span(
		Class("kcard-pips"),
		g.Attr("title", fmt.Sprintf("importance %d/5", importance)),
		g.Group(pips),
	)
}
```

`ui.Pips(x, 5, "")` defaults the title to `importance x/5` — exactly what the
deleted helpers produced. Do NOT remove the `fmt` import (still used at the
importance input, the `used ×N` meta, and the skills param line).

- [ ] **Step 6: Verify the knowledge cards render byte-identically + full suite**

Run:
```bash
go test ./internal/feature/knowledgecards/... -v 2>&1 | tail -25
go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
```
Expected: all knowledgecards golden tests PASS unchanged (the pip markup is
identical), full suite green, build clean.

- [ ] **Step 7: Commit**

```bash
git add internal/ui/pip.go internal/ui/pip_test.go internal/feature/knowledgecards/memory.go
git commit -m "$(printf 'feat(ui): add Pips atom; collapse memoryPips/recordPips into it\n\nui.Pips(level, max, title) replaces the two byte-identical pip helpers in\nknowledgecards (empty title defaults to \"importance level/max\"). Golden\ntests unchanged — markup is identical.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 4: `ui.Card`, `ui.Stitch`, `ui.FolkBand` atoms

Three simple class-wrapper atoms from the export (all `<div class="…">`). New —
no existing usage to refactor; they go straight into the storybook (Task 6).

**Files:**
- Create: `internal/ui/card.go`, `internal/ui/card_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/ui/card_test.go`:
```go
package ui_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestCard(t *testing.T) {
	got := render(t, ui.Card(g.Text("hi")))
	if want := `<div class="card">hi</div>`; got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestStitch(t *testing.T) {
	got := render(t, ui.Stitch())
	if want := `<div class="stitch"></div>`; got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestFolkBand(t *testing.T) {
	got := render(t, ui.FolkBand())
	if want := `<div class="folk-band"></div>`; got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run 'TestCard|TestStitch|TestFolkBand' -v`
Expected: FAIL — `undefined: ui.Card` / `ui.Stitch` / `ui.FolkBand`.

- [ ] **Step 3: Implement the atoms**

Create `internal/ui/card.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Card is the generic parchment content panel (the gold corner notch is the
// .card::after pseudo-element, not markup).
func Card(children ...g.Node) g.Node {
	return h.Div(append([]g.Node{h.Class("card")}, children...)...)
}

// Stitch is a 2px dashed folk separator between sections. Pass extra attributes
// (e.g. an inline Style margin override) through the variadic.
func Stitch(attrs ...g.Node) g.Node {
	return h.Div(append([]g.Node{h.Class("stitch")}, attrs...)...)
}

// FolkBand is the horizontal woven carpet stripe. Use sparingly in dense UI.
func FolkBand(attrs ...g.Node) g.Node {
	return h.Div(append([]g.Node{h.Class("folk-band")}, attrs...)...)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run 'TestCard|TestStitch|TestFolkBand' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/ui/card.go internal/ui/card_test.go
git commit -m "$(printf 'feat(ui): add Card, Stitch, FolkBand atoms\n\nClass-only wrappers from the export (.card, .stitch, .folk-band).\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 5: `ui.Avatar` + `ui.Icon` atoms

`Avatar` is the wood-framed pixel portrait (`.balaur-avatar` span + img, with a
`--avatar-size` custom property and `data-state` for the thinking glow). `Icon`
is the pixel tool-icon `<img>` helper. Both new.

**Files:**
- Create: `internal/ui/avatar.go`, `internal/ui/avatar_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/ui/avatar_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestAvatarDecorativeDefaults(t *testing.T) {
	got := render(t, ui.Avatar(ui.AvatarProps{Src: "/static/avatars/balaur-01.png"}))
	for _, want := range []string{
		`class="balaur-avatar balaur-avatar-balaur"`,
		`data-kind="balaur"`,
		`data-state="idle"`,
		`style="--avatar-size:54px"`,
		`aria-hidden="true"`,
		`<img src="/static/avatars/balaur-01.png" alt="" decoding="async">`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("avatar missing %q in: %s", want, got)
		}
	}
}

func TestAvatarNamedNotHidden(t *testing.T) {
	got := render(t, ui.Avatar(ui.AvatarProps{Src: "/x.png", Kind: "soul", State: "thinking", Alt: "Wise", Size: 96}))
	if strings.Contains(got, "aria-hidden") {
		t.Errorf("named avatar (Alt set) must not be aria-hidden: %s", got)
	}
	for _, want := range []string{`balaur-avatar-soul`, `data-state="thinking"`, `style="--avatar-size:96px"`, `alt="Wise"`} {
		if !strings.Contains(got, want) {
			t.Errorf("avatar missing %q in: %s", want, got)
		}
	}
}

func TestIcon(t *testing.T) {
	got := render(t, ui.Icon("scroll"))
	if want := `<img class="tool-icon" src="/static/icons/scroll.png" alt="">`; got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run 'TestAvatar|TestIcon' -v`
Expected: FAIL — `undefined: ui.Avatar` / `ui.AvatarProps` / `ui.Icon`.

- [ ] **Step 3: Implement the atoms**

Create `internal/ui/avatar.go`:
```go
package ui

import (
	"strconv"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// AvatarProps configures an Avatar. Kind defaults to "balaur", Size to 54,
// State to "idle". An empty Alt marks the portrait decorative (aria-hidden).
type AvatarProps struct {
	Src   string
	Kind  string // "balaur" (default) or "soul"
	State string // "idle" (default), "thinking", "working"
	Alt   string
	Size  int
}

// Avatar renders a Hearthwood portrait: the beveled wood frame (.balaur-avatar)
// holding a borderless pixel-art img. State drives the basm-glow via data-state;
// Size sets the --avatar-size custom property.
func Avatar(p AvatarProps) g.Node {
	kind := p.Kind
	if kind == "" {
		kind = "balaur"
	}
	state := p.State
	if state == "" {
		state = "idle"
	}
	size := p.Size
	if size == 0 {
		size = 54
	}
	attrs := []g.Node{
		h.Class("balaur-avatar balaur-avatar-" + kind),
		g.Attr("data-kind", kind),
		g.Attr("data-state", state),
		h.Style("--avatar-size:" + strconv.Itoa(size) + "px"),
	}
	if p.Alt == "" {
		attrs = append(attrs, g.Attr("aria-hidden", "true"))
	}
	attrs = append(attrs, h.Img(h.Src(p.Src), h.Alt(p.Alt), g.Attr("decoding", "async")))
	return h.Span(attrs...)
}

// Icon renders a pixel-art tool icon by name from /static/icons/{name}.png,
// borderless and pixelated (the .tool-icon class). Names: scroll, tome, key,
// quill, orb, lens, shield, check, bell, gem, flame, hourglass, rune_x.
func Icon(name string) g.Node {
	return h.Img(h.Class("tool-icon"), h.Src("/static/icons/"+name+".png"), h.Alt(""))
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run 'TestAvatar|TestIcon' -v`
Expected: PASS (all three).

- [ ] **Step 5: Commit**

```bash
git add internal/ui/avatar.go internal/ui/avatar_test.go
git commit -m "$(printf 'feat(ui): add Avatar and Icon atoms\n\nui.Avatar renders the wood-framed pixel portrait (.balaur-avatar, data-state\nglow, --avatar-size); empty Alt -> aria-hidden. ui.Icon(name) renders a\n/static/icons/<name>.png tool icon.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 6: Showcase the new atoms in `/storybook`

Extend `storybook.Body()` with sections for the six new atom groups so the
gallery actually displays them.

**Files:**
- Modify: `internal/feature/storybook/storybook.go`, `internal/feature/storybook/storybook_test.go`

- [ ] **Step 1: Add the assertions (failing test first)**

Replace the `for _, want := range []string{ ... }` slice in
`internal/feature/storybook/storybook_test.go`'s `TestBodyRendersAtoms` with the
extended list:
```go
	for _, want := range []string{
		`<h1`,
		`class="btn btn-primary"`,
		`class="btn btn-ghost"`,
		`class="btn btn-wood"`,
		`class="btn btn-primary btn-sm"`,
		`class="tag"`,
		`class="kcard-pips"`,
		`class="pip pip-on"`,
		`class="card"`,
		`class="balaur-avatar`,
		`class="tool-icon"`,
		`class="stitch"`,
		`class="folk-band"`,
	} {
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/feature/storybook/ -v`
Expected: FAIL — the new substrings (`class="tag"`, `class="kcard-pips"`, …) are
not yet in the body.

- [ ] **Step 3: Extend the gallery body**

In `internal/feature/storybook/storybook.go`, replace the `Body()` function with
(adds the new sections after Buttons; reuses the existing `section` helper):
```go
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
		section("Tags",
			ui.Tag(g.Text("daily")),
			ui.Tag(g.Text("⟳ weekly")),
		),
		section("Importance pips",
			ui.Pips(1, 5, ""),
			ui.Pips(3, 5, ""),
			ui.Pips(5, 5, ""),
		),
		section("Card",
			ui.Card(h.H3(g.Text("A parchment card")), h.P(g.Text("Body text on parchment."))),
		),
		section("Avatars",
			ui.Avatar(ui.AvatarProps{Src: "/static/avatars/balaur-01.png", Kind: "balaur", Alt: "Wise"}),
			ui.Avatar(ui.AvatarProps{Src: "/static/avatars/balaur-01.png", State: "thinking"}),
			ui.Avatar(ui.AvatarProps{Src: "/static/avatars/soul-01.png", Kind: "soul", Alt: "Owner"}),
		),
		section("Icons",
			ui.Icon("scroll"), ui.Icon("tome"), ui.Icon("quill"), ui.Icon("lens"), ui.Icon("flame"),
		),
		section("Separators",
			ui.Stitch(),
			ui.FolkBand(),
		),
	)
}
```

- [ ] **Step 4: Run to verify it passes + full suite + view**

Run:
```bash
go test ./internal/feature/storybook/ -v 2>&1 | tail -8
go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./... && go vet ./...
```
Expected: storybook test PASS, full suite green, build+vet clean. Optionally view:
```bash
go run . &  sleep 3
curl -sS http://127.0.0.1:8090/storybook | grep -oE 'class="(tag|kcard-pips|card|balaur-avatar[^"]*|tool-icon|stitch|folk-band)"' | sort -u
kill %1
```
Expected: the new atom classes appear.

- [ ] **Step 5: Commit**

```bash
git add internal/feature/storybook/storybook.go internal/feature/storybook/storybook_test.go
git commit -m "$(printf 'feat(storybook): showcase Tag, Pips, Card, Avatar, Icon, separators\n\nAdd gallery sections for the new core atoms; the storybook now renders the\nfull class-only atom set on an empty DB.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification

- [ ] Run the whole suite + build + vet + diff check:
```bash
go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check
```
Expected: all green, no whitespace errors.

- [ ] Confirm the dedupes are real: `grep -rn "memoryPips\|recordPips" internal/` prints nothing; `grep -rn 'Span(Class("tag")' internal/feature/` prints nothing (the task chip now uses `ui.Tag`).

## What this slice delivers / what's next

**Delivered:** seven core class-only atoms (`Tag`, `Pips`, `Card`, `Stitch`,
`FolkBand`, `Avatar`, `Icon`), two real dedupes (pips, task tag), a shared test
helper, and a fuller storybook.

**Next (Plan 02b):** the new-CSS atoms/molecules — Badge, Toggle, Toast, Alert,
Tooltip, Skeleton, List/ListItem, **Tabs**, TextField, Select, Breadcrumb,
Pagination, SectionLabel, ScreenTitle — each pairing a Go atom with new `basm.css`
rules (the only group in Phase 2 that adds CSS).
