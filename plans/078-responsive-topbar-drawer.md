# Plan 078: Give the product topbar a real mobile treatment with an accessible off-canvas nav

> **Executor instructions**: Follow step by step. Run every Verify and confirm before moving on. On a STOP condition, stop and report — do not improvise. When done, update the 078 row in plans/readme.md (add the row if it is not present yet, matching the existing column format).
>
> **Drift check (run first)**: `git diff --stat 12a2ff5..HEAD -- internal/ui/shell/shell.go internal/ui/shell/shell_test.go internal/web/assets/static/basm.css internal/web/assets/static/basm.js internal/feature/storybook/stories_navigation.go internal/feature/storybook/story.go` — if any in-scope file changed since this plan was written, compare the "Current state" excerpts below to the live code; on mismatch, STOP.

## Status
- **Priority**: P1
- **Effort**: M
- **Risk**: MED
- **Depends on**: none
- **Category**: responsiveness/a11y
- **Planned at**: commit `12a2ff5`, 2026-06-17

## Why this matters
The product topbar (`shell.Topbar`) lays out the brand, six domain links, and two theme buttons as flat siblings in a single **non-wrapping** flex row. Below ~720px that row is wider than the viewport: measured live at 390px the document scrolls horizontally (`document.scrollWidth` 632 vs `innerWidth` 390), so **Heads / Settings and the theme toggles run off-screen and become unreachable on a phone**. The only mobile rule today merely shrinks the nav gap and font (still overflows). Touch targets are also far below the 44px minimum (nav links ~18px tall, theme buttons ~21px). A tested off-canvas drawer pattern already ships in the SAME package for the storybook (`shell.SidebarPage` burger + `.sb-backdrop` + `.sb-side` translateX, toggled by `basmToggleNav`), so the lowest-risk fix reuses that one mechanism for the product topbar — but that drawer has a real a11y gap (offscreen links stay tabbable; no focus move, focus trap, or Escape) that must be fixed before it carries the product nav. This advances DESIGN.md's Hearthwood canon (square corners, wood chrome, no rounded/blur) and the SKILL.md rule that UI is composed from typed gomponents, not hand-rolled markup.

## Current state

### `internal/ui/shell/shell.go` — the product topbar (the thing to fix)
The topbar is a flat `<header class="topbar">` with brand + a non-wrapping `<nav>` of six links + two theme buttons as siblings (shell.go:74-100):
```go
func Topbar(active string) g.Node {
	return h.Header(h.Class("topbar"),
		h.A(h.Class("brand"), h.Href("/"),
			h.Img(h.Class("crest"), h.Src("/static/crest.png"), h.Alt(""), g.Attr("decoding", "async")),
			g.Text("Balaur"),
		),
		h.Nav(
			navLink("/focus/quests", "Quests", "quests", active),
			navLink("/focus/memory", "Knowledge", "knowledge", active),
			navLink("/focus/lifelog", "Life", "life", active),
			navLink("/focus/journal", "Journal", "journal", active),
			navLink("/focus/heads", "Heads", "heads", active),
			navLink("/focus/settings", "Settings", "settings", active),
		),
		h.Button(h.Class("theme-cycle"), h.Type("button"),
			g.Attr("onclick", "basmCycleTheme()"), ...),
		h.Button(h.Class("theme-toggle"), h.Type("button"),
			g.Attr("onclick", "basmToggleTheme()"), ...),
	)
}
```
`navLink` (shell.go:102-109) returns a bare `h.A` with optional `aria-current="page"`. gomponents idiom: `import g "maragu.dev/gomponents"` + `h "maragu.dev/gomponents/html"` (QUALIFIED `h.`); arbitrary attrs via `g.Attr`; SVG/raw via `g.El`/`g.Raw`. `internal/ui/shell` may NOT import `internal/feature/*`.

### `internal/ui/shell/sidebar.go` — the EXEMPLAR drawer markup (reuse the shape, do not import)
`SidebarPage` already emits the proven drawer structure (sidebar.go:82-95): a `.sb-topbar` with a `.sb-burger` button (`onclick="basmToggleNav()"`, `aria-label="Open navigation"`, `aria-expanded="false"`), the rail node, the canvas, and a closing `.sb-backdrop` (`onclick="basmToggleNav()"`):
```go
h.Header(h.Class("sb-topbar"),
	h.Button(h.Class("sb-burger"), h.Type("button"), g.Attr("onclick", "basmToggleNav()"),
		h.Aria("label", "Open navigation"), h.Aria("expanded", "false"), g.Text("☰")),
	...),
p.Sidebar,
h.Main(h.Class("sb-canvas"), ...),
h.Div(h.Class("sb-backdrop"), g.Attr("onclick", "basmToggleNav()")),
```
This is the markup shape to mirror in the product topbar — but as its OWN classes so the storybook drawer is never regressed.

### `internal/web/assets/static/basm.css` — topbar + theme + drawer rules
- `.topbar` (basm.css:295-309): `display:flex; align-items:center; gap:14px; height:62px; padding:0 6vw; ... z-index:5;` — **no `flex-wrap`, no overflow handling**.
- `.topbar nav` (basm.css:338-347): `{ margin-left:auto; display:flex; gap:18px; }`, links `font-size:12px; text-transform:uppercase; color:var(--chrome-fg);` hover `var(--gold)`. **Nav links are inline `<a>` — no padding, hit area ~18px tall.**
- `.theme-toggle, .theme-cycle` (basm.css:919-932): `font-size:13px; border:1px solid var(--chrome-fg); border-radius:var(--radius); padding:3px 9px; color:var(--chrome-fg); margin-left:10px; flex-shrink:0;` — hit area ~21px tall.
- `.dock-btn` (basm.css:2421-2424): `background:none; border:0; padding:2px 6px; color:var(--chrome-fg); font:15px/1 var(--font-mono);` — small hit area (chat dock header).
- **The ONLY current mobile topbar rule is in the `@media (max-width: 480px)` block (basm.css:1996-2006), NOT 640px** — the SPEC's "640px" lead was off; the real lines are:
  ```css
  @media (max-width: 480px) {
    ...
    .topbar nav { gap: 12px; }
    .topbar nav a { font-size: 11px; }
  }
  ```
- The storybook drawer CSS (basm.css:3029-3054) is the responsive exemplar to mirror — note it uses literal z-index values `60` (`.sb-side`), `55` (`.sb-backdrop`), and `50` (`.sb-topbar`), `transform:translateX(-104%)` closed / `transform:none` open via `.is-open`, and `@media (prefers-reduced-motion: reduce) { .sb-side { transition: none; } }`:
  ```css
  .sb-side {
    position: fixed; inset: 0 auto 0 0; width: min(86vw, 322px); z-index: 60;
    transform: translateX(-104%); transition: transform .2s var(--ease-crisp); box-shadow: 6px 0 0 rgba(0, 0, 0, .4);
  }
  .sb-side.is-open { transform: none; }
  .sb-backdrop.is-open { display: block; position: fixed; inset: 0; z-index: 55; background: rgba(8, 5, 2, .62); }
  ```
- CSS rule blocks are appended at the END of basm.css under a `/* ── Section ── */` comment banner. New rules for THIS plan go at the end, under their own banner. The file is 3218 lines.

### `internal/web/assets/static/basm.js` — the existing toggle (the a11y gap to fix for the new drawer)
`basmToggleNav` (basm.js:200-204) toggles `aria-expanded` only — **no focus move, no focus trap, no Escape, and the offscreen `.sb-side` links stay tabbable**:
```js
window.basmToggleNav = function () {
  var open = document.documentElement.classList.toggle('sb-nav-open');
  document.querySelectorAll('.sb-side, .sb-backdrop').forEach(function (el) { el.classList.toggle('is-open', open); });
  document.querySelectorAll('.sb-burger').forEach(function (b) { b.setAttribute('aria-expanded', open ? 'true' : 'false'); });
};
```
Theme functions live at the top (`basmToggleTheme` basm.js:10, `basmCycleTheme` basm.js:37). The file is vanilla, no framework, no build step — keep the new drawer toggle the same way. New JS appends at the END of the file under a `// ── Section ──` comment.

### Design tokens / constraints to honor (DESIGN.md "Hearthwood")
- `--radius:0` (square corners), 2px outlines, bevels — NEVER rounded/blur.
- Colors are `var(--token)` ONLY (no raw hex; `rgba(...)` is allowed for scrims, as the existing `.sb-backdrop` does).
- `--chrome` is ALWAYS dark in both modes; on wood use `--chrome-fg`/`--gold` (dock-light tokens), NOT ink/page tokens (the legibility trap).
- Canonical breakpoint for this drawer: **720px** (tablet). The topbar nav overflows by ~720px on real content, and 720px is an existing documented constant — use `@media (max-width: 720px)` for the product topbar drawer (independent of the storybook drawer's 920px, which is a wider rail).

## Commands you will need
| Purpose | Command | Expected |
| --- | --- | --- |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test (shell) | `go test ./internal/ui/shell/...` | ok |
| Storybook render | `go test ./internal/feature/storybook/...` | ok |
| Full test | `go test ./...` | all pass |
| Format | `gofmt -l internal/ui/shell/shell.go internal/feature/storybook/stories_navigation.go` | empty output |
| Whitespace | `git diff --check` | no output |
| Run app | `go run . serve --http=127.0.0.1:8090` (it may already be serving on :8090) | serves UI |

> Sandbox note: in a TLS-intercepting Hyperagent sandbox, Go commands need the GOPROXY shim — see `docs/hyperagent-sandbox.md` (GOSUMDB stays on).

## Scope
**In scope** (only files you may modify):
- `internal/ui/shell/shell.go` — emit the burger + off-canvas drawer markup as gomponents nodes inside `Topbar`.
- `internal/ui/shell/shell_test.go` — assert the new burger/drawer markup (ids/aria/classes).
- `internal/web/assets/static/basm.css` — new `@media (max-width: 720px)` topbar-drawer rules + touch-target padding; appended at end under a new banner.
- `internal/web/assets/static/basm.js` — a new, product-scoped toggle (`basmToggleTopnav`) with focus move + trap + Escape + `inert`; appended at end under a new banner.
- `internal/feature/storybook/stories_navigation.go` — update `topbarStory` to document the responsive drawer (add a do/dont about mobile + a note). (Do NOT register a new story id unless you decide to add one — `topbar` already exists; if you add a separate "responsive topbar" variant keep it inside `topbarStory`.)

**Out of scope** (do NOT touch, even though related):
- `internal/ui/shell/sidebar.go` and the storybook `SidebarPage` drawer + `basmToggleNav` — they ship and work; regressing them breaks the storybook. **Do not retrofit `inert` onto `.sb-side`.** (STOP condition if you think you must.)
- The domain nav targets/routes (`/focus/quests` etc.) — navigation is unchanged; only the layout/affordance changes.
- The chat dock (`#dock`, `.dock-btn`, `basmToggleNav` for the rail) — `.dock-btn` touch-target is mentioned in the SPEC but is a separate surface; **deferred** to keep this plan surgical (note it in Maintenance).
- The canonical `--z-*`/`--space-*` tokens from the shared plan set — they do NOT exist at HEAD and this plan does not depend on the plan that adds them. Use the same literal z-index values the existing `.sb-side`/`.sb-backdrop` use (60/55/50) so this plan is self-contained.

## Git workflow
Branch `improve/078-responsive-topbar-drawer`. Conventional commits (e.g. `feat(web): responsive topbar off-canvas nav` / `fix(a11y): trap focus in topbar drawer`). Do NOT push or open a PR unless explicitly told.

## Steps

### Step 1: Branch and confirm the baseline overflow
Create the branch. Then confirm the live problem so the fix is measurable.
- `git switch -c improve/078-responsive-topbar-drawer`
- Run the app (`go run . serve --http=127.0.0.1:8090` if not already serving). Open `http://127.0.0.1:8090/focus/quests`, set viewport width to 390px (DevTools device toolbar or `await page.setViewport({width:390,height:800})`), and evaluate `document.documentElement.scrollWidth` vs `window.innerWidth`. Record the BEFORE value (expected: scrollWidth > innerWidth, ~632 vs 390).

**Verify**: BEFORE measurement captured — `scrollWidth > innerWidth` at 390px (the bug reproduces). If it does NOT reproduce, STOP and report (the overflow may already be fixed).

### Step 2: Add the burger + off-canvas drawer markup to `Topbar` (gomponents, not raw markup)
Edit `internal/ui/shell/shell.go`. Restructure `Topbar` so the existing brand + nav + theme buttons stay (desktop unchanged), and ADD:
1. A burger button that opens the drawer, hidden on desktop via CSS. Place it as a child of `.topbar`:
   ```go
   h.Button(h.Class("topnav-burger"), h.Type("button"),
       g.Attr("onclick", "basmToggleTopnav()"),
       h.Aria("label", "Open navigation"),
       h.Aria("expanded", "false"),
       h.Aria("controls", "topnav-drawer"),
       g.Text("☰"),
   ),
   ```
2. Give the existing desktop `<nav>` a class so CSS can hide it on mobile: change `h.Nav(...)` to `h.Nav(h.Class("topnav-desktop"), ...)`.
3. A drawer aside + backdrop, AFTER the theme buttons, holding a SECOND copy of the same nav links (use the existing `navLink` helper so routes/aria stay identical) plus the theme buttons so they are reachable on mobile:
   ```go
   h.Div(h.Class("topnav-backdrop"), g.Attr("onclick", "basmToggleTopnav()")),
   h.Aside(h.ID("topnav-drawer"), h.Class("topnav-drawer"),
       h.Aria("hidden", "true"),
       h.Nav(h.Class("topnav-drawer-nav"),
           navLink("/focus/quests", "Quests", "quests", active),
           navLink("/focus/memory", "Knowledge", "knowledge", active),
           navLink("/focus/lifelog", "Life", "life", active),
           navLink("/focus/journal", "Journal", "journal", active),
           navLink("/focus/heads", "Heads", "heads", active),
           navLink("/focus/settings", "Settings", "settings", active),
       ),
   ),
   ```
Keep the brand, the desktop `<nav class="topnav-desktop">`, and the two theme buttons exactly as they are today (the theme buttons stay in the bar; on mobile they shrink to icon hit-targets via Step 4, but stay reachable — the SPEC requires theme buttons reachable on mobile, and keeping them in the bar is simpler than duplicating them). To avoid duplicating the six `navLink` calls, extract them into a small unexported `func topbarLinks(active string) []g.Node` and call it from both the desktop nav and the drawer nav.

Keep the DOM ORDER such that the existing `shell_test.go` substrings still match (the desktop nav links must still render with their current `aria-current`/href text). Do not change `navLink`.

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go test ./internal/ui/shell/...` → the EXISTING TestPage still passes (desktop nav substrings unchanged). If it fails because the duplicated drawer copy now makes a substring match twice, that is fine for `strings.Contains` — but if you reordered DOM and broke a match, fix the order. STOP if you cannot satisfy the existing assertions without changing routes.

### Step 3: Add the responsive CSS (drawer ≤720px, desktop unchanged)
Edit `internal/web/assets/static/basm.css`. Append at the END of the file a new banner section. Mirror the storybook drawer's proven rules but with the new product classes, hidden by default on desktop:
```css
/* ══ Section: Responsive product topbar — off-canvas nav ≤720px ══════════ */
.topnav-burger { display: none; }
.topnav-drawer { display: none; }
.topnav-backdrop { display: none; }

@media (max-width: 720px) {
  .topbar .topnav-desktop { display: none; }
  .topnav-burger {
    display: inline-flex; align-items: center; justify-content: center;
    margin-left: auto; min-width: 44px; min-height: 44px;
    font-family: var(--font-mono); font-size: 18px; line-height: 1; cursor: pointer;
    background: none; border: 1px solid var(--chrome-fg); border-radius: var(--radius);
    color: var(--gold);
  }
  .topnav-drawer {
    display: block; position: fixed; inset: 62px auto 0 0; width: min(86vw, 322px); z-index: 60;
    background-color: var(--chrome); background-image: var(--wood-planks), var(--grain-warm); background-size: auto, 4px 4px;
    border-right: 2px solid var(--outline-2);
    transform: translateX(-104%); transition: transform .2s var(--ease-crisp);
    box-shadow: 6px 0 0 rgba(0, 0, 0, .4); padding: 12px;
  }
  .topnav-drawer.is-open { transform: none; }
  .topnav-drawer-nav { display: flex; flex-direction: column; gap: 4px; }
  .topnav-drawer-nav a {
    display: flex; align-items: center; min-height: 44px; padding: 0 12px;
    font-family: var(--font-mono); font-size: 13px; text-transform: uppercase; letter-spacing: .06em;
    color: var(--chrome-fg); text-decoration: none; border: 1px solid transparent;
  }
  .topnav-drawer-nav a:hover,
  .topnav-drawer-nav a[aria-current="page"] { color: var(--gold); border-color: var(--gold-deep); }
  .topnav-backdrop.is-open {
    display: block; position: fixed; inset: 0; z-index: 55; background: rgba(8, 5, 2, .62);
  }
}
@media (prefers-reduced-motion: reduce) { .topnav-drawer { transition: none; } }
```
Also bump the in-bar touch targets so the burger and theme buttons clear 44px on touch (the SPEC's Phase 2). In the SAME `@media (max-width: 720px)` block, expand the theme buttons' hit area without enlarging the visual chrome — e.g.:
```css
  .topbar .theme-toggle, .topbar .theme-cycle { min-height: 44px; padding: 0 12px; }
```
Honor Hearthwood: `--radius` is 0 so corners stay square; use only `var(--token)` and the one `rgba` scrim already established. Do NOT touch the existing `@media (max-width: 480px) { .topbar nav ... }` rules — they apply to `.topbar nav` which now also matches the drawer nav, but the 480px block only sets gap/font and is harmless; leave it (note it in Maintenance).

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0 (CSS is a static asset, build proves nothing broke)
- `git diff --check` → no output (no trailing whitespace)

### Step 4: Add the accessible toggle in `basm.js` (focus move + trap + Escape + inert) — REQUIRED
Edit `internal/web/assets/static/basm.js`. Append at the END a new section. This is the product drawer toggle; it must NOT reuse `basmToggleNav` (that one is storybook-scoped and lacks a11y). Implement `basmToggleTopnav` as vanilla JS:
- Toggle `.is-open` on `.topnav-drawer` and `.topnav-backdrop`.
- Sync `aria-expanded` on `.topnav-burger` and `aria-hidden` on `#topnav-drawer`.
- On OPEN: remember `document.activeElement` (to restore on close), move focus to the first focusable in the drawer (`#topnav-drawer a`), and set `inert` on the page regions outside the drawer/backdrop so the rest of the page is untabbable. Simplest safe approach: set `inert` (and `aria-hidden="true"`) on the `.topbar`'s other children and `#main`/`#dock` — OR, lower-risk, just keep the drawer's own offscreen links out of the tab order when CLOSED by toggling `inert` on `.topnav-drawer` itself (closed → `inert`, open → not). Do BOTH for completeness: closed drawer is `inert`; when open, trap Tab within it.
- Trap Tab: on `keydown` Tab inside the open drawer, wrap focus between first and last focusable.
- Escape: `keydown` Escape closes the drawer and restores focus to the burger.
- Guard: only act on the product drawer (`#topnav-drawer`), never `.sb-side`.

Target shape:
```js
// ── Product topbar off-canvas nav (accessible) ─────────────────────
// Separate from basmToggleNav (storybook). Closed drawer is inert (untabbable);
// open moves focus in, traps Tab, closes on Escape/backdrop, restores focus.
(function () {
  function drawer()  { return document.getElementById('topnav-drawer'); }
  function backdrop(){ return document.querySelector('.topnav-backdrop'); }
  function burger()  { return document.querySelector('.topnav-burger'); }
  function focusables(d) { return d.querySelectorAll('a[href], button:not([disabled])'); }
  var lastFocus = null;

  function setClosedInert() { var d = drawer(); if (d && !d.classList.contains('is-open')) d.inert = true; }
  document.addEventListener('DOMContentLoaded', setClosedInert);

  window.basmToggleTopnav = function () {
    var d = drawer(); if (!d) return;
    var open = !d.classList.contains('is-open');
    d.classList.toggle('is-open', open);
    var b = backdrop(); if (b) b.classList.toggle('is-open', open);
    d.inert = !open;
    d.setAttribute('aria-hidden', open ? 'false' : 'true');
    var bg = burger(); if (bg) bg.setAttribute('aria-expanded', open ? 'true' : 'false');
    if (open) {
      lastFocus = document.activeElement;
      var f = focusables(d); if (f.length) f[0].focus();
    } else if (lastFocus && lastFocus.focus) {
      lastFocus.focus();
    }
  };

  document.addEventListener('keydown', function (e) {
    var d = drawer(); if (!d || !d.classList.contains('is-open')) return;
    if (e.key === 'Escape') { e.preventDefault(); window.basmToggleTopnav(); return; }
    if (e.key !== 'Tab') return;
    var f = focusables(d); if (!f.length) return;
    var first = f[0], last = f[f.length - 1];
    if (e.shiftKey && document.activeElement === first) { e.preventDefault(); last.focus(); }
    else if (!e.shiftKey && document.activeElement === last) { e.preventDefault(); first.focus(); }
  });

  // Navigating dismisses the drawer.
  document.addEventListener('click', function (e) {
    var d = drawer();
    if (d && d.classList.contains('is-open') && e.target.closest('#topnav-drawer a')) {
      window.basmToggleTopnav();
    }
  });
})();
```
Keep it framework-free (no Datastar signals needed — this is pure chrome state, same posture as `basmToggleNav`).

**Verify**:
- `git diff --check` → no output
- The new function appears: `grep -c "basmToggleTopnav" internal/web/assets/static/basm.js` → ≥ 2 (the target JS shape below contains 3 occurrences — the `window.basmToggleTopnav = function` definition plus the two internal calls in the Escape handler and the link-click handler). The burger `onclick="basmToggleTopnav()"` lives in `shell.go`, NOT this file, so do NOT count it here.

### Step 5: Update the storybook story to document the responsive behavior
Edit `internal/feature/storybook/stories_navigation.go`, `topbarStory()`. The story renders `shell.Topbar("quests")`, which now also contains the burger + drawer markup (hidden on desktop, so the desktop tile is visually unchanged). Update the `Blurb` and add a Do/Dont entry documenting the mobile drawer, e.g. add to `Dos`: `"On phones (≤720px) the domain links collapse into an accessible off-canvas drawer (the burger); keep that one mechanism."` Do NOT register a new story id — `topbar` already covers it. If you want a visible mobile preview, you MAY add a second `Variant` that wraps the topbar in a fixed-width 360px container styled to force the mobile media query — but media queries key off VIEWPORT not container width, so a container will NOT trigger them; therefore do NOT fake it. Document the behavior in prose only.

**Verify**:
- `go test ./internal/feature/storybook/...` → ok (TestAllStoriesRender still passes — the topbar story renders)
- `go vet ./...` → exit 0
- `gofmt -l internal/feature/storybook/stories_navigation.go internal/ui/shell/shell.go` → empty

### Step 6: Update shell_test.go to assert the new markup
Edit `internal/ui/shell/shell_test.go`. Add assertions (in a new test `TestTopbarDrawer` or extend `TestPage`) that the rendered page contains:
- the burger: `class="topnav-burger"` and `onclick="basmToggleTopnav()"` and `aria-controls="topnav-drawer"`
- the drawer container: `id="topnav-drawer"` with `class="topnav-drawer"`
- the backdrop: `class="topnav-backdrop"`
- the desktop nav still carries `class="topnav-desktop"`
Match existing style (table of want-substrings + `strings.Contains`). Keep using `shell.Page(...)` (which calls `Topbar`).

**Verify**:
- `go test ./internal/ui/shell/...` → ok (new + existing tests pass)

### Step 7: Full verification + visual check in BOTH modes
Run the full gate, then verify visually.

**Verify (commands)**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass
- `git diff --check` → no output
- `gofmt -l internal/ui/shell/shell.go internal/ui/shell/shell_test.go internal/feature/storybook/stories_navigation.go` → empty

**Verify (visual — run the app, both modes)**: open `http://127.0.0.1:8090/focus/quests`. For EACH of the 4 widths 390 / 560 / 768 / 1280 and for BOTH modes (force via `document.documentElement.className='theme-hearthwood dark'` then `...='theme-hearthwood light'`):
- 390 & 560 (≤720px): `document.documentElement.scrollWidth <= window.innerWidth` (NO horizontal scroll — the AFTER must be ≤, vs the BEFORE 632>390). The burger is visible; the desktop links are hidden. Clicking the burger opens the drawer (slides in, gold links, square corners), Tab cycles inside it, Escape closes it and returns focus to the burger, clicking the backdrop closes it. All six domain links + Settings are reachable.
- 768 & 1280 (>720px): the desktop nav + theme buttons render exactly as before; the burger and drawer are display:none.
- Reduced motion: with `prefers-reduced-motion: reduce` (DevTools rendering emulation), the drawer does NOT animate the slide (snaps).
- Theme buttons at ≤720px are ≥44px tall (measure `getBoundingClientRect().height`) and still legible on wood (`--chrome-fg`/`--gold`, NOT ink tokens).

## Test plan
- **`internal/ui/shell/shell_test.go`** — pattern is the existing `TestPage` (a `[]string` of want-substrings + `strings.Contains`). Add `TestTopbarDrawer` asserting `topnav-burger`, `basmToggleTopnav()`, `id="topnav-drawer"`, `topnav-drawer`, `topnav-backdrop`, `topnav-desktop`. Verify: `go test ./internal/ui/shell/...`.
- **Storybook** — `topbarStory()` already renders `shell.Topbar`; the catalog test `TestAllStoriesRender` (and `tours_test.go`) fail if it can't render. No new story id; just updated prose/dos. Verify: `go test ./internal/feature/storybook/...`.
- **No JS/CSS unit tests** in this repo — the JS/CSS behavior is covered by the manual visual check in Step 7 (the only mechanism available; document the BEFORE/AFTER scrollWidth numbers in the returned summary).

## Done criteria
- [ ] `CGO_ENABLED=0 go build ./...` → exit 0
- [ ] `go vet ./...` → exit 0
- [ ] `go test ./...` → all pass (incl. `internal/ui/shell` and `internal/feature/storybook`)
- [ ] `gofmt -l` on changed `.go` files → empty
- [ ] `git diff --check` → no output
- [ ] `grep -n "topnav-burger" internal/ui/shell/shell.go` → matches (burger added)
- [ ] `grep -n "basmToggleTopnav" internal/web/assets/static/basm.js` → matches (toggle defined)
- [ ] `grep -n "topnav-drawer" internal/web/assets/static/basm.css` → matches (drawer CSS added)
- [ ] Only in-scope files changed: `git diff --name-only` (working tree; or `git diff --name-only 12a2ff5..HEAD` if you have already committed — the plan does not require a commit) lists ONLY the five in-scope files (+ `plans/readme.md`).
- [ ] **VISUAL (both modes, CSS+markup changed)**: at 390px and 560px `document.documentElement.scrollWidth <= window.innerWidth` (was ~632>390 before); drawer opens/closes; Escape closes + restores focus; Tab is trapped; reduced-motion disables the slide; theme buttons ≥44px at ≤720px; desktop ≥768px unchanged.
- [ ] `plans/readme.md` 078 row updated (add the row if it is not present yet, matching the existing column format).

## STOP conditions
- The Step 1 BEFORE measurement does NOT reproduce horizontal overflow at 390px (problem already fixed elsewhere) — STOP and report.
- The existing `TestPage` assertions cannot be satisfied without changing nav routes/labels — STOP (do not change `/focus/*` targets; that is out of scope).
- The drawer needs client framework state (Datastar signals) to work — STOP; it must stay vanilla like `basmToggleNav`. If you cannot make focus-trap/inert work with plain JS, report what's blocking.
- Applying `inert` regresses the existing storybook drawer (`.sb-side`) or any other surface — STOP; `inert` must be scoped to the NEW `#topnav-drawer` only, never `.sb-side`. Report if you find yourself wanting to touch `.sb-side`/`basmToggleNav`.
- Any Verify command fails twice in a row after a fix attempt — STOP and report the exact output.
- A drift-check mismatch (Step 0) — STOP.

## Maintenance notes
- **Two drawer mechanisms now coexist**: `basmToggleNav` (storybook `.sb-side`, 920px, no a11y trap) and `basmToggleTopnav` (product `#topnav-drawer`, 720px, accessible). A future cleanup could unify them by promoting the accessible toggle to also drive the storybook rail — but that touches `sidebar.go`/`SidebarPage`, out of scope here. Document the divergence so a reviewer doesn't "dedupe" them blindly.
- **Tokenization follow-up (deferred)**: the new CSS uses literal z-index `60/55` and breakpoint `720px` to stay self-contained (the canonical `--z-overlay`/`--z-drawer`/`--z-scrim` and breakpoint constants do NOT exist at HEAD). When the shared token plan lands, migrate these literals to the named tokens in one pass.
- **`.dock-btn` touch target (deferred)**: the SPEC flagged `.dock-btn` (basm.css:2421-2424) as sub-44px, but it lives in the chat dock header (a different surface) — bumping it is a separate, lower-priority change; left out to keep this surgical.
- **`@media (max-width: 480px) .topbar nav`** (basm.css:1996-2006) still sets `gap`/`font-size` on `.topbar nav`; because the drawer nav is `.topnav-drawer-nav` (not `.topbar nav`) it is unaffected, and the desktop `.topnav-desktop` nav is `display:none` ≤720px so the 480px rule is dead-but-harmless. A reviewer should confirm it isn't visually leaking; a later cleanup could delete it.
- A reviewer should scrutinize: (1) focus is actually trapped (Tab/Shift-Tab wrap), (2) `inert` makes closed-drawer links untabbable (Tab through the page never lands on a hidden link), (3) the desktop layout is byte-for-byte unchanged above 720px, (4) theme buttons remain reachable on mobile (kept in the bar).
