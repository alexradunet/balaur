# Plan 097: Frame each in-chat artifact as a self-contained titled "sub-window"

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm the expected result before moving to the
> next step. If anything in the "STOP conditions" section occurs, stop and
> report — do not improvise. A reviewer dispatched you and maintains
> `plans/README.md`, so do NOT edit the index.
>
> **Drift check (run first)**:
> `git diff --stat 58d7182..HEAD -- internal/web/cards.go internal/web/chatstream.go internal/web/recap.go internal/ui/chat/ internal/web/assets/static/basm.css internal/web/assets/static/basm.js internal/feature/storybook/ internal/self/knowledge.md`
> If any in-scope file changed since this plan was written, compare the
> "Current state" excerpts against the live code before proceeding; on a
> mismatch, treat it as a STOP condition.

## Status

- **Priority**: P2
- **Effort**: M
- **Risk**: MED
- **Depends on**: none (builds on plans 088–094, already landed)
- **Category**: dx / direction (UI presentation)
- **Planned at**: commit `58d7182`, 2026-06-18

## Why this matters

When an in-chat artifact (a domain card summoned via the sidebar door
`/ui/show/{type}`, or the agent's `card_show` / `show_cards`) is **active**
(not yet aged out by the cap), it renders **frameless**: a bare
`<div class="k-inline">` holding the Focus-size card body, indented to the
chat gutter with no border, no surface, no header, and no bottom edge. Its
internal sections bleed straight into the next chat message. A title only
appears *after* the artifact collapses (the `.artifact-chip`, which is
`display:none` until `.artifact--collapsed`). The owner cannot tell where one
artifact ends and the next message begins.

This plan wraps every in-chat artifact in an always-visible **titled
"sub-window"**: a header bar (icon + artifact name) atop a bordered parchment
body with a closed bottom edge — the Hearthwood window-chrome vocabulary
(`--surface`, `--parch-edge`, `--parch-bevel`, square 0px panels). The frame
makes each artifact read as a discrete, finished, self-contained unit, so the
message boundary is obvious. The collapse/cap mechanism is preserved unchanged
(server `capArtifacts` + JS `balaurCapArtifacts` toggle `.artifact--collapsed`);
only the **visual** of the expanded and collapsed states changes — collapsed
artifacts fold to just their title bar instead of swapping to a separate chip.

The frame lives as a new `chat.Artifact` gomponents organism (mirroring the
existing `chat.Cluster`), so the storybook stays the source of truth and the
chrome is defined once.

## Current state

### The wrapper today — `internal/web/cards.go`

`artifactWrap` (lines 197–212) produces the frameless markup; `artifactChip`
(lines 182–195) is the collapse-only "shown earlier" summary:

```go
// activeArtifactCap bounds how many artifacts stay fully rendered in the chat;
// older ones collapse to a static "shown earlier" chip (plan 094). The live
// path enforces the same cap client-side (balaurCapArtifacts in basm.js).
const activeArtifactCap = 3

// artifactChip is the static, non-interactive summary shown when an artifact
// is collapsed. icon is a /static/icons stem ("" → no icon).
func artifactChip(title, icon string) g.Node {
	if title == "" {
		title = "Artifact"
	}
	kids := []g.Node{g.Attr("class", "artifact-chip"), g.Attr("aria-hidden", "true")}
	if icon != "" {
		kids = append(kids, g.El("img", g.Attr("class", "artifact-chip-icon"),
			g.Attr("src", "/static/icons/"+icon+".png"), g.Attr("alt", ""), g.Attr("decoding", "async")))
	}
	kids = append(kids, g.El("span", g.Text(title+" (shown earlier)")))
	return g.El("div", kids...)
}

// artifactWrap wraps a rendered artifact body in the .artifact container with
// its (hidden) chip. collapsed adds .artifact--collapsed (CSS then hides the
// body and reveals the chip). innerID, when set, is placed on the .k-inline
// body (preserves the live path's tool-card id).
func artifactWrap(title, icon string, collapsed bool, innerID string, body template.HTML) g.Node {
	cls := "artifact"
	if collapsed {
		cls += " artifact--collapsed"
	}
	inner := []g.Node{g.Attr("class", "k-inline")}
	if innerID != "" {
		inner = append(inner, g.Attr("id", innerID))
	}
	inner = append(inner, g.Raw(string(body)))
	return g.El("div", g.Attr("class", cls), artifactChip(title, icon), g.El("div", inner...))
}
```

Produces (active):
```html
<div class="artifact">
  <div class="artifact-chip" aria-hidden="true">…icon… <span>{title} (shown earlier)</span></div>
  <div class="k-inline" id="{innerID}">{body}</div>
</div>
```

### The two callers (do NOT change their call signatures)

`internal/web/chatstream.go:209-223` — the live SSE path:
```go
func (s *chatStream) endTool(content string, card template.HTML, artTitle, artIcon string) {
	s.morphNode(chat.ToolRow(chat.ToolRowProps{
		Tool: s.toolName, Icon: toolIconFile(s.toolName), ID: s.toolID, BodyID: s.toolBody, Content: content,
	}))
	if card == "" {
		return
	}
	if artTitle == "" { // proposal etc. — not an artifact; keep the plain inline card
		s.appendNode(g.El("div", g.Attr("class", "k-inline"), g.Attr("id", s.toolID+"-card"), g.Raw(string(card))))
		return
	}
	s.appendNode(artifactWrap(artTitle, artIcon, false, s.toolID+"-card", card))
}
```

`internal/web/recap.go:204-210` — the reload path (inside `renderMessages`, `case "tool"`):
```go
if mv.CardBody != "" {
	if mv.ArtifactTitle != "" {
		nodes = append(nodes, artifactWrap(mv.ArtifactTitle, mv.ArtifactIcon, mv.ArtifactCollapsed, "", mv.CardBody))
	} else {
		nodes = append(nodes, g.El("div", g.Attr("class", "k-inline"), g.Raw(string(mv.CardBody))))
	}
}
```

**Key invariant**: only true artifacts (`artTitle != ""` / `mv.ArtifactTitle != ""`)
go through `artifactWrap`. Proposals and plain inline cards render as bare
`.k-inline` and must stay frameless. Preserve this gate exactly.

### The cluster path — `internal/web/cards.go:167-175`

```go
func (h *handlers) artifactBody(title string, cs []cards.Card) template.HTML {
	nodes := make([]g.Node, 0, len(cs))
	for _, c := range cs {
		nodes = append(nodes, g.Raw(string(h.cardHTML(c.Type, c.Params))))
	}
	var b strings.Builder
	_ = chat.Cluster(chat.ClusterProps{Title: title, Cards: nodes}).Render(&b)
	return template.HTML(b.String())
}
```

The cluster's `title` is passed BOTH to `chat.Cluster` (which renders a
`.k-cluster-head` heading) AND, by the caller, to `artifactWrap` as the artifact
title. After this plan the window header owns the title, so the cluster's own
heading would be a duplicate — Step 4 removes it from the `chat.Cluster` call.

### The cap CSS — `internal/web/assets/static/basm.css:1038-1054`

```css
.k-inline { margin: 10px 0; max-width: 480px; }
/* In chat, an embedded record rides the message column: same left edge
   and full width as the speech panels — same UI, same space. */
.chat .k-inline { margin: 0 0 0 var(--chat-gutter, 124px); max-width: none; }
.chat .k-inline .kcard { padding: 15px 18px; }

/* Artifact cap (plan 094): collapsed artifacts hide the body, show the chip. */
.artifact-chip { display: none; }
.artifact--collapsed > .k-inline { display: none; }
.artifact--collapsed > .artifact-chip {
  display: flex; align-items: center; gap: var(--space-2);
  margin: 10px 0; padding: var(--space-2) var(--space-3);
  font-family: var(--font-mono); font-size: 11px; text-transform: uppercase;
  letter-spacing: .06em; color: var(--ink-muted);
}
.chat .artifact--collapsed > .artifact-chip { margin-left: var(--chat-gutter, 124px); }
.artifact-chip-icon { width: 16px; height: 16px; }
```

The `.k-inline` rules (first block, lines 1038–1042) are still used by the
**proposal / plain inline card** path — KEEP them. The `.artifact-chip*` /
`.artifact--collapsed > .k-inline` rules (lines 1044–1054) are replaced in Step 5.

### The JS cap — `internal/web/assets/static/basm.js:147-160` (NO change, for reference)

```js
var ACTIVE_ARTIFACT_CAP = 3;
function balaurCapArtifacts() {
  var chat = document.getElementById('chat');
  if (!chat) return;
  var arts = chat.querySelectorAll('.artifact');
  var cutoff = arts.length - ACTIVE_ARTIFACT_CAP;
  arts.forEach(function (el, i) {
    el.classList.toggle('artifact--collapsed', i < cutoff);
  });
}
```

This toggles `.artifact--collapsed` on `.artifact` elements. The new frame keeps
both class names (`artifact`, `artifact--collapsed`) on the same root element, so
this JS keeps working unchanged. Do NOT edit basm.js.

### The exemplar organism — `internal/ui/chat/cluster.go` (the pattern to mirror)

```go
package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ClusterProps configures the chat.Cluster organism — one conversation
// artifact holding N pre-rendered cards. Children are rendered by the caller
// (web layer via h.cardHTML), so internal/ui/chat imports no feature/cards.
type ClusterProps struct {
	Title string   // optional heading; omitted when ""
	Cards []g.Node // pre-rendered card nodes, in order
}

// Cluster renders a titled vertical stack of cards as one inline artifact.
func Cluster(p ClusterProps) g.Node {
	kids := []g.Node{h.Class("k-cluster")}
	if p.Title != "" {
		kids = append(kids, h.Header(h.Class("k-cluster-head"), h.H3(g.Text(p.Title))))
	}
	body := make([]g.Node, 0, len(p.Cards)+1)
	body = append(body, h.Class("k-cluster-body"))
	body = append(body, p.Cards...)
	kids = append(kids, h.Div(body...))
	return h.Div(kids...)
}
```

### Design constraints (from `DESIGN.md` / the Hearthwood tokens in basm.css)

Reuse — do NOT invent new colors or radii:
- Surface: `background-color: var(--surface)` + `background-image: var(--grain-ink); background-size: 4px 4px;`
- Border: `2px solid var(--parch-edge)` (panels are **square** — `--radius` is `0px`; do not round)
- Elevation: `box-shadow: var(--parch-bevel)`
- Title-bar surface: `var(--surface-2)`; muted text: `var(--ink-muted)`; mono labels: `var(--font-mono)`
- Spacing scale: `--space-2` (8px), `--space-3` (12px), `--space-4` (16px)
- Icons live at `/static/icons/{stem}.png`, rendered `image-rendering: pixelated`

## Commands you will need

| Purpose      | Command                                   | Expected on success      |
|--------------|-------------------------------------------|--------------------------|
| Format       | `gofmt -l internal/`                      | no output (all formatted)|
| Vet          | `go vet ./...`                            | exit 0, no findings      |
| Build (CGO!) | `CGO_ENABLED=0 go build ./...`            | exit 0                   |
| Test (pkg)   | `go test ./internal/ui/chat/ ./internal/web/ ./internal/feature/storybook/` | all pass |
| Test (all)   | `go test ./...`                           | all pass                 |
| Diff check   | `git diff --check`                        | no whitespace errors     |

`CGO_ENABLED=0` is mandatory for this repo — the build must pass with it.
`gofmt` is enforced; run it (a PostToolUse hook may also auto-format).

## Suggested executor toolkit

- This is server-rendered gomponents + Datastar (no Node build, no JS framework).
  Follow the `ui-development` skill workflow if available: check the storybook
  first, reuse/extend a component, add/update its story in the same change.
- Match the existing gomponents idiom in `internal/ui/chat/` exactly (named
  `*Props` struct + a single constructor func returning `g.Node`).

## Scope

**In scope** (the only files you should modify or create):
- `internal/ui/chat/artifact.go` (create) — the `chat.Artifact` organism
- `internal/ui/chat/artifact_test.go` (create) — organism markup tests
- `internal/web/cards.go` (modify) — rewrite `artifactWrap` to delegate to
  `chat.Artifact`; delete `artifactChip`; update the `activeArtifactCap` comment;
  drop the cluster's duplicate title in `artifactBody`
- `internal/web/assets/static/basm.css` (modify) — replace the
  `.artifact-chip*` block with the titled-window frame rules
- `internal/feature/storybook/stories_chat.go` (modify) — add `chatartifactStory()`
- `internal/feature/storybook/story.go` (modify) — register `chatartifactStory()`
  in the `stories` slice
- `internal/web/show_test.go` (modify) — assert the frame class in the SSE output
- `internal/self/knowledge.md` (modify) — describe the titled-window frame

**Out of scope** (do NOT touch, even though they look related):
- `internal/web/chatstream.go` and `internal/web/recap.go` — the `artifactWrap`
  **call sites and signatures stay identical**. You are only changing what
  `artifactWrap` returns. Do not edit these files.
- `internal/web/assets/static/basm.js` — the cap JS works unchanged; the class
  contract (`artifact` / `artifact--collapsed`) is preserved.
- `internal/web/recap.go`'s `capArtifacts` and the `ArtifactCollapsed` field —
  the cap *logic* is unchanged.
- `internal/web/artifact_cap_test.go` — it tests `capArtifacts` logic, not
  markup; it must keep passing untouched.
- The `chat.ToolRow` rendered above the artifact (its presence/redundancy for
  sidebar-summoned artifacts is a separate concern — see Maintenance notes).
- The `.k-inline` rules in basm.css (lines 1038–1042) — still used by the
  proposal / plain-inline path; leave them.
- Any change to `cards.Validate`, the card registry, or any feature card body.

## Git workflow

- A reviewer dispatched you into an isolated worktree. Work there; do NOT
  commit, push, merge, or open a PR. The reviewer handles all git operations.
- Leave `plans/README.md` alone — the reviewer maintains the index.

## Steps

### Step 1: Create the `chat.Artifact` organism

Create `internal/ui/chat/artifact.go`. Mirror `cluster.go`'s structure exactly
(named `*Props` struct + one constructor). Target shape:

```go
package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ArtifactProps configures the chat.Artifact organism — the titled "sub-window"
// frame around one in-chat artifact (a single Focus-size card, or a cluster).
// Body is pre-rendered by the caller (web layer), so internal/ui/chat imports no
// feature/cards. Collapsed is the aged-out cap state (set server-side by
// capArtifacts and client-side by balaurCapArtifacts): the body hides and only
// the title bar remains.
type ArtifactProps struct {
	Title     string // artifact name shown in the title bar; "" → "Artifact"
	Icon      string // /static/icons stem ("" → no icon)
	Collapsed bool   // aged-out: adds .artifact--collapsed (CSS hides the body)
	InnerID   string // optional id on the body div (live path's tool-card id)
	Body      g.Node // pre-rendered artifact body (a Focus card or a Cluster)
}

// Artifact frames its body as a self-contained titled window: an always-visible
// .artifact-head bar (icon + title) atop a bordered .artifact-body. The window
// edge tells the owner where one artifact ends and the next message begins.
// The enclosing #chat append is done by the gateway (endTool / renderMessages).
func Artifact(p ArtifactProps) g.Node {
	title := p.Title
	if title == "" {
		title = "Artifact"
	}
	cls := "artifact"
	if p.Collapsed {
		cls += " artifact--collapsed"
	}

	head := []g.Node{h.Class("artifact-head")}
	if p.Icon != "" {
		head = append(head, h.Img(h.Class("artifact-head-icon"),
			h.Src("/static/icons/"+p.Icon+".png"), h.Alt(""), g.Attr("decoding", "async")))
	}
	head = append(head, h.Span(h.Class("artifact-head-title"), g.Text(title)))

	body := []g.Node{h.Class("artifact-body")}
	if p.InnerID != "" {
		body = append(body, h.ID(p.InnerID))
	}
	body = append(body, p.Body)

	return h.Div(h.Class(cls), h.Header(head...), h.Div(body...))
}
```

**Verify**: `CGO_ENABLED=0 go build ./internal/ui/chat/` → exit 0.

### Step 2: Add the organism test

Create `internal/ui/chat/artifact_test.go`. Model it on the rendering pattern in
`internal/ui/chat/message_test.go` (use the same render-to-string helper that
file uses — likely `helpers_test.go`'s render function; reuse whatever
`message_test.go` / `toolrow_test.go` use, do not invent a new one).

Cover:
- **expanded**: `chat.Artifact(ArtifactProps{Title: "Quests", Icon: "scroll", Body: g.Text("BODY")})`
  → output contains `artifact-head`, `artifact-head-title`, `Quests`,
  `artifact-body`, `/static/icons/scroll.png`, `BODY`; and does **not** contain
  `artifact--collapsed`.
- **collapsed**: same with `Collapsed: true` → output contains `artifact--collapsed`.
- **no icon**: `Title: "Memory"`, no `Icon` → output contains `Memory` and does
  **not** contain `/static/icons/`.
- **empty title falls back**: `Title: ""` → output contains `Artifact`.
- **innerID**: `InnerID: "tool-3-card"` → output contains `id="tool-3-card"`.

**Verify**: `go test ./internal/ui/chat/` → all pass (including the new test).

### Step 3: Rewrite `artifactWrap` and delete `artifactChip` in `internal/web/cards.go`

Replace the `artifactChip` function (lines 182–195) and `artifactWrap` function
(lines 197–212) with a single thin adapter that delegates to `chat.Artifact`.
Keep the `artifactWrap(title, icon, collapsed, innerID, body)` signature
**identical** (both callers pass these five args):

```go
// artifactWrap frames a rendered artifact body as a self-contained titled
// "sub-window" (plan 097), delegating to the chat.Artifact organism so the
// chrome lives once in the design system. collapsed (the cap, plan 094) folds
// the window down to its title bar. innerID, when set, rides the body div
// (preserves the live path's tool-card id).
func artifactWrap(title, icon string, collapsed bool, innerID string, body template.HTML) g.Node {
	return chat.Artifact(chat.ArtifactProps{
		Title:     title,
		Icon:      icon,
		Collapsed: collapsed,
		InnerID:   innerID,
		Body:      g.Raw(string(body)),
	})
}
```

Also update the `activeArtifactCap` doc comment (line ~178): change
"older ones collapse to a static \"shown earlier\" chip (plan 094)" to
"older ones fold to their title bar (plan 094)".

Confirm `chat` and `g` (gomponents) are already imported in `cards.go` (they
are — `cards.go` already uses `chat.Cluster` and `g.Raw`). Do not add imports
that become unused; after deleting `artifactChip`, verify nothing else
references it.

**Verify**:
- `grep -rn "artifactChip\|artifact-chip" internal/` → no matches.
- `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 4: Drop the duplicate cluster title in `artifactBody`

In `internal/web/cards.go`, `artifactBody` (lines 167–175): the window header
now owns the title, so stop passing it into `chat.Cluster`. Change:

```go
_ = chat.Cluster(chat.ClusterProps{Title: title, Cards: nodes}).Render(&b)
```
to:
```go
// The artifact window header (artifactWrap) owns the title now; render the
// cluster body untitled so the heading isn't duplicated (plan 097).
_ = chat.Cluster(chat.ClusterProps{Cards: nodes}).Render(&b)
```

The `title` parameter is still used by the caller (passed to `artifactWrap` as
the window title), so the function signature is unchanged; only the
`chat.Cluster` call drops `Title`. If `title` becomes an unused parameter inside
`artifactBody` after this edit, that is fine — it stays in the signature because
callers still pass it (Go does not flag unused function parameters).

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0.

### Step 5: Replace the artifact CSS in `internal/web/assets/static/basm.css`

Replace the block at lines 1044–1054 (the `/* Artifact cap (plan 094) … */`
comment through `.artifact-chip-icon { … }`) with the titled-window frame.
**Keep** the `.k-inline` rules at lines 1038–1042 untouched. New block:

```css
/* ── In-chat artifact: titled "sub-window" frame (plan 097) ───────────────── */
/* Each summoned artifact (a Focus card or a cluster) is framed as a            */
/* self-contained window so the owner sees where one ends and the next message */
/* begins. Replaces the frameless body + collapse-only chip (plan 094).        */
.artifact {
  margin: 10px 0;
  border: 2px solid var(--parch-edge);
  background-color: var(--surface);
  background-image: var(--grain-ink);
  background-size: 4px 4px;
  box-shadow: var(--parch-bevel);
}
.chat .artifact { margin-left: var(--chat-gutter, 124px); }
.artifact-head {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  background: var(--surface-2);
  border-bottom: 2px solid var(--parch-edge);
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 700;
  text-transform: uppercase;
  letter-spacing: .06em;
  color: var(--ink);
}
.artifact-head-icon { width: 16px; height: 16px; image-rendering: pixelated; }
.artifact-body { padding: var(--space-4); }
.artifact-body .kcard { padding: 15px 18px; }
/* Aged-out (cap, plan 094): fold to the title bar; body hidden. */
.artifact--collapsed > .artifact-body { display: none; }
.artifact--collapsed .artifact-head { color: var(--ink-muted); }
.artifact--collapsed .artifact-head-title::after {
  content: " · shown earlier";
  font-weight: 400;
  opacity: .8;
}
```

**Verify**:
- `grep -n "artifact-chip\|artifact--collapsed > .k-inline" internal/web/assets/static/basm.css` → no matches.
- `grep -n "artifact-head\|.artifact-body\|.chat .artifact " internal/web/assets/static/basm.css` → matches present.
- `grep -n ".k-inline" internal/web/assets/static/basm.css` → the lines 1038–1042 block still present.

### Step 6: Add the storybook story

In `internal/feature/storybook/stories_chat.go`, add a `chatartifactStory()`
function modeled on `chatclusterStory()` (find it at the `ID: "chatcluster"`
line). `taskcards` is already imported in that file (the cluster story uses
`taskcards.TaskCard`). Target:

```go
func chatartifactStory() Story {
	sample := taskcards.TaskCard(taskcards.TaskView{
		ID: "t1", Title: "Ship the sidebar rework", Status: "open", DueLine: "due today 18:00",
	})
	return Story{
		ID: "chatartifact", Group: "Chat", Title: "Artifact", Wide: true,
		Blurb: "The titled 'sub-window' frame around one in-chat artifact (a Focus card or a cluster). " +
			"An always-visible .artifact-head bar (icon + name) tops a bordered .artifact-body, so the owner " +
			"sees where one artifact ends and the next message begins. Body is pre-rendered by the web layer; " +
			"the organism imports no feature/cards. Collapsed is the aged-out cap state (plan 094) — body hidden, " +
			"title bar kept.",
		Variants: []Variant{
			{"expanded", chat.Artifact(chat.ArtifactProps{Title: "Quests", Icon: "scroll", Body: sample})},
			{"collapsed", chat.Artifact(chat.ArtifactProps{Title: "Quests", Icon: "scroll", Collapsed: true, Body: sample})},
			{"no icon", chat.Artifact(chat.ArtifactProps{Title: "Memory", Body: sample})},
		},
		Props: []Prop{
			{"Title", "string", `""`, "Artifact name shown in the .artifact-head title bar. Empty falls back to \"Artifact\"."},
			{"Icon", "string", `""`, "/static/icons stem shown left of the title. Omit for no icon."},
			{"Collapsed", "bool", "false", "Aged-out cap state: adds .artifact--collapsed; CSS hides the body, keeps the title bar."},
			{"InnerID", "string", `""`, "Optional id on the body div — preserves the live path's tool-card id."},
			{"Body", "g.Node", "nil", "Pre-rendered artifact body (a Focus card or a chat.Cluster). The organism never renders cards itself."},
		},
		Dos: []string{
			"Pass a pre-rendered body g.Node from the web layer (cardFocusHTML / Cluster).",
			"Let the cap (capArtifacts + balaurCapArtifacts) toggle Collapsed / .artifact--collapsed.",
		},
		Donts: []string{
			"Import internal/feature or internal/cards from internal/ui/chat.",
			"Wrap proposals or plain inline cards in Artifact — those stay frameless (.k-inline).",
		},
	}
}
```

Match the exact field names of the `Story`, `Variant`, and `Prop` types as used
by `chatclusterStory()` — if any field name differs from the above (e.g.
`Variant` is `{label, node}`), copy the working shape from `chatclusterStory`.

Then register it: in `internal/feature/storybook/story.go`, the `stories` slice
literal (starts at line 53) lists `chatclusterStory(),` (line ~85). Add
`chatartifactStory(),` on the next line.

**Verify**: `CGO_ENABLED=0 go build ./...` → exit 0; `go test ./internal/feature/storybook/` → pass.

### Step 7: Update the `show_test.go` assertion

In `internal/web/show_test.go`, the "GET /ui/show/quests → 200 …" subtest
(around line 36) has:
```go
ExpectedContent: []string{
	"datastar-patch-elements",
	"selector #chat",
	"mode append",
	"quest-stack", // flat stack (ui.Focus), not the summary tile
},
```
Add `"artifact-head"` to that slice (the framed window's title-bar class now
appears in the SSE patch). Do not remove the existing entries.

**Verify**: `go test ./internal/web/ -run TestUIShow` → pass.

### Step 8: Update `internal/self/knowledge.md`

The artifact-injection-door paragraph (around line 112) ends "…SSE-appends the
rendered card tile to #chat — no navigation, no page load, no LLM." Append one
sentence describing the frame, e.g.:

> Each in-chat artifact is framed as a self-contained titled sub-window
> (chat.Artifact: an .artifact-head bar with icon + name atop a bordered
> .artifact-body), so the owner can see where one artifact ends; aged-out
> artifacts (the cap, plan 094) fold to their title bar.

Keep the edit to a sentence or two; do not restructure the surrounding prose.

**Verify**: `grep -n "sub-window\|artifact-head\|chat.Artifact" internal/self/knowledge.md` → match present.

### Step 9: Full verification sweep

Run, in order:
1. `gofmt -l internal/` → no output.
2. `go vet ./...` → exit 0.
3. `CGO_ENABLED=0 go build ./...` → exit 0.
4. `go test ./...` → all pass.
5. `git diff --check` → no whitespace errors.

## Test plan

- **New** `internal/ui/chat/artifact_test.go` — organism markup, per Step 2
  (expanded / collapsed / no-icon / empty-title fallback / innerID). Model after
  `internal/ui/chat/message_test.go`.
- **Modified** `internal/web/show_test.go` — the existing quests-show subtest
  gains a `"artifact-head"` content assertion (Step 7), proving the live SSE
  path emits the framed window.
- **Unchanged, must still pass** `internal/web/artifact_cap_test.go` — the
  cap *logic* (`capArtifacts`) is untouched; if any of its tests fail, you broke
  something out of scope — STOP.
- Verification: `go test ./...` → all pass, including the new organism test.

## Done criteria

Machine-checkable. ALL must hold:

- [ ] `gofmt -l internal/` → no output
- [ ] `go vet ./...` → exit 0
- [ ] `CGO_ENABLED=0 go build ./...` → exit 0
- [ ] `go test ./...` → all pass; `internal/ui/chat/artifact_test.go` exists and passes
- [ ] `grep -rn "artifactChip\|artifact-chip" internal/` → no matches
- [ ] `grep -n "artifact-head" internal/web/assets/static/basm.css internal/ui/chat/artifact.go` → matches in both
- [ ] `grep -n "chatartifactStory" internal/feature/storybook/` → defined in stories_chat.go AND registered in story.go
- [ ] `git diff --check` → clean
- [ ] Only the 8 in-scope files are modified/created (`git status` shows nothing else)

## STOP conditions

Stop and report back (do not improvise) if:

- The drift check shows any in-scope file changed since `58d7182` and the
  "Current state" excerpts no longer match the live code.
- `artifactWrap` turns out to have callers other than `chatstream.go:222` and
  `recap.go:206` (grep `artifactWrap` before editing) — a third caller may pass
  different args and would break with a signature you can't see here.
- The `Story` / `Variant` / `Prop` struct fields in storybook differ enough that
  the Step 6 literal won't compile even after copying `chatclusterStory`'s shape.
- `internal/web/artifact_cap_test.go` fails after your changes.
- A verification fails twice after a reasonable fix attempt.
- The change appears to require editing any out-of-scope file (especially
  `chatstream.go`, `recap.go`, or `basm.js`).

## Maintenance notes

For whoever owns this code next:

- **The cap is split across three places** that must agree on the class
  contract `.artifact` / `.artifact--collapsed`: server `capArtifacts`
  (`recap.go`), client `balaurCapArtifacts` (`basm.js`), and the CSS
  (`basm.css`). This plan touches only the CSS visual; if you ever rename the
  collapsed class, change all three.
- **Reviewer should scrutinize**: that proposals and plain inline cards still
  render frameless (bare `.k-inline`, not the window) — the `artTitle == ""` /
  `mv.ArtifactTitle != ""` gate in the two callers is the guard. And that the
  cluster path no longer double-renders its title (Step 4).
- **Deferred (out of this plan), flag to the user if it still looks off**:
  - The `chat.ToolRow` rendered *above* the window. For sidebar-summoned
    artifacts (the `/ui/show` door, where there is no real "tool" action), the
    tool row + window header can read as redundant. Suppressing the tool row for
    door-injected artifacts (vs. agent `card_show`) is a clean follow-up but
    touches `messageViews` / `renderMessages` logic — out of scope here.
  - "More granularity" of dense Focus surfaces (slicing big managers the way
    plan 095 sliced Knowledge into per-category cards) is a separate direction,
    not this presentation-only frame.
