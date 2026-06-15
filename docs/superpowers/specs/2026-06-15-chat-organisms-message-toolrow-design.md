# Chat Organisms — Message + ToolRow (catalog slice) — Design

**Status:** Design approved (catalog-only scope) · **Date:** 2026-06-15

## Context

The gomponents atomic catalog is complete (22 atoms in `internal/ui`, all shown in
the routed storybook). The next phase is **organisms** — components that compose
atoms and that will eventually *replace* the live `html/template` chat surface.
This slice builds the first two, **catalog-only**: `Message` and `ToolRow`, shown
as storybook stories. It does NOT touch the live chat streaming path
(`internal/web/chatstream.go`, `web/templates/chat-messages.html`) — a later slice
wires the organisms into the gateway.

The chat CSS already lives in `internal/web/assets/static/basm.css` (basm.css IS
the Hearthwood export): `.msg`, `.msg-user`/`.msg-balaur`, `.msg-with-avatar`,
`.portrait`, `.who`, `.msg-main`, `.body`, `.msg-pending`, `.thinking-dots`,
`.msg-tool`, `.tool-icon`, `--chat-gutter`. So the organisms are **class-only** —
no new CSS — exactly like the `Tabs` atom.

## Decisions (locked)

1. **Catalog-only scope.** Build `chat.Message` + `chat.ToolRow` + a storybook
   "Chat" story group. The live `chat-msg-*` templates and `chatstream.go` are
   untouched. Wiring is a separate, later slice.
2. **New package `internal/ui/chat`.** The decided home for chat organisms
   (per AGENTS.md architecture). It imports `internal/ui` to compose atoms.
3. **Compose `ui.Avatar`.** `Message` renders its portrait via `ui.Avatar`, not
   hand-rolled markup. `ui.Avatar` emits a harmless *superset* of the live
   template's avatar span (it adds `data-state` + `style="--avatar-size:54px"`);
   visually identical, and its `State:"thinking"` drives the pending glow via the
   `.balaur-avatar[data-state="thinking"]` selector — cleaner than the live
   template's `balaur-avatar-live` class. Golden tests assert the organism's
   actual output (the superset), not the live template's byte-for-byte markup.

## Architecture

### Reference: the live markup these mirror (from `web/templates/chat-messages.html`)

```html
<!-- user -->
<div class="msg msg-user msg-with-avatar">
  <figure class="portrait">
    <span class="balaur-avatar balaur-avatar-soul" data-kind="soul" aria-hidden="true"><img src="{SoulAvatarURL}" alt="" decoding="async"></span>
    <figcaption class="who">{OwnerName}</figcaption>
  </figure>
  <div class="msg-main"><div class="body">{Content}</div></div>
</div>
<!-- balaur (assistant); pending adds msg-pending + thinking-dots body -->
<div class="msg msg-balaur msg-with-avatar">
  <figure class="portrait">
    <span class="balaur-avatar balaur-avatar-balaur" data-kind="balaur" aria-hidden="true"><img src="{BalaurAvatarURL}" alt="" decoding="async"></span>
    <figcaption class="who">{WhoLabel}{ · Origin}</figcaption>
  </figure>
  <div class="msg-main"><div class="body">{Content}</div></div>
</div>
<!-- tool -->
<div class="msg msg-tool">
  <div class="who"><img class="tool-icon" src="/static/icons/{icon}.png" alt="" aria-hidden="true">tool · {Tool}</div>
  <div class="body">{Content}</div>
</div>
```

### `chat.Message` — `internal/ui/chat/message.go`

```go
type MessageProps struct {
	Role      string // "user" | "balaur" (anything != "user" is balaur)
	Who       string // display name; "" → "You" (user) / "Balaur" (balaur)
	Origin    string // optional; balaur only, rendered " · {Origin}" after the name
	AvatarSrc string // portrait image URL
	Content   string // body text
	Pending   bool   // assistant generating: adds msg-pending + thinking-dots when Content==""
}
func Message(p MessageProps) g.Node
```

Renders `<div class="msg msg-user|msg-balaur msg-with-avatar[ msg-pending]">` →
`<figure class="portrait">` holding `ui.Avatar` (Kind `soul` for user else
`balaur`; State `thinking` when Pending && balaur, else `idle`; empty Alt →
decorative) + `<figcaption class="who">` (name, plus ` · Origin` for a balaur with
an Origin) → `<div class="msg-main"><div class="body">…</div></div>`. The body is
`<span class="thinking thinking-dots">thinking</span>` when `Pending && Content==""`,
else the `Content` text.

### `chat.ToolRow` — `internal/ui/chat/toolrow.go`

```go
type ToolRowProps struct {
	Tool    string // tool name, rendered "tool · {Tool}"
	Icon    string // /static/icons name (composed via ui.Icon)
	Content string // result text
}
func ToolRow(p ToolRowProps) g.Node
```

Renders `<div class="msg msg-tool"><div class="who">{ui.Icon(Icon)}tool · {Tool}</div>
<div class="body">{Content}</div></div>`. `ui.Icon` emits the `.tool-icon` img
(`/static/icons/{Icon}.png`).

### Storybook — new "Chat" group

`internal/feature/storybook/storybook.go` gains `chatMessageCanvas()` and
`chatToolRowCanvas()` (it already imports `ui`; add `internal/ui/chat`). The
registry (`story.go`) gains two entries under a new "Chat" group:
```go
{"chatmessage", "Chat", "Message", chatMessageCanvas},
{"chattoolrow", "Chat", "ToolRow", chatToolRowCanvas},
```
`chatMessageCanvas` shows a balaur turn, a user turn, and a pending turn;
`chatToolRowCanvas` shows a couple of tool rows. Fixtures use a real avatar/crest
URL under `/static/` so the portrait renders.

## Testing

- `internal/ui/chat` gets its own `helpers_test.go` with a `render(t, node) string`
  (same shape as `internal/ui/helpers_test.go` — the packages don't share test
  helpers).
- `Message` golden tests: a balaur message renders `class="msg msg-balaur
  msg-with-avatar"`, the `.portrait`/`.who`/`.msg-main`/`.body` structure, the
  composed `balaur-avatar balaur-avatar-balaur` span, and ` · Origin` when set; a
  user message renders `msg-user` + `balaur-avatar-soul` + the bare name; a pending
  balaur message adds `msg-pending` and the `thinking thinking-dots` body.
- `ToolRow` golden test: `class="msg msg-tool"`, the `.who` line with the
  `tool-icon` img and `tool · {Tool}`, and the `.body` result.
- Story registry test (existing) covers the two new stories automatically; handler
  smoke is covered by the existing storybook route tests.

## Known limitations / deferred

- **Not wired into live chat.** `chatstream.go`/`chat.go` and the `chat-msg-*`
  templates are untouched this slice. When wired later, the avatar markup gains
  `data-state`/`--avatar-size` (the ui.Avatar superset) — budget golden/template
  updates then, and verify the SSE morph targets (`BubbleID`/`BodyID`) still align.
- Inline cards (`.k-inline` after a tool row), DialogueChoices, and the Composer
  are separate organisms, not in this slice.
- TypewriterText reveal is dropped for v1 (render final text), per the prior
  brainstorm.
