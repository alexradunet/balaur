# Plan 079: Close four verified accessibility gaps — reduced-motion, dark-mode contrast, control names/state, skip link

> **Executor instructions**: Follow step by step. Run every Verify and confirm before moving on. On a STOP condition, stop and report — do not improvise. When done, update the 079 row in plans/readme.md (add the row if it is not present yet, matching the existing column format).
>
> **Drift check (run first)**: `git diff --stat 12a2ff5..HEAD -- internal/web/assets/static/basm.css internal/web/assets/static/basm.js internal/ui/shell/shell.go internal/ui/shell/sidebar.go internal/ui/composer.go internal/ui/shell/shell_test.go internal/ui/composer_test.go web/templates/home.html` — if any in-scope file changed since this plan was written, compare the "Current state" excerpts to the live code; on mismatch, STOP.

## Status
- **Priority**: P1
- **Effort**: S
- **Risk**: LOW
- **Depends on**: none
- **Category**: a11y
- **Planned at**: commit `12a2ff5`, 2026-06-17

## Why this matters
Four independent, high-confidence accessibility defects ship today. (A1) The chat "thinking" portrait glow that arrived in the gomponents chat port animates forever even when the OS requests reduced motion — the reduced-motion block only suppresses the *legacy* avatar. DESIGN.md:370 sets the bar: "All animation respects `prefers-reduced-motion`." (A2) The recap hint on the dark wood dock renders `--smoke` text at ~2.68:1 contrast (below WCAG's 3:1 for large text) because `--smoke` is a parchment token that flips dark in dark mode, but it sits on the always-dark `--chrome-2` wood — DESIGN.md's "legibility trap" calls for a dock-light token (`--chrome-fg`) on wood. (A3) The persistent chrome (theme toggle, palette cycle, composer tool wells, sound bell) exposes no accessible names or pressed state — a screen reader announces unlabeled buttons; the repo has zero `aria-pressed`. (A4) There is no skip link, so keyboard users must tab through the whole topbar nav on every page. Each fix is small, local, and verifiable.

## Current state

### A1 — reduced-motion regression (basm.css)
The live pending-glow rule (the migrated chat organism), confirmed at `internal/web/assets/static/basm.css:3152`:
```
3152:.cmsg-pending .cmsg-portrait { animation: basm-glow 1.6s ease-in-out infinite; }
```
The reduced-motion block, confirmed at `internal/web/assets/static/basm.css:939-951` (the `@media` opens at 940):
```
939:/* ── Reduced motion ──────────────────────────────────────────────── */
940:@media (prefers-reduced-motion: reduce) {
941:  .btn, .btn:hover, .btn:active { transition: none; transform: none; }
942:  .msg-pending .balaur-avatar,
943:  .balaur-avatar[data-state="thinking"],
944:  .balaur-avatar[data-state="working"] { animation: none; }
...
951:}
```
It suppresses only the legacy `.msg-pending .balaur-avatar`, never the live `.cmsg-pending .cmsg-portrait`. **Fix**: add `.cmsg-pending .cmsg-portrait { animation: none; }` inside this block.

### A2 — dark-mode legibility trap (basm.css + home.html)
`.recap-hint` text color, confirmed at `internal/web/assets/static/basm.css:1202-1209`:
```
1202:.recap-hint {
1203:  font-family: var(--font-mono);
1204:  font-size: 12px;
1205:  text-transform: uppercase;
1206:  letter-spacing: .06em;
1207:  color: var(--smoke);
1208:  text-align: center;
1209:}
```
`--smoke` flips with mode, confirmed at `internal/web/assets/static/basm.css:37`:
```
37:  --smoke:     light-dark(#8f7a52, #6b5639);
```
The dock background is the always-dark wood, confirmed at `internal/web/assets/static/basm.css:2364,2372`:
```
2364:#dock {
2372:  background-color: var(--chrome-2);
```
`--chrome-2` is constant-dark and `--chrome-fg` is its paired text token, confirmed at `internal/web/assets/static/basm.css:24-25`:
```
24:  --chrome-2:  light-dark(#2c1a0c, #1d0f06);  /* deeper wood inset */
25:  --chrome-fg: light-dark(#d6bb92, #b59872);  /* text on wood */
```
**`.recap-hint` IS live** (do not skip): it is rendered at `web/templates/home.html:15` inside `#recap.recap-zone`, and although `#dock .recap-zone` is hidden globally (`internal/web/assets/static/basm.css:2384`), it is re-shown on Home at `internal/web/assets/static/basm.css:3190-3198`:
```
2384:#dock .recap-zone, #dock .recap-band { display: none; }
...
3190:html.home #dock .recap-zone,
3191:html.home #dock .recap-band {
...
3198:html.home #dock .recap-zone, html.home #dock .recap-band { display: block; }
```
In dark mode `--smoke` = `#6b5639` on `--chrome-2` = `#1d0f06` ≈ **2.68:1** (fails WCAG 3:1 for large text). **Fix**: change `.recap-hint` `color` from `var(--smoke)` to `var(--chrome-fg)` (= `#b59872` on `#1d0f06` ≈ 5.7:1).

**Dead siblings — do NOT change (documented in your summary)**: `.model-choice` / `.model-choice-disabled .model-detail` / `.model-detail` at `internal/web/assets/static/basm.css:1654-1668` also use `--smoke`, but the live model card (`internal/feature/modelcards/modelcard.go:53-54`) renders `.model-detail-line`, NOT `.model-detail`/`.model-choice`. A repo grep confirms `model-choice`/`model-detail` (exact, word-boundary) appears only in basm.css and a *negative* test assertion (`internal/web/templates_test.go:57` asserts `model-choice-list` is **absent**). These rules have no live render path — leave them.

### A3 — no accessible names/state on persistent chrome
Theme buttons, confirmed at `internal/ui/shell/shell.go:88-101` (SPEC said 88-98 — drift, use the lines below):
```
88:		h.Button(h.Class("theme-cycle"), h.Type("button"),
89:			g.Attr("onclick", "basmCycleTheme()"),
90:			h.Title("Cycle theme"), h.Aria("label", "Cycle theme"),
91:			g.Text("Hearth"),
92:		),
93:		h.Button(h.Class("theme-toggle"), h.Type("button"),
94:			g.Attr("onclick", "basmToggleTheme()"),
95:			h.Title("Toggle light/dark mode"),
96:			h.Aria("label", "Toggle light/dark mode"),
97:			g.Text("◑"),
98:		),
```
The two buttons already carry `aria-label`; what is missing is **pressed/current state** (the repo has zero `aria-pressed`). The JS that keeps the glyph/title in sync is at `internal/web/assets/static/basm.js`:
- `basmUpdateThemeButtons()` (lines 24-30): rewrites `.theme-toggle` `textContent`/`title` from `isLight`.
- `basmUpdatePaletteButtons()` (lines 56-68): rewrites `.theme-cycle` `textContent`/`title` from the current palette label.

Composer tool wells + sound bell, confirmed at `internal/ui/composer.go:75-81` (SPEC said 77-81 — drift, use the lines below):
```
75:	toolRow := []g.Node{h.Class("composer-tools")}
76:	for _, t := range tools {
77:		toolRow = append(toolRow, h.Button(h.Class("composer-tool"), h.Type("button"),
78:			h.Img(h.Src("/static/icons/"+t+".png"), h.Alt(""), g.Attr("decoding", "async"))))
79:	}
80:	toolRow = append(toolRow, h.Button(h.Class("composer-tool composer-sound"), h.Type("button"),
81:		h.Img(h.Src("/static/icons/bell.png"), h.Alt(""), g.Attr("decoding", "async"))))
```
These are icon-only (`Alt("")`, no `aria-label`). **They are not functional**: a repo grep finds no handler for `composer-tool`/tool-well in `internal/web/*.go`, and the live composer (`internal/web/home.go:29-35`) sets no tool behavior. Per the SPEC's "if the wells are not yet functional, render them disabled with an explanatory label" — give each well an `aria-label` derived from the icon name and mark it `disabled`; do the same for the sound bell.

`Tools` are generic icon names (default `scroll`/`tome`/`lens`; see `internal/ui/composer.go:70-72`). `internal/ui` is an atom layer and must NOT import `internal/feature/*`, so the label must derive from the icon name string alone (no domain lookup). Use a plain title-cased label, e.g. `aria-label="scroll (coming soon)"`.

### A4 — no skip link
`shell.Page` body, confirmed at `internal/ui/shell/shell.go:39-46` (the `h.Body(` opens at 39):
```
39:		h.Body(
40:			Topbar(p.Active),
41:			h.Div(h.Class("with-sidebar"),
42:				h.Main(h.ID("main"), p.Body),
43:			),
44:			h.Aside(h.ID("dock"), p.Dock),
45:		),
46:	)
```
`SidebarPage` body, confirmed at `internal/ui/shell/sidebar.go:72-99` (the `h.Body(` opens at 81, the `.sb-canvas` `<main>` is at 90-93 — note its main has `h.Class("sb-canvas")` not `h.ID("main")`):
```
81:			h.Body(
82:				h.Div(h.Class("sb-root"),
83:					h.Header(h.Class("sb-topbar"), ... ),
...
90:					h.Main(h.Class("sb-canvas"),
91:						h.Header(h.Class("sb-crumb"), g.Text(crumb)),
92:						p.Body,
93:					),
...
```
There is no `.skip-link` or `.sr-only` class anywhere in the repo (grep confirms). **Fix**: emit `<a class="skip-link" href="#main">Skip to content</a>` as the FIRST child of `<body>` in `shell.Page`; in `SidebarPage` add `h.ID("main")` to the `.sb-canvas` main (so the anchor target exists) and emit the same skip link as the first body child. Add a `.skip-link` CSS rule (offscreen until `:focus`) appended at the end of basm.css under a Section banner, using the `--z-tooltip` tier if defined, else a literal `z-index: 30`.

### Conventions to match
- gomponents atoms: `import g "maragu.dev/gomponents"` + `h "maragu.dev/gomponents/html"` (qualified `h.`). `internal/ui` must never import `internal/feature/*`. Pass attrs via `g.Attr`/`h.Aria`.
- `h.Aria("pressed", "true")` renders `aria-pressed="true"`; `h.Aria("label", "...")` renders `aria-label`. `h.Disabled()` renders the `disabled` attribute.
- CSS: square corners (`--radius:0`), `var(--token)` colors only, single-dash class names, new rule blocks appended at the END of basm.css under a `/* ── ... ── */` banner. Light/dark mode flips `light-dark()` tokens — `--chrome*` tokens are the dock-light family that stays correct on wood in BOTH modes.
- Tests: standard `testing`, `strings.Contains` assertions (see `internal/ui/shell/shell_test.go`, `internal/ui/composer_test.go`). Storybook stories render in `TestAllStoriesRender`.

## Commands you will need
| Purpose | Command | Expected |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test (all) | `go test ./...` | all pass |
| Test (shell) | `go test ./internal/ui/shell/...` | ok |
| Test (composer/ui) | `go test ./internal/ui/...` | ok |
| Storybook render | `go test ./internal/feature/storybook/...` | ok |
| Format | `gofmt -l internal/ui/shell/shell.go internal/ui/shell/sidebar.go internal/ui/composer.go internal/ui/shell/shell_test.go internal/ui/composer_test.go` | empty output |
| Whitespace | `git diff --check` | no output |
| Grep aria-pressed | `grep -rn "aria-pressed\|Aria(\"pressed\"" internal/` | the new occurrences only |

> Sandbox note: in a TLS-intercepting Hyperagent sandbox, `go` commands need the GOPROXY shim — see `docs/hyperagent-sandbox.md`. Keep GOSUMDB on.

## Scope
**In scope** (only files you may modify):
- `internal/web/assets/static/basm.css` — A1 (reduced-motion line), A2 (`.recap-hint` color), A4 (new `.skip-link` rule).
- `internal/ui/shell/shell.go` — A3 (`aria-pressed` on theme toggle + palette name), A4 (skip link in `Page`).
- `internal/ui/shell/sidebar.go` — A4 (skip link + `id="main"` on `.sb-canvas` in `SidebarPage`).
- `internal/web/assets/static/basm.js` — A3 (sync `aria-pressed` on `.theme-toggle` in `basmUpdateThemeButtons`; sync `aria-label` to current palette in `basmUpdatePaletteButtons`).
- `internal/ui/composer.go` — A3 (`aria-label` + `disabled` on tool wells and the sound bell).
- `internal/ui/shell/shell_test.go` — assert skip link.
- `internal/ui/composer_test.go` — assert tool-well labels.

**Out of scope** (do NOT touch):
- Topbar responsiveness / overflow (plan 078) — but A3's additive `aria-pressed` on the same theme buttons is in scope here; keep edits additive (only add attributes, do not restructure the buttons).
- Drawer/burger focus trap (plan 078, finding A5).
- `.model-choice`/`.model-detail` rules (basm.css:1654-1668) — dead, no live render path; leave them and note in your summary.
- `web/templates/home.html` — `.recap-hint` markup is fine; only its CSS color changes. Do not edit the template.
- `internal/feature/modelcards/modelcard.go` — not affected.
- The legacy `.msg-pending .balaur-avatar` rule — leave it; the legacy avatar still exists.

## Git workflow
Branch `improve/079-a11y-quick-wins`. Conventional commits (e.g. `fix(web): suppress chat glow under reduced-motion`, `fix(web): recap hint contrast on dark wood`, `feat(ui): accessible names + state for chrome controls`, `feat(ui): skip link to main content`). Do NOT push or open a PR unless told.

## Steps

### Step 1: A1 — suppress the live pending glow under reduced-motion
In `internal/web/assets/static/basm.css`, inside the `@media (prefers-reduced-motion: reduce)` block (lines 940-951), add a line next to the existing avatar suppression (after line 944's `.balaur-avatar[data-state="working"] { animation: none; }`):
```
  .cmsg-pending .cmsg-portrait { animation: none; }
```
**Verify**: `grep -n "cmsg-pending .cmsg-portrait" internal/web/assets/static/basm.css` → two lines: the `animation: basm-glow` rule (~3152) AND the new `animation: none` inside the reduced-motion block (~945). Confirm the new line is BETWEEN lines 940 and 951 (inside the `@media`): `awk 'NR>=940 && NR<=953' internal/web/assets/static/basm.css | grep "cmsg-portrait"` returns the `none` line.

### Step 2: A2 — fix recap-hint contrast on the dark wood dock
In `internal/web/assets/static/basm.css`, change the `.recap-hint` color (line 1207) from `var(--smoke)` to `var(--chrome-fg)`:
```
  color: var(--chrome-fg);
```
Do NOT touch `.model-choice`/`.model-detail` (1654-1668) — they are dead.
**Verify**: `awk 'NR>=1202 && NR<=1208' internal/web/assets/static/basm.css | grep -c "var(--chrome-fg)"` → `1`; and `awk 'NR>=1202 && NR<=1208' internal/web/assets/static/basm.css | grep -c "var(--smoke)"` → `0`.

### Step 3: A3 — accessible state on the theme toggle (shell.go + basm.js)
**STOP check first**: the palette cycle is a 3-state control (hearthwood/forest/dungeon), so `aria-pressed` is semantically wrong for it (boolean only). For the palette cycle, do NOT add `aria-pressed`; instead keep its `aria-label` and make it reflect the current palette (Step 4 below). If you conclude `aria-pressed` would be semantically wrong even for the binary light/dark `theme-toggle`, STOP and report.

In `internal/ui/shell/shell.go`, the `.theme-toggle` button (lines 93-98) is the binary light/dark control. Add `h.Aria("pressed", "false")` to it (false = dark, the default state seeded by the no-flash script; JS will flip it to "true" in light mode):
```
		h.Button(h.Class("theme-toggle"), h.Type("button"),
			g.Attr("onclick", "basmToggleTheme()"),
			h.Title("Toggle light/dark mode"),
			h.Aria("label", "Toggle light/dark mode"),
			h.Aria("pressed", "false"),
			g.Text("◑"),
		),
```
Then in `internal/web/assets/static/basm.js`, inside `basmUpdateThemeButtons()` (lines 24-30), set `aria-pressed` to reflect light mode alongside the existing `textContent`/`title` writes:
```
  document.querySelectorAll('.theme-toggle').forEach(btn => {
    btn.textContent = isLight ? '◑' : '☼';
    btn.title       = isLight ? 'Switch to dark mode' : 'Switch to light mode';
    btn.setAttribute('aria-pressed', isLight ? 'true' : 'false');
  });
```
(`basmUpdateThemeButtons` already runs on `DOMContentLoaded` at line 32, so the initial state is corrected after the no-flash class is applied.)
**Verify**: `grep -n 'Aria("pressed"' internal/ui/shell/shell.go` → 1 hit on `.theme-toggle`; `grep -n "setAttribute('aria-pressed'" internal/web/assets/static/basm.js` → 1 hit in `basmUpdateThemeButtons`.

### Step 4: A3 — palette cycle accessible name reflects current palette (basm.js)
In `internal/web/assets/static/basm.js`, inside `basmUpdatePaletteButtons()` (lines 56-68), set the `.theme-cycle` button's `aria-label` to include the active palette (mirror the existing `title` write at line 63). After/next to `btn.title = 'Cycle theme (now ' + labels[cur] + ')';`:
```
    btn.setAttribute('aria-label', 'Cycle theme (now ' + labels[cur] + ')');
```
(The static `aria-label="Cycle theme"` set in shell.go line 90 is the pre-JS fallback; this updates it on load and on each cycle. `basmUpdatePaletteButtons` already runs on `DOMContentLoaded` at line 69.) Leave the static shell.go `aria-label` as-is.
**Verify**: `grep -n "setAttribute('aria-label', 'Cycle theme" internal/web/assets/static/basm.js` → 1 hit.

### Step 5: A3 — label + disable the composer tool wells and bell (composer.go)
In `internal/ui/composer.go`, the tool-row loop (lines 75-81). Because the wells are non-functional, render each with a derived `aria-label` and `disabled`. Replace the loop + bell with:
```
	toolRow := []g.Node{h.Class("composer-tools")}
	for _, t := range tools {
		toolRow = append(toolRow, h.Button(h.Class("composer-tool"), h.Type("button"),
			h.Disabled(), h.Aria("label", t+" (coming soon)"),
			h.Img(h.Src("/static/icons/"+t+".png"), h.Alt(""), g.Attr("decoding", "async"))))
	}
	toolRow = append(toolRow, h.Button(h.Class("composer-tool composer-sound"), h.Type("button"),
		h.Disabled(), h.Aria("label", "Sound (coming soon)"),
		h.Img(h.Src("/static/icons/bell.png"), h.Alt(""), g.Attr("decoding", "async"))))
```
Keep `Alt("")` on the `<img>` (decorative; the button name carries the label). Do not import any feature package; the label derives from the icon-name string only.
**Verify**: `go test ./internal/ui/... ./internal/feature/storybook/...` (after updating the test in Step 8) → ok. Manual: `grep -n 'Aria("label", t+' internal/ui/composer.go` → 1 hit; `grep -n 'Aria("label", "Sound' internal/ui/composer.go` → 1 hit.

> NOTE: existing tests in `internal/ui/composer_test.go` assert the exact button strings WITHOUT `disabled`/`aria-label` (e.g. line 20: `<button class="composer-tool" type="button"><img ...>`). Those assertions will break — Step 8 updates them. The build/vet stays green; only `go test` for the composer changes until Step 8.

### Step 6: A4 — skip link in shell.Page (shell.go)
In `internal/ui/shell/shell.go`, in the `h.Body(...)` of `Page` (lines 39-45), insert the skip link as the FIRST body child, before `Topbar`:
```
		h.Body(
			h.A(h.Class("skip-link"), h.Href("#main"), g.Text("Skip to content")),
			Topbar(p.Active),
			h.Div(h.Class("with-sidebar"),
				h.Main(h.ID("main"), p.Body),
			),
			h.Aside(h.ID("dock"), p.Dock),
		),
```
(`<main id="main">` already exists at line 42, so `href="#main"` resolves.)
**Verify**: `grep -n 'skip-link' internal/ui/shell/shell.go` → 1 hit.

### Step 7: A4 — skip link in SidebarPage + give its main an id (sidebar.go)
In `internal/ui/shell/sidebar.go`, in `SidebarPage`'s `h.Body(...)` (line 81): add the skip link as the FIRST body child, and add `h.ID("main")` to the `.sb-canvas` main (line 90) so the target exists:
```
			h.Body(
				h.A(h.Class("skip-link"), h.Href("#main"), g.Text("Skip to content")),
				h.Div(h.Class("sb-root"),
					...
					h.Main(h.Class("sb-canvas"), h.ID("main"),
						h.Header(h.Class("sb-crumb"), g.Text(crumb)),
						p.Body,
					),
					...
```
**Verify**: `grep -n 'skip-link' internal/ui/shell/sidebar.go` → 1 hit; `grep -n 'h.ID("main")' internal/ui/shell/sidebar.go` → 1 hit.

### Step 8: A4 — the .skip-link CSS rule (basm.css)
Append at the END of `internal/web/assets/static/basm.css` (after the last existing rule, ~line 3218) a new Section banner + rule. Use the `--z-tooltip` token if it exists in `:root` (grep `--z-tooltip` in basm.css); if it does NOT exist, use the literal `z-index: 30`. Square corners, token colors, single-dash class:
```
/* ── Skip link — keyboard a11y; offscreen until focused ──────────── */
.skip-link {
  position: absolute;
  left: -9999px;
  top: 0;
  z-index: var(--z-tooltip, 30);
  padding: 8px 12px;
  background-color: var(--chrome);
  color: var(--chrome-fg);
  border: 2px solid var(--gold-deep);
  font-family: var(--font-mono);
  font-size: 12px;
  text-decoration: none;
}
.skip-link:focus { left: 8px; top: 8px; }
```
**Verify**: `grep -n '.skip-link' internal/web/assets/static/basm.css` → 2 hits (base + `:focus`). `tail -16 internal/web/assets/static/basm.css` shows the new block at the file end.

### Step 9: update tests (see Test plan), then run the full gate
**Verify** (all must pass): `gofmt -l internal/ui/shell/shell.go internal/ui/shell/sidebar.go internal/ui/composer.go internal/ui/shell/shell_test.go internal/ui/composer_test.go` (empty) → `go vet ./...` (exit 0) → `CGO_ENABLED=0 go build ./...` (exit 0) → `go test ./...` (all pass) → `git diff --check` (no output).

## Test plan

### shell_test.go — assert the skip link
In `internal/ui/shell/shell_test.go`, in `TestPage` add to the `want` slice (the rendered `Page` is already in `got`):
```
		`<a class="skip-link" href="#main">Skip to content</a>`,
```
Add a check that it is the first body child (the skip link must appear before `<header class="topbar"`): after the loop, assert `strings.Index(got, "skip-link") < strings.Index(got, "topbar")`. Pattern: the existing `if strings.Contains(...)` negative checks at the end of `TestPage`.
Optionally extend `TestPage` (or a new test mirroring it) to render `shell.SidebarPage` and assert the same skip-link string plus `id="main"` on `.sb-canvas` — see `internal/ui/shell/sidebar_test.go` for the SidebarPage render pattern.

### composer_test.go — assert tool-well labels (and fix broken asserts)
In `internal/ui/composer_test.go`, `TestComposer` (lines 20-21) currently asserts the OLD button strings:
```
`<button class="composer-tool" type="button"><img src="/static/icons/scroll.png" alt="" decoding="async"></button>`,
`<button class="composer-tool composer-sound" type="button"><img src="/static/icons/bell.png" alt="" decoding="async"></button>`,
```
Update them to the new disabled+labelled markup (match the exact attribute order your Step 5 produces — render once and copy if unsure):
```
`<button class="composer-tool" type="button" disabled aria-label="scroll (coming soon)"><img src="/static/icons/scroll.png" alt="" decoding="async"></button>`,
`<button class="composer-tool composer-sound" type="button" disabled aria-label="Sound (coming soon)"><img src="/static/icons/bell.png" alt="" decoding="async"></button>`,
```
Keep the decorative `alt=""` assertion. The existing `TestComposerDefaults` only greps for the icon `.png` substring, so it still passes.
**Verify**: `go test ./internal/ui/...` → ok.

### Storybook
The Composer story (`internal/feature/storybook/stories_chat.go:76-123`) renders the same component, so `TestAllStoriesRender` covers the markup change automatically. The component's props/variants are unchanged (no new prop), so no story prop-table edit is required; if the story documents tool-well behavior in a do/don't note, leave it — the wells were always non-functional in the catalog. **Verify**: `go test ./internal/feature/storybook/...` → ok.

## Done criteria
- [ ] `CGO_ENABLED=0 go build ./...` exit 0.
- [ ] `go vet ./...` exit 0.
- [ ] `go test ./...` all pass (incl. `internal/ui/shell`, `internal/ui`, `internal/feature/storybook`).
- [ ] `grep -n "cmsg-pending .cmsg-portrait" internal/web/assets/static/basm.css` → 2 hits (one inside the reduced-motion `@media`, lines 940-951).
- [ ] `awk 'NR>=1202 && NR<=1208' internal/web/assets/static/basm.css | grep -c "var(--chrome-fg)"` → 1; `... grep -c "var(--smoke)"` → 0.
- [ ] `grep -rn 'aria-pressed\|Aria("pressed"' internal/` returns exactly the new theme-toggle occurrence (Go) + the basm.js `setAttribute('aria-pressed'` line — and nothing else.
- [ ] `grep -n "skip-link" internal/ui/shell/shell.go internal/ui/shell/sidebar.go internal/web/assets/static/basm.css` → 1 + 1 + 2 hits.
- [ ] `gofmt -l <changed .go files>` empty; `git diff --check` no output.
- [ ] Only in-scope files changed (`git status --porcelain` lists only the seven scoped files + this plan + readme).
- [ ] plans/readme.md 079 row updated (add the row if it is not present yet, matching the existing column format).
- [ ] **VISUAL (both modes)** — run the app (it may already serve on 127.0.0.1:8090; else `go run . serve --http=127.0.0.1:8090`), open `/` (Home, where the recap hint shows on the dock):
  - In dark mode (`document.documentElement.className='theme-hearthwood dark'`): measure the `.recap-hint` computed color vs the `#dock` background — contrast ≥ 4.5:1 (was ~2.68:1). In light mode (`'theme-hearthwood light'`) it stays legible.
  - With OS reduce-motion enabled (or DevTools "Emulate prefers-reduced-motion: reduce"), trigger a pending chat turn — the `.cmsg-portrait` does NOT animate (no glow).
  - Tab from page load — the FIRST focusable element is the "Skip to content" skip link, and it becomes visible on focus; activating it moves focus to `#main`.
  - On the storybook surface (`/storybook`) the skip link is present and targets `#main` (the `.sb-canvas`).

## STOP conditions
- The drift check shows an in-scope file changed since `12a2ff5` and the "Current state" excerpt no longer matches — STOP and report the mismatch.
- `.recap-hint` turns out to have NO live render path after all (e.g. `web/templates/home.html:15` removed and no other caller) — then it is dead: SKIP A2, note it, and still do A1/A3/A4.
- Adding `aria-pressed` to the binary `.theme-toggle` proves semantically wrong (it is a toggle button, so `aria-pressed` is correct — but if your reading disagrees) — use a labelled name instead and report. For the 3-state palette cycle, `aria-pressed` is already excluded by design (Step 4 uses a labelled name).
- Any Verify command fails twice after a fix attempt — STOP and report the command + output.
- A step requires editing a file outside Scope to succeed — STOP and report which file and why.

## Maintenance notes
- **Reviewer scrutiny**: (1) the new reduced-motion line is INSIDE the `@media` block (not after its closing `}`); (2) `.recap-hint` uses `--chrome-fg` (dock-light), not `--smoke` (parchment); (3) `aria-pressed` is on the binary theme toggle only, never on the 3-state palette cycle; (4) the skip link is the first body child in BOTH page shells and its target `#main` exists in both (`Page` already had it; `SidebarPage` gained `id="main"` on `.sb-canvas`).
- **Future interaction**: plan 078 touches the same topbar buttons (responsiveness); this plan's `aria-pressed`/`aria-label` edits are purely additive attributes, so a later restructure must preserve them. If the composer tool wells ever become functional, remove the `disabled` + "(coming soon)" suffix and wire real `aria-label`/`aria-pressed` per tool, and update `composer_test.go`.
- **Deferred (not this plan)**: the drawer/burger focus trap (plan 078 A5); a shared `--z-tooltip` token may be introduced by another 075-085 plan — this plan's `.skip-link` uses `var(--z-tooltip, 30)` so it works whether or not the token exists yet. The dead `.model-choice`/`.model-detail` rules (basm.css:1654-1668) are a separate dead-code cleanup, not in scope here.
