# Plan 115: Live dialogue choices render via a new `chat.Choices` gomponents organism instead of the `chat-choices` template

> **Executor instructions**: Follow this plan step by step. Run every
> verification command and confirm its expected result before moving on. If a
> "STOP conditions" item occurs, stop and report — do not improvise. When done,
> update this plan's status row in `plans/readme.md` unless a reviewer told you
> they maintain the index.
>
> **Drift check (run first)**: `git diff --stat ea79dae..HEAD -- internal/web/chatstream.go internal/web/chat.go internal/ui/chat internal/feature/storybook web/templates/chat-messages.html`
> If any in-scope file changed since this plan was written, compare the "Current
> state" excerpts against the live code; on a mismatch, treat it as a STOP
> condition.

## Status

- **Priority**: P2
- **Effort**: S–M
- **Risk**: LOW
- **Depends on**: none (independent of 111–114; all of 111–115 must land before 116/117)
- **Category**: migration / tech-debt
- **Planned at**: commit `0dd2457`, 2026-06-19 — **refreshed 2026-06-22 against `ea79dae`; see "## Refresh" below**

## Refresh (2026-06-22, against `ea79dae`)

Still **valid and unstarted — zero drift** in the in-scope code: `chat-choices`
`ExecuteTemplate` → `chatstream.go:262`, `appendChoices` `chatstream.go:255-267`,
`choicesView` `chat.go:18`, `tools.Choice` `choices.go:20`, `renderNode`
`chatstream.go:77`, `RemoveElement(".choices")` `chatstream.go:120` — all exact.
The `chat.Choices` organism + story are built NEW (no `choices.go` in
`internal/ui/chat` yet — confirmed). Story registry: `chattoolrowStory()` still at
`story.go:82` (insert the new story right after it). `chat.go` lost an unrelated
`cardURL` helper (plan 124) — lines unaffected. Done-criteria greps hold
(ExecuteTemplate in chatstream.go = 1, "chat-choices" = 2 today; both → 0 after).

## Why this matters

`chat-choices` is the **only** fragment in `web/templates/chat-messages.html`
still executed at runtime — every other fragment there (the `chat-msg-*` /
`chat-balaur-*` / `chat-tool-row` markup) is already dead, ported to
`chat.Message` / `chat.ToolRow` / `chat.MessageBody`. Porting `chat-choices` to a
new `chat.Choices` organism removes the last `ExecuteTemplate` caller in the live
chat stream and lets plan 117 delete `chat-messages.html`. Unlike the chatbar,
this IS a reusable chat-surface organism, so it lives in `internal/ui/chat` with
a storybook story, exactly like `chat.Message` and `chat.ToolRow`.

## Current state

- `web/templates/chat-messages.html` defines `chat-choices` (data = `choicesView`):
  ```html
  {{define "chat-choices"}}
  <div class="choices" id="choices-{{.Nonce}}">
    <div class="choices-panel">
      <div class="choices-kicker">{{.Prompt}}</div>
      {{range $i, $c := .Choices}}
      <button class="choice" type="button" data-label="{{$c.Label}}"
              data-on:click="$message = el.dataset.label; @post('/ui/chat')">
        <span class="choice-key">{{addOne $i}}</span>
        <span class="choice-label">{{$c.Label}}</span>
        {{with $c.Hint}}<span class="choice-hint">{{.}}</span>{{end}}
      </button>
      {{end}}
    </div>
    <figure class="portrait">
      <span class="balaur-avatar balaur-avatar-soul" data-kind="soul" aria-hidden="true"><img src="{{.SoulAvatarURL}}" alt="" decoding="async"></span>
      <figcaption class="who">{{.OwnerName}}</figcaption>
    </figure>
  </div>
  {{end}}
  ```
  Note `addOne $i` renders a **1-based** index (`$i` is 0-based; the key shows
  `i+1`). `{{with $c.Hint}}` omits the hint span when `Hint` is empty.

- `internal/web/chat.go:18-24` — the payload:
  ```go
  type choicesView struct {
  	Prompt        string
  	Nonce         string // unique per render, used for element IDs
  	Choices       []tools.Choice
  	SoulAvatarURL string
  	OwnerName     string
  }
  ```
  `tools.Choice` (`internal/tools/choices.go:20`) = `{Label string; Hint string}`.

- `internal/web/chatstream.go:255-267` — `appendChoices` executes the template:
  ```go
  func (s *chatStream) appendChoices(prompt string, choices []tools.Choice) {
  	cv := choicesView{
  		Prompt: prompt, Nonce: newNonce(), Choices: choices,
  		SoulAvatarURL: s.soulURL, OwnerName: s.ownerName,
  	}
  	var b strings.Builder
  	if err := s.h.tmpl.ExecuteTemplate(&b, "chat-choices", cv); err != nil {
  		s.h.app.Logger().Warn("chat fragment render failed", "fragment", "chat-choices", "err", err)
  		return
  	}
  	_ = s.sse.PatchElements(b.String(), datastar.WithSelectorID("chat"), datastar.WithModeAppend())
  }
  ```

- **Conventions to match** — the existing chat organisms are the exact template:
  - `internal/ui/chat/message.go` and `toolrow.go`: package `chat`, "Class-only:
    all chat CSS lives in basm.css", imports only `g "maragu.dev/gomponents"` and
    `h "maragu.dev/gomponents/html"`. A props struct + a constructor returning
    `g.Node`. **It must not import `internal/tools`** (layering) — define a local
    `ChoiceItem` struct and have the web caller map `tools.Choice` → it.
  - Storybook story shape: `internal/feature/storybook/stories_chat.go`
    (`chatmessageStory`, `chattoolrowStory`) build a `storybook.Story` with
    `Variants`, `Props`, `Dos`, `Donts`; the registry list is
    `internal/feature/storybook/story.go:53-108` (`chattoolrowStory()` is at line 82).
  - The stream renders chat nodes via `s.renderNode(n)` then `PatchElements` —
    `chatstream.go:77` `renderNode`, `:87` `appendNode`.

## Commands you will need

| Purpose | Command | Expected on success |
|---------|---------|---------------------|
| Build (CGO-free) | `CGO_ENABLED=0 go build ./...` | exit 0 |
| Vet | `go vet ./...` | exit 0 |
| Tests | `go test ./...` | all pass, exit 0 |
| Format check | `gofmt -l internal/` | empty output |
| Whitespace | `git diff --check` | no output |

Sandbox note: in a TLS-intercepting sandbox (Hyperagent), Go commands need the
GOPROXY shim — see `docs/hyperagent-sandbox.md`.

## Scope

**In scope**:
- `internal/ui/chat/choices.go` (create)
- `internal/ui/chat/choices_test.go` (create)
- `internal/feature/storybook/stories_chat.go` (add `chatchoicesStory`)
- `internal/feature/storybook/story.go` (register `chatchoicesStory()`)
- `internal/web/chatstream.go` (repoint `appendChoices`)

**Out of scope** (do NOT touch):
- `web/templates/chat-messages.html` — plan 117 deletes it.
- The `chat-msg-*` dead fragments — they're already replaced; plan 117 deletes them.
- The `html/template` import / `endTool`'s `template.HTML` param in `chatstream.go`
  — that's the bridge, removed in plan 116. **This plan leaves the import in place.**
- `internal/tools/choices.go` — read-only; do not change `tools.Choice`.

## Git workflow

- Branch: `improve/115-chat-choices-gomponents`.
- One commit; conventional message, e.g.
  `feat(ui/chat): Choices organism; render dialogue choices via gomponents (plan 115)`.
- Do NOT push or open a PR unless instructed.

## Steps

### Step 1: Create the `chat.Choices` organism

Create `internal/ui/chat/choices.go` (faithful port; 1-based key, hint omitted
when empty, verbatim `data-on:click`):
```go
package chat

import (
	"strconv"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ChoiceItem is one selectable dialogue reply.
type ChoiceItem struct {
	Label string
	Hint  string // optional mono hint
}

// ChoicesProps configures the live dialogue-choice panel.
type ChoicesProps struct {
	Prompt        string
	Nonce         string // unique per render — element id is choices-<Nonce>
	OwnerName     string
	SoulAvatarSrc string
	Choices       []ChoiceItem
}

// Choices renders the live dialogue-choice panel: a kicker + numbered choice
// buttons beside the owner portrait. Each button writes its label into the
// $message signal and @posts the next turn; the stream removes the panel after
// a choice is taken (a choice can be used once). Port of the chat-choices template.
func Choices(p ChoicesProps) g.Node {
	buttons := make([]g.Node, 0, len(p.Choices)+1)
	buttons = append(buttons, h.Div(h.Class("choices-kicker"), g.Text(p.Prompt)))
	for i, c := range p.Choices {
		kids := []g.Node{
			h.Class("choice"), h.Type("button"),
			g.Attr("data-label", c.Label),
			g.Attr("data-on:click", "$message = el.dataset.label; @post('/ui/chat')"),
			h.Span(h.Class("choice-key"), g.Text(strconv.Itoa(i+1))),
			h.Span(h.Class("choice-label"), g.Text(c.Label)),
		}
		if c.Hint != "" {
			kids = append(kids, h.Span(h.Class("choice-hint"), g.Text(c.Hint)))
		}
		buttons = append(buttons, h.Button(kids...))
	}
	return h.Div(h.Class("choices"), h.ID("choices-"+p.Nonce),
		h.Div(h.Class("choices-panel"), g.Group(buttons)),
		h.Figure(h.Class("portrait"),
			h.Span(h.Class("balaur-avatar balaur-avatar-soul"),
				g.Attr("data-kind", "soul"), g.Attr("aria-hidden", "true"),
				h.Img(h.Src(p.SoulAvatarSrc), h.Alt(""), g.Attr("decoding", "async"))),
			h.FigCaption(h.Class("who"), g.Text(p.OwnerName)),
		),
	)
}
```

**Verify**: `go build ./internal/ui/chat/` → exit 0.

### Step 2: Add the storybook story

In `internal/feature/storybook/stories_chat.go`, add a `chatchoicesStory()`
modeled on `chattoolrowStory()` (a couple of `Variant`s — e.g. one with hints,
one without — a `Props` table, and Do/Don't notes). Then register it in
`internal/feature/storybook/story.go` by inserting `chatchoicesStory(),` right
after `chattoolrowStory(),` in the `stories` slice (line ~82).

**Verify**:
- `go build ./internal/feature/storybook/` → exit 0
- `go test ./internal/feature/storybook/...` → pass (the `story_test.go` registry
  test confirms every registered story id resolves).

### Step 3: Repoint `appendChoices`

In `internal/web/chatstream.go`, replace the template execution with the organism:
```go
func (s *chatStream) appendChoices(prompt string, choices []tools.Choice) {
	items := make([]chat.ChoiceItem, len(choices))
	for i, c := range choices {
		items[i] = chat.ChoiceItem{Label: c.Label, Hint: c.Hint}
	}
	node := chat.Choices(chat.ChoicesProps{
		Prompt: prompt, Nonce: newNonce(), OwnerName: s.ownerName,
		SoulAvatarSrc: s.soulURL, Choices: items,
	})
	_ = s.sse.PatchElements(s.renderNode(node), datastar.WithSelectorID("chat"), datastar.WithModeAppend())
}
```
`chatstream.go` already imports `chat "github.com/alexradunet/balaur/internal/ui/chat"`
and has the `renderNode` method (`chatstream.go:77`). The `choicesView` struct in
`chat.go` is now unused unless something else references it — grep
`choicesView`; if `appendChoices` was its only user, delete the struct from
`chat.go:18-24` (and remove the `tools` import there only if it becomes unused).

**Verify**: `go build ./internal/web/` → exit 0.

### Step 4: Build, vet, test

**Verify**:
- `CGO_ENABLED=0 go build ./...` → exit 0
- `go vet ./...` → exit 0
- `go test ./...` → all pass, exit 0
- `gofmt -l internal/` → empty

## Test plan

- Create `internal/ui/chat/choices_test.go` (model after
  `internal/ui/chat/message_test.go` if present, else render to a
  `strings.Builder` and assert with `strings.Contains`):
  - two choices, second with a hint → output contains `id="choices-<nonce>"`,
    `class="choices-panel"`, exactly two `class="choice"` buttons, the keys `1`
    and `2`, one `choice-hint`, the owner name, and the verbatim
    `data-on:click="$message = el.dataset.label; @post('/ui/chat')"`.
  - a choice with empty `Hint` → no `choice-hint` span for it.
- The storybook `story_test.go` covers the new story's registration.
- `templates_test.go` still parses `chat-messages.html` and may execute
  `chat-choices` — leave it; plan 117 removes it with the file.
- Verification: `go test ./internal/ui/chat/... ./internal/feature/storybook/... ./internal/web/...` → all pass.

## Done criteria

- [ ] `CGO_ENABLED=0 go build ./...` exits 0
- [ ] `go vet ./...` exits 0
- [ ] `go test ./...` exits 0; `internal/ui/chat/choices_test.go` exists and passes
- [ ] `gofmt -l internal/` prints nothing
- [ ] `git diff --check` prints nothing
- [ ] `grep -rn 'ExecuteTemplate' internal/web/chatstream.go` returns **no** matches
- [ ] `grep -rn '"chat-choices"' internal/web/*.go` (excluding `_test.go`) returns **no** matches
- [ ] `chatchoicesStory` is registered in `internal/feature/storybook/story.go`
- [ ] `web/templates/chat-messages.html` still exists (untouched)
- [ ] No files outside the in-scope list are modified (`git status`)
- [ ] `plans/readme.md` status row updated

## STOP conditions

Stop and report (do not improvise) if:

- The "Current state" excerpts don't match the live code (drift since `0dd2457`).
- `choicesView` / `tools.Choice` no longer have the fields above.
- `appendChoices` is NOT the only consumer of `choicesView` and you can't safely
  remove the struct (leave it and note it instead).

## Maintenance notes

- After 111–115 land, **zero** `ExecuteTemplate` calls remain and all 11
  `web/templates/*.html` are dead; plan 117 deletes them.
- `chat.Choices` is now the single source of choice-panel markup, with a
  storybook story — future changes happen there, not in a template.
- Reviewer: confirm the `$message = el.dataset.label; @post('/ui/chat')` click
  expression and the `choices-<nonce>` id are byte-identical — the stream removes
  the panel with `RemoveElement(".choices")` (`chatstream.go:120`), which keys off
  the `.choices` class.
