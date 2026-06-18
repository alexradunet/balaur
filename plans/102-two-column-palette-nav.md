# Plan 102 — Two-column shell + `/`-command palette navigation

- **Written against commit:** `4e933d7` (re-anchor onto 101 before executing)
- **Priority:** P1 · **Effort:** L · **Risk:** MED–HIGH
- **Depends on:** **101** (the `/ui/show` door must be non-polluting — the palette fires it)
- **Status:** TODO

> **Owner directive (verbatim):** *"I want to remove the left sidebar and just
> leave the main chat and the right artifact page … If the user wants to open a
> specific page in artifact, we will add some sort of autocomplete to the
> composer so that on /quests or /settings to display the page artifact …"*
> Decision (locked via AskUserQuestion): **delete the rail entirely** — no
> burger, no drawer; the `/`-palette is the only launcher.

## Why this matters

After the canvas pivot (098–100), the left rail's only remaining job is to
**launch an artifact you haven't opened yet** — re-opening, sub-navigation, and
active-state already live in chips and in-panel tabs. A launcher is exactly what
a `/`-palette in the composer does better: it is keyboard-first, it is where the
owner's hands already are, and removing the rail hands a whole column back to
chat + panel. This is the logical conclusion of the canvas direction.

The composer is already wired for this: its textarea is **two-way bound** to a
Datastar signal (`data-bind:message`), and the composer root **seeds** that
signal (`data-signals:message="''"`). So a palette can:

- **appear / filter** purely client-side via `data-show` expressions over
  `$message` (presentational — no round-trip per keystroke), and
- **navigate** by firing `@get('/ui/show/{type}')` (server-driven, the
  non-polluting door from plan 101) and clearing the draft.

The only new JavaScript is ~8 lines extending the existing `balaurSubmitOnEnter`
so that pressing Enter on a `/…` draft selects the first visible command instead
of posting it as a chat message. Everything else is SSR gomponents.

This plan also **retires the mobile rail chrome that plan 100 added** (the
`html.app .app-topbar` burger + the `html.app .sb-side` off-canvas drawer) —
there is no rail left to reveal. Mobile navigation becomes: chat full-width →
`/`-palette → the panel slides in as the 098 overlay. The generic `shell.Sidebar`
atom and the storybook's own `.sb-side`/`basmToggleNav` Page drawer are **NOT**
touched (the storybook shell still uses them).

## Current state (read these first)

### `internal/ui/shell/chatshell.go` — the home shell (entire file)

```go
type ChatShellProps struct {
	Title   string
	Sidebar g.Node
	Dock    g.Node
	Panel   g.Node // the single-active right panel (chat.Panel)
}

func ChatShell(p ChatShellProps) g.Node {
	return g.Group([]g.Node{
		g.Raw("<!doctype html>"),
		h.HTML(
			h.Lang("en"), h.Class("app"),
			h.Head(pageHead(), h.TitleEl(g.Text(p.Title+" · Balaur"))),
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
		),
	})
}
```

`ChatShell` is used **only** by `internal/web/home.go` (confirmed: sole caller),
so changing `ChatShellProps` is safe.

### `internal/web/home.go` — `domainSidebar()` + wiring (lines 26-122)

`composerNode` (lines 26-34) renders the live composer; `domainSidebar()`
(lines 49-85) builds the rail; `homePage` (lines 99-122) assembles the shell:

```go
func composerNode(d homeData) g.Node {
	return ui.Composer(ui.ComposerProps{
		AvatarSrc:   d.SoulAvatarURL,
		Placeholder: d.ChatPlaceholder,
		PostURL:     "/ui/chat",
		ID:          "chat-draft",
		Disabled:    !d.ChatReady,
	})
}
...
	page := shell.ChatShell(shell.ChatShellProps{
		Title:   "Home",
		Sidebar: shell.Sidebar(domainSidebar()),
		Dock:    dockNode,
		Panel:   h.restoredPanelNode(),
	})
```

`domainSidebar()`'s items are the palette's command set (same targets):
Quests→`/ui/show/quests`, Life→`/ui/show/lifelog`,
Knowledge→`/ui/show/memory?category=fact`, Skills→`/ui/show/skills`,
Settings→`/ui/show/settings?section=profile`.

### `internal/ui/composer.go` — the composer atom (lines 31-152)

`ComposerProps` (lines 31-50) and `Composer` (lines 58-152). Key facts:
- Live textarea (line 103-105): `h.Name("message")`, and when `live`:
  `data-bind:message`, `onkeydown="balaurSubmitOnEnter(event)"`, `Required`, `AutoFocus`.
- Root (line 131-150): `class="composer"`, optional `ID`, and when `live`
  `data-signals:message="''"`; then the corner brackets, `.composer-top`, and
  `main` (the form). The palette must mount **inside** this root so it can be
  absolutely positioned against the composer.

### `internal/web/assets/static/basm.js` — `balaurSubmitOnEnter` (lines 135-140)

```js
window.balaurSubmitOnEnter = function (event) {
  if (event.key !== 'Enter' || event.shiftKey || event.altKey ||
      event.ctrlKey || event.metaKey || event.isComposing) return;
  event.preventDefault();
  event.currentTarget.form?.requestSubmit();
};
```

### CSS to change in `internal/web/assets/static/basm.css`

- `--w-panel: 480px` (line 167) — unchanged here (plan 103 makes it dynamic).
- `html.app .app-shell { grid-template-columns: 274px 1fr var(--w-panel); … }`
  (lines 3307-3312) — the **3-column** grid → **2-column** `1fr var(--w-panel)`.
- `html.app .app-topbar { display: none; }` (line 3304) — **delete** (topbar gone).
- `@media (max-width: 720px) { … }` block (lines 3381-3417) — contains
  `html.app .app-topbar` (3385-3398) and `html.app .sb-side` off-canvas drawer
  (3401-3407) — **delete both** (rail/topbar gone). **KEEP** the 1-column grid
  collapse and `html.app.panel-open #panel.app-panel { transform: translateX(0); }`
  (3413) — the panel still slides in as a mobile overlay (098 behavior).

### `internal/feature/storybook/stories_navigation.go` — `sidebarStory` (lines 110-180)

Its comment (110-111) says it *"Mirror[s] the live `domainSidebar()` helper in
home.go so the story documents the rail."* That helper is being deleted, so the
comment is stale. The generic `shell.Sidebar` atom it renders is **still used**
by `internal/web/storybook.go:62` (the storybook's own left nav), so keep the
atom + the story — just reframe the comment (Step 6).

## The change — ordered steps

### Step 1 — Re-anchor + drift check

This plan must execute **on top of plan 101**. Confirm 101 landed:

```
grep -rn "/ui/panel" --include=*.go . ; test $? -ne 0    # 101 deleted the second door
grep -n 'typ == "close"' internal/web/show.go            # 101's merged uiShow
git rev-parse --short HEAD
```

If `/ui/panel` still exists, **STOP** — execute 101 first.

### Step 2 — New atom: `ui.CommandPalette`

Create `internal/ui/command_palette.go`:

```go
package ui

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// CommandItem is one entry in the composer command palette: a display Label, the
// lowercase slash Key the owner types to filter to it, an optional pixel Icon
// stem, and the @get URL that opens it in the panel (the non-polluting /ui/show
// door, plan 101).
type CommandItem struct {
	Label string
	Key   string
	Icon  string
	URL   string
}

// CommandPalette renders the composer's /-command menu. It appears when the
// draft starts with "/" and filters as the owner types — both via data-show
// expressions over the $message signal (presentational; no round-trip). The
// menu mounts inside the composer (CSS anchors it above the textarea). Selecting
// an item fires @get(URL) and clears the draft ($message = ''), which also hides
// the menu (startsWith('/') becomes false). Enter-to-select is handled by
// balaurSubmitOnEnter (basm.js).
func CommandPalette(items []CommandItem) g.Node {
	list := []g.Node{h.Class("cmd-list")}
	for _, it := range items {
		// Prefix match on the typed query (everything after the leading '/').
		show := "$message.startsWith('/') && '" + it.Key +
			"'.startsWith($message.slice(1).toLowerCase().trim())"
		item := []g.Node{
			h.Class("cmd-item"),
			g.Attr("data-show", show),
			h.Href(it.URL), // no-JS fallback
			g.Attr("data-on:click__prevent", "@get('"+it.URL+"'); $message = ''"),
		}
		if it.Icon != "" {
			item = append(item, h.Img(h.Class("cmd-item-icon"),
				h.Src("/static/icons/"+it.Icon+".png"), h.Alt(""), g.Attr("decoding", "async")))
		}
		item = append(item,
			h.Span(h.Class("cmd-item-label"), g.Text(it.Label)),
			h.Span(h.Class("cmd-item-key"), g.Text("/"+it.Key)),
		)
		list = append(list, h.A(item...))
	}
	return h.Div(
		h.Class("cmd-palette"),
		g.Attr("data-show", "$message.startsWith('/')"),
		h.Div(list...),
	)
}
```

> Datastar note: `data-show` evaluates as a JS expression with signals as
> `$name`. `'quests'.startsWith($message.slice(1).toLowerCase().trim())` is valid
> JS; when the draft is exactly `/`, the slice is `""` and every item shows
> (the full menu) — the desired behavior. Do **not** server-filter; the command
> set is tiny and fixed, and a round-trip per keystroke would violate the
> SSR-but-snappy intent.

### Step 3 — Mount the palette inside `ui.Composer`

In `internal/ui/composer.go`, add a slot to `ComposerProps`:

```go
	// Palette is an optional command menu rendered inside the composer root so
	// CSS can anchor it above the textarea (plan 102). It self-shows via Datastar.
	Palette g.Node
```

In `Composer` (the root assembly, after `main` at line ~149), append the palette
when present:

```go
	root = append(root,
		h.Span(h.Class("dlg-corner dlg-corner-tl")),
		// … existing corners + composer-top + main …
		main,
	)
	if p.Palette != nil {
		root = append(root, p.Palette)
	}
	return h.Div(root...)
```

(The palette is a child of `.composer` so `position:absolute` anchors to it.)

### Step 4 — Replace `domainSidebar()` with the palette command set; drop the rail

In `internal/web/home.go`:

1. **Delete `domainSidebar()`** (lines 49-85) entirely.
2. Add a `commandPaletteNode()` helper building the same five targets:

```go
// commandPaletteNode is the composer /-command menu: the navigation launcher
// that replaced the domain rail (plan 102). Each item opens its artifact in the
// panel via the non-polluting /ui/show door (plan 101).
func commandPaletteNode() g.Node {
	return ui.CommandPalette([]ui.CommandItem{
		{Label: "Quests", Key: "quests", Icon: "scroll", URL: "/ui/show/quests"},
		{Label: "Life", Key: "life", Icon: "orb", URL: "/ui/show/lifelog"},
		{Label: "Knowledge", Key: "knowledge", Icon: "tome", URL: "/ui/show/memory?category=fact"},
		{Label: "Skills", Key: "skills", Icon: "key", URL: "/ui/show/skills"},
		{Label: "Settings", Key: "settings", URL: "/ui/show/settings?section=profile"},
	})
}
```

3. Wire it into `composerNode` (+ a discovery hint on the composer foot):

```go
func composerNode(d homeData) g.Node {
	return ui.Composer(ui.ComposerProps{
		AvatarSrc:   d.SoulAvatarURL,
		Placeholder: d.ChatPlaceholder,
		Hint:        "enter speaks · / for pages",
		PostURL:     "/ui/chat",
		ID:          "chat-draft",
		Disabled:    !d.ChatReady,
		Palette:     commandPaletteNode(),
	})
}
```

4. In `homePage`, drop the `Sidebar` field from the `ChatShellProps` literal.

### Step 5 — Two-column `ChatShell`

In `internal/ui/shell/chatshell.go`:

1. Remove the `Sidebar g.Node` field from `ChatShellProps`.
2. In `ChatShell`, delete the `.app-topbar` `<header>` and the `.sb-backdrop`
   `<div>`, drop `p.Sidebar` from the `.app-shell` grid, and **relocate the theme
   toggle** here (see the boxed note below — it currently lives only in the rail
   footer, which is being deleted). The body becomes:

```go
h.Body(
	h.A(h.Class("skip-link"), h.Href("#chat"), g.Text("Skip to content")),
	h.Div(h.Class("app-shell"),
		h.Aside(h.ID("dock"), h.Class("app-dock"), p.Dock),
		h.Aside(h.ID("panel"), h.Class("app-panel"), p.Panel),
	),
	// Global chrome: the light/dark toggle used to live in the rail footer.
	// The rail is gone, so it moves here as a low-key fixed control.
	h.Div(h.Class("app-chrome"),
		h.Button(h.Class("theme-toggle"), h.Type("button"),
			g.Attr("onclick", "basmToggleTheme()"),
			h.Title("Toggle light/dark mode"),
			h.Aria("label", "Toggle light/dark mode"), h.Aria("pressed", "false"),
			g.Text("◑")),
	),
),
```

> **Theme-toggle relocation (required, not optional).** Today the
> `class="theme-toggle"` button exists **only** inside `domainSidebar()`'s Footer
> (`home.go:75-81`). Step 4 deletes `domainSidebar()`, so without this move the
> home page loses its only light/dark switch (a real UX regression) **and**
> `TestHomeFullChat` — which keeps `class="theme-toggle"` in `ExpectedContent`
> (Step 9) — fails the gate. Rendering it in `ChatShell`'s `.app-chrome` keeps
> the toggle on the page and the test assertion valid. (The old rail-footer
> `<a href="/">Home</a>` is dropped — on a single-page app, reload = home, and
> the catch-all already redirects unknown paths home.)

3. Update the doc comment (the off-canvas-drawer paragraph is no longer true) —
   describe the two-column chat+panel layout and the composer-palette nav.

4. **Fix the stale empty-panel copy** (cross-plan find): `chat.Panel`'s
   placeholder (`internal/ui/chat/panel.go:25`) reads *"Pick a domain from the
   rail, or ask Balaur to show you something."* — the rail is gone. Change it to
   *"Type / for pages, or ask Balaur to show you something."* This plan owns the
   rail removal, so it owns this copy fix.

`basmToggleNav` is **still referenced** by `internal/ui/shell/sidebar.go` (the
storybook Page drawer), so do **not** delete it from `basm.js`.

### Step 6 — CSS: 2-column grid; delete the dead rail/topbar rules; palette styling

In `internal/web/assets/static/basm.css`:

1. `html.app .app-shell` (lines 3307-3312): grid columns
   `274px 1fr var(--w-panel)` → `1fr var(--w-panel)`.
2. Delete `html.app .app-topbar { display: none; }` (line 3304).
3. In the `@media (max-width: 720px)` block (3381-3417): delete the
   `html.app .app-topbar` rule (3385-3398) and **all** the now-dead rail/drawer
   rules — the `html.app .sb-side` off-canvas rule (3401-3407) **and its
   companions** `html.app.sb-nav-open .sb-side` and `html.app .sb-backdrop.is-open`
   (~3408-3409), which only exist to reveal the deleted home drawer. **Keep** the
   1-column grid collapse (chat full-width) and the
   `html.app.panel-open #panel.app-panel { transform: translateX(0); }` overlay
   rule (3413). After editing, the mobile layout is: one column (chat), panel
   slides in as the `panel-open` overlay. Verify no `html.app .sb-` rule survives:
   `grep -n "html.app .sb-\|html.app.sb-nav-open" internal/web/assets/static/basm.css` → 0 hits.
4. Add command-palette chrome. Match existing tokens/idiom (reference the
   `.choices-panel` / `.sb-nav-item` styling for color + the `--space-*` scale +
   `--z-overlay` for layering). The shape:

```css
html.app .composer { position: relative; }   /* anchor for the palette */
.cmd-palette {
  position: absolute;
  left: var(--space-3); right: var(--space-3);
  bottom: calc(100% + var(--space-2));
  z-index: var(--z-overlay);
  /* parchment/wood surface + gold-deep border + bevel, per the design tokens */
  max-height: 40vh; overflow-y: auto;
}
.cmd-list { display: flex; flex-direction: column; }
.cmd-item { display: flex; align-items: center; gap: var(--space-2);
  padding: var(--space-2) var(--space-3); text-decoration: none; }
.cmd-item:hover { /* gold-tinted hover, like .sb-nav-item:hover */ }
.cmd-item-icon { width: 20px; height: 20px; image-rendering: pixelated; }
.cmd-item-label { flex: 1; }
.cmd-item-key { font-family: var(--font-pixel); font-size: 12px; color: var(--gold); opacity: .7; }
```

(`data-show` toggles `display` so hidden items collapse — `.cmd-item` needs no
explicit hidden style.)

5. Add chrome for the relocated theme toggle — a low-key fixed control so it
   doesn't crowd the chat:

```css
html.app .app-chrome {
  position: fixed; right: var(--space-2); bottom: var(--space-2);
  z-index: var(--z-sticky);
}
/* .theme-toggle already has chrome from the old rail footer — reuse it as-is. */
```

### Step 7 — Enter-selects-first-command (basm.js, ~8 lines)

Extend `balaurSubmitOnEnter` (basm.js lines 135-140):

```js
window.balaurSubmitOnEnter = function (event) {
  if (event.key !== 'Enter' || event.shiftKey || event.altKey ||
      event.ctrlKey || event.metaKey || event.isComposing) return;
  event.preventDefault();
  var ta = event.currentTarget;
  // Slash-command: pick the first visible palette item instead of sending.
  if (ta.value.trimStart().startsWith('/')) {
    var palette = ta.closest('.composer') &&
      ta.closest('.composer').querySelector('.cmd-palette');
    var first = palette && Array.prototype.find.call(
      palette.querySelectorAll('.cmd-item'),
      function (el) { return el.offsetParent !== null; }); // visible
    if (first) first.click();   // triggers the item's data-on:click @get
    return;                     // never post a "/foo" line to chat
  }
  ta.form && ta.form.requestSubmit();
};
```

`el.offsetParent !== null` is the visibility check (hidden items are
`display:none`). A programmatic `.click()` dispatches a real `click` event, which
Datastar's `data-on:click` listener handles.

### Step 8 — Storybook: CommandPalette story + reframe sidebarStory

> Real layout (verified): the Composer story is `composerStory()` in
> `internal/feature/storybook/stories_chat.go:75`, `Group: "Chat"`. There is **no**
> `stories_inputs.go` and **no** "Inputs" group. Stories are registered in the
> fixed `stories` slice in `internal/feature/storybook/story.go` (e.g.
> `composerStory(),` is line 83) — an unregistered story silently never renders.

1. Add `func commandpaletteStory() Story` in
   `internal/feature/storybook/stories_chat.go` with `Group: "Chat"`, rendering
   `ui.CommandPalette` with the five fixture items and documenting: props
   (`Label`/`Key`/`Icon`/`URL`), the `$message`-driven show/filter, the do
   ("fires the non-polluting /ui/show door; clears the draft") and don't ("don't
   server-filter per keystroke"). To make it visible in the static storybook (no
   live `$message`), seed `data-signals:message="'/'"` on the story wrapper (or
   note it renders hidden until a draft begins with `/`).
2. **Register it**: append `commandpaletteStory(),` to the `stories` slice in
   `internal/feature/storybook/story.go` (right after `composerStory(),`, ~line 83).
3. In `stories_navigation.go`, **reframe** `sidebarStory`'s comment (lines
   110-111): it no longer mirrors a home rail (home navigates via the composer
   palette as of plan 102); it documents the generic `shell.Sidebar` atom, still
   used by the storybook's own nav. Keep the fixture + the `/ui/show/...` actions
   (still the valid door).

### Step 9 — Tests

**`internal/web/home_test.go` — `TestHomeFullChat`:**
- Remove from `ExpectedContent`: `class="sb-side"`, `class="sb-nav-icon"`, the
  three rail `@get` lines (34, 36, 37), `class="app-topbar"`, `class="sb-burger"`,
  `onclick="basmToggleNav()"`, `class="sb-backdrop"`.
- Add to `ExpectedContent`: `class="cmd-palette"`; the palette's quests action
  `@get(&#39;/ui/show/quests&#39;)` (HTML-escaped, as the file already escapes);
  the settings action `@get(&#39;/ui/show/settings?section=profile&#39;)`; the
  composer signal binding `data-bind:message`; keep `class="app-shell"`,
  `id="chat"`, `id="panel"`, `id="panel-inner"`, `class="theme-toggle"`.
- Add to `NotExpectedContent`: `class="sb-side"`, `class="app-topbar"`,
  `basmToggleNav()`.
- **Make the `<html>` assertion class-agnostic** (forward-compat with plan 103):
  `home_test.go:25` asserts the exact string `<html lang="en" class="app">`.
  Plan 103 adds a `panel-collapsed` class to that element, which would break the
  closing `">`. Change this assertion to the **prefix** `<html lang="en" class="app`
  (no closing quote) so it tolerates extra classes. Do this now so 103 need not
  touch `home_test.go`'s shell assertion.
- Update the test's doc comment to describe the two-column palette shell.
- `TestHomePanelRestore` and `TestHomeDockSelectorIDs` are unchanged — verify
  they still pass (the panel + dock ids are untouched).

**New: `internal/ui/command_palette_test.go`** — render `ui.CommandPalette` with
two items and assert: root `class="cmd-palette"` + `data-show="$message.startsWith('/')"`;
each item carries `class="cmd-item"`, a per-item `data-show` containing
`.startsWith(`, the `@get('…')` action **and** `$message = ''`, and the
`/key` label. Follow the table-driven render-to-string pattern in the existing
`internal/ui/*_test.go` files (e.g. `composer`/`button` tests).

**`internal/ui/shell/sidebar_test.go`** — unchanged (generic atom; the home shell
no longer mounts it, but the atom's contract is the same). Verify it still passes.
Note for the executor: this file's `sb-burger` / `basmToggleNav()` / `sb-backdrop`
assertions (lines ~30, 34) test `shell.SidebarPage` (the **storybook** Page
drawer), **not** the home `ChatShell` — they are out of scope and stay green.
Do not panic when a grep finds `sb-burger`/`basmToggleNav` still present here.

### Step 10 — Docs

- `internal/self/knowledge.md`: replace the "left domain sidebar rail" IA
  description with "two columns — chat + the single-active panel — and a
  composer `/`-command palette as the navigation launcher (plan 102)." Note the
  rail + the plan-100 mobile burger/drawer are retired; mobile = chat full-width
  + palette + the panel overlay.
- `DESIGN.md`: same IA correction wherever the rail is described as the nav.
- Grep both for `sidebar`/`rail`/`app-topbar`/`burger` and fix stale prose.

## Files in scope

- `internal/ui/command_palette.go` (new), `internal/ui/command_palette_test.go` (new)
- `internal/ui/composer.go` (add `Palette` slot)
- `internal/web/home.go` (delete `domainSidebar`, add `commandPaletteNode`, wire composer)
- `internal/ui/shell/chatshell.go` (drop `Sidebar`, topbar, backdrop → two columns; add `.app-chrome` theme toggle)
- `internal/ui/chat/panel.go` (fix the stale "Pick a domain from the rail" empty-panel copy)
- `internal/web/assets/static/basm.css` (2-col grid; delete dead rail/topbar/drawer rules; palette CSS; `.app-chrome`)
- `internal/web/assets/static/basm.js` (`balaurSubmitOnEnter` slash branch)
- `internal/feature/storybook/stories_navigation.go` (reframe `sidebarStory` comment)
- `internal/feature/storybook/stories_chat.go` (add `commandpaletteStory`, `Group: "Chat"`)
- `internal/feature/storybook/story.go` (register `commandpaletteStory()` in the `stories` slice)
- `internal/web/home_test.go` (flip `TestHomeFullChat`; class-agnostic `<html>` assertion)
- `internal/self/knowledge.md`, `DESIGN.md`

## Files explicitly OUT of scope (do not touch)

- `internal/ui/shell/sidebar.go` + `sidebar_test.go` — the generic atom stays
  (storybook Page uses it).
- `basmToggleNav` in `basm.js` — still used by the storybook Page drawer.
- `internal/web/storybook.go` — the storybook's own `shell.Sidebar` nav stays.
- The `/ui/show` handler + the 40+ card-footer links (plan 101 already made them
  non-polluting; the palette reuses the same door).
- The panel itself (`chat.Panel`, `panel.go`) — plan 103 adds collapse/resize.
- `--w-panel` value — plan 103 makes it dynamic.

## Done criteria (machine-checkable)

```
# 1. The home shell renders no rail/topbar, and a palette:
#    (run the server or the test) GET / contains cmd-palette, not sb-side/app-topbar
CGO_ENABLED=0 go test ./internal/web/ -run TestHomeFullChat -count=1

# 2. domainSidebar is gone; commandPaletteNode exists:
grep -rn "func domainSidebar" internal/web/ ; test $? -ne 0
grep -n "func commandPaletteNode" internal/web/home.go

# 3. ChatShellProps no longer has Sidebar:
grep -n "Sidebar" internal/ui/shell/chatshell.go ; test $? -ne 0

# 4. No app-topbar in the rendered home shell (CSS rule + markup gone):
grep -n "app-topbar" internal/ui/shell/chatshell.go ; test $? -ne 0

# 5. Full gates:
gofmt -l internal/ | wc -l            # 0
CGO_ENABLED=0 go build ./...
go vet ./...
CGO_ENABLED=0 go test ./... -count=1   # exit 0
git diff --check
```

## Test plan

- `TestHomeFullChat` (rewritten) is the contract: rail/topbar markup absent,
  palette present, panel + chat ids intact.
- `command_palette_test.go` pins the atom's markup (the `data-show` filter
  expressions + the `@get … ; $message = ''` action).
- `TestHomePanelRestore` + `TestHomeDockSelectorIDs` prove the panel + the SSE
  selector ids survive the shell change untouched.
- **Manual (owner) verification — flagged, not automatable here:** there is no
  browser in the executor/review loop, so the *interactive* palette behavior
  (typing `/` reveals the menu, typing filters, click/Enter opens the artifact
  and clears the draft, the rail is truly gone, mobile chat is full-width with
  the panel overlay) must be eyeballed by the owner. The markup + wiring are
  test-verified; the live interaction is not.

## Maintenance note

- **Two-way binding is load-bearing.** The palette depends on the composer's
  `data-bind:message` + `data-signals:message`. If a future change renames or
  scopes that signal, the palette's `data-show`/`$message = ''` expressions and
  the basm.js slash branch must move with it. Keep the signal name `message`.
- **Discovery affordance.** The rail provided discovery ("what can I open?"); the
  owner accepted losing it. The composer-foot hint ("/ for pages") is the only
  remaining affordance — keep it, and if discovery proves too thin, the cheapest
  restoration is a `/` "show all" header in the palette, not bringing back the
  rail.
- **Deep-links deferred (YAGNI).** `/settings models` style sub-view commands are
  not in slice 1 — sub-views are reachable as in-panel tabs once the page opens.
  Add them only on real demand.
- Plan 103 layers collapse + free-drag resize on this two-column shell.

## Escape hatches

- If `ui.Composer` turns out to feed surfaces beyond the home master composer
  (e.g. branch chat) and the palette should not appear there, scope
  `commandPaletteNode()` to the home `composerNode` only (it already is) and
  **STOP** before adding it to any shared path; report the call sites.
- If Datastar rejects the `.startsWith(...)` expression in `data-show` at runtime
  (it should not — it is plain JS), fall back to a server-rendered `@get`-per-item
  with client filtering via a simpler `$message.includes` and **report** the
  Datastar version in use.
- If removing the `html.app .sb-side`/`app-topbar` CSS breaks an unrelated
  surface (grep shows another `html.app` consumer of those rules), **STOP** and
  report — only the home shell should be affected.
