# Plan 091: Sidebar chrome — head/model switchers, recap, theme/palette into the rail + responsive/a11y polish

> **Executor instructions**: Follow step by step. Run every **Verify** and confirm its expected output before moving on. On a STOP condition, stop and report — do not improvise. This is the LAST plan of the 4-plan single-page program (088 → 089 → 090 → 091). It ENRICHES the minimal sidebar that plan **088** shipped (088's footer is only a light/dark toggle + a Home link) into the full companion-control surface, and closes the responsive/a11y gaps on the new surfaces. When done, update the 091 row in `plans/readme.md` (the advisor owns the index — add the row only if absent, matching the existing column format; do NOT rewrite prose).
>
> **Drift check (run first)**:
> `git diff --stat 3136bad..HEAD -- internal/ui/shell/sidebar.go internal/ui/chat/dock.go internal/web/home.go internal/web/models.go internal/web/heads.go internal/web/storybook.go internal/web/web.go internal/feature/storybook/stories_chat.go internal/feature/storybook/stories_navigation.go web/templates/home.html internal/web/assets/static/basm.css internal/web/assets/static/basm.js`
> If any of those changed since `1a06330`, compare the "Current state" excerpts below to the live code; on mismatch NOT explained by the reconciliation note below, STOP and report. Confirm `maragu.dev/gomponents-datastar v0.3.3` and `github.com/starfederation/datastar-go v1.2.2` are still in `go.mod`.
>
> **RECONCILED FOR CURRENT MAIN (2026-06-17, anchor `1a06330`) — READ FIRST.** Plans 086/087/088/089/090 have all LANDED on main. This plan's premises are CONFIRMED true against the live tree; the deltas to absorb:
> - `domainSidebar()` exists (`internal/web/home.go:47`) WITH the minimal footer 088 shipped (a `theme-toggle` button + a `<a href="/">` Home link). **ENRICH it in place — do NOT recreate it.** Add the head switcher to the `Brand` slot and the model switcher into the `Footer` (alongside the existing theme toggle; add the palette buttons + a recap affordance).
> - **The switcher `/focus/settings` links are ALREADY re-pointed to `/ui/show/settings`** (089 did it — confirmed at `web/templates/home.html:24,29,37`). So Step 2's "repoint the manage links" is ALREADY DONE — just VERIFY it (`grep -n 'focus/settings' web/templates/home.html` → empty); do not re-point again.
> - Pin 1 HOLDS: `chat.HeadSwitcher`/`ModelSwitcher` gomponents organisms still do NOT exist (`grep 'func HeadSwitcher' internal/` → empty); the switchers are still `{{define "chat_bar"}}`/`"model_switcher"`/`"head_switcher"` in `web/templates/home.html`. Inject them via `g.Raw` (Step 1A). Pin 2 HOLDS: `patchChatbar` patches `#chatbar` (`internal/web/models.go:122`), `setActiveHead` patches `#head-switcher` (`internal/web/heads.go:33`) — PRESERVE those ids verbatim, do NOT touch the handlers.
> - `sidebarStory()` exists (`internal/feature/storybook/stories_navigation.go:110`, from 088) — ENRICH it (add the chrome variants); the dock Story's `Switchers` note is in `stories_chat.go`.
> - `internal/web/assets/static/basm.css` now ends with 088's `.app-shell`/`#dock.app-dock`/`.sb-nav-icon` + 090's `.k-cluster`/`.tasks-stack` rules — APPEND your rail-chrome/drawer/`--measure` rules AFTER those, at the very end.
> - `internal/web/models.go` was heavily changed by 086/087 (model download/runtime) — you do NOT edit its handlers (Pin 2); only confirm the `#chatbar` patch target. No overlap with your work.
>
> **Dependency note**: this plan assumes plan **088** has landed — i.e. `shell.ChatShell`, `domainSidebar()` in `internal/web`, the extended `shell.SidebarItem{Icon, Action}`, `GET /ui/show/{type}`, and the `.app-shell` / `#dock.app-dock` CSS all exist. Where this plan cites a function that 088 introduces, it is marked **(introduced by 088)**. If 088 has NOT landed, STOP — this plan has nothing to enrich.
>
> **Commit anchor**: `3136bad` (2026-06-17). This plan adds NO migration.

## Status
- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: plan **088** (the single-page `ChatShell` + `domainSidebar()` + minimal footer this plan enriches). 088 must land first. Independent of 089/090 (touches different surfaces), but landing after 089 avoids re-reasoning about the retired chrome.
- **Category**: feature + polish (owner-decided, LOCKED program)
- **Planned at**: commit `3136bad`, 2026-06-17

## Why this matters
Plan 088 ships a deliberately MINIMAL sidebar footer (a light/dark toggle + a Home link) so the single-page shell is *usable* — but the companion controls that lived in the old dock chrome are now homeless. Concretely: the head switcher, the model switcher (+ its "Manage models" / model-modal), the recap telescope, and the theme **palette** picker all used to hang off the right-rail dock or the topbar. After 088 the home dock mounts `Switchers: nil` (see Current state) — so those controls render NOWHERE on the new home. This plan gives them a home: the left rail's Brand/Footer slots become the persistent companion-control surface.

It also closes the polish/a11y debt the new surfaces inherit, per the `ui-development` skill's mandatory checklist: a `--measure` reading-width cap on artifact long-form text, a `prefers-reduced-motion` guard for any new animation, `aria-pressed` on the toggles, ≥44px touch targets on persistent rail controls, wood-chrome legibility verified in both light and dark, and a real off-canvas sidebar drawer for phones (rendered at body level so it escapes the grid's stacking context — the lesson from plan 078's topbar drawer).

This is the last plan; keep scope tight. Anything genuinely optional is marked **DEFERRED** below, not padded into scope.

## The decisive correctness pins (read before coding)

**Pin 1 — the switchers are STILL `html/template` fragments, NOT gomponents organisms.** The brief mentions "reuse the EXISTING `chat.HeadSwitcher` / `chat.ModelSwitcher` organisms." **Those organisms do not exist.** `grep -rn 'func HeadSwitcher\|func ModelSwitcher\|func Switcher' internal/` returns nothing. The switchers live as `{{define "head_switcher"}}` / `{{define "model_switcher"}}` / `{{define "chat_bar"}}` in `web/templates/home.html` and are rendered via `h.tmpl.ExecuteTemplate(...)` in `patchChatbar` (`internal/web/models.go:111`) and `setActiveHead` (`internal/web/heads.go:30`). So "inject as g.Node into the sidebar" cannot mean "call a gomponents organism." Pick ONE of the two strategies in Step 2 — the recommended one keeps the existing templates and injects their rendered HTML via `g.Raw`, respecting the no-feature-import layering rule without porting markup that 084 already deferred.

**Pin 2 — the SSE patch-target IDs are a hard contract.** `setActiveHead` patches `#head-switcher`; `patchChatbar` patches `#chatbar` (outer) and `#chat-draft`; the model modal is `#model-modal`. If the switchers move into the sidebar, their wrapper element IDs (`#head-switcher`, `#chatbar`, `#model-modal`) MUST be preserved verbatim so those existing SSE handlers keep finding them. Moving the *location* of `#chatbar` in the DOM is fine (Datastar patches by id, not position); renaming the id is NOT. See Current state for the exact handlers.

**Pin 3 — wood-chrome legibility is always-dark.** The rail (`.sb-side`) is wood: `background-color: var(--chrome)` with `--chrome-fg` text (`basm.css:25` `--chrome-fg: light-dark(#d6bb92, #b59872)` — a warm tan that stays legible on wood in BOTH light and dark because the wood itself does not flip). Any new rail control MUST use `--chrome-fg` / `--gold` / `--gold-deep` (the wood-safe tokens), NEVER parchment tokens like `--ink` / `--smoke` / `--muted` (which flip and go invisible on wood — exactly the A2 contrast bug plan 079 fixed for the recap hint). The switcher templates already use `--chrome-fg` (see `head-switcher-choice` at `basm.css:1732`), so reusing them is legibility-safe by construction; any NEW CSS you add for the rail must follow the same rule.

## Current state (confirmed excerpts — re-read at HEAD before trusting)

**`internal/web/home.go:62-68`** — the home dock TODAY mounts NO switchers (the `Switchers` slot is omitted ⇒ nil). 088 keeps this; THIS plan adds the switchers to the rail, not the dock:
```go
62 	dockNode := chat.Dock(chat.DockProps{
63 		Variant:   chat.DockHome,
64 		HasRecap:  dock.HasRecap,
65 		NowMillis: dock.NowMillis,
66 		Convo:     g.Raw(string(dock.ChatBodyHTML)),
67 		Composer:  composerNode(dock),
68 	})
```

**`internal/ui/chat/dock.go:51-68`** — `Dock` appends `p.Switchers` after the Composer and `#model-modal`. The `Switchers` slot is the dock's seam; it is nil on home today. The `<dialog id="model-modal">` is emitted unconditionally inside the dock:
```go
51 func Dock(p DockProps, attrs ...g.Node) g.Node {
...
56 	wrapAttrs = append(wrapAttrs,
57 		dockGrip(),
58 		dockHead(),
59 		recapZone(p.HasRecap),
60 		dockConvo(p.Convo),
61 		nudgePoll(p.NowMillis),
62 		p.Composer,
63 		h.Dialog(h.ID("model-modal"), g.Attr("aria-labelledby", "model-modal-title")),
64 		p.Switchers,
65 	)
66 	return h.Div(wrapAttrs...)
67 }
```

**`internal/ui/chat/dock.go:88-96`** — `recapZone` (the telescope sentinel) renders ONLY when `HasRecap`; it lazy-loads recap bands on intersect. This stays in the dock by default; the rail gets a "Recap" *affordance* (a button that scrolls to / triggers it), not a duplicate sentinel — see Step 3:
```go
88 func recapZone(hasRecap bool) g.Node {
89 	if !hasRecap {
90 		return nil
91 	}
92 	return h.Div(h.ID("recap"), h.Class("recap-zone"),
93 		g.Attr("data-on:intersect__once", "@get('/ui/recap/bands')"),
94 		h.P(h.Class("recap-hint"), g.Text("◇ further back…")),
95 	)
96 }
```

**`web/templates/home.html:11-65`** — the switcher fragments (STILL templates, deferred from plan 084). NOTE the `/focus/settings?...` hrefs are RETIRED by plan 089 — if 089 has landed, those links 404; this plan repoints them to the deterministic settings door (`/ui/show/settings`) when it touches the fragment:
```html
11 {{define "chat_bar"}}
12 <div class="chatbar chatbar-slim" id="chatbar"
13      {{if not .ChatReady}}data-on:interval__duration.2s="@get('/ui/chatbar')"{{end}}>
14   {{template "head_switcher" .}}
15   {{template "model_switcher" .}}
16 </div>
17 {{end}}
18
19 {{define "model_switcher"}}
20 <section class="model-switcher" aria-label="Model">
21   <div class="model-switcher-head">
22     <span class="model-switcher-kicker">Model</span>
23     {{if .ActiveModel}}<span class="model-current">{{.ActiveModel}}</span>{{end}}
24     <a class="model-switcher-manage" href="/focus/settings?section=models">Manage models →</a>
25   </div>
...
45 {{define "head_switcher"}}
46 <section class="head-switcher" id="head-switcher" aria-label="Head">
47   <span class="model-switcher-kicker">Head</span>
48   <span class="head-switcher-current">{{.ActiveHeadName}}</span>
49   <ul class="head-switcher-list">
50     {{range .HeadChoices}}
...
54         <button type="submit" class="head-switcher-choice{{if .Active}} head-switcher-choice-active{{end}}"
55                 data-attr:disabled="$streaming"
56                 {{if .Active}}aria-current="true"{{end}}>
```

**`internal/web/models.go:107-128`** — `patchChatbar` re-renders `chat_bar` into `#chatbar` (outer) and `#chat-draft`. The 2s poll lives on `#chatbar` until a model is ready. **Whatever wraps the switchers in the rail must still carry the id `chatbar` so this handler patches it:**
```go
111 func (h *handlers) patchChatbar(sse *datastar.ServerSentEventGenerator, data homeData) error {
112 	var b strings.Builder
113 	if err := h.tmpl.ExecuteTemplate(&b, "chat_bar", data); err != nil {
114 		return err
115 	}
116 	if err := sse.PatchElements(b.String(),
117 		datastar.WithSelectorID("chatbar"), datastar.WithModeOuter()); err != nil {
118 		return nil // client gone
119 	}
```

**`internal/web/heads.go:27-39`** — `setActiveHead` patches `#head-switcher` (outer) after a head switch, and the heads manage card `#ucard-heads` if present. **The id `head-switcher` is the contract:**
```go
27 	sse := datastar.NewSSE(e.Response, e.Request)
28 	// Refresh the dock switcher (always present).
29 	var sw strings.Builder
30 	if err := h.tmpl.ExecuteTemplate(&sw, "head_switcher", data); err != nil {
31 		return e.InternalServerError("rendering head switcher", err)
32 	}
33 	_ = sse.PatchElements(sw.String(), datastar.WithSelectorID("head-switcher"), datastar.WithModeOuter())
```

**`internal/web/storybook.go:88-112`** — the storybook sidebar Footer is the exemplar for the rail footer chrome (theme toggle + palette buttons). Copy this shape (it already carries `aria-label` + `aria-pressed` via basm.js sync):
```go
97 		Footer: g.Group([]g.Node{
98 			hh.Div(hh.Class("sb-foot-row"),
99 				hh.Span(hh.Class("sb-foot-label"), g.Text("Theme")),
100 				hh.Button(hh.Class("theme-toggle sb-foot-mode"), hh.Type("button"),
101 					g.Attr("onclick", "basmToggleTheme()"),
102 					hh.Title("Toggle day / night"), hh.Aria("label", "Toggle light/dark mode"),
103 					g.Text("◑")),
104 			),
105 			hh.Div(hh.Class("sb-foot-themes"),
106 				paletteBtn("hearthwood", "Hearth"),
107 				paletteBtn("forest", "Forest"),
108 				paletteBtn("dungeon", "Dungeon"),
109 			),
110 			hh.Div(hh.Class("sb-foot-count"), g.Text(strconv.Itoa(len(storybook.Stories()))+" components")),
111 		}),
```

**`internal/web/storybook.go:31-36`** — `paletteBtn` (reusable; wired to `basmSetPalette`, syncs `is-active` via basm.js). The rail footer reuses this exact helper:
```go
31 // paletteBtn renders one footer palette button wired to basmSetPalette.
32 func paletteBtn(key, label string) g.Node {
33 	return hh.Button(hh.Class("sb-theme-btn"), hh.Type("button"),
34 		g.Attr("data-theme", key), g.Attr("onclick", "basmSetPalette('"+key+"')"),
35 		hh.Title("Theme: "+label), g.Text(label))
36 }
```

**`internal/ui/shell/sidebar.go:24-58`** — `SidebarProps{Brand, Sections, Footer}` and `Sidebar(...)`: Brand → `<header class="sb-brand">`, Footer → `<footer class="sb-foot">`. These two slots are where the companion chrome goes (Brand = head identity, Footer = model/theme/recap). The slots take ANY `g.Node`, so injecting `g.Raw(renderedSwitcherHTML)` is layering-clean (shell never imports a feature):
```go
24 // SidebarProps configures a Sidebar. Brand (optional) is the header content;
25 // Footer (optional) is the pinned bottom slot (e.g. a theme toggle).
26 type SidebarProps struct {
27 	Brand    g.Node
28 	Sections []SidebarSection
29 	Footer   g.Node
30 }
...
50 	if p.Brand != nil {
51 		kids = append(kids, h.Header(h.Class("sb-brand"), p.Brand))
52 	}
53 	kids = append(kids, h.Nav(h.Class("sb-nav"), g.Group(groups)))
54 	if p.Footer != nil {
55 		kids = append(kids, h.Footer(h.Class("sb-foot"), p.Footer))
56 	}
57 	return h.Aside(kids...)
```

**`internal/web/assets/static/basm.css:2683-2728`** — the rail CSS (`.sb-side` wood, `.sb-foot`, `.sb-foot-row`, `.sb-foot-themes`, `.sb-theme-btn`, `.sb-foot-mode`). The rail is wood with `--chrome-fg` text; the footer controls already use `--chrome-fg` borders. NEW rail chrome reuses these classes / token rules:
```css
2684 .sb-side {
2685   display: flex; flex-direction: column; height: 100vh;
2686   position: sticky; top: 0; overflow-y: auto;
2687   background-color: var(--chrome); background-image: var(--wood-planks), var(--grain-warm); ...
2688   border-right: 2px solid var(--outline-2); box-shadow: var(--bevel-up);
2689 }
2720 .sb-foot { border-top: 2px solid var(--outline-2); padding: var(--space-3) 14px; flex-shrink: 0; }
2723 .sb-foot-mode { ... border: 1px solid var(--chrome-fg); color: var(--chrome-fg); padding: 3px 9px; cursor: pointer; }
2725 .sb-theme-btn { flex: 1; ... border: 1px solid var(--chrome-fg); color: var(--chrome-fg); padding: var(--space-1) var(--space-2); cursor: pointer; }
```

**`internal/web/assets/static/basm.css:3076-3101`** — the PROVEN off-canvas drawer for the STORYBOOK sidebar (`SidebarPage`). It is fixed at body level, translateX off-screen ≤920px, with `.sb-backdrop` scrim and a `prefers-reduced-motion` guard. The product `.app-shell` rail (088) does NOT yet have this — Step 4 adds an equivalent for it (the rail there is NOT the storybook `.sb-root` grid):
```css
3079 @media (max-width: 920px) {
3080   .sb-root { grid-template-columns: 1fr; }
3081   .sb-topbar { display: flex; ... position: sticky; top: 0; z-index: var(--z-overlay); ... }
3086   .sb-burger { ... background: none; border: 1px solid var(--chrome-fg); color: var(--gold); padding: 5px 10px; ... }
3092   .sb-side {
3093     position: fixed; inset: 0 auto 0 0; width: min(86vw, 322px); z-index: var(--z-drawer);
3094     transform: translateX(-104%); transition: transform .2s var(--ease-crisp); box-shadow: 6px 0 0 rgba(0, 0, 0, .4);
3095   }
3096   .sb-side.is-open { transform: none; }
3097   .sb-backdrop.is-open { display: block; position: fixed; inset: 0; z-index: var(--z-scrim); background: rgba(8, 5, 2, .62); }
3098   .sb-canvas { height: auto; padding: 26px var(--space-4) 80px; }
3100 } @media (max-width: 540px) { .sb-canvas { padding: 20px 10px 72px; } }
3101 @media (prefers-reduced-motion: reduce) { .sb-side { transition: none; } }
```

**`internal/web/assets/static/basm.js:194-203`** — `basmToggleNav` (storybook drawer) toggles `sb-nav-open` on `<html>` and `is-open` on `.sb-side`/`.sb-backdrop`, syncs `aria-expanded` on `.sb-burger`, and auto-closes when a `.sb-nav-item` is clicked. The product drawer (Step 4) reuses this exact function IF the product shell renders the same `.sb-side`/`.sb-backdrop`/`.sb-burger` classes — confirm 088's `ChatShell` does (it renders `shell.Sidebar` ⇒ `.sb-side`). If `ChatShell` lacks a burger/backdrop, add them at body level (Step 4):
```js
194 window.basmToggleNav = function () {
195   var open = document.documentElement.classList.toggle('sb-nav-open');
196   document.querySelectorAll('.sb-side, .sb-backdrop').forEach(function (el) { el.classList.toggle('is-open', open); });
197   document.querySelectorAll('.sb-burger').forEach(function (b) { b.setAttribute('aria-expanded', open ? 'true' : 'false'); });
198 };
199 document.addEventListener('click', function (e) {
200   if (e.target.closest('.sb-side .sb-nav-item') && document.documentElement.classList.contains('sb-nav-open')) {
201     window.basmToggleNav();
202   }
203 });
```

**`internal/web/assets/static/basm.js:24-31`** — `basmUpdateThemeButtons` already syncs `aria-pressed` on every `.theme-toggle` (added by plan 079). So a rail theme toggle that carries `theme-toggle` gets correct `aria-pressed` for free:
```js
24 function basmUpdateThemeButtons() {
25   const isLight = document.documentElement.classList.contains('light');
26   document.querySelectorAll('.theme-toggle').forEach(btn => {
27     btn.textContent = isLight ? '◑' : '☼';
28     btn.title       = isLight ? 'Switch to dark mode' : 'Switch to light mode';
29     btn.setAttribute('aria-pressed', isLight ? 'true' : 'false');
30   });
31 }
```

**`internal/web/assets/static/basm.css:972-984`** — the `prefers-reduced-motion` block. Any NEW animation selector this plan introduces (e.g. a rail drawer transition, a switcher hover) MUST be added INSIDE this block (not after its closing `}`):
```css
972 @media (prefers-reduced-motion: reduce) {
973   .btn, .btn:hover, .btn:active { transition: none; transform: none; }
...
983   .task-fresh { animation: none; }
984 }
```

**`internal/web/assets/static/basm.css:166-187`** — the layout tokens this plan uses (no raw values): `--measure: 68ch` (the artifact prose cap), `--space-*`, the `--z-*` tier (`--z-overlay: 50`, `--z-scrim: 55`, `--z-drawer: 60`):
```css
166 --w-chat-overlay: 940px;
167 --measure: 68ch;          /* prose reading-width cap (~66–75ch legibility) */
...
183 --z-base:    1;
184 --z-sticky:  5;
186 --z-overlay: 50;
187 --z-scrim:   55;
188 --z-drawer:  60;
```

**`internal/feature/storybook/stories_chat.go:172-187`** — the dock Story's `Switchers` prop doc currently says "still a template fragment — deferred from this plan." After this plan, the switchers ARE mounted (in the rail), so the Story should reflect that. The **navigation/sidebar Story** (added by 088 in `stories_navigation.go`) is where the enriched footer/brand chrome is documented (Step 5):
```go
178 			{"Switchers", "g.Node", "nil", "The chatbar/head-switcher node, pre-rendered by the caller (still a template fragment — deferred from this plan)."},
```

## Commands you will need
| Purpose | Command | Expected |
|---|---|---|
| Drift check | see header | excerpts match; go.mod pins intact |
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Test (all) | `go test ./...` | all pass |
| Web tests | `go test ./internal/web/...` | pass |
| Shell tests | `go test ./internal/ui/shell/...` | pass |
| Storybook render gate | `go test ./internal/feature/storybook/...` | `TestAllStoriesRender` + `TestStoriesUniqueAndLookup` pass |
| Switcher ids still patched | `grep -n 'WithSelectorID("chatbar")\|WithSelectorID("head-switcher")' internal/web/` | both present (unchanged handlers) |
| No parchment tokens on the rail | manual: `grep -n` the NEW rail CSS for `--ink\|--smoke\|--muted` | none |
| aria-pressed on toggles | `grep -rn 'aria-pressed\|Aria("pressed"' internal/` | new rail toggle + the 079 ones, nothing stray |
| gofmt | `gofmt -l internal/` | no output |
| Whitespace | `git diff --check` | no output |

Sandbox note: a TLS-intercepting Hyperagent sandbox needs the GOPROXY shim (`docs/hyperagent-sandbox.md`).

## Scope
**In scope (this plan):**
- `internal/web/home.go` (or a new `internal/web/sidebar.go`) — enrich `domainSidebar()` **(introduced by 088)**: put the head switcher + active-head identity into the `Brand` slot, and the model switcher + theme toggle + palette picker + a Recap affordance into the `Footer` slot. Inject the EXISTING switcher template HTML via `g.Raw` (recommended) — `domainSidebar()` already runs in `internal/web`, which holds `h.tmpl` and `h.homeData()`.
- `web/templates/home.html` — keep the `head_switcher` / `model_switcher` / `chat_bar` fragments (they ARE the switchers); repoint their `/focus/settings?...` hrefs to the deterministic door `/ui/show/settings` (which 088 + the registry provide) so they survive 089's `/focus` retirement. Do NOT rename `#head-switcher` / `#chatbar` / `#model-modal`.
- `internal/web/assets/static/basm.css` — append (at END): rail-scoped layout rules so the switcher sections sit cleanly in the wood Brand/Footer (no parchment tokens); a `--measure` cap on artifact long-form text; a body-level off-canvas rail drawer for the product `.app-shell` at ≤720px (burger + backdrop) with its transition added to the `prefers-reduced-motion` block; ≥44px min-height on persistent rail controls.
- `internal/web/assets/static/basm.js` — ONLY if `ChatShell` needs a product burger/backdrop that `basmToggleNav` does not already cover; reuse `basmToggleNav` if the classes match (preferred — no JS change).
- `internal/feature/storybook/stories_navigation.go` (the sidebar Story added by 088) — show the enriched chrome states (head switcher in Brand, model switcher + theme + palette in Footer); update the dock Story's `Switchers` prop note (`stories_chat.go:178`) to reflect that switchers now mount.
- Tests: extend the 088 sidebar/home tests to assert the rail carries `#head-switcher`, an element with id `chatbar`, the `theme-toggle`, and the palette buttons; assert the switcher patch-target ids are unchanged.

**Out of scope (explicit — do NOT do here):**
- Porting the switcher `html/template` fragments to gomponents organisms (`chat.HeadSwitcher`/`chat.ModelSwitcher`) — that port was DEFERRED by plan 084 and is a separate refactor; reuse the templates via `g.Raw`. (DEFERRED — record in Maintenance.)
- Any change to the deterministic `/ui/show` endpoint, the `card_show` NL path, the artifact markers, or clusters (plan 090).
- Retiring `shell.Topbar` / `/focus` / boards / dropping a collection (plan 089).
- Any new migration; any change to the SSE selector IDs (`#chat`, `#dock-convo`, `#chat-draft`, `#nudge-poll`, `#recap`, `#model-modal`, `#head-switcher`, `#chatbar`).
- A true "new conversation" / conversation-switcher (no multi-thread plumbing exists for the master thread — DEFERRED).
- A full keyboard focus-trap on the rail drawer matching the topbar drawer's (plan 078). The product rail items INJECT (they do not navigate), so the drawer auto-closes on item click via the existing `basmToggleNav` handler; a basic accessible drawer (burger `aria-expanded`, backdrop click-to-close, `prefers-reduced-motion`) is in scope, the full Tab-trap/Escape/inert machinery is DEFERRED to a follow-up unless the executor finds it trivially reusable.

## Git workflow
- Branch `improve/091-sidebar-chrome-polish` off `main` (stacked on 088's branch if 088 is unmerged).
- Conventional commits, e.g.:
  - `feat(web): mount head + model switchers in the single-page sidebar rail`
  - `feat(web): theme toggle + palette picker + recap affordance in the rail footer`
  - `fix(web): repoint switcher manage links to the deterministic settings door`
  - `feat(css): rail switcher chrome, off-canvas drawer ≤720px, --measure artifact cap`
  - `docs(storybook): sidebar story shows enriched companion chrome`
  - `test(web): rail carries the switcher patch-target ids unchanged`
- Do NOT push or open a PR unless explicitly told.

## Steps

### Step 1: Decide the injection strategy (read both, pick A)
The switchers are templates (Pin 1). Two ways to get them into the shell-generic sidebar:
- **A (RECOMMENDED — KISS, no porting)**: in `internal/web` (which owns `h.tmpl` + `h.homeData()`), render the existing `head_switcher` and `chat_bar`/`model_switcher` templates to strings, and inject them as `g.Raw(...)` nodes into `shell.SidebarProps.Brand` / `.Footer`. `shell` never sees a feature type — it gets opaque `g.Node`s. This reuses the live SSE-patch targets (`#head-switcher`, `#chatbar`) verbatim, so `setActiveHead`/`patchChatbar` keep working with ZERO handler changes.
- **B (rejected here)**: port `head_switcher`/`model_switcher` to `chat.HeadSwitcher`/`chat.ModelSwitcher` gomponents organisms. This is the 084-deferred refactor; it touches `setActiveHead`/`patchChatbar` to render the organism instead of the template, risks the patch-target contract, and is more code. Out of scope (Maintenance note records it as the proper follow-up).

Proceed with **A**. **Verify**: no code yet — just confirm `grep -rn 'func HeadSwitcher\|func ModelSwitcher' internal/` is EMPTY (organisms truly don't exist). If they DO exist (a prior plan ported them), STOP and re-read — this plan's premise changed.

### Step 2: Inject the head switcher into the Brand slot and the model switcher into the Footer
In `internal/web` (`home.go` or a new `sidebar.go`), give `domainSidebar()` access to the rendered switcher HTML. Because `domainSidebar()` is called from `homePage` which already has `dock` data via `h.dockData()`, the cleanest seam is to make `domainSidebar` a METHOD on `*handlers` (or pass it the rendered fragments). Render the fragments with `h.homeData()`:
```go
// railBrand renders the head-identity + head switcher for the rail Brand slot.
// It reuses the existing head_switcher template (#head-switcher patch target),
// injected as g.Raw so internal/ui/shell stays feature-agnostic.
func (h *handlers) railSwitchersHTML(data homeData) (brand, foot g.Node, err error) {
	var hs strings.Builder
	if err = h.tmpl.ExecuteTemplate(&hs, "head_switcher", data); err != nil {
		return nil, nil, err
	}
	var cb strings.Builder
	if err = h.tmpl.ExecuteTemplate(&cb, "chat_bar", data); err != nil { // wraps model_switcher (+head, see note)
		return nil, nil, err
	}
	return g.Raw(hs.String()), g.Raw(cb.String()), nil
}
```
- **Note on `chat_bar`**: it currently wraps BOTH `head_switcher` AND `model_switcher` (`home.html:14-15`). For the rail you want head identity in Brand and model in Footer — so EITHER (a) split: put `head_switcher` in Brand and a NEW `model_bar` fragment carrying only `model_switcher` (keeping `id="chatbar"`) in Footer; OR (b) keep `chat_bar` whole in the Footer and put a lighter head-identity (`.ActiveHeadName` + avatar, no list) in Brand. **Pick (a)** so the head picker is prominent in the Brand and the model row sits with the theme controls in the Footer. The wrapper carrying `id="chatbar"` MUST remain (Pin 2) — so the new `model_bar` fragment is `<div class="chatbar chatbar-slim" id="chatbar">{{template "model_switcher" .}}</div>` (the 2s not-ready poll stays on it). `head_switcher` already has `id="head-switcher"` — keep it.
- Repoint the `/focus/settings?section=models` href in `model_switcher` (`home.html:24,29`) and the profile href (`:37`) to `/ui/show/settings` (the deterministic door 088 ships), with a `data-on:click__prevent="@get('/ui/show/settings')"` mirroring `SidebarItem.Action`, keeping the `href` as the no-JS fallback. After 089 the `/focus/*` routes are gone, so this keeps the links live. (If 089 has NOT landed, `/ui/show/settings` still works AND `/focus` still works — either way correct.)

Then in `domainSidebar` (now a method, or fed the nodes), set:
```go
	brand, modelFoot, err := h.railSwitchersHTML(data)
	// Brand = crest + head switcher; Footer = model_bar + theme toggle + palette + recap affordance.
	props.Brand = g.Group([]g.Node{crestNode(), brand})
	props.Footer = g.Group([]g.Node{modelFoot, railFooterControls(data.HasRecap)})
```
**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `grep -n 'WithSelectorID("head-switcher")\|WithSelectorID("chatbar")' internal/web/*.go` still returns the unchanged `heads.go`/`models.go` lines (you did NOT touch the handlers).

### Step 3: Theme toggle + palette picker + recap affordance in the Footer
Add a `railFooterControls(hasRecap bool) g.Node` in `internal/web` mirroring the storybook footer (`storybook.go:97-110`) — REUSE `paletteBtn` (it is already in package `web`, `storybook.go:32`):
```go
func railFooterControls(hasRecap bool) g.Node {
	kids := []g.Node{
		hh.Div(hh.Class("sb-foot-row"),
			hh.Span(hh.Class("sb-foot-label"), g.Text("Theme")),
			hh.Button(hh.Class("theme-toggle sb-foot-mode"), hh.Type("button"),
				g.Attr("onclick", "basmToggleTheme()"),
				hh.Title("Toggle day / night"), hh.Aria("label", "Toggle light/dark mode"),
				hh.Aria("pressed", "false"), g.Text("◑")),
		),
		hh.Div(hh.Class("sb-foot-themes"),
			paletteBtn("hearthwood", "Hearth"),
			paletteBtn("forest", "Forest"),
			paletteBtn("dungeon", "Dungeon"),
		),
	}
	if hasRecap {
		// Recap AFFORDANCE — scrolls the chat to the recap telescope (#recap stays
		// in the dock; this is a jump-link, not a duplicate sentinel).
		kids = append(kids, hh.A(hh.Class("sb-foot-recap"), hh.Href("#recap"),
			hh.Title("Earlier conversations"), g.Text("◇ Recap")))
	}
	return g.Group(kids)
}
```
- `aria-pressed="false"` is a safe initial value; `basmUpdateThemeButtons` (`basm.js:24`) overwrites it on load to reflect the real light/dark state (it already targets `.theme-toggle`). The palette buttons are a 3-state control, so they get NO `aria-pressed` (plan 079's rule — `aria-pressed` is boolean-only); `basmUpdatePaletteButtons` (`basm.js:52`) toggles their `is-active` class.
- The Recap affordance: `#recap` is the dock's telescope sentinel (`dock.go:92`), which lazy-loads bands on intersect. A `<a href="#recap">` scrolls it into view (triggering the intersect load) without duplicating the sentinel. It renders only when `HasRecap` (older history exists). This is the minimal honest "recap access" — a fuller recap drawer is DEFERRED.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `grep -rn 'aria-pressed\|Aria("pressed"' internal/` → the new rail toggle + the 079 occurrences only (no stray ones, no `aria-pressed` on palette buttons).

### Step 4: CSS — rail chrome, --measure cap, off-canvas drawer, 44px targets (append at END)
Append to the END of `internal/web/assets/static/basm.css` under a new Section banner. ALL token-based, square corners, single-dash classes, wood-safe tokens on the rail:
1. **Rail switcher fit** — the `head_switcher`/`model_switcher` were styled for the dock; in the wood rail Brand/Footer they need only small overrides (the templates already use `--chrome-fg`). Scope new rules under `.sb-brand` / `.sb-foot`, e.g. `.sb-foot .model-switcher { margin: 0; }`, `.sb-brand .head-switcher { padding: 0; }`. Do NOT restyle the switchers globally (the dock variant may still mount them elsewhere). Verify they read on wood in both modes (Pin 3 — `--chrome-fg`/`--gold`, never `--ink`/`--smoke`/`--muted`).
2. **`--measure` artifact cap** — long-form prose inside an injected artifact card must not exceed the reading-width cap. Add a rule capping prose within the chat artifact card body, e.g. `#chat .k-inline .card-prose, #chat .k-inline p { max-width: var(--measure); }` (confirm the actual prose element class used by the cards via `grep`; use the real class, do NOT invent). This satisfies the ui-development "apply --measure to long-form text" requirement.
3. **Off-canvas rail drawer ≤720px** — the product `.app-shell` (088) is a `[sidebar | chat]` grid; on a phone the rail must go off-canvas. Mirror the proven storybook drawer (`basm.css:3092-3097`) but scoped to the product shell. If 088's `ChatShell` already renders `.sb-side` (it does — `shell.Sidebar` ⇒ `.sb-side`), reuse `basmToggleNav` (`basm.js:194`) and add at body level: a `.sb-burger` (in a small top affordance, since `ChatShell` has no `.sb-topbar`) and a `.sb-backdrop` (Step 4a). CSS:
   ```css
   @media (max-width: 720px) {
     .app-shell { grid-template-columns: 1fr; }
     .app-shell .sb-side {
       position: fixed; inset: 0 auto 0 0; width: min(86vw, 322px); z-index: var(--z-drawer);
       transform: translateX(-104%); transition: transform .2s var(--ease-crisp); box-shadow: 6px 0 0 rgba(0, 0, 0, .4);
     }
     .app-shell .sb-side.is-open { transform: none; }
     .app-shell .sb-backdrop.is-open { display: block; position: fixed; inset: 0; z-index: var(--z-scrim); background: rgba(8, 5, 2, .62); }
     .app-burger { display: inline-flex; ... min-height: 44px; min-width: 44px; ... }
   }
   ```
   And add the transition to the reduced-motion block: edit the existing `@media (prefers-reduced-motion: reduce)` at `:972-984` (or the rail one at `:3101`) to include `.app-shell .sb-side { transition: none; }`. **The drawer must be at BODY level / fixed (not inside an `overflow:hidden`/`transform` ancestor) so it escapes the grid's stacking context** — `position:fixed` against the viewport achieves this; confirm `.app-shell` does NOT set `transform`/`filter`/`contain` (which would re-root the fixed element). This is the plan-078 stacking-context lesson.
4. **44px touch targets** — persistent rail controls (theme toggle, palette buttons, head-switcher choices, the burger) get `min-height: 44px` (and `min-width: 44px` for icon-only buttons) at `≤720px`, expanding the hit area without enlarging the visual chrome (mirror plan 078 `:217`). The `head-switcher-choice` pills (`basm.css:1732`) are `~.15rem` padding — bump their touch target in the `≤720px` block only.

**Verify**: `git diff --check` → no output; `grep -n -- '--ink\|--smoke\|--muted' <the new rail CSS lines>` → none (eyeball the appended block); build + load `/` (Test plan).

### Step 4a: Product shell burger + backdrop (CSS/markup only if needed)
088's `ChatShell` renders `<html class="app">` + `.app-shell` (sidebar + dock) with NO `.sb-topbar`/`.sb-burger`/`.sb-backdrop`. For the phone drawer to open, the shell needs a burger + backdrop. Two options:
- **A (preferred, no shell edit)**: render the burger and backdrop as part of the DOMAIN sidebar in `internal/web` (they are product chrome, not generic shell) — e.g. prepend a `.app-burger` button (`onclick="basmToggleNav()"`, `aria-label="Open navigation"`, `aria-expanded="false"`) and append a `.sb-backdrop` (`onclick="basmToggleNav()"`) as siblings inside the `domainSidebar`-built nodes that `ChatShell` places. BUT `basmToggleNav` toggles `.sb-side`/`.sb-backdrop` globally, so the classes must match: the rail is `.sb-side` (good), and you add `.sb-backdrop`. The burger lives ≤720px only (hidden via CSS otherwise).
- **B (minimal shell edit)**: if `ChatShell` cannot accept a burger/backdrop without an edit, add them to `ChatShell` in `internal/ui/shell` guarded by a prop, OR (simplest) emit them unconditionally in `ChatShell` (they are display:none ≥720px). Keep `shell` generic — the burger calls `basmToggleNav()` (already global). A one-line additive shell change is acceptable if Option A proves awkward; do NOT restructure `ChatShell`.

Choose A if `ChatShell` accepts arbitrary sidebar nodes; B otherwise. Either way reuse `basmToggleNav` — NO new JS unless the classes genuinely differ. **Verify**: `go test ./internal/ui/shell/...` (if you touched `ChatShell`); load `/` at 390px width → burger opens the rail, backdrop closes it, clicking a domain item injects + auto-closes.

### Step 5: Storybook — show the enriched chrome
Per the storybook-source-of-truth workflow:
1. In `internal/feature/storybook/stories_navigation.go` (the `sidebarStory()` 088 added — confirm it exists; if 088 named it differently, find it via `grep -n 'Group: "Navigation"' internal/feature/storybook/`), add Variants showing the enriched chrome: a Brand with a head-identity + head-switcher fixture, and a Footer with the model row + theme toggle + palette buttons + a Recap affordance. Because the switchers are templates (not gomponents), render representative STATIC markup in the story fixture (the story shows the visual state; the live wiring is the templates). Do NOT import `internal/web` into the storybook (layering) — hand-build a small fixture `g.Node` that LOOKS like the rail footer (reuse `shell.Sidebar` with fixture Brand/Footer nodes).
2. Update the dock Story prop note (`stories_chat.go:178`): change "still a template fragment — deferred from this plan" to reflect that the switchers now mount in the **sidebar rail** (Brand/Footer), not the dock — e.g. `"The chatbar/head-switcher node. On the single-page shell the switchers live in the sidebar rail (plan 091); the dock Switchers slot is unused on home."`.

**Verify**: `go test ./internal/feature/storybook/...` → `TestAllStoriesRender` + `TestStoriesUniqueAndLookup` pass.

### Step 6: Tests
1. **`internal/web/home_test.go`** (or the 088-added home assertions): extend to assert the rendered home page now contains: `id="head-switcher"` (head switcher in Brand), an element with `id="chatbar"` (model switcher wrapper in Footer — Pin 2), `class="theme-toggle"`, the palette buttons (`data-theme="hearthwood"`/`forest`/`dungeon`), and — when the fixture has recap — the `sb-foot-recap` affordance. Confirm the page still has NO `.topbar` (088's invariant) and the dock selector IDs still pass (`TestHomeDockSelectorIDs`).
2. **Patch-target regression**: assert (a unit assertion on the rendered home, or a comment-backed grep test) that `#head-switcher` and `#chatbar` exist on the page so `setActiveHead`/`patchChatbar` find their targets. The existing `setActiveHead`/`patchChatbar` handler tests (if any) must still pass unchanged.
3. **Switcher manage-link repoint**: a string assertion that `model_switcher` no longer references `/focus/settings` (so it survives 089) — e.g. the rendered home does NOT contain `href="/focus/settings` and DOES contain `/ui/show/settings`. (Skip this assertion if 089 has not landed and you chose to keep `/focus` as a fallback — but the repoint is preferred regardless.)

**Verify**: `go test ./internal/web/... ./internal/ui/shell/...` → all pass.

### Step 7: Full verification + index
Run the Done-criteria gate. Update the 091 row in `plans/readme.md`.

## Test plan
- **Web render**: the home page (`GET /`) carries the head switcher (`#head-switcher`) in the rail Brand, the model switcher (in an `id="chatbar"` wrapper) + theme toggle + palette buttons in the rail Footer, and the Recap affordance when recap exists. NO topbar. The dock SSE selector IDs are intact (`TestHomeDockSelectorIDs`).
- **Patch-target contract**: `setActiveHead` (`POST /ui/heads/active`) still patches `#head-switcher` and the head choice flips active; `patchChatbar` (the 2s `/ui/chatbar` poll while no model) still patches `#chatbar` and stops once ready. These pass UNCHANGED (you only moved the elements' location, not their ids).
- **a11y**: the rail theme toggle reflects `aria-pressed` (synced by `basmUpdateThemeButtons` on load and toggle); palette buttons carry NO `aria-pressed` (3-state); the burger carries `aria-expanded` synced by `basmToggleNav`.
- **Responsive (manual, browser — `run`/`verify`/Playwright)**: at ≥1024px the rail is the persistent left column with switchers + footer chrome; at 390px the rail is off-canvas, the burger opens it (a body-level fixed drawer above the chat), the backdrop closes it, clicking a domain item injects a card AND auto-closes the drawer. With `prefers-reduced-motion`, the drawer has no slide transition.
- **Legibility (manual, both modes)**: toggle light/dark AND each palette — the rail head/model switchers, theme toggle, palette buttons, and recap link all stay legible on the wood rail (they use `--chrome-fg`/`--gold`). The artifact prose in `#chat` is capped at `--measure`.
- **Storybook**: `go test ./internal/feature/storybook/...` (`TestAllStoriesRender`).

## Done criteria
- [ ] `CGO_ENABLED=0 go build ./...` → exit 0; `go vet ./...` → exit 0; `gofmt -l internal/` → no output; `git diff --check` → no output.
- [ ] `go test ./...` → all pass, including the extended home/sidebar assertions and the unchanged switcher-handler tests.
- [ ] The single-page home rail carries the head switcher (`#head-switcher`, Brand), the model switcher (inside an `id="chatbar"` wrapper, Footer), the `theme-toggle`, the three `paletteBtn`s, and a Recap affordance when recap exists.
- [ ] The switcher SSE patch-target ids (`#head-switcher`, `#chatbar`, `#model-modal`) are UNCHANGED; `setActiveHead` + `patchChatbar` patch them with no handler edits (`grep` confirms the `WithSelectorID(...)` lines are untouched).
- [ ] The switcher "Manage models" / profile links no longer point at `/focus/settings` (repointed to `/ui/show/settings`), so they survive plan 089's `/focus` retirement.
- [ ] No NEW rail CSS uses parchment tokens (`--ink`/`--smoke`/`--muted`); the rail uses wood-safe `--chrome-fg`/`--gold`, verified legible in light AND dark across all three palettes.
- [ ] Artifact long-form prose in `#chat` is capped at `--measure`.
- [ ] The product `.app-shell` rail goes off-canvas at ≤720px as a body-level fixed drawer (burger + backdrop), reusing `basmToggleNav`; its transition is inside the `prefers-reduced-motion` block; persistent rail controls are ≥44px touch targets at ≤720px.
- [ ] The theme toggle carries `aria-pressed` (synced by basm.js); palette buttons carry none; the burger carries `aria-expanded`.
- [ ] The sidebar Story shows the enriched Brand/Footer chrome; the dock Story's `Switchers` prop note is updated; `TestAllStoriesRender` passes.
- [ ] No new migration; no SSE selector-ID changes.
- [ ] `plans/readme.md` 091 row updated.

## STOP conditions (specific to this plan)
- **088 has not landed** (`shell.ChatShell` / `domainSidebar` / `.app-shell` / `/ui/show` absent) — STOP; there is nothing to enrich.
- **Drift**: a cited file or the `gomponents-datastar`/`datastar-go` go.mod pin changed since `3136bad` and an excerpt no longer matches — STOP and report.
- **You are about to rename `#head-switcher`, `#chatbar`, or `#model-modal`** — STOP. Those are the live SSE patch targets (Pin 2); moving the element is fine, renaming breaks `setActiveHead`/`patchChatbar`/the modal opener.
- **You are about to port the switcher templates to gomponents organisms** — STOP. That refactor was deferred by plan 084 and is OUT of scope here; inject the existing template HTML via `g.Raw` (Pin 1, Step 1A).
- **A new rail rule uses a parchment token** (`--ink`/`--smoke`/`--muted`) on the wood rail — STOP; it will go invisible in one theme (the A2 contrast bug 079 fixed). Use `--chrome-fg`/`--gold`.
- **The off-canvas drawer renders behind the chat / inside a clipped ancestor** — STOP; the drawer must be `position:fixed` at body level (no `transform`/`filter`/`contain` on `.app-shell`), per plan 078's stacking-context lesson.
- **You added an animation without a `prefers-reduced-motion` guard** — STOP; add its selector to the existing reduced-motion `@media` block.
- **A Verify fails twice** after a fix attempt — STOP and report the command + output.

## Maintenance notes
- **Switcher organisms are still deferred**: the proper end-state is `chat.HeadSwitcher` / `chat.ModelSwitcher` gomponents organisms (deferred from plan 084), with `setActiveHead`/`patchChatbar` rendering the organism and the storybook holding a real Story. This plan deliberately reuses the `html/template` fragments via `g.Raw` to avoid that refactor's risk to the patch-target contract. When the port happens: keep `#head-switcher`/`#chatbar` ids, render the organism in the handlers, and replace the `g.Raw` injection here with the organism node. Until then, `web/templates/home.html` is load-bearing — do not delete it.
- **Recap is an affordance, not a relocation**: `#recap` (the telescope sentinel, `dock.go:92`) stays in the dock and lazy-loads on intersect. The rail's `◇ Recap` is a jump-link to it. A fuller "recap drawer in the rail" is DEFERRED — record a follow-up if the owner wants recap fully out of the scroll.
- **Two drawer mechanisms coexist by design**: `basmToggleNav` (storybook + now the product rail) and `basmToggleTopnav` (the old topbar drawer, plan 078 — retired by 089). After 089, `basmToggleTopnav` and `#topnav-drawer` may be dead; if 089 leaves them, note it. This plan only uses `basmToggleNav`.
- **`aria-pressed` discipline**: binary toggles only (the theme toggle). The 3-state palette cycle uses `is-active` class state, never `aria-pressed` (plan 079's rule). Do not regress this.
- **Scope discipline**: this is the LAST plan of the program. Resist adding a settings-in-rail panel, a notifications bell, or per-head theming — those are new features, not this plan's polish remit. Record any such idea as a follow-up, do not build it here.
