# Plan 098 — Right-panel canvas: artifacts open in a single-active panel, chips in chat

- **Status:** TODO
- **Priority:** P1
- **Effort:** L
- **Risk:** MED–HIGH (reverses two recently-merged locked decisions: 088's
  "chat is the only primary surface" and 097's inline artifact frame; touches
  the live SSE path, the reload path, layout, and three docs)
- **Planned against commit:** `a1955f8`
- **Date:** 2026-06-18
- **Depends on:** — (foundation for 099, 100; supersedes the premise of 091)

> **Executor: read this whole file before touching anything.** You have zero
> context from the design conversation that produced it. Everything you need is
> here. Run the drift check (Step 0) first. Honor the STOP conditions. Commit
> per step. This plan reverses parts of plans 088 and 097 **on purpose** — that
> is the owner's explicit, confirmed decision, not a mistake; do not "preserve"
> the inline-artifact behavior out of caution.

---

## Why this change

Today every "artifact" (a domain card like Quests/Life/Settings, summoned
either by an owner clicking the left rail or by the agent's `card_show`) is
**appended inline into the chat transcript** (`#chat`). The transcript is the
only surface; artifacts stack in it, capped at 3 active (plan 094), each framed
as a titled "sub-window" (plan 097).

The owner has decided this stacks badly and clutters the conversation. The new
model — confirmed via an explicit decision, and the well-validated "canvas"
pattern (think ChatGPT canvas / Claude artifacts panel) — is:

- **A dedicated right panel holds ONE active artifact at a time.** Summoning a
  new one replaces it (single-active is automatic with an inner/morph swap — no
  cap bookkeeping needed).
- **The chat keeps the conversation plus a compact "re-open chip"** for each
  artifact that was summoned. The chip is the durable transcript trace; clicking
  it re-opens that artifact in the panel. This preserves plan 088's
  "an artifact is a permanent, auditable transcript entry" invariant (the
  persisted `role=tool` row stays; it just renders as a chip, not the full card)
  while removing the clutter.
- **On reload, the panel restores the last-active artifact** (one tiny
  `owner_settings` pointer — no schema migration).

This is fully achievable with server-side rendering + Datastar SSE (no SPA, no
client router, no Node) — the existing left rail already fires Datastar `@get`
actions, the existing door already returns SSE patches, and Datastar can patch
multiple regions in one response (used today in `models.go`).

### What this reverses, on purpose

- **Plan 088** locked "retire the right-rail dock; chat is the only primary
  surface." This plan re-introduces a right region — but as a content panel, not
  the old fixed chat dock. That is the owner's call.
- **Plan 097** framed each *inline* artifact as a titled sub-window
  (`chat.Artifact`). Inline artifacts go away; 097's visual design (the
  `.artifact-head` bar + bordered body) **moves into the panel chrome**
  (`chat.Panel`), so the work is relocated, not wasted.
- **Plan 094** capped *inline* artifacts at 3. With one active in the panel, the
  cap is moot and is **removed** (server `capArtifacts`, client
  `balaurCapArtifacts`, the `.artifact--collapsed` CSS).

### Scope boundary inside the program

This plan ships the **panel + both summon doors (owner click AND the agent's
live `card_show`/`show_cards`) + reload + restore + cap retirement**. It does
NOT collapse the rail's grouped sub-entries into in-panel tabs — that is plan
**099** ("apply the same treatment to the other sidebar entries"). The rail
stays exactly as it is today; only *where its clicks render* changes.

---

## Step 0 — Drift check (do this first)

```sh
git rev-parse --short HEAD                 # expect a1955f8 (or rebase this plan)
git grep -n "html.app .app-shell" internal/web/assets/static/basm.css   # the grid rule (~line 3342)
git grep -n "WithSelectorID(\"chat\"), datastar.WithModeAppend()" internal/web/show.go
git grep -n "activeArtifactCap = 3" internal/web/cards.go
git grep -n "func balaurCapArtifacts" internal/web/assets/static/basm.js
```

If `html.app .app-shell` is the 2-column grid (`grid-template-columns: 274px
1fr`, multi-line, ~lines 3342–3347), `show.go` still appends to `#chat`, and the
cap still exists, the tree matches this plan. **If any is already gone or
changed, STOP and report** — someone has moved this area and the excerpts below
may be stale.

> Note: `grep "grid-template-columns: 274px 1fr"` returns TWO hits — line ~2594
> (`.sb-root`, the storybook shell — a DECOY, do not touch) and line ~3344
> (`html.app .app-shell`, the real target). Always scope edits to the
> `html.app .app-shell` rule. The grid rule is multi-line and preceded by a
> plan-088 comment block (~3335–3341) that still says "two-column grid" / "chat
> IS the only primary surface" — you will update that comment in Step 2.

Verification baseline at `a1955f8`: `gofmt -l` clean, `go vet ./...` ok,
`go test ./...` ok, `CGO_ENABLED=0 go build ./...` ok. Re-run these now to
confirm a green starting point before you change anything.

> Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need the
> GOPROXY shim — see `docs/hyperagent-sandbox.md`.

---

## The target architecture (read once, then follow the steps)

```
┌─ rail (274px) ─┬──── chat (1fr) ─────┬──── panel (--w-panel) ──┐
│ Quests         │ you: show my quests │ ┌─ ⚔ QUEST LOG    ✕ ──┐ │
│ Life           │ ai: here you go ↓   │ │  ▸ Ship 1.0         │ │  #panel
│ Knowledge…     │ [⚔ Quest log  open ▸]   │  ▸ Fix the bug      │ │   └ #panel-inner (chat.Panel)
│ Skills         │      ↑ re-open chip │ │  ▸ Write the doc    │ │       ├ .panel-head (icon+title+✕)
│ Settings       │                     │ └─────────────────────┘ │       └ #panel-body (the artifact)
└────────────────┴─────────────────────┴─────────────────────────┘
```

- **One door, two callers, same destination.** A rail click hits
  `GET /ui/show/{type}` (`show.go`); the agent's `card_show`/`show_cards` flow
  through `chatstream.go`'s `endTool`. Both currently append the full card to
  `#chat`. After this plan both **morph the panel** with the card and **append a
  chip** to `#chat`.
- **Single-active is free.** Morphing `#panel-inner` by its root id (the
  existing `morphNode` idiom — selector-less `PatchElements`, the node carries
  `id="panel-inner"`) replaces whatever was there. No cap, no "active" flag.
- **Re-open is by re-derivation, not by stored id.** A single card's chip
  re-opens via the same URL that summoned it: `@get('/ui/show/{type}?{query}')`.
  No new endpoint, no message-id plumbing. **Clusters** (`show_cards`, multi-card,
  agent-only — plan 090) have no such URL, so a cluster chip is a **non-clickable
  label** (documented limitation; clusters are rare and agent-driven).
- **Restore = one setting.** The owner door writes
  `owner_settings["panel_active"] = "/ui/show/{type}?{query}"`. `homePage` reads
  it and renders that artifact into the panel on load. Empty/missing → an empty
  placeholder.
- **Mobile (≤720px).** No room for a third column (the rail is already
  `display:none` at 720px). The panel becomes a fixed overlay drawer that
  auto-opens when content is summoned and closes via the ✕ / Escape / backdrop.
  Keep this **minimal** here; full a11y parity + the rail's own mobile reveal is
  plan 100. (Reuse the existing accessible drawer scaffolding pattern from the
  product topnav drawer in `basm.js`, lines ~226–271.)

---

## Current state (real excerpts at `a1955f8`)

### `internal/ui/shell/chatshell.go` — the 2-column shell (no panel today)

```go
func ChatShell(p ChatShellProps) g.Node {
	return g.Group([]g.Node{
		g.Raw("<!doctype html>"),
		h.HTML(
			h.Lang("en"), h.Class("app"),
			h.Head(pageHead(), h.TitleEl(g.Text(p.Title+" · Balaur"))),
			h.Body(
				h.A(h.Class("skip-link"), h.Href("#chat"), g.Text("Skip to content")),
				h.Div(h.Class("app-shell"),
					p.Sidebar,
					h.Aside(h.ID("dock"), h.Class("app-dock"), p.Dock),
				),
			),
		),
	})
}
```
`ChatShellProps` is `{ Title string; Sidebar g.Node; Dock g.Node }`.

### `internal/web/show.go` — the owner door (appends to `#chat`)

```go
func (h *handlers) uiShow(e *core.RequestEvent) error {
	typ := e.Request.PathValue("type")
	spec, ok := cards.Get(typ)
	if !ok { return e.NotFoundError("no such card type", nil) }
	params, err := cards.Validate(typ, queryToMap(e.Request.URL.Query()))
	if err != nil { return e.BadRequestError("invalid card params: "+err.Error(), err) }

	marker := tools.MarkUICard(typ, params, "showing the owner the "+spec.Label+" card")
	master, err := conversation.Master(h.app)
	if err != nil { return e.InternalServerError("resolving master conversation", err) }
	rec, err := conversation.AppendOriginRec(h.app, master.Id,
		llm.Message{Role: "tool", Content: marker}, typ, "")
	if err != nil { return e.InternalServerError("persisting artifact", err) }

	body := h.renderMessages(h.messageViews([]*core.Record{rec}))
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(string(body),
		datastar.WithSelectorID("chat"), datastar.WithModeAppend())
	return nil
}
```

### `internal/web/chatstream.go` — the agent live path (`endTool`, appends to `#chat`)

```go
func (s *chatStream) handleToolResult(ev agent.Event) {
	if typ, query, rest, ok := tools.ParseUICard(ev.Text); ok {
		if spec, ok := cards.Get(typ); ok {
			s.endTool(rest, s.h.uicardBody(typ, query), spec.Label, spec.Icon)
		} else {
			s.endTool(rest, s.h.uicardBody(typ, query), typ, "")
		}
		return
	}
	// … choices / proposal / refresh …
	if title, cs, rest, ok := tools.ParseArtifact(ev.Text); ok {
		s.endTool(rest, s.h.artifactBody(title, cs), title, "")
		return
	}
	s.endTool(clipText(ev.Text, 2000), "", "", "")
}

func (s *chatStream) endTool(content string, card template.HTML, artTitle, artIcon string) {
	s.morphNode(chat.ToolRow(chat.ToolRowProps{
		Tool: s.toolName, Icon: toolIconFile(s.toolName), ID: s.toolID, BodyID: s.toolBody, Content: content,
	}))
	if card == "" { return }
	if artTitle == "" { // proposal etc. — keep the plain inline card
		s.appendNode(g.El("div", g.Attr("class", "k-inline"), g.Attr("id", s.toolID+"-card"), g.Raw(string(card))))
		return
	}
	s.appendNode(artifactWrap(artTitle, artIcon, false, s.toolID+"-card", card))
}
```
Note `handleToolResult` has `typ`+`query` for the uicard case but only passes
`label`+`icon`+`card` into `endTool`. You will thread `typ`+`query` through so
the chip can re-derive its URL. Proposals/choices/refresh keep the existing
`endTool` inline behavior **unchanged**.

### `internal/web/recap.go` — the reload path + the cap

```go
// renderMessages, tool branch:
case "tool":
	nodes = append(nodes, chat.ToolRow(chat.ToolRowProps{
		Tool: mv.Tool, Icon: toolIconFile(mv.Tool), Content: mv.Content,
	}))
	if mv.CardBody != "" {
		if mv.ArtifactTitle != "" {
			nodes = append(nodes, artifactWrap(mv.ArtifactTitle, mv.ArtifactIcon, mv.ArtifactCollapsed, "", mv.CardBody))
		} else {
			nodes = append(nodes, g.El("div", g.Attr("class", "k-inline"), g.Raw(string(mv.CardBody))))
		}
	}

// messageViews, tool branch (uicard / cluster cases set the artifact fields):
if typ, query, rest, ok := tools.ParseUICard(mv.Content); ok {
	mv.CardBody = h.uicardBody(typ, query)
	mv.Content = rest
	if spec, ok := cards.Get(typ); ok {
		mv.ArtifactTitle, mv.ArtifactIcon = spec.Label, spec.Icon
	}
} else if /* choices */ … } else if kind, id, rest, ok := tools.ParseProposal(mv.Content); ok {
	mv.CardBody, mv.Content = h.proposalBody(kind, id), rest
} else if /* refresh */ … } else if title, cs, rest, ok := tools.ParseArtifact(mv.Content); ok {
	mv.CardBody, mv.Content = h.artifactBody(title, cs), rest
	mv.ArtifactTitle = title
}
…
capArtifacts(out)   // ← remove this call

// capArtifacts marks all but the newest activeArtifactCap collapsed — REMOVE the func.
```

### `internal/web/cards.go` — `artifactWrap`, `activeArtifactCap`

```go
const activeArtifactCap = 3   // ← remove (plan 094, now moot)

func artifactWrap(title, icon string, collapsed bool, innerID string, body template.HTML) g.Node {
	return chat.Artifact(chat.ArtifactProps{Title: title, Icon: icon, Collapsed: collapsed, InnerID: innerID, Body: g.Raw(string(body))})
}   // ← remove (no more inline artifacts)
```
Keep `uicardBody`, `cardFocusHTML`, `artifactBody` — they render the panel body.

### `internal/web/assets/static/basm.js` — the client cap

```js
var ACTIVE_ARTIFACT_CAP = 3;
function balaurCapArtifacts() { /* toggles .artifact--collapsed on #chat .artifact */ }

document.addEventListener('DOMContentLoaded', () => {
  const chat = document.getElementById('chat');
  if (!chat) return;
  balaurScrollToLatest();
  balaurCapArtifacts();
  new MutationObserver(() => { balaurCapArtifacts(); balaurScrollToLatest(); })
    .observe(chat, { childList: true, subtree: true });
});
```
Remove `balaurCapArtifacts` and its two calls; **keep** `balaurScrollToLatest`
and the observer (it still keeps the latest chat line in view).

### `internal/web/assets/static/basm.css` — grid + inline-artifact CSS + tokens

```css
:root {
  --w-chat-home: 1800px;
  /* z-index tiers */ --z-base:1; --z-overlay:50; --z-scrim:55; --z-drawer:60;
}

/* Two-column grid: [rail | dock]. */
html.app .app-shell { display: grid; grid-template-columns: 274px 1fr; height: 100dvh; overflow: hidden; }

@media (max-width: 720px) {
  html.app .app-shell { grid-template-columns: 1fr; }
  html.app .app-shell .sb-side { display: none; }
}

/* Inline artifact frame (plan 097) — to be REMOVED (replaced by .panel chrome). */
.artifact { margin: 10px 0; border: 2px solid var(--parch-edge); background-color: var(--surface);
  background-image: var(--grain-ink); background-size: 4px 4px; box-shadow: var(--parch-bevel); }
.chat .artifact { margin-left: var(--chat-gutter, 124px); }
.artifact-head { display:flex; align-items:center; gap:var(--space-2); padding:var(--space-2) var(--space-3);
  background:var(--surface-2); border-bottom:2px solid var(--parch-edge); font-family:var(--font-mono);
  font-size:11px; font-weight:700; text-transform:uppercase; letter-spacing:.06em; color:var(--ink); }
.artifact-head-icon { width:16px; height:16px; image-rendering:pixelated; }
.artifact-body { padding: var(--space-4); }
.artifact-body .kcard { padding: 15px 18px; }
.artifact--collapsed > .artifact-body { display: none; }
.artifact--collapsed .artifact-head { color: var(--ink-muted); }
.artifact--collapsed .artifact-head-title::after { content: " · shown earlier"; font-weight:400; opacity:.8; }
```

### Persistence seam (no migration — `owner_settings` already exists)

```go
// internal/store/owner_settings.go
func GetOwnerSetting(app core.App, key, defaultVal string) string  // returns default on miss/empty
func SetOwnerSetting(app core.App, key, value string) error        // upsert
```

---

## Steps

### Step 1 — New `chat.Panel` and `chat.ArtifactChip` organisms (+ stories)

Create `internal/ui/chat/panel.go`. `chat.Panel` is the panel frame — it reuses
plan 097's `.artifact-head` visual language as `.panel-head`. Body is
pre-rendered by the web layer (same `g.Node` injection seam as `chat.Artifact`,
so this package imports no `feature/cards`).

```go
package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// PanelProps configures chat.Panel — the single-active right-panel frame.
// Empty (Title=="" && Body==nil) renders the placeholder ("nothing open").
// The root id is "panel-inner" so the gateway can morph it in place
// (selector-less PatchElements by root id) to swap the active artifact.
type PanelProps struct {
	Title string // artifact name in the head bar
	Icon  string // /static/icons stem ("" → no icon)
	Body  g.Node // pre-rendered artifact body (a Focus card or a Cluster)
}

// Panel frames the active artifact: a .panel-head bar (icon + title + a close
// control) atop the scrollable #panel-body. One panel is active at a time; the
// gateway replaces this node's content to switch artifacts.
func Panel(p PanelProps) g.Node {
	inner := []g.Node{h.ID("panel-inner")}
	if p.Title == "" && p.Body == nil {
		inner = append(inner, h.Div(h.Class("panel-empty"),
			g.Text("Pick a domain from the rail, or ask Balaur to show you something.")))
		return h.Div(inner...)
	}
	head := []g.Node{h.Class("panel-head")}
	if p.Icon != "" {
		head = append(head, h.Img(h.Class("panel-head-icon"),
			h.Src("/static/icons/"+p.Icon+".png"), h.Alt(""), g.Attr("decoding", "async")))
	}
	head = append(head,
		h.Span(h.Class("panel-head-title"), g.Text(p.Title)),
		// Close control: clears the panel (and the persisted pointer) via @get.
		h.Button(h.Class("panel-close"), h.Type("button"),
			g.Attr("data-on:click__prevent", "@get('/ui/panel/close')"),
			h.Aria("label", "Close panel"), g.Text("✕")),
	)
	inner = append(inner,
		h.Header(head...),
		h.Div(h.ID("panel-body"), h.Class("panel-body"), p.Body),
	)
	return h.Div(inner...)
}

// ArtifactChip is the durable transcript trace of a summoned artifact: a compact
// re-open affordance in #chat. ReopenURL set → clickable (@get re-summons into
// the panel; Href is the no-JS fallback). ReopenURL "" → a non-clickable label
// (clusters, which have no deterministic re-open URL).
type ArtifactChipProps struct {
	Title     string
	Icon      string
	ReopenURL string
}

func ArtifactChip(p ArtifactChipProps) g.Node {
	kids := []g.Node{h.Class("art-chip")}
	if p.Icon != "" {
		kids = append(kids, h.Img(h.Class("art-chip-icon"),
			h.Src("/static/icons/"+p.Icon+".png"), h.Alt(""), g.Attr("decoding", "async")))
	}
	kids = append(kids, h.Span(h.Class("art-chip-label"), g.Text(p.Title)))
	if p.ReopenURL == "" {
		kids = append(kids, h.Span(h.Class("art-chip-hint"), g.Text("shown earlier")))
		return h.Div(kids...) // non-clickable
	}
	kids = append(kids,
		h.Span(h.Class("art-chip-hint"), g.Text("open ▸")),
		h.Href(p.ReopenURL),
		g.Attr("data-on:click__prevent", "@get('"+p.ReopenURL+"')"),
	)
	return h.A(kids...)
}
```

> The `data-on:click__prevent` attribute string mirrors how `sidebarItem`
> (`internal/ui/shell/sidebar.go`) renders rail actions via
> `data.On("click", action, data.ModifierPrevent)`. Using the literal
> `g.Attr("data-on:click__prevent", …)` here is fine and keeps `chat` free of
> the `gomponents-datastar` import; if you prefer, import
> `data "maragu.dev/gomponents-datastar"` and use `data.On(...)` to match
> `sidebar.go` exactly. Either renders identically — pick one and be consistent.
>
> In gomponents, attributes (`h.Class`, `h.Href`, `g.Attr`) and element children
> (`h.Span`, `h.Img`) may appear in **any order** within the variadic — the mixed
> order in `ArtifactChip` (children, then `h.Href`/`g.Attr`) is intentional and
> renders correctly. Do not reorder it to "fix" it.
>
> The panel-close control's `@get('/ui/panel/close')` target is registered in
> Step 8; the storybook only renders the markup (it does not hit the endpoint),
> so the close button in the `chatpanelStory` is inert there until the app runs.

Add `internal/ui/chat/panel_test.go` mirroring the existing
`internal/ui/chat/artifact_test.go` (use the shared `render(t, …)` helper in
`internal/ui/chat/helpers_test.go`). Cover, at minimum:

- `chat.Panel` with title+icon+body → contains `panel-head`, `panel-head-title`,
  the title text, `id="panel-body"`, `panel-close`, the icon URL, the body text.
- `chat.Panel{}` (empty) → contains `panel-empty` and the placeholder copy; does
  NOT contain `panel-head`.
- `chat.ArtifactChip` with `ReopenURL` set → renders `<a`, `art-chip`,
  `data-on:click__prevent`, the URL, `open ▸`.
- `chat.ArtifactChip` with `ReopenURL=""` → non-clickable (`shown earlier`, no
  `data-on:click`).

Add stories in `internal/feature/storybook/stories_chat.go` (mirror
`chatartifactStory`, which you will remove in Step 9 — so net you are *replacing*
it): a `chatpanelStory` (variants: with-artifact / empty) and a
`chatartifactchipStory` (variants: clickable / non-clickable). `stories_chat.go`
already imports `taskcards` and `chat` (the existing `chatartifactStory` uses a
`taskcards.TaskCard` sample body) — **reuse those imports**, don't add new ones.
Register both in `internal/feature/storybook/story.go` in the story slice,
adjacent to the `chatartifactStory()` line (which you remove in Step 9).

**Verify:**
```sh
gofmt -l internal/ui/chat/panel.go internal/feature/storybook/stories_chat.go
go test ./internal/ui/chat/...
go build ./internal/feature/storybook/...
```
Expect: gofmt prints nothing; the new panel/chip tests pass.

### Step 2 — CSS: panel column + chrome + chip + mobile drawer; retire inline-artifact CSS

In `internal/web/assets/static/basm.css`:

1. Add a `--w-panel` token in `:root`. `--w-chat-home` is at line ~165, with
   `--w-chat-overlay` (166) and `--measure` (167) right after — add
   `--w-panel: 480px;` on the line after `--w-chat-overlay`. The z-index tiers
   (`--z-base`/`--z-sticky`/`--z-scrim`/`--z-drawer`) are a SEPARATE `:root`
   block (~183–189) and already exist — **do not re-declare them**, just use them.
2. Change the app grid to three columns. The real rule is multi-line at lines
   ~3342–3347 (preceded by a stale plan-088 comment at ~3335–3341). Make a
   **surgical** edit: change ONLY the `grid-template-columns` value line to
   `grid-template-columns: 274px 1fr var(--w-panel);` (keep the multi-line block
   format — do not collapse the rule to one line). Also update the comment above
   it from "two-column grid \[rail | dock]" / "chat IS the only primary surface"
   to "three-column grid \[rail | chat | panel]". The edited rule reads:
   ```css
   /* Three-column grid: [rail | chat | panel]. */
   html.app .app-shell {
     display: grid;
     grid-template-columns: 274px 1fr var(--w-panel);
     height: 100dvh;
     overflow: hidden;
   }
   ```
   Then add the panel column styling (a new rule block):
   ```css
   html.app #panel.app-panel {
     position: relative; height: 100%; overflow-y: auto; z-index: var(--z-base);
     border-left: 2px solid var(--parch-edge);
     background-color: var(--surface); background-image: var(--grain-ink); background-size: 4px 4px;
   }
   #panel-inner { display: flex; flex-direction: column; min-height: 100%; }
   .panel-empty { padding: var(--space-6); color: var(--ink-muted); font-family: var(--font-mono); font-size: 13px; }
   .panel-head {
     display: flex; align-items: center; gap: var(--space-2); padding: var(--space-2) var(--space-3);
     background: var(--surface-2); border-bottom: 2px solid var(--parch-edge);
     font-family: var(--font-mono); font-size: 11px; font-weight: 700; text-transform: uppercase;
     letter-spacing: .06em; color: var(--ink); position: sticky; top: 0; z-index: var(--z-sticky);
   }
   .panel-head-icon { width: 16px; height: 16px; image-rendering: pixelated; }
   .panel-head-title { flex: 1 1 auto; }
   .panel-close { background: none; border: 0; cursor: pointer; color: var(--ink-muted); font-size: 13px; line-height: 1; padding: 2px 4px; }
   .panel-close:hover { color: var(--ink); }
   .panel-body { padding: var(--space-4); }
   .panel-body .kcard { padding: 15px 18px; }
   ```
3. Add the re-open chip styling (a compact inline affordance in the transcript;
   align it with the chat gutter like artifacts were):
   ```css
   .art-chip {
     display: inline-flex; align-items: center; gap: var(--space-2);
     margin: 6px 0 6px var(--chat-gutter, 124px);
     padding: var(--space-1) var(--space-3); text-decoration: none;
     border: 2px solid var(--parch-edge); background: var(--surface-2); box-shadow: var(--drop-hard);
     font-family: var(--font-mono); font-size: 11px; font-weight: 700; text-transform: uppercase;
     letter-spacing: .06em; color: var(--ink); cursor: pointer;
   }
   a.art-chip:hover { background: var(--surface-3, var(--surface)); }
   .art-chip-icon { width: 14px; height: 14px; image-rendering: pixelated; }
   .art-chip-hint { color: var(--ink-muted); font-weight: 400; }
   ```
4. **Mobile drawer** — in the existing `@media (max-width: 720px)` block, take the
   panel out of the grid and make it a fixed overlay that slides in when open:
   ```css
   @media (max-width: 720px) {
     html.app .app-shell { grid-template-columns: 1fr; }   /* keep */
     html.app .app-shell .sb-side { display: none; }       /* keep */
     html.app #panel.app-panel {
       position: fixed; top: 0; right: 0; bottom: 0; width: min(92vw, var(--w-panel));
       z-index: var(--z-drawer); transform: translateX(100%); transition: transform .15s steps(3, end);
     }
     html.app.panel-open #panel.app-panel { transform: translateX(0); }
     html.app.panel-open::after { /* scrim */
       content: ""; position: fixed; inset: 0; z-index: var(--z-scrim); background: rgba(0,0,0,.45);
     }
   }
   ```
5. **Remove** the entire inline-artifact block (the `.artifact`, `.chat .artifact`,
   `.artifact-head*`, `.artifact-body*`, `.artifact--collapsed*` rules shown in
   Current state). They are replaced by `.panel-*`.

**Verify:** `CGO_ENABLED=0 go build ./...` (CSS is embedded; build still passes),
and `git grep -n "\.artifact" internal/web/assets/static/basm.css` returns
nothing (no orphaned inline-artifact rules; `.art-chip` is a different class).

### Step 3 — `ChatShell` gains a `Panel` region

In `internal/ui/shell/chatshell.go`: add `Panel g.Node` to `ChatShellProps` and
render it as a third `<aside>` after the dock:

```go
type ChatShellProps struct {
	Title   string
	Sidebar g.Node
	Dock    g.Node
	Panel   g.Node // the single-active right panel (chat.Panel)
}

// … inside ChatShell, the .app-shell div:
h.Div(h.Class("app-shell"),
	p.Sidebar,
	h.Aside(h.ID("dock"), h.Class("app-dock"), p.Dock),
	h.Aside(h.ID("panel"), h.Class("app-panel"), p.Panel),
),
```

**Verify:** `go build ./internal/ui/shell/...`. (`homePage` won't compile until
Step 8 passes `Panel:` — that's expected; do Step 8 before the package-level
build. If you want this step green in isolation, temporarily pass
`Panel: nil` in `homePage` and tighten it in Step 8.)

### Step 4 — Web helpers: render the panel + the chip; parse the restore pointer

Create `internal/web/panel.go`:

```go
package web

// panel.go — the single-active right-panel canvas (plan 098). Both summon doors
// (the owner's /ui/show and the agent's card_show/show_cards) render the active
// artifact here and drop a re-open chip into #chat. The panel survives reload via
// the owner_settings "panel_active" pointer (a re-summon URL).

import (
	"net/url"
	"strings"

	g "maragu.dev/gomponents"

	"github.com/alexradunet/balaur/internal/cards"
	"github.com/alexradunet/balaur/internal/store"
	"github.com/alexradunet/balaur/internal/ui/chat"
)

const panelActiveKey = "panel_active"

// showURL is the canonical re-summon/restore URL for a single card.
func showURL(typ, query string) string {
	if query == "" {
		return "/ui/show/" + typ
	}
	return "/ui/show/" + typ + "?" + query
}

// panelNode renders chat.Panel for one single-card artifact (typ + raw query
// string). The body is the full Focus surface (same as the old inline path).
func (h *handlers) panelNode(typ, query string) g.Node {
	title, icon := typ, ""
	if spec, ok := cards.Get(typ); ok {
		title, icon = spec.Label, spec.Icon
	}
	return chat.Panel(chat.PanelProps{Title: title, Icon: icon, Body: g.Raw(string(h.uicardBody(typ, query)))})
}

// panelClusterNode renders chat.Panel for an agent cluster (show_cards).
func (h *handlers) panelClusterNode(title string, cs []cards.Card) g.Node {
	return chat.Panel(chat.PanelProps{Title: title, Body: g.Raw(string(h.artifactBody(title, cs)))})
}

// emptyPanelNode is the placeholder shown when nothing is open.
func emptyPanelNode() g.Node { return chat.Panel(chat.PanelProps{}) }

// renderNodeHTML renders a node to an HTML string for SSE patching. There is NO
// free node→string helper in package web today (chatstream.go has only the
// METHOD renderNode on *chatStream, unusable from a *handlers method), so define
// it here. panel.go already imports strings and g.
func renderNodeHTML(n g.Node) string {
	var b strings.Builder
	_ = n.Render(&b)
	return b.String()
}

// chipNode renders the transcript re-open chip for a single card.
func (h *handlers) chipNode(typ, query string) g.Node {
	title, icon := typ, ""
	if spec, ok := cards.Get(typ); ok {
		title, icon = spec.Label, spec.Icon
	}
	return chat.ArtifactChip(chat.ArtifactChipProps{Title: title, Icon: icon, ReopenURL: showURL(typ, query)})
}

// clusterChipNode renders a non-clickable chip for an agent cluster.
func clusterChipNode(title string) g.Node {
	return chat.ArtifactChip(chat.ArtifactChipProps{Title: title})
}

// restoredPanelNode reads the persisted panel_active pointer and renders that
// artifact, or the empty placeholder. Only single-card URLs restore; anything
// else (or a parse failure) → empty.
func (h *handlers) restoredPanelNode() g.Node {
	raw := store.GetOwnerSetting(h.app, panelActiveKey, "")
	typ, query, ok := parseShowURL(raw)
	if !ok {
		return emptyPanelNode()
	}
	if _, ok := cards.Get(typ); !ok {
		return emptyPanelNode()
	}
	return h.panelNode(typ, query)
}

// parseShowURL splits "/ui/show/{type}?{query}" → (type, query, ok).
func parseShowURL(raw string) (typ, query string, ok bool) {
	const prefix = "/ui/show/"
	if !strings.HasPrefix(raw, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(raw, prefix)
	if i := strings.IndexByte(rest, '?'); i >= 0 {
		typ, query = rest[:i], rest[i+1:]
	} else {
		typ = rest
	}
	if typ == "" {
		return "", "", false
	}
	// Defensive: ensure query is valid form-encoding (drop it if not).
	if query != "" {
		if _, err := url.ParseQuery(query); err != nil {
			query = ""
		}
	}
	return typ, query, true
}
```

> `artifactBody` (cards.go:167) already renders `chat.Cluster` **untitled**
> (cards.go:173 — the old `artifactWrap` owned the title). The panel head now
> owns the title, so `panelClusterNode` produces no duplicate heading. Do not
> pass a title into the Cluster.

Add `internal/web/panel_unit_test.go` covering `parseShowURL` (with/without
query, bad prefix, empty type) and `showURL`. These are pure helpers — table
tests, no PocketBase.

**Verify:** `gofmt -l internal/web/panel.go internal/web/panel_unit_test.go &&
go test ./internal/web/ -run 'ParseShowURL|ShowURL'` (after the package compiles
— you may need Steps 5–9 done first for the package to build; run the focused
test at the end of Step 9).

### Step 5 — Owner door (`show.go`): morph the panel, append the chip, persist

The real `uiShow` (show.go:22–56) is: `PathValue("type")` → `cards.Get` →
`cards.Validate(typ, queryToMap(...))` → `tools.MarkUICard(typ, params, …)` →
`conversation.Master` → `conversation.AppendOriginRec(..., role:"tool",
origin:"")` → then the final two lines:
```go
	body := h.renderMessages(h.messageViews([]*core.Record{rec}))
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(string(body), datastar.WithSelectorID("chat"), datastar.WithModeAppend())
	return nil
```
Keep the entire block up to and including `AppendOriginRec` **unchanged** (the
chip's reload trace depends on that persisted `role=tool`/`origin=""` row, and
`params` still feeds `MarkUICard`). REPLACE only those final two lines with:

```go
	// Derive the canonical query from the marker we just built, so the live
	// chip/panel/restore URL is byte-identical to what the reload path
	// (recap.messageViews → tools.ParseUICard) produces for the same artifact.
	_, queryStr, _, _ := tools.ParseUICard(marker)

	// Single-active panel: morph #panel-inner with this artifact; drop a re-open
	// chip into #chat; remember it as the last-active artifact.
	_ = store.SetOwnerSetting(h.app, panelActiveKey, showURL(typ, queryStr))

	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(renderNodeHTML(h.panelNode(typ, queryStr)))                 // morph by root id "panel-inner"
	_ = sse.PatchElements(renderNodeHTML(h.chipNode(typ, queryStr)),
		datastar.WithSelectorID("chat"), datastar.WithModeAppend())
	return nil
```

**`show.go` does NOT import `internal/store` today** (its imports are `core`,
`datastar`, `cards`, `conversation`, `llm`, `tools`). **Add**
`"github.com/alexradunet/balaur/internal/store"` to its import block. (`showURL`,
`renderNodeHTML`, `panelNode`, `chipNode`, `panelActiveKey` all live in the same
`web` package via `panel.go` — no other new import needed; `renderNodeHTML` is
defined in Step 4.) `uiShow` no longer references `renderMessages`/`messageViews`
— that's fine, they remain used elsewhere.

**Verify:** `CGO_ENABLED=0 go build ./...` + the updated `show_test.go` (Step 10).

### Step 6 — Agent live path (`chatstream.go`): route artifacts to the panel + chip

Add a panel-aware artifact emitter and call it from `handleToolResult` for the
uicard and cluster cases. **Leave the proposal / choices / refresh cases calling
the existing `endTool` unchanged** (they are conversational, not domain
artifacts, and stay inline).

```go
// handleToolResult — change ONLY the uicard and artifact(cluster) branches:
	if typ, query, rest, ok := tools.ParseUICard(ev.Text); ok {
		s.endArtifactCard(rest, typ, query)
		return
	}
	// … choices / proposal / refresh: unchanged (s.endTool(...)) …
	if title, cs, rest, ok := tools.ParseArtifact(ev.Text); ok {
		s.endArtifactCluster(rest, title, cs)
		return
	}
	s.endTool(clipText(ev.Text, 2000), "", "", "")
}

// endArtifactCard morphs the tool row, then routes a single card to the panel
// (single-active) and drops a re-open chip into #chat. Mirrors the owner door
// (show.go) so live and reload agree.
func (s *chatStream) endArtifactCard(content, typ, query string) {
	s.morphNode(chat.ToolRow(chat.ToolRowProps{
		Tool: s.toolName, Icon: toolIconFile(s.toolName), ID: s.toolID, BodyID: s.toolBody, Content: content,
	}))
	_ = store.SetOwnerSetting(s.h.app, panelActiveKey, showURL(typ, query))
	s.morphNode(s.h.panelNode(typ, query))                  // morph #panel-inner
	s.appendNode(s.h.chipNode(typ, query))                  // chip → #chat
}

// endArtifactCluster routes an agent cluster to the panel with a non-clickable
// chip (clusters have no deterministic re-open URL — plan 090). Clusters do not
// update panel_active (no restore URL).
func (s *chatStream) endArtifactCluster(content, title string, cs []cards.Card) {
	s.morphNode(chat.ToolRow(chat.ToolRowProps{
		Tool: s.toolName, Icon: toolIconFile(s.toolName), ID: s.toolID, BodyID: s.toolBody, Content: content,
	}))
	s.morphNode(s.h.panelClusterNode(title, cs))
	s.appendNode(clusterChipNode(title))
}
```

**`chatstream.go` imports `cards` but NOT `internal/store` today** (Go imports
are per-file — do not assume "the package already imports it"). **Add**
`"github.com/alexradunet/balaur/internal/store"` to `chatstream.go`'s import
block (same canonical path as `recap.go`). `cards` is already imported there.

The old `endTool`'s artifact branch (the `artifactWrap(...)` append) is now dead
for uicard/cluster but `endTool` is still called for proposals/choices/refresh —
leave `endTool` in place but **remove its final `artifactWrap` branch** since
`artTitle` is now never non-empty from any caller. Concretely, `endTool` becomes:

```go
func (s *chatStream) endTool(content string, card template.HTML, artTitle, artIcon string) {
	s.morphNode(chat.ToolRow(chat.ToolRowProps{
		Tool: s.toolName, Icon: toolIconFile(s.toolName), ID: s.toolID, BodyID: s.toolBody, Content: content,
	}))
	if card == "" {
		return
	}
	// Only proposals reach here with a card; they stay inline in the transcript.
	s.appendNode(g.El("div", g.Attr("class", "k-inline"), g.Attr("id", s.toolID+"-card"), g.Raw(string(card))))
}
```
(The `artTitle`/`artIcon` params are now always "" from callers. **Prefer the
simpler signature** `endTool(content string, card template.HTML)` and update the
**4 remaining callers** — choices (~line 187), proposal (~192), refresh (~196),
and the plain-text fallback (~206) — each currently passes two trailing `""`
args you must drop. Don't miss the fallback at ~206: it is not a
"proposal/choices/refresh" branch and is easy to overlook. This is in-scope
cleanup of your own change.)

**Verify:** `CGO_ENABLED=0 go build ./...`; `go test ./internal/web/...`.

### Step 7 — Reload path (`recap.go`): render chips, drop the cap

In `messageViews`, for the **uicard** case, stop rendering the heavy body and
record the re-summon coordinates instead; for the **cluster** case, keep the
title only. Add `ArtifactType` and `ArtifactQuery` to the `messageView` struct
(find it — it carries `Role/Tool/Content/Origin/CardBody/ArtifactTitle/
ArtifactIcon/ArtifactCollapsed/…`; **remove `ArtifactCollapsed`**, it is now
unused).

```go
// messageViews, tool branch:
if typ, query, rest, ok := tools.ParseUICard(mv.Content); ok {
	mv.Content = rest
	mv.ArtifactType, mv.ArtifactQuery = typ, query
	if spec, ok := cards.Get(typ); ok {
		mv.ArtifactTitle, mv.ArtifactIcon = spec.Label, spec.Icon
	} else {
		mv.ArtifactTitle = typ
	}
	// NOTE: do NOT set mv.CardBody — the artifact lives in the panel, not inline.
} else if /* choices: unchanged */ } else if kind, id, rest, ok := tools.ParseProposal(mv.Content); ok {
	mv.CardBody, mv.Content = h.proposalBody(kind, id), rest   // proposals stay inline (unchanged)
} else if /* refresh: unchanged */ } else if title, cs, rest, ok := tools.ParseArtifact(mv.Content); ok {
	mv.Content = rest
	mv.ArtifactTitle = title
	// cluster: non-clickable chip; ArtifactType stays "" (no re-open URL).
}
…
// REMOVE: capArtifacts(out)
```

Delete the `capArtifacts` function entirely.

In `renderMessages`, change the tool branch to emit a chip for artifacts and
keep proposals inline:

```go
case "tool":
	nodes = append(nodes, chat.ToolRow(chat.ToolRowProps{
		Tool: mv.Tool, Icon: toolIconFile(mv.Tool), Content: mv.Content,
	}))
	switch {
	case mv.ArtifactType != "": // single card → clickable re-open chip
		nodes = append(nodes, h.chipNode(mv.ArtifactType, mv.ArtifactQuery))
	case mv.ArtifactTitle != "": // cluster → non-clickable chip
		nodes = append(nodes, clusterChipNode(mv.ArtifactTitle))
	case mv.CardBody != "": // proposal etc. → inline k-inline (unchanged)
		nodes = append(nodes, g.El("div", g.Attr("class", "k-inline"), g.Raw(string(mv.CardBody))))
	}
```

> `renderMessages` is a method on `*handlers` (`h`), so `h.chipNode` is in scope.

**Ordering hazard:** removing `ArtifactCollapsed` + `capArtifacts` here will make
`go test ./internal/web/...` **fail to compile** until Step 9 deletes
`artifact_cap_test.go` (it references `capArtifacts`/`ArtifactCollapsed`). At
this step run only `CGO_ENABLED=0 go build ./...` (test files are excluded from
that). If you prefer, delete `internal/web/artifact_cap_test.go` now (it is
listed in Step 9 anyway) so the package test compiles again. Defer the full
`go test` to Step 10.

**Verify:** `CGO_ENABLED=0 go build ./...` (exit 0).

### Step 8 — `homePage` restores the last-active panel

In `internal/web/home.go`, `homePage`, pass the restored panel into `ChatShell`:

```go
	page := shell.ChatShell(shell.ChatShellProps{
		Title:   "Home",
		Sidebar: shell.Sidebar(domainSidebar()),
		Dock:    dockNode,
		Panel:   h.restoredPanelNode(),
	})
```

`homePage` (home.go) needs no new import — `h.restoredPanelNode()` returns
`g.Node` and home.go already imports `g`.

Add a **panel-close** route + handler so the `✕` works. **Put the handler in
`panel.go`** (it already imports `store`); add `core` and `datastar` to
`panel.go`'s imports for it:
```go
// panel.go imports gain:
//   "github.com/pocketbase/pocketbase/core"
//   "github.com/starfederation/datastar-go/datastar"

func (h *handlers) panelClose(e *core.RequestEvent) error {
	_ = store.SetOwnerSetting(h.app, panelActiveKey, "")
	sse := datastar.NewSSE(e.Response, e.Request)
	_ = sse.PatchElements(renderNodeHTML(emptyPanelNode())) // morph #panel-inner → empty
	return nil
}
```
Register `GET /ui/panel/close` in `web.go` next to `GET /ui/show/{type}` (mirror
that registration exactly — same router group / guard; look at how `/ui/show/{type}`
is mounted and copy the pattern). Note `/ui/panel/{...}` must not collide with any
existing `/ui/panel*` route (grep first: `git grep -n '"/ui/panel' internal/web`).

**Verify:** `CGO_ENABLED=0 go build ./...`; then confirm `home_test.go` still
passes (it renders `/` — the new `#panel` aside must not break existing
assertions; if `home_test` asserts the exact shell structure, update it to allow
the `#panel` aside).

### Step 9 — Retire the inline-artifact machinery

Remove, now that nothing renders inline artifacts:

- `internal/web/cards.go`: delete `activeArtifactCap` and `artifactWrap`.
- `internal/web/recap.go`: `capArtifacts` already deleted (Step 7); confirm no
  references remain.
- `internal/web/assets/static/basm.js`: delete `ACTIVE_ARTIFACT_CAP` +
  `balaurCapArtifacts` + its two call sites; keep `balaurScrollToLatest` and the
  `#chat` MutationObserver (now just `() => { balaurScrollToLatest(); }`).
- `internal/ui/chat/artifact.go`: **delete** the file (the `chat.Artifact`
  organism). Its visual language now lives in `chat.Panel`.
- `internal/ui/chat/artifact_test.go`: delete.
- `internal/feature/storybook/stories_chat.go`: remove `chatartifactStory`;
  `internal/feature/storybook/story.go`: remove its registration. (You added the
  panel/chip stories in Step 1.)
- `internal/web/artifact_cap_test.go` (plan 094's unit tests of `capArtifacts`):
  delete — the function is gone.
- `internal/web/assets/static/basm.css`: the `.artifact*` rules were removed in
  Step 2 — confirm.

Add a small client hook for the **mobile drawer auto-open**: when `#panel-inner`
gains content on a narrow viewport, add `panel-open` to `<html>`; the `✕`/scrim
remove it. Append to `basm.js`:

```js
// ── Right panel: auto-open the mobile drawer when an artifact is summoned ──
document.addEventListener('DOMContentLoaded', () => {
  var inner = document.getElementById('panel-inner');
  if (!inner) return;
  var isNarrow = function () { return window.matchMedia('(max-width: 720px)').matches; };
  new MutationObserver(function () {
    if (isNarrow() && !inner.querySelector('.panel-empty')) {
      document.documentElement.classList.add('panel-open');
    }
  }).observe(inner, { childList: true, subtree: true });
  // The scrim is a ::after pseudo-element (NOT a clickable DOM node) — so close
  // on any click that lands outside #panel and .sb-side while the drawer is open.
  document.addEventListener('click', function (e) {
    if (document.documentElement.classList.contains('panel-open') &&
        !e.target.closest('#panel') && !e.target.closest('.sb-side')) {
      document.documentElement.classList.remove('panel-open');
    }
  });
  document.addEventListener('keydown', function (e) {
    if (e.key === 'Escape') document.documentElement.classList.remove('panel-open');
  });
});
```
> Keep this minimal. The close `✕` already clears panel content server-side
> (`/ui/panel/close`); this hook only handles the slide-in/out *visibility* on
> narrow screens. Full a11y parity (focus-trap, `inert`) is plan 100.

**Verify:**
```sh
# Anchor chat.Artifact( so it does NOT match the NEW chat.ArtifactChip organism.
git grep -nE 'artifactWrap|activeArtifactCap|balaurCapArtifacts|ArtifactCollapsed|capArtifacts|chat\.Artifact\(' internal/
```
Expect: no hits (all in production AND tests are gone). If a hit remains, you
missed a reference — resolve it. Then `CGO_ENABLED=0 go build ./...` and
`go test ./...` (this is the first point both can be green again).

### Step 10 — Tests

Update/author tests so the suite pins the new behavior:

- **`internal/web/show_test.go`** (rewrite the assertions — the door no longer
  appends the full card to `#chat`; it morphs the panel and appends a chip):
  - The `quests → 200` subtest's `ExpectedContent` becomes: `datastar-patch-elements`,
    `id="panel-inner"` (the panel morph), `quest-stack` (the Focus body, now in
    the panel), `art-chip` and `selector #chat` + `mode append` (the chip
    append). Remove the old `artifact-head` assertion (that was the inline frame).
  - The persistence `AfterTestFunc` (role=tool, origin="", marker) is **unchanged**
    — keep it; it still must hold.
  - Add an `AfterTestFunc` assertion (or a new subtest) that
    `store.GetOwnerSetting(app, "panel_active", "")` == `/ui/show/quests` after
    the call.
  - Keep the `chatNudges` no-dup subtest and the `404`/`400` subtests as-is.
- **`internal/web/panel_unit_test.go`** (new, Step 4): `parseShowURL`/`showURL`.
- **`internal/ui/chat/panel_test.go`** (new, Step 1).
- **`internal/web/recap` reload test**: add or adjust a test that a persisted
  uicard tool row renders as an `art-chip` (not an `.artifact`) and that a
  persisted cluster row renders a non-clickable chip. If a test asserted the old
  inline `artifact`/cap behavior, repoint it.
- **`internal/web/handlers_test.go` `TestChatCardShow`**: this test (touched by
  plan 097) asserts the agent `card_show` renders `class="artifact"` +
  `artifact-head` + `id="ucard-today"`. After this plan, the live agent path
  morphs the panel + appends a chip. Repoint the assertions to the panel/chip
  markup (`id="panel-inner"`, `art-chip`, and keep a body assertion like
  `id="ucard-today"` which now lives inside the panel). This is a mechanical
  assertion update (the test stays meaningful).
- **`internal/web/home_test.go`**: if it asserts the shell structure, allow the
  new `#panel` aside; add a subtest that after `/ui/show/quests`, a fresh `GET /`
  renders the quests artifact inside `#panel-inner` (restore-last-active) and an
  `art-chip` in `#chat`.
- **`internal/web/handlers_test.go` `TestUICardHistoryRendersCardInline`** (≈line
  472): this test hand-builds `mv.CardBody = h.uicardBody(typ, query)` and renders
  the `chat-msg-tool` html/template, asserting `class="k-inline"` + `id="ucard-today"`
  for a uicard. After this plan a uicard reloads as a **chip** (no `CardBody`), so
  this test now encodes the *retired* inline path. It will still PASS untouched
  (it bypasses `messageViews`/`renderMessages`), but leaving it **lies about the
  reload contract** — so **repoint it**: assert that
  `renderMessages(messageViews([uicard tool row]))` produces `art-chip` and NOT
  `k-inline`. (Do not delete the `chat-msg-tool` html/template itself — it is
  still used by other tool-row tests; only this assertion is wrong now.)
- **`internal/web/handlers_test.go` `TestSettingsPages` (≈115) and `TestHeadsFocus`
  (≈311)**: these hit the owner door `/ui/show` (rewritten in Step 5) and assert
  body-internal ids (`identity-card`, `models-panel`, `ucard-heads`) — NOT the
  inline frame. Those ids still appear in the SSE response (now inside the panel
  morph), so **these tests pass unchanged — do NOT touch them.** (Listed here so
  you don't waste effort "fixing" passing tests.)
- Delete `internal/web/artifact_cap_test.go` (Step 9; or earlier per Step 7's
  ordering note).

**Verify (full gate — must all be green before you stop):**
```sh
gofmt -l .                       # prints nothing
go vet ./...                     # exit 0
CGO_ENABLED=0 go build ./...     # exit 0
go test ./...                    # no FAIL / panic
git diff --check                 # no whitespace errors
```

### Step 11 — Docs (same-commit truth sync — this repo's standing contract)

- **`internal/self/knowledge.md`** — the inline-artifact / `#chat`-append claims
  are **scattered**, not in one section. Grep first:
  `git grep -n "inline artifact\|#chat\|sub-window\|/ui/show\|appends" internal/self/knowledge.md`,
  and fix **every** hit (the critic found them around lines ~104, ~112–118, ~136,
  ~138–141, ~170, ~176). The replacement narrative: "A summoned artifact (owner
  rail click via `/ui/show/{type}`, or the agent's `card_show`/`show_cards`)
  renders in the single-active **right panel** (`chat.Panel`, `#panel-inner`); the
  chat keeps a compact re-open chip (`chat.ArtifactChip`) as the durable,
  auditable transcript trace. The panel restores the last-active artifact on
  reload via `owner_settings["panel_active"]`. Clusters render in the panel with a
  non-clickable chip." Remove the cap (plan 094) and inline-frame (plan 097)
  descriptions.
- **`DESIGN.md`** §3 (honesty ledger / IA) — also scattered. Grep
  `git grep -n "#chat\|artifact\|/ui/show" DESIGN.md` and fix every hit that
  asserts inline/stream rendering: the door description (~lines 93–94,
  "SSE-appends a rendered card artifact to #chat — no navigation"), the
  "rendered as artifacts into the conversation stream" / "two doors converge on
  #chat" wording, and the per-domain "artifact at /ui/show/{X}" phrases (~125,
  134, 136, 147, 154, 389) which now open in the **panel**, not the stream.
- **Storybook "Don'ts"** in `internal/feature/storybook/stories_navigation.go`:
  the Sidebar **Blurb** at ~line 147 ("…injects its card into the live #chat via
  a Datastar @get" — this is a Blurb, not a Don't, so don't skip it while scanning
  for "Don't") and the Don'ts at ~187–188 ("inject the card into #chat instead",
  "navigation belongs in the sidebar") encode the retired inline IA. Repoint
  "into #chat" → "into the right panel". Do **not** add in-panel-tabs guidance yet
  — that is plan 099 (the line-188 "no category tabs inside artifacts" Don't stays
  as-is for now; 099 will revisit it).
- If `.tours/` references any line you moved (run `go test ./... -run Tours`),
  fix the anchor in the same commit.

**Verify:** `go test ./... -run Tours` (if tours exist) + the full gate again.

---

## Files in scope

- `internal/ui/chat/panel.go` (new), `internal/ui/chat/panel_test.go` (new)
- `internal/ui/chat/artifact.go` (delete), `internal/ui/chat/artifact_test.go` (delete)
- `internal/ui/shell/chatshell.go` (add Panel region)
- `internal/web/panel.go` (new), `internal/web/panel_unit_test.go` (new)
- `internal/web/show.go` (re-target the door)
- `internal/web/chatstream.go` (route agent artifacts to the panel; simplify `endTool`)
- `internal/web/recap.go` (chips on reload; delete `capArtifacts`)
- `internal/web/cards.go` (delete `artifactWrap`, `activeArtifactCap`)
- `internal/web/home.go` (restore panel; panel-close handler)
- `internal/web/web.go` (register `GET /ui/panel/close`)
- `internal/web/assets/static/basm.css` (panel grid/chrome/chip/drawer; remove `.artifact*`)
- `internal/web/assets/static/basm.js` (drop the cap; mobile drawer auto-open)
- `internal/feature/storybook/stories_chat.go` + `story.go` (panel/chip stories; drop artifact story)
- `internal/feature/storybook/stories_navigation.go` (soften inline-IA Don'ts)
- Tests: `show_test.go`, `handlers_test.go`, `home_test.go`, recap reload test;
  delete `artifact_cap_test.go`
- Docs: `internal/self/knowledge.md`, `DESIGN.md`, a tour anchor if needed

## Files explicitly OUT of scope (do not touch)

- The left rail's structure / grouped sub-entries (`domainSidebar` in `home.go`)
  beyond what `homePage` needs to pass `Panel:` — **collapsing sub-entries into
  in-panel tabs is plan 099.** The rail stays as-is.
- `internal/cards/*` (the registry), `internal/tools/ui.go` / `artifact.go` (the
  markers) — the marker format is unchanged; this plan only changes *where the
  rendered output lands*.
- The agent loop / `internal/turn` / `internal/agent` — no message-id plumbing is
  needed (re-open is by URL, not id).
- `internal/ui/chat/cluster.go` — `chat.Cluster` is reused verbatim as a cluster
  panel body.
- Full a11y drawer parity + the left rail's mobile reveal + switcher placement —
  plan 100 (re-scope of 091).

## Done criteria (machine-checkable)

```sh
gofmt -l .                                   # empty
go vet ./...                                 # exit 0
CGO_ENABLED=0 go build ./...                 # exit 0
go test ./...                                # no FAIL / panic
git diff --check                             # clean
git grep -n "grid-template-columns: 274px 1fr var(--w-panel)" internal/web/assets/static/basm.css   # present
git grep -n "id=\"panel\"" internal/ui/shell/chatshell.go                                          # present
git grep -nE 'artifactWrap|activeArtifactCap|capArtifacts|balaurCapArtifacts|chat\.Artifact\(' internal/  # empty (note: chat\.Artifact\( — NOT chat.ArtifactChip)
git grep -n "panel_active" internal/web/panel.go internal/web/show.go                              # present
```
Behavioral (manual, `make run`): clicking a rail entry opens that artifact in the
right panel and drops a chip in chat; clicking another rail entry **replaces** the
panel (only one active); clicking a chip re-opens its artifact; reload restores
the last-active artifact in the panel; the agent's `card_show` opens in the panel
too; the `✕` clears the panel; on a ≤720px viewport the panel is an overlay that
slides in on summon. **Eyeball each Focus surface in the 480px panel** (quests,
lifelog, a memory slice, skills, each settings section, day): if a card overflows
or truncates, **widen `--w-panel`** (e.g. 560px) rather than redesigning the card,
and note any that still read cramped in the PR. The executor has no visual
feedback loop — if you cannot verify this, say so explicitly in your report.

## Maintenance notes / what review must check

- **Live == reload invariant (plan 088's #1 pin).** The owner door (`show.go`),
  the agent live path (`chatstream.go`), and the reload path (`recap.go`) must
  all produce the same chip + the same panel body for the same artifact. All
  three derive the chip query from `tools.ParseUICard(...)` of the same marker
  (show.go parses the marker it just built; chatstream/recap parse the persisted
  marker), so the chip `ReopenURL` is **byte-identical** across paths — that is
  the point of deriving from the marker rather than the raw request URL. Review
  should diff the three; do not write a byte-equality test against `RawQuery`.
- **`panel_active` is a re-summon URL, re-validated on use.** It is owner-only
  data in `owner_settings`; `restoredPanelNode` and `uiShow` both re-`Validate`,
  so a stale/garbage value degrades to the empty panel, never an error page.
- **Clusters are the one asymmetry**: non-clickable chip, no restore. If 099 or a
  later plan makes clusters re-openable, it will add a by-id door — note it then.
- The **width** of the Focus card inside a 480px panel is the open visual
  question. The Focus surfaces were sized for the wide chat column. If they read
  cramped, widen `--w-panel` or make the Focus cards reflow — flag in review;
  do not over-engineer here.

## Escape hatches — STOP and report instead of improvising

- If the live agent path turns out to need the persisted message id (it should
  NOT — re-open is by URL), **STOP**: that means an assumption here is wrong;
  report it rather than plumbing ids through the turn pipeline.
- If removing `chat.Artifact` breaks a consumer you did not expect (grep first:
  `git grep -n "chat.Artifact\|ArtifactProps" internal/`), **STOP** and report
  the consumer.
- If `home_test.go`/`handlers_test.go` assert structure you cannot reconcile with
  the panel without gutting the test's meaning, **STOP** and report — do not
  delete assertions to make them pass.
- If the three-column grid breaks the chat column's existing centering
  (`--w-chat-home`) badly, note it and proceed with a sensible `max-width`; do not
  redesign the chat column here.
```
