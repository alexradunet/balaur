# Plan 088: Single-page chat shell + left domain sidebar + deterministic artifact injection

> **Executor instructions**: Follow step by step. Run every **Verify** and confirm its expected output before moving on. On a STOP condition, stop and report — do not improvise. This plan is the MILESTONE vertical of a 4-plan program (088 → 089 → 090 → 091); it ADDS the new single-page shell **without** removing the old `shell.Page`/`Topbar`/`/focus`/boards (plan 089 deletes those). When done, update the 088 row in `plans/readme.md` (the advisor owns the index — add the row only if absent, matching the existing column format; do NOT rewrite prose).
>
> **Drift check (run first)**:
> `git diff --stat 3136bad..HEAD -- internal/ui/shell/sidebar.go internal/ui/shell/shell.go internal/ui/chat/dock.go internal/web/home.go internal/web/web.go internal/web/recap.go internal/web/cards.go internal/web/chatstream.go internal/web/tasks.go internal/web/storybook.go internal/conversation/conversation.go internal/cards/cards.go internal/tools/ui.go internal/feature/storybook internal/web/assets/static/basm.css`
> If any of those changed since this plan was written, compare the "Current state" excerpts below to the live code; on mismatch, STOP and report. Confirm `maragu.dev/gomponents-datastar v0.3.3` and `github.com/starfederation/datastar-go v1.2.2` are still in `go.mod` (the Datastar action wiring below depends on the `data.On(...)` helper at that version).
>
> **Commit anchor**: `3136bad` (2026-06-17). **Next free migration timestamp**: `1750850000` (this plan adds NO migration — boards drop is plan 089).

## Status
- **Priority**: P1
- **Effort**: L
- **Risk**: MED–HIGH
- **Depends on**: nothing (first of the program). Plans **089, 090, 091** all depend on this one.
- **Category**: feature (owner-decided, LOCKED)
- **Planned at**: commit `3136bad`, 2026-06-17

## Why this matters
The owner has locked a UI rebuild: retire the top-nav + per-page (`/focus/{type}`) + right-rail dock layout, and ship ONE page — a **left domain sidebar + the chat as the only primary surface**. Two doors produce the SAME thing: an **artifact** (a gomponents card) rendered INTO the conversation stream.
1. **Deterministic door** — clicking a sidebar domain item does NOT navigate; it injects that domain's card into the chat. No LLM, instant, free.
2. **Conversational door** — a natural-language request makes the agent call `card_show`, which renders the exact same card into the chat. This door already works today (`internal/tools/ui.go` → `MarkUICard` → `chatstream.handleToolResult`).

Both converge on the SAME card registry (`internal/cards`) via the SAME render seam (`h.cardHTML`). A summoned artifact must be a **permanent, live transcript entry**: it survives reload and re-renders from current data each time — exactly like `card_show` does today via the `uicard` marker re-parsed in `recap.messageViews`.

This plan delivers the milestone: GET `/` renders the sidebar + full-canvas chat, and a sidebar click injects + persists a single-card artifact. **Clusters are plan 090. Retiring the old topbar/focus/boards is plan 089.** So this plan must add the new shell as a NEW composition path (used only by home) and leave `shell.Page` + `Topbar` + `/focus` intact for the other pages until 089.

## The decisive correctness pin (read before coding)
There are TWO separate render paths for a chat artifact, and a new artifact MUST work in BOTH or it shows live but vanishes on reload:
- **Live stream**: `chatstream.handleToolResult` (`internal/web/chatstream.go:176`) parses the marker out of a tool-result string and appends a server-rendered card via `endTool`.
- **Reload replay**: `recap.messageViews` (`internal/web/recap.go:237`) re-parses the SAME marker out of the persisted `messages` row content and re-renders.

The deterministic sidebar injection does NOT go through the agent loop, so it has no `tool_result` event. Instead, this plan **persists a `messages` row whose `content` carries the existing `uicard` marker** (`tools.MarkUICard`), so the **already-existing** `recap.messageViews` `uicard` branch re-renders it on reload with ZERO new reload code. The only new live-path code is the deterministic endpoint that appends the card once and writes that row. This is the KISS win: reuse the marker, reuse the reload branch, add one GET endpoint.

**Persist the row exactly like a `card_show` result: `role:"tool"`, `origin:""`** (NOT a new `origin:"artifact"`). This is load-bearing for a second reason: the nudge poller `chatNudges` (`internal/web/tasks.go:311`) queries `origin != '' && created > {:since}` every 30s — so a row with a *non-empty* origin would be **re-appended by the next poll**, duplicating the just-injected card. An empty origin sidesteps the poller entirely (it only collects agent-initiated `nudge`/`briefing`/`check` rows), needs **no change to `tasks.go`**, and `recap.messageViews` re-renders it regardless of origin (it keys only on `role == "tool"` + the marker). Provenance, if ever needed, is already in `tool_name` (set to the card type here vs `"card_show"` for the agent door). So: do NOT introduce an `origin:"artifact"` value and do NOT touch the `chatNudges` filter.

## Current state (confirmed excerpts — re-read at HEAD before trusting)

**`internal/ui/shell/sidebar.go:10-16`** — `SidebarItem` today is a plain `<a href>` with only a color `Dot`, no icon, no Datastar action:
```go
10 // SidebarItem is one nav link. Active marks the current page. Dot is an optional
11 // CSS color for the leading group-dot (e.g. "var(--teal)").
12 type SidebarItem struct {
13 	Label, Href string
14 	Active      bool
15 	Dot         string
16 }
```

**`internal/ui/shell/sidebar.go:102-116`** — `sidebarItem` renders an `<a href>` (full page load), dot only:
```go
102 func sidebarItem(it SidebarItem) g.Node {
103 	cls := "sb-nav-item"
104 	if it.Active {
105 		cls += " sb-nav-item-active"
106 	}
107 	attrs := []g.Node{h.Class(cls), h.Href(it.Href)}
108 	if it.Active {
109 		attrs = append(attrs, h.Aria("current", "page"))
110 	}
111 	if it.Dot != "" {
112 		attrs = append(attrs, h.Span(h.Class("sb-nav-dot"), h.Style("--sb-nav-dot:"+it.Dot)))
113 	}
114 	attrs = append(attrs, h.Span(g.Text(it.Label)))
115 	return h.A(attrs...)
116 }
```
The `shell` package imports ONLY `strconv`, `g "maragu.dev/gomponents"`, `h "maragu.dev/gomponents/html"`.

**`internal/ui/shell/sidebar.go:36-58`** — `Sidebar(SidebarProps)` is the generic reusable rail (`SidebarProps{Brand, Sections, Footer}`); `Sidebar(...)` returns `h.Aside(... h.Class("sb-side"))`. Currently the only caller is `internal/web/storybook.go:sidebarFor`.

**`internal/ui/shell/shell.go:20-53`** — `PageProps` + `Page` (the CURRENT product shell: topbar + `#main` inside `.with-sidebar` + right-rail `#dock`):
```go
20 type PageProps struct {
21 	Title     string
22 	Active    string
23 	HTMLClass string
24 	Body      g.Node
25 	Dock      g.Node
26 }
...
29 func Page(p PageProps) g.Node {
30 	html := []g.Node{h.Lang("en")}
31 	if p.HTMLClass != "" {
32 		html = append(html, h.Class(p.HTMLClass))
33 	}
34 	html = append(html,
35 		h.Head( pageHead(), h.TitleEl(g.Text(p.Title+" · Balaur")) ),
36 		h.Body(
37 			h.A(h.Class("skip-link"), h.Href("#main"), g.Text("Skip to content")),
38 			Topbar(p.Active),
39 			h.Div(h.Class("with-sidebar"), h.Main(h.ID("main"), p.Body)),
40 			h.Aside(h.ID("dock"), p.Dock),
41 			topnavDrawer(p.Active),
42 		),
43 	)
44 	return g.Group([]g.Node{ g.Raw("<!doctype html>"), h.HTML(html...) })
45 }
```
(Lines re-numbered for brevity; the real file has `pageHead()`/`Topbar`/`topnavDrawer` exactly as named. `pageHead()` at `:57` emits the meta + basm.css + no-flash script + datastar.js + basm.js.)

**`internal/web/home.go:57-80`** — `homePage` builds the home dock and wraps it in `shell.Page(... HTMLClass:"home", Dock: dockNode)`:
```go
62 	dockNode := chat.Dock(chat.DockProps{
63 		Variant:   chat.DockHome,
64 		HasRecap:  dock.HasRecap,
65 		NowMillis: dock.NowMillis,
66 		Convo:     g.Raw(string(dock.ChatBodyHTML)),
67 		Composer:  composerNode(dock),
68 	})
69 	page := shell.Page(shell.PageProps{
70 		Title:     "Home",
71 		Active:    "home",
72 		HTMLClass: "home",
73 		Dock:      dockNode,
74 	})
```

**`internal/ui/chat/dock.go:51-68`** — `Dock` emits `#dock-convo > #chat` (the append target), `#nudge-poll`, the composer slot, and a `dock-v-{variant}` wrapper class. **These IDs (`#chat`, `#dock-convo`, `#nudge-poll`, `#chat-draft` from the composer, `#model-modal`, `#recap`) are the SSE hot-path contract — do not change them.** `dockConvo` at `:101`:
```go
101 func dockConvo(convo g.Node) g.Node {
102 	return h.Div(h.ID("dock-convo"),
103 		h.Section(h.Class("chat"), h.ID("chat"), g.Attr("aria-live", "polite"),
104 			convo,
105 		),
106 	)
107 }
```

**`internal/web/chatstream.go:84-93`** — the artifact append + morph primitives (copy this PatchElements shape):
```go
84 // appendNode appends a rendered component as the last child of #chat.
85 func (s *chatStream) appendNode(n g.Node) {
86 	_ = s.sse.PatchElements(s.renderNode(n),
87 		datastar.WithSelectorID("chat"), datastar.WithModeAppend())
88 }
```

**`internal/web/chatstream.go:176-209`** — `handleToolResult` (consumer order uicard → choices → proposal → refresh → plain) + `endTool` (appends a `k-inline` div with the rendered card):
```go
176 func (s *chatStream) handleToolResult(ev agent.Event) {
177 	if typ, query, rest, ok := tools.ParseUICard(ev.Text); ok {
178 		s.endTool(rest, s.h.uicardBody(typ, query))
179 		return
180 	}
...
202 func (s *chatStream) endTool(content string, card template.HTML) {
203 	s.morphNode(chat.ToolRow(...))
204 	if card != "" {
205 		s.appendNode(g.El("div", g.Attr("class", "k-inline"), g.Attr("id", s.toolID+"-card"), g.Raw(string(card))))
206 	}
207 }
```

**DETERMINISTIC-APPEND PRECEDENT — `internal/web/tasks.go:305-328`** (`chatNudges`): a GET endpoint that opens its OWN `datastar.NewSSE` and `PatchElements(..., WithSelectorID("chat"), WithModeAppend())`. Copy this shape for `/ui/show`:
```go
305 func (h *handlers) chatNudges(e *core.RequestEvent) error {
...
322 	sse := datastar.NewSSE(e.Response, e.Request)
323 	_ = sse.PatchElements(string(h.renderMessages(h.messageViews(recs))), datastar.WithSelectorID("chat"), datastar.WithModeAppend())
324 	_ = sse.MarshalAndPatchSignals(struct {
325 		NudgeSince int64 `json:"nudgeSince"`
326 	}{last})
327 	return nil
328 }
```

**RELOAD RE-RENDER — `internal/web/recap.go:262-274`** (the `uicard` branch this plan reuses verbatim; only `Role == "tool"` rows are scanned for markers):
```go
262 		if mv.Role == "tool" {
263 			if typ, query, rest, ok := tools.ParseUICard(mv.Content); ok {
264 				mv.CardBody = h.uicardBody(typ, query)
265 				mv.Content = rest
266 			} else if _, _, modelText, ok := tools.ParseChoices(mv.Content); ok {
...
```
And `renderMessages` at `:196-202` renders a `tool` row + (when `CardBody != ""`) a `k-inline` div — identical chrome to the live `endTool`. **So persisting an `artifact` as a `role:"tool"` row carrying the `uicard` marker makes the existing reload branch render it with NO new code.**

**The marker + its registry validation — `internal/tools/ui.go:26-58`**:
```go
26 const UICardMarker = "\x00balaur-uicard:"
30 func MarkUICard(typ string, params map[string]string, modelText string) string {
...
44 func ParseUICard(s string) (typ, query, rest string, ok bool) {
...
54 	if _, found := cards.Get(typ); !found {   // registry validation: ok only for a real card type
55 		return "", "", rest, false
56 	}
57 	return typ, query, rest, true
58 }
```
`MarkUICard`/`ParseUICard` are EXPORTED, in `internal/tools`, which is allowed to be imported by `internal/web` (already imported in `recap.go:17` and `chatstream.go:13`).

**The render seam — `internal/web/cards.go:120-150`** (`cardHTML` validates via `cards.Validate` + renders via `ui.LookupCard`, error-strips on failure; `uicardBody` parses a url query then calls `cardHTML`):
```go
120 func (h *handlers) cardHTML(typ string, params map[string]string) template.HTML {
121 	if _, ok := cards.Get(typ); !ok { return cardErrorStrip("no such card type: " + typ) }
124 	cleaned, err := cards.Validate(typ, params)
125 	if err != nil { return cardErrorStrip(err.Error()) }
...
133 	return template.HTML(b.String())
134 }
147 func (h *handlers) uicardBody(typ, query string) template.HTML {
148 	vals, _ := url.ParseQuery(query)
149 	return h.cardHTML(typ, queryToMap(vals))
150 }
```

**Persistence — `internal/conversation/conversation.go:65-87`** (`AppendOrigin` writes a `messages` row: role/content/tool_name/origin/tool_payload):
```go
65 func AppendOrigin(app core.App, conversationID string, msg llm.Message, toolName, origin string) error {
...
70 	rec := core.NewRecord(col)
71 	rec.Set("conversation", conversationID)
72 	rec.Set("role", msg.Role)
73 	rec.Set("content", msg.Content)
74 	if toolName != "" { rec.Set("tool_name", toolName) }
77 	if origin != "" { rec.Set("origin", origin) }
...
```
`RecentTurns` (`:97-114`) excludes `role:"tool"` rows from MODEL context — so an `artifact` persisted as `role:"tool"` will NOT pollute the model's context (correct; the model summons cards itself via `card_show`). `History` (`:143`) and the home `dockData`/`messageViews` path (`web.go:286-305`, `recap.go:237`) load ALL roles for display.

**`internal/web/web.go:199-237`** — route registration + the `guardLocalUI` BindFunc (`:107-128`, `:172`). `guardLocalUI` rejects non-loopback hosts and cross-origin state-changers; a GET with no `Origin` passes. New routes register inside `Register`:
```go
199 	se.Router.GET("/", h.root)
200 	se.Router.GET("/storybook", h.storybookHome)
...
212 	se.Router.GET("/ui/chat/nudges", h.chatNudges)
...
236 	se.Router.GET("/ui/cards/{type}", h.uiCard)
237 	se.Router.GET("/focus/{type}", h.focusPage)
```

**`internal/feature/settingscards/settingsfocus.go:334-346`** — the EXACT inject-not-navigate precedent (kept `Href` as a no-JS fallback, added a typed Datastar click action with prevent-default):
```go
334 func settingsNavLink(active, section, label string) g.Node {
...
339 	href := "/focus/settings?section=" + section
340 	return A(
341 		Class(cls),
342 		Href(href),
343 		data.On("click", "@get('"+href+"')", data.ModifierPrevent),
344 		g.Text(label),
345 	)
346 }
```
imports `data "maragu.dev/gomponents-datastar"`. Its test (`settingsfocus_test.go:126`) asserts the rendered string `data-on:click__prevent="@get(&#39;/focus/settings?section=profile&#39;)"`. **This is the template for the SidebarItem action.**

**Sidebar CSS — `internal/web/assets/static/basm.css:2682-2727`** — `.sb-side` (sticky rail, wood), `.sb-nav`, `.sb-nav-item` (a flex `<a>` with `gap:10px`), `.sb-nav-dot` (6×6), `.sb-foot`. **`.sb-nav-item` already lays out an icon-sized leading element via `gap:10px`** — adding an `<img>` before the label needs only a small icon-size rule.

**Home full-canvas CSS — `internal/web/assets/static/basm.css:3217-3261`** — `html.home #dock { left:0; width:auto; z-index:var(--z-overlay); padding-inline:var(--pad); }` centers the chat column at `--w-chat-home` (1800px) and hides the dock-grip/dock-head. **This is the full-canvas trick the single-page layout generalizes.** Width tokens `--w-chat-home:1800px`/`--w-chat-overlay:940px` at `:165-166`; `--z-overlay:50`/`--z-drawer:60` at `:186-188`.

**`internal/ui/shell/shell_test.go`** — `TestPage` (`:12`) asserts the topbar nav + skip-link-before-topbar; `TestTopbarDrawer` (`:74`); `TestPageHTMLClass` (`:108`) asserts `HTMLClass` lands on `<html>`. These test `shell.Page` (the OLD shell, still used by `/focus` until 089) — **leave them; add NEW tests for the new shell.**

**`internal/web/home_test.go:14-38`** — `TestHomeFullChat` asserts the home page contains the topbar nav links (`<a href="/focus/quests">Quests</a>`, `<a href="/focus/settings">Settings</a>`) and `<main id="main"></main>`. **The new home shell has NO topbar and NO `#main` — these assertions MUST be migrated** (Step 9). `TestHomeDockSelectorIDs` (`:52`) asserts the SSE selector IDs — those MUST still pass (the dock is unchanged).

## Commands you will need
| Purpose | Command | Expected |
|---|---|---|
| Drift check | see header | excerpts match; go.mod pins intact |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test (all) | `go test ./...` | all pass |
| Shell tests | `go test ./internal/ui/shell/...` | pass |
| Web tests | `go test ./internal/web/...` | pass |
| Storybook render gate | `go test ./internal/feature/storybook/...` | `TestAllStoriesRender` passes |
| Route registered | `grep -n '/ui/show/{type}' internal/web/web.go` | present |
| gofmt | `gofmt -l internal/` | no output |
| Whitespace | `git diff --check` | no output |

Sandbox note: a TLS-intercepting Hyperagent sandbox needs the GOPROXY shim (`docs/hyperagent-sandbox.md`).

## Scope
**In scope (this plan):**
- `internal/ui/shell/sidebar.go` — extend `SidebarItem` with `Icon` and `Action` fields (additive, back-compatible); render an `<img>` icon and a `data-on:click__prevent` action while keeping `Href` as a no-JS fallback.
- `internal/ui/shell/sidebar.go` — a NEW `ChatShell` composition (left `Sidebar` + full-canvas chat dock as the primary column, no topbar/`#main`/right-rail) used ONLY by home. Keep `Page`/`Topbar`/`SidebarPage` intact.
- `internal/web/home.go` — switch `homePage` to render the new `ChatShell` (with the domain sidebar + the existing `chat.Dock`). The dock node and its IDs are unchanged.
- `internal/web/cards.go` or a new `internal/web/show.go` — a NEW `GET /ui/show/{type}` handler: validate the type, persist a `messages` row (`role=tool`, `origin=""`) carrying the `uicard` marker via `conversation.AppendOriginRec`, then render that saved record through `messageViews`/`renderMessages` and append into `#chat`.
- `internal/conversation/conversation.go` — add `AppendOriginRec(...) (*core.Record, error)` (additive; `AppendOrigin` delegates to it) + a unit test. Needed so the handler can re-render the exact persisted row (Step 5a).
- `internal/web/web.go` — register `GET /ui/show/{type}`.
- `internal/web/assets/static/basm.css` — generalize the home full-canvas dock into the permanent two-column `[sidebar | chat]` layout; small `.sb-nav-item img` icon rule. Append new rules at END.
- `internal/feature/storybook/stories_navigation.go` (+ `story.go` registry) — a Story for the product sidebar variant (icons + injecting items) and/or the single-page chat shell.
- `internal/web/home_test.go` + a new `internal/ui/shell/sidebar_test.go` — migrate the home topbar assertions to the new shell; cover the new `SidebarItem` fields.

**Out of scope (explicit — do NOT do here):**
- **Retiring** `shell.Topbar`, the `/focus/{type}` pages, the right-rail dock layout, or **dropping the boards collection** — that is plan **089**. The old shell + `/focus` MUST keep working after this plan.
- **Ad-hoc clusters / multi-card artifacts / a `show_cards` tool / a bare "tasks" cluster card** — plan **090** (new `ArtifactMarker`, touches chatstream + recap again).
- **Head/model switchers, recap relocation, theme/palette moved into the sidebar, measure caps, reduced-motion/a11y polish** — plan **091**. This plan adds only a MINIMAL sidebar footer (light/dark toggle + a "new conversation"/home affordance) so the shell is usable.
- Any new migration, any change to the `card_show` NL path (it already works), any change to the SSE selector IDs.

## Git workflow
- Branch `improve/088-single-page-chat-sidebar` off `main`.
- Conventional commits, e.g.:
  - `feat(shell): SidebarItem gains Icon + Datastar inject Action (back-compatible)`
  - `feat(shell): ChatShell — single-page sidebar + full-canvas chat`
  - `feat(web): GET /ui/show/{type} deterministic artifact injection + persist`
  - `feat(web): home renders the single-page chat shell`
  - `feat(css): generalize the home full-canvas dock into the sidebar+chat layout`
  - `test(web,shell): migrate home topbar assertions; cover new SidebarItem`
  - `docs(storybook): product sidebar story (icons + injecting items)`
- Do NOT push or open a PR unless explicitly told.

## Steps

### Step 1: Extend `SidebarItem` (additive, back-compatible)
In `internal/ui/shell/sidebar.go`:
1. Add the import `data "maragu.dev/gomponents-datastar"` (the shell package may import it — it is NOT a feature package, so no layering violation). Keep `g`/`h`/`strconv`.
2. Extend the struct (append fields; do NOT reorder — keep `Label, Href, Active, Dot` first so existing positional/field callers are unaffected):
   ```go
   type SidebarItem struct {
   	Label, Href string
   	Active      bool
   	Dot         string
   	Icon        string // optional pixel-icon stem under /static/icons (e.g. "shield"); renders <img> before the label
   	Action      string // optional Datastar expression for click__prevent (e.g. "@get('/ui/show/quests')"); empty = plain <a href> navigation
   }
   ```
3. Update `sidebarItem` so it renders the icon (when set) and the inject action (when set), keeping `Href` as the no-JS fallback. Mirror `settingsNavLink` exactly:
   ```go
   func sidebarItem(it SidebarItem) g.Node {
   	cls := "sb-nav-item"
   	if it.Active {
   		cls += " sb-nav-item-active"
   	}
   	attrs := []g.Node{h.Class(cls), h.Href(it.Href)}
   	if it.Active {
   		attrs = append(attrs, h.Aria("current", "page"))
   	}
   	if it.Action != "" {
   		attrs = append(attrs, data.On("click", it.Action, data.ModifierPrevent))
   	}
   	if it.Icon != "" {
   		attrs = append(attrs, h.Img(h.Class("sb-nav-icon"), h.Src("/static/icons/"+it.Icon+".png"), h.Alt(""), g.Attr("decoding", "async")))
   	}
   	if it.Dot != "" {
   		attrs = append(attrs, h.Span(h.Class("sb-nav-dot"), h.Style("--sb-nav-dot:"+it.Dot)))
   	}
   	attrs = append(attrs, h.Span(g.Text(it.Label)))
   	return h.A(attrs...)
   }
   ```
   The storybook caller sets neither `Icon` nor `Action`, so its output is BYTE-IDENTICAL to today (the `if` guards skip). **Verify**: `go test ./internal/ui/shell/...` (existing storybook sidebar rendering via `go test ./internal/feature/storybook/...` must still pass — confirm `data-on:click` does NOT appear when `Action==""`).

### Step 2: The new `ChatShell` composition (single page, no topbar/#main/rail)
In `internal/ui/shell/shell.go` (or a new `internal/ui/shell/chatshell.go` in the same package) add a NEW function — do NOT modify `Page`:
```go
// ChatShellProps configures the single-page companion shell: a left domain
// Sidebar and the chat dock as the only primary surface (no topbar, no #main,
// no right rail). Sidebar is a pre-built shell.Sidebar node; Dock is the
// chat.Dock node (its #chat/#dock-convo/#nudge-poll ids are the SSE contract).
type ChatShellProps struct {
	Title   string
	Sidebar g.Node
	Dock    g.Node
}

// ChatShell renders the full <html> document for the single-page chat surface.
// The .app-shell wrapper is the [sidebar | chat] two-column grid (see basm.css).
func ChatShell(p ChatShellProps) g.Node {
	return g.Group([]g.Node{
		g.Raw("<!doctype html>"),
		h.HTML(h.Lang("en"), h.Class("app"),
			h.Head(pageHead(), h.TitleEl(g.Text(p.Title+" · Balaur"))),
			h.Body(
				h.A(h.Class("skip-link"), h.Href("#chat"), g.Text("Skip to chat")),
				h.Div(h.Class("app-shell"),
					p.Sidebar,
					h.Aside(h.ID("dock"), h.Class("app-dock"), p.Dock),
				),
			),
		),
	})
}
```
Notes:
- `<html class="app">` is the new full-canvas hook (parallels `html.home` but is the permanent layout; the skip-link points at `#chat` since there is no `#main`).
- Keep the `#dock` id on the `<aside>` (the CSS + JS `basmToggleDockFull` reference `#dock`; even though the single page has no rail mode, reusing `#dock` keeps the existing dock CSS working — the `app-dock` class is the new layout hook).
- `pageHead()` is reused verbatim (loads basm.css/datastar.js/basm.js + no-flash).

**Verify**: `go build ./internal/ui/shell/...` → exit 0.

### Step 3: The domain sidebar (feature-side, in `internal/web`)
The sidebar's domain sections are PRODUCT data (which domains, which icons, which card type each injects), so build them in `internal/web` — NOT in `shell` (shell stays generic). Add a helper in `internal/web/home.go` (or a new `internal/web/sidebar.go`):
```go
// domainSidebar builds the left domain rail for the single-page shell. Each item
// injects its domain's card into the chat via GET /ui/show/{type} (deterministic,
// no LLM). Icons are the registry icon stems (cards.Spec.Icon) under /static/icons.
func domainSidebar() shell.SidebarProps {
	item := func(label, typ, icon string) shell.SidebarItem {
		return shell.SidebarItem{
			Label:  label,
			Href:   "/ui/show/" + typ, // no-JS fallback (still appends, see handler)
			Icon:   icon,
			Action: "@get('/ui/show/" + typ + "')",
		}
	}
	return shell.SidebarProps{
		Brand: /* crest + "Balaur" — mirror storybook.go sidebarFor Brand */ ,
		Sections: []shell.SidebarSection{{
			Label: "Domains",
			Items: []shell.SidebarItem{
				item("Quests", "quests", "scroll"),
				item("Knowledge", "memory", "tome"),
				item("Life", "lifelog", "orb"),
				item("Journal", "journal", "quill"),
				item("Heads", "heads", "shield"),   // NOT "tome" — Knowledge already uses tome; keep stems distinct
				item("Settings", "settings", "key"),
			},
		}},
		Footer: /* minimal: light/dark toggle + a "new / home" link — see Step 4 */ ,
	}
}
```
- The card types and icons are pinned to `internal/cards/cards.go` (`quests`/`memory`/`lifelog`/`journal`/`heads`/`settings` all exist; icon stems chosen from the available `/static/icons/*.png`: `scroll, tome, orb, quill, key, shield, lens, gem, flame, hourglass, bell, check, rune_x`). **The prompt named shield/lens/gem/scroll/key as the domain pixel icons — pick from the registry's `Spec.Icon` where sensible, but any existing icon stem is fine; do NOT invent new icon files.** Confirm each chosen stem file exists before using it (`ls internal/web/assets/static/icons/`).
- The owner-decided domain list is **Quests / Knowledge / Life / Journal / Heads / Settings**.

**Verify**: `go build ./internal/web/...` → exit 0.

### Step 4: Minimal sidebar footer (theme toggle + new-conversation/home)
In `domainSidebar`'s `Footer`, render only:
- The light/dark toggle (copy the button shape from `internal/web/storybook.go:100-104`: `theme-toggle` class, `onclick="basmToggleTheme()"`, `aria-pressed="false"`, `aria-label`). The richer palette picker + head/model switchers + recap relocation are **plan 091** — do NOT add them here.
- A "new conversation / home" affordance — a plain `<a href="/">` link (label e.g. "Home" or a crest), since `/` IS the conversation home. (A true "new conversation" requires conversation-switching plumbing that does not exist for the master thread — do NOT build it; the home link is the honest minimal affordance.)

Keep the footer to those two controls. **Verify**: render it in the storybook story (Step 7) and confirm it shows.

### Step 5: The deterministic endpoint `GET /ui/show/{type}`
Add `func (h *handlers) uiShow(e *core.RequestEvent) error` (in `internal/web/cards.go` or a new `internal/web/show.go`). It must: validate, append the card live, AND persist the marker row. Shape (copy `chatNudges` for the SSE open + append; copy `uiCard` for validation):
```go
// uiShow handles GET /ui/show/{type}?params — the DETERMINISTIC artifact door.
// A sidebar click (Datastar @get) injects the domain's card into #chat without
// the agent. It is persisted as a messages row carrying the SAME uicard marker
// card_show uses, so recap.messageViews re-renders it on reload (one code path,
// two doors). Single card only — clusters are plan 090.
func (h *handlers) uiShow(e *core.RequestEvent) error {
	typ := e.Request.PathValue("type")
	if _, ok := cards.Get(typ); !ok {
		return e.NotFoundError("no such card type", nil)
	}
	params, err := cards.Validate(typ, queryToMap(e.Request.URL.Query()))
	if err != nil {
		return e.BadRequestError("invalid card params", err)
	}

	// Persist EXACTLY like a card_show result: role:"tool", origin:"" (empty).
	// The marker content is what recap.messageViews re-parses on reload; modelText
	// is the visible tool-row line (mirrors card_show: "showing the owner the
	// <Label> card"). AppendOriginRec (Step 5a) returns the saved record so the live
	// append renders through the SAME messageViews path as reload — guaranteeing
	// live == reload. tool_name = typ records provenance; origin stays "" so the
	// chatNudges (origin != '') poller never re-appends it.
	spec, _ := cards.Get(typ)
	modelText := "showing the owner the " + spec.Label + " card"
	marker := tools.MarkUICard(typ, params, modelText)
	master, mErr := conversation.Master(h.app)
	if mErr != nil {
		return e.InternalServerError("master conversation", mErr)
	}
	rec, aErr := conversation.AppendOriginRec(h.app, master.Id,
		llm.Message{Role: "tool", Content: marker}, typ, "") // origin "" — NOT "artifact"
	if aErr != nil {
		return e.InternalServerError("persisting artifact", aErr)
	}

	// Render the same tool-row + card chrome the live stream + reload produce by
	// running the just-saved record through the EXACT reload path (messageViews
	// parses the marker into CardBody; renderMessages emits the tool row + the
	// k-inline card). This is why live and reload are byte-identical.
	body := h.renderMessages(h.messageViews([]*core.Record{rec}))

	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(string(body), datastar.WithSelectorID("chat"), datastar.WithModeAppend())
	return nil
}
```
**Step 5a — `conversation.AppendOriginRec` (additive, resolves the "AppendOrigin returns no record" gap):** `AppendOrigin` returns only `error`, so the handler cannot render the exact persisted row without it. Add a thin variant in `internal/conversation/conversation.go` that returns the saved `*core.Record`, and make the existing `AppendOrigin` delegate to it (byte-identical behavior; do not change `AppendOrigin`'s signature — other callers depend on it):
```go
// AppendOriginRec is AppendOrigin but returns the saved record (so a caller can
// re-render exactly what was persisted). AppendOrigin delegates to it.
func AppendOriginRec(app core.App, conversationID string, msg llm.Message, toolName, origin string) (*core.Record, error) {
	// ... the current AppendOrigin body, returning (rec, nil) on success ...
}
func AppendOrigin(app core.App, conversationID string, msg llm.Message, toolName, origin string) error {
	_, err := AppendOriginRec(app, conversationID, msg, toolName, origin)
	return err
}
```
Add an `internal/conversation` unit test that `AppendOriginRec` returns a non-nil record whose `id`/`content` round-trip, and that `AppendOrigin` still behaves as before. This keeps the live append on the SAME `messageViews` code path as reload — no hand-built `messageView`, no second query, no drift.

**The unique-id requirement**: the live stream mints per-element ids from `s.base` (a per-turn nonce, `chatstream.go:65`). `/ui/show` has no nonce. `renderMessages` for a `tool` row emits `chat.ToolRow` + a `k-inline` div WITHOUT an id (see `recap.go:201` — the reload path's `k-inline` has no id), and `chat.Message`/`ToolRow` from history carry no streaming id either. So this path collides with nothing — **do not add per-turn ids**; rendering the persisted record through `messageViews`/`renderMessages` (Step 5a) is exactly what makes the live append and the reload consistent. Confirm by reading `recap.go:188-212` (`renderMessages`; the `tool` case is ~`:196-202`) that the history render path uses no per-element ids.

**`guardLocalUI` contract**: `/ui/show/...` is a GET → the `guardLocalUI` Origin check (`web.go:119`) only fires on non-GET, so a same-origin Datastar `@get` passes; the host check still applies (loopback). No form/JSON body to read — params come from the URL query (`cards.Validate(queryToMap(query))`), same as `uiCard`. **Verify**: the route is reachable in a test (Step 8).

Add the imports `conversation`, `tools`, `llm` to the handler file as needed (all already imported elsewhere in `internal/web`).

### Step 6: Register the route
In `internal/web/web.go` inside `Register`, beside the other `/ui/...` GETs (e.g. after `:236` `se.Router.GET("/ui/cards/{type}", h.uiCard)`):
```go
	se.Router.GET("/ui/show/{type}", h.uiShow)
```
**Verify**: `grep -n '/ui/show/{type}' internal/web/web.go` → present; `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 7: Switch home to the new shell
In `internal/web/home.go`, change `homePage` to render `shell.ChatShell` with the domain sidebar + the existing dock node (do NOT change the `chat.Dock(...)` call — only the wrapping shell):
```go
	page := shell.ChatShell(shell.ChatShellProps{
		Title:   "Home",
		Sidebar: shell.Sidebar(domainSidebar()),
		Dock:    dockNode,
	})
```
Drop the now-unused `Active:"home"`/`HTMLClass:"home"` PageProps. Keep `composerNode`, `dockData`, the `dockNode` build, and the render/error handling. **Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `go vet ./...` → exit 0.

### Step 8: CSS — generalize the full-canvas dock into the [sidebar | chat] layout
Append to the END of `internal/web/assets/static/basm.css` (tokens only, square corners, single-dash classes; new rules at end):
1. `.app-shell` — the two-column grid: `display:grid; grid-template-columns: <sidebar-w> 1fr; min-height:100vh;` (reuse `--sidebar-w` or a new `--w-rail` token; the storybook rail uses `274px` — match it). The left column is the `Sidebar`'s `.sb-side` (already sticky/wood); the right column is `#dock.app-dock`.
2. `#dock.app-dock` — generalize the `html.home #dock` rules (`basm.css:3224-3261`): full-height flex column, `z-index:var(--z-base)` (it is now an in-grid column, not an overlay), the chat column centers at `--w-chat-home`, `padding-inline:var(--pad)`, dock-grip/dock-head hidden. **Do NOT touch the existing `html.home #dock` block** (still used by the old home path? No — home now uses `.app`; but `/focus` + the `.dock-full` overlay still use `#dock`, so leave all existing `#dock` rules intact and ADD `.app-dock` overrides). Reuse the exact column-centering rules from `:3238-3261` (chat/composer `max-width:var(--w-chat-home); margin:auto`).
3. `.sb-nav-icon` — `width:16px; height:16px; flex-shrink:0; image-rendering:pixelated;` (the `.sb-nav-item` flex `gap:10px` already spaces it; mirror `.sb-brand img` pixelation at `:2694`).
4. Responsive: at **`≤720px`** (the canonical breakpoint — see the 540/720/920 scale) collapse `.app-shell` to a single column (`grid-template-columns: 1fr`) so the rail stacks full-width above the chat. **Keep it MINIMAL** — the polished off-canvas drawer (burger + backdrop + ≥44px targets) is plan **091**. The minimal stacking adds no transition, so no `prefers-reduced-motion` guard is needed here; if you add any transition, add its selector to the existing reduced-motion block. STOP-worthy check: at ~390px there must be no horizontal scroll.
5. Legibility: the sidebar sits on wood (`--chrome`/`--chrome-fg`, always-dark — see `.sb-side`); the chat dock is the existing parchment-on-wood. No new color decisions. Verify the icon `<img>` is visible on the wood rail in both light and dark (pixel icons are theme-agnostic PNGs — fine).

**Verify**: `git diff --check` → no output; build the app and load `/` (see Test plan) — sidebar on the left, chat fills the rest, a domain click injects a card. If you cannot run a browser, rely on the handler test (Step 9) + the storybook story.

### Step 9: Tests — migrate home, cover the new pieces
1. **`internal/web/home_test.go`**: `TestHomeFullChat` currently asserts the topbar nav + `<main id="main"></main>`. The new home has neither. Migrate (do NOT delete the test — repoint its assertions):
   - REMOVE: `<html lang="en" class="home">`, `<main id="main"></main>`, the `<a href="/focus/quests">Quests</a>` / `<a href="/focus/settings">Settings</a>` topbar assertions, and the `>Today</a>` NotExpected (no topbar at all now).
   - ADD: `<html lang="en" class="app">`, `class="app-shell"`, `class="sb-side"`, the domain sidebar items (e.g. `>Quests</a>` is now inside a `.sb-nav-item` with `data-on:click__prevent="@get(&#39;/ui/show/quests&#39;)"`), `class="sb-nav-icon"`, and the footer `theme-toggle`.
   - KEEP `TestHomeDockSelectorIDs` passing unchanged (the dock IDs `#chat`/`#dock-convo`/`#chat-draft`/`#nudge-poll`/`#model-modal`/`data-signals:streaming` are all still present — `chat.Dock` is unchanged). If it fails, the dock node was altered — STOP, revert.
   - `TestFocusDockSelectorIDs` (`/focus/quests`) MUST still pass — `/focus` still uses the OLD `shell.Page`. Do NOT touch it.
2. **NEW `internal/ui/shell/sidebar_test.go`**: assert (a) a plain item (no Icon/Action) renders byte-identical to before — `<a class="sb-nav-item" href="..."><span>Label</span></a>` with the dot, and NO `data-on:click`; (b) an item with `Icon` renders `<img class="sb-nav-icon" src="/static/icons/scroll.png" ...>`; (c) an item with `Action:"@get('/ui/show/quests')"` renders `data-on:click__prevent="@get(&#39;/ui/show/quests&#39;)"` and STILL keeps `href` (no-JS fallback).
3. **NEW handler test in `internal/web/handlers_test.go`** (or `show_test.go`) using the `newWebApp`/`tests.ApiScenario` harness (see `home_test.go` for the page pattern and the existing `chat/nudges` test for the SSE-append assertion pattern): `GET /ui/show/quests` returns 200, `Content-Type: text/event-stream`, with a Datastar patch appending the quests card markup to `#chat` (mode append, selector `#chat`), AND a `messages` row persisted (`role=tool`, `origin=""`, content has the `\x00balaur-uicard:` marker). Assert reload re-renders: load `/`, confirm the persisted artifact card appears in the chat body (it flows through `dockData` → `messageViews` → the `uicard` branch). Assert a subsequent `GET /ui/chat/nudges?since=<page-load>` does NOT re-append the card (the `origin=""` row is excluded by the poller's `origin != ''` filter). Also assert `GET /ui/show/bogus` → 404.
4. **`shell_test.go`** (`TestPage`, `TestTopbarDrawer`, `TestPageHTMLClass`): LEAVE THEM — they test the OLD `shell.Page` which 089 retires. Do not gut them.

**Verify**: `go test ./internal/web/... ./internal/ui/shell/...` → all pass.

### Step 10: Storybook — the product sidebar story
Per the storybook-source-of-truth workflow (`ui-development` skill): add a Story so the new sidebar variant is documented. In `internal/feature/storybook/stories_navigation.go` add a `sidebarStory()` (Group "Navigation"), and register it in `story.go`'s `stories` slice (Step `story.go:53-103`). Render a `shell.Sidebar` with a `Domains` section of injecting items (Icon + Action set) + the minimal footer, marking one active. Use `Wide: true` (the rail is full-bleed, like `topbarStory`). Document props: the new `Icon` + `Action` fields. Do/Don't: "Use a sidebar item's `Action` to inject a card into the chat (deterministic door); keep `Href` as the no-JS fallback" / "Don't put per-page navigation here — clicking injects, it does not navigate."
- Optionally add a `chatshellStory()` rendering the whole `ChatShell` (sidebar + a stub dock) — but a full-document `<html>` does not embed cleanly in a storybook tile, so the sidebar-only variant is the pragmatic choice. If you skip the full-shell story, note it in the story blurb.

**Verify**: `go test ./internal/feature/storybook/...` → `TestAllStoriesRender` passes; `TestStoriesUniqueAndLookup` passes (the new id is unique).

### Step 11: Full verification + index
Run the Done-criteria gate (below). Update the 088 row in `plans/readme.md`.

## Test plan
- **Unit (`internal/ui/shell`)**: the new `sidebar_test.go` — plain item byte-stable, Icon renders `<img class="sb-nav-icon">`, Action renders `data-on:click__prevent` and keeps `href`. The existing `shell_test.go` (`TestPage`/`TestTopbarDrawer`/`TestPageHTMLClass`) stay green (old shell untouched).
- **Web handler**: `GET /ui/show/{type}` — 200 + SSE append to `#chat` for a valid type; 404 for an unknown type; 400 for a required-param-missing type (e.g. a type whose spec has a `Required` param, validated by `cards.Validate`). A `messages` row is persisted with `role=tool`, `origin=""`, marker content. Reload (`GET /`) re-renders the persisted artifact via the existing `uicard` reload branch. Model the SSE-append assertion on the existing `chatNudges` test (search `internal/web/*_test.go` for `chat/nudges` / `WithModeAppend`): assert the response is `text/event-stream` and the patch targets `#chat` (append) with the card markup. Add a poll-no-duplicate test (a `/ui/chat/nudges` call after `/ui/show` returns no extra card).
- **Selector-ID contract**: `TestHomeDockSelectorIDs` + `TestFocusDockSelectorIDs` both pass — the SSE hot path is intact on the new home AND the still-present `/focus` pages.
- **Storybook**: `go test ./internal/feature/storybook/...` (`TestAllStoriesRender`).
- **Manual (browser, if available — see `run`/`verify` skills or Chrome over Playwright)**: load `/` → left domain sidebar + full-canvas chat; click "Quests" → the quests card appends into the conversation, no page navigation, no LLM; reload → the card is still there (re-rendered live from data); type "show me my quest list" to the model (if a model is configured) → the `card_show` NL door appends the same card. Check `messages` shows the persisted `role=tool`/`origin=""` row (with the `uicard` marker). Verify light AND dark mode legibility (toggle in the footer): sidebar text on wood (`--chrome-fg`) and the chat parchment both readable.

## Done criteria
- [ ] `CGO_ENABLED=0 go build ./...` → exit 0; `go vet ./...` → exit 0; `gofmt -l internal/` → no output; `git diff --check` → no output.
- [ ] `go test ./...` → all pass, including the new `sidebar_test.go`, the `/ui/show` handler test, and the migrated `home_test.go`.
- [ ] `grep -n '/ui/show/{type}' internal/web/web.go` → present.
- [ ] `SidebarItem` gained `Icon` + `Action` (additive); the storybook sidebar caller renders byte-identical to `3136bad` (no `data-on:click` when `Action==""`).
- [ ] `GET /` renders `<html ... class="app">` + `.app-shell` + the domain `.sb-side` + the chat dock; NO `.topbar`, NO `<main id="main">`, NO right-rail layout. (Verify by grep on the rendered home test output.)
- [ ] A sidebar domain click injects that domain's card into `#chat` (deterministic, no LLM) AND it survives reload (persisted `messages` row, `role=tool`, `origin=""`, `uicard` marker, re-rendered by the EXISTING `recap.messageViews` `uicard` branch — no new reload code), via `conversation.AppendOriginRec` so live == reload.
- [ ] A `/ui/chat/nudges` poll fired AFTER a `/ui/show` does NOT duplicate the injected card (the `origin=""` row is invisible to the `origin != ''` poller); `tasks.go` is unchanged.
- [ ] The `card_show` NL door is unchanged and still works (no edits to `internal/tools/ui.go` or `chatstream.handleToolResult`).
- [ ] The SSE selector IDs (`#chat`, `#dock-convo`, `#chat-draft`, `#nudge-poll`, `#model-modal`, `data-signals:streaming`) are present on the new home AND on `/focus/quests` (both selector-ID tests pass).
- [ ] `shell.Page` + `Topbar` + `SidebarPage` + all `/focus/{type}` routes + boards routes STILL work (old shell untouched — plan 089's job to remove them).
- [ ] A storybook Story for the product sidebar variant (icons + injecting items) exists and `TestAllStoriesRender` passes.
- [ ] No new migration added; next free timestamp remains `1750850000`.
- [ ] `plans/readme.md` 088 row updated.

## STOP conditions (specific to this plan)
- **Drift**: a cited file or the `gomponents-datastar`/`datastar-go` go.mod pin changed since `3136bad` and an excerpt no longer matches — STOP and report.
- **A selector-ID test breaks** (`TestHomeDockSelectorIDs` or `TestFocusDockSelectorIDs`) — STOP. You altered the dock's contract; the streaming hot path depends on those exact IDs. Revert the dock change and re-do via the wrapping shell only.
- **The injected artifact shows live but vanishes on reload** (or vice-versa) — STOP. The live append and the reload render must both go through the `uicard` marker; the persisted-row `Role` MUST be `"tool"` (the only role `recap.messageViews` scans for markers — `recap.go:262`). If you persisted it as `assistant`/`user`, the reload branch never fires.
- **Tempted to remove the topbar/`/focus`/boards or drop the boards collection** — STOP. That is plan **089**; removing them here breaks the still-live `/focus` pages and their tests.
- **Tempted to add multi-card clusters, a `show_cards` tool, a new marker, or a bare "tasks" cluster card** — STOP. That is plan **090** (new `ArtifactMarker`, touches chatstream + recap again).
- **Tempted to add head/model switchers, palette picker, or recap relocation into the sidebar** — STOP. Plan **091**. The footer here is ONLY the light/dark toggle + a home link.
- **`internal/ui/shell` reaches into `internal/feature/*`** (to build domain sections) — STOP. The domain sections (which card each item injects) are built in `internal/web` and passed in as `shell.Sidebar(...)` nodes; `shell` stays generic.
- **A Verify fails twice** after a fix attempt — STOP and report the command + output.

## Maintenance notes
- **Two render paths, one marker**: the deterministic door (`/ui/show`) and the reload door (`recap.messageViews`) both render the artifact from the SAME `\x00balaur-uicard:` marker via `h.cardHTML`/`uicardBody`. If plan 090 adds a NEW `ArtifactMarker` for clusters, it MUST be handled in BOTH `chatstream.handleToolResult` AND `recap.messageViews` (and, if `/ui/show` learns clusters, here too). The "one marker handled in two places" rule is the recurring trap.
- **Why `origin:""` and NOT a new `origin:"artifact"` (the double-append trap, resolved by design)**: the nudge poller `chatNudges` (`tasks.go:311`) runs every 30s and appends every `messages` row with `origin != '' && created > {:since}`. If the deterministic artifact were persisted with a non-empty origin, that poll would **re-append the just-injected card** (the `/ui/show` append is immediate/client-side; the poll fires later and sees a newer-than-cursor row) — a visible duplicate. Persisting with `origin:""` makes the row invisible to the poller (it only collects agent-initiated `nudge`/`briefing`/`check` rows) with **no change to `tasks.go`**, and `recap.messageViews` still re-renders it (it keys on `role=="tool"` + the marker, not origin). The row is also excluded from `RecentTurns` model context (role `tool`). Provenance lives in `tool_name` (the card type) if ever needed. Do NOT add an `origin:"artifact"` value and do NOT touch the `chatNudges` filter — that path was considered and rejected as more moving parts for no benefit. Still add a Step 9 test asserting that a `/ui/chat/nudges` poll after a `/ui/show` does NOT duplicate the card (the regression net for this reasoning).
- **Tool-row chrome on a sidebar click (deliberate, refinable)**: reusing the `role=="tool"` reload branch means a deterministic injection renders with a tool-row above the card (like `card_show`). That is the KISS choice (one render path, zero new branches). If the owner later wants a *bare* card (no tool row) for sidebar clicks, that is a small `messageViews`/`renderMessages` refinement — record it as a plan 091 follow-up, do not build a second render path here.
- **The old shell is deliberately left intact** — plan 089 deletes `Topbar`/`/focus`/boards and rewrites `DESIGN.md` + `internal/self/knowledge.md`. After 088, `internal/self/knowledge.md` is briefly stale (home is single-page, but `/focus` still exists); 089 reconciles it. Do NOT rewrite knowledge.md here beyond a one-line note if you touch it at all (out of scope — leave to 089).
- **Sidebar layering**: `SidebarItem.Action` is a free Datastar expression string built in `internal/web` from a registry-validated card type (`/ui/show/<type>`); the endpoint re-validates via `cards.Get`/`cards.Validate`, so a hand-edited URL cannot render an unknown type (same defense as `uiCard`).
