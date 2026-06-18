# Plan 100 — Mobile rail reveal: make the domain rail reachable on narrow viewports (supersedes 091)

- **Status:** TODO (expanded to step detail against the post-099 tree)
- **Priority:** P2
- **Effort:** S–M
- **Risk:** LOW–MED (CSS-led; reuses the proven `basmToggleNav` off-canvas drawer
  with NO new JS; the markup/wiring is Go-test-verifiable, the visual slide needs
  an owner eyeball — no browser in the executor/review loop)
- **Planned against commit:** `eab6723` (plans 098 `176ed55` + 099 `6d64c6d` merged)
- **Date:** 2026-06-18
- **Depends on:** **098** (panel + ≤720px overlay) and **099** (top-level rail) — both DONE
- **Supersedes:** **091** (its remaining goals are re-scoped here / deferred below)

> **Executor: read this whole file first.** Plans 098 + 099 shipped the
> right-panel canvas and in-panel tabs. The panel already works on mobile (098
> made it a slide-in overlay ≤720px). The remaining gap: the **left domain rail
> is `display:none` ≤720px with no way to open it**, so on a phone you cannot
> reach Quests/Life/Knowledge/Skills/Settings at all. This plan adds a burger +
> off-canvas rail drawer, reusing the EXISTING proven pattern. Run the drift
> check (Step 0). Commit per step. Touch only in-scope files.

---

## Why this change (and what is deferred)

The canvas program (098/099) is complete on desktop — the primary surface for a
loopback-first personal app. On a narrow viewport the panel slides in (098), but
the **rail is unreachable**: the live `html.app .app-shell` collapses to one
column and sets `.sb-side { display: none }` with **no burger and no drawer**
wired to the live shell (the accessible `.topnav-drawer`/`basmToggleTopnav` JS
exists but is bound to the *retired* legacy topbar, not `html.app`). So mobile
navigation is broken — you can only summon artifacts by *asking* Balaur.

This plan makes the rail reachable on mobile by reusing the **already-shipped,
already-proven** off-canvas pattern the storybook shell uses:
`basmToggleNav()` + `.sb-side.is-open` + `.sb-backdrop.is-open` + an `.sb-burger`
(`internal/web/assets/static/basm.js:218-227`, CSS at `basm.css:2950-2970`). No
new JavaScript — only markup in `ChatShell` and CSS scoped to `html.app` ≤720px.

### Explicitly DEFERRED (out of scope — a future 091-successor cycle)

These were bundled into the old 091 / the 100 stub but are orthogonal, lower
value, and (the visual/a11y ones) unverifiable without a browser. Do **not** do
them here; they are noted so the index stays honest:

- **Panel focus-trap a11y parity.** 098's mobile panel drawer uses a class toggle
  + click-outside + Escape (functional) but lacks `inert`-when-closed + Tab focus
  trap. A refinement, not a blocker.
- **Sidebar chrome.** Head/model switchers + a recap affordance in the rail
  (091's core). A feature-add, orthogonal to responsiveness.
- **Owner-resizable panel width.** The orphaned `.dock-grip` + `--sidebar-w`
  drag-resize JS (`basm.js:191-212`) could be revived for the panel or retired as
  dead code — a separate decision.

---

## Step 0 — Drift check

```sh
git rev-parse --short HEAD                 # expect eab6723 (098+099 merged)
git grep -n "func ChatShell" internal/ui/shell/chatshell.go
git grep -n "window.basmToggleNav" internal/web/assets/static/basm.js          # the reused toggle
git grep -n "html.app .app-shell .sb-side { display: none; }" internal/web/assets/static/basm.css  # the gap to fix
git grep -n "func SidebarPage" internal/ui/shell/sidebar.go                    # the burger/backdrop markup to mirror
```
All must be present. If `basmToggleNav` is gone or the `html.app … .sb-side {
display: none }` rule is already replaced, STOP — someone changed this area.

Baseline at `eab6723`: `gofmt -l` clean, `go vet ./...` ok, `go test ./...` ok,
`CGO_ENABLED=0 go build ./...` ok. Confirm green first.

> Sandbox note: TLS-intercepting sandbox → GOPROXY shim per `docs/hyperagent-sandbox.md`.

---

## Current state (real excerpts at `eab6723`)

### `internal/ui/shell/chatshell.go` — the live shell (no topbar/burger)

```go
func ChatShell(p ChatShellProps) g.Node {
	return g.Group([]g.Node{
		g.Raw("<!doctype html>"),
		h.HTML(h.Lang("en"), h.Class("app"),
			h.Head(pageHead(), h.TitleEl(g.Text(p.Title+" · Balaur"))),
			h.Body(
				h.A(h.Class("skip-link"), h.Href("#chat"), g.Text("Skip to content")),
				h.Div(h.Class("app-shell"),
					p.Sidebar,
					h.Aside(h.ID("dock"), h.Class("app-dock"), p.Dock),
					h.Aside(h.ID("panel"), h.Class("app-panel"), p.Panel),
				),
			),
		),
	})
}
```

### `internal/ui/shell/sidebar.go` — `SidebarPage` topbar + backdrop to MIRROR

```go
		h.Header(h.Class("sb-topbar"),
			h.Button(h.Class("sb-burger"), h.Type("button"), g.Attr("onclick", "basmToggleNav()"),
				h.Aria("label", "Open navigation"), h.Aria("expanded", "false"), g.Text("☰")),
			h.Img(h.Class("crest"), h.Src("/static/crest.png"), h.Alt(""), g.Attr("decoding", "async")),
			h.Span(h.Class("sb-topbar-brand"), g.Text("Balaur")),
		),
		// … sidebar, canvas …
		h.Div(h.Class("sb-backdrop"), g.Attr("onclick", "basmToggleNav()")),
```

### `internal/web/assets/static/basm.js:218-227` — the reused toggle (NO change)

```js
window.basmToggleNav = function () {
  var open = document.documentElement.classList.toggle('sb-nav-open');
  document.querySelectorAll('.sb-side, .sb-backdrop').forEach(function (el) { el.classList.toggle('is-open', open); });
  document.querySelectorAll('.sb-burger').forEach(function (b) { b.setAttribute('aria-expanded', open ? 'true' : 'false'); });
};
document.addEventListener('click', function (e) {
  if (e.target.closest('.sb-side .sb-nav-item') && document.documentElement.classList.contains('sb-nav-open')) {
    window.basmToggleNav();   // navigating dismisses the drawer
  }
});
```
This already toggles `.sb-side`/`.sb-backdrop`/`.sb-burger` globally and dismisses
on a nav-item click — exactly what the live rail needs. **No JS edit required.**

### `internal/web/assets/static/basm.css` — the storybook off-canvas (the pattern) + the live gap

```css
/* storybook .sb-root drawer (the proven pattern), ~2950-2970, ≤920px: */
.sb-topbar { display: none; }
.sb-backdrop { display: none; }
@media (max-width: 920px) {
  .sb-topbar { /* shown: fixed top bar with burger+crest+brand */ }
  .sb-burger { /* 44px touch target */ }
  .sb-root .sb-side { position: fixed; inset: 0 auto 0 0; width: min(86vw, 322px); z-index: var(--z-drawer);
    transform: translateX(-104%); transition: …; }
  html.sb-nav-open .sb-root .sb-side { transform: none; }      /* (verify exact selector in file) */
  .sb-backdrop.is-open { display: block; position: fixed; inset: 0; z-index: var(--z-scrim); background: rgba(8,5,2,.62); }
}

/* the live app gap, ~3377-3389, ≤720px: */
@media (max-width: 720px) {
  html.app .app-shell { grid-template-columns: 1fr; }
  html.app .app-shell .sb-side { display: none; }   /* ← the gap: rail vanishes, no reveal */
  html.app #panel.app-panel { /* slide-in overlay (098) */ }
  html.app.panel-open #panel.app-panel { transform: translateX(0); }
}
```
Read the real `.sb-side` off-canvas rules (~2966) and the live ≤720px block
(~3378) in full before editing — match the exact selectors and tokens the file
already uses (`--z-drawer`, `--z-scrim`, the slide transition).

---

## Steps

### Step 1 — `ChatShell` gains a mobile topbar + backdrop

In `internal/ui/shell/chatshell.go`, add an `.app-topbar` (burger + crest +
brand) as the FIRST child of `<body>` (before `.app-shell`) and a `.sb-backdrop`
as the LAST child. Mirror `SidebarPage`'s markup so `basmToggleNav` finds the
same classes. Reuse the `.sb-burger`/`.sb-topbar`/`.sb-backdrop` classes (so the
existing CSS + JS apply) but gate visibility to the app via an `.app-topbar`
wrapper.

```go
			h.Body(
				h.A(h.Class("skip-link"), h.Href("#chat"), g.Text("Skip to content")),
				// Mobile-only top bar: burger reveals the rail drawer (≤720px).
				h.Header(h.Class("app-topbar"),
					h.Button(h.Class("sb-burger"), h.Type("button"), g.Attr("onclick", "basmToggleNav()"),
						h.Aria("label", "Open navigation"), h.Aria("expanded", "false"), g.Text("☰")),
					h.Img(h.Class("crest"), h.Src("/static/crest.png"), h.Alt(""), g.Attr("decoding", "async")),
					h.Span(h.Class("sb-topbar-brand"), g.Text("Balaur")),
				),
				h.Div(h.Class("app-shell"),
					p.Sidebar,
					h.Aside(h.ID("dock"), h.Class("app-dock"), p.Dock),
					h.Aside(h.ID("panel"), h.Class("app-panel"), p.Panel),
				),
				h.Div(h.Class("sb-backdrop"), g.Attr("onclick", "basmToggleNav()")),
			),
```
No new import (uses `h`/`g` already imported).

**Verify:** `CGO_ENABLED=0 go build ./internal/ui/shell/...`.

### Step 2 — CSS: hide the topbar on desktop; reveal the rail drawer ≤720px

In `internal/web/assets/static/basm.css`:

1. Desktop default (near the `html.app` shell rules, ~3340): `.app-topbar { display: none; }` (it only exists for mobile).
2. In the **live** `@media (max-width: 720px)` block (~3378), REPLACE
   `html.app .app-shell .sb-side { display: none; }` with the off-canvas drawer
   (model it on the storybook `.sb-side` rules at ~2966 — same tokens/transition):
   ```css
   @media (max-width: 720px) {
     html.app .app-shell { grid-template-columns: 1fr; }

     /* Mobile top bar (burger). */
     html.app .app-topbar {
       display: flex; align-items: center; gap: var(--space-2);
       height: 56px; padding: 0 var(--space-3);
       background-color: var(--chrome); background-image: var(--wood-planks), var(--grain-warm);
       background-size: auto, 4px 4px; border-bottom: 2px solid var(--outline-2);
       position: sticky; top: 0; z-index: var(--z-sticky);
     }
     html.app .app-topbar .sb-burger {
       min-width: 44px; min-height: 44px; display: inline-flex; align-items: center; justify-content: center;
       background: none; border: 1px solid var(--chrome-fg); color: var(--gold);
       font-size: 18px; line-height: 1; cursor: pointer;
     }
     html.app .app-topbar .sb-topbar-brand { font-family: var(--font-pixel); font-size: 13px; letter-spacing: .08em; text-transform: uppercase; color: var(--gold); }
     html.app .app-topbar .crest { width: 28px; height: 28px; image-rendering: pixelated; }

     /* Rail becomes an off-canvas drawer (was display:none). */
     html.app .sb-side {
       display: flex; position: fixed; inset: 56px auto 0 0; width: min(86vw, 322px);
       z-index: var(--z-drawer); transform: translateX(-104%);
       transition: transform .2s var(--ease-crisp); box-shadow: 6px 0 0 rgba(0,0,0,.4);
     }
     html.app.sb-nav-open .sb-side { transform: none; }
     html.app .sb-backdrop.is-open { display: block; position: fixed; inset: 0; z-index: var(--z-scrim); background: rgba(8,5,2,.62); }

     /* Panel overlay (098) — unchanged; keep the existing rules in this block. */
     html.app #panel.app-panel { /* … existing 098 rules … */ }
     html.app.panel-open #panel.app-panel { transform: translateX(0); }
     html.app.panel-open::after { /* … existing 098 scrim … */ }
   }
   ```
   Keep the existing 098 panel rules in this block intact — only the `.sb-side`
   line changes and the `.app-topbar`/drawer rules are added.
3. Desktop: ensure `.sb-backdrop` stays hidden (the base `.sb-backdrop { display:
   none }` at ~2951 already covers it globally — confirm; `.is-open` only applies
   inside the ≤720px block).

> Use the EXACT token names the file already uses (`--chrome`, `--wood-planks`,
> `--grain-warm`, `--outline-2`, `--ease-crisp`, `--z-drawer`, `--z-scrim`,
> `--z-sticky`, `--gold`, `--chrome-fg` — all confirmed present). If any does not
> exist, grep for the one the storybook `.sb-topbar`/`.sb-side` block uses
> (~2954-2970) and match it.
>
> **Specificity trap:** the storybook reveal rule is `.sb-side.is-open { transform:
> none }` (specificity 0,2,0). Your new base rule `html.app .sb-side` is 0,2,1 —
> HIGHER — so a bare `.sb-side.is-open` would lose and the drawer would not slide
> in. The reveal selector MUST out-specify the base: use
> `html.app.sb-nav-open .sb-side { transform: none; }` (0,3,1) as shown above, or
> `html.app .sb-side.is-open` (0,3,1). `basmToggleNav` sets BOTH `sb-nav-open` on
> `<html>` AND `.is-open` on `.sb-side`, so either selector fires.

**Verify:** `CGO_ENABLED=0 go build ./...` (CSS is embedded). No automated visual
check exists — see the eyeball note in Done criteria.

### Step 3 — Tests (the verifiable part)

- **`internal/web/home_test.go` `TestHomeFullChat`**: add `ExpectedContent`
  assertions that `GET /` now renders the mobile chrome (present in markup at all
  widths; CSS hides it on desktop): `class="app-topbar"`, `class="sb-burger"`,
  `onclick="basmToggleNav()"`, and `class="sb-backdrop"`. Keep all existing
  assertions (the rail entries, `id="panel"`, etc.).
- No JS test (basm.js is reused unchanged; the dismiss-on-nav-item handler
  already exists). No new Go logic to unit-test.

**Verify (full gate):**
```sh
gofmt -l .                       # empty
go vet ./...                     # exit 0
CGO_ENABLED=0 go build ./...     # exit 0
go test ./...                    # no FAIL / panic
git diff --check                 # clean
```

### Step 4 — Docs

- **`internal/self/knowledge.md`**: the shell description (the three-column
  `.app-shell` paragraph) — add one sentence: on narrow viewports (≤720px) the
  rail collapses to an off-canvas drawer reached via the `.app-topbar` burger
  (`basmToggleNav`), and the panel becomes a slide-in overlay (098).
- **`DESIGN.md`**: only if it describes the responsive behavior of the shell;
  add the mobile-rail-drawer note if so, else no change (say so).
- No storybook change — `basmToggleNav` and the `.sb-*` classes are already
  storied via `SidebarPage`/`sidebarStory`.
- Tour anchor: `go test ./... -run Tours` — fix if a moved line breaks one.

**Verify:** `go test ./... -run Tours` + the full gate.

---

## Files in scope

- `internal/ui/shell/chatshell.go` (add `.app-topbar` + `.sb-backdrop`)
- `internal/web/assets/static/basm.css` (desktop-hide `.app-topbar`; ≤720px rail
  drawer + topbar, replacing the `.sb-side { display:none }` line)
- `internal/web/home_test.go` (assert the mobile chrome renders)
- `internal/self/knowledge.md` (one sentence), `DESIGN.md` (only if it has shell
  responsive prose), a tour anchor if needed

## Files explicitly OUT of scope

- `internal/web/assets/static/basm.js` — NO change (reuse `basmToggleNav`).
- `chat.Panel` / the panel mechanism / `/ui/show` / `/ui/panel` — unchanged.
- The storybook shell (`SidebarPage`) and the legacy `.topnav-drawer` — unchanged.
- The DEFERRED items above (panel focus-trap a11y, switcher chrome, panel resize)
  — do NOT start them.

## Done criteria (machine-checkable)

```sh
gofmt -l .                                   # empty
go vet ./...                                 # exit 0
CGO_ENABLED=0 go build ./...                 # exit 0
go test ./...                                # no FAIL / panic
git diff --check                             # clean
git grep -n 'class="app-topbar"' internal/ui/shell/chatshell.go              # present
git grep -n "html.app .app-topbar" internal/web/assets/static/basm.css       # present
git grep -nc "html.app .app-shell .sb-side { display: none; }" internal/web/assets/static/basm.css || echo "old display:none rule replaced (OK)"
```
**Owner eyeball (cannot be automated — no browser in the loop):** on a ≤720px
viewport, the burger opens the rail as a left drawer over a scrim; tapping a
domain navigates AND dismisses the drawer; the panel still slides in on summon;
desktop is unchanged (no topbar, three columns). Flag this clearly in the PR — the
markup/wiring is test-verified, the visual slide is not.

## Maintenance notes

- This reuses `basmToggleNav` (the simpler class-toggle drawer), NOT the
  accessible `basmToggleTopnav` (focus-trap/inert). That is deliberate scope: the
  rail drawer matches the storybook rail's existing behavior. Upgrading the rail
  AND panel drawers to full focus-trap a11y is the deferred refinement.
- Two `.sb-side` consumers now exist (storybook `.sb-root` ≤920px, live `html.app`
  ≤720px). They share the `.sb-side`/`.sb-backdrop`/`.sb-nav-open` contract and
  `basmToggleNav`; keep them in sync if that contract changes.

## Escape hatches — STOP and report

- If a token referenced above (`--ease-crisp`, `--wood-planks`, etc.) does not
  exist in `basm.css`, STOP and report rather than inventing one — match the real
  storybook `.sb-topbar`/`.sb-side` block.
- If adding `.app-topbar` as a `<body>` child (outside `.app-shell`) breaks the
  desktop `100dvh` layout (e.g. the shell overflows by the topbar height even
  though it is `display:none`), STOP and report — `display:none` should remove it
  from flow on desktop, but if not, note the overflow rather than hacking heights.
- If `TestHomeFullChat` or another home test asserts an exact body structure that
  the new topbar/backdrop breaks, update the assertion to include them (they are
  correct new markup) — but if that means gutting a meaningful assertion, STOP.
