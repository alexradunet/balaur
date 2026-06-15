# Gomponents Organisms — Message + ToolRow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the first two chat organisms — `chat.Message` and `chat.ToolRow` — to a new `internal/ui/chat` package (composing `internal/ui` atoms), registered as storybook stories under a new "Chat" group. Catalog-only: the live chat templates and streaming path are NOT touched.

**Architecture:** Both organisms are class-only (all chat CSS already exists in `basm.css`). `Message` renders the `msg msg-user`/`msg-balaur msg-with-avatar` portrait+bubble, composing `ui.Avatar` for the portrait. `ToolRow` renders the `msg msg-tool` row, composing `ui.Icon`. New package gets its own test `render` helper.

**Tech Stack:** Go, gomponents (`h.Figure`, `h.FigCaption` exist in v1.3.0), `internal/ui` atoms.

**Spec:** `docs/superpowers/specs/2026-06-15-chat-organisms-message-toolrow-design.md`.

**Conventions:** package `chat` uses QUALIFIED `g`/`h` imports (NO dot-import). NO new CSS this slice (all chat classes already in basm.css). Tests are `package chat_test` and use a package-local `render(t, node)` helper (Task 1 Step 0 creates it). Stories: append canvas funcs to `internal/feature/storybook/storybook.go` and `Story` entries to the `stories` slice in `internal/feature/storybook/story.go` (positional `{ID, Group, Title, Canvas}`). After each task: `go test ./...`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`. If `git status` shows any file other than the task's own as modified (e.g. a stray `chatstream.go` from a linter), do NOT stage it — `git checkout --` it.

Verified facts (do not re-derive): `internal/ui/chat` does NOT exist yet. `ui.Avatar(ui.AvatarProps{Src, Kind, State, Alt, Size})` renders `<span class="balaur-avatar balaur-avatar-{kind}" data-kind="{kind}" data-state="{state}" style="--avatar-size:{size}px"[ aria-hidden="true" when Alt==""]><img src="{Src}" alt="{Alt}" decoding="async"></span>` (Kind defaults "balaur", State "idle", Size 54). `ui.Icon(name)` renders `<img class="tool-icon" src="/static/icons/{name}.png" alt="">`. Classes `.msg`, `.msg-user`, `.msg-balaur`, `.msg-with-avatar`, `.portrait`, `.who`, `.msg-main`, `.body`, `.msg-pending`, `.thinking-dots`, `.msg-tool`, `.tool-icon` all exist in `basm.css`. `/static/crest.png` exists (story fixture image).

---

## File Structure

- **Create** `internal/ui/chat/helpers_test.go`, `internal/ui/chat/message.go`+`_test.go`, `internal/ui/chat/toolrow.go`+`_test.go`.
- **Modify** `internal/feature/storybook/storybook.go` (import chat + canvas funcs) + `internal/feature/storybook/story.go` (register two Chat stories).

---

## Task 1: `chat.Message` + story

The portrait+bubble organism for a user or balaur turn, composing `ui.Avatar`.

**Files:** Create `internal/ui/chat/helpers_test.go`, `internal/ui/chat/message.go`, `internal/ui/chat/message_test.go`; modify `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`.

- [ ] **Step 0: Create the package test helper**

Create `internal/ui/chat/helpers_test.go`:
```go
package chat_test

import (
	"strings"
	"testing"

	g "maragu.dev/gomponents"
)

// render renders a gomponents node to its HTML string, failing the test on
// error. Shared by the organism tests in this package.
func render(t *testing.T, n g.Node) string {
	t.Helper()
	var b strings.Builder
	if err := n.Render(&b); err != nil {
		t.Fatalf("render: %v", err)
	}
	return b.String()
}
```

- [ ] **Step 1: Write the failing test**

Create `internal/ui/chat/message_test.go`:
```go
package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestMessageBalaur(t *testing.T) {
	got := render(t, chat.Message(chat.MessageProps{
		Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png", Content: "Hello there.",
	}))
	for _, want := range []string{
		`<div class="msg msg-balaur msg-with-avatar">`,
		`<figure class="portrait">`,
		`<span class="balaur-avatar balaur-avatar-balaur" data-kind="balaur" data-state="idle"`,
		`<img src="/static/crest.png" alt="" decoding="async">`,
		`<figcaption class="who">Balaur</figcaption>`,
		`<div class="msg-main"><div class="body">Hello there.</div></div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("balaur message missing %q in: %s", want, got)
		}
	}
}

func TestMessageBalaurOrigin(t *testing.T) {
	got := render(t, chat.Message(chat.MessageProps{Role: "balaur", Who: "Balaur", Origin: "nudge", AvatarSrc: "/static/crest.png", Content: "x"}))
	if !strings.Contains(got, `<figcaption class="who">Balaur · nudge</figcaption>`) {
		t.Errorf("origin should render after the name: %s", got)
	}
}

func TestMessageUserDefaultName(t *testing.T) {
	got := render(t, chat.Message(chat.MessageProps{Role: "user", AvatarSrc: "/static/crest.png", Content: "Hi"}))
	for _, want := range []string{
		`<div class="msg msg-user msg-with-avatar">`,
		`class="balaur-avatar balaur-avatar-soul" data-kind="soul"`,
		`<figcaption class="who">You</figcaption>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("user message missing %q in: %s", want, got)
		}
	}
}

func TestMessagePending(t *testing.T) {
	got := render(t, chat.Message(chat.MessageProps{Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png", Pending: true}))
	if !strings.Contains(got, `<div class="msg msg-balaur msg-with-avatar msg-pending">`) {
		t.Errorf("pending should add msg-pending: %s", got)
	}
	if !strings.Contains(got, `data-state="thinking"`) {
		t.Errorf("pending balaur avatar should be thinking: %s", got)
	}
	if !strings.Contains(got, `<div class="body"><span class="thinking thinking-dots">thinking</span></div>`) {
		t.Errorf("pending empty body should be thinking-dots: %s", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/chat/ -run TestMessage -v` — Expected: FAIL (`undefined: chat.Message`/`chat.MessageProps`).

- [ ] **Step 3: Implement the organism**

Create `internal/ui/chat/message.go`:
```go
// Package chat holds the chat-surface organisms (Message, ToolRow, …) that
// compose internal/ui atoms. Class-only: all chat CSS lives in basm.css.
package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// MessageProps configures a chat Message. Role "user" renders the owner turn
// (soul avatar, name only); anything else renders a balaur turn (balaur avatar,
// optional Origin after the name). Pending marks an assistant turn mid-generation
// — it adds msg-pending and, when Content is empty, a thinking-dots body.
type MessageProps struct {
	Role      string
	Who       string
	Origin    string
	AvatarSrc string
	Content   string
	Pending   bool
}

// Message renders a chat turn: a portrait (ui.Avatar + name caption) beside a
// speech bubble.
func Message(p MessageProps) g.Node {
	user := p.Role == "user"

	cls := "msg msg-balaur msg-with-avatar"
	kind := "balaur"
	if user {
		cls = "msg msg-user msg-with-avatar"
		kind = "soul"
	}
	if p.Pending {
		cls += " msg-pending"
	}

	who := p.Who
	if who == "" {
		if user {
			who = "You"
		} else {
			who = "Balaur"
		}
	}
	caption := []g.Node{h.Class("who"), g.Text(who)}
	if !user && p.Origin != "" {
		caption = append(caption, g.Text(" · "+p.Origin))
	}

	state := "idle"
	if p.Pending && !user {
		state = "thinking"
	}

	var body g.Node
	if p.Pending && p.Content == "" {
		body = h.Span(h.Class("thinking thinking-dots"), g.Text("thinking"))
	} else {
		body = g.Text(p.Content)
	}

	return h.Div(h.Class(cls),
		h.Figure(h.Class("portrait"),
			ui.Avatar(ui.AvatarProps{Src: p.AvatarSrc, Kind: kind, State: state}),
			h.FigCaption(caption...),
		),
		h.Div(h.Class("msg-main"), h.Div(h.Class("body"), body)),
	)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/chat/ -run TestMessage -v` — Expected: PASS (all four).

- [ ] **Step 5: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, add the import (in the existing import block, after the `ui` import):
```go
	"github.com/alexradunet/balaur/internal/ui/chat"
```
Then append at the end of the file:
```go

func chatMessageCanvas() g.Node {
	return section("Message",
		chat.Message(chat.MessageProps{Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png", Content: "Noted — I'll remind you at 6pm. Anything else for the book?"}),
		chat.Message(chat.MessageProps{Role: "user", Who: "You", AvatarSrc: "/static/crest.png", Content: "Add: water the tomatoes every 2 days."}),
		chat.Message(chat.MessageProps{Role: "balaur", Who: "Balaur", AvatarSrc: "/static/crest.png", Pending: true}),
	)
}
```
In `internal/feature/storybook/story.go`, add to the `stories` slice immediately AFTER the `{"screentitle", "Typography", "ScreenTitle", screenTitleCanvas},` line:
```go
	{"chatmessage", "Chat", "Message", chatMessageCanvas},
```

- [ ] **Step 6: Verify + commit**

Run:
```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/chat/ -run TestMessage && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
git status --short
```
Expected: PASS, full suite green, build clean. If `git status --short` shows any file other than `internal/ui/chat/helpers_test.go`, `internal/ui/chat/message.go`, `internal/ui/chat/message_test.go`, `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`, do NOT stage it — `git checkout -- <file>`. Then:
```bash
git add internal/ui/chat/helpers_test.go internal/ui/chat/message.go internal/ui/chat/message_test.go internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add chat.Message organism + storybook story\n\nNew internal/ui/chat package. chat.Message(MessageProps) — portrait (composes\nui.Avatar) + speech bubble for a user/balaur turn; Pending adds msg-pending +\nthinking-dots. Class-only (chat CSS already in basm.css). New Chat story group.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: `chat.ToolRow` + story

The tool-invocation row, composing `ui.Icon`.

**Files:** Create `internal/ui/chat/toolrow.go`, `internal/ui/chat/toolrow_test.go`; modify `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/chat/toolrow_test.go`:
```go
package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestToolRow(t *testing.T) {
	got := render(t, chat.ToolRow(chat.ToolRowProps{
		Tool: "task_add", Icon: "scroll", Content: "added task: water the tomatoes · every 2 days 18:00",
	}))
	for _, want := range []string{
		`<div class="msg msg-tool">`,
		`<div class="who"><img class="tool-icon" src="/static/icons/scroll.png" alt="">tool · task_add</div>`,
		`<div class="body">added task: water the tomatoes · every 2 days 18:00</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("tool row missing %q in: %s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/chat/ -run TestToolRow -v` — Expected: FAIL (`undefined: chat.ToolRow`/`chat.ToolRowProps`).

- [ ] **Step 3: Implement the organism**

Create `internal/ui/chat/toolrow.go`:
```go
package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"

	"github.com/alexradunet/balaur/internal/ui"
)

// ToolRowProps configures a ToolRow. Tool is the tool name (rendered "tool ·
// {Tool}"); Icon is a /static/icons name (composed via ui.Icon); Content is the
// result text.
type ToolRowProps struct {
	Tool    string
	Icon    string
	Content string
}

// ToolRow renders a tool-invocation row: a wood-inset line with the tool icon +
// name and the result body, indented under the chat gutter.
func ToolRow(p ToolRowProps) g.Node {
	return h.Div(h.Class("msg msg-tool"),
		h.Div(h.Class("who"), ui.Icon(p.Icon), g.Text("tool · "+p.Tool)),
		h.Div(h.Class("body"), g.Text(p.Content)),
	)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/chat/ -run TestToolRow -v` — Expected: PASS.

- [ ] **Step 5: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append at the end of the file:
```go

func chatToolRowCanvas() g.Node {
	return section("ToolRow",
		chat.ToolRow(chat.ToolRowProps{Tool: "task_add", Icon: "scroll", Content: "added task: water the tomatoes · every 2 days 18:00"}),
		chat.ToolRow(chat.ToolRowProps{Tool: "remember", Icon: "tome", Content: "saved: prefers tea over coffee"}),
	)
}
```
In `internal/feature/storybook/story.go`, add to the `stories` slice immediately AFTER the `{"chatmessage", "Chat", "Message", chatMessageCanvas},` line:
```go
	{"chattoolrow", "Chat", "ToolRow", chatToolRowCanvas},
```

- [ ] **Step 6: Verify + commit**

Run:
```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/chat/ -run TestToolRow && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./... && go vet ./...
git status --short
```
Expected: PASS, full suite green, build+vet clean. If `git status --short` shows any file other than the four task files, do NOT stage it — `git checkout -- <file>`. Then:
```bash
git add internal/ui/chat/toolrow.go internal/ui/chat/toolrow_test.go internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add chat.ToolRow organism + storybook story\n\nchat.ToolRow(ToolRowProps) — the msg-tool invocation row (composes ui.Icon for\nthe tool icon). Class-only. Registered under the Chat story group.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification

- [ ] `go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check` — all green.
- [ ] Sidebar has a new **Chat** group (Message, ToolRow); `/storybook/chatmessage` and `/storybook/chattoolrow` render 200 with the `msg`/`msg-tool` markup (verify via `curl …/storybook/chatmessage | grep msg-balaur`, NOT just a screenshot — a stale debug binary on :8090 will silently serve the Overview fallback).
- [ ] Live chat untouched: `git diff --stat main..HEAD` shows NO changes to `internal/web/chatstream.go`, `internal/web/chat.go`, or `web/templates/chat-messages.html`.

## What this delivers / what's next

**Delivered:** the first two chat organisms (`chat.Message`, `chat.ToolRow`) in a new `internal/ui/chat` package, composing atoms, shown in a new storybook Chat group. The atom-composition organism pattern is proven end-to-end.

**Next:** more organisms (DialogueChoices, the Composer trio, inline cards) — then the WIRING slice that routes `chatstream.go`/`chat.go` through these organisms and retires the `chat-msg-*` templates (budget golden/template updates for the ui.Avatar superset markup + verify SSE morph-target IDs).
