# Gomponents Organisms — DialogueChoices + MessageDraft Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add two more chat organisms — `chat.DialogueChoices` and `chat.MessageDraft` — to `internal/ui/chat`, composing atoms, registered as storybook stories under the existing "Chat" group. Catalog-only (static; the Datastar wiring is a later slice).

**Architecture:** Both are class-only (the chat CSS already exists in basm.css). They follow the LIVE template structure (the html/template they will eventually replace), compose `ui.Avatar` (soul portrait) and — for MessageDraft — `ui.Button` (the Speak button). Continues the Message/ToolRow organism pattern.

**Tech Stack:** Go, gomponents (`h.Form`/`h.Textarea`/`h.Rows`/`h.Name`/`h.Placeholder`/`h.Figure`/`h.FigCaption` all exist in v1.3.0), `internal/ui` atoms.

**Conventions:** package `chat` uses QUALIFIED `g`/`h` imports (NO dot-import). NO new CSS (all chat classes already in basm.css). Tests are `package chat_test` using the existing package-local `render(t, node)` helper in `internal/ui/chat/helpers_test.go`. Stories: append canvas funcs to `internal/feature/storybook/storybook.go` and `Story` entries to `internal/feature/storybook/story.go`. Story canvases wrap content in `h.Div(h.Class("chat"), …)` so `--portrait-size` resolves (the lesson from chat.Message). After each task: `go test ./...`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`. If `git status` shows any non-task file modified, `git checkout --` it.

Verified facts: `ui.Avatar(ui.AvatarProps{Src, Kind:"soul"})` renders `<span class="balaur-avatar balaur-avatar-soul" data-kind="soul" data-state="idle" style="--avatar-size:54px" aria-hidden="true"><img src="{Src}" alt="" decoding="async"></span>`. `ui.Button(ui.ButtonProps{Size:"sm"}, children...)` renders `<button class="btn btn-primary btn-sm" {children-attrs}>{children-text}</button>` (default variant primary; class first, then children in order). Classes `.choices`, `.choices-panel`, `.choices-kicker`, `.choice`, `.choice-key`, `.choice-label`, `.choice-hint`, `.msg`, `.msg-user`, `.msg-draft`, `.msg-main`, `.chat-form`, `.msg-draft-foot`, `.msg-draft-hint`, `.portrait`, `.who`, `.chat` all exist in basm.css. `/static/crest.png` exists. The Chat story group's last entry is `{"chattoolrow", "Chat", "ToolRow", chatToolRowCanvas}`.

---

## File Structure

- **Create** `internal/ui/chat/choices.go`+`_test.go`, `internal/ui/chat/draft.go`+`_test.go`.
- **Modify** `internal/feature/storybook/storybook.go` (canvas funcs) + `internal/feature/storybook/story.go` (two Chat stories).

---

## Task 1: `chat.DialogueChoices` + story

**Files:** Create `internal/ui/chat/choices.go`, `internal/ui/chat/choices_test.go`; modify `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/chat/choices_test.go`:
```go
package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestDialogueChoices(t *testing.T) {
	got := render(t, chat.DialogueChoices(chat.ChoicesProps{
		Prompt:    "How should I log this?",
		AvatarSrc: "/static/crest.png",
		Choices: []chat.Choice{
			{Label: "As a quick note", Hint: "1 line"},
			{Label: "As a full entry"},
		},
	}))
	for _, want := range []string{
		`<div class="choices">`,
		`<div class="choices-panel"><div class="choices-kicker">How should I log this?</div>`,
		`<button class="choice" type="button"><span class="choice-key">1</span><span class="choice-label">As a quick note</span><span class="choice-hint">1 line</span></button>`,
		`<button class="choice" type="button"><span class="choice-key">2</span><span class="choice-label">As a full entry</span></button>`,
		`<figure class="portrait">`,
		`class="balaur-avatar balaur-avatar-soul"`,
		`<figcaption class="who">You</figcaption>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("choices missing %q in: %s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/chat/ -run TestDialogueChoices -v` — Expected: FAIL (`undefined: chat.DialogueChoices`/`chat.ChoicesProps`/`chat.Choice`).

- [ ] **Step 3: Implement the organism**

Create `internal/ui/chat/choices.go`:
```go
package chat

import (
	"strconv"

	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// Choice is one dialogue option: a Label (the spoken reply) and an optional Hint.
type Choice struct {
	Label string
	Hint  string
}

// ChoicesProps configures a DialogueChoices panel. Prompt is the kicker question
// (omitted when empty); Choices are the numbered options; AvatarSrc + Who are the
// owner's mirrored soul portrait (Who defaults "You").
type ChoicesProps struct {
	Prompt    string
	Choices   []Choice
	AvatarSrc string
	Who       string
}

// DialogueChoices renders the chat choice panel: a kicker over numbered choice
// buttons, beside the owner's soul portrait. Static (catalog) — the click action
// is wired by the gateway later.
func DialogueChoices(p ChoicesProps) g.Node {
	who := p.Who
	if who == "" {
		who = "You"
	}
	panel := []g.Node{h.Class("choices-panel")}
	if p.Prompt != "" {
		panel = append(panel, h.Div(h.Class("choices-kicker"), g.Text(p.Prompt)))
	}
	for i, c := range p.Choices {
		btn := []g.Node{
			h.Class("choice"), h.Type("button"),
			h.Span(h.Class("choice-key"), g.Text(strconv.Itoa(i+1))),
			h.Span(h.Class("choice-label"), g.Text(c.Label)),
		}
		if c.Hint != "" {
			btn = append(btn, h.Span(h.Class("choice-hint"), g.Text(c.Hint)))
		}
		panel = append(panel, h.Button(btn...))
	}
	return h.Div(h.Class("choices"),
		h.Div(panel...),
		h.Figure(h.Class("portrait"),
			ui.Avatar(ui.AvatarProps{Src: p.AvatarSrc, Kind: "soul"}),
			h.FigCaption(h.Class("who"), g.Text(who)),
		),
	)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/chat/ -run TestDialogueChoices -v` — Expected: PASS.

- [ ] **Step 5: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append:
```go

func dialogueChoicesCanvas() g.Node {
	return section("DialogueChoices",
		h.Div(h.Class("chat"),
			chat.DialogueChoices(chat.ChoicesProps{
				Prompt:    "How should I log this?",
				AvatarSrc: "/static/crest.png",
				Choices: []chat.Choice{
					{Label: "As a quick note", Hint: "1 line"},
					{Label: "As a full journal entry"},
					{Label: "Don't save it", Hint: "skip"},
				},
			}),
		),
	)
}
```
In `internal/feature/storybook/story.go`, add immediately AFTER the `{"chattoolrow", "Chat", "ToolRow", chatToolRowCanvas},` line:
```go
	{"dialoguechoices", "Chat", "DialogueChoices", dialogueChoicesCanvas},
```

- [ ] **Step 6: Verify + commit**

```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/chat/ -run TestDialogueChoices && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
git status --short
```
Expected: PASS, suite green, build clean. If `git status --short` shows any file other than `internal/ui/chat/choices.go`, `internal/ui/chat/choices_test.go`, `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`, do NOT stage it — `git checkout -- <file>`. Then:
```bash
git add internal/ui/chat/choices.go internal/ui/chat/choices_test.go internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add chat.DialogueChoices organism + storybook story\n\nchat.DialogueChoices(ChoicesProps) — the chat choice panel (kicker + numbered\n.choice buttons) beside the owner soul portrait (composes ui.Avatar). Class-only.\nRegistered under the Chat group.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: `chat.MessageDraft` + story

**Files:** Create `internal/ui/chat/draft.go`, `internal/ui/chat/draft_test.go`; modify `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/chat/draft_test.go`:
```go
package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestMessageDraft(t *testing.T) {
	got := render(t, chat.MessageDraft(chat.DraftProps{
		AvatarSrc:   "/static/crest.png",
		Placeholder: "Speak; I am listening.",
	}))
	for _, want := range []string{
		`<div class="msg msg-user msg-draft">`,
		`class="balaur-avatar balaur-avatar-soul"`,
		`<figcaption class="who">You</figcaption>`,
		`<div class="msg-main"><form class="chat-form">`,
		`<textarea name="message" placeholder="Speak; I am listening." rows="2" autocomplete="off"></textarea>`,
		`<div class="msg-draft-foot"><span class="msg-draft-hint">enter to speak · shift+enter for a new line</span><button class="btn btn-primary btn-sm" type="submit">Speak</button></div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("draft missing %q in: %s", want, got)
		}
	}
}

func TestMessageDraftOverrides(t *testing.T) {
	got := render(t, chat.MessageDraft(chat.DraftProps{AvatarSrc: "/static/crest.png", Placeholder: "x", Who: "Alex", Hint: "go", SendLabel: "Send", Value: "draft text"}))
	if !strings.Contains(got, `<figcaption class="who">Alex</figcaption>`) {
		t.Errorf("Who override missing: %s", got)
	}
	if !strings.Contains(got, `>draft text</textarea>`) {
		t.Errorf("Value should be the textarea content: %s", got)
	}
	if !strings.Contains(got, `<span class="msg-draft-hint">go</span>`) || !strings.Contains(got, `type="submit">Send</button>`) {
		t.Errorf("Hint/SendLabel overrides missing: %s", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/chat/ -run TestMessageDraft -v` — Expected: FAIL (`undefined: chat.MessageDraft`/`chat.DraftProps`).

- [ ] **Step 3: Implement the organism**

Create `internal/ui/chat/draft.go`:
```go
package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// DraftProps configures a MessageDraft (the owner's editable draft bubble). All
// optional except AvatarSrc/Placeholder in practice: Who defaults "You", Hint and
// SendLabel default to the live copy, Value is the textarea content.
type DraftProps struct {
	AvatarSrc   string
	Who         string
	Placeholder string
	Hint        string
	SendLabel   string
	Value       string
}

// MessageDraft renders the owner's draft bubble: a soul portrait beside a parchment
// bubble holding the textarea form + a foot (hint + Speak button). Static (catalog)
// — the form's Datastar bindings are wired by the gateway later.
func MessageDraft(p DraftProps) g.Node {
	who := p.Who
	if who == "" {
		who = "You"
	}
	hint := p.Hint
	if hint == "" {
		hint = "enter to speak · shift+enter for a new line"
	}
	send := p.SendLabel
	if send == "" {
		send = "Speak"
	}
	return h.Div(h.Class("msg msg-user msg-draft"),
		h.Figure(h.Class("portrait"),
			ui.Avatar(ui.AvatarProps{Src: p.AvatarSrc, Kind: "soul"}),
			h.FigCaption(h.Class("who"), g.Text(who)),
		),
		h.Div(h.Class("msg-main"),
			h.Form(h.Class("chat-form"),
				h.Textarea(h.Name("message"), h.Placeholder(p.Placeholder), h.Rows("2"), g.Attr("autocomplete", "off"), g.Text(p.Value)),
				h.Div(h.Class("msg-draft-foot"),
					h.Span(h.Class("msg-draft-hint"), g.Text(hint)),
					ui.Button(ui.ButtonProps{Size: "sm"}, h.Type("submit"), g.Text(send)),
				),
			),
		),
	)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/chat/ -run TestMessageDraft -v` — Expected: PASS (both).

- [ ] **Step 5: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append:
```go

func messageDraftCanvas() g.Node {
	return section("MessageDraft",
		h.Div(h.Class("chat"),
			chat.MessageDraft(chat.DraftProps{
				AvatarSrc:   "/static/crest.png",
				Placeholder: "Speak; I am listening.",
			}),
		),
	)
}
```
In `internal/feature/storybook/story.go`, add immediately AFTER the `{"dialoguechoices", "Chat", "DialogueChoices", dialogueChoicesCanvas},` line:
```go
	{"messagedraft", "Chat", "MessageDraft", messageDraftCanvas},
```

- [ ] **Step 6: Verify + commit**

```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/chat/ -run TestMessageDraft && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./... && go vet ./...
git status --short
```
Expected: PASS, suite green, build+vet clean. Stage only the four files, then:
```bash
git add internal/ui/chat/draft.go internal/ui/chat/draft_test.go internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add chat.MessageDraft organism + storybook story\n\nchat.MessageDraft(DraftProps) — the owner draft bubble: soul portrait + parchment\nform (textarea + foot hint + Speak button, composes ui.Avatar/ui.Button).\nClass-only. Registered under the Chat group.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification (controller — visual)

- [ ] `go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check` — green.
- [ ] Chat group now lists Message / ToolRow / DialogueChoices / MessageDraft; `/storybook/dialoguechoices` and `/storybook/messagedraft` render 200 (content-assert `choices-panel` / `msg-draft`, not just status — stale-debug-binary guard).
- [ ] Screenshot both (Hearthwood): DialogueChoices shows the gold-corner-bracket panel with numbered choices + mirrored portrait; MessageDraft shows the dashed-border draft bubble with the textarea + Speak button.
- [ ] Live chat untouched: `git diff --stat main..HEAD` shows NO changes to `web/templates/*.html` or `internal/web/chat*.go`.

## What this delivers / what's next

**Delivered:** two more chat organisms (`DialogueChoices`, `MessageDraft`); the Chat group now has 4. Catalog-only.

**Next:** `ChatBar` (the head/model switcher ledge — more stateful, its own slice), then the WIRING slice that routes the gateway through these organisms and retires the chat templates.
