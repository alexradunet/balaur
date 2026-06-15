# Gomponents Organisms — ChatBar (HeadSwitcher + ModelSwitcher) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add the chat-input ledge organisms — `chat.HeadSwitcher`, `chat.ModelSwitcher`, and a `chat.ChatBar` that composes them — to `internal/ui/chat`, registered as storybook stories under the existing "Chat" group. Catalog-only, ready-state (the Datastar head-switch forms and the download/empty/error states are behavioral, deferred to the wiring slice).

**Architecture:** All class-only (chat CSS already in basm.css). `HeadSwitcher` is the persona picker (`<section class="head-switcher">` with a `.head-switcher-list` of choice buttons). `ModelSwitcher` is the model line + profile link (`<section class="model-switcher">`). `ChatBar` wraps both in the `.chatbar.chatbar-slim` ledge. One storybook-chrome CSS override is needed because `.chatbar` is `position: fixed`.

**Tech Stack:** Go, gomponents (`h.Section`/`h.Ul`/`h.Li`/`h.Aria`/`h.ID` all exist in v1.3.0), `internal/ui` atoms (none composed here — the avatars are bare `.px` imgs, not `ui.Avatar`).

**Conventions:** package `chat` uses QUALIFIED `g`/`h` imports (NO dot-import). NO new component CSS (chat classes exist) EXCEPT one storybook-display override appended to basm.css. Tests are `package chat_test` using the existing `render(t, node)` helper. Stories append to `internal/feature/storybook/storybook.go` + `story.go`. After each task: `go test ./...`, `CGO_ENABLED=0 go build ./...`, `go vet ./...`. If `git status` shows a non-task file modified, `git checkout --` it.

Verified facts: classes `.chatbar`, `.chatbar-slim`, `.head-switcher`, `.head-switcher-current`, `.head-switcher-list`, `.head-switcher-choice`, `.head-switcher-choice-active`, `.model-switcher`, `.model-switcher-head`, `.model-switcher-kicker`, `.model-current`, `.model-switcher-manage`, `.chatbar-profile-link`, `.chatbar-profile-href`, `.px`, `.balaur-avatar`, `.balaur-avatar-soul` all exist in basm.css. `.chatbar` is `position: fixed` (basm.css), overridden to static only under `#dock` — so the storybook needs `.sb-canvas .chatbar { position: static }`. `/static/crest.png` exists. The Chat group's last story entry is `{"messagedraft", "Chat", "MessageDraft", messageDraftCanvas}`. gomponents `g.Text` auto-escapes `&` → `&amp;`.

The live markup these mirror (ready state, from `web/templates/home.html`):
```html
<section class="head-switcher" aria-label="Head">
  <span class="model-switcher-kicker">Head</span>
  <span class="head-switcher-current">{ActiveHead}</span>
  <ul class="head-switcher-list">
    <li>…<button class="head-switcher-choice[ head-switcher-choice-active]"><img class="px" src=…><span>{Name}</span></button>…</li>
  </ul>
</section>
<section class="model-switcher" aria-label="Model">
  <div class="model-switcher-head">
    <span class="model-switcher-kicker">Model</span>
    <span class="model-current">{ActiveModel}</span>
    <a class="model-switcher-manage" href="/focus/settings?section=models">Manage models &amp; APIs →</a>
  </div>
  <div class="chatbar-profile-link">
    <span class="balaur-avatar balaur-avatar-soul" aria-hidden="true"><img class="px" src={AvatarSrc} alt="" decoding="async"></span>
    <a href="/focus/settings?section=profile" class="chatbar-profile-href">Your avatar &amp; profile →</a>
  </div>
</section>
```
(Catalog omits the live `<li><form data-on:submit><input hidden>` wrapper + `data-attr:disabled` — behavioral, wired later. Renders `<li><button type="button">` instead.)

---

## File Structure

- **Create** `internal/ui/chat/headswitcher.go`+`_test.go`, `internal/ui/chat/modelswitcher.go`+`_test.go`, `internal/ui/chat/chatbar.go`+`_test.go`.
- **Modify** `internal/feature/storybook/storybook.go` (canvas funcs) + `internal/feature/storybook/story.go` (3 Chat stories) + `internal/web/assets/static/basm.css` (one storybook override, Task 3).

---

## Task 1: `chat.ModelSwitcher` + story

**Files:** Create `internal/ui/chat/modelswitcher.go`, `internal/ui/chat/modelswitcher_test.go`; modify `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/chat/modelswitcher_test.go`:
```go
package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestModelSwitcher(t *testing.T) {
	got := render(t, chat.ModelSwitcher(chat.ModelSwitcherProps{
		ActiveModel: "gemma3:4b", AvatarSrc: "/static/crest.png",
	}))
	for _, want := range []string{
		`<section class="model-switcher" aria-label="Model">`,
		`<span class="model-switcher-kicker">Model</span>`,
		`<span class="model-current">gemma3:4b</span>`,
		`<a class="model-switcher-manage" href="/focus/settings?section=models">`,
		`<div class="chatbar-profile-link"><span class="balaur-avatar balaur-avatar-soul" aria-hidden="true"><img class="px" src="/static/crest.png" alt="" decoding="async"></span>`,
		`<a href="/focus/settings?section=profile" class="chatbar-profile-href">`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("model switcher missing %q in: %s", want, got)
		}
	}
}

func TestModelSwitcherNoModel(t *testing.T) {
	got := render(t, chat.ModelSwitcher(chat.ModelSwitcherProps{AvatarSrc: "/static/crest.png"}))
	if strings.Contains(got, "model-current") {
		t.Errorf("no ActiveModel should omit .model-current: %s", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/chat/ -run TestModelSwitcher -v` — Expected: FAIL (`undefined: chat.ModelSwitcher`/`chat.ModelSwitcherProps`).

- [ ] **Step 3: Implement the organism**

Create `internal/ui/chat/modelswitcher.go`:
```go
package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ModelSwitcherProps configures a ModelSwitcher (ready state). ActiveModel is the
// current model label (the .model-current pill is omitted when empty); AvatarSrc is
// the owner's profile avatar.
type ModelSwitcherProps struct {
	ActiveModel string
	AvatarSrc   string
}

// ModelSwitcher renders the model line + profile link of the chat ledge. Static
// (catalog) — the download/empty/error states are wired by the gateway later.
func ModelSwitcher(p ModelSwitcherProps) g.Node {
	head := []g.Node{
		h.Class("model-switcher-head"),
		h.Span(h.Class("model-switcher-kicker"), g.Text("Model")),
	}
	if p.ActiveModel != "" {
		head = append(head, h.Span(h.Class("model-current"), g.Text(p.ActiveModel)))
	}
	head = append(head, h.A(h.Class("model-switcher-manage"), h.Href("/focus/settings?section=models"), g.Text("Manage models & APIs →")))
	return h.Section(h.Class("model-switcher"), h.Aria("label", "Model"),
		h.Div(head...),
		h.Div(h.Class("chatbar-profile-link"),
			h.Span(h.Class("balaur-avatar balaur-avatar-soul"), h.Aria("hidden", "true"),
				h.Img(h.Class("px"), h.Src(p.AvatarSrc), h.Alt(""), g.Attr("decoding", "async"))),
			h.A(h.Href("/focus/settings?section=profile"), h.Class("chatbar-profile-href"), g.Text("Your avatar & profile →")),
		),
	)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/chat/ -run TestModelSwitcher -v` — Expected: PASS (both).

- [ ] **Step 5: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append:
```go

func modelSwitcherCanvas() g.Node {
	return section("ModelSwitcher",
		chat.ModelSwitcher(chat.ModelSwitcherProps{ActiveModel: "gemma3:4b", AvatarSrc: "/static/crest.png"}),
	)
}
```
In `internal/feature/storybook/story.go`, add immediately AFTER the `{"messagedraft", "Chat", "MessageDraft", messageDraftCanvas},` line:
```go
	{"modelswitcher", "Chat", "ModelSwitcher", modelSwitcherCanvas},
```

- [ ] **Step 6: Verify + commit**

```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/chat/ -run TestModelSwitcher && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
git status --short
```
Expected: PASS, suite green, build clean. Stage only the four files, then:
```bash
git add internal/ui/chat/modelswitcher.go internal/ui/chat/modelswitcher_test.go internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add chat.ModelSwitcher organism + storybook story\n\nchat.ModelSwitcher(ModelSwitcherProps) — the ready-state model line (kicker +\nmodel pill + manage link) and profile link of the chat ledge. Class-only.\nRegistered under the Chat group.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 2: `chat.HeadSwitcher` + story

**Files:** Create `internal/ui/chat/headswitcher.go`, `internal/ui/chat/headswitcher_test.go`; modify `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/chat/headswitcher_test.go`:
```go
package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestHeadSwitcher(t *testing.T) {
	got := render(t, chat.HeadSwitcher(chat.HeadSwitcherProps{
		ActiveHead: "Balaur",
		Heads: []chat.Head{
			{Name: "Balaur", AvatarSrc: "/static/crest.png", Active: true},
			{Name: "Scholar", AvatarSrc: "/static/crest.png"},
		},
	}))
	for _, want := range []string{
		`<section class="head-switcher" aria-label="Head">`,
		`<span class="model-switcher-kicker">Head</span>`,
		`<span class="head-switcher-current">Balaur</span>`,
		`<ul class="head-switcher-list">`,
		`<li><button class="head-switcher-choice head-switcher-choice-active" type="button" aria-current="true"><img class="px" src="/static/crest.png" alt="" decoding="async"><span>Balaur</span></button></li>`,
		`<li><button class="head-switcher-choice" type="button"><img class="px" src="/static/crest.png" alt="" decoding="async"><span>Scholar</span></button></li>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("head switcher missing %q in: %s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/chat/ -run TestHeadSwitcher -v` — Expected: FAIL (`undefined: chat.HeadSwitcher`/`chat.HeadSwitcherProps`/`chat.Head`).

- [ ] **Step 3: Implement the organism**

Create `internal/ui/chat/headswitcher.go`:
```go
package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// Head is one persona choice: a Name, an AvatarSrc, and whether it's Active.
type Head struct {
	Name      string
	AvatarSrc string
	Active    bool
}

// HeadSwitcherProps configures a HeadSwitcher. ActiveHead is the current persona
// name; Heads are the choices.
type HeadSwitcherProps struct {
	ActiveHead string
	Heads      []Head
}

// HeadSwitcher renders the persona picker of the chat ledge: a labelled list of
// head choice buttons (active one marked). Static (catalog) — the select-a-head
// Datastar form is wired by the gateway later.
func HeadSwitcher(p HeadSwitcherProps) g.Node {
	items := make([]g.Node, 0, len(p.Heads))
	for _, hd := range p.Heads {
		cls := "head-switcher-choice"
		if hd.Active {
			cls += " head-switcher-choice-active"
		}
		btn := []g.Node{h.Class(cls), h.Type("button")}
		if hd.Active {
			btn = append(btn, h.Aria("current", "true"))
		}
		btn = append(btn,
			h.Img(h.Class("px"), h.Src(hd.AvatarSrc), h.Alt(""), g.Attr("decoding", "async")),
			h.Span(g.Text(hd.Name)),
		)
		items = append(items, h.Li(h.Button(btn...)))
	}
	return h.Section(h.Class("head-switcher"), h.Aria("label", "Head"),
		h.Span(h.Class("model-switcher-kicker"), g.Text("Head")),
		h.Span(h.Class("head-switcher-current"), g.Text(p.ActiveHead)),
		h.Ul(append([]g.Node{h.Class("head-switcher-list")}, items...)...),
	)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/chat/ -run TestHeadSwitcher -v` — Expected: PASS.

- [ ] **Step 5: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append:
```go

func headSwitcherCanvas() g.Node {
	return section("HeadSwitcher",
		chat.HeadSwitcher(chat.HeadSwitcherProps{
			ActiveHead: "Balaur",
			Heads: []chat.Head{
				{Name: "Balaur", AvatarSrc: "/static/crest.png", Active: true},
				{Name: "Scholar", AvatarSrc: "/static/crest.png"},
				{Name: "Planner", AvatarSrc: "/static/crest.png"},
			},
		}),
	)
}
```
In `internal/feature/storybook/story.go`, add immediately AFTER the `{"modelswitcher", "Chat", "ModelSwitcher", modelSwitcherCanvas},` line:
```go
	{"headswitcher", "Chat", "HeadSwitcher", headSwitcherCanvas},
```

- [ ] **Step 6: Verify + commit**

```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/chat/ -run TestHeadSwitcher && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./...
git status --short
```
Expected: PASS, suite green, build clean. Stage only the four files, then:
```bash
git add internal/ui/chat/headswitcher.go internal/ui/chat/headswitcher_test.go internal/feature/storybook/storybook.go internal/feature/storybook/story.go
git commit -m "$(printf 'feat(ui): add chat.HeadSwitcher organism + storybook story\n\nchat.HeadSwitcher(HeadSwitcherProps) — the persona picker of the chat ledge: a\nlabelled list of head choice buttons (active marked). Class-only. Registered\nunder the Chat group.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Task 3: `chat.ChatBar` (composes both) + story + storybook override

**Files:** Create `internal/ui/chat/chatbar.go`, `internal/ui/chat/chatbar_test.go`; modify `internal/feature/storybook/storybook.go`, `internal/feature/storybook/story.go`, `internal/web/assets/static/basm.css`.

- [ ] **Step 1: Write the failing test**

Create `internal/ui/chat/chatbar_test.go`:
```go
package chat_test

import (
	"strings"
	"testing"

	"github.com/alexradunet/balaur/internal/ui/chat"
)

func TestChatBar(t *testing.T) {
	got := render(t, chat.ChatBar(chat.ChatBarProps{
		ActiveHead:  "Balaur",
		Heads:       []chat.Head{{Name: "Balaur", AvatarSrc: "/static/crest.png", Active: true}},
		ActiveModel: "gemma3:4b",
		AvatarSrc:   "/static/crest.png",
	}))
	for _, want := range []string{
		`<div class="chatbar chatbar-slim">`,
		`<section class="head-switcher" aria-label="Head">`,
		`<section class="model-switcher" aria-label="Model">`,
		`<span class="head-switcher-current">Balaur</span>`,
		`<span class="model-current">gemma3:4b</span>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("chatbar missing %q in: %s", want, got)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `go test ./internal/ui/chat/ -run TestChatBar -v` — Expected: FAIL (`undefined: chat.ChatBar`/`chat.ChatBarProps`).

- [ ] **Step 3: Implement the organism**

Create `internal/ui/chat/chatbar.go`:
```go
package chat

import (
	g "maragu.dev/gomponents"
	h "maragu.dev/gomponents/html"
)

// ChatBarProps configures the ChatBar ledge — the union of the HeadSwitcher and
// ModelSwitcher props.
type ChatBarProps struct {
	ActiveHead  string
	Heads       []Head
	ActiveModel string
	AvatarSrc   string
}

// ChatBar renders the wood input ledge: the HeadSwitcher beside the ModelSwitcher.
// Static (catalog). In the live app this is fixed to the dock bottom; the storybook
// shows it inline via a .sb-canvas .chatbar override.
func ChatBar(p ChatBarProps) g.Node {
	return h.Div(h.Class("chatbar chatbar-slim"),
		HeadSwitcher(HeadSwitcherProps{ActiveHead: p.ActiveHead, Heads: p.Heads}),
		ModelSwitcher(ModelSwitcherProps{ActiveModel: p.ActiveModel, AvatarSrc: p.AvatarSrc}),
	)
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `go test ./internal/ui/chat/ -run TestChatBar -v` — Expected: PASS.

- [ ] **Step 5: Add the storybook override CSS**

Append at the END of `internal/web/assets/static/basm.css`:
```css

/* ── Storybook: show the fixed .chatbar inline in the canvas ─────────────── */
.sb-canvas .chatbar { position: static; }
```

- [ ] **Step 6: Add the canvas + register the story**

In `internal/feature/storybook/storybook.go`, append:
```go

func chatBarCanvas() g.Node {
	return section("ChatBar",
		chat.ChatBar(chat.ChatBarProps{
			ActiveHead: "Balaur",
			Heads: []chat.Head{
				{Name: "Balaur", AvatarSrc: "/static/crest.png", Active: true},
				{Name: "Scholar", AvatarSrc: "/static/crest.png"},
				{Name: "Planner", AvatarSrc: "/static/crest.png"},
			},
			ActiveModel: "gemma3:4b",
			AvatarSrc:   "/static/crest.png",
		}),
	)
}
```
In `internal/feature/storybook/story.go`, add immediately AFTER the `{"headswitcher", "Chat", "HeadSwitcher", headSwitcherCanvas},` line:
```go
	{"chatbar", "Chat", "ChatBar", chatBarCanvas},
```

- [ ] **Step 7: Verify + commit**

```bash
cd /home/alex/Projects/balaur
go test ./internal/ui/chat/ -run TestChatBar && go test ./... 2>&1 | grep -E "FAIL" || echo "FULL SUITE GREEN"
CGO_ENABLED=0 go build ./... && go vet ./...
tail -4 internal/web/assets/static/basm.css | grep -nE ":[^;{]*#[0-9a-fA-F]{3,6}\b" || echo "NO RAW HEX"
git status --short
```
Expected: PASS, suite green, build+vet clean, NO RAW HEX. Stage only the five files, then:
```bash
git add internal/ui/chat/chatbar.go internal/ui/chat/chatbar_test.go internal/feature/storybook/storybook.go internal/feature/storybook/story.go internal/web/assets/static/basm.css
git commit -m "$(printf 'feat(ui): add chat.ChatBar organism + storybook story\n\nchat.ChatBar(ChatBarProps) — the wood input ledge composing HeadSwitcher +\nModelSwitcher. Class-only. Storybook override renders the fixed .chatbar inline.\nRegistered under the Chat group.\n\nCo-Authored-By: Claude Opus 4.8 (1M context) <noreply@anthropic.com>')"
```

---

## Final verification (controller — visual)

- [ ] `go vet ./... && go test ./... && CGO_ENABLED=0 go build ./... && git diff --check` — green.
- [ ] Chat group lists Message / ToolRow / DialogueChoices / MessageDraft / ModelSwitcher / HeadSwitcher / ChatBar; `/storybook/chatbar` renders 200 (content-assert `chatbar-slim`, not just status).
- [ ] Screenshot ChatBar (Hearthwood): the wood ledge with the head choices (Balaur active) + the model line ("gemma3:4b") + profile link, shown inline (not pinned to the viewport bottom).
- [ ] Live chat untouched: `git diff --stat main..HEAD -- web/templates/*.html internal/web/chat*.go` empty.

## What this delivers / what's next

**Delivered:** the chat-ledge organisms (`HeadSwitcher`, `ModelSwitcher`, `ChatBar`); the Chat catalog now covers the full chat surface (7 organisms). Catalog-only — the head-switch Datastar forms + download/empty/error states are deferred.

**Next:** the **WIRING slice** — route the chat gateway (`chatstream.go`/`chat.go`/`home.html`) through these organisms (adding Datastar passthrough), retire the `chat-msg-*`/`chat_draft`/`chat-choices`/`chat_bar` template defines, verify the SSE morph IDs. The first real deletion of live `html/template` surface → Phase 3 (storybook → `/`) → cut boards + delete `web/`.
