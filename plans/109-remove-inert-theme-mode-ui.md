# Plan 109 — Remove the now-inert theme/mode UI + JS machinery

- **Status:** TODO
- **Priority:** P2
- **Effort:** M
- **Risk of the fix:** LOW
- **Depends on:** 108 (single Hearthwood dark theme — merged to main `21266d5`). This plan is written against post-108 main.
- **Written against commit:** `a067bed` (run `git rev-parse --short HEAD` on the executor's base; if it differs, the line numbers below are stale — match by the quoted code strings, which are stable, not by line number).

---

## Why this matters

Plan 108 collapsed the CSS to a single fixed theme: `color-scheme: dark`, every
token a single value, no `light-dark()`, no `:root.light/.dark`, no
forest/dungeon palette blocks. The look is correct and final.

But the **controls that used to switch theme/mode/palette are still in the UI**,
and they are now **no-ops**:

- The light/dark toggle button (`◑`) bottom-right of the chat shell calls
  `basmToggleTheme()`, which adds `.light` to `<html>`. After 108, `.light` has
  no CSS effect — clicking it does nothing visible. Confusing dead control.
- The storybook sidebar footer has a "Theme" mode toggle + three palette
  buttons (Hearth/Forest/Dungeon). The palette buttons call `basmSetPalette()`
  which adds `.theme-forest`/`.theme-dungeon` — classes 108 stripped of all
  token blocks, so they do nothing.
- Settings → **Appearance** is a whole tab whose only content is the palette
  picker — now entirely inert.
- The `<head>` no-flash script still reads `basm-theme` / `basm-palette` from
  `localStorage` and applies `.dark` + `.theme-hearthwood`. Both are redundant
  now (base `:root` already carries `color-scheme: dark`; `theme-hearthwood`
  was never a CSS selector — Hearthwood is the base `:root`).
- `basm.js` still defines `basmToggleTheme`, `basmUpdateThemeButtons`,
  `basmSetPalette`, `basmUpdatePaletteButtons` and wires two `DOMContentLoaded`
  hooks — all dead.
- The switcher CSS (`.theme-toggle`, `.appearance-themes`/`-theme-btn`,
  `.sb-foot-mode`/`-themes`, `.sb-theme-btn`, the `html.theme-*` highlight,
  `html.app .app-chrome`) styles markup that's about to vanish.

This plan **removes that dead machinery**. It is a pure deletion/cleanup change:
no behavior changes for the owner (the app is already always Hearthwood dark),
the rendered look is unchanged, and a handful of markup tests that assert the
now-removed controls get updated.

**This is the second half of the theming simplification. 108 = CSS collapse
(done). 109 = remove the inert UI/JS (this plan).**

---

## Repo conventions to follow

- Go UI is **gomponents** (`g "maragu.dev/gomponents"`, `h "...gomponents/html"`,
  some files dot-import the html package). `gofmt` is law — a PostToolUse hook
  runs `gofmt -w` on edited Go files, but still verify with `gofmt -l`.
- CSS lives in one file: `internal/web/assets/static/basm.css` (embedded via
  `embed.FS`). JS in `internal/web/assets/static/basm.js` (no build step).
- Tests are standard `testing`, table-driven, substring-asserting against
  rendered HTML. No assertion frameworks.
- **Surgical changes only.** Delete exactly the dead theme/mode/palette code.
  Do NOT "improve" adjacent code, do NOT touch the pre-existing retired-topbar
  dead code beyond the one orphan line called out in Step 6, do NOT restyle.
- When a deletion orphans an import/function, remove the orphan too — but verify
  with the build, don't guess.

---

## Files IN scope

1. `internal/web/assets/static/basm.js` — delete the 4 theme/palette functions + 2 hooks.
2. `internal/ui/shell/shell.go` — strip theme+palette from `noFlashScript` (keep dock state).
3. `internal/ui/shell/chatshell.go` — remove the `.app-chrome` toggle div.
4. `internal/feature/settingscards/settingsfocus.go` — remove the Appearance tab/section + its 2 helper funcs.
5. `internal/web/storybook.go` — remove the footer mode-toggle + palette buttons + `paletteBtn` helper.
6. `internal/feature/storybook/stories_navigation.go` — remove the theme-toggle from the sidebar story fixture.
7. `internal/web/assets/static/basm.css` — delete the orphaned switcher CSS rules.
8. Tests: `internal/web/home_test.go`, `internal/feature/settingscards/settingsfocus_test.go`, `internal/ui/shell/shell_test.go`.
9. Docs: `DESIGN.md` (topbar-toggle line), `internal/web/assets/css_tokens_test.go` (stale comment only).

## Files explicitly OUT of scope — DO NOT TOUCH

- **`basm.css` token block / `color-scheme` / forest-dungeon** — that was plan
  108; it's done. Do not re-collapse, do not re-add anything.
- **`internal/self/knowledge.md`** — its "palette" mentions are all the
  **command palette** (the `/`-command nav launcher), NOT theming. It does not
  describe theme/mode, so it needs no change. Confirm with
  `grep -niE "light.?dark|theme toggle|appearance|forest|dungeon" internal/self/knowledge.md`
  → expect no theming hits (only command-palette lines).
- **`htmlClass := "app"` in chatshell.go** — KEEP. There are ~31 `html.app …`
  CSS rules that depend on it; only the `.app-chrome` child rule goes.
- **`--wood-planks`** token and `TestThemePaletteBlocks` assertions — leave as-is.
- **The retired-topbar `@media` block** in basm.css (`.topnav-drawer*`,
  `.topbar*`) — pre-existing dead code from plan 089. Touch ONLY the single
  `.topbar .theme-toggle` line per Step 6; leave the rest alone.
- **`DESIGN.md:193`** "shifts the palette from Forest at Dusk…" — that is
  historical context about how the *Hearthwood* palette derived from the old
  repo palette, NOT the forest/dungeon theme variant. Leave it.
- **`cards_test.go` `TestUiCardPalette`** — `ucard-palette` is the UI card
  chooser, not a theme palette. Not in scope.

---

## Steps

> Lines shift as you delete. After each file, re-grep. Match by the quoted
> strings below (stable), not by line number.

### Step 1 — `basm.js`: delete the theme + palette function blocks

Open `internal/web/assets/static/basm.js`. Delete the **entire** region from the
comment `// ── Light/dark theme toggle ──…` through the second palette
`DOMContentLoaded` hook — i.e. everything from this line:

```js
// ── Light/dark theme toggle ────────────────────────────────────────
```

down to and including:

```js
document.addEventListener('DOMContentLoaded', basmUpdatePaletteButtons);
```

That removes: the toggle comment block, `window.basmToggleTheme`,
`basmUpdateThemeButtons`, its hook, the palette comment block,
`window.basmSetPalette`, `basmUpdatePaletteButtons`, and its hook.

**Keep** the file header (lines 1–3, `/* basm.js — … just the platform. */`) and
everything from `// ── Chatbar height → CSS custom property ──…` onward. Leave
exactly one blank line between the header comment and the chatbar comment.

Verify:
```
grep -cE "basmToggleTheme|basmSetPalette|basmUpdateThemeButtons|basmUpdatePaletteButtons" internal/web/assets/static/basm.js
```
→ **0**.

### Step 2 — `shell.go`: strip theme/palette from the no-flash script

In `internal/ui/shell/shell.go`, replace the `noFlashScript` constant. Current:

```go
// noFlashScript applies the saved theme + dock state before first paint, so the
// page never flashes the wrong colour scheme. Ported verbatim from layout.html.
const noFlashScript = `(function(){var d=document.documentElement;d.classList.add(localStorage.getItem('basm-theme')||'dark');d.classList.add('theme-'+(localStorage.getItem('basm-palette')||'hearthwood'));if(localStorage.getItem('basm-dock-full')==='1')d.classList.add('dock-full');var w=parseInt(localStorage.getItem('basm-dock-w'),10);if(w>=280&&w<=720)d.style.setProperty('--sidebar-w',w+'px');}());`
```

Replace with (drop the two `classList.add` calls for theme + palette; keep the
dock-full + sidebar-w logic; update the comment):

```go
// noFlashScript applies the saved dock state before first paint, so the page
// never flashes the wrong sidebar width. The theme is fixed Hearthwood dark
// (color-scheme: dark in basm.css), so no theme/palette class is applied.
const noFlashScript = `(function(){var d=document.documentElement;if(localStorage.getItem('basm-dock-full')==='1')d.classList.add('dock-full');var w=parseInt(localStorage.getItem('basm-dock-w'),10);if(w>=280&&w<=720)d.style.setProperty('--sidebar-w',w+'px');}());`
```

Verify:
```
grep -cE "basm-theme|basm-palette|theme-'" internal/ui/shell/shell.go
```
→ **0**. And `grep -c "basm-dock-full" internal/ui/shell/shell.go` → **1** (dock kept).

### Step 3 — `chatshell.go`: remove the `.app-chrome` toggle

In `internal/ui/shell/chatshell.go`, delete the `.app-chrome` block (the comment
+ the div):

```go
				// Global chrome: the light/dark toggle used to live in the rail footer.
				// The rail is gone, so it moves here as a low-key fixed control.
				h.Div(h.Class("app-chrome"),
					h.Button(h.Class("theme-toggle"), h.Type("button"),
						g.Attr("onclick", "basmToggleTheme()"),
						h.Title("Toggle light/dark mode"),
						h.Aria("label", "Toggle light/dark mode"), h.Aria("pressed", "false"),
						g.Text("◑")),
				),
```

This sits inside a parent `g.Group`/`h.Div` list — remove the whole node
(including its trailing comma) so the surrounding slice stays valid Go.

Also update the doc comment near the top of the file (it currently reads
`// as a fixed overlay (plan 098). The global theme toggle lives in .app-chrome.`)
— drop the second sentence so it no longer references the removed toggle:

```go
// as a fixed overlay (plan 098).
```

Verify: `grep -cE "app-chrome|theme-toggle|basmToggleTheme" internal/ui/shell/chatshell.go` → **0**.
`g` and `h` imports are still used elsewhere in the file — the build in Step 8
confirms no orphaned import.

### Step 4 — `settingsfocus.go`: remove the Appearance tab + section

Three edits in `internal/feature/settingscards/settingsfocus.go`:

**(a)** In `settingsTabs`, drop the Appearance tab:
```go
	defs := []t{{"Profile", "profile"}, {"Appearance", "appearance"}, {"Models", "models"}, {"Heads", "heads"}}
```
→
```go
	defs := []t{{"Profile", "profile"}, {"Models", "models"}, {"Heads", "heads"}}
```

**(b)** In `BuildSettingsFocus`, remove `"appearance"` from the known-sections
guard and delete its data-switch case:
```go
	switch section {
	case "models", "heads", "appearance":
		// known sections
```
→
```go
	switch section {
	case "models", "heads":
		// known sections
```
and delete:
```go
	case "appearance":
		// static — no data fetch
```

**(c)** In `SettingsFocus`, delete the appearance render case:
```go
	case "appearance":
		content = AppearanceSection()
```

**(d)** Delete the two now-unused helper functions entirely:
`AppearanceSection()` (the `// AppearanceSection renders …` comment + func) and
`appearanceThemeBtn()` (the `// appearanceThemeBtn renders …` comment + func).

Verify:
```
grep -cE "appearance|Appearance" internal/feature/settingscards/settingsfocus.go
```
→ **0**. (Section comment line 4 lists "Models, Heads, Appearance" — update that
doc comment too: change `Models, Heads, Appearance` to `Models, Heads`.) After
that, the grep is 0.

### Step 5 — `storybook.go`: remove the footer switcher + `paletteBtn`

In `internal/web/storybook.go`:

**(a)** Delete the `paletteBtn` helper (comment + func):
```go
// paletteBtn renders one footer palette button wired to basmSetPalette.
func paletteBtn(key, label string) g.Node {
	return hh.Button(hh.Class("sb-theme-btn"), hh.Type("button"),
		g.Attr("data-theme", key), g.Attr("onclick", "basmSetPalette('"+key+"')"),
		hh.Title("Theme: "+label), g.Text(label))
}
```

**(b)** In `sidebarFor`'s `Footer`, delete the `sb-foot-row` (Theme label + mode
toggle) and `sb-foot-themes` (palette buttons) divs, keeping only the count:
```go
		Footer: g.Group([]g.Node{
			hh.Div(hh.Class("sb-foot-row"),
				hh.Span(hh.Class("sb-foot-label"), g.Text("Theme")),
				hh.Button(hh.Class("theme-toggle sb-foot-mode"), hh.Type("button"),
					g.Attr("onclick", "basmToggleTheme()"),
					hh.Title("Toggle day / night"), hh.Aria("label", "Toggle light/dark mode"),
					g.Text("◑")),
			),
			hh.Div(hh.Class("sb-foot-themes"),
				paletteBtn("hearthwood", "Hearth"),
				paletteBtn("forest", "Forest"),
				paletteBtn("dungeon", "Dungeon"),
			),
			hh.Div(hh.Class("sb-foot-count"), g.Text(strconv.Itoa(len(storybook.Stories()))+" components")),
		}),
```
→
```go
		Footer: hh.Div(hh.Class("sb-foot-count"), g.Text(strconv.Itoa(len(storybook.Stories()))+" components")),
```
(A single node is fine for the `Footer g.Node` field — no need for `g.Group`.)

`strconv`, `hh`, `g`, `storybook` imports all stay used. Verify:
```
grep -cE "paletteBtn|sb-foot-row|sb-foot-themes|sb-foot-mode|sb-theme-btn|basmToggleTheme|basmSetPalette" internal/web/storybook.go
```
→ **0**.

### Step 6 — `stories_navigation.go`: drop the toggle from the sidebar fixture

In `internal/feature/storybook/stories_navigation.go`, the `sidebarStory()`
fixture footer has a theme-toggle button + a Home link. Remove just the button,
keep the Home link:
```go
	footer := g.Group([]g.Node{
		h.Button(h.Class("theme-toggle"), h.Type("button"),
			g.Attr("onclick", "basmToggleTheme()"),
			h.Title("Toggle light/dark mode"),
			h.Aria("label", "Toggle light/dark mode"),
			h.Aria("pressed", "false"),
			g.Text("◑"),
		),
		h.A(h.Href("/"), g.Text("Home")),
	})
```
→
```go
	footer := g.Group([]g.Node{
		h.A(h.Href("/"), g.Text("Home")),
	})
```
Verify: `grep -cE "theme-toggle|basmToggleTheme" internal/feature/storybook/stories_navigation.go` → **0**.

### Step 7 — `basm.css`: delete the orphaned switcher rules

Delete these rule blocks (match by selector; each is contiguous):

1. The **theme-toggle** block and its comment header:
   ```css
   /* ── Theme toggle ────────────────────────────────────────────────── */

   .theme-toggle { … }
   .theme-toggle:hover { … }
   ```
   (Leave the `/* ── Reduced motion ── */` section that follows.)

2. In the storybook-footer group, the four orphaned rules
   (`.sb-foot-row`, `.sb-foot-label`, `.sb-foot-mode`, `.sb-foot-themes`,
   `.sb-theme-btn`, `.sb-theme-btn.is-active`). **Keep** `.sb-foot` and
   `.sb-foot-count` (still used). Concretely, delete these lines:
   ```css
   .sb-foot-row { … }
   .sb-foot-label { … }
   .sb-foot-mode { … }
   .sb-foot-themes { … }
   .sb-theme-btn { … }
   .sb-theme-btn.is-active { … }
   ```

3. The **Appearance settings** block — delete from the section comment
   `/* ══ Section: Appearance settings — palette picker ═…` through the closing
   `}` of the `html.theme-dungeon …` highlight rule:
   ```css
   /* ══ Section: Appearance settings — palette picker ═══…
      … (comment) … */
   .appearance-themes { … }
   .appearance-theme-btn { … }
   .appearance-theme-btn:hover { … }
   html.theme-hearthwood .appearance-theme-btn[data-theme="hearthwood"],
   html.theme-forest .appearance-theme-btn[data-theme="forest"],
   html.theme-dungeon .appearance-theme-btn[data-theme="dungeon"] { … }
   ```
   **Keep** the preceding `/* rail variant inherits #dock's fixed width … */`
   comment — it belongs to the dock rail above, not to Appearance.

4. The **app-chrome** block and its comment:
   ```css
   /* ── App chrome: relocated theme toggle (plan 102) ───…
   html.app .app-chrome { … }
   ```

5. The orphan reference inside the retired-topbar `@media` block — delete the
   single line (and only this line; leave the rest of the `@media` block):
   ```css
   /* Bump theme-button touch targets to 44px on mobile; keep visual padding. */
   .topbar .theme-toggle { min-height: 44px; padding: 0 var(--space-3); }
   ```
   (Delete both the comment line and the rule line.)

Verify:
```
grep -cE "\.theme-toggle|\.app-chrome|\.appearance-themes|\.appearance-theme-btn|\.sb-theme-btn|\.sb-foot-mode|\.sb-foot-themes|\.sb-foot-row|\.sb-foot-label|theme-hearthwood|theme-forest|theme-dungeon" internal/web/assets/static/basm.css
```
→ **0**. And confirm the kept rules survive:
```
grep -cE "\.sb-foot \{|\.sb-foot-count" internal/web/assets/static/basm.css
```
→ **2**.

### Step 8 — Update the markup tests + docs

**(a) `internal/web/home_test.go`** — the home page no longer renders the
toggle. Remove from `ExpectedContent`:
```go
			// theme toggle relocated to .app-chrome (plan 102)
			`class="theme-toggle"`,
```
and ADD to `NotExpectedContent` (locks the removal as a regression guard):
```go
			`class="theme-toggle"`, // theme/mode switcher removed (plan 109 — single fixed theme)
```

**(b) `internal/feature/settingscards/settingsfocus_test.go`** — delete the
entire `TestSettingsFocusAppearanceSection` function (the `// TestSettingsFocus…`
comment + func, asserting the appearance picker). It no longer exists. The
other section tests (profile/models/heads, the `k-tabs` assertions) are
untouched.

**(c) `internal/ui/shell/shell_test.go`** — the no-flash script no longer
applies theme/palette. Remove these three `want` strings:
```go
		`localStorage.getItem('basm-theme')`,
		`localStorage.getItem('basm-palette')`,
		`'theme-'`,
```
and ADD one assertion for the retained dock logic so the script stays under test:
```go
		`localStorage.getItem('basm-dock-full')`,
```

**(d) `internal/web/assets/css_tokens_test.go`** — fix the now-stale comment on
`TestThemePaletteBlocks`. It currently ends:
```go
// gone. (.theme-toggle / --wood-planks remain until plan 109 removes the UI.)
```
→
```go
// gone. (--wood-planks remains — it's a grain token, not theme UI.)
```
(Comment only — do NOT change the test's assertions.)

**(e) `DESIGN.md`** — update the profile-section sentence that references the
topbar toggle. Current:
```
and Balaur head picker (16 options) · a light/dark theme toggle in the topbar persisted to
`localStorage` · switchable head personas: a dock head switcher
```
→ remove the toggle clause:
```
and Balaur head picker (16 options) · switchable head personas: a dock head switcher
```

### Step 9 — Gates

Run from the worktree root:
```
gofmt -l internal/ui/shell/shell.go internal/ui/shell/chatshell.go internal/feature/settingscards/settingsfocus.go internal/web/storybook.go internal/feature/storybook/stories_navigation.go internal/web/home_test.go internal/feature/settingscards/settingsfocus_test.go internal/ui/shell/shell_test.go internal/web/assets/css_tokens_test.go
go vet ./...
CGO_ENABLED=0 go build ./...
go test ./...
git diff --check
```
All must be clean / exit 0. `go test ./...` must show no FAIL.

### Step 10 — Visual (if a browser/model is available; else say so)

Build and serve against a copy of `pb_data`, force-load the chat shell and the
storybook, and confirm:
- The chat shell renders identically, with **no** `◑` toggle bottom-right.
- `/storybook` sidebar footer shows only the component count (no Theme row / no
  palette buttons).
- `/ui/show/settings?section=profile` tab strip shows **Profile · Models ·
  Heads** (no Appearance tab).
- The whole app is still Hearthwood dark.

If no browser is available, state that explicitly — the change is deletion-only
and fully covered by the markup tests, so deterministic verification is
sufficient, but note the visual was not captured.

---

## Done criteria (machine-checkable)

```bash
# No theme/mode/palette JS remains:
grep -rcE "basmToggleTheme|basmSetPalette|basmUpdateThemeButtons|basmUpdatePaletteButtons" \
  internal/web/assets/static/basm.js internal/ui/shell internal/web/storybook.go \
  internal/feature/storybook/stories_navigation.go internal/feature/settingscards/settingsfocus.go
# → every file reports 0

# No switcher markup classes remain in Go:
grep -rnE "app-chrome|appearance-theme|sb-foot-mode|sb-foot-themes|sb-theme-btn|\"theme-toggle\"|class=\"theme-toggle" \
  internal --include=*.go | grep -v _test
# → only home_test.go's NotExpectedContent guard may match; no production .go file

# No switcher CSS remains:
grep -cE "\.theme-toggle|\.app-chrome|\.appearance-theme|\.sb-theme-btn|\.sb-foot-mode|\.sb-foot-themes|\.sb-foot-row|theme-forest|theme-dungeon" \
  internal/web/assets/static/basm.css
# → 0

# No-flash script keeps dock, drops theme:
grep -c "basm-dock-full" internal/ui/shell/shell.go   # → 1
grep -cE "basm-theme|basm-palette" internal/ui/shell/shell.go   # → 0

# Gates:
gofmt -l <edited go files>      # → empty
go vet ./...                    # → exit 0
CGO_ENABLED=0 go build ./...    # → exit 0
go test ./...                   # → all ok, no FAIL
git diff --check                # → no output
```

Plus: `git status --porcelain` shows only the in-scope files (9 source/doc +
3 test files listed above; nothing else).

---

## Test plan

No new test files. Existing markup tests are updated to match the removed
controls, and `home_test.go` gains a `NotExpectedContent` guard so the toggle
can't silently come back. `TestSettingsFocusAppearanceSection` is deleted (its
subject is gone). `shell_test.go` swaps its theme assertions for a dock-state
assertion so the no-flash script stays under test. The full suite
(`go test ./...`) is the regression gate — it already covers home rendering,
settings sections, the shell page, and the storybook sidebar.

---

## Maintenance note

After this lands, Balaur has exactly one theme with no switching surface. If a
future change wants to reintroduce theming, it should restore both halves
(token `light-dark()` pairs + `color-scheme` in 108's territory, and the
controls here) — not just one. The `html.app` class and the dock-state portion
of `noFlashScript` are unrelated to theming and must stay. Watch in review:
that no `g.Group` wrapper was left empty, no import was orphaned (the build
catches this), and that only the single `.topbar .theme-toggle` line was removed
from the retired-topbar `@media` block (the rest is pre-existing dead code,
out of scope).

---

## Escape hatches

- **If `grep` finds a `basmToggleTheme`/`basmSetPalette` call site NOT listed in
  this plan** (i.e. outside the 5 mapped Go files + basm.js), STOP and report —
  the surface drifted since `a067bed` and the plan needs updating.
- **If removing the Appearance tab breaks a test that asserts a specific tab
  count or the "Appearance" label** beyond `TestSettingsFocusAppearanceSection`,
  STOP and report which test — do not weaken an unrelated assertion to make it
  pass.
- **If `knowledge.md` turns out to describe theme/mode/palette** (not just the
  command palette) after all, STOP and report — that would mean a self-knowledge
  update belongs in this change, which changes scope.
- **If the build reports an orphaned import** you can't cleanly resolve by
  removing the dead code, STOP and report rather than restructuring the file.
