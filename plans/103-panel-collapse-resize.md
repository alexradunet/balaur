# Plan 103 — Panel collapse + free-drag resize (server-persisted)

- **Written against commit:** `4e933d7` (re-anchor onto 102 before executing)
- **Priority:** P2 · **Effort:** M–L · **Risk:** MED
- **Depends on:** **102** (the two-column chat+panel shell), **101** (the `/ui/show` door)
- **Status:** TODO

> **Owner directive (verbatim):** *"The artifact page should be collapsable or
> made bigger/smaller as the same with the chat. They should be responsive and
> grow based on their allowed space."* Decision (locked via AskUserQuestion):
> **collapse + free drag** — a draggable divider; the gesture is JS, the
> committed width is persisted server-side.

## Why this matters

With the two-column shell (102), chat and panel split the viewport at a fixed
`--w-panel: 480px`. The owner wants to (a) **collapse** the panel so chat fills
the screen, and (b) **drag** the divider to rebalance — with the chosen size
**persisted server-side** so it survives reload (the SSR-state rule: the
*gesture* may be client-side, but the *committed* state lives in
`owner_settings`, exactly like `panel_active`).

The codebase already has both patterns to copy:

- **Collapse** mirrors `basmToggleDockFull` (basm.js 183-187): a class on
  `<html>` toggled by a tiny helper. The only change is persistence target —
  `owner_settings` (server) instead of `localStorage`, so it is rendered
  server-side on load (no flash, no client-only state).
- **Drag-resize** mirrors the `.dock-grip` rail resizer (basm.js 191-212):
  `pointerdown` → capture → `pointermove` sets a CSS var (clamped) →
  `pointerup` commits. The only change: commit `@post`s the width to the server
  instead of `localStorage.setItem`.

No new dependency, CGO stays off, and the panel grows/shrinks responsively
because the grid track is `var(--w-panel)` against a `1fr` chat column.

## Expected state after plans 101 + 102 land (NOT current `main`)

> This plan executes **on top of 101 and 102**. The excerpts below show the tree
> as those two plans leave it — they are deliberately **not** current `main`
> (on `main`, `chatshell.go` still has a `Sidebar` field, `chat/panel.go`'s close
> button still `@get`s `/ui/panel/close`, and `uiShow` has no `close` branch). The
> Step-1 drift check enforces that 101+102 have landed before you apply any anchor
> here; if they have not, **STOP**.

### `internal/ui/shell/chatshell.go` — AFTER plan 102 (two columns)

By the time this plan runs, `ChatShell` renders (102's end state):

```go
h.Body(
	h.A(h.Class("skip-link"), h.Href("#chat"), g.Text("Skip to content")),
	h.Div(h.Class("app-shell"),
		h.Aside(h.ID("dock"), h.Class("app-dock"), p.Dock),
		h.Aside(h.ID("panel"), h.Class("app-panel"), p.Panel),
	),
),
```

with `<html lang="en" class="app">`. `ChatShellProps` (post-102) is
`{Title, Dock, Panel}`.

### `internal/ui/chat/panel.go` — `chat.Panel` head bar (lines 21-45)

The `.panel-head` already carries the close control (post-101 it `@get`s
`/ui/show/close`):

```go
head = append(head,
	h.Span(h.Class("panel-head-title"), g.Text(p.Title)),
	h.Button(h.Class("panel-close"), h.Type("button"),
		g.Attr("data-on:click__prevent", "@get('/ui/show/close')"),
		h.Aria("label", "Close panel"), g.Text("✕")),
)
```

### `internal/web/assets/static/basm.js` — the two exemplars

```js
// Full-screen toggle (class on <html> + localStorage). lines 183-187
window.basmToggleDockFull = function () {
  const on = document.documentElement.classList.toggle('dock-full');
  localStorage.setItem('basm-dock-full', on ? '1' : '0');
  balaurScrollToLatest();
};

// Drag the left grip to resize the rail. lines 191-212
document.addEventListener('pointerdown', (e) => {
  const grip = e.target.closest('.dock-grip');
  if (!grip || document.documentElement.classList.contains('dock-full')) return;
  e.preventDefault();
  grip.setPointerCapture(e.pointerId);
  grip.classList.add('dragging');
  const onMove = (ev) => {
    const w = Math.max(280, Math.min(720, window.innerWidth - ev.clientX));
    document.documentElement.style.setProperty('--sidebar-w', w + 'px');
  };
  const onUp = () => {
    grip.classList.remove('dragging');
    document.removeEventListener('pointermove', onMove);
    document.removeEventListener('pointerup', onUp);
    document.removeEventListener('pointercancel', onUp);
    const w = parseInt(document.documentElement.style.getPropertyValue('--sidebar-w'), 10);
    if (w) localStorage.setItem('basm-dock-w', String(w));
  };
  document.addEventListener('pointermove', onMove);
  document.addEventListener('pointerup', onUp);
  document.addEventListener('pointercancel', onUp);
});
```

### `internal/web/assets/static/basm.css`

- `:root { --w-panel: 480px; }` (line 167) — the default the inline override falls back to.
- `html.app .app-shell { grid-template-columns: 1fr var(--w-panel); … }` (post-102).
- `html.app #panel.app-panel { position: relative; … }` (lines ~3315-3319) — so a
  `.panel-resizer` child can be absolutely positioned on its left edge.

### `internal/store/owner_settings.go`

`GetOwnerSetting(app, key, default) string` and `SetOwnerSetting(app, key, value) error`.
No migration needed — `owner_settings` is the existing KV store (same as
`panel_active`).

### `internal/web/show.go` — `uiShow` (post-101)

The merged non-polluting door. This plan adds two lines so opening an artifact
**expands** the panel and `close` **collapses** it (see Step 6).

## The change — ordered steps

### Step 1 — Re-anchor + drift check (BOTH 101 and 102 must have landed)

```
grep -n 'typ == "close"' internal/web/show.go                      # 101 landed (uiShow owns close)
grep -rn "/ui/panel" --include=*.go internal/ ; test $? -ne 0      # 101 landed (GET door gone)
grep -n "Sidebar" internal/ui/shell/chatshell.go ; test $? -ne 0   # 102 landed (rail removed)
grep -n "func commandPaletteNode" internal/web/home.go             # 102 landed
git rev-parse --short HEAD
```

If `uiShow` has **no** `typ == "close"` branch → **STOP**, execute 101 first.
If `chatshell.go` still has a `Sidebar` field → **STOP**, execute 102 first.
Every "Expected state" anchor above assumes both; do not apply them against `main`.

### Step 2 — Persisted panel state: keys + read helper

Add to `internal/web/panel.go` (next to `panelActiveKey`):

```go
const (
	panelCollapsedKey = "panel_collapsed" // "1" collapsed, "0" expanded, "" = derive
	panelWidthKey     = "panel_width"     // integer px as a string, "" = CSS default
)

// panelCollapsed reports whether the panel should render collapsed. Explicit
// "1"/"0" win; unset derives from emptiness — a panel with nothing open
// collapses so chat fills the screen (plan 103 "collapse-when-empty").
func (h *handlers) panelCollapsed() bool {
	switch store.GetOwnerSetting(h.app, panelCollapsedKey, "") {
	case "1":
		return true
	case "0":
		return false
	default:
		return store.GetOwnerSetting(h.app, panelActiveKey, "") == ""
	}
}

// panelWidthCSS returns the inline "--w-panel: <px>px" override, or "" to use the
// CSS default. The value is clamped on write (Step 5) so render trusts it but
// re-clamps defensively.
func (h *handlers) panelWidthCSS() string {
	raw := store.GetOwnerSetting(h.app, panelWidthKey, "")
	if raw == "" {
		return ""
	}
	px, err := strconv.Atoi(raw)
	if err != nil || px < panelMinPx {
		return ""
	}
	if px > panelMaxPx {
		px = panelMaxPx
	}
	return "--w-panel:" + strconv.Itoa(px) + "px"
}
```

Add `panelMinPx = 320` and `panelMaxPx = 1100` consts. These match the JS clamp
(Step 4).

> **Imports (do not under-add).** `internal/web/panel.go` currently imports
> `net/url`, `strings`, + pocketbase/datastar/gomponents/cards/store. This plan's
> new code needs **both** `strconv` (the helpers + handlers) **and** `net/http`
> (Step 6's `http.StatusNoContent`). Add **both** to the import block — adding
> only `strconv` leaves `http` undefined and the package will not compile.

### Step 3 — Render the persisted state in the shell

Extend `ChatShellProps` (chatshell.go):

```go
type ChatShellProps struct {
	Title          string
	Dock           g.Node
	Panel          g.Node
	PanelCollapsed bool   // render with the panel hidden, chat full-width
	PanelStyle     string // inline "--w-panel:<px>px" override, or ""
}
```

In `ChatShell`:
- `<html>` class: `"app"` + (PanelCollapsed ? `" panel-collapsed"` : `""`).
- **Render `PanelStyle` (the `--w-panel` override) inline on the `<html>`
  element**, NOT on `.app-shell`. This is load-bearing — see the cascade note in
  Step 5. So: when `PanelStyle != ""`, add `g.Attr("style", p.PanelStyle)` to the
  `h.HTML(...)` element's attrs (alongside `h.Lang`/`h.Class`). The grid track
  `var(--w-panel)` on `.app-shell` (basm.css:3309) **inherits** the value from
  `<html>`. Do **not** also put it on `.app-shell` — a second declaration there
  would shadow the `<html>` one and break the live drag.
- Add the resizer as the panel's first child and a reveal handle as a sibling:

```go
h.Div(append(shellAttrs, // h.Class("app-shell") [+ style]
	h.Aside(h.ID("dock"), h.Class("app-dock"), p.Dock),
	h.Aside(h.ID("panel"), h.Class("app-panel"),
		h.Button(h.Class("panel-resizer"), h.Type("button"),
			h.Aria("label", "Resize panel"), h.TabIndex("-1")),
		p.Panel,
	),
	// Reveal handle: shown only when collapsed (CSS), re-opens the panel.
	h.Button(h.Class("panel-reveal"), h.Type("button"),
		g.Attr("onclick", "basmTogglePanel()"),
		h.Aria("label", "Show panel"), g.Text("‹")),
)...),
```

(Use a `[]g.Node` builder for the `.app-shell` attrs so the optional `style` is
clean.)

In `internal/web/home.go` `homePage`, pass the state:

```go
page := shell.ChatShell(shell.ChatShellProps{
	Title:          "Home",
	Dock:           dockNode,
	Panel:          h.restoredPanelNode(),
	PanelCollapsed: h.panelCollapsed(),
	PanelStyle:     h.panelWidthCSS(),
})
```

### Step 4 — `chat.Panel`: a collapse control in the head bar

In `internal/ui/chat/panel.go`, add a collapse button to `.panel-head` (before
the close ✕):

```go
h.Button(h.Class("panel-collapse"), h.Type("button"),
	g.Attr("onclick", "basmTogglePanel()"),
	h.Aria("label", "Collapse panel"), g.Text("›")),
```

### Step 5 — basm.js: `basmTogglePanel()` + the panel drag, both server-persisted

Add to `internal/web/assets/static/basm.js` (near the dock helpers):

```js
// Panel collapse: class on <html>, persisted server-side (plan 103).
window.basmTogglePanel = function () {
  var on = document.documentElement.classList.toggle('panel-collapsed');
  fetch('/ui/panel/collapse', {
    method: 'POST', headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
    body: 'on=' + (on ? '1' : '0'),
  });
};

// Drag the panel divider to resize. Commits the width to the server on release.
document.addEventListener('pointerdown', function (e) {
  var grip = e.target.closest('.panel-resizer');
  if (!grip) return;
  e.preventDefault();
  grip.setPointerCapture(e.pointerId);
  grip.classList.add('dragging');
  var onMove = function (ev) {
    var w = Math.max(320, Math.min(1100, window.innerWidth - ev.clientX));
    document.documentElement.style.setProperty('--w-panel', w + 'px');
  };
  var onUp = function () {
    grip.classList.remove('dragging');
    document.removeEventListener('pointermove', onMove);
    document.removeEventListener('pointerup', onUp);
    document.removeEventListener('pointercancel', onUp);
    var w = parseInt(document.documentElement.style.getPropertyValue('--w-panel'), 10);
    if (w) fetch('/ui/panel/width', {
      method: 'POST', headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: 'px=' + w,
    });
  };
  document.addEventListener('pointermove', onMove);
  document.addEventListener('pointerup', onUp);
  document.addEventListener('pointercancel', onUp);
});
```

> **CSS-cascade note (get this right — it is the difference between a working
> and a dead drag).** CSS custom properties **inherit**, and an element's *own*
> inline declaration always beats a value *inherited* from an ancestor. So both
> the server render (Step 3) and this drag MUST set `--w-panel` on the **same**
> element: `<html>` (`document.documentElement`). `.app-shell` then inherits it
> and its `var(--w-panel)` grid track follows. If the server instead rendered the
> override on `.app-shell`, that element's own inline value would shadow the
> `<html>` value this drag sets, and after the first persisted resize the divider
> would visibly do **nothing** on reload. Set it on `<html>` in both places.
> (A same-origin `fetch` POST carries an `Origin` header matching `Host`, so
> `guardLocalUI` admits it.)

### Step 6 — Server: persist endpoints + open/close collapse coupling

In `internal/web/panel.go`, add handlers:

```go
// uiPanelCollapse persists the panel collapsed flag (POST /ui/panel/collapse,
// form: on=0|1). The client already applied the class; this just remembers it.
func (h *handlers) uiPanelCollapse(e *core.RequestEvent) error {
	on := "0"
	if e.Request.FormValue("on") == "1" {
		on = "1"
	}
	_ = store.SetOwnerSetting(h.app, panelCollapsedKey, on)
	return e.NoContent(http.StatusNoContent)
}

// uiPanelWidth persists the dragged panel width (POST /ui/panel/width, form:
// px=NNN), clamped to [panelMinPx, panelMaxPx].
func (h *handlers) uiPanelWidth(e *core.RequestEvent) error {
	px, err := strconv.Atoi(e.Request.FormValue("px"))
	if err != nil {
		return e.BadRequestError("bad width", err)
	}
	if px < panelMinPx {
		px = panelMinPx
	}
	if px > panelMaxPx {
		px = panelMaxPx
	}
	_ = store.SetOwnerSetting(h.app, panelWidthKey, strconv.Itoa(px))
	return e.NoContent(http.StatusNoContent)
}
```

Register in `internal/web/web.go` (next to the `/ui/show` route):

```go
se.Router.POST("/ui/panel/collapse", h.uiPanelCollapse)
se.Router.POST("/ui/panel/width", h.uiPanelWidth)
```

> Route-conflict check: `/ui/panel/collapse` and `/ui/panel/width` are **POST**;
> plan 101 deleted the **GET** `/ui/panel/{type}`. Go's `ServeMux` keys on
> method+pattern, so these static POST patterns do not collide with anything.
> Confirm `grep -n "ui/panel" internal/web/web.go` shows only these two POSTs.
>
> Namespace note: 101's invariant is "no **GET** `/ui/panel` navigation door,"
> not "the string `/ui/panel` never appears." These POST state endpoints
> intentionally reuse the `panel` namespace (they are panel operations).
> Re-running 101's source grep after this plan will correctly show them — that is
> expected, not a regression.

Couple open/close to collapse in `uiShow` (show.go): opening any artifact should
reveal the panel; `close` should collapse it. Add to the merged `uiShow`:

```go
	if typ == "close" {
		_ = store.SetOwnerSetting(h.app, panelCollapsedKey, "1") // collapse on close
		return h.panelClose(e)
	}
	...
	_ = store.SetOwnerSetting(h.app, panelActiveKey, showURL(typ, queryStr))
	_ = store.SetOwnerSetting(h.app, panelCollapsedKey, "0")     // expand on open
```

### Step 7 — CSS: collapsed grid, resizer, reveal handle

In `internal/web/assets/static/basm.css`:

```css
/* Collapsed (DESKTOP only): chat fills the row, panel hidden. The display:none
   MUST be desktop-scoped — on ≤720px the panel is a transform-based overlay
   revealed by .panel-open (098), and the fresh-app default is collapsed
   (collapse-when-empty), so an unscoped display:none would make the mobile
   overlay un-openable. */
@media (min-width: 721px) {
  html.app.panel-collapsed .app-shell { grid-template-columns: 1fr; }
  html.app.panel-collapsed #panel.app-panel { display: none; }
}

/* Draggable divider on the panel's left edge. */
html.app .panel-resizer {
  position: absolute; left: -3px; top: 0; bottom: 0; width: 6px;
  border: 0; padding: 0; background: transparent; cursor: col-resize;
  z-index: var(--z-sticky);
}
html.app .panel-resizer:hover,
html.app .panel-resizer.dragging { background: var(--gold-deep); opacity: .5; }

/* Reveal handle: hidden unless collapsed; sits on the right edge. */
html.app .panel-reveal {
  position: fixed; right: 0; top: 50%; transform: translateY(-50%);
  display: none; /* shown only when collapsed */
  /* small gold tab, match .sb-burger chrome */
  z-index: var(--z-sticky);
}
html.app.panel-collapsed .panel-reveal { display: block; }

/* Collapse/close controls in the head bar share .panel-close chrome. */
html.app .panel-collapse { /* same look as .panel-close */ }
```

On `≤720px` the panel is already an overlay (098); collapse there just means the
overlay stays closed — verify the resizer is harmless (it is `display:none` with
the panel when collapsed; when the overlay is open it sits at its left edge).
Keep the mobile rules minimal — do not reintroduce rail chrome.

### Step 8 — Tests

**`internal/web/panel_unit_test.go`** (or a new `panel_state_test.go`):
- `panelCollapsed` table test: `"1"`→true; `"0"`→false; unset + `panel_active=""`
  →true; unset + `panel_active` set →false. Use a temp-dir app
  (`store` test helpers) + `SetOwnerSetting`.
- `panelWidthCSS`: unset→`""`; `"600"`→`"--w-panel:600px"`; `"99"` (below min)→`""`;
  `"5000"`→clamped to `--w-panel:1100px`; non-numeric→`""`.
- `POST /ui/panel/collapse` (on=1 / on=0) sets `panel_collapsed`; `POST
  /ui/panel/width` (px=600) sets `panel_width=600`; px below/above clamp; bad px
  →400. Follow the `ApiScenario` + `AfterTestFunc`-reads-`GetOwnerSetting`
  pattern already in `show_test.go`/`panel_unit_test.go`.

**`internal/web/home_test.go`** — add `TestHomePanelChrome`:
- Default render (fresh app, nothing open): the `<html>` carries `class="app panel-collapsed"`
  (collapse-when-empty) and `class="panel-resizer"` + `class="panel-reveal"` are present.
- With `panel_active` set + `panel_collapsed="0"` + `panel_width="600"`: the `<html>`
  class lacks `panel-collapsed`, and **the `<html>` element carries
  `style="--w-panel:600px"`** (per Step 3 — the width override renders on `<html>`,
  NOT `.app-shell`; assert it on the right element or the test will mislead).
  (Render via the `h.restoredPanelNode()`/handler-direct pattern that
  `TestHomePanelRestore` uses to avoid nested `ApiScenario`.)
- **No edit to `TestHomeFullChat` is needed**: plan 102 already changed its
  `<html>` assertion to the class-agnostic prefix `<html lang="en" class="app`,
  which tolerates the added `panel-collapsed` class. Verify it still passes.

### Step 9 — Storybook + docs

- The panel head bar gained a collapse control — update the `chat.Panel` story
  (wherever it lives, e.g. `stories_chat.go`) to render the new `.panel-collapse`
  button and note both controls are inert in the storybook. Add a one-line note
  documenting the resizer + reveal handle as shell chrome (not part of the Panel
  organism itself).
- `internal/self/knowledge.md` + `DESIGN.md`: document that the panel is
  collapsible (persisted `panel_collapsed`, collapse-when-empty default) and
  owner-resizable by dragging the divider (persisted `panel_width`, clamped
  320–1100px), with the gesture in `basm.js` and the committed state in
  `owner_settings`.

## Files in scope

- `internal/web/panel.go` (state keys, `panelCollapsed`, `panelWidthCSS`,
  `uiPanelCollapse`, `uiPanelWidth`, consts, `strconv`/`http` imports)
- `internal/web/web.go` (two POST routes)
- `internal/web/show.go` (open→expand, close→collapse coupling)
- `internal/web/home.go` (pass `PanelCollapsed`/`PanelStyle`)
- `internal/ui/shell/chatshell.go` (props + resizer + reveal handle + html class/style)
- `internal/ui/chat/panel.go` (collapse control in `.panel-head`)
- `internal/web/assets/static/basm.js` (`basmTogglePanel` + panel drag)
- `internal/web/assets/static/basm.css` (collapsed grid, resizer, reveal, controls)
- tests: `internal/web/panel_unit_test.go` (or `panel_state_test.go`), `internal/web/home_test.go`
- storybook chat story; `internal/self/knowledge.md`, `DESIGN.md`

## Files explicitly OUT of scope (do not touch)

- `basmToggleDockFull` / the `.dock-grip` rail resizer — the chat dock's own
  resize is unrelated; copy the *pattern*, don't modify it.
- The `/ui/show` door's morph/validation logic (101) beyond the two
  collapse-coupling lines.
- The command palette (102), `chat.Message`/`ToolRow`, recap, model/heads routes.
- `--w-panel`'s `:root` default value (167) — keep 480px as the fallback.

## Done criteria (machine-checkable)

```
# 1. State helpers + endpoints exist:
grep -n "func (h \*handlers) panelCollapsed\|panelWidthCSS\|uiPanelCollapse\|uiPanelWidth" internal/web/panel.go
grep -n "/ui/panel/collapse\|/ui/panel/width" internal/web/web.go

# 2. Shell renders the collapse class + resizer:
grep -n "panel-collapsed\|panel-resizer\|panel-reveal" internal/ui/shell/chatshell.go

# 3. JS toggle + drag persist to the server (fetch, not localStorage):
grep -n "basmTogglePanel\|/ui/panel/width\|/ui/panel/collapse" internal/web/assets/static/basm.js

# 4. Full gates:
gofmt -l internal/ | wc -l            # 0
CGO_ENABLED=0 go build ./...
go vet ./...
CGO_ENABLED=0 go test ./... -count=1   # exit 0
git diff --check
```

## Test plan

- `panelCollapsed` / `panelWidthCSS` unit tests pin the SSR state logic incl. the
  collapse-when-empty default and the clamp.
- The two POST handler tests pin server persistence + clamp + bad-input 400.
- `TestHomePanelChrome` proves the rendered shell reflects the persisted state
  (class + inline width).
- **Manual (owner) verification — flagged, not automatable here:** no browser in
  the loop, so the live *gesture* (dragging the divider resizes smoothly and the
  width survives reload; the collapse toggle hides the panel and the reveal
  handle restores it; open-expands / close-collapses) must be eyeballed. The
  persisted-state rendering and the endpoints are test-verified; the pointer
  drag and the `--w-panel` cascade target (Step 5 note) are not.

## Maintenance note

- **Where the var is set during drag matters.** Step 5 sets `--w-panel` on
  `<html>` while the server renders it on `.app-shell`. If a future refactor
  moves the grid var to a different element, move the JS `setProperty` target to
  match (the cascade must resolve through the same element). The owner-eyeball
  step exists to catch a mismatch.
- **Collapse is coupled to open/close.** `uiShow` expands on open and collapses
  on close; the explicit toggle overrides for the current session and persists.
  If a future change adds another way to open the panel, set
  `panel_collapsed="0"` there too, or the panel will open hidden.
- **No migration.** Panel state is KV in `owner_settings`; if these ever need to
  be per-head or per-device, that is a schema decision, not this plan.

## Escape hatches

- If the `--w-panel` drag has no visual effect (the var resolves through
  `.app-shell`, not `<html>`), switch the JS `setProperty` target to the
  `.app-shell` element (query it) and re-test — do not ship a dead drag.
- If `e.NoContent` / `http.StatusNoContent` is not the codebase's idiom for an
  empty 200/204 (check an existing POST write handler), match that idiom instead.
- If coupling collapse to `uiShow` (Step 6) breaks an existing `/ui/show` test
  that asserts `panel_collapsed`, reconcile by updating that assertion — but if
  it reveals a non-home caller of `uiShow` that should not auto-expand, **STOP**
  and report.
