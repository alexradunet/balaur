# Gomponents Atomic — Display Atoms (Plan 02b-i) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add four static display/feedback atoms to `internal/ui` — `Badge`, `Alert`, `Tooltip`, `Skeleton` — each pairing a typed gomponents atom with **new tokenized `basm.css` rules**, plus the storybook gallery's missing layout CSS, and showcase them at `/storybook`.

**Architecture:** Unlike Plan 02a's class-only atoms, these need *new* CSS. The export ships them as inline styles; this slice lifts each into a `basm.css` rule that consumes existing Hearthwood tokens (no raw hex). Atoms are `func(...) g.Node` in package `ui` with the qualified `h "maragu.dev/gomponents/html"` import. The Go render is golden-tested; the CSS is verified by build + the storybook view.

**Tech Stack:** Go, gomponents (`maragu.dev/gomponents` + `.../html`), vanilla `basm.css` (no build step).

**Scope:** Plan 02b-i of the migration (`docs/superpowers/specs/2026-06-15-gomponents-atomic-storybook-design.md`, Phase 2b). The remaining 2b atoms — forms (TextField/Select/Toggle), nav (Tabs/Breadcrumb/Pagination), List, and the text helpers — follow in later 02b sub-plans.

**Design provenance:** The Go atoms and CSS below were extracted from the export and vetted by an adversarial CSS review (workflow `wf_43be3abc-75a`): zero raw hex, all tokens verified to exist in `:root`, no class collisions. Two adjustments were applied to the vetted output: (1) class names normalized to the repo's **single-dash** convention (basm.css uses no BEM `--`/`__`), and (2) all new CSS is **appended at the end of `basm.css`** (so `.alert .tool-icon` follows the existing `.tool-icon` rules).

**Conventions for every task:**
- Package `ui` files: `g "maragu.dev/gomponents"` + `h "maragu.dev/gomponents/html"` (qualified; no dot-import).
- New CSS goes at the **very end** of `internal/web/assets/static/basm.css`.
- After each task: `go test ./...` (full suite), `CGO_ENABLED=0 go build ./...`, `go vet ./...`.
- Atom tests are `package ui_test` and use the shared `render(t, node)` helper (already in `internal/ui/helpers_test.go`).

---

## File Structure

**Created:**
- `internal/ui/badge.go` + `badge_test.go`
- `internal/ui/alert.go` + `alert_test.go`
- `internal/ui/tooltip.go` + `tooltip_test.go`
- `internal/ui/skeleton.go` + `skeleton_test.go`

**Modified:**
- `internal/web/assets/static/basm.css` — append gallery + Badge + Alert + Tooltip + Skeleton rule blocks.
- `internal/feature/storybook/storybook.go` + `storybook_test.go` — showcase the new atoms.

---

## Task 1: Storybook gallery CSS

The gallery markup (`.sb`/`.sb-section`/`.sb-row`) has no CSS rule, so it's
unstyled. Add restrained, tokenized layout CSS. CSS-only — no Go, no golden test
(verified by build + the storybook view).

**Files:**
- Modify: `internal/web/assets/static/basm.css` (append at end)

- [ ] **Step 1: Append the gallery CSS block to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`:
```css

/* ── Storybook gallery chrome (dev surface) ───────────────────────────
   .sb sits inside <main id="main">, which already centers + pads (0 6vw);
   restate only the vertical rhythm here or the 6vw compounds to 12vw. */
.sb { padding: 40px 0 96px; }
.sb > h1 { font-size: 30px; margin: 0 0 4px; }
.sb-section { padding: 28px 0 4px; border-top: 2px dashed var(--hair); }
.sb-section:first-of-type { border-top: 0; }
.sb-section > h2 {
  display: flex; align-items: center; gap: 12px; margin: 0 0 18px;
  font-family: var(--font-mono); font-size: 11px; font-weight: 700;
  letter-spacing: .08em; text-transform: uppercase; color: var(--muted);
}
.sb-section > h2::after {
  content: ""; flex: 1; height: 2px;
  background: linear-gradient(to right, var(--hair) 50%, transparent 50%) 0 0 / 8px 2px repeat-x;
  opacity: .7;
}
.sb-row { display: flex; flex-wrap: wrap; align-items: center; gap: 18px 22px; }
```

- [ ] **Step 2: Verify build + suite + the token test still passes**

Run:
```bash
CGO_ENABLED=0 go build ./... && go test ./internal/web/assets/ ./internal/feature/storybook/ && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
```
Expected: build clean, the assets token test still passes (no raw hex / undefined tokens introduced), storybook test passes, full suite green. Optionally view `/storybook` (`go run .`, open `http://127.0.0.1:8090/storybook`) — sections now have vertical rhythm, mono kickers with a stitch rule, and rows wrap with gaps.

- [ ] **Step 3: Commit**

```bash
git add internal/web/assets/static/basm.css
git commit -m "$(printf 'feat(storybook): add gallery layout CSS\n\nStyle the .sb/.sb-section/.sb-row chrome (vertical rhythm, mono section\nkickers with a stitch rule, wrapping rows). Vertical padding only — <main>\nalready applies the 0 6vw horizontal frame.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: `ui.Badge` atom

A small count/status chip with tone variants (gold/ember/teal/wood) and a
dot-vs-pill shape. New CSS (tokenized; the export's hardcoded `#1c0d04`/`#06120f`
map to `--ink`/`--surface`).

**Files:**
- Create: `internal/ui/badge.go`, `internal/ui/badge_test.go`
- Modify: `internal/web/assets/static/basm.css` (append)

- [ ] **Step 1: Write the failing test**

Create `internal/ui/badge_test.go`:
```go
package ui_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestBadge(t *testing.T) {
	cases := []struct {
		name  string
		node  g.Node
		want  string
	}{
		{"gold default pill", ui.Badge(ui.BadgeProps{}, g.Text("3")), `<span class="badge badge-gold">3</span>`},
		{"ember pill", ui.Badge(ui.BadgeProps{Tone: ui.BadgeEmber}, g.Text("9")), `<span class="badge badge-ember">9</span>`},
		{"teal pill", ui.Badge(ui.BadgeProps{Tone: ui.BadgeTeal}, g.Text("new")), `<span class="badge badge-teal">new</span>`},
		{"wood dot", ui.Badge(ui.BadgeProps{Tone: ui.BadgeWood, Dot: true}), `<span class="badge-dot badge-wood"></span>`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := render(t, c.node); got != c.want {
				t.Fatalf("\n got: %s\nwant: %s", got, c.want)
			}
		})
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestBadge -v`
Expected: FAIL — `undefined: ui.Badge` / `ui.BadgeProps` / `ui.BadgeEmber` etc.

- [ ] **Step 3: Implement the atom**

Create `internal/ui/badge.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// BadgeTone selects the Badge color triple. The zero value is BadgeGold.
type BadgeTone string

const (
	BadgeGold  BadgeTone = "gold"  // default / brand
	BadgeEmber BadgeTone = "ember" // urgent
	BadgeTeal  BadgeTone = "teal"  // info
	BadgeWood  BadgeTone = "wood"  // neutral
)

// BadgeProps configures a Badge. Tone defaults to BadgeGold. When Dot is true
// the badge renders as a bare 9px marker and any children are ignored.
type BadgeProps struct {
	Tone BadgeTone
	Dot  bool
}

// Badge is a small count / status chip. Tones: gold (default), ember (urgent),
// teal (info), wood (neutral). Set Dot for a bare marker instead of a pill.
func Badge(props BadgeProps, children ...g.Node) g.Node {
	tone := props.Tone
	if tone == "" {
		tone = BadgeGold
	}
	toneClass := "badge-" + string(tone)
	if props.Dot {
		return h.Span(h.Class("badge-dot " + toneClass))
	}
	return h.Span(h.Class("badge "+toneClass), g.Group(children))
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestBadge -v`
Expected: PASS (all 4 subcases).

- [ ] **Step 5: Append the Badge CSS to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`:
```css

/* ── Badge — small count / status chip ───────────────────────────────
   Tones gold (default) / ember (urgent) / teal (info) / wood (neutral).
   .badge is the pill; .badge-dot the bare marker; tone classes set the
   shared --badge-bg/--badge-bd (+ pill --badge-fg). Export hex #1c0d04 ->
   --ink, #06120f -> --surface, white sheen -> --bevel-light. */
.badge {
  display: inline-flex; align-items: center; justify-content: center;
  min-width: 20px; height: 20px; padding: 0 6px; box-sizing: border-box;
  font-family: var(--font-mono); font-size: 11px; font-weight: 700;
  line-height: 1; letter-spacing: .02em;
  color: var(--badge-fg); background: var(--badge-bg);
  border: 2px solid var(--badge-bd); border-radius: var(--radius);
  box-shadow: inset 0 2px 0 var(--bevel-light);
}
.badge-dot {
  display: inline-block; width: 9px; height: 9px;
  background: var(--badge-bg); border: 2px solid var(--badge-bd);
  border-radius: var(--radius);
}
.badge-gold  { --badge-bg: var(--gold);      --badge-bd: var(--gold-deep);  --badge-fg: var(--ink); }
.badge-ember { --badge-bg: var(--ember);     --badge-bd: var(--ember-deep); --badge-fg: var(--ink); }
.badge-teal  { --badge-bg: var(--teal-deep); --badge-bd: var(--outline-2);  --badge-fg: var(--surface); }
.badge-wood  { --badge-bg: var(--chrome-2);  --badge-bd: var(--outline-2);  --badge-fg: var(--chrome-fg); }
```

- [ ] **Step 6: Verify build + suite**

Run:
```bash
CGO_ENABLED=0 go build ./... && go test ./internal/web/assets/ && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
```
Expected: build clean, the assets token test still passes (no raw hex / undefined tokens), full suite green.

- [ ] **Step 7: Commit**

```bash
git add internal/ui/badge.go internal/ui/badge_test.go internal/web/assets/static/basm.css
git commit -m "$(printf 'feat(ui): add Badge atom\n\nui.Badge(BadgeProps{Tone,Dot}, children) — gold/ember/teal/wood pill or dot.\nNew tokenized .badge CSS (export hex mapped to --ink/--surface/--bevel-light).\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 3: `ui.Alert` atom

A parchment callout band: a 2-column (icon + body) grid with an asymmetric thick
left accent stripe; tone drives the stripe/kicker color and default icon. It
composes the existing `Icon` atom.

**Files:**
- Create: `internal/ui/alert.go`, `internal/ui/alert_test.go`
- Modify: `internal/web/assets/static/basm.css` (append)

- [ ] **Step 1: Write the failing test**

Create `internal/ui/alert_test.go`:
```go
package ui_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestAlertWarn(t *testing.T) {
	got := render(t, ui.Alert(ui.AlertProps{Tone: "warn", Title: "Caution"}, g.Text("Heads up.")))
	for _, want := range []string{
		`class="alert alert-warn"`,
		`role="alert"`,
		`<img class="tool-icon" src="/static/icons/shield.png" alt="">`,
		`<div class="alert-kicker">Caution</div>`,
		`<div class="alert-body">Heads up.</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("alert missing %q in: %s", want, got)
		}
	}
}

func TestAlertInfoDefaultsNoTitle(t *testing.T) {
	got := render(t, ui.Alert(ui.AlertProps{}, g.Text("note")))
	for _, want := range []string{`class="alert alert-info"`, `role="note"`, `src="/static/icons/orb.png"`, `<div class="alert-body">note</div>`} {
		if !strings.Contains(got, want) {
			t.Errorf("alert missing %q in: %s", want, got)
		}
	}
	if strings.Contains(got, "alert-kicker") {
		t.Errorf("empty Title must omit the kicker row: %s", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestAlert -v`
Expected: FAIL — `undefined: ui.Alert` / `ui.AlertProps`.

- [ ] **Step 3: Implement the atom**

Create `internal/ui/alert.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// AlertProps configures an Alert callout. Tone defaults to "info" and drives the
// left-accent stripe, the kicker color, and the default icon. Title is the
// optional uppercase mono kicker (empty omits that row). Icon overrides the
// tone's default icon name.
type AlertProps struct {
	Tone  string // "info" (default), "warn", "danger"
	Title string
	Icon  string
}

// alertTone maps a tone to its CSS modifier class, ARIA role, and default icon.
// Unknown tones fall back to info (matching the export's map[tone]||info).
func alertTone(tone string) (cls, role, icon string) {
	switch tone {
	case "warn":
		return "alert-warn", "alert", "shield"
	case "danger":
		return "alert-danger", "alert", "flame"
	default:
		return "alert-info", "note", "orb"
	}
}

// Alert renders the Hearthwood callout band: a parchment surface with an
// asymmetric thick left accent stripe and a 2-column icon/body grid. role is
// "note" for info and "alert" for warn/danger. Pass the message as children.
func Alert(p AlertProps, children ...g.Node) g.Node {
	cls, role, defIcon := alertTone(p.Tone)
	icon := p.Icon
	if icon == "" {
		icon = defIcon
	}

	body := []g.Node{}
	if p.Title != "" {
		body = append(body, h.Div(h.Class("alert-kicker"), g.Text(p.Title)))
	}
	body = append(body, h.Div(append([]g.Node{h.Class("alert-body")}, children...)...))

	return h.Div(
		h.Class("alert "+cls),
		g.Attr("role", role),
		Icon(icon),
		h.Div(body...),
	)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestAlert -v`
Expected: PASS (both subtests).

- [ ] **Step 5: Append the Alert CSS to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`
(it must come after the existing `.tool-icon` rules — appending at EOF satisfies
that):
```css

/* ── Alert — parchment callout band ──────────────────────────────────
   2-col (icon + body) grid with an asymmetric thick left accent stripe.
   Tone drives the stripe + kicker color and the default icon; role is set
   in markup (note for info, alert for warn/danger). */
.alert {
  display: grid; grid-template-columns: 26px 1fr; column-gap: 12px;
  align-items: start; padding: 13px 16px 14px;
  background: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px;
  color: var(--ink);
  border: 2px solid var(--parch-edge); border-left: 6px solid var(--alert-edge);
  box-shadow: var(--parch-bevel);
}
/* Icon atom sizing inside the band (wins on specificity over img.tool-icon). */
.alert .tool-icon { width: 22px; height: 22px; margin: 1px 0 0 0; image-rendering: pixelated; }
.alert-kicker {
  margin-bottom: 4px; font-family: var(--font-mono); font-size: 11px;
  font-weight: 700; letter-spacing: .06em; text-transform: uppercase; color: var(--alert-kick);
}
.alert-body { font-size: 14px; line-height: 1.5; }
.alert-info   { --alert-edge: var(--gold-deep);  --alert-kick: var(--gold-ink); }
.alert-warn   { --alert-edge: var(--ember-deep); --alert-kick: var(--ember-deep); }
.alert-danger { --alert-edge: var(--ember-red);  --alert-kick: var(--ember-red); }
```

- [ ] **Step 6: Verify build + suite**

Run:
```bash
CGO_ENABLED=0 go build ./... && go test ./internal/web/assets/ && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
```
Expected: build clean, assets token test passes, full suite green.

- [ ] **Step 7: Commit**

```bash
git add internal/ui/alert.go internal/ui/alert_test.go internal/web/assets/static/basm.css
git commit -m "$(printf 'feat(ui): add Alert atom\n\nui.Alert(AlertProps{Tone,Title,Icon}, children) — parchment callout with an\nasymmetric left accent stripe; info/warn/danger tones, composes ui.Icon.\nNew tokenized .alert CSS.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 4: `ui.Tooltip` atom

A wood label revealed on hover/focus — pure CSS, no client JS. Wraps a trigger
child.

**Files:**
- Create: `internal/ui/tooltip.go`, `internal/ui/tooltip_test.go`
- Modify: `internal/web/assets/static/basm.css` (append)

- [ ] **Step 1: Write the failing test**

Create `internal/ui/tooltip_test.go`:
```go
package ui_test

import (
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestTooltipTop(t *testing.T) {
	got := render(t, ui.Tooltip(ui.TooltipProps{Label: "Keep it"}, g.Text("x")))
	want := `<span class="tooltip">x<span class="tooltip-bubble" role="tooltip">Keep it</span></span>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}

func TestTooltipBottom(t *testing.T) {
	got := render(t, ui.Tooltip(ui.TooltipProps{Label: "hi", Position: "bottom"}, g.Text("x")))
	want := `<span class="tooltip tooltip-bottom">x<span class="tooltip-bubble" role="tooltip">hi</span></span>`
	if got != want {
		t.Fatalf("\n got: %s\nwant: %s", got, want)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestTooltip -v`
Expected: FAIL — `undefined: ui.Tooltip` / `ui.TooltipProps`.

- [ ] **Step 3: Implement the atom**

Create `internal/ui/tooltip.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// TooltipProps configures a Tooltip. Position is "top" (default) or "bottom".
type TooltipProps struct {
	Label    string
	Position string
}

// Tooltip wraps a trigger child and reveals a wood label on hover/focus. Pure
// CSS: the bubble shows on :hover / :focus-within of the wrapper — no client JS.
func Tooltip(props TooltipProps, child g.Node) g.Node {
	cls := "tooltip"
	if props.Position == "bottom" {
		cls = "tooltip tooltip-bottom"
	}
	return h.Span(
		h.Class(cls),
		child,
		h.Span(h.Class("tooltip-bubble"), h.Role("tooltip"), g.Text(props.Label)),
	)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestTooltip -v`
Expected: PASS (both subtests).

- [ ] **Step 5: Append the Tooltip CSS to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`:
```css

/* ── Tooltip — wood label revealed on hover/focus (pure CSS) ──────────── */
.tooltip { position: relative; display: inline-flex; }
.tooltip-bubble {
  position: absolute; left: 50%; transform: translateX(-50%);
  bottom: 100%; margin-bottom: 8px; z-index: 30; white-space: nowrap;
  font-family: var(--font-mono); font-size: 11px; letter-spacing: .02em;
  color: var(--chrome-fg); background-color: var(--chrome);
  background-image: var(--wood-planks), var(--grain-warm); background-size: auto, 4px 4px;
  border: 2px solid var(--outline-2); box-shadow: var(--bevel-up); padding: 5px 9px;
  opacity: 0; visibility: hidden; pointer-events: none;
}
.tooltip-bottom .tooltip-bubble { bottom: auto; top: 100%; margin-bottom: 0; margin-top: 8px; }
.tooltip:hover .tooltip-bubble,
.tooltip:focus-within .tooltip-bubble { opacity: 1; visibility: visible; }
```

- [ ] **Step 6: Verify build + suite**

Run:
```bash
CGO_ENABLED=0 go build ./... && go test ./internal/web/assets/ && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
```
Expected: build clean, assets token test passes, full suite green.

- [ ] **Step 7: Commit**

```bash
git add internal/ui/tooltip.go internal/ui/tooltip_test.go internal/web/assets/static/basm.css
git commit -m "$(printf 'feat(ui): add Tooltip atom\n\nui.Tooltip(TooltipProps{Label,Position}, child) — wood label revealed on\nhover/focus, pure CSS (no JS). top/bottom positions.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 5: `ui.Skeleton` atom

A carved parchment loading placeholder with a sliding sheen (`@keyframes
sk-sheen`, paused under `prefers-reduced-motion`). Variants line/block/avatar,
with optional per-instance dimension overrides.

**Files:**
- Create: `internal/ui/skeleton.go`, `internal/ui/skeleton_test.go`
- Modify: `internal/web/assets/static/basm.css` (append)

- [ ] **Step 1: Write the failing test**

Create `internal/ui/skeleton_test.go`:
```go
package ui_test

import (
	"testing"

	"github.com/alexradunet/balaur/internal/ui"
)

func TestSkeleton(t *testing.T) {
	if got := render(t, ui.Skeleton(ui.SkeletonProps{})); got != `<span class="skeleton skeleton-line" aria-hidden="true"></span>` {
		t.Errorf("line default: %s", got)
	}
	if got := render(t, ui.Skeleton(ui.SkeletonProps{Variant: "block"})); got != `<span class="skeleton skeleton-block" aria-hidden="true"></span>` {
		t.Errorf("block: %s", got)
	}
	if got := render(t, ui.Skeleton(ui.SkeletonProps{Variant: "avatar", Size: "54px"})); got != `<span class="skeleton skeleton-avatar" aria-hidden="true" style="--sk-w:54px;--sk-h:54px"></span>` {
		t.Errorf("avatar+size: %s", got)
	}
	if got := render(t, ui.SkeletonLine("60%")); got != `<span class="skeleton skeleton-line" aria-hidden="true" style="--sk-w:60%"></span>` {
		t.Errorf("SkeletonLine width: %s", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/ -run TestSkeleton -v`
Expected: FAIL — `undefined: ui.Skeleton` / `ui.SkeletonProps` / `ui.SkeletonLine`.

- [ ] **Step 3: Implement the atom**

Create `internal/ui/skeleton.go`:
```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// SkeletonProps configures a Skeleton loading placeholder. Variant defaults to
// "line". Width/Height (line/block) and Size (avatar square) are optional CSS
// length overrides ("100%", "120px"); zero-values use the variant's CSS default.
type SkeletonProps struct {
	Variant string // "line" (default), "block", "avatar"
	Width   string
	Height  string
	Size    string
}

// Skeleton renders a carved parchment loading placeholder with a sliding sheen
// (.skeleton + .skeleton-<variant>, animated by @keyframes sk-sheen). Purely
// decorative: aria-hidden, no children. Only genuine per-instance dimension
// overrides are emitted as the --sk-w / --sk-h custom properties.
func Skeleton(p SkeletonProps) g.Node {
	variant := p.Variant
	if variant == "" {
		variant = "line"
	}
	attrs := []g.Node{
		h.Class("skeleton skeleton-" + variant),
		g.Attr("aria-hidden", "true"),
	}
	if style := skeletonStyle(variant, p); style != "" {
		attrs = append(attrs, h.Style(style))
	}
	return h.Span(attrs...)
}

// skeletonStyle builds the inline custom-property override, or "" when the
// caller relies on the variant defaults. For "avatar", Size drives both axes.
func skeletonStyle(variant string, p SkeletonProps) string {
	if variant == "avatar" {
		if p.Size != "" {
			return "--sk-w:" + p.Size + ";--sk-h:" + p.Size
		}
		return ""
	}
	var s string
	if p.Width != "" {
		s = "--sk-w:" + p.Width
	}
	if p.Height != "" {
		if s != "" {
			s += ";"
		}
		s += "--sk-h:" + p.Height
	}
	return s
}

// SkeletonLine is the common case: a single text-line placeholder. width is an
// optional CSS length ("" keeps the 100% default).
func SkeletonLine(width string) g.Node {
	return Skeleton(SkeletonProps{Variant: "line", Width: width})
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/ -run TestSkeleton -v`
Expected: PASS (all 4 assertions).

- [ ] **Step 5: Append the Skeleton CSS to the end of basm.css**

Append exactly this at the very end of `internal/web/assets/static/basm.css`:
```css

/* ── Skeleton — carved parchment loading placeholder w/ sliding sheen ───
   The warm sheen uses --bevel-light (not a cold white) to stay on-brand.
   Variants: .skeleton-line (default), .skeleton-block, .skeleton-avatar. */
.skeleton {
  --sk-w: 100%; --sk-h: 13px; display: block; width: var(--sk-w); height: var(--sk-h);
  border: 2px solid var(--parch-edge); border-radius: var(--radius);
  background-color: var(--surface-2);
  background-image: linear-gradient(100deg, transparent 30%, var(--bevel-light) 50%, transparent 70%);
  background-size: 220% 100%; background-repeat: no-repeat;
  box-shadow: var(--bevel-in); animation: sk-sheen 1.25s linear infinite;
}
.skeleton-block { --sk-h: 64px; }
.skeleton-avatar { --sk-w: 48px; --sk-h: 48px; }
@keyframes sk-sheen { from { background-position: 200% 0; } to { background-position: -100% 0; } }
@media (prefers-reduced-motion: reduce) { .skeleton { animation: none; } }
```

- [ ] **Step 6: Verify build + suite**

Run:
```bash
CGO_ENABLED=0 go build ./... && go test ./internal/web/assets/ && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
```
Expected: build clean, assets token test passes, full suite green.

- [ ] **Step 7: Commit**

```bash
git add internal/ui/skeleton.go internal/ui/skeleton_test.go internal/web/assets/static/basm.css
git commit -m "$(printf 'feat(ui): add Skeleton atom\n\nui.Skeleton(SkeletonProps{Variant,Width,Height,Size}) + ui.SkeletonLine — a\ncarved parchment loading placeholder with a sliding sheen (@keyframes\nsk-sheen, paused under prefers-reduced-motion). line/block/avatar variants.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 6: Showcase the display atoms in `/storybook`

**Files:**
- Modify: `internal/feature/storybook/storybook.go`, `storybook_test.go`

- [ ] **Step 1: Extend the test assertions (failing first)**

In `internal/feature/storybook/storybook_test.go`, add these entries to the
`want` slice in `TestBodyRendersAtoms` (after the existing `class="folk-band"`
line, inside the slice):
```go
		`class="badge badge-gold"`,
		`class="alert alert-warn"`,
		`class="tooltip"`,
		`class="skeleton skeleton-line"`,
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/feature/storybook/ -v`
Expected: FAIL — the new substrings are not yet rendered.

- [ ] **Step 3: Add the sections to the gallery**

In `internal/feature/storybook/storybook.go`, inside `Body()`, add these sections
just before the closing `)` of the `h.Div(h.Class("sb"), ...)` call (after the
`section("Separators", ...)` block):
```go
		section("Badges",
			ui.Badge(ui.BadgeProps{}, g.Text("3")),
			ui.Badge(ui.BadgeProps{Tone: ui.BadgeEmber}, g.Text("9")),
			ui.Badge(ui.BadgeProps{Tone: ui.BadgeTeal}, g.Text("new")),
			ui.Badge(ui.BadgeProps{Tone: ui.BadgeWood}, g.Text("draft")),
			ui.Badge(ui.BadgeProps{Tone: ui.BadgeGold, Dot: true}),
			ui.Badge(ui.BadgeProps{Tone: ui.BadgeEmber, Dot: true}),
		),
		section("Alerts",
			ui.Alert(ui.AlertProps{Tone: "info", Title: "Heads up"}, g.Text("Your data stays on the box unless you switch models yourself.")),
			ui.Alert(ui.AlertProps{Tone: "warn", Title: "Caution"}, g.Text("This action enables OS access for the session.")),
			ui.Alert(ui.AlertProps{Tone: "danger", Title: "Stop"}, g.Text("This will permanently delete the record.")),
		),
		section("Tooltip",
			ui.Tooltip(ui.TooltipProps{Label: "Keep it"}, ui.Button(ui.ButtonProps{Variant: "ghost"}, g.Text("hover me"))),
		),
		section("Skeletons",
			ui.SkeletonLine("100%"),
			ui.SkeletonLine("60%"),
			ui.Skeleton(ui.SkeletonProps{Variant: "block"}),
			ui.Skeleton(ui.SkeletonProps{Variant: "avatar"}),
		),
```

- [ ] **Step 4: Run to verify it passes + full suite + view**

Run:
```bash
go test ./internal/feature/storybook/ -v 2>&1 | tail -6
go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./... && go vet ./...
```
Expected: storybook test PASS, full suite green, build+vet clean. Optionally view
`/storybook` and **eyeball the Badge pills under BOTH light and dark themes**
(toggle ◑) — the gold/ember pill labels are the one flagged contrast risk.

- [ ] **Step 5: Commit**

```bash
git add internal/feature/storybook/storybook.go internal/feature/storybook/storybook_test.go
git commit -m "$(printf 'feat(storybook): showcase Badge, Alert, Tooltip, Skeleton\n\nAdd gallery sections for the four display atoms.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification

- [ ] Run the whole suite + build + vet + diff check:
```bash
go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check
```
Expected: all green, no whitespace errors.

- [ ] Confirm no raw hex crept into the new CSS:
```bash
tail -90 internal/web/assets/static/basm.css | grep -nE "#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX in the appended blocks"
```
Expected: `NO RAW HEX…` (every color is a `var(--token)`).

## What this slice delivers / what's next

**Delivered:** four display atoms (`Badge`, `Alert`, `Tooltip`, `Skeleton`) with
tokenized CSS, the storybook gallery layout CSS, and a fuller `/storybook`.

**Known caveat (for the storybook eyeball):** the gold/ember Badge pill labels
use `--ink` (near-black) on a `light-dark()` accent that darkens in light mode —
verify legibility; if it fails, lighten the light-mode `--badge-fg` for those two
tones.

**Next (02b sub-plans):** forms (`TextField`, `Select`, `Toggle`), nav
(`Tabs`, `Breadcrumb`, `Pagination`), `List`/`ListItem`, and the `SectionLabel`/
`ScreenTitle` text helpers.
