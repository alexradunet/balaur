# Storybook Source-of-Truth — Slice 1: Responsive Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the storybook shell responsive — below 920px the sidebar becomes an off-canvas drawer toggled by a hamburger topbar (with a backdrop) — and fix the component wrap/reflow offenders.

**Architecture:** Tightly-coupled change across three files: new tokenized CSS (the `≤920px`/`≤520px` breakpoints + `.sb-topbar`/`.sb-burger`/`.sb-backdrop` chrome + component fixes), `SidebarPage` markup (emit the topbar + backdrop), and a `basmToggleNav()` vanilla handler in `basm.js`. Server-rendered; the drawer is a CSS-class toggle.

**Tech Stack:** vanilla `basm.css` + `basm.js` (no build step), Go (gomponents shell).

**Spec:** `docs/superpowers/specs/2026-06-16-storybook-source-of-truth-design.md` (section 3).

**Conventions:** New CSS appends at the END of `basm.css`, tokenized (`var(--token)`, no raw hex; raw `rgba()` allowed for the backdrop scrim/drawer shadow, matching existing usage). After the task: `go test ./...`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`. If `git status` shows a non-task file modified, `git checkout --` it.

Verified facts: `.sb-root { display: grid; grid-template-columns: 232px 1fr; min-height: 100vh }`, `.sb-side { …position: sticky; top: 0; overflow-y: auto… }`, `.sb-canvas { height: 100vh; overflow-y: auto; padding: 0 5vw 96px }` all exist; `.sb-topbar`/`.sb-backdrop`/`.sb-burger` do NOT. `SidebarPage` (`internal/ui/shell/sidebar.go:64`) emits `.sb-root > [p.Sidebar] + main.sb-canvas > (header.sb-crumb + p.Body)`. The sidebar component renders `<aside class="sb-side">`. `basm.js` has no nav toggle. Token `--ease-crisp` exists. The crest is `/static/crest.png`.

---

## Task 1: Responsive shell + component CSS

**Files:** Modify `internal/web/assets/static/basm.css`.

- [ ] **Step 1: Append the responsive shell + chrome + component fixes** at the END of `basm.css`:
```css

/* ── Storybook responsive shell — off-canvas drawer ≤920px ──────────────── */
.sb-topbar { display: none; }
.sb-backdrop { display: none; }
@media (max-width: 920px) {
  .sb-root { grid-template-columns: 1fr; }
  .sb-topbar {
    display: flex; align-items: center; gap: 12px; position: sticky; top: 0; z-index: 50;
    padding: 10px 16px; border-bottom: 2px solid var(--outline-2);
    background-color: var(--chrome); background-image: var(--wood-planks), var(--grain-warm); background-size: auto, 4px 4px;
  }
  .sb-burger {
    font-family: var(--font-mono); font-size: 18px; line-height: 1; cursor: pointer;
    background: none; border: 1px solid var(--chrome-fg); color: var(--gold); padding: 5px 10px; border-radius: var(--radius);
  }
  .sb-topbar-brand { font-family: var(--font-pixel); font-size: 13px; letter-spacing: .08em; text-transform: uppercase; color: var(--gold); }
  .sb-topbar .crest { width: 28px; height: 28px; image-rendering: pixelated; }
  .sb-side {
    position: fixed; inset: 0 auto 0 0; width: min(86vw, 322px); z-index: 60;
    transform: translateX(-104%); transition: transform .2s var(--ease-crisp); box-shadow: 6px 0 0 rgba(0, 0, 0, .4);
  }
  .sb-side.is-open { transform: none; }
  .sb-backdrop.is-open { display: block; position: fixed; inset: 0; z-index: 55; background: rgba(8, 5, 2, .62); }
  .sb-canvas { height: auto; padding: 26px 16px 80px; }
}
@media (max-width: 520px) { .sb-canvas { padding: 20px 10px 72px; } }
@media (prefers-reduced-motion: reduce) { .sb-side { transition: none; } }

/* ── Responsive component wrap/reflow fixes ─────────────────────────────── */
.dayentry-content { min-width: 0; }
.dayentry-title { overflow-wrap: anywhere; }
@media (max-width: 480px) { .dayentry { grid-template-columns: 44px 18px 1fr; column-gap: 9px; } }
@media (max-width: 520px) { .composer-top { grid-template-columns: 1fr auto; } }
.list-title, .list-sub { overflow-wrap: anywhere; }
```

- [ ] **Step 2: Verify CSS lands (build embeds it)**
```bash
cd /home/alex/Projects/balaur
CGO_ENABLED=0 go build ./... && echo BUILD_OK
tail -40 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
```
Expected: build clean, NO RAW HEX (the rgba scrims/shadow are fine — they're not hex).

(No commit yet — Task 2/3 add the markup + JS that use these classes; commit together at the end of Task 3, OR commit per-task. Commit now is fine too; the classes are inert without markup. Commit:)
```bash
git add internal/web/assets/static/basm.css
git commit -m "$(printf 'feat(css): responsive storybook shell (off-canvas drawer) + component wrap fixes\n\nAdds the ≤920px off-canvas .sb-side drawer + .sb-topbar/.sb-burger/.sb-backdrop\nchrome + ≤520px padding + reduced-motion; wrap-protects .dayentry/.composer-top/\n.list-item. Markup + toggle JS follow.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: SidebarPage topbar + backdrop markup

**Files:** Modify `internal/ui/shell/sidebar.go`, `internal/ui/shell/shell_test.go` (or the storybook handler test) for a light assertion.

- [ ] **Step 1: Write the failing test**

`internal/ui/shell/sidebar.go`'s `SidebarPage` is rendered for the storybook. Add a test that the rendered page includes the mobile topbar + backdrop. If `internal/ui/shell/sidebar_test.go` exists, add to it; else create it:
```go
package shell_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/ui/shell"
)

func TestSidebarPageResponsiveChrome(t *testing.T) {
	var b strings.Builder
	page := shell.SidebarPage(shell.SidebarPageProps{Title: "X", Sidebar: g.Text("SIDE"), Body: g.Text("BODY")})
	if err := page.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	got := b.String()
	for _, want := range []string{
		`<header class="sb-topbar">`,
		`<button class="sb-burger" type="button" onclick="basmToggleNav()"`,
		`<div class="sb-backdrop" onclick="basmToggleNav()"></div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("sidebar page missing %q in: %s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/shell/ -run TestSidebarPageResponsiveChrome -v` — Expected: FAIL.

- [ ] **Step 3: Add the markup**

In `internal/ui/shell/sidebar.go`, change the `SidebarPage` body so the `.sb-root` has the topbar as its first child and a backdrop as its last child:
```go
		h.Body(
			h.Div(h.Class("sb-root"),
				h.Header(h.Class("sb-topbar"),
					h.Button(h.Class("sb-burger"), h.Type("button"), g.Attr("onclick", "basmToggleNav()"),
						h.Aria("label", "Open navigation"), h.Aria("expanded", "false"), g.Text("☰")),
					h.Img(h.Class("crest"), h.Src("/static/crest.png"), h.Alt(""), g.Attr("decoding", "async")),
					h.Span(h.Class("sb-topbar-brand"), g.Text("Balaur")),
				),
				p.Sidebar,
				h.Main(h.Class("sb-canvas"),
					h.Header(h.Class("sb-crumb"), g.Text(crumb)),
					p.Body,
				),
				h.Div(h.Class("sb-backdrop"), g.Attr("onclick", "basmToggleNav()")),
			),
		),
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/shell/ -run TestSidebarPageResponsiveChrome -v` — Expected: PASS. Also run the existing `TestPage` (the app shell) — unaffected.

- [ ] **Step 5: Verify + commit**
```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/shell/ && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
git status --short
```
Expected: PASS, suite green, build clean. Stage only the two files, then:
```bash
git add internal/ui/shell/sidebar.go internal/ui/shell/sidebar_test.go
git commit -m "$(printf 'feat(shell): SidebarPage mobile topbar + backdrop for the off-canvas drawer\n\nEmits a .sb-topbar (hamburger + crest + brand) and a .sb-backdrop, both wired to\nbasmToggleNav(); shown only ≤920px via CSS. aria-expanded/label on the burger.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 3: `basmToggleNav` JS

**Files:** Modify `internal/web/assets/static/basm.js`.

- [ ] **Step 1: Add the toggle**

In `internal/web/assets/static/basm.js`, append near the other UI helpers (e.g. after the palette/theme block):
```js

// ── Storybook off-canvas nav drawer ────────────────────────────────
// The sidebar (.sb-side) is fixed off-screen ≤920px; the .sb-topbar burger
// and the .sb-backdrop both toggle it. Closes on backdrop click and on any
// nav-item click (so navigating dismisses the drawer).
window.basmToggleNav = function () {
  var open = document.documentElement.classList.toggle('sb-nav-open');
  document.querySelectorAll('.sb-side, .sb-backdrop').forEach(function (el) { el.classList.toggle('is-open', open); });
  document.querySelectorAll('.sb-burger').forEach(function (b) { b.setAttribute('aria-expanded', open ? 'true' : 'false'); });
};
document.addEventListener('click', function (e) {
  if (e.target.closest('.sb-side .sb-nav-item') && document.documentElement.classList.contains('sb-nav-open')) {
    window.basmToggleNav();
  }
});
```

- [ ] **Step 2: Verify + commit**
```bash
cd /home/alex/Projects/balaur
node --check internal/web/assets/static/basm.js && echo "JS PARSES" || echo "(node unavailable)"
CGO_ENABLED=0 go build ./... && echo BUILD_OK
git status --short
```
Expected: JS parses, build clean. Stage only basm.js:
```bash
git add internal/web/assets/static/basm.js
git commit -m "$(printf 'feat(js): basmToggleNav — open/close the storybook off-canvas drawer\n\nToggles is-open on .sb-side + .sb-backdrop (burger + backdrop), syncs\naria-expanded, and closes on nav-item click.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification (controller — visual, mobile viewports)

- [ ] `go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check` — green.
- [ ] Build + serve; screenshot `/storybook/button` at **360px** and **768px** widths (CDP `Emulation.setDeviceMetricsOverride` or a narrow `--window-size`): the sidebar is hidden, the `.sb-topbar` hamburger shows, the canvas is full-width with no horizontal overflow.
- [ ] Drive the drawer: click the burger → `.sb-side` slides in + backdrop appears; click the backdrop → closes. (CDP `Runtime.evaluate('basmToggleNav()')` then screenshot, or click.)
- [ ] Screenshot a component page known to be tight (`/storybook/dayentry`, `/storybook/composer`) at 360px — no overflow; long titles wrap.
- [ ] Desktop (≥1024px) unchanged: `.sb-topbar` hidden, the 2-col grid intact.

## What this delivers / what's next

**Delivered:** a responsive storybook — off-canvas drawer on mobile + component wrap/reflow fixes.

**Next (Slice 2):** the enriched Story model (`Blurb`, `Variants`, `Props`, `Dos`, `Donts`) + the per-component `storybook.Page(s)` render (blurb → variant tiles → props table → Do/Don't) + the page CSS, migrating the Atoms group as proof.
